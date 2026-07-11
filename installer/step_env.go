package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

type envModel struct {
	ctx    *Context
	inputs []textinput.Model
	focus  int
	ready  bool
}

const (
	envModel_ = iota
	envAPIKey
	envHSM
	envDebug
)

func newEnvModel(ctx *Context) envModel {
	model := textinput.New()
	model.Placeholder = "gpt-4o-mini"
	model.Prompt = "GOOB_MODEL: "
	model.Focus()

	apiKey := textinput.New()
	apiKey.Placeholder = "(leave blank to keep existing)"
	apiKey.Prompt = "GOOB_API_KEY: "
	apiKey.EchoMode = textinput.EchoPassword

	hsm := textinput.New()
	hsm.Placeholder = "1 or 0"
	hsm.Prompt = "GOOB_HSM: "

	debug := textinput.New()
	debug.Placeholder = "1 or 0"
	debug.Prompt = "DEBUG: "

	return envModel{
		ctx:    ctx,
		inputs: []textinput.Model{model, apiKey, hsm, debug},
	}
}

func (m envModel) Init() tea.Cmd {
	return func() tea.Msg {
		return envLoadedMsg{}
	}
}

type envLoadedMsg struct{}

func (m envModel) Update(msg tea.Msg) (envModel, tea.Cmd) {
	switch msg := msg.(type) {
	case envLoadedMsg:
		m.ready = true
		// load existing values
		if envPath, err := EnvPath(); err == nil {
			if data, err := os.ReadFile(envPath); err == nil {
				lines := ParseEnv(data)
				for _, l := range lines {
					if !l.IsKV {
						continue
					}
					switch l.Key {
					case "GOOB_MODEL":
						m.inputs[envModel_].SetValue(l.Value)
					case "GOOB_HSM":
						m.inputs[envHSM].SetValue(l.Value)
					case "DEBUG":
						m.inputs[envDebug].SetValue(l.Value)
					}
				}
			}
		}
		return m, textinput.Blink
	case tea.KeyMsg:
		if !m.ready {
			return m, nil
		}
		switch {
		case key.Matches(msg, keys.Next):
			if m.focus == len(m.inputs)-1 {
				m.buildPlan()
				return m, nextStep
			}
			m.inputs[m.focus].Blur()
			m.focus++
			m.inputs[m.focus].Focus()
			return m, textinput.Blink
		case key.Matches(msg, key.NewBinding(key.WithKeys("tab"))):
			m.inputs[m.focus].Blur()
			m.focus = (m.focus + 1) % len(m.inputs)
			m.inputs[m.focus].Focus()
			return m, textinput.Blink
		case key.Matches(msg, key.NewBinding(key.WithKeys("shift+tab"))):
			m.inputs[m.focus].Blur()
			m.focus = (m.focus - 1 + len(m.inputs)) % len(m.inputs)
			m.inputs[m.focus].Focus()
			return m, textinput.Blink
		}
	}
	// update focused input
	var cmd tea.Cmd
	m.inputs[m.focus], cmd = m.inputs[m.focus].Update(msg)
	return m, cmd
}

func (m *envModel) buildPlan() {
	updates := make(map[string]string)
	if v := m.inputs[envModel_].Value(); v != "" {
		updates["GOOB_MODEL"] = v
	}
	if v := m.inputs[envAPIKey].Value(); v != "" {
		updates["GOOB_API_KEY"] = v
	}
	if v := m.inputs[envHSM].Value(); v != "" {
		updates["GOOB_HSM"] = v
	}
	if v := m.inputs[envDebug].Value(); v != "" {
		updates["DEBUG"] = v
	}
	m.ctx.EnvUpdates = updates
	if len(updates) == 0 {
		return
	}
	envPath, err := EnvPath()
	if err != nil {
		return
	}
	m.ctx.Plan.Add(NewWriteFile(envPath, fmt.Sprintf("Update %s", envPath), func(current []byte) ([]byte, error) {
		lines := ParseEnv(current)
		merged := MergeEnv(lines, updates)
		return SerializeEnv(merged), nil
	}))
}

func (m envModel) View() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render("Environment configuration"))
	b.WriteString("\n\n")
	if !m.ready {
		b.WriteString("Loading...\n")
		return b.String()
	}
	for _, input := range m.inputs {
		b.WriteString(input.View())
		b.WriteString("\n")
	}
	b.WriteString("\n")
	b.WriteString(dimStyle.Render("tab/shift+tab navigate, enter continue"))
	return b.String()
}
