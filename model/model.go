package model

// FileChange represents a single planned change to a file.
type FileChange struct {
	Path    string
	Content []string
	Source   string
	RawBlock string // The full original code block, e.g., "```go\n...\n```"
}

// DiffBlock represents a raw diff block from the source content.
type DiffBlock struct {
	FilePath   string
	RawContent string
}

// ToolBlock represents a raw tool block from the source content.
type ToolBlock struct {
	Content string
}

// Summary holds the results of an operation for display.
type Summary struct {
	Created  []string
	Modified []string
	Failed   []string
	Message  string
}
