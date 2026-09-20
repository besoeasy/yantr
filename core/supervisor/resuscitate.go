package supervisor

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	dockerctr "github.com/docker/docker/api/types/container"

	"core/compose"
	"core/podman"
	"core/shared"
)

// Resuscitate sequentially starts all stacks that were marked as running before shutdown.
func Resuscitate(appsDir string, getComposeCmd func() (string, []string, error)) {
	stacks := GetRunningStacks()
	if len(stacks) == 0 {
		SetBootStatus(BootStatus{
			Resuscitating: false,
			Total:         0,
			Completed:     0,
			Message:       "No stacks to resuscitate",
		})
		return
	}

	total := len(stacks)
	SetBootStatus(BootStatus{
		Resuscitating: true,
		Total:         total,
		Completed:     0,
		Message:       fmt.Sprintf("Resuscitating %d stacks...", total),
	})

	shared.Log("info", fmt.Sprintf("[supervisor] starting boot resuscitation for %d stack(s)", total))

	// Inspect running containers to see which stacks are already active
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	activeContainers, err := podman.ContainerList(ctx, dockerctr.ListOptions{All: false})
	cancel()

	activeProjects := make(map[string]bool)
	if err == nil {
		for _, c := range activeContainers {
			if proj := compose.ComposeProjectLabel(c.Labels); proj != "" {
				activeProjects[proj] = true
			}
		}
	}

	completed := 0

	for _, s := range stacks {
		SetBootStatus(BootStatus{
			Resuscitating: true,
			Total:         total,
			Completed:     completed,
			CurrentStack:  s.ProjectID,
			Message:       fmt.Sprintf("Resuscitating stack %s (%d/%d)", s.ProjectID, completed+1, total),
		})

		// If all containers for this project are already up, skip
		if activeProjects[s.ProjectID] {
			shared.Log("info", fmt.Sprintf("[supervisor] stack %q is already running (%d/%d)", s.ProjectID, completed+1, total))
			completed++
			continue
		}

		appPath := filepath.Join(appsDir, s.AppID)
		ref := compose.GetProjectComposeRef(appPath, s.ProjectID)

		if _, err := os.Stat(ref.ComposePath); err != nil {
			shared.Log("warn", fmt.Sprintf("[supervisor] compose file missing for stack %q: %v", s.ProjectID, err))
			completed++
			continue
		}

		cmdName, cmdArgs, err := getComposeCmd()
		if err != nil {
			shared.Log("error", fmt.Sprintf("[supervisor] failed to get compose command for %q: %v", s.ProjectID, err))
			completed++
			continue
		}

		envMap, _ := compose.GetComposeProcessEnv(appPath, s.ProjectID, podman.SocketPath, podman.HostSocket())
		var envList []string
		for k, v := range envMap {
			envList = append(envList, k+"="+v)
		}

		args := append(cmdArgs, "-p", s.ProjectID, "-f", ref.ComposeFile, "up", "-d")
		shared.Log("info", fmt.Sprintf("[supervisor] resuscitating stack %q (%d/%d)...", s.ProjectID, completed+1, total))

		execCtx, execCancel := context.WithTimeout(context.Background(), 2*time.Minute)
		cmd := exec.CommandContext(execCtx, cmdName, args...)
		cmd.Dir = appPath
		cmd.Env = append(os.Environ(), envList...)

		var stdout, stderr bytes.Buffer
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr

		runErr := cmd.Run()
		execCancel()

		if runErr != nil {
			shared.Log("error", fmt.Sprintf("[supervisor] failed to resuscitate stack %q: %v\n%s", s.ProjectID, runErr, stderr.String()))
		} else {
			shared.Log("info", fmt.Sprintf("[supervisor] stack %q resuscitated successfully (%d/%d)", s.ProjectID, completed+1, total))
		}

		completed++
		// Gentle delay between sequential stack startups to prevent I/O and CPU spikes
		time.Sleep(500 * time.Millisecond)
	}

	SetBootStatus(BootStatus{
		Resuscitating: false,
		Total:         total,
		Completed:     completed,
		Message:       fmt.Sprintf("Resuscitation complete (%d/%d)", completed, total),
	})
	shared.Log("info", fmt.Sprintf("[supervisor] boot resuscitation complete: %d/%d stacks online", completed, total))
}
