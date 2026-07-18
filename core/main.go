package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"sync"
	"syscall"
	"time"

	dockernet "github.com/docker/docker/api/types/network"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"core/apps"
	"core/auth"
	"core/caddy"
	"core/docker"
	"core/selfinstall"
	"core/shared"
	"core/telemetry"
)

// suppress unused import warning for dockernet (used in NetworkList)
var _ = dockernet.NetworkingConfig{}

const serverPort = 5252

// Timeout constants for spawnExec — sized by operation class.
const (
	spawnTimeoutShort  = 10 * time.Second // version probes, quick queries
	spawnTimeoutMedium = 10 * time.Minute // compose down, watchtower
	spawnTimeoutLong   = 30 * time.Minute // compose up (may pull large images)
)

// validAppID rejects any app ID that could be used for path traversal.
var validAppID = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]*$`)

var version = "0.0.0" // injected at build time

var publicPaths = map[string]bool{
	"/api/health":       true,
	"/api/version":      true,
	"/api/setup/status": true,
	"/api/setup/admin":  true,
	"/api/auth/login":   true,
}

// ─── Response helpers ─────────────────────────────────────────────────────────

func jsonResp(w http.ResponseWriter, status int, body interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func jsonErr(w http.ResponseWriter, status int, code, message string) {
	jsonResp(w, status, map[string]interface{}{
		"success": false,
		"code":    code,
		"message": message,
		"error":   message,
	})
}

func parseJSON(w http.ResponseWriter, r *http.Request, v interface{}) bool {
	if err := json.NewDecoder(r.Body).Decode(v); err != nil {
		jsonErr(w, http.StatusBadRequest, "INVALID_JSON", "Invalid JSON body: "+err.Error())
		return false
	}
	return true
}

func coalesce(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

// ─── Directory helpers ────────────────────────────────────────────────────────

func getAppsDir() string {
	if d := os.Getenv("YANTR_APPS_DIR"); d != "" {
		return d
	}
	exe, _ := os.Executable()
	return filepath.Join(filepath.Dir(exe), "apps")
}

func getDistDir() string {
	exe, _ := os.Executable()
	return filepath.Join(filepath.Dir(exe), "dist")
}

func getBaseAppID(projectID string) string {
	return shared.GetBaseAppID(projectID)
}

// spawnExec runs an external command with a context (timeout) and optional env/cwd.
// Always pass a context with a deadline — use the spawnTimeout* constants.
func spawnExec(ctx context.Context, name string, args []string, env map[string]string, cwd string) (string, string, int, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	if cwd != "" {
		cmd.Dir = cwd
	}
	if env != nil {
		var envList []string
		for k, v := range env {
			envList = append(envList, k+"="+v)
		}
		cmd.Env = envList
	}
	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	exitCode := 0
	if err != nil {
		if cmd.ProcessState != nil {
			exitCode = cmd.ProcessState.ExitCode()
		} else {
			exitCode = 1
		}
	}
	return stdout.String(), stderr.String(), exitCode, err
}

// ─── Compose command detection ────────────────────────────────────────────────

var (
	cachedComposeMu   sync.Mutex
	cachedComposeOnce sync.Once
	cachedComposeCmd  string
	cachedComposeArgs []string
	cachedComposeErr  error
)

func getComposeCommand() (string, []string, error) {
	cachedComposeOnce.Do(func() {
		env := map[string]string{"DOCKER_HOST": "unix://" + docker.SocketPath}
		ctx, cancel := context.WithTimeout(context.Background(), spawnTimeoutShort)
		defer cancel()
		if _, _, code, err := spawnExec(ctx, "docker", []string{"compose", "version"}, env, ""); err == nil && code == 0 {
			cachedComposeCmd = "docker"
			cachedComposeArgs = []string{"compose"}
			return
		}
		ctx2, cancel2 := context.WithTimeout(context.Background(), spawnTimeoutShort)
		defer cancel2()
		if _, _, code, err := spawnExec(ctx2, "docker-compose", []string{"version"}, env, ""); err == nil && code == 0 {
			cachedComposeCmd = "docker-compose"
			return
		}
		cachedComposeErr = fmt.Errorf("docker compose is not available")
	})
	return cachedComposeCmd, cachedComposeArgs, cachedComposeErr
}

// ─── Label parsing ────────────────────────────────────────────────────────────

type appLabelSet struct {
	App     string `json:"app"`
	Service string `json:"service"`
}

func parseAppLabels(labels map[string]string) appLabelSet {
	return appLabelSet{
		App:     labels["yantr.app"],
		Service: labels["yantr.app"],
	}
}

// ─── Catalog helpers ──────────────────────────────────────────────────────────

func entryStr(e *apps.App, field string) string {
	if e == nil {
		return ""
	}
	switch field {
	case "name":
		return e.Name
	case "logo":
		return e.Logo
	case "short_description":
		return e.ShortDescription
	case "description":
		return e.Description
	case "website":
		return e.Website
	}
	return ""
}

func entrySlice(e *apps.App, field string) interface{} {
	if e == nil {
		return []string{}
	}
	switch field {
	case "tags":
		if e.Tags != nil {
			return e.Tags
		}
		return []string{}
	case "usecases":
		if e.Usecases != nil {
			return e.Usecases
		}
		return []string{}
	}
	return []string{}
}

func entryPorts(e *apps.App) interface{} {
	if e == nil || e.Ports == nil {
		return []apps.PortInfo{}
	}
	return e.Ports
}

func getCatalogMap() map[string]*apps.App {
	cat, _ := apps.GetCatalogCached(false)
	m := map[string]*apps.App{}
	if cat == nil {
		return m
	}
	for i := range cat.Apps {
		m[cat.Apps[i].ID] = &cat.Apps[i]
	}
	return m
}

// ─── Middleware ───────────────────────────────────────────────────────────────

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func securityHeadersMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		w.Header().Set("Permissions-Policy", "camera=(), microphone=(), geolocation=()")
		next.ServeHTTP(w, r)
	})
}

func authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		isProtected := strings.HasPrefix(path, "/api/")
		if !isProtected || publicPaths[path] {
			next.ServeHTTP(w, r)
			return
		}

		cfg, err := auth.LoadAuthConfig(false)
		if err != nil {
			shared.Log("warn", "[auth] LoadAuthConfig error: "+err.Error())
			jsonErr(w, http.StatusServiceUnavailable, "SETUP_REQUIRED", "Setup required")
			return
		}

		token := auth.ExtractBearerToken(r.Header.Get("Authorization"))
		if token == "" {
			shared.Log("warn", "[auth] no token in request")
		}

		if cfg == nil {
			// No admin configured yet — treat the first valid self-signed token
			// as the bootstrap admin key and register it automatically.
			if token == "" {
				shared.Log("warn", "[auth] bootstrap: no token provided, rejecting")
				jsonErr(w, http.StatusUnauthorized, "UNAUTHORIZED", "Unauthorized")
				return
			}

			newCfg, bootstrapErr := auth.BootstrapFromToken(token)
			if bootstrapErr != nil {
				shared.Log("warn", "[auth] bootstrap failed: "+bootstrapErr.Error())
				jsonErr(w, http.StatusUnauthorized, "UNAUTHORIZED", "Unauthorized")
				return
			}
			shared.Log("info", "[auth] bootstrapped admin public key: "+newCfg.PublicKeyHex)
			next.ServeHTTP(w, r)
			return
		}

		if err := auth.VerifyToken(token, cfg); err != nil {
			shared.Log("warn", fmt.Sprintf("[auth] VerifyToken failed for pubkey %s: %v", cfg.PublicKeyHex, err))
			jsonErr(w, http.StatusUnauthorized, "UNAUTHORIZED", "Unauthorized")
			return
		}

		next.ServeHTTP(w, r)
	})
}

// ─── Browse proxy ─────────────────────────────────────────────────────────────

func browseProxyHandler(w http.ResponseWriter, r *http.Request) {
	parts := strings.SplitN(r.URL.Path, "/", 4)
	if len(parts) < 3 || parts[2] == "" {
		http.Error(w, "invalid browse path", http.StatusBadRequest)
		return
	}
	volumeName := parts[2]
	p := browserRegistry.GetPort(volumeName)
	if p == 0 {
		http.Error(w, fmt.Sprintf("Volume browser for %q is not running. Start it from the Volumes page.", volumeName), http.StatusServiceUnavailable)
		return
	}
	target, _ := url.Parse(fmt.Sprintf("http://localhost:%d", p))
	proxy := httputil.NewSingleHostReverseProxy(target)
	proxy.ServeHTTP(w, r)
}

// ─── Handlers: health / version / auth ───────────────────────────────────────

// ─── Handlers: apps ───────────────────────────────────────────────────────────

func handleAppLogo(w http.ResponseWriter, r *http.Request) {
	appID := chi.URLParam(r, "id")
	if !validAppID.MatchString(appID) {
		http.Error(w, "invalid app id", http.StatusBadRequest)
		return
	}
	svgPath := filepath.Join(getAppsDir(), appID, "logo.svg")
	if _, err := os.Stat(svgPath); err != nil {
		http.Error(w, "logo not found", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "image/svg+xml")
	w.Header().Set("Cache-Control", "public, max-age=3600")
	http.ServeFile(w, r, svgPath)
}

// ─── Handlers: containers ─────────────────────────────────────────────────────

// ─── Handlers: stacks ─────────────────────────────────────────────────────────

// ─── Handlers: images ─────────────────────────────────────────────────────────

// ─── Handlers: volumes ────────────────────────────────────────────────────────

// ─── Handlers: system ─────────────────────────────────────────────────────────

// ─── Handlers: proxy ──────────────────────────────────────────────────────────

// ─── Handlers: autoupdate ─────────────────────────────────────────────────────

// ─── Temporary-install reaper ─────────────────────────────────────────────────

// sweepExpiredContainers finds all running containers whose yantr.expireAt label
// is in the past and tears them down. Stacks are removed via `docker compose down`
// (which also cleans up networks/volumes). Standalone containers are stopped and
// removed directly. Called every minute from a background goroutine.

// ─── Main ─────────────────────────────────────────────────────────────────────

func main() {
	// Bootstrap self-install if needed
	if bootstrapped, err := selfinstall.RunIfNeeded(); err != nil {
		fmt.Fprintln(os.Stderr, "selfinstall error:", err)
		os.Exit(1)
	} else if bootstrapped {
		os.Exit(0)
	}

	// Configure apps directory
	appsPath := getAppsDir()
	apps.SetAppsDir(appsPath)
	shared.Log("info", "📂 Apps directory: "+appsPath)
	shared.Log("info", fmt.Sprintf("🏗️  Architecture: %s/%s", runtime.GOOS, runtime.GOARCH))

	telemetry.StartPresenceScheduler(version)

	// Router
	r := chi.NewRouter()
	r.Use(middleware.Recoverer)
	r.Use(corsMiddleware)
	r.Use(authMiddleware)
	r.Use(securityHeadersMiddleware)

	// Browse proxy
	r.HandleFunc("/browse/*", browseProxyHandler)

	// Public routes
	r.Get("/api/health", handleHealth)
	r.Get("/api/version", handleVersion)
	r.Get("/api/setup/status", handleSetupStatus)
	r.Post("/api/setup/admin", handleSetupAdmin)
	r.Post("/api/auth/login", handleLogin)

	// Protected routes
	r.Get("/api/logs", handleLogs)
	r.Get("/api/apps", handleApps)
	r.Get("/api/apps/{id}/logo", handleAppLogo)
	r.Get("/api/apps/{id}/check-arch", handleCheckArch)
	r.Post("/api/deploy", handleDeploy)
	r.Get("/api/containers", handleContainers)
	r.Get("/api/containers/{id}", handleContainerDetail)
	r.Get("/api/containers/{id}/stats", handleContainerStats)
	r.Get("/api/containers/{id}/logs", handleContainerLogs)
	r.Delete("/api/containers/{id}", handleContainerDelete)
	r.Post("/api/containers/{id}/start", handleContainerStart)
	r.Post("/api/containers/{id}/stop", handleContainerStop)
	r.Post("/api/containers/{id}/restart", handleContainerRestart)
	r.Get("/api/stacks/{projectId}", handleStackDetail)
	r.Delete("/api/stacks/{projectId}", handleStackDelete)
	r.Post("/api/stacks/{projectId}/restart", handleStackRestart)
	r.Get("/api/images", handleImages)
	r.Get("/api/image-details/{id}", handleImageDetails)
	r.Delete("/api/images/{id}", handleImageDelete)
	r.Get("/api/volumes", handleVolumes)
	r.Get("/api/volumes/browsers", handleVolumeBrowserList)
	r.Post("/api/volumes/{name}/browse", handleVolumeBrowseStart)
	r.Delete("/api/volumes/{name}/browse", handleVolumeBrowseStop)
	r.Delete("/api/volumes/{name}", handleVolumeDelete)
	r.Get("/api/system/info", handleSystemInfo)
	r.Post("/api/system/prune", handleSystemPrune)
	r.Get("/api/ports/used", handlePortsUsed)
	r.Post("/api/ports/suggest", handlePortsSuggest)
	r.Get("/api/network/identity", handleNetworkIdentity)
	r.Get("/api/proxy", handleProxyList)
	r.Post("/api/proxy/reload", handleProxyReload)
	r.Post("/api/autoupdate/run", handleAutoupdateRun)

	// SPA static serving (production)
	distDir := getDistDir()
	uiBase := coalesce(os.Getenv("UI_BASE_PATH"), "/")
	if _, err := os.Stat(distDir); err == nil {
		shared.Log("info", "📦 Serving Vue.js UI from: "+distDir)
		fs := http.FileServer(http.Dir(distDir))
		prefix := strings.TrimSuffix(uiBase, "/")
		r.Handle(prefix+"/*", http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			path := strings.TrimPrefix(req.URL.Path, prefix)
			if path == "" {
				path = "/"
			}
			if _, err := os.Stat(filepath.Join(distDir, path)); os.IsNotExist(err) {
				http.ServeFile(w, req, filepath.Join(distDir, "index.html"))
				return
			}
			http.StripPrefix(prefix, fs).ServeHTTP(w, req)
		}))
		r.Get("/", func(w http.ResponseWriter, req *http.Request) {
			http.ServeFile(w, req, filepath.Join(distDir, "index.html"))
		})
	}

	// Start Caddy in background
	go func() {
		shared.Log("info", "🔒 Starting embedded Caddy proxy")
		if err := caddy.StartCaddy(); err != nil {
			shared.Log("warn", "⚠️  [CADDY] "+err.Error())
		}
	}()

	// Start temporary-install reaper (checks every minute)
	go func() {
		shared.Log("info", "🧹 Starting temporary-install reaper (1 min interval)")
		ticker := time.NewTicker(1 * time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			sweepExpiredContainers()
		}
	}()

	// Server startup log
	shared.Log("info", strings.Repeat("=", 50))
	shared.Log("info", "🚀 Yantr Core Server Started (Go)")
	shared.Log("info", strings.Repeat("=", 50))
	shared.Log("info", fmt.Sprintf("📡 Port: %d", serverPort))
	shared.Log("info", fmt.Sprintf("🔌 Socket: %s", docker.SocketPath))
	shared.Log("info", fmt.Sprintf("🌐 Access: http://localhost:%d", serverPort))
	shared.Log("info", strings.Repeat("=", 50))

	srv := &http.Server{
		Addr:         fmt.Sprintf("0.0.0.0:%d", serverPort),
		Handler:      r,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 120 * time.Second,
	}

	// Graceful shutdown
	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGTERM, syscall.SIGINT)
		sig := <-sigCh
		shared.Log("info", fmt.Sprintf("Received %s, shutting down...", sig))
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = srv.Shutdown(ctx)
		caddy.StopCaddy()
		browserRegistry.StopAll()
		os.Exit(0)
	}()

	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		fmt.Fprintln(os.Stderr, "Server error:", err)
		os.Exit(1)
	}
}
