package main

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
)

type applyModel struct {
	ctx      *Context
	applied  bool
	ready    bool
}

func newApplyModel(ctx *Context) applyModel {
	return applyModel{ctx: ctx}
}

func (m applyModel) Init() tea.Cmd {
	return func() tea.Msg {
		return applyReadyMsg{}
	}
}

type applyReadyMsg struct{}
type applyDoneMsg struct{ results []ApplyResult }

func (m applyModel) Update(msg tea.Msg) (applyModel, tea.Cmd) {
	switch msg := msg.(type) {
	case applyReadyMsg:
		m.ready = true
	case applyDoneMsg:
		m.ctx.Results = msg.results
		m.applied = true
		return m, nextStep
	case tea.KeyMsg:
		if !m.ready || m.applied {
			return m, nil
		}
		switch {
		case key.Matches(msg, keys.Next):
			if len(m.ctx.Plan.Steps) == 0 {
				return m, nextStep
			}
			return m, func() tea.Msg {
				results := m.ctx.Plan.ApplyAll()
				return applyDoneMsg{results: results}
			}
		}
	}
	return m, nil
}

func (m applyModel) View() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render("Review & Apply"))
	b.WriteString("\n\n")
	if !m.ready {
		b.WriteString("Loading...\n")
		return b.String()
	}
	if len(m.ctx.Plan.Steps) == 0 {
		b.WriteString("No changes to apply.\n\n")
		b.WriteString(dimStyle.Render("Press enter to continue"))
		return b.String()
	}
	b.WriteString("The following changes will be made:\n\n")
	for i, step := range m.ctx.Plan.Steps {
		b.WriteString(fmt.Sprintf("  %d. %s\n", i+1, step.Describe()))
	}
	b.WriteString("\n")
	b.WriteString("Backups will be created as .goob-bak files.\n\n")
	b.WriteString(dimStyle.Render("Press enter to apply, q to quit"))
	return b.String()
}
