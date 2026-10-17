package main

import (
	"net/http"
	"net/url"
	"strings"

	"core/apps"
	"core/nip19"
	"core/system"

	"github.com/go-chi/chi/v5"
)

// Developer pages are served by the Yantr UI on hash routes, and the data comes
// from two places on purpose:
//
//   - /api/developers/{key} reads only the local catalogue. It is what makes the
//     page exist at all, and it works with no internet at all.
//   - /api/nostr/relays tells the UI which relays to ask. The kind-0 metadata
//     itself is fetched by the browser (ui/src/utils/relay.js), which is also
//     where comments are read and published.
//
// The kind-0 used to be proxied through /api/nostr/profile/{key}. That made
// Yantr a Nostr client on behalf of keys it did not issue: the endpoint needed a
// gate against becoming an open relay proxy, and a week-long cache for data the
// reader's own browser could have asked for once. The query now leaves from the
// machine that will display it, which also means the answer can be verified —
// the browser has nostr-tools, the backend never had a schnorr verifier.

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

// handleNostrRelays tells the UI which relays to ask for Nostr data.
//
// The whole payload is configuration, not data: which relays Yantr talks to is
// YANTR_NOSTR_RELAYS, and the browser holds every Nostr conversation itself.
// Serving the set here is what keeps that override in one place — a host whose
// network blocks the public relays redirects every read and every publish by
// setting one variable — and it is why the UI has a fallback list at all
// (utils/nostr.js) rather than a second copy that can be edited.
//
// No key is involved and nothing is fetched, so this needs no catalogue gate:
// there is no query to proxy and no cache to bound. The UI falls back to its own
// list if this endpoint cannot answer.
func handleNostrRelays(w http.ResponseWriter, r *http.Request) {
	relays := system.NostrRelays()
	if len(relays) == 0 {
		// Unreachable by construction (defaultNostrRelays never returns empty),
		// but an empty list would read to the UI as "use no relays", which is
		// worse than the built-in set. Say so rather than serve nothing.
		jsonErr(w, 500, "NOSTR_RELAYS_EMPTY", "No Nostr relay configured")
		return
	}
	jsonResp(w, 200, map[string]interface{}{"success": true, "relays": relays})
}
