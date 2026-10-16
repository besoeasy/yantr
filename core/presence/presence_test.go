package presence

import "testing"

// env builds a getenv func over a fixed map.
func env(pairs map[string]string) func(string) string {
	return func(key string) string { return pairs[key] }
}

func TestResolveConfigDefaults(t *testing.T) {
	topic, enabled, disabledBy := resolveConfig(env(nil))

	if topic != "https://ntfy.sh/yantr" {
		t.Errorf("default topic = %q, want the public ntfy topic", topic)
	}
	if !enabled {
		t.Error("default should be enabled")
	}
	if disabledBy != "" {
		t.Errorf("disabledBy = %q, want empty", disabledBy)
	}
}

// The rename from YANTR_TELEMETRY to YANTR_PRESENCE must not re-enable pings
// for installs that had already opted out under the old name.
func TestResolveConfigLegacyOptOutStillHonoured(t *testing.T) {
	cases := []struct {
		name  string
		pairs map[string]string
	}{
		{"legacy name", map[string]string{"YANTR_TELEMETRY": "false"}},
		{"current name", map[string]string{"YANTR_PRESENCE": "false"}},
		{"both names", map[string]string{"YANTR_TELEMETRY": "false", "YANTR_PRESENCE": "false"}},
		{"legacy name alongside a topic", map[string]string{
			"YANTR_TELEMETRY":      "false",
			"YANTR_PRESENCE_TOPIC": "https://example.test/topic",
		}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, enabled, disabledBy := resolveConfig(env(tc.pairs))
			if enabled {
				t.Fatal("expected pings to be disabled")
			}
			if disabledBy == "" {
				t.Error("expected the disabling variable to be reported for the log line")
			}
		})
	}
}

func TestResolveConfigTopicOverride(t *testing.T) {
	cases := []struct {
		name  string
		pairs map[string]string
		want  string
	}{
		{"current name", map[string]string{"YANTR_PRESENCE_TOPIC": "https://example.test/a"}, "https://example.test/a"},
		{"legacy name", map[string]string{"YANTR_TELEMETRY_TOPIC": "https://example.test/b"}, "https://example.test/b"},
		{
			"current name wins",
			map[string]string{
				"YANTR_PRESENCE_TOPIC":  "https://example.test/new",
				"YANTR_TELEMETRY_TOPIC": "https://example.test/old",
			},
			"https://example.test/new",
		},
		{"trailing slashes trimmed", map[string]string{"YANTR_PRESENCE_TOPIC": "https://example.test/c///"}, "https://example.test/c"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			topic, enabled, _ := resolveConfig(env(tc.pairs))
			if topic != tc.want {
				t.Errorf("topic = %q, want %q", topic, tc.want)
			}
			if !enabled {
				t.Error("a topic override must not change enablement")
			}
		})
	}
}
