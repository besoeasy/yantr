// Package podman provides a shared Podman API client and helper utilities.
package podman

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	dockerclient "github.com/docker/docker/client"
)

// Client is the shared Podman API client instance.
var Client *dockerclient.Client

// SocketPath is the Podman socket path used by the client.
var SocketPath string

func resolveSocketPath() string {
	if s := os.Getenv("CONTAINER_HOST"); s != "" {
		s = strings.TrimPrefix(s, "unix://")
		if _, err := os.Stat(s); err == nil {
			return s
		}
	}
	if s := os.Getenv("PODMAN_SOCKET"); s != "" {
		return s
	}
	if s := os.Getenv("DOCKER_SOCKET"); s != "" {
		return s
	}

	// 1. Check XDG_RUNTIME_DIR (standard user session path)
	if xdg := os.Getenv("XDG_RUNTIME_DIR"); xdg != "" {
		p := xdg + "/podman/podman.sock"
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}

	// 2. Check current UID user socket
	uid := os.Getuid()
	userSocket := fmt.Sprintf("/run/user/%d/podman/podman.sock", uid)
	if _, err := os.Stat(userSocket); err == nil {
		return userSocket
	}

	// 3. Check standard rootless UID 1000
	if _, err := os.Stat("/run/user/1000/podman/podman.sock"); err == nil {
		return "/run/user/1000/podman/podman.sock"
	}

	// 4. Container-mounted or rootful podman paths
	for _, candidate := range []string{
		"/run/podman/podman.sock",
		"/var/run/podman/podman.sock",
	} {
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
	}

	// Default fallback to container mount path
	return "/run/podman/podman.sock"
}

func init() {
	SocketPath = resolveSocketPath()

	transport := &http.Transport{
		DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
			return (&net.Dialer{Timeout: 10 * time.Second}).DialContext(ctx, "unix", SocketPath)
		},
	}
	httpClient := &http.Client{Transport: transport}

	var err error
	Client, err = dockerclient.NewClientWithOpts(
		dockerclient.WithHost("unix://"+SocketPath),
		dockerclient.WithHTTPClient(httpClient),
		dockerclient.WithAPIVersionNegotiation(),
	)
	if err != nil {
		panic("failed to create Podman client: " + err.Error())
	}
}

// Background returns a context suitable for Podman API calls.
func Background() context.Context {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	_ = cancel
	return ctx
}

// Ctx returns a plain background context for operations that might take a while.
func Ctx() context.Context {
	return context.Background()
}
