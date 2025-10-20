# CLI Usage

`itf` is a command-line tool that reads markdown content from standard input or the clipboard and applies the contained file changes.

## Basic Workflow

1.  Copy markdown content from a web UI, document, or any other source. The content should contain file blocks or diff blocks.
2.  Run `itf` in your terminal.

`itf` will parse the content and apply the changes.

```bash
# Read from clipboard
itf

# Read from stdin
cat content.md | itf
pbpaste | itf # on macOS
```

## Input Formats

`itf` recognizes two main types of blocks in markdown: file blocks and diff blocks.

### File Blocks

A file block is a standard markdown code block preceded by a line containing the file path in backticks.

**Example: Creating a new file**

````
`path/to/new_file.go`
```go
package main

func main() {
    // ...
}
````

Running `itf` with this content on the clipboard will create `path/to/new_file.go`.

**Example: Modifying an existing file**

If `path/to/new_file.go` already exists, `itf` will overwrite its content.

### Diff Blocks

A diff block is a code block with the language identifier `diff`. It should contain a standard unified diff.

**Example: Applying a patch**

```diff
--- a/src/main.go
+++ b/src/main.go
@@ -1,5 +1,6 @@
 package main

 func main() {
-	println("Hello, ITF!")
+	println("Hello, world!")
+	println("Another line")
 }
```

`itf` will attempt to apply this patch to `src/main.go`. It is robust and can correct diffs that are slightly out of date.

### Delete Blocks

A delete block is a code block with the language identifier `delete`. It should contain a list of file paths to be deleted, one per line.

**Example: Deleting files**

```delete
path/to/obsolete_file.go
old_data.json
```

`itf` will move these files to a trash directory within its state folder (`.itf/trash/`) to allow for undoing the operation.

### Rename Blocks

A rename block is a code block with the language identifier `rename`. It should contain a list of old and new file paths, separated by a space, one pair per line.

**Example: Renaming files**

```rename
src/old_name.go src/new_name.go
config.yaml config.yml
```

`itf` will rename these files. This operation can also be undone.

## Command-Line Flags

`itf` provides several flags to control its behavior.

| Flag                | Shorthand | Description                                                                       |
| ------------------- | --------- | --------------------------------------------------------------------------------- |
| `--extension`       | `-e`      | Filter by file extension (e.g., `-e go -e js`). Use `-e diff` for diff-only mode. |
| `--buffer`          | `-b`      | Apply changes to Neovim buffers without saving them to disk.                      |
| `--undo`            | `-u`      | Undo the last operation.                                                          |
| `--redo`            | `-r`      | Redo the last undone operation.                                                   |
| `--output-tool`     | `-t`      | Print the content of `tool` blocks instead of applying changes.                   |
| `--output-diff-fix` | `-o`      | Print a corrected version of the diffs found in the input.                        |
| `--no-animation`    |           | Disable the loading spinner and progress updates.                                 |
| `--completion`      |           | Generate a shell completion script (e.g., `bash`, `zsh`).                         |
| `--help`            | `-h`      | Show the help message.                                                            |

### Filtering by Extension

You can process only files with specific extensions.

```bash
# Only process .go and .md files
pbpaste | itf -e go -e md
```

### Diff-Only Mode

To process _only_ diff blocks and ignore all file blocks, use `-e diff`.

```bash
pbpaste | itf -e diff
```

### Undo and Redo

`itf` keeps a history of operations. You can easily undo and redo changes.

```bash
# Undo the last set of changes
itf -u

# Redo the changes you just undid
itf -r
```
