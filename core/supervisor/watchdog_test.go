package supervisor

import (
	"strings"
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

// shortID guards the log-only truncation of a container ID.
//
// The watchdog runs on its own goroutine (started with context.Background()), so
// middleware.Recoverer does not cover it: a panic in handleDieEvent takes down
// the whole process. A die event's Actor.ID is engine-supplied and only
// guaranteed non-empty, so the previous id[:12] panicked on any ID shorter than
// 12 characters.
func TestShortIDNeverPanicsOnShortIDs(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"", ""},
		{"a", "a"},
		{"abc", "abc"},
		{"abcdefghijk", "abcdefghijk"},           // 11 chars: one short of the old panic
		{"abcdefghijkl", "abcdefghijkl"},         // 12 chars: exactly the boundary
		{"abcdefghijklm", "abcdefghijkl"},        // 13 chars: truncates
		{"0123456789abcdef0123", "0123456789ab"}, // 20 chars: truncates to 12
		{"c3f1a2b4d5e6f7a8b9c0d1e2f3a4b5c6d7e8f9a0b1c2d3e4f5a6b7c8d9e0f1a2",
			"c3f1a2b4d5e6"},
	}
	for _, tc := range cases {
		got := shortID(tc.in)
		if got != tc.want {
			t.Errorf("shortID(%q) = %q, want %q", tc.in, got, tc.want)
		}
		if len(tc.in) >= 12 && len(got) != 12 {
			t.Errorf("shortID(%q) length = %d, want 12 for a full-length ID", tc.in, len(got))
		}
	}
}

// Every ID length from 0 up past the boundary must survive, which is the actual
// crash condition: any short Actor.ID reaching handleDieEvent.
func TestShortIDHandlesEveryLengthUpToBoundary(t *testing.T) {
	for n := 0; n <= 16; n++ {
		id := strings.Repeat("x", n)
		got := shortID(id)
		want := id
		if n > 12 {
			want = id[:12]
		}
		if got != want {
			t.Errorf("shortID(len %d) = %q, want %q", n, got, want)
		}
	}
}
