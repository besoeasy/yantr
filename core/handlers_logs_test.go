package main

import (
	"strconv"
	"testing"
)

// The container logs endpoint used to pass ?tail straight through to podman,
// where "all" streams the container's entire log into process memory. It also
// accepted any junk and surfaced podman's 400 as a 500.
func TestParseLogTail(t *testing.T) {
	t.Run("accepted and clamped", func(t *testing.T) {
		cases := []struct {
			in   string
			want string
		}{
			{"", "100"},               // default
			{"  ", "100"},             // blank
			{"200", "200"},            // what the UI sends
			{"1", "1"},
			{"1000", "1000"},
			{"1001", "1000"},          // clamped
			{"999999", "1000"},        // clamped hard
			{" 500 ", "500"},          // trimmed
			{"all", "1000"},           // never passed through unbounded
			{"ALL", "1000"},           // case-insensitive
		}
		for _, tc := range cases {
			got, err := parseLogTail(tc.in)
			if err != nil {
				t.Errorf("parseLogTail(%q) unexpected error: %v", tc.in, err)
				continue
			}
			if got != tc.want {
				t.Errorf("parseLogTail(%q) = %q, want %q", tc.in, got, tc.want)
			}
		}
	})

	t.Run("rejected", func(t *testing.T) {
		// Note: a whitespace-only value is *not* here — it trims to empty and is
		// treated as absent, which is the sensible reading.
		for _, in := range []string{"0", "-1", "-100", "garbage", "10.5", "1e3", "0x10", "null", "1 2"} {
			if got, err := parseLogTail(in); err == nil {
				t.Errorf("parseLogTail(%q) = %q, want an error", in, got)
			}
		}
	})
}

// No accepted input may exceed the cap, whatever the caller sends.
func TestParseLogTailNeverExceedsCap(t *testing.T) {
	for _, in := range []string{"1", "500", "1000", "1001", "100000", "999999999", "2147483647", "all"} {
		got, err := parseLogTail(in)
		if err != nil {
			t.Fatalf("parseLogTail(%q): %v", in, err)
		}
		var n int
		n, err = strconv.Atoi(got)
		if err != nil {
			t.Fatalf("parseLogTail(%q) = %q, not an integer", in, got)
		}
		if n > maxLogTail {
			t.Errorf("parseLogTail(%q) = %d, exceeds cap %d", in, n, maxLogTail)
		}
		if n <= 0 {
			t.Errorf("parseLogTail(%q) = %d, must be positive", in, n)
		}
	}
}
