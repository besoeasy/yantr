package apps

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"core/nip19"

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

// Externally-verified keys, the same vectors core/nip19 tests against.
const (
	fiatjafNpub = "npub180cvv07tjdrrgpa0j7j7tmnyl2yr6yr7l8j4s3evf6u64th6gkwsyjh6w6"
	jackNpub    = "npub1sg6plzptd64u62a878hep2kev88swjh3tw00gjsfl8f237lmu63q0uf63m"
	// A real-world nprofile published by njump.me, relay hints and all.
	fiatjafNprofile = "nprofile1qqsrhuxx8l9ex335q7he0f09aej04zpazpl0ne2cgukyawd24mayt8gpz3mhxue69uhhyetvv9ujuerpd46hxtnfduqs6amnwvaz7tmwdaejumr0dsfg70ls"

	// Well-formed bech32 that is not a profile key, and one that is a key with a
	// broken checksum.
	noteKey32          = "note1sg6plzptd64u62a878hep2kev88swjh3tw00gjsfl8f237lmu63q7k28gn"
	nsecKey32          = "nsec1sg6plzptd64u62a878hep2kev88swjh3tw00gjsfl8f237lmu63qr2zmhw"
	npubBrokenChecksum = "npub1sg6plzptd64u62a878hep2kev88swjh3tw00gjsfl8f237lmu63q0uf63q"
)

func TestParseDevelopersAcceptsNostrKeys(t *testing.T) {
	got := parseDevelopers([]string{fiatjafNpub, jackNpub})

	want := []DeveloperInfo{
		{Key: fiatjafNpub, URL: "https://njump.me/" + fiatjafNpub},
		{Key: jackNpub, URL: "https://njump.me/" + jackNpub},
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

// An nprofile is a profile identifier too — it is what a client hands out when it
// wants to suggest where to find someone, and it carries relay hints the field
// has no use for but no reason to reject.
func TestParseDevelopersAcceptsAnNprofile(t *testing.T) {
	got := parseDevelopers([]string{fiatjafNprofile})

	if len(got) != 1 {
		t.Fatalf("got %+v, want 1 developer", got)
	}
	if got[0].Key != fiatjafNprofile || got[0].URL != "https://njump.me/"+fiatjafNprofile {
		t.Errorf("got %+v, want the nprofile kept with its njump URL", got[0])
	}
}

// The same person written as an npub and as an nprofile is one developer, not
// two. This is the reason deduplication compares the decoded pubkey rather than
// the string: the two spellings are different strings and the same account, and a
// catalog containing both would otherwise credit them twice and build two pages.
func TestParseDevelopersDeduplicatesOneAccountWrittenTwoWays(t *testing.T) {
	got := parseDevelopers([]string{fiatjafNpub, fiatjafNprofile})

	if len(got) != 1 {
		t.Fatalf("got %d developers %+v, want 1", len(got), got)
	}
	// The first spelling survives: the field does not rewrite what it was given.
	if got[0].Key != fiatjafNpub {
		t.Errorf("key = %q, want the first spelling %q kept", got[0].Key, fiatjafNpub)
	}
}

// A bech32 checksum is not a judgement call, so this test does not need to reason
// about trust the way the old forge-logging allow-list did: anything that is not
// a profile key is dropped, full stop.
func TestParseDevelopersDropsEverythingThatIsNotAKey(t *testing.T) {
	got := parseDevelopers([]string{
		"",                     // empty
		"   ",                  // whitespace only
		"besoeasy",             // a forge login from before
		"https://github.com/x", // a URL
		"mastodon:someone",     // the old platform:handle spelling
		"npub1q",               // too short to hold a checksum
		noteKey32,              // an event id
		nsecKey32,              // a private key
		npubBrokenChecksum,     // a key with one character changed
		"npub1sg6plzptd64u62a878hep2kev88swjh3tw00gjsfl8f237lmu63q0uf63b", // 'b' is not bech32
	})

	if len(got) != 0 {
		t.Fatalf("got %+v, want every entry dropped", got)
	}
}

// Entries written before Nostr was the only platform carry a "nostr:" prefix.
// Tolerated rather than dropped, so the change did not silently uncredit anyone.
func TestParseDevelopersToleratesTheNostrPrefix(t *testing.T) {
	got := parseDevelopers([]string{"nostr:" + fiatjafNpub})

	if len(got) != 1 {
		t.Fatalf("got %+v, want 1 developer", got)
	}
	if got[0].Key != fiatjafNpub {
		t.Errorf("key = %q, want the prefix stripped to %q", got[0].Key, fiatjafNpub)
	}
}

// Deduplicated on the decoded pubkey, so the same person written in two cases
// is one developer rather than two.
func TestParseDevelopersDeduplicatesOnTheDecodedKey(t *testing.T) {
	got := parseDevelopers([]string{jackNpub, strings.ToUpper(jackNpub)})

	if len(got) != 1 {
		t.Fatalf("got %d developers %+v, want 1 after dedupe", len(got), got)
	}
	// The first spelling survives: the field does not rewrite a key it did not
	// author.
	if got[0].Key != jackNpub {
		t.Errorf("key = %q, want the first spelling %q kept", got[0].Key, jackNpub)
	}
}

func TestParseDevelopersPreservesKeyCase(t *testing.T) {
	// bech32 is lowercase by spec, but a mixed-case key still verifies and the
	// field must not silently rewrite what it was given.
	got := parseDevelopers([]string{strings.ToUpper(jackNpub)})
	if len(got) != 1 || got[0].Key != strings.ToUpper(jackNpub) {
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

func TestDeveloperKeysListsThemInCatalogOrder(t *testing.T) {
	app := &App{Developers: parseDevelopers([]string{fiatjafNpub, jackNpub})}

	got := app.DeveloperKeys()
	if len(got) != 2 || got[0] != fiatjafNpub || got[1] != jackNpub {
		t.Errorf("DeveloperKeys() = %v, want [%s %s]", got, fiatjafNpub, jackNpub)
	}
	if got := (*App)(nil).DeveloperKeys(); got != nil {
		t.Errorf("DeveloperKeys() on a nil app = %v, want nil", got)
	}
}

// Every declared developer in the real catalog must be a decodable key with a
// profile URL. parseDevelopers already enforces the first, so this is a guard
// against that check being weakened — and the second, because a link to nowhere
// is worse than no link.
func TestEveryCatalogDeveloperIsAUsableKey(t *testing.T) {
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
			if _, err := nip19.PubkeyHex(d.Key); err != nil {
				t.Errorf("app %q credits %q, which is not a usable NIP-19 key: %v", app.ID, d.Key, err)
			}
			if d.URL == "" {
				t.Errorf("app %q developer %s has no profile URL", app.ID, d.Key)
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
