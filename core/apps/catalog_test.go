package apps

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestParseXyPortsShowDefaultsToWorkPort(t *testing.T) {
	services := map[string]interface{}{"web": nil}

	ports := parseXyPorts([]xyPort{
		{Port: 8080, Protocol: "HTTP", Label: "Web UI", Service: "web", Show: true},
		{Port: 51413, Protocol: "TCP", Label: "Peer Port", Service: "web"},
	}, services)

	if len(ports) != 2 {
		t.Fatalf("expected 2 ports, got %d", len(ports))
	}
	if !ports[0].Show {
		t.Errorf("port 8080 has show: true and must be user-facing")
	}
	// Deny by default: an omitted show key is a work port, not a visible one.
	if ports[1].Show {
		t.Errorf("port 51413 omits show and must be treated as a work port")
	}
}

func TestParseXyPortsShowUpgradesOnDuplicate(t *testing.T) {
	services := map[string]interface{}{"web": nil}

	// The same port declared twice, the work-port entry first. Reading order
	// must not decide visibility, or a duplicate declaration could hide a port
	// another entry exposes.
	ports := parseXyPorts([]xyPort{
		{Port: 8080, Protocol: "HTTP", Label: "Web UI", Service: "web"},
		{Port: 8080, Protocol: "HTTP", Label: "Web UI", Service: "web", Show: true},
	}, services)

	if len(ports) != 1 {
		t.Fatalf("expected dedup to keep 1 port, got %d", len(ports))
	}
	if !ports[0].Show {
		t.Errorf("show must be OR-ed across duplicate entries, got show=false")
	}
}

func TestParseXyPortsDropsUnknownServiceRegardlessOfShow(t *testing.T) {
	services := map[string]interface{}{"web": nil}

	ports := parseXyPorts([]xyPort{
		{Port: 8080, Protocol: "HTTP", Show: true, Label: "Ghost", Service: "does-not-exist"},
		{Port: 9090, Protocol: "HTTP", Show: true, Label: "Web UI", Service: "web"},
	}, services)

	if len(ports) != 1 {
		t.Fatalf("expected the unknown-service entry to be dropped, got %d ports", len(ports))
	}
	if ports[0].Port != 9090 {
		t.Errorf("expected the surviving entry to be 9090, got %d", ports[0].Port)
	}
}

func TestShownPortsSplitsAccessFromWork(t *testing.T) {
	a := &App{Ports: []PortInfo{
		{Port: 8080, Protocol: "HTTP", Label: "Web UI", Show: true},
		{Port: 51413, Protocol: "TCP", Label: "Peer Port"},
	}}

	shown := a.ShownPorts()
	if len(shown) != 1 || shown[0].Port != 8080 {
		t.Fatalf("expected only the show:true port, got %+v", shown)
	}
	// The full list stays intact: work ports are still published and still
	// identify their service.
	if len(a.Ports) != 2 {
		t.Errorf("ShownPorts must not mutate the full port list, got %d", len(a.Ports))
	}

	var nilApp *App
	if got := nilApp.ShownPorts(); got != nil {
		t.Errorf("expected nil for a nil app, got %+v", got)
	}
}

// ─── Catalog-wide audit ───────────────────────────────────────────────────────

// appsDirForTest resolves the repository's apps/ directory relative to this
// package so the audit below reads the real catalog rather than fixtures.
// ─── Developers ───────────────────────────────────────────────────────────────

func TestParseDevelopersAcceptsKnownPlatforms(t *testing.T) {
	got := parseDevelopers([]string{
		"github:besoeasy",
		"gitlab:username2",
		"codeberg:someone",
		"nostr:npub1qqqq",
	})

	want := []DeveloperInfo{
		{Platform: "github", Handle: "besoeasy", URL: "https://github.com/besoeasy"},
		{Platform: "gitlab", Handle: "username2", URL: "https://gitlab.com/username2"},
		{Platform: "codeberg", Handle: "someone", URL: "https://codeberg.org/someone"},
		{Platform: "nostr", Handle: "npub1qqqq", URL: "https://njump.me/npub1qqqq"},
	}

	if len(got) != len(want) {
		t.Fatalf("got %d developers %+v, want %d", len(got), got, len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("entry %d = %+v, want %+v", i, got[i], want[i])
		}
	}
}

// An entry that cannot be trusted is dropped, never guessed at: a wrong one
// becomes a dead link, and on GitHub an @mention of a stranger in every issue
// filed about the app.
func TestParseDevelopersDropsUntrustworthyEntries(t *testing.T) {
	got := parseDevelopers([]string{
		"besoeasy",             // no platform separator
		"https://github.com/x", // a URL, not platform:handle
		"mastodon:someone",     // platform not in the allow-list
		"github:",              // empty handle
		":besoeasy",            // empty platform
		"github:  ",            // whitespace-only handle
		"github:has space",     // handle cannot contain spaces
		"github:a/b",           // nor path separators
	})

	if len(got) != 0 {
		t.Fatalf("got %+v, want every entry dropped", got)
	}
}

func TestParseDevelopersNormalisesHandle(t *testing.T) {
	got := parseDevelopers([]string{"  GitHub:@Besoeasy  ", "GITHUB:besoeasy"})

	// Case is normalised on the forge platforms (they are case-insensitive), so
	// the two spellings must collapse to one developer rather than two.
	if len(got) != 1 {
		t.Fatalf("got %d developers %+v, want 1 after dedupe", len(got), got)
	}
	if got[0].Platform != "github" || got[0].Handle != "besoeasy" {
		t.Errorf("got %+v, want github:besoeasy", got[0])
	}
}

func TestParseDevelopersPreservesNostrKeyCase(t *testing.T) {
	// bech32 is lowercase by spec, but the field must not silently rewrite a key
	// it did not author.
	got := parseDevelopers([]string{"nostr:npub1ABCDEF"})
	if len(got) != 1 || got[0].Handle != "npub1ABCDEF" {
		t.Fatalf("got %+v, want the key preserved verbatim", got)
	}
}

func TestParseDevelopersNeverReturnsNil(t *testing.T) {
	// The UI distinguishes "no developers declared" from "field missing" by
	// shape, and a null would collapse the two.
	if got := parseDevelopers(nil); got == nil {
		t.Error("parseDevelopers(nil) returned nil, want an empty slice")
	}
}

func TestDevelopersForFiltersByPlatform(t *testing.T) {
	app := &App{Developers: parseDevelopers([]string{"github:a", "gitlab:a", "github:b"})}

	got := app.DevelopersFor("github")
	if len(got) != 2 || got[0] != "a" || got[1] != "b" {
		t.Errorf("DevelopersFor(github) = %v, want [a b]", got)
	}
	if got := app.DevelopersFor("nostr"); got != nil {
		t.Errorf("DevelopersFor(nostr) = %v, want nil", got)
	}
}

// Every declared developer in the real catalog must resolve to a profile URL.
// A platform that survives validation but derives no URL would render a link to
// nowhere.
func TestEveryCatalogDeveloperHasAProfileURL(t *testing.T) {
	dir := appsDirForTest(t)

	previous := appsDir
	appsDir = dir
	SetAppsDir(dir)
	t.Cleanup(func() { appsDir = previous })

	cat, err := loadCatalog()
	if err != nil {
		t.Fatalf("load catalog: %v", err)
	}

	for _, app := range cat.Apps {
		for _, d := range app.Developers {
			if !validDeveloperPlatforms[d.Platform] {
				t.Errorf("app %q declares unknown platform %q", app.ID, d.Platform)
			}
			if d.URL == "" {
				t.Errorf("app %q developer %s:%s has no profile URL", app.ID, d.Platform, d.Handle)
			}
			if strings.ContainsAny(d.Handle, " \t/:?#") {
				t.Errorf("app %q developer handle %q contains URL metacharacters", app.ID, d.Handle)
			}
		}
	}
}

func appsDirForTest(t *testing.T) string {
	t.Helper()
	dir, err := filepath.Abs(filepath.Join("..", "..", "apps"))
	if err != nil {
		t.Fatalf("resolve apps dir: %v", err)
	}
	if _, err := os.Stat(dir); err != nil {
		t.Skipf("apps catalog not present at %s", dir)
	}
	return dir
}

// Every app that publishes ports must mark at least one of them show: true.
// Without one, the stack view renders an empty port grid and the user has no
// way to reach the app at all. Apps with no user-facing surface (databases,
// file shares) are the legitimate exception and are allow-listed here.
func TestEveryAppWithPortsExposesAtLeastOneAccessPort(t *testing.T) {
	dir := appsDirForTest(t)

	previous := appsDir
	appsDir = dir
	SetAppsDir(dir)
	t.Cleanup(func() { appsDir = previous })

	cat, err := loadCatalog()
	if err != nil {
		t.Fatalf("load catalog: %v", err)
	}

	// Apps that genuinely publish no browser-reachable surface.
	allowed := map[string]bool{
		"mariadb": true, // 3306/tcp — database, no UI
		"mongodb": true, // 27017/tcp — database, no UI
		"samba":   true, // 445/tcp — SMB file share, no UI
	}

	for _, app := range cat.Apps {
		if len(app.Ports) == 0 {
			continue
		}
		shown := app.ShownPorts()
		if len(shown) == 0 {
			if allowed[app.ID] {
				continue
			}
			t.Errorf("app %q publishes %d port(s) but marks none show: true — "+
				"the stack view would render no reachable port", app.ID, len(app.Ports))
			continue
		}
		// A show:true port that a browser cannot open renders a dead "Open"
		// button, which is the exact noise this flag exists to remove.
		for _, p := range shown {
			if p.Protocol != "HTTP" && p.Protocol != "HTTPS" {
				t.Errorf("app %q port %d/%s is marked show: true but %s has no "+
					"browser UI to open", app.ID, p.Port, p.Protocol, p.Protocol)
			}
		}
	}
}

// Ports whose label names a protocol client or a diagnostics endpoint are
// plumbing even when they are HTTP, and must stay work ports. This guards the
// judgement calls the migration made by hand.
func TestDiagnosticPortsStayWorkPorts(t *testing.T) {
	dir := appsDirForTest(t)

	previous := appsDir
	appsDir = dir
	SetAppsDir(dir)
	t.Cleanup(func() { appsDir = previous })

	cat, err := loadCatalog()
	if err != nil {
		t.Fatalf("load catalog: %v", err)
	}

	for _, app := range cat.Apps {
		for _, p := range app.ShownPorts() {
			switch p.Label {
			case "Metrics / pprof", "Postgres client", "gRPC service":
				t.Errorf("app %q port %d %q is protocol plumbing and must not be "+
					"marked show: true", app.ID, p.Port, p.Label)
			}
		}
	}
}

// Rule 5 — prebuilt images only. A service declaring `build:` makes
// podman-compose compile the image on the host through Yantr's mounted socket,
// which is the remote-build path that needed a seccomp workaround no longer in
// the image. Nothing in the catalog exercises it, so the rule is enforced here
// rather than left to a hand-written Dockerfile that rule 5 also forbids.
func TestNoServiceDeclaresABuildStep(t *testing.T) {
	dir := appsDirForTest(t)

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read apps dir: %v", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		composePath := filepath.Join(dir, entry.Name(), "compose.yml")
		content, err := os.ReadFile(composePath)
		if err != nil {
			t.Errorf("app %q: read compose.yml: %v", entry.Name(), err)
			continue
		}
		var doc composeFile
		if err := yaml.Unmarshal(content, &doc); err != nil {
			t.Errorf("app %q: parse compose.yml: %v", entry.Name(), err)
			continue
		}
		for service, raw := range doc.Services {
			svc, ok := raw.(map[string]interface{})
			if !ok {
				continue
			}
			if _, found := svc["build"]; found {
				t.Errorf("app %q service %q declares build: — the catalog ships "+
					"prebuilt images only (AGENTS.md rule 5); a Dockerfile in the "+
					"app folder is not part of the format", entry.Name(), service)
			}
		}
	}
}
