import { createToken } from './crypto.js'

const PUBLIC_PATHS = new Set([
  '/api/health',
  '/api/version',
  '/api/setup/status',
  '/api/setup/admin',
  '/api/auth/login',
])

export let nativeFetch = null
let fetchInstalled = false

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
  if (!pathname.startsWith('/api/')) return false
  return !PUBLIC_PATHS.has(pathname)
}

export function installYantrFetchAuth({ getPrivateKeyHex, onUnauthorized }) {
  if (fetchInstalled || typeof window === 'undefined') return

  nativeFetch = window.fetch.bind(window)
  window.fetch = async (input, init = undefined) => {
    const url = getRequestUrl(input)
    if (!shouldAttachAuth(url)) {
      return nativeFetch(input, init)
    }

    const privateKeyHex = getPrivateKeyHex()
    if (!privateKeyHex) {
      return nativeFetch(input, init)
    }

    const token   = await createToken(privateKeyHex)
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
      onUnauthorized(response.status)
    }

    return response
  }

  fetchInstalled = true
}
