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
import { generateDeterministicSecretHex, createToken } from '../utils/crypto.js'
import { installYantrFetchAuth, nativeFetch } from '../utils/fetchInterceptor.js'

export const SECRET_KEY_STORAGE = 'yantr-secret-key'   // 64-char hex, 32 bytes

const authState = reactive({
  booting:       true,
  configured:    false,
  authenticated: false,
  secretHex:     '',   // 64 hex chars (32 bytes)
  error:         '',
})

let bootstrapPromise = null

// ─── Storage helpers ──────────────────────────────────────────────────────────

function getStoredSecretHex() {
  return sessionStorage.getItem(SECRET_KEY_STORAGE) || ''
}

function storeIdentity(secretHex) {
  authState.secretHex = secretHex
  sessionStorage.setItem(SECRET_KEY_STORAGE, secretHex)
}

function clearStoredIdentity() {
  authState.secretHex    = ''
  authState.authenticated = false
  sessionStorage.removeItem(SECRET_KEY_STORAGE)
}

function setUnauthenticated(message = '') {
  authState.authenticated = false
  authState.error = message
}

// ─── Login ────────────────────────────────────────────────────────────────────

async function loginWithSecret(secretHex) {
  const token    = await createToken(secretHex)
  const response = await nativeFetch('/api/auth/login', {
    method:  'POST',
    headers: { Authorization: `Bearer ${token}` },
  })
  const data = await response.json().catch(() => ({}))
  if (!response.ok || !data.success) {
    throw new Error(data.error || 'Authentication failed')
  }
  authState.authenticated = true
  authState.error         = ''
  storeIdentity(secretHex)
  return data || null
}

// ─── Bootstrap ────────────────────────────────────────────────────────────────

export async function bootstrapYantrAuth() {
  installYantrFetchAuth({
    getSecretHex: () => authState.secretHex || getStoredSecretHex(),
    onUnauthorized: (status) => {
      clearStoredIdentity()
      authState.booting    = false
      authState.configured = status !== 503
      setUnauthenticated('Session expired. Sign in again.')
    }
  })

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
    if (!secretHex) {
      setUnauthenticated('')
      authState.booting = false
      return
    }

    try {
      await loginWithSecret(secretHex)
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

// ─── Token Generation for External Use ────────────────────────────────────────

export async function generateAuthToken() {
  const secretHex = authState.secretHex || getStoredSecretHex()
  if (!secretHex) return null
  return await createToken(secretHex)
}

export function openVolumeBrowser(volumeName) {
  console.log(`[Volume Browser] Opening browser for volume: ${volumeName}`)
  const url = new URL(`/browse/${volumeName}/`, window.location.origin)
  window.open(url.toString(), '_blank')
}

// ─── Setup ────────────────────────────────────────────────────────────────────

export async function setupYantrAdmin({ password, pin }) {
  if (!nativeFetch) {
    installYantrFetchAuth({
      getSecretHex: () => authState.secretHex || getStoredSecretHex(),
      onUnauthorized: (status) => {
        clearStoredIdentity()
        authState.booting    = false
        authState.configured = status !== 503
        setUnauthenticated('Session expired. Sign in again.')
      }
    })
  }

  if (!password || !pin) throw new Error('Password and PIN are required')

  // Generate a deterministic 32-byte key
  const secretHex = await generateDeterministicSecretHex(password, pin)

  const response = await nativeFetch('/api/setup/admin', {
    method:  'POST',
    headers: { 'Content-Type': 'application/json' },
    body:    JSON.stringify({ secretHex }),
  })
  const data = await response.json().catch(() => ({}))

  if (!response.ok || !data.success) {
    throw new Error(data.error || 'Failed to save admin configuration')
  }

  authState.configured = true
  // Login immediately using the freshly generated key
  await loginWithSecret(secretHex)
}

// ─── Login (by user re-entering their username+key or password-derived) ───────

/**
 * loginYantr — login by generating the key from password and pin.
 */
export async function loginYantr({ password, pin }) {
  if (!nativeFetch) {
    installYantrFetchAuth({
      getSecretHex: () => authState.secretHex || getStoredSecretHex(),
      onUnauthorized: (status) => {
        clearStoredIdentity()
        authState.booting    = false
        authState.configured = status !== 503
        setUnauthenticated('Session expired. Sign in again.')
      }
    })
  }

  if (!password || !pin) throw new Error('Password and PIN are required')
  
  const secretHex = await generateDeterministicSecretHex(password, pin)

  await loginWithSecret(secretHex)
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
    generateAuthToken,
    openVolumeBrowser,
  }
}