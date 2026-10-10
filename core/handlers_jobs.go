package main

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
)

// handleJobsList returns recent operations with optional filtering.
func handleJobsList(w http.ResponseWriter, r *http.Request) {
	limit := 20
	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 {
			limit = parsed
		}
	}

	target := strings.TrimSpace(r.URL.Query().Get("target"))
	jobType := strings.TrimSpace(r.URL.Query().Get("type"))

	all := globalJobs.List(50)
	filtered := make([]JobSnapshot, 0, len(all))

	for _, j := range all {
		if target != "" && j.Target != target {
			continue
		}
		if jobType != "" && j.Type != jobType {
			continue
		}
		filtered = append(filtered, j)
		if len(filtered) >= limit {
			break
		}
	}

	jsonResp(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"jobs":    filtered,
	})
}

// handleJobsActive returns any currently running job(s), optionally filtered by target.
func handleJobsActive(w http.ResponseWriter, r *http.Request) {
	target := strings.TrimSpace(r.URL.Query().Get("target"))

	if target != "" {
		active := globalJobs.FindActiveByTarget(target)
		if active == nil {
			jsonResp(w, http.StatusOK, map[string]interface{}{
				"success": true,
				"active":  false,
			})
			return
		}
		jsonResp(w, http.StatusOK, map[string]interface{}{
			"success": true,
			"active":  true,
			"job":     active.Snapshot(),
		})
		return
	}

	all := globalJobs.List(50)
	running := make([]JobSnapshot, 0)
	for _, j := range all {
		if j.Status == JobStatusRunning {
			running = append(running, j)
		}
	}

	jsonResp(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"jobs":    running,
	})
}

// handleJobDetail returns details and full logs for a specific job.
func handleJobDetail(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		jsonErr(w, http.StatusBadRequest, "ID_REQUIRED", "Job ID is required")
		return
	}

	j := globalJobs.Get(id)
	if j == nil {
		jsonErr(w, http.StatusNotFound, "JOB_NOT_FOUND", "Job not found")
		return
	}

	jsonResp(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"job":     j.Snapshot(),
	})
}
