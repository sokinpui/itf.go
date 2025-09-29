package main

import (
	"fmt"
	"os"

	"github.com/sokinpui/itf.go/cli"
	"github.com/sokinpui/itf.go/internal/tui"
	"github.com/sokinpui/itf.go/itf"

	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	cfg, err := cli.ParseFlags()
	if err != nil {
		// pflag already prints the error message.
		os.Exit(1)
	}

	app, err := itf.New(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize application: %v\n", err)
		os.Exit(1)
	}

	// Flags that print to stdout and should not run the TUI.
	if cfg.OutputDiffFix || cfg.OutputTool {
		if _, err := app.Execute(); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
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
