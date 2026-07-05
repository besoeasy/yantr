// Package auth provides stateless Ed25519 token verification for Yantr.
//
// Token format:
//
//	base64(JSON{ publickey, signature, message, timestamp, nonce })
//	  - publickey: Ed25519 public key, hex-encoded (64 chars / 32 bytes)
//	  - message:   "{timestamp}:{nonce}"
//	  - signature: Ed25519 signature over raw message bytes, hex-encoded (128 chars / 64 bytes)
//	  - timestamp: Unix milliseconds (JS Date.now())
//	  - nonce:     random 16-byte hex string
//
// The server stores only the admin public key (no secret).
// Tokens are valid for 1 minute (timestamp checked server-side).
//
// Bootstrap flow (no admin configured):
//
//	If neither YANTR_ADMIN_PUBLIC_KEY env var nor /data/auth.json exist,
//	the first request that presents a valid Ed25519 self-signed token
//	automatically becomes the admin. No explicit setup step required.
//
// Configuration:
//
//	Env var:   YANTR_ADMIN_PUBLIC_KEY (64-char hex, Ed25519 public key)
//	File:      /data/auth.json         { "publicKeyHex": "..." }
//
// Login flow:
//
//	Client signs { message: "timestamp:nonce" } with Ed25519 private key,
//	sends base64 JSON as Bearer token. Server verifies with crypto/ed25519
//	(Go stdlib) and checks timestamp is within 1 minute.
package auth

import (
	"crypto/ed25519"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// AuthConfig holds the persisted auth configuration.
type AuthConfig struct {
	PublicKeyHex string `json:"publicKeyHex"` // Ed25519 public key, 64 hex chars (32 bytes)
	CreatedAt    string `json:"createdAt,omitempty"`
}

// ErrAlreadyConfigured is returned by SaveAuthConfig when an admin already exists.
var ErrAlreadyConfigured = errors.New("admin is already configured")

var (
	mu           sync.RWMutex
	setupMu      sync.Mutex
	cachedConfig *AuthConfig
	memoryConfig *AuthConfig
)

var dataDir = func() string {
	if d := os.Getenv("YANTR_DATA_DIR"); d != "" {
		return d
	}
	return "/data"
}()

func authFilePath() string {
	return filepath.Join(dataDir, "auth.json")
}

// validatePubKeyHex checks that a string is a valid 64-char hex Ed25519 public key.
func validatePubKeyHex(s string) error {
	if len(s) != 64 {
		return fmt.Errorf("Ed25519 public key must be 64 hex chars (32 bytes), got %d", len(s))
	}
	b, err := hex.DecodeString(s)
	if err != nil {
		return fmt.Errorf("public key is not valid hex: %w", err)
	}
	if len(b) != ed25519.PublicKeySize {
		return fmt.Errorf("public key wrong size: %d", len(b))
	}
	return nil
}

// LoadAuthConfig returns the current auth configuration (env → memory → file).
func LoadAuthConfig(forceRefresh bool) (*AuthConfig, error) {
	if cfg := readEnvAuthConfig(); cfg != nil {
		return cfg, nil
	}

	mu.RLock()
	if memoryConfig != nil {
		cfg := *memoryConfig
		mu.RUnlock()
		return &cfg, nil
	}
	if !forceRefresh && cachedConfig != nil {
		cfg := *cachedConfig
		mu.RUnlock()
		return &cfg, nil
	}
	mu.RUnlock()

	return readAuthFile(forceRefresh)
}

func readEnvAuthConfig() *AuthConfig {
	pubKeyHex := os.Getenv("YANTR_ADMIN_PUBLIC_KEY")
	if pubKeyHex == "" {
		return nil
	}
	pubKeyHex = strings.ToLower(strings.TrimSpace(pubKeyHex))
	if err := validatePubKeyHex(pubKeyHex); err != nil {
		return nil
	}
	return &AuthConfig{PublicKeyHex: pubKeyHex}
}

func readAuthFile(forceRefresh bool) (*AuthConfig, error) {
	data, err := os.ReadFile(authFilePath())
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}

	var cfg AuthConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	cfg.PublicKeyHex = strings.ToLower(strings.TrimSpace(cfg.PublicKeyHex))
	if err := validatePubKeyHex(cfg.PublicKeyHex); err != nil {
		return nil, fmt.Errorf("invalid publicKeyHex in auth.json: %w", err)
	}

	mu.Lock()
	cachedConfig = &cfg
	mu.Unlock()

	return &cfg, nil
}

// SaveAuthConfig persists a new auth configuration.
func SaveAuthConfig(publicKeyHex string) (*AuthConfig, error) {
	setupMu.Lock()
	defer setupMu.Unlock()

	if readEnvAuthConfig() != nil {
		return nil, fmt.Errorf("auth is managed by environment variable")
	}
	if existing, _ := readAuthFile(false); existing != nil {
		return nil, ErrAlreadyConfigured
	}
	mu.RLock()
	alreadyInMem := memoryConfig != nil
	mu.RUnlock()
	if alreadyInMem {
		return nil, ErrAlreadyConfigured
	}

	publicKeyHex = strings.ToLower(strings.TrimSpace(publicKeyHex))
	if err := validatePubKeyHex(publicKeyHex); err != nil {
		return nil, err
	}

	cfg := &AuthConfig{
		PublicKeyHex: publicKeyHex,
		CreatedAt:    time.Now().UTC().Format(time.RFC3339),
	}

	data, err := json.Marshal(cfg)
	if err != nil {
		return nil, err
	}

	if err := os.MkdirAll(filepath.Dir(authFilePath()), 0755); err != nil {
		return nil, err
	}
	if err := os.WriteFile(authFilePath(), data, 0600); err != nil {
		return nil, err
	}

	mu.Lock()
	memoryConfig = cfg
	cachedConfig = cfg
	mu.Unlock()

	return cfg, nil
}

// BootstrapFromToken auto-registers the first public key as admin when no
// admin is configured yet. It verifies the token is a valid self-signed
// Ed25519 token (proving the caller controls the key), then saves the
// public key from the token as the admin.
func BootstrapFromToken(token string) (*AuthConfig, error) {
	// Reject if already configured (env or file).
	if readEnvAuthConfig() != nil {
		return nil, fmt.Errorf("auth is managed by environment variable")
	}
	if existing, _ := readAuthFile(false); existing != nil {
		return nil, ErrAlreadyConfigured
	}
	mu.RLock()
	alreadyInMem := memoryConfig != nil
	mu.RUnlock()
	if alreadyInMem {
		return nil, ErrAlreadyConfigured
	}

	// Decode and parse the token.
	tok, err := decodeToken(token)
	if err != nil {
		return nil, err
	}

	// Validate timestamp.
	nowMs := time.Now().UnixMilli()
	if tok.Timestamp <= 0 || abs64(nowMs-tok.Timestamp) > int64(60*1000) {
		return nil, fmt.Errorf("token expired")
	}

	// Verify the signature against the token's own public key.
	if err := verifyEd25519(tok.PublicKey, tok.Signature, tok.Message); err != nil {
		return nil, fmt.Errorf("invalid signature: %w", err)
	}

	// All good — save this public key as admin.
	return SaveAuthConfig(tok.PublicKey)
}

// authToken is the JSON structure sent by the frontend as a Bearer token (base64-encoded).
type authToken struct {
	PublicKey string `json:"publickey"`
	Signature string `json:"signature"`
	Message   string `json:"message"`
	Timestamp int64  `json:"timestamp"`
	Nonce     string `json:"nonce"`
}

func decodeToken(token string) (*authToken, error) {
	decoded, err := base64.StdEncoding.DecodeString(token)
	if err != nil {
		decoded, err = base64.RawStdEncoding.DecodeString(token)
		if err != nil {
			return nil, fmt.Errorf("invalid token encoding")
		}
	}
	var tok authToken
	if err := json.Unmarshal(decoded, &tok); err != nil {
		return nil, fmt.Errorf("invalid token format")
	}
	if tok.PublicKey == "" || tok.Signature == "" || tok.Message == "" {
		return nil, fmt.Errorf("missing token fields")
	}
	return &tok, nil
}

// VerifyToken verifies an Ed25519 auth token against the stored public key.
func VerifyToken(token string, cfg *AuthConfig) error {
	tok, err := decodeToken(token)
	if err != nil {
		return err
	}

	// Check the public key matches the configured admin key.
	if strings.ToLower(tok.PublicKey) != strings.ToLower(cfg.PublicKeyHex) {
		return fmt.Errorf("unknown public key")
	}

	// Check timestamp (milliseconds from JS Date.now()).
	nowMs := time.Now().UnixMilli()
	if tok.Timestamp <= 0 || abs64(nowMs-tok.Timestamp) > int64(60*1000) {
		return fmt.Errorf("token expired or clock skew too large")
	}

	// Verify Ed25519 signature over the raw message.
	if err := verifyEd25519(tok.PublicKey, tok.Signature, tok.Message); err != nil {
		return fmt.Errorf("invalid signature: %w", err)
	}

	return nil
}

func abs64(x int64) int64 {
	if x < 0 {
		return -x
	}
	return x
}

// verifyEd25519 verifies an Ed25519 signature over the raw message string.
// Uses Go stdlib crypto/ed25519 — no external dependencies.
func verifyEd25519(pubKeyHex, sigHex, message string) error {
	pubBytes, err := hex.DecodeString(pubKeyHex)
	if err != nil || len(pubBytes) != ed25519.PublicKeySize {
		return fmt.Errorf("invalid public key")
	}
	sigBytes, err := hex.DecodeString(sigHex)
	if err != nil || len(sigBytes) != ed25519.SignatureSize {
		return fmt.Errorf("invalid signature (expected 64 bytes)")
	}
	if !ed25519.Verify(ed25519.PublicKey(pubBytes), []byte(message), sigBytes) {
		return fmt.Errorf("signature verification failed")
	}
	return nil
}

// ExtractBearerToken extracts the token from the Authorization header.
func ExtractBearerToken(authHeader string) string {
	authHeader = strings.TrimSpace(authHeader)
	if !strings.HasPrefix(strings.ToLower(authHeader), "bearer ") {
		return ""
	}
	return strings.TrimSpace(authHeader[7:])
}
