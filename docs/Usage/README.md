# Usage as a Go Library

The `itf` package can be used as a library in other Go applications to programmatically apply file changes from markdown content.

## Installation

To use `itf` in your project, add it as a dependency:

```bash
go get github.com/sokinpui/itf.go
```

## Applying Changes

The primary function for library usage is `itf.Apply`. It parses a string containing markdown with file or diff blocks and applies the changes to the filesystem.

### Example

Here is an example of how to use `itf.Apply` to create a new file.

```go
package main

import (
	"fmt"
	"log"

	"github.com/sokinpui/itf.go/cli"
	"github.com/sokinpui/itf.go/itf"
)

func main() {
	// Content with a file block to create a new file.
	const content = "`src/main.go`\n" +
		"```go\n" +
		"package main\n\n" +
		"func main() {\n" +
		"\tprintln(\"Hello, ITF!\")\n" +
		"}\n" +
		"```"

	// Configuration for the Apply function.
	// An empty config uses default settings.
	config := cli.Config{}

	// Apply the changes.
	summary, err := itf.Apply(content, config)
	if err != nil {
		log.Fatalf("Failed to apply changes: %v", err)
	}

	fmt.Println("Operation summary:")
	fmt.Printf("  Created: %v\n", summary["Created"])
	fmt.Printf("  Modified: %v\n", summary["Modified"])
	fmt.Printf("  Failed: %v\n", summary["Failed"])
}
```

## Extracting Tool Calls

You can also extract tool call blocks from markdown content using `itf.GetToolCall`.

### Example

```go
package main

import (
	"fmt"
	"log"

	"github.com/sokinpui/itf.go/cli"
	"github.com/sokinpui/itf.go/itf"
)

func main() {
	const content = "Some text...\n" +
		"```tool\n" +
		"{\n" +
		"    \"tool\": \"my_tool\",\n" +
		"    \"args\": [\"--arg1\", \"value1\"]\n" +
		"}\n" +
		"```"

	config := cli.Config{}

	toolCalls, err := itf.GetToolCall(string(content), config)
	if err != nil {
		log.Fatalf("Failed to get tool calls: %v", err)
	}

	fmt.Println("Extracted Tool Call:")
	fmt.Println(toolCalls)
}
```
