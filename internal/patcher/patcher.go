package patcher

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/sokinpui/itf.go/internal/fs"
	"github.com/sokinpui/itf.go/model"
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
func GeneratePatchedContents(diffs []model.DiffBlock, resolver *fs.PathResolver, extensions []string) ([]model.FileChange, []string, error) {
	if len(diffs) == 0 {
		return nil, nil, nil
	}

	var changes []model.FileChange
	var failedPaths []string
	for _, diff := range diffs {
		fullPath := resolver.Resolve(diff.FilePath)
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
			failedPaths = append(failedPaths, fullPath)
			continue
		}

		appliedContent, err := applyPatch(diff.FilePath, patchedContent, resolver)
		if err != nil {
			failedPaths = append(failedPaths, fullPath)
			continue
		}

		changes = append(changes, model.FileChange{
			Path:     fullPath,
			Content:  appliedContent,
			Source:   "diff",
			RawBlock: fmt.Sprintf("```diff\n%s\n```", diff.RawContent),
		})
	}
	return changes, failedPaths, nil
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

	devNull := "/dev/null"
	cmd := exec.Command("patch", "-s", "-p1", "--no-backup-if-mismatch", "-r", devNull, "-o", "-", sourcePath)
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
