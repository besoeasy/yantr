/**
 * crypto.js — secp256k1 auth for Yantr
 *
 * Key derivation: sha256(password:pin) iterated → private key
 * Auth token: base64(JSON{ publickey, signature, message:"timestamp:nonce" })
 * Signature: secp256k1 over sha256(message)
 *
 * Compatible with the Go core/auth package which verifies secp256k1 signatures.
 *
 * Uses @noble/curves/secp256k1 (v3+) which has built-in HMAC support —
 * no manual hmacSha256Sync wiring needed.
 */
import { secp256k1 } from '@noble/curves/secp256k1.js'
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
 * Derive a deterministic secp256k1 private key from password + pin.
 * Uses iterative sha256 hashing until a valid scalar is found.
 */
export async function derivePrivateKey(password, pin) {
  const enc = new TextEncoder()
  const seed = enc.encode(`${password}:${pin}`)

  // Initial hash
  let key = nobleSha256(seed)

  // Iterate sha256 based on seed length (mirrors old JS behaviour)
  const rounds = Math.max(seed.length, 1)
  for (let i = 1; i < rounds; i++) {
    key = nobleSha256(key)
  }

  // Ensure the key is a valid secp256k1 private key scalar
  while (true) {
    try {
      secp256k1.getPublicKey(key, true)
      return bytesToHex(key)
    } catch {
      key = nobleSha256(key)
    }
  }
}

// ─── Public key ───────────────────────────────────────────────────────────────

/** Get compressed public key hex from private key hex. */
export function getPublicKey(privateKeyHex) {
  const pub = secp256k1.getPublicKey(hexToBytes(privateKeyHex), true)
  return bytesToHex(pub)
}

// ─── Auth token ───────────────────────────────────────────────────────────────

/**
 * Create a signed auth token for Yantr API.
 * Format: base64(JSON{ publickey, signature, message, timestamp, nonce })
 *   message = "timestamp:nonce"
 *   signature = secp256k1 over sha256(message)
 */
export async function createToken(privateKeyHex) {
  const publicKeyHex = getPublicKey(privateKeyHex)
  const timestamp = Date.now()
  const nonceBytes = await randomBytes(16)
  const nonce = bytesToHex(nonceBytes)
  const message = `${timestamp}:${nonce}`

  const msgHash = nobleSha256(new TextEncoder().encode(message))
  const sig = secp256k1.sign(msgHash, hexToBytes(privateKeyHex))
  const signature = bytesToHex(sig.toCompactRawBytes())

  return base64Encode({ publickey: publicKeyHex, signature, message, timestamp, nonce })
}
