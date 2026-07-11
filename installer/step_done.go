package main

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
)

type doneModel struct {
	ctx *Context
}

func newDoneModel(ctx *Context) doneModel {
	return doneModel{ctx: ctx}
}

func (m doneModel) Update(msg tea.Msg) (doneModel, tea.Cmd) {
	if msg, ok := msg.(tea.KeyMsg); ok && key.Matches(msg, keys.Next, keys.Quit) {
		return m, tea.Quit
	}
	return m, nil
}

func (m doneModel) View() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render("Setup complete!"))
	b.WriteString("\n\n")
	// show results
	if len(m.ctx.Results) > 0 {
		for _, r := range m.ctx.Results {
			icon := okStyle.Render("✓")
			status := "applied"
			if r.Skipped {
				icon = dimStyle.Render("○")
				status = "unchanged"
			} else if r.Err != nil {
				icon = errStyle.Render("✗")
				status = fmt.Sprintf("error: %v", r.Err)
			}
			b.WriteString(fmt.Sprintf("  %s %s - %s\n", icon, r.Mutation.Describe(), status))
		}
		b.WriteString("\n")
	}
	b.WriteString("Next steps:\n")
	b.WriteString("  1. Run the pet with agent reactivity:\n")
	b.WriteString("     " + selectedStyle.Render("GOOB_HSM=1 just run") + "\n")
	b.WriteString("  2. Start a Claude Code session to test the integration\n")
	b.WriteString("\n")
	b.WriteString(dimStyle.Render("Press enter or q to exit"))
	return b.String()
}
