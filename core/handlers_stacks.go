package main

import (
	"context"
	"core/apps"
	"core/compose"
	"core/docker"
	"core/shared"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	dockerctr "github.com/docker/docker/api/types/container"
	"github.com/go-chi/chi/v5"
)

func handleStackDetail(w http.ResponseWriter, r *http.Request) {
	projectID := chi.URLParam(r, "projectId")
	all, err := docker.ContainerList(context.Background(), dockerctr.ListOptions{All: true})
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
		for _, p := range c.Ports {
			if p.PublicPort == 0 {
				continue
			}
			portSpecificKey := fmt.Sprintf("yantr.service.%d", p.PrivatePort)
			svc := coalesce(c.Labels[portSpecificKey], strings.TrimPrefix(c.Names[0], "/"), "unknown")
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
		info, err := docker.ContainerInspect(context.Background(), c.ID)
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
			"service":       coalesce(lbl.Service, name),
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

func handleStackDelete(w http.ResponseWriter, r *http.Request) {
	projectID := chi.URLParam(r, "projectId")

	all, err := docker.ContainerList(context.Background(), dockerctr.ListOptions{All: true})
	if err != nil {
		jsonErr(w, 500, "DOCKER_ERROR", err.Error())
		return
	}

	var projectContainers []dockerctr.Summary
	for _, c := range all {
		if c.Labels["com.docker.compose.project"] == projectID {
			projectContainers = append(projectContainers, c)
		}
	}

	if len(projectContainers) == 0 {
		jsonErr(w, 404, "STACK_NOT_FOUND", "No containers found for this stack")
		return
	}

	baseID := getBaseAppID(projectID)
	appPath := filepath.Join(apps.GetAppsDir(), baseID)
	ref := compose.GetProjectComposeRef(appPath, projectID)

	if _, statErr := os.Stat(ref.ComposePath); statErr == nil {
		if cmdName, cmdArgs, err := getComposeCommand(); err == nil {
			env, _ := compose.GetComposeProcessEnv(appPath, projectID, docker.SocketPath)
			args := append(cmdArgs, "-p", projectID, "-f", ref.ComposeFile, "down")
			shared.Log("info", fmt.Sprintf("[stack] removing: project=%s", projectID))
			downCtx, downCancel := context.WithTimeout(context.Background(), spawnTimeoutMedium)
			out, errStr, exitCode, err := spawnExec(downCtx, cmdName, args, env, appPath)
			downCancel()
			if out != "" {
				for _, line := range strings.Split(strings.TrimSpace(out), "\n") {
					if line != "" {
						shared.Log("info", "[stack] "+line)
					}
				}
			}
			if errStr != "" {
				for _, line := range strings.Split(strings.TrimSpace(errStr), "\n") {
					if line != "" {
						shared.Log("info", "[stack] "+line)
					}
				}
			}
			if exitCode == 0 {
				shared.Log("info", fmt.Sprintf("[stack] removed: project=%s", projectID))
				compose.DeleteProjectCompose(appPath, projectID)
				jsonResp(w, 200, map[string]interface{}{
					"success": true,
					"message": fmt.Sprintf("Stack '%s' removed successfully", projectID),
					"removed": true,
				})
				return
			}
			shared.Log("error", fmt.Sprintf("[stack] compose down failed: project=%s exit=%d err=%v", projectID, exitCode, err))
			jsonErr(w, 500, "STACK_REMOVE_FAILED", fmt.Sprintf("docker compose down failed (exit %d)", exitCode))
			return
		}
	}

	for _, c := range projectContainers {
		id := c.ID
		if c.State == "running" {
			_ = docker.ContainerStop(context.Background(), id, dockerctr.StopOptions{})
		}
		_ = docker.ContainerRemove(context.Background(), id, dockerctr.RemoveOptions{})
	}

	jsonResp(w, 200, map[string]interface{}{
		"success": true,
		"message": fmt.Sprintf("Stack '%s' removed successfully", projectID),
	})
}

func handleStackRestart(w http.ResponseWriter, r *http.Request) {
	projectID := chi.URLParam(r, "projectId")

	baseID := getBaseAppID(projectID)
	appPath := filepath.Join(apps.GetAppsDir(), baseID)
	ref := compose.GetProjectComposeRef(appPath, projectID)

	if _, statErr := os.Stat(ref.ComposePath); statErr != nil {
		jsonErr(w, 404, "STACK_NOT_FOUND", "Stack compose file not found")
		return
	}

	cmdName, cmdArgs, err := getComposeCommand()
	if err != nil {
		jsonErr(w, 500, "COMPOSE_NOT_FOUND", "docker compose command not found")
		return
	}

	env, _ := compose.GetComposeProcessEnv(appPath, projectID, docker.SocketPath)
	args := append(cmdArgs, "-p", projectID, "-f", ref.ComposeFile, "restart")
	shared.Log("info", fmt.Sprintf("[stack] restarting: project=%s", projectID))
	restartCtx, restartCancel := context.WithTimeout(context.Background(), spawnTimeoutMedium)
	out, errStr, exitCode, err := spawnExec(restartCtx, cmdName, args, env, appPath)
	restartCancel()

	if out != "" {
		for _, line := range strings.Split(strings.TrimSpace(out), "\n") {
			if line != "" {
				shared.Log("info", "[stack] "+line)
			}
		}
	}
	if errStr != "" {
		for _, line := range strings.Split(strings.TrimSpace(errStr), "\n") {
			if line != "" {
				shared.Log("info", "[stack] "+line)
			}
		}
	}

	if exitCode == 0 {
		shared.Log("info", fmt.Sprintf("[stack] restarted: project=%s", projectID))
		jsonResp(w, 200, map[string]interface{}{
			"success": true,
			"message": fmt.Sprintf("Stack '%s' restarted successfully", projectID),
		})
		return
	}

	shared.Log("error", fmt.Sprintf("[stack] compose restart failed: project=%s exit=%d err=%v", projectID, exitCode, err))
	jsonErr(w, 500, "STACK_RESTART_FAILED", fmt.Sprintf("docker compose restart failed (exit %d)", exitCode))
}
