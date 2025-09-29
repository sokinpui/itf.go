package itf

import (
	"fmt"

	"github.com/sokinpui/itf.go/cli"
)

// Apply parses the given content string and applies the changes to files.
// It returns a summary of the operations in a map.
func Apply(content string, config cli.Config) (map[string][]string, error) {
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

func GetToolCall(content string, config cli.Config) (string, error) {
	cliCfg := &cli.Config{
		Buffer:     config.Buffer,
		Extensions: config.Extensions,
	}

	app, err := New(cliCfg)
	if err != nil {
		return "", fmt.Errorf("failed to initialize itf app: %w", err)
	}

	toolCalls, err := app.GetToolCalls(content)
	if err != nil {
		return "", err
	}

	return toolCalls, nil

}
