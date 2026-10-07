import { describe, expect, it } from 'vitest'
import { cpuPercentBetween, createCpuRateCalculator } from './cpu.js'

// Real measurements taken from a rootless Podman 5.x host (24 cores, cgroups v2)
// against a container running a single-core busy loop. system_cpu_usage tracked
// the user's cgroup slice, not the host, while online_cpus reported all 24.
const BUSY_LOOP = {
  prev: { usage: 1224579521000, systemUsage: 6600020093000, onlineCpus: 24, sampledAtMs: 1_000 },
  curr: {
    usage: 1224579521000 + 2_019_766_000,
    systemUsage: 6600020093000 + 2_884_550_000,
    onlineCpus: 24,
    sampledAtMs: 3_000,
  },
}

describe('cpuPercentBetween', () => {
  it('reports ~100% for a container pegging one core, not the slice-inflated rate', () => {
    const pct = cpuPercentBetween(BUSY_LOOP.prev, BUSY_LOOP.curr)
    expect(pct).toBeGreaterThanOrEqual(95)
    expect(pct).toBeLessThanOrEqual(105)

    // The Docker formula on the same counters is what this regression guards.
    const dCpu = BUSY_LOOP.curr.usage - BUSY_LOOP.prev.usage
    const dSys = BUSY_LOOP.curr.systemUsage - BUSY_LOOP.prev.systemUsage
    const docker = (dCpu / dSys) * BUSY_LOOP.curr.onlineCpus * 100
    expect(docker).toBeGreaterThan(1000)
    expect(pct).toBeLessThan(docker / 10)
  })

  it('scales above 100 when a container uses several cores', () => {
    const pct = cpuPercentBetween(
      { usage: 0, onlineCpus: 24, sampledAtMs: 0 },
      { usage: 4_000_000_000, onlineCpus: 24, sampledAtMs: 2000 },
    )
    expect(pct).toBeCloseTo(200, 0)
  })

  it('reports 0% when the container burned no CPU', () => {
    const pct = cpuPercentBetween(
      { usage: 5_000, onlineCpus: 24, sampledAtMs: 0 },
      { usage: 5_000, onlineCpus: 24, sampledAtMs: 2000 },
    )
    expect(pct).toBe(0)
  })

  it('caps at cores * 100 rather than reporting impossible usage', () => {
    const pct = cpuPercentBetween(
      { usage: 0, onlineCpus: 4, sampledAtMs: 0 },
      { usage: 100_000_000_000, onlineCpus: 4, sampledAtMs: 1000 },
    )
    expect(pct).toBe(400)
  })

  it('returns null for a missing, stale, or counter-reset baseline', () => {
    expect(cpuPercentBetween(null, BUSY_LOOP.curr)).toBeNull()
    expect(cpuPercentBetween(BUSY_LOOP.prev, null)).toBeNull()
    // Same millisecond: no elapsed time to normalize against.
    expect(
      cpuPercentBetween(
        { usage: 0, onlineCpus: 24, sampledAtMs: 1000 },
        { usage: 10_000_000, onlineCpus: 24, sampledAtMs: 1000 },
      ),
    ).toBeNull()
    // Container recreated between samples.
    expect(
      cpuPercentBetween(
        { usage: 9_000_000_000, onlineCpus: 24, sampledAtMs: 0 },
        { usage: 1_000_000, onlineCpus: 24, sampledAtMs: 2000 },
      ),
    ).toBeNull()
  })

  it('treats a missing onlineCpus as a single core instead of dividing by zero', () => {
    const pct = cpuPercentBetween(
      { usage: 0, sampledAtMs: 0 },
      { usage: 1_000_000_000, sampledAtMs: 1000 },
    )
    expect(pct).toBeCloseTo(100, 0)
  })
})

describe('createCpuRateCalculator', () => {
  it('needs a baseline before it can report a rate', () => {
    const rate = createCpuRateCalculator()
    expect(rate.push(BUSY_LOOP.prev)).toBeNull()
    const pct = rate.push(BUSY_LOOP.curr)
    expect(pct).toBeGreaterThanOrEqual(95)
    expect(pct).toBeLessThanOrEqual(105)
  })

  it('forgets the baseline on reset', () => {
    const rate = createCpuRateCalculator()
    rate.push(BUSY_LOOP.prev)
    rate.reset()
    expect(rate.push(BUSY_LOOP.curr)).toBeNull()
  })
})