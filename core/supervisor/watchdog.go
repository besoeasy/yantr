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

// restartHistoryTTL is the only window isFlapping ever consults. An entry whose
// newest timestamp is older than this can never influence a flap decision, so
// it is pure retained memory.
const restartHistoryTTL = 60 * time.Second

// ForgetContainer drops a container's restart history. Call it when a container
// is removed; otherwise every ID the watchdog has ever seen is retained for the
// life of the process, because isFlapping only prunes keys it is called with.
func ForgetContainer(containerID string) {
	if containerID == "" {
		return
	}
	restartHistoryMu.Lock()
	defer restartHistoryMu.Unlock()
	delete(restartHistory, containerID)
}

// sweepRestartHistory drops entries with no restart inside restartHistoryTTL,
// bounding the map to containers that died recently.
func sweepRestartHistory() {
	cutoff := time.Now().Add(-restartHistoryTTL)

	restartHistoryMu.Lock()
	defer restartHistoryMu.Unlock()
	for id, times := range restartHistory {
		if len(times) == 0 || times[len(times)-1].Before(cutoff) {
			delete(restartHistory, id)
		}
	}
}

// StartHistorySweeper periodically prunes stale restart history entries.
func StartHistorySweeper(ctx context.Context) {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			shared.Log("info", "[watchdog] history sweeper stopped")
			return
		case <-ticker.C:
			sweepRestartHistory()
		}
	}
}

// isFlapping checks if a container has crashed too many times recently.
func isFlapping(containerID string) bool {
	restartHistoryMu.Lock()
	defer restartHistoryMu.Unlock()

	now := time.Now()
	cutoff := now.Add(-restartHistoryTTL)

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

// shortID truncates a container ID for log output.
//
// The watchdog runs on its own goroutine, started with context.Background(), so
// middleware.Recoverer does not cover it: a panic here takes down the whole
// process. A die event's Actor.ID is engine-supplied and only guaranteed
// non-empty, so slicing it directly panics on any ID shorter than 12 chars.
func shortID(id string) string {
	if len(id) > 12 {
		return id[:12]
	}
	return id
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
	service := compose.ComposeServiceLabel(msg.Actor.Attributes)

	// Don't auto-restart if the stack was intentionally stopped by the user, if
	// this individual service was, or if the stack is currently being torn down
	// (reaper / stack delete / container delete).
	if !ShouldAutoRestart(projectID, service) {
		return
	}

	inspectCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	info, err := podman.ContainerInspect(inspectCtx, containerID)
	cancel()
	if err != nil {
		return
	}

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

	policy := ""
	if info.HostConfig != nil {
		policy = string(info.HostConfig.RestartPolicy.Name)
	}

	// Stacks with no restart policy or explicitly set to 'no' are not auto-restarted
	if policy == "no" {
		return
	}

	if isFlapping(containerID) {
		shared.Log("warn", fmt.Sprintf("[watchdog] container %s (project %s) crashed repeatedly (>5 times in 60s). Suppressing restart to prevent flap loop.", shortID(containerID), projectID))
		return
	}

	shared.Log("warn", fmt.Sprintf("[watchdog] container %s (project %s, service %s) died unexpectedly (exit %d). Auto-restarting...",
		shortID(containerID), projectID, service, exitCode))

	// Deliberately restart the single container rather than the project.
	//
	// The watchdog's job is crash recovery, not reconciliation: ContainerStart
	// re-runs the exact same container spec, so it does not re-pull, re-create,
	// or disturb the project's healthy services. `compose up -d` would converge
	// the project correctly but churns every service on each crash and can
	// cascade, which is the wrong trade for a container that merely died.
	//
	// The known gap is dependency ordering: if the dead service is a database
	// the app depends on, the app stays running against a downed dependency
	// until it exits on its own and `restart: unless-stopped` reorders the
	// stack. Accepted deliberately — see issue #96. Compose reordering on every
	// crash is the more expensive failure.
	startCtx, startCancel := context.WithTimeout(context.Background(), 10*time.Second)
	startErr := podman.ContainerStart(startCtx, containerID, dockerctr.StartOptions{})
	startCancel()

	if startErr != nil {
		shared.Log("error", fmt.Sprintf("[watchdog] failed to restart container %s: %v", shortID(containerID), startErr))
	} else {
		shared.Log("info", fmt.Sprintf("[watchdog] container %s restarted successfully", shortID(containerID)))
	}
}
