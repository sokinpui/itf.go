package itf

import (
	"fmt"

	"github.com/sokinpui/itf.go/cli"
)

// Config for using itf as a library.
type Config struct {
	// Update buffers without saving them to disk.
	Buffer bool
	// Filter by extension. Use 'diff' to process only diff blocks (e.g., 'py', 'js', 'diff').
	Extensions []string
}

// Apply parses the given content string and applies the changes to files.
// It returns a summary of the operations in a map.
func Apply(content string, config Config) (map[string][]string, error) {
	cliCfg := &cli.Config{
		Buffer:     config.Buffer,
		Extensions: config.Extensions,
	}

	app, err := New(cliCfg)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize itf app: %w", err)
	}

	summary, err := app.processAndApply(content)
	if err != nil {
		return nil, err
	}

	result := map[string][]string{
		"Created":  summary.Created,
		"Modified": summary.Modified,
		"Failed":   summary.Failed,
	}

	return result, nil
}
