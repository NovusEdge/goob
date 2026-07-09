package main

import (
	"errors"
	"fmt"
	"math/rand"
	"os"
	"strings"
	"time"

	"github.com/NimbleMarkets/ntcharts/canvas/runes"
	"github.com/NimbleMarkets/ntcharts/linechart/streamlinechart"
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
	cute        streamlinechart.Model // joke "cuteness" meter — random flux
	cuteVal     float64
	logs        viewport.Model

	stats    Stats
	statsErr error
	w, h     int
}

// newCute builds the cuteness stream chart at a given size (rebuilt on resize).
func newCute(w, h int) streamlinechart.Model {
	c := streamlinechart.New(w, h,
		streamlinechart.WithYRange(0, 100),
		streamlinechart.WithStyles(runes.ArcLineStyle,
			lipgloss.NewStyle().Foreground(cAccent)))
	c.SetViewYRange(0, 100)
	return c
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
		petSpark:    sparkline.New(24, 2, sparkline.WithStyle(lipgloss.NewStyle().Foreground(cCyan))),
		daemonSpark: sparkline.New(24, 2, sparkline.WithStyle(lipgloss.NewStyle().Foreground(cGreen))),
		cute:        newCute(24, 4),
		cuteVal:     72, // starts adorable
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
		leftW, rightW := layout(msg.Width - 4)
		m.logs.Width = rightW                 // right column inner width
		m.logs.Height = max(3, msg.Height-14) // fills under banner + daemon panel + footer
		// Cuteness chart fills the sidebar space below control+cpu (~15 rows of
		// chrome above it), aligning the sidebar bottom with the log pane.
		m.cute = newCute(leftW-2, max(3, msg.Height-20))
	case tickMsg:
		// sampleCPU walks /proc synchronously on the Update goroutine each tick;
		// acceptable because the pet/daemon process trees are tiny.
		m.petSpark.Push(m.pet.sampleCPU())
		m.petSpark.Draw()
		m.daemonSpark.Push(m.daemon.sampleCPU())
		m.daemonSpark.Draw()
		// Cuteness: a bounded random walk — flails but stays adorable.
		m.cuteVal = clampF(m.cuteVal+(rand.Float64()-0.5)*20, 5, 100)
		m.cute.Push(m.cuteVal)
		m.cute.Draw()
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
	cAccent = lipgloss.Color("212") // pink — banner + primary accent
	cGreen  = lipgloss.Color("42")
	cCyan   = lipgloss.Color("45")
	cYellow = lipgloss.Color("214")
	cText   = lipgloss.Color("252")
	cLabel  = lipgloss.Color("245")
	cDim    = lipgloss.Color("240")

	appStyle    = lipgloss.NewStyle().Margin(1, 2, 0, 2) // top breathing room + side margins
	bannerStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("231")).Background(cAccent).Padding(0, 2)
	titleStyle  = lipgloss.NewStyle().Bold(true).Foreground(cCyan)
	labelStyle  = lipgloss.NewStyle().Foreground(cLabel)
	valStyle    = lipgloss.NewStyle().Foreground(cText)
	keyStyle    = lipgloss.NewStyle().Foreground(cYellow).Bold(true)
	upStyle     = lipgloss.NewStyle().Foreground(cGreen).Bold(true)
	downStyle   = lipgloss.NewStyle().Foreground(cDim)
	spendStyle  = lipgloss.NewStyle().Foreground(cGreen).Bold(true)
	cuteStyle   = lipgloss.NewStyle().Foreground(cAccent).Bold(true)
	footStyle   = lipgloss.NewStyle().Foreground(cDim).MarginTop(1)
)

func panel(w int, border lipgloss.Color) lipgloss.Style {
	return lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).
		BorderForeground(border).Padding(0, 1).Width(w)
}

func keys(runKey, stopKey string) string {
	return keyStyle.Render("["+runKey+"]") + labelStyle.Render("run ") +
		keyStyle.Render("["+stopKey+"]") + labelStyle.Render("stop")
}

func statusLine(s *Service, runKey, stopKey string) string {
	name := valStyle.Bold(true).Width(8).Render(s.name)
	state := downStyle.Width(12).Render("○ stopped")
	if s.running() {
		state = upStyle.Width(12).Render(fmt.Sprintf("● pid %d", s.pid()))
	}
	return name + state + keys(runKey, stopKey)
}

func short(s string, n int) string {
	if len(s) > n && n > 1 {
		return s[:n-1] + "…"
	}
	return s
}

func clampF(v, lo, hi float64) float64 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

// layout splits the usable width w into a fixed-ish left sidebar and a flexible
// right column. Two panels each render at contentWidth+2 (border), plus a 1-col
// gap between columns → leftW + rightW + 5 == w. Both View and the resize
// handler call this so the viewport width matches what View draws.
func layout(w int) (leftW, rightW int) {
	leftW = 38
	if leftW+34 > w { // narrow terminal: give the sidebar less
		leftW = max(24, w*2/5)
	}
	rightW = w - 5 - leftW
	if rightW < 24 {
		rightW = 24
	}
	return leftW, rightW
}

func (m model) View() string {
	if m.w == 0 {
		return "\n  starting…"
	}
	w := m.w - 4 // usable width inside appStyle's side margins
	leftW, rightW := layout(w)
	row := func(l, v string) string { return labelStyle.Width(6).Render(l) + v }

	banner := bannerStyle.Width(w).Render("🐱  goob control panel")

	// Left column: control + cpu, stacked.
	control := panel(leftW, cAccent).Render(strings.Join([]string{
		titleStyle.Render("control"),
		statusLine(m.pet, "r", "s"),
		statusLine(m.daemon, "d", "x"),
	}, "\n"))
	cpu := panel(leftW, cCyan).Render(strings.Join([]string{
		titleStyle.Render("cpu"),
		row("pet", valStyle.Render(fmt.Sprintf("%5.1f%%", m.pet.cpu))),
		m.petSpark.View(),
		row("daem", valStyle.Render(fmt.Sprintf("%5.1f%%", m.daemon.cpu))),
		m.daemonSpark.View(),
	}, "\n"))
	cuteTitle := titleStyle.Render("cuteness ") +
		cuteStyle.Render("♡ ") + valStyle.Render(fmt.Sprintf("%.0f%%", m.cuteVal))
	cutePanel := panel(leftW, cAccent).Render(cuteTitle + "\n" + m.cute.View())
	left := lipgloss.NewStyle().MarginRight(1).Render(
		lipgloss.JoinVertical(lipgloss.Left, control, cpu, cutePanel))

	// Right column: daemon stats above the logs.
	var infoBody string
	if m.statsErr != nil {
		infoBody = downStyle.Render("○ unreachable")
	} else {
		infoBody = strings.Join([]string{
			row("model", valStyle.Render(short(m.stats.Model, rightW-9))),
			row("ticks", valStyle.Render(fmt.Sprintf("%d", m.stats.Ticks))),
			row("spend", spendStyle.Render(fmt.Sprintf("$%.4f", m.stats.Spend))),
			row("last", valStyle.Render(fmt.Sprintf("%.0f ms", m.stats.Latency))),
		}, "\n")
	}
	info := panel(rightW, cYellow).Render(titleStyle.Render("daemon") + "\n" + infoBody)
	logs := panel(rightW, cDim).Render(titleStyle.Render("daemon logs") + "\n" + m.logs.View())
	right := lipgloss.JoinVertical(lipgloss.Left, info, logs)

	body := lipgloss.JoinHorizontal(lipgloss.Top, left, right)

	foot := footStyle.Render(keys("r", "s") + labelStyle.Render(" pet   ") +
		keys("d", "x") + labelStyle.Render(" daemon   ") +
		keyStyle.Render("[q]") + labelStyle.Render("quit"))

	return appStyle.Render(lipgloss.JoinVertical(lipgloss.Left, banner, body, foot))
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
