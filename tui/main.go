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
	w        int
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
		petSpark:    sparkline.New(28, 3),
		daemonSpark: sparkline.New(28, 3),
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
		m.w = msg.Width
		m.logs.Width = msg.Width - 2
		m.logs.Height = max(3, msg.Height-13)
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
		dot, state = upStyle.Render("●"), fmt.Sprintf("running  pid %d", s.pid())
	}
	return fmt.Sprintf("%-7s %s %-20s  [%s] run  [%s] stop",
		s.name, dot, state, runKey, stopKey)
}

func (m model) View() string {
	header := panelStyle.Width(58).Render(strings.Join([]string{
		titleStyle.Render("goob control"),
		statusLine(m.pet, "r", "s"),
		statusLine(m.daemon, "d", "x"),
	}, "\n"))

	cpu := panelStyle.Width(34).Render(strings.Join([]string{
		titleStyle.Render("cpu"),
		fmt.Sprintf("pet  %5.1f%%", m.pet.cpu),
		m.petSpark.View(),
		fmt.Sprintf("daem %5.1f%%", m.daemon.cpu),
		m.daemonSpark.View(),
	}, "\n"))

	daemonInfo := "daemon unreachable"
	if m.statsErr == nil {
		daemonInfo = strings.Join([]string{
			fmt.Sprintf("model  %s", m.stats.Model),
			fmt.Sprintf("ticks  %d", m.stats.Ticks),
			fmt.Sprintf("spend  $%.4f", m.stats.Spend),
			fmt.Sprintf("last   %.0fms", m.stats.Latency),
		}, "\n")
	}
	info := panelStyle.Width(22).Render(titleStyle.Render("daemon") + "\n" + daemonInfo)

	mid := lipgloss.JoinHorizontal(lipgloss.Top, cpu, info)
	logs := panelStyle.Width(58).Render(titleStyle.Render("daemon logs") + "\n" + m.logs.View())

	return lipgloss.JoinVertical(lipgloss.Left, header, mid, logs,
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
