// Package compose provides YAML compose file operations using gopkg.in/yaml.v3.
//
// This replaces stack-compose.js using Go's YAML library. The compose-go
// library requires a full Docker context, so we use gopkg.in/yaml.v3 for
// direct manipulation, which gives us full control over the YAML structure.
package compose

import (
	"bufio"
	"fmt"
	"time"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

// ComposeDoc is a parsed Docker Compose file.
type ComposeDoc map[string]interface{}

// ProjectComposeFileName returns the hidden compose filename for a project instance.
func ProjectComposeFileName(projectID string) string {
	return ".compose." + projectID + ".yml"
}

// ProjectComposePath returns the full path to the project-specific compose file.
func ProjectComposePath(appPath, projectID string) string {
	return filepath.Join(appPath, ProjectComposeFileName(projectID))
}

// ProjectEnvFileName returns the hidden env filename for a project instance.
func ProjectEnvFileName(projectID string) string {
	return ".env." + projectID
}

// ProjectEnvPath returns the full path to the project-specific env file.
func ProjectEnvPath(appPath, projectID string) string {
	return filepath.Join(appPath, ProjectEnvFileName(projectID))
}

// ComposeRef holds information about which compose file to use for a project.
type ComposeRef struct {
	ComposePath      string
	ComposeFile      string
	IsProjectCompose bool
}

// GetProjectComposeRef returns the compose file reference for a given project.
// Prefers the project-specific hidden file; falls back to compose.yml.
func GetProjectComposeRef(appPath, projectID string) ComposeRef {
	projectPath := ProjectComposePath(appPath, projectID)
	if _, err := os.Stat(projectPath); err == nil {
		return ComposeRef{
			ComposePath:      projectPath,
			ComposeFile:      ProjectComposeFileName(projectID),
			IsProjectCompose: true,
		}
	}
	return ComposeRef{
		ComposePath:      filepath.Join(appPath, "compose.yml"),
		ComposeFile:      "compose.yml",
		IsProjectCompose: false,
	}
}

// Parse parses YAML compose content into a generic map.
func Parse(content string) (ComposeDoc, error) {
	var doc map[string]interface{}
	if err := yaml.Unmarshal([]byte(content), &doc); err != nil {
		return nil, fmt.Errorf("invalid compose YAML: %w", err)
	}
	if doc == nil {
		return nil, fmt.Errorf("empty compose file")
	}
	return doc, nil
}

// Stringify serializes a ComposeDoc back to YAML.
func Stringify(doc ComposeDoc) (string, error) {
	data, err := yaml.Marshal(doc)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// WriteProjectCompose writes a project-specific compose file and returns its ref.
func WriteProjectCompose(appPath, projectID, content string) (ComposeRef, error) {
	composePath := ProjectComposePath(appPath, projectID)
	if err := os.WriteFile(composePath, []byte(content), 0644); err != nil {
		return ComposeRef{}, err
	}
	return ComposeRef{
		ComposePath:      composePath,
		ComposeFile:      ProjectComposeFileName(projectID),
		IsProjectCompose: true,
	}, nil
}

// WriteProjectEnv writes the env file for a project. Removes the file if empty.
func WriteProjectEnv(appPath, projectID string, environment map[string]interface{}) (string, error) {
	envPath := ProjectEnvPath(appPath, projectID)

	var lines []string
	for k, v := range environment {
		k = strings.TrimSpace(k)
		if k == "" || v == nil {
			continue
		}
		val := fmt.Sprintf("%v", v)
		if strings.TrimSpace(val) == "" {
			continue
		}
		lines = append(lines, k+"="+val)
	}

	if len(lines) == 0 {
		_ = os.Remove(envPath)
		return "", nil
	}

	content := strings.Join(lines, "\n") + "\n"
	if err := os.WriteFile(envPath, []byte(content), 0600); err != nil {
		return "", err
	}
	return envPath, nil
}

// LoadProjectEnv reads the project env file (or fallback .env) and returns key=value pairs.
func LoadProjectEnv(appPath, projectID string) (map[string]string, error) {
	candidates := []string{
		ProjectEnvPath(appPath, projectID),
		filepath.Join(appPath, ".env"),
	}
	for _, p := range candidates {
		data, err := os.ReadFile(p)
		if err == nil {
			return ParseEnvFile(string(data)), nil
		}
	}
	return map[string]string{}, nil
}

// ParseEnvFile parses a simple KEY=VALUE env file.
func ParseEnvFile(content string) map[string]string {
	env := map[string]string{}
	scanner := bufio.NewScanner(strings.NewReader(content))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		idx := strings.Index(line, "=")
		if idx <= 0 {
			continue
		}
		key := strings.TrimSpace(line[:idx])
		val := line[idx+1:]
		if (strings.HasPrefix(val, `"`) && strings.HasSuffix(val, `"`)) ||
			(strings.HasPrefix(val, `'`) && strings.HasSuffix(val, `'`)) {
			val = val[1 : len(val)-1]
		}
		if key != "" {
			env[key] = val
		}
	}
	return env
}

// GetComposeProcessEnv builds the environment for running docker compose.
func GetComposeProcessEnv(appPath, projectID, dockerSocket string) (map[string]string, error) {
	projectEnv, err := LoadProjectEnv(appPath, projectID)
	if err != nil {
		projectEnv = map[string]string{}
	}

	// Start with current process environment
	env := map[string]string{}
	for _, e := range os.Environ() {
		idx := strings.Index(e, "=")
		if idx > 0 {
			env[e[:idx]] = e[idx+1:]
		}
	}
	// Overlay project env
	for k, v := range projectEnv {
		env[k] = v
	}
	// Set Docker host
	env["DOCKER_HOST"] = "unix://" + dockerSocket

	return env, nil
}

// DeleteProjectCompose removes both the project compose file and env file.
func DeleteProjectCompose(appPath, projectID string) {
	_ = os.Remove(ProjectComposePath(appPath, projectID))
	_ = os.Remove(ProjectEnvPath(appPath, projectID))
}

// BuildProjectComposeContent applies project-level transforms to a base compose file.
func BuildProjectComposeContent(baseContent string, opts TransformOptions) (string, error) {
	doc, err := Parse(baseContent)
	if err != nil {
		return "", err
	}
	if err := ApplyTransforms(doc, opts); err != nil {
		return "", err
	}
	return Stringify(doc)
}

// TransformOptions holds the options for compose transforms.
type TransformOptions struct {
	ProjectID        string
	AppID            string
	ExpiresIn        float64
	CustomPortMappings map[string]interface{}
	ExtraEnv         map[string]interface{}
	MasterApp        string
}

// ApplyTransforms applies all project-level transforms to the compose document in place.
func ApplyTransforms(doc ComposeDoc, opts TransformOptions) error {
	services := getServices(doc)

	// Instance transforms
	instanceID := getInstanceID(opts.ProjectID, opts.AppID)
	if instanceID > 0 {
		applyInstanceTransforms(doc, services, instanceID)
	}

	// Custom port mappings
	if len(opts.CustomPortMappings) > 0 {
		applyCustomPortMappings(services, opts.CustomPortMappings)
	}

	// Extra env
	if len(opts.ExtraEnv) > 0 {
		applyExtraEnv(services, opts.ExtraEnv)
	}

	// Expiration labels — deploy-time expiresIn (hours) takes precedence;
	// fall back to x-yantr.expireAt (absolute unix timestamp) from compose.yml.
	if opts.ExpiresIn > 0 {
		applyExpirationLabels(services, opts.ExpiresIn)
	} else if xyantr, ok := doc["x-yantr"].(map[string]interface{}); ok {
		if expireAtRaw, exists := xyantr["expireAt"]; exists {
			var expireAt int64
			switch v := expireAtRaw.(type) {
			case int:
				expireAt = int64(v)
			case int64:
				expireAt = v
			case float64:
				expireAt = int64(v)
			case string:
				expireAt, _ = strconv.ParseInt(v, 10, 64)
			}
			if expireAt > 0 {
				applyAbsoluteExpirationLabels(services, expireAt)
			}
		}
	}

	// Caddy master label
	if strings.TrimSpace(opts.MasterApp) != "" {
		applyCaddyMasterLabel(services, strings.TrimSpace(opts.MasterApp))
	}

	// Yantr app identity label
	if strings.TrimSpace(opts.AppID) != "" {
		applyYantrAppLabel(services, strings.TrimSpace(opts.AppID))
	}

	return nil
}

func applyYantrAppLabel(services map[string]interface{}, appID string) {
	for _, svcRaw := range services {
		svc, ok := svcRaw.(map[string]interface{})
		if !ok {
			continue
		}
		labels := ensureLabelsMap(svc)
		labels["yantr.app"] = appID
	}
}

func getServices(doc ComposeDoc) map[string]interface{} {
	if svcs, ok := doc["services"].(map[string]interface{}); ok {
		return svcs
	}
	return map[string]interface{}{}
}

func getInstanceID(projectID, appID string) int {
	if projectID == "" || appID == "" || projectID == appID {
		return 0
	}
	prefix := appID + "-"
	if !strings.HasPrefix(projectID, prefix) {
		return 0
	}
	numStr := projectID[len(prefix):]
	n, err := strconv.Atoi(numStr)
	if err != nil || n <= 1 {
		return 0
	}
	return n
}

func applyInstanceTransforms(doc ComposeDoc, services map[string]interface{}, instanceID int) {
	suffix := strconv.Itoa(instanceID)

	for _, svcRaw := range services {
		svc, ok := svcRaw.(map[string]interface{})
		if !ok {
			continue
		}
		if name, ok := svc["container_name"].(string); ok && strings.TrimSpace(name) != "" {
			svc["container_name"] = strings.TrimSpace(name) + "-" + suffix
		}
		if vols, ok := svc["volumes"].([]interface{}); ok {
			for i, v := range vols {
				if s, ok := v.(string); ok {
					vols[i] = suffixServiceVolume(s, suffix)
				}
			}
		}
	}

	if volumes, ok := doc["volumes"].(map[string]interface{}); ok {
		next := map[string]interface{}{}
		for name, cfg := range volumes {
			newName := name + "_" + suffix
			if cfgMap, ok := cfg.(map[string]interface{}); ok {
				if n, ok := cfgMap["name"].(string); ok && strings.TrimSpace(n) != "" {
					cfgMapCopy := map[string]interface{}{}
					for k, v := range cfgMap {
						cfgMapCopy[k] = v
					}
					cfgMapCopy["name"] = strings.TrimSpace(n) + "_" + suffix
					next[newName] = cfgMapCopy
					continue
				}
			}
			next[newName] = cfg
		}
		doc["volumes"] = next
	}
}

func suffixServiceVolume(entry, instanceID string) string {
	parts := strings.SplitN(entry, ":", 2)
	if len(parts) < 2 {
		return entry
	}
	src := parts[0]
	if src == "" || strings.HasPrefix(src, "/") || strings.HasPrefix(src, "./") ||
		strings.HasPrefix(src, "../") || strings.Contains(src, "${") {
		return entry
	}
	return src + "_" + instanceID + ":" + parts[1]
}

func applyCustomPortMappings(services map[string]interface{}, customPortMappings map[string]interface{}) {
	for _, svcRaw := range services {
		svc, ok := svcRaw.(map[string]interface{})
		if !ok {
			continue
		}
		ports, ok := svc["ports"].([]interface{})
		if !ok {
			continue
		}
		for i, portRaw := range ports {
			portStr, ok := portRaw.(string)
			if !ok {
				continue
			}
			parsed := parseComposePortString(portStr)
			if parsed == nil {
				continue
			}
			key := fmt.Sprintf("%d/%s", parsed.Target, parsed.Protocol)
			if mappedPort, ok := customPortMappings[key]; ok {
				hostPort := fmt.Sprintf("%v", mappedPort)
				ports[i] = hostPort + ":" + strconv.Itoa(parsed.Target) +
					func() string {
						if parsed.Protocol != "tcp" {
							return "/" + parsed.Protocol
						}
						return ""
					}()
			}
		}
		svc["ports"] = ports
	}
}

func applyExtraEnv(services map[string]interface{}, extraEnv map[string]interface{}) {
	for _, svcRaw := range services {
		svc, ok := svcRaw.(map[string]interface{})
		if !ok {
			continue
		}
		// Normalize environment to map
		switch env := svc["environment"].(type) {
		case []interface{}:
			envMap := map[string]interface{}{}
			for _, entry := range env {
				if s, ok := entry.(string); ok {
					idx := strings.Index(s, "=")
					if idx > 0 {
						envMap[s[:idx]] = s[idx+1:]
					} else {
						envMap[s] = nil
					}
				}
			}
			svc["environment"] = envMap
		case nil:
			svc["environment"] = map[string]interface{}{}
		}
		envMap, ok := svc["environment"].(map[string]interface{})
		if !ok {
			continue
		}
		for k, v := range extraEnv {
			envMap[strings.TrimSpace(k)] = v
		}
	}
}

func applyExpirationLabels(services map[string]interface{}, expiresInHours float64) {
	if expiresInHours <= 0 {
		return
	}
	expireAt := int64(float64(unixNow()) + expiresInHours*3600)
	applyAbsoluteExpirationLabels(services, expireAt)
}

// applyAbsoluteExpirationLabels stamps yantr.expireAt / yantr.temporary labels
// with an already-computed absolute unix timestamp.
func applyAbsoluteExpirationLabels(services map[string]interface{}, expireAt int64) {
	expireStr := strconv.FormatInt(expireAt, 10)
	for _, svcRaw := range services {
		svc, ok := svcRaw.(map[string]interface{})
		if !ok {
			continue
		}
		labels := ensureLabelsMap(svc)
		labels["yantr.expireAt"] = expireStr
		labels["yantr.temporary"] = "true"
	}
}

func applyCaddyMasterLabel(services map[string]interface{}, masterApp string) {
	for _, svcRaw := range services {
		svc, ok := svcRaw.(map[string]interface{})
		if !ok {
			continue
		}
		labels := ensureLabelsMap(svc)
		labels["yantr.caddy.master"] = masterApp
	}
}

func ensureLabelsMap(svc map[string]interface{}) map[string]interface{} {
	switch l := svc["labels"].(type) {
	case map[string]interface{}:
		return l
	default:
		m := map[string]interface{}{}
		svc["labels"] = m
		return m
	}
}

// portEntry represents a parsed Docker compose port entry.
type portEntry struct {
	Published string
	Target    int
	Protocol  string
	HostIP    string
}

func parseComposePortString(s string) *portEntry {
	s = strings.Trim(s, `"'`)
	// strip /protocol
	proto := "tcp"
	if idx := strings.LastIndex(s, "/"); idx >= 0 {
		p := strings.ToLower(s[idx+1:])
		if p == "tcp" || p == "udp" {
			proto = p
			s = s[:idx]
		}
	}

	parts := strings.Split(s, ":")
	switch len(parts) {
	case 1:
		n, err := strconv.Atoi(parts[0])
		if err != nil {
			return nil
		}
		return &portEntry{Target: n, Protocol: proto}
	case 2:
		target, err := strconv.Atoi(parts[1])
		if err != nil {
			return nil
		}
		return &portEntry{Published: parts[0], Target: target, Protocol: proto}
	case 3:
		target, err := strconv.Atoi(parts[2])
		if err != nil {
			return nil
		}
		return &portEntry{HostIP: parts[0], Published: parts[1], Target: target, Protocol: proto}
	}
	return nil
}

// SetServicePortBindings updates the ports list for a compose service.
func SetServicePortBindings(svc map[string]interface{}, bindings []PortBinding) {
	ports, _ := svc["ports"].([]interface{})

	bindingMap := map[string]PortBinding{}
	for _, b := range bindings {
		key := fmt.Sprintf("%d/%s", b.ContainerPort, b.Protocol)
		bindingMap[key] = b
	}

	seen := map[string]bool{}
	var nextPorts []interface{}

	for _, portRaw := range ports {
		portStr, ok := portRaw.(string)
		if !ok {
			nextPorts = append(nextPorts, portRaw)
			continue
		}
		parsed := parseComposePortString(portStr)
		if parsed == nil {
			nextPorts = append(nextPorts, portRaw)
			continue
		}
		key := fmt.Sprintf("%d/%s", parsed.Target, parsed.Protocol)
		binding, hasBind := bindingMap[key]
		if !hasBind {
			nextPorts = append(nextPorts, portRaw)
			continue
		}
		nextPorts = append(nextPorts, formatPortBinding(binding))
		seen[key] = true
	}

	for _, b := range bindings {
		key := fmt.Sprintf("%d/%s", b.ContainerPort, b.Protocol)
		if !seen[key] {
			nextPorts = append(nextPorts, formatPortBinding(b))
		}
	}

	svc["ports"] = nextPorts
}

// PortBinding is a host:container port binding.
type PortBinding struct {
	HostPort      *int
	ContainerPort int
	Protocol      string
}

func formatPortBinding(b PortBinding) string {
	proto := ""
	if b.Protocol != "tcp" {
		proto = "/" + b.Protocol
	}
	if b.HostPort == nil {
		return strconv.Itoa(b.ContainerPort) + proto
	}
	return strconv.Itoa(*b.HostPort) + ":" + strconv.Itoa(b.ContainerPort) + proto
}

// ParseDockerPortInput parses a Docker port string like "8080", "8080:8080", "53:53/udp".
type ParsedPort struct {
	HostPort          *int
	ContainerPort     int
	Protocol          string
	HasExplicitHost   bool
}

func ParseDockerPortInput(input string) *ParsedPort {
	input = strings.TrimSpace(input)
	if input == "" {
		return nil
	}

	proto := "tcp"
	if idx := strings.LastIndex(input, "/"); idx >= 0 {
		p := strings.ToLower(input[idx+1:])
		if p == "tcp" || p == "udp" {
			proto = p
			input = input[:idx]
		}
	}

	parts := strings.Split(input, ":")
	switch len(parts) {
	case 1:
		n, err := strconv.Atoi(parts[0])
		if err != nil || !isValidPort(n) {
			return nil
		}
		return &ParsedPort{ContainerPort: n, Protocol: proto}
	case 2:
		hostPort, err1 := strconv.Atoi(parts[0])
		containerPort, err2 := strconv.Atoi(parts[1])
		if err1 != nil || err2 != nil || !isValidPort(hostPort) || !isValidPort(containerPort) {
			return nil
		}
		return &ParsedPort{HostPort: &hostPort, ContainerPort: containerPort, Protocol: proto, HasExplicitHost: true}
	}
	return nil
}

// ApplyCurrentPublishedPorts syncs published port bindings from running containers into the compose doc.
func ApplyCurrentPublishedPorts(doc ComposeDoc, servicePorts map[string][]PortBinding) {
	services := getServices(doc)
	for svcName, bindings := range servicePorts {
		svcRaw, ok := services[svcName]
		if !ok {
			continue
		}
		svc, ok := svcRaw.(map[string]interface{})
		if !ok {
			continue
		}
		SetServicePortBindings(svc, bindings)
	}
}

func isValidPort(p int) bool {
	return p >= 1 && p <= 65535
}

func unixNow() int64 {
	return time.Now().Unix()
}
