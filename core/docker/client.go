// Package docker provides a shared Docker client and helper utilities.
package docker

import (
	"context"
	"net"
	"net/http"
	"os"
	"time"

	dockerclient "github.com/docker/docker/client"
)

// Client is the shared Docker API client instance.
var Client *dockerclient.Client

// SocketPath is the Docker socket path used by the client.
var SocketPath string

func init() {
	SocketPath = os.Getenv("DOCKER_SOCKET")
	if SocketPath == "" {
		SocketPath = "/var/run/docker.sock"
	}

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
		panic("failed to create Docker client: " + err.Error())
	}
}

// Background returns a context suitable for Docker API calls.
func Background() context.Context {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	_ = cancel // callers should use this pattern; for long-running calls use context.Background()
	return ctx
}

// Ctx returns a plain background context for operations that might take a while.
func Ctx() context.Context {
	return context.Background()
}
