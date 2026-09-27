// Package auth provides stateless Ed25519 token verification for Yantr.
//
// Token format:
//
//	base64(JSON{ publickey, signature, message, timestamp, nonce })
//	  - publickey: Ed25519 public key, hex-encoded (64 chars / 32 bytes)
//	  - message:   "{timestamp}:{nonce}"  <-- the signed bytes
//	  - signature: Ed25519 signature over raw message bytes, hex-encoded (128 chars / 64 bytes)
//	  - timestamp: Unix milliseconds (JS Date.now())
//	  - nonce:     random 16-byte hex string
//
// Only message is covered by the signature, so the timestamp and nonce are
// parsed back out of it rather than read from the sibling JSON fields — those
// are unauthenticated and a captured token could otherwise be replayed forever
// by rewriting timestamp. The client still sends them; they are redundant and
// ignored. A nonce is remembered for the token's lifetime, so the same signed
// message cannot be presented twice even inside the validity window.
//
// The server stores only the admin public key (no secret).
// Tokens are valid for 60 seconds (tokenTTL).
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
//	(Go stdlib), checks the signed timestamp is within 60s, and rejects a
//	reused nonce.
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
	"strconv"
	"strings"
	"sync"
	"time"

	"core/shared"
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
	// Atomic: a truncated auth.json is a truncated public key, which fails
	// validatePubKeyHex and makes the install permanently unbootable with no
	// recovery path other than deleting the file and re-running setup.
	if err := shared.WriteFileAtomic(authFilePath(), data, 0600); err != nil {
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

	// Decode and parse the token. This derives the timestamp from the signed
	// message rather than trusting the token's own timestamp field.
	tok, err := decodeToken(token)
	if err != nil {
		return nil, err
	}

	if err := checkTokenAge(tok.Timestamp); err != nil {
		return nil, err
	}

	// Verify the signature against the token's own public key.
	if err := verifyEd25519(tok.PublicKey, tok.Signature, tok.Message); err != nil {
		return nil, fmt.Errorf("invalid signature: %w", err)
	}

	// Good signature — burn the nonce so this exact token cannot be replayed.
	if !consumeNonce(tok.Nonce, tok.Timestamp+int64(tokenTTL/time.Millisecond)) {
		return nil, fmt.Errorf("token replayed")
	}

	// All good — save this public key as admin.
	return SaveAuthConfig(tok.PublicKey)
}

// authToken is the JSON structure sent by the frontend as a Bearer token
// (base64-encoded).
//
// Timestamp and Nonce are populated by decodeToken from the signed Message,
// never from the wire values. The client still sends them, but they are
// redundant with Message and therefore unauthenticated: a captured token could
// otherwise have its timestamp rewritten to Date.now() and be replayed forever,
// because the signature only ever covered Message. See parseSignedMessage.
type authToken struct {
	PublicKey string `json:"publickey"`
	Signature string `json:"signature"`
	Message   string `json:"message"`
	Timestamp int64  `json:"timestamp"`
	Nonce     string `json:"nonce"`
}

// tokenTTL is how long a signed message stays acceptable. It bounds both the
// clock-skew tolerance and how long a captured token can be replayed, and it is
// the same 60s window a nonce is retained for.
const tokenTTL = 60 * time.Second

// minNonceLen is the shortest nonce accepted in a signed message.
const minNonceLen = 8

// maxTrackedNonces caps the replay cache. Only nonces from tokens that already
// passed signature verification are ever recorded, so this is a bound on the
// legitimate client's request rate, not on attacker input. The UI mints a fresh
// token per API call, so a few thousand entries covers far more than 60s of
// polling from any realistic number of clients.
const maxTrackedNonces = 4096

var (
	seenNoncesMu sync.Mutex
	// seenNonces maps a nonce to the unix-milli deadline after which it can
	// never be presented again (its token has expired).
	seenNonces = make(map[string]int64)
)

// consumeNonce records nonce and reports whether it was unused.
//
// Returns false if the nonce was already seen, which means this exact signed
// message is being presented a second time — a replay. Entries are pruned on
// every call, so the map only ever holds nonces still inside the tokenTTL
// window; at the cap the entry closest to expiry is dropped, since it is the
// one that stops mattering soonest.
func consumeNonce(nonce string, expiresAtMs int64) bool {
	seenNoncesMu.Lock()
	defer seenNoncesMu.Unlock()

	now := time.Now().UnixMilli()
	for n, exp := range seenNonces {
		if exp <= now {
			delete(seenNonces, n)
		}
	}

	if _, replay := seenNonces[nonce]; replay {
		return false
	}

	if len(seenNonces) >= maxTrackedNonces {
		var oldest string
		var oldestExp int64
		for n, exp := range seenNonces {
			if oldest == "" || exp < oldestExp {
				oldest, oldestExp = n, exp
			}
		}
		delete(seenNonces, oldest)
	}

	seenNonces[nonce] = expiresAtMs
	return true
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
	// Bind Timestamp/Nonce to the signed bytes. Any value the client put in
	// those JSON fields is discarded here, so every downstream check reads
	// from the message the signature actually covers.
	ts, nonce, err := parseSignedMessage(tok.Message)
	if err != nil {
		return nil, err
	}
	tok.Timestamp = ts
	tok.Nonce = nonce
	return &tok, nil
}

// parseSignedMessage splits the signed message into its timestamp and nonce.
//
// The wire format is "{timestamp}:{nonce}" — the same string the client signs
// (ui/src/utils/crypto.js, createToken). Parsing it here is what makes the
// 60-second expiry enforceable: the timestamp that gets checked is covered by
// the signature, so it cannot be altered without invalidating it.
func parseSignedMessage(message string) (int64, string, error) {
	tsStr, nonce, found := strings.Cut(message, ":")
	if !found {
		return 0, "", fmt.Errorf("malformed token message")
	}
	ts, err := strconv.ParseInt(tsStr, 10, 64)
	if err != nil || ts <= 0 {
		return 0, "", fmt.Errorf("malformed token timestamp")
	}
	// A constant or absent nonce would make every token minted in the same
	// millisecond collide, so the second one would be rejected as a replay.
	// The client sends 16 random bytes (32 hex chars); this floor just keeps a
	// degenerate token from locking itself out.
	if len(nonce) < minNonceLen {
		return 0, "", fmt.Errorf("malformed token nonce")
	}
	return ts, nonce, nil
}

// VerifyToken verifies an Ed25519 auth token against the stored public key.
//
// The order matters. The timestamp and nonce are read from the signed message
// (see decodeToken), the signature is checked next, and only then is the nonce
// consumed — so an attacker without the private key can neither reach the nonce
// cache nor grow it.
func VerifyToken(token string, cfg *AuthConfig) error {
	tok, err := decodeToken(token)
	if err != nil {
		return err
	}

	// Check the public key matches the configured admin key.
	if strings.ToLower(tok.PublicKey) != strings.ToLower(cfg.PublicKeyHex) {
		return fmt.Errorf("unknown public key")
	}

	// Reject a message whose signed timestamp is outside the window. This is
	// only enforceable because the timestamp comes from the signed bytes.
	if err := checkTokenAge(tok.Timestamp); err != nil {
		return err
	}

	// Verify Ed25519 signature over the raw message.
	if err := verifyEd25519(tok.PublicKey, tok.Signature, tok.Message); err != nil {
		return fmt.Errorf("invalid signature: %w", err)
	}

	// The signature is good, so this token really was minted by the admin key.
	// Now refuse it if the very same message is being replayed.
	if !consumeNonce(tok.Nonce, tok.Timestamp+int64(tokenTTL/time.Millisecond)) {
		return fmt.Errorf("token replayed")
	}

	return nil
}

// checkTokenAge rejects a signed timestamp outside the tokenTTL window, in
// either direction — a token from the future is as much a clock-skew problem as
// an old one, and accepting it would widen the replay window.
func checkTokenAge(tsMs int64) error {
	skewMs := int64(tokenTTL / time.Millisecond)
	if abs64(time.Now().UnixMilli()-tsMs) > skewMs {
		return fmt.Errorf("token expired or clock skew too large")
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
