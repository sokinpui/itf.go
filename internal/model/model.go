package model

// FileChange represents a single planned change to a file.
type FileChange struct {
	Path    string
	Content []string
}

// DiffBlock represents a raw diff block from the source content.
type DiffBlock struct {
	FilePath   string
	RawContent string
}
