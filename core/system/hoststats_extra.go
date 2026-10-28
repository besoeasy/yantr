package system

import (
	"bufio"
	"os"
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
