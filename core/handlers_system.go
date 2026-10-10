package main

import (
	"context"
	"core/apps"
	"core/auth"
	"core/compose"
	"core/podman"
	"core/shared"
	"core/supervisor"
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

func sweepExpiredContainers() {
	all, err := podman.ContainerList(context.Background(), dockerctr.ListOptions{All: false})
	if err != nil {
		shared.Log("warn", "[reaper] failed to list containers: "+err.Error())
		return
	}

	now := time.Now().Unix()

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
		project := compose.ComposeProjectLabel(c.Labels)
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

	for projectID, meta := range expiredProjects {
		// The reaper is a single goroutine on a 1-minute ticker, so this must
		// never block: a project lock held by a 10-minute `compose down` would
		// stall reaping of every *other* project, silently. Skip and retry on
		// the next tick instead.
		release, locked := shared.TryLockProject(projectID)
		if !locked {
			shared.Log("info", fmt.Sprintf("[reaper] project %s busy, skipping this tick", projectID))
			continue
		}

		shared.Log("info", fmt.Sprintf("[reaper] removing expired stack: %s", projectID))
		// Tell the watchdog to back off so it doesn't restart containers
		// mid-teardown and race the compose down.
		supervisor.MarkStackRemoving(projectID)
		appPath := filepath.Join(apps.GetAppsDir(), meta.appID)
		ref := compose.GetProjectComposeRef(appPath, projectID)
		removed := false
		if _, statErr := os.Stat(ref.ComposePath); statErr == nil {
			if cmdName, cmdArgs, cmdErr := getComposeCommand(); cmdErr == nil {
				env, _ := compose.GetComposeProcessEnv(appPath, projectID, podman.SocketPath, podman.HostSocket())
				args := append(cmdArgs, "-p", projectID, "-f", ref.ComposeFile, "down")
				reaperCtx, reaperCancel := context.WithTimeout(context.Background(), spawnTimeoutMedium)
				_, _, exitCode, _ := spawnExec(reaperCtx, cmdName, args, env, appPath)
				reaperCancel()
				if exitCode == 0 {
					compose.DeleteProjectCompose(appPath, projectID)
					supervisor.RecordStackRemoved(projectID)
					shared.Log("info", fmt.Sprintf("[reaper] stack %s removed", projectID))
					removed = true
				}
			}
		}
		if !removed {
			shared.Log("warn", fmt.Sprintf("[reaper] compose down failed for %s — force-removing containers", projectID))
			if stale, listErr := podman.ContainerList(context.Background(), dockerctr.ListOptions{All: true}); listErr == nil {
				for _, c := range stale {
					if compose.ComposeProjectLabel(c.Labels) == projectID {
						_ = podman.ContainerRemove(context.Background(), c.ID, dockerctr.RemoveOptions{Force: true})
					}
				}
			}
			compose.DeleteProjectCompose(appPath, projectID)
			supervisor.RecordStackRemoved(projectID)
		}
		supervisor.UnmarkStackRemoving(projectID)
		release()
	}

	for _, id := range standaloneIDs {
		shared.Log("info", fmt.Sprintf("[reaper] removing expired standalone container: %s", id))
		_ = podman.ContainerStop(context.Background(), id, dockerctr.StopOptions{})
		_ = podman.ContainerRemove(context.Background(), id, dockerctr.RemoveOptions{})
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
	info, err := podman.Info(context.Background())
	if err != nil {
		jsonErr(w, 500, "SYSTEM_INFO_FETCH_FAILED", err.Error())
		return
	}
	podmanMap := map[string]interface{}{
		"version": info.ServerVersion,
		"containers": map[string]interface{}{
			"total": info.Containers, "running": info.ContainersRunning,
			"paused": info.ContainersPaused, "stopped": info.ContainersStopped,
		},
		"images": info.Images,
	}
	jsonResp(w, 200, map[string]interface{}{
		"success": true,
		"info": map[string]interface{}{
			"cpu":     map[string]interface{}{"cores": info.NCPU},
			"memory":  map[string]interface{}{"total": info.MemTotal},
			"storage": map[string]interface{}{"driver": info.Driver},
			"podman":  podmanMap,
			"docker":  podmanMap, // preserved for UI backward-compatibility
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
		if pruned, err := podman.ImagesPrune(context.Background(), filters); err == nil {
			shared.Log("info", fmt.Sprintf("[prune] images: removed=%d reclaimed=%d bytes", len(pruned.ImagesDeleted), pruned.SpaceReclaimed))
			results["images"] = map[string]interface{}{
				"count": len(pruned.ImagesDeleted), "spaceReclaimed": pruned.SpaceReclaimed,
			}
		} else {
			shared.Log("warn", "[prune] images failed: "+err.Error())
		}
	}
	if body.Volumes {
		if pruned, err := podman.VolumesPrune(context.Background(), dockerfilters.NewArgs()); err == nil {
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
	ctrs, err := podman.ContainerList(context.Background(), dockerctr.ListOptions{})
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
	ctrs, _ := podman.ContainerList(context.Background(), dockerctr.ListOptions{})
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

func handleAutoupdateRun(w http.ResponseWriter, r *http.Request) {
	var body struct {
		ContainerIDs []string `json:"containerIds"`
	}
	if !parseJSON(w, r, &body) {
		return
	}
	ctrs, _ := podman.ContainerList(context.Background(), dockerctr.ListOptions{})
	idSet := map[string]bool{}
	for _, id := range body.ContainerIDs {
		idSet[id] = true
	}

	// Only containers belonging to a Compose project are updatable. Standalone
	// containers (e.g. the volume browser) are filtered out of the container
	// list, so they are never offered to the UI, and they are rebuilt from
	// scratch on their next Start anyway — there is nothing to update here.
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
		if !match {
			continue
		}
		if project := compose.ComposeProjectLabel(c.Labels); project != "" {
			projectSet[project] = true
		}
	}

	if len(projectSet) == 0 {
		jsonErr(w, 404, "CONTAINERS_NOT_RUNNING", "None of the provided container IDs are currently running")
		return
	}

	var allStdout, allStderr strings.Builder
	updatedCount := 0

	job := globalJobs.Create("autoupdate", "system", "Auto-update containers")
	job.SetProgress("Checking for updates...")

	cmdName, cmdArgs, cmdErr := getComposeCommand()
	for projectID := range projectSet {
		if cmdErr != nil {
			allStderr.WriteString(fmt.Sprintf("[update] podman compose not available for %s: %v\n", projectID, cmdErr))
			continue
		}

		// Per project, inside the loop: locking once around the whole loop would
		// let one slow project's 30-minute `up -d` block every other project in
		// the same request and hold the job open for all of them.
		release, locked := shared.TryLockProject(projectID)
		if !locked {
			shared.Log("warn", fmt.Sprintf("[update] skipping %s: another operation in progress", projectID))
			allStderr.WriteString(fmt.Sprintf("[update] skipped %s: another operation is already in progress\n", projectID))
			continue
		}

		baseID := getBaseAppID(projectID)

		appPath := filepath.Join(apps.GetAppsDir(), baseID)
		ref := compose.GetProjectComposeRef(appPath, projectID)

		if _, statErr := os.Stat(ref.ComposePath); statErr != nil {
			allStderr.WriteString(fmt.Sprintf("[update] compose.yml not found for %s\n", projectID))
			release()
			continue
		}

		env, _ := compose.GetComposeProcessEnv(appPath, projectID, podman.SocketPath, podman.HostSocket())

		// Snapshot the stack's image IDs before the pull so we can tell whether
		// anything actually changed. Sniffing the pull output for phrases like
		// "downloaded newer image" only matches Docker Compose v2 wording —
		// podman-compose says "Copying blob"/"Writing manifest" instead, which
		// made every real update report as "already up to date".
		stackImages := stackImageRefs(ref.ComposePath)
		before, beforeErr := podman.LocalImageIDs(podman.Background())

		shared.Log("info", fmt.Sprintf("[update] pulling latest images for stack: %s", projectID))
		job.SetProgress(fmt.Sprintf("Pulling images for %s...", projectID))
		pullCtx, pullCancel := context.WithTimeout(context.Background(), spawnTimeoutLong)
		pullArgs := append(cmdArgs, "-p", projectID, "-f", ref.ComposeFile, "pull")
		outPull, errPull, exitPull, _ := spawnExecJob(pullCtx, job, cmdName, pullArgs, env, appPath)
		pullCancel()

		allStdout.WriteString(outPull + "\n")
		allStderr.WriteString(errPull + "\n")

		if exitPull != 0 {
			shared.Log("error", fmt.Sprintf("[update] pull failed for %s (exit=%d)", projectID, exitPull))
			release()
			continue
		}

		// Compare image IDs before/after the pull. When the snapshots prove
		// nothing changed, skip the recreate entirely: `up -d` would restart
		// every container for a guaranteed no-op. When the comparison is
		// inconclusive (snapshot failed, or the compose file declared no
		// images), recreate anyway — failing to adopt a real update is worse
		// than an unnecessary restart — but do not count it as an update.
		after, afterErr := podman.LocalImageIDs(podman.Background())
		if afterErr != nil {
			shared.Log("warn", "[update] could not read image list after pull: "+afterErr.Error())
		}
		check := classifyUpdate(stackImages, before, beforeErr, after, afterErr)

		if check == updateUnchanged {
			shared.Log("info", fmt.Sprintf("[update] stack %s is already up to date", projectID))
			job.SetProgress(fmt.Sprintf("Stack %s is already up to date", projectID))
			release()
			continue
		}
		if check == updateUnknown {
			shared.Log("warn", fmt.Sprintf("[update] could not determine whether images changed for %s — recreating to be safe", projectID))
		}

		shared.Log("info", fmt.Sprintf("[update] recreating stack: %s", projectID))
		job.SetProgress(fmt.Sprintf("Recreating stack %s...", projectID))
		upCtx, upCancel := context.WithTimeout(context.Background(), spawnTimeoutLong)
		upArgs := append(cmdArgs, "-p", projectID, "-f", ref.ComposeFile, "up", "-d")
		outUp, errUp, exitUp, _ := spawnExecJob(upCtx, job, cmdName, upArgs, env, appPath)
		upCancel()

		allStdout.WriteString(outUp + "\n")
		allStderr.WriteString(errUp + "\n")

		if exitUp != 0 {
			shared.Log("error", fmt.Sprintf("[update] 'up -d' failed for %s (exit=%d)", projectID, exitUp))
			release()
			continue
		}

		// Count as updated only when a newer image was provably pulled AND the
		// stack came up cleanly. An inconclusive comparison still recreates
		// (above) but is never reported as an update: under-reporting is the
		// safer failure mode.
		if check == updateChanged {
			updatedCount++
			shared.Log("info", fmt.Sprintf("[update] stack %s was updated", projectID))
			telemetry.TrackUpdatesForContainers([]string{projectID})
		} else {
			shared.Log("info", fmt.Sprintf("[update] stack %s recreated (change inconclusive)", projectID))
		}

		release()
	}

	// Pull + recreate orphans the superseded images (they lose their tag).
	// Reclaim them now, but only when something actually updated — a no-op
	// update must not delete anything. dangling=true removes untagged layers
	// only, the same safe default as the manual prune endpoint.
	prunedImages := 0
	var prunedBytes uint64
	if updatedCount > 0 {
		filters := dockerfilters.NewArgs()
		filters.Add("dangling", "true")
		if pruned, err := podman.ImagesPrune(context.Background(), filters); err == nil {
			prunedImages = len(pruned.ImagesDeleted)
			prunedBytes = pruned.SpaceReclaimed
			shared.Log("info", fmt.Sprintf("[update] pruned %d dangling images, reclaimed %d bytes", prunedImages, prunedBytes))
		} else {
			shared.Log("warn", "[update] post-update prune failed: "+err.Error())
		}
	}

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

	job.Complete(map[string]interface{}{
		"success":      true,
		"updatedCount": updatedCount,
		"prunedImages": prunedImages, "spaceReclaimed": prunedBytes,
	})

	jsonResp(w, 200, map[string]interface{}{
		"success":      true,
		"jobId":        job.ID,
		"updatedCount": updatedCount,
		"prunedImages": prunedImages, "spaceReclaimed": prunedBytes,
		"output":   allStdout.String(),
		"warnings": allStderr.String(),
	})
}

// stackImageRefs returns the normalized image references declared by a compose
// file. An unreadable or malformed file yields no references, which makes
// classifyUpdate return updateUnknown rather than guessing.
func stackImageRefs(composePath string) []string {
	data, err := os.ReadFile(composePath)
	if err != nil {
		return nil
	}
	doc, err := compose.Parse(string(data))
	if err != nil {
		return nil
	}
	refs := compose.ServiceImages(doc)
	normalized := make([]string, 0, len(refs))
	for _, ref := range refs {
		if n := compose.NormalizeImageRef(ref); n != "" {
			normalized = append(normalized, n)
		}
	}
	return normalized
}

// updateCheck classifies the result of a pull for a stack.
type updateCheck int

const (
	// updateUnchanged means the before/after snapshots prove no image ID
	// behind the stack's references moved. The recreate can be skipped.
	updateUnchanged updateCheck = iota
	// updateChanged means an image ID provably moved (or an image appeared
	// that was not local before — a first-time pull).
	updateChanged
	// updateUnknown means the comparison is inconclusive: a snapshot failed
	// or the compose file declared no images. The caller recreates anyway
	// but must not report it as an update.
	updateUnknown
)

// classifyUpdate is the pure decision behind the update flow, split out so it
// can be tested without a live Podman socket.
func classifyUpdate(refs []string, before map[string]string, beforeErr error, after map[string]string, afterErr error) updateCheck {
	if beforeErr != nil || afterErr != nil || len(refs) == 0 {
		return updateUnknown
	}
	if imageIDsDiffer(refs, before, after) {
		return updateChanged
	}
	return updateUnchanged
}

// imageIDsDiffer is the pure comparison behind classifyUpdate, split out so it
// can be tested without a live Podman socket.
func imageIDsDiffer(refs []string, before, after map[string]string) bool {
	for _, ref := range refs {
		prev, hadBefore := before[ref]
		now, hasAfter := after[ref]
		if !hadBefore && hasAfter {
			return true
		}
		if hasAfter && prev != now {
			return true
		}
	}
	return false
}
