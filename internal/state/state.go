package state

import (
	"encoding/json"
	"itf/internal/ui"
	"os"
	"path/filepath"
	"sort"
	"time"
)

const stateFileName = ".itf_state.json"

// Operation represents a single file operation (create or modify).
type Operation struct {
	Path   string `json:"path"`
	Action string `json:"action"`
}

// HistoryEntry represents one complete run of the tool.
type HistoryEntry struct {
	Timestamp  string      `json:"timestamp"`
	Operations []Operation `json:"operations"`
}

// State represents the entire state file.
type State struct {
	History      []HistoryEntry `json:"history"`
	CurrentIndex int            `json:"current_index"`
}

// Manager handles the lifecycle of the state file.
type Manager struct {
	statePath string
	state     *State
}

// New creates and loads a state manager.
func New() (*Manager, error) {
	m := &Manager{
		statePath: filepath.Join(".", stateFileName),
	}
	if err := m.load(); err != nil {
		ui.Warning("Could not load state file, starting fresh: %v", err)
		m.state = &State{CurrentIndex: -1}
	}
	return m, nil
}

func (m *Manager) load() error {
	data, err := os.ReadFile(m.statePath)
	if err != nil {
		if os.IsNotExist(err) {
			m.state = &State{CurrentIndex: -1}
			return nil
		}
		return err
	}
	if err := json.Unmarshal(data, &m.state); err != nil {
		return err
	}
	return nil
}

func (m *Manager) save() {
	data, err := json.MarshalIndent(m.state, "", "  ")
	if err != nil {
		ui.Error("Failed to serialize state: %v", err)
		return
	}
	if err := os.WriteFile(m.statePath, data, 0644); err != nil {
		ui.Error("Failed to write state file '%s': %v", m.statePath, err)
	}
}

// Write adds a new set of operations to the history.
func (m *Manager) Write(operations []Operation) {
	if m.state.CurrentIndex < len(m.state.History)-1 {
		m.state.History = m.state.History[:m.state.CurrentIndex+1]
	}

	newEntry := HistoryEntry{
		Timestamp:  time.Now().UTC().Format(time.RFC3339),
		Operations: operations,
	}
	m.state.History = append(m.state.History, newEntry)
	m.state.CurrentIndex++
	m.save()
	ui.Info("\nSaved run state for revertability to '%s'", stateFileName)
}

// GetOperationsToRevert gets the last operations and moves the history pointer.
func (m *Manager) GetOperationsToRevert() []Operation {
	if m.state.CurrentIndex < 0 {
		ui.Error("No history found in '%s'. Nothing to revert.", stateFileName)
		return nil
	}
	ops := m.state.History[m.state.CurrentIndex].Operations
	m.state.CurrentIndex--
	m.save()
	return ops
}

// GetOperationsToRedo gets the next operations and moves the history pointer.
func (m *Manager) GetOperationsToRedo() []Operation {
	nextIndex := m.state.CurrentIndex + 1
	if nextIndex >= len(m.state.History) {
		ui.Error("No operations to redo. Already at the latest change.")
		return nil
	}
	m.state.CurrentIndex = nextIndex
	ops := m.state.History[m.state.CurrentIndex].Operations
	m.save()
	return ops
}

// CreateOperations prepares a list of operations from file changes.
func CreateOperations(updatedFiles []string, fileActions map[string]string) []Operation {
	ops := make([]Operation, len(updatedFiles))
	for i, f := range updatedFiles {
		ops[i] = Operation{Path: f, Action: fileActions[f]}
	}
	sort.Slice(ops, func(i, j int) bool {
		return ops[i].Path < ops[j].Path
	})
	return ops
}
