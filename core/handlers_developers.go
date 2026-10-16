package main

import (
	"core/apps"
	"core/system"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
)

// Developer pages are served by the Yantr UI on hash routes, but the two data
// halves come from separate places on purpose:
//
//   - /api/developers/{handle} reads only the local catalogue. It is what makes
//     the page exist at all, and it works with no internet, no GitHub account
//     and no rate-limit budget.
//   - /api/github/user/{login} and .../repos enrich that page with GitHub
//     profile data, cached for a week. They are optional by construction: the
//     UI renders the app list whether or not either succeeds.

// developerRef parses a "platform:handle" path segment, defaulting to GitHub so
// the friendly /#/developer/besoeasy short form works. The second return is
// false when the reference cannot be interpreted at all.
func developerRef(raw string) (string, string, bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", "", false
	}

	platform, handle := "github", raw
	if before, after, ok := strings.Cut(raw, ":"); ok {
		platform = strings.ToLower(strings.TrimSpace(before))
		handle = strings.TrimSpace(after)
	}

	handle = strings.TrimPrefix(handle, "@")
	if handle == "" {
		return "", "", false
	}

	switch platform {
	case "github", "gitlab", "codeberg", "nostr":
	default:
		return "", "", false
	}
	return platform, handle, true
}

// handleDeveloperApps answers with every catalogue app that declares the given
// developer, plus the developer identity itself. Nothing leaves the host.
func handleDeveloperApps(w http.ResponseWriter, r *http.Request) {
	platform, handle, ok := developerRef(chi.URLParam(r, "handle"))
	if !ok {
		jsonErr(w, 400, "INVALID_DEVELOPER", "Developer reference must be \"handle\" or \"platform:handle\"")
		return
	}

	found := apps.AppsByDeveloper(platform, handle)
	if len(found) == 0 {
		jsonErr(w, 404, "DEVELOPER_NOT_FOUND", "No app in the catalog declares "+platform+":"+handle)
		return
	}

	// Expand to values so the JSON shape matches the /api/apps entries the UI
	// already knows how to render.
	list := make([]apps.App, 0, len(found))
	for _, a := range found {
		list = append(list, *a)
	}

	jsonResp(w, 200, map[string]interface{}{
		"success":   true,
		"developer": map[string]interface{}{"platform": platform, "handle": handle},
		"count":     len(list),
		"apps":      list,
	})
}

// handleGitHubUser proxies one developer profile from GitHub's API.
//
// The proxy exists for the same reasons the repo-stats one does: the
// unauthenticated API allows 60 requests an hour per source IP, which a NAT'd
// homelab shares, and a developer page should not hand every visitor's IP to
// GitHub. It is also the reason a login must be declared in the catalogue
// before it is served — otherwise this endpoint is an open relay to
// api.github.com and an unbounded cache to go with it.
func handleGitHubUser(w http.ResponseWriter, r *http.Request) {
	login, ok := catalogGitHubDeveloper(w, r)
	if !ok {
		return
	}

	force := r.URL.Query().Get("force") == "true"
	user, err := system.GetGitHubUserCached(login, force)
	if err != nil {
		// 502, not 500: the failure is upstream, and the UI treats this as
		// "no profile card", which is a supported state.
		jsonErr(w, 502, "GITHUB_PROFILE_UNAVAILABLE", err.Error())
		return
	}

	jsonResp(w, 200, map[string]interface{}{"success": true, "user": user})
}

// handleGitHubUserRepos proxies one developer's public repositories.
func handleGitHubUserRepos(w http.ResponseWriter, r *http.Request) {
	login, ok := catalogGitHubDeveloper(w, r)
	if !ok {
		return
	}

	force := r.URL.Query().Get("force") == "true"
	repos, err := system.GetGitHubUserReposCached(login, force)
	if err != nil {
		jsonErr(w, 502, "GITHUB_REPOS_UNAVAILABLE", err.Error())
		return
	}

	jsonResp(w, 200, map[string]interface{}{"success": true, "repos": repos})
}

// catalogGitHubDeveloper validates the {login} path param and resolves it to a
// catalogue-declared GitHub developer handle. It writes the error response
// itself and reports false when the caller should stop.
//
// Only lowercase catalogue handles are matched, mirroring the normalisation in
// apps.parseDevelopers, so a request cannot mint a cache entry for an arbitrary
// GitHub account.
func catalogGitHubDeveloper(w http.ResponseWriter, r *http.Request) (string, bool) {
	login := strings.ToLower(strings.TrimSpace(chi.URLParam(r, "login")))
	if login == "" {
		jsonErr(w, 400, "INVALID_LOGIN", "A GitHub login is required")
		return "", false
	}

	if len(apps.AppsByDeveloper("github", login)) == 0 {
		jsonErr(w, 404, "DEVELOPER_NOT_FOUND", "No app in the catalog declares github:"+login)
		return "", false
	}
	return login, true
}
