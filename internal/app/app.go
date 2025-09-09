package app

import (
	"fmt"
	"runtime/debug"

	"itf/internal/cli"
	"itf/internal/fs"
	"itf/internal/nvim"
	"itf/internal/parser"
	"itf/internal/patcher"
	"itf/internal/source"
	"itf/internal/state"
	"itf/internal/ui"
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
	sourceProvider := source.New(cfg)

	return &App{
		cfg:            cfg,
		stateManager:   stateManager,
		pathResolver:   pathResolver,
		sourceProvider: sourceProvider,
	}, nil
}

// Run executes the main application logic based on parsed flags.
func (a *App) Run() (err error) {
	// Centralized panic recovery to provide stack traces for unexpected errors.
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
func (a *App) processContent() error {
	content, err := a.sourceProvider.GetContent()
	if err != nil {
		return err
	}
	if content == "" {
		ui.Warning("Source is empty. Nothing to process.")
		return nil
	}

	plan, err := parser.CreatePlan(content, a.pathResolver, a.cfg.Extensions)
	if err != nil {
		return fmt.Errorf("failed to create execution plan: %w", err)
	}
	if len(plan.Changes) == 0 {
		ui.Warning("No valid changes were generated. Nothing to do.")
		return nil
	}

	if ok := fs.ConfirmAndCreateDirs(plan.DirsToCreate); !ok {
		return nil // User cancelled operation.
	}

	return a.applyChanges(plan)
}

// applyChanges connects to Neovim and applies the planned file changes.
func (a *App) applyChanges(plan *parser.ExecutionPlan) error {
	manager, err := nvim.New()
	if err != nil {
		return err
	}
	defer manager.Close()

	updatedFiles, failedFiles := manager.ApplyChanges(plan.Changes)
	ui.PrintUpdateSummary(updatedFiles, failedFiles)

	if len(updatedFiles) > 0 {
		if a.cfg.Save {
			manager.SaveAllBuffers()
			ops := state.CreateOperations(updatedFiles, plan.FileActions)
			a.stateManager.Write(ops)
		} else {
			ui.Warning("\nChanges are not saved. Use -s/--save to persist them.")
			ui.Warning("Revert will not be available for this operation.")
		}
	}
	return nil
}

// fixAndPrintDiffs corrects diffs from the source and prints them to stdout.
func (a *App) fixAndPrintDiffs() error {
	content, err := a.sourceProvider.GetContent()
	if err != nil {
		return err
	}
	if content == "" {
		return nil
	}

	diffs := parser.ExtractDiffBlocks(content)
	for _, diff := range diffs {
		corrected, err := patcher.CorrectDiff(diff, a.pathResolver, a.cfg.Extensions)
		if err != nil {
			ui.Warning("Skipping diff block for '%s': %v", diff.FilePath, err)
			continue
		}
		if corrected != "" {
			fmt.Print(corrected)
		}
	}
	return nil
}

// revertLastOperation handles the undo logic.
func (a *App) revertLastOperation() error {
	ops := a.stateManager.GetOperationsToRevert()
	if len(ops) == 0 {
		return nil // Message already printed by state manager.
	}

	ui.Header("--- Reverting last operation ---")
	ui.Info("Found %d file(s) to revert:", len(ops))
	for _, op := range ops {
		ui.Path("- %s (action: %s)", op.Path, op.Action)
	}

	manager, err := nvim.New()
	if err != nil {
		return err
	}
	defer manager.Close()

	reverted, failed := manager.RevertFiles(ops)
	ui.PrintRevertSummary(reverted, failed)
	return nil
}

// redoLastOperation handles the redo logic.
func (a *App) redoLastOperation() error {
	ops := a.stateManager.GetOperationsToRedo()
	if len(ops) == 0 {
		return nil // Message already printed by state manager.
	}

	ui.Header("--- Redoing last reverted operation ---")
	ui.Info("Found %d file(s) to redo:", len(ops))
	for _, op := range ops {
		ui.Path("- %s (action: %s)", op.Path, op.Action)
	}

	manager, err := nvim.New()
	if err != nil {
		return err
	}
	defer manager.Close()

	redone, failed := manager.RedoFiles(ops)
	ui.PrintRedoSummary(redone, failed)
	return nil
}
