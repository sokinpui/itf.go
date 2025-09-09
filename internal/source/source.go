package source

import (
	"fmt"
	"io"
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
		content, err := io.ReadAll(os.Stdin)
		if err != nil {
			return "", fmt.Errorf("failed to read from stdin: %w", err)
		}
		return string(content), nil
	}

	content, err := clipboard.ReadAll()
	if err != nil {
		return "", fmt.Errorf("failed to read from clipboard: %w", err)
	}
	if strings.TrimSpace(content) == "" {
		return "", nil
	}
	return string(content), nil
}
