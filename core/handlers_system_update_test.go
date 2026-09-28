package main

import (
	"errors"
	"strings"
	"testing"
)

func TestImageIDsDiffer(t *testing.T) {
	const alpine = "docker.io/library/alpine:latest"
	const pg = "docker.io/library/postgres:16-alpine"

	cases := []struct {
		name          string
		refs          []string
		before, after map[string]string
		want          bool
	}{
		{
			name:   "digest changed on a rolling tag",
			refs:   []string{alpine},
			before: map[string]string{alpine: "sha256:old"},
			after:  map[string]string{alpine: "sha256:new"},
			want:   true,
		},
		{
			name:   "identical digests means no update",
			refs:   []string{alpine},
			before: map[string]string{alpine: "sha256:same"},
			after:  map[string]string{alpine: "sha256:same"},
			want:   false,
		},
		{
			name:   "first-time pull of an image not previously local",
			refs:   []string{pg},
			before: map[string]string{alpine: "sha256:old"},
			after:  map[string]string{alpine: "sha256:old", pg: "sha256:new"},
			want:   true,
		},
		{
			name:   "one of several stack images updated",
			refs:   []string{alpine, pg},
			before: map[string]string{alpine: "sha256:a", pg: "sha256:b"},
			after:  map[string]string{alpine: "sha256:a", pg: "sha256:b2"},
			want:   true,
		},
		{
			name:   "unrelated image churn is ignored",
			refs:   []string{alpine},
			before: map[string]string{alpine: "sha256:a"},
			after:  map[string]string{alpine: "sha256:a", "docker.io/library/redis:7": "sha256:z"},
			want:   false,
		},
		{
			name:   "image absent from both snapshots is inconclusive",
			refs:   []string{pg},
			before: map[string]string{alpine: "sha256:a"},
			after:  map[string]string{alpine: "sha256:a"},
			want:   false,
		},
		{
			name:   "no image refs declared",
			refs:   nil,
			before: map[string]string{alpine: "sha256:a"},
			after:  map[string]string{alpine: "sha256:b"},
			want:   false,
		},
		{
			name:   "nil snapshots on both sides",
			refs:   []string{alpine},
			before: nil,
			after:  nil,
			want:   false,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := imageIDsDiffer(c.refs, c.before, c.after); got != c.want {
				t.Errorf("imageIDsDiffer = %v, want %v", got, c.want)
			}
		})
	}
}

// TestOldStringHeuristicMissesRealPulls is the regression guard for the bug
// this change fixes. The output below is verbatim podman-compose 1.6.0 output
// from a successful first-time pull of alpine:3.19 — a real update. The old
// implementation decided "was this an update?" by searching that output for
// Docker Compose v2 phrasings, none of which podman-compose emits, so every
// real update was reported as "already up to date".
func TestOldStringHeuristicMissesRealPulls(t *testing.T) {
	const realPodmanComposePullOutput = `Trying to pull docker.io/library/alpine:3.19...
Getting image source signatures
Copying blob sha256:17a39c0ba978cc27001e9c56a480f98106e1ab74bd56eb302f9fd4cf758ea43f
Copying config sha256:83b2b6703a620bf2e001ab57f7adc414d891787b3c59859b1b62909e48dd2242
Writing manifest to image destination
83b2b6703a620bf2e001ab57f7adc414d891787b3c59859b1b62909e48dd2242`

	combined := strings.ToLower(realPodmanComposePullOutput)
	oldDetected := strings.Contains(combined, "downloaded newer image") ||
		strings.Contains(combined, "pull complete") ||
		strings.Contains(combined, "digest:")

	if oldDetected {
		t.Fatal("expected the old string heuristic to miss this real update; " +
			"if podman-compose changed its wording this regression test is stale")
	}

	// The new detection, given the matching engine-side snapshot pair.
	const ref = "docker.io/library/alpine:3.19"
	detected := imageIDsDiffer([]string{ref},
		map[string]string{}, // nothing local before the pull
		map[string]string{ref: "sha256:83b2b6703a620bf2e001ab57f7adc414d891787b3c59859b1b62909e48dd2242"},
	)
	if !detected {
		t.Error("digest-diff detection should report this pull as an update")
	}
}

func TestClassifyUpdate(t *testing.T) {
	const alpine = "docker.io/library/alpine:latest"
	snap := func(id string) map[string]string { return map[string]string{alpine: id} }
	snapErr := errors.New("snapshot failed")

	cases := []struct {
		name                string
		refs                []string
		before, after       map[string]string
		beforeErr, afterErr error
		want                updateCheck
	}{
		{
			name:   "identical digests skip the recreate",
			refs:   []string{alpine},
			before: snap("sha256:same"), after: snap("sha256:same"),
			want: updateUnchanged,
		},
		{
			name:   "moved digest recreates and counts",
			refs:   []string{alpine},
			before: snap("sha256:old"), after: snap("sha256:new"),
			want: updateChanged,
		},
		{
			name:   "first-time pull counts as changed",
			refs:   []string{alpine},
			before: map[string]string{}, after: snap("sha256:new"),
			want: updateChanged,
		},
		{
			name:   "unrelated image churn is unchanged",
			refs:   []string{alpine},
			before: snap("sha256:a"), after: map[string]string{alpine: "sha256:a", "docker.io/library/redis:7": "sha256:z"},
			want: updateUnchanged,
		},
		{
			name:   "pre-pull snapshot failure recreates without counting",
			refs:   []string{alpine},
			before: nil, after: snap("sha256:new"),
			beforeErr: snapErr,
			want:      updateUnknown,
		},
		{
			name:   "post-pull snapshot failure recreates without counting",
			refs:   []string{alpine},
			before: snap("sha256:old"), after: nil,
			afterErr: snapErr,
			want:     updateUnknown,
		},
		{
			name:   "no image refs recreates without counting",
			refs:   nil,
			before: snap("sha256:a"), after: snap("sha256:b"),
			want: updateUnknown,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := classifyUpdate(c.refs, c.before, c.beforeErr, c.after, c.afterErr); got != c.want {
				t.Errorf("classifyUpdate = %v, want %v", got, c.want)
			}
		})
	}
}
