package main

import (
	"bytes"
	"strings"

	"github.com/BurntSushi/toml"
)

// CodexTOMLHandler handles Codex config.toml (notify = ["python3", "path"]).
type CodexTOMLHandler struct{}

func (h CodexTOMLHandler) Installed(current []byte, a Agent) (bool, error) {
	if len(current) == 0 {
		return false, nil
	}
	var doc map[string]any
	if _, err := toml.Decode(string(current), &doc); err != nil {
		return false, err
	}
	notify, ok := doc["notify"].([]any)
	if !ok {
		return false, nil
	}
	for _, v := range notify {
		if s, ok := v.(string); ok && strings.Contains(s, "goob_codex_notify.py") {
			return true, nil
		}
	}
	return false, nil
}

func (h CodexTOMLHandler) Install(current []byte, a Agent, hookCmd string) ([]byte, error) {
	var doc map[string]any
	if len(current) == 0 {
		doc = make(map[string]any)
	} else if _, err := toml.Decode(string(current), &doc); err != nil {
		return nil, err
	}
	// hookCmd for Codex is the path to goob_codex_notify.py
	// The notify array is ["python3", "<path>"]
	doc["notify"] = []string{"python3", hookCmd}
	var buf bytes.Buffer
	enc := toml.NewEncoder(&buf)
	if err := enc.Encode(doc); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (h CodexTOMLHandler) Remove(current []byte, a Agent) ([]byte, error) {
	if len(current) == 0 {
		return current, nil
	}
	var doc map[string]any
	if _, err := toml.Decode(string(current), &doc); err != nil {
		return nil, err
	}
	delete(doc, "notify")
	var buf bytes.Buffer
	enc := toml.NewEncoder(&buf)
	if err := enc.Encode(doc); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
