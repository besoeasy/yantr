package system

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// writeFixture drops content into a temp dir and returns its full path.
func writeFixture(t *testing.T, name, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write %s: %v", name, err)
	}
	return path
}

const procStatFixture = `cpu  1000 20 300 8000 100 0 50 0 0 0
cpu0 500 10 150 4000 50 0 25 0 0 0
cpu1 500 10 150 4000 50 0 25 0 0 0
intr 12345
ctxt 6789
btime 1700000000
processes 4242
procs_running 2
procs_blocked 0
`

func TestReadProcStat(t *testing.T) {
	got, err := readProcStat(writeFixture(t, "stat", procStatFixture))
	if err != nil {
		t.Fatalf("readProcStat: %v", err)
	}
	if len(got) != 3 { // aggregate + two cores
		t.Fatalf("got %d entries, want 3", len(got))
	}
	// aggregate: total 9470, idle 8000+100
	if got[0].total != 9470 || got[0].idle != 8100 {
		t.Errorf("aggregate total/idle = %d/%d, want 9470/8100", got[0].total, got[0].idle)
	}
	// cpu0: total 4735, idle 4050
	if got[1].total != 4735 || got[1].idle != 4050 {
		t.Errorf("cpu0 total/idle = %d/%d, want 4735/4050", got[1].total, got[1].idle)
	}
}

func TestReadProcStatMissing(t *testing.T) {
	if _, err := readProcStat(filepath.Join(t.TempDir(), "nope")); err == nil {
		t.Fatal("expected an error for a missing /proc/stat")
	}
}

func TestCPUPercentages(t *testing.T) {
	// A second window: 1000 more total jiffies, 200 of them idle → 80% busy.
	prev := []cpuTimes{{idle: 8100, total: 9470}, {idle: 4050, total: 4735}, {idle: 4050, total: 4735}}
	cur := []cpuTimes{{idle: 8300, total: 10470}, {idle: 4150, total: 5235}, {idle: 4150, total: 5235}}

	total, cores := cpuPercentages(prev, cur, 2)
	if math.Abs(total-80) > 0.001 {
		t.Errorf("total = %.3f, want 80", total)
	}
	if len(cores) != 2 {
		t.Fatalf("got %d cores, want 2", len(cores))
	}
	for i, c := range cores {
		if math.Abs(c-80) > 0.001 {
			t.Errorf("core %d = %.3f, want 80", i, c)
		}
	}
}

// The rootless container-stats bug this guards against: when no jiffies elapsed
// between two readings, dividing by the zero delta produces +Inf, which the UI
// would render as a pinned meter.
func TestCPUPercentagesZeroDeltaIsNotInfinite(t *testing.T) {
	same := []cpuTimes{{idle: 8100, total: 9470}, {idle: 4050, total: 4735}}
	total, cores := cpuPercentages(same, same, 1)
	if math.IsInf(total, 0) || math.IsNaN(total) {
		t.Fatalf("total = %v, want a finite number", total)
	}
	for i, c := range cores {
		if math.IsInf(c, 0) || math.IsNaN(c) {
			t.Fatalf("core %d = %v, want a finite number", i, c)
		}
	}
}

func TestCPUPercentagesMismatchedCPUCount(t *testing.T) {
	// A core disappearing mid-window (hotplug) must not be reported.
	prev := []cpuTimes{{idle: 8100, total: 9470}, {idle: 4050, total: 4735}}
	cur := []cpuTimes{{idle: 8300, total: 10470}}
	if _, cores := cpuPercentages(prev, cur, 1); len(cores) != 0 {
		t.Errorf("got %d cores, want 0 on a CPU-count change", len(cores))
	}
}

const meminfoFixture = `MemTotal:       16000000 kB
MemFree:         1000000 kB
MemAvailable:   12000000 kB
Buffers:          200000 kB
Cached:          3000000 kB
SwapTotal:       4000000 kB
SwapFree:        3000000 kB
`

func TestReadMeminfo(t *testing.T) {
	got, err := readMeminfo(writeFixture(t, "meminfo", meminfoFixture))
	if err != nil {
		t.Fatalf("readMeminfo: %v", err)
	}
	if !got.Ready {
		t.Error("Ready = false, want true")
	}
	if got.TotalBytes != 16000000*1024 {
		t.Errorf("TotalBytes = %d, want %d", got.TotalBytes, 16000000*1024)
	}
	// used = total - available, so caches are not double-counted as used.
	if got.UsedBytes != 4000000*1024 {
		t.Errorf("UsedBytes = %d, want %d", got.UsedBytes, 4000000*1024)
	}
	if want := 25.0; math.Abs(got.UsedPct-25) > 0.001 {
		t.Errorf("UsedPct = %.3f, want %.3f", got.UsedPct, want)
	}
	if got.SwapTotalBytes != 4000000*1024 || got.SwapUsedBytes != 1000000*1024 {
		t.Errorf("swap = %d/%d, want %d/%d", got.SwapUsedBytes, got.SwapTotalBytes, 1000000*1024, 4000000*1024)
	}
}

func TestReadMeminfoWithoutMemAvailable(t *testing.T) {
	// Kernels before 3.14 have no MemAvailable; the fallback must not treat the
	// whole machine as used.
	fixture := "MemTotal:       1000 kB\nMemFree:         400 kB\nBuffers:         100 kB\nCached:          300 kB\nSwapTotal:          0 kB\nSwapFree:           0 kB\n"
	got, err := readMeminfo(writeFixture(t, "meminfo", fixture))
	if err != nil {
		t.Fatalf("readMeminfo: %v", err)
	}
	if got.UsedBytes != 200*1024 {
		t.Errorf("UsedBytes = %d, want %d", got.UsedBytes, 200*1024)
	}
}

func TestReadLoadavg(t *testing.T) {
	got, err := readLoadavg(writeFixture(t, "loadavg", "0.52 1.08 0.99 3/412 12345\n"))
	if err != nil {
		t.Fatalf("readLoadavg: %v", err)
	}
	if !got.Ready || got.One != 0.52 || got.Five != 1.08 || got.Fifteen != 0.99 {
		t.Errorf("load = %+v, want 0.52/1.08/0.99", got)
	}
	if got.Running != 3 || got.Total != 412 {
		t.Errorf("procs = %d/%d, want 3/412", got.Running, got.Total)
	}
}

func TestReadUptime(t *testing.T) {
	got, err := readUptime(writeFixture(t, "uptime", "4820113.42 3221456.11\n"))
	if err != nil {
		t.Fatalf("readUptime: %v", err)
	}
	if math.Abs(got-4820113.42) > 0.001 {
		t.Errorf("uptime = %.3f, want 4820113.42", got)
	}
}

const netDevFixture = `Inter-|   Receive                                                |  Transmit
 face |bytes    packets errs drop fifo frame compressed multicast|bytes    packets errs drop fifo colls carrier compressed
    lo: 1000    10    0    0    0     0          0         0     1000    10    0    0    0     0       0          0
  eth0: 500000  400   0    0    0     0          0         0    250000  200   0    0    0     0       0          0
`

func TestReadNetDev(t *testing.T) {
	got, err := readNetDev(writeFixture(t, "netdev", netDevFixture))
	if err != nil {
		t.Fatalf("readNetDev: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("got %d interfaces, want 2", len(got))
	}
	if got["eth0"].rx != 500000 || got["eth0"].tx != 250000 {
		t.Errorf("eth0 = %+v, want rx=500000 tx=250000", got["eth0"])
	}
}

func TestReadNetDevMalformedLine(t *testing.T) {
	fixture := "Inter-|   Receive\n face |bytes\ngarbage line without colon\n"
	if _, err := readNetDev(writeFixture(t, "netdev", fixture)); err == nil {
		t.Fatal("expected an error when no interface line parses")
	}
}

func TestRateBits(t *testing.T) {
	// 1 MB in 2 seconds → 4 Mbit/s.
	if got := rateBits(500000+1000000, 500000, 2); math.Abs(got-4000000) > 1 {
		t.Errorf("rate = %v, want 4000000", got)
	}
	// A counter that went backwards means the interface bounced.
	if got := rateBits(10, 20, 1); got != -1 {
		t.Errorf("rate on counter reset = %v, want -1", got)
	}
}

func TestReadDisks(t *testing.T) {
	// The temp dir is on a tmpfs, which readDisks skips as noise — remove that
	// entry so the arithmetic and dedup are actually exercised here.
	skipFSTypes["tmpfs"] = false
	t.Cleanup(func() { skipFSTypes["tmpfs"] = true })

	original := diskCandidates
	diskCandidates = []string{t.TempDir(), t.TempDir()}
	t.Cleanup(func() { diskCandidates = original })

	got, err := readDisks(sources{})
	if err != nil {
		t.Fatalf("readDisks: %v", err)
	}
	// Both candidates live on the same filesystem (two dirs under one tmpfs),
	// so deduping by st_dev must collapse them into a single entry — the same
	// thing / and /home do on an ext4 host.
	if len(got) != 1 {
		t.Fatalf("got %d disks, want 1 after deduping one filesystem", len(got))
	}
	if got[0].FSType == "" {
		t.Error("FSType is empty, want the hexadecimal filesystem magic")
	}
	if got[0].TotalBytes == 0 {
		t.Errorf("TotalBytes = 0, want a real size")
	}
	if got[0].UsedPct < 0 || got[0].UsedPct > 100 {
		t.Errorf("UsedPct = %v, want a percentage", got[0].UsedPct)
	}
}

func TestReadDisksWithoutHostBind(t *testing.T) {
	// No /host bind on an existing install: the candidate paths do not exist
	// and the section must degrade, not report a made-up 0%.
	got, err := readDisks(sources{hostRoot: filepath.Join(t.TempDir(), "not-bound")})
	if err == nil {
		t.Fatalf("readDisks returned %+v, want an error when nothing resolves", got)
	}
}

// The first sample has no rate baseline, so CPU and network must report
// not-ready rather than a zero that the UI would draw as an idle machine.
func TestSnapshotFirstSampleHasNoRates(t *testing.T) {
	dir := t.TempDir()
	c := &collector{sources: sources{
		stat:     writeFixture(t, "stat", procStatFixture),
		meminfo:  writeFixture(t, "meminfo", meminfoFixture),
		loadavg:  writeFixture(t, "loadavg", "0.52 1.08 0.99 3/412 12345\n"),
		uptime:   writeFixture(t, "uptime", "4820113.42 3221456.11\n"),
		netDev:   writeFixture(t, "netdev", netDevFixture),
		hostRoot: dir,
		sysRoot:  dir,
	}}

	got := c.snapshot()
	if got.CPU.Ready {
		t.Error("CPU.Ready = true on the first sample, want false until a second reading exists")
	}
	if len(got.CPU.Cores) != 0 {
		t.Errorf("got %+v on the first sample, want an empty array before any rate exists", got.CPU.Cores)
	}
	if got.CPU.Count != 2 {
		t.Errorf("CPU.Count = %d, want 2", got.CPU.Count)
	}
	if !got.Mem.Ready || !got.Load.Ready {
		t.Errorf("mem/load not ready: %+v %+v", got.Mem, got.Load)
	}
	if got.UptimeSec == 0 {
		t.Error("UptimeSec = 0, want the parsed uptime")
	}
	if len(got.Net) != 0 {
		t.Errorf("got %+v on the first sample, want no rates yet", got.Net)
	}

	// A second sample against an unchanged /proc now has a baseline: wall-clock
	// time passed, but no jiffies did. That is a genuinely idle machine, so 0%
	// is the honest answer — what must never happen is +Inf or NaN reaching
	// JSON, which is the failure mode cpuPercentages guards against.
	second := c.snapshot()
	if !second.CPU.Ready {
		t.Error("CPU.Ready = false on the second sample, want true")
	}
	if second.CPU.TotalPct != 0 {
		t.Errorf("CPU.TotalPct = %v on an unchanged /proc, want 0", second.CPU.TotalPct)
	}
	if math.IsNaN(second.CPU.TotalPct) || math.IsInf(second.CPU.TotalPct, 0) {
		t.Errorf("CPU.TotalPct = %v, want a finite number", second.CPU.TotalPct)
	}
}

func TestSnapshotDegradesWhenHostSourcesMissing(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "not-mounted")
	c := &collector{sources: sources{
		stat: missing, meminfo: missing, loadavg: missing,
		uptime: missing, netDev: missing,
		hostRoot: missing, sysRoot: missing,
	}}

	got := c.snapshot()
	for _, want := range []string{"cpu", "memory", "load", "network", "disk", "temperature"} {
		found := false
		for _, d := range got.Degraded {
			if d == want {
				found = true
			}
		}
		if !found {
			t.Errorf("Degraded = %v, missing %q", got.Degraded, want)
		}
	}
	// Slices must serialize as [] and never as null, or the UI has to guard
	// against a null before it can map over anything.
	if got.Net == nil || got.Disk == nil || got.DiskIO == nil || got.Sensors == nil || got.Degraded == nil || got.CPU.Cores != nil {
		t.Errorf("nil slices would serialize as JSON null: %+v", got)
	}
	if got.TempC != nil {
		t.Errorf("TempC = %v, want nil when no sensor is readable", *got.TempC)
	}
}

// hwmonFile drops file content into a hwmon chip directory, creating the chip
// as needed, and returns its chip root for assertions.
func hwmonFile(t *testing.T, dir, chip, file, content string) {
	chipDir := filepath.Join(dir, "class", "hwmon", chip)
	if err := os.MkdirAll(chipDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(chipDir, file), []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}

func hwmonFan(t *testing.T, dir, chip, file string, rpm int) {
	hwmonFile(t, dir, chip, file, fmt.Sprintf("%d\n", rpm))
}

func TestReadFanSpeedPicksTheFastestPlausibleFan(t *testing.T) {
	dir := t.TempDir()
	hwmonFan(t, dir, "hwmon1", "fan1_input", 3036) // acpi_fan, the faster of the two real ones
	hwmonFan(t, dir, "hwmon2", "fan1_input", 0)    // a stopped fan is a real observation, not a dead sensor
	hwmonFan(t, dir, "hwmon3", "fan1_input", 15000) // above the ceiling: an unconnected pin, skipped

	got, err := readFanSpeed(sources{sysRoot: dir})
	if err != nil {
		t.Fatalf("readFanSpeed: %v", err)
	}
	if math.Abs(*got-3036) > 0.001 {
		t.Errorf("fan = %v, want 3036", *got)
	}
}

func TestReadFanSpeedAllStoppedIsStillAReading(t *testing.T) {
	dir := t.TempDir()
	hwmonFan(t, dir, "hwmon1", "fan1_input", 0)

	got, err := readFanSpeed(sources{sysRoot: dir})
	if err != nil {
		t.Fatalf("readFanSpeed: %v", err)
	}
	if *got != 0 {
		t.Errorf("fan = %v, want 0 (fans idle at zero on real boards)", *got)
	}
}

func TestReadFanSpeedNoSensorIsAnError(t *testing.T) {
	if _, err := readFanSpeed(sources{sysRoot: t.TempDir()}); err == nil {
		t.Fatal("expected an error when no fan input exists")
	}
}

func TestReadTemperatureReturnsTheHottest(t *testing.T) {
	dir := t.TempDir()
	hwmonFile(t, dir, "hwmon1", "temp1_input", "42500\n")  // 42.5°C: cool, but not the headline
	hwmonFile(t, dir, "hwmon2", "temp1_input", "62500\n")  // 62.5°C: the reading that can hurt hardware
	hwmonFile(t, dir, "hwmon3", "temp1_input", "150000\n") // 150°C: implausible, skipped

	got, err := readTemperature(sources{sysRoot: dir})
	if err != nil {
		t.Fatalf("readTemperature: %v", err)
	}
	if math.Abs(*got-62.5) > 0.001 {
		t.Errorf("temp = %v, want 62.5 (hottest plausible)", *got)
	}
}

// A desktop board's Super I/O chip: two temperature slots, one fan slot. Fan 1
// pairs with temp 1 by index and carries its label; temp 2 stays an unpaired
// single. A hot NVMe with a stopped fan and a cool CPU with spinning fans must
// not blur into one value.
func TestReadSensorsPairsBySlotWithinAChip(t *testing.T) {
	dir := t.TempDir()
	hwmonFile(t, dir, "hwmon1", "name", "nct6775\n")
	hwmonFile(t, dir, "hwmon1", "fan1_input", "1200\n")
	hwmonFile(t, dir, "hwmon1", "temp1_input", "42500\n")
	hwmonFile(t, dir, "hwmon1", "temp1_label", "SYSTIN\n")
	hwmonFile(t, dir, "hwmon1", "temp2_input", "62500\n")
	hwmonFile(t, dir, "hwmon1", "temp2_label", "CPUTIN\n")

	got, err := readSensors(sources{sysRoot: dir})
	if err != nil {
		t.Fatalf("readSensors: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("got %+v, want 2 entries", got)
	}
	paired := got[0]
	if paired.Chip != "nct6775" || paired.Label != "SYSTIN" ||
		paired.FanRPM == nil || *paired.FanRPM != 1200 ||
		paired.TempC == nil || *paired.TempC != 42.5 {
		t.Errorf("paired slot = %+v, want nct6775 SYSTIN 42.5°C @ 1200 RPM", paired)
	}
	single := got[1]
	if single.TempC == nil || *single.TempC != 62.5 || single.FanRPM != nil || single.Label != "CPUTIN" {
		t.Errorf("unpaired temp slot = %+v, want CPUTIN 62.5°C with a null fan", single)
	}
}

func TestReadSensorsFanOnlyChip(t *testing.T) {
	dir := t.TempDir()
	hwmonFile(t, dir, "hwmon1", "name", "acpi_fan\n")
	hwmonFile(t, dir, "hwmon1", "fan1_input", "3036\n")

	got, err := readSensors(sources{sysRoot: dir})
	if err != nil {
		t.Fatalf("readSensors: %v", err)
	}
	if len(got) != 1 || got[0].FanRPM == nil || *got[0].FanRPM != 3036 || got[0].TempC != nil {
		t.Errorf("fan-only chip = %+v, want acpi_fan 3036 RPM with a null temp", got)
	}
	_ = got[0].Chip
}

func TestReadSensorsIncludesThermalZonesAndKeepsOrder(t *testing.T) {
	dir := t.TempDir()
	hwmonFile(t, dir, "hwmon1", "name", "k10temp\n")
	hwmonFile(t, dir, "hwmon1", "temp1_input", "42500\n")

	zone := filepath.Join(dir, "class", "thermal", "thermal_zone0")
	if err := os.MkdirAll(zone, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(zone, "temp"), []byte("52000\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(zone, "type"), []byte("acpitz\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	got, err := readSensors(sources{sysRoot: dir})
	if err != nil {
		t.Fatalf("readSensors: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("got %+v, want 2 entries", got)
	}
	// Hottest first among singles: the zone at 52 beats k10temp at 42.5.
	if got[0].Chip != "acpitz" || *got[0].TempC != 52 || got[0].FanRPM != nil {
		t.Errorf("first = %+v, want acpitz 52°C temp-only", got[0])
	}
	if got[1].Chip != "k10temp" || *got[1].TempC != 42.5 {
		t.Errorf("second = %+v, want k10temp 42.5°C", got[1])
	}
}

func TestReadSensorsNoSensorsIsAnError(t *testing.T) {
	if _, err := readSensors(sources{sysRoot: t.TempDir()}); err == nil {
		t.Fatal("expected an error when no chip or zone is readable")
	}
}

func TestReadTemperatureRejectsImplausibleReadings(t *testing.T) {
	dir := t.TempDir()
	zone := filepath.Join(dir, "class", "thermal", "thermal_zone0")
	if err := os.MkdirAll(zone, 0o755); err != nil {
		t.Fatal(err)
	}
	// A dead sensor reporting 0 m°C must read as missing, not as a freezing CPU.
	if err := os.WriteFile(filepath.Join(zone, "temp"), []byte("0\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if got, err := readTemperature(sources{sysRoot: dir}); err == nil {
		t.Fatalf("readTemperature = %v, want an error for a 0 reading", *got)
	}

	if err := os.WriteFile(filepath.Join(zone, "temp"), []byte("47500\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	got, err := readTemperature(sources{sysRoot: dir})
	if err != nil {
		t.Fatalf("readTemperature: %v", err)
	}
	if math.Abs(*got-47.5) > 0.001 {
		t.Errorf("temp = %v, want 47.5", *got)
	}
}

func TestHostPathResolution(t *testing.T) {
	if got := (sources{hostRoot: "/host"}).hostPath("/etc"); got != "/host/etc" {
		t.Errorf("hostPath = %q, want /host/etc", got)
	}
	// With no bind the prefix is empty and the host path is already the real
	// path, which is how a pre-upgrade install keeps working.
	if got := (sources{}).hostPath("/etc"); got != "/etc" {
		t.Errorf("hostPath = %q, want /etc", got)
	}
	if got := (sources{sysRoot: "/host/sys"}).sysPath("/class/thermal"); !strings.HasPrefix(got, "/host/sys") {
		t.Errorf("sysPath = %q, want a /host/sys prefix", got)
	}
}

const diskstatsFixture = `   8       0 nvme0n1 2821 0 625244 301 189 214 13994 311 0 0 0 0 0 0
   8       1 nvme0n1p1 2821 0 625244 301 189 214 13994 311 0 0 0 0 0 0
   7       0 loop0 0 0 0 0 0 0 0 0 0 0 0 0 0 0
   9       0 md0 412 0 100000 200 88 0 2000 40 0 0 0 0 0 0
 253       0 dm-0 9000 0 200000 400 1000 0 50000 60 0 0 0 0 0 0
   1       0 ram0 0 0 0 0 0 0 0 0 0 0 0 0 0 0
   8      16 sda 10 0 100 1 5 0 40 2 0 0 0 0 0 0
`

func TestReadDiskstatsFiltersToDeviceNames(t *testing.T) {
	got, err := readDiskstats(writeFixture(t, "diskstats", diskstatsFixture))
	if err != nil {
		t.Fatalf("readDiskstats: %v", err)
	}
	// Partitions re-count their parent (nvme0n1p1 duplicates nvme0n1), loop and
	// ram devices are kernel noise. Only whole disks and dm mappings remain.
	for _, name := range []string{"nvme0n1", "md0", "dm-0", "sda"} {
		if _, ok := got[name]; !ok {
			t.Errorf("device %q missing from %+v", name, keysOfDiskIO(got))
		}
	}
	for _, name := range []string{"nvme0n1p1", "loop0", "ram0"} {
		if _, ok := got[name]; ok {
			t.Errorf("noise device %q must be filtered out", name)
		}
	}
	if got["nvme0n1"].reads != 2821 || got["nvme0n1"].readSectors != 625244 ||
		got["nvme0n1"].writes != 189 || got["nvme0n1"].writeSectors != 13994 {
		t.Errorf("nvme0n1 = %+v", got["nvme0n1"])
	}
}

func TestReadDiskstatsRejectsGarbage(t *testing.T) {
	if _, err := readDiskstats(writeFixture(t, "diskstats", "nothing here\n")); err == nil {
		t.Fatal("expected an error when no device line parses")
	}
}

func TestDiskIORates(t *testing.T) {
	prev := map[string]diskIOCounters{
		"nvme0n1": {reads: 1000, readSectors: 20000, writes: 100, writeSectors: 2000},
		"seen":    {reads: 50, readSectors: 100, writes: 0, writeSectors: 0},
	}
	cur := map[string]diskIOCounters{
		"nvme0n1":   {reads: 2000, readSectors: 30512, writes: 160, writeSectors: 4000},
		"seen":      {reads: 60, readSectors: 120, writes: 0, writeSectors: 0},
		"reset":     {reads: 10, readSectors: 99999, writes: 0, writeSectors: 0},
		"midwindow": {reads: 70, readSectors: 140, writes: 0, writeSectors: 0},
	}
	prev["reset"] = diskIOCounters{reads: 5, readSectors: 100000, writes: 0, writeSectors: 0}
	// "midwindow" deliberately has no prev entry: it appeared between samples.

	got := diskIORates(prev, cur, 2)
	// "reset" is a backward counter (the device was re-attached) and
	// "midwindow" has no baseline — both must wait, leaving nvme0n1 and seen.
	if len(got) != 2 {
		t.Fatalf("got %+v, want nvme0n1 and seen", got)
	}
	var d DiskIOStats
	for _, g := range got {
		if g.Device == "nvme0n1" {
			d = g
		}
	}
	if d.Device != "nvme0n1" {
		t.Fatalf("nvme0n1 missing from %+v", got)
	}
	// 10512 sectors over 2 s at 512 bytes per sector → 2.6 MiB/s.
	if want := 10512 * 512 / 2; math.Abs(d.ReadBps-float64(want)) > 0.001 {
		t.Errorf("ReadBps = %v, want %v", d.ReadBps, want)
	}
	if want := 2000 * 512 / 2; math.Abs(d.WriteBps-float64(want)) > 0.001 {
		t.Errorf("WriteBps = %v, want %v", d.WriteBps, want)
	}
	if math.Abs(d.ReadsPerSec-500) > 0.001 || math.Abs(d.WritesPerSec-30) > 0.001 {
		t.Errorf("IOPS = %v/%v, want 500/30", d.ReadsPerSec, d.WritesPerSec)
	}
	// The busiest device sorts first: 2.69 MB total against seen's 5 KB/s.
	if got[0].Device != "nvme0n1" {
		t.Errorf("sort put %q first, want nvme0n1", got[0].Device)
	}

	// No elapsed time cannot produce a rate — division by zero would.
	if got := diskIORates(prev, cur, 0); got != nil {
		t.Errorf("zero elapsed yielded %+v, want nil", got)
	}
}

// keysOfDiskIO exists only so the test above can print the map's keys without
// pulling in yet another import for one error message.
func keysOfDiskIO(m map[string]diskIOCounters) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
