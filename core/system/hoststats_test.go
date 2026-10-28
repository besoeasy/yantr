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
Slab:             500000 kB
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
	if got.CachedBytes != 3000000*1024 || got.SlabBytes != 500000*1024 || got.BuffersBytes != 200000*1024 {
		t.Errorf("breakdown = cached %d buffers %d slab %d, want 3M/200k/500k kB ×1024",
			got.CachedBytes, got.BuffersBytes, got.SlabBytes)
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

// mountsFixture writes a mount table shaped like the one readDisks consumes:
// targets are paths that exist right now (temp dirs), because readDisks stats
// targets verbatim — the container's own table is exactly this, with bind
// targets already rewritten. tmpfs is removed from the skip list so the temp
// dir's own filesystem passes the type filter like a real disk would.
func mountsFixture(t *testing.T) (string, string) {
	t.Helper()
	skipFSTypes["tmpfs"] = false
	t.Cleanup(func() { skipFSTypes["tmpfs"] = true })

	root := t.TempDir()
	dirA := filepath.Join(root, "mnt-a")
	dirB := filepath.Join(root, "mnt-b")
	dirC := filepath.Join(root, "mnt-c")
	nfsDir := filepath.Join(root, "mnt-nfs")
	for _, d := range []string{dirA, dirB, dirC, nfsDir} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	// One store mounted twice, a second store name that is physically the same
	// filesystem (the dedupe's real-world case: LVM subvolumes), and a pseudo-fs
	// plus a network filesystem with a device that is not a block node.
	mounts := fmt.Sprintf(
		"/dev/vda1 %s ext4 rw 0 0\n/dev/vda1 %s ext4 rw 0 0\n/dev/vdb1 %s xfs rw 0 0\ntmpfs %s tmpfs rw 0 0\nfreenas:/export %s nfs4 rw 0 0\nportal %s fuse.portal rw 0 0\n",
		dirA, dirB, dirC, filepath.Join(root, "run"), nfsDir, filepath.Join(root, "doc"))
	path := filepath.Join(root, "mounts")
	if err := os.WriteFile(path, []byte(mounts), 0o600); err != nil {
		t.Fatal(err)
	}
	return path, dirA
}

func TestReadDisks(t *testing.T) {
	mountsPath, dirA := mountsFixture(t)
	got, err := readDisks(mountsPath, "")
	if err != nil {
		t.Fatalf("readDisks: %v", err)
	}
	// Every fixture mountpoint lives on the caller's filesystem, so by st_dev
	// they are ONE store: the double-mount of vda1 and the nfs line fold into
	// whichever entry statfs'd first, while the tmpfs and portal lines drop on
	// reportableMount. Type filtering and device heuristics are pinned
	// separately on reportableMount.
	if len(got) != 1 {
		t.Fatalf("got %+v, want one entry per distinct store (dedupe applied)", got)
	}
	// dirA is the first statfs-able line, so it wins the dedupe against the
	// vda1 remount, the differently-named vdb1 line and the nfs line — all of
	// which live on the caller's filesystem here and share its st_dev.
	if got[0].Mount != dirA {
		t.Errorf("mount = %q, want %q (first statfs-able line wins the dedupe)", got[0].Mount, dirA)
	}
	if got[0].FSType != "ext4" {
		t.Errorf("FSType = %q, want ext4 straight from the mounts file", got[0].FSType)
	}
	for _, d := range got {
		if d.TotalBytes == 0 {
			t.Errorf("%s: TotalBytes = 0, want a real size", d.Mount)
		}
		if d.UsedPct < 0 || d.UsedPct > 100 {
			t.Errorf("%s: UsedPct = %v, want a percentage", d.Mount, d.UsedPct)
		}
	}
}

func TestMountLabel(t *testing.T) {
	cases := []struct{ target, hostRoot, want string }{
		{"/host", "/host", "/"},
		{"/host/boot", "/host", "/boot"},
		{"/srv", "/host", "/srv"},
		{"/var", "", "/var"},
		{"/hostish", "/host", "/hostish"},
	}
	for _, tc := range cases {
		if got := mountLabel(tc.target, tc.hostRoot); got != tc.want {
			t.Errorf("mountLabel(%q, %q) = %q, want %q", tc.target, tc.hostRoot, got, tc.want)
		}
	}
}

func TestReadDisksMissingMountsFile(t *testing.T) {
	got, err := readDisks(filepath.Join(t.TempDir(), "not-mounted"), "")
	if err == nil || got != nil {
		t.Errorf("readDisks on a missing table = %+v, %v; want an error so the snapshot can degrade", got, err)
	}
}

func TestReportableMount(t *testing.T) {
	cases := []struct {
		name string
		in   mountEntry
		want bool
	}{
		{"block", mountEntry{"/dev/vda1", "/", "ext4"}, true},
		{"second store", mountEntry{"/dev/vdb1", "/var", "xfs"}, true},
		{"nfs names its server", mountEntry{"freenas:/export", "/mnt", "nfs4"}, true},
		{"zfs datasets have no block node", mountEntry{"tank/media", "/media", "zfs"}, true},
		{"fuse portal is noise", mountEntry{"portal", "/run/doc", "fuse.portal"}, false},
		{"efi boot partition is noise", mountEntry{"/dev/nvme0n1p1", "/boot/efi", "vfat"}, false},
		{"overlay is the container's own", mountEntry{"overlay", "/", "overlay"}, false},
		{"pipe device with no path", mountEntry{"shm", "/dev/shm", "tmpfs"}, false},
	}
	for _, tc := range cases {
		if got := reportableMount(tc.in); got != tc.want {
			t.Errorf("%s: reportableMount(%+v) = %v, want %v", tc.name, tc.in, got, tc.want)
		}
	}
}

func TestIsSkipFSTypeMatchesFuseVariants(t *testing.T) {
	if !isSkipFSType("fuse.portal") || !isSkipFSType("fuse.gvfs-fuse-daemon") {
		t.Error("fuse. variants must skip; they carry no dashboard-relevant data")
	}
	if isSkipFSType("ext4") || isSkipFSType("zfs") {
		t.Error("real data filesystems must not skip")
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
		mounts: missing, hostRoot: missing, sysRoot: missing,
	}}

	got := c.snapshot()
	for _, want := range []string{"cpu", "memory", "load", "network", "disk", "temperature", "gpu", "vmstat"} {
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
	hwmonFan(t, dir, "hwmon1", "fan1_input", 3036)  // acpi_fan, the faster of the two real ones
	hwmonFan(t, dir, "hwmon2", "fan1_input", 0)     // a stopped fan is a real observation, not a dead sensor
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

func TestReadSensorsCarriesChipTripPoints(t *testing.T) {
	dir := t.TempDir()
	hwmonFile(t, dir, "hwmon1", "name", "nvme\n")
	hwmonFile(t, dir, "hwmon1", "temp1_input", "52500\n")
	hwmonFile(t, dir, "hwmon1", "temp1_crit", "85000\n")
	hwmonFile(t, dir, "hwmon2", "name", "k10temp\n")
	// k10temp ships no threshold on many boards, so TripC must stay null there
	// rather than inherit a neighbour's number.
	hwmonFile(t, dir, "hwmon2", "temp1_input", "42500\n")

	got, err := readSensors(sources{sysRoot: dir})
	if err != nil {
		t.Fatalf("readSensors: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("got %+v, want 2 entries", got)
	}
	if got[0].Chip != "nvme" || got[0].TripC == nil || *got[0].TripC != 85 {
		t.Errorf("nvme = %+v, want 52.5°C with the chip's own 85°C trip", got[0])
	}
	if got[1].Chip != "k10temp" || got[1].TripC != nil {
		t.Errorf("k10temp = %+v, want a null trip: no threshold shipped", got[1])
	}
}

func TestReadSensorsZoneTripPointsFallOutHottest(t *testing.T) {
	dir := t.TempDir()
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
	if err := os.WriteFile(filepath.Join(zone, "trip_point_0_temp"), []byte("90000\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(zone, "trip_point_1_temp"), []byte("125000\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	got, err := readSensors(sources{sysRoot: dir})
	if err != nil {
		t.Fatalf("readSensors: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("got %+v, want the acpitz entry only", got)
	}
	if got[0].TripC == nil || *got[0].TripC != 125 {
		t.Errorf("zone trip = %v, want 125 (the hottest threshold is the one it races)", got[0].TripC)
	}
}

func TestReadGpusKeepsCardsThatExposeReadings(t *testing.T) {
	dir := t.TempDir()
	cardDev := filepath.Join(dir, "class", "drm", "card0", "device")
	// A DRm node the kernel filled in but whose driver exposes none of the
	// utilization files (i915 looks like this): it must fold out of the list.
	if err := os.MkdirAll(filepath.Join(dir, "class", "drm", "card7", "device"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(cardDev, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cardDev, "uevent"), []byte("DRIVER=amdgpu\nVERSION=1\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	for item, content := range map[string]string{
		"gpu_busy_percent":    "0\n",
		"mem_info_vram_used":  "1673891840",
		"mem_info_vram_total": "8573157376",
		"power1_average":      "37000000",
	} {
		if err := os.WriteFile(filepath.Join(cardDev, item), []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	// The driver's temp rides along from a hwmon chip that names itself amdgpu.
	hwmonFile(t, dir, "hwmon1", "name", "amdgpu\n")
	hwmonFile(t, dir, "hwmon1", "temp1_input", "41000\n")
	hwmonFile(t, dir, "hwmon1", "temp1_crit", "120000\n")

	got := readGpus(sources{sysRoot: dir})
	if len(got) != 1 {
		t.Fatalf("got %+v, want only card0 (card7 exposed nothing)", got)
	}
	g := got[0]
	if g.Card != "card0" || g.Driver != "amdgpu" || g.BusyPct == nil || *g.BusyPct != 0 {
		t.Errorf("gpu = %+v, want card0 amdgpu busy 0 (an idle GPU is a real reading)", g)
	}
	if g.VramUsedBytes != 1673891840 || g.VramTotalBytes != 8573157376 {
		t.Errorf("vram = %d/%d, want the fixture values", g.VramUsedBytes, g.VramTotalBytes)
	}
	if g.TempC == nil || math.Abs(*g.TempC-41) > 0.001 || g.TripC == nil || *g.TripC != 120 {
		t.Errorf("temp = %v/%v, want 41°C against a 120°C trip", g.TempC, g.TripC)
	}
	// amdgpu reports microwatts; 37000000 is a 37 W idle draw.
	if g.PowerW == nil || math.Abs(*g.PowerW-37) > 0.001 {
		t.Errorf("power = %v, want 37 W from the microwatt file", g.PowerW)
	}
}

func TestReadGpusNoSysfsIsNotAnError(t *testing.T) {
	got := readGpus(sources{sysRoot: t.TempDir()})
	if len(got) != 0 {
		t.Errorf("got %+v, want an empty list", got)
	}
}

func TestVMRates(t *testing.T) {
	prev := vmCounters{pswpin: 1000, pswpout: 500}
	cur := vmCounters{pswpin: 2512, pswpout: 2000}
	in, outBps := vmRates(prev, cur, 2)
	// 1512 pages in over 2 s at the 4 KiB page size: 3.072 MiB/s.
	if math.Abs(in-1512*4096/2) > 0.001 {
		t.Errorf("swap-in = %v, want %v", in, 1512*4096/2)
	}
	if math.Abs(outBps-1500*4096/2) > 0.001 {
		t.Errorf("swap-out = %v, want %v", outBps, 1500*4096/2)
	}

	// A backwards counter (accounting restarted) drops the whole window: a
	// negative rate would print megabytes flying backwards in time.
	if in, outBps := vmRates(vmCounters{pswpin: 5000}, cur, 2); in != 0 || outBps != 0 {
		t.Errorf("backwards counter = %v/%v, want 0/0 for the dropped window", in, outBps)
	}
	if in, outBps := vmRates(prev, cur, 0); in != 0 || outBps != 0 {
		t.Errorf("zero elapsed = %v/%v, want 0/0", in, outBps)
	}
}

func TestReadVMStatRejectsWrongShape(t *testing.T) {
	// A vmstat missing the swap keys is a kernel we do not know how to read;
	// it must degrade rather than report a flat zero line.
	if _, err := readVMStat(sources{hostRoot: t.TempDir()}); err == nil {
		t.Fatal("expected an error when vmstat does not exist")
	}
	dir := t.TempDir()
	procDir := filepath.Join(dir, "proc")
	if err := os.MkdirAll(procDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(procDir, "vmstat"), []byte("nr_free_pages 1000\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := readVMStat(sources{hostRoot: dir}); err == nil {
		t.Fatal("expected an error when no swap key is present")
	}
}

func TestReadVMStatParsesCounters(t *testing.T) {
	dir := t.TempDir()
	procDir := filepath.Join(dir, "proc")
	if err := os.MkdirAll(procDir, 0o755); err != nil {
		t.Fatal(err)
	}
	body := "pgpgin 100\npswpin 12\npswpout 3\nnr_dirty 20\nnr_writeback 5\nother_key 9\n"
	if err := os.WriteFile(filepath.Join(procDir, "vmstat"), []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	got, err := readVMStat(sources{hostRoot: dir})
	if err != nil {
		t.Fatalf("readVMStat: %v", err)
	}
	if got.pswpin != 12 || got.pswpout != 3 {
		t.Errorf("swap counters = %d/%d, want 12/3", got.pswpin, got.pswpout)
	}
	// Page counts convert to bytes at the kernel's page size.
	if got.dirty != 20*4096 || got.writeback != 5*4096 {
		t.Errorf("dirty/writeback = %d/%d, want byte-converted page counts", got.dirty, got.writeback)
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
   8      16 sda 10 0 100 1 5 0 40 2 7 350 0 0 0
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
	// The utilization pair rides on the same line: 7 ops in flight, 350 ms
	// of accumulated busy time on sda.
	if got["sda"].inFlight != 7 || got["sda"].ioTicks != 350 {
		t.Errorf("sda = %+v, want inFlight 7 / ioTicks 350", got["sda"])
	}
}

func TestReadDiskstatsRejectsGarbage(t *testing.T) {
	if _, err := readDiskstats(writeFixture(t, "diskstats", "nothing here\n")); err == nil {
		t.Fatal("expected an error when no device line parses")
	}
}

func TestDiskIORates(t *testing.T) {
	prev := map[string]diskIOCounters{
		"nvme0n1": {reads: 1000, readSectors: 20000, writes: 100, writeSectors: 2000, ioTicks: 40000, inFlight: 4},
		"seen":    {reads: 50, readSectors: 100, writes: 0, writeSectors: 0},
	}
	cur := map[string]diskIOCounters{
		"nvme0n1":   {reads: 2000, readSectors: 30512, writes: 160, writeSectors: 4000, ioTicks: 41600, inFlight: 0},
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
	// 1600 ms busy across a 2000 ms window: 80% of the window the device was
	// doing work — the saturation signal the byte rates cannot carry.
	if d.UtilPct != 80 {
		t.Errorf("UtilPct = %v, want 80 (1600ms busy in a 2000ms window)", d.UtilPct)
	}
	// Queue depth is instantaneous at read time.
	if d.InFlight != 0 {
		t.Errorf("InFlight = %d, want the current tick's queue depth", d.InFlight)
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
