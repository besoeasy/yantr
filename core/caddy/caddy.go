// Package caddy manages the embedded Caddy subprocess and its Caddyfile configuration.
//
// Caddy runs as a child process inside the Yantr container. Its config is
// derived entirely from Docker container labels — no database, no state file.
//
// Label schema (written by InjectCaddyAuthLabels at deploy time):
//   yantr.caddy.enabled      = "true"
//   yantr.caddy.serve.port   = "<port Caddy auth proxy listens on>"
//   yantr.caddy.target.port  = "<app's native container port>"
//   yantr.caddy.auth.user    = "<username>"
//   yantr.caddy.auth.pass    = "<bcrypt hash, hex-encoded>"  (password discarded after hashing)
//
// x-auth compose block (read at deploy time, stripped before container runs):
//   x-auth:
//     port: 3002       # Caddy listens here
//     username: admin
//     password: secret # plain text — bcrypted, then discarded
package caddy

import (
	"bytes"
	"context"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	dockertypes "github.com/docker/docker/api/types/container"
	dockerfilters "github.com/docker/docker/api/types/filters"
	"core/docker"
	"core/shared"
)

const adminPort = 2019

var (
	mu           sync.Mutex
	caddyCmd     *exec.Cmd
	caddyRunning bool
)

// ProxyRoute describes a single Caddy reverse-proxy rule.
type ProxyRoute struct {
	ContainerName string `json:"containerName"`
	ContainerID   string `json:"containerId"`
	ProjectID     string `json:"projectId,omitempty"`
	ServePort     int    `json:"servePort"`
	ContainerPort int    `json:"containerPort"`
	TargetPort    int    `json:"targetPort"`
	AuthUser      string `json:"authUser,omitempty"`
	AuthPassHash  string `json:"-"` // raw bcrypt, never sent to frontend
}

// XAuth holds the parsed x-auth block from a compose.yml.
type XAuth struct {
	Port     int    `yaml:"port"`
	Username string `yaml:"username"`
	Password string `yaml:"password"`
}

// IsRunning reports whether the Caddy subprocess is alive.
func IsRunning() bool {
	mu.Lock()
	defer mu.Unlock()
	return caddyRunning
}

// StartCaddy spawns the caddy binary and waits for its admin API.
func StartCaddy() error {
	mu.Lock()
	defer mu.Unlock()

	if caddyRunning {
		return nil
	}

	cmd := exec.Command("caddy", "run")
	if err := cmd.Start(); err != nil {
		if strings.Contains(err.Error(), "executable file not found") {
			shared.Log("warn", "⚠️  caddy binary not found — embedded proxy unavailable")
			return nil
		}
		return fmt.Errorf("failed to start caddy: %w", err)
	}
	caddyCmd = cmd
	caddyRunning = true

	go func() {
		_ = cmd.Wait()
		mu.Lock()
		caddyRunning = false
		caddyCmd = nil
		mu.Unlock()
		shared.Log("info", "[caddy] process exited")
	}()

	// Wait for admin API
	if !waitForAdminAPI(20, 300*time.Millisecond) {
		shared.Log("warn", "⚠️  Caddy admin API did not become ready in time")
		return nil
	}

	shared.Log("info", "🔒 Caddy started (admin API :2019)")
	if err := ReloadCaddyConfig(); err != nil {
		shared.Log("warn", "[caddy] initial config push failed: "+err.Error())
	}
	return nil
}

// StopCaddy kills the Caddy subprocess.
func StopCaddy() {
	mu.Lock()
	defer mu.Unlock()
	if caddyCmd != nil {
		_ = caddyCmd.Process.Kill()
		caddyCmd = nil
	}
	caddyRunning = false
}

// ReloadCaddyConfig rebuilds the Caddyfile from container labels and pushes it.
func ReloadCaddyConfig() error {
	proxies, err := GetCaddyProxies()
	if err != nil {
		return err
	}
	if len(proxies) == 0 {
		shared.Log("info", "[caddy] No proxy routes configured")
		return nil
	}
	caddyfile := buildCaddyfile(proxies)
	return pushCaddyfile(caddyfile, len(proxies))
}

// GetCaddyProxies scans running containers for yantr.caddy.* labels.
func GetCaddyProxies() ([]ProxyRoute, error) {
	filters := dockerfilters.NewArgs()
	containers, err := docker.Client.ContainerList(context.Background(), dockertypes.ListOptions{
		Filters: filters,
	})
	if err != nil {
		return nil, err
	}

	var proxies []ProxyRoute
	for _, c := range containers {
		labels := c.Labels
		if labels["yantr.caddy.enabled"] != "true" {
			continue
		}
		servePort, err := strconv.Atoi(labels["yantr.caddy.serve.port"])
		if err != nil || servePort == 0 {
			continue
		}
		containerPort, err := strconv.Atoi(labels["yantr.caddy.target.port"])
		if err != nil || containerPort == 0 {
			continue
		}

		// Resolve the actual host port Docker assigned
		targetPort := 0
		for _, pb := range c.Ports {
			if int(pb.PrivatePort) == containerPort && pb.PublicPort > 0 {
				targetPort = int(pb.PublicPort)
				break
			}
		}
		if targetPort == 0 {
			containerName := ""
			if len(c.Names) > 0 {
				containerName = strings.TrimPrefix(c.Names[0], "/")
			}
			shared.Log("warn", fmt.Sprintf("[caddy] No host port for container port %d on %s — skipping", containerPort, containerName))
			continue
		}

		name := ""
		if len(c.Names) > 0 {
			name = strings.TrimPrefix(c.Names[0], "/")
		}

		passHash := normalizeStoredHash(labels["yantr.caddy.auth.pass"])

		proxies = append(proxies, ProxyRoute{
			ContainerName: name,
			ContainerID:   c.ID,
			ProjectID:     labels["com.docker.compose.project"],
			ServePort:     servePort,
			ContainerPort: containerPort,
			TargetPort:    targetPort,
			AuthUser:      labels["yantr.caddy.auth.user"],
			AuthPassHash:  passHash,
		})
	}
	return proxies, nil
}

// ─── x-auth deploy-time wiring ────────────────────────────────────────────────

// ParseXAuth extracts the x-auth block from a parsed compose doc.
// Returns nil if the block is absent or incomplete.
func ParseXAuth(doc map[string]interface{}) *XAuth {
	raw, ok := doc["x-auth"]
	if !ok || raw == nil {
		return nil
	}
	m, ok := raw.(map[string]interface{})
	if !ok {
		return nil
	}

	port := 0
	switch v := m["port"].(type) {
	case int:
		port = v
	case float64:
		port = int(v)
	}

	username, _ := m["username"].(string)
	password, _ := m["password"].(string)

	if port == 0 || username == "" || password == "" {
		return nil
	}
	return &XAuth{Port: port, Username: username, Password: password}
}

// InjectCaddyAuthLabels hashes the x-auth password and injects yantr.caddy.*
// labels into every service in the compose doc. Returns the target app port
// (from yantr.port.N labels) and any error.
//
// The x-auth block is left in the doc (Docker Compose ignores x-* fields).
// The plain-text password is never written to the container — only the
// hex-encoded bcrypt hash reaches the label store.
func InjectCaddyAuthLabels(doc map[string]interface{}, auth *XAuth) error {
	if auth == nil {
		return nil
	}

	// Hash the plain-text password via the caddy binary
	hash, err := HashPassword(auth.Password)
	if err != nil {
		return fmt.Errorf("x-auth: failed to hash password: %w", err)
	}
	encodedHash := EncodeHash(hash)

	services, ok := doc["services"].(map[string]interface{})
	if !ok || len(services) == 0 {
		return fmt.Errorf("x-auth: compose has no services")
	}

	for _, svcRaw := range services {
		svc, ok := svcRaw.(map[string]interface{})
		if !ok {
			continue
		}

		// Find the app port from existing yantr.port.N labels
		appPort := resolveAppPort(svc)

		// Ensure labels map exists
		labels, ok := svc["labels"].(map[string]interface{})
		if !ok {
			labels = map[string]interface{}{}
			svc["labels"] = labels
		}

		labels["yantr.caddy.enabled"] = "true"
		labels["yantr.caddy.serve.port"] = strconv.Itoa(auth.Port)
		labels["yantr.caddy.target.port"] = strconv.Itoa(appPort)
		labels["yantr.caddy.auth.user"] = auth.Username
		labels["yantr.caddy.auth.pass"] = encodedHash
		// Plain password is intentionally NOT stored in any label
	}

	shared.Log("info", fmt.Sprintf("[caddy] x-auth: injected auth labels → serve :%d, target :%d, user %s",
		auth.Port, resolveAppPortFromDoc(doc), auth.Username))
	return nil
}

// resolveAppPort finds the first yantr.port.N label value port number from a service.
func resolveAppPort(svc map[string]interface{}) int {
	labels, ok := svc["labels"].(map[string]interface{})
	if !ok {
		return 0
	}
	for key := range labels {
		if strings.HasPrefix(key, "yantr.port.") {
			portStr := strings.TrimPrefix(key, "yantr.port.")
			if n, err := strconv.Atoi(portStr); err == nil {
				return n
			}
		}
	}
	return 0
}

func resolveAppPortFromDoc(doc map[string]interface{}) int {
	services, ok := doc["services"].(map[string]interface{})
	if !ok {
		return 0
	}
	for _, svcRaw := range services {
		svc, ok := svcRaw.(map[string]interface{})
		if !ok {
			continue
		}
		if p := resolveAppPort(svc); p != 0 {
			return p
		}
	}
	return 0
}

// ─── Password hashing ─────────────────────────────────────────────────────────

// HashPassword calls `caddy hash-password` to produce a bcrypt hash.
func HashPassword(plaintext string) (string, error) {
	out, err := exec.Command("caddy", "hash-password", "--plaintext", plaintext).Output()
	if err != nil {
		return "", fmt.Errorf("caddy hash-password failed: %w", err)
	}
	return strings.TrimSpace(string(out)), nil
}

// EncodeHash hex-encodes a bcrypt hash for safe storage in Docker labels.
func EncodeHash(bcryptHash string) string {
	return hex.EncodeToString([]byte(bcryptHash))
}

// normalizeStoredHash decodes a hex-encoded bcrypt hash from a Docker label.
func normalizeStoredHash(value string) string {
	if value == "" {
		return ""
	}
	b, err := hex.DecodeString(strings.TrimSpace(value))
	if err != nil {
		return ""
	}
	return string(b)
}

// ─── Caddyfile generation ─────────────────────────────────────────────────────

var bcryptRe = regexp.MustCompile(`^\$2[aby]\$\d{2}\$[./A-Za-z0-9]{53}$`)

func isValidBcryptHash(value string) bool {
	return bcryptRe.MatchString(value)
}

func buildCaddyfile(proxies []ProxyRoute) string {
	var sb strings.Builder
	for i, p := range proxies {
		if i > 0 {
			sb.WriteString("\n\n")
		}
		if p.AuthUser != "" && p.AuthPassHash != "" && !isValidBcryptHash(p.AuthPassHash) {
			shared.Log("warn", fmt.Sprintf("[caddy] Corrupted bcrypt hash for proxy :%d — blocking access", p.ServePort))
			sb.WriteString(fmt.Sprintf(":%d {\n    respond \"Auth proxy misconfigured. Disable and re-enable in Yantr.\" 503\n}", p.ServePort))
			continue
		}

		sb.WriteString(fmt.Sprintf(":%d {\n", p.ServePort))
		if p.AuthUser != "" && p.AuthPassHash != "" {
			sb.WriteString(fmt.Sprintf("    basic_auth * {\n        %s %s\n    }\n", p.AuthUser, p.AuthPassHash))
		}
		sb.WriteString(fmt.Sprintf("    reverse_proxy localhost:%d\n}", p.TargetPort))
	}
	return sb.String()
}

func pushCaddyfile(caddyfile string, count int) error {
	url := fmt.Sprintf("http://127.0.0.1:%d/load", adminPort)
	resp, err := http.Post(url, "text/caddyfile", bytes.NewBufferString(caddyfile))
	if err != nil {
		return fmt.Errorf("[caddy] reload request failed: %w", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 300 {
		return fmt.Errorf("[caddy] config load failed (%d): %s", resp.StatusCode, string(body))
	}
	shared.Log("info", fmt.Sprintf("🔒 Caddy reloaded: %d active proxy route(s)", count))
	return nil
}

func waitForAdminAPI(retries int, delay time.Duration) bool {
	url := fmt.Sprintf("http://127.0.0.1:%d/config/", adminPort)
	for i := 0; i < retries; i++ {
		resp, err := http.Get(url)
		if err == nil {
			resp.Body.Close()
			return true
		}
		time.Sleep(delay)
	}
	return false
}
