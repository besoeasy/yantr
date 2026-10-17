package main

import (
	"net/http"

	"core/presence"
)

// handleFleetStats serves the Fleet page: an aggregate of the nodes that have
// pinged the public presence feed — how many, where, on what OS, with how much
// RAM, and which apps they run.
//
// The endpoint is named for what it returns, not for where the data comes from.
// The feed itself is the presence feed, and it keeps that name: it is a wire
// format with a public topic and an opt-out, not an internal detail.
func handleFleetStats(w http.ResponseWriter, r *http.Request) {
	force := r.URL.Query().Get("refresh") == "1"
	stats, err := presence.GetFleetStatsCached(force)
	if err != nil {
		jsonErr(w, http.StatusBadGateway, "FLEET_FETCH_FAILED", err.Error())
		return
	}
	jsonResp(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"stats":   stats,
	})
}
