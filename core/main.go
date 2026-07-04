package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"

	dockerctr "github.com/docker/docker/api/types/container"
	dockerfilters "github.com/docker/docker/api/types/filters"
	dockerimage "github.com/docker/docker/api/types/image"
	dockernet "github.com/docker/docker/api/types/network"
	dockervol "github.com/docker/docker/api/types/volume"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"core/apps"
	"core/auth"
	"core/caddy"
	"core/compose"
	"core/docker"
	"core/selfinstall"
	"core/shared"
	"core/system"
	"core/telemetry"
)

// suppress unused import warning for dockernet (used in NetworkList)
var _ = dockernet.NetworkingConfig{}

const serverPort = 5252

var version = "0.0.0" // injected at build time

var publicPaths = map[string]bool{
	"/api/health":       true,
	"/api/version":      true,
	"/api/setup/status": true,
	"/api/setup/admin":  true,
	"/api/auth/login":   true,
}

// ─── Response helpers ─────────────────────────────────────────────────────────

func jsonResp(w http.ResponseWriter, status int, body interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func jsonErr(w http.ResponseWriter, status int, code, message string) {
	jsonResp(w, status, map[string]interface{}{
		"success": false,
		"code":    code,
		"message": message,
		"error":   message,
	})
}

func parseJSON(w http.ResponseWriter, r *http.Request, v interface{}) bool {
	if err := json.NewDecoder(r.Body).Decode(v); err != nil {
		jsonErr(w, http.StatusBadRequest, "INVALID_JSON", "Invalid JSON body: "+err.Error())
		return false
	}
	return true
}

func coalesce(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

// ─── Directory helpers ────────────────────────────────────────────────────────

func getAppsDir() string {
	if d := os.Getenv("YANTR_APPS_DIR"); d != "" {
		return d
	}
	exe, _ := os.Executable()
	return filepath.Join(filepath.Dir(exe), "apps")
}

func getDistDir() string {
	exe, _ := os.Executable()
	return filepath.Join(filepath.Dir(exe), "dist")
}

func getBaseAppID(projectID string) string {
	return shared.GetBaseAppID(projectID)
}

// ─── Process spawning ─────────────────────────────────────────────────────────

func spawnExec(name string, args []string, env map[string]string, cwd string) (string, string, int, error) {
	cmd := exec.Command(name, args...)
	if cwd != "" {
		cmd.Dir = cwd
	}
	if env != nil {
		var envList []string
		for k, v := range env {
			envList = append(envList, k+"="+v)
		}
		cmd.Env = envList
	}
	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	exitCode := 0
	if err != nil {
		if cmd.ProcessState != nil {
			exitCode = cmd.ProcessState.ExitCode()
		} else {
			exitCode = 1
		}
	}
	return stdout.String(), stderr.String(), exitCode, err
}

// ─── Compose command detection ────────────────────────────────────────────────

var cachedCompose struct {
	cmd  string
	args []string
	done bool
}

func getComposeCommand() (string, []string, error) {
	if cachedCompose.done {
		return cachedCompose.cmd, cachedCompose.args, nil
	}
	env := map[string]string{"DOCKER_HOST": "unix://" + docker.SocketPath}
	if _, _, code, err := spawnExec("docker", []string{"compose", "version"}, env, ""); err == nil && code == 0 {
		cachedCompose = struct{ cmd string; args []string; done bool }{"docker", []string{"compose"}, true}
		return cachedCompose.cmd, cachedCompose.args, nil
	}
	if _, _, code, err := spawnExec("docker-compose", []string{"version"}, env, ""); err == nil && code == 0 {
		cachedCompose = struct{ cmd string; args []string; done bool }{"docker-compose", nil, true}
		return cachedCompose.cmd, cachedCompose.args, nil
	}
	return "", nil, fmt.Errorf("docker compose is not available")
}

// ─── Label parsing ────────────────────────────────────────────────────────────

type appLabelSet struct {
	App     string `json:"app"`
	Service string `json:"service"`
}

func parseAppLabels(labels map[string]string) appLabelSet {
	return appLabelSet{
		App:     labels["yantr.app"],
		Service: labels["yantr.service"],
	}
}

// ─── Catalog helpers ──────────────────────────────────────────────────────────

func entryStr(e *apps.App, field string) string {
	if e == nil {
		return ""
	}
	switch field {
	case "name":
		return e.Name
	case "logo":
		return e.Logo
	case "short_description":
		return e.ShortDescription
	case "description":
		return e.Description
	case "website":
		return e.Website
	}
	return ""
}

func entrySlice(e *apps.App, field string) interface{} {
	if e == nil {
		return []string{}
	}
	switch field {
	case "tags":
		if e.Tags != nil {
			return e.Tags
		}
		return []string{}
	case "usecases":
		if e.Usecases != nil {
			return e.Usecases
		}
		return []string{}
	}
	return []string{}
}

func entryPorts(e *apps.App) interface{} {
	if e == nil || e.Ports == nil {
		return []apps.PortInfo{}
	}
	return e.Ports
}

func getCatalogMap() map[string]*apps.App {
	cat, _ := apps.GetCatalogCached(false)
	m := map[string]*apps.App{}
	if cat == nil {
		return m
	}
	for i := range cat.Apps {
		m[cat.Apps[i].ID] = &cat.Apps[i]
	}
	return m
}

// ─── Middleware ───────────────────────────────────────────────────────────────

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		isProtected := strings.HasPrefix(path, "/api/") || strings.HasPrefix(path, "/browse/")
		if !isProtected || publicPaths[path] {
			next.ServeHTTP(w, r)
			return
		}
		cfg, err := auth.LoadAuthConfig(false)
		if err != nil || cfg == nil {
			jsonErr(w, http.StatusServiceUnavailable, "SETUP_REQUIRED", "Setup required")
			return
		}
		token := auth.ExtractBearerToken(r.Header.Get("Authorization"))
		if _, err := auth.VerifyToken(token, cfg); err != nil {
			jsonErr(w, http.StatusUnauthorized, "UNAUTHORIZED", "Unauthorized")
			return
		}
		next.ServeHTTP(w, r)
	})
}

// ─── Browse proxy ─────────────────────────────────────────────────────────────

func browseProxyHandler(w http.ResponseWriter, r *http.Request) {
	parts := strings.SplitN(r.URL.Path, "/", 4)
	if len(parts) < 3 || parts[2] == "" {
		http.Error(w, "invalid browse path", http.StatusBadRequest)
		return
	}
	volumeName := parts[2]
	p := browserRegistry.GetPort(volumeName)
	if p == 0 {
		http.Error(w, fmt.Sprintf("Volume browser for %q is not running. Start it from the Volumes page.", volumeName), http.StatusServiceUnavailable)
		return
	}
	target, _ := url.Parse(fmt.Sprintf("http://localhost:%d", p))
	proxy := httputil.NewSingleHostReverseProxy(target)
	proxy.ServeHTTP(w, r)
}

// ─── Handlers: health / version / auth ───────────────────────────────────────

func handleHealth(w http.ResponseWriter, r *http.Request) {
	jsonResp(w, 200, map[string]interface{}{
		"success": true, "status": "ok",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
		"version":   version,
	})
}

func handleVersion(w http.ResponseWriter, r *http.Request) {
	jsonResp(w, 200, map[string]interface{}{"success": true, "version": version})
}

func handleSetupStatus(w http.ResponseWriter, r *http.Request) {
	cfg, _ := auth.LoadAuthConfig(false)
	jsonResp(w, 200, map[string]interface{}{"success": true, "configured": cfg != nil})
}

func handleSetupAdmin(w http.ResponseWriter, r *http.Request) {
	if existing, _ := auth.LoadAuthConfig(false); existing != nil {
		jsonErr(w, 409, "SETUP_ALREADY_CONFIGURED", "Yantr is already configured")
		return
	}
	var body struct {
		Username  string `json:"username"`
		SecretHex string `json:"secretHex"`
	}
	if !parseJSON(w, r, &body) {
		return
	}
	cfg, err := auth.SaveAuthConfig(body.Username, body.SecretHex)
	if err != nil {
		jsonErr(w, 400, "INVALID_SETUP_ADMIN_REQUEST", err.Error())
		return
	}
	jsonResp(w, 201, map[string]interface{}{"success": true, "configured": true, "username": cfg.Username})
}

func handleLogin(w http.ResponseWriter, r *http.Request) {
	cfg, _ := auth.LoadAuthConfig(false)
	if cfg == nil {
		jsonErr(w, 409, "SETUP_REQUIRED", "Setup required")
		return
	}
	token := auth.ExtractBearerToken(r.Header.Get("Authorization"))
	username, err := auth.VerifyToken(token, cfg)
	if err != nil {
		jsonErr(w, 401, "UNAUTHORIZED", "Unauthorized")
		return
	}
	jsonResp(w, 200, map[string]interface{}{
		"success": true, "authenticated": true,
		"user": map[string]interface{}{"username": username},
	})
}

func handleLogs(w http.ResponseWriter, r *http.Request) {
	limit := shared.MaxLogs
	if n, err := strconv.Atoi(r.URL.Query().Get("limit")); err == nil && n > 0 {
		limit = n
	}
	level := r.URL.Query().Get("level")
	logs := shared.GetLogs(level, limit)
	jsonResp(w, 200, map[string]interface{}{
		"success": true, "count": len(logs), "maxLogs": shared.MaxLogs, "logs": logs,
	})
}

// ─── Handlers: apps ───────────────────────────────────────────────────────────

func handleApps(w http.ResponseWriter, r *http.Request) {
	refresh := r.URL.Query().Get("refresh") == "1" || r.URL.Query().Get("refresh") == "true"
	cat, err := apps.GetCatalogCached(refresh)
	if err != nil {
		jsonErr(w, 500, "APPS_LOAD_FAILED", err.Error())
		return
	}
	jsonResp(w, 200, map[string]interface{}{"success": true, "count": cat.Count, "apps": cat.Apps})
}

func handleCheckArch(w http.ResponseWriter, r *http.Request) {
	appID := chi.URLParam(r, "id")
	composePath := filepath.Join(apps.GetAppsDir(), appID, "compose.yml")
	content, err := os.ReadFile(composePath)
	if err != nil {
		jsonErr(w, 404, "APP_NOT_FOUND", "App not found")
		return
	}
	re := regexp.MustCompile(`image:\s*([^\s\n]+)`)
	m := re.FindSubmatch(content)
	if m == nil {
		jsonErr(w, 400, "IMAGE_NOT_FOUND", "Could not extract image name from compose file")
		return
	}
	jsonResp(w, 200, map[string]interface{}{
		"success": true, "appId": appID,
		"image": strings.TrimSpace(string(m[1])),
		"supported": "unknown", "systemArch": system.GetSystemArch(), "imageArch": "unknown",
	})
}

func handleDeploy(w http.ResponseWriter, r *http.Request) {
	var body struct {
		AppID              string                 `json:"appId"`
		Environment        map[string]interface{} `json:"environment"`
		ExtraEnv           map[string]interface{} `json:"extraEnv"`
		ExpiresIn          float64                `json:"expiresIn"`
		InstanceID         int                    `json:"instanceId"`
		MasterApp          string                 `json:"masterApp"`
		CustomPortMappings map[string]interface{} `json:"customPortMappings"`
	}
	if !parseJSON(w, r, &body) {
		return
	}
	if body.AppID == "" {
		jsonErr(w, 400, "APP_ID_REQUIRED", "appId is required")
		return
	}

	appPath := filepath.Join(apps.GetAppsDir(), body.AppID)
	baseContent, err := os.ReadFile(filepath.Join(appPath, "compose.yml"))
	if err != nil {
		jsonErr(w, 404, "APP_NOT_FOUND", fmt.Sprintf("App '%s' not found or has no compose.yml", body.AppID))
		return
	}

	// Check external networks
	if doc, err := compose.Parse(string(baseContent)); err == nil {
		if networks, ok := doc["networks"].(map[string]interface{}); ok {
			var missing []string
			for netName, netCfgRaw := range networks {
				if netCfg, ok := netCfgRaw.(map[string]interface{}); ok {
					if ext, _ := netCfg["external"].(bool); ext {
						name := coalesce(func() string {
							if n, ok := netCfg["name"].(string); ok {
								return n
							}
							return ""
						}(), netName)
						nets, err := docker.Client.NetworkList(context.Background(), dockernet.ListOptions{
							Filters: dockerfilters.NewArgs(dockerfilters.Arg("name", name)),
						})
						if err != nil || !func() bool {
							for _, n := range nets {
								if n.Name == name {
									return true
								}
							}
							return false
						}() {
							missing = append(missing, name)
						}
					}
				}
			}
			if len(missing) > 0 {
				jsonErr(w, 400, "MISSING_NETWORKS",
					fmt.Sprintf("Required network(s) %s do not exist.", strings.Join(missing, ", ")))
				return
			}
		}
	}

	// Build extra env
	extraEnv := map[string]interface{}{}
	for k, v := range body.ExtraEnv {
		k = strings.TrimSpace(k)
		if k == "" || v == nil {
			continue
		}
		if s := fmt.Sprintf("%v", v); strings.TrimSpace(s) != "" {
			extraEnv[k] = s
		}
	}

	projectName := body.AppID
	if body.InstanceID > 1 {
		projectName = fmt.Sprintf("%s-%d", body.AppID, body.InstanceID)
	}

	if _, err := compose.WriteProjectEnv(appPath, projectName, body.Environment); err != nil {
		jsonErr(w, 500, "ENV_WRITE_FAILED", err.Error())
		return
	}

	modifiedContent, err := compose.BuildProjectComposeContent(string(baseContent), compose.TransformOptions{
		ProjectID: projectName, AppID: body.AppID,
		ExpiresIn: body.ExpiresIn, CustomPortMappings: body.CustomPortMappings,
		ExtraEnv: extraEnv, MasterApp: body.MasterApp,
	})
	if err != nil {
		jsonErr(w, 500, "COMPOSE_BUILD_FAILED", err.Error())
		return
	}

	ref, err := compose.WriteProjectCompose(appPath, projectName, modifiedContent)
	if err != nil {
		jsonErr(w, 500, "COMPOSE_WRITE_FAILED", err.Error())
		return
	}

	cmdName, cmdArgs, err := getComposeCommand()
	if err != nil {
		jsonErr(w, 500, "COMPOSE_NOT_FOUND", err.Error())
		return
	}
	composeEnv, _ := compose.GetComposeProcessEnv(appPath, projectName, docker.SocketPath)
	args := append(cmdArgs, "-p", projectName, "-f", ref.ComposeFile, "up", "-d")
	stdout, stderr, exitCode, _ := spawnExec(cmdName, args, composeEnv, appPath)
	if exitCode != 0 {
		jsonErr(w, 500, "DEPLOYMENT_FAILED", coalesce(stderr, stdout))
		return
	}

	jsonResp(w, 200, map[string]interface{}{
		"success": true,
		"message": fmt.Sprintf("App '%s' deployed successfully", body.AppID),
		"appId": body.AppID, "output": stdout, "warnings": stderr,
		"temporary": body.ExpiresIn > 0,
	})

	telemetry.TrackInstall(body.AppID)
}

// ─── Handlers: containers ─────────────────────────────────────────────────────

func handleContainers(w http.ResponseWriter, r *http.Request) {
	containers, err := docker.Client.ContainerList(context.Background(), dockerctr.ListOptions{All: true})
	if err != nil {
		jsonErr(w, 500, "CONTAINERS_FETCH_FAILED", err.Error())
		return
	}
	catalogMap := getCatalogMap()

	// Find Yantr projects
	yantrProjects := map[string]bool{}
	for _, c := range containers {
		lbl := parseAppLabels(c.Labels)
		if project := c.Labels["com.docker.compose.project"]; lbl.App != "" && project != "" {
			yantrProjects[project] = true
		}
	}

	var result []map[string]interface{}
	for _, c := range containers {
		lbl := parseAppLabels(c.Labels)
		project := c.Labels["com.docker.compose.project"]
		if lbl.App == "" && project != "" && yantrProjects[project] {
			continue
		}

		baseID := getBaseAppID(project)
		appID := coalesce(lbl.App, baseID)
		if appID == "" && len(c.Names) > 0 {
			appID = strings.TrimPrefix(c.Names[0], "/")
		}
		name := ""
		if len(c.Names) > 0 {
			name = strings.TrimPrefix(c.Names[0], "/")
		}
		entry := catalogMap[appID]

		result = append(result, map[string]interface{}{
			"id": c.ID, "name": name, "image": c.Image, "imageId": c.ImageID,
			"state": c.State, "status": c.Status, "created": c.Created,
			"ports": c.Ports, "labels": c.Labels, "appLabels": lbl,
			"app": map[string]interface{}{
				"id": appID, "projectId": coalesce(project, name, "unknown"),
				"service":           coalesce(lbl.Service, name, "unknown"),
				"name":              coalesce(entryStr(entry, "name"), lbl.Service, name),
				"logo":              entryStr(entry, "logo"),
				"tags":              entrySlice(entry, "tags"),
				"ports":             entryPorts(entry),
				"short_description": entryStr(entry, "short_description"),
				"description":       entryStr(entry, "description"),
				"usecases":          entrySlice(entry, "usecases"),
				"website":           entryStr(entry, "website"),
			},
		})
	}
	if result == nil {
		result = []map[string]interface{}{}
	}
	jsonResp(w, 200, map[string]interface{}{"success": true, "count": len(result), "containers": result})
}

func handleContainerDetail(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	info, err := docker.Client.ContainerInspect(context.Background(), id)
	if err != nil {
		jsonErr(w, 404, "CONTAINER_NOT_FOUND", "Container not found")
		return
	}
	lbl := parseAppLabels(info.Config.Labels)
	project := info.Config.Labels["com.docker.compose.project"]
	appID := coalesce(lbl.App, getBaseAppID(project), strings.TrimPrefix(info.Name, "/"))
	entry := getCatalogMap()[appID]
	name := strings.TrimPrefix(info.Name, "/")

	jsonResp(w, 200, map[string]interface{}{
		"success": true,
		"container": map[string]interface{}{
			"id": info.ID, "name": name, "image": info.Config.Image, "imageId": info.Image,
			"state": info.State.Status, "stateDetails": info.State,
			"created": info.Created, "ports": info.NetworkSettings.Ports,
			"mounts": info.Mounts, "env": info.Config.Env, "labels": lbl,
			"expireAt": info.Config.Labels["yantr.expireAt"],
			"app": map[string]interface{}{
				"id": appID, "projectId": coalesce(project, name),
				"service": coalesce(lbl.Service, name), "name": coalesce(entryStr(entry, "name"), lbl.Service, name),
				"logo": entryStr(entry, "logo"), "tags": entrySlice(entry, "tags"),
				"ports": entryPorts(entry), "short_description": entryStr(entry, "short_description"),
				"description": entryStr(entry, "description"), "usecases": entrySlice(entry, "usecases"),
				"website": entryStr(entry, "website"),
			},
		},
	})
}

func handleContainerStats(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	statsResp, err := docker.Client.ContainerStats(context.Background(), id, false)
	if err != nil {
		jsonErr(w, 500, "STATS_FETCH_FAILED", err.Error())
		return
	}
	defer statsResp.Body.Close()
	var stats dockerctr.StatsResponse
	if err := json.NewDecoder(statsResp.Body).Decode(&stats); err != nil {
		jsonErr(w, 500, "STATS_DECODE_FAILED", err.Error())
		return
	}
	cpuDelta := float64(stats.CPUStats.CPUUsage.TotalUsage - stats.PreCPUStats.CPUUsage.TotalUsage)
	sysDelta := float64(stats.CPUStats.SystemUsage - stats.PreCPUStats.SystemUsage)
	cpuPct := 0.0
	if sysDelta > 0 {
		cpuPct = (cpuDelta / sysDelta) * float64(stats.CPUStats.OnlineCPUs) * 100
	}
	rawMem := float64(stats.MemoryStats.Usage)
	limit := float64(stats.MemoryStats.Limit)
	cache := float64(stats.MemoryStats.Stats["inactive_file"])
	if cache == 0 {
		cache = float64(stats.MemoryStats.Stats["cache"])
	}
	memUsage := rawMem - cache
	if memUsage < 0 {
		memUsage = 0
	}
	memPct := 0.0
	if limit > 0 {
		memPct = (memUsage / limit) * 100
	}
	var netRx, netTx float64
	for _, n := range stats.Networks {
		netRx += float64(n.RxBytes)
		netTx += float64(n.TxBytes)
	}
	var blkR, blkW float64
	for _, io := range stats.BlkioStats.IoServiceBytesRecursive {
		switch io.Op {
		case "Read":
			blkR += float64(io.Value)
		case "Write":
			blkW += float64(io.Value)
		}
	}
	jsonResp(w, 200, map[string]interface{}{
		"success": true,
		"stats": map[string]interface{}{
			"cpu":     map[string]interface{}{"percent": fmt.Sprintf("%.2f", cpuPct), "usage": stats.CPUStats.CPUUsage.TotalUsage},
			"memory":  map[string]interface{}{"usage": memUsage, "rawUsage": rawMem, "cache": cache, "limit": limit, "percent": fmt.Sprintf("%.2f", memPct)},
			"network": map[string]interface{}{"rx": netRx, "tx": netTx},
			"blockIO": map[string]interface{}{"read": blkR, "write": blkW},
		},
	})
}

func handleContainerLogs(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	tail := coalesce(r.URL.Query().Get("tail"), "100")
	logsBody, err := docker.Client.ContainerLogs(context.Background(), id, dockerctr.LogsOptions{
		ShowStdout: true, ShowStderr: true, Tail: tail, Timestamps: true,
	})
	if err != nil {
		jsonErr(w, 500, "LOGS_FETCH_FAILED", err.Error())
		return
	}
	defer logsBody.Close()
	buf := make([]byte, 32*1024)
	var raw strings.Builder
	for {
		n, err := logsBody.Read(buf)
		if n > 0 {
			raw.Write(buf[:n])
		}
		if err != nil {
			break
		}
	}
	var lines []string
	for _, line := range strings.Split(raw.String(), "\n") {
		if len(line) > 8 {
			lines = append(lines, line[8:])
		}
	}
	if lines == nil {
		lines = []string{}
	}
	jsonResp(w, 200, map[string]interface{}{"success": true, "logs": lines})
}

func handleContainerDelete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	info, err := docker.Client.ContainerInspect(context.Background(), id)
	if err != nil {
		jsonErr(w, 404, "CONTAINER_NOT_FOUND", "Container not found")
		return
	}
	name := strings.TrimPrefix(info.Name, "/")
	project := info.Config.Labels["com.docker.compose.project"]

	if project != "" {
		baseID := getBaseAppID(project)
		appPath := filepath.Join(apps.GetAppsDir(), baseID)
		ref := compose.GetProjectComposeRef(appPath, project)
		if _, statErr := os.Stat(ref.ComposePath); statErr == nil {
			if cmdName, cmdArgs, err := getComposeCommand(); err == nil {
				env, _ := compose.GetComposeProcessEnv(appPath, project, docker.SocketPath)
				args := append(cmdArgs, "-p", project, "-f", ref.ComposeFile, "down")
				_, _, exitCode, _ := spawnExec(cmdName, args, env, appPath)
				if exitCode == 0 {
					compose.DeleteProjectCompose(appPath, project)
					jsonResp(w, 200, map[string]interface{}{
						"success": true, "message": fmt.Sprintf("App stack '%s' removed successfully", project),
						"container": name, "stackRemoved": true,
					})
					return
				}
			}
		}
	}

	if info.State.Running {
		_ = docker.Client.ContainerStop(context.Background(), id, dockerctr.StopOptions{})
	}
	if err := docker.Client.ContainerRemove(context.Background(), id, dockerctr.RemoveOptions{}); err != nil {
		jsonErr(w, 500, "CONTAINER_REMOVE_FAILED", err.Error())
		return
	}
	jsonResp(w, 200, map[string]interface{}{"success": true, "message": fmt.Sprintf("Container '%s' removed successfully", name)})
}

// ─── Handlers: stacks ─────────────────────────────────────────────────────────

func handleStackDetail(w http.ResponseWriter, r *http.Request) {
	projectID := chi.URLParam(r, "projectId")
	all, err := docker.Client.ContainerList(context.Background(), dockerctr.ListOptions{All: true})
	if err != nil {
		jsonErr(w, 500, "DOCKER_ERROR", err.Error())
		return
	}
	var pcs []dockerctr.Summary
	for _, c := range all {
		if c.Labels["com.docker.compose.project"] == projectID {
			pcs = append(pcs, c)
		}
	}
	if len(pcs) == 0 {
		jsonErr(w, 404, "STACK_NOT_FOUND", "Stack not found or no containers")
		return
	}

	baseID := getBaseAppID(projectID)
	entry := getCatalogMap()[baseID]

	portMap := map[string]map[string]interface{}{}
	for _, c := range pcs {
		svc := coalesce(c.Labels["yantr.service"], strings.TrimPrefix(c.Names[0], "/"), "unknown")
		for _, p := range c.Ports {
			if p.PublicPort == 0 {
				continue
			}
			key := fmt.Sprintf("%d:%d:%s:%s", p.PublicPort, p.PrivatePort, p.Type, svc)
			if _, ok := portMap[key]; !ok {
				portMap[key] = map[string]interface{}{
					"hostPort": p.PublicPort, "containerPort": p.PrivatePort,
					"protocol": p.Type, "service": svc,
				}
			}
		}
	}
	pPorts := make([]map[string]interface{}, 0, len(portMap))
	for _, p := range portMap {
		pPorts = append(pPorts, p)
	}

	var services []map[string]interface{}
	for _, c := range pcs {
		lbl := parseAppLabels(c.Labels)
		info, err := docker.Client.ContainerInspect(context.Background(), c.ID)
		if err != nil {
			continue
		}
		mountMap := map[string]map[string]interface{}{}
		for _, m := range c.Mounts {
			if _, ok := mountMap[m.Destination]; !ok {
				mountMap[m.Destination] = map[string]interface{}{
					"type": m.Type, "source": m.Source, "destination": m.Destination, "mode": m.Mode, "name": m.Name,
				}
			}
		}
		mounts := make([]map[string]interface{}, 0, len(mountMap))
		for _, m := range mountMap {
			mounts = append(mounts, m)
		}
		var networks []map[string]interface{}
		for netName, nc := range info.NetworkSettings.Networks {
			if nc.IPAddress == "" {
				continue
			}
			networks = append(networks, map[string]interface{}{
				"name": netName, "ipAddress": nc.IPAddress, "gateway": nc.Gateway, "aliases": nc.Aliases,
			})
		}
		name := ""
		if len(c.Names) > 0 {
			name = strings.TrimPrefix(c.Names[0], "/")
		}
		services = append(services, map[string]interface{}{
			"id": c.ID, "name": name, "composeService": c.Labels["com.docker.compose.service"],
			"image": c.Image, "state": c.State, "status": c.Status, "created": c.Created,
			"rawPorts": c.Ports, "mounts": mounts, "networks": networks,
			"service": coalesce(c.Labels["yantr.service"], lbl.Service, name),
			"hasYantrLabel": lbl.App != "",
		})
	}

	var caddyProxies []map[string]interface{}
	for _, c := range pcs {
		if c.Labels["yantr.caddy.enabled"] != "true" {
			continue
		}
		sp, _ := strconv.Atoi(c.Labels["yantr.caddy.serve.port"])
		tp, _ := strconv.Atoi(c.Labels["yantr.caddy.target.port"])
		if sp == 0 {
			continue
		}
		caddyProxies = append(caddyProxies, map[string]interface{}{
			"servePort": sp, "targetPort": tp,
			"authEnabled": c.Labels["yantr.caddy.auth.user"] != "",
			"authUser":    c.Labels["yantr.caddy.auth.user"],
			"service":     c.Labels["com.docker.compose.service"],
		})
	}
	if caddyProxies == nil {
		caddyProxies = []map[string]interface{}{}
	}

	var appInfo interface{}
	if entry != nil {
		appInfo = map[string]interface{}{
			"name": entry.Name, "logo": entry.Logo, "tags": entry.Tags,
			"ports": entry.Ports, "short_description": entry.ShortDescription,
			"website": entry.Website, "customapp": entry.CustomApp,
		}
	}

	jsonResp(w, 200, map[string]interface{}{
		"success": true,
		"stack": map[string]interface{}{
			"projectId": projectID, "appId": baseID, "app": appInfo,
			"publishedPorts": pPorts, "services": services, "caddyProxies": caddyProxies,
		},
	})
}

func handleStackPorts(w http.ResponseWriter, r *http.Request) {
	projectID := chi.URLParam(r, "projectId")
	var body struct {
		ServiceName string `json:"serviceName"`
		PortMapping string `json:"portMapping"`
	}
	if !parseJSON(w, r, &body) {
		return
	}
	parsed := compose.ParseDockerPortInput(body.PortMapping)
	if parsed == nil {
		jsonErr(w, 400, "INVALID_PORT_MAPPING", "portMapping must be a valid Docker port format")
		return
	}

	all, _ := docker.Client.ContainerList(context.Background(), dockerctr.ListOptions{All: true})
	var pcs []dockerctr.Summary
	for _, c := range all {
		if c.Labels["com.docker.compose.project"] == projectID {
			pcs = append(pcs, c)
		}
	}
	if len(pcs) == 0 {
		jsonErr(w, 404, "STACK_NOT_FOUND", "Stack not found or no containers")
		return
	}

	svcSet := map[string]bool{}
	for _, c := range pcs {
		if svc := c.Labels["com.docker.compose.service"]; svc != "" {
			svcSet[svc] = true
		}
	}
	svcName := body.ServiceName
	if svcName == "" && len(svcSet) == 1 {
		for k := range svcSet {
			svcName = k
		}
	}
	if svcName == "" {
		jsonErr(w, 400, "SERVICE_NAME_REQUIRED", "serviceName is required for multi-service stacks")
		return
	}

	baseID := getBaseAppID(projectID)
	appPath := filepath.Join(apps.GetAppsDir(), baseID)
	ref := compose.GetProjectComposeRef(appPath, projectID)
	content, err := os.ReadFile(ref.ComposePath)
	if err != nil {
		jsonErr(w, 500, "COMPOSE_READ_FAILED", err.Error())
		return
	}
	doc, err := compose.Parse(string(content))
	if err != nil {
		jsonErr(w, 500, "COMPOSE_PARSE_FAILED", err.Error())
		return
	}
	services, _ := doc["services"].(map[string]interface{})
	svcRaw, ok := services[svcName]
	if !ok {
		jsonErr(w, 400, "SERVICE_NOT_FOUND_IN_COMPOSE", fmt.Sprintf("Service '%s' not found", svcName))
		return
	}
	svc, _ := svcRaw.(map[string]interface{})
	compose.SetServicePortBindings(svc, []compose.PortBinding{{
		HostPort: parsed.HostPort, ContainerPort: parsed.ContainerPort, Protocol: parsed.Protocol,
	}})

	newContent, _ := compose.Stringify(doc)
	newRef, err := compose.WriteProjectCompose(appPath, projectID, newContent)
	if err != nil {
		jsonErr(w, 500, "COMPOSE_WRITE_FAILED", err.Error())
		return
	}
	cmdName, cmdArgs, _ := getComposeCommand()
	env, _ := compose.GetComposeProcessEnv(appPath, projectID, docker.SocketPath)
	args := append(cmdArgs, "-p", projectID, "-f", newRef.ComposeFile, "up", "-d")
	stdout, stderr, exitCode, _ := spawnExec(cmdName, args, env, appPath)
	if exitCode != 0 {
		jsonErr(w, 500, "STACK_REDEPLOY_FAILED", stderr)
		return
	}

	pm := strconv.Itoa(parsed.ContainerPort)
	if parsed.HostPort != nil {
		pm = strconv.Itoa(*parsed.HostPort) + ":" + pm
	}
	if parsed.Protocol != "tcp" {
		pm += "/" + parsed.Protocol
	}
	jsonResp(w, 200, map[string]interface{}{
		"success": true,
		"message": fmt.Sprintf("Opened %s on service '%s'", pm, svcName),
		"output": stdout, "warnings": stderr,
	})
}

// ─── Handlers: images ─────────────────────────────────────────────────────────

func handleImages(w http.ResponseWriter, r *http.Request) {
	images, err := docker.Client.ImageList(context.Background(), dockerimage.ListOptions{})
	if err != nil {
		jsonErr(w, 500, "IMAGES_FETCH_FAILED", err.Error())
		return
	}
	ctrs, _ := docker.Client.ContainerList(context.Background(), dockerctr.ListOptions{All: true})
	usedIDs := map[string]bool{}
	for _, c := range ctrs {
		usedIDs[c.ImageID] = true
	}

	type imgItem struct {
		ID        string   `json:"id"`
		ShortID   string   `json:"shortId"`
		Tags      []string `json:"tags"`
		Created   int64    `json:"created"`
		Size      string   `json:"size"`
		SizeBytes int64    `json:"sizeBytes"`
		IsUsed    bool     `json:"isUsed"`
	}
	var all []imgItem
	for _, img := range images {
		tags := img.RepoTags
		if len(tags) == 0 {
			tags = []string{"<none>:<none>"}
		}
		shortID := img.ID
		if len(shortID) > 19 {
			shortID = shortID[7:19]
		}
		all = append(all, imgItem{
			ID: img.ID, ShortID: shortID, Tags: tags, Created: img.Created,
			Size: fmt.Sprintf("%.2f", float64(img.Size)/(1024*1024)), SizeBytes: img.Size,
			IsUsed: usedIDs[img.ID],
		})
	}
	if all == nil {
		all = []imgItem{}
	}
	var used, unused []imgItem
	var total, unusedSize int64
	for _, img := range all {
		total += img.SizeBytes
		if img.IsUsed {
			used = append(used, img)
		} else {
			unused = append(unused, img)
			unusedSize += img.SizeBytes
		}
	}
	jsonResp(w, 200, map[string]interface{}{
		"success": true, "total": len(all), "used": len(used), "unused": len(unused),
		"totalSize": fmt.Sprintf("%.2f", float64(total)/(1024*1024)),
		"unusedSize": fmt.Sprintf("%.2f", float64(unusedSize)/(1024*1024)),
		"images": all, "usedImages": used, "unusedImages": unused,
	})
}

func handleImageDelete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	info, _, err := docker.Client.ImageInspectWithRaw(context.Background(), id)
	if err != nil {
		jsonErr(w, 404, "IMAGE_NOT_FOUND", "Image not found")
		return
	}
	if _, err := docker.Client.ImageRemove(context.Background(), id, dockerimage.RemoveOptions{}); err != nil {
		jsonErr(w, 500, "IMAGE_REMOVE_FAILED", err.Error())
		return
	}
	jsonResp(w, 200, map[string]interface{}{"success": true, "message": "Image removed successfully", "imageId": id, "tags": info.RepoTags})
}

// ─── Handlers: volumes ────────────────────────────────────────────────────────

func handleVolumes(w http.ResponseWriter, r *http.Request) {
	vols, err := docker.Client.VolumeList(context.Background(), dockervol.ListOptions{})
	if err != nil {
		jsonErr(w, 500, "VOLUMES_FETCH_FAILED", err.Error())
		return
	}
	ctrs, _ := docker.Client.ContainerList(context.Background(), dockerctr.ListOptions{All: true})
	usedVols := map[string]bool{}
	for _, c := range ctrs {
		for _, m := range c.Mounts {
			if m.Type == "volume" {
				usedVols[m.Name] = true
			}
		}
	}

	type volItem struct {
		Name       string            `json:"name"`
		Driver     string            `json:"driver"`
		Mountpoint string            `json:"mountpoint"`
		CreatedAt  string            `json:"createdAt"`
		Labels     map[string]string `json:"labels"`
		IsBrowsing bool              `json:"isBrowsing"`
		IsUsed     bool              `json:"isUsed"`
		Size       string            `json:"size"`
		SizeBytes  int64             `json:"sizeBytes"`
	}

	var enriched []volItem
	for _, v := range vols.Volumes {
		if v.Labels == nil {
			v.Labels = map[string]string{}
		}
		enriched = append(enriched, volItem{
			Name: v.Name, Driver: v.Driver, Mountpoint: v.Mountpoint, CreatedAt: v.CreatedAt,
			Labels: v.Labels, IsBrowsing: browserRegistry.IsBrowsing(v.Name), IsUsed: usedVols[v.Name],
		})
	}
	if enriched == nil {
		enriched = []volItem{}
	}
	var used, unused []volItem
	for _, v := range enriched {
		if v.IsUsed {
			used = append(used, v)
		} else {
			unused = append(unused, v)
		}
	}
	jsonResp(w, 200, map[string]interface{}{
		"success": true, "total": len(enriched), "used": len(used), "unused": len(unused),
		"totalSize": "0.00", "unusedSize": "0.00",
		"volumes": enriched, "usedVolumes": used, "unusedVolumes": unused,
	})
}

func handleVolumeDelete(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	if err := docker.Client.VolumeRemove(context.Background(), name, false); err != nil {
		if strings.Contains(err.Error(), "in use") {
			jsonErr(w, 409, "VOLUME_IN_USE", fmt.Sprintf("Volume '%s' is currently in use", name))
			return
		}
		jsonErr(w, 500, "VOLUME_REMOVE_FAILED", err.Error())
		return
	}
	jsonResp(w, 200, map[string]interface{}{"success": true, "message": fmt.Sprintf("Volume '%s' removed", name), "volume": name})
}

func handleVolumeBrowserList(w http.ResponseWriter, r *http.Request) {
	jsonResp(w, 200, browserRegistry.List())
}

func handleVolumeBrowseStart(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	var body struct {
		ExpiryMinutes int `json:"expiryMinutes"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body) // optional

	vols, _ := docker.Client.VolumeList(context.Background(), dockervol.ListOptions{})
	found := false
	for _, v := range vols.Volumes {
		if v.Name == name {
			found = true
			break
		}
	}
	if !found {
		jsonErr(w, 404, "VOLUME_NOT_FOUND", "Volume not found")
		return
	}
	p, err := browserRegistry.Start(name, body.ExpiryMinutes)
	if err != nil {
		jsonErr(w, 500, "VOLUME_BROWSER_START_FAILED", err.Error())
		return
	}
	jsonResp(w, 200, map[string]interface{}{"success": true, "port": p, "message": "Volume browser started"})
}

func handleVolumeBrowseStop(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	if !browserRegistry.Stop(name) {
		jsonErr(w, 404, "VOLUME_BROWSER_NOT_FOUND", "No active browser for this volume")
		return
	}
	jsonResp(w, 200, map[string]interface{}{"success": true, "message": "Volume browser stopped"})
}

// ─── Handlers: system ─────────────────────────────────────────────────────────

func handleSystemInfo(w http.ResponseWriter, r *http.Request) {
	info, err := docker.Client.Info(context.Background())
	if err != nil {
		jsonErr(w, 500, "SYSTEM_INFO_FETCH_FAILED", err.Error())
		return
	}
	jsonResp(w, 200, map[string]interface{}{
		"success": true,
		"info": map[string]interface{}{
			"cpu": map[string]interface{}{"cores": info.NCPU},
			"memory": map[string]interface{}{"total": info.MemTotal},
			"storage": map[string]interface{}{"driver": info.Driver},
			"docker": map[string]interface{}{
				"version": info.ServerVersion,
				"containers": map[string]interface{}{
					"total": info.Containers, "running": info.ContainersRunning,
					"paused": info.ContainersPaused, "stopped": info.ContainersStopped,
				},
				"images": info.Images,
			},
			"os": map[string]interface{}{
				"type": info.OSType, "name": info.OperatingSystem,
				"arch": info.Architecture, "kernel": info.KernelVersion,
			},
			"name": info.Name,
		},
	})
}

func handleSystemPrune(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Images  bool `json:"images"`
		Volumes bool `json:"volumes"`
	}
	if !parseJSON(w, r, &body) {
		return
	}
	if !body.Images && !body.Volumes {
		jsonErr(w, 400, "PRUNE_TARGET_REQUIRED", "At least one prune target must be selected")
		return
	}
	results := map[string]interface{}{
		"images":  map[string]interface{}{"count": 0, "spaceReclaimed": 0},
		"volumes": map[string]interface{}{"count": 0, "spaceReclaimed": 0},
	}
	if body.Images {
		filters := dockerfilters.NewArgs()
		filters.Add("dangling", "false") // removes all unused images, not just dangling
		if pruned, err := docker.Client.ImagesPrune(context.Background(), filters); err == nil {
			results["images"] = map[string]interface{}{
				"count": len(pruned.ImagesDeleted), "spaceReclaimed": pruned.SpaceReclaimed,
			}
		} else {
			shared.Log("warn", "system: images prune failed: "+err.Error())
		}
	}
	if body.Volumes {
		if pruned, err := docker.Client.VolumesPrune(context.Background(), dockerfilters.NewArgs()); err == nil {
			results["volumes"] = map[string]interface{}{
				"count": len(pruned.VolumesDeleted), "spaceReclaimed": pruned.SpaceReclaimed,
			}
		} else {
			shared.Log("warn", "system: volumes prune failed: "+err.Error())
		}
	}
	jsonResp(w, 200, map[string]interface{}{"success": true, "results": results})
}

func handlePortsUsed(w http.ResponseWriter, r *http.Request) {
	ctrs, err := docker.Client.ContainerList(context.Background(), dockerctr.ListOptions{})
	if err != nil {
		jsonErr(w, 500, "USED_PORTS_FETCH_FAILED", err.Error())
		return
	}
	portSet := map[int]bool{}
	for _, c := range ctrs {
		for _, p := range c.Ports {
			if p.PublicPort > 0 {
				portSet[int(p.PublicPort)] = true
			}
		}
	}
	ports := make([]int, 0, len(portSet))
	for p := range portSet {
		ports = append(ports, p)
	}
	jsonResp(w, 200, map[string]interface{}{"success": true, "count": len(ports), "ports": ports})
}

func handlePortsSuggest(w http.ResponseWriter, r *http.Request) {
	var body struct {
		AppID string `json:"appId"`
		Ports []struct{ IsNamed bool `json:"isNamed"` } `json:"ports"`
	}
	if !parseJSON(w, r, &body) {
		return
	}
	ctrs, _ := docker.Client.ContainerList(context.Background(), dockerctr.ListOptions{})
	used := map[int]bool{}
	for _, c := range ctrs {
		for _, p := range c.Ports {
			if p.PublicPort > 0 {
				used[int(p.PublicPort)] = true
			}
		}
	}
	cur := 5255
	type sug struct {
		IsNamed       bool `json:"isNamed"`
		SuggestedPort int  `json:"suggestedPort"`
		IsOriginal    bool `json:"isOriginal"`
	}
	var suggestions []sug
	for _, p := range body.Ports {
		if !p.IsNamed {
			suggestions = append(suggestions, sug{IsNamed: false, IsOriginal: true})
			continue
		}
		for used[cur] {
			cur++
		}
		suggestions = append(suggestions, sug{IsNamed: true, SuggestedPort: cur, IsOriginal: false})
		used[cur] = true
		cur++
	}
	if suggestions == nil {
		suggestions = []sug{}
	}
	jsonResp(w, 200, map[string]interface{}{"success": true, "appId": body.AppID, "suggestions": suggestions})
}

func handleNetworkIdentity(w http.ResponseWriter, r *http.Request) {
	force := r.URL.Query().Get("force") == "true"
	identity, err := system.GetPublicIPIdentityCached(force)
	if err != nil {
		jsonErr(w, 500, "IDENTITY_FETCH_FAILED", err.Error())
		return
	}
	jsonResp(w, 200, map[string]interface{}{"success": true, "identity": identity})
}

// ─── Handlers: proxy ──────────────────────────────────────────────────────────

func handleProxyList(w http.ResponseWriter, r *http.Request) {
	proxies, _ := caddy.GetCaddyProxies()
	if proxies == nil {
		proxies = []caddy.ProxyRoute{}
	}
	jsonResp(w, 200, map[string]interface{}{"success": true, "proxies": proxies, "caddyRunning": caddy.IsRunning()})
}

func handleProxyReload(w http.ResponseWriter, r *http.Request) {
	if !caddy.IsRunning() {
		_ = caddy.StartCaddy()
	}
	_ = caddy.ReloadCaddyConfig()
	jsonResp(w, 200, map[string]interface{}{"success": true, "caddyRunning": caddy.IsRunning()})
}

func handleProxyEnable(w http.ResponseWriter, r *http.Request) {
	var body struct {
		ProjectID   string `json:"projectId"`
		ServiceName string `json:"serviceName"`
		ServePort   int    `json:"servePort"`
		TargetPort  int    `json:"targetPort"`
		AuthUser    string `json:"authUser"`
		AuthPass    string `json:"authPass"`
	}
	if !parseJSON(w, r, &body) {
		return
	}
	if (body.AuthUser != "") != (body.AuthPass != "") {
		jsonErr(w, 400, "PROXY_AUTH_FIELDS_MISMATCH", "Both authUser and authPass must be provided together")
		return
	}

	baseID := getBaseAppID(body.ProjectID)
	appPath := filepath.Join(apps.GetAppsDir(), baseID)
	ref := compose.GetProjectComposeRef(appPath, body.ProjectID)
	content, err := os.ReadFile(ref.ComposePath)
	if err != nil {
		jsonErr(w, 500, "COMPOSE_READ_FAILED", err.Error())
		return
	}
	doc, err := compose.Parse(string(content))
	if err != nil {
		jsonErr(w, 500, "COMPOSE_PARSE_FAILED", err.Error())
		return
	}

	var passHash string
	if body.AuthUser != "" {
		h, err := caddy.HashPassword(body.AuthPass)
		if err != nil {
			jsonErr(w, 500, "PROXY_PASSWORD_HASH_FAILED", err.Error())
			return
		}
		passHash = caddy.EncodeHash(h)
	}

	services, _ := doc["services"].(map[string]interface{})
	svcName := body.ServiceName
	if svcName == "" {
		for k := range services {
			svcName = k
			break
		}
	}
	svc, _ := services[svcName].(map[string]interface{})
	if svc == nil {
		jsonErr(w, 400, "SERVICE_NOT_FOUND_IN_COMPOSE", fmt.Sprintf("Service '%s' not found", svcName))
		return
	}

	labels, _ := svc["labels"].(map[string]interface{})
	if labels == nil {
		labels = map[string]interface{}{}
		svc["labels"] = labels
	}
	labels["yantr.caddy.enabled"] = "true"
	labels["yantr.caddy.serve.port"] = strconv.Itoa(body.ServePort)
	labels["yantr.caddy.target.port"] = strconv.Itoa(body.TargetPort)
	if passHash != "" {
		labels["yantr.caddy.auth.user"] = body.AuthUser
		labels["yantr.caddy.auth.pass"] = passHash
	} else {
		delete(labels, "yantr.caddy.auth.user")
		delete(labels, "yantr.caddy.auth.pass")
	}

	newContent, _ := compose.Stringify(doc)
	newRef, err := compose.WriteProjectCompose(appPath, body.ProjectID, newContent)
	if err != nil {
		jsonErr(w, 500, "COMPOSE_WRITE_FAILED", err.Error())
		return
	}
	cmdName, cmdArgs, _ := getComposeCommand()
	env, _ := compose.GetComposeProcessEnv(appPath, body.ProjectID, docker.SocketPath)
	args := append(cmdArgs, "-p", body.ProjectID, "-f", newRef.ComposeFile, "up", "-d")
	_, stderr, exitCode, _ := spawnExec(cmdName, args, env, appPath)
	if exitCode != 0 {
		jsonErr(w, 500, "PROXY_REDEPLOY_FAILED", stderr)
		return
	}
	if !caddy.IsRunning() {
		_ = caddy.StartCaddy()
	} else {
		_ = caddy.ReloadCaddyConfig()
	}
	jsonResp(w, 200, map[string]interface{}{
		"success": true,
		"message": fmt.Sprintf("Caddy proxy enabled: :%d → localhost:%d", body.ServePort, body.TargetPort),
	})
}

func handleProxyDisable(w http.ResponseWriter, r *http.Request) {
	var body struct {
		ProjectID   string `json:"projectId"`
		ServiceName string `json:"serviceName"`
	}
	if !parseJSON(w, r, &body) {
		return
	}
	baseID := getBaseAppID(body.ProjectID)
	appPath := filepath.Join(apps.GetAppsDir(), baseID)
	ref := compose.GetProjectComposeRef(appPath, body.ProjectID)
	content, _ := os.ReadFile(ref.ComposePath)
	doc, _ := compose.Parse(string(content))
	if doc != nil {
		services, _ := doc["services"].(map[string]interface{})
		svcName := body.ServiceName
		if svcName == "" {
			for k := range services {
				svcName = k
				break
			}
		}
		if svc, ok := services[svcName].(map[string]interface{}); ok {
			if labels, ok := svc["labels"].(map[string]interface{}); ok {
				delete(labels, "yantr.caddy.enabled")
				delete(labels, "yantr.caddy.serve.port")
				delete(labels, "yantr.caddy.target.port")
				delete(labels, "yantr.caddy.auth.user")
				delete(labels, "yantr.caddy.auth.pass")
			}
		}
		newContent, _ := compose.Stringify(doc)
		newRef, _ := compose.WriteProjectCompose(appPath, body.ProjectID, newContent)
		cmdName, cmdArgs, _ := getComposeCommand()
		env, _ := compose.GetComposeProcessEnv(appPath, body.ProjectID, docker.SocketPath)
		args := append(cmdArgs, "-p", body.ProjectID, "-f", newRef.ComposeFile, "up", "-d")
		spawnExec(cmdName, args, env, appPath)
	}
	_ = caddy.ReloadCaddyConfig()
	jsonResp(w, 200, map[string]interface{}{
		"success": true, "message": fmt.Sprintf("Caddy proxy disabled for '%s'", body.ProjectID),
	})
}

// ─── Handlers: autoupdate ─────────────────────────────────────────────────────

func runWatchtower(containerNames []string) (string, string, int, error) {
	args := []string{
		"run", "--rm",
		"-v", docker.SocketPath + ":/var/run/docker.sock",
		"-e", "DOCKER_API_VERSION=1.44",
		"containrrr/watchtower",
		"--run-once", "--cleanup",
	}
	args = append(args, containerNames...)
	env := map[string]string{
		"DOCKER_HOST":        "unix://" + docker.SocketPath,
		"DOCKER_API_VERSION": "1.44",
	}
	return spawnExec("docker", args, env, "")
}

func handleAutoupdateRun(w http.ResponseWriter, r *http.Request) {
	var body struct {
		ContainerIDs []string `json:"containerIds"`
	}
	if !parseJSON(w, r, &body) {
		return
	}
	ctrs, _ := docker.Client.ContainerList(context.Background(), dockerctr.ListOptions{})
	idSet := map[string]bool{}
	for _, id := range body.ContainerIDs {
		idSet[id] = true
	}
	var names []string
	for _, c := range ctrs {
		match := idSet[c.ID]
		if !match {
			for _, id := range body.ContainerIDs {
				if strings.HasPrefix(c.ID, id) {
					match = true
					break
				}
			}
		}
		if match && len(c.Names) > 0 {
			names = append(names, strings.TrimPrefix(c.Names[0], "/"))
		}
	}
	if len(names) == 0 {
		jsonErr(w, 404, "CONTAINERS_NOT_RUNNING", "None of the provided container IDs are currently running")
		return
	}
	stdout, stderr, exitCode, err := runWatchtower(names)
	if err != nil {
		jsonErr(w, 500, "AUTOUPDATE_FAILED", err.Error())
		return
	}
	jsonResp(w, 200, map[string]interface{}{"success": true, "exitCode": exitCode, "output": stdout, "warnings": stderr})

	// Very simple heuristic to detect if updates were found
	if strings.Contains(stdout, "Found new") || strings.Contains(stdout, "updating") || strings.Contains(stdout, "updated") {
		telemetry.TrackUpdatesForContainers(names)
	}
}

func handleAutoupdateSelf(w http.ResponseWriter, r *http.Request) {
	name := coalesce(os.Getenv("YANTR_CONTAINER_NAME"), "yantr")
	stdout, stderr, exitCode, err := runWatchtower([]string{name})
	if err != nil {
		jsonErr(w, 500, "SELF_UPDATE_FAILED", err.Error())
		return
	}
	jsonResp(w, 200, map[string]interface{}{"success": true, "exitCode": exitCode, "output": stdout, "warnings": stderr})
	
	if strings.Contains(stdout, "Found new") || strings.Contains(stdout, "updating") || strings.Contains(stdout, "updated") {
		telemetry.TrackSelfUpdate(1, version)
	}
}

// ─── Main ─────────────────────────────────────────────────────────────────────

func main() {
	// Bootstrap self-install if needed
	if bootstrapped, err := selfinstall.RunIfNeeded(); err != nil {
		fmt.Fprintln(os.Stderr, "selfinstall error:", err)
		os.Exit(1)
	} else if bootstrapped {
		os.Exit(0)
	}

	// Configure apps directory
	appsPath := getAppsDir()
	apps.SetAppsDir(appsPath)
	shared.Log("info", "📂 Apps directory: "+appsPath)
	shared.Log("info", fmt.Sprintf("🏗️  Architecture: %s/%s", runtime.GOOS, runtime.GOARCH))

	telemetry.StartPresenceScheduler(version)

	// Router
	r := chi.NewRouter()
	r.Use(middleware.Recoverer)
	r.Use(corsMiddleware)
	r.Use(authMiddleware)

	// Browse proxy
	r.HandleFunc("/browse/*", browseProxyHandler)

	// Public routes
	r.Get("/api/health", handleHealth)
	r.Get("/api/version", handleVersion)
	r.Get("/api/setup/status", handleSetupStatus)
	r.Post("/api/setup/admin", handleSetupAdmin)
	r.Post("/api/auth/login", handleLogin)

	// Protected routes
	r.Get("/api/logs", handleLogs)
	r.Get("/api/apps", handleApps)
	r.Get("/api/apps/{id}/check-arch", handleCheckArch)
	r.Post("/api/deploy", handleDeploy)
	r.Get("/api/containers", handleContainers)
	r.Get("/api/containers/{id}", handleContainerDetail)
	r.Get("/api/containers/{id}/stats", handleContainerStats)
	r.Get("/api/containers/{id}/logs", handleContainerLogs)
	r.Delete("/api/containers/{id}", handleContainerDelete)
	r.Get("/api/stacks/{projectId}", handleStackDetail)
	r.Post("/api/stacks/{projectId}/ports", handleStackPorts)
	r.Get("/api/images", handleImages)
	r.Delete("/api/images/{id}", handleImageDelete)
	r.Get("/api/volumes", handleVolumes)
	r.Get("/api/volumes/browsers", handleVolumeBrowserList)
	r.Post("/api/volumes/{name}/browse", handleVolumeBrowseStart)
	r.Delete("/api/volumes/{name}/browse", handleVolumeBrowseStop)
	r.Delete("/api/volumes/{name}", handleVolumeDelete)
	r.Get("/api/system/info", handleSystemInfo)
	r.Post("/api/system/prune", handleSystemPrune)
	r.Get("/api/ports/used", handlePortsUsed)
	r.Post("/api/ports/suggest", handlePortsSuggest)
	r.Get("/api/network/identity", handleNetworkIdentity)
	r.Get("/api/proxy", handleProxyList)
	r.Post("/api/proxy/reload", handleProxyReload)
	r.Post("/api/proxy/enable", handleProxyEnable)
	r.Post("/api/proxy/disable", handleProxyDisable)
	r.Post("/api/autoupdate/run", handleAutoupdateRun)
	r.Post("/api/autoupdate/self", handleAutoupdateSelf)

	// SPA static serving (production)
	distDir := getDistDir()
	uiBase := coalesce(os.Getenv("UI_BASE_PATH"), "/")
	if _, err := os.Stat(distDir); err == nil {
		shared.Log("info", "📦 Serving Vue.js UI from: "+distDir)
		fs := http.FileServer(http.Dir(distDir))
		prefix := strings.TrimSuffix(uiBase, "/")
		r.Handle(prefix+"/*", http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			path := strings.TrimPrefix(req.URL.Path, prefix)
			if path == "" {
				path = "/"
			}
			if _, err := os.Stat(filepath.Join(distDir, path)); os.IsNotExist(err) {
				http.ServeFile(w, req, filepath.Join(distDir, "index.html"))
				return
			}
			http.StripPrefix(prefix, fs).ServeHTTP(w, req)
		}))
		r.Get("/", func(w http.ResponseWriter, req *http.Request) {
			http.ServeFile(w, req, filepath.Join(distDir, "index.html"))
		})
	}

	// Start Caddy in background
	go func() {
		shared.Log("info", "🔒 Starting embedded Caddy proxy")
		if err := caddy.StartCaddy(); err != nil {
			shared.Log("warn", "⚠️  [CADDY] "+err.Error())
		}
	}()

	// Server startup log
	shared.Log("info", strings.Repeat("=", 50))
	shared.Log("info", "🚀 Yantr Core Server Started (Go)")
	shared.Log("info", strings.Repeat("=", 50))
	shared.Log("info", fmt.Sprintf("📡 Port: %d", serverPort))
	shared.Log("info", fmt.Sprintf("🔌 Socket: %s", docker.SocketPath))
	shared.Log("info", fmt.Sprintf("🌐 Access: http://localhost:%d", serverPort))
	shared.Log("info", strings.Repeat("=", 50))

	srv := &http.Server{
		Addr:         fmt.Sprintf("0.0.0.0:%d", serverPort),
		Handler:      r,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 120 * time.Second,
	}

	// Graceful shutdown
	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGTERM, syscall.SIGINT)
		sig := <-sigCh
		shared.Log("info", fmt.Sprintf("Received %s, shutting down...", sig))
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = srv.Shutdown(ctx)
		caddy.StopCaddy()
		browserRegistry.StopAll()
		os.Exit(0)
	}()

	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		fmt.Fprintln(os.Stderr, "Server error:", err)
		os.Exit(1)
	}
}
