package main

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
)

type doctorModel struct {
	ctx   *Context
	ready bool
}

func newDoctorModel(ctx *Context) doctorModel {
	return doctorModel{ctx: ctx}
}

func (m doctorModel) Init() tea.Cmd {
	return func() tea.Msg {
		return depsCheckedMsg{deps: CheckDeps()}
	}
}

type depsCheckedMsg struct{ deps []Dep }

func (m doctorModel) Update(msg tea.Msg) (doctorModel, tea.Cmd) {
	switch msg := msg.(type) {
	case depsCheckedMsg:
		m.ctx.Deps = msg.deps
		m.ready = true
	case tea.KeyMsg:
		if m.ready && key.Matches(msg, keys.Next) {
			return m, nextStep
		}
	}
	return m, nil
}

func (m doctorModel) View() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render("Checking dependencies..."))
	b.WriteString("\n\n")
	if !m.ready {
		b.WriteString("Scanning...\n")
		return b.String()
	}
	allOk := true
	for _, d := range m.ctx.Deps {
		icon := okStyle.Render("✓")
		status := okStyle.Render(d.Version)
		if !d.Found {
			if d.Required {
				icon = errStyle.Render("✗")
				status = errStyle.Render("not found")
				allOk = false
			} else {
				icon = warnStyle.Render("○")
				status = warnStyle.Render("not found (optional)")
			}
		} else if !d.Ok {
			icon = errStyle.Render("✗")
			status = errStyle.Render(fmt.Sprintf("%s (need %s+)", d.Version, MinVersions[d.Name]))
			allOk = false
		}
		b.WriteString(fmt.Sprintf("  %s %s %s\n", icon, d.Name, status))
		if d.Hint != "" && (!d.Found || !d.Ok) {
			b.WriteString(fmt.Sprintf("    %s\n", dimStyle.Render(d.Hint)))
		}
	}
	b.WriteString("\n")
	if allOk {
		b.WriteString(okStyle.Render("All required dependencies found."))
	} else {
		b.WriteString(warnStyle.Render("Some dependencies missing. Install them and re-run."))
	}
	b.WriteString("\n\n")
	b.WriteString(dimStyle.Render("Press enter to continue"))
	return b.String()
}
