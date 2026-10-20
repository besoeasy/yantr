package presence

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

func TestAggregateLatestPresenceWins(t *testing.T) {
	events := []Event{
		{Event: "presence", Node: "n1", TS: 1, Country: "US", OS: "Debian", RAMGB: 8, Apps: []string{"jellyfin"}},
		{Event: "presence", Node: "n1", TS: 2, Country: "DE", OS: "Debian", RAMGB: 32, Apps: []string{"immich", "jellyfin"}},
		{Event: "presence", Node: "n2", TS: 2, Country: "DE", OS: "Unraid", RAMGB: 16, Apps: []string{"nextcloud"}},
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
	if found["jellyfin"] != 1 || found["immich"] != 1 || found["nextcloud"] != 1 {
		t.Fatalf("apps=%+v", stats.Apps)
	}
	body, _ := json.Marshal(stats)
	if len(body) == 0 {
		t.Fatal("empty json")
	}
}
