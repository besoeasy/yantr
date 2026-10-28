// Package system — host metrics collected natively from procfs.
//
// Yantr runs with --network host and without --pid host, which means the files
// this file reads are the HOST's, not the container's: /proc/stat,
// /proc/meminfo, /proc/loadavg, /proc/uptime, /proc/diskstats and /proc/net/dev
// are not PID- or network-namespaced, and the network namespace is the host's.
// So CPU, memory, load, uptime, disk I/O and per-interface traffic are accurate
// with no bind mount at all.
//
// Two things are not: filesystem usage needs the real filesystems, and hardware
// sensors live in /sys. Those need the read-only binds the installer adds
// (`Volume=/:/host:ro`, `Volume=/sys:/sys:ro`). When they are absent the
// collector reports those sections under Degraded rather than failing — an
// upgraded host that has not re-run the installer keeps working.
package system

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
)

// ---------------------------------------------------------------- data shapes

// HostMetrics is the whole payload behind GET /api/system/metrics. Pointer
// fields and the Ready flags exist so "unknown" is never rendered as a real
// zero: a tile bound for a missing /host bind must show as unavailable, not as
// a healthy-looking 0%.
type HostMetrics struct {
	CPU       CPUStats    `json:"cpu"`
	Mem       MemStats    `json:"mem"`
	Load      LoadStats   `json:"load"`
	UptimeSec float64     `json:"uptimeSec"`
	Net       []NetStats  `json:"net"`
	Disk      []DiskStats `json:"disk"`
	// DiskIO is a per-device I/O rate, reported like Net: empty until the
	// second sample, because a rate derived from cumulative sector counters
	// cannot exist from one reading.
	DiskIO []DiskIOStats `json:"diskIO"`
	// Sensors: one entry per fan/temp slot, paired by slot index within a
	// chip — the only grouping the kernel itself vouches for. Chips that
	// expose only a temperature or only a fan still get an entry; the fields
	// the chip lacks are null rather than zero.
	Sensors []SensorStats `json:"sensors"`
	TempC   *float64      `json:"tempC"`
	FanRPM  *float64      `json:"fanRPM"`
	// Pressure is the kernel's PSI (pressure stall information) for CPU,
	// memory and IO. Point-in-time, so it is valid from the first sample.
	Pressure PressureStats `json:"pressure"`
	// Gpu is one entry per DRM card whose driver exposes the sysfs files a
	// rootless container can read. An empty list is a normal state — an Intel
	// iGPU or a headless server exposes none of these files — so it degrades
	// nothing; only a missing sysfs bind adds the "gpu" degraded key.
	Gpu []GpuStats `json:"gpu"`
	// Board is the DMI identity of the machine (vendor/board/product).
	Board    BoardInfo `json:"board"`
	Degraded []string  `json:"degraded"`
}

// GpuStats is one GPU the kernel exposes statistics for. Every dynamic field
// is a pointer so "driver present, valuation missing" stays null rather than
// collapsing into a healthy-looking zero — amdgpu writes gpu_busy_percent and
// the mem_info_vram pair; the proprietary and i915 drivers write none of them.
type GpuStats struct {
	// Card is the DRM node name ("card0"), the one stable identifier /sys gives.
	Card string `json:"card"`
	// Driver is the uevent DRIVER= value ("amdgpu"), cheap but unambiguous.
	Driver string `json:"driver,omitempty"`
	// BusyPct is the driver's own utilization estimate; null when the file
	// does not exist rather than 0, because a zero and an absent reading are
	// different facts.
	BusyPct        *float64 `json:"busyPct"`
	VramUsedBytes  uint64   `json:"vramUsedBytes"`
	VramTotalBytes uint64   `json:"vramTotalBytes"`
	// TempC and TripC ride along from the driver's hwmon chip when it has one.
	TempC *float64 `json:"tempC"`
	TripC *float64 `json:"tripC,omitempty"`
}

// SensorStats is one slot on one hwmon chip: fan i and temperature i under the
// same chip directory are the pairing the kernel documents with its own
// numbering (fan1_input reads what temp1_input heats). A label file, where the
// driver ships one, travels with the temperature it labels.
type SensorStats struct {
	// Chip is the readable chip name from the name attribute ("nct6775",
	// "nvme", "acpitz"), falling back to the directory name.
	Chip string `json:"chip"`
	// Label is the optional tempN_label text ("Package id 0", "SYSTIN").
	Label string `json:"label,omitempty"`
	// TempC and FanRPM: a chip that only measures reports a null fan; a chip
	// that only spins reports a null temperature.
	TempC  *float64 `json:"tempC"`
	FanRPM *float64 `json:"fanRPM"`
	// TripC is the chip's own threshold for this slot (tempN_crit, falling
	// back to tempN_max, or the zone's trip point). Pointers, not numbers:
	// a board that ships no threshold must not look like one tripping at 0°C.
	TripC *float64 `json:"tripC,omitempty"`
}

type CPUStats struct {
	// Ready is false on the very first sample: a CPU percentage is a rate, so
	// it needs two readings to exist. The UI holds at "—" rather than drawing
	// a flat zero line that reads like a dead host.
	Ready    bool      `json:"ready"`
	TotalPct float64   `json:"totalPct"`
	Cores    []float64 `json:"cores"`
	// Count is the number of cores the kernel is currently reporting. There is
	// deliberately no separate "online" count: a core only appears in
	// /proc/stat while it is online, so Count already is that number, and a
	// second field derived from "did this core do work" would read as
	// activity while claiming to be availability.
	Count     int    `json:"count"`
	ModelName string `json:"modelName,omitempty"`
}

type MemStats struct {
	Ready          bool    `json:"ready"`
	TotalBytes     uint64  `json:"totalBytes"`
	UsedBytes      uint64  `json:"usedBytes"`
	AvailableBytes uint64  `json:"availableBytes"`
	UsedPct        float64 `json:"usedPct"`
	SwapTotalBytes uint64  `json:"swapTotalBytes"`
	SwapUsedBytes  uint64  `json:"swapUsedBytes"`
	// The non-working-set pages the kernel accounts for itself. A bar that
	// reads near-full with a large cache is usually fine; without these the
	// only story the card can tell is how full, never to where it went.
	CachedBytes  uint64 `json:"cachedBytes"`
	BuffersBytes uint64 `json:"buffersBytes"`
	SlabBytes    uint64 `json:"slabBytes"`
}

type LoadStats struct {
	Ready   bool    `json:"ready"`
	One     float64 `json:"one"`
	Five    float64 `json:"five"`
	Fifteen float64 `json:"fifteen"`
	// Running/Total are the process counts that ride along on the same line.
	Running int `json:"running"`
	Total   int `json:"total"`
}

type NetStats struct {
	Iface string  `json:"iface"`
	RxBps float64 `json:"rxBps"`
	TxBps float64 `json:"txBps"`
	// Link is the physical-layer state read from sysfs: negotiated speed in
	// Mbps (0 when the driver does not report one, e.g. virtual interfaces),
	// duplex ("full"/"half"/"unknown"), and operstate ("up"/"down"/...).
	SpeedMbps int    `json:"speedMbps"`
	Duplex    string `json:"duplex,omitempty"`
	OperState string `json:"operState,omitempty"`
}

// PressureEntry is one row of a /proc/pressure/* file: "some" (at least one
// task stalled) and "full" (all tasks stalled) averages plus the cumulative
// stall counter in microseconds.
type PressureEntry struct {
	SomeAvg10  float64 `json:"someAvg10"`
	SomeAvg60  float64 `json:"someAvg60"`
	SomeAvg300 float64 `json:"someAvg300"`
	SomeTotal  uint64  `json:"someTotal"`
	FullAvg10  float64 `json:"fullAvg10"`
	FullAvg60  float64 `json:"fullAvg60"`
	FullAvg300 float64 `json:"fullAvg300"`
	FullTotal  uint64  `json:"fullTotal"`
}

// PressureStats aggregates the three PSI resources the kernel exposes.
type PressureStats struct {
	Ready bool          `json:"ready"`
	CPU   PressureEntry `json:"cpu"`
	Mem   PressureEntry `json:"mem"`
	IO    PressureEntry `json:"io"`
}

// BoardInfo is the machine identity read from DMI under /sys. Fields are
// empty (and the widget renders unavailable) on boards without DMI or when
// the sys bind is missing.
type BoardInfo struct {
	Vendor  string `json:"vendor,omitempty"`
	Board   string `json:"board,omitempty"`
	Product string `json:"product,omitempty"`
	Chassis string `json:"chassis,omitempty"`
}

type DiskStats struct {
	Mount string `json:"mount"`
	// FSType is the filesystem's own name from /proc/mounts ("ext4", "xfs",
	// "btrfs"): enough to tell filesystems apart in a bug report, without
	// leaning on a device node a rootless container cannot honor.
	FSType     string  `json:"fsType"`
	UsedPct    float64 `json:"usedPct"`
	TotalBytes uint64  `json:"totalBytes"`
	FreeBytes  uint64  `json:"freeBytes"`
}

type DiskIOStats struct {
	Device string `json:"device"`
	// Bps are byte rates, derived from the sector counters at their fixed
	// 512-byte size — the same convention sysstat and iostat use.
	ReadBps  float64 `json:"readBps"`
	WriteBps float64 `json:"writeBps"`
	// IOPS are the raw completed-operation counters per second, so an NVMe
	// doing many small reads does not hide behind a headline of bytes.
	ReadsPerSec  float64 `json:"readsPerSec"`
	WritesPerSec float64 `json:"writesPerSec"`
}

// --------------------------------------------------------------- path sources

// sources is every filesystem location the collector touches. It is a struct
// rather than a set of constants so tests can point it at fixture files and
// assert on parsed output without a live host.
type sources struct {
	stat      string
	meminfo   string
	loadavg   string
	uptime    string
	netDev    string
	cpuinfo   string
	diskstats string
	sysRoot   string // /sys, or the host bind when there is one
	hostRoot  string // prefix that makes host paths visible, "" when not bound
}

// hostRoots are probed in order. The installer binds the host filesystem at
// /host; an empty prefix means "we are already looking at the real thing",
// which is the case when Yantr runs without the bind (or in development).
var hostRootCandidates = []string{"/host", ""}

var sysRootCandidates = []string{"/sys", ""}

func detectSources() sources {
	s := sources{
		stat:      "/proc/stat",
		meminfo:   "/proc/meminfo",
		loadavg:   "/proc/loadavg",
		uptime:    "/proc/uptime",
		netDev:    "/proc/net/dev",
		cpuinfo:   "/proc/cpuinfo",
		diskstats: "/proc/diskstats",
	}
	for _, root := range hostRootCandidates {
		if root == "" || isDir(filepath.Join(root, "proc")) {
			s.hostRoot = root
			break
		}
	}
	for _, root := range sysRootCandidates {
		if root == "" || isDir(filepath.Join(root, "class")) {
			s.sysRoot = root
			break
		}
	}
	return s
}

func isDir(path string) bool {
	fi, err := os.Stat(path)
	return err == nil && fi.IsDir()
}

// hostPath resolves a host-absolute path against the detected host root.
func (s sources) hostPath(p string) string {
	if s.hostRoot == "" {
		return p
	}
	return filepath.Join(s.hostRoot, p)
}

// sysPath resolves a /sys-absolute path against the detected sys root.
func (s sources) sysPath(p string) string {
	if s.sysRoot == "" {
		return p
	}
	return filepath.Join(s.sysRoot, p)
}

// ---------------------------------------------------------------- the sampler

type cpuTimes struct{ idle, total uint64 }

type netCounters struct {
	rx, tx uint64
}

// diskIOCounters holds the four counters a rate needs from one /proc/diskstats
// line: completed operation counts (for IOPS) and sector counts (for bytes).
type diskIOCounters struct {
	reads, writes uint64
	readSectors   uint64
	writeSectors  uint64
}

// collector holds the previous reading, because CPU and network rates are
// deltas between two samples and cannot be derived from one.
type collector struct {
	sources sources

	mu        sync.Mutex
	prevCPU   []cpuTimes
	prevCPUAt time.Time
	prevNet   map[string]netCounters
	prevNetAt time.Time
	// prevDiskIO baseline follows the same rule as prevNet: a device that
	// exists in cur but not in prev (it appeared mid-window) is skipped for
	// one tick rather than reported against an unknown baseline.
	prevDiskIO   map[string]diskIOCounters
	prevDiskIOAt time.Time
}

var hostCollector = &collector{sources: detectSources()}

// HostMetrics returns a full sample. Safe for concurrent use; each call
// advances the rate baseline, so the first call after a restart reports CPU
// and network as not-ready.
func HostSnapshot() HostMetrics { return hostCollector.snapshot() }

func (c *collector) snapshot() HostMetrics {
	s := c.sources
	now := time.Now()
	out := HostMetrics{
		Net:      []NetStats{},
		Disk:     []DiskStats{},
		DiskIO:   []DiskIOStats{},
		Sensors:  []SensorStats{},
		Gpu:      []GpuStats{},
		Degraded: []string{},
	}

	cpuCounters, cpuErr := readProcStat(s.stat)
	if cpuErr != nil {
		out.Degraded = append(out.Degraded, "cpu")
	} else {
		out.CPU.Count = len(cpuCounters) - 1 // the aggregate "cpu" line is not a core

		c.mu.Lock()
		if len(c.prevCPU) == len(cpuCounters) && !c.prevCPUAt.IsZero() {
			if elapsed := now.Sub(c.prevCPUAt).Seconds(); elapsed > 0 {
				out.CPU.TotalPct, out.CPU.Cores = cpuPercentages(c.prevCPU, cpuCounters, elapsed)
				out.CPU.Ready = true
			}
		}
		c.prevCPU, c.prevCPUAt = cpuCounters, now
		c.mu.Unlock()
		if out.CPU.Cores == nil {
			// Every array in this payload serializes as [] and never as null,
			// so the UI can map over them without a guard on every field.
			out.CPU.Cores = []float64{}
		}
	}

	if mem, err := readMeminfo(s.meminfo); err == nil {
		out.Mem = mem
	} else {
		out.Degraded = append(out.Degraded, "memory")
	}

	if load, err := readLoadavg(s.loadavg); err == nil {
		out.Load = load
	} else {
		out.Degraded = append(out.Degraded, "load")
	}

	if up, err := readUptime(s.uptime); err == nil {
		out.UptimeSec = up
	}

	netCounters, netErr := readNetDev(s.netDev)
	if netErr != nil {
		out.Degraded = append(out.Degraded, "network")
	} else {
		c.mu.Lock()
		if c.prevNet != nil && now.Sub(c.prevNetAt).Seconds() > 0 {
			elapsed := now.Sub(c.prevNetAt).Seconds()
			ifaces := make([]NetStats, 0, len(netCounters))
			for name, cur := range netCounters {
				if name == "lo" || strings.HasPrefix(name, "veth") || strings.HasPrefix(name, "podman") || strings.HasPrefix(name, "br-") {
					continue
				}
				prev, seen := c.prevNet[name]
				if !seen {
					continue // interface came up after the last sample
				}
				rx, tx := rateBits(cur.rx, prev.rx, elapsed), rateBits(cur.tx, prev.tx, elapsed)
				if rx < 0 || tx < 0 {
					continue // counter reset (interface bounced); a negative rate is a lie
				}
				speed, duplex, operState := readNetLink(s, name)
				ifaces = append(ifaces, NetStats{Iface: name, RxBps: rx, TxBps: tx, SpeedMbps: speed, Duplex: duplex, OperState: operState})
			}
			sort.Slice(ifaces, func(i, j int) bool { return ifaces[i].Iface < ifaces[j].Iface })
			out.Net = ifaces
		}
		c.prevNet, c.prevNetAt = netCounters, now
		c.mu.Unlock()
	}

	if disks, err := readDisks(s); err == nil && len(disks) > 0 {
		out.Disk = disks
	} else {
		out.Degraded = append(out.Degraded, "disk")
	}

	diskCounters, diskErr := readDiskstats(s.diskstats)
	if diskErr != nil {
		out.Degraded = append(out.Degraded, "diskio")
	} else {
		c.mu.Lock()
		if c.prevDiskIO != nil && now.Sub(c.prevDiskIOAt).Seconds() > 0 {
			out.DiskIO = diskIORates(c.prevDiskIO, diskCounters, now.Sub(c.prevDiskIOAt).Seconds())
		}
		c.prevDiskIO, c.prevDiskIOAt = diskCounters, now
		c.mu.Unlock()
	}

	if temp, err := readTemperature(s); err == nil {
		out.TempC = temp
	} else {
		out.Degraded = append(out.Degraded, "temperature")
	}

	if fan, err := readFanSpeed(s); err == nil {
		out.FanRPM = fan
	} else {
		out.Degraded = append(out.Degraded, "fans")
	}

	if sensors, err := readSensors(s); err == nil {
		out.Sensors = sensors
	}
	// No degraded key here on purpose: the aggregates above already degrade
	// with the same missing-/sys condition, and an empty sensor list is a
	// display question the card answers with its existing em-dash rows.

	// CPU model name is static but cheap to re-read; missing file just leaves
	// the field empty rather than degrading the whole CPU card.
	if model := readCPUModelName(s.hostPath("/proc/cpuinfo")); model != "" {
		out.CPU.ModelName = model
	}

	if pressure, err := readPressure(s); err == nil {
		out.Pressure = pressure
		out.Pressure.Ready = true
	} else {
		out.Degraded = append(out.Degraded, "pressure")
	}

	if board, err := readBoard(s); err == nil {
		out.Board = board
	} else {
		out.Degraded = append(out.Degraded, "board")
	}

	// GPU stats are a sysfs read, so a host without the bind reads nothing at
	// all — that is a degradation (the machine may have a GPU it would be
	// lying quietly about), not the same as "no card exposes these files",
	// which is a normal desktop and degrades nothing.
	if _, err := os.Stat(s.sysPath("/class/drm")); err == nil {
		out.Gpu = readGpus(s)
	} else {
		out.Degraded = append(out.Degraded, "gpu")
	}

	return out
}

// ------------------------------------------------------------- /proc parsing

// readProcStat returns the aggregate "cpu" line first, then one entry per
// core. Each entry keeps idle and total separately because the percentage is
// (totalDelta - idleDelta) / totalDelta.
func readProcStat(path string) ([]cpuTimes, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var out []cpuTimes
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		fields := strings.Fields(sc.Text())
		if len(fields) < 5 || !strings.HasPrefix(fields[0], "cpu") {
			continue
		}
		var vals []uint64
		bad := false
		for _, fld := range fields[1:] {
			v, err := strconv.ParseUint(fld, 10, 64)
			if err != nil {
				bad = true
				break
			}
			vals = append(vals, v)
		}
		if bad || len(vals) < 4 {
			continue
		}
		var sum uint64
		for _, v := range vals {
			sum += v
		}
		idle := vals[3] + vals[4] // idle + iowait, the convention top and htop use
		out = append(out, cpuTimes{idle: idle, total: sum})
	}
	if err := sc.Err(); err != nil {
		return nil, err
	}
	if len(out) == 0 {
		return nil, os.ErrNotExist
	}
	return out, nil
}

// cpuPercentages turns two /proc/stat readings into busy percentages. It is
// pure so the arithmetic can be pinned by tests, and it refuses to divide by a
// zero delta: a window with no elapsed jiffies yields ok=false rather than
// +Inf, which is the failure mode the container stats endpoint already had to
// work around (see handlers_containers.go on rootless system_delta).
func cpuPercentages(prev, cur []cpuTimes, elapsedSec float64) (total float64, cores []float64) {
	if len(prev) != len(cur) || len(cur) == 0 || elapsedSec <= 0 {
		return 0, nil
	}
	cores = make([]float64, 0, len(cur)-1)
	for i := range cur {
		dTotal := float64(cur[i].total - prev[i].total)
		dIdle := float64(cur[i].idle - prev[i].idle)
		pct := 0.0
		if dTotal > 0 && cur[i].total >= prev[i].total && cur[i].idle >= prev[i].idle {
			pct = (dTotal - dIdle) / dTotal * 100
		}
		if pct < 0 {
			pct = 0
		}
		if pct > 100 {
			pct = 100
		}
		if i == 0 {
			total = pct
			continue
		}
		cores = append(cores, pct)
	}
	return total, cores
}

func readMeminfo(path string) (MemStats, error) {
	fields, err := readKVFile(path)
	if err != nil {
		return MemStats{}, err
	}
	total := fields["MemTotal"]
	available, ok := fields["MemAvailable"]
	if !ok { // very old kernels without MemAvailable
		available = fields["MemFree"] + fields["Buffers"] + fields["Cached"]
	}
	used := uint64(0)
	if total > available {
		used = total - available
	}
	out := MemStats{
		Ready:          total > 0,
		TotalBytes:     total * 1024,
		UsedBytes:      used * 1024,
		AvailableBytes: available * 1024,
	}
	if total > 0 {
		out.UsedPct = float64(used) / float64(total) * 100
	}
	if swapTotal := fields["SwapTotal"]; swapTotal > 0 {
		out.SwapTotalBytes = swapTotal * 1024
		usedSwap := swapTotal - fields["SwapFree"]
		if swapTotal >= fields["SwapFree"] {
			out.SwapUsedBytes = usedSwap * 1024
		}
	}
	// Cached, Buffers and Slab exist on every kernel the install section names;
	// a zero means the kernel reported zero, not that the row is unknown,
	// because these go into rows that already show the em-dash when the whole
	// meminfo read failed.
	out.CachedBytes = fields["Cached"] * 1024
	out.BuffersBytes = fields["Buffers"] * 1024
	out.SlabBytes = fields["Slab"] * 1024
	return out, nil
}

func readLoadavg(path string) (LoadStats, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return LoadStats{}, err
	}
	fields := strings.Fields(string(raw))
	if len(fields) < 3 {
		return LoadStats{}, os.ErrNotExist
	}
	one, err1 := strconv.ParseFloat(fields[0], 64)
	five, err2 := strconv.ParseFloat(fields[1], 64)
	fifteen, err3 := strconv.ParseFloat(fields[2], 64)
	if err1 != nil || err2 != nil || err3 != nil {
		return LoadStats{}, os.ErrNotExist
	}
	out := LoadStats{Ready: true, One: one, Five: five, Fifteen: fifteen}
	// "0.42 0.58 0.59 1/234 5678" — the run/total pair is the 4th field.
	if len(fields) >= 4 {
		if parts := strings.SplitN(fields[3], "/", 2); len(parts) == 2 {
			out.Running, _ = strconv.Atoi(parts[0])
			out.Total, _ = strconv.Atoi(parts[1])
		}
	}
	return out, nil
}

func readUptime(path string) (float64, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return 0, err
	}
	fields := strings.Fields(string(raw))
	if len(fields) == 0 {
		return 0, os.ErrNotExist
	}
	return strconv.ParseFloat(fields[0], 64)
}

// readNetDev parses /proc/net/dev. Because Yantr shares the host network
// namespace these are the host's interfaces, which is the whole reason the
// installer sets Network=host.
func readNetDev(path string) (map[string]netCounters, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	out := map[string]netCounters{}
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := sc.Text()
		colon := strings.Index(line, ":")
		if colon < 0 {
			continue // the two header lines have no interface name
		}
		name := strings.TrimSpace(line[:colon])
		if name == "" || strings.ContainsAny(name, " \t") {
			continue
		}
		cols := strings.Fields(line[colon+1:])
		if len(cols) < 9 {
			continue
		}
		rx, err1 := strconv.ParseUint(cols[0], 10, 64)
		tx, err2 := strconv.ParseUint(cols[8], 10, 64)
		if err1 != nil || err2 != nil {
			continue
		}
		out[name] = netCounters{rx: rx, tx: tx}
	}
	if err := sc.Err(); err != nil {
		return nil, err
	}
	if len(out) == 0 {
		return nil, os.ErrNotExist
	}
	return out, nil
}

func rateBits(cur, prev uint64, elapsedSec float64) float64 {
	if cur < prev || elapsedSec <= 0 {
		return -1
	}
	return float64(cur-prev) * 8 / elapsedSec
}

// readKVFile parses "Key:   12345 kB" style proc files into raw kilobytes.
func readKVFile(path string) (map[string]uint64, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	out := map[string]uint64{}
	for _, line := range strings.Split(string(raw), "\n") {
		colon := strings.Index(line, ":")
		if colon < 0 {
			continue
		}
		key := strings.TrimSpace(line[:colon])
		value := strings.Fields(line[colon+1:])
		if key == "" || len(value) == 0 {
			continue
		}
		v, err := strconv.ParseUint(value[0], 10, 64)
		if err != nil {
			continue
		}
		out[key] = v
	}
	if len(out) == 0 {
		return nil, os.ErrNotExist
	}
	return out, nil
}

// -------------------------------------------------------------- disk and temp

// skipFSTypes never carry user data and would only add noise to the card.
// The keys are the filesystem TYPE NAMES /proc/mounts writes — matching them
// against statfs's hex magic (the old lookup) matches nothing, which is a bug
// only deactivated code can survive.
var skipFSTypes = map[string]bool{
	"tmpfs": true, "devtmpfs": true, "devfs": true, "proc": true, "sysfs": true,
	"cgroup": true, "cgroup2": true, "overlay": true, "squashfs": true,
	"securityfs": true, "pstore": true, "debugfs": true, "tracefs": true,
	"configfs": true, "fusectl": true, "mqueue": true, "hugetlbfs": true,
	"bpf": true, "autofs": true, "binfmt_misc": true, "nsfs": true,
	// Boot partitions and ephemeral plumbing: real filesystems, not facts a
	// dashboard can act on. fuse.* is matched by prefix below (every fuse
	// mount names an accessory: fuse.portal, fusectl aside).
	"vfat": true, "msdos": true, "devpts": true, "efivarfs": true,
	"ramfs": true, "rpc_pipefs": true, "selinuxfs": true, "swap": true,
	"9p": true, "udf": true, "iso9660": true, "binder": true,
}

// isSkipFSType matches a filesystem type against the skip list, with the "fuse."
// family handled by prefix so fuse.portal variants never squeak through.
func isSkipFSType(fsType string) bool {
	if skipFSTypes[fsType] {
		return true
	}
	return strings.HasPrefix(fsType, "fuse.")
}

// maxReportedDisks bounds the payload the same way maxDiskDevices does: a NAS
// with a NFS map imports the world, and a card reads a handful at most.
const maxReportedDisks = 6

// mountEntry is one parsed /proc/mounts line.
type mountEntry struct {
	device string
	target string
	fsType string
}

// reportableMount is the filter readDisks runs before spending a statfs call:
// pseudo filesystems and boot noise skip; a device that names a block node, a
// network store ("server:/export") or a zfs dataset stays on. Pure, so the
// rules are pinned by tests without a live mount table.
func reportableMount(e mountEntry) bool {
	if isSkipFSType(e.fsType) {
		return false
	}
	dev := e.device
	return strings.HasPrefix(dev, "/dev/") || strings.Contains(dev, "/") || e.fsType == "zfs"
}

// readMounts parses the host's /proc/mounts (the bind makes it the HOST's view;
// without the bind it is the container's own, which is still a self-honest
// list — it simply has less to show).
func readMounts(path string) ([]mountEntry, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	out := []mountEntry{}
	for _, line := range strings.Split(string(raw), "\n") {
		fields := strings.Fields(line)
		if len(fields) < 3 {
			continue
		}
		// Octal escapes (spaces in mount targets) are rare and typically
		// system-owned paths a card skips anyway; a target with escapes is
		// not worth missing the real ones over.
		out = append(out, mountEntry{device: fields[0], target: fields[1], fsType: fields[2]})
	}
	if len(out) == 0 {
		return nil, os.ErrNotExist
	}
	return out, nil
}

// readDisks enumerates the host's real filesystems from /proc/mounts, instead
// of the hardcoded candidate list it used to stat. The candidate list could
// not see beyond "/" plus four conventional paths, so a NAS's /srv/export or a
// dedicated /media disk was invisible no matter how full it ran. The mounts
// file sees everything; the filtering here decides what a dashboard can act on:
// pseudo-filesystems and boot noise skip, everything else stats, and dedupe
// collapses two paths onto one store so the same bytes never double-report.
func readDisks(s sources) ([]DiskStats, error) {
	mounts, err := readMounts(s.hostPath("/proc/mounts"))
	if err != nil {
		return nil, err
	}

	out := make([]DiskStats, 0, 4)
	seenDevice := map[uint64]bool{}

	for _, minfo := range mounts {
		if !reportableMount(minfo) {
			continue
		}
		path := s.hostPath(minfo.target)
		var st syscall.Statfs_t
		if err := syscall.Statfs(path, &st); err != nil {
			continue
		}
		total := st.Blocks * uint64(st.Bsize)
		free := st.Bavail * uint64(st.Bsize)
		if total == 0 {
			continue
		}
		// root and, say, /var are frequently two paths into one filesystem;
		// reporting both would double the same bytes on the card. stat(2)'s
		// device id is the conservative key — it never merges two real
		// filesystems, so the failure mode is a redundant tile rather than a
		// full data disk going invisible. Note that btrfs gives each subvolume
		// its own st_dev, so a btrfs host legitimately reports subvolumes
		// separately even when the numbers match, because filling one does not
		// fill the other.
		fi, err := os.Stat(path)
		if err != nil {
			continue
		}
		if sys, ok := fi.Sys().(*syscall.Stat_t); ok {
			if seenDevice[sys.Dev] {
				continue
			}
			seenDevice[sys.Dev] = true
		}
		used := uint64(0)
		if total > free {
			used = total - free
		}
		out = append(out, DiskStats{
			Mount:      minfo.target,
			FSType:     minfo.fsType,
			UsedPct:    float64(used) / float64(total) * 100,
			TotalBytes: total,
			FreeBytes:  free,
		})
	}

	// Root first: the filesystem "everything else fills up when it fails".
	// The rest by size, big stores before dot-partitions.
	sort.SliceStable(out, func(i, j int) bool {
		if (out[i].Mount == "/" || out[j].Mount == "/") && out[i].Mount != out[j].Mount {
			return out[i].Mount == "/"
		}
		return out[i].TotalBytes > out[j].TotalBytes
	})
	if len(out) > maxReportedDisks {
		out = out[:maxReportedDisks]
	}
	if len(out) == 0 {
		return nil, os.ErrNotExist
	}
	return out, nil
}

// ---------------------------------------------------------------- I/O rates

// diskIODeviceRe matches the whole-disk device names a dashboard understands,
// and nothing else: loop and ram devices are kernel noise, partitions carry
// their parent's counters again (nvme0n1p2 re-counts nvme0n1), and a dm- entry
// is a real filesystem target (LUKS, LVM) even though it stacks on the disk
// beneath it. Keeping dm- mirrors what iostat reports.
var diskIODeviceRe = regexp.MustCompile(
	`^(nvme[0-9]+n[0-9]+|sd[a-z]+|vd[a-z]+|hd[a-z]+|xvd[a-z]+|mmcblk[0-9]+|md[0-9]+|dm-[0-9]+)$`)

// readDiskstats parses /proc/diskstats into per-device counters, kept only for
// names the widget can make sense of. It returns an error when nothing usable
// parsed so the snapshot can mark diskio degraded instead of empty — an empty
// map and a failed read must not look the same to the rate baseline.
func readDiskstats(path string) (map[string]diskIOCounters, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	out := map[string]diskIOCounters{}
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		fields := strings.Fields(sc.Text())
		// "major minor name reads merged sectors ms ..." — 14 fields before
		// the extra discard/flush columns modern kernels append; 13 is the
		// minimum that reaches sectors_written (index 9  after reads_merged).
		if len(fields) < 13 {
			continue
		}
		if !diskIODeviceRe.MatchString(fields[2]) {
			continue
		}
		reads, err1 := strconv.ParseUint(fields[3], 10, 64)
		readSectors, err2 := strconv.ParseUint(fields[5], 10, 64)
		writes, err3 := strconv.ParseUint(fields[7], 10, 64)
		writeSectors, err4 := strconv.ParseUint(fields[9], 10, 64)
		if err1 != nil || err2 != nil || err3 != nil || err4 != nil {
			continue
		}
		out[fields[2]] = diskIOCounters{
			reads:        reads,
			readSectors:  readSectors,
			writes:       writes,
			writeSectors: writeSectors,
		}
	}
	if err := sc.Err(); err != nil {
		return nil, err
	}
	if len(out) == 0 {
		return nil, os.ErrNotExist
	}
	return out, nil
}

// sectorBytes is the kernel's fixed counter unit for /proc/diskstats. It is
// not the device's logical sector size: sectors are counted in 512-byte units
// by convention, matching what iostat reports.
const sectorBytes = 512

// maxDiskDevices bounds the payload: a stacked LVM-on-LUKS setup can report a
// dozen dm- entries, and a widget reads four at most even on a NAS.
const maxDiskDevices = 8

// diskIORates turns two counter maps into per-device byte rates and IOPS. It
// is pure so the arithmetic is pinned by tests, and it follows the net rules:
// a device missing from prev waited its whole life for this baseline, so it is
// skipped for one tick, and a counter that went backwards means the device was
// re-attached — a negative rate is a lie.
func diskIORates(prev, cur map[string]diskIOCounters, elapsedSec float64) []DiskIOStats {
	if len(prev) == 0 || elapsedSec <= 0 {
		return nil
	}
	out := make([]DiskIOStats, 0, len(cur))
	for name, c := range cur {
		p, seen := prev[name]
		if !seen {
			continue
		}
		dRead := int64(c.readSectors) - int64(p.readSectors)
		dWrite := int64(c.writeSectors) - int64(p.writeSectors)
		dReads := int64(c.reads) - int64(p.reads)
		dWrites := int64(c.writes) - int64(p.writes)
		if dRead < 0 || dWrite < 0 || dReads < 0 || dWrites < 0 {
			continue
		}
		out = append(out, DiskIOStats{
			Device:       name,
			ReadBps:      float64(dRead) * sectorBytes / elapsedSec,
			WriteBps:     float64(dWrite) * sectorBytes / elapsedSec,
			ReadsPerSec:  float64(dReads) / elapsedSec,
			WritesPerSec: float64(dWrites) / elapsedSec,
		})
	}
	// Busiest first, so a card that shows the top device finds it at index 0
	// without having to compute what "busiest" means twice.
	sort.Slice(out, func(i, j int) bool {
		ti := out[i].ReadBps + out[i].WriteBps
		tj := out[j].ReadBps + out[j].WriteBps
		if ti != tj {
			return ti > tj
		}
		return out[i].Device < out[j].Device
	})
	if len(out) > maxDiskDevices {
		out = out[:maxDiskDevices]
	}
	return out
}

// maxPlausibleFanRPM is the sanity ceiling for a hwmon fan reading. Desktop
// fans top out around 3000 RPM and 1U blowers near 10000; anything above is a
// sensor reading garbage on an unconnected pin, and reporting it would be
// worse than reporting no fan at all.
const maxPlausibleFanRPM = 12000

// readFanSpeed returns the fastest fan attached to the host, in RPM.
//
// Unlike temperature, a genuine 0 is a real observation — plenty of boards
// idle their fans at 0 — so it is reported rather than discarded; the nil
// return is reserved for "no fan sensor readable", which is the degraded
// state. Far from every board exposes hwmon fans, so erroring quietly is the
// expected path on headless servers no less than on fanless SBCs.
func readFanSpeed(s sources) (*float64, error) {
	inputs, err := filepath.Glob(s.sysPath("/class/hwmon/hwmon*/fan*_input"))
	if err != nil || len(inputs) == 0 {
		return nil, os.ErrNotExist
	}
	var best float64 = -1
	for _, path := range inputs {
		raw, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		rpm, err := strconv.ParseFloat(strings.TrimSpace(string(raw)), 64)
		if err != nil || rpm < 0 || rpm > maxPlausibleFanRPM {
			continue
		}
		if rpm > best {
			best = rpm
		}
	}
	if best < 0 {
		return nil, os.ErrNotExist
	}
	return &best, nil
}

// readCelsius parses one milliCelsius temp file into the plausible window.
// Everything outside it reports as missing rather than as a believable-looking
// 0°C or 1000°C, which is the same deal every sensor reader in this file makes.
func readCelsius(path string) (*float64, bool) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, false
	}
	milli, err := strconv.ParseFloat(strings.TrimSpace(string(raw)), 64)
	if err != nil {
		return nil, false
	}
	c := milli / 1000
	if c <= 0 || c >= 120 {
		return nil, false
	}
	return &c, true
}

// readTemperature walks the ACPI thermal zones and the hwmon temperature
// inputs alike and returns the hottest plausible reading. It used to return
// the first zone it happened to find, which made the dashboard's headline
// temperature a lottery of enumeration order — the number a human reacts to
// is the hottest one, since that is the reading that can hurt the hardware.
func readTemperature(s sources) (*float64, error) {
	var candidates []string
	if zones, err := filepath.Glob(s.sysPath("/class/thermal/thermal_zone*/temp")); err == nil {
		candidates = append(candidates, zones...)
	}
	if hwmons, err := filepath.Glob(s.sysPath("/class/hwmon/hwmon*/temp*_input")); err == nil {
		candidates = append(candidates, hwmons...)
	}
	var hottest *float64
	for _, path := range candidates {
		c, ok := readCelsius(path)
		if !ok {
			continue
		}
		if hottest == nil || *c > *hottest {
			hottest = c
		}
	}
	if hottest == nil {
		return nil, os.ErrNotExist
	}
	return hottest, nil
}

// maxSensorReadings bounds the payload: a many-slot Super I/O chip plus GPU
// and NVMe sensors can list a dozen entries, and a card reads four at most.
const maxSensorReadings = 8

// readSensors returns one entry per fan/temp slot, paired by index within the
// same hwmon chip, plus the ACPI thermal zones as temperature-only entries.
//
// Slot-index pairing is a deliberate convention, not an inference from
// proximity: the kernel numbers fanN_input and tempN_input on a chip in the
// order the driver registers them, which on multi-slot Super I/O chips is the
// correlation board designers ship. It is not a physical guarantee — a chip
// whose fan 1 cools the thing temp 2 measures will pair wrong either way, and
// inventing cross-slot pairs would claim precision the kernel does not offer.
// Chips that expose only one side keep their single field and null out the
// other, so a hot NVMe with a stopped fan is always distinguishable from a
// cool CPU that is actively being pushed air.
func readSensors(s sources) ([]SensorStats, error) {
	out := []SensorStats{}

	chips, _ := filepath.Glob(s.sysPath("/class/hwmon/hwmon*"))
	for _, chipDir := range chips {
		chip := filepath.Base(chipDir)
		if raw, err := os.ReadFile(filepath.Join(chipDir, "name")); err == nil && strings.TrimSpace(string(raw)) != "" {
			chip = strings.TrimSpace(string(raw))
		}

		// Slot maps first, pairing second: file enumeration order must not
		// decide which fan sits next to which temperature.
		temps := map[int]*float64{}
		labels := map[int]string{}
		fans := map[int]*float64{}
		trips := map[int]*float64{}
		for _, path := range globSlotPaths(chipDir, "temp*_input") {
			n := slotIndex(path)
			if c, ok := readCelsius(path); ok {
				temps[n] = c
			}
			// The path itself ends in _input; the label is its sibling file.
			if label, ok := readLabel(strings.TrimSuffix(path, "_input") + "_label"); ok {
				labels[n] = label
			}
			// The chip's own trip for this slot: tempN_crit first (the number
			// the kernel throttles at) then tempN_max (the advisory warning).
			// A threshold is a number the BOARD shipped, so it may be higher
			// than the plausibility band a live reading must fit; it gets its
			// own parser rather than the reading one.
			if trip, ok := readThresholdC(filepath.Join(chipDir, fmt.Sprintf("temp%d_crit", n))); ok {
				trips[n] = trip
			} else if trip, ok := readThresholdC(filepath.Join(chipDir, fmt.Sprintf("temp%d_max", n))); ok {
				trips[n] = trip
			}
		}
		for _, path := range globSlotPaths(chipDir, "fan*_input") {
			n := slotIndex(path)
			raw, err := os.ReadFile(path)
			if err != nil {
				continue
			}
			rpm, err := strconv.ParseFloat(strings.TrimSpace(string(raw)), 64)
			if err != nil || rpm < 0 || rpm > maxPlausibleFanRPM {
				continue
			}
			fans[n] = &rpm
		}

		for n, rpm := range fans {
			entry := SensorStats{Chip: chip, FanRPM: rpm, TripC: trips[n]}
			if temp, ok := temps[n]; ok {
				entry.TempC = temp
				entry.Label = labels[n]
			}
			out = append(out, entry)
		}
		for n, temp := range temps {
			if _, paired := fans[n]; paired {
				continue
			}
			out = append(out, SensorStats{Chip: chip, Label: labels[n], TempC: temp, TripC: trips[n]})
		}
	}

	// ACPI zones have no fans of their own; they project as temperature-only
	// entries so "acpitz" still shows where its heat sits. A zone ships its
	// thresholds as trip_point_N_temp files; the hottest trip is the one the
	// temperature is racing.
	if zones, err := filepath.Glob(s.sysPath("/class/thermal/thermal_zone*")); err == nil {
		for _, zoneDir := range zones {
			temp, ok := readCelsius(filepath.Join(zoneDir, "temp"))
			if !ok {
				continue
			}
			name := filepath.Base(zoneDir)
			if raw, err := os.ReadFile(filepath.Join(zoneDir, "type")); err == nil && strings.TrimSpace(string(raw)) != "" {
				name = strings.TrimSpace(string(raw))
			}
			entry := SensorStats{Chip: name, TempC: temp}
			for _, tripPath := range globSlotPaths(zoneDir, "trip_point_*_temp") {
				if trip, ok := readThresholdC(tripPath); ok && (entry.TripC == nil || *trip > *entry.TripC) {
					entry.TripC = trip
				}
			}
			out = append(out, entry)
		}
	}

	if len(out) == 0 {
		return nil, os.ErrNotExist
	}

	// Paired slots first (the card's reason to exist), then hottest, then the
	// loudest fan-only chips; name as the deterministic tie-break.
	sort.SliceStable(out, func(i, j int) bool {
		pi, pj := out[i].TempC != nil && out[i].FanRPM != nil, out[j].TempC != nil && out[j].FanRPM != nil
		if pi != pj {
			return pi
		}
		ti, tj := tempKey(out[i]), tempKey(out[j])
		if ti != tj {
			return ti > tj
		}
		fi, fj := fanKey(out[i]), fanKey(out[j])
		if fi != fj {
			return fi > fj
		}
		return out[i].Chip < out[j].Chip
	})
	if len(out) > maxSensorReadings {
		out = out[:maxSensorReadings]
	}
	return out, nil
}

// tempKey and fanKey order entries in the sort above, treating a missing
// reading as the coldest and the slowest respectively so singles never beat
// a paired slot on their half of the key alone.
func tempKey(s SensorStats) float64 {
	if s.TempC == nil {
		return -1
	}
	return *s.TempC
}

func fanKey(s SensorStats) float64 {
	if s.FanRPM == nil {
		return -1
	}
	return *s.FanRPM
}

// globSlotPaths returns the files matching one input pattern in a chip.
func globSlotPaths(chipDir, pattern string) []string {
	paths, err := filepath.Glob(filepath.Join(chipDir, pattern))
	if err != nil {
		return nil
	}
	return paths
}

// slotIndex extracts the N from a tempN_input / fanN_input path, best-effort:
// a name that carries no index folds into slot 0 rather than being dropped,
// since the convention there is still one slot per reading.
func slotIndex(path string) int {
	base := strings.TrimSuffix(filepath.Base(path), "_input")
	digits := strings.Map(func(r rune) rune {
		if r >= '0' && r <= '9' {
			return r
		}
		return -1
	}, base)
	n, _ := strconv.Atoi(digits)
	return n
}

// readLabel reads an optional tempN_label. Labels are driver-authored text;
// anything unreadable just leaves the label empty rather than hinting at a
// broken measurement.
func readLabel(path string) (string, bool) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return "", false
	}
	label := strings.TrimSpace(string(raw))
	if label == "" {
		return "", false
	}
	return label, true
}
