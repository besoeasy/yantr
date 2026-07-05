/**
 * useYantrAuth.js
 *
 * Auth composable using secp256k1 keypair auth.
 *
 * Key derivation: sha256(password:pin) iterated → secp256k1 private key
 * Auth token: base64(JSON{ publickey, signature, message:"timestamp:nonce" })
 *
 * Setup: derive private key → extract public key → POST /api/setup/admin { publicKeyHex }
 * Login: derive private key → sign token → POST /api/auth/login with Bearer token
 *
 * Compatible with the Go core/auth package.
 */
import { reactive, readonly } from 'vue'
import { derivePrivateKey, getPublicKey, createToken } from '../utils/crypto.js'
import { installYantrFetchAuth, nativeFetch } from '../utils/fetchInterceptor.js'

export const PRIVATE_KEY_STORAGE = 'yantr-private-key'  // 64-char hex, 32 bytes

const authState = reactive({
  booting:       true,
  configured:    false,
  authenticated: false,
  privateKeyHex: '',
  error:         '',
})

let bootstrapPromise = null

// ─── Storage helpers ──────────────────────────────────────────────────────────

function getStoredPrivateKeyHex() {
  return sessionStorage.getItem(PRIVATE_KEY_STORAGE) || ''
}

function storeIdentity(privateKeyHex) {
  authState.privateKeyHex = privateKeyHex
  sessionStorage.setItem(PRIVATE_KEY_STORAGE, privateKeyHex)
}

function clearStoredIdentity() {
  authState.privateKeyHex = ''
  authState.authenticated  = false
  sessionStorage.removeItem(PRIVATE_KEY_STORAGE)
}

function setUnauthenticated(message = '') {
  authState.authenticated = false
  authState.error = message
}

// ─── Login ────────────────────────────────────────────────────────────────────

async function loginWithPrivateKey(privateKeyHex) {
  const token    = await createToken(privateKeyHex)
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
  storeIdentity(privateKeyHex)
  return data || null
}

// ─── Bootstrap ────────────────────────────────────────────────────────────────

export async function bootstrapYantrAuth() {
  installYantrFetchAuth({
    getPrivateKeyHex: () => authState.privateKeyHex || getStoredPrivateKeyHex(),
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

    const privateKeyHex = getStoredPrivateKeyHex()
    if (!privateKeyHex) {
      setUnauthenticated('')
      authState.booting = false
      return
    }

    try {
      await loginWithPrivateKey(privateKeyHex)
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
  const privateKeyHex = authState.privateKeyHex || getStoredPrivateKeyHex()
  if (!privateKeyHex) return null
  return await createToken(privateKeyHex)
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
      getPrivateKeyHex: () => authState.privateKeyHex || getStoredPrivateKeyHex(),
      onUnauthorized: (status) => {
        clearStoredIdentity()
        authState.booting    = false
        authState.configured = status !== 503
        setUnauthenticated('Session expired. Sign in again.')
      }
    })
  }

  if (!password || !pin) throw new Error('Password and PIN are required')

  // Derive deterministic private key from password + pin
  const privateKeyHex = await derivePrivateKey(password, pin)
  const publicKeyHex  = getPublicKey(privateKeyHex)

  const response = await nativeFetch('/api/setup/admin', {
    method:  'POST',
    headers: { 'Content-Type': 'application/json' },
    body:    JSON.stringify({ publicKeyHex }),
  })
  const data = await response.json().catch(() => ({}))

  if (!response.ok || !data.success) {
    throw new Error(data.error || 'Failed to save admin configuration')
  }

  authState.configured = true
  // Login immediately using the freshly derived private key
  await loginWithPrivateKey(privateKeyHex)
}

// ─── Login ────────────────────────────────────────────────────────────────────

/**
 * loginYantr — derive private key from password + pin and sign a token.
 */
export async function loginYantr({ password, pin }) {
  if (!nativeFetch) {
    installYantrFetchAuth({
      getPrivateKeyHex: () => authState.privateKeyHex || getStoredPrivateKeyHex(),
      onUnauthorized: (status) => {
        clearStoredIdentity()
        authState.booting    = false
        authState.configured = status !== 503
        setUnauthenticated('Session expired. Sign in again.')
      }
    })
  }

  if (!password || !pin) throw new Error('Password and PIN are required')

  const privateKeyHex = await derivePrivateKey(password, pin)
  await loginWithPrivateKey(privateKeyHex)
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