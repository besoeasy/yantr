// Package shared provides logging, caching, and cross-module utilities.
package shared

import (
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

// IsLikelyIPFSCid returns true if the string looks like a CIDv0 or CIDv1.
func IsLikelyIPFSCid(value string) bool {
	if len(value) == 46 && len(value) > 2 && value[:2] == "Qm" {
		return true
	}
	if len(value) > 20 && value[0] == 'b' {
		return true
	}
	return false
}

// NormalizeAppLogo converts a raw logo field to a full URL or returns empty string.
func NormalizeAppLogo(logoRaw string) string {
	if logoRaw == "" {
		return ""
	}
	if contains(logoRaw, "://") {
		return logoRaw
	}
	if IsLikelyIPFSCid(logoRaw) {
		return "https://ipfs.io/ipfs/" + logoRaw
	}
	return ""
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		func() bool {
			for i := 0; i <= len(s)-len(substr); i++ {
				if s[i:i+len(substr)] == substr {
					return true
				}
			}
			return false
		}())
}
