package supervisor

import (
	"context"
	"fmt"
	"sync"
	"time"

	dockerctr "github.com/docker/docker/api/types/container"
	dockerevents "github.com/docker/docker/api/types/events"
	dockerfilters "github.com/docker/docker/api/types/filters"

	"core/compose"
	"core/podman"
	"core/shared"
)

var (
	restartHistoryMu sync.Mutex
	restartHistory   = make(map[string][]time.Time)
)

// isFlapping checks if a container has crashed too many times recently.
func isFlapping(containerID string) bool {
	restartHistoryMu.Lock()
	defer restartHistoryMu.Unlock()

	now := time.Now()
	cutoff := now.Add(-60 * time.Second)

	var recent []time.Time
	for _, t := range restartHistory[containerID] {
		if t.After(cutoff) {
			recent = append(recent, t)
		}
	}

	if len(recent) >= 5 {
		restartHistory[containerID] = recent
		return true
	}

	recent = append(recent, now)
	restartHistory[containerID] = recent
	return false
}

// StartWatchdog runs the background event stream listener that detects and revives crashed containers.
func StartWatchdog(ctx context.Context) {
	shared.Log("info", "[watchdog] live event watchdog loop started")

	for {
		select {
		case <-ctx.Done():
			shared.Log("info", "[watchdog] watchdog stopped")
			return
		default:
		}

		filters := dockerfilters.NewArgs(
			dockerfilters.Arg("type", "container"),
			dockerfilters.Arg("event", "die"),
		)

		msgCh, errCh := podman.Client.Events(ctx, dockerevents.ListOptions{Filters: filters})

		for {
			select {
			case <-ctx.Done():
				return

			case err, ok := <-errCh:
				if !ok || err == nil {
					break
				}
				shared.Log("warn", fmt.Sprintf("[watchdog] event stream error: %v, reconnecting in 5s...", err))
				time.Sleep(5 * time.Second)
				goto reconnect

			case msg, ok := <-msgCh:
				if !ok {
					goto reconnect
				}
				handleDieEvent(ctx, msg)
			}
		}

	reconnect:
		time.Sleep(2 * time.Second)
	}
}

func handleDieEvent(ctx context.Context, msg dockerevents.Message) {
	containerID := msg.Actor.ID
	if containerID == "" {
		return
	}

	projectID := compose.ComposeProjectLabel(msg.Actor.Attributes)
	if projectID == "" {
		return
	}

	// Don't auto-restart if the stack was intentionally stopped by the user
	// or is currently being torn down (reaper / stack delete / container delete).
	if !IsStackRunning(projectID) || IsStackRemoving(projectID) {
		return
	}

	inspectCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	info, err := podman.ContainerInspect(inspectCtx, containerID)
	cancel()
	if err != nil {
		return
	}

	// If it exited with code 0 without being killed, it completed cleanly
	if info.State != nil && info.State.ExitCode == 0 && !info.State.OOMKilled {
		return
	}

	exitCode := 0
	if info.State != nil {
		exitCode = info.State.ExitCode
	}

	// Exit 143 (SIGTERM) means the container was gracefully stopped, not crashed.
	// This happens during intentional teardowns (compose down, podman stop) and
	// should not be treated as an unexpected death.
	if exitCode == 143 {
		return
	}

	// Check restart policy
	policy := ""
	if info.HostConfig != nil {
		policy = string(info.HostConfig.RestartPolicy.Name)
	}

	// Stacks with no restart policy or explicitly set to 'no' are not auto-restarted
	if policy == "no" {
		return
	}

	if isFlapping(containerID) {
		shared.Log("warn", fmt.Sprintf("[watchdog] container %s (project %s) crashed repeatedly (>5 times in 60s). Suppressing restart to prevent flap loop.", containerID[:12], projectID))
		return
	}

	shared.Log("warn", fmt.Sprintf("[watchdog] container %s (project %s, service %s) died unexpectedly (exit %d). Auto-restarting...",
		containerID[:12], projectID, compose.ComposeServiceLabel(msg.Actor.Attributes), exitCode))

	startCtx, startCancel := context.WithTimeout(context.Background(), 10*time.Second)
	startErr := podman.ContainerStart(startCtx, containerID, dockerctr.StartOptions{})
	startCancel()

	if startErr != nil {
		shared.Log("error", fmt.Sprintf("[watchdog] failed to restart container %s: %v", containerID[:12], startErr))
	} else {
		shared.Log("info", fmt.Sprintf("[watchdog] container %s restarted successfully", containerID[:12]))
	}
}
