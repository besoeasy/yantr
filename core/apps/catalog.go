// Package apps provides the apps catalog reader with caching.
package apps

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"core/shared"

	"gopkg.in/yaml.v3"
)

// EnvGenerator describes how to auto-generate an environment variable value.
type EnvGenerator struct {
	Length  int    `yaml:"length,omitempty" json:"length,omitempty"`
	Charset string `yaml:"charset,omitempty" json:"charset,omitempty"`
}

// EnvVar describes an environment variable extracted from compose.yml.
type EnvVar struct {
	Name    string `json:"name"`
	EnvVar  string `json:"envVar"`
	Default string `json:"default"`
}

// PortMapping describes a port mapping found in compose.yml.
type PortMapping struct {
	HostPort      string `json:"hostPort"`
	ContainerPort string `json:"containerPort"`
	Protocol      string `json:"protocol"`
}

// PortInfo describes a port exposed by an app — derived from x-yantr.ports.
type PortInfo struct {
	Port     int    `json:"port"`
	Protocol string `json:"protocol"`
	Label    string `json:"label"`
	Service  string `json:"service,omitempty"`

	// Show marks a port the user is meant to interact with. It gates the
	// "Open" affordance in the UI and is the reason it is serialised
	// unconditionally: the UI has to distinguish "work port" (Show false)
	// from "no catalog entry at all", and omitting the field would collapse
	// both into undefined.
	//
	// Deny by default — a port with no `show` key in x-yantr.ports is a work
	// port: it is still published and still serves its protocol, it just has
	// no UI surface. See parseXyPorts.
	Show bool `json:"show"`
}

// DeveloperInfo identifies one author of an app. Only Platform and Handle are
// authored in compose.yml: URL is derived here so a platform's URL scheme
// changes in exactly one place, and display data (name, avatar) is never
// authored at all — that comes from the platform's API, when it has one.
type DeveloperInfo struct {
	Platform string `json:"platform"`
	Handle   string `json:"handle"`
	URL      string `json:"url"`
}

// ShownPorts returns only the ports the app marked show: true, in catalog
// order. Callers that render a port list should use this rather than Ports so
// work ports stay out of the user's way; Ports remains the full set because
// port metadata is needed even for work ports (a work port still identifies
// its owning service, and still gets published).
func (a *App) ShownPorts() []PortInfo {
	if a == nil {
		return nil
	}
	var out []PortInfo
	for _, p := range a.Ports {
		if p.Show {
			out = append(out, p)
		}
	}
	if out == nil {
		return []PortInfo{}
	}
	return out
}

// App represents a single app in the catalog.
type App struct {
	ID               string                  `json:"id"`
	Name             string                  `json:"name"`
	Logo             string                  `json:"logo"`
	Tags             []string                `json:"tags"`
	Ports            []PortInfo              `json:"ports"`
	ShortDescription string                  `json:"short_description"`
	Description      string                  `json:"description"`
	Usecases         []string                `json:"usecases"`
	Website          string                  `json:"website"`
	Notes            []string                `json:"notes,omitempty"`
	Developers       []DeveloperInfo         `json:"developers"`
	Path             string                  `json:"path"`
	ComposePath      string                  `json:"composePath"`
	Environment      []EnvVar                `json:"environment"`
	EnvGenerators    map[string]EnvGenerator `json:"envGenerators"`
	ComposePorts     []PortMapping           `json:"composePorts"`
}

// Catalog is the result returned from the apps directory.
type Catalog struct {
	Apps  []App `json:"apps"`
	Count int   `json:"count"`

	// Index maps app ID to a pointer into Apps. It is built once per cache fill
	// alongside Apps, so it cannot drift from the slice it indexes and it
	// invalidates together with it. Callers that only need a membership test
	// must use IsCatalogApp rather than rebuilding their own map per request.
	//
	// Treated as read-only by consumers.
	Index map[string]*App `json:"-"`
}

// GetCatalogIndex returns the cached app ID -> *App index. The map is owned by
// the catalog cache and must not be mutated by callers.
func GetCatalogIndex() map[string]*App {
	cat, _ := GetCatalogCached(false)
	if cat == nil || cat.Index == nil {
		return map[string]*App{}
	}
	return cat.Index
}

// IsCatalogApp reports whether id is a known catalog app. Cheaper than
// GetCatalogIndex for the common membership check.
func IsCatalogApp(id string) bool {
	cat, _ := GetCatalogCached(false)
	if cat == nil || cat.Index == nil {
		return false
	}
	_, ok := cat.Index[id]
	return ok
}

var (
	cacheMu  sync.Mutex
	cache    *Catalog
	cacheExp time.Time
	cacheTTL = 60 * time.Second
)

var appsDir = func() string {
	if d := os.Getenv("YANTR_APPS_DIR"); d != "" {
		return d
	}
	exe, _ := os.Executable()
	return filepath.Join(filepath.Dir(exe), "apps")
}()

// SetAppsDir overrides the apps directory (used by main.go).
func SetAppsDir(dir string) {
	appsDir = dir
}

// GetAppsDir returns the current apps directory.
func GetAppsDir() string {
	return appsDir
}

// GetCatalogCached returns the apps catalog, using a 60s TTL cache.
func GetCatalogCached(forceRefresh bool) (*Catalog, error) {
	cacheMu.Lock()
	defer cacheMu.Unlock()

	if !forceRefresh && cache != nil && time.Now().Before(cacheExp) {
		return cache, nil
	}

	cat, err := loadCatalog()
	if err != nil {
		return nil, err
	}
	cache = cat
	cacheExp = time.Now().Add(cacheTTL)
	return cat, nil
}

// ─── x-yantr YAML structure ───────────────────────────────────────────────────

// xYantr matches the x-yantr extension block in compose.yml.
type xYantr struct {
	Name             string                  `yaml:"name"`
	Tags             []string                `yaml:"tags"`
	ShortDescription string                  `yaml:"short_description"`
	Description      string                  `yaml:"description"`
	Usecases         []string                `yaml:"usecases"`
	Website          string                  `yaml:"website"`
	Notes            []string                `yaml:"notes"`
	Developers       []string                `yaml:"developers"`
	EnvGenerators    map[string]EnvGenerator `yaml:"env_generators"`
	Ports            []xyPort                `yaml:"ports"`
}

// xyPort is a single structured port entry under x-yantr.ports.
type xyPort struct {
	Port     int    `yaml:"port"`
	Protocol string `yaml:"protocol"`
	Label    string `yaml:"label"`
	Service  string `yaml:"service"`

	// Show opts the port into the user-facing UI. Absent means false: see
	// PortInfo.Show.
	Show bool `yaml:"show"`
}

// composeFile is a minimal representation of the top-level compose.yml structure.
type composeFile struct {
	XYantr   xYantr                  `yaml:"x-yantr"`
	Services map[string]interface{} `yaml:"services"`
}

// ─── Catalog loader ───────────────────────────────────────────────────────────

func loadCatalog() (*Catalog, error) {
	entries, err := os.ReadDir(appsDir)
	if err != nil {
		shared.Log("warn", "apps: failed to read appsDir "+appsDir+": "+err.Error())
		return &Catalog{Apps: []App{}, Count: 0, Index: map[string]*App{}}, nil
	}

	var apps []App
	var mu sync.Mutex
	var wg sync.WaitGroup

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		wg.Add(1)
		go func(entry os.DirEntry) {
			defer wg.Done()
			appPath := filepath.Join(appsDir, entry.Name())
			composePath := filepath.Join(appPath, "compose.yml")

			if _, err := os.Stat(composePath); err != nil {
				return
			}

			composeContent, err := os.ReadFile(composePath)
			if err != nil {
				shared.Log("warn", "apps: failed to read "+composePath+": "+err.Error())
				return
			}

			var cf composeFile
			if err := yaml.Unmarshal(composeContent, &cf); err != nil || cf.XYantr.Name == "" {
				return
			}

			meta := cf.XYantr
			composeStr := string(composeContent)

			// Structured display ports — single source of truth.
			ports := parseXyPorts(meta.Ports, cf.Services)

			envVars := parseEnvVars(composeStr)
			composePorts := parseComposePorts(composeStr)

			// Normalise slices — never return null in JSON
			tags := meta.Tags
			if tags == nil {
				tags = []string{}
			}
			usecases := meta.Usecases
			if usecases == nil {
				usecases = []string{}
			}
			// parseDevelopers always returns a non-nil slice, so the app page
			// can tell "no developers declared" from "field missing" without a
			// null check on every consumer.
			developers := parseDevelopers(meta.Developers)
			envGenerators := meta.EnvGenerators
			if envGenerators == nil {
				envGenerators = map[string]EnvGenerator{}
			}

			app := App{
				ID:               entry.Name(),
				Name:             meta.Name,
				Logo:             shared.NormalizeAppLogo(appPath),
				Tags:             tags,
				Ports:            ports,
				ShortDescription: meta.ShortDescription,
				Description:      coalesce(meta.Description, meta.ShortDescription),
				Usecases:         usecases,
			Website:          meta.Website,
			Notes:            meta.Notes,
			Developers:       developers,
				Path:             appPath,
				ComposePath:      composePath,
				Environment:      envVars,
				EnvGenerators:    envGenerators,
				ComposePorts:     composePorts,
			}

			mu.Lock()
			apps = append(apps, app)
			mu.Unlock()
		}(entry)
	}

	wg.Wait()

	if apps == nil {
		apps = []App{}
	} else {
		sort.Slice(apps, func(i, j int) bool {
			return apps[i].ID < apps[j].ID
		})
	}

	// Build the ID index once, here, so every consumer shares one map per cache
	// fill instead of rebuilding it per request. Pointers stay valid for as long
	// as the cache entry does because the cache replaces the whole Catalog
	// rather than mutating Apps in place.
	index := make(map[string]*App, len(apps))
	for i := range apps {
		index[apps[i].ID] = &apps[i]
	}

	return &Catalog{Apps: apps, Count: len(apps), Index: index}, nil
}

// ─── Developer parsing (x-yantr.developers) ───────────────────────────────────

// validDeveloperPlatforms is the closed set for x-yantr.developers entries.
//
// The set is closed for the same reason validDisplayProtocols is: an entry is
// turned into a profile link on the app page and, for GitHub, an @mention in a
// pre-filled issue. An unrecognised prefix has no profile URL to derive, so it
// would render as a dead link or mention the wrong account. Drop it instead.
var validDeveloperPlatforms = map[string]bool{
	"github":   true,
	"gitlab":   true,
	"codeberg": true,
	"nostr":    true,
}

// developerProfileURL derives the canonical profile URL for one entry. Kept
// separate from parseDevelopers so the URL scheme for a platform lives in one
// place; an empty return means "no profile page", which cannot happen for a
// platform that survived validation.
func developerProfileURL(platform, handle string) string {
	switch platform {
	case "github":
		return "https://github.com/" + handle
	case "gitlab":
		return "https://gitlab.com/" + handle
	case "codeberg":
		return "https://codeberg.org/" + handle
	case "nostr":
		// njump routes any NIP-19 identifier without assuming the reader has a
		// particular client, which a bare nostr: scheme link does.
		return "https://njump.me/" + handle
	}
	return ""
}

// parseDevelopers normalises x-yantr.developers ("platform:handle") into a
// deduplicated list of DeveloperInfo, preserving author order.
//
// Entries are dropped, never guessed at: a missing separator, an unknown
// platform, an empty handle, or a handle containing URL metacharacters all mean
// the entry cannot be trusted, and a wrong entry here is worse than a missing
// one — on GitHub it subscribes a stranger to every issue filed about the app.
//
// Handles are lowercased on the forge platforms because those are
// case-insensitive, so `Besoeasy` and `besoeasy` must not both survive as
// separate developers. Nostr keys keep their case: bech32 is lowercase by
// spec, but the field should not silently rewrite a key it did not author.
func parseDevelopers(in []string) []DeveloperInfo {
	seen := map[string]bool{}
	out := []DeveloperInfo{}

	for _, raw := range in {
		platform, handle, ok := strings.Cut(strings.TrimSpace(raw), ":")
		if !ok {
			continue
		}
		platform = strings.ToLower(strings.TrimSpace(platform))
		handle = strings.TrimSpace(handle)
		// Tolerate the "@handle" spelling, which is how people write these.
		handle = strings.TrimPrefix(handle, "@")

		if handle == "" || !validDeveloperPlatforms[platform] {
			continue
		}
		if strings.ContainsAny(handle, " \t\n/?#") {
			continue
		}
		if platform != "nostr" {
			handle = strings.ToLower(handle)
		}

		key := platform + ":" + handle
		if seen[key] {
			continue
		}
		seen[key] = true

		out = append(out, DeveloperInfo{
			Platform: platform,
			Handle:   handle,
			URL:      developerProfileURL(platform, handle),
		})
	}

	return out
}

// DevelopersFor returns the handles this app declares on one platform, in
// catalog order. Callers that need to test membership (the developer route) or
// build mentions (the issue body) should use this rather than re-walking
// Developers and re-deciding what counts.
func (a *App) DevelopersFor(platform string) []string {
	if a == nil {
		return nil
	}
	var out []string
	for _, d := range a.Developers {
		if d.Platform == platform {
			out = append(out, d.Handle)
		}
	}
	return out
}

// AppsByDeveloper returns every catalog app declaring platform:handle, in
// catalog order (which loadCatalog sorts by ID).
//
// It reads only the local catalog on purpose. The developer route enriches the
// page with GitHub profile data when it can, but the app list must survive
// GitHub being unreachable, rate-limited or simply not the developer's
// platform — Yantr is rootless and expected to run against no internet at all.
func AppsByDeveloper(platform, handle string) []*App {
	cat, _ := GetCatalogCached(false)
	if cat == nil {
		return []*App{}
	}
	if platform != "nostr" {
		handle = strings.ToLower(handle)
	}
	out := []*App{}
	for i := range cat.Apps {
		for _, d := range cat.Apps[i].Developers {
			if d.Platform == platform && d.Handle == handle {
				out = append(out, &cat.Apps[i])
				break
			}
		}
	}
	return out
}

// ─── Structured port parsing (x-yantr.ports) ──────────────────────────────────

// validDisplayProtocols is the closed set for x-yantr.ports[].protocol.
var validDisplayProtocols = map[string]bool{
	"HTTP": true, "HTTPS": true, "TCP": true, "UDP": true,
}

// parseXyPorts normalises x-yantr.ports into a deduplicated, sorted PortInfo list.
// Key is (port, PROTOCOL, service) so multi-service / multi-protocol apps keep
// their service dimension. Unknown services are kept (forward-compat) but the
// common mistake — labeling a port owned by another service — is fixed at
// migration time by assigning service to the actual exposing service.
//
// `show` is deny-by-default: a port without it is a work port. Work ports are
// kept in the list (they are still published, and they still identify their
// service) and reach the UI only behind its "All" toggle. When the same
// (port, protocol, service) is declared twice, Show is OR-ed rather than taken
// from whichever entry happened to be read first, so a duplicate work-port
// declaration can never silently hide a port another entry exposes.
func parseXyPorts(in []xyPort, services map[string]interface{}) []PortInfo {
	idxByKey := map[string]int{}
	var ports []PortInfo

	for _, p := range in {
		if p.Port < 1 || p.Port > 65535 {
			continue
		}
		proto := strings.ToUpper(strings.TrimSpace(p.Protocol))
		if proto == "" {
			proto = "TCP"
		}
		if !validDisplayProtocols[proto] {
			continue
		}
		label := strings.TrimSpace(p.Label)
		if label == "" {
			label = fmt.Sprintf("Port %d", p.Port)
		}
		service := strings.TrimSpace(p.Service)
		// Drop references to services that don't exist — these are the
		// owncloud-style bogus entries the old label system hid via dedup.
		// Keep empty service (unspecified) for forward-compat.
		if service != "" && services != nil {
			if _, ok := services[service]; !ok {
				continue
			}
		}
		key := fmt.Sprintf("%d/%s/%s", p.Port, proto, service)
		if i, ok := idxByKey[key]; ok {
			if p.Show {
				ports[i].Show = true
			}
			continue
		}
		idxByKey[key] = len(ports)
		ports = append(ports, PortInfo{
			Port:     p.Port,
			Protocol: proto,
			Label:    label,
			Service:  service,
			Show:     p.Show,
		})
	}

	if ports == nil {
		ports = []PortInfo{}
	} else {
		sort.Slice(ports, func(i, j int) bool {
			if ports[i].Port != ports[j].Port {
				return ports[i].Port < ports[j].Port
			}
			if ports[i].Protocol != ports[j].Protocol {
				return ports[i].Protocol < ports[j].Protocol
			}
			return ports[i].Service < ports[j].Service
		})
	}
	return ports
}

// ─── Env var parsing ──────────────────────────────────────────────────────────

var (
	// Matches list-style env: - KEY=${VAR:-default} or - KEY=${VAR}
	envListRe = regexp.MustCompile(`-\s+([A-Za-z_][A-Za-z0-9_]*)=\$\{([A-Za-z_][A-Za-z0-9_]*):?-?([^}]*)\}`)
	// Matches map-style env:   KEY: ${VAR:-default} or KEY: ${VAR}
	envMapRe = regexp.MustCompile(`(?m)^\s+([A-Za-z_][A-Za-z0-9_]*):\s*\$\{([A-Za-z_][A-Za-z0-9_]*):?-?([^}]*)\}`)
	// Matches fixed port: - "8096:8096" or - 8096:8096
	fixedPortRe = regexp.MustCompile(`-\s*["']?(\d+):(\d+)(?:/(tcp|udp))?["']?`)
	// Matches auto port: - "8096"
	autoPortRe = regexp.MustCompile(`-\s*["'](\d+)["'](?:\s|$)`)
)

func parseEnvVars(composeStr string) []EnvVar {
	seen := map[string]bool{}
	var vars []EnvVar

	for _, m := range envListRe.FindAllStringSubmatch(composeStr, -1) {
		if seen[m[2]] {
			continue
		}
		seen[m[2]] = true
		vars = append(vars, EnvVar{Name: m[1], EnvVar: m[2], Default: m[3]})
	}

	for _, m := range envMapRe.FindAllStringSubmatch(composeStr, -1) {
		if seen[m[2]] {
			continue
		}
		seen[m[2]] = true
		vars = append(vars, EnvVar{Name: m[1], EnvVar: m[2], Default: m[3]})
	}

	if vars == nil {
		vars = []EnvVar{}
	}
	return vars
}

func parseComposePorts(composeStr string) []PortMapping {
	portSet := map[string]bool{}
	var ports []PortMapping

	for _, m := range fixedPortRe.FindAllStringSubmatch(composeStr, -1) {
		proto := m[3]
		if proto == "" {
			proto = "tcp"
		}
		ports = append(ports, PortMapping{HostPort: m[1], ContainerPort: m[2], Protocol: proto})
		portSet[m[2]] = true
	}

	for _, m := range autoPortRe.FindAllStringSubmatch(composeStr, -1) {
		if portSet[m[1]] {
			continue
		}
		portSet[m[1]] = true
		ports = append(ports, PortMapping{HostPort: m[1], ContainerPort: m[1], Protocol: "tcp"})
	}

	if ports == nil {
		ports = []PortMapping{}
	}
	return ports
}

func coalesce(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}
