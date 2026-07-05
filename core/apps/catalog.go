// Package apps provides the apps catalog reader with caching.
package apps

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
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

// PortInfo describes a port exposed by an app — derived from yantr.port.N labels.
type PortInfo struct {
	Port     int    `json:"port"`
	Protocol string `json:"protocol"`
	Label    string `json:"label"`
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
	CustomApp        bool                    `json:"customapp"`
	Notes            []string                `json:"notes,omitempty"`
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
	Logo             string                  `yaml:"logo"`
	Tags             []string                `yaml:"tags"`
	ShortDescription string                  `yaml:"short_description"`
	Description      string                  `yaml:"description"`
	Usecases         []string                `yaml:"usecases"`
	Website          string                  `yaml:"website"`
	CustomApp        bool                    `yaml:"customapp"`
	Notes            []string                `yaml:"notes"`
	EnvGenerators    map[string]EnvGenerator `yaml:"env_generators"`
}

// composeFile is a minimal representation of the top-level compose.yml structure.
type composeFile struct {
	XYantr   xYantr                    `yaml:"x-yantr"`
	Services map[string]composeService `yaml:"services"`
}

// composeService captures only the labels we care about.
type composeService struct {
	Labels map[string]string `yaml:"labels"`
}

// ─── Catalog loader ───────────────────────────────────────────────────────────

func loadCatalog() (*Catalog, error) {
	entries, err := os.ReadDir(appsDir)
	if err != nil {
		shared.Log("warn", "apps: failed to read appsDir "+appsDir+": "+err.Error())
		return &Catalog{Apps: []App{}, Count: 0}, nil
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

			// Derive port info from yantr.port.N labels across all services
			ports := parsePortLabels(cf.Services)

			// Parse env vars and port mappings from raw compose text
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
			envGenerators := meta.EnvGenerators
			if envGenerators == nil {
				envGenerators = map[string]EnvGenerator{}
			}

			app := App{
				ID:               entry.Name(),
				Name:             meta.Name,
				Logo:             shared.NormalizeAppLogo(meta.Logo),
				Tags:             tags,
				Ports:            ports,
				ShortDescription: meta.ShortDescription,
				Description:      coalesce(meta.Description, meta.ShortDescription),
				Usecases:         usecases,
				Website:          meta.Website,
				CustomApp:        meta.CustomApp,
				Notes:            meta.Notes,
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
	return &Catalog{Apps: apps, Count: len(apps)}, nil
}

// ─── Label-based port parsing ─────────────────────────────────────────────────

// parsePortLabels scans every service's labels for yantr.port.N: "PROTOCOL"
// and yantr.service.N: "Label", returning a deduplicated PortInfo list.
func parsePortLabels(services map[string]composeService) []PortInfo {
	seen := map[int]bool{}
	var ports []PortInfo

	for _, svc := range services {
		for key, protocol := range svc.Labels {
			// key format: yantr.port.{N}
			if !strings.HasPrefix(key, "yantr.port.") {
				continue
			}
			portStr := strings.TrimPrefix(key, "yantr.port.")
			n, err := strconv.Atoi(portStr)
			if err != nil || seen[n] {
				continue
			}
			seen[n] = true
			serviceLabel := coalesce(svc.Labels[fmt.Sprintf("yantr.service.%d", n)], fmt.Sprintf("Port %d", n))
			ports = append(ports, PortInfo{
				Port:     n,
				Protocol: strings.ToUpper(protocol),
				Label:    serviceLabel,
			})
		}
	}

	if ports == nil {
		ports = []PortInfo{}
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
