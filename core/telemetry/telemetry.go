// Package telemetry implements ntfy.sh ping tracking for analytics.
package telemetry

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
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
		telemetryTopic = strings.TrimRight(t, "/")
	}
	if os.Getenv("YANTR_TELEMETRY") == "false" {
		isEnabled = false
		shared.Log("info", "[telemetry] disabled via YANTR_TELEMETRY=false")
	} else {
		shared.Log("info", fmt.Sprintf("[telemetry] enabled → %s", telemetryTopic))
	}
}

// Event is the JSON body posted to the ntfy topic.
type Event struct {
	Event   string   `json:"event"`
	Node    string   `json:"node,omitempty"`
	TS      int64    `json:"ts,omitempty"`
	App     string   `json:"app,omitempty"`
	Apps    []string `json:"apps,omitempty"`
	Country string   `json:"country,omitempty"`
	OS      string   `json:"os,omitempty"`
	Arch    string   `json:"arch,omitempty"`
	Cores   int      `json:"cores,omitempty"`
	RAMGB   int64    `json:"ram_gb,omitempty"`
	Stacks  int      `json:"stacks,omitempty"`
	Version string   `json:"v,omitempty"`
	Updated int      `json:"updated,omitempty"`
}

type nodeFile struct {
	ID string `json:"id"`
}

var (
	nodeOnce sync.Once
	nodeID   string
)

func dataDir() string {
	if d := os.Getenv("YANTR_DATA_DIR"); d != "" {
		return d
	}
	return "/data"
}

// NodeID returns a stable anonymous id for this install.
func NodeID() string {
	nodeOnce.Do(func() {
		path := filepath.Join(dataDir(), "node.json")
		if raw, err := os.ReadFile(path); err == nil {
			var nf nodeFile
			if json.Unmarshal(raw, &nf) == nil && nf.ID != "" {
				nodeID = nf.ID
				return
			}
		}
		buf := make([]byte, 12)
		if _, err := rand.Read(buf); err != nil {
			nodeID = fmt.Sprintf("tmp-%d", time.Now().UnixNano())
			return
		}
		nodeID = hex.EncodeToString(buf)
		_ = os.MkdirAll(dataDir(), 0o755)
		body, _ := json.Marshal(nodeFile{ID: nodeID})
		_ = os.WriteFile(path, body, 0o600)
	})
	return nodeID
}

// TopicURL returns the configured ntfy topic.
func TopicURL() string {
	return telemetryTopic
}

// Enabled reports whether telemetry is on.
func Enabled() bool {
	return isEnabled
}

// Ping sends a telemetry event as JSON.
func Ping(event string, fields map[string]interface{}) {
	if !isEnabled {
		shared.Log("info", fmt.Sprintf("[telemetry] skipped (disabled): event=%s", event))
		return
	}
	go func() {
		payload := map[string]interface{}{
			"event": event,
			"node":  NodeID(),
			"ts":    time.Now().Unix(),
		}
		for k, v := range fields {
			if v == nil || v == "" {
				continue
			}
			payload[k] = v
		}
		body, err := json.Marshal(payload)
		if err != nil {
			shared.Log("warn", fmt.Sprintf("[telemetry] json encode failed: event=%s err=%v", event, err))
			return
		}

		req, err := http.NewRequestWithContext(context.Background(), "POST", telemetryTopic, bytes.NewReader(body))
		if err != nil {
			shared.Log("warn", fmt.Sprintf("[telemetry] failed to build request: event=%s err=%v", event, err))
			return
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Title", event)
		req.Header.Set("Tags", event)

		client := &http.Client{Timeout: 5 * time.Second}
		resp, err := client.Do(req)
		if err != nil {
			shared.Log("warn", fmt.Sprintf("[telemetry] send failed: event=%s err=%v", event, err))
			return
		}
		defer resp.Body.Close()
		shared.Log("info", fmt.Sprintf("[telemetry] sent: %s → %d", string(body), resp.StatusCode))
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
			country = strings.ToUpper(identity.CountryCode)
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

		apps := listYantrApps()
		stacks := len(apps)

		shared.Log("info", fmt.Sprintf("[telemetry] presence: country=%s os=%s arch=%s cores=%d ram=%dGB stacks=%d apps=%d v=%s",
			country, osName, info.Architecture, info.NCPU, ramGB, stacks, len(apps), version))

		Ping("presence", map[string]interface{}{
			"country": country,
			"os":      osName,
			"arch":    info.Architecture,
			"cores":   info.NCPU,
			"ram_gb":  ramGB,
			"stacks":  stacks,
			"apps":    apps,
			"v":       version,
		})
	}()
}

func listYantrApps() []string {
	ctrs, err := docker.ContainerList(context.Background(), container.ListOptions{All: true})
	if err != nil {
		return []string{}
	}
	seen := map[string]bool{}
	var apps []string
	for _, c := range ctrs {
		app := c.Labels["yantr.app"]
		if app == "" || seen[app] {
			continue
		}
		seen[app] = true
		apps = append(apps, app)
		if len(apps) >= 50 {
			break
		}
	}
	if apps == nil {
		return []string{}
	}
	return apps
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
