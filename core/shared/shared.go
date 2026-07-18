// Package shared provides logging, caching, and cross-module utilities.
package shared

import (
	"os"
	"path/filepath"
	"sync"
	"time"
)

// LogEntry represents a single log line stored in the circular buffer.
type LogEntry struct {
	Timestamp string `json:"timestamp"`
	Level     string `json:"level"`
	Message   string `json:"message"`
}

const MaxLogs = 1000

var (
	logMu sync.Mutex
	Logs  []LogEntry
)

// Log appends a log entry to the circular buffer and prints to stdout/stderr.
func Log(level, message string) {
	entry := LogEntry{
		Timestamp: time.Now().UTC().Format(time.RFC3339Nano),
		Level:     level,
		Message:   message,
	}

	logMu.Lock()
	Logs = append(Logs, entry)
	if len(Logs) > MaxLogs {
		Logs = Logs[1:]
	}
	logMu.Unlock()

	// Always print to stdout so Docker logs are useful
	println("[" + entry.Timestamp + "] [" + level + "] " + message)
}

// GetLogs returns a copy of the current log buffer (most-recent last).
func GetLogs(level string, limit int) []LogEntry {
	logMu.Lock()
	defer logMu.Unlock()

	filtered := make([]LogEntry, 0, len(Logs))
	for _, e := range Logs {
		if level == "" || e.Level == level {
			filtered = append(filtered, e)
		}
	}
	if limit > 0 && len(filtered) > limit {
		filtered = filtered[len(filtered)-limit:]
	}
	// Reverse for most-recent-first display
	for i, j := 0, len(filtered)-1; i < j; i, j = i+1, j-1 {
		filtered[i], filtered[j] = filtered[j], filtered[i]
	}
	return filtered
}

// GetBaseAppID strips instance suffixes (e.g. "myapp-2" → "myapp").
func GetBaseAppID(projectID string) string {
	if projectID == "" {
		return projectID
	}
	// Strip trailing -<number>
	for i := len(projectID) - 1; i >= 0; i-- {
		c := projectID[i]
		if c >= '0' && c <= '9' {
			continue
		}
		if c == '-' && i < len(projectID)-1 && i > 0 {
			return projectID[:i]
		}
		break
	}
	return projectID
}

// NowMs returns the current Unix timestamp in milliseconds.
func NowMs() int64 {
	return time.Now().UnixMilli()
}

// NormalizeAppLogo returns a logo URL. If logo.svg exists in appPath, returns a local API URL.
func NormalizeAppLogo(appPath string) string {
	if appPath != "" {
		svgPath := appPath + "/logo.svg"
		if info, err := os.Stat(svgPath); err == nil && !info.IsDir() {
			appID := filepath.Base(appPath)
			return "/api/apps/" + appID + "/logo"
		}
	}
	return ""
}
