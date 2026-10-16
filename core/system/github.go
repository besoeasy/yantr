package system

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"regexp"
	"strings"
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

// ─── Developer profiles ───────────────────────────────────────────────────────
//
// An app can declare its authors in x-yantr.developers (core/apps/catalog.go).
// These two fetchers turn a GitHub handle into something worth looking at: the
// account itself, and their other repositories. The app list on a developer
// page comes from the local catalog and never from here.

// GitHubUser is the public profile of one GitHub account.
type GitHubUser struct {
	Login       string `json:"login"`
	Name        string `json:"name"`
	AvatarURL   string `json:"avatarUrl"`
	Bio         string `json:"bio"`
	Company     string `json:"company"`
	Blog        string `json:"blog"`
	Location    string `json:"location"`
	HTMLURL     string `json:"htmlUrl"`
	PublicRepos int    `json:"publicRepos"`
	Followers   int    `json:"followers"`
	Following   int    `json:"following"`
	CreatedAt   string `json:"createdAt"`
	FetchedAt   string `json:"fetchedAt"`
}

// GitHubRepo is one public repository, for the "their other projects" half of a
// developer page. Deliberately a fraction of the API's payload: the UI shows a
// list, not a repo manager.
type GitHubRepo struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	HTMLURL     string   `json:"htmlUrl"`
	Language    string   `json:"language"`
	Stars       int      `json:"stars"`
	Forks       int      `json:"forks"`
	Topics      []string `json:"topics"`
	Archived    bool     `json:"archived"`
	PushedAt    string   `json:"pushedAt"`
}

// githubProfileAPI is a var rather than a const so tests can point it at a stub.
var githubProfileAPI = "https://api.github.com"

const (
	// Seven days. A profile is reference data, not a reading — a bio, a
	// follower count and a repo list barely move in a week — while the
	// unauthenticated API allows 60 requests an hour per source IP, shared by
	// everyone behind the same NAT. A week of cache means viewing a developer
	// page essentially never spends a request.
	githubProfileTTL = 7 * 24 * time.Hour

	// A miss is cached far more briefly than a hit. A handle that 404s today
	// may be a typo we fix this afternoon, or an account created tomorrow;
	// neither should stay buried for a week.
	githubProfileMissTTL = time.Hour

	// Cap on the repo list. GitHub's own maximum page size, and more rows than
	// the page shows anyway.
	githubReposPerPage = 100
)

// githubLoginRe approximates GitHub's own handle rule (1–39 characters,
// alphanumeric with hyphens, no leading or trailing hyphen). It matters beyond
// tidiness: the login is interpolated into the upstream URL path, so anything
// outside this set (a slash, a dot-dot) must never reach the request builder.
//
// Written without a lookahead for consecutive hyphens because Go's regexp is
// RE2 and has none; permitting them is harmless, permitting a dot or slash is
// not, and that is what this actually guards.
var githubLoginRe = regexp.MustCompile(`^[A-Za-z0-9](?:[A-Za-z0-9-]{0,37}[A-Za-z0-9])?$`)

type githubUserEntry struct {
	user *GitHubUser
	err  error
	exp  time.Time
}

type githubRepoEntry struct {
	repos []GitHubRepo
	err   error
	exp   time.Time
}

var (
	ghProfileMu sync.Mutex
	ghUserCache = map[string]githubUserEntry{}
	ghRepoCache = map[string]githubRepoEntry{}
)

// githubGet performs one authenticated-if-possible GET and decodes the JSON body
// into out.
func githubGet(path string, out interface{}) error {
	client := &http.Client{Timeout: 6 * time.Second}

	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", githubProfileAPI+path, nil)
	if err != nil {
		return err
	}
	// GitHub rejects requests without a User-Agent.
	req.Header.Set("User-Agent", "yantr")
	req.Header.Set("Accept", "application/vnd.github+json")

	// Optional token. Without one the public API allows 60 requests an hour per
	// source IP, which a shared homelab NAT exhausts quickly; with one, 5000.
	// Unset is a supported configuration, not a degraded one — the cache is
	// sized so that a cold install rarely needs a request at all.
	if token := strings.TrimSpace(os.Getenv("GITHUB_TOKEN")); token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("github responded %d", resp.StatusCode)
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

// GetGitHubUserCached returns one GitHub profile, cached for a week.
func GetGitHubUserCached(login string, forceRefresh bool) (*GitHubUser, error) {
	if !githubLoginRe.MatchString(login) {
		return nil, fmt.Errorf("invalid github login %q", login)
	}

	ghProfileMu.Lock()
	defer ghProfileMu.Unlock()

	if e, ok := ghUserCache[login]; ok && !forceRefresh && time.Now().Before(e.exp) {
		return e.user, e.err
	}

	user, err := fetchGitHubUser(login)

	// A refresh failure must not wipe a good cached value, exactly as in
	// GetRepoStatsCached: an offline host and a rate-limited one look the same
	// from here, and a week-old bio beats a blank page.
	if err != nil {
		if e, ok := ghUserCache[login]; ok && e.user != nil {
			return e.user, nil
		}
		ghUserCache[login] = githubUserEntry{err: err, exp: time.Now().Add(githubProfileMissTTL)}
		return nil, err
	}

	ghUserCache[login] = githubUserEntry{user: user, exp: time.Now().Add(githubProfileTTL)}
	return user, nil
}

func fetchGitHubUser(login string) (*GitHubUser, error) {
	var raw struct {
		Login       string `json:"login"`
		Name        string `json:"name"`
		AvatarURL   string `json:"avatar_url"`
		Bio         string `json:"bio"`
		Company     string `json:"company"`
		Blog        string `json:"blog"`
		Location    string `json:"location"`
		HTMLURL     string `json:"html_url"`
		PublicRepos int    `json:"public_repos"`
		Followers   int    `json:"followers"`
		Following   int    `json:"following"`
		CreatedAt   string `json:"created_at"`
	}
	if err := githubGet("/users/"+login, &raw); err != nil {
		return nil, err
	}

	return &GitHubUser{
		Login:       raw.Login,
		Name:        raw.Name,
		AvatarURL:   raw.AvatarURL,
		Bio:         raw.Bio,
		Company:     raw.Company,
		Blog:        raw.Blog,
		Location:    raw.Location,
		HTMLURL:     raw.HTMLURL,
		PublicRepos: raw.PublicRepos,
		Followers:   raw.Followers,
		Following:   raw.Following,
		CreatedAt:   raw.CreatedAt,
		FetchedAt:   time.Now().UTC().Format(time.RFC3339),
	}, nil
}

// GetGitHubUserReposCached returns one GitHub account's public repositories,
// cached for a week. Forks are dropped: they are not the developer's projects,
// and on an active account they would otherwise bury the ones that are.
func GetGitHubUserReposCached(login string, forceRefresh bool) ([]GitHubRepo, error) {
	if !githubLoginRe.MatchString(login) {
		return nil, fmt.Errorf("invalid github login %q", login)
	}

	ghProfileMu.Lock()
	defer ghProfileMu.Unlock()

	if e, ok := ghRepoCache[login]; ok && !forceRefresh && time.Now().Before(e.exp) {
		return e.repos, e.err
	}

	repos, err := fetchGitHubUserRepos(login)
	if err != nil {
		if e, ok := ghRepoCache[login]; ok && e.repos != nil {
			return e.repos, nil
		}
		ghRepoCache[login] = githubRepoEntry{err: err, exp: time.Now().Add(githubProfileMissTTL)}
		return nil, err
	}

	ghRepoCache[login] = githubRepoEntry{repos: repos, exp: time.Now().Add(githubProfileTTL)}
	return repos, nil
}

func fetchGitHubUserRepos(login string) ([]GitHubRepo, error) {
	var raw []struct {
		Name        string   `json:"name"`
		Description string   `json:"description"`
		HTMLURL     string   `json:"html_url"`
		Language    string   `json:"language"`
		Stars       int      `json:"stargazers_count"`
		Forks       int      `json:"forks_count"`
		Topics      []string `json:"topics"`
		Archived    bool     `json:"archived"`
		Fork        bool     `json:"fork"`
		PushedAt    string   `json:"pushed_at"`
	}

	path := fmt.Sprintf("/users/%s/repos?per_page=%d&sort=pushed", login, githubReposPerPage)
	if err := githubGet(path, &raw); err != nil {
		return nil, err
	}

	repos := []GitHubRepo{}
	for _, r := range raw {
		if r.Fork {
			continue
		}
		topics := r.Topics
		if topics == nil {
			topics = []string{}
		}
		repos = append(repos, GitHubRepo{
			Name:        r.Name,
			Description: r.Description,
			HTMLURL:     r.HTMLURL,
			Language:    r.Language,
			Stars:       r.Stars,
			Forks:       r.Forks,
			Topics:      topics,
			Archived:    r.Archived,
			PushedAt:    r.PushedAt,
		})
	}
	return repos, nil
}
