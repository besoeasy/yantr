import { onUnmounted, ref } from 'vue'

/**
 * usePolling — runs a fetch on an interval with the hazards handled.
 *
 * The app previously had ~15 hand-rolled `let x = null; onMounted(...);
 * onUnmounted(clearInterval)` blocks, none of which guarded against the
 * failure modes that matter once a panel is open for a while:
 *
 *  - **Overlapping ticks.** A slow request lets the next tick start before the
 *    previous one lands, so responses pile up and can resolve out of order.
 *  - **Stale writes.** A response for a request the caller no longer cares
 *    about still overwrites current state.
 *  - **Running in a hidden tab.** Polling a dashboard nobody is looking at
 *    wastes a request per tick, and on a homelab that is real CPU and socket
 *    churn on the host.
 *
 * This owns only the timer, the in-flight flag, the visibility gate and the
 * abort. It deliberately does NOT own the fetch: call sites differ materially
 * (some fan out to two endpoints, some need a user-controlled pause, some need
 * a key-scoped value from their parent), and forcing one shape would have meant
 * rewriting every view's error semantics at once.
 *
 * @param {Function} fn        async task, invoked with an AbortSignal as its
 *                             ONLY argument. If your task takes options as its
 *                             first parameter, wrap it: `() => myTask(opts)`.
 *                             Passing it directly would have the signal
 *                             destructured as those options.
 * @param {number}   ms        interval in milliseconds
 * @param {object}   [opts]
 * @param {boolean}  [opts.immediate=true]  run once on start
 * @param {Function} [opts.shouldRun]       extra predicate, e.g. a tab is active
 */
export function usePolling(fn, ms, opts = {}) {
  const { immediate = true, shouldRun = null } = opts

  const running = ref(false)
  const inFlight = ref(false)
  const lastError = ref(null)

  let timer = null
  let controller = null
  let stopped = true
  let hiddenPause = false

  async function tick() {
    // Never let two ticks overlap: a slow endpoint would otherwise queue
    // requests and deliver them out of order.
    if (inFlight.value) return
    if (shouldRun && !shouldRun()) return

    inFlight.value = true
    controller = typeof AbortController === 'function' ? new AbortController() : null
    try {
      await fn(controller ? controller.signal : undefined)
      lastError.value = null
    } catch (e) {
      // A poll failing is normal (backend restarting, container stopped). Record
      // it for the caller but never throw into the timer, which would otherwise
      // surface as an unhandled rejection every tick.
      if (e && e.name !== 'AbortError') {
        lastError.value = e
      }
    } finally {
      inFlight.value = false
      controller = null
    }
  }

  function schedule() {
    clearTimer()
    if (stopped) return
    timer = setInterval(tick, ms)
  }

  function clearTimer() {
    if (timer !== null) {
      clearInterval(timer)
      timer = null
    }
  }

  function start() {
    stopped = false
    if (immediate) tick()
    schedule()
  }

  function stop() {
    stopped = true
    clearTimer()
    if (controller) {
      controller.abort()
      controller = null
    }
  }

  /** Run now, bypassing the interval, still respecting the in-flight guard. */
  function refresh() {
    return tick()
  }

  // Pause while the tab is hidden; refresh immediately when it comes back so
  // the view is not stale on return.
  function onVisibilityChange() {
    if (typeof document === 'undefined') return
    if (document.hidden) {
      hiddenPause = true
      clearTimer()
    } else if (hiddenPause) {
      hiddenPause = false
      tick()
      schedule()
    }
  }

  if (typeof document !== 'undefined' && typeof document.addEventListener === 'function') {
    document.addEventListener('visibilitychange', onVisibilityChange)
  }

  onUnmounted(() => {
    stop()
    if (typeof document !== 'undefined' && typeof document.removeEventListener === 'function') {
      document.removeEventListener('visibilitychange', onVisibilityChange)
    }
  })

  return { start, stop, refresh, running, inFlight, lastError }
}
