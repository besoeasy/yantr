package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os/exec"
	"time"

	"core/podman"

	"github.com/go-chi/chi/v5"
)

// Probe known shell paths inside the container. $SHELL and the image's default
// command do not reliably identify a shell that can actually be executed.
var shellCandidates = []string{"/bin/bash", "/bin/ash", "/bin/sh"}

type shellProbe func(context.Context, string, string) (bool, error)

func probeContainerShell(ctx context.Context, containerID, shell string) (bool, error) {
	cmd := exec.CommandContext(ctx, "podman", "exec", containerID, shell, "-c", ":")
	if err := cmd.Run(); err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) && exitErr.ExitCode() == 127 {
			return false, nil // executable not present in this container
		}
		return false, err
	}
	return true, nil
}

func detectContainerShell(ctx context.Context, containerID string, probe shellProbe) (string, error) {
	for _, shell := range shellCandidates {
		found, err := probe(ctx, containerID, shell)
		if err != nil {
			return "", err
		}
		if found {
			return shell, nil
		}
	}
	return "", nil // e.g. a distroless image
}

func handleContainerShell(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()

	info, err := podman.ContainerInspect(ctx, id)
	if err != nil {
		jsonErr(w, http.StatusNotFound, "CONTAINER_NOT_FOUND", "Container not found")
		return
	}
	if info.State == nil || !info.State.Running {
		jsonErr(w, http.StatusConflict, "CONTAINER_NOT_RUNNING", "Start the container to detect its shell")
		return
	}

	shell, err := detectContainerShell(ctx, info.ID, probeContainerShell)
	if err != nil {
		jsonErr(w, http.StatusBadGateway, "SHELL_DETECTION_FAILED", fmt.Sprintf("Could not detect container shell: %v", err))
		return
	}
	jsonResp(w, http.StatusOK, map[string]interface{}{"success": true, "shell": shell})
}
