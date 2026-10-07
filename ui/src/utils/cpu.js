/**
 * cpu.js — derive a real CPU usage rate from cumulative counters.
 *
 * The stats endpoint returns counters, not a percentage. Podman leaves
 * precpu_stats all-zero unless the request asks for stream=1, so percent has to
 * be differenced from two consecutive samples. The UI already polls every 2s,
 * so this costs nothing.
 *
 * The rate is normalized against elapsed wall clock, not the engine's
 * system_cpu_usage. See cpuPercentBetween for why that distinction matters.
 */

/**
 * Computes CPU percent between two counter samples.
 *
 * @param {object|null} prev  previous sample: {usage, sampledAtMs, onlineCpus}
 * @param {object|null} curr  current sample, same shape
 * @returns {number|null} percent 0..100*cores, or null when not computable yet
 */
export function cpuPercentBetween(prev, curr) {
  if (!prev || !curr) return null

  const dCpu = Number(curr.usage) - Number(prev.usage)
  const dWall = Number(curr.sampledAtMs) - Number(prev.sampledAtMs)

  // A counter that went backwards means the container was recreated between the
  // two samples. The baseline is worthless; start a new one instead of
  // reporting a large negative spike.
  if (!(dCpu >= 0) || !(dWall > 0)) return null

  const cores = Number(curr.onlineCpus) || Number(prev.onlineCpus) || 1

  // Normalize against elapsed wall clock: one fully saturated core accrues
  // 1e6 ns of CPU time per millisecond.
  //
  // The Docker formula ((dCpu/dSys)*onlineCpus*100) is deliberately not used.
  // It assumes system_cpu_usage spans every core on the host, but rootless
  // Podman reports it for the user's cgroup slice only, while online_cpus is
  // the host's core count. The two denominators describe different scopes, so
  // the result is inflated by the slice's share of the machine: measured on a
  // 24-core host, a container pegging a single core reported 1680%.
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
