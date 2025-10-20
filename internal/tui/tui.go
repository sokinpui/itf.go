package tui

import (
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/sokinpui/itf.go/itf"
	"github.com/sokinpui/itf.go/model"
)

// --- Styles ---
var (
	headerStyle  = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("63")) // Mauve
	createdStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("81"))            // Cyan
	renamedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("212"))           // Purple
	successStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("78"))            // Green
	deletedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("204"))           // Pink
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

// TUI handles the terminal user interface.
type TUI struct {
	app             *itf.App
	noAnimation     bool
	spinner         spinner
	mu              sync.Mutex
	progressCurrent int
	progressTotal   int
}

// New creates a new TUI.
func New(app *itf.App, noAnimation bool) *TUI {
	return &TUI{
		app:         app,
		noAnimation: noAnimation,
		spinner:     newSpinner(),
	}
}

// Run starts the TUI, executes the application logic, and displays the results.
func (t *TUI) Run() error {
	if t.noAnimation {
		summary, err := t.app.Execute()
		if err != nil {
			if e, ok := err.(*itf.DetailedError); ok {
				fmt.Fprintf(os.Stderr, "\n--- Stack Trace ---\n%s\n", e.Stack)
			}
			return err
		}
		fmt.Print(t.renderSummary(summary))
		return nil
	}

	t.app.SetProgressCallback(func(current, total int) {
		t.mu.Lock()
		defer t.mu.Unlock()
		t.progressCurrent = current
		t.progressTotal = total
	})

	done := make(chan struct{})
	go func() {
		for {
			select {
			case <-done:
				return
			case <-time.After(100 * time.Millisecond):
				t.spinner.tick()
				t.renderProgress()
			}
		}
	}()

	summary, err := t.app.Execute()
	close(done)

	fmt.Print("\r\x1b[K") // Clear the progress line

	if err != nil {
		if e, ok := err.(*itf.DetailedError); ok {
			fmt.Fprintf(os.Stderr, "\n--- Stack Trace ---\n%s\n", e.Stack)
		}
		return err
	}

	fmt.Print(t.renderSummary(summary))
	return nil
}

func (t *TUI) renderProgress() {
	t.mu.Lock()
	defer t.mu.Unlock()

	var b strings.Builder
	b.WriteString(fmt.Sprintf("%s Processing files... ", t.spinner.View()))
	if t.progressTotal > 0 {
		b.WriteString(fmt.Sprintf("%d / %d", t.progressCurrent, t.progressTotal))
	} else {
		b.WriteString("Initializing...")
	}
	fmt.Printf("\r%s\x1b[K", b.String())
}

func (t *TUI) renderSummary(summary model.Summary) string {
	var b strings.Builder

	if summary.Message != "" {
		b.WriteString(headerStyle.Render(summary.Message))
		b.WriteString("\n\n")
	}

	hasContent := false
	if len(summary.Created) > 0 {
		hasContent = true
		b.WriteString(createdStyle.Render("Created:"))
		b.WriteString("\n")
		for _, f := range summary.Created {
			b.WriteString(fmt.Sprintf("  %s\n", pathStyle.Render(f)))
		}
	}
	if len(summary.Modified) > 0 {
		hasContent = true
		b.WriteString(successStyle.Render("Modified:"))
		b.WriteString("\n")
		for _, f := range summary.Modified {
			b.WriteString(fmt.Sprintf("  %s\n", pathStyle.Render(f)))
		}
	}
	if len(summary.Renamed) > 0 {
		hasContent = true
		b.WriteString(renamedStyle.Render("Renamed:"))
		b.WriteString("\n")
		for _, f := range summary.Renamed {
			b.WriteString(fmt.Sprintf("  %s\n", pathStyle.Render(f)))
		}
	}
	if len(summary.Deleted) > 0 {
		hasContent = true
		b.WriteString(deletedStyle.Render("Deleted:"))
		b.WriteString("\n")
		for _, f := range summary.Deleted {
			b.WriteString(fmt.Sprintf("  %s\n", pathStyle.Render(f)))
		}
	}

	if len(summary.Failed) > 0 {
		hasContent = true
		b.WriteString(errorStyle.Render("Failed:"))
		b.WriteString("\n")
		for _, f := range summary.Failed {
			b.WriteString(fmt.Sprintf("  %s\n", pathStyle.Render(f)))
		}
	}

	if !hasContent && summary.Message == "" {
		b.WriteString(faintStyle.Render("Nothing to do."))
	}

	return b.String()
}
