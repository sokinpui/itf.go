package patcher

import (
	"fmt"
	"strings"
	"unicode/utf8"
)

const (
	similarityThreshold = 0.8 // Minimum similarity score (80%) to be considered a match.
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

// levenshteinDistance calculates the number of edits (insertions, deletions,
// or substitutions) required to change one string into the other.
func levenshteinDistance(a, b string) int {
	// Using rune slices to handle multi-byte characters correctly.
	s1 := []rune(a)
	s2 := []rune(b)
	n1, n2 := len(s1), len(s2)

	if n1 == 0 {
		return n2
	}
	if n2 == 0 {
		return n1
	}

	// Initialize the dynamic programming matrix.
	// The extra row and column are for the empty string case.
	matrix := make([][]int, n1+1)
	for i := range matrix {
		matrix[i] = make([]int, n2+1)
	}

	for i := 0; i <= n1; i++ {
		matrix[i][0] = i
	}
	for j := 0; j <= n2; j++ {
		matrix[0][j] = j
	}

	// Fill the rest of the matrix.
	for i := 1; i <= n1; i++ {
		for j := 1; j <= n2; j++ {
			cost := 0
			if s1[i-1] != s2[j-1] {
				cost = 1
			}
			// Find the minimum of the three adjacent cells (deletion, insertion, substitution).
			minVal := matrix[i-1][j] + 1         // Deletion
			if matrix[i][j-1]+1 < minVal {       // Insertion
				minVal = matrix[i][j-1] + 1
			}
			if matrix[i-1][j-1]+cost < minVal { // Substitution
				minVal = matrix[i-1][j-1] + cost
			}
			matrix[i][j] = minVal
		}
	}

	return matrix[n1][n2]
}

// lineSimilarity calculates a similarity score between 0.0 and 1.0 for two strings.
func lineSimilarity(a, b string) float64 {
	distance := float64(levenshteinDistance(a, b))
	lenA := float64(utf8.RuneCountInString(a))
	lenB := float64(utf8.RuneCountInString(b))

	if lenA == 0 && lenB == 0 {
		return 1.0
	}

	// The score is 1 minus the normalized distance.
	return 1.0 - (distance / (max(lenA, lenB)))
}

// matchBlock finds the starting line number of a `block` of code within `source`.
// It uses a fuzzy matching algorithm based on Levenshtein distance to find the
// best possible match, even if the source has been slightly modified.
func matchBlock(source, block []string) int {
	if len(block) == 0 {
		return -1
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

	if len(filteredSource) < len(normalizedBlock) {
		return -1
	}

	bestScore := -1.0
	bestMatchLine := -1

	// Use a sliding window to find the best matching block.
	for i := 0; i <= len(filteredSource)-len(normalizedBlock); i++ {
		currentTotalScore := 0.0
		for j, blockLine := range normalizedBlock {
			sourceLine := filteredSource[i+j]
			currentTotalScore += lineSimilarity(sourceLine, blockLine)
		}

		avgScore := currentTotalScore / float64(len(normalizedBlock))

		if avgScore > bestScore {
			bestScore = avgScore
			bestMatchLine = originalLineNumbers[i]
		}
	}

	if bestScore >= similarityThreshold {
		return bestMatchLine
	}

	return -1 // No match found that meets the threshold.
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
