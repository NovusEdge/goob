package main

import "testing"

func TestRegistryAgents(t *testing.T) {
	if len(Registry) == 0 {
		t.Fatal("registry should have agents")
	}
	for _, a := range Registry {
		if a.ID == "" {
			t.Error("agent should have ID")
		}
		if a.Name == "" {
			t.Error("agent should have Name")
		}
		if a.ConfigPath == "" {
			t.Error("agent should have ConfigPath")
		}
		if a.Handler == nil {
			t.Errorf("agent %s should have Handler", a.ID)
		}
		// validate EventMap values are valid tokens (except Codex which uses dispatcher)
		if a.ID != "codex" {
			for event, token := range a.EventMap {
				if !ValidTokens[token] {
					t.Errorf("agent %s event %s maps to invalid token %s", a.ID, event, token)
				}
			}
		}
	}
}

func TestFindAgent(t *testing.T) {
	a := FindAgent("claude-code")
	if a == nil {
		t.Fatal("should find claude-code")
	}
	if a.Name != "Claude Code" {
		t.Errorf("expected Claude Code, got %s", a.Name)
	}
	if FindAgent("nonexistent") != nil {
		t.Error("should return nil for unknown agent")
	}
}
