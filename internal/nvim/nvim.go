package nvim

import (
	"fmt"
	"itf/internal/model"
	"itf/internal/state"
	"itf/internal/ui"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/neovim/go-client/nvim"
)

const undoDir = "~/.local/state/nvim/undo/"

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
			ui.Info("-> Connected to running Neovim instance at '%s'", addr)
			return &Manager{nvim: v}, nil
		}
	}

	// If that fails, start a temporary headless instance.
	ui.Info("-> No running Neovim instance found. Starting a temporary one...")
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
	ui.Success("-> Started temporary instance with socket '%s'", socketPath)
	return m, nil
}

// configureTempInstance sets up undofile for persistent history.
func (m *Manager) configureTempInstance() {
	ui.Info("-> Configuring temporary instance for persistent undo...")
	home, _ := os.UserHomeDir()
	expandedUndoDir := strings.Replace(undoDir, "~", home, 1)
	os.MkdirAll(expandedUndoDir, 0755)

	b := m.nvim.NewBatch()
	b.Command("set undofile")
	b.Command(fmt.Sprintf("set undodir=%s", expandedUndoDir))
	b.Command("set noswapfile")
	if err := b.Execute(); err != nil {
		ui.Warning("Failed to configure temp nvim instance: %v", err)
	}
}

// Close disconnects from Neovim and cleans up if it was self-started.
func (m *Manager) Close() {
	if m.nvim != nil {
		m.nvim.Close()
	}
	if m.isSelfStarted && m.cmd != nil && m.cmd.Process != nil {
		ui.Info("-> Closing temporary Neovim instance...")
		if err := m.cmd.Process.Kill(); err == nil {
			m.cmd.Wait()
			os.RemoveAll(filepath.Dir(m.socketPath))
			ui.Success("-> Temporary Neovim instance terminated.")
		}
	}
}

// ApplyChanges updates Neovim buffers with the provided file contents.
func (m *Manager) ApplyChanges(changes []model.FileChange) (updated, failed []string) {
	total := len(changes)
	bar := ui.NewProgressBar(total, "Updating buffers:")
	bar.Start()

	for _, change := range changes {
		if m.updateBuffer(change.Path, change.Content) {
			updated = append(updated, change.Path)
		} else {
			failed = append(failed, change.Path)
		}
		bar.Increment()
	}
	bar.Finish()
	return updated, failed
}

func (m *Manager) updateBuffer(filePath string, content []string) bool {
	absPath, err := filepath.Abs(filePath)
	if err != nil {
		ui.Error("\nInvalid file path '%s': %v", filePath, err)
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
		ui.Error("\nError processing '%s': %v", filePath, err)
		return false
	}
	return true
}

// SaveAllBuffers writes all modified buffers to disk.
func (m *Manager) SaveAllBuffers() {
	ui.Info("\nSaving all modified buffers...")
	if err := m.nvim.Command("wa!"); err != nil {
		ui.Error("Neovim API Error saving buffers: %v", err)
	} else {
		ui.Success("Save complete.")
	}
}

// RevertFiles reverts a set of operations.
func (m *Manager) RevertFiles(ops []state.Operation) (reverted, failed []string) {
	bar := ui.NewProgressBar(len(ops), "Reverting files:")
	bar.Start()
	for _, op := range ops {
		if m.revertFile(op.Path, op.Action) {
			reverted = append(reverted, op.Path)
		} else {
			failed = append(failed, op.Path)
		}
		bar.Increment()
	}
	bar.Finish()
	return reverted, failed
}

func (m *Manager) revertFile(filePath, action string) bool {
	absPath, _ := filepath.Abs(filePath)
	b := m.nvim.NewBatch()
	b.Command(fmt.Sprintf("edit! %s", absPath))
	b.Command("undo")

	if err := b.Execute(); err != nil {
		ui.Error("\nError reverting '%s': %v", filePath, err)
		return false
	}

	if action == "create" {
		lines, err := m.nvim.BufferLines(0, 0, -1, false)
		if err == nil && len(lines) == 0 {
			m.nvim.Command("bwipeout!")
			os.Remove(absPath)
			return true
		}
	}

	if err := m.nvim.Command("write"); err != nil {
		ui.Error("\nError saving reverted file '%s': %v", filePath, err)
		return false
	}
	return true
}

// RedoFiles redoes a set of operations.
func (m *Manager) RedoFiles(ops []state.Operation) (redone, failed []string) {
	bar := ui.NewProgressBar(len(ops), "Redoing files:")
	bar.Start()
	for _, op := range ops {
		if m.redoFile(op.Path) {
			redone = append(redone, op.Path)
		} else {
			failed = append(failed, op.Path)
		}
		bar.Increment()
	}
	bar.Finish()
	return redone, failed
}

func (m *Manager) redoFile(filePath string) bool {
	absPath, _ := filepath.Abs(filePath)
	b := m.nvim.NewBatch()
	b.Command(fmt.Sprintf("edit! %s", absPath))
	b.Command("redo")
	b.Command("write")
	if err := b.Execute(); err != nil {
		ui.Error("\nError redoing '%s': %v", filePath, err)
		return false
	}
	return true
}
