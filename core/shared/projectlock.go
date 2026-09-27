package shared

import "sync"

// Project lock — mutual exclusion per compose project.
//
// Deploy, stack delete, container delete and auto-update all read-modify-write
// the same two per-project files (.compose.<projectID>.yml and
// .env.<projectID>) and then shell out to `podman compose` for that project.
// With no exclusion, two overlapping requests interleave their writes and run
// two compose operations against the same project at once. The failure is
// sticky rather than loud: a delete can `compose down` and delete the compose
// file in the middle of a deploy, after which the deploy still reports success
// and state.json records a running stack whose compose file no longer exists.
//
// The existing `removing` flag in the supervisor package is not a lock: nothing
// acquires it, it is a broadcast flag read only by the watchdog, and two
// concurrent deletes both proceed.
//
// This is deliberately keyed rather than a single global mutex so that unrelated
// projects stay fully parallel. Taking it inside compose.WriteProjectCompose
// would be wrong: that would only cover the file write, not the compose run that
// follows, which is where the actual race is.
var (
	projectLocksMu sync.Mutex
	projectLocks   = make(map[string]*sync.Mutex)
)

// TryLockProject attempts to take the lock for projectID without blocking.
//
// Returns release (always non-nil, safe to defer) and true when the lock was
// acquired; false when someone else already holds it. Non-blocking on purpose:
// the HTTP handlers should answer 409 rather than sit on a mutex while the
// client's write deadline expires, and the reaper must skip a project for this
// tick rather than stall reaping every other project behind a 10-minute
// `compose down`.
func TryLockProject(projectID string) (release func(), acquired bool) {
	if projectID == "" {
		return func() {}, true
	}

	projectLocksMu.Lock()
	mu, ok := projectLocks[projectID]
	if !ok {
		mu = &sync.Mutex{}
		projectLocks[projectID] = mu
	}
	projectLocksMu.Unlock()

	if !mu.TryLock() {
		return func() {}, false
	}
	return mu.Unlock, true
}

// IsProjectLocked reports whether a project is currently locked.
func IsProjectLocked(projectID string) bool {
	if projectID == "" {
		return false
	}
	projectLocksMu.Lock()
	mu, ok := projectLocks[projectID]
	projectLocksMu.Unlock()
	if !ok {
		return false
	}
	// TryLock is the only non-destructive probe available on sync.Mutex: if it
	// succeeds, nothing held it, so release immediately and report unlocked.
	if mu.TryLock() {
		mu.Unlock()
		return false
	}
	return true
}
