package main

import (
	"context"
	"core/apps"
	"core/compose"
	"core/docker"
	"core/shared"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	dockerctr "github.com/docker/docker/api/types/container"
	"github.com/go-chi/chi/v5"
)

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
				shared.Log("info", fmt.Sprintf("[container] removing stack: project=%s container=%s", project, name))
				downCtx, downCancel := context.WithTimeout(context.Background(), spawnTimeoutMedium)
				out, errStr, exitCode, _ := spawnExec(downCtx, cmdName, args, env, appPath)
				downCancel()
				if out != "" {
					for _, line := range strings.Split(strings.TrimSpace(out), "\n") {
						if line != "" {
							shared.Log("info", "[container] "+line)
						}
					}
				}
				if errStr != "" {
					for _, line := range strings.Split(strings.TrimSpace(errStr), "\n") {
						if line != "" {
							shared.Log("info", "[container] "+line)
						}
					}
				}
				if exitCode == 0 {
					shared.Log("info", fmt.Sprintf("[container] stack removed: project=%s", project))
					compose.DeleteProjectCompose(appPath, project)
					jsonResp(w, 200, map[string]interface{}{
						"success": true, "message": fmt.Sprintf("App stack '%s' removed successfully", project),
						"container": name, "stackRemoved": true,
					})
					return
				}
				shared.Log("error", fmt.Sprintf("[container] compose down failed: project=%s exit=%d", project, exitCode))
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

func handleContainerStart(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := docker.Client.ContainerStart(context.Background(), id, dockerctr.StartOptions{}); err != nil {
		jsonErr(w, 500, "CONTAINER_START_FAILED", err.Error())
		return
	}
	jsonResp(w, 200, map[string]interface{}{"success": true, "message": "Container started successfully"})
}

func handleContainerStop(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := docker.Client.ContainerStop(context.Background(), id, dockerctr.StopOptions{}); err != nil {
		jsonErr(w, 500, "CONTAINER_STOP_FAILED", err.Error())
		return
	}
	jsonResp(w, 200, map[string]interface{}{"success": true, "message": "Container stopped successfully"})
}

func handleContainerRestart(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := docker.Client.ContainerRestart(context.Background(), id, dockerctr.StopOptions{}); err != nil {
		jsonErr(w, 500, "CONTAINER_RESTART_FAILED", err.Error())
		return
	}
	jsonResp(w, 200, map[string]interface{}{"success": true, "message": "Container restarted successfully"})
}
