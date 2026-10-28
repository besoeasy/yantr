package system

import (
	"bufio"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

// readCPUModelName pulls the "model name" line from /proc/cpuinfo. Returns ""
// when the file is missing or has no such line (e.g. ARM boards that only
// expose "Processor" or "Hardware").
func readCPUModelName(path string) string {
	f, err := os.Open(path)
	if err != nil {
		return ""
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := sc.Text()
		if !strings.HasPrefix(line, "model name") {
			continue
		}
		if colon := strings.Index(line, ":"); colon >= 0 {
			if v := strings.TrimSpace(line[colon+1:]); v != "" {
				return v
			}
		}
	}
	return ""
}

// readPressure parses the three /proc/pressure/* files into PressureStats.
// A missing file (older kernel, or a host without PSI) leaves that entry
// zeroed and is reported as a single "pressure" degraded key by the caller.
func readPressure(s sources) (PressureStats, error) {
	out := PressureStats{}
	parsed := 0
	type target struct {
		path string
		dst  *PressureEntry
	}
	targets := []target{
		{s.hostPath("/proc/pressure/cpu"), &out.CPU},
		{s.hostPath("/proc/pressure/memory"), &out.Mem},
		{s.hostPath("/proc/pressure/io"), &out.IO},
	}
	for _, t := range targets {
		entry, err := readPressureFile(t.path)
		if err != nil {
			continue
		}
		*t.dst = entry
		parsed++
	}
	if parsed == 0 {
		return out, os.ErrNotExist
	}
	return out, nil
}

func readPressureFile(path string) (PressureEntry, error) {
	f, err := os.Open(path)
	if err != nil {
		return PressureEntry{}, err
	}
	defer f.Close()
	var e PressureEntry
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		fields := strings.Fields(sc.Text())
		if len(fields) < 5 {
			continue
		}
		// "some avg10=... avg60=... avg300=... total=..." or "full ..."
		kind := fields[0]
		avg10, avg60, avg300, total, ok := parsePressureFields(fields[1:])
		if !ok {
			continue
		}
		if kind == "some" {
			e.SomeAvg10, e.SomeAvg60, e.SomeAvg300, e.SomeTotal = avg10, avg60, avg300, total
		} else if kind == "full" {
			e.FullAvg10, e.FullAvg60, e.FullAvg300, e.FullTotal = avg10, avg60, avg300, total
		}
	}
	return e, nil
}

func parsePressureFields(fields []string) (float64, float64, float64, uint64, bool) {
	var a10, a60, a300 float64
	var total uint64
	okCount := 0
	for _, f := range fields {
		if kv := strings.SplitN(f, "=", 2); len(kv) == 2 {
			switch kv[0] {
			case "avg10":
				if v, err := strconv.ParseFloat(kv[1], 64); err == nil {
					a10 = v
					okCount++
				}
			case "avg60":
				if v, err := strconv.ParseFloat(kv[1], 64); err == nil {
					a60 = v
					okCount++
				}
			case "avg300":
				if v, err := strconv.ParseFloat(kv[1], 64); err == nil {
					a300 = v
					okCount++
				}
			case "total":
				if v, err := strconv.ParseUint(kv[1], 10, 64); err == nil {
					total = v
					okCount++
				}
			}
		}
	}
	if okCount < 4 {
		return 0, 0, 0, 0, false
	}
	return a10, a60, a300, total, true
}

// readBoard collects DMI identity from /sys/class/dmi/id. Missing files are
// tolerated; an error is returned only when nothing at all was readable.
func readBoard(s sources) (BoardInfo, error) {
	b := BoardInfo{}
	got := false
	read := func(rel string, dst *string) {
		raw, err := os.ReadFile(s.sysPath(rel))
		if err != nil {
			return
		}
		v := strings.TrimSpace(string(raw))
		if v == "" {
			return
		}
		*dst = v
		got = true
	}
	read("/class/dmi/id/sys_vendor", &b.Vendor)
	read("/class/dmi/id/board_name", &b.Board)
	read("/class/dmi/id/product_name", &b.Product)
	// chassis_type is a numeric code; store it as-is for the UI to label.
	if raw, err := os.ReadFile(s.sysPath("/class/dmi/id/chassis_type")); err == nil {
		if v := strings.TrimSpace(string(raw)); v != "" {
			b.Chassis = v
			got = true
		}
	}
	if !got {
		return b, os.ErrNotExist
	}
	return b, nil
}

// readNetLink reads the physical-layer attributes for one interface from sysfs.
// Returns zero values when the files are missing or unreadable (virtual
// interfaces, or a host without the sys bind).
func readNetLink(s sources, iface string) (int, string, string) {
	speed := 0
	if raw, err := os.ReadFile(s.sysPath("/class/net/" + iface + "/speed")); err == nil {
		if v, err := strconv.Atoi(strings.TrimSpace(string(raw))); err == nil && v > 0 {
			speed = v
		}
	}
	duplex := ""
	if raw, err := os.ReadFile(s.sysPath("/class/net/" + iface + "/duplex")); err == nil {
		duplex = strings.TrimSpace(string(raw))
	}
	operState := ""
	if raw, err := os.ReadFile(s.sysPath("/class/net/" + iface + "/operstate")); err == nil {
		operState = strings.TrimSpace(string(raw))
	}
	return speed, duplex, operState
}

// readUintFile parses a plain decimal file ("123456\n") into an unsigned raw
// value. The boolean, not a pointer, is the contract: a missing file and a
// real 0 both concern the caller, and each is a different fact.
func readUintFile(path string) (uint64, bool) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return 0, false
	}
	v, err := strconv.ParseUint(strings.TrimSpace(string(raw)), 10, 64)
	if err != nil {
		return 0, false
	}
	return v, true
}

// readThresholdC parses a hwmon threshold file (tempN_crit / tempN_max /
// trip_point_N_temp) into Celsius. Thresholds are numbers the BOARD shipped:
// they must be plausible hardware numbers, but unlike live readings they
// routinely sit above 100°C (GPU thermal shells settle at 120), so they get a
// wider window than readCelsius's live band instead of being dropped into it.
func readThresholdC(path string) (*float64, bool) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, false
	}
	milli, err := strconv.ParseFloat(strings.TrimSpace(string(raw)), 64)
	if err != nil {
		return nil, false
	}
	c := milli / 1000
	if c <= 0 || c > 250 {
		return nil, false
	}
	return &c, true
}

// readBusyPct parses the driver-authored utilization file. A real "0" is an
// honest idle reading (the file is runtime-generated, unlike a static 0), so a
// parseable zero IS reported; only a missing or garbage file yields nil.
func readBusyPct(path string) *float64 {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	v, err := strconv.ParseFloat(strings.TrimSpace(string(raw)), 64)
	if err != nil || v < 0 || v > 100 {
		return nil
	}
	return &v
}

// readGpuTemp hunts the driver's hwmon chip for its edge temperature and the
// trip it throttles at. Two chip layouts exist in the wild: amdgpu hangs an
// hwmon child off the device itself, and drivers that share the global hwmon
// tree register under their own name — this checks both, in that order.
func readGpuTemp(s sources, devDir, driver string) (*float64, *float64) {
	temps, _ := filepath.Glob(filepath.Join(devDir, "hwmon", "hwmon*", "temp1_input"))
	if len(temps) == 0 && driver != "" {
		for _, chipDir := range globSlotPaths(s.sysPath("/class/hwmon"), "hwmon*") {
			if raw, err := os.ReadFile(filepath.Join(chipDir, "name")); err == nil &&
				strings.TrimSpace(string(raw)) == driver {
				temps, _ = filepath.Glob(filepath.Join(chipDir, "temp1_input"))
				break
			}
		}
	}
	if len(temps) == 0 {
		return nil, nil
	}
	dir := filepath.Dir(temps[0])
	temp, _ := readCelsius(temps[0])
	var trip *float64
	if t, ok := readThresholdC(filepath.Join(dir, "temp1_crit")); ok {
		trip = t
	} else if t, ok := readThresholdC(filepath.Join(dir, "temp1_max")); ok {
		trip = t
	}
	return temp, trip
}

// maxReportedGpus bounds the payload: a workstation with five GPUs of which
// one matters to a homelab dashboard should not write a page of JSON per tick.
const maxReportedGpus = 2

// readGpus enumerates the DRM cards and keeps the ones whose driver exposes
// utilization or VRAM files. Intel's i915 and the proprietary NVIDIA driver
// expose none of these sysfs files rootless, and those cards fold out of the
// list — an empty list is "the kernel exposes nothing", which the widget
// renders as a quiet not-exposed state rather than an error.
func readGpus(s sources) []GpuStats {
	out := []GpuStats{}
	cards, _ := filepath.Glob(s.sysPath("/class/drm/card[0-9]*"))
	sort.Strings(cards)
	for _, cardDir := range cards {
		devDir := filepath.Join(cardDir, "device")
		driver := ""
		if raw, err := os.ReadFile(filepath.Join(devDir, "uevent")); err == nil {
			for _, line := range strings.Split(string(raw), "\n") {
				if !strings.HasPrefix(line, "DRIVER=") {
					continue
				}
				if v := strings.TrimSpace(line[len("DRIVER="):]); v != "" && v != "unknown" {
					driver = v
				}
			}
		}
		busy := readBusyPct(filepath.Join(devDir, "gpu_busy_percent"))
		vramUsed, _ := readUintFile(filepath.Join(devDir, "mem_info_vram_used"))
		vramTotal, _ := readUintFile(filepath.Join(devDir, "mem_info_vram_total"))

		// A card that exposes no readings at all contributes nothing to the
		// card — but a card that exposes SOME (vram with no busy on a variant
		// kernel) is still worth listing, with a null busy the UI renders as
		// an em-dash.
		if busy == nil && vramUsed == 0 && vramTotal == 0 && driver == "" {
			continue
		}
		tempC, tripC := readGpuTemp(s, devDir, driver)
		out = append(out, GpuStats{
			Card:           filepath.Base(cardDir),
			Driver:         driver,
			BusyPct:        busy,
			VramUsedBytes:  vramUsed,
			VramTotalBytes: vramTotal,
			TempC:          tempC,
			TripC:          tripC,
		})
		if len(out) == maxReportedGpus {
			break
		}
	}
	return out
}
