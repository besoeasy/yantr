package main

import (
	"bytes"
	"context"
	"core/podman"
	"core/shared"
	"encoding/json"
	"fmt"
	"net/http"
	"os/exec"
	"strings"
	"time"

	dockertypes "github.com/docker/docker/api/types"
	dockerctr "github.com/docker/docker/api/types/container"
	dockervol "github.com/docker/docker/api/types/volume"
	"github.com/go-chi/chi/v5"
)

func handleVolumes(w http.ResponseWriter, r *http.Request) {
	vols, err := podman.VolumeList(context.Background(), dockervol.ListOptions{})
	if err != nil {
		jsonErr(w, 500, "VOLUMES_FETCH_FAILED", err.Error())
		return
	}

	// Fetch real volume sizes via DiskUsage (VolumeList doesn't return sizes).
	volSizes := map[string]int64{}
	if du, duErr := podman.DiskUsage(context.Background(), dockertypes.DiskUsageOptions{}); duErr == nil {
		for _, v := range du.Volumes {
			if v != nil && v.UsageData != nil && v.UsageData.Size >= 0 {
				volSizes[v.Name] = v.UsageData.Size
			}
		}
	}

	ctrs, _ := podman.ContainerList(context.Background(), dockerctr.ListOptions{All: true})
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
		sz := volSizes[v.Name]
		enriched = append(enriched, volItem{
			Name: v.Name, Driver: v.Driver, Mountpoint: v.Mountpoint, CreatedAt: v.CreatedAt,
			Labels: v.Labels, IsBrowsing: browserRegistry.IsBrowsing(v.Name), IsUsed: usedVols[v.Name],
			SizeBytes: sz, Size: fmt.Sprintf("%.2f", float64(sz)/(1024*1024)),
		})
	}
	if enriched == nil {
		enriched = []volItem{}
	}

	var used, unused []volItem
	var totalBytes, unusedBytes int64
	for _, v := range enriched {
		totalBytes += v.SizeBytes
		if v.IsUsed {
			used = append(used, v)
		} else {
			unused = append(unused, v)
			unusedBytes += v.SizeBytes
		}
	}
	jsonResp(w, 200, map[string]interface{}{
		"success": true, "total": len(enriched), "used": len(used), "unused": len(unused),
		"totalSize":  fmt.Sprintf("%.2f", float64(totalBytes)/(1024*1024)),
		"unusedSize": fmt.Sprintf("%.2f", float64(unusedBytes)/(1024*1024)),
		"volumes":    enriched, "usedVolumes": used, "unusedVolumes": unused,
	})
}

func handleVolumeDelete(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	if err := podman.VolumeRemove(context.Background(), name, false); err != nil {
		if strings.Contains(err.Error(), "in use") {
			jsonErr(w, 409, "VOLUME_IN_USE", fmt.Sprintf("Volume '%s' is currently in use", name))
			return
		}
		jsonErr(w, 500, "VOLUME_REMOVE_FAILED", err.Error())
		return
	}
	jsonResp(w, 200, map[string]interface{}{"success": true, "message": fmt.Sprintf("Volume '%s' removed", name), "volume": name})
}

func isValidVolumeName(name string) bool {
	if len(name) == 0 || len(name) > 255 {
		return false
	}
	for _, r := range name {
		if !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' || r == '-' || r == '.') {
			return false
		}
	}
	return true
}

func handleVolumeExport(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	if !isValidVolumeName(name) {
		jsonErr(w, 400, "INVALID_VOLUME_NAME", "Invalid volume name")
		return
	}

	vols, err := podman.VolumeList(r.Context(), dockervol.ListOptions{})
	if err != nil {
		jsonErr(w, 500, "PODMAN_ERROR", err.Error())
		return
	}
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

	timestamp := time.Now().Format("20060102-150405")
	filename := fmt.Sprintf("%s-backup-%s.tar.gz", name, timestamp)

	w.Header().Set("Content-Type", "application/gzip")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	w.WriteHeader(http.StatusOK)

	if f, ok := w.(http.Flusher); ok {
		f.Flush()
	}

	shared.Log("info", fmt.Sprintf("[volume-export] starting backup export for volume '%s'", name))

	// Run ephemeral container to tar and gzip the volume content
	// Read-only mount (:ro) ensures the running app data is not modified
	cmd := exec.CommandContext(r.Context(), "podman", "run", "--rm", "-v", name+":/volume_data:ro", "docker.io/library/alpine:latest", "tar", "czf", "-", "-C", "/volume_data", ".")
	cmd.Stdout = w

	var stderrBuf bytes.Buffer
	cmd.Stderr = &stderrBuf

	if err := cmd.Run(); err != nil {
		if r.Context().Err() != nil {
			shared.Log("info", fmt.Sprintf("[volume-export] download cancelled by client for volume '%s'", name))
			return
		}
		shared.Log("error", fmt.Sprintf("[volume-export] failed to export volume '%s': %v, stderr: %s", name, err, stderrBuf.String()))
		return
	}

	shared.Log("info", fmt.Sprintf("[volume-export] successfully completed backup for volume '%s'", name))
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

	vols, _ := podman.VolumeList(context.Background(), dockervol.ListOptions{})
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
