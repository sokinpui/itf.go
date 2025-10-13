package tui

import (
	"fmt"
	"os"
	"time"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/sokinpui/itf.go/itf"
	"github.com/sokinpui/itf.go/model"
)

// --- Styles ---
var (
	headerStyle  = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("63")) // Mauve
	createdStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("81"))            // Cyan
	successStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("78"))            // Green
	errorStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("197"))           // Red
	pathStyle    = lipgloss.NewStyle()
	faintStyle   = lipgloss.NewStyle().Faint(true)
)

// --- Spinner ---
type spinner struct {
	frames []string
	index  int
}

func newSpinner() spinner {
	return spinner{
		frames: []string{"⣾", "⣽", "⣻", "⢿", "⡿", "⣟", "⣯", "⣷"},
	}
}

func (s *spinner) tick() {
	s.index = (s.index + 1) % len(s.frames)
}

func (s spinner) View() string {
	return s.frames[s.index]
}

// --- Messages ---
type summaryMsg struct {
	model.Summary
}

type errorMsg struct{ err error }

func (e errorMsg) Error() string { return e.err.Error() }

type tickMsg time.Time

type progressMsg struct {
	current int
	total   int
}

// --- Model ---
type Model struct {
	app             *itf.App
	summary         model.Summary
	err             error
	done            bool
	spinner         spinner
	progressCurrent int
	progressTotal   int
	program         *tea.Program
	noAnimation     bool
}

func New(app *itf.App, noAnimation bool) *Model {
	return &Model{
		app:         app,
		spinner:     newSpinner(),
		noAnimation: noAnimation,
	}
}

func (m *Model) SetProgram(p *tea.Program) {
	m.program = p
}

func (m *Model) Init() tea.Cmd {
	if m.noAnimation {
		return m.runApp
	}
	return tea.Batch(m.runApp, m.spinnerTick())
}

func (m *Model) spinnerTick() tea.Cmd {
	return tea.Tick(100*time.Millisecond, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		}

	case summaryMsg:
		m.summary = msg.Summary
		m.done = true
		return m, tea.Quit

	case errorMsg:
		m.err = msg
		m.done = true
		return m, tea.Quit

	case tickMsg:
		if !m.done && !m.noAnimation {
			m.spinner.tick()
			return m, m.spinnerTick()
		}
		return m, nil

	case progressMsg:
		m.progressCurrent = msg.current
		m.progressTotal = msg.total
		return m, nil
	}
	return m, nil
}

func (m *Model) View() string {
	if m.err != nil {
		return errorStyle.Render("Error: ", m.err.Error())
	}

	if !m.done {
		if m.noAnimation {
			return ""
		}

		var b strings.Builder
		b.WriteString(fmt.Sprintf("%s Processing files...\n", m.spinner.View()))
		if m.progressTotal > 0 {
			b.WriteString(fmt.Sprintf("  %d / %d", m.progressCurrent, m.progressTotal))
		} else {
			b.WriteString("  Initializing...")
		}
		return b.String()
	}

	return m.renderSummary()
}

func (m *Model) renderSummary() string {
	var b strings.Builder

	if m.summary.Message != "" {
		b.WriteString(headerStyle.Render(m.summary.Message))
		b.WriteString("\n\n")
	}

	hasContent := false
	if len(m.summary.Created) > 0 {
		hasContent = true
		b.WriteString(createdStyle.Render("Created:"))
		b.WriteString("\n")
		for _, f := range m.summary.Created {
			b.WriteString(fmt.Sprintf("  %s\n", pathStyle.Render(f)))
		}
	}
	if len(m.summary.Modified) > 0 {
		hasContent = true
		b.WriteString(successStyle.Render("Modified:"))
		b.WriteString("\n")
		for _, f := range m.summary.Modified {
			b.WriteString(fmt.Sprintf("  %s\n", pathStyle.Render(f)))
		}
	}
	if len(m.summary.Failed) > 0 {
		hasContent = true
		b.WriteString(errorStyle.Render("Failed:"))
		b.WriteString("\n")
		for _, f := range m.summary.Failed {
			b.WriteString(fmt.Sprintf("  %s\n", pathStyle.Render(f)))
		}
	}

	if !hasContent && m.summary.Message == "" {
		b.WriteString(faintStyle.Render("Nothing to do."))
	}

	return b.String()
}

func (m *Model) runApp() tea.Msg {
	if m.program != nil && !m.noAnimation {
		m.app.SetProgressCallback(func(current, total int) {
			m.program.Send(progressMsg{current: current, total: total})
		})
	}

	summary, err := m.app.Execute()
	if err != nil {
		// Check for detailed error to print stack
		if e, ok := err.(*itf.DetailedError); ok {
			// The TUI will exit, so we can print to stderr here for the stack trace.
			fmt.Fprintf(os.Stderr, "\n--- Stack Trace ---\n%s\n", e.Stack)
		}
		return errorMsg{err}
	}
	return summaryMsg{
		Summary: summary,
	}
}
