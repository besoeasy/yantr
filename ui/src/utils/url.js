/**
 * url.js — shared URL construction for container/service port links.
 *
 * Both StackView and ContainerDetail render "Open" links to the host port a
 * service is published on. They each had their own `appUrl`, and the StackView
 * copy produced a malformed URL on an IPv6 host: a bare `2001:db8::1:8080`
 * cannot be parsed by the WHATWG URL parser, which requires IPv6 literals to be
 * bracketed. The link silently rendered as a dead href. This is the
 * single implementation both views now use.
 */

/**
 * Normalizes a protocol string to a bare scheme ("https" from "https://",
 * "https:", "https", or "" ). Returns "http" when nothing usable is present.
 */
export function normalizeScheme(protocol) {
  const cleaned = String(protocol ?? '')
    .replace('://', '')
    .replace(':', '')
    .trim()
    .toLowerCase()
  return cleaned === 'https' ? 'https' : 'http'
}

/**
 * True when a protocol describes something a browser can navigate to.
 * x-yantr.ports allows HTTP, HTTPS, TCP and UDP; a raw TCP/UDP port has no
 * meaningful "Open" action even though the port is still listed.
 */
export function isNavigableProtocol(protocol) {
  const cleaned = String(protocol ?? '').replace('://', '').replace(':', '').trim().toLowerCase()
  return cleaned === 'http' || cleaned === 'https'
}

/**
 * Builds a link to a published host port on this machine.
 *
 * @param {number|string} port        host port, or any string containing one
 * @param {string} [protocol]         HTTP / HTTPS / TCP / UDP / "http://"
 * @returns {string} an absolute URL, or '' when the port is unusable
 */
export function appUrl(port, protocol = 'http') {
  const scheme = normalizeScheme(protocol)

  let host = window.location.hostname || 'localhost'
  // WHATWG URL requires IPv6 literals in brackets. Already-bracketed hosts
  // (as from location.hostname in some browsers) must not be double-wrapped.
  if (host.includes(':') && !host.startsWith('[')) {
    host = `[${host}]`
  }

  const portMatch = String(port ?? '').trim().match(/\d+/)
  if (!portMatch || parseInt(portMatch[0], 10) <= 0) {
    // No usable port (absent, zero, or non-numeric): link to the host only
    // rather than emitting a dead ":0" target.
    return `${scheme}://${host}`
  }

  return `${scheme}://${host}:${portMatch[0]}`
}
