/** Convert ArrayBuffer to base64url (no padding). */
export function bufferToBase64url(buf) {
  const bytes = new Uint8Array(buf)
  let str = ''
  for (const b of bytes) str += String.fromCharCode(b)
  return btoa(str).replace(/\+/g, '-').replace(/\//g, '_').replace(/=/g, '')
}

/** Convert a plain string to base64url. */
export function strToBase64url(s) {
  return bufferToBase64url(new TextEncoder().encode(s))
}

/** Hex-decode a string to Uint8Array. */
export function hexToBytes(hex) {
  const arr = new Uint8Array(hex.length / 2)
  for (let i = 0; i < arr.length; i++) {
    arr[i] = parseInt(hex.slice(i * 2, i * 2 + 2), 16)
  }
  return arr
}

// ---------------------------------------------------------------------------
// Pure-JS SHA-256 — used as fallback when SubtleCrypto is unavailable (HTTP).
// Based on the public-domain implementation by Chris Veness.
// ---------------------------------------------------------------------------
function sha256Pure(msgBytes) {
  const K = [
    0x428a2f98, 0x71374491, 0xb5c0fbcf, 0xe9b5dba5,
    0x3956c25b, 0x59f111f1, 0x923f82a4, 0xab1c5ed5,
    0xd807aa98, 0x12835b01, 0x243185be, 0x550c7dc3,
    0x72be5d74, 0x80deb1fe, 0x9bdc06a7, 0xc19bf174,
    0xe49b69c1, 0xefbe4786, 0x0fc19dc6, 0x240ca1cc,
    0x2de92c6f, 0x4a7484aa, 0x5cb0a9dc, 0x76f988da,
    0x983e5152, 0xa831c66d, 0xb00327c8, 0xbf597fc7,
    0xc6e00bf3, 0xd5a79147, 0x06ca6351, 0x14292967,
    0x27b70a85, 0x2e1b2138, 0x4d2c6dfc, 0x53380d13,
    0x650a7354, 0x766a0abb, 0x81c2c92e, 0x92722c85,
    0xa2bfe8a1, 0xa81a664b, 0xc24b8b70, 0xc76c51a3,
    0xd192e819, 0xd6990624, 0xf40e3585, 0x106aa070,
    0x19a4c116, 0x1e376c08, 0x2748774c, 0x34b0bcb5,
    0x391c0cb3, 0x4ed8aa4a, 0x5b9cca4f, 0x682e6ff3,
    0x748f82ee, 0x78a5636f, 0x84c87814, 0x8cc70208,
    0x90befffa, 0xa4506ceb, 0xbef9a3f7, 0xc67178f2,
  ]

  let h0 = 0x6a09e667, h1 = 0xbb67ae85, h2 = 0x3c6ef372, h3 = 0xa54ff53a
  let h4 = 0x510e527f, h5 = 0x9b05688c, h6 = 0x1f83d9ab, h7 = 0x5be0cd19

  const l = msgBytes.length
  // Pre-processing: adding padding bits
  const extra = ((l + 9) % 64 === 0) ? 0 : 64 - ((l + 9) % 64)
  const padded = new Uint8Array(l + 9 + extra)
  padded.set(msgBytes)
  padded[l] = 0x80
  const bitLen = l * 8
  const dv = new DataView(padded.buffer)
  dv.setUint32(padded.length - 4, bitLen >>> 0, false)
  dv.setUint32(padded.length - 8, Math.floor(bitLen / 0x100000000), false)

  const rotr = (x, n) => (x >>> n) | (x << (32 - n))

  for (let i = 0; i < padded.length; i += 64) {
    const w = new Uint32Array(64)
    for (let j = 0; j < 16; j++) w[j] = dv.getUint32(i + j * 4, false)
    for (let j = 16; j < 64; j++) {
      const s0 = rotr(w[j - 15], 7) ^ rotr(w[j - 15], 18) ^ (w[j - 15] >>> 3)
      const s1 = rotr(w[j - 2], 17) ^ rotr(w[j - 2], 19) ^ (w[j - 2] >>> 10)
      w[j] = (w[j - 16] + s0 + w[j - 7] + s1) >>> 0
    }

    let [a, b, c, d, e, f, g, h] = [h0, h1, h2, h3, h4, h5, h6, h7]

    for (let j = 0; j < 64; j++) {
      const S1  = rotr(e, 6) ^ rotr(e, 11) ^ rotr(e, 25)
      const ch  = (e & f) ^ (~e & g)
      const tmp1 = (h + S1 + ch + K[j] + w[j]) >>> 0
      const S0  = rotr(a, 2) ^ rotr(a, 13) ^ rotr(a, 22)
      const maj = (a & b) ^ (a & c) ^ (b & c)
      const tmp2 = (S0 + maj) >>> 0
      h = g; g = f; f = e
      e = (d + tmp1) >>> 0
      d = c; c = b; b = a
      a = (tmp1 + tmp2) >>> 0
    }

    h0 = (h0 + a) >>> 0; h1 = (h1 + b) >>> 0
    h2 = (h2 + c) >>> 0; h3 = (h3 + d) >>> 0
    h4 = (h4 + e) >>> 0; h5 = (h5 + f) >>> 0
    h6 = (h6 + g) >>> 0; h7 = (h7 + h) >>> 0
  }

  const out = new Uint8Array(32)
  const odv = new DataView(out.buffer)
  ;[h0, h1, h2, h3, h4, h5, h6, h7].forEach((v, i) => odv.setUint32(i * 4, v, false))
  return out
}

/** Pure-JS HMAC-SHA256 returning a Uint8Array. */
function hmacSha256Pure(keyBytes, msgBytes) {
  let k = keyBytes
  if (k.length > 64) k = sha256Pure(k)
  const block = new Uint8Array(64)
  block.set(k)
  const ipad = block.map(b => b ^ 0x36)
  const opad = block.map(b => b ^ 0x5c)

  const inner = new Uint8Array(64 + msgBytes.length)
  inner.set(ipad)
  inner.set(msgBytes, 64)

  const innerHash = sha256Pure(inner)

  const outer = new Uint8Array(64 + 32)
  outer.set(opad)
  outer.set(innerHash, 64)

  return sha256Pure(outer)
}

// ---------------------------------------------------------------------------
// Whether SubtleCrypto is available (requires secure context or localhost).
// ---------------------------------------------------------------------------
const hasSubtle = typeof crypto !== 'undefined' && crypto.subtle != null

/** SHA-256 hash — works on both HTTP and HTTPS. Returns Uint8Array. */
async function sha256(data) {
  if (hasSubtle) {
    return new Uint8Array(await crypto.subtle.digest('SHA-256', data))
  }
  return sha256Pure(data instanceof Uint8Array ? data : new Uint8Array(data))
}

/** Import or represent a raw hex key for HMAC-SHA256. */
async function importHmacKeyInternal(secretHex) {
  if (hasSubtle) {
    const bytes = hexToBytes(secretHex)
    return crypto.subtle.importKey(
      'raw',
      bytes,
      { name: 'HMAC', hash: 'SHA-256' },
      false,
      ['sign'],
    )
  }
  // Fallback: return the raw bytes — used directly in hmacSha256Pure
  return hexToBytes(secretHex)
}

// ---------------------------------------------------------------------------
// Public API
// ---------------------------------------------------------------------------

/** Import a raw hex key as a CryptoKey for HMAC-SHA256. */
export async function importHmacKey(secretHex) {
  return importHmacKeyInternal(secretHex)
}

/** Generate a deterministic 32-byte key from password + pin, returned as hex. */
export async function generateDeterministicSecretHex(password, pin) {
  const enc = new TextEncoder()
  const data = enc.encode(`${password}:${pin}`)
  const hash = await sha256(data)
  return Array.from(hash).map(b => b.toString(16).padStart(2, '0')).join('')
}

/** Create a signed HMAC-SHA256 JWT token. Valid for 2 hours. */
export async function createToken(secretHex) {
  const header  = strToBase64url(JSON.stringify({ alg: 'HS256', typ: 'JWT' }))
  const now     = Math.floor(Date.now() / 1000)
  const payload = strToBase64url(JSON.stringify({ iat: now, exp: now + 7200 }))
  const signingInput = `${header}.${payload}`

  const msgBytes = new TextEncoder().encode(signingInput)

  let sigBytes
  if (hasSubtle) {
    const key = await importHmacKeyInternal(secretHex)
    sigBytes = new Uint8Array(await crypto.subtle.sign('HMAC', key, msgBytes))
  } else {
    sigBytes = hmacSha256Pure(hexToBytes(secretHex), msgBytes)
  }

  const sig = bufferToBase64url(sigBytes)
  return `${signingInput}.${sig}`
}
