package supervisor

import (
	"testing"
	"time"
)

// resetRestartHistory clears package state so tests do not observe each other.
func resetRestartHistory() {
	restartHistoryMu.Lock()
	defer restartHistoryMu.Unlock()
	restartHistory = make(map[string][]time.Time)
}

func restartHistoryLen() int {
	restartHistoryMu.Lock()
	defer restartHistoryMu.Unlock()
	return len(restartHistory)
}

// A container that flaps is not restarted, and the decision must be stable for
// the whole window.
//
// The threshold is deliberately ">5 deaths", matching the watchdog's own log
// message ("crashed repeatedly (>5 times in 60s)"). isFlapping records the death
// before deciding, so the 6th death inside the window is the first one
// suppressed — by which point 5 restarts have already been issued.
func TestIsFlappingAfterRepeatedDeaths(t *testing.T) {
	resetRestartHistory()
	const id = "flappy"

	for i := 1; i <= 5; i++ {
		if isFlapping(id) {
			t.Fatalf("flap reported on death %d, want suppression to start at the 6th", i)
		}
	}
	if !isFlapping(id) {
		t.Error("6 deaths inside the window should report a flap")
	}
	if !isFlapping(id) {
		t.Error("a flapping container should stay flapping")
	}
}

// Deaths older than the window must not accumulate into a flap verdict.
func TestIsFlappingIgnoresOldDeaths(t *testing.T) {
	resetRestartHistory()
	const id = "recovering"

	// Seed four deaths just outside the window.
	old := time.Now().Add(-restartHistoryTTL - time.Minute)
	restartHistoryMu.Lock()
	restartHistory[id] = []time.Time{old, old, old, old}
	restartHistoryMu.Unlock()

	if isFlapping(id) {
		t.Error("stale deaths should not count toward a flap")
	}
}

// The map used to retain every container ID the watchdog ever saw, because
// isFlapping only prunes the key it is called with.
func TestSweepRestartHistoryDropsStaleEntries(t *testing.T) {
	resetRestartHistory()

	stale := time.Now().Add(-restartHistoryTTL - time.Minute)
	fresh := time.Now()

	restartHistoryMu.Lock()
	restartHistory["stale"] = []time.Time{stale, stale}
	restartHistory["fresh"] = []time.Time{fresh}
	restartHistory["empty"] = nil
	restartHistoryMu.Unlock()

	sweepRestartHistory()

	if got := restartHistoryLen(); got != 1 {
		t.Fatalf("after sweep map holds %d entries, want 1 (only the fresh one)", got)
	}
	restartHistoryMu.Lock()
	_, stillThere := restartHistory["fresh"]
	restartHistoryMu.Unlock()
	if !stillThere {
		t.Error("sweep dropped an entry that is still inside the window")
	}
}

func TestForgetContainer(t *testing.T) {
	resetRestartHistory()

	restartHistoryMu.Lock()
	restartHistory["doomed"] = []time.Time{time.Now()}
	restartHistory["kept"] = []time.Time{time.Now()}
	restartHistoryMu.Unlock()

	ForgetContainer("doomed")
	ForgetContainer("") // must be a no-op, not a panic

	restartHistoryMu.Lock()
	_, doomedThere := restartHistory["doomed"]
	_, keptThere := restartHistory["kept"]
	restartHistoryMu.Unlock()

	if doomedThere {
		t.Error("ForgetContainer did not drop the entry")
	}
	if !keptThere {
		t.Error("ForgetContainer dropped an unrelated entry")
	}
}
