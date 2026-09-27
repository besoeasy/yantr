/**
 * cpu.js — derive a real CPU usage rate from cumulative counters.
 *
 * The stats endpoint returns counters, not a percentage. Podman leaves
 * precpu_stats all-zero unless the request asks for stream=1, so the usual
 * (cpu_delta / system_delta) * online_cpus * 100 degenerates to
 * (container_lifetime_ns / host_lifetime_ns) — a monotonic ramp that has
 * nothing to do with current load. Differencing two consecutive samples is the
 * fix, and the UI already polls every 2s, so it costs nothing.
 */

/**
 * Computes CPU percent between two counter samples.
 *
 * @param {object|null} prev  previous sample: {usage, systemUsage, sampledAtMs, onlineCpus}
 * @param {object|null} curr  current sample, same shape
 * @returns {number|null} percent 0..100*cores, or null when not computable yet
 */
export function cpuPercentBetween(prev, curr) {
  if (!prev || !curr) return null

  const dCpu = Number(curr.usage) - Number(prev.usage)
  const dSys = Number(curr.systemUsage) - Number(prev.systemUsage)
  const dWall = Number(curr.sampledAtMs) - Number(prev.sampledAtMs)

  // A counter that went backwards means the container was recreated between the
  // two samples. The baseline is worthless; start a new one instead of
  // reporting a large negative spike.
  if (!(dCpu >= 0) || !(dSys >= 0) || !(dWall > 0)) return null

  const cores = Number(curr.onlineCpus) || Number(prev.onlineCpus) || 1

  // Preferred: the Docker/podman definition — share of total host CPU time,
  // scaled so that 100% means one fully saturated core. A container pegging
  // several cores therefore reads above 100, which is what `podman stats` shows
  // and what the previous (broken) server-side math also intended.
  if (dSys > 0) {
    return clampPercent((dCpu / dSys) * cores * 100, cores)
  }

  // Fallback for engines that report no host system time: scale by wall clock.
  // One saturated core accrues 1e6 ns of CPU time per millisecond.
  const nsPerMsPerCore = 1e6
  const coresUsed = dCpu / (dWall * nsPerMsPerCore)
  return clampPercent(coresUsed * 100, cores)
}

/**
 * Clamps to a sane range: never negative, and never above the physical ceiling
 * of cores * 100, which anything higher would indicate as bad data rather than
 * real usage.
 */
function clampPercent(value, cores) {
  if (!Number.isFinite(value) || value < 0) return 0
  const cap = (Number(cores) || 1) * 100
  return Math.round(Math.min(value, cap) * 100) / 100
}

/**
 * A stateful rate calculator: feed it each sample, get back the current percent.
 * Holds the previous sample so callers do not have to.
 */
export function createCpuRateCalculator() {
  let prev = null

  return {
    /**
     * @returns {number|null} percent for the interval ending at `sample`
     */
    push(sample) {
      if (!sample) {
        prev = null
        return null
      }
      const pct = cpuPercentBetween(prev, sample)
      prev = sample
      return pct
    },
    /** Forget the baseline, e.g. when switching containers. */
    reset() {
      prev = null
    },
  }
}
