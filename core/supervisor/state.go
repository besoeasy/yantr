package supervisor

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// StackState represents the persisted state of a deployed compose stack.
type StackState struct {
	ProjectID string `json:"projectId"`
	AppID     string `json:"appId"`
	Status    string `json:"status"` // "running" or "stopped"
	UpdatedAt int64  `json:"updatedAt"`
}

// State is the schema stored in state.json.
type State struct {
	Version int                   `json:"version"`
	Stacks  map[string]StackState `json:"stacks"`
}

// BootStatus tracks the resuscitation progress on startup for UI telemetry.
type BootStatus struct {
	Resuscitating bool   `json:"resuscitating"`
	Total         int    `json:"total"`
	Completed     int    `json:"completed"`
	CurrentStack  string `json:"currentStack,omitempty"`
	Message       string `json:"message,omitempty"`
}

var (
	stateMu    sync.RWMutex
	appState   = State{Version: 1, Stacks: make(map[string]StackState)}
	bootStatus = BootStatus{Resuscitating: false}
	bootMu     sync.RWMutex
)

func getDataDir() string {
	if d := os.Getenv("YANTR_DATA_DIR"); d != "" {
		return d
	}
	return "/data"
}

func stateFilePath() string {
	return filepath.Join(getDataDir(), "state.json")
}

// Init loads state.json from disk and discovers any existing project compose files.
func Init(appsDir string) {
	stateMu.Lock()
	defer stateMu.Unlock()

	p := stateFilePath()
	if data, err := os.ReadFile(p); err == nil {
		var s State
		if err := json.Unmarshal(data, &s); err == nil && s.Stacks != nil {
			appState = s
		}
	}

	if appState.Stacks == nil {
		appState.Stacks = make(map[string]StackState)
	}

	// Auto-discover deployed compose files if appsDir is provided
	if appsDir != "" {
		matches, err := filepath.Glob(filepath.Join(appsDir, "*", "compose.*.yml"))
		if err == nil {
			for _, m := range matches {
				base := filepath.Base(m) // compose.<projectID>.yml
				parts := strings.Split(base, ".")
				if len(parts) >= 3 {
					projectID := parts[1]
					appID := filepath.Base(filepath.Dir(m))
					if _, exists := appState.Stacks[projectID]; !exists {
						appState.Stacks[projectID] = StackState{
							ProjectID: projectID,
							AppID:     appID,
							Status:    "running",
							UpdatedAt: time.Now().Unix(),
						}
					}
				}
			}
		}
	}

	saveStateLocked()
}

func saveStateLocked() {
	p := stateFilePath()
	_ = os.MkdirAll(filepath.Dir(p), 0755)
	data, err := json.MarshalIndent(appState, "", "  ")
	if err == nil {
		_ = os.WriteFile(p, data, 0644)
	}
}

// RecordStackDeployed persists that a stack has been deployed and should be running.
func RecordStackDeployed(projectID, appID string) {
	stateMu.Lock()
	defer stateMu.Unlock()
	appState.Stacks[projectID] = StackState{
		ProjectID: projectID,
		AppID:     appID,
		Status:    "running",
		UpdatedAt: time.Now().Unix(),
	}
	saveStateLocked()
}

// RecordStackStopped marks that a stack was intentionally stopped by the user.
func RecordStackStopped(projectID string) {
	stateMu.Lock()
	defer stateMu.Unlock()
	if s, ok := appState.Stacks[projectID]; ok {
		s.Status = "stopped"
		s.UpdatedAt = time.Now().Unix()
		appState.Stacks[projectID] = s
		saveStateLocked()
	}
}

// RecordStackRemoved removes the stack from persisted state entirely.
func RecordStackRemoved(projectID string) {
	stateMu.Lock()
	defer stateMu.Unlock()
	delete(appState.Stacks, projectID)
	saveStateLocked()
}

// GetRunningStacks returns all stacks whose intended state is "running".
func GetRunningStacks() []StackState {
	stateMu.RLock()
	defer stateMu.RUnlock()
	var list []StackState
	for _, s := range appState.Stacks {
		if s.Status == "running" {
			list = append(list, s)
		}
	}
	return list
}

// IsStackRunning returns whether the stack is marked as running.
func IsStackRunning(projectID string) bool {
	stateMu.RLock()
	defer stateMu.RUnlock()
	s, ok := appState.Stacks[projectID]
	return ok && s.Status == "running"
}

// GetBootStatus returns the current resuscitation status.
func GetBootStatus() BootStatus {
	bootMu.RLock()
	defer bootMu.RUnlock()
	return bootStatus
}

// SetBootStatus updates the resuscitation status.
func SetBootStatus(status BootStatus) {
	bootMu.Lock()
	defer bootMu.Unlock()
	bootStatus = status
}
