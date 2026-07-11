package main

import (
	"strings"
	"testing"
)

func TestParseEnv(t *testing.T) {
	data := `# comment
GOOB_MODEL=gpt-4o-mini
export GOOB_HSM=1
DEBUG="test value"
`
	lines := ParseEnv([]byte(data))
	if len(lines) != 4 {
		t.Errorf("expected 4 lines, got %d", len(lines))
	}
	// check comment
	if lines[0].IsKV {
		t.Error("comment should not be KV")
	}
	// check simple KV
	if lines[1].Key != "GOOB_MODEL" || lines[1].Value != "gpt-4o-mini" {
		t.Errorf("expected GOOB_MODEL=gpt-4o-mini, got %s=%s", lines[1].Key, lines[1].Value)
	}
	// check export
	if !lines[2].Export || lines[2].Key != "GOOB_HSM" {
		t.Error("expected export GOOB_HSM")
	}
	// check quoted value
	if lines[3].Value != "test value" {
		t.Errorf("expected 'test value', got '%s'", lines[3].Value)
	}
}

func TestMergeEnv_Update(t *testing.T) {
	data := `GOOB_MODEL=old
GOOB_HSM=0
`
	lines := ParseEnv([]byte(data))
	merged := MergeEnv(lines, map[string]string{
		"GOOB_MODEL": "new",
		"GOOB_HSM":   "", // empty = preserve
	})
	out := string(SerializeEnv(merged))
	if !strings.Contains(out, "GOOB_MODEL=new") {
		t.Error("should update GOOB_MODEL")
	}
	if !strings.Contains(out, "GOOB_HSM=0") {
		t.Error("should preserve GOOB_HSM")
	}
}

func TestMergeEnv_AddNew(t *testing.T) {
	data := `GOOB_MODEL=test
`
	lines := ParseEnv([]byte(data))
	merged := MergeEnv(lines, map[string]string{
		"DEBUG": "1",
	})
	out := string(SerializeEnv(merged))
	if !strings.Contains(out, "GOOB_MODEL=test") {
		t.Error("should preserve existing")
	}
	if !strings.Contains(out, "DEBUG=1") {
		t.Error("should add new key")
	}
}

func TestMergeEnv_PreservesComments(t *testing.T) {
	data := `# this is a comment
GOOB_MODEL=test
`
	lines := ParseEnv([]byte(data))
	merged := MergeEnv(lines, map[string]string{
		"GOOB_MODEL": "new",
	})
	out := string(SerializeEnv(merged))
	if !strings.Contains(out, "# this is a comment") {
		t.Error("should preserve comments")
	}
}
