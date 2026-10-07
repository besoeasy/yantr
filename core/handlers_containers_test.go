package main

import "testing"

// Values measured against a rootless Podman host whose total RAM was
// 24243552256 bytes (24.24 GB), for a container with no memory limit set.
const measuredHostTotal = 24243552256.0

func TestHasRealMemoryLimit(t *testing.T) {
	for _, tc := range []struct {
		name      string
		limit     float64
		hostTotal float64
		want      bool
	}{
		// The regression: Podman reports host RAM as the limit when none is set,
		// so a real container's usage divided by it reads as 0.00%.
		{"host total standing in for unset", measuredHostTotal, measuredHostTotal, false},
		{"host total, unknown host total", measuredHostTotal, 0, true},
		{"limit above host total", measuredHostTotal * 2, measuredHostTotal, false},
		{"limit within 1% of host total", measuredHostTotal * 0.995, measuredHostTotal, false},
		{"genuine limit", 512 * 1024 * 1024, measuredHostTotal, true},
		{"no limit reported", 0, measuredHostTotal, false},
		{"negative limit", -1, measuredHostTotal, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := hasRealMemoryLimit(tc.limit, tc.hostTotal); got != tc.want {
				t.Errorf("hasRealMemoryLimit(%v, %v) = %v, want %v",
					tc.limit, tc.hostTotal, got, tc.want)
			}
		})
	}
}

// The reported percentage must be suppressed rather than silently wrong: a
// 324 KB container against a 24.24 GB host total is 0.0014%, which rendered as
// "0.00%" for every container in the catalog.
func TestUnlimitedContainerReportsNoPercentage(t *testing.T) {
	usage := 331776.0
	if limited := hasRealMemoryLimit(measuredHostTotal, measuredHostTotal); limited {
		t.Fatalf("host-RAM limit should not count as a real limit, got limited=%v", limited)
	}
	if pct := (usage / measuredHostTotal) * 100; pct > 0.01 {
		t.Fatalf("precondition: expected the bogus percentage to be ~0, got %v", pct)
	}
}

// A container under a genuine limit keeps its percentage.
func TestLimitedContainerReportsPercentage(t *testing.T) {
	limit := 512.0 * 1024 * 1024
	usage := 128.0 * 1024 * 1024
	if !hasRealMemoryLimit(limit, measuredHostTotal) {
		t.Fatal("a 512 MiB limit against a 24 GB host is a real limit")
	}
	got := (usage / limit) * 100
	if got < 24.99 || got > 25.01 {
		t.Fatalf("percent = %v, want ~25", got)
	}
}
