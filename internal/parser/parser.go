package parser

import (
	"fmt"
	"github.com/sokinpui/itf.go/internal/fs"
	"github.com/sokinpui/itf.go/internal/model"
	"github.com/sokinpui/itf.go/internal/patcher"
	"path/filepath"
	"regexp"
	"strings"
)

// ExecutionPlan contains all the changes and setup needed for an operation.
type ExecutionPlan struct {
	Changes      []model.FileChange
	FileActions  map[string]string // Maps absolute path to "create" or "modify"
	DirsToCreate map[string]struct{}
	Failed       []string // Files that failed during planning (e.g., bad patch)
}

var (
	// pathInHintRegex extracts a path from a hint line, e.g., `path/to/file.go`.
	pathInHintRegex = regexp.MustCompile("^`([^`\n]+)`")
)

// CreatePlan parses content and generates a plan of file changes.
func CreatePlan(content string, resolver *fs.PathResolver, extensions []string) (*ExecutionPlan, error) {
	allBlocks, err := ExtractCodeBlocks([]byte(content))
	if err != nil {
		return nil, fmt.Errorf("failed to parse markdown content: %w", err)
	}

	// If '.diff' is the ONLY extension, we are in a special diff-only mode.
	isDiffOnlyMode := len(extensions) == 1 && extensions[0] == ".diff"

	var fileBlocks []model.FileChange
	if !isDiffOnlyMode {
		fileBlocks = parseFileBlocks(allBlocks, resolver, extensions)
	}

	diffBlocks := extractDiffBlocksFromParsed(allBlocks)

	patcherExtensions := extensions
	if isDiffOnlyMode {
		// In diff-only mode, don't filter patches by extension.
		patcherExtensions = []string{}
	}
	patchedChanges, failedPatches, err := patcher.GeneratePatchedContents(diffBlocks, resolver, patcherExtensions)
	if err != nil {
		return nil, fmt.Errorf("failed during patch generation: %w", err)
	}

	// Combine changes, letting file blocks overwrite diff patches for the same file.
	finalChanges := make(map[string]model.FileChange)
	for _, change := range patchedChanges {
		finalChanges[change.Path] = change
	}
	for _, block := range fileBlocks {
		finalChanges[block.Path] = block
	}

	// Convert map to slice for ordered processing.
	planChanges := make([]model.FileChange, 0, len(finalChanges))
	targetPaths := make([]string, 0, len(finalChanges))
	for _, change := range finalChanges {
		planChanges = append(planChanges, change)
		targetPaths = append(targetPaths, change.Path)
	}

	actions, dirs := fs.GetFileActionsAndDirs(targetPaths)
	return &ExecutionPlan{
		Changes:      planChanges,
		FileActions:  actions,
		DirsToCreate: dirs,
		Failed:       failedPatches,
	}, nil
}

func parseFileBlocks(allBlocks []CodeBlock, resolver *fs.PathResolver, extensions []string) []model.FileChange {
	var blocks []model.FileChange

	for _, block := range allBlocks {
		if block.Lang == "diff" {
			continue // Diffs are handled separately.
		}

		filePath := extractPathFromHint(block.Hint)
		if filePath == "" {
			continue
		}

		if !hasAllowedExtension(filePath, extensions) {
			continue
		}

		blockContent := strings.TrimRight(block.Content, "\n")
		lines := strings.Split(blockContent, "\n")
		// Handle empty blocks correctly.
		if len(lines) == 1 && lines[0] == "" {
			lines = []string{}
		}

		blocks = append(blocks, model.FileChange{
			Path:    resolver.Resolve(filePath),
			Content: lines,
			Source:  "codeblock",
		})
	}
	return blocks
}

// ExtractDiffBlocks finds all diff blocks in the content.
func ExtractDiffBlocks(content string) []model.DiffBlock {
	allBlocks, err := ExtractCodeBlocks([]byte(content))
	if err != nil {
		// This function is also used for non-critical paths like --output-diff-fix,
		// so we just return nil. The error isn't critical here.
		return nil
	}
	return extractDiffBlocksFromParsed(allBlocks)
}

// extractDiffBlocksFromParsed is a helper to process already-parsed blocks.
func extractDiffBlocksFromParsed(allBlocks []CodeBlock) []model.DiffBlock {
	var diffs []model.DiffBlock

	for _, block := range allBlocks {
		if block.Lang != "diff" {
			continue
		}

		rawContent := strings.TrimSpace(block.Content)
		filePath := patcher.ExtractPathFromDiff(rawContent)
		if filePath == "" {
			// Silently skip blocks without a path.
			continue
		}

		diffs = append(diffs, model.DiffBlock{
			FilePath:   filePath,
			RawContent: rawContent,
		})
	}
	return diffs
}

func extractPathFromHint(hint string) string {
	hint = strings.TrimSpace(hint)

	// A path hint must be enclosed in backticks, e.g., `path/to/file.go`
	if match := pathInHintRegex.FindStringSubmatch(hint); len(match) > 1 {
		path := strings.TrimSpace(match[1])
		// Disallow spaces to avoid capturing commands like `go run main.go` as a path.
		if !strings.Contains(path, " ") {
			return path
		}
	}

	return ""
}

func hasAllowedExtension(path string, extensions []string) bool {
	if len(extensions) == 0 {
		return true
	}
	ext := filepath.Ext(path)
	for _, allowedExt := range extensions {
		if ext == allowedExt {
			return true
		}
	}
	return false
}
