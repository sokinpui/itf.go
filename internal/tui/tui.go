package tui

import (
	"fmt"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/sokinpui/itf/internal/app"
	"github.com/sokinpui/itf/internal/model"
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

// --- Messages ---
type summaryMsg struct {
	model.Summary
}

type errorMsg struct{ err error }

func (e errorMsg) Error() string { return e.err.Error() }

// --- Model ---
type Model struct {
	app     *app.App
	summary model.Summary
	err     error
	done    bool
}

func New(app *app.App) Model {
	return Model{
		app: app,
	}
}

func (m Model) Init() tea.Cmd {
	return m.runApp
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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
	}
	return m, nil
}

func (m Model) View() string {
	if !m.done {
		return ""
	}

	if m.err != nil {
		return errorStyle.Render("Error: ", m.err.Error())
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
