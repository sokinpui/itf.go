package itf_test

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/sokinpui/itf.go/cli"
	"github.com/sokinpui/itf.go/itf"
)

func TestApply(t *testing.T) {
	chdirToRoot(t)

	t.Cleanup(func() {
		os.RemoveAll("web")
	})

	config := cli.Config{}

	// Use inline content that creates a file, so the test is self-contained.
	const content = "`web/src/index.js`\n```js\nconsole.log(\"hello world\");\n```"

	app, err := itf.Apply(content, config)
	if err != nil {
		t.Fatal(err)
	}
	if len(app["Created"]) == 0 {
		t.Fatal("expected files to be created, but none were")
	}
	if !strings.HasSuffix(app["Created"][0], "web/src/index.js") {
		t.Fatalf("expected 'web/src/index.js' to be created, got '%s'", app["Created"][0])
	}
}

func TestGetToolCall(t *testing.T) {
	chdirToRoot(t)

	config := cli.Config{}
	filepath := "tmp.test"
	content, err := os.ReadFile(filepath)
	if err != nil {
		t.Fatalf("failed to read tmp.test: %v", err)
	}

	toolCalls, err := itf.GetToolCall(string(content), config)
	if err != nil {
		t.Fatal(err)
	}

	fmt.Println(toolCalls)

	expected := `{
    "tool": "test_tool",
    "args": [
        "--option1", "value1",
        "--option2", "value2"
    ]
}`
	if toolCalls != expected {
		t.Errorf("GetToolCall() mismatch:\ngot:\n'%s'\nwant:\n'%s'", toolCalls, expected)
	}
}

// chdirToRoot changes the working directory to the project root for the duration of a test.
// It registers a cleanup function to restore the original working directory.
func chdirToRoot(t *testing.T) {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get current working directory: %v", err)
	}

	// Tests run inside the 'itf' package, so we go up one level to the project root.
	if err := os.Chdir(".."); err != nil {
		t.Fatalf("failed to change directory to project root: %v", err)
	}

	t.Cleanup(func() {
		if err := os.Chdir(wd); err != nil {
			t.Fatalf("failed to restore working directory: %v", err)
		}
	})
}
