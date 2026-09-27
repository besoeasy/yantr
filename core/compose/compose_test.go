package compose

import "testing"

func TestNormalizePortOverridesTyped(t *testing.T) {
	raw := []interface{}{
		map[string]interface{}{"containerPort": float64(8080), "protocol": "tcp", "hostPort": float64(9090)},
		map[string]interface{}{"containerPort": "51820", "protocol": "UDP", "hostPort": "51821"},
		map[string]interface{}{"containerPort": 0, "protocol": "tcp", "hostPort": 1},
		map[string]interface{}{"containerPort": 80, "protocol": "bogus", "hostPort": 8080},
	}
	got := NormalizePortOverrides(raw)
	if len(got) != 2 {
		t.Fatalf("expected 2 overrides, got %+v", got)
	}
	if got[0] != (PortOverride{ContainerPort: 8080, Protocol: "tcp", HostPort: 9090}) {
		t.Fatalf("unexpected first override: %+v", got[0])
	}
	if got[1] != (PortOverride{ContainerPort: 51820, Protocol: "udp", HostPort: 51821}) {
		t.Fatalf("unexpected second override: %+v", got[1])
	}
}

func TestNormalizePortOverridesLegacyMap(t *testing.T) {
	raw := map[string]interface{}{
		"8080/tcp":  float64(9090),
		"51820/UDP": "51821",
		"bad":       1,
		"0/tcp":     1,
	}
	got := NormalizePortOverrides(raw)
	if len(got) != 2 {
		t.Fatalf("expected 2 overrides, got %+v", got)
	}
	byKey := map[string]int{}
	for _, o := range got {
		byKey[o.Protocol] = o.HostPort
		if o.ContainerPort != 8080 && o.ContainerPort != 51820 {
			t.Fatalf("unexpected override: %+v", o)
		}
	}
	if byKey["tcp"] != 9090 || byKey["udp"] != 51821 {
		t.Fatalf("unexpected overrides: %+v", got)
	}
}

func TestApplyCustomPortMappingsTyped(t *testing.T) {
	doc, err := Parse(`
services:
  app:
    image: example:latest
    ports:
      - "8080"
      - "51820/udp"
`)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	err = ApplyTransforms(doc, TransformOptions{
		ProjectID: "test", AppID: "test",
		CustomPortMappings: []PortOverride{
			{ContainerPort: 8080, Protocol: "tcp", HostPort: 9090},
			{ContainerPort: 51820, Protocol: "udp", HostPort: 51920},
		},
		HostDockerSocket: "/run/podman.sock",
	})
	if err != nil {
		t.Fatalf("ApplyTransforms: %v", err)
	}
	svcs := getServices(doc)
	svc := svcs["app"].(map[string]interface{})
	ports := svc["ports"].([]interface{})
	var strs []string
	for _, p := range ports {
		strs = append(strs, p.(string))
	}
	want := map[string]bool{"9090:8080": true, "51920:51820/udp": true}
	if len(strs) != 2 {
		t.Fatalf("expected 2 ports, got %v", strs)
	}
	for _, s := range strs {
		if !want[s] {
			t.Fatalf("unexpected port %q in %v", s, strs)
		}
	}
}

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
