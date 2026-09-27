package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
)

func TestHandleJobsEndpoints(t *testing.T) {
	// Setup test router with handlers
	r := chi.NewRouter()
	r.Get("/api/jobs", handleJobsList)
	r.Get("/api/jobs/active", handleJobsActive)
	r.Get("/api/jobs/{id}", handleJobDetail)

	// Create a test job
	j := globalJobs.Create("deploy", "test-app", "Deploy test-app")
	j.AddLog("Step 1 done")

	// 1. Test GET /api/jobs
	req := httptest.NewRequest("GET", "/api/jobs", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var listResp struct {
		Success bool  `json:"success"`
		Jobs    []Job `json:"jobs"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &listResp); err != nil {
		t.Fatalf("failed to parse list resp: %v", err)
	}
	if !listResp.Success || len(listResp.Jobs) == 0 {
		t.Fatalf("expected at least 1 job in list")
	}

	// 2. Test GET /api/jobs/active?target=test-app
	req = httptest.NewRequest("GET", "/api/jobs/active?target=test-app", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var activeResp struct {
		Success bool `json:"success"`
		Active  bool `json:"active"`
		Job     *Job `json:"job"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &activeResp); err != nil {
		t.Fatalf("failed to parse active resp: %v", err)
	}
	if !activeResp.Active || activeResp.Job == nil || activeResp.Job.ID != j.ID {
		t.Fatalf("expected active job matching %s", j.ID)
	}

	// 3. Test GET /api/jobs/{id}
	req = httptest.NewRequest("GET", "/api/jobs/"+j.ID, nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var detailResp struct {
		Success bool `json:"success"`
		Job     Job  `json:"job"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &detailResp); err != nil {
		t.Fatalf("failed to parse detail resp: %v", err)
	}
	if detailResp.Job.ID != j.ID || len(detailResp.Job.Logs) != 1 {
		t.Fatalf("unexpected detail resp job: %+v", detailResp.Job)
	}

	// Complete the job
	j.Complete(map[string]interface{}{"ok": true})

	// Check active again
	req = httptest.NewRequest("GET", "/api/jobs/active?target=test-app", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)

	_ = json.Unmarshal(w.Body.Bytes(), &activeResp)
	if activeResp.Active {
		t.Fatalf("expected active=false after completion")
	}
}
