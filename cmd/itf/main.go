package main

import (
	"fmt"
	"os"

	application "itf/internal/app"
	"itf/internal/cli"
	"itf/internal/ui"
)

func main() {
	cfg, err := cli.ParseFlags()
	if err != nil {
		// pflag already prints the error message.
		os.Exit(1)
	}

	app, err := application.New(cfg)
	if err != nil {
		ui.Error("Failed to initialize application: %v", err)
		os.Exit(1)
	}

	if err := app.Run(); err != nil {
		// The app.Run() method is expected to print its own errors.
		// This is a final catch-all.
		ui.Error("An unexpected error occurred: %v", err)
		// Add a more detailed error for debugging if needed.
		if e, ok := err.(*application.DetailedError); ok {
			fmt.Fprintf(os.Stderr, "\n--- Stack Trace ---\n%s\n", e.Stack)
		}
		os.Exit(1)
	}
}
