// Package selfinstall implements the bootstrap self-install logic.
//
// When the user runs the minimal install:
//
//	docker run -v /var/run/docker.sock:/var/run/docker.sock ghcr.io/besoeasy/yantr
//
// This detects the container is NOT fully configured (host network + volumes mount),
// then uses the Docker socket to:
//  1. Remove any existing "yantr" container
//  2. Launch a new container with full production configuration
//  3. Exit this bootstrap container
package selfinstall

import (
	"context"
	"fmt"
	"os"
	"strings"

	dockertypes "github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	"core/docker"
)

var (
	containerName = envOr("YANTR_CONTAINER_NAME", "yantr")
	imageName     = envOr("YANTR_IMAGE", "ghcr.io/besoeasy/yantr")
	socketPath    = envOr("DOCKER_SOCKET", "/var/run/docker.sock")
)

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

// RunIfNeeded checks whether we are fully configured. Returns true if the
// bootstrap ran (caller should exit); false if normal startup should continue.
func RunIfNeeded() (bool, error) {
	fully, err := isFullyConfigured()
	if err != nil {
		fmt.Printf("[selfinstall] Could not check configuration: %v\n", err)
		return false, nil
	}
	if fully {
		return false, nil
	}

	fmt.Println("[selfinstall] Minimal install detected. Launching fully-configured yantr container...")

	if err := removeExisting(containerName); err != nil {
		return false, fmt.Errorf("failed to remove existing container: %w", err)
	}

	if err := launchFullContainer(); err != nil {
		return false, fmt.Errorf("failed to launch container: %w", err)
	}

	fmt.Println("[selfinstall] yantr container started successfully.")
	fmt.Println("[selfinstall] Access the UI at http://localhost:5252")
	fmt.Println("[selfinstall] This bootstrap container will now exit.")
	return true, nil
}

func isFullyConfigured() (bool, error) {
	hostname, err := os.Hostname()
	if err != nil {
		return true, nil
	}

	info, err := docker.ContainerInspect(context.Background(), hostname)
	if err != nil {
		// Not inside a container (dev mode) — treat as fully configured
		return true, nil
	}

	hc := info.HostConfig
	name := strings.TrimPrefix(info.Name, "/")

	hasHostNetwork := hc.NetworkMode == "host"
	hasVolMount := false
	for _, bind := range hc.Binds {
		if strings.Contains(bind, "/var/lib/docker/volumes") {
			hasVolMount = true
			break
		}
	}
	isNamed := name == containerName

	return hasHostNetwork && hasVolMount && isNamed, nil
}

func removeExisting(name string) error {
	info, err := docker.ContainerInspect(context.Background(), name)
	if err != nil {
		return nil // doesn't exist
	}
	if info.State.Running {
		fmt.Printf("[selfinstall] Stopping existing %q container...\n", name)
		_ = docker.ContainerStop(context.Background(), name, dockertypes.StopOptions{Timeout: intPtr(5)})
	}
	fmt.Printf("[selfinstall] Removing existing %q container...\n", name)
	return docker.ContainerRemove(context.Background(), name, dockertypes.RemoveOptions{Force: true})
}

func launchFullContainer() error {
	config := &dockertypes.Config{
		Image: imageName,
		Env:   buildEnv(),
	}
	hostConfig := &dockertypes.HostConfig{
		NetworkMode:   "host",
		RestartPolicy: dockertypes.RestartPolicy{Name: "unless-stopped"},
		Binds: []string{
			socketPath + ":/var/run/docker.sock",
			"/var/lib/docker/volumes:/var/lib/docker/volumes",
		},
	}
	networkConfig := &network.NetworkingConfig{}

	resp, err := docker.ContainerCreate(
		context.Background(),
		config,
		hostConfig,
		networkConfig,
		nil,
		containerName,
	)
	if err != nil {
		return err
	}
	return docker.ContainerStart(context.Background(), resp.ID, dockertypes.StartOptions{})
}

var passThroughEnvKeys = []string{
	"TZ", "YANTR_CONTAINER_NAME", "YANTR_IMAGE",
	"YANTR_AUTH_SECRET", "YANTR_AUTH_USERNAME",
	"YANTR_SELFUPDATE", "YANTR_SELFUPDATE_INTERVAL",
	"DOCKER_SOCKET", "UI_BASE_PATH", "VITE_BASE_PATH",
}

func buildEnv() []string {
	var env []string
	for _, k := range passThroughEnvKeys {
		if v := os.Getenv(k); v != "" {
			env = append(env, k+"="+v)
		}
	}
	return env
}

func intPtr(n int) *int { return &n }
