// Package apps provides the apps catalog reader with caching.
package apps

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"core/shared"
)

// PortInfo describes a port exposed by an app.
type PortInfo struct {
	Port     int    `json:"port"`
	Protocol string `json:"protocol"`
	Label    string `json:"label"`
}

// EnvGenerator describes how to auto-generate an environment variable value.
type EnvGenerator struct {
	Length  int    `json:"length,omitempty"`
	Charset string `json:"charset,omitempty"`
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
	cacheMu   sync.Mutex
	cache     *Catalog
	cacheExp  time.Time
	cacheTTL  = 60 * time.Second
)

var appsDir = func() string {
	// Resolved relative to the binary or passed via env
	if d := os.Getenv("YANTR_APPS_DIR"); d != "" {
		return d
	}
	// Default: sibling apps/ directory relative to the binary's parent
	exe, _ := os.Executable()
	return filepath.Join(filepath.Dir(exe), "..", "apps")
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

// infoJSON matches the structure of info.json files.
type infoJSON struct {
	Name             string                            `json:"name"`
	Logo             string                            `json:"logo"`
	Tags             []string                          `json:"tags"`
	Ports            []PortInfo                        `json:"ports"`
	ShortDescription string                            `json:"short_description"`
	Description      string                            `json:"description"`
	Usecases         []string                          `json:"usecases"`
	Website          string                            `json:"website"`
	CustomApp        bool                              `json:"customapp"`
	Notes            []string                          `json:"notes"`
	EnvGenerators    map[string]json.RawMessage        `json:"env_generators"`
	EnvGeneratorsAlt map[string]json.RawMessage        `json:"envGenerators"`
}

var (
	// Matches list-style env: - KEY=${VAR:-default} or - KEY=${VAR}
	envListRe    = regexp.MustCompile(`-\s+([A-Za-z_][A-Za-z0-9_]*)=\$\{([A-Za-z_][A-Za-z0-9_]*):?-?([^}]*)\}`)
	// Matches map-style env:   KEY: ${VAR:-default} or KEY: ${VAR}
	envMapRe     = regexp.MustCompile(`(?m)^\s+([A-Za-z_][A-Za-z0-9_]*):\s*\$\{([A-Za-z_][A-Za-z0-9_]*):?-?([^}]*)\}`)
	// Matches fixed port: - "8096:8096" or - 8096:8096
	fixedPortRe  = regexp.MustCompile(`-\s*["']?(\d+):(\d+)(?:/(tcp|udp))?["']?`)
	// Matches auto port: - "8096"
	autoPortRe   = regexp.MustCompile(`-\s*["'](\d+)["'](?:\s|$)`)
)

func loadCatalog() (*Catalog, error) {
	entries, err := os.ReadDir(appsDir)
	if err != nil {
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
			infoPath := filepath.Join(appPath, "info.json")

			if _, err := os.Stat(composePath); err != nil {
				return
			}
			if _, err := os.Stat(infoPath); err != nil {
				return
			}

			infoData, err := os.ReadFile(infoPath)
			if err != nil {
				shared.Log("warn", "apps: failed to read "+infoPath+": "+err.Error())
				return
			}
			var info infoJSON
			if err := json.Unmarshal(infoData, &info); err != nil || info.Name == "" {
				return
			}

			composeContent, err := os.ReadFile(composePath)
			if err != nil {
				return
			}
			composeStr := string(composeContent)

			// Parse env vars
			envVars := parseEnvVars(composeStr)

			// Parse port mappings
			composePorts := parseComposePorts(composeStr)

			// Parse env generators
			rawGens := info.EnvGenerators
			if len(rawGens) == 0 {
				rawGens = info.EnvGeneratorsAlt
			}
			envGenerators := make(map[string]EnvGenerator)
			for k, v := range rawGens {
				var gen EnvGenerator
				if err := json.Unmarshal(v, &gen); err == nil {
					envGenerators[k] = gen
				}
			}

			logo := shared.NormalizeAppLogo(info.Logo)

			tags := info.Tags
			if tags == nil {
				tags = []string{}
			}
			ports := info.Ports
			if ports == nil {
				ports = []PortInfo{}
			}
			usecases := info.Usecases
			if usecases == nil {
				usecases = []string{}
			}

			app := App{
				ID:               entry.Name(),
				Name:             info.Name,
				Logo:             logo,
				Tags:             tags,
				Ports:            ports,
				ShortDescription: info.ShortDescription,
				Description:      coalesce(info.Description, info.ShortDescription),
				Usecases:         usecases,
				Website:          info.Website,
				CustomApp:        info.CustomApp,
				Notes:            info.Notes,
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
