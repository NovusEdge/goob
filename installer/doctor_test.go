package main

import "testing"

func TestVersionGE(t *testing.T) {
	tests := []struct {
		v, min string
		want   bool
	}{
		{"4.7", "4.7", true},
		{"4.8", "4.7", true},
		{"4.6", "4.7", false},
		{"5.0", "4.7", true},
		{"3.12", "3.0", true},
		{"2.9", "3.0", false},
		{"", "4.7", false},
		{"4.7.stable", "4.7", true},
	}
	for _, tc := range tests {
		got := versionGE(tc.v, tc.min)
		if got != tc.want {
			t.Errorf("versionGE(%q, %q) = %v, want %v", tc.v, tc.min, got, tc.want)
		}
	}
}
