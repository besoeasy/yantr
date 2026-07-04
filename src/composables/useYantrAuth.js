/**
 * useYantrAuth.js
 *
 * Auth composable using browser SubtleCrypto (HMAC-SHA256 JWT) instead of daku.
 *
 * Token format: base64url(header).base64url(payload).base64url(signature)
 *   header:  {"alg":"HS256","typ":"JWT"}
 *   payload: {"sub":"<username>","iat":<unix>,"exp":<unix+1h>}
 *   signature: HMAC-SHA256(header.payload, 32-byte key)
 *
 * This is fully compatible with the Go core/auth package.
 * The 32-byte key (secretHex) is stored in localStorage and sent once to the
 * server during setup. All subsequent tokens are signed client-side.
 */
import { reactive, readonly } from 'vue'

const SECRET_KEY_STORAGE = 'yantr-secret-key'   // 64-char hex, 32 bytes
const USERNAME_STORAGE   = 'yantr-username'

const PUBLIC_PATHS = new Set([
  '/api/health',
  '/api/version',
  '/api/setup/status',
  '/api/setup/admin',
  '/api/auth/login',
])

const authState = reactive({
  booting:       true,
  configured:    false,
  authenticated: false,
  user:          null,
  secretHex:     '',   // 64 hex chars (32 bytes)
  error:         '',
})

let bootstrapPromise = null
let fetchInstalled   = false
let nativeFetch      = null

// ─── Crypto helpers ──────────────────────────────────────────────────────────

/** Convert ArrayBuffer to base64url (no padding). */
function bufferToBase64url(buf) {
  const bytes = new Uint8Array(buf)
  let str = ''
  for (const b of bytes) str += String.fromCharCode(b)
  return btoa(str).replace(/\+/g, '-').replace(/\//g, '_').replace(/=/g, '')
}

/** Convert a plain string to base64url. */
function strToBase64url(s) {
  return bufferToBase64url(new TextEncoder().encode(s))
}

/** Import a raw hex key as a CryptoKey for HMAC-SHA256. */
async function importHmacKey(secretHex) {
  const bytes = hexToBytes(secretHex)
  return crypto.subtle.importKey(
    'raw',
    bytes,
    { name: 'HMAC', hash: 'SHA-256' },
    false,
    ['sign'],
  )
}

/** Hex-decode a string to Uint8Array. */
function hexToBytes(hex) {
  const arr = new Uint8Array(hex.length / 2)
  for (let i = 0; i < arr.length; i++) {
    arr[i] = parseInt(hex.slice(i * 2, i * 2 + 2), 16)
  }
  return arr
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
async function createToken(secretHex, username) {
  const header  = strToBase64url(JSON.stringify({ alg: 'HS256', typ: 'JWT' }))
  const now     = Math.floor(Date.now() / 1000)
  const payload = strToBase64url(JSON.stringify({ sub: username, iat: now, exp: now + 7200 }))
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

// ─── Storage helpers ──────────────────────────────────────────────────────────

function getStoredSecretHex() {
  return sessionStorage.getItem(SECRET_KEY_STORAGE) || ''
}

function storeIdentity(secretHex, username) {
  authState.secretHex = secretHex
  sessionStorage.setItem(SECRET_KEY_STORAGE, secretHex)
  localStorage.setItem(USERNAME_STORAGE, username) // Username can persist across sessions
}

function clearStoredIdentity() {
  authState.secretHex    = ''
  authState.authenticated = false
  authState.user         = null
  sessionStorage.removeItem(SECRET_KEY_STORAGE)
}

function setUnauthenticated(message = '') {
  authState.authenticated = false
  authState.user  = null
  authState.error = message
}

// ─── Request helpers ──────────────────────────────────────────────────────────

function getRequestUrl(input) {
  const source = typeof input === 'string' ? input : input?.url || '/'
  return new URL(source, window.location.origin)
}

function isYantrRequest(url) {
  const configured = window.VITE_API_URL
    ? new URL(window.VITE_API_URL, window.location.origin).origin
    : window.location.origin
  return url.origin === window.location.origin || url.origin === configured
}

function shouldAttachAuth(url) {
  if (!isYantrRequest(url)) return false
  const pathname = url.pathname || '/'
  if (!(pathname.startsWith('/api/') || pathname.startsWith('/browse/'))) return false
  return !PUBLIC_PATHS.has(pathname)
}

// ─── Login ────────────────────────────────────────────────────────────────────

async function loginWithSecret(secretHex, username) {
  const token    = await createToken(secretHex, username)
  const response = await nativeFetch('/api/auth/login', {
    method:  'POST',
    headers: { Authorization: `Bearer ${token}` },
  })
  const data = await response.json().catch(() => ({}))
  if (!response.ok || !data.success) {
    throw new Error(data.error || 'Authentication failed')
  }
  authState.authenticated = true
  authState.user          = data.user || null
  authState.error         = ''
  storeIdentity(secretHex, data.user?.username || username || localStorage.getItem(USERNAME_STORAGE) || '')
  return data.user || null
}

// ─── Fetch interceptor ────────────────────────────────────────────────────────

export function installYantrFetchAuth() {
  if (fetchInstalled || typeof window === 'undefined') return

  nativeFetch = window.fetch.bind(window)
  window.fetch = async (input, init = undefined) => {
    const url = getRequestUrl(input)
    if (!shouldAttachAuth(url)) {
      return nativeFetch(input, init)
    }

    const secretHex = authState.secretHex || getStoredSecretHex()
    const username  = localStorage.getItem(USERNAME_STORAGE) || ''
    if (!secretHex) {
      return nativeFetch(input, init)
    }

    const token   = await createToken(secretHex, username)
    const headers = new Headers(
      init?.headers
        || (input instanceof Request ? input.headers : undefined)
        || undefined
    )
    headers.set('Authorization', `Bearer ${token}`)

    const response = input instanceof Request
      ? await nativeFetch(new Request(input, { headers }), init)
      : await nativeFetch(input, { ...(init || {}), headers })

    if (response.status === 401 || response.status === 503) {
      clearStoredIdentity()
      authState.booting    = false
      authState.configured = response.status !== 503
      setUnauthenticated('Session expired. Sign in again.')
    }

    return response
  }

  fetchInstalled = true
}

// ─── Bootstrap ────────────────────────────────────────────────────────────────

export async function bootstrapYantrAuth() {
  if (!nativeFetch) installYantrFetchAuth()
  if (bootstrapPromise) return bootstrapPromise

  bootstrapPromise = (async () => {
    authState.booting = true
    authState.error   = ''

    const response = await nativeFetch('/api/setup/status')
    const data     = await response.json().catch(() => ({}))
    authState.configured = !!data.configured

    if (!data.configured) {
      clearStoredIdentity()
      authState.booting = false
      return
    }

    const secretHex = getStoredSecretHex()
    const username  = localStorage.getItem(USERNAME_STORAGE) || ''
    if (!secretHex || !username) {
      setUnauthenticated('')
      authState.booting = false
      return
    }

    try {
      await loginWithSecret(secretHex, username)
    } catch {
      clearStoredIdentity()
      setUnauthenticated('Sign in to unlock Yantr.')
    } finally {
      authState.booting = false
    }
  })()

  try {
    await bootstrapPromise
  } finally {
    bootstrapPromise = null
  }
}

// ─── Setup ────────────────────────────────────────────────────────────────────

export async function setupYantrAdmin({ username }) {
  if (!nativeFetch) installYantrFetchAuth()

  const normalizedUsername = String(username || '').trim()
  if (!normalizedUsername) throw new Error('Username is required')

  // Generate a fresh 32-byte random key
  const secretHex = generateSecretHex()

  const response = await nativeFetch('/api/setup/admin', {
    method:  'POST',
    headers: { 'Content-Type': 'application/json' },
    body:    JSON.stringify({ username: normalizedUsername, secretHex }),
  })
  const data = await response.json().catch(() => ({}))

  if (!response.ok || !data.success) {
    throw new Error(data.error || 'Failed to save admin configuration')
  }

  authState.configured = true
  // Login immediately using the freshly generated key
  await loginWithSecret(secretHex, normalizedUsername)
}

// ─── Login (by user re-entering their username+key or password-derived) ───────

/**
 * loginYantr — re-login using the key stored in localStorage.
 * The user simply enters their username; the key is retrieved from storage.
 * If no key is found (new device), they must set up again.
 */
export async function loginYantr({ username }) {
  if (!nativeFetch) installYantrFetchAuth()

  const normalizedUsername = String(username || '').trim()
  const secretHex = getStoredSecretHex()

  if (!secretHex) {
    throw new Error('No saved credentials on this device. Please set up Yantr again.')
  }

  localStorage.setItem(USERNAME_STORAGE, normalizedUsername)
  await loginWithSecret(secretHex, normalizedUsername)
}

// ─── Logout ───────────────────────────────────────────────────────────────────

export function logoutYantr() {
  clearStoredIdentity()
  setUnauthenticated('Sign in to unlock Yantr.')
}

// ─── Composable ───────────────────────────────────────────────────────────────

export function useYantrAuth() {
  return {
    authState: readonly(authState),
    bootstrapYantrAuth,
    setupYantrAdmin,
    loginYantr,
    logoutYantr,
  }
}