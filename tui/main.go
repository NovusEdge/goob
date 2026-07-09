package main

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/NimbleMarkets/ntcharts/sparkline"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type tickMsg time.Time
type statsMsg struct {
	stats Stats
	err   error
}

type model struct {
	pet    *Service
	daemon *Service

	petSpark    sparkline.Model
	daemonSpark sparkline.Model
	logs        viewport.Model

	stats    Stats
	statsErr error
	w, h     int
}

func tickCmd() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg { return tickMsg(t) })
}

func fetchCmd() tea.Cmd {
	return func() tea.Msg {
		s, err := fetchStats()
		return statsMsg{stats: s, err: err}
	}
}

func initialModel() model {
	root := projectRoot()
	vp := viewport.New(0, 0)
	return model{
		pet:         newService("pet", "run", root),
		daemon:      newService("daemon", "daemon", root),
		petSpark:    sparkline.New(24, 2),
		daemonSpark: sparkline.New(24, 2),
		logs:        vp,
		statsErr:    errors.New("connecting…"),
	}
}

func (m model) Init() tea.Cmd {
	return tea.Batch(tickCmd(), fetchCmd())
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			m.pet.stop()
			m.daemon.stop()
			return m, tea.Quit
		case "r":
			m.pet.start()
			return m, nil
		case "s":
			m.pet.stop()
			return m, nil
		case "d":
			m.daemon.start()
			return m, nil
		case "x":
			m.daemon.stop()
			return m, nil
		}
	case tea.WindowSizeMsg:
		m.w, m.h = msg.Width, msg.Height
		m.logs.Width = max(20, msg.Width-4)   // full width minus log panel border+padding
		m.logs.Height = max(3, msg.Height-12) // rows left under the top row + footer
	case tickMsg:
		// sampleCPU walks /proc synchronously on the Update goroutine each tick;
		// acceptable because the pet/daemon process trees are tiny.
		m.petSpark.Push(m.pet.sampleCPU())
		m.petSpark.Draw()
		m.daemonSpark.Push(m.daemon.sampleCPU())
		m.daemonSpark.Draw()
		m.logs.SetContent(m.daemon.logText())
		m.logs.GotoBottom()
		return m, tea.Batch(tickCmd(), fetchCmd())
	case statsMsg:
		m.stats, m.statsErr = msg.stats, msg.err
	}
	var cmd tea.Cmd
	m.logs, cmd = m.logs.Update(msg)
	return m, cmd
}

var (
	titleStyle = lipgloss.NewStyle().Bold(true)
	upStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("42"))
	downStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	panelStyle = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).Padding(0, 1)
)

func statusLine(s *Service, runKey, stopKey string) string {
	dot, state := downStyle.Render("○"), "stopped"
	if s.running() {
		dot, state = upStyle.Render("●"), fmt.Sprintf("pid %d", s.pid())
	}
	return fmt.Sprintf("%-7s %s %-12s [%s]run [%s]stop", s.name, dot, state, runKey, stopKey)
}

func (m model) View() string {
	if m.w == 0 {
		return "starting…"
	}
	// Three panels share the top row across the full width. Each bordered+padded
	// panel adds 4 cols (border 2 + padding 2), so content widths sum to w-12.
	// cpu just needs to hold the 24-wide sparkline; status/info take the rest.
	statusW, cpuW := 42, 28
	if statusW+cpuW+16 > m.w-12 { // narrow terminal: shrink to fit
		statusW = max(24, (m.w-12)*4/10)
		cpuW = 28
	}
	infoW := m.w - 12 - statusW - cpuW
	if infoW < 16 {
		infoW = 16
	}

	status := panelStyle.Width(statusW).Render(strings.Join([]string{
		titleStyle.Render("goob control"),
		statusLine(m.pet, "r", "s"),
		statusLine(m.daemon, "d", "x"),
	}, "\n"))

	cpu := panelStyle.Width(cpuW).Render(strings.Join([]string{
		titleStyle.Render("cpu"),
		fmt.Sprintf("pet  %5.1f%%", m.pet.cpu),
		m.petSpark.View(),
		fmt.Sprintf("daem %5.1f%%", m.daemon.cpu),
		m.daemonSpark.View(),
	}, "\n"))

	daemonInfo := downStyle.Render("unreachable")
	if m.statsErr == nil {
		daemonInfo = strings.Join([]string{
			fmt.Sprintf("model %s", m.stats.Model),
			fmt.Sprintf("ticks %d", m.stats.Ticks),
			fmt.Sprintf("spend $%.4f", m.stats.Spend),
			fmt.Sprintf("last  %.0fms", m.stats.Latency),
		}, "\n")
	}
	info := panelStyle.Width(infoW).Render(titleStyle.Render("daemon") + "\n" + daemonInfo)

	top := lipgloss.JoinHorizontal(lipgloss.Top, status, cpu, info)
	logs := panelStyle.Width(m.w - 4).Render(titleStyle.Render("daemon logs") + "\n" + m.logs.View())

	return lipgloss.JoinVertical(lipgloss.Left, top, logs,
		downStyle.Render("[r/s] pet  [d/x] daemon  [q] quit"))
}

func main() {
	m := initialModel()
	final, err := tea.NewProgram(m, tea.WithAltScreen()).Run()
	if err != nil {
		fmt.Fprintln(os.Stderr, "tui error:", err)
		os.Exit(1)
	}
	fm := final.(model)
	fm.pet.stop()
	fm.daemon.stop()
	deadline := time.Now().Add(5 * time.Second)
	for (fm.pet.running() || fm.daemon.running()) && time.Now().Before(deadline) {
		time.Sleep(50 * time.Millisecond)
	}
}
