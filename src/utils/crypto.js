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

/** Import a raw hex key as a CryptoKey for HMAC-SHA256. */
export async function importHmacKey(secretHex) {
  const bytes = hexToBytes(secretHex)
  return crypto.subtle.importKey(
    'raw',
    bytes,
    { name: 'HMAC', hash: 'SHA-256' },
    false,
    ['sign'],
  )
}

/** Generate a deterministic 32-byte key from password + pin, returned as hex. */
export async function generateDeterministicSecretHex(password, pin) {
  const enc = new TextEncoder()
  const data = enc.encode(`${password}:${pin}`)
  const hash = await crypto.subtle.digest('SHA-256', data)
  const hashArray = Array.from(new Uint8Array(hash))
  return hashArray.map(b => b.toString(16).padStart(2, '0')).join('')
}

/** Create a signed HMAC-SHA256 JWT token. Valid for 2 hours. */
export async function createToken(secretHex) {
  const header  = strToBase64url(JSON.stringify({ alg: 'HS256', typ: 'JWT' }))
  const now     = Math.floor(Date.now() / 1000)
  const payload = strToBase64url(JSON.stringify({ iat: now, exp: now + 7200 }))
  const signingInput = `${header}.${payload}`

  const key = await importHmacKey(secretHex)
  const sigBuf = await crypto.subtle.sign(
    'HMAC',
    key,
    new TextEncoder().encode(signingInput),
  )
  const sig = bufferToBase64url(sigBuf)
  return `${signingInput}.${sig}`
}
