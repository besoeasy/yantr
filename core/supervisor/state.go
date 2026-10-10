package supervisor

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"core/compose"
	"core/shared"
)

// StackState represents the persisted state of a deployed compose stack.
type StackState struct {
	ProjectID string `json:"projectId"`
	AppID     string `json:"appId"`
	Status    string `json:"status"` // "running" or "stopped"
	UpdatedAt int64  `json:"updatedAt"`

	// StoppedServices lists the compose service names the user intentionally
	// stopped, for stacks whose Status is still "running".
	//
	// Desired state is tracked per service rather than per container because
	// container IDs are ephemeral — `compose up -d` recreates containers with new
	// IDs — whereas the compose service name is stable.
	//
	// This is what makes "stop one service of a multi-service stack" behave:
	// the project stays "running" so its other services keep crash recovery,
	// and resuscitate brings the non-stopped subset back without resurrecting
	// the one the user stopped. Before this existed, stopping any single service
	// marked the whole project stopped, which silently disabled the watchdog and
	// boot resuscitation for every sibling service.
	//
	// Absent in older state.json files; nil then correctly means "nothing
	// intentionally stopped".
	StoppedServices []string `json:"stoppedServices,omitempty"`
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

	// removingMu guards the in-memory set of stacks currently being torn down.
	// This is separate from persisted state so a failed teardown doesn't
	// permanently flip a stack to "stopped".
	removingMu sync.Mutex
	removing   = make(map[string]bool)
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

	if appsDir != "" {
		// The pattern comes from the compose package, which owns the filename
		// convention, so the glob cannot drift from what is actually written.
		matches, err := filepath.Glob(filepath.Join(appsDir, "*", compose.ProjectComposeGlobPattern))
		if err == nil {
			for _, m := range matches {
				projectID := compose.ProjectIDFromComposeFileName(filepath.Base(m))
				if projectID == "" {
					continue
				}
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

	saveStateLocked()
}

// saveStateLocked persists appState atomically. Callers must hold stateMu.
//
// The write is logged rather than silently dropped: it previously used
// `_ = os.WriteFile(...)`, so an unwritable or full /data failed invisibly and
// the in-memory state drifted away from disk with no indication of why.
func saveStateLocked() {
	p := stateFilePath()
	_ = os.MkdirAll(filepath.Dir(p), 0755)
	data, err := json.MarshalIndent(appState, "", "  ")
	if err != nil {
		shared.Log("error", "[supervisor] state marshal failed: "+err.Error())
		return
	}
	if err := shared.WriteFileAtomic(p, data, 0644); err != nil {
		shared.Log("error", "[supervisor] state write failed at "+p+": "+err.Error())
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
//
// This is the whole-project primitive, for stack-level stop. Prefer
// RecordStackServiceStopped for a single service: flipping the project to
// "stopped" also turns off watchdog crash recovery and boot resuscitation for
// every other service in the stack.
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

// RecordStackServiceStopped marks one compose service as intentionally stopped,
// leaving the rest of the stack running and recoverable.
//
// service may be empty, in which case there is no service dimension to record
// and the call is a no-op.
func RecordStackServiceStopped(projectID, service string) {
	service = strings.TrimSpace(service)
	if projectID == "" || service == "" {
		return
	}
	stateMu.Lock()
	defer stateMu.Unlock()
	s, ok := appState.Stacks[projectID]
	if !ok {
		// Unknown project: nothing to persist. IsStackRunning already reports
		// false for it, so crash recovery is off regardless.
		return
	}
	if !containsString(s.StoppedServices, service) {
		s.StoppedServices = append(s.StoppedServices, service)
		sort.Strings(s.StoppedServices)
	}
	s.UpdatedAt = time.Now().Unix()
	appState.Stacks[projectID] = s
	saveStateLocked()
}

// RecordStackServiceStarted clears the intentionally-stopped mark for one
// compose service, so the watchdog may revive it and boot resuscitation brings
// it back. Starting a service of a fully stopped project reopens the project.
func RecordStackServiceStarted(projectID, service string) {
	service = strings.TrimSpace(service)
	if projectID == "" || service == "" {
		return
	}
	stateMu.Lock()
	defer stateMu.Unlock()
	s, ok := appState.Stacks[projectID]
	if !ok {
		return
	}
	if filtered := removeString(s.StoppedServices, service); len(filtered) != len(s.StoppedServices) {
		s.StoppedServices = filtered
	}
	s.Status = "running"
	s.UpdatedAt = time.Now().Unix()
	appState.Stacks[projectID] = s
	saveStateLocked()
}

// IsServiceStopped reports whether a compose service was intentionally stopped.
func IsServiceStopped(projectID, service string) bool {
	stateMu.RLock()
	defer stateMu.RUnlock()
	s, ok := appState.Stacks[projectID]
	if !ok {
		return false
	}
	return containsString(s.StoppedServices, strings.TrimSpace(service))
}

// GetStackStoppedServices returns the intentionally-stopped services of a
// stack, or nil when the stack is unknown.
func GetStackStoppedServices(projectID string) []string {
	stateMu.RLock()
	defer stateMu.RUnlock()
	s, ok := appState.Stacks[projectID]
	if !ok {
		return nil
	}
	out := make([]string, len(s.StoppedServices))
	copy(out, s.StoppedServices)
	return out
}

// ShouldAutoRestart reports whether the watchdog may restart a container. The
// project must be running, not mid-teardown, and the container's own service
// must not have been stopped on purpose.
func ShouldAutoRestart(projectID, service string) bool {
	if !IsStackRunning(projectID) || IsStackRemoving(projectID) {
		return false
	}
	return !IsServiceStopped(projectID, service)
}

func containsString(list []string, want string) bool {
	for _, v := range list {
		if v == want {
			return true
		}
	}
	return false
}

func removeString(list []string, drop string) []string {
	out := list[:0]
	for _, v := range list {
		if v != drop {
			out = append(out, v)
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
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

// MarkStackRemoving records that a stack teardown is in progress so the
// watchdog won't auto-restart its containers mid-teardown.
func MarkStackRemoving(projectID string) {
	removingMu.Lock()
	defer removingMu.Unlock()
	removing[projectID] = true
}

// UnmarkStackRemoving clears the teardown-in-progress flag for a stack.
func UnmarkStackRemoving(projectID string) {
	removingMu.Lock()
	defer removingMu.Unlock()
	delete(removing, projectID)
}

// IsStackRemoving reports whether a stack teardown is currently in progress.
func IsStackRemoving(projectID string) bool {
	removingMu.Lock()
	defer removingMu.Unlock()
	return removing[projectID]
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
