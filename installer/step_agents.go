package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
)

type agentsModel struct {
	ctx     *Context
	cursor  int
	states  []agentState
	ready   bool
}

type agentState struct {
	agent     Agent
	installed bool
	exists    bool // config file exists
	selected  bool
}

func newAgentsModel(ctx *Context) agentsModel {
	return agentsModel{ctx: ctx}
}

func (m agentsModel) Init() tea.Cmd {
	return func() tea.Msg {
		var states []agentState
		for _, a := range Registry {
			path := ExpandPath(a.ConfigPath)
			exists := false
			installed := false
			if data, err := os.ReadFile(path); err == nil {
				exists = true
				installed, _ = a.Handler.Installed(data, a)
			}
			states = append(states, agentState{
				agent:     a,
				installed: installed,
				exists:    exists,
				selected:  installed, // pre-select installed agents
			})
		}
		return agentsLoadedMsg{states: states}
	}
}

type agentsLoadedMsg struct{ states []agentState }

func (m agentsModel) Update(msg tea.Msg) (agentsModel, tea.Cmd) {
	switch msg := msg.(type) {
	case agentsLoadedMsg:
		m.states = msg.states
		m.ready = true
		// sync context
		for _, s := range m.states {
			m.ctx.SelectedIDs[s.agent.ID] = s.selected
		}
	case tea.KeyMsg:
		if !m.ready {
			return m, nil
		}
		switch {
		case key.Matches(msg, keys.Next):
			m.buildPlan()
			return m, nextStep
		case key.Matches(msg, key.NewBinding(key.WithKeys("up", "k"))):
			if m.cursor > 0 {
				m.cursor--
			}
		case key.Matches(msg, key.NewBinding(key.WithKeys("down", "j"))):
			if m.cursor < len(m.states)-1 {
				m.cursor++
			}
		case key.Matches(msg, keys.Select):
			m.states[m.cursor].selected = !m.states[m.cursor].selected
			m.ctx.SelectedIDs[m.states[m.cursor].agent.ID] = m.states[m.cursor].selected
		}
	}
	return m, nil
}

func (m *agentsModel) buildPlan() {
	for _, s := range m.states {
		s := s // capture loop variable
		a := s.agent
		path := ExpandPath(a.ConfigPath)
		if s.selected && !s.installed {
			// install hook
			hookPath, err := HookPath()
			if err != nil {
				continue
			}
			// ponytail: Codex uses a different dispatcher
			cmdPath := hookPath
			if a.ID == "codex" {
				cmdPath, err = CodexDispatcherPath()
				if err != nil {
					continue
				}
			}
			// capture for closure
			agent, cmd, p := a, cmdPath, path
			desc := fmt.Sprintf("Add goob hooks to %s (%s)", a.Name, path)
			m.ctx.Plan.Add(NewWriteFile(p, desc, func(current []byte) ([]byte, error) {
				return agent.Handler.Install(current, agent, cmd)
			}))
		} else if !s.selected && s.installed {
			// capture for closure
			agent, p := a, path
			desc := fmt.Sprintf("Remove goob hooks from %s (%s)", a.Name, path)
			m.ctx.Plan.Add(NewWriteFile(p, desc, func(current []byte) ([]byte, error) {
				return agent.Handler.Remove(current, agent)
			}))
		}
	}
}

func (m agentsModel) View() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render("Select AI coding agents to integrate"))
	b.WriteString("\n\n")
	if !m.ready {
		b.WriteString("Loading...\n")
		return b.String()
	}
	for i, s := range m.states {
		cursor := " "
		if i == m.cursor {
			cursor = ">"
		}
		check := "[ ]"
		if s.selected {
			check = selectedStyle.Render("[x]")
		}
		status := ""
		if s.installed {
			status = okStyle.Render(" (installed)")
		} else if !s.exists {
			status = dimStyle.Render(" (config not found)")
		}
		b.WriteString(fmt.Sprintf("%s %s %s%s\n", cursor, check, s.agent.Name, status))
	}
	b.WriteString("\n")
	b.WriteString(dimStyle.Render("↑/↓ navigate, space toggle, enter continue"))
	return b.String()
}
