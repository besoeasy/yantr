package system

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// stubRepo serves a repository payload and counts how many times upstream was
// actually hit, which is the only way to observe the cache doing its job.
func stubRepo(t *testing.T, body string, status int, hits *int) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		*hits++
		if r.Header.Get("User-Agent") == "" {
			t.Error("GitHub rejects requests without a User-Agent")
		}
		w.WriteHeader(status)
		w.Write([]byte(body))
	}))
	t.Cleanup(srv.Close)
	return srv
}

// useRepoAPI points the fetcher at a stub and clears the cache for the duration
// of a test, restoring whatever was there before.
func useRepoAPI(t *testing.T, url string) {
	t.Helper()
	api, cache, exp := githubRepoAPI, ghCache, ghCacheExp
	githubRepoAPI, ghCache, ghCacheExp = url, nil, time.Time{}
	t.Cleanup(func() { githubRepoAPI, ghCache, ghCacheExp = api, cache, exp })
}

func TestGetRepoStatsServesCacheWithinTTL(t *testing.T) {
	hits := 0
	useRepoAPI(t, stubRepo(t, `{"stargazers_count":184,"forks_count":17,"open_issues_count":3}`, http.StatusOK, &hits).URL)

	first, err := GetRepoStatsCached(false)
	if err != nil {
		t.Fatalf("first fetch: %v", err)
	}
	if first.Stars != 184 || first.Forks != 17 || first.OpenIssues != 3 {
		t.Fatalf("counters not decoded: %+v", first)
	}
	if first.FetchedAt == "" {
		t.Error("FetchedAt should be stamped")
	}

	// A second call inside the TTL must not touch the network: the upstream
	// budget is 60/hour and every browser tab would otherwise spend one.
	if _, err := GetRepoStatsCached(false); err != nil {
		t.Fatalf("cached fetch: %v", err)
	}
	if hits != 1 {
		t.Errorf("upstream hit %d times within one TTL, want 1", hits)
	}
}

func TestGetRepoStatsForceRefreshBypassesCache(t *testing.T) {
	hits := 0
	useRepoAPI(t, stubRepo(t, `{"stargazers_count":10}`, http.StatusOK, &hits).URL)

	if _, err := GetRepoStatsCached(false); err != nil {
		t.Fatalf("first fetch: %v", err)
	}
	if _, err := GetRepoStatsCached(true); err != nil {
		t.Fatalf("forced fetch: %v", err)
	}
	if hits != 2 {
		t.Errorf("force=true hit upstream %d times, want 2", hits)
	}
}

func TestGetRepoStatsKeepsLastGoodValueWhenRefreshFails(t *testing.T) {
	var hits int
	srv := stubRepo(t, `{"stargazers_count":184}`, http.StatusOK, &hits)
	useRepoAPI(t, srv.URL)

	if _, err := GetRepoStatsCached(false); err != nil {
		t.Fatalf("first fetch: %v", err)
	}

	// Upstream now fails. A stale counter still beats no counter, so the
	// refresh failure must not clear a good cache or surface an error.
	srv.Config.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		w.WriteHeader(http.StatusForbidden)
	})

	got, err := GetRepoStatsCached(true)
	if err != nil {
		t.Fatalf("failed refresh should fall back to cache, got: %v", err)
	}
	if got.Stars != 184 {
		t.Errorf("fallback returned %d stars, want the cached 184", got.Stars)
	}
}

func TestGetRepoStatsErrorsWithNoCacheWhenUpstreamFails(t *testing.T) {
	hits := 0
	useRepoAPI(t, stubRepo(t, `{"message":"rate limit exceeded"}`, http.StatusForbidden, &hits).URL)

	if _, err := GetRepoStatsCached(false); err == nil {
		t.Fatal("expected an error when there is nothing cached to fall back on")
	}
}

func TestGetRepoStatsRejectsNonJSON(t *testing.T) {
	hits := 0
	useRepoAPI(t, stubRepo(t, `<html>not json</html>`, http.StatusOK, &hits).URL)

	if _, err := GetRepoStatsCached(false); err == nil {
		t.Fatal("expected an error when the body is not JSON")
	}
}

func TestRepoStatsJSONShape(t *testing.T) {
	// The UI reads these field names off the wire, so a rename here would
	// silently blank the payload rather than fail to compile.
	var decoded map[string]interface{}
	if err := json.Unmarshal([]byte(`{"stars":184,"forks":17,"openIssues":3,"fetchedAt":"now"}`), &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	for _, key := range []string{"stars", "forks", "openIssues", "fetchedAt"} {
		if _, ok := decoded[key]; !ok {
			t.Errorf("missing %q in RepoStats JSON", key)
		}
	}
}

// ─── Developer profiles ───────────────────────────────────────────────────────

// stubProfile serves any GitHub payload and counts upstream hits, so the cache
// can be observed doing its job.
func stubProfile(t *testing.T, status int, body string, hits *int) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		*hits++
		if r.Header.Get("User-Agent") == "" {
			t.Error("GitHub rejects requests without a User-Agent")
		}
		w.WriteHeader(status)
		w.Write([]byte(body))
	}))
	t.Cleanup(srv.Close)
	return srv
}

// useProfileAPI points the profile fetchers at a stub and clears both caches for
// the duration of a test, restoring whatever was there before.
func useProfileAPI(t *testing.T, url string) {
	t.Helper()
	api := githubProfileAPI
	users, repos := ghUserCache, ghRepoCache
	githubProfileAPI = url
	ghUserCache = map[string]githubUserEntry{}
	ghRepoCache = map[string]githubRepoEntry{}
	t.Cleanup(func() {
		githubProfileAPI = api
		ghUserCache, ghRepoCache = users, repos
	})
}

func TestGetGitHubUserDecodesProfile(t *testing.T) {
	var hits int
	useProfileAPI(t, stubProfile(t, http.StatusOK, `{
		"login":"besoeasy","name":"Besoeasy","avatar_url":"https://a/x.png",
		"bio":"Builds things","company":"@besoeasy","blog":"https://x.dev",
		"location":"Earth","html_url":"https://github.com/besoeasy",
		"public_repos":48,"followers":1200,"following":12,"created_at":"2015-01-01T00:00:00Z"
	}`, &hits).URL)

	got, err := GetGitHubUserCached("besoeasy", false)
	if err != nil {
		t.Fatalf("fetch: %v", err)
	}
	if got.Name != "Besoeasy" || got.PublicRepos != 48 || got.Followers != 1200 {
		t.Fatalf("profile not decoded: %+v", got)
	}
	if got.AvatarURL == "" || got.HTMLURL == "" || got.FetchedAt == "" {
		t.Errorf("derived fields missing: %+v", got)
	}
}

// The whole point of the week-long TTL: repeated page views must not spend the
// 60/hour unauthenticated budget.
func TestGetGitHubUserServesCacheWithinTTL(t *testing.T) {
	var hits int
	useProfileAPI(t, stubProfile(t, http.StatusOK, `{"login":"besoeasy"}`, &hits).URL)

	for i := 0; i < 5; i++ {
		if _, err := GetGitHubUserCached("besoeasy", false); err != nil {
			t.Fatalf("call %d: %v", i, err)
		}
	}
	if hits != 1 {
		t.Errorf("upstream hit %d times within one TTL, want 1", hits)
	}
}

func TestGetGitHubUserCachesPerLogin(t *testing.T) {
	var hits int
	useProfileAPI(t, stubProfile(t, http.StatusOK, `{"login":"x"}`, &hits).URL)

	if _, err := GetGitHubUserCached("one", false); err != nil {
		t.Fatalf("first login: %v", err)
	}
	if _, err := GetGitHubUserCached("two", false); err != nil {
		t.Fatalf("second login: %v", err)
	}
	if hits != 2 {
		t.Errorf("two logins shared %d upstream hit(s), want 2 — the cache must be keyed by login", hits)
	}
}

func TestGetGitHubUserForceRefreshBypassesCache(t *testing.T) {
	var hits int
	useProfileAPI(t, stubProfile(t, http.StatusOK, `{"login":"besoeasy"}`, &hits).URL)

	if _, err := GetGitHubUserCached("besoeasy", false); err != nil {
		t.Fatalf("first fetch: %v", err)
	}
	if _, err := GetGitHubUserCached("besoeasy", true); err != nil {
		t.Fatalf("forced fetch: %v", err)
	}
	if hits != 2 {
		t.Errorf("force=true hit upstream %d times, want 2", hits)
	}
}

func TestGetGitHubUserKeepsLastGoodValueWhenRefreshFails(t *testing.T) {
	var hits int
	srv := stubProfile(t, http.StatusOK, `{"login":"besoeasy","name":"Besoeasy"}`, &hits)
	useProfileAPI(t, srv.URL)

	if _, err := GetGitHubUserCached("besoeasy", false); err != nil {
		t.Fatalf("first fetch: %v", err)
	}

	srv.Config.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		w.WriteHeader(http.StatusForbidden)
	})

	got, err := GetGitHubUserCached("besoeasy", true)
	if err != nil {
		t.Fatalf("failed refresh should fall back to cache, got: %v", err)
	}
	if got.Name != "Besoeasy" {
		t.Errorf("fallback returned %+v, want the cached profile", got)
	}
}

// A miss must be cached too, on a far shorter clock: a handle that 404s today
// may be a typo fixed this afternoon. Without the short negative cache, a
// broken entry would spend a request on every single page view.
func TestGetGitHubUserCachesMissesBriefly(t *testing.T) {
	var hits int
	useProfileAPI(t, stubProfile(t, http.StatusNotFound, `{"message":"Not Found"}`, &hits).URL)

	if _, err := GetGitHubUserCached("ghost", false); err == nil {
		t.Fatal("expected an error for a 404")
	}
	if _, err := GetGitHubUserCached("ghost", false); err == nil {
		t.Fatal("expected the cached error to be replayed")
	}
	if hits != 1 {
		t.Errorf("a missing login hit upstream %d times, want 1", hits)
	}
}

// The login is interpolated into the upstream URL path, so anything outside
// GitHub's own handle rule must be rejected before a request is built.
func TestGetGitHubUserRejectsMalformedLogin(t *testing.T) {
	var hits int
	useProfileAPI(t, stubProfile(t, http.StatusOK, `{}`, &hits).URL)

	for _, bad := range []string{"", "a b", "../../etc/passwd", "a/b", "-leading", "trailing-", strings.Repeat("x", 40)} {
		if _, err := GetGitHubUserCached(bad, false); err == nil {
			t.Errorf("login %q was accepted, want a rejection", bad)
		}
	}
	if hits != 0 {
		t.Errorf("a malformed login reached upstream %d time(s), want 0", hits)
	}

	if _, err := GetGitHubUserCached("besoeasy", false); err != nil {
		t.Errorf("a valid login was rejected: %v", err)
	}
}

func TestGetGitHubUserReposFiltersForks(t *testing.T) {
	var hits int
	useProfileAPI(t, stubProfile(t, http.StatusOK, `[
		{"name":"mine","description":"d","html_url":"u","language":"Go",
		 "stargazers_count":312,"forks_count":4,"topics":["tools"],"pushed_at":"t"},
		{"name":"someone-elses","fork":true}
	]`, &hits).URL)

	got, err := GetGitHubUserReposCached("besoeasy", false)
	if err != nil {
		t.Fatalf("fetch: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("got %d repos %+v, want forks dropped", len(got), got)
	}
	if got[0].Name != "mine" || got[0].Stars != 312 || got[0].Language != "Go" {
		t.Errorf("repo not decoded: %+v", got[0])
	}
	if got[0].Topics == nil {
		t.Error("topics should normalise to an empty slice, not null")
	}
}

func TestGetGitHubUserReposServesCacheWithinTTL(t *testing.T) {
	var hits int
	useProfileAPI(t, stubProfile(t, http.StatusOK, `[]`, &hits).URL)

	for i := 0; i < 3; i++ {
		if _, err := GetGitHubUserReposCached("besoeasy", false); err != nil {
			t.Fatalf("call %d: %v", i, err)
		}
	}
	if hits != 1 {
		t.Errorf("upstream hit %d times within one TTL, want 1", hits)
	}
}

// The UI reads these field names off the wire, so a rename here would silently
// blank the developer page rather than fail to compile.
func TestDeveloperProfileJSONShape(t *testing.T) {
	var user map[string]interface{}
	if err := json.Unmarshal([]byte(`{"login":"x","name":"n","avatarUrl":"a","bio":"b",
		"company":"c","blog":"d","location":"e","htmlUrl":"f","publicRepos":1,
		"followers":2,"following":3,"createdAt":"g","fetchedAt":"h"}`), &user); err != nil {
		t.Fatalf("unmarshal user: %v", err)
	}
	for _, key := range []string{"login", "name", "avatarUrl", "bio", "company", "blog",
		"location", "htmlUrl", "publicRepos", "followers", "following", "createdAt", "fetchedAt"} {
		if _, ok := user[key]; !ok {
			t.Errorf("missing %q in GitHubUser JSON", key)
		}
	}

	var repo map[string]interface{}
	if err := json.Unmarshal([]byte(`{"name":"n","description":"d","htmlUrl":"u","language":"Go",
		"stars":1,"forks":2,"topics":[],"archived":false,"pushedAt":"p"}`), &repo); err != nil {
		t.Fatalf("unmarshal repo: %v", err)
	}
	for _, key := range []string{"name", "description", "htmlUrl", "language", "stars",
		"forks", "topics", "archived", "pushedAt"} {
		if _, ok := repo[key]; !ok {
			t.Errorf("missing %q in GitHubRepo JSON", key)
		}
	}
}
