package app

import (
	"fmt"
	"runtime/debug"

	"itf/internal/cli"
	"itf/internal/fs"
	"itf/internal/model"
	"itf/internal/nvim"
	"itf/internal/parser"
	"itf/internal/patcher"
	"itf/internal/source"
	"itf/internal/state"
)

// App orchestrates the entire application logic.
type App struct {
	cfg            *cli.Config
	stateManager   *state.Manager
	pathResolver   *fs.PathResolver
	sourceProvider *source.SourceProvider
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
	case a.cfg.Revert:
		return a.revertLastOperation()
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
	if len(plan.Changes) == 0 {
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

	updatedFiles, failedFiles := manager.ApplyChanges(plan.Changes)

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
			} else { // "codeblock"
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

	return model.Summary{
		Created:  created,
		Modified: append(diffApplied, modifiedByExt...),
		Failed:   failedFiles,
	}, nil
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

// revertLastOperation handles the undo logic.
func (a *App) revertLastOperation() (model.Summary, error) {
	ops := a.stateManager.GetOperationsToRevert()
	if len(ops) == 0 {
		return model.Summary{Message: "No operation to revert."}, nil
	}

	manager, err := nvim.New()
	if err != nil {
		return model.Summary{}, err
	}
	defer manager.Close()

	reverted, failed := manager.RevertFiles(ops)

	return model.Summary{
		Modified: reverted,
		Failed:   failed,
		Message:  "Reverted last operation.",
	}, nil
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

	redone, failed := manager.RedoFiles(ops)

	return model.Summary{
		Modified: redone,
		Failed:   failed,
		Message:  "Redid last reverted operation.",
	}, nil
}
