package itf

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime/debug"
	"strings"

	"github.com/sokinpui/itf.go/cli"
	"github.com/sokinpui/itf.go/internal/fs"
	"github.com/sokinpui/itf.go/internal/nvim"
	"github.com/sokinpui/itf.go/internal/parser"
	"github.com/sokinpui/itf.go/internal/patcher"
	"github.com/sokinpui/itf.go/internal/source"
	"github.com/sokinpui/itf.go/internal/state"
	"github.com/sokinpui/itf.go/model"
)

// ProgressUpdate is a callback function to report progress.
type ProgressUpdate func(current, total int)

// App orchestrates the entire application logic.
type App struct {
	cfg              *cli.Config
	stateManager     *state.Manager
	pathResolver     *fs.PathResolver
	sourceProvider   *source.SourceProvider
	progressCallback ProgressUpdate
}

// DetailedError enhances a standard error with a stack trace.
type DetailedError struct {
	Err   error
	Stack []byte
}

func (e *DetailedError) Error() string {
	return e.Err.Error()
}

// New creates a new App instance.
func New(cfg *cli.Config) (*App, error) {
	stateManager, err := state.New()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize state manager: %w", err)
	}
	pathResolver := fs.NewPathResolver(cfg.LookupDirs)
	sourceProvider := source.New()

	return &App{
		cfg:            cfg,
		stateManager:   stateManager,
		pathResolver:   pathResolver,
		sourceProvider: sourceProvider,
	}, nil
}

// SetProgressCallback sets a function to be called for progress updates.
func (a *App) SetProgressCallback(cb ProgressUpdate) {
	a.progressCallback = cb
}

// Parse creates a plan from content and returns a map of file paths to their new content.
func (a *App) Parse(content string) (map[string]string, error) {
	plan, err := parser.CreatePlan(content, a.pathResolver, a.cfg.Extensions)
	if err != nil {
		return nil, fmt.Errorf("failed to create execution plan: %w", err)
	}

	changes := make(map[string]string)
	for _, change := range plan.Changes {
		changes[change.Path] = strings.Join(change.Content, "\n")
	}

	return changes, nil
}

// Apply takes a map of file paths to content and applies the changes.
func (a *App) Apply(changes map[string]string) (model.Summary, error) {
	planChanges := make([]model.FileChange, 0, len(changes))
	targetPaths := make([]string, 0, len(changes))

	for path, content := range changes {
		planChanges = append(planChanges, model.FileChange{
			Path:    path,
			Content: strings.Split(content, "\n"),
			Source:  "library",
		})
		targetPaths = append(targetPaths, path)
	}

	actions, dirs := fs.GetFileActionsAndDirs(targetPaths)
	plan := &parser.ExecutionPlan{
		Changes:      planChanges,
		FileActions:  actions,
		DirsToCreate: dirs,
		Failed:       []string{},
	}

	if err := fs.CreateDirs(plan.DirsToCreate); err != nil {
		return model.Summary{}, err
	}

	return a.applyChanges(plan)
}

// Execute executes the main application logic based on parsed flags.
func (a *App) Execute() (summary model.Summary, err error) {
	// Centralized panic recovery.
	defer func() {
		if r := recover(); r != nil {
			err = &DetailedError{
				Err:   fmt.Errorf("internal panic: %v", r),
				Stack: debug.Stack(),
			}
		}
	}()

	switch {
	case a.cfg.Undo:
		return a.undoLastOperation()
	case a.cfg.Redo:
		return a.redoLastOperation()
	case a.cfg.OutputDiffFix:
		return a.fixAndPrintDiffs()
	default:
		return a.processContent()
	}
}

// processContent handles the core logic of parsing source, planning changes,
// and applying them in Neovim.
func (a *App) processContent() (model.Summary, error) {
	content, err := a.sourceProvider.GetContent()
	if err != nil {
		return model.Summary{}, err
	}
	if content == "" {
		return model.Summary{Message: "Source is empty. Nothing to process."}, nil
	}

	plan, err := parser.CreatePlan(content, a.pathResolver, a.cfg.Extensions)
	if err != nil {
		return model.Summary{}, fmt.Errorf("failed to create execution plan: %w", err)
	}
	if len(plan.Changes) == 0 && len(plan.Failed) == 0 {
		return model.Summary{Message: "No valid changes were generated. Nothing to do."}, nil
	}

	if err := fs.CreateDirs(plan.DirsToCreate); err != nil {
		return model.Summary{}, err
	}

	return a.applyChanges(plan)
}

// applyChanges connects to Neovim and applies the planned file changes.
func (a *App) applyChanges(plan *parser.ExecutionPlan) (model.Summary, error) {
	manager, err := nvim.New()
	if err != nil {
		return model.Summary{}, err
	}
	defer manager.Close()

	total := len(plan.Changes)
	var nvimProgressCb func(int)
	if a.progressCallback != nil {
		a.progressCallback(0, total)
		nvimProgressCb = func(current int) {
			a.progressCallback(current, total)
		}
	}

	updatedFiles, failedFromNvim := manager.ApplyChanges(plan.Changes, nvimProgressCb)
	allFailedFiles := append(plan.Failed, failedFromNvim...)

	// Categorize files for the summary.
	diffApplied := []string{}
	modifiedByExt := []string{}
	created := []string{}

	changeSourceMap := make(map[string]string, len(plan.Changes))
	for _, change := range plan.Changes {
		changeSourceMap[change.Path] = change.Source
	}

	for _, path := range updatedFiles {
		action := plan.FileActions[path]
		source := changeSourceMap[path]

		if action == "create" {
			created = append(created, path)
		} else if action == "modify" {
			if source == "diff" {
				diffApplied = append(diffApplied, path)
			} else { // "codeblock" or "library"
				modifiedByExt = append(modifiedByExt, path)
			}
		}
	}

	if len(updatedFiles) > 0 {
		if !a.cfg.Buffer { // Save by default
			manager.SaveAllBuffers()
			ops := state.CreateOperations(updatedFiles, plan.FileActions)
			a.stateManager.Write(ops)
		} else {
			// TODO: Add this info to the summary message if needed.
		}
	}

	summary := model.Summary{
		Created:  created,
		Modified: append(diffApplied, modifiedByExt...),
		Failed:   allFailedFiles,
	}
	a.relativizeSummaryPaths(&summary)
	return summary, nil
}

// fixAndPrintDiffs corrects diffs from the source and prints them to stdout.
func (a *App) fixAndPrintDiffs() (model.Summary, error) {
	content, err := a.sourceProvider.GetContent()
	if err != nil {
		return model.Summary{}, err
	}
	if content == "" {
		return model.Summary{}, nil
	}

	diffs := parser.ExtractDiffBlocks(content)
	for _, diff := range diffs {
		corrected, err := patcher.CorrectDiff(diff, a.pathResolver, a.cfg.Extensions)
		if err != nil {
			// Silently skip failures for this mode.
			continue
		}
		if corrected != "" {
			fmt.Print(corrected)
		}
	}
	return model.Summary{}, nil
}

// undoLastOperation handles the undo logic.
func (a *App) undoLastOperation() (model.Summary, error) {
	ops := a.stateManager.GetOperationsToUndo()
	if len(ops) == 0 {
		return model.Summary{Message: "No operation to undo."}, nil
	}

	manager, err := nvim.New()
	if err != nil {
		return model.Summary{}, err
	}
	defer manager.Close()

	total := len(ops)
	var nvimProgressCb func(int)
	if a.progressCallback != nil {
		a.progressCallback(0, total)
		nvimProgressCb = func(current int) {
			a.progressCallback(current, total)
		}
	}

	undone, failed := manager.UndoFiles(ops, nvimProgressCb)

	summary := model.Summary{
		Modified: undone,
		Failed:   failed,
		Message:  "Undid last operation.",
	}
	a.relativizeSummaryPaths(&summary)
	return summary, nil
}

// redoLastOperation handles the redo logic.
func (a *App) redoLastOperation() (model.Summary, error) {
	ops := a.stateManager.GetOperationsToRedo()
	if len(ops) == 0 {
		return model.Summary{Message: "No operation to redo."}, nil
	}

	manager, err := nvim.New()
	if err != nil {
		return model.Summary{}, err
	}
	defer manager.Close()

	total := len(ops)
	var nvimProgressCb func(int)
	if a.progressCallback != nil {
		a.progressCallback(0, total)
		nvimProgressCb = func(current int) {
			a.progressCallback(current, total)
		}
	}

	redone, failed := manager.RedoFiles(ops, nvimProgressCb)

	summary := model.Summary{
		Modified: redone,
		Failed:   failed,
		Message:  "Redid last undone operation.",
	}
	a.relativizeSummaryPaths(&summary)
	return summary, nil
}

// relativizeSummaryPaths converts absolute file paths in a summary to be
// relative to the current working directory for cleaner display.
func (a *App) relativizeSummaryPaths(summary *model.Summary) {
	wd, err := os.Getwd()
	if err != nil {
		// Cannot get CWD, so we can't make paths relative.
		// Return without changing anything.
		return
	}

	makeRelative := func(absPaths []string) []string {
		relPaths := make([]string, len(absPaths))
		for i, p := range absPaths {
			rel, err := filepath.Rel(wd, p)
			if err != nil {
				relPaths[i] = p // Fallback to absolute path
			} else {
				relPaths[i] = rel
			}
		}
		return relPaths
	}

	summary.Created = makeRelative(summary.Created)
	summary.Modified = makeRelative(summary.Modified)
	summary.Failed = makeRelative(summary.Failed)
}
