package main

import (
	"strings"
	"testing"
)

func TestRingCapsAndKeepsNewest(t *testing.T) {
	r := newRing(3)
	for _, s := range []string{"a", "b", "c", "d"} {
		r.add(s)
	}
	got := r.text()
	if got != "b\nc\nd" {
		t.Fatalf("want %q, got %q", "b\nc\nd", got)
	}
	if n := len(strings.Split(got, "\n")); n != 3 {
		t.Fatalf("want 3 lines, got %d", n)
	}
}

func TestRingEmpty(t *testing.T) {
	if got := newRing(5).text(); got != "" {
		t.Fatalf("want empty, got %q", got)
	}
}
