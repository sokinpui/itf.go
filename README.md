Golang rewrite of python [itf](https://github.com/sokinpui/itf)

# ITF: Insert To File

`itf` is a command-line tool that parses markdown content from stdin or your clipboard and applies the changes to your local files. It's designed to streamline workflows with Large Language Models (LLMs) by eliminating the need to manually copy and paste code snippets.

It can create new files, modify existing ones using code blocks, or apply changes from diff blocks.

## Features

- **Clipboard & Pipe Integration**: Reads content directly from your clipboard or standard input.
- **File & Diff Block Parsing**: Intelligently parses markdown to identify file paths and content for file creation/modification, as well as diff hunks for patching.
- **Neovim Integration**: Uses Neovim under the hood to apply changes, either to files on disk or just to buffers. It can connect to a running Neovim instance or start its own headless one.
- **Undo/Redo**: Supports undoing and redoing file operations.
- **Interactive TUI**: Provides real-time feedback on the operations being performed.
- **Tool Call Extraction**: Can extract and print `tool` code blocks.
- **Extensible as a Library**: Can be used as a Go library in other projects.

## Installation

You can install `itf` using `go install`:

```bash
go install github.com/sokinpui/itf.go/cmd/itf@latest
```

## Basic Usage

Copy some markdown content containing a file block to your clipboard, then run:

```bash
itf
```

Or, pipe content to it:

```bash
pbpaste | itf
```

### Example

Given the following content on your clipboard:

`path/to/hello.go`
```go
package main

func main() {
	println("Hello, ITF!")
}
```

Running `itf` will create a new file at `path/to/hello.go` with the specified content.

## Documentation

For more detailed information, please refer to the documentation in the [`docs`](./docs) directory.

- [Installation](./docs/Installation/README.md)
- [Usage](./docs/Usage/README.md)
- [API (Library Usage)](./docs/Api/README.md)
- [Architecture](./docs/Architecture/README.md)
