package main

import (
	"net/http"

	"core/presence"
)

func handlePresenceStats(w http.ResponseWriter, r *http.Request) {
	force := r.URL.Query().Get("refresh") == "1"
	stats, err := presence.GetFleetStatsCached(force)
	if err != nil {
		jsonErr(w, http.StatusBadGateway, "PRESENCE_FETCH_FAILED", err.Error())
		return
	}
	jsonResp(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"stats":   stats,
	})
}
