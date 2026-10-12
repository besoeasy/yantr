package system

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"
)

// RepoStats holds the public GitHub counters for the Yantr repository.
type RepoStats struct {
	Stars      int    `json:"stars"`
	Forks      int    `json:"forks"`
	OpenIssues int    `json:"openIssues"`
	FetchedAt  string `json:"fetchedAt"`
}

// githubRepoAPI is the unauthenticated repository endpoint. It answers 60
// requests per hour per source IP, so the result is cached generously and the
// upstream call is made at most once per TTL no matter how many browser tabs
// or cards ask for it.
//
// A var rather than a const so tests can point it at a stub server.
var githubRepoAPI = "https://api.github.com/repos/besoeasy/yantr"

// 30 minutes. A star count is a trend, not a live reading — refreshing it more
// often would spend the hourly rate limit to show a number that will not have
// moved, and would make the endpoint a liability the moment it runs out.
const githubCacheTTL = 30 * time.Minute

var (
	ghMu       sync.Mutex
	ghCache    *RepoStats
	ghCacheExp time.Time
)

// GetRepoStatsCached returns the GitHub repository counters, cached for 30 minutes.
func GetRepoStatsCached(forceRefresh bool) (*RepoStats, error) {
	ghMu.Lock()
	defer ghMu.Unlock()

	if !forceRefresh && ghCache != nil && time.Now().Before(ghCacheExp) {
		return ghCache, nil
	}

	stats, err := fetchRepoStats()
	if err != nil {
		// A refresh failure must not wipe a good cached value: the host may be
		// offline or GitHub may be rate-limiting, and a stale count is far more
		// useful than none.
		if ghCache != nil {
			return ghCache, nil
		}
		return nil, err
	}

	ghCache = stats
	ghCacheExp = time.Now().Add(githubCacheTTL)
	return stats, nil
}

func fetchRepoStats() (*RepoStats, error) {
	client := &http.Client{Timeout: 6 * time.Second}

	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", githubRepoAPI, nil)
	if err != nil {
		return nil, err
	}
	// GitHub rejects requests without a User-Agent.
	req.Header.Set("User-Agent", "yantr")
	req.Header.Set("Accept", "application/vnd.github+json")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("github responded %d", resp.StatusCode)
	}

	var raw struct {
		StargazersCount int `json:"stargazers_count"`
		ForksCount      int `json:"forks_count"`
		OpenIssuesCount int `json:"open_issues_count"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, err
	}

	return &RepoStats{
		Stars:      raw.StargazersCount,
		Forks:      raw.ForksCount,
		OpenIssues: raw.OpenIssuesCount,
		FetchedAt:  time.Now().UTC().Format(time.RFC3339),
	}, nil
}