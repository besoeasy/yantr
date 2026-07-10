package main

import (
	"context"
	"core/apps"
	"core/auth"
	"core/caddy"
	"core/compose"
	"core/docker"
	"core/shared"
	"core/system"
	"core/telemetry"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	dockerctr "github.com/docker/docker/api/types/container"
	dockerfilters "github.com/docker/docker/api/types/filters"
)

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
	wctx, wcancel := context.WithTimeout(context.Background(), spawnTimeoutMedium)
	defer wcancel()
	return spawnExec(wctx, "docker", args, env, "")
}

func sweepExpiredContainers() {
	all, err := docker.ContainerList(context.Background(), dockerctr.ListOptions{All: false})
	if err != nil {
		shared.Log("warn", "[reaper] failed to list containers: "+err.Error())
		return
	}

	now := time.Now().Unix()

	// Collect expired projects (deduplicated) and standalone containers.
	type projectMeta struct{ appID, project string }
	expiredProjects := map[string]projectMeta{}
	var standaloneIDs []string

	for _, c := range all {
		expireAtStr, ok := c.Labels["yantr.expireAt"]
		if !ok {
			continue
		}
		expireAt, err := strconv.ParseInt(expireAtStr, 10, 64)
		if err != nil || expireAt <= 0 || now < expireAt {
			continue // not expired yet
		}
		project := c.Labels["com.docker.compose.project"]
		if project != "" {
			if _, seen := expiredProjects[project]; !seen {
				expiredProjects[project] = projectMeta{
					appID:   getBaseAppID(project),
					project: project,
				}
			}
		} else {
			standaloneIDs = append(standaloneIDs, c.ID)
		}
	}

	// Tear down expired Compose stacks.
	for projectID, meta := range expiredProjects {
		shared.Log("info", fmt.Sprintf("[reaper] removing expired stack: %s", projectID))
		appPath := filepath.Join(apps.GetAppsDir(), meta.appID)
		ref := compose.GetProjectComposeRef(appPath, projectID)
		removed := false
		if _, statErr := os.Stat(ref.ComposePath); statErr == nil {
			if cmdName, cmdArgs, cmdErr := getComposeCommand(); cmdErr == nil {
				env, _ := compose.GetComposeProcessEnv(appPath, projectID, docker.SocketPath)
				args := append(cmdArgs, "-p", projectID, "-f", ref.ComposeFile, "down")
				reaperCtx, reaperCancel := context.WithTimeout(context.Background(), spawnTimeoutMedium)
				_, _, exitCode, _ := spawnExec(reaperCtx, cmdName, args, env, appPath)
				reaperCancel()
				if exitCode == 0 {
					compose.DeleteProjectCompose(appPath, projectID)
					shared.Log("info", fmt.Sprintf("[reaper] stack %s removed", projectID))
					removed = true
				}
			}
		}
		if !removed {
			// Fallback: force-remove every container in the project.
			shared.Log("warn", fmt.Sprintf("[reaper] compose down failed for %s — force-removing containers", projectID))
			if stale, listErr := docker.ContainerList(context.Background(), dockerctr.ListOptions{All: true}); listErr == nil {
				for _, c := range stale {
					if c.Labels["com.docker.compose.project"] == projectID {
						_ = docker.ContainerStop(context.Background(), c.ID, dockerctr.StopOptions{})
						_ = docker.ContainerRemove(context.Background(), c.ID, dockerctr.RemoveOptions{})
					}
				}
			}
			compose.DeleteProjectCompose(appPath, projectID)
		}
	}

	// Tear down standalone expired containers.
	for _, id := range standaloneIDs {
		shared.Log("info", fmt.Sprintf("[reaper] removing expired standalone container: %s", id))
		_ = docker.ContainerStop(context.Background(), id, dockerctr.StopOptions{})
		_ = docker.ContainerRemove(context.Background(), id, dockerctr.RemoveOptions{})
	}
}

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
	var body struct {
		PublicKeyHex string `json:"publicKeyHex"`
	}
	if !parseJSON(w, r, &body) {
		return
	}
	_, err := auth.SaveAuthConfig(body.PublicKeyHex)
	if err != nil {
		if errors.Is(err, auth.ErrAlreadyConfigured) {
			jsonErr(w, 409, "SETUP_ALREADY_CONFIGURED", "Yantr is already configured")
			return
		}
		jsonErr(w, 400, "INVALID_SETUP_ADMIN_REQUEST", err.Error())
		return
	}
	jsonResp(w, 201, map[string]interface{}{"success": true, "configured": true})
}

func handleLogin(w http.ResponseWriter, r *http.Request) {
	cfg, _ := auth.LoadAuthConfig(false)
	if cfg == nil {
		jsonErr(w, 409, "SETUP_REQUIRED", "Setup required")
		return
	}
	token := auth.ExtractBearerToken(r.Header.Get("Authorization"))
	err := auth.VerifyToken(token, cfg)
	if err != nil {
		jsonErr(w, 401, "UNAUTHORIZED", "Unauthorized")
		return
	}
	jsonResp(w, 200, map[string]interface{}{
		"success": true, "authenticated": true,
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

func handleSystemInfo(w http.ResponseWriter, r *http.Request) {
	info, err := docker.Info(context.Background())
	if err != nil {
		jsonErr(w, 500, "SYSTEM_INFO_FETCH_FAILED", err.Error())
		return
	}
	jsonResp(w, 200, map[string]interface{}{
		"success": true,
		"info": map[string]interface{}{
			"cpu":     map[string]interface{}{"cores": info.NCPU},
			"memory":  map[string]interface{}{"total": info.MemTotal},
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
		// dangling=true: only remove untagged layers, not ALL unused images.
		// This is the safe default — avoids deleting images for stopped containers.
		filters := dockerfilters.NewArgs()
		filters.Add("dangling", "true")
		if pruned, err := docker.ImagesPrune(context.Background(), filters); err == nil {
			shared.Log("info", fmt.Sprintf("[prune] images: removed=%d reclaimed=%d bytes", len(pruned.ImagesDeleted), pruned.SpaceReclaimed))
			results["images"] = map[string]interface{}{
				"count": len(pruned.ImagesDeleted), "spaceReclaimed": pruned.SpaceReclaimed,
			}
		} else {
			shared.Log("warn", "[prune] images failed: "+err.Error())
		}
	}
	if body.Volumes {
		if pruned, err := docker.VolumesPrune(context.Background(), dockerfilters.NewArgs()); err == nil {
			shared.Log("info", fmt.Sprintf("[prune] volumes: removed=%d reclaimed=%d bytes", len(pruned.VolumesDeleted), pruned.SpaceReclaimed))
			results["volumes"] = map[string]interface{}{
				"count": len(pruned.VolumesDeleted), "spaceReclaimed": pruned.SpaceReclaimed,
			}
		} else {
			shared.Log("warn", "[prune] volumes failed: "+err.Error())
		}
	}
	jsonResp(w, 200, map[string]interface{}{"success": true, "results": results})
}

func handlePortsUsed(w http.ResponseWriter, r *http.Request) {
	ctrs, err := docker.ContainerList(context.Background(), dockerctr.ListOptions{})
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
		Ports []struct {
			IsNamed bool `json:"isNamed"`
		} `json:"ports"`
	}
	if !parseJSON(w, r, &body) {
		return
	}
	ctrs, _ := docker.ContainerList(context.Background(), dockerctr.ListOptions{})
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

func handleAutoupdateRun(w http.ResponseWriter, r *http.Request) {
	var body struct {
		ContainerIDs []string `json:"containerIds"`
	}
	if !parseJSON(w, r, &body) {
		return
	}
	ctrs, _ := docker.ContainerList(context.Background(), dockerctr.ListOptions{})
	idSet := map[string]bool{}
	for _, id := range body.ContainerIDs {
		idSet[id] = true
	}

	var watchtowerNames []string
	projectSet := map[string]bool{}

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
		if match {
			project := c.Labels["com.docker.compose.project"]
			if project != "" {
				projectSet[project] = true
			} else if len(c.Names) > 0 {
				watchtowerNames = append(watchtowerNames, strings.TrimPrefix(c.Names[0], "/"))
			}
		}
	}

	if len(projectSet) == 0 && len(watchtowerNames) == 0 {
		jsonErr(w, 404, "CONTAINERS_NOT_RUNNING", "None of the provided container IDs are currently running")
		return
	}

	// Load catalog once to check customapp flag per project.
	catalog, _ := apps.GetCatalogCached(false)
	isCustomApp := func(baseID string) bool {
		if catalog == nil {
			return false
		}
		for i := range catalog.Apps {
			if catalog.Apps[i].ID == baseID {
				return catalog.Apps[i].CustomApp
			}
		}
		return false
	}

	var allStdout, allStderr strings.Builder
	updatedCount := 0

	// 1. Process Compose Projects natively
	cmdName, cmdArgs, cmdErr := getComposeCommand()
	for projectID := range projectSet {
		if cmdErr != nil {
			allStderr.WriteString(fmt.Sprintf("[update] docker compose not available for %s: %v\n", projectID, cmdErr))
			continue
		}
		baseID := getBaseAppID(projectID)

		// Guard: never auto-update customapp stacks (locally built images have no registry).
		if isCustomApp(baseID) {
			shared.Log("warn", fmt.Sprintf("[update] skipping customapp stack %s — auto-update disabled", projectID))
			continue
		}

		appPath := filepath.Join(apps.GetAppsDir(), baseID)
		ref := compose.GetProjectComposeRef(appPath, projectID)

		if _, statErr := os.Stat(ref.ComposePath); statErr != nil {
			allStderr.WriteString(fmt.Sprintf("[update] compose.yml not found for %s\n", projectID))
			continue
		}

		env, _ := compose.GetComposeProcessEnv(appPath, projectID, docker.SocketPath)
		shared.Log("info", fmt.Sprintf("[update] pulling latest images for stack: %s", projectID))
		pullCtx, pullCancel := context.WithTimeout(context.Background(), spawnTimeoutLong)
		pullArgs := append(cmdArgs, "-p", projectID, "-f", ref.ComposeFile, "pull")
		outPull, errPull, exitPull, _ := spawnExec(pullCtx, cmdName, pullArgs, env, appPath)
		pullCancel()

		allStdout.WriteString(outPull + "\n")
		allStderr.WriteString(errPull + "\n")

		if exitPull != 0 {
			shared.Log("error", fmt.Sprintf("[update] pull failed for %s (exit=%d)", projectID, exitPull))
			continue
		}

		// Detect whether any image was actually newer before recreating.
		// We only check pull output — this avoids relying on locale-sensitive
		// strings from 'up' which vary across Docker/Compose versions.
		pullCombined := strings.ToLower(outPull + "\n" + errPull)
		newerImage := strings.Contains(pullCombined, "downloaded newer image") ||
			strings.Contains(pullCombined, "pull complete") ||
			strings.Contains(pullCombined, "digest:")

		shared.Log("info", fmt.Sprintf("[update] recreating stack: %s", projectID))
		upCtx, upCancel := context.WithTimeout(context.Background(), spawnTimeoutLong)
		upArgs := append(cmdArgs, "-p", projectID, "-f", ref.ComposeFile, "up", "-d")
		outUp, errUp, exitUp, _ := spawnExec(upCtx, cmdName, upArgs, env, appPath)
		upCancel()

		allStdout.WriteString(outUp + "\n")
		allStderr.WriteString(errUp + "\n")

		if exitUp != 0 {
			shared.Log("error", fmt.Sprintf("[update] 'up -d' failed for %s (exit=%d)", projectID, exitUp))
			continue
		}

		// Count as updated only when a newer image was pulled AND the stack came up cleanly.
		if newerImage {
			updatedCount++
			shared.Log("info", fmt.Sprintf("[update] stack %s was updated", projectID))
			telemetry.TrackUpdatesForContainers([]string{projectID})
		} else {
			shared.Log("info", fmt.Sprintf("[update] stack %s is already up to date", projectID))
		}
	}

	// 2. Process standalone containers using Watchtower
	if len(watchtowerNames) > 0 {
		shared.Log("info", fmt.Sprintf("[update] running watchtower for standalone containers: %s", strings.Join(watchtowerNames, ", ")))
		wOut, wErr, wExit, err := runWatchtower(watchtowerNames)
		if err != nil {
			allStderr.WriteString(fmt.Sprintf("[update] watchtower error: %v\n", err))
		} else {
			allStdout.WriteString(wOut + "\n")
			allStderr.WriteString(wErr + "\n")

			// Count only the containers that Watchtower actually mentions by name as updated.
			// This avoids overcounting when only a subset of the requested containers had updates.
			wCombined := strings.ToLower(wOut + "\n" + wErr)
			var updatedNames []string
			for _, name := range watchtowerNames {
				if strings.Contains(wCombined, strings.ToLower(name)) &&
					(strings.Contains(wCombined, "found new") || strings.Contains(wCombined, "updating") || strings.Contains(wCombined, "updated")) {
					updatedNames = append(updatedNames, name)
				}
			}
			if len(updatedNames) > 0 {
				updatedCount += len(updatedNames)
				shared.Log("info", fmt.Sprintf("[update] images updated for: %s", strings.Join(updatedNames, ", ")))
				telemetry.TrackUpdatesForContainers(updatedNames)
			} else {
				shared.Log("info", fmt.Sprintf("[update] no updates found for: %s (exit=%d)", strings.Join(watchtowerNames, ", "), wExit))
			}
		}
	}

	// Dump logs cleanly
	for _, line := range strings.Split(strings.TrimSpace(allStdout.String()), "\n") {
		if line != "" {
			shared.Log("info", "[update] "+line)
		}
	}
	for _, line := range strings.Split(strings.TrimSpace(allStderr.String()), "\n") {
		if line != "" {
			shared.Log("info", "[update] "+line)
		}
	}

	jsonResp(w, 200, map[string]interface{}{
		"success":      true,
		"updatedCount": updatedCount,
		"output":       allStdout.String(),
		"warnings":     allStderr.String(),
	})
}

// findSelfContainerName resolves the name of the Yantr container.
// Priority:
//  1. YANTR_CONTAINER_NAME env var (explicit override)
//  2. Scan running containers for one whose image contains the YANTR_IMAGE
//     value (default: "yantr") — covers any tag / registry prefix.
//  3. Hard fallback: "yantr"
func findSelfContainerName() string {
	if name := os.Getenv("YANTR_CONTAINER_NAME"); name != "" {
		return name
	}
	imageName := strings.ToLower(coalesce(os.Getenv("YANTR_IMAGE"), "yantr"))
	ctrs, err := docker.ContainerList(context.Background(), dockerctr.ListOptions{All: false})
	if err == nil {
		for _, c := range ctrs {
			if strings.Contains(strings.ToLower(c.Image), imageName) && len(c.Names) > 0 {
				name := strings.TrimPrefix(c.Names[0], "/")
				shared.Log("info", fmt.Sprintf("[update:self] resolved container %q from image %q", name, c.Image))
				return name
			}
		}
	}
	shared.Log("warn", "[update:self] could not resolve container by image, falling back to \"yantr\"")
	return "yantr"
}

const selfUpdateDelay = 10 * time.Minute

// runSelfUpdateNow runs Watchtower for the given container name and logs the result.
// Must always be called from a goroutine — Watchtower may stop this process if
// a newer image is available.
func runSelfUpdateNow(name string) {
	shared.Log("info", "[update:self] running watchtower for: "+name)
	stdout, stderr, exitCode, err := runWatchtower([]string{name})
	if err != nil {
		shared.Log("error", "[update:self] watchtower error: "+err.Error())
		return
	}
	for _, line := range strings.Split(strings.TrimSpace(stdout), "\n") {
		if line != "" {
			shared.Log("info", "[update:self] "+line)
		}
	}
	for _, line := range strings.Split(strings.TrimSpace(stderr), "\n") {
		if line != "" {
			shared.Log("info", "[update:self] "+line)
		}
	}
	wCombined := strings.ToLower(stdout + "\n" + stderr)
	updated := strings.Contains(wCombined, "found new") || strings.Contains(wCombined, "updating") || strings.Contains(wCombined, "updated")
	if updated {
		shared.Log("info", "[update:self] Yantr updated — restarting")
		telemetry.TrackSelfUpdate(1, version)
	} else {
		shared.Log("info", fmt.Sprintf("[update:self] no update found (exit=%d)", exitCode))
	}
}
