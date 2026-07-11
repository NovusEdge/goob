package main

import (
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Steps in the wizard.
const (
	StepWelcome = iota
	StepDoctor
	StepAgents
	StepEnv
	StepApply
	StepDone
)

// Context holds shared state across steps.
type Context struct {
	Deps        []Dep
	SelectedIDs map[string]bool // agent IDs to install
	EnvUpdates  map[string]string
	Plan        *Plan
	Results     []ApplyResult
}

// Model is the root wizard model.
type Model struct {
	step    int
	ctx     *Context
	welcome welcomeModel
	doctor  doctorModel
	agents  agentsModel
	env     envModel
	apply   applyModel
	done    doneModel
	width   int
	height  int
}

func NewModel() Model {
	ctx := &Context{
		SelectedIDs: make(map[string]bool),
		EnvUpdates:  make(map[string]string),
		Plan:        &Plan{},
	}
	return Model{
		step:    StepWelcome,
		ctx:     ctx,
		welcome: newWelcomeModel(),
		doctor:  newDoctorModel(ctx),
		agents:  newAgentsModel(ctx),
		env:     newEnvModel(ctx),
		apply:   newApplyModel(ctx),
		done:    newDoneModel(ctx),
	}
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, keys.Quit):
			return m, tea.Quit
		}
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	case nextStepMsg:
		m.step++
		return m, m.initCurrentStep()
	}
	var cmd tea.Cmd
	switch m.step {
	case StepWelcome:
		m.welcome, cmd = m.welcome.Update(msg)
	case StepDoctor:
		m.doctor, cmd = m.doctor.Update(msg)
	case StepAgents:
		m.agents, cmd = m.agents.Update(msg)
	case StepEnv:
		m.env, cmd = m.env.Update(msg)
	case StepApply:
		m.apply, cmd = m.apply.Update(msg)
	case StepDone:
		m.done, cmd = m.done.Update(msg)
	}
	return m, cmd
}

func (m Model) View() string {
	var content string
	switch m.step {
	case StepWelcome:
		content = m.welcome.View()
	case StepDoctor:
		content = m.doctor.View()
	case StepAgents:
		content = m.agents.View()
	case StepEnv:
		content = m.env.View()
	case StepApply:
		content = m.apply.View()
	case StepDone:
		content = m.done.View()
	}
	return lipgloss.NewStyle().Padding(1, 2).Render(content)
}

func (m Model) initCurrentStep() tea.Cmd {
	switch m.step {
	case StepDoctor:
		return m.doctor.Init()
	case StepAgents:
		return m.agents.Init()
	case StepEnv:
		return m.env.Init()
	case StepApply:
		return m.apply.Init()
	}
	return nil
}

type nextStepMsg struct{}

func nextStep() tea.Msg { return nextStepMsg{} }

// Key bindings.
type keyMap struct {
	Next   key.Binding
	Back   key.Binding
	Select key.Binding
	Quit   key.Binding
}

var keys = keyMap{
	Next:   key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "continue")),
	Back:   key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "back")),
	Select: key.NewBinding(key.WithKeys(" "), key.WithHelp("space", "select")),
	Quit:   key.NewBinding(key.WithKeys("q", "ctrl+c"), key.WithHelp("q", "quit")),
}

// Styles.
var (
	titleStyle   = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("205"))
	okStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("42"))
	warnStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("214"))
	errStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
	dimStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	selectedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("86"))
)
