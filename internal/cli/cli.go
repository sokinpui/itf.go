package cli

import (
	"fmt"

	"github.com/spf13/pflag"
)

// Config holds all the command-line flag values.
type Config struct {
	Save           bool
	Clipboard      bool
	OutputDiffFix  bool
	Revert         bool
	Redo           bool
	File           bool
	Diff           bool
	Auto           bool
	LookupDirs     []string
	Extensions     []string
}

// ParseFlags defines and parses command-line flags using pflag.
func ParseFlags() (*Config, error) {
	cfg := &Config{}

	// Define flags
	pflag.BoolVarP(&cfg.Save, "save", "s", false, "Save all modified buffers in Neovim after the update.")
	pflag.BoolVarP(&cfg.Clipboard, "clipboard", "c", false, "Parse content from the clipboard instead of 'itf.txt'.")
	pflag.BoolVarP(&cfg.OutputDiffFix, "output-diff-fix", "o", false, "Print the diff that corrected start and count.")
	pflag.StringSliceVarP(&cfg.LookupDirs, "lookup-dir", "l", []string{}, "Change directory to look for files (default: current directory).")
	pflag.StringSliceVarP(&cfg.Extensions, "extension", "e", []string{}, "Filter to process only files with the specified extensions (e.g., 'py', 'js').")

	// Mutually exclusive history group
	pflag.BoolVarP(&cfg.Revert, "revert", "r", false, "Revert the last operation.")
	pflag.BoolVarP(&cfg.Redo, "redo", "R", false, "Redo the last reverted operation.")

	// Mutually exclusive mode group
	pflag.BoolVarP(&cfg.File, "file", "f", false, "Ignore diff blocks, parse content files blocks only.")
	pflag.BoolVarP(&cfg.Diff, "diff", "d", false, "Parse only diff blocks, ignore content file blocks.")
	pflag.BoolVarP(&cfg.Auto, "auto", "a", false, "Parse both diff blocks and content file blocks (default).")

	pflag.Usage = func() {
		fmt.Println("Usage: itf [flags]")
		fmt.Println("\nParse clipboard content or 'itf.txt' to update files and load them into Neovim.")
		fmt.Println("\nFlags:")
		pflag.PrintDefaults()
	}

	pflag.Parse()

	// Validate mutually exclusive flags
	if cfg.Revert && cfg.Redo {
		return nil, fmt.Errorf("error: --revert and --redo are mutually exclusive")
	}
	modeFlags := 0
	if cfg.File {
		modeFlags++
	}
	if cfg.Diff {
		modeFlags++
	}
	if cfg.Auto {
		modeFlags++
	}
	if modeFlags > 1 {
		return nil, fmt.Errorf("error: --file, --diff, and --auto are mutually exclusive")
	}

	// If no mode is specified, default to auto.
	if !cfg.File && !cfg.Diff {
		cfg.Auto = true
	}

	// Normalize extensions
	for i, ext := range cfg.Extensions {
		if ext[0] != '.' {
			cfg.Extensions[i] = "." + ext
		}
	}

	return cfg, nil
}
