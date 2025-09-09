package source

import (
	"fmt"
	"io"
	"itf/internal/ui"
	"os"
	"strings"

	"github.com/atotto/clipboard"
)

// SourceProvider determines and retrieves the source content.
type SourceProvider struct{}

// New creates a new SourceProvider.
func New() *SourceProvider {
	return &SourceProvider{}
}

// GetContent retrieves content from stdin (if piped) or the clipboard.
func (sp *SourceProvider) GetContent() (string, error) {
	stat, _ := os.Stdin.Stat()
	isPiped := (stat.Mode() & os.ModeCharDevice) == 0

	if isPiped {
		ui.Header("--- Reading from stdin ---")
		content, err := io.ReadAll(os.Stdin)
		if err != nil {
			return "", fmt.Errorf("failed to read from stdin: %w", err)
		}
		return string(content), nil
	}

	ui.Header("--- Reading from clipboard ---")
	content, err := clipboard.ReadAll()
	if err != nil {
		return "", fmt.Errorf("failed to read from clipboard: %w", err)
	}
	if strings.TrimSpace(content) == "" {
		ui.Warning("Clipboard is empty. Nothing to process.")
		return "", nil
	}
	return string(content), nil
}
