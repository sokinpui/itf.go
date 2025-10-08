package patcher

import (
	"fmt"
	"strings"
)

// This file is a direct port of the logic from diff_corrector.py.

// getTargetBlock creates a "search pattern" from a diff hunk.
// It uses only lines that are guaranteed to be in the original source file
// (context ` ` and removed `-` lines). It also ignores empty lines to make
// matching more robust against whitespace-only changes.
func getTargetBlock(diff []string) []string {
	var block []string
	for _, line := range diff {
		var content string
		isTarget := false

		if strings.HasPrefix(line, "-") {
			content = line[1:]
			isTarget = true
		} else if strings.HasPrefix(line, " ") {
			content = line[1:]
			isTarget = true
		}

		if isTarget && strings.TrimSpace(content) != "" {
			block = append(block, content)
		}
	}
	return block
}

// normalizeLineForMatching prepares a line for comparison by trimming whitespace
// and normalizing all internal whitespace sequences to a single space.
func normalizeLineForMatching(line string) string {
	return strings.Join(strings.Fields(line), " ")
}

// matchBlock finds the starting line number of a `block` of code within `source`.
// It is designed to be resilient to whitespace and empty line changes. It works by:
// 1. Temporarily filtering out empty lines from the source.
// 2. Keeping a map of filtered line numbers back to their original line numbers.
// 3. Performing a whitespace-normalized comparison to find the block.
// 4. Returning the original line number where the match began.
func matchBlock(source, block []string, startLine int) (int, int) {
	if len(block) == 0 {
		return -1, -1
	}

	normalizedBlock := make([]string, len(block))
	for i, line := range block {
		normalizedBlock[i] = normalizeLineForMatching(line)
	}

	var filteredSource []string
	var originalLineNumbers []int
	for i, line := range source {
		normalizedLine := normalizeLineForMatching(line)
		if normalizedLine != "" {
			filteredSource = append(filteredSource, normalizedLine)
			originalLineNumbers = append(originalLineNumbers, i+1)
		}
	}

	searchStartIndex := 0
	if startLine > 1 {
		for i, lineNum := range originalLineNumbers {
			if lineNum >= startLine {
				searchStartIndex = i
				break
			}
		}
	}

	for i := searchStartIndex; i <= len(filteredSource)-len(normalizedBlock); i++ {
		match := true
		for j := 0; j < len(normalizedBlock); j++ {
			if filteredSource[i+j] != normalizedBlock[j] {
				match = false
				break
			}
		}
		if match {
			matchStartLine := originalLineNumbers[i]
			matchEndLine := originalLineNumbers[i+len(normalizedBlock)-1]
			return matchStartLine, matchEndLine
		}
	}
	return -1, -1
}

func buildHunkHeader(oldStart, oldLines, newStart, newLines int) string {
	return fmt.Sprintf("@@ -%d,%d +%d,%d @@\n", oldStart, oldLines, newStart, newLines)
}

func parseDiffToHunks(diffLines []string) [][]string {
	var hunks [][]string
	var currentHunk []string

	for _, line := range diffLines {
		if strings.HasPrefix(line, "---") || strings.HasPrefix(line, "+++") {
			continue
		}
		if strings.HasPrefix(line, "@@") {
			if len(currentHunk) > 0 {
				hunks = append(hunks, currentHunk)
			}
			currentHunk = nil
		} else if strings.HasPrefix(line, "+") || strings.HasPrefix(line, "-") || strings.HasPrefix(line, " ") {
			currentHunk = append(currentHunk, line)
		}
	}
	if len(currentHunk) > 0 {
		hunks = append(hunks, currentHunk)
	}
	return hunks
}

func correctDiffHunks(sourceLines []string, rawDiffContent, sourceFilePath string) (string, error) {
	diffLines := strings.Split(rawDiffContent, "\n")
	hunks := parseDiffToHunks(diffLines)
	if len(hunks) == 0 {
		return "", nil
	}

	var correctedParts []string
	correctedParts = append(correctedParts, fmt.Sprintf("--- a/%s\n", sourceFilePath))
	correctedParts = append(correctedParts, fmt.Sprintf("+++ b/%s\n", sourceFilePath))

	lineDiffOffset := 0
	lastMatchEndLine := 0
	for _, hunk := range hunks {
		targetBlock := getTargetBlock(hunk)
		oldStart, matchEndLine := matchBlock(sourceLines, targetBlock, lastMatchEndLine+1)
		if oldStart == -1 {
			// Continue trying to correct other hunks, but warn the user.
			return "", fmt.Errorf("could not find matching block for a hunk")
		}
		lastMatchEndLine = matchEndLine

		addCount, removeCount := 0, 0
		for _, line := range hunk {
			if strings.HasPrefix(line, "+") {
				addCount++
			} else if strings.HasPrefix(line, "-") {
				removeCount++
			}
		}
		contextCount := len(hunk) - addCount - removeCount

		oldLines := contextCount + removeCount
		newLines := contextCount + addCount
		newStart := oldStart + lineDiffOffset

		header := buildHunkHeader(oldStart, oldLines, newStart, newLines)
		correctedParts = append(correctedParts, header)
		for _, line := range hunk {
			correctedParts = append(correctedParts, line+"\n")
		}

		lineDiffOffset += newLines - oldLines
	}

	return strings.Join(correctedParts, ""), nil
}
