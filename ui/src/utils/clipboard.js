/**
 * clipboard.js — copy text in a browser that may not be a secure context.
 *
 * `navigator.clipboard` exists only on a secure context (HTTPS, localhost or
 * 127.0.0.1). Yantr is normally opened over plain HTTP on a LAN address — the
 * backend serves 5252 with no TLS — so on the address people actually use, the
 * whole object is `undefined`. A bare `navigator.clipboard.writeText()` then
 * throws a TypeError, which is what the container page reported as "Could not
 * copy the shell command": not a refused permission, but a missing API.
 *
 * The textarea + `document.execCommand('copy')` path is deprecated, but it is
 * the one write that still works outside a secure context, so it is the
 * fallback rather than the primary. Both paths are tried before giving up.
 */

/**
 * Copies `text`, resolving to whether it reached the clipboard.
 *
 * Never throws: a caller that needs to report a failure gets `false` instead of
 * an exception it has to guess the cause of.
 *
 * @param {string} text
 * @returns {Promise<boolean>}
 */
export async function copyText(text) {
  if (!text) return false

  if (navigator.clipboard?.writeText) {
    try {
      await navigator.clipboard.writeText(text)
      return true
    } catch {
      // Permission denied, or the document is not focused. The legacy path
      // has no such preconditions, so it is still worth trying.
    }
  }

  return legacyCopy(text)
}

/**
 * Selects a temporary off-screen textarea and copies it.
 *
 * The textarea has to stay in the layout (`display: none` is not selectable),
 * so it is parked above the viewport and restored afterwards. The user's
 * current selection is put back, because copying from a page must not look
 * like the page lost its place.
 */
function legacyCopy(text) {
  if (typeof document === 'undefined') return false

  const area = document.createElement('textarea')
  area.value = text
  area.setAttribute('readonly', '')
  area.style.position = 'fixed'
  area.style.top = '-1000px'
  area.style.opacity = '0'

  const selection = document.getSelection()
  const previousRange = selection && selection.rangeCount > 0 ? selection.getRangeAt(0) : null

  document.body.appendChild(area)
  try {
    area.select()
    return document.execCommand('copy')
  } catch {
    return false
  } finally {
    document.body.removeChild(area)
    if (selection && previousRange) {
      selection.removeAllRanges()
      selection.addRange(previousRange)
    }
  }
}
