package main

import (
	"context"
	"fmt"
	"io"
	"net"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"

	dockerctr "github.com/docker/docker/api/types/container"
	dockerimage "github.com/docker/docker/api/types/image"
	dockernet "github.com/docker/docker/api/types/network"
	dockernat "github.com/docker/go-connections/nat"

	"core/podman"
)

var (
	// browserRegistry is the global instance of the volume browser manager.
	browserRegistry = newVolumeBrowserRegistry()

	invalidNameChars = regexp.MustCompile(`[^a-zA-Z0-9_.-]`)
)

// browser tracks an ephemeral dufs container.
type browser struct {
	containerID string
	port        int
	expireAt    int64 // unix timestamp, 0 = no expiry
}

// volumeBrowserRegistry manages ephemeral dufs browser containers.
type volumeBrowserRegistry struct {
	mu       sync.Mutex
	browsers map[string]*browser
	reserved map[int]bool
}

func newVolumeBrowserRegistry() *volumeBrowserRegistry {
	r := &volumeBrowserRegistry{
		browsers: map[string]*browser{},
		reserved: map[int]bool{},
	}
	// Clean up any stale or orphan browser containers from previous runs or crashes on startup
	go r.cleanupOrphans()

	// Cleanup expired browsers every minute
	go func() {
		ticker := time.NewTicker(time.Minute)
		for range ticker.C {
			r.cleanupExpired()
		}
	}()
	return r
}

func browserContainerName(volumeName string) string {
	clean := invalidNameChars.ReplaceAllString(volumeName, "-")
	return fmt.Sprintf("y-fs-%s", clean)
}

func isBrowserContainer(names []string, labels map[string]string) bool {
	if labels != nil && labels["yantr.system"] == "browser" {
		return true
	}
	for _, n := range names {
		clean := strings.TrimPrefix(n, "/")
		if strings.HasPrefix(clean, "y-fs-") || strings.HasPrefix(clean, "yantr-browse-") {
			return true
		}
	}
	return false
}

func (r *volumeBrowserRegistry) cleanupOrphans() {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	stopTimeout := 1
	ctrs, err := podman.ContainerList(ctx, dockerctr.ListOptions{All: true})
	if err != nil {
		return
	}
	for _, c := range ctrs {
		if isBrowserContainer(c.Names, c.Labels) {
			_ = podman.ContainerStop(ctx, c.ID, dockerctr.StopOptions{Timeout: &stopTimeout})
			_ = podman.ContainerRemove(ctx, c.ID, dockerctr.RemoveOptions{Force: true})
		}
	}
}

const defaultBrowserImage = "docker.io/sigoden/dufs:latest"

func resolveBrowserImage() string {
	if img := os.Getenv("YANTR_BROWSER_IMAGE"); img != "" {
		return img
	}
	return defaultBrowserImage
}

func ensureBrowserImage(ctx context.Context, imageName string) error {
	// If image already exists locally, nothing to do
	if _, _, err := podman.ImageInspectWithRaw(ctx, imageName); err == nil {
		return nil
	}
	// Pull the minimal browser image on demand
	reader, err := podman.ImagePull(ctx, imageName, dockerimage.PullOptions{})
	if err != nil {
		return fmt.Errorf("failed to pull %q: %w", imageName, err)
	}
	defer reader.Close()
	_, _ = io.Copy(io.Discard, reader)
	return nil
}

func (r *volumeBrowserRegistry) findFreePort() (int, error) {
	for attempt := 0; attempt < 10; attempt++ {
		l, err := net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			return 0, err
		}
		p := l.Addr().(*net.TCPAddr).Port
		l.Close()

		r.mu.Lock()
		taken := r.reserved[p]
		if !taken {
			r.reserved[p] = true
		}
		r.mu.Unlock()
		if !taken {
			return p, nil
		}
	}
	return 0, fmt.Errorf("could not find a free port")
}

// Start spawns an ephemeral dufs browser container for a volume.
func (r *volumeBrowserRegistry) Start(volumeName string, expiryMinutes int) (int, error) {
	if expiryMinutes <= 0 {
		expiryMinutes = 30 // default to 30 min auto-expiry to avoid permanent container sprawl
	}

	r.mu.Lock()
	// If this volume is already active, refresh expiry and reuse port
	if b, ok := r.browsers[volumeName]; ok {
		b.expireAt = time.Now().Unix() + int64(expiryMinutes*60)
		p := b.port
		r.mu.Unlock()
		return p, nil
	}

	// Single-active-browser policy: stop any other active browser container to eliminate spam
	var toStop []string
	for name := range r.browsers {
		toStop = append(toStop, name)
	}
	r.mu.Unlock()

	for _, name := range toStop {
		r.Stop(name)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Verify volume exists
	if _, err := podman.VolumeInspect(ctx, volumeName); err != nil {
		return 0, fmt.Errorf("failed to inspect volume %q: %w", volumeName, err)
	}

	p, err := r.findFreePort()
	if err != nil {
		return 0, err
	}

	imageName := resolveBrowserImage()
	if err := ensureBrowserImage(ctx, imageName); err != nil {
		r.mu.Lock()
		delete(r.reserved, p)
		r.mu.Unlock()
		return 0, fmt.Errorf("failed to ensure browser image %q: %w", imageName, err)
	}

	containerName := browserContainerName(volumeName)

	// Clean up any stale container with the same name before creating
	_ = podman.ContainerRemove(ctx, containerName, dockerctr.RemoveOptions{Force: true})

	stopTimeout := 1
	config := &dockerctr.Config{
		Image: imageName,
		Cmd: []string{
			"/data",
			"--port", "5000",
			"--allow-all",
			"--path-prefix", "/browse/" + volumeName,
		},
		ExposedPorts: dockernat.PortSet{
			"5000/tcp": struct{}{},
		},
		Labels: map[string]string{
			"yantr.system": "browser",
			"yantr.volume": volumeName,
		},
		StopTimeout: &stopTimeout,
	}

	hostConfig := &dockerctr.HostConfig{
		Binds: []string{
			volumeName + ":/data:z",
		},
		PortBindings: dockernat.PortMap{
			"5000/tcp": []dockernat.PortBinding{
				{
					HostIP:   "127.0.0.1",
					HostPort: fmt.Sprintf("%d", p),
				},
			},
		},
		RestartPolicy: dockerctr.RestartPolicy{
			Name: "no",
		},
	}

	created, err := podman.ContainerCreate(ctx, config, hostConfig, &dockernet.NetworkingConfig{}, nil, containerName)
	if err != nil {
		r.mu.Lock()
		delete(r.reserved, p)
		r.mu.Unlock()
		return 0, fmt.Errorf("failed to create browser container for %q: %w", volumeName, err)
	}

	if err := podman.ContainerStart(ctx, created.ID, dockerctr.StartOptions{}); err != nil {
		r.mu.Lock()
		delete(r.reserved, p)
		r.mu.Unlock()
		_ = podman.ContainerRemove(ctx, created.ID, dockerctr.RemoveOptions{Force: true})
		return 0, fmt.Errorf("failed to start browser container for %q: %w", volumeName, err)
	}

	expireAt := time.Now().Unix() + int64(expiryMinutes*60)

	b := &browser{containerID: created.ID, port: p, expireAt: expireAt}

	r.mu.Lock()
	r.browsers[volumeName] = b
	r.mu.Unlock()

	return p, nil
}

// Stop stops and removes the ephemeral browser container for a volume.
func (r *volumeBrowserRegistry) Stop(volumeName string) bool {
	r.mu.Lock()
	b, ok := r.browsers[volumeName]
	if !ok {
		r.mu.Unlock()
		// Also clean up any lingering container with this name
		go func() {
			containerName := browserContainerName(volumeName)
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			_ = podman.ContainerRemove(ctx, containerName, dockerctr.RemoveOptions{Force: true})
		}()
		return false
	}
	delete(r.browsers, volumeName)
	delete(r.reserved, b.port)
	r.mu.Unlock()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	stopTimeout := 1
	_ = podman.ContainerStop(ctx, b.containerID, dockerctr.StopOptions{Timeout: &stopTimeout})
	_ = podman.ContainerRemove(ctx, b.containerID, dockerctr.RemoveOptions{Force: true})
	return true
}

// IsBrowsing reports whether a browser container is active for the volume.
func (r *volumeBrowserRegistry) IsBrowsing(volumeName string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	_, ok := r.browsers[volumeName]
	return ok
}

// GetPort returns the port for a volume browser, or 0 if not active.
func (r *volumeBrowserRegistry) GetPort(volumeName string) int {
	r.mu.Lock()
	defer r.mu.Unlock()
	if b, ok := r.browsers[volumeName]; ok {
		return b.port
	}
	return 0
}

type browserInfo struct {
	VolumeName string `json:"volumeName"`
	Port       int    `json:"port"`
	ExpireAt   int64  `json:"expireAt"`
}

// List returns all active browsers.
func (r *volumeBrowserRegistry) List() []browserInfo {
	r.mu.Lock()
	defer r.mu.Unlock()
	result := make([]browserInfo, 0, len(r.browsers))
	for name, b := range r.browsers {
		result = append(result, browserInfo{VolumeName: name, Port: b.port, ExpireAt: b.expireAt})
	}
	return result
}

// StopAll stops and removes all active browser containers.
func (r *volumeBrowserRegistry) StopAll() {
	r.mu.Lock()
	browsers := r.browsers
	r.browsers = map[string]*browser{}
	r.reserved = map[int]bool{}
	r.mu.Unlock()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	stopTimeout := 1
	for _, b := range browsers {
		_ = podman.ContainerStop(ctx, b.containerID, dockerctr.StopOptions{Timeout: &stopTimeout})
		_ = podman.ContainerRemove(ctx, b.containerID, dockerctr.RemoveOptions{Force: true})
	}

	// Clean up any remaining orphan containers
	r.cleanupOrphans()
}

func (r *volumeBrowserRegistry) cleanupExpired() {
	now := time.Now().Unix()
	var toStop []string

	r.mu.Lock()
	for name, b := range r.browsers {
		if b.expireAt > 0 && now >= b.expireAt {
			toStop = append(toStop, name)
		}
	}
	r.mu.Unlock()

	for _, name := range toStop {
		r.Stop(name)
	}
}
