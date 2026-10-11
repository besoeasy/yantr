package podman

import (
	"os"
	"testing"
)

// TestHostSocketPrefersEnv is the regression guard for the install.sh
// contract: HOST_PODMAN_SOCKET is set by the installer and must win, so no
// runtime discovery (and no cached value) is involved.
func TestHostSocketPrefersEnv(t *testing.T) {
	t.Setenv("HOST_PODMAN_SOCKET", "/run/user/1000/podman/podman.sock")
	if got := HostSocket(); got != "/run/user/1000/podman/podman.sock" {
		t.Fatalf("HostSocket() = %q, want the HOST_PODMAN_SOCKET value", got)
	}
}

// A value copied straight out of `podman info` arrives URI-prefixed.
func TestHostSocketStripsUnixPrefix(t *testing.T) {
	t.Setenv("HOST_PODMAN_SOCKET", "unix:///run/user/1000/podman/podman.sock")
	if got := HostSocket(); got != "/run/user/1000/podman/podman.sock" {
		t.Fatalf("HostSocket() = %q, want the unix:// prefix stripped", got)
	}
}

// Without the env var there is no fallback discovery — guessing from inside
// the container can only yield the in-container mount point, which does not
// exist on a rootless host. The value is still returned rather than empty so
// callers fail with a clear path, and the omission is reported on stderr.
func TestHostSocketFallsBackToSocketPath(t *testing.T) {
	t.Setenv("HOST_PODMAN_SOCKET", "")
	if got := HostSocket(); got != SocketPath {
		t.Fatalf("HostSocket() = %q, want fallback %q", got, SocketPath)
	}
}

// The value must be re-read on every call. Caching it once (as an earlier
// sync.Once did) is what let a single transient failure pin a wrong path for
// the whole process lifetime.
func TestHostSocketIsNotCached(t *testing.T) {
	t.Setenv("HOST_PODMAN_SOCKET", "/first.sock")
	if got := HostSocket(); got != "/first.sock" {
		t.Fatalf("HostSocket() = %q, want /first.sock", got)
	}
	t.Setenv("HOST_PODMAN_SOCKET", "/second.sock")
	if got := HostSocket(); got != "/second.sock" {
		t.Fatalf("HostSocket() = %q, want the updated /second.sock", got)
	}
}

func TestResolveSocketPathPrefersExistingContainerHost(t *testing.T) {
	// A path that exists on disk must win, even when it is not the default.
	dir := t.TempDir()
	sock := dir + "/podman.sock"
	f, err := os.Create(sock)
	if err != nil {
		t.Fatalf("create temp socket: %v", err)
	}
	_ = f.Close()

	t.Setenv("CONTAINER_HOST", "unix://"+sock)
	if got := resolveSocketPath(); got != sock {
		t.Fatalf("resolveSocketPath() = %q, want %q", got, sock)
	}
}

// A CONTAINER_HOST that does not exist is ignored rather than adopted,
// otherwise a stale value would point the client at a dead socket.
func TestResolveSocketPathIgnoresMissingContainerHost(t *testing.T) {
	t.Setenv("CONTAINER_HOST", "unix:///nonexistent/yantr-test.sock")
	t.Setenv("PODMAN_SOCKET", "")
	t.Setenv("DOCKER_SOCKET", "")
	if got := resolveSocketPath(); got != "/run/podman/podman.sock" {
		t.Fatalf("resolveSocketPath() = %q, want the default mount point", got)
	}
}
