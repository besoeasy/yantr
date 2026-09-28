package main

import (
	"context"
	"core/apps"
	"core/compose"
	"core/podman"
	"core/shared"
	"core/supervisor"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"

	dockerctr "github.com/docker/docker/api/types/container"
	"github.com/go-chi/chi/v5"
)

// ─── Stable response ordering ────────────────────────────────────────────────
// The stack response is assembled from Go maps, whose iteration order Go
// randomises on every pass. The stack view polls this endpoint every 8s, so
// leaving the slices unsorted made rows visibly reshuffle on each refresh.
// These helpers impose a total order on the response so repeated polls of an
// unchanged stack return byte-identical slices.

// mapUint16 reads a uint16 field from a response fragment, tolerating a missing
// or mistyped value.
func mapUint16(m map[string]interface{}, key string) uint16 {
	if v, ok := m[key].(uint16); ok {
		return v
	}
	return 0
}

// mapString reads a string field from a response fragment, tolerating a missing
// or mistyped value.
func mapString(m map[string]interface{}, key string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}

// sortPublishedPorts orders ports by host port, then container port, then
// service, then protocol. Ports are compared numerically, not as the composite
// map key strings, so 9000 sorts after 8080 rather than before it.
func sortPublishedPorts(ports []map[string]interface{}) {
	sort.SliceStable(ports, func(i, j int) bool {
		a, b := ports[i], ports[j]
		if ha, hb := mapUint16(a, "hostPort"), mapUint16(b, "hostPort"); ha != hb {
			return ha < hb
		}
		if ca, cb := mapUint16(a, "containerPort"), mapUint16(b, "containerPort"); ca != cb {
			return ca < cb
		}
		if sa, sb := mapString(a, "service"), mapString(b, "service"); sa != sb {
			return sa < sb
		}
		return mapString(a, "protocol") < mapString(b, "protocol")
	})
}

// sortMounts orders mounts by container destination, then source.
func sortMounts(mounts []map[string]interface{}) {
	sort.SliceStable(mounts, func(i, j int) bool {
		a, b := mounts[i], mounts[j]
		if da, db := mapString(a, "destination"), mapString(b, "destination"); da != db {
			return da < db
		}
		return mapString(a, "source") < mapString(b, "source")
	})
}

// sortServices orders a stack's services by compose service name, then container
// name. The pre-sort order came from ContainerList, which makes no ordering
// guarantee.
func sortServices(services []map[string]interface{}) {
	sort.SliceStable(services, func(i, j int) bool {
		a, b := services[i], services[j]
		if sa, sb := mapString(a, "composeService"), mapString(b, "composeService"); sa != sb {
			return sa < sb
		}
		return mapString(a, "name") < mapString(b, "name")
	})
}

func handleStackDetail(w http.ResponseWriter, r *http.Request) {
	projectID := chi.URLParam(r, "projectId")
	all, err := podman.ContainerList(context.Background(), dockerctr.ListOptions{All: true})
	if err != nil {
		jsonErr(w, 500, "PODMAN_ERROR", err.Error())
		return
	}
	var pcs []dockerctr.Summary
	for _, c := range all {
		if compose.ComposeProjectLabel(c.Labels) == projectID {
			pcs = append(pcs, c)
		}
	}
	if len(pcs) == 0 {
		jsonErr(w, 404, "STACK_NOT_FOUND", "Stack not found or no containers")
		return
	}

	baseID := getBaseAppID(projectID)
	entry := apps.GetCatalogIndex()[baseID]

	// Catalog display metadata (x-yantr.ports) indexed by container port.
	// Prefer the entry whose service matches the running compose service so
	// multi-service apps keep their service dimension.
	type portMeta struct {
		label, displayProtocol, service string
	}
	byPort := map[uint16][]apps.PortInfo{}
	if entry != nil {
		for _, pi := range entry.Ports {
			if pi.Port >= 1 && pi.Port <= 65535 {
				byPort[uint16(pi.Port)] = append(byPort[uint16(pi.Port)], pi)
			}
		}
	}
	lookupMeta := func(privatePort uint16, composeService string) portMeta {
		cands := byPort[privatePort]
		if len(cands) == 0 {
			return portMeta{}
		}
		for _, c := range cands {
			if c.Service != "" && c.Service == composeService {
				return portMeta{label: c.Label, displayProtocol: c.Protocol, service: c.Service}
			}
		}
		// No service-specific match — use the first entry for this port.
		return portMeta{label: cands[0].Label, displayProtocol: cands[0].Protocol, service: cands[0].Service}
	}
	primaryServices := map[string]bool{}
	if entry != nil {
		for _, pi := range entry.Ports {
			if pi.Service != "" {
				primaryServices[pi.Service] = true
			}
		}
	}

	portMap := map[string]map[string]interface{}{}
	for _, c := range pcs {
		composeService := compose.ComposeServiceLabel(c.Labels)
		if composeService == "" {
			composeService = strings.TrimPrefix(c.Names[0], "/")
		}
		for _, p := range c.Ports {
			if p.PublicPort == 0 {
				continue
			}
			meta := lookupMeta(p.PrivatePort, composeService)
			svc := meta.service
			if svc == "" {
				svc = coalesce(composeService, strings.TrimPrefix(c.Names[0], "/"), "unknown")
			}
			key := fmt.Sprintf("%d:%d:%s:%s", p.PublicPort, p.PrivatePort, p.Type, svc)
			if _, ok := portMap[key]; !ok {
				portMap[key] = map[string]interface{}{
					"hostPort": p.PublicPort, "containerPort": p.PrivatePort,
					"protocol": p.Type, "service": svc,
					"label": meta.label, "displayProtocol": meta.displayProtocol,
				}
			}
		}
	}
	pPorts := make([]map[string]interface{}, 0, len(portMap))
	for _, p := range portMap {
		pPorts = append(pPorts, p)
	}
	sortPublishedPorts(pPorts)

	var services []map[string]interface{}
	for _, c := range pcs {
		lbl := parseAppLabels(c.Labels)
		info, err := podman.ContainerInspect(context.Background(), c.ID)
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
		sortMounts(mounts)
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
		composeService := compose.ComposeServiceLabel(c.Labels)
		// Primary = service that owns a display port in x-yantr.ports.
		// Falls back to managed-stack membership when catalog has no ports.
		isPrimary := primaryServices[composeService]
		if entry != nil && len(entry.Ports) == 0 {
			isPrimary = lbl.App != ""
		}
		services = append(services, map[string]interface{}{
			"id": c.ID, "name": name, "composeService": composeService,
			"image": c.Image, "state": c.State, "status": c.Status, "created": c.Created,
			"rawPorts": c.Ports, "mounts": mounts, "networks": networks,
			"service":       coalesce(composeService, lbl.Service, name),
			"hasYantrLabel": isPrimary,
		})
	}
	sortServices(services)

	var appInfo interface{}
	if entry != nil {
		appInfo = map[string]interface{}{
			"name": entry.Name, "logo": entry.Logo, "tags": entry.Tags,
			"ports": entry.Ports, "short_description": entry.ShortDescription,
			"website": entry.Website,
		}
	}

	jsonResp(w, 200, map[string]interface{}{
		"success": true,
		"stack": map[string]interface{}{
			"projectId": projectID, "appId": baseID, "app": appInfo,
			"publishedPorts": pPorts, "services": services,
		},
	})
}

// handleStackEnv returns the stored environment for a project instance so the
// deploy form can prefill it when editing an existing install. Only the
// project .env file is returned — extra vars baked into the generated compose
// file are not recoverable here and must be re-entered.
func handleStackEnv(w http.ResponseWriter, r *http.Request) {
	projectID := chi.URLParam(r, "projectId")
	if projectID == "" {
		jsonErr(w, 400, "PROJECT_ID_REQUIRED", "projectId is required")
		return
	}
	baseID := getBaseAppID(projectID)
	if !validAppID.MatchString(baseID) {
		jsonErr(w, 400, "INVALID_PROJECT_ID", "Invalid project ID")
		return
	}

	// Only expose env for a project that is actually running.
	all, err := podman.ContainerList(context.Background(), dockerctr.ListOptions{All: true})
	if err != nil {
		jsonErr(w, 500, "PODMAN_ERROR", err.Error())
		return
	}
	running := false
	for _, c := range all {
		if compose.ComposeProjectLabel(c.Labels) == projectID {
			running = true
			break
		}
	}
	if !running {
		jsonErr(w, 404, "STACK_NOT_FOUND", "Stack not found or no containers")
		return
	}

	appPath := filepath.Join(apps.GetAppsDir(), baseID)
	if _, statErr := os.Stat(filepath.Join(appPath, "compose.yml")); statErr != nil {
		jsonErr(w, 404, "APP_NOT_FOUND", "App not found")
		return
	}

	env, err := compose.LoadProjectEnv(appPath, projectID)
	if err != nil {
		jsonResp(w, 200, map[string]interface{}{
			"success": true, "projectId": projectID, "exists": false,
			"env": map[string]string{},
		})
		return
	}
	jsonResp(w, 200, map[string]interface{}{
		"success": true, "projectId": projectID, "exists": true, "env": env,
	})
}

func handleStackDelete(w http.ResponseWriter, r *http.Request) {
	projectID := chi.URLParam(r, "projectId")

	all, err := podman.ContainerList(context.Background(), dockerctr.ListOptions{All: true})
	if err != nil {
		jsonErr(w, 500, "PODMAN_ERROR", err.Error())
		return
	}

	var projectContainers []dockerctr.Summary
	for _, c := range all {
		if compose.ComposeProjectLabel(c.Labels) == projectID {
			projectContainers = append(projectContainers, c)
		}
	}

	if len(projectContainers) == 0 {
		jsonErr(w, 404, "STACK_NOT_FOUND", "No containers found for this stack")
		return
	}

	// Tell the watchdog to back off while we tear the stack down, so it
	// doesn't race us by restarting containers mid-removal.
	supervisor.MarkStackRemoving(projectID)
	defer supervisor.UnmarkStackRemoving(projectID)

	// Acquired before MarkStackRemoving so the watchdog also stays out of the
	// way of a teardown that is queued behind a deploy: without this, a pending
	// `down` has not set the flag yet and the watchdog can restart a container
	// the teardown is about to kill.
	release, locked := shared.TryLockProject(projectID)
	if !locked {
		jsonErr(w, 409, "PROJECT_BUSY",
			fmt.Sprintf("Another operation is already in progress for '%s'. Retry shortly.", projectID))
		return
	}
	defer release()

	baseID := getBaseAppID(projectID)
	appPath := filepath.Join(apps.GetAppsDir(), baseID)
	ref := compose.GetProjectComposeRef(appPath, projectID)

	if _, statErr := os.Stat(ref.ComposePath); statErr == nil {
		if cmdName, cmdArgs, err := getComposeCommand(); err == nil {
			env, _ := compose.GetComposeProcessEnv(appPath, projectID, podman.SocketPath, podman.HostSocket())
			args := append(cmdArgs, "-p", projectID, "-f", ref.ComposeFile, "down")
			shared.Log("info", fmt.Sprintf("[stack] removing: project=%s", projectID))
			job := globalJobs.Create("stack_delete", projectID, fmt.Sprintf("Delete stack %s", projectID))
			job.SetProgress(fmt.Sprintf("Removing stack %s...", projectID))
			downCtx, downCancel := context.WithTimeout(context.Background(), spawnTimeoutMedium)
			out, errStr, exitCode, err := spawnExecJob(downCtx, job, cmdName, args, env, appPath)
			downCancel()
			if exitCode == 0 {
				shared.Log("info", fmt.Sprintf("[stack] removed: project=%s", projectID))
				compose.DeleteProjectCompose(appPath, projectID)
				supervisor.RecordStackRemoved(projectID)
				forgetStackContainers(projectContainers)
				job.Complete(map[string]interface{}{"success": true, "removed": true})
				jsonResp(w, 200, map[string]interface{}{
					"success": true,
					"jobId":   job.ID,
					"message": fmt.Sprintf("Stack '%s' removed successfully", projectID),
					"removed": true,
				})
				return
			}
			job.Fail(fmt.Errorf("compose down failed: %s", coalesce(errStr, out)), exitCode)
			shared.Log("error", fmt.Sprintf("[stack] compose down failed: project=%s exit=%d err=%v", projectID, exitCode, err))
			jsonErr(w, 500, "STACK_REMOVE_FAILED", fmt.Sprintf("podman compose down failed (exit %d)", exitCode))
			return
		}
	}

	for _, c := range projectContainers {
		id := c.ID
		if c.State == "running" {
			_ = podman.ContainerStop(context.Background(), id, dockerctr.StopOptions{})
		}
		_ = podman.ContainerRemove(context.Background(), id, dockerctr.RemoveOptions{})
		supervisor.ForgetContainer(id)
	}

	compose.DeleteProjectCompose(appPath, projectID)
	supervisor.RecordStackRemoved(projectID)

	jsonResp(w, 200, map[string]interface{}{
		"success": true,
		"message": fmt.Sprintf("Stack '%s' removed successfully", projectID),
	})
}

// forgetStackContainers drops watchdog restart history for every container in a
// torn-down stack.
func forgetStackContainers(ctrs []dockerctr.Summary) {
	for _, c := range ctrs {
		supervisor.ForgetContainer(c.ID)
	}
}

func handleStackRestart(w http.ResponseWriter, r *http.Request) {
	projectID := chi.URLParam(r, "projectId")

	release, locked := shared.TryLockProject(projectID)
	if !locked {
		jsonErr(w, 409, "PROJECT_BUSY",
			fmt.Sprintf("Another operation is already in progress for '%s'. Retry shortly.", projectID))
		return
	}
	defer release()

	baseID := getBaseAppID(projectID)
	appPath := filepath.Join(apps.GetAppsDir(), baseID)
	ref := compose.GetProjectComposeRef(appPath, projectID)

	if _, statErr := os.Stat(ref.ComposePath); statErr != nil {
		jsonErr(w, 404, "STACK_NOT_FOUND", "Stack compose file not found")
		return
	}

	cmdName, cmdArgs, err := getComposeCommand()
	if err != nil {
		jsonErr(w, 500, "COMPOSE_NOT_FOUND", "podman compose command not found")
		return
	}

	env, _ := compose.GetComposeProcessEnv(appPath, projectID, podman.SocketPath, podman.HostSocket())
	args := append(cmdArgs, "-p", projectID, "-f", ref.ComposeFile, "restart")
	shared.Log("info", fmt.Sprintf("[stack] restarting: project=%s", projectID))
	job := globalJobs.Create("stack_restart", projectID, fmt.Sprintf("Restart stack %s", projectID))
	job.SetProgress(fmt.Sprintf("Restarting stack %s...", projectID))
	restartCtx, restartCancel := context.WithTimeout(context.Background(), spawnTimeoutMedium)
	out, errStr, exitCode, err := spawnExecJob(restartCtx, job, cmdName, args, env, appPath)
	restartCancel()

	if exitCode == 0 {
		shared.Log("info", fmt.Sprintf("[stack] restarted: project=%s", projectID))
		supervisor.RecordStackDeployed(projectID, baseID)
		job.Complete(map[string]interface{}{"success": true, "restarted": true})
		jsonResp(w, 200, map[string]interface{}{
			"success": true,
			"jobId":   job.ID,
			"message": fmt.Sprintf("Stack '%s' restarted successfully", projectID),
		})
		return
	}

	job.Fail(fmt.Errorf("compose restart failed: %s", coalesce(errStr, out)), exitCode)
	shared.Log("error", fmt.Sprintf("[stack] compose restart failed: project=%s exit=%d err=%v", projectID, exitCode, err))
	jsonErr(w, 500, "STACK_RESTART_FAILED", fmt.Sprintf("podman compose restart failed (exit %d)", exitCode))
}
