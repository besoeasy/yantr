package presence

import "testing"

// env builds a getenv func over a fixed map.
func env(pairs map[string]string) func(string) string {
	return func(key string) string { return pairs[key] }
}

func TestResolveConfigDefaults(t *testing.T) {
	topic, enabled := resolveConfig(env(nil))

	if topic != "https://ntfy.sh/yantr" {
		t.Errorf("default topic = %q, want the public ntfy topic", topic)
	}
	if !enabled {
		t.Error("default should be enabled")
	}
}

func TestResolveConfigOptOut(t *testing.T) {
	cases := []struct {
		name   string
		pairs  map[string]string
		wantOn bool
	}{
		{"not set", nil, true},
		{"explicitly true", map[string]string{"YANTR_PRESENCE": "true"}, true},
		{"false disables", map[string]string{"YANTR_PRESENCE": "false"}, false},
		{"false wins over a topic", map[string]string{
			"YANTR_PRESENCE":       "false",
			"YANTR_PRESENCE_TOPIC": "https://example.test/topic",
		}, false},
		// Anything that is not exactly "false" leaves pings on: the variable
		// is an explicit opt-out, not a truthy flag.
		{"FALSE is not false", map[string]string{"YANTR_PRESENCE": "FALSE"}, true},
		{"0 is not false", map[string]string{"YANTR_PRESENCE": "0"}, true},
		{"empty is not false", map[string]string{"YANTR_PRESENCE": ""}, true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, enabled := resolveConfig(env(tc.pairs))
			if enabled != tc.wantOn {
				t.Errorf("enabled = %v, want %v", enabled, tc.wantOn)
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
		{"not set uses the public topic", nil, "https://ntfy.sh/yantr"},
		{"override", map[string]string{"YANTR_PRESENCE_TOPIC": "https://example.test/a"}, "https://example.test/a"},
		{"trailing slashes trimmed", map[string]string{"YANTR_PRESENCE_TOPIC": "https://example.test/c///"}, "https://example.test/c"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			topic, enabled := resolveConfig(env(tc.pairs))
			if topic != tc.want {
				t.Errorf("topic = %q, want %q", topic, tc.want)
			}
			if !enabled {
				t.Error("a topic override must not change enablement")
			}
		})
	}
}
