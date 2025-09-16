package state

import (
	"fmt"
	"github.com/sokinpui/itf/internal/fs"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

const stateFileName = ".state.itf"

// Operation represents a single file operation (create or modify).
type Operation struct {
	Path        string
	Action      string
	ContentHash string // SHA256 hash of the file content after operation
}

// HistoryEntry represents one complete run of the tool.
type HistoryEntry struct {
	Timestamp  int64
	Operations []Operation
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
		m.state = &State{CurrentIndex: -1, History: []HistoryEntry{}}
	}
	return m, nil
}

func (m *Manager) load() error {
	data, err := os.ReadFile(m.statePath)
	if err != nil {
		if os.IsNotExist(err) {
			m.state = &State{CurrentIndex: -1, History: []HistoryEntry{}}
			return nil
		}
		return err
	}

	content := string(data)
	// Normalize line endings to LF
	content = strings.ReplaceAll(content, "\r\n", "\n")
	blocks := strings.Split(content, "\n\n")

	if len(blocks) == 0 || blocks[0] == "" {
		m.state = &State{CurrentIndex: -1, History: []HistoryEntry{}}
		return nil
	}

	// First block is current index
	index, err := strconv.Atoi(strings.TrimSpace(blocks[0]))
	if err != nil {
		return fmt.Errorf("invalid state file: could not parse current index: %w", err)
	}

	m.state = &State{CurrentIndex: index, History: []HistoryEntry{}}

	if len(blocks) < 2 {
		return nil // Only index, no history
	}

	historyBlocks := blocks[1:]
	for _, block := range historyBlocks {
		block = strings.TrimSpace(block)
		if block == "" {
			continue
		}
		lines := strings.Split(block, "\n")
		if len(lines) == 0 {
			continue
		}

		ts, err := strconv.ParseInt(lines[0], 10, 64)
		if err != nil {
			return fmt.Errorf("invalid state file: could not parse timestamp from '%s': %w", lines[0], err)
		}

		entry := HistoryEntry{Timestamp: ts}
		opLines := lines[1:]
		if len(opLines)%3 != 0 {
			return fmt.Errorf("invalid state file: incomplete operation record")
		}

		for i := 0; i < len(opLines); i += 3 {
			op := Operation{
				Action:      opLines[i],
				Path:        opLines[i+1],
				ContentHash: opLines[i+2],
			}
			entry.Operations = append(entry.Operations, op)
		}
		m.state.History = append(m.state.History, entry)
	}

	return nil
}

func (m *Manager) save() {
	var blocks []string

	// Current index block
	blocks = append(blocks, fmt.Sprintf("%d", m.state.CurrentIndex))

	// History entry blocks
	for _, entry := range m.state.History {
		var entryBuilder strings.Builder
		entryBuilder.WriteString(fmt.Sprintf("%d\n", entry.Timestamp))

		opLines := []string{}
		for _, op := range entry.Operations {
			opLines = append(opLines, op.Action)
			opLines = append(opLines, op.Path)
			opLines = append(opLines, op.ContentHash)
		}
		entryBuilder.WriteString(strings.Join(opLines, "\n"))
		blocks = append(blocks, entryBuilder.String())
	}

	content := strings.Join(blocks, "\n\n")

	if err := os.WriteFile(m.statePath, []byte(content), 0644); err != nil {
		// TODO: Propagate this error. For now, it fails silently.
	}
}

// Write adds a new set of operations to the history.
func (m *Manager) Write(operations []Operation) {
	if m.state.CurrentIndex < len(m.state.History)-1 {
		m.state.History = m.state.History[:m.state.CurrentIndex+1]
	}

	newEntry := HistoryEntry{
		Timestamp:  time.Now().UTC().Unix(),
		Operations: operations,
	}
	m.state.History = append(m.state.History, newEntry)
	m.state.CurrentIndex++
	m.save()
}

// GetOperationsToRevert gets the last operations and moves the history pointer.
func (m *Manager) GetOperationsToRevert() []Operation {
	if m.state.CurrentIndex < 0 {
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
		hash, err := fs.GetFileSHA256(f)
		if err != nil {
			// If hashing fails, the hash will be empty, revert will likely fail the check.
		}
		ops[i] = Operation{
			Path:        f,
			Action:      fileActions[f],
			ContentHash: hash,
		}
	}
	sort.Slice(ops, func(i, j int) bool {
		return ops[i].Path < ops[j].Path
	})
	return ops
}
