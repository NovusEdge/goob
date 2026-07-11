package main

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
)

func TestSettingsJSONHandler_InstallEmpty(t *testing.T) {
	h := SettingsJSONHandler{}
	a := Agent{
		ID:       "test",
		EventMap: map[string]string{"SessionStart": "wake"},
	}
	out, err := h.Install(nil, a, "/path/to/goob_hook.py")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(out), "goob_hook.py") {
		t.Error("output should contain hook marker")
	}
	if !strings.Contains(string(out), "--token wake") {
		t.Error("output should contain token directive")
	}
}

func TestSettingsJSONHandler_InstallPreservesExisting(t *testing.T) {
	h := SettingsJSONHandler{}
	a := Agent{
		ID:       "test",
		EventMap: map[string]string{"SessionStart": "wake"},
	}
	existing := `{"permissions": {"allow": ["Bash(ls:*)"]}}`
	out, err := h.Install([]byte(existing), a, "/path/to/goob_hook.py")
	if err != nil {
		t.Fatal(err)
	}
	var doc map[string]any
	if err := json.Unmarshal(out, &doc); err != nil {
		t.Fatal(err)
	}
	// check permissions preserved
	if _, ok := doc["permissions"]; !ok {
		t.Error("existing permissions should be preserved")
	}
	// check hooks added
	if _, ok := doc["hooks"]; !ok {
		t.Error("hooks should be added")
	}
}

func TestSettingsJSONHandler_Idempotent(t *testing.T) {
	h := SettingsJSONHandler{}
	a := Agent{
		ID:       "test",
		EventMap: map[string]string{"SessionStart": "wake"},
	}
	first, err := h.Install(nil, a, "/path/to/goob_hook.py")
	if err != nil {
		t.Fatal(err)
	}
	second, err := h.Install(first, a, "/path/to/goob_hook.py")
	if err != nil {
		t.Fatal(err)
	}
	// compare parsed JSON (order-independent)
	var doc1, doc2 map[string]any
	if err := json.Unmarshal(first, &doc1); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(second, &doc2); err != nil {
		t.Fatal(err)
	}
	// count hooks - should be same (no duplicates)
	hooks1 := doc1["hooks"].(map[string]any)["SessionStart"].(map[string]any)["hooks"].([]any)
	hooks2 := doc2["hooks"].(map[string]any)["SessionStart"].(map[string]any)["hooks"].([]any)
	if len(hooks1) != len(hooks2) {
		t.Errorf("hook counts differ: %d vs %d (duplicates?)", len(hooks1), len(hooks2))
	}
	if len(hooks1) != 1 {
		t.Errorf("expected 1 hook, got %d", len(hooks1))
	}
}

func TestSettingsJSONHandler_RoundTrip(t *testing.T) {
	h := SettingsJSONHandler{}
	a := Agent{
		ID:       "test",
		EventMap: map[string]string{"SessionStart": "wake"},
	}
	original := `{}`
	installed, err := h.Install([]byte(original), a, "/path/to/goob_hook.py")
	if err != nil {
		t.Fatal(err)
	}
	removed, err := h.Remove(installed, a)
	if err != nil {
		t.Fatal(err)
	}
	// should return to original (empty hooks)
	var doc map[string]any
	if err := json.Unmarshal(removed, &doc); err != nil {
		t.Fatal(err)
	}
	if _, ok := doc["hooks"]; ok {
		t.Error("hooks should be empty after remove")
	}
}

func TestSettingsJSONHandler_RealFixture(t *testing.T) {
	data, err := os.ReadFile("testdata/claude_settings.json")
	if err != nil {
		t.Skip("no real fixture available")
	}
	h := SettingsJSONHandler{}
	a := Agent{
		ID:       "claude-code",
		EventMap: map[string]string{"SessionStart": "wake", "Stop": "done"},
	}
	out, err := h.Install(data, a, "/path/to/goob_hook.py")
	if err != nil {
		t.Fatal(err)
	}
	// verify original content preserved
	var doc map[string]any
	if err := json.Unmarshal(out, &doc); err != nil {
		t.Fatal(err)
	}
	if _, ok := doc["permissions"]; !ok {
		t.Error("permissions should be preserved")
	}
	// verify hooks added
	if !strings.Contains(string(out), "goob_hook.py") {
		t.Error("hook should be added")
	}
}

func TestCodexTOMLHandler_Install(t *testing.T) {
	h := CodexTOMLHandler{}
	a := Agent{ID: "codex"}
	existing := `model = "gpt-4o"
sandbox = true
`
	out, err := h.Install([]byte(existing), a, "/path/to/goob_codex_notify.py")
	if err != nil {
		t.Fatal(err)
	}
	s := string(out)
	if !strings.Contains(s, "goob_codex_notify.py") {
		t.Error("should contain dispatcher path")
	}
	if !strings.Contains(s, "model") {
		t.Error("should preserve existing keys")
	}
}

func TestCodexTOMLHandler_RoundTrip(t *testing.T) {
	h := CodexTOMLHandler{}
	a := Agent{ID: "codex"}
	original := `model = "gpt-4o"
sandbox = true
`
	installed, err := h.Install([]byte(original), a, "/path/to/goob_codex_notify.py")
	if err != nil {
		t.Fatal(err)
	}
	removed, err := h.Remove(installed, a)
	if err != nil {
		t.Fatal(err)
	}
	// should have model and sandbox but no notify
	s := string(removed)
	if strings.Contains(s, "notify") {
		t.Error("notify should be removed")
	}
	if !strings.Contains(s, "model") {
		t.Error("model should be preserved")
	}
}
