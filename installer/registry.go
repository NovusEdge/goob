package main

// ValidTokens are the goob tokens the hook recognizes.
var ValidTokens = map[string]bool{
	"wake": true, "thinking": true, "working": true,
	"subagent": true, "done": true, "sleep": true,
}

// Agent describes a coding agent that can receive goob hooks.
type Agent struct {
	ID         string
	Name       string
	ConfigPath string // ~ expanded at runtime
	Handler    FormatHandler
	EventMap   map[string]string // agent event name -> goob token
}

// FormatHandler parses and modifies an agent's config format.
type FormatHandler interface {
	// Installed returns true if goob hooks are present.
	Installed(current []byte, a Agent) (bool, error)
	// Install adds goob hooks, returning the new content.
	// hookCmd is the absolute path to goob_hook.py with args.
	Install(current []byte, a Agent, hookCmd string) ([]byte, error)
	// Remove strips goob hooks, returning the cleaned content.
	Remove(current []byte, a Agent) ([]byte, error)
}

// Registry holds all known agents.
var Registry = []Agent{}

// RegisterAgent adds an agent to the registry.
func RegisterAgent(a Agent) {
	Registry = append(Registry, a)
}

// FindAgent returns an agent by ID, or nil if not found.
func FindAgent(id string) *Agent {
	for i := range Registry {
		if Registry[i].ID == id {
			return &Registry[i]
		}
	}
	return nil
}
