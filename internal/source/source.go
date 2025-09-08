package source

import (
	"fmt"
	"itf/internal/cli"
	"itf/internal/ui"
	"os"
	"path/filepath"
	"strings"

	"github.com/atotto/clipboard"
)

const sourceFileName = "itf.txt"

// SourceProvider determines and retrieves the source content.
type SourceProvider struct {
	cfg *cli.Config
}

// New creates a new SourceProvider.
func New(cfg *cli.Config) *SourceProvider {
	return &SourceProvider{cfg: cfg}
}

// GetContent retrieves content from the clipboard or a file based on flags.
func (sp *SourceProvider) GetContent() (string, error) {
	// Special auto-detection logic for auto or diff-fix modes.
	isAutoOrFix := sp.cfg.Auto || sp.cfg.OutputDiffFix
	if isAutoOrFix {
		if !sp.cfg.OutputDiffFix {
			ui.Header("--- Auto mode: searching for content ---")
		}
		content, _ := clipboard.ReadAll()
		if strings.TrimSpace(content) != "" {
			ui.Info("-> Found content in clipboard.")
			return content, nil
		}
		ui.Info("-> Clipboard is empty, falling back to '%s'.", sourceFileName)
		return sp.readFromFile()
	}

	if sp.cfg.Clipboard {
		return clipboard.ReadAll()
	}

	return sp.readFromFile()
}

func (sp *SourceProvider) readFromFile() (string, error) {
	filePath := filepath.Join(".", sourceFileName)
	content, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			ui.Error("Source file '%s' not found.", sourceFileName)
			ui.Info("Use -c to read from clipboard or -a for auto-detection.")
			return "", nil // Return empty string, not an error, to allow graceful exit.
		}
		return "", fmt.Errorf("error reading source file '%s': %w", sourceFileName, err)
	}
	return string(content), nil
}
