/**
 * crypto.js — Ed25519 auth for Yantr
 *
 * Key derivation: sha256(password:pin) iterated → 32-byte Ed25519 seed
 * Auth token:     base64(JSON{ publickey, signature, message:"timestamp:nonce" })
 * Signature:      Ed25519 over raw message bytes (not a hash)
 *
 * Compatible with Go stdlib crypto/ed25519 — zero external deps on server.
 *
 * Key sizes:
 *   private key (seed): 32 bytes / 64 hex chars
 *   public key:         32 bytes / 64 hex chars
 *   signature:          64 bytes / 128 hex chars
 */
import { ed25519 } from '@noble/curves/ed25519.js'
import { sha256 as nobleSha256 } from '@noble/hashes/sha2.js'

// ─── Helpers ──────────────────────────────────────────────────────────────────

function bytesToHex(bytes) {
  return Array.from(bytes, b => b.toString(16).padStart(2, '0')).join('')
}

function hexToBytes(hex) {
  const arr = new Uint8Array(hex.length / 2)
  for (let i = 0; i < arr.length; i++)
    arr[i] = parseInt(hex.slice(i * 2, i * 2 + 2), 16)
  return arr
}

async function randomBytes(length) {
  const bytes = new Uint8Array(length)
  crypto.getRandomValues(bytes)
  return bytes
}

function base64Encode(obj) {
  return btoa(JSON.stringify(obj))
}

// ─── Private key derivation ───────────────────────────────────────────────────

/**
 * Derive a deterministic Ed25519 private key (seed) from password + pin.
 * Any 32-byte value is a valid Ed25519 seed — no rejection sampling needed.
 */
export async function derivePrivateKey(password, pin) {
  const enc = new TextEncoder()
  const seed = enc.encode(`${password}:${pin}`)

  // Initial hash — sha256 always produces 32 bytes, valid as Ed25519 seed
  let key = nobleSha256(seed)

  // Iterate sha256 based on seed length for key stretching
  const rounds = Math.max(seed.length, 1)
  for (let i = 1; i < rounds; i++) {
    key = nobleSha256(key)
  }

  return bytesToHex(key)
}

// ─── Public key ───────────────────────────────────────────────────────────────

/** Derive Ed25519 public key (32 bytes) from private key seed hex. */
export function getPublicKey(privateKeyHex) {
  const pub = ed25519.getPublicKey(hexToBytes(privateKeyHex))
  return bytesToHex(pub)
}

// ─── Auth token ───────────────────────────────────────────────────────────────

/**
 * Create a signed auth token for Yantr API.
 * Format: base64(JSON{ publickey, signature, message, timestamp, nonce })
 *   message   = "timestamp:nonce"
 *   signature = Ed25519 over raw message bytes
 */
export async function createToken(privateKeyHex) {
  const publicKeyHex = getPublicKey(privateKeyHex)
  const timestamp = Date.now()
  const nonceBytes = await randomBytes(16)
  const nonce = bytesToHex(nonceBytes)
  const message = `${timestamp}:${nonce}`

  // Ed25519 signs raw bytes — no pre-hashing
  const msgBytes = new TextEncoder().encode(message)
  const sig = ed25519.sign(msgBytes, hexToBytes(privateKeyHex))
  const signature = bytesToHex(sig)

  return base64Encode({ publickey: publicKeyHex, signature, message, timestamp, nonce })
}
