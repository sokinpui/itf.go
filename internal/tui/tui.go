package tui

import (
	"fmt"
	"itf/internal/app"
	"itf/internal/model"
	"os"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// --- Styles ---
var (
	headerStyle  = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("63")) // Mauve
	successStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("78"))  // Green
	errorStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("197"))  // Red
	pathStyle    = lipgloss.NewStyle()
	faintStyle   = lipgloss.NewStyle().Faint(true)
)

// --- Messages ---
type summaryMsg struct {
	model.Summary
}

type errorMsg struct{ err error }

func (e errorMsg) Error() string { return e.err.Error() }

// --- Model ---
type Model struct {
	app     *app.App
	spinner spinner.Model
	state   state
	summary summaryMsg
	err     error
}

type state int

const (
	stateProcessing state = iota
	stateSummary
	stateError
)

func New(app *app.App) Model {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))
	return Model{
		app:     app,
		spinner: s,
		state:   stateProcessing,
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(m.spinner.Tick, m.runApp)
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		}

	case summaryMsg:
		m.state = stateSummary
		m.summary = msg
		return m, tea.Quit

	case errorMsg:
		m.state = stateError
		m.err = msg
		return m, tea.Quit

	default:
		var cmd tea.Cmd
		if m.state == stateProcessing {
			m.spinner, cmd = m.spinner.Update(msg)
		}
		return m, cmd
	}
	return m, nil
}

func (m Model) View() string {
	switch m.state {
	case stateProcessing:
		return fmt.Sprintf("%s Processing...", m.spinner.View())
	case stateError:
		return errorStyle.Render("Error: ", m.err.Error())
	case stateSummary:
		return m.renderSummary()
	default:
		return ""
	}
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
		b.WriteString(successStyle.Render("Created:"))
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
	summary, err := m.app.Execute()
	if err != nil {
		// Check for detailed error to print stack
		if e, ok := err.(*app.DetailedError); ok {
			// The TUI will exit, so we can print to stderr here for the stack trace.
			fmt.Fprintf(os.Stderr, "\n--- Stack Trace ---\n%s\n", e.Stack)
		}
		return errorMsg{err}
	}
	return summaryMsg{
		Summary: summary,
	}
}
