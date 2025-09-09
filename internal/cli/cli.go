package cli

import (
	"fmt"

	"github.com/spf13/pflag"
)

// Config holds all the command-line flag values.
type Config struct {
	Buffer         bool
	OutputDiffFix  bool
	Revert         bool
	Redo           bool
	LookupDirs     []string
	Extensions     []string
}

// ParseFlags defines and parses command-line flags using pflag.
func ParseFlags() (*Config, error) {
	cfg := &Config{}

	// Define flags
	pflag.BoolVarP(&cfg.Buffer, "buffer", "b", false, "Update buffers in Neovim without saving them to disk (changes are saved by default).")
	pflag.BoolVarP(&cfg.OutputDiffFix, "output-diff-fix", "o", false, "Print the diff that corrected start and count.")
	pflag.StringSliceVarP(&cfg.LookupDirs, "lookup-dir", "l", []string{}, "Change directory to look for files (default: current directory).")
	pflag.StringSliceVarP(&cfg.Extensions, "extension", "e", []string{}, "Filter by extension. Use 'diff' to process only diff blocks (e.g., 'py', 'js', 'diff').")

	// Mutually exclusive history group
	pflag.BoolVarP(&cfg.Revert, "revert", "r", false, "Revert the last operation.")
	pflag.BoolVarP(&cfg.Redo, "redo", "R", false, "Redo the last reverted operation.")

	pflag.Usage = func() {
		fmt.Println("Usage: itf [flags]")
		fmt.Println("\nParse content from stdin (pipe) or clipboard to update files in Neovim.")
		fmt.Println("\nExample: pbpaste | itf -e py")
		fmt.Println("\nFlags:")
		pflag.PrintDefaults()
	}

	pflag.Parse()

	// Validate mutually exclusive flags
	if cfg.Revert && cfg.Redo {
		return nil, fmt.Errorf("error: --revert and --redo are mutually exclusive")
	}

	// Normalize extensions
	for i, ext := range cfg.Extensions {
		if len(ext) > 0 && ext[0] != '.' {
			cfg.Extensions[i] = "." + ext
		}
	}

	return cfg, nil
}
