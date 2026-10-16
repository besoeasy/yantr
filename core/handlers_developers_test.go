package main

import (
	"context"
	"encoding/json"
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

func TestDeveloperRefDefaultsToGitHub(t *testing.T) {
	// The short form is the whole point of /#/developer/besoeasy.
	platform, handle, ok := developerRef("besoeasy")
	if !ok || platform != "github" || handle != "besoeasy" {
		t.Fatalf("developerRef(besoeasy) = %q, %q, %v; want github, besoeasy, true", platform, handle, ok)
	}
}

func TestDeveloperRefParsesExplicitPlatform(t *testing.T) {
	for _, tc := range []struct {
		raw      string
		platform string
		handle   string
	}{
		{"gitlab:username2", "gitlab", "username2"},
		{"codeberg:someone", "codeberg", "someone"},
		{"nostr:npub1qqq", "nostr", "npub1qqq"},
		{"GitLab:Upper", "gitlab", "Upper"},
		{"github:@besoeasy", "github", "besoeasy"},
		{"  github:besoeasy  ", "github", "besoeasy"},
	} {
		platform, handle, ok := developerRef(tc.raw)
		if !ok || platform != tc.platform || handle != tc.handle {
			t.Errorf("developerRef(%q) = %q, %q, %v; want %q, %q, true",
				tc.raw, platform, handle, ok, tc.platform, tc.handle)
		}
	}
}

// The UI percent-encodes the colon when it builds the request path, and chi
// routes on the raw path, so the encoded form has to decode to the same
// reference — otherwise every link the UI generates would 404.
func TestDeveloperRefDecodesPercentEncoding(t *testing.T) {
	for _, tc := range []struct {
		raw      string
		platform string
		handle   string
	}{
		{"github%3Abesoeasy", "github", "besoeasy"},
		{"gitlab%3Ausername2", "gitlab", "username2"},
		{"GitLab%3AUpper", "gitlab", "Upper"},
		{"github%3A%40besoeasy", "github", "besoeasy"},
		// A bare handle has nothing to decode and must be unaffected.
		{"besoeasy", "github", "besoeasy"},
	} {
		platform, handle, ok := developerRef(tc.raw)
		if !ok || platform != tc.platform || handle != tc.handle {
			t.Errorf("developerRef(%q) = %q, %q, %v; want %q, %q, true",
				tc.raw, platform, handle, ok, tc.platform, tc.handle)
		}
	}
}

// Decoding a reference must not turn a path segment into a path.
func TestDeveloperRefRejectsEncodedSeparators(t *testing.T) {
	for _, bad := range []string{"..%2Fetc", "a%2Fb", "%2F", "github%3Aa%2Fb", "a%5Cb"} {
		if _, _, ok := developerRef(bad); ok {
			t.Errorf("developerRef(%q) was accepted, want a rejection", bad)
		}
	}
}

func TestDeveloperRefRejectsUnknownPlatformAndEmptyHandle(t *testing.T) {
	// An unknown platform has no profile URL to derive, so the reference cannot
	// be served rather than being silently treated as a GitHub login.
	for _, bad := range []string{"", "  ", "mastodon:someone", "github:", "github:@", ":"} {
		if _, _, ok := developerRef(bad); ok {
			t.Errorf("developerRef(%q) was accepted, want a rejection", bad)
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

const demoCompose = `x-yantr:
  name: "Demo"
  tags: [tools, utility, self-hosted, homelab, podman]
  short_description: "A demo app for the developer route."
  description: >
    A demo app used to prove that the developer route reads the local catalog
    and needs no third party to answer.
  usecases: ["Prove the route works.", "Prove it needs no network."]
  website: "https://example.com"
  developers: ["github:besoeasy", "gitlab:someone"]
  ports:
    - {port: 8080, protocol: HTTP, show: true, label: "Web UI", service: demo}

services:
  demo:
    image: ghcr.io/example/demo:latest
    ports:
      - "8080"
`

// The developer route must answer from the local catalog alone. This is what
// makes it survive GitHub being unreachable, rate-limited, or not the
// developer's platform at all — Yantr is rootless and expected to run offline.
func TestHandleDeveloperAppsServesFromLocalCatalog(t *testing.T) {
	writeTempCatalog(t, demoCompose)

	w := developerRequest(t, "besoeasy")
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", w.Code, w.Body.String())
	}

	var body struct {
		Success   bool `json:"success"`
		Count     int  `json:"count"`
		Developer struct {
			Platform string `json:"platform"`
			Handle   string `json:"handle"`
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
	if body.Developer.Platform != "github" || body.Developer.Handle != "besoeasy" {
		t.Errorf("developer = %+v, want github:besoeasy", body.Developer)
	}
}

func TestHandleDeveloperAppsMatchesOtherPlatforms(t *testing.T) {
	writeTempCatalog(t, demoCompose)

	w := developerRequest(t, "gitlab:someone")
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", w.Code, w.Body.String())
	}
}

// The encoded form must reach the same handler result, since that is the form
// the UI actually sends.
func TestHandleDeveloperAppsAcceptsEncodedColon(t *testing.T) {
	writeTempCatalog(t, demoCompose)

	// Simulates what chi hands the handler for "?/api/developers/gitlab%3Asomeone":
	// the raw path segment, still percent-encoded.
	w := developerRequest(t, "gitlab%3Asomeone")
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", w.Code, w.Body.String())
	}
}

func TestHandleDeveloperAppsNotFoundAndBadRef(t *testing.T) {
	writeTempCatalog(t, demoCompose)

	// A handle no app declares is a 404, not an empty 200: the UI uses the
	// difference to decide between "unknown developer" and "no apps yet".
	if w := developerRequest(t, "nosuchdev"); w.Code != http.StatusNotFound {
		t.Errorf("status for unknown developer = %d, want 404", w.Code)
	}
	if w := developerRequest(t, "mastodon:someone"); w.Code != http.StatusBadRequest {
		t.Errorf("status for unknown platform = %d, want 400", w.Code)
	}
}

// The GitHub proxies are not an open relay: a login must be declared by some
// app in the catalog before it is served. Otherwise the endpoint would proxy
// arbitrary api.github.com traffic and grow an unbounded cache.
func TestCatalogGitHubDeveloperOnlyServesDeclaredHandles(t *testing.T) {
	writeTempCatalog(t, demoCompose)

	r := httptest.NewRequest(http.MethodGet, "/api/github/user/besoeasy", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("login", "besoeasy")
	r = r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))

	if login, ok := catalogGitHubDeveloper(httptest.NewRecorder(), r); !ok || login != "besoeasy" {
		t.Fatalf("declared login not accepted: %q %v", login, ok)
	}

	for _, undeclared := range []string{"torvalds", "randomauthor"} {
		r := httptest.NewRequest(http.MethodGet, "/api/github/user/"+undeclared, nil)
		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("login", undeclared)
		r = r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))

		w := httptest.NewRecorder()
		if _, ok := catalogGitHubDeveloper(w, r); ok {
			t.Errorf("undeclared login %q was served, want a 404", undeclared)
		}
		if w.Code != http.StatusNotFound {
			t.Errorf("status for undeclared login %q = %d, want 404", undeclared, w.Code)
		}
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
  developers: ["nostr:npub180cvv07tjdrrgpa0j7j7tmnyl2yr6yr7l8j4s3evf6u64th6gkwsyjh6w6"]
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
const undeclaredNpub = "npub1sg6plzptd64u62a878hep2kev88swjh3tw00gjsfl8f237lmu63q0uf63m"

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

// The relay proxy is not an open relay, for the same reason the GitHub one is
// not: a key must be declared by some app before it is queried, or the endpoint
// proxies arbitrary traffic to a public relay and grows an unbounded cache.
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

// Nostr keys are matched case-sensitively, unlike GitHub logins. bech32 is
// lowercase by spec and the catalog does not rewrite a key's case, so an
// uppercased request names a key no app declares — and a 404 is the honest
// answer rather than silently serving a different string's cache entry.
func TestHandleNostrProfileMatchesKeysCaseSensitively(t *testing.T) {
	writeTempCatalog(t, demoNostrCompose)

	w := nostrProfileRequest(t, strings.ToUpper("npub180cvv07tjdrrgpa0j7j7tmnyl2yr6yr7l8j4s3evf6u64th6gkwsyjh6w6"))
	if w.Code != http.StatusNotFound {
		t.Errorf("status for an uppercased key = %d, want 404 (body %s)", w.Code, w.Body.String())
	}
}
