package compose

import "testing"

// ParseEnvFile strips a matching pair of surrounding quotes. The regression
// guard here is the one-character value: `"` satisfies both HasPrefix and
// HasSuffix, and the old val[1:len(val)-1] was val[1:0], which panics. This is
// reachable from the deploy form (a lone quote is accepted as an env value,
// written to the project .env, then read back here) and ParseEnvFile is called
// from the reaper and boot goroutines, where a panic is not recovered.
func TestParseEnvFile(t *testing.T) {
	cases := []struct {
		name    string
		content string
		want    map[string]string
	}{
		{
			name:    "plain values",
			content: "FOO=bar\nBAZ=qux\n",
			want:    map[string]string{"FOO": "bar", "BAZ": "qux"},
		},
		{
			name:    "strips double quotes",
			content: `PASSWORD="hunter2"`,
			want:    map[string]string{"PASSWORD": "hunter2"},
		},
		{
			name:    "strips single quotes",
			content: `PASSWORD='hunter2'`,
			want:    map[string]string{"PASSWORD": "hunter2"},
		},
		{
			name:    "empty quoted value",
			content: `EMPTY=""`,
			want:    map[string]string{"EMPTY": ""},
		},
		{
			// The panic case. A single double quote is not a quoted string, so it
			// is kept verbatim rather than slicing out of range.
			name:    "lone double quote does not panic",
			content: `QUOTE="`,
			want:    map[string]string{"QUOTE": `"`},
		},
		{
			name:    "lone single quote does not panic",
			content: `QUOTE='`,
			want:    map[string]string{"QUOTE": `'`},
		},
		{
			// Both quotes present but mismatched: HasSuffix/HasPrefix disagree,
			// so nothing is stripped.
			name:    "mismatched quotes preserved",
			content: `VAL="'`,
			want:    map[string]string{"VAL": `"'`},
		},
		{
			// Opens with a quote, never closes it: left as-is.
			name:    "unterminated quote preserved",
			content: `VAL="abc`,
			want:    map[string]string{"VAL": `"abc`},
		},
		{
			// Quotes in the middle are not surrounding quotes.
			name:    "inner quotes preserved",
			content: `VAL=ab"cd"ef`,
			want:    map[string]string{"VAL": `ab"cd"ef`},
		},
		{
			// Two characters: the length guard must not reject this.
			name:    "two quote characters",
			content: `VAL=""`,
			want:    map[string]string{"VAL": ""},
		},
		{
			name:    "empty value",
			content: "EMPTY=",
			want:    map[string]string{"EMPTY": ""},
		},
		{
			name:    "value containing equals sign",
			content: "URL=postgres://u:p@h/db?x=1",
			want:    map[string]string{"URL": "postgres://u:p@h/db?x=1"},
		},
		{
			name:    "key is trimmed",
			content: "  SPACED  =value",
			want:    map[string]string{"SPACED": "value"},
		},
		{
			name:    "comments and blanks skipped",
			content: "# comment\n\nFOO=bar\n   \n#another=ignored\n",
			want:    map[string]string{"FOO": "bar"},
		},
		{
			name:    "line without equals skipped",
			content: "JUST_A_KEY\nFOO=bar\n=novalue\n",
			want:    map[string]string{"FOO": "bar"},
		},
		{
			name:    "empty input",
			content: "",
			want:    map[string]string{},
		},
		{
			name:    "lone quote does not poison later keys",
			content: "A=\"\nB=keep\n",
			want:    map[string]string{"A": `"`, "B": "keep"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := ParseEnvFile(tc.content)
			if len(got) != len(tc.want) {
				t.Fatalf("ParseEnvFile(%q) = %v, want %v", tc.content, got, tc.want)
			}
			for k, want := range tc.want {
				if got[k] != want {
					t.Errorf("ParseEnvFile(%q)[%q] = %q, want %q", tc.content, k, got[k], want)
				}
			}
		})
	}
}

// A round trip through the writer is what actually makes the lone-quote case
// reachable, so exercise the pair together rather than the parser alone.
//
// WriteProjectEnv emits values unquoted ("KEY=" + value), so a value of a bare
// double quote lands in the .env as QUOTE=" and comes straight back into
// ParseEnvFile — the exact input that used to panic.
func TestParseEnvFileRoundTripsLoneQuoteThroughWriter(t *testing.T) {
	env := map[string]interface{}{
		"QUOTE":   `"`,
		"APOSTRO": `'`,
		"NORMAL":  "value",
		"HAS_EQ":  "a=b",
		"SPACED":  "one two",
	}
	dir := t.TempDir()
	if _, err := WriteProjectEnv(dir, "proj", env); err != nil {
		t.Fatalf("WriteProjectEnv: %v", err)
	}

	got, err := LoadProjectEnv(dir, "proj")
	if err != nil {
		t.Fatalf("LoadProjectEnv: %v", err)
	}

	for k, want := range map[string]string{
		"QUOTE":   `"`,
		"APOSTRO": `'`,
		"NORMAL":  "value",
		"HAS_EQ":  "a=b",
		"SPACED":  "one two",
	} {
		if got[k] != want {
			t.Errorf("round trip %s = %q, want %q", k, got[k], want)
		}
	}
}
