package main

import (
	"context"
	"fmt"
	"net"
	"os/exec"
	"sync"
	"time"

	"core/docker"
)

// browserRegistry is the global instance of the volume browser manager.
var browserRegistry = newVolumeBrowserRegistry()

// browser tracks a running dufs process.
type browser struct {
	process  *exec.Cmd
	port     int
	expireAt int64 // unix timestamp, 0 = no expiry
}

// volumeBrowserRegistry manages dufs browser processes.
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
	// Cleanup expired browsers every minute
	go func() {
		for range time.Tick(time.Minute) {
			r.cleanupExpired()
		}
	}()
	return r
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

// Start spawns a dufs browser for a volume.
func (r *volumeBrowserRegistry) Start(volumeName string, expiryMinutes int) (int, error) {
	r.mu.Lock()
	if b, ok := r.browsers[volumeName]; ok {
		p := b.port
		r.mu.Unlock()
		return p, nil
	}
	r.mu.Unlock()

	// Resolve the real mountpoint via the Docker API instead of hardcoding
	// /var/lib/docker/volumes/<name>/_data, which breaks on non-default data-root.
	vol, err := docker.VolumeInspect(context.Background(), volumeName)
	if err != nil {
		return 0, fmt.Errorf("failed to inspect volume %q: %w", volumeName, err)
	}
	dataPath := vol.Mountpoint
	if dataPath == "" {
		return 0, fmt.Errorf("volume %q has no mountpoint", volumeName)
	}

	p, err := r.findFreePort()
	if err != nil {
		return 0, err
	}

	cmd := exec.Command("dufs", dataPath,
		"--port", fmt.Sprintf("%d", p),
		"--allow-all",
		"--path-prefix", "/browse/"+volumeName,
	)
	if err := cmd.Start(); err != nil {
		r.mu.Lock()
		delete(r.reserved, p)
		r.mu.Unlock()
		return 0, fmt.Errorf("failed to start dufs: %w", err)
	}

	expireAt := int64(0)
	if expiryMinutes > 0 {
		expireAt = time.Now().Unix() + int64(expiryMinutes*60)
	}

	b := &browser{process: cmd, port: p, expireAt: expireAt}

	r.mu.Lock()
	r.browsers[volumeName] = b
	r.mu.Unlock()

	// Watch for process exit
	go func() {
		_ = cmd.Wait()
		r.mu.Lock()
		delete(r.reserved, p)
		delete(r.browsers, volumeName)
		r.mu.Unlock()
	}()

	return p, nil
}

// Stop kills a browser for a volume.
func (r *volumeBrowserRegistry) Stop(volumeName string) bool {
	r.mu.Lock()
	b, ok := r.browsers[volumeName]
	if !ok {
		r.mu.Unlock()
		return false
	}
	delete(r.browsers, volumeName)
	delete(r.reserved, b.port)
	r.mu.Unlock()

	if b.process != nil {
		_ = b.process.Process.Kill()
	}
	return true
}

// IsBrowsing reports whether a browser is active for the volume.
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

// StopAll kills all browser processes.
func (r *volumeBrowserRegistry) StopAll() {
	r.mu.Lock()
	defer r.mu.Unlock()
	for name, b := range r.browsers {
		if b.process != nil {
			_ = b.process.Process.Kill()
		}
		delete(r.browsers, name)
	}
	r.reserved = map[int]bool{}
}

func (r *volumeBrowserRegistry) cleanupExpired() {
	now := time.Now().Unix()
	r.mu.Lock()
	defer r.mu.Unlock()
	for name, b := range r.browsers {
		if b.expireAt > 0 && now >= b.expireAt {
			if b.process != nil {
				_ = b.process.Process.Kill()
			}
			delete(r.reserved, b.port)
			delete(r.browsers, name)
		}
	}
}
