package apps

import (
	"os"
	"path/filepath"
	"testing"
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
