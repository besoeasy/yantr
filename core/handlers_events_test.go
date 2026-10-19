package main

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	dockerevents "github.com/docker/docker/api/types/events"
)

// One daemon event line, in the shape podman's compat endpoint actually
// writes: both the current Action and the legacy status fields live here in
// the wild, so the mapper has to accept the shape rather than a favorite.
const eventLineFixture = `{
  "time": 1729400000, "timeNano": 1729400000123456789,
  "type": "container", "Action": "die",
  "Actor": { "ID": "abc123", "Attributes": { "name": "glance", "exitCode": "137" } }
}`

func TestEventDTOsFromMapsAndOrders(t *testing.T) {
	var oldest, newest dockerevents.Message
	if err := json.Unmarshal([]byte(eventLineFixture), &oldest); err != nil {
		t.Fatalf("fixture decode: %v", err)
	}
	newest = oldest
	// newest = oldest copies the struct but shares the Attributes map —
	// renaming through newest would rename oldest too. A fresh map keeps the
	// two fixtures separate.
	newest.Actor.Attributes = map[string]string{"name": "arrived-later"}
	newest.Action = "start"

	// Three raw entries: a non-container, the named container, then one that
	// arrived after it. Output must be newest-first, container-only and named.
	raw := []dockerevents.Message{
		{Type: "image", Action: "pull", Actor: dockerevents.Actor{Attributes: map[string]string{"name": "fedora"}}},
		oldest,
		newest,
	}

	got := eventDTOsFrom(raw)
	if len(got) != 2 {
		t.Fatalf("got %+v, want 2 container entries (image events are not the killing floor's business)", got)
	}
	if got[0].Name != "arrived-later" || got[0].Action != "start" {
		t.Errorf("first = %+v, want arrived-later start", got[0])
	}
	if got[1].Name != "glance" || got[1].Action != "die" {
		t.Errorf("second = %+v, want glance die", got[1])
	}
	if got[1].AtMs != 1729400000123 {
		t.Errorf("AtMs = %d, want the TimeNano projection 1729400000123", got[1].AtMs)
	}
}

// Old daemons (and some podman edge paths) still fill status and leave Action
// blank; the mapper reads either without a flag.
func TestEventDTOsFromLegacyStatus(t *testing.T) {
	line := fmt.Sprintf(`{"time": 1729400000, "type": "container", "status": "start", "Actor": {"Attributes": {"name": "keel"}}}`)
	var m dockerevents.Message
	if err := json.Unmarshal([]byte(line), &m); err != nil {
		t.Fatalf("fixture decode: %v", err)
	}

	got := eventDTOsFrom([]dockerevents.Message{m})
	if len(got) != 1 || got[0].Action != "start" || got[0].Name != "keel" {
		t.Errorf("got %+v, want keel start via the legacy status field", got)
	}
	// Seconds-only timestamps project to their milli value.
	if got[0].AtMs != 1729400000000 {
		t.Errorf("AtMs = %d, want 1729400000000", got[0].AtMs)
	}
}

func TestEventDTOsFromCapsNewest(t *testing.T) {
	raw := make([]dockerevents.Message, maxRawEvents+40)
	for i := range raw {
		raw[i] = dockerevents.Message{
			Type:   dockerevents.ContainerEventType,
			Action: "input",
			Actor:  dockerevents.Actor{Attributes: map[string]string{"name": fmt.Sprintf("container-%02d", i)}},
			Time:   int64(time.Now().Unix()) - int64(maxRawEvents+40-i),
		}
	}

	got := eventDTOsFrom(raw)
	if len(got) != maxSystemEvents {
		t.Fatalf("got %d events, want the %d visible tail", len(got), maxSystemEvents)
	}
	// Newest first: the last-indexed entry must open the list.
	if got[0].Name != fmt.Sprintf("container-%02d", maxRawEvents+39) {
		t.Errorf("head = %s, want the newest container", got[0].Name)
	}
}
