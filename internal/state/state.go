package state

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/sokinpui/itf.go/internal/fs"
	"github.com/sokinpui/itf.go/model"
)

const (
	stateDirName  = ".itf"
	stateFileName = "state.itf"
	TrashDir      = "trash"
)

// Operation represents a single file operation (create or modify).
type Operation struct {
	Path        string
	Action      string
	ContentHash string // SHA256 hash of the file content after operation
	NewPath     string
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
	StateDir  string
}

// findGitRoot finds the root of the git repository.
func findGitRoot() (string, error) {
	cmd := exec.Command("git", "rev-parse", "--show-toplevel")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}

// New creates and loads a state manager.
func New() (*Manager, error) {
	rootDir, err := findGitRoot()
	if err != nil {
		rootDir, err = os.Getwd()
		if err != nil {
			return nil, fmt.Errorf("could not get current working directory: %w", err)
		}
	}

	stateDir := filepath.Join(rootDir, stateDirName)
	if err := os.MkdirAll(stateDir, 0755); err != nil {
		return nil, fmt.Errorf("could not create state directory: %w", err)
	}
	m := &Manager{
		statePath: filepath.Join(stateDir, stateFileName),
		StateDir:  stateDir,
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

		i := 0
		for i < len(opLines) {
			if i+3 > len(opLines) {
				return fmt.Errorf("invalid state file: incomplete operation record")
			}
			action := opLines[i]
			op := Operation{
				Action:      action,
				Path:        opLines[i+1],
				ContentHash: opLines[i+2],
			}
			i += 3
			if action == "rename" {
				if i >= len(opLines) {
					return fmt.Errorf("invalid state file: incomplete rename operation record")
				}
				op.NewPath = opLines[i]
				i++
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
			if op.Action == "rename" {
				opLines = append(opLines, op.NewPath)
			}
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

// GetOperationsToUndo gets the last operations and moves the history pointer.
func (m *Manager) GetOperationsToUndo() []Operation {
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
func (m *Manager) CreateOperations(updatedFiles []string, fileActions map[string]string, renames []model.FileRename) []Operation {
	ops := make([]Operation, 0, len(updatedFiles))
	trashPath := filepath.Join(m.StateDir, TrashDir)
	wd, err := os.Getwd()
	if err != nil {
		// This is unlikely to fail, but if it does, it's a critical error.
		panic(fmt.Sprintf("could not get current working directory: %v", err))
	}
	renameMap := make(map[string]string)
	for _, r := range renames {
		renameMap[r.OldPath] = r.NewPath
	}

	for _, f := range updatedFiles {
		action := fileActions[f]
		var hash, pathForHash, newPath string
		var opErr error

		switch action {
		case "delete":
			relPath, err := filepath.Rel(wd, f)
			if err != nil {
				relPath = filepath.Base(f)
			}
			pathForHash = filepath.Join(trashPath, relPath)
		case "rename":
			newPath = renameMap[f]
			pathForHash = newPath // hash the new file
		default: // create, modify
			pathForHash = f
		}

		hash, opErr = fs.GetFileSHA256(pathForHash)
		if opErr != nil {
			// If hashing fails, the hash will be empty, revert will likely fail the check.
			hash = ""
		}
		ops = append(ops, Operation{
			Path:        f,
			Action:      action,
			ContentHash: hash,
			NewPath:     newPath,
		})
	}
	sort.Slice(ops, func(i, j int) bool {
		return ops[i].Path < ops[j].Path
	})
	return ops
}
