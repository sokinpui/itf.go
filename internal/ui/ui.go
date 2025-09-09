package ui

import (
	"fmt"
	"os"
	"strings"

	"github.com/fatih/color"
)

var (
	HeaderColor  = color.New(color.FgBlue, color.Bold)
	InfoColor    = color.New(color.FgCyan)
	SuccessColor = color.New(color.FgGreen)
	WarningColor = color.New(color.FgYellow)
	ErrorColor   = color.New(color.FgRed)
	PathColor    = color.New(color.FgYellow)
	PromptColor  = color.New(color.FgMagenta)
)

func Header(format string, a ...interface{}) {
	HeaderColor.Fprintf(os.Stderr, format+"\n", a...)
}

func Info(format string, a ...interface{}) {
	InfoColor.Fprintf(os.Stderr, format+"\n", a...)
}

func Success(format string, a ...interface{}) {
	SuccessColor.Fprintf(os.Stderr, format+"\n", a...)
}

func Warning(format string, a ...interface{}) {
	WarningColor.Fprintf(os.Stderr, format+"\n", a...)
}

func Error(format string, a ...interface{}) {
	ErrorColor.Fprintf(os.Stderr, format+"\n", a...)
}

func Path(format string, a ...interface{}) {
	PathColor.Fprintf(os.Stderr, "  "+format+"\n", a...)
}

func Prompt(format string, a ...interface{}) string {
	return PromptColor.Sprintf(format, a...)
}

// --- Summaries ---

func PrintUpdateSummary(diffApplied, modifiedByExt, created, failed []string) {
	Header("\n--- Update Summary ---")

	if len(diffApplied) == 0 && len(modifiedByExt) == 0 && len(created) == 0 && len(failed) == 0 {
		Info("No files were updated.")
		return
	}

	if len(diffApplied) > 0 {
		Success("Applied diff to %d file(s):", len(diffApplied))
		for _, f := range diffApplied {
			fmt.Printf("  - %s\n", f)
		}
	}
	if len(modifiedByExt) > 0 {
		Success("Modified %d file(s) via code block:", len(modifiedByExt))
		for _, f := range modifiedByExt {
			fmt.Printf("  - %s\n", f)
		}
	}
	if len(created) > 0 {
		Success("Created %d new file(s):", len(created))
		for _, f := range created {
			fmt.Printf("  - %s\n", f)
		}
	}
	if len(failed) > 0 {
		Error("Failed to process %d file(s):", len(failed))
		for _, f := range failed {
			fmt.Printf("  - %s\n", f)
		}
	}
}

func PrintRevertSummary(reverted, failed []string) {
	Header("\n--- Revert Summary ---")
	if len(reverted) > 0 {
		Success("Successfully reverted %d file(s):", len(reverted))
		for _, f := range reverted {
			fmt.Printf("  - %s\n", f)
		}
	}
	if len(failed) > 0 {
		Error("Failed to revert %d file(s):", len(failed))
		for _, f := range failed {
			fmt.Printf("  - %s\n", f)
		}
	}
}

func PrintRedoSummary(redone, failed []string) {
	Header("\n--- Redo Summary ---")
	if len(redone) > 0 {
		Success("Successfully redid %d file(s):", len(redone))
		for _, f := range redone {
			fmt.Printf("  - %s\n", f)
		}
	}
	if len(failed) > 0 {
		Error("Failed to redo %d file(s):", len(failed))
		for _, f := range failed {
			fmt.Printf("  - %s\n", f)
		}
	}
}

// --- Progress Bar ---

type ProgressBar struct {
	total  int
	prefix string
	current int
}

func NewProgressBar(total int, prefix string) *ProgressBar {
	return &ProgressBar{total: total, prefix: prefix}
}

func (p *ProgressBar) Start() {
	p.draw()
}

func (p *ProgressBar) Increment() {
	p.current++
	p.draw()
}

func (p *ProgressBar) Finish() {
	fmt.Fprintln(os.Stderr)
}

func (p *ProgressBar) draw() {
	if p.total == 0 {
		return
	}
	const barLength = 40
	percent := float64(p.current) / float64(p.total)
	filledLength := int(percent * barLength)
	bar := strings.Repeat("█", filledLength) + strings.Repeat("-", barLength-filledLength)
	
	percentStr := fmt.Sprintf("%.1f%%", percent*100)
	countStr := fmt.Sprintf("[%d/%d]", p.current, p.total)

	fmt.Fprintf(os.Stderr, "\r%s |%s| %s %s", p.prefix, bar, countStr, percentStr)
}
