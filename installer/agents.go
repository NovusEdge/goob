package main

func init() {
	// Claude Code - verified settings.json format
	RegisterAgent(Agent{
		ID:         "claude-code",
		Name:       "Claude Code",
		ConfigPath: "~/.claude/settings.json",
		Handler:    SettingsJSONHandler{},
		EventMap: map[string]string{
			"SessionStart":      "wake",
			"UserPromptSubmit":  "thinking",
			"PreToolUse":        "working",
			"PostToolUse":       "working",
			"SubagentStop":      "subagent",
			"Stop":              "done",
			"SessionEnd":        "sleep",
		},
	})

	// Cursor - hooks.json (same structure as Claude)
	RegisterAgent(Agent{
		ID:         "cursor",
		Name:       "Cursor",
		ConfigPath: "~/.cursor/hooks.json",
		Handler:    SettingsJSONHandler{},
		EventMap: map[string]string{
			"SessionStart":      "wake",
			"UserPromptSubmit":  "thinking",
			"PreToolUse":        "working",
			"PostToolUse":       "working",
			"Stop":              "done",
			"SessionEnd":        "sleep",
		},
	})

	// Gemini CLI - same settings.json format as Claude
	RegisterAgent(Agent{
		ID:         "gemini",
		Name:       "Gemini CLI",
		ConfigPath: "~/.gemini/settings.json",
		Handler:    SettingsJSONHandler{},
		EventMap: map[string]string{
			"SessionStart":      "wake",
			"UserPromptSubmit":  "thinking",
			"PreToolUse":        "working",
			"PostToolUse":       "working",
			"Stop":              "done",
			"SessionEnd":        "sleep",
		},
	})

	// Codex - config.toml with notify program
	// Uses goob_codex_notify.py dispatcher instead of goob_hook.py
	RegisterAgent(Agent{
		ID:         "codex",
		Name:       "Codex",
		ConfigPath: "~/.codex/config.toml",
		Handler:    CodexTOMLHandler{},
		EventMap:   map[string]string{}, // event mapping is in the dispatcher script
	})
}
