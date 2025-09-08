package parser

import (
	"fmt"
	"itf/internal/model"
	"itf/internal/fs"
	"itf/internal/patcher"
	"itf/internal/ui"
	"path/filepath"
	"regexp"
	"strings"
)

// Mode defines the parsing strategy.
type Mode int

const (
	ModeAuto Mode = iota
	ModeFiles
	ModeDiffs
)

// ExecutionPlan contains all the changes and setup needed for an operation.
type ExecutionPlan struct {
	Changes      []model.FileChange
	FileActions  map[string]string // Maps absolute path to "create" or "modify"
	DirsToCreate map[string]struct{}
}

var (
	blockWithHintRegex = regexp.MustCompile(
		`(?m)(?:^(?P<hint_line>[^\n]*)\n(?:\s*\n)?)?` +
			`^` + "```" + `(?P<lang>[a-z]*)\s*\n` +
			`(?P<content>[\s\S]*?)` +
			`^\s*` + "```" + `\s*$`)
	pathInHintRegex = regexp.MustCompile("`([^`\\n]+)`")
)

// CreatePlan parses content and generates a plan of file changes.
func CreatePlan(content string, mode Mode, resolver *fs.PathResolver, extensions []string) (*ExecutionPlan, error) {
	var fileBlocks []model.FileChange
	var diffBlocks []model.DiffBlock

	if mode == ModeAuto || mode == ModeFiles {
		fileBlocks = parseFileBlocks(content, resolver, extensions)
	}
	if mode == ModeAuto || mode == ModeDiffs {
		diffBlocks = ExtractDiffBlocks(content)
	}

	if mode == ModeAuto {
		ui.Header("--- Auto-detecting and applying changes ---")
	}

	patchedChanges, err := patcher.GeneratePatchedContents(diffBlocks, resolver, extensions)
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
	}, nil
}

func parseFileBlocks(content string, resolver *fs.PathResolver, extensions []string) []model.FileChange {
	matches := blockWithHintRegex.FindAllStringSubmatch(content, -1)
	var blocks []model.FileChange

	for _, match := range matches {
		result := make(map[string]string)
		for i, name := range blockWithHintRegex.SubexpNames() {
			if i != 0 && name != "" {
				result[name] = match[i]
			}
		}

		if result["lang"] == "diff" {
			continue // Diffs are handled separately.
		}

		filePath := extractPathFromHint(result["hint_line"])
		if filePath == "" {
			continue
		}

		if !hasAllowedExtension(filePath, extensions) {
			continue
		}

		blockContent := strings.TrimRight(result["content"], "\n")
		lines := strings.Split(blockContent, "\n")
		// Handle empty blocks correctly.
		if len(lines) == 1 && lines[0] == "" {
			lines = []string{}
		}

		blocks = append(blocks, model.FileChange{
			Path:    resolver.Resolve(filePath),
			Content: lines,
		})
	}
	return blocks
}

// ExtractDiffBlocks finds all diff blocks in the content.
func ExtractDiffBlocks(content string) []model.DiffBlock {
	matches := blockWithHintRegex.FindAllStringSubmatch(content, -1)
	var diffs []model.DiffBlock

	for _, match := range matches {
		result := make(map[string]string)
		for i, name := range blockWithHintRegex.SubexpNames() {
			if i != 0 && name != "" {
				result[name] = match[i]
			}
		}

		if result["lang"] != "diff" {
			continue
		}

		rawContent := strings.TrimSpace(result["content"])
		filePath := patcher.ExtractPathFromDiff(rawContent)
		if filePath == "" {
			ui.Warning("Found a diff block but could not extract a file path. Skipping.")
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
	if hint == "" {
		return ""
	}

	if match := pathInHintRegex.FindStringSubmatch(hint); len(match) > 1 {
		path := strings.TrimSpace(match[1])
		if !strings.Contains(path, " ") {
			return path
		}
	}

	cleaned := strings.TrimPrefix(hint, "#")
	cleaned = strings.Trim(cleaned, "* ")
	if !strings.Contains(cleaned, " ") {
		return cleaned
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
