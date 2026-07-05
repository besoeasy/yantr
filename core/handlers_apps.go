package main

import (
	"context"
	"core/apps"
	"core/caddy"
	"core/compose"
	"core/docker"
	"core/shared"
	"core/system"
	"core/telemetry"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	dockerfilters "github.com/docker/docker/api/types/filters"
	dockernet "github.com/docker/docker/api/types/network"
	"github.com/go-chi/chi/v5"
)

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
	if !validAppID.MatchString(appID) {
		jsonErr(w, 400, "INVALID_APP_ID", "Invalid app ID")
		return
	}
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
		"image":     strings.TrimSpace(string(m[1])),
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
		Auth               *struct {
			Enabled    bool   `json:"enabled"`
			Port       int    `json:"port"`
			TargetPort int    `json:"targetPort"`
			Username   string `json:"username"`
			Password   string `json:"password"`
		} `json:"auth"`
	}
	if !parseJSON(w, r, &body) {
		return
	}
	if body.AppID == "" {
		jsonErr(w, 400, "APP_ID_REQUIRED", "appId is required")
		return
	}
	if !validAppID.MatchString(body.AppID) {
		jsonErr(w, 400, "INVALID_APP_ID", "Invalid app ID")
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

	// ── x-auth: parse and inject Caddy auth labels ─────────────────────────
	if doc, parseErr := compose.Parse(modifiedContent); parseErr == nil {
		auth := caddy.ParseXAuth(doc)
		if body.Auth != nil && body.Auth.Enabled {
			auth = &caddy.XAuth{
				Port:       body.Auth.Port,
				TargetPort: body.Auth.TargetPort,
				Username:   body.Auth.Username,
				Password:   body.Auth.Password,
			}
		}
		if auth != nil {
			if injectErr := caddy.InjectCaddyAuthLabels(doc, auth); injectErr != nil {
				shared.Log("warn", "[deploy] x-auth inject failed: "+injectErr.Error())
			} else if rebuilt, rebuildErr := compose.Stringify(doc); rebuildErr == nil {
				modifiedContent = rebuilt
			}
		}
	}

	ref, err := compose.WriteProjectCompose(appPath, projectName, modifiedContent)
	if err != nil {
		jsonErr(w, 500, "COMPOSE_WRITE_FAILED", err.Error())
		return
	}

	cmdName, cmdArgs, err := getComposeCommand()
	if err != nil {
		shared.Log("error", "[deploy] compose command not found: "+err.Error())
		jsonErr(w, 500, "COMPOSE_NOT_FOUND", err.Error())
		return
	}
	composeEnv, _ := compose.GetComposeProcessEnv(appPath, projectName, docker.SocketPath)
	args := append(cmdArgs, "-p", projectName, "-f", ref.ComposeFile, "up", "-d")
	shared.Log("info", fmt.Sprintf("[deploy] starting: app=%s project=%s cmd=%s %s", body.AppID, projectName, cmdName, strings.Join(args, " ")))
	deployCtx, deployCancel := context.WithTimeout(context.Background(), spawnTimeoutLong)
	defer deployCancel()
	stdout, stderr, exitCode, _ := spawnExec(deployCtx, cmdName, args, composeEnv, appPath)
	if stdout != "" {
		for _, line := range strings.Split(strings.TrimSpace(stdout), "\n") {
			if line != "" {
				shared.Log("info", "[deploy] "+line)
			}
		}
	}
	if stderr != "" {
		for _, line := range strings.Split(strings.TrimSpace(stderr), "\n") {
			if line != "" {
				shared.Log("info", "[deploy] "+line)
			}
		}
	}
	if exitCode != 0 {
		shared.Log("error", fmt.Sprintf("[deploy] FAILED: app=%s exit=%d", body.AppID, exitCode))
		jsonErr(w, 500, "DEPLOYMENT_FAILED", coalesce(stderr, stdout))
		return
	}
	shared.Log("info", fmt.Sprintf("[deploy] SUCCESS: app=%s project=%s", body.AppID, projectName))

	// Reload Caddy so new yantr.caddy.* labels are picked up
	if caddy.IsRunning() {
		if reloadErr := caddy.ReloadCaddyConfig(); reloadErr != nil {
			shared.Log("warn", "[deploy] caddy reload after deploy failed: "+reloadErr.Error())
		} else {
			shared.Log("info", "[deploy] caddy reloaded")
		}
	}

	jsonResp(w, 200, map[string]interface{}{
		"success": true,
		"message": fmt.Sprintf("App '%s' deployed successfully", body.AppID),
		"appId":   body.AppID, "output": stdout, "warnings": stderr,
		"temporary": body.ExpiresIn > 0,
	})

	telemetry.TrackInstall(body.AppID)
}
