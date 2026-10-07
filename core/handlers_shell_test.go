package main

import (
	"context"
	"errors"
	"reflect"
	"testing"
)

func TestDetectContainerShell(t *testing.T) {
	for _, tc := range []struct {
		name    string
		found   string
		wantTry []string
		wantErr bool
	}{
		{name: "bash preferred", found: "/bin/bash", wantTry: []string{"/bin/bash"}},
		{name: "ash fallback", found: "/bin/ash", wantTry: []string{"/bin/bash", "/bin/ash"}},
		{name: "sh fallback", found: "/bin/sh", wantTry: shellCandidates},
		{name: "distroless", wantTry: shellCandidates},
		{name: "probe error", wantTry: []string{"/bin/bash"}, wantErr: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var tried []string
			probe := func(_ context.Context, id, shell string) (bool, error) {
				if id != "container-id" {
					t.Fatalf("unexpected container ID: %s", id)
				}
				tried = append(tried, shell)
				if tc.wantErr {
					return false, errors.New("podman unavailable")
				}
				return shell == tc.found, nil
			}
			got, err := detectContainerShell(context.Background(), "container-id", probe)
			if (err != nil) != tc.wantErr {
				t.Fatalf("detectContainerShell error = %v, wantErr = %v", err, tc.wantErr)
			}
			if got != tc.found {
				t.Errorf("shell = %q, want %q", got, tc.found)
			}
			if !reflect.DeepEqual(tried, tc.wantTry) {
				t.Errorf("probes = %v, want %v", tried, tc.wantTry)
			}
		})
	}
}
