// Package podman provides a shared Podman API client and helper utilities.
package podman

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	dockerclient "github.com/docker/docker/client"
)

// Client is the shared Podman API client instance.
var Client *dockerclient.Client

// SocketPath is the Podman socket path used by the client.
var SocketPath string

func resolveSocketPath() string {
	for _, key := range []string{"CONTAINER_HOST", "PODMAN_SOCKET", "DOCKER_SOCKET"} {
		s := os.Getenv(key)
		if s == "" {
			continue
		}
		s = strings.TrimPrefix(s, "unix://")
		if _, err := os.Stat(s); err == nil {
			return s
		}
		if key != "CONTAINER_HOST" {
			return s
		}
	}

	// Fallback: the path install.sh bind-mounts the socket to inside the
	// container (install.sh).
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

var warnHostSocketOnce sync.Once

// HostSocket returns the Podman socket path on the host system.
//
// install.sh pins this in the Quadlet as HOST_PODMAN_SOCKET, so the normal
// path is a plain environment read. There is deliberately no fallback
// discovery: guessing the host path from inside the container can only ever
// produce /run/podman/podman.sock, which does not exist on a rootless host —
// so a silent wrong answer is worse than an explicit complaint.
func HostSocket() string {
	if s := os.Getenv("HOST_PODMAN_SOCKET"); s != "" {
		return strings.TrimPrefix(s, "unix://")
	}
	warnHostSocketOnce.Do(func() {
		fmt.Fprintln(os.Stderr, "yantr: HOST_PODMAN_SOCKET is not set, so the host Podman socket "+
			"cannot be identified. Apps mounting /var/run/docker.sock will not deploy correctly. "+
			"install.sh sets it for you.")
	})
	return SocketPath
}
