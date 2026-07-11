package main

import (
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
)

type welcomeModel struct{}

func newWelcomeModel() welcomeModel {
	return welcomeModel{}
}

func (m welcomeModel) Update(msg tea.Msg) (welcomeModel, tea.Cmd) {
	if msg, ok := msg.(tea.KeyMsg); ok && key.Matches(msg, keys.Next) {
		return m, nextStep
	}
	return m, nil
}

func (m welcomeModel) View() string {
	return titleStyle.Render("goob setup wizard") + "\n\n" +
		"This wizard will:\n" +
		"  1. Check dependencies (godot, python3, uv, go)\n" +
		"  2. Register goob hooks with your AI coding agents\n" +
		"  3. Set up your .env file\n\n" +
		dimStyle.Render("Press enter to continue, q to quit")
}
