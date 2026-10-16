package main

import (
	"errors"
	"net/http"
	"net/url"
	"strings"

	"core/apps"
	"core/nip19"
	"core/system"

	"github.com/go-chi/chi/v5"
)

// Developer pages are served by the Yantr UI on hash routes, but the two data
// halves come from separate places on purpose:
//
//   - /api/developers/{key} reads only the local catalogue. It is what makes the
//     page exist at all, and it works with no internet at all.
//   - /api/nostr/profile/{key} enriches that page with the account's kind-0
//     metadata, fetched from relays and cached for a week. It is optional by
//     construction: the UI renders the app list and the QR code whether or not
//     it succeeds, because both are built from the catalog's own key.

// developerKey reads the {key} path param as a Nostr profile identifier.
//
// The segment is percent-decoded first. chi routes on the raw path, so anything
// the client encoded — which is every route param the UI builds — would
// otherwise arrive still encoded and match no catalog entry. A malformed escape
// is not fatal: it falls through to the value validation below, which rejects it.
//
// There is no platform prefix to strip. `developers` holds Nostr keys and
// nothing else, so a "nostr:" prefix would be a prefix around a constant.
func developerKey(raw string) (string, bool) {
	if decoded, err := url.PathUnescape(raw); err == nil {
		raw = decoded
	}

	key := strings.TrimSpace(raw)
	if key == "" {
		return "", false
	}

	// Validated before the catalog lookup so a malformed key is a bad request
	// rather than a 404, which is the more honest answer: the catalog is not
	// what is wrong with it.
	if _, err := nip19.PubkeyHex(key); err != nil {
		return "", false
	}
	return key, true
}

// handleDeveloperApps answers with every catalogue app that credits the given
// Nostr account, plus the identity itself. Nothing leaves the host.
func handleDeveloperApps(w http.ResponseWriter, r *http.Request) {
	key, ok := developerKey(chi.URLParam(r, "handle"))
	if !ok {
		jsonErr(w, 400, "INVALID_DEVELOPER", "Not a Nostr public key (npub1… or nprofile1…)")
		return
	}

	found := apps.AppsByDeveloper(key)
	if len(found) == 0 {
		jsonErr(w, 404, "DEVELOPER_NOT_FOUND", "No app in the catalog credits "+key)
		return
	}

	// Expanded to values so the JSON shape matches the /api/apps entries the UI
	// already knows how to render.
	list := make([]apps.App, 0, len(found))
	for _, a := range found {
		list = append(list, *a)
	}

	jsonResp(w, 200, map[string]interface{}{
		"success":   true,
		"developer": map[string]interface{}{"key": key, "label": "Nostr"},
		"count":     len(list),
		"apps":      list,
	})
}

// handleNostrProfile proxies one developer's kind-0 metadata from the relays.
//
// The relay is a third party that may be slow, blocked or rate-limiting, so its
// answer enriches a page rather than being required by one: the developer page's
// QR code is drawn from the catalog's own key and never consults this.
func handleNostrProfile(w http.ResponseWriter, r *http.Request) {
	key, ok := catalogNostrDeveloper(w, r)
	if !ok {
		return
	}

	force := r.URL.Query().Get("force") == "true"
	profile, err := system.GetNostrProfileCached(key, force)
	if err != nil {
		// An account that has not published a profile is a 404 — a real answer,
		// and one the UI words differently from an unreachable relay.
		if errors.Is(err, system.ErrNostrNoProfile) {
			jsonErr(w, 404, "NOSTR_PROFILE_NOT_FOUND", err.Error())
			return
		}
		jsonErr(w, 502, "NOSTR_PROFILE_UNAVAILABLE", err.Error())
		return
	}

	jsonResp(w, 200, map[string]interface{}{"success": true, "profile": profile})
}

// catalogNostrDeveloper validates the {key} path param against the catalogue. It
// writes the error response itself and reports false when the caller should stop.
//
// Only keys some app declares are served. Without that check this endpoint would
// proxy queries to public relays for any key anyone asked about, which makes it
// an open relay and its cache unbounded — the same reasoning the GitHub proxies
// used, which is why they could go when the platform did.
//
// Matching is case-sensitive. bech32 is lowercase by spec, but the catalog
// deliberately does not rewrite a key's case on the way in, so a key differing
// only in case is a different string here.
func catalogNostrDeveloper(w http.ResponseWriter, r *http.Request) (string, bool) {
	raw := chi.URLParam(r, "key")
	if decoded, err := url.PathUnescape(raw); err == nil {
		raw = decoded
	}

	if strings.TrimSpace(raw) == "" {
		jsonErr(w, 400, "INVALID_NOSTR_KEY", "A Nostr public key is required")
		return "", false
	}

	key, ok := developerKey(raw)
	if !ok {
		jsonErr(w, 400, "INVALID_NOSTR_KEY", "Not a Nostr profile key")
		return "", false
	}

	if len(apps.AppsByDeveloper(key)) == 0 {
		jsonErr(w, 404, "DEVELOPER_NOT_FOUND", "No app in the catalog credits "+key)
		return "", false
	}
	return key, true
}
