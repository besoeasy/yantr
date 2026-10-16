package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
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
