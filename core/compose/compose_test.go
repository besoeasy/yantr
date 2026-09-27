package compose

import "testing"

func TestNormalizeImageRef(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		// Docker Hub shorthand — the form most catalog apps use.
		{"alpine", "docker.io/library/alpine:latest"},
		{"alpine:3.19", "docker.io/library/alpine:3.19"},
		{"postgres:16-alpine", "docker.io/library/postgres:16-alpine"},
		{"mariadb:11", "docker.io/library/mariadb:11"},
		{"docker.io/library/alpine:latest", "docker.io/library/alpine:latest"},

		// Explicit registry with an implicit library namespace. bitmagnet's
		// compose.yml uses this form for postgres.
		{"docker.io/postgres:16-alpine", "docker.io/library/postgres:16-alpine"},
		{"index.docker.io/redis:7", "docker.io/library/redis:7"},

		// Registries with a dot in the host are already fully qualified.
		{"ghcr.io/bitmagnet-io/bitmagnet:latest", "ghcr.io/bitmagnet-io/bitmagnet:latest"},
		{"quay.io/podman/hello:latest", "quay.io/podman/hello:latest"},

		// Non-Docker registries with a bare multi-segment path get the
		// docker.io prefix, matching how Podman reports them.
		{"bitnami/nginx:1.27", "docker.io/bitnami/nginx:1.27"},
		{"linuxserver/sonarr:latest", "docker.io/linuxserver/sonarr:latest"},

		// A registry port is not a tag.
		{"registry.example.com:5000/team/app:v1", "registry.example.com:5000/team/app:v1"},

		// Digest pins resolve to their tag so they match the same image.
		{"alpine@sha256:abc123", "docker.io/library/alpine:latest"},
		{"ghcr.io/foo/bar:1.0@sha256:deadbeef", "ghcr.io/foo/bar:1.0"},

		{"", ""},
		{"   ", ""},
	}
	for _, c := range cases {
		if got := NormalizeImageRef(c.in); got != c.want {
			t.Errorf("NormalizeImageRef(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestServiceImages(t *testing.T) {
	doc, err := Parse(`
x-yantr:
  name: test
services:
  app:
    image: ghcr.io/example/app:latest
  db:
    image: postgres:16-alpine
  cache:
    image: redis:7
  app2:
    image: ghcr.io/example/app:latest
  worker:
    command: ["true"]
  blank:
    image: "   "
`)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	got := ServiceImages(doc)
	// Sorted and de-duplicated by normalized ref; the service with no image
	// key and the blank one are both skipped.
	want := []string{"ghcr.io/example/app:latest", "postgres:16-alpine", "redis:7"}
	if len(got) != len(want) {
		t.Fatalf("ServiceImages = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("ServiceImages = %v, want %v", got, want)
		}
	}
}

func TestServiceImagesNoServices(t *testing.T) {
	doc, err := Parse("x-yantr:\n  name: empty\n")
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if got := ServiceImages(doc); len(got) != 0 {
		t.Fatalf("expected no images, got %v", got)
	}
}

// TestNormalizeMatchesPodmanRepoTags pins the normalization against the exact
// RepoTags strings Podman emits, so a mismatch here surfaces as a failing test
// rather than a silently missed update.
func TestNormalizeMatchesPodmanRepoTags(t *testing.T) {
	// Captured from `podman images` / the ImageList API on a rootless host.
	podmanRepoTags := []string{
		"docker.io/library/postgres:16-alpine",
		"docker.io/library/redis:7",
		"ghcr.io/bitmagnet-io/bitmagnet:latest",
		"docker.io/bitnami/nginx:1.27",
		"docker.io/library/alpine:latest",
	}
	// The same images as they appear in compose files.
	composeRefs := []string{
		"postgres:16-alpine",
		"redis:7",
		"ghcr.io/bitmagnet-io/bitmagnet:latest",
		"bitnami/nginx:1.27",
		"alpine",
	}
	for i, ref := range composeRefs {
		if got := NormalizeImageRef(ref); got != podmanRepoTags[i] {
			t.Errorf("NormalizeImageRef(%q) = %q, want %q", ref, got, podmanRepoTags[i])
		}
	}
}
