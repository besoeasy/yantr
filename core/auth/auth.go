// Package auth provides stateless token verification for Yantr.
//
// Token format (replaces daku):
//
//	A JWT-like structure: base64url(header).base64url(payload).base64url(signature)
//	- Header: {"alg":"HS256","typ":"JWT"}
//	- Payload: {"sub":"<username>","iat":<unix>,"exp":<unix>}
//	- Signature: HMAC-SHA256(header.payload, secret)
//
// The secret is a 32-byte random value stored in /data/auth.json.
// The frontend generates tokens using browser SubtleCrypto (importKey + sign with
// the same HS256/HMAC-SHA256 algorithm), so no custom crypto library is needed.
//
// Setup flow:
//
//	POST /api/setup/admin  { username, secret (hex-encoded 32-byte key) }
//	→ saves to /data/auth.json
//
// Login flow:
//
//	The client signs { sub: username, iat, exp } with its stored secret key,
//	sends as Bearer token. Server verifies HMAC and timestamp.
package auth

import (
	"crypto/hmac"
	"crypto/sha256"
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
	Username  string `json:"username"`
	SecretHex string `json:"secretHex"` // 32-byte HMAC key, hex-encoded
	CreatedAt string `json:"createdAt,omitempty"`
}

// ErrAlreadyConfigured is returned by SaveAuthConfig when an admin already exists.
var ErrAlreadyConfigured = errors.New("admin is already configured")

var (
	mu           sync.RWMutex
	setupMu      sync.Mutex // serialises the check-then-write in SaveAuthConfig
	cachedConfig *AuthConfig
	memoryConfig *AuthConfig // set after first successful setup in same process
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

// LoadAuthConfig returns the current auth configuration (env → memory → file).
func LoadAuthConfig(forceRefresh bool) (*AuthConfig, error) {
	// 1. Env var override
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
	secretHex := os.Getenv("YANTR_AUTH_SECRET")
	if secretHex == "" {
		secretHex = os.Getenv("YANTR_SECRET")
	}
	if len(secretHex) != 64 { // 32 bytes = 64 hex chars
		return nil
	}
	if _, err := hex.DecodeString(secretHex); err != nil {
		return nil
	}
	username := os.Getenv("YANTR_AUTH_USERNAME")
	if username == "" {
		username = "admin"
	}
	return &AuthConfig{
		Username:  username,
		SecretHex: strings.ToLower(secretHex),
	}
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
	if len(cfg.SecretHex) != 64 {
		return nil, fmt.Errorf("invalid secretHex length in auth.json")
	}
	if _, err := hex.DecodeString(cfg.SecretHex); err != nil {
		return nil, fmt.Errorf("invalid secretHex in auth.json: %w", err)
	}

	mu.Lock()
	cachedConfig = &cfg
	mu.Unlock()

	return &cfg, nil
}

// SaveAuthConfig persists a new auth configuration.
// It is safe for concurrent callers: the existence check and the write happen
// inside a single mutex so two simultaneous requests cannot both succeed.
func SaveAuthConfig(username, secretHex string) (*AuthConfig, error) {
	setupMu.Lock()
	defer setupMu.Unlock()

	if readEnvAuthConfig() != nil {
		return nil, fmt.Errorf("auth is managed by environment variable")
	}
	// Authoritative check inside the lock.
	if existing, _ := readAuthFile(false); existing != nil {
		return nil, ErrAlreadyConfigured
	}
	mu.RLock()
	alreadyInMem := memoryConfig != nil
	mu.RUnlock()
	if alreadyInMem {
		return nil, ErrAlreadyConfigured
	}

	username = strings.TrimSpace(username)
	secretHex = strings.ToLower(strings.TrimSpace(secretHex))

	if username == "" {
		return nil, fmt.Errorf("username is required")
	}
	if len(secretHex) != 64 {
		return nil, fmt.Errorf("secretHex must be a 64-character hex string (32 bytes)")
	}
	if _, err := hex.DecodeString(secretHex); err != nil {
		return nil, fmt.Errorf("secretHex is not valid hex: %w", err)
	}

	cfg := &AuthConfig{
		Username:  username,
		SecretHex: secretHex,
		CreatedAt: time.Now().UTC().Format(time.RFC3339),
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

// jwtHeader is the standard HS256 header, base64url-encoded (no padding).
var jwtHeader = base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"HS256","typ":"JWT"}`))

// VerifyToken verifies a HS256 JWT-style token against the stored secret.
// Returns the username on success.
func VerifyToken(token string, cfg *AuthConfig) (string, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return "", fmt.Errorf("invalid token format")
	}

	header, payloadB64, sigB64 := parts[0], parts[1], parts[2]
	if header != jwtHeader {
		return "", fmt.Errorf("invalid token header")
	}

	// Verify signature
	key, err := hex.DecodeString(cfg.SecretHex)
	if err != nil {
		return "", fmt.Errorf("server auth misconfigured")
	}

	mac := hmac.New(sha256.New, key)
	mac.Write([]byte(header + "." + payloadB64))
	expectedSig := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))

	if !hmac.Equal([]byte(sigB64), []byte(expectedSig)) {
		return "", fmt.Errorf("invalid token signature")
	}

	// Decode payload
	payloadData, err := base64.RawURLEncoding.DecodeString(payloadB64)
	if err != nil {
		return "", fmt.Errorf("invalid token payload encoding")
	}

	var payload struct {
		Sub string `json:"sub"`
		Exp int64  `json:"exp"`
		Iat int64  `json:"iat"`
	}
	if err := json.Unmarshal(payloadData, &payload); err != nil {
		return "", fmt.Errorf("invalid token payload")
	}

	now := time.Now().Unix()
	if payload.Exp > 0 && now > payload.Exp {
		return "", fmt.Errorf("token expired")
	}
	if payload.Iat > now+60 { // allow 60s clock skew
		return "", fmt.Errorf("token issued in the future")
	}

	return payload.Sub, nil
}

// ExtractBearerToken extracts the token from the Authorization header.
func ExtractBearerToken(authHeader string) string {
	authHeader = strings.TrimSpace(authHeader)
	if !strings.HasPrefix(strings.ToLower(authHeader), "bearer ") {
		return ""
	}
	return strings.TrimSpace(authHeader[7:])
}
