// Package telemetry implements ntfy.sh ping tracking for analytics.
package telemetry

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"core/docker"
	"core/shared"
	"core/system"

	"github.com/docker/docker/api/types/container"
)

var telemetryTopic = "https://ntfy.sh/yantr"
var isEnabled = true

func init() {
	if t := os.Getenv("YANTR_TELEMETRY_TOPIC"); t != "" {
		telemetryTopic = t
	}
	if os.Getenv("YANTR_TELEMETRY") == "false" {
		isEnabled = false
		shared.Log("info", "[telemetry] disabled via YANTR_TELEMETRY=false")
	} else {
		shared.Log("info", fmt.Sprintf("[telemetry] enabled → %s", telemetryTopic))
	}
}

// Ping sends a telemetry event to the ntfy topic.
func Ping(event string, fields map[string]interface{}) {
	if !isEnabled {
		shared.Log("info", fmt.Sprintf("[telemetry] skipped (disabled): event=%s", event))
		return
	}
	go func() {
		var parts []string
		for k, v := range fields {
			if v != nil && v != "" {
				parts = append(parts, fmt.Sprintf("%s=%v", k, v))
			}
		}
		body := event
		if len(parts) > 0 {
			body = event + " " + strings.Join(parts, " ")
		}

		req, err := http.NewRequestWithContext(context.Background(), "POST", telemetryTopic, bytes.NewBufferString(body))
		if err != nil {
			shared.Log("warn", fmt.Sprintf("[telemetry] failed to build request: event=%s err=%v", event, err))
			return
		}
		req.Header.Set("Title", event)
		req.Header.Set("Tags", event)

		client := &http.Client{Timeout: 5 * time.Second}
		resp, err := client.Do(req)
		if err != nil {
			shared.Log("warn", fmt.Sprintf("[telemetry] send failed: event=%s err=%v", event, err))
			return
		}
		defer resp.Body.Close()
		shared.Log("info", fmt.Sprintf("[telemetry] sent: event=%s %s → %d", event, strings.Join(parts, " "), resp.StatusCode))
	}()
}

// TrackInstall sends an install event.
func TrackInstall(appID string) {
	shared.Log("info", fmt.Sprintf("[telemetry] tracking install: app=%s", appID))
	Ping("install", map[string]interface{}{"app": appID})
}

// TrackSelfUpdate sends a self-update event.
func TrackSelfUpdate(updatedCount int, version string) {
	shared.Log("info", fmt.Sprintf("[telemetry] tracking self-update: updated=%d v=%s", updatedCount, version))
	Ping("selfupdate", map[string]interface{}{
		"updated": updatedCount,
		"v":       version,
	})
}

// TrackUpdatesForContainers tracks update events for specific container names.
func TrackUpdatesForContainers(containerNames []string) {
	if !isEnabled || len(containerNames) == 0 {
		return
	}
	go func() {
		wanted := map[string]bool{}
		for _, name := range containerNames {
			wanted[strings.TrimPrefix(name, "/")] = true
		}

		ctrs, err := docker.ContainerList(context.Background(), container.ListOptions{All: true})
		if err != nil {
			shared.Log("warn", "[telemetry] TrackUpdatesForContainers: failed to list containers: "+err.Error())
			return
		}

		appIDs := map[string]bool{}
		for _, c := range ctrs {
			var name string
			if len(c.Names) > 0 {
				name = strings.TrimPrefix(c.Names[0], "/")
			}
			appID := c.Labels["yantr.app"]
			if name != "" && wanted[name] && appID != "" {
				appIDs[appID] = true
			}
		}

		for appID := range appIDs {
			shared.Log("info", fmt.Sprintf("[telemetry] tracking update: app=%s", appID))
			Ping("update", map[string]interface{}{"app": appID})
		}
	}()
}

// SendPresence sends a daily presence event containing system info.
func SendPresence(version string) {
	if !isEnabled {
		return
	}
	go func() {
		shared.Log("info", "[telemetry] sending presence ping")
		info, err := docker.Info(context.Background())
		if err != nil {
			shared.Log("warn", "[telemetry] presence: failed to get docker info: "+err.Error())
			return
		}

		identity, _ := system.GetPublicIPIdentityCached(false)
		country := "??"
		if identity != nil && identity.CountryCode != "" {
			country = identity.CountryCode
		}

		osName := info.OperatingSystem
		if osName == "" || osName == "unknown" {
			osName = "unknown"
		} else {
			osName = strings.ReplaceAll(osName, "Debian GNU/Linux", "Debian")
		}

		ramGB := int64(0)
		if info.MemTotal > 0 {
			ramGB = info.MemTotal / (1024 * 1024 * 1024)
		}

		stacks := countYantrStacks()

		shared.Log("info", fmt.Sprintf("[telemetry] presence: country=%s os=%s arch=%s cores=%d ram=%dGB stacks=%d v=%s",
			country, osName, info.Architecture, info.NCPU, ramGB, stacks, version))

		Ping("presence", map[string]interface{}{
			"country": country,
			"os":      osName,
			"arch":    info.Architecture,
			"cores":   info.NCPU,
			"ram_gb":  ramGB,
			"stacks":  stacks,
			"v":       version,
		})
	}()
}

func countYantrStacks() int {
	ctrs, err := docker.ContainerList(context.Background(), container.ListOptions{All: true})
	if err != nil {
		return 0
	}
	projects := map[string]bool{}
	for _, c := range ctrs {
		app := c.Labels["yantr.app"]
		proj := c.Labels["com.docker.compose.project"]
		if app != "" && proj != "" {
			projects[proj] = true
		}
	}
	return len(projects)
}

// StartPresenceScheduler starts a background task to send presence daily.
func StartPresenceScheduler(version string) {
	if !isEnabled {
		shared.Log("info", "[telemetry] presence scheduler not started (disabled)")
		return
	}
	shared.Log("info", "[telemetry] presence scheduler started (interval=24h)")
	SendPresence(version)
	go func() {
		ticker := time.NewTicker(24 * time.Hour)
		defer ticker.Stop()
		for range ticker.C {
			shared.Log("info", "[telemetry] presence tick (24h interval)")
			SendPresence(version)
		}
	}()
}
