package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	dockertypes "github.com/docker/docker/api/types"
	dockerctr "github.com/docker/docker/api/types/container"
	dockerimage "github.com/docker/docker/api/types/image"
	dockernet "github.com/docker/docker/api/types/network"
	dockernat "github.com/docker/go-connections/nat"

	"core/podman"
	"core/shared"
)

var (
	browserRegistry = newVolumeBrowserRegistry()

	invalidNameChars = regexp.MustCompile(`[^a-zA-Z0-9_.-]`)
)

// defaultBrowserExpiryMinutes is the auto-expiry applied to new browsers and to
// browsers re-adopted after a core restart.
const defaultBrowserExpiryMinutes = 30

// browserReadyTimeout bounds how long Start waits for dufs to accept
// connections before giving up and tearing the container down.
const browserReadyTimeout = 10 * time.Second

// browserMaintainInterval drives the background ticker that expires old
// browsers and evicts entries whose container died out from under us.
const browserMaintainInterval = 30 * time.Second

type browser struct {
	containerID string
	port        int
	expireAt    int64 // unix timestamp, 0 = no expiry
}

type volumeBrowserRegistry struct {
	mu       sync.Mutex
	startMu  sync.Mutex
	browsers map[string]*browser
}

func newVolumeBrowserRegistry() *volumeBrowserRegistry {
	r := &volumeBrowserRegistry{
		browsers: map[string]*browser{},
	}

	// Re-adopt running browser containers (e.g. after a core restart) instead
	// of wiping them: the label + port binding carry everything needed.
	go r.restoreBrowsers()

	go func() {
		ticker := time.NewTicker(browserMaintainInterval)
		for range ticker.C {
			r.cleanupExpired()
			r.reconcile()
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

// restoreBrowsers re-adopts running browser containers found at boot so a core
// restart doesn't orphan live sessions. Stopped or unparseable browser
// containers are removed.
func (r *volumeBrowserRegistry) restoreBrowsers() {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	ctrs, err := podman.ContainerList(ctx, dockerctr.ListOptions{All: true})
	if err != nil {
		return
	}
	for _, c := range ctrs {
		if !isBrowserContainer(c.Names, c.Labels) {
			continue
		}
		info, err := podman.ContainerInspect(ctx, c.ID)
		if err != nil {
			continue
		}
		volumeName := ""
		if info.Config != nil && info.Config.Labels != nil {
			volumeName = info.Config.Labels["yantr.volume"]
		}
		if volumeName == "" && c.Labels != nil {
			volumeName = c.Labels["yantr.volume"]
		}
		if volumeName == "" {
			// No volume identity — cannot proxy it, drop it.
			_ = podman.ContainerRemove(ctx, c.ID, dockerctr.RemoveOptions{Force: true})
			continue
		}
		if info.State == nil || !info.State.Running {
			_ = podman.ContainerRemove(ctx, c.ID, dockerctr.RemoveOptions{Force: true})
			continue
		}
		port, ok := browserHostPort(info)
		if !ok {
			_ = podman.ContainerRemove(ctx, c.ID, dockerctr.RemoveOptions{Force: true})
			continue
		}
		r.mu.Lock()
		r.browsers[volumeName] = &browser{
			containerID: c.ID,
			port:        port,
			expireAt:    time.Now().Unix() + int64(defaultBrowserExpiryMinutes*60),
		}
		r.mu.Unlock()
		shared.Log("info", fmt.Sprintf("[browser] re-adopted browser for volume %q on port %d", volumeName, port))
	}
}

// sweepBrowserContainers removes every browser container, registered or not.
// Used at shutdown after the registry has been drained.
func (r *volumeBrowserRegistry) sweepBrowserContainers(ctx context.Context) {
	ctrs, err := podman.ContainerList(ctx, dockerctr.ListOptions{All: true})
	if err != nil {
		return
	}
	stopTimeout := 1
	for _, c := range ctrs {
		if isBrowserContainer(c.Names, c.Labels) {
			_ = podman.ContainerStop(ctx, c.ID, dockerctr.StopOptions{Timeout: &stopTimeout})
			_ = podman.ContainerRemove(ctx, c.ID, dockerctr.RemoveOptions{Force: true})
		}
	}
}

// browserHostPort extracts the Podman-assigned host port for the dufs
// container port (5000/tcp) from an inspect result.
func browserHostPort(info dockertypes.ContainerJSON) (int, bool) {
	portKey := dockernat.Port("5000/tcp")
	if info.NetworkSettings != nil {
		if bindings, ok := info.NetworkSettings.Ports[portKey]; ok && len(bindings) > 0 {
			if p, err := strconv.Atoi(bindings[0].HostPort); err == nil && p > 0 {
				return p, true
			}
		}
	}
	if info.HostConfig != nil {
		if bindings, ok := info.HostConfig.PortBindings[portKey]; ok && len(bindings) > 0 {
			if p, err := strconv.Atoi(bindings[0].HostPort); err == nil && p > 0 {
				return p, true
			}
		}
	}
	return 0, false
}

// waitForBrowserReady polls dufs until it accepts HTTP connections. Any HTTP
// response (even an error status) proves the server is up; connection failures
// mean it isn't ready yet.
func waitForBrowserReady(port int, volumeName string, timeout time.Duration) error {
	client := &http.Client{Timeout: 2 * time.Second}
	url := fmt.Sprintf("http://127.0.0.1:%d/browse/%s/", port, volumeName)
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		resp, err := client.Get(url)
		if err == nil {
			_, _ = io.Copy(io.Discard, resp.Body)
			_ = resp.Body.Close()
			return nil
		}
		time.Sleep(150 * time.Millisecond)
	}
	return fmt.Errorf("browser for %q did not become ready within %s", volumeName, timeout)
}

const defaultBrowserImage = "docker.io/sigoden/dufs:latest"

func resolveBrowserImage() string {
	if img := os.Getenv("YANTR_BROWSER_IMAGE"); img != "" {
		return img
	}
	return defaultBrowserImage
}

func ensureBrowserImage(ctx context.Context, imageName string) error {
	if _, _, err := podman.ImageInspectWithRaw(ctx, imageName); err == nil {
		return nil
	}
	reader, err := podman.ImagePull(ctx, imageName, dockerimage.PullOptions{})
	if err != nil {
		return fmt.Errorf("failed to pull %q: %w", imageName, err)
	}
	defer reader.Close()
	_, _ = io.Copy(io.Discard, reader)
	return nil
}

func (r *volumeBrowserRegistry) Start(volumeName string, expiryMinutes int) (int, error) {
	if expiryMinutes <= 0 {
		expiryMinutes = defaultBrowserExpiryMinutes // default to 30 min auto-expiry to avoid permanent container sprawl
	}

	// Serialize starts so concurrent requests can't race port allocation,
	// container creation, or the single-active-browser eviction below.
	r.startMu.Lock()
	defer r.startMu.Unlock()

	r.mu.Lock()
	if b, ok := r.browsers[volumeName]; ok {
		r.mu.Unlock()
		// Self-heal the fast path: if the tracked container died, drop the
		// stale entry and fall through to recreate it.
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		info, err := podman.ContainerInspect(ctx, b.containerID)
		cancel()
		if err == nil && info.State != nil && info.State.Running {
			r.mu.Lock()
			if cur, ok := r.browsers[volumeName]; ok && cur.containerID == b.containerID {
				cur.expireAt = time.Now().Unix() + int64(expiryMinutes*60)
				p := cur.port
				r.mu.Unlock()
				return p, nil
			}
			r.mu.Unlock()
		} else {
			r.mu.Lock()
			if cur, ok := r.browsers[volumeName]; ok && cur.containerID == b.containerID {
				delete(r.browsers, volumeName)
			}
			r.mu.Unlock()
			go func(id string) {
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()
				_ = podman.ContainerRemove(ctx, id, dockerctr.RemoveOptions{Force: true})
			}(b.containerID)
		}
	} else {
		r.mu.Unlock()
	}

	// Single-active-browser policy: stop any other active browser container to eliminate spam
	r.mu.Lock()
	var toStop []string
	for name := range r.browsers {
		toStop = append(toStop, name)
	}
	r.mu.Unlock()

	for _, name := range toStop {
		r.Stop(name)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	if _, err := podman.VolumeInspect(ctx, volumeName); err != nil {
		return 0, fmt.Errorf("failed to inspect volume %q: %w", volumeName, err)
	}

	imageName := resolveBrowserImage()
	if err := ensureBrowserImage(ctx, imageName); err != nil {
		return 0, fmt.Errorf("failed to ensure browser image %q: %w", imageName, err)
	}

	containerName := browserContainerName(volumeName)

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
			"5000/tcp": {},
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
			// Empty HostPort: Podman auto-allocates a free host port,
			// eliminating the check-then-bind race of pre-picking one.
			"5000/tcp": []dockernat.PortBinding{
				{
					HostIP: "127.0.0.1",
				},
			},
		},
		RestartPolicy: dockerctr.RestartPolicy{
			Name: "no",
		},
	}

	cleanup := func(id string) {
		_ = podman.ContainerRemove(ctx, id, dockerctr.RemoveOptions{Force: true})
	}

	created, err := podman.ContainerCreate(ctx, config, hostConfig, &dockernet.NetworkingConfig{}, nil, containerName)
	if err != nil {
		return 0, fmt.Errorf("failed to create browser container for %q: %w", volumeName, err)
	}

	if err := podman.ContainerStart(ctx, created.ID, dockerctr.StartOptions{}); err != nil {
		cleanup(created.ID)
		return 0, fmt.Errorf("failed to start browser container for %q: %w", volumeName, err)
	}

	info, err := podman.ContainerInspect(ctx, created.ID)
	if err != nil {
		cleanup(created.ID)
		return 0, fmt.Errorf("failed to inspect browser container for %q: %w", volumeName, err)
	}
	p, ok := browserHostPort(info)
	if !ok {
		cleanup(created.ID)
		return 0, fmt.Errorf("browser container for %q has no assigned host port", volumeName)
	}

	if err := waitForBrowserReady(p, volumeName, browserReadyTimeout); err != nil {
		stopTimeout := 1
		_ = podman.ContainerStop(ctx, created.ID, dockerctr.StopOptions{Timeout: &stopTimeout})
		cleanup(created.ID)
		return 0, err
	}

	expireAt := time.Now().Unix() + int64(expiryMinutes*60)

	b := &browser{containerID: created.ID, port: p, expireAt: expireAt}

	r.mu.Lock()
	r.browsers[volumeName] = b
	r.mu.Unlock()

	return p, nil
}

func (r *volumeBrowserRegistry) Stop(volumeName string) bool {
	r.mu.Lock()
	b, ok := r.browsers[volumeName]
	if !ok {
		r.mu.Unlock()
		go func() {
			containerName := browserContainerName(volumeName)
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			_ = podman.ContainerRemove(ctx, containerName, dockerctr.RemoveOptions{Force: true})
		}()
		return false
	}
	delete(r.browsers, volumeName)
	r.mu.Unlock()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	stopTimeout := 1
	_ = podman.ContainerStop(ctx, b.containerID, dockerctr.StopOptions{Timeout: &stopTimeout})
	_ = podman.ContainerRemove(ctx, b.containerID, dockerctr.RemoveOptions{Force: true})
	return true
}

func (r *volumeBrowserRegistry) IsBrowsing(volumeName string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	_, ok := r.browsers[volumeName]
	return ok
}

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

func (r *volumeBrowserRegistry) List() []browserInfo {
	r.mu.Lock()
	defer r.mu.Unlock()
	result := make([]browserInfo, 0, len(r.browsers))
	for name, b := range r.browsers {
		result = append(result, browserInfo{VolumeName: name, Port: b.port, ExpireAt: b.expireAt})
	}
	return result
}

func (r *volumeBrowserRegistry) StopAll() {
	r.mu.Lock()
	browsers := r.browsers
	r.browsers = map[string]*browser{}
	r.mu.Unlock()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	stopTimeout := 1
	for _, b := range browsers {
		_ = podman.ContainerStop(ctx, b.containerID, dockerctr.StopOptions{Timeout: &stopTimeout})
		_ = podman.ContainerRemove(ctx, b.containerID, dockerctr.RemoveOptions{Force: true})
	}

	r.sweepBrowserContainers(ctx)
}

// reconcile evicts registry entries whose container died or vanished outside
// the registry (OOM, manual `podman rm`, daemon restart), so IsBrowsing and
// the proxy stop reporting a browser that can never serve.
func (r *volumeBrowserRegistry) reconcile() {
	r.mu.Lock()
	snapshot := make(map[string]*browser, len(r.browsers))
	for name, b := range r.browsers {
		snapshot[name] = b
	}
	r.mu.Unlock()

	if len(snapshot) == 0 {
		return
	}

	for name, b := range snapshot {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		info, err := podman.ContainerInspect(ctx, b.containerID)
		cancel()
		alive := err == nil && info.State != nil && info.State.Running
		if alive {
			continue
		}
		r.mu.Lock()
		if cur, ok := r.browsers[name]; ok && cur.containerID == b.containerID {
			delete(r.browsers, name)
		}
		r.mu.Unlock()
		shared.Log("info", fmt.Sprintf("[browser] evicted dead browser for volume %q", name))
		go func(id string) {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			_ = podman.ContainerRemove(ctx, id, dockerctr.RemoveOptions{Force: true})
		}(b.containerID)
	}
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
