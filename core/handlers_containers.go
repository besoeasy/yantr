package main

import (
	"context"
	"core/apps"
	"core/compose"
	"core/podman"
	"core/shared"
	"core/supervisor"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	dockerctr "github.com/docker/docker/api/types/container"
	"github.com/go-chi/chi/v5"
)

func handleContainers(w http.ResponseWriter, r *http.Request) {
	containers, err := podman.ContainerList(context.Background(), dockerctr.ListOptions{All: true})
	if err != nil {
		jsonErr(w, 500, "CONTAINERS_FETCH_FAILED", err.Error())
		return
	}
	catalogMap := apps.GetCatalogIndex()

	// Find Yantr projects
	yantrProjects := map[string]bool{}
	for _, c := range containers {
		lbl := parseAppLabels(c.Labels)
		if project := compose.ComposeProjectLabel(c.Labels); lbl.App != "" && project != "" {
			yantrProjects[project] = true
		}
	}

	var result []map[string]interface{}
	for _, c := range containers {
		if c.Labels["yantr.system"] == "browser" {
			continue
		}
		isBrowser := false
		for _, n := range c.Names {
			clean := strings.TrimPrefix(n, "/")
			if strings.HasPrefix(clean, "y-fs-") || strings.HasPrefix(clean, "yantr-browse-") {
				isBrowser = true
				break
			}
		}
		if isBrowser {
			continue
		}

		lbl := parseAppLabels(c.Labels)
		project := compose.ComposeProjectLabel(c.Labels)
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
	info, err := podman.ContainerInspect(context.Background(), id)
	if err != nil {
		jsonErr(w, 404, "CONTAINER_NOT_FOUND", "Container not found")
		return
	}

	lbl := parseAppLabels(info.Config.Labels)
	project := compose.ComposeProjectLabel(info.Config.Labels)
	appID := coalesce(lbl.App, getBaseAppID(project), strings.TrimPrefix(info.Name, "/"))
	entry := apps.GetCatalogIndex()[appID]
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

// handleContainerStats returns one instantaneous sample of container resource
// usage.
//
// CPU is reported as raw cumulative counters, not as a percentage. Podman only
// populates precpu_stats when the request asks for stream=1; a single-shot
// request (stream=0) therefore always returns precpu_stats as all zeros, so the
// usual
//
//	(cpu_delta / system_delta) * online_cpus * 100
//
// degenerates to (container_lifetime_ns / host_lifetime_ns), which is a
// monotonic ramp unrelated to current load — a container pegging a core reports
// single digits for as long as it lives, and a falling load still shows a rising
// number.
//
// Using stream=1 is not the fix: the first object in a stream has a
// back-to-back precpu with a ~0ms window, so system_delta is frequently 0 and the
// result silently becomes 0.00. The correct computation needs two samples, and
// the UI already polls every 2s — so the counters go out raw and
// ContainerResources.vue differences consecutive samples client-side, with no
// extra requests and no added latency.
//
// system_cpu_usage is deliberately not sent. Rootless Podman reports it for the
// user's cgroup slice rather than the whole host, so pairing it with the
// host-wide online_cpus yields a rate inflated by the slice's share of the
// machine (measured: 1680% for a container pegging one core). The client
// normalizes against sampledAtMs instead; see ui/src/utils/cpu.js.
func handleContainerStats(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	statsResp, err := podman.ContainerStats(ctx, id, false)
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
			// Cumulative counters; the client differences consecutive samples.
			// `percent` is intentionally absent rather than wrong.
			"cpu": map[string]interface{}{
				"usage":       stats.CPUStats.CPUUsage.TotalUsage,
				"onlineCpus":  stats.CPUStats.OnlineCPUs,
				"sampledAtMs": shared.NowMs(),
			},
			"memory":  map[string]interface{}{"usage": memUsage, "rawUsage": rawMem, "cache": cache, "limit": limit, "percent": fmt.Sprintf("%.2f", memPct)},
			"network": map[string]interface{}{"rx": netRx, "tx": netTx},
			"blockIO": map[string]interface{}{"read": blkR, "write": blkW},
		},
	})
}

// maxLogTail caps the ?tail parameter for container logs.
//
// Podman honours Tail: "all", which makes the engine stream the container's
// entire log. Without a cap, one authenticated request against a chatty
// container is an unbounded read held entirely in memory before a single byte
// reaches the client. The UI only ever asks for 200, so this is invisible to it.
const maxLogTail = 1000

// maxLogBytes is a backstop against a container whose engine-side tail filter
// is not honoured. Whatever happens upstream, the process will not buffer more
// than this per request.
const maxLogBytes = 8 << 20 // 8 MiB

// parseLogTail validates and clamps ?tail.
//
// "all" is deliberately collapsed to the cap rather than passed through: the
// whole point is that the engine must never be asked for an unbounded read.
func parseLogTail(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "100", nil
	}
	if strings.EqualFold(raw, "all") {
		return strconv.Itoa(maxLogTail), nil
	}
	n, err := strconv.Atoi(raw)
	if err != nil {
		return "", fmt.Errorf("tail must be a positive integer or \"all\"")
	}
	if n <= 0 {
		return "", fmt.Errorf("tail must be greater than zero")
	}
	if n > maxLogTail {
		n = maxLogTail
	}
	return strconv.Itoa(n), nil
}

func handleContainerLogs(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	tail, err := parseLogTail(r.URL.Query().Get("tail"))
	if err != nil {
		jsonErr(w, 400, "INVALID_TAIL", err.Error())
		return
	}

	// A wedged engine socket would otherwise hold this goroutine forever; the
	// route carries no withWriteTimeout wrapper of its own.
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	logsBody, err := podman.ContainerLogs(ctx, id, dockerctr.LogsOptions{
		ShowStdout: true, ShowStderr: true, Tail: tail, Timestamps: true,
	})
	if err != nil {
		jsonErr(w, 500, "LOGS_FETCH_FAILED", err.Error())
		return
	}
	defer logsBody.Close()

	var raw strings.Builder
	if _, err := io.Copy(&raw, io.LimitReader(logsBody, maxLogBytes)); err != nil {
		jsonErr(w, 500, "LOGS_READ_FAILED", err.Error())
		return
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
	info, err := podman.ContainerInspect(context.Background(), id)
	if err != nil {
		jsonErr(w, 404, "CONTAINER_NOT_FOUND", "Container not found")
		return
	}
	name := strings.TrimPrefix(info.Name, "/")
	project := compose.ComposeProjectLabel(info.Config.Labels)

	if project != "" {
		// Tell the watchdog to back off while we tear the stack down, so it
		// doesn't race us by restarting containers mid-removal.
		supervisor.MarkStackRemoving(project)
		defer supervisor.UnmarkStackRemoving(project)

		// Acquired before the flag so a teardown queued behind a deploy does not
		// let the watchdog in while it waits.
		release, locked := shared.TryLockProject(project)
		if !locked {
			jsonErr(w, 409, "PROJECT_BUSY",
				fmt.Sprintf("Another operation is already in progress for '%s'. Retry shortly.", project))
			return
		}
		defer release()

		baseID := getBaseAppID(project)
		appPath := filepath.Join(apps.GetAppsDir(), baseID)
		ref := compose.GetProjectComposeRef(appPath, project)
		if _, statErr := os.Stat(ref.ComposePath); statErr == nil {
			if cmdName, cmdArgs, err := getComposeCommand(); err == nil {
				env, _ := compose.GetComposeProcessEnv(appPath, project, podman.SocketPath, podman.HostSocket())
				args := append(cmdArgs, "-p", project, "-f", ref.ComposeFile, "down")
				shared.Log("info", fmt.Sprintf("[container] removing stack: project=%s container=%s", project, name))
				job := globalJobs.Create("container_delete", id, fmt.Sprintf("Delete container %s", name))
				job.SetProgress(fmt.Sprintf("Removing stack for container %s...", name))
				downCtx, downCancel := context.WithTimeout(context.Background(), spawnTimeoutMedium)
				out, errStr, exitCode, _ := spawnExecJob(downCtx, job, cmdName, args, env, appPath)
				downCancel()
				if exitCode == 0 {
					shared.Log("info", fmt.Sprintf("[container] stack removed: project=%s", project))
					compose.DeleteProjectCompose(appPath, project)
					supervisor.RecordStackRemoved(project)
					supervisor.ForgetContainer(id)
					job.Complete(map[string]interface{}{"success": true, "stackRemoved": true})
					jsonResp(w, 200, map[string]interface{}{
						"success": true, "jobId": job.ID, "message": fmt.Sprintf("App stack '%s' removed successfully", project),
						"container": name, "stackRemoved": true,
					})
					return
				}
				job.Fail(fmt.Errorf("compose down failed: %s", coalesce(errStr, out)), exitCode)
				shared.Log("error", fmt.Sprintf("[container] compose down failed: project=%s exit=%d", project, exitCode))
			}
		}
	}

	if err := podman.ContainerRemove(context.Background(), id, dockerctr.RemoveOptions{Force: true}); err != nil {
		jsonErr(w, 500, "CONTAINER_REMOVE_FAILED", err.Error())
		return
	}
	// The container is gone; drop its watchdog restart history.
	supervisor.ForgetContainer(id)
	jsonResp(w, 200, map[string]interface{}{"success": true, "message": fmt.Sprintf("Container '%s' removed successfully", name)})
}

func handleContainerStart(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := podman.ContainerStart(context.Background(), id, dockerctr.StartOptions{}); err != nil {
		jsonErr(w, 500, "CONTAINER_START_FAILED", err.Error())
		return
	}
	// Starting one service must not mark the whole project deployed — that would
	// silently re-arm the watchdog and boot resuscitation for services the user
	// stopped. Only this service's stop mark is cleared.
	recordContainerServiceStarted(id)
	jsonResp(w, 200, map[string]interface{}{"success": true, "message": "Container started successfully"})
}

func handleContainerStop(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := podman.ContainerStop(context.Background(), id, dockerctr.StopOptions{}); err != nil {
		jsonErr(w, 500, "CONTAINER_STOP_FAILED", err.Error())
		return
	}
	// Record the stop against this service only. Marking the whole project
	// stopped would disable crash recovery and boot resuscitation for every
	// sibling service in a multi-service stack.
	recordContainerServiceStopped(id)
	jsonResp(w, 200, map[string]interface{}{"success": true, "message": "Container stopped successfully"})
}

func handleContainerRestart(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := podman.ContainerRestart(context.Background(), id, dockerctr.StopOptions{}); err != nil {
		jsonErr(w, 500, "CONTAINER_RESTART_FAILED", err.Error())
		return
	}
	// Restart is an explicit user request, so it also clears an existing
	// intentional-stop mark. Previously this handler recorded nothing, leaving a
	// project that had been stopped marked stopped forever.
	recordContainerServiceStarted(id)
	jsonResp(w, 200, map[string]interface{}{"success": true, "message": "Container restarted successfully"})
}

// recordContainerServiceStopped marks the stopped container's compose service as
// intentionally stopped. Standalone containers have no project label and are
// simply skipped.
func recordContainerServiceStopped(id string) {
	project, service, ok := containerProjectAndService(id)
	if !ok {
		return
	}
	supervisor.RecordStackServiceStopped(project, service)
	shared.Log("info", fmt.Sprintf("[container] recorded service stop: project=%s service=%s", project, service))
}

func recordContainerServiceStarted(id string) {
	project, service, ok := containerProjectAndService(id)
	if !ok {
		return
	}
	supervisor.RecordStackServiceStarted(project, service)
	shared.Log("info", fmt.Sprintf("[container] recorded service start: project=%s service=%s", project, service))
}

// containerProjectAndService reads the compose project and service labels off a
// container, falling back to the container name when the service label is absent.
func containerProjectAndService(id string) (project, service string, ok bool) {
	info, err := podman.ContainerInspect(context.Background(), id)
	if err != nil || info.Config == nil {
		return "", "", false
	}
	project = compose.ComposeProjectLabel(info.Config.Labels)
	if project == "" {
		return "", "", false
	}
	service = compose.ComposeServiceLabel(info.Config.Labels)
	if service == "" {
		service = strings.TrimPrefix(info.Name, "/")
	}
	return project, service, true
}
