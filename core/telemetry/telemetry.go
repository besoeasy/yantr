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
	}
}

// Ping sends a telemetry event to the ntfy topic.
func Ping(event string, fields map[string]interface{}) {
	if !isEnabled {
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
			return
		}
		req.Header.Set("Title", event)
		req.Header.Set("Tags", event)

		client := &http.Client{Timeout: 5 * time.Second}
		_, _ = client.Do(req)
	}()
}

// TrackInstall sends an install event.
func TrackInstall(appID string) {
	Ping("install", map[string]interface{}{"app": appID})
}

// TrackSelfUpdate sends a self-update event.
func TrackSelfUpdate(updatedCount int, version string) {
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

		ctrs, err := docker.Client.ContainerList(context.Background(), container.ListOptions{All: true})
		if err != nil {
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
		info, err := docker.Client.Info(context.Background())
		if err != nil {
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
	ctrs, err := docker.Client.ContainerList(context.Background(), container.ListOptions{All: true})
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
		return
	}
	SendPresence(version)
	go func() {
		ticker := time.NewTicker(24 * time.Hour)
		defer ticker.Stop()
		for range ticker.C {
			SendPresence(version)
		}
	}()
}
