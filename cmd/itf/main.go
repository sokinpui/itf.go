package main

import (
	"fmt"
	"os"

	application "github.com/sokinpui/itf.go/internal/app"
	"github.com/sokinpui/itf.go/internal/cli"
	"github.com/sokinpui/itf.go/internal/tui"

	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	cfg, err := cli.ParseFlags()
	if err != nil {
		// pflag already prints the error message.
		os.Exit(1)
	}

	app, err := application.New(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize application: %v\n", err)
		os.Exit(1)
	}

	// The --output-diff-fix flag prints to stdout and should not run the TUI.
	if cfg.OutputDiffFix {
		if _, err := app.Execute(); err != nil {
			fmt.Fprintf(os.Stderr, "Error fixing diffs: %v\n", err)
			os.Exit(1)
		}
		return
	}

	model := tui.New(app, cfg)
	p := tea.NewProgram(model)
	model.SetProgram(p)
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error running program: %v\n", err)
		os.Exit(1)
	}
}
