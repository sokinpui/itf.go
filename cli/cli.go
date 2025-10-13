package cli

import (
	"fmt"
	"os"

	"github.com/sokinpui/itf.go/internal/tui"
	"github.com/sokinpui/itf.go/itf"
	"github.com/spf13/cobra"
)

// Config holds all the command-line flag values.
type Config struct {
	Buffer        bool
	OutputTool    bool
	OutputDiffFix bool
	Undo          bool
	Redo          bool
	NoAnimation   bool
	Extensions    []string
	Completion    string
}

var cfg = &Config{}

var rootCmd = &cobra.Command{
	Use:   "itf",
	Short: "Parse content from stdin or clipboard to update files.",
	Long: `Parse content from stdin (pipe) or clipboard to update files in Neovim.

Example: pbpaste | itf -e py`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if cfg.Completion != "" {
			switch cfg.Completion {
			case "bash":
				return cmd.Root().GenBashCompletion(os.Stdout)
			case "zsh":
				return cmd.Root().GenZshCompletion(os.Stdout)
			case "fish":
				return cmd.Root().GenFishCompletion(os.Stdout, true)
			case "powershell":
				return cmd.Root().GenPowerShellCompletionWithDesc(os.Stdout)
			default:
				return fmt.Errorf("unsupported shell for completion: %s", cfg.Completion)
			}
		}

		// Validate mutually exclusive flags
		if cfg.Undo && cfg.Redo {
			return fmt.Errorf("error: --undo and --redo are mutually exclusive")
		}

		// Normalize extensions
		for i, ext := range cfg.Extensions {
			if len(ext) > 0 && ext[0] != '.' {
				cfg.Extensions[i] = "." + ext
			}
		}

		itfCfg := &itf.Config{
			Buffer:        cfg.Buffer,
			OutputTool:    cfg.OutputTool,
			OutputDiffFix: cfg.OutputDiffFix,
			Undo:          cfg.Undo,
			Redo:          cfg.Redo,
			Extensions:    cfg.Extensions,
		}
		app, err := itf.New(itfCfg)
		if err != nil {
			return fmt.Errorf("failed to initialize application: %w", err)
		}

		// Flags that print to stdout and should not run the TUI.
		if cfg.OutputDiffFix || cfg.OutputTool {
			if _, err := app.Execute(); err != nil {
				return fmt.Errorf("error: %w", err)
			}
			return nil
		}

		ui := tui.New(app, cfg.NoAnimation)
		if err := ui.Run(); err != nil {
			return err
		}
		return nil
	},
}

func init() {
	rootCmd.Flags().StringVar(&cfg.Completion,
		"completion",
		"", "Generate completion script for your shell (bash|zsh|fish|powershell)")
	rootCmd.Flags().BoolVarP(&cfg.Buffer, "buffer", "b", false, "Update buffers in Neovim without saving them to disk (changes are saved by default).")
	rootCmd.Flags().BoolVarP(&cfg.OutputTool, "output-tool", "t", false, "Print the content of tool blocks.")
	rootCmd.Flags().BoolVarP(&cfg.OutputDiffFix, "output-diff-fix", "o", false, "Print the diff that corrected start and count.")
	rootCmd.Flags().BoolVar(&cfg.NoAnimation, "no-animation", false, "Disable loading spinner and progress updates.")
	rootCmd.Flags().StringSliceVarP(&cfg.Extensions, "extension", "e", []string{}, "Filter by extension. Use 'diff' to process only diff blocks (e.g., 'py', 'js', 'diff').")
	rootCmd.Flags().BoolVarP(&cfg.Undo, "undo", "u", false, "Undo the last operation.")
	rootCmd.Flags().BoolVarP(&cfg.Redo, "redo", "r", false, "Redo the last undone operation.")

	// Disable the default help command to prefer the --help flag
	rootCmd.SetHelpCommand(&cobra.Command{
		Use:    "no-help",
		Hidden: true,
	})
}

// Execute runs the root command.
func Execute() error {
	return rootCmd.Execute()
}
