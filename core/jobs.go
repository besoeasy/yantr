package main

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"core/shared"
)

type JobStatus string

const (
	JobStatusRunning   JobStatus = "running"
	JobStatusCompleted JobStatus = "completed"
	JobStatusFailed    JobStatus = "failed"
)

const maxJobLogs = 500
const maxRetainedJobs = 50

type Job struct {
	ID          string                 `json:"id"`
	Type        string                 `json:"type"`   // "deploy", "autoupdate", "stack_delete", "stack_restart", "container_delete"
	Target      string                 `json:"target"` // appId, projectId, or containerId
	Title       string                 `json:"title"`
	Status      JobStatus              `json:"status"`
	Progress    string                 `json:"progress"`
	Logs        []string               `json:"logs"`
	ExitCode    int                    `json:"exitCode"`
	Result      map[string]interface{} `json:"result,omitempty"`
	Error       string                 `json:"error,omitempty"`
	CreatedAt   time.Time              `json:"createdAt"`
	CompletedAt *time.Time             `json:"completedAt,omitempty"`

	mu sync.RWMutex `json:"-"`
}

func (j *Job) AddLog(line string) {
	line = strings.TrimSpace(line)
	if line == "" {
		return
	}
	j.mu.Lock()
	defer j.mu.Unlock()
	j.Logs = append(j.Logs, line)
	if len(j.Logs) > maxJobLogs {
		j.Logs = j.Logs[len(j.Logs)-maxJobLogs:]
	}
}

func (j *Job) SetProgress(progress string) {
	j.mu.Lock()
	defer j.mu.Unlock()
	j.Progress = progress
}

func (j *Job) Complete(result map[string]interface{}) {
	j.mu.Lock()
	defer j.mu.Unlock()
	now := time.Now()
	j.CompletedAt = &now
	j.Status = JobStatusCompleted
	j.Result = result
	if j.Progress == "" {
		j.Progress = "Completed successfully"
	}
}

func (j *Job) Fail(err error, exitCode int) {
	j.mu.Lock()
	defer j.mu.Unlock()
	now := time.Now()
	j.CompletedAt = &now
	j.Status = JobStatusFailed
	j.ExitCode = exitCode
	if err != nil {
		j.Error = err.Error()
	} else if exitCode != 0 {
		j.Error = fmt.Sprintf("Process exited with code %d", exitCode)
	}
	if j.Progress == "" {
		j.Progress = "Failed"
	}
}

func (j *Job) Snapshot() Job {
	j.mu.RLock()
	defer j.mu.RUnlock()
	logsCopy := make([]string, len(j.Logs))
	copy(logsCopy, j.Logs)
	return Job{
		ID:          j.ID,
		Type:        j.Type,
		Target:      j.Target,
		Title:       j.Title,
		Status:      j.Status,
		Progress:    j.Progress,
		Logs:        logsCopy,
		ExitCode:    j.ExitCode,
		Result:      j.Result,
		Error:       j.Error,
		CreatedAt:   j.CreatedAt,
		CompletedAt: j.CompletedAt,
	}
}

// JobStore manages background operations in memory.
type JobStore struct {
	mu   sync.RWMutex
	jobs map[string]*Job
	seq  []string
}

var globalJobs = &JobStore{
	jobs: make(map[string]*Job),
	seq:  make([]string, 0),
}

func generateJobID(prefix string) string {
	b := make([]byte, 6)
	_, _ = rand.Read(b)
	return fmt.Sprintf("job_%s_%s", prefix, hex.EncodeToString(b))
}

func (s *JobStore) Create(jobType, target, title string) *Job {
	s.mu.Lock()
	defer s.mu.Unlock()

	id := generateJobID(jobType)
	j := &Job{
		ID:        id,
		Type:      jobType,
		Target:    target,
		Title:     title,
		Status:    JobStatusRunning,
		Progress:  "Started",
		Logs:      make([]string, 0),
		CreatedAt: time.Now(),
	}

	s.jobs[id] = j
	s.seq = append(s.seq, id)

	// Enforce retention limit
	if len(s.seq) > maxRetainedJobs {
		removeID := s.seq[0]
		s.seq = s.seq[1:]
		delete(s.jobs, removeID)
	}

	return j
}

func (s *JobStore) Get(id string) *Job {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.jobs[id]
}

func (s *JobStore) FindActiveByTarget(target string) *Job {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, j := range s.jobs {
		j.mu.RLock()
		running := j.Status == JobStatusRunning && j.Target == target
		j.mu.RUnlock()
		if running {
			return j
		}
	}
	return nil
}

func (s *JobStore) List(limit int) []Job {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if limit <= 0 || limit > len(s.seq) {
		limit = len(s.seq)
	}

	result := make([]Job, 0, limit)
	// Iterate in reverse (newest first)
	for i := len(s.seq) - 1; i >= 0 && len(result) < limit; i-- {
		id := s.seq[i]
		if j, ok := s.jobs[id]; ok {
			result = append(result, j.Snapshot())
		}
	}
	return result
}

// lineWriter buffers stream bytes and invokes onLine on complete newlines.
type lineWriter struct {
	builder *strings.Builder
	buf     bytes.Buffer
	onLine  func(string)
	mu      sync.Mutex
}

func newLineWriter(builder *strings.Builder, onLine func(string)) *lineWriter {
	return &lineWriter{
		builder: builder,
		onLine:  onLine,
	}
}

func (w *lineWriter) Write(p []byte) (n int, err error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	w.builder.Write(p)
	w.buf.Write(p)

	for {
		line, err := w.buf.ReadBytes('\n')
		if err != nil {
			// Incomplete line remains in buffer
			w.buf.Write(line)
			break
		}
		str := strings.TrimRight(string(line), "\r\n")
		if str != "" && w.onLine != nil {
			w.onLine(str)
		}
	}

	return len(p), nil
}

func (w *lineWriter) Flush() {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.buf.Len() > 0 {
		str := strings.TrimSpace(w.buf.String())
		if str != "" && w.onLine != nil {
			w.onLine(str)
		}
		w.buf.Reset()
	}
}

// spawnExecJob runs external command with context, streaming lines directly into Job logs.
func spawnExecJob(ctx context.Context, job *Job, name string, args []string, env map[string]string, cwd string) (string, string, int, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	if cwd != "" {
		cmd.Dir = cwd
	}
	if env != nil {
		var envList []string
		for k, v := range env {
			envList = append(envList, k+"="+v)
		}
		cmd.Env = append(os.Environ(), envList...)
	}

	var stdout, stderr strings.Builder

	logHandler := func(line string) {
		if job != nil {
			job.AddLog(line)
		}
		shared.Log("info", fmt.Sprintf("[%s] %s", job.Type, line))
	}

	outW := newLineWriter(&stdout, logHandler)
	errW := newLineWriter(&stderr, logHandler)

	cmd.Stdout = outW
	cmd.Stderr = errW

	err := cmd.Run()
	outW.Flush()
	errW.Flush()

	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			exitCode = 1
		}
	}

	return stdout.String(), stderr.String(), exitCode, err
}
