package main

import (
	"os"
	"regexp"
	"testing"
)

// TestClaudeEventMapMatchesPython verifies the Go registry's Claude EventMap
// matches the EVENTS dict in goob_hook.py.
func TestClaudeEventMapMatchesPython(t *testing.T) {
	// read the Python file
	data, err := os.ReadFile("../hooks/goob_hook.py")
	if err != nil {
		t.Fatalf("failed to read goob_hook.py: %v", err)
	}
	// extract EVENTS dict entries with regex
	// matches: "EventName": "token",
	re := regexp.MustCompile(`"(\w+)":\s*"(\w+)"`)
	matches := re.FindAllStringSubmatch(string(data), -1)
	pyEvents := make(map[string]string)
	for _, m := range matches {
		if len(m) == 3 && ValidTokens[m[2]] {
			pyEvents[m[1]] = m[2]
		}
	}
	if len(pyEvents) == 0 {
		t.Fatal("failed to parse EVENTS from goob_hook.py")
	}
	// get Claude agent from registry
	claude := FindAgent("claude-code")
	if claude == nil {
		t.Fatal("claude-code not in registry")
	}
	// compare maps
	for event, token := range claude.EventMap {
		if pyToken, ok := pyEvents[event]; !ok {
			t.Errorf("Go has event %s but Python doesn't", event)
		} else if pyToken != token {
			t.Errorf("event %s: Go=%s, Python=%s", event, token, pyToken)
		}
	}
	for event, token := range pyEvents {
		if _, ok := claude.EventMap[event]; !ok {
			t.Errorf("Python has event %s=%s but Go doesn't", event, token)
		}
	}
}
