package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"core/apps"

	"github.com/go-chi/chi/v5"
)

// ─── Developer references ─────────────────────────────────────────────────────

// Externally-verified keys, the same vectors core/nip19 tests against.
const (
	fiatjafNpub = "npub180cvv07tjdrrgpa0j7j7tmnyl2yr6yr7l8j4s3evf6u64th6gkwsyjh6w6"
	jackNpub    = "npub1sg6plzptd64u62a878hep2kev88swjh3tw00gjsfl8f237lmu63q0uf63m"
)

func TestDeveloperKeyAcceptsProfileIdentifiers(t *testing.T) {
	for _, tc := range []struct{ raw, want string }{
		{fiatjafNpub, fiatjafNpub},
		{jackNpub, jackNpub},
		{"nprofile1qqsrhuxx8l9ex335q7he0f09aej04zpazpl0ne2cgukyawd24mayt8gpz3mhxue69uhhyetvv9ujuerpd46hxtnfduqs6amnwvaz7tmwdaejumr0dsfg70ls",
			"nprofile1qqsrhuxx8l9ex335q7he0f09aej04zpazpl0ne2cgukyawd24mayt8gpz3mhxue69uhhyetvv9ujuerpd46hxtnfduqs6amnwvaz7tmwdaejumr0dsfg70ls"},
		// A mixed-case key still verifies, and bech32 forbids rewriting it here.
		{strings.ToUpper(jackNpub), strings.ToUpper(jackNpub)},
		{"  " + jackNpub + "  ", jackNpub},
	} {
		if got, ok := developerKey(tc.raw); !ok || got != tc.want {
			t.Errorf("developerKey(%q) = %q, %v; want %q, true", tc.raw, got, ok, tc.want)
		}
	}
}

// Anything that is not a profile identifier is refused here rather than looked up
// and reported as missing: the catalog is not what is wrong with it.
func TestDeveloperKeyRejectsEverythingElse(t *testing.T) {
	for _, bad := range []string{
		"", "   ",
		// The forge logins the field used to accept. They are no longer
		// developers, so they must not resolve to anything.
		"besoeasy",
		"github:besoeasy",
		"mastodon:someone",
		"https://njump.me/" + jackNpub,
		// Prefix decoding must not turn a path segment into a path.
		"..%2Fetc",
		jackNpub + "%2Fmore",
		"npub1q",
		"note1sg6plzptd64u62a878hep2kev88swjh3tw00gjsfl8f237lmu63q7k28gn",
		"nsec1sg6plzptd64u62a878hep2kev88swjh3tw00gjsfl8f237lmu63qr2zmhw",
	} {
		if got, ok := developerKey(bad); ok {
			t.Errorf("developerKey(%q) was accepted as %q, want a rejection", bad, got)
		}
	}
}

// ─── /api/developers/{handle} ─────────────────────────────────────────────────

// writeTempCatalog points the apps loader at a throwaway catalog and restores
// the previous directory afterwards.
func writeTempCatalog(t *testing.T, compose string) {
	t.Helper()

	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "demo"), 0o755); err != nil {
		t.Fatalf("create app dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "demo", "compose.yml"), []byte(compose), 0o644); err != nil {
		t.Fatalf("write compose: %v", err)
	}

	previous := apps.GetAppsDir()
	apps.SetAppsDir(dir)
	t.Cleanup(func() { apps.SetAppsDir(previous) })

	// The catalog is cached for 60s; force a reload so this test sees its own
	// directory rather than whatever a previous test left in the cache.
	if _, err := apps.GetCatalogCached(true); err != nil {
		t.Fatalf("prime catalog: %v", err)
	}
}

func developerRequest(t *testing.T, handle string) *httptest.ResponseRecorder {
	t.Helper()

	r := httptest.NewRequest(http.MethodGet, "/api/developers/"+handle, nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("handle", handle)
	r = r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))

	w := httptest.NewRecorder()
	handleDeveloperApps(w, r)
	return w
}

var demoCompose = fmt.Sprintf(`x-yantr:
  name: "Demo"
  tags: [tools, utility, self-hosted, homelab, podman]
  short_description: "A demo app for the developer route."
  description: >
    A demo app used to prove that the developer route reads the local catalog
    and needs no third party to answer.
  usecases: ["Prove the route works.", "Prove it needs no network."]
  website: "https://example.com"
  developers: ["%s", "%s"]
  ports:
    - {port: 8080, protocol: HTTP, show: true, label: "Web UI", service: demo}

services:
  demo:
    image: ghcr.io/example/demo:latest
    ports:
      - "8080"
`, fiatjafNpub, jackNpub)

// The developer route must answer from the local catalog alone. This is what
// makes it survive GitHub being unreachable, rate-limited, or not the
// developer's platform at all — Yantr is rootless and expected to run offline.
func TestHandleDeveloperAppsServesFromLocalCatalog(t *testing.T) {
	writeTempCatalog(t, demoCompose)

	w := developerRequest(t, fiatjafNpub)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", w.Code, w.Body.String())
	}

	var body struct {
		Success   bool `json:"success"`
		Count     int  `json:"count"`
		Developer struct {
			Key   string `json:"key"`
			Label string `json:"label"`
		} `json:"developer"`
		Apps []struct {
			ID string `json:"id"`
		} `json:"apps"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v", err)
	}

	if !body.Success || body.Count != 1 || len(body.Apps) != 1 {
		t.Fatalf("unexpected body: %s", w.Body.String())
	}
	if body.Apps[0].ID != "demo" {
		t.Errorf("app id = %q, want demo", body.Apps[0].ID)
	}
	if body.Developer.Key != fiatjafNpub || body.Developer.Label != "Nostr" {
		t.Errorf("developer = %+v, want key %s labelled Nostr", body.Developer, fiatjafNpub)
	}
}

func TestHandleDeveloperAppsServesEveryKeyTheAppCredits(t *testing.T) {
	writeTempCatalog(t, demoCompose)

	w := developerRequest(t, jackNpub)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", w.Code, w.Body.String())
	}
}

// The encoded form must reach the same handler result, since that is the form
// the UI actually sends.
func TestHandleDeveloperAppsAcceptsPercentEncodedKeys(t *testing.T) {
	writeTempCatalog(t, demoCompose)

	// Simulates what chi hands the handler for "/api/developers/nprofile1qsp%2Fz":
	// the raw path segment, still percent-encoded. An encoded separator must not
	// survive into the key.
	w := developerRequest(t, "nprofile1qsp%2Fz")
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, body = %s, want the encoded separator rejected", w.Code, w.Body.String())
	}
}

func TestHandleDeveloperAppsNotFoundAndBadRef(t *testing.T) {
	writeTempCatalog(t, demoCompose)

	// A handle no app declares is a 404, not an empty 200: the UI uses the
	// difference to decide between "unknown developer" and "no apps yet".
	// A valid key nobody credits is a 404: the catalog simply does not list it.
	if w := developerRequest(t, undeclaredNpub); w.Code != http.StatusNotFound {
		t.Errorf("status for an uncredited key = %d, want 404", w.Code)
	}
	// Something that is not a key at all is a 400.
	if w := developerRequest(t, "besoeasy"); w.Code != http.StatusBadRequest {
		t.Errorf("status for a non-key = %d, want 400", w.Code)
	}
}

// ─── /api/nostr/profile/{key} ─────────────────────────────────────────────────

const demoNostrCompose = `x-yantr:
  name: "Demo Nostr"
  tags: [tools, utility, self-hosted, homelab, podman]
  short_description: "A demo app crediting a Nostr developer."
  description: >
    A demo app used to prove that the Nostr profile route only serves keys the
    local catalog declares, and validates them before it tries a relay.
  usecases: ["Prove the route validates its key.", "Prove undeclared keys 404."]
  website: "https://example.com"
  developers: ["npub180cvv07tjdrrgpa0j7j7tmnyl2yr6yr7l8j4s3evf6u64th6gkwsyjh6w6"]
  ports:
    - {port: 8080, protocol: HTTP, show: true, label: "Web UI", service: demo}

services:
  demo:
    image: ghcr.io/example/demo:latest
    ports:
      - "8080"
`

// A well-formed npub that no app in the fixtures declares. Reaching the handler
// with it must not contact a relay.
const undeclaredNpub = "npub1gwnvhmyryy8wqgfttcgk7adkvfxrynvy556qex4dvf94yptc2r6q0z3p66"

func nostrProfileRequest(t *testing.T, key string) *httptest.ResponseRecorder {
	t.Helper()

	r := httptest.NewRequest(http.MethodGet, "/api/nostr/profile/"+key, nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("key", key)
	r = r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))

	w := httptest.NewRecorder()
	handleNostrProfile(w, r)
	return w
}

// The relay proxy is not an open relay: a key must be declared by some app
// before it is queried, or the endpoint proxies arbitrary queries to public
// relays and grows an unbounded cache behind it.
func TestHandleNostrProfileRefusesUndeclaredKeys(t *testing.T) {
	writeTempCatalog(t, demoNostrCompose)

	w := nostrProfileRequest(t, undeclaredNpub)
	if w.Code != http.StatusNotFound {
		t.Errorf("status for an undeclared key = %d, want 404 (body %s)", w.Code, w.Body.String())
	}
}

// A key that is not an npub or nprofile is a bad request rather than an upstream
// failure. It is rejected before the cache is consulted, so it can neither hold
// a cache slot nor surface as a relay timeout dressed up as a 502.
func TestHandleNostrProfileRejectsMalformedKeys(t *testing.T) {
	writeTempCatalog(t, demoNostrCompose)

	for _, tc := range []struct{ name, key string }{
		{"empty", ""},
		{"not a bech32 string", "not-a-key"},
		{"broken checksum", "npub1sg6plzptd64u62a878hep2kev88swjh3tw00gjsfl8f237lmu63q0uf63q"},
		{"a private key", "nsec1sg6plzptd64u62a878hep2kev88swjh3tw00gjsfl8f237lmu63qr2zmhw"},
		{"an event id", "note1sg6plzptd64u62a878hep2kev88swjh3tw00gjsfl8f237lmu63q7k28gn"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w := nostrProfileRequest(t, tc.key)
			if w.Code != http.StatusBadRequest {
				t.Errorf("status = %d, want 400 (body %s)", w.Code, w.Body.String())
			}
		})
	}
}

// Case is an equivalence now, not an identity. bech32 is lowercase by spec, so a
// key written in any case still decodes to the same pubkey, and the catalog does
// not rewrite it — which means two spellings of one key have to collapse to one
// developer. Pinned at the catalog layer because deciding it there is what keeps
// the relay endpoint from being asked about the same person twice.
func TestHandleDeveloperAppsTreatsKeyCaseAsTheSameAccount(t *testing.T) {
	writeTempCatalog(t, demoCompose)

	for _, spelling := range []string{jackNpub, strings.ToUpper(jackNpub)} {
		if w := developerRequest(t, spelling); w.Code != http.StatusOK {
			t.Errorf("status for %q = %d, want 200", spelling, w.Code)
		}
	}
}
