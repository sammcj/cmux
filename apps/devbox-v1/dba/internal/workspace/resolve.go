// internal/workspace/resolve.go
package workspace

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/dba-cli/dba/internal/db"
)

// Workspace struct is defined in workspace.go - this file provides resolution functions

// Resolve resolves a workspace from an ID or path
func Resolve(idOrPath string) (*Workspace, error) {
	// Check if it's a workspace ID (starts with ws_)
	if strings.HasPrefix(idOrPath, "ws_") {
		return ResolveByID(idOrPath)
	}

	// Otherwise treat as path
	return ResolveByPath(idOrPath)
}

// ResolveByID finds a workspace by its ID
func ResolveByID(id string) (*Workspace, error) {
	// First, try to find the workspace in the database
	database, err := db.Get()
	if err == nil {
		var ws Workspace
		var createdAtStr, lastActiveStr string

		err := database.QueryRow(`
			SELECT id, name, path, template, base_port, status, created_at, last_active
			FROM workspaces WHERE id = ?
		`, id).Scan(
			&ws.ID,
			&ws.Name,
			&ws.Path,
			&ws.Template,
			&ws.BasePort,
			&ws.Status,
			&createdAtStr,
			&lastActiveStr,
		)

		if err == nil {
			// Found in database, load additional state
			ws.CreatedAt, _ = time.Parse("2006-01-02 15:04:05", createdAtStr)
			ws.LastActive, _ = time.Parse("2006-01-02 15:04:05", lastActiveStr)
			ws.Ports, _ = loadPortsFromDB(ws.ID)

			// Determine project path
			ws.ProjectPath = ws.Path
			if StateExists(ws.Path) {
				// If state file exists, load full state
				if state, err := LoadState(ws.Path); err == nil {
					ws.Git = state.Git
					ws.Packages = state.Packages
					ws.Morph = state.Morph
					// For created workspaces, project is a subdirectory
					projectPath := filepath.Join(ws.Path, "project")
					if _, err := os.Stat(projectPath); err == nil {
						ws.ProjectPath = projectPath
					}
				}
			}

			return &ws, nil
		}
	}

	// Fallback: try filesystem-based resolution for backwards compatibility
	home := os.Getenv("DBA_HOME")
	if home == "" {
		homeDir, _ := os.UserHomeDir()
		home = filepath.Join(homeDir, ".dba")
	}

	wsPath := filepath.Join(home, "workspaces", id)
	if _, err := os.Stat(wsPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("workspace %s not found", id)
	}

	return loadWorkspace(wsPath)
}

// ResolveByPath finds a workspace from a filesystem path
func ResolveByPath(path string) (*Workspace, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}

	return loadWorkspace(absPath)
}

// ResolveFromCwd tries to find a workspace from current directory
func ResolveFromCwd() (*Workspace, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return nil, err
	}

	// Walk up the directory tree looking for .dba/id file
	dir := cwd
	for {
		idFile := filepath.Join(dir, ".dba", "id")
		if _, err := os.Stat(idFile); err == nil {
			return loadWorkspace(dir)
		}

		// Also check for devbox.json with DBA_WORKSPACE_ID
		devboxFile := filepath.Join(dir, "devbox.json")
		if ws := tryLoadFromDevbox(devboxFile); ws != nil {
			return ws, nil
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}

	return nil, fmt.Errorf("not in a DBA workspace")
}

// loadWorkspace loads workspace state from a directory
func loadWorkspace(path string) (*Workspace, error) {
	// Try loading from state.json if it exists
	if StateExists(path) {
		return Load(path)
	}

	// Fall back to basic loading from .dba/id file
	idFile := filepath.Join(path, ".dba", "id")
	idBytes, err := os.ReadFile(idFile)
	if err != nil {
		return nil, fmt.Errorf("cannot read workspace ID: %w", err)
	}

	id := strings.TrimSpace(string(idBytes))

	// Basic workspace structure for workspaces without full state
	ws := &Workspace{
		ID:          id,
		Path:        path,
		ProjectPath: filepath.Join(path, "project"),
		Ports:       make(map[string]int),
	}

	return ws, nil
}

func tryLoadFromDevbox(path string) *Workspace {
	// Parse devbox.json and check for DBA_WORKSPACE_ID in env
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}

	var devbox struct {
		Env map[string]string `json:"env"`
	}
	if err := json.Unmarshal(data, &devbox); err != nil {
		return nil
	}

	// Check for DBA_WORKSPACE_ID
	wsID, ok := devbox.Env["DBA_WORKSPACE_ID"]
	if !ok || wsID == "" {
		return nil
	}

	// Try to resolve by ID
	ws, err := ResolveByID(wsID)
	if err != nil {
		// If can't resolve by ID, try to load from the devbox.json directory
		dir := filepath.Dir(path)
		wsPath, ok := devbox.Env["DBA_WORKSPACE_PATH"]
		if ok && wsPath != "" && StateExists(wsPath) {
			ws, err = Load(wsPath)
			if err == nil {
				return ws
			}
		}
		// Fallback: try loading from current directory
		if StateExists(dir) {
			ws, err = Load(dir)
			if err == nil {
				return ws
			}
		}
		return nil
	}

	return ws
}
