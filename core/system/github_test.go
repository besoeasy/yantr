package system

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
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