package patcher

import (
	"fmt"
	"strings"
)

// This file is a direct port of the logic from diff_corrector.py.

func getTargetBlock(diff []string) []string {
	var block []string
	for _, line := range diff {
		if strings.HasPrefix(line, "-") {
			block = append(block, line[1:])
		} else if !strings.HasPrefix(line, "+") {
			block = append(block, line)
		}
	}
	return block
}

func matchBlock(source, block []string) int {
	strippedBlock := make([]string, len(block))
	for i, line := range block {
		strippedBlock[i] = strings.TrimSpace(line)
	}
	strippedSource := make([]string, len(source))
	for i, line := range source {
		strippedSource[i] = strings.TrimSpace(line)
	}

	for i := 0; i <= len(strippedSource)-len(strippedBlock); i++ {
		match := true
		for j := 0; j < len(strippedBlock); j++ {
			if strippedSource[i+j] != strippedBlock[j] {
				match = false
				break
			}
		}
		if match {
			return i + 1 // 1-based line number
		}
	}
	return -1
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
	for _, hunk := range hunks {
		targetBlock := getTargetBlock(hunk)
		oldStart := matchBlock(sourceLines, targetBlock)
		if oldStart == -1 {
			// Continue trying to correct other hunks, but warn the user.
			return "", fmt.Errorf("could not find matching block for a hunk")
		}

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
