package supervisor

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	dockerctr "github.com/docker/docker/api/types/container"

	"core/compose"
	"core/podman"
	"core/shared"
)

// projectService identifies one compose service within a project. Boot
// resuscitation reasons about (project, service) pairs rather than whole
// projects, because a stack can have some services intentionally stopped.
type projectService struct{ project, service string }

// servicesToResuscitate returns the compose services of a stack that boot
// resuscitation should bring up: those declared by the project compose file,
// minus the ones the user intentionally stopped, minus those already running.
//
// An empty result means there is nothing to do for this stack. A compose file
// that cannot be read or parsed yields nil, which the caller treats as
// "revive the whole project" — the safe direction, since a bare
// `compose up -d` is the pre-existing behaviour.
func servicesToResuscitate(composePath, projectID string, active map[projectService]bool) []string {
	doc, err := readComposeDoc(composePath)
	if err != nil {
		return nil
	}
	stopped := GetStackStoppedServices(projectID)

	var want []string
	for _, name := range compose.ServiceNames(doc) {
		if containsString(stopped, name) {
			continue
		}
		if active[projectService{projectID, name}] {
			continue
		}
		want = append(want, name)
	}
	return want
}

func readComposeDoc(path string) (compose.ComposeDoc, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return compose.Parse(string(data))
}

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

	// Inspect running containers to see which (project, service) pairs are
	// already active. The key is the pair, not the project: a stack with one
	// intentionally stopped service still needs its remaining services checked.
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	activeContainers, err := podman.ContainerList(ctx, dockerctr.ListOptions{All: false})
	cancel()

	active := make(map[projectService]bool)
	if err == nil {
		for _, c := range activeContainers {
			proj := compose.ComposeProjectLabel(c.Labels)
			if proj == "" {
				continue
			}
			svc := compose.ComposeServiceLabel(c.Labels)
			if svc == "" && len(c.Names) > 0 {
				svc = strings.TrimPrefix(c.Names[0], "/")
			}
			active[projectService{proj, svc}] = true
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

		appPath := filepath.Join(appsDir, s.AppID)
		ref := compose.GetProjectComposeRef(appPath, s.ProjectID)

		// Determine the service subset to revive: every service declared by the
		// project compose file, minus the ones the user intentionally stopped.
		// When nothing is stopped this is the whole project, preserving the
		// previous `compose up -d` behaviour.
		upServices := servicesToResuscitate(ref.ComposePath, s.ProjectID, active)
		if len(upServices) == 0 {
			shared.Log("info", fmt.Sprintf("[supervisor] stack %q needs nothing (all services up or intentionally stopped) (%d/%d)",
				s.ProjectID, completed+1, total))
			completed++
			continue
		}

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
		// Naming the services narrows the operation to what is actually down, so
		// a service the user stopped stays stopped across a reboot.
		args = append(args, upServices...)
		shared.Log("info", fmt.Sprintf("[supervisor] resuscitating stack %q (%d/%d) services=%v...",
			s.ProjectID, completed+1, total, upServices))

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
