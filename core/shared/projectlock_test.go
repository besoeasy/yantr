package shared

import (
	"sync"
	"testing"
)

func TestTryLockProjectExcludes(t *testing.T) {
	release, ok := TryLockProject("immich")
	if !ok {
		t.Fatal("first acquisition should succeed")
	}
	if !IsProjectLocked("immich") {
		t.Error("project should report as locked while held")
	}

	// A second attempt on the same project must fail rather than block.
	release2, ok2 := TryLockProject("immich")
	if ok2 {
		t.Error("second acquisition of a held lock must fail")
	}
	// release2 must still be safe to call, or callers that `defer release()`
	// unconditionally would panic.
	release2()

	release()
	if IsProjectLocked("immich") {
		t.Error("project should be unlocked after release")
	}

	// Re-acquirable after release. The handle must be released again, or this
	// project stays locked for the rest of the test binary and every later test
	// that touches it spuriously fails.
	release3, ok3 := TryLockProject("immich")
	if !ok3 {
		t.Fatal("lock should be re-acquirable after release")
	}
	release3()
}

func TestProjectLockIsKeyed(t *testing.T) {
	releaseA, okA := TryLockProject("immich")
	if !okA {
		t.Fatal("failed to lock immich")
	}
	defer releaseA()

	// A different project must not be blocked.
	releaseB, okB := TryLockProject("nextcloud")
	if !okB {
		t.Fatal("an unrelated project must not be blocked by immich")
	}
	releaseB()

	if IsProjectLocked("nextcloud") {
		t.Error("nextcloud should be unlocked after release")
	}
}

func TestTryLockProjectEmpty(t *testing.T) {
	release, ok := TryLockProject("")
	if !ok {
		t.Error("an empty project ID should be treated as trivially acquirable")
	}
	release()
	if IsProjectLocked("") {
		t.Error("an empty project ID should never report as locked")
	}
}

func TestTryLockProjectIsReentrantSafeAcrossGoroutines(t *testing.T) {
	const goroutines = 32

	var wg sync.WaitGroup
	var mu sync.Mutex
	acquired := make([]bool, goroutines)

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			release, ok := TryLockProject("contended")
			mu.Lock()
			acquired[i] = ok
			mu.Unlock()
			if ok {
				release()
			}
		}(i)
	}
	wg.Wait()

	held := 0
	for _, ok := range acquired {
		if ok {
			held++
		}
	}
	if held == 0 {
		t.Error("at least one goroutine should have acquired the lock")
	}
	if IsProjectLocked("contended") {
		t.Error("lock should be free once all goroutines are done")
	}
}

// Repeatedly locking the same project must reuse one entry, not add one per
// call. Asserted as a before/after delta rather than an absolute size, because
// every other test in this package legitimately adds its own projects to the
// same table.
func TestProjectLockTableIsStable(t *testing.T) {
	countEntries := func() int {
		projectLocksMu.Lock()
		defer projectLocksMu.Unlock()
		return len(projectLocks)
	}

	const project = "stability-probe"
	// First acquisition creates the entry.
	release, ok := TryLockProject(project)
	if !ok {
		t.Fatal("expected to acquire")
	}
	release()
	before := countEntries()

	for i := 0; i < 200; i++ {
		release, ok := TryLockProject(project)
		if !ok {
			t.Fatalf("iteration %d: expected to acquire", i)
		}
		release()
	}

	if after := countEntries(); after != before {
		t.Errorf("lock table grew from %d to %d entries over 200 lock cycles for one project", before, after)
	}
}
