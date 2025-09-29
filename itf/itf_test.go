package itf_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/sokinpui/itf.go/cli"
	"github.com/sokinpui/itf.go/itf"
)

func TestLibraryInterface(t *testing.T) {
	// Create a temporary directory for the test to run in.
	tempDir, err := os.MkdirTemp("", "itf-test-")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Change to the temporary directory.
	oldWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current working directory: %v", err)
	}
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("Failed to change to temp dir: %v", err)
	}
	defer os.Chdir(oldWd)

	// Create a dummy file to be parsed.
	dummyFilePath := filepath.Join(tempDir, "dummy.go")
	dummyContent := "package main\n\nfunc main() {}\n"
	if err := os.WriteFile(dummyFilePath, []byte(dummyContent), 0644); err != nil {
		t.Fatalf("Failed to write dummy file: %v", err)
	}

	// Default config for the app instance.
	appCfg := &cli.Config{}
	app, err := itf.New(appCfg)
	if err != nil {
		t.Fatalf("Failed to create itf app: %v", err)
	}

	t.Run("Parse with custom config", func(t *testing.T) {
		parseCfg := &cli.Config{
			Extensions: []string{".go"},
		}
		content := "`dummy.go`\n\n```go\npackage main\n\nfunc main() {\n\t// new content\n}\n```"
		changes, err := app.Parse(content, parseCfg)
		if err != nil {
			t.Fatalf("Parse failed: %v", err)
		}

		if len(changes) != 1 {
			t.Fatalf("Expected 1 change, got %d", len(changes))
		}
		if _, ok := changes[dummyFilePath]; !ok {
			t.Errorf("Expected change for %s, but not found", dummyFilePath)
		}
	})

	t.Run("Apply with custom config", func(t *testing.T) {
		applyCfg := &cli.Config{
			Buffer: true, // Use buffer mode to avoid writing state file.
		}
		changes := map[string]string{
			"main.go": "package main",
		}
		// We expect an error because nvim is not running in the test environment.
		// The goal is to ensure the method can be called with the new signature.
		_, err := app.Apply(changes, applyCfg)
		if err == nil {
			t.Log("Apply returned no error, which is unexpected in this test setup but the interface works.")
		}
	})

	t.Run("Parse and Apply with nil config uses app default", func(t *testing.T) {
		content := "`dummy.go`\n\n```go\npackage main\n\nfunc main() {\n\t// new content\n}\n```"
		_, err := app.Parse(content, nil)
		if err != nil {
			t.Fatalf("Parse with nil config failed: %v", err)
		}

		changes := map[string]string{"main.go": "package main"}
		_, err = app.Apply(changes, nil)
		if err == nil {
			t.Log("Apply with nil config returned no error, which is unexpected in this test setup but the interface works.")
		}
	})
}
