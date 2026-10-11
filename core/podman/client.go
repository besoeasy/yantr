// Package podman provides a shared Podman API client and helper utilities.
package podman

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
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
	// container (website/install.sh).
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

// HostSocket returns the Podman socket path on the host system.
//
// install.sh pins this in the Quadlet as HOST_PODMAN_SOCKET, so the installed
// path is a plain environment read with no subprocess involved. Deployments
// that skip install.sh — a hand-rolled `podman run`, for example — fall back
// to asking the Podman API, and finally to SocketPath (the in-container
// mount point).
func HostSocket() string {
	if s := os.Getenv("HOST_PODMAN_SOCKET"); s != "" {
		return strings.TrimPrefix(s, "unix://")
	}
	if s := discoverHostSocket(); s != "" {
		warnHostSocketOnce("HOST_PODMAN_SOCKET is unset; discovered the host Podman socket at " +
			s + ". Re-run install.sh to set it explicitly.")
		return s
	}
	warnHostSocketOnce("HOST_PODMAN_SOCKET is unset and the host Podman socket could not be " +
		"discovered; apps mounting /var/run/docker.sock may fail.")
	return SocketPath
}

var (
	hostSocketMu     sync.Mutex
	discoveredSocket string
	warnOnce         sync.Once
)

func warnHostSocketOnce(msg string) {
	warnOnce.Do(func() { fmt.Fprintln(os.Stderr, "yantr: "+msg) })
}

// discoverHostSocket asks the Podman API where its socket lives on the host.
//
// Only a successful result is cached. An earlier implementation cached the
// first result unconditionally through a sync.Once, so a single timeout
// pinned a value for the whole process lifetime — and the fallback that
// followed (/run/podman/podman.sock) does not exist on a rootless host, so
// every app mounting /var/run/docker.sock failed until Yantr restarted.
func discoverHostSocket() string {
	hostSocketMu.Lock()
	cached := discoveredSocket
	hostSocketMu.Unlock()
	if cached != "" {
		return cached
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	out, err := exec.CommandContext(ctx, "podman", "info", "--format", "{{.Host.RemoteSocket.Path}}").Output()
	if err != nil {
		return ""
	}
	val := strings.TrimPrefix(strings.TrimSpace(string(out)), "unix://")
	if val == "" || val == "<no value>" {
		return ""
	}

	hostSocketMu.Lock()
	discoveredSocket = val
	hostSocketMu.Unlock()
	return val
}
