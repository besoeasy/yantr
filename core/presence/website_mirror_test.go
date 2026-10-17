package presence

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"testing"
)

// The site keeps its own copy of this aggregation, because it has no backend:
// website/fleet.js does at build time and again in the browser what stats.go
// does in the app. The two cannot share code — a Go test cannot import a
// browser, and the site is built on a GitHub Pages runner that has Node and no
// Go — so the parts that must not drift are checked here instead, by reading
// website/fleet.js as text.
//
// This is the only wiring between the two implementations, which is why both
// values below have to stay parseable. What is not checked here is the shape of
// the aggregation itself: that one is kept in step by the comments on both
// sides, and by the site showing the same numbers the Fleet page shows.

// Matches an entry of FLEET_COUNTRIES, with a bare or a quoted key.
var jsCountryEntry = regexp.MustCompile(`(?m)^\s{2}'?([A-Z?]{2})'?:\s*'([^']*)',$`)

// Matches export const FLEET_LIST_LIMIT = 12; in website/fleet.js.
var jsListLimit = regexp.MustCompile(`(?m)^export const FLEET_LIST_LIMIT = (\d+);$`)

// fleetJSSource reads the site's copy of the aggregate. Skipped rather than
// failed when the website is not in the tree, the same way the app catalog
// tests treat apps/.
func fleetJSSource(t *testing.T) string {
	t.Helper()
	path := filepath.Join("..", "..", "website", "fleet.js")
	raw, err := os.ReadFile(filepath.Clean(path))
	if err != nil {
		t.Skipf("website not present at %s", path)
	}
	return string(raw)
}

// websiteCountries parses the country table out of the site's module.
func websiteCountries(t *testing.T, src string) map[string]string {
	t.Helper()
	start := strings.Index(src, "export const FLEET_COUNTRIES = {")
	if start < 0 {
		t.Fatal("website/fleet.js no longer declares 'export const FLEET_COUNTRIES = {'")
	}
	rest := src[start:]
	end := strings.Index(rest, "\n};")
	if end < 0 {
		t.Fatal("unterminated FLEET_COUNTRIES table in website/fleet.js")
	}

	out := map[string]string{}
	for _, m := range jsCountryEntry.FindAllStringSubmatch(rest[:end], -1) {
		out[m[1]] = m[2]
	}
	if len(out) < 100 {
		t.Fatalf("parsed %d country names from website/fleet.js, expected the whole table — has it been reformatted?", len(out))
	}
	return out
}

// TestWebsiteCountryTableMatchesApp holds the app's table, which is the source
// of truth, against the site's. A code that only one side knows renders as a
// bare code on yantr.org while the app shows a country name, which is the kind
// of drift nobody sees until someone reads both tables side by side.
func TestWebsiteCountryTableMatchesApp(t *testing.T) {
	js := websiteCountries(t, fleetJSSource(t))

	codes := make([]string, 0, len(countries))
	for code := range countries {
		codes = append(codes, code)
	}
	for code := range js {
		if _, ok := countries[code]; !ok {
			codes = append(codes, code)
		}
	}
	sort.Strings(codes)

	for _, code := range codes {
		app, inApp := countries[code]
		name, inSite := js[code]
		switch {
		case !inSite:
			t.Errorf("app knows %s (%q), the website table has no entry", code, app.Name)
		case !inApp:
			t.Errorf("website table knows %s (%q), the app's table has no entry", code, name)
		case name != app.Name:
			t.Errorf("%s: app calls it %q, the website calls it %q", code, app.Name, name)
		}
	}
}

// TestWebsiteListLimitMatchesApp holds the OS and app caps together: the site
// would quietly show a different number of rows than the Fleet page if these
// drifted apart.
func TestWebsiteListLimitMatchesApp(t *testing.T) {
	m := jsListLimit.FindStringSubmatch(fleetJSSource(t))
	if m == nil {
		t.Fatal("website/fleet.js no longer declares 'export const FLEET_LIST_LIMIT = <n>;'")
	}
	limit, err := strconv.Atoi(m[1])
	if err != nil {
		t.Fatalf("FLEET_LIST_LIMIT in website/fleet.js is not a number: %v", err)
	}
	if limit != fleetListLimit {
		t.Errorf("website/fleet.js lists %d OSes and %d apps, the app lists %d", limit, limit, fleetListLimit)
	}
}
