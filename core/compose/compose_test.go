package compose

import (
	"os"
	"path/filepath"
	"strconv"
	"testing"
)

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

// ─── Project compose filename convention ─────────────────────────────────────
// The glob and the writer used to live apart, the glob omitted the leading dot,
// and stack auto-discovery silently matched nothing while its test passed. These
// two tests pin the convention from both ends.

func TestProjectComposeFileName(t *testing.T) {
	cases := []struct{ projectID, want string }{
		{"jellyfin", ".compose.jellyfin.yml"},
		{"immich-2", ".compose.immich-2.yml"},
		{"", ".compose..yml"},
	}
	for _, tc := range cases {
		if got := ProjectComposeFileName(tc.projectID); got != tc.want {
			t.Errorf("ProjectComposeFileName(%q) = %q, want %q", tc.projectID, got, tc.want)
		}
	}
}

// TestProjectComposeGlobPatternMatchesWriter guards the exact regression: the
// glob pattern must match what ProjectComposeFileName produces, including the
// hidden-file leading dot that a hand-written pattern tends to drop.
func TestProjectComposeGlobPatternMatchesWriter(t *testing.T) {
	for _, projectID := range []string{"jellyfin", "immich-2", "vaultwarden-17"} {
		produced := ProjectComposeFileName(projectID)
		// A real match, not a string-suffix guess: the historical bug was the
		// pattern omitting the leading dot, which no amount of prefix reasoning
		// would have caught.
		ok, err := filepath.Match(ProjectComposeGlobPattern, produced)
		if err != nil {
			t.Fatalf("bad glob pattern %q: %v", ProjectComposeGlobPattern, err)
		}
		if !ok {
			t.Errorf("glob pattern %q does not match produced filename %q", ProjectComposeGlobPattern, produced)
		}
		// And the inverse must recover the ID, including one containing a dot.
		if got := ProjectIDFromComposeFileName(produced); got != projectID {
			t.Errorf("ProjectIDFromComposeFileName(%q) = %q, want %q", produced, got, projectID)
		}
	}
	if got := ProjectIDFromComposeFileName(ProjectComposeFileName("weird.id")); got != "weird.id" {
		t.Errorf("project ID containing a dot did not round-trip: got %q", got)
	}
}

// TestProjectComposeGlobMatchesOnDisk proves the pattern resolves against the
// filesystem the way the supervisor uses it.
func TestProjectComposeGlobMatchesOnDisk(t *testing.T) {
	dir := t.TempDir()
	appDir := filepath.Join(dir, "jellyfin")
	if err := os.MkdirAll(appDir, 0755); err != nil {
		t.Fatal(err)
	}
	// The real per-project file, plus a decoy production never writes.
	for _, name := range []string{ProjectComposeFileName("jellyfin"), "compose.jellyfin.yml"} {
		if err := os.WriteFile(filepath.Join(appDir, name), []byte("services: {}"), 0644); err != nil {
			t.Fatal(err)
		}
	}

	matches, err := filepath.Glob(filepath.Join(dir, "*", ProjectComposeGlobPattern))
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != 1 {
		t.Fatalf("expected exactly 1 match, got %v", matches)
	}
	if got := ProjectIDFromComposeFileName(filepath.Base(matches[0])); got != "jellyfin" {
		t.Fatalf("recovered project ID %q, want %q", got, "jellyfin")
	}
}

// ─── Docker socket policy ────────────────────────────────────────────────────
// applyDockerSocketTransform is the security boundary that lets Docker-compatible
// apps reach the engine without host-root access: the ONLY legal host-side
// socket source is the ${HOST_PODMAN_SOCKET} placeholder. A regression here
// silently widens that boundary, so both the accepting and the rejecting paths
// are pinned here.

func TestApplyDockerSocketTransformResolvesPlaceholder(t *testing.T) {
	doc, err := Parse(`
services:
  glances:
    image: example/glances:latest
    volumes:
      - ${HOST_PODMAN_SOCKET}:/var/run/docker.sock:ro
      - data:/data
`)
	if err != nil {
		t.Fatal(err)
	}
	if err := applyDockerSocketTransform(getServices(doc), "/run/user/1000/podman/podman.sock"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	vols := getServices(doc)["glances"].(map[string]interface{})["volumes"].([]interface{})
	want := "/run/user/1000/podman/podman.sock:/var/run/docker.sock:ro"
	if vols[0] != want {
		t.Errorf("socket source = %q, want %q", vols[0], want)
	}
	if vols[1] != "data:/data" {
		t.Errorf("non-socket volume was rewritten: %q", vols[1])
	}
}

func TestApplyDockerSocketTransformResolvesShortPlaceholder(t *testing.T) {
	doc, err := Parse(`
services:
  portainer:
    image: example/portainer:latest
    volumes:
      - $HOST_PODMAN_SOCKET:/var/run/docker.sock
`)
	if err != nil {
		t.Fatal(err)
	}
	if err := applyDockerSocketTransform(getServices(doc), "/run/podman/podman.sock"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	vols := getServices(doc)["portainer"].(map[string]interface{})["volumes"].([]interface{})
	want := "/run/podman/podman.sock:/var/run/docker.sock"
	if vols[0] != want {
		t.Errorf("socket source = %q, want %q", vols[0], want)
	}
}

func TestApplyDockerSocketTransformLongForm(t *testing.T) {
	doc, err := Parse(`
services:
  app:
    image: example/app:latest
    volumes:
      - type: bind
        source: ${HOST_PODMAN_SOCKET}
        target: /var/run/docker.sock
`)
	if err != nil {
		t.Fatal(err)
	}
	if err := applyDockerSocketTransform(getServices(doc), "/run/user/1000/podman/podman.sock"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	entry := getServices(doc)["app"].(map[string]interface{})["volumes"].([]interface{})[0].(map[string]interface{})
	if entry["source"] != "/run/user/1000/podman/podman.sock" {
		t.Errorf("long-form source = %v, want resolved host socket", entry["source"])
	}
}

// TestApplyDockerSocketTransformRejectsOtherSockets is the security-critical
// case: any host-side socket path other than the placeholder must abort.
func TestApplyDockerSocketTransformRejectsOtherSockets(t *testing.T) {
	forbidden := []string{
		"/var/run/docker.sock",
		"/run/docker.sock",
		"/run/podman/podman.sock",
		"/run/user/1000/podman/podman.sock",
		"/host/var/run/docker.sock",
		"/var/run/containerd.sock",
		"./local.sock",
		"podman.sock",
	}
	for _, src := range forbidden {
		docText := "services:\n  app:\n    image: example/app:latest\n    volumes:\n      - " + src + ":/var/run/docker.sock:ro\n"
		doc, err := Parse(docText)
		if err != nil {
			t.Fatalf("%s: Parse: %v", src, err)
		}
		if err := applyDockerSocketTransform(getServices(doc), "/run/user/1000/podman/podman.sock"); err == nil {
			t.Errorf("host source %q was accepted, want rejection", src)
		}
	}
}

// The guard must not become a blanket "reject any absolute bind" rule: ordinary
// non-socket binds are still the app author's business.
func TestApplyDockerSocketTransformAllowsNonSocketBinds(t *testing.T) {
	allowed := []string{
		"./config:/config",
		"/srv/media:/media",
		"../shared:/shared",
		"named-volume:/data",
		"/etc/localtime:/etc/localtime:ro",
		"/dev/dri:/dev/dri",
	}
	for _, entry := range allowed {
		docText := "services:\n  app:\n    image: example/app:latest\n    volumes:\n      - \"" + entry + "\"\n"
		doc, err := Parse(docText)
		if err != nil {
			t.Fatalf("%s: Parse: %v", entry, err)
		}
		if err := applyDockerSocketTransform(getServices(doc), "/run/user/1000/podman/podman.sock"); err != nil {
			t.Errorf("non-socket bind %q was rejected: %v", entry, err)
		}
	}
}

func TestApplyDockerSocketTransformRejectsLongFormOtherSocket(t *testing.T) {
	doc, err := Parse(`
services:
  app:
    image: example/app:latest
    volumes:
      - type: bind
        source: /var/run/docker.sock
        target: /var/run/docker.sock
`)
	if err != nil {
		t.Fatal(err)
	}
	if err := applyDockerSocketTransform(getServices(doc), "/run/user/1000/podman/podman.sock"); err == nil {
		t.Error("long-form /var/run/docker.sock was accepted, want rejection")
	}
}

// A placeholder that cannot be resolved must fail the deploy rather than mount
// the literal string as a path.
func TestApplyDockerSocketTransformRejectsUnresolvablePlaceholder(t *testing.T) {
	doc, err := Parse(`
services:
  app:
    image: example/app:latest
    volumes:
      - ${HOST_PODMAN_SOCKET}:/var/run/docker.sock
`)
	if err != nil {
		t.Fatal(err)
	}
	if err := applyDockerSocketTransform(getServices(doc), ""); err == nil {
		t.Error("unresolvable placeholder was accepted, want error")
	}
}

// ─── Instance transforms ─────────────────────────────────────────────────────

func TestGetInstanceID(t *testing.T) {
	cases := []struct {
		projectID, appID string
		want             int
	}{
		{"jellyfin", "jellyfin", 0},   // primary instance
		{"jellyfin-1", "jellyfin", 0}, // suffix 1 is the primary
		{"jellyfin-2", "jellyfin", 2},
		{"jellyfin-17", "jellyfin", 17},
		{"immich-3", "jellyfin", 0},   // appID mismatch
		{"other-2", "jellyfin", 0},    // appID mismatch
		{"", "jellyfin", 0},           // no project
		{"jellyfin-2", "", 0},         // no app
		{"jellyfin-0", "jellyfin", 0}, // never a valid instance number
		{"jellyfin-abc", "jellyfin", 0},
		{"my-app-2", "my-app", 2},
	}
	for _, tc := range cases {
		if got := getInstanceID(tc.projectID, tc.appID); got != tc.want {
			t.Errorf("getInstanceID(%q, %q) = %d, want %d", tc.projectID, tc.appID, got, tc.want)
		}
	}
}

func TestApplyInstanceTransforms(t *testing.T) {
	doc, err := Parse(`
services:
  app:
    image: example/app:latest
    container_name: hardcoded-name
    volumes:
      - app-data:/data
      - ./config:/config
      - /host/path:/host
      - ../rel:/rel
      - ${SUBST}:/subst

volumes:
  app-data:
    name: app-data
`)
	if err != nil {
		t.Fatal(err)
	}
	applyInstanceTransforms(doc, getServices(doc), 2)

	svc := getServices(doc)["app"].(map[string]interface{})
	if _, ok := svc["container_name"]; ok {
		t.Error("container_name should be removed so instances do not collide")
	}

	vols := svc["volumes"].([]interface{})
	want := []string{
		"app-data_2:/data", // named volume gets suffixed
		"./config:/config", // relative bind left alone
		"/host/path:/host", // absolute bind left alone
		"../rel:/rel",      // parent-relative bind left alone
		"${SUBST}:/subst",  // interpolated source left alone
	}
	if len(vols) != len(want) {
		t.Fatalf("got %d volumes, want %d: %v", len(vols), len(want), vols)
	}
	for i, w := range want {
		if vols[i] != w {
			t.Errorf("volumes[%d] = %q, want %q", i, vols[i], w)
		}
	}

	topVols := doc["volumes"].(map[string]interface{})
	if _, ok := topVols["app-data"]; ok {
		t.Error("original top-level volume key should be renamed, not kept")
	}
	renamed, ok := topVols["app-data_2"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected app-data_2 volume, got %v", topVols)
	}
	if renamed["name"] != "app-data_2" {
		t.Errorf("explicit volume name = %v, want app-data_2", renamed["name"])
	}
}

func TestApplyInstanceTransformsNoExplicitVolumeName(t *testing.T) {
	doc, err := Parse(`
services:
  app:
    image: example/app:latest
    volumes:
      - app-data:/data

volumes:
  app-data:
`)
	if err != nil {
		t.Fatal(err)
	}
	applyInstanceTransforms(doc, getServices(doc), 3)
	topVols := doc["volumes"].(map[string]interface{})
	if _, ok := topVols["app-data_3"]; !ok {
		t.Errorf("expected app-data_3, got %v", topVols)
	}
	if _, ok := topVols["app-data"]; ok {
		t.Errorf("original key should not survive, got %v", topVols)
	}
}

// ─── Expiration labels ───────────────────────────────────────────────────────

func TestApplyExpirationLabels(t *testing.T) {
	doc, err := Parse(`
services:
  app:
    image: example/app:latest
    labels:
      - existing=keepme
`)
	if err != nil {
		t.Fatal(err)
	}
	before := unixNow()
	applyExpirationLabels(getServices(doc), 2)
	after := unixNow()

	labels := getServices(doc)["app"].(map[string]interface{})["labels"].(map[string]interface{})
	if labels["yantr.temporary"] != "true" {
		t.Errorf("yantr.temporary = %v, want true", labels["yantr.temporary"])
	}
	if labels["existing"] != "keepme" {
		t.Errorf("pre-existing label lost: %v", labels)
	}
	raw, ok := labels["yantr.expireAt"].(string)
	if !ok {
		t.Fatalf("yantr.expireAt missing or not a string: %v", labels["yantr.expireAt"])
	}
	expireAt, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		t.Fatalf("yantr.expireAt %q is not an integer: %v", raw, err)
	}
	lo, hi := before+2*3600, after+2*3600
	if expireAt < lo || expireAt > hi {
		t.Errorf("yantr.expireAt = %d, want within [%d, %d]", expireAt, lo, hi)
	}
}

func TestApplyExpirationLabelsIgnoresNonPositive(t *testing.T) {
	doc, err := Parse(`
services:
  app:
    image: example/app:latest
`)
	if err != nil {
		t.Fatal(err)
	}
	applyExpirationLabels(getServices(doc), 0)
	svc := getServices(doc)["app"].(map[string]interface{})
	if _, ok := svc["labels"]; ok {
		t.Error("a zero/negative duration must not stamp labels")
	}
}

func TestApplyAbsoluteExpirationLabels(t *testing.T) {
	doc, err := Parse(`
services:
  a:
    image: example/a:latest
  b:
    image: example/b:latest
`)
	if err != nil {
		t.Fatal(err)
	}
	applyAbsoluteExpirationLabels(getServices(doc), 1893456000)
	for name, svcRaw := range getServices(doc) {
		labels := svcRaw.(map[string]interface{})["labels"].(map[string]interface{})
		if labels["yantr.expireAt"] != "1893456000" {
			t.Errorf("service %q expireAt = %v, want 1893456000", name, labels["yantr.expireAt"])
		}
		if labels["yantr.temporary"] != "true" {
			t.Errorf("service %q temporary = %v, want true", name, labels["yantr.temporary"])
		}
	}
}

// A compose file may declare an absolute x-yantr.expireAt instead of a
// deploy-time duration.
func TestApplyTransformsHonoursAbsoluteExpiry(t *testing.T) {
	doc, err := Parse(`
x-yantr:
  expireAt: 1893456000
services:
  app:
    image: example/app:latest
`)
	if err != nil {
		t.Fatal(err)
	}
	if err := ApplyTransforms(doc, TransformOptions{
		ProjectID: "app", AppID: "app", HostDockerSocket: "/run/podman.sock",
	}); err != nil {
		t.Fatal(err)
	}
	labels := getServices(doc)["app"].(map[string]interface{})["labels"].(map[string]interface{})
	if labels["yantr.expireAt"] != "1893456000" {
		t.Errorf("x-yantr.expireAt not applied: %v", labels)
	}
}

// ─── Port string parsing ─────────────────────────────────────────────────────

func TestParseComposePortString(t *testing.T) {
	cases := []struct {
		in            string
		wantNil       bool
		published     string
		target        int
		proto, hostIP string
	}{
		{in: "8080", target: 8080, proto: "tcp"},
		{in: "53/udp", target: 53, proto: "udp"},
		{in: `"9090:80"`, published: "9090", target: 80, proto: "tcp"},
		{in: "127.0.0.1:8080:80/udp", published: "8080", target: 80, proto: "udp", hostIP: "127.0.0.1"},
		{in: "'51820:51820'", published: "51820", target: 51820, proto: "tcp"},
		{in: "8080/bogus", wantNil: true}, // unknown protocol is not stripped, so Atoi fails
		{in: "not-a-port", wantNil: true},
		{in: "a:b:c:d", wantNil: true},
		{in: "", wantNil: true},
	}
	for _, tc := range cases {
		got := parseComposePortString(tc.in)
		if tc.wantNil {
			if got != nil {
				t.Errorf("parseComposePortString(%q) = %+v, want nil", tc.in, got)
			}
			continue
		}
		if got == nil {
			t.Errorf("parseComposePortString(%q) = nil, want a value", tc.in)
			continue
		}
		if got.Published != tc.published || got.Target != tc.target ||
			got.Protocol != tc.proto || got.HostIP != tc.hostIP {
			t.Errorf("parseComposePortString(%q) = %+v, want published=%q target=%q proto=%q hostIP=%q",
				tc.in, got, tc.published, tc.target, tc.proto, tc.hostIP)
		}
	}
}

func TestParseComposePortInput(t *testing.T) {
	cases := []struct {
		in              string
		wantNil         bool
		hasExplicitHost bool
		host, container int
		proto           string
	}{
		{in: "8080", container: 8080, proto: "tcp"},
		{in: "9090:80", hasExplicitHost: true, host: 9090, container: 80, proto: "tcp"},
		{in: "53:53/udp", hasExplicitHost: true, host: 53, container: 53, proto: "udp"},
		{in: "0", wantNil: true},
		{in: "70000", wantNil: true},
		{in: "9090:70000", wantNil: true},
		{in: "127.0.0.1:8080:80", wantNil: true}, // host IP form is not accepted here
		{in: "notaport", wantNil: true},
		{in: "  ", wantNil: true},
		{in: "", wantNil: true},
	}
	for _, tc := range cases {
		got := ParseComposePortInput(tc.in)
		if tc.wantNil {
			if got != nil {
				t.Errorf("ParseComposePortInput(%q) = %+v, want nil", tc.in, got)
			}
			continue
		}
		if got == nil {
			t.Errorf("ParseComposePortInput(%q) = nil, want a value", tc.in)
			continue
		}
		if got.ContainerPort != tc.container || got.Protocol != tc.proto || got.HasExplicitHost != tc.hasExplicitHost {
			t.Errorf("ParseComposePortInput(%q) = %+v, want container=%d proto=%q explicit=%v",
				tc.in, got, tc.container, tc.proto, tc.hasExplicitHost)
		}
		if tc.hasExplicitHost {
			if got.HostPort == nil || *got.HostPort != tc.host {
				t.Errorf("ParseComposePortInput(%q) hostPort = %v, want %d", tc.in, got.HostPort, tc.host)
			}
		} else if got.HostPort != nil {
			t.Errorf("ParseComposePortInput(%q) should have no host port, got %d", tc.in, *got.HostPort)
		}
	}
}
