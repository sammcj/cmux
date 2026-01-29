// internal/workspace/state.go
package workspace

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

// SaveState persists the workspace state to disk
func (w *Workspace) SaveState() error {
	state := State{
		ID:         w.ID,
		Name:       w.Name,
		Template:   w.Template,
		Status:     w.Status,
		BasePort:   w.BasePort,
		Ports:      w.Ports,
		Packages:   w.Packages,
		CreatedAt:  w.CreatedAt,
		LastActive: w.LastActive,
		Git:        w.Git,
		Morph:      w.Morph,
	}

	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}

	// Ensure the state directory exists
	if err := os.MkdirAll(w.StateDir(), 0755); err != nil {
		return err
	}

	return os.WriteFile(w.StatePath(), data, 0644)
}

// LoadState loads workspace state from disk
func LoadState(wsPath string) (*State, error) {
	statePath := filepath.Join(wsPath, ".dba", "state.json")

	data, err := os.ReadFile(statePath)
	if err != nil {
		return nil, err
	}

	var state State
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, err
	}

	return &state, nil
}

// Load loads a workspace from a path
func Load(wsPath string) (*Workspace, error) {
	state, err := LoadState(wsPath)
	if err != nil {
		return nil, err
	}

	return &Workspace{
		ID:          state.ID,
		Name:        state.Name,
		Path:        wsPath,
		ProjectPath: filepath.Join(wsPath, "project"),
		Template:    state.Template,
		Status:      state.Status,
		BasePort:    state.BasePort,
		Ports:       state.Ports,
		Packages:    state.Packages,
		CreatedAt:   state.CreatedAt,
		LastActive:  state.LastActive,
		Git:         state.Git,
		Morph:       state.Morph,
	}, nil
}

// UpdateLastActive updates the last active timestamp
func (w *Workspace) UpdateLastActive() error {
	w.LastActive = time.Now()
	return w.SaveState()
}

// SetStatus updates the workspace status
func (w *Workspace) SetStatus(status string) error {
	w.Status = status
	return w.SaveState()
}

// StateExists checks if a workspace state file exists at the given path
func StateExists(wsPath string) bool {
	statePath := filepath.Join(wsPath, ".dba", "state.json")
	_, err := os.Stat(statePath)
	return err == nil
}
