// This file is the read side of the presence package. It fetches the topic that
// presence.go posts to and reduces it to the Fleet view.
//
// It lives here rather than in a package of its own because it reads exactly
// what this package writes — the aggregate cannot be fetched without knowing
// the topic, the event shape and the opt-out. So the package keeps the feed's
// name and the vocabulary that goes with it. What gets named "fleet" is the
// view built from it: the API endpoint, the page, and the types below.

package presence

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"core/shared"
)

// NamedCount is a labeled tally.
type NamedCount struct {
	ID    string  `json:"id,omitempty"`
	Name  string  `json:"name"`
	Count int     `json:"count"`
	Code  string  `json:"code,omitempty"`
	Lat   float64 `json:"lat,omitempty"`
	Lng   float64 `json:"lng,omitempty"`
}

// RAMStats is min/max/avg RAM across nodes.
type RAMStats struct {
	Min   int64   `json:"min"`
	Max   int64   `json:"max"`
	Avg   float64 `json:"avg"`
	Unit  string  `json:"unit"`
	Count int     `json:"count"`
}

// FleetStats is the aggregated dashboard payload.
type FleetStats struct {
	Nodes     int          `json:"nodes"`
	Events    int          `json:"events"`
	Countries []NamedCount `json:"countries"`
	OS        []NamedCount `json:"os"`
	RAM       RAMStats     `json:"ram"`
	Apps      []NamedCount `json:"apps"`
	Window    string       `json:"window"`
	FetchedAt string       `json:"fetchedAt"`
}

type ntfyMessage struct {
	ID      string `json:"id"`
	Time    int64  `json:"time"`
	Event   string `json:"event"`
	Message string `json:"message"`
	Title   string `json:"title"`
}

var (
	statsMu  sync.Mutex
	stats    *FleetStats
	statsExp time.Time
	statsTTL = 45 * time.Second
)

func cacheFilePath() string {
	if d := os.Getenv("YANTR_DATA_DIR"); d != "" {
		return filepath.Join(d, "presence-cache.json")
	}
	return "/data/presence-cache.json"
}

func loadStatsFromDisk() *FleetStats {
	raw, err := os.ReadFile(cacheFilePath())
	if err != nil {
		return nil
	}
	var out FleetStats
	if json.Unmarshal(raw, &out) != nil {
		return nil
	}
	return &out
}

func saveStatsToDisk(s *FleetStats) {
	if s == nil {
		return
	}
	raw, err := json.Marshal(s)
	if err != nil {
		return
	}
	_ = os.MkdirAll(filepath.Dir(cacheFilePath()), 0o755)
	// Self-healing cache, so atomicity is not critical here — but it costs
	// nothing and avoids serving a half-written cache to the Fleet page.
	_ = shared.WriteFileAtomic(cacheFilePath(), raw, 0o600)
}

// GetFleetStatsCached returns the aggregated fleet view, cached briefly.
func GetFleetStatsCached(force bool) (*FleetStats, error) {
	statsMu.Lock()
	defer statsMu.Unlock()

	if !force && stats != nil && time.Now().Before(statsExp) {
		return stats, nil
	}

	out, err := fetchAndAggregate()
	if err != nil {
		// Serve the last known stats when the upstream feed is unreachable,
		// so the dashboard (and map) still render during outages.
		if stats != nil {
			shared.Log("warn", "[presence] fetch failed, serving cached stats: "+err.Error())
			return stats, nil
		}
		if disk := loadStatsFromDisk(); disk != nil {
			shared.Log("warn", "[presence] fetch failed, serving disk-cached stats: "+err.Error())
			stats = disk
			return disk, nil
		}
		return nil, err
	}
	stats = out
	statsExp = time.Now().Add(statsTTL)
	saveStatsToDisk(out)
	return out, nil
}

func fetchAndAggregate() (*FleetStats, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 12*time.Second)
	defer cancel()

	url := TopicURL() + "/json?poll=1&since=all"
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/x-ndjson")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("ntfy fetch: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("ntfy fetch: status %d", resp.StatusCode)
	}

	var events []Event
	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var msg ntfyMessage
		if err := json.Unmarshal([]byte(line), &msg); err != nil {
			continue
		}
		if msg.Event != "" && msg.Event != "message" {
			continue
		}
		evt, ok := ParseEvent(msg.Message)
		if !ok {
			continue
		}
		if evt.TS == 0 && msg.Time > 0 {
			evt.TS = msg.Time
		}
		if evt.Node == "" {
			evt.Node = msg.ID
		}
		events = append(events, evt)
	}
	if err := scanner.Err(); err != nil {
		shared.Log("warn", "[presence] stats scan: "+err.Error())
	}

	return Aggregate(events), nil
}

// ParseEvent accepts a JSON event payload.
func ParseEvent(raw string) (Event, bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return Event{}, false
	}
	var evt Event
	if err := json.Unmarshal([]byte(raw), &evt); err != nil || evt.Event == "" {
		return Event{}, false
	}
	evt.Country = strings.ToUpper(strings.TrimSpace(evt.Country))
	return evt, true
}

// Aggregate reduces events into dashboard stats. Latest presence wins per node.
func Aggregate(events []Event) *FleetStats {
	type nodeSnap struct {
		evt Event
		ts  int64
	}
	nodes := map[string]nodeSnap{}
	installs := map[string]int{}

	for _, evt := range events {
		switch evt.Event {
		case "presence":
			id := evt.Node
			if id == "" {
				id = fmt.Sprintf("fp:%s|%s|%s|%d|%d|%s", evt.Country, evt.OS, evt.Arch, evt.Cores, evt.RAMGB, evt.Version)
			}
			prev, ok := nodes[id]
			if !ok || evt.TS >= prev.ts {
				nodes[id] = nodeSnap{evt: evt, ts: evt.TS}
			}
		case "install":
			if evt.App != "" {
				installs[evt.App]++
			}
		}
	}

	countryCounts := map[string]int{}
	osCounts := map[string]int{}
	appCounts := map[string]int{}
	var ramValues []int64

	for _, snap := range nodes {
		evt := snap.evt
		country := evt.Country
		if country == "" {
			country = "??"
		}
		countryCounts[country]++

		osName := evt.OS
		if osName == "" {
			osName = "unknown"
		}
		osCounts[osName]++

		if evt.RAMGB > 0 {
			ramValues = append(ramValues, evt.RAMGB)
		}
		for _, app := range evt.Apps {
			if app != "" {
				appCounts[app]++
			}
		}
	}

	// Install pings still count when a node has not sent presence yet.
	for app, n := range installs {
		if appCounts[app] == 0 {
			appCounts[app] = n
		}
	}

	ram := RAMStats{Unit: "GB"}
	if len(ramValues) > 0 {
		ram.Min = ramValues[0]
		ram.Max = ramValues[0]
		var sum int64
		for _, v := range ramValues {
			sum += v
			if v < ram.Min {
				ram.Min = v
			}
			if v > ram.Max {
				ram.Max = v
			}
		}
		ram.Avg = float64(sum) / float64(len(ramValues))
		ram.Count = len(ramValues)
	}

	return &FleetStats{
		Nodes:     len(nodes),
		Events:    len(events),
		Countries: countryList(countryCounts),
		OS:        namedList(osCounts, 12),
		RAM:       ram,
		Apps:      appList(appCounts, 12),
		Window:    "ntfy cache",
		FetchedAt: time.Now().UTC().Format(time.RFC3339),
	}
}

func countryList(counts map[string]int) []NamedCount {
	out := make([]NamedCount, 0, len(counts))
	for code, n := range counts {
		meta := LookupCountry(code)
		out = append(out, NamedCount{
			Code:  code,
			Name:  meta.Name,
			Count: n,
			Lat:   meta.Lat,
			Lng:   meta.Lng,
		})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Count == out[j].Count {
			return out[i].Name < out[j].Name
		}
		return out[i].Count > out[j].Count
	})
	return out
}

func namedList(counts map[string]int, limit int) []NamedCount {
	out := make([]NamedCount, 0, len(counts))
	for name, n := range counts {
		out = append(out, NamedCount{Name: name, Count: n})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Count == out[j].Count {
			return out[i].Name < out[j].Name
		}
		return out[i].Count > out[j].Count
	})
	if limit > 0 && len(out) > limit {
		out = out[:limit]
	}
	return out
}

func appList(counts map[string]int, limit int) []NamedCount {
	out := make([]NamedCount, 0, len(counts))
	for id, n := range counts {
		out = append(out, NamedCount{ID: id, Name: id, Count: n})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Count == out[j].Count {
			return out[i].Name < out[j].Name
		}
		return out[i].Count > out[j].Count
	})
	if limit > 0 && len(out) > limit {
		out = out[:limit]
	}
	return out
}
