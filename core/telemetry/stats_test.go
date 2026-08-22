package telemetry

import (
	"encoding/json"
	"testing"
)

func TestParseEventJSON(t *testing.T) {
	raw := `{"event":"presence","node":"abc123","country":"de","os":"Debian","ram_gb":16,"apps":["jellyfin","immich"]}`
	evt, ok := ParseEvent(raw)
	if !ok {
		t.Fatal("expected json event")
	}
	if evt.Event != "presence" || evt.Node != "abc123" || evt.Country != "DE" {
		t.Fatalf("parsed %+v", evt)
	}
	if evt.RAMGB != 16 || len(evt.Apps) != 2 {
		t.Fatalf("fields %+v", evt)
	}
}

func TestParseEventLegacy(t *testing.T) {
	raw := `presence v=20260730 country=CN os=Debian 12 (bookworm) arch=x86_64 cores=2 ram_gb=7 stacks=1`
	evt, ok := ParseEvent(raw)
	if !ok || evt.Country != "CN" || evt.OS != "Debian 12 (bookworm)" || evt.RAMGB != 7 || evt.Cores != 2 {
		t.Fatalf("legacy %+v ok=%v", evt, ok)
	}
}

func TestAggregateLatestPresenceWins(t *testing.T) {
	events := []Event{
		{Event: "presence", Node: "n1", TS: 1, Country: "US", OS: "Debian", RAMGB: 8, Apps: []string{"jellyfin"}},
		{Event: "presence", Node: "n1", TS: 2, Country: "DE", OS: "Debian", RAMGB: 32, Apps: []string{"immich", "jellyfin"}},
		{Event: "presence", Node: "n2", TS: 2, Country: "DE", OS: "Unraid", RAMGB: 16, Apps: []string{"nextcloud"}},
		{Event: "install", App: "paperless-ngx"},
	}
	stats := Aggregate(events)
	if stats.Nodes != 2 {
		t.Fatalf("nodes=%d", stats.Nodes)
	}
	if stats.RAM.Min != 16 || stats.RAM.Max != 32 {
		t.Fatalf("ram=%+v", stats.RAM)
	}
	if stats.Countries[0].Code != "DE" || stats.Countries[0].Count != 2 {
		t.Fatalf("countries=%+v", stats.Countries)
	}
	found := map[string]int{}
	for _, app := range stats.Apps {
		found[app.ID] = app.Count
	}
	if found["jellyfin"] != 1 || found["immich"] != 1 || found["nextcloud"] != 1 || found["paperless-ngx"] != 1 {
		t.Fatalf("apps=%+v", stats.Apps)
	}
	body, _ := json.Marshal(stats)
	if len(body) == 0 {
		t.Fatal("empty json")
	}
}

func TestAggregateFingerprintsLegacyNodes(t *testing.T) {
	a, _ := ParseEvent(`presence country=US os=QTS 5.2.9 arch=x86_64 cores=4 ram_gb=15 stacks=0 v=20260730`)
	b, _ := ParseEvent(`presence country=US os=QTS 5.2.9 arch=x86_64 cores=4 ram_gb=15 stacks=0 v=20260730`)
	stats := Aggregate([]Event{a, b})
	if stats.Nodes != 1 {
		t.Fatalf("expected 1 fingerprinted node, got %d", stats.Nodes)
	}
}
