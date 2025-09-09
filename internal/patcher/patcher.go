package patcher

import (
	"bytes"
	"fmt"
	"itf/internal/fs"
	"itf/internal/model"
	"itf/internal/ui"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

// filePathRegex extracts the file path from a '+++ b/...' line.
var filePathRegex = regexp.MustCompile(`(?m)^\+\+\+ b/(?P<path>.*?)(\s|$)`)

// ExtractPathFromDiff finds the file path in a raw diff string.
func ExtractPathFromDiff(content string) string {
	match := filePathRegex.FindStringSubmatch(content)
	if len(match) > 1 {
		return strings.TrimSpace(match[1])
	}
	return ""
}

// GeneratePatchedContents corrects and applies diffs to produce final file contents.
func GeneratePatchedContents(diffs []model.DiffBlock, resolver *fs.PathResolver, extensions []string) ([]model.FileChange, error) {
	if len(diffs) == 0 {
		return nil, nil
	}
	ui.Info("\nFound %d diff block(s) to process.", len(diffs))

	var changes []model.FileChange
	for _, diff := range diffs {
		if len(extensions) > 0 {
			ext := filepath.Ext(diff.FilePath)
			allowed := false
			for _, allowedExt := range extensions {
				if ext == allowedExt {
					allowed = true
					break
				}
			}
			if !allowed {
				continue
			}
		}

		patchedContent, err := CorrectDiff(diff, resolver, extensions)
		if err != nil {
			ui.Warning("  -> Diff correction failed for %s. Skipping: %v", diff.FilePath, err)
			continue
		}

		appliedContent, err := applyPatch(diff.FilePath, patchedContent, resolver)
		if err != nil {
			ui.Error("  -> Failed to apply patch for %s: %v", diff.FilePath, err)
			continue
		}

		ui.Success("  -> Successfully generated patch for: %s", diff.FilePath)
		changes = append(changes, model.FileChange{
			Path:    resolver.Resolve(diff.FilePath),
			Content: appliedContent,
			Source:  "diff",
		})
	}
	return changes, nil
}

// CorrectDiff prepares a valid patch from a raw diff block.
func CorrectDiff(diff model.DiffBlock, resolver *fs.PathResolver, extensions []string) (string, error) {
	sourcePath := resolver.ResolveExisting(diff.FilePath)
	var sourceLines []string
	if sourcePath != "" {
		content, err := os.ReadFile(sourcePath)
		if err == nil {
			sourceLines = strings.Split(string(content), "\n")
		}
	}

	ui.Info("  -> Correcting diff for: %s", diff.FilePath)
	return correctDiffHunks(sourceLines, diff.RawContent, diff.FilePath)
}

func applyPatch(filePath, patchContent string, resolver *fs.PathResolver) ([]string, error) {
	sourcePath := resolver.ResolveExisting(filePath)
	if sourcePath == "" {
		// Create a temporary empty file for patch to apply against (for new files).
		tmpFile, err := os.CreateTemp("", "itf-source-")
		if err != nil {
			return nil, fmt.Errorf("failed to create temp file: %w", err)
		}
		sourcePath = tmpFile.Name()
		defer os.Remove(sourcePath)
		tmpFile.Close()
	}

	cmd := exec.Command("patch", "-s", "-p1", "--no-backup-if-mismatch", "-o", "-", sourcePath)
	cmd.Stdin = strings.NewReader(patchContent)

	var out bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("`patch` command failed: %s", stderr.String())
	}

	// Read output and split into lines, trimming a potential trailing newline from patch.
	return strings.Split(strings.TrimSuffix(out.String(), "\n"), "\n"), nil
}
