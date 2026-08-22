package main

import (
	"net/http"

	"core/telemetry"
)

func handleTelemetryStats(w http.ResponseWriter, r *http.Request) {
	force := r.URL.Query().Get("refresh") == "1"
	stats, err := telemetry.GetFleetStatsCached(force)
	if err != nil {
		jsonErr(w, http.StatusBadGateway, "TELEMETRY_FETCH_FAILED", err.Error())
		return
	}
	jsonResp(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"stats":   stats,
	})
}
