package nvim

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/neovim/go-client/nvim"

	"github.com/sokinpui/itf.go/internal/fs"
	"github.com/sokinpui/itf.go/internal/state"
	"github.com/sokinpui/itf.go/model"
)

const (
	undoDir = "~/.local/state/nvim/undo/"
)

// Manager handles the connection and interaction with a Neovim instance.
type Manager struct {
	nvim          *nvim.Nvim
	isSelfStarted bool
	cmd           *exec.Cmd
	socketPath    string
}

// New creates a new Neovim manager, connecting to an existing instance
// or starting a new headless one.
func New() (*Manager, error) {
	// Try to connect to a running instance first.
	if addr := os.Getenv("NVIM_LISTEN_ADDRESS"); addr != "" {
		v, err := nvim.Dial(addr)
		if err == nil {
			return &Manager{nvim: v}, nil
		}
	}

	// If that fails, start a temporary headless instance.
	tmpDir, err := os.MkdirTemp("", "itf-nvim-")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp dir for nvim: %w", err)
	}
	socketPath := filepath.Join(tmpDir, "nvim.sock")

	cmd := exec.Command("nvim", "--headless", "--clean", "--listen", socketPath)
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start headless nvim: %w. Is 'nvim' in your PATH?", err)
	}

	// Wait for the socket file to appear.
	for i := 0; i < 20; i++ {
		if _, err := os.Stat(socketPath); err == nil {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}

	v, err := nvim.Dial(socketPath)
	if err != nil {
		cmd.Process.Kill()
		return nil, fmt.Errorf("failed to connect to headless nvim: %w", err)
	}

	m := &Manager{
		nvim:          v,
		isSelfStarted: true,
		cmd:           cmd,
		socketPath:    socketPath,
	}
	m.configureTempInstance()
	return m, nil
}

// configureTempInstance sets up undofile for persistent history.
func (m *Manager) configureTempInstance() {
	home, _ := os.UserHomeDir()
	expandedUndoDir := strings.Replace(undoDir, "~", home, 1)
	os.MkdirAll(expandedUndoDir, 0755)

	b := m.nvim.NewBatch()
	b.Command("set undofile")
	b.Command(fmt.Sprintf("set undodir=%s", expandedUndoDir))
	b.Command("set noswapfile")
	if err := b.Execute(); err != nil {
		// Non-fatal error, just log it somewhere if needed in the future.
	}
}

// Close disconnects from Neovim and cleans up if it was self-started.
func (m *Manager) Close() {
	if m.nvim != nil {
		m.nvim.Close()
	}
	if m.isSelfStarted && m.cmd != nil && m.cmd.Process != nil {
		if err := m.cmd.Process.Kill(); err == nil {
			m.cmd.Wait()
			os.RemoveAll(filepath.Dir(m.socketPath))
		}
	}
}

// processSequentially is a generic helper function to run a set of jobs sequentially.
func processSequentially[T any](
	items []T,
	processFn func(item T) (path string, success bool),
	progressCb func(int),
) (succeeded, failed []string) {
	numItems := len(items)
	if numItems == 0 {
		return nil, nil
	}

	for i, item := range items {
		path, success := processFn(item)
		if success {
			succeeded = append(succeeded, path)
		} else {
			failed = append(failed, path)
		}
		if progressCb != nil {
			progressCb(i + 1)
		}
	}

	return succeeded, failed
}

// ApplyChanges updates Neovim buffers with the provided file contents.
func (m *Manager) ApplyChanges(changes []model.FileChange, progressCb func(int)) (updated, failed []string) {
	processFn := func(change model.FileChange) (string, bool) {
		return change.Path, m.updateBuffer(change.Path, change.Content)
	}
	return processSequentially(changes, processFn, progressCb)
}

func (m *Manager) updateBuffer(filePath string, content []string) bool {
	absPath, err := filepath.Abs(filePath)
	if err != nil {
		return false
	}

	byteContent := make([][]byte, len(content))
	for i, s := range content {
		byteContent[i] = []byte(s)
	}

	b := m.nvim.NewBatch()
	b.Command(fmt.Sprintf("edit %s", absPath))
	b.SetBufferLines(0, 0, -1, true, byteContent)

	if err := b.Execute(); err != nil {
		return false
	}
	return true
}

// SaveAllBuffers writes all modified buffers to disk.
func (m *Manager) SaveAllBuffers() {
	if err := m.nvim.Command("wa!"); err != nil {
		// TODO: Propagate this error? For now, it's fire-and-forget.
	}
}

// UndoFiles reverts a set of operations.
func (m *Manager) UndoFiles(ops []state.Operation, stateDir string, progressCb func(int)) (undone, failed []string) {
	processFn := func(op state.Operation) (string, bool) {
		return op.Path, m.undoFile(op, stateDir)
	}
	return processSequentially(ops, processFn, progressCb)
}

func (m *Manager) undoFile(op state.Operation, stateDir string) bool {
	if op.Action == "delete" {
		trashPath := filepath.Join(stateDir, state.TrashDir)
		wd, _ := os.Getwd()
		if err := fs.RestoreFileFromTrash(op.Path, trashPath, wd); err != nil {
			return false
		}
		// Safety check: after restoring, does hash match?
		restoredHash, err := fs.GetFileSHA256(op.Path)
		if err != nil || restoredHash != op.ContentHash {
			// Something is wrong. Maybe move it back to trash? For now, fail.
			os.Remove(op.Path) // cleanup
			return false
		}
		return true
	}

	if op.Action == "rename" {
		// Undo rename is renaming NewPath back to OldPath (op.Path)
		currentHash, err := fs.GetFileSHA256(op.NewPath)
		if err != nil || currentHash != op.ContentHash {
			return false
		}
		if _, err := os.Stat(op.Path); !os.IsNotExist(err) {
			// Don't overwrite an existing file at the original path.
			return false
		}
		return os.Rename(op.NewPath, op.Path) == nil
	}

	currentHash, err := fs.GetFileSHA256(op.Path)
	if err != nil {
		if os.IsNotExist(err) {
			// If the file doesn't exist, the undo of a 'create' is successful.
			return true
		}
		return false
	}

	// Core safety check: if the file has been changed, abort the undo for this file.
	if currentHash != op.ContentHash {
		return false
	}

	// If hashes match, it's safe to proceed.
	if op.Action == "create" {
		if err := os.Remove(op.Path); err != nil {
			return false
		}

		// Attempt to remove parent directory if it's empty
		parentDir := filepath.Dir(op.Path)
		if isEmpty, _ := fs.IsEmpty(parentDir); isEmpty {
			if err := os.Remove(parentDir); err == nil {
				// Successfully removed empty parent, no need to log.
			}
		}
		return true
	}

	// This is for "modify" action, since "create" is handled above.
	absPath, _ := filepath.Abs(op.Path)
	b := m.nvim.NewBatch()
	b.Command(fmt.Sprintf("edit! %s", absPath))
	b.Command("undo")
	b.Command("write")
	if err := b.Execute(); err != nil {
		return false
	}

	return true
}

// RedoFiles redoes a set of operations.
func (m *Manager) RedoFiles(ops []state.Operation, stateDir string, progressCb func(int)) (redone, failed []string) {
	processFn := func(op state.Operation) (string, bool) {
		switch op.Action {
		case "delete":
			return op.Path, m.redoDelete(op, stateDir)
		case "create", "modify":
			return op.Path, m.redoFile(op.Path)
		case "rename":
			return op.Path, m.redoRename(op)
		default:
			return op.Path, false
		}
	}
	return processSequentially(ops, processFn, progressCb)
}

func (m *Manager) redoRename(op state.Operation) bool {
	// Redo rename is renaming OldPath (op.Path) to NewPath
	currentHash, err := fs.GetFileSHA256(op.Path)
	if err != nil || currentHash != op.ContentHash {
		return false
	}
	if _, err := os.Stat(op.NewPath); !os.IsNotExist(err) {
		// Don't overwrite an existing file at the new path.
		return false
	}

	return os.Rename(op.Path, op.NewPath) == nil
}

func (m *Manager) redoDelete(op state.Operation, stateDir string) bool {
	// Safety check: does the file on disk match the hash we have?
	currentHash, err := fs.GetFileSHA256(op.Path)
	if err != nil || currentHash != op.ContentHash {
		// File is not what we expect. Don't touch it.
		return false
	}

	trashPath := filepath.Join(stateDir, state.TrashDir)
	wd, _ := os.Getwd()
	return fs.TrashFile(op.Path, trashPath, wd) == nil
}

func (m *Manager) redoFile(filePath string) bool {
	absPath, _ := filepath.Abs(filePath)
	b := m.nvim.NewBatch()
	b.Command(fmt.Sprintf("edit! %s", absPath))
	b.Command("redo")
	b.Command("write")
	if err := b.Execute(); err != nil {
		return false
	}
	return true
}
