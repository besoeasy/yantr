// Package auth provides stateless secp256k1 token verification for Yantr.
//
// Token format:
//
//	base64(JSON{ publickey, signature, message, timestamp, nonce })
//	  - publickey: compressed secp256k1 public key, hex-encoded (66 chars)
//	  - message:   "{timestamp}:{nonce}"
//	  - signature: secp256k1 compact signature over sha256(message), hex-encoded
//	  - timestamp: Unix milliseconds (JS Date.now())
//	  - nonce:     random 16-byte hex string
//
// The server stores only the admin public key (no secret).
// Tokens are valid for 1 minute (timestamp checked server-side).
//
// Bootstrap flow (no admin configured):
//
//	If neither YANTR_ADMIN_PUBLIC_KEY env var nor /data/auth.json exist,
//	the first request that presents a valid secp256k1 self-signed token
//	automatically becomes the admin. No explicit setup step required.
//
// Configuration:
//
//	Env var:   YANTR_ADMIN_PUBLIC_KEY (66-char hex, compressed secp256k1)
//	File:      /data/auth.json         { "publicKeyHex": "..." }
//
// Setup flow:
//
//	POST /api/setup/admin  { publicKeyHex }
//	→ saves to /data/auth.json
//
// Login flow:
//
//	Client signs { message: "timestamp:nonce" } with private key,
//	sends base64 JSON as Bearer token. Server verifies secp256k1 signature
//	and checks timestamp is within 1 minute.
package auth

import (
	"crypto/elliptic"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// AuthConfig holds the persisted auth configuration.
type AuthConfig struct {
	PublicKeyHex string `json:"publicKeyHex"` // compressed secp256k1 pubkey, 66 hex chars
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
	if len(pubKeyHex) != 66 {
		return nil
	}
	if _, err := hex.DecodeString(pubKeyHex); err != nil {
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
	if len(cfg.PublicKeyHex) != 66 {
		return nil, fmt.Errorf("invalid publicKeyHex length in auth.json")
	}
	if _, err := hex.DecodeString(cfg.PublicKeyHex); err != nil {
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
	if len(publicKeyHex) != 66 {
		return nil, fmt.Errorf("publicKeyHex must be a 66-character hex string (33 bytes compressed)")
	}
	if _, err := hex.DecodeString(publicKeyHex); err != nil {
		return nil, fmt.Errorf("publicKeyHex is not valid hex: %w", err)
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
// secp256k1 token (so the caller actually controls the key), then saves
// the public key from the token as the admin.
//
// Returns the newly saved AuthConfig on success, or an error if the token
// is invalid / an admin is already configured.
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

	// Validate timestamp.
	nowMs := time.Now().UnixMilli()
	if tok.Timestamp <= 0 || abs64(nowMs-tok.Timestamp) > int64(60*1000) {
		return nil, fmt.Errorf("token expired")
	}

	// Verify the signature against the token's own public key.
	msgHash := sha256.Sum256([]byte(tok.Message))
	if err := verifySecp256k1(tok.PublicKey, tok.Signature, msgHash[:]); err != nil {
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

// VerifyToken verifies a secp256k1 auth token against the stored public key.
func VerifyToken(token string, cfg *AuthConfig) error {
	// Decode base64
	decoded, err := base64.StdEncoding.DecodeString(token)
	if err != nil {
		// Try RawStdEncoding (no padding)
		decoded, err = base64.RawStdEncoding.DecodeString(token)
		if err != nil {
			return fmt.Errorf("invalid token encoding")
		}
	}

	var tok authToken
	if err := json.Unmarshal(decoded, &tok); err != nil {
		return fmt.Errorf("invalid token format")
	}

	if tok.PublicKey == "" || tok.Signature == "" || tok.Message == "" {
		return fmt.Errorf("missing token fields")
	}

	// Check the public key matches the configured admin key
	if strings.ToLower(tok.PublicKey) != strings.ToLower(cfg.PublicKeyHex) {
		return fmt.Errorf("unknown public key")
	}

	// Check timestamp (stored in milliseconds from JS Date.now())
	nowMs := time.Now().UnixMilli()
	maxAgeMs := int64(60 * 1000) // 1 minute
	if tok.Timestamp <= 0 || abs64(nowMs-tok.Timestamp) > maxAgeMs {
		return fmt.Errorf("token expired or clock skew too large")
	}

	// Verify the signature: secp256k1 over sha256(message)
	msgHash := sha256.Sum256([]byte(tok.Message))
	if err := verifySecp256k1(tok.PublicKey, tok.Signature, msgHash[:]); err != nil {
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

// verifySecp256k1 verifies a compact secp256k1 signature (64 bytes, r||s) over msgHash.
// Uses manual ECDSA verification with the secp256k1 curve parameters.
func verifySecp256k1(pubKeyHex, sigHex string, msgHash []byte) error {
	pubBytes, err := hex.DecodeString(pubKeyHex)
	if err != nil {
		return fmt.Errorf("invalid pubkey hex")
	}
	sigBytes, err := hex.DecodeString(sigHex)
	if err != nil {
		return fmt.Errorf("invalid signature hex")
	}
	if len(sigBytes) != 64 {
		return fmt.Errorf("signature must be 64 bytes (compact r||s)")
	}

	curve := secp256k1Curve()
	x, y := decompressPoint(curve, pubBytes)
	if x == nil {
		return fmt.Errorf("invalid compressed public key")
	}

	r := new(big.Int).SetBytes(sigBytes[:32])
	s := new(big.Int).SetBytes(sigBytes[32:])

	// Manual ECDSA verification:
	// w = s⁻¹ mod n
	// u1 = hash·w mod n
	// u2 = r·w mod n
	// (x1,y1) = u1·G + u2·Q
	// valid if x1 mod n == r
	n := curve.Params().N
	if r.Sign() <= 0 || r.Cmp(n) >= 0 {
		return fmt.Errorf("r out of range")
	}
	if s.Sign() <= 0 || s.Cmp(n) >= 0 {
		return fmt.Errorf("s out of range")
	}

	w := new(big.Int).ModInverse(s, n)
	hashInt := new(big.Int).SetBytes(msgHash)
	u1 := new(big.Int).Mul(hashInt, w)
	u1.Mod(u1, n)
	u2 := new(big.Int).Mul(r, w)
	u2.Mod(u2, n)

	// u1·G
	x1, y1 := curve.ScalarBaseMult(u1.Bytes())
	// u2·Q
	x2, y2 := curve.ScalarMult(x, y, u2.Bytes())
	// add
	rx, _ := curve.Add(x1, y1, x2, y2)
	rx.Mod(rx, n)

	if rx.Cmp(r) != 0 {
		return fmt.Errorf("signature verification failed")
	}
	return nil
}

// decompressPoint decompresses a SEC1 compressed point (33 bytes: 02/03 prefix + X).
func decompressPoint(curve elliptic.Curve, compressed []byte) (*big.Int, *big.Int) {
	if len(compressed) != 33 {
		return nil, nil
	}
	prefix := compressed[0]
	if prefix != 0x02 && prefix != 0x03 {
		return nil, nil
	}

	x := new(big.Int).SetBytes(compressed[1:])
	p := curve.Params().P

	// y² = x³ + ax + b  (for secp256k1, a=0, b=7)
	// y² = x³ + 7
	x3 := new(big.Int).Mul(x, x)
	x3.Mul(x3, x)
	x3.Add(x3, curve.Params().B)
	x3.Mod(x3, p)

	y := new(big.Int).ModSqrt(x3, p)
	if y == nil {
		return nil, nil
	}

	// Pick the correct root based on the prefix parity
	if y.Bit(0) != uint(prefix&1) {
		y.Sub(p, y)
	}

	if !curve.IsOnCurve(x, y) {
		return nil, nil
	}
	return x, y
}

// secp256k1Curve returns an elliptic.Curve implementation for secp256k1.
// Go's stdlib does not include secp256k1, so we define its parameters manually.
func secp256k1Curve() elliptic.Curve {
	return secp256k1CurveInstance
}

var secp256k1CurveInstance elliptic.Curve = newSecp256k1()

func newSecp256k1() elliptic.Curve {
	p, _ := new(big.Int).SetString("FFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFEFFFFFC2F", 16)
	n, _ := new(big.Int).SetString("FFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFEBAAEDCE6AF48A03BBFD25E8CD0364141", 16)
	b, _ := new(big.Int).SetString("0000000000000000000000000000000000000000000000000000000000000007", 16)
	gx, _ := new(big.Int).SetString("79BE667EF9DCBBAC55A06295CE870B07029BFCDB2DCE28D959F2815B16F81798", 16)
	gy, _ := new(big.Int).SetString("483ADA7726A3C4655DA4FBFC0E1108A8FD17B448A68554199C47D08FFB10D4B8", 16)

	return &elliptic.CurveParams{
		P:       p,
		N:       n,
		B:       b,
		Gx:      gx,
		Gy:      gy,
		BitSize: 256,
		Name:    "secp256k1",
	}
}

// ExtractBearerToken extracts the token from the Authorization header.
func ExtractBearerToken(authHeader string) string {
	authHeader = strings.TrimSpace(authHeader)
	if !strings.HasPrefix(strings.ToLower(authHeader), "bearer ") {
		return ""
	}
	return strings.TrimSpace(authHeader[7:])
}
