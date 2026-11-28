package main

import "testing"

func TestIsValidVolumeName(t *testing.T) {
	valid := []string{"data", "my-volume", "My_Vol.1", "a"}
	for _, name := range valid {
		if !isValidVolumeName(name) {
			t.Errorf("isValidVolumeName(%q) = false, want true", name)
		}
	}
	invalid := []string{"", "has space", "slash/name", "question?", "emoji💥", string(make([]byte, 300))}
	for _, name := range invalid {
		if isValidVolumeName(name) {
			t.Errorf("isValidVolumeName(%q) = true, want false", name)
		}
	}
}
