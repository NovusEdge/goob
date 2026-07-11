package main

import (
	"encoding/json"
	"fmt"
	"strings"
)

// Marker: any hook command containing this is a goob hook.
const HookMarker = "goob_hook.py"

// SettingsJSONHandler handles Claude Code / Cursor style settings.json.
// Structure: { "hooks": { "<Event>": { "hooks": [ {type, command}, ... ] } } }
type SettingsJSONHandler struct{}

func (h SettingsJSONHandler) Installed(current []byte, a Agent) (bool, error) {
	if len(current) == 0 {
		return false, nil
	}
	var doc map[string]any
	if err := json.Unmarshal(current, &doc); err != nil {
		return false, err
	}
	hooks, ok := doc["hooks"].(map[string]any)
	if !ok {
		return false, nil
	}
	for _, eventData := range hooks {
		ed, ok := eventData.(map[string]any)
		if !ok {
			continue
		}
		hookList, ok := ed["hooks"].([]any)
		if !ok {
			continue
		}
		for _, h := range hookList {
			hm, ok := h.(map[string]any)
			if !ok {
				continue
			}
			if cmd, ok := hm["command"].(string); ok && strings.Contains(cmd, HookMarker) {
				return true, nil
			}
		}
	}
	return false, nil
}

func (h SettingsJSONHandler) Install(current []byte, a Agent, hookCmd string) ([]byte, error) {
	// remove existing goob hooks first (idempotent)
	cleaned, err := h.Remove(current, a)
	if err != nil {
		return nil, err
	}
	var doc map[string]any
	if len(cleaned) == 0 {
		doc = make(map[string]any)
	} else if err := json.Unmarshal(cleaned, &doc); err != nil {
		return nil, err
	}
	hooks, ok := doc["hooks"].(map[string]any)
	if !ok {
		hooks = make(map[string]any)
		doc["hooks"] = hooks
	}
	// add hooks for each mapped event
	for agentEvent, goobToken := range a.EventMap {
		cmd := fmt.Sprintf("%s --token %s", hookCmd, goobToken)
		entry := map[string]any{"type": "command", "command": cmd}
		eventData, ok := hooks[agentEvent].(map[string]any)
		if !ok {
			eventData = make(map[string]any)
			hooks[agentEvent] = eventData
		}
		hookList, ok := eventData["hooks"].([]any)
		if !ok {
			hookList = []any{}
		}
		hookList = append(hookList, entry)
		eventData["hooks"] = hookList
	}
	return json.MarshalIndent(doc, "", "  ")
}

func (h SettingsJSONHandler) Remove(current []byte, a Agent) ([]byte, error) {
	if len(current) == 0 {
		return current, nil
	}
	var doc map[string]any
	if err := json.Unmarshal(current, &doc); err != nil {
		return nil, err
	}
	hooks, ok := doc["hooks"].(map[string]any)
	if !ok {
		return current, nil
	}
	for eventName, eventData := range hooks {
		ed, ok := eventData.(map[string]any)
		if !ok {
			continue
		}
		hookList, ok := ed["hooks"].([]any)
		if !ok {
			continue
		}
		filtered := []any{}
		for _, h := range hookList {
			hm, ok := h.(map[string]any)
			if !ok {
				filtered = append(filtered, h)
				continue
			}
			cmd, _ := hm["command"].(string)
			if !strings.Contains(cmd, HookMarker) {
				filtered = append(filtered, h)
			}
		}
		if len(filtered) == 0 {
			delete(ed, "hooks")
		} else {
			ed["hooks"] = filtered
		}
		if len(ed) == 0 {
			delete(hooks, eventName)
		}
	}
	if len(hooks) == 0 {
		delete(doc, "hooks")
	}
	return json.MarshalIndent(doc, "", "  ")
}
