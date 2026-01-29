// internal/workspace/list.go
package workspace

import (
	"path/filepath"
	"time"

	"github.com/dba-cli/dba/internal/config"
	"github.com/dba-cli/dba/internal/db"
)

// ListOptions are options for listing workspaces
type ListOptions struct {
	Status string // Filter by status
}

// List returns all workspaces by querying the database
func List(cfg *config.Config, opts ListOptions) ([]*Workspace, error) {
	database, err := db.Get()
	if err != nil {
		return nil, err
	}

	// Build query with optional status filter
	query := `
		SELECT w.id, w.name, w.path, w.template, w.base_port, w.status, w.created_at, w.last_active
		FROM workspaces w
	`
	args := []interface{}{}

	if opts.Status != "" {
		query += " WHERE w.status = ?"
		args = append(args, opts.Status)
	}

	query += " ORDER BY w.created_at DESC"

	rows, err := database.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var workspaces []*Workspace

	for rows.Next() {
		var ws Workspace
		var createdAtStr, lastActiveStr string

		err := rows.Scan(
			&ws.ID,
			&ws.Name,
			&ws.Path,
			&ws.Template,
			&ws.BasePort,
			&ws.Status,
			&createdAtStr,
			&lastActiveStr,
		)
		if err != nil {
			continue // Skip rows with scan errors
		}

		// Parse timestamps
		ws.CreatedAt, _ = time.Parse("2006-01-02 15:04:05", createdAtStr)
		ws.LastActive, _ = time.Parse("2006-01-02 15:04:05", lastActiveStr)

		// Set project path
		ws.ProjectPath = filepath.Join(ws.Path, "project")
		// For init workspaces, project path is the workspace path itself
		if !StateExists(ws.Path) || ws.Path == ws.ProjectPath {
			ws.ProjectPath = ws.Path
		}

		// Load ports from database
		ws.Ports, _ = loadPortsFromDB(ws.ID)

		// Try to load full state from filesystem for additional data
		if StateExists(ws.Path) {
			if state, err := LoadState(ws.Path); err == nil {
				ws.Git = state.Git
				ws.Packages = state.Packages
				// Use filesystem state for more accurate project path
				if state.ID == ws.ID {
					ws.ProjectPath = filepath.Join(ws.Path, "project")
				}
			}
		}

		workspaces = append(workspaces, &ws)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return workspaces, nil
}

// loadPortsFromDB loads port allocations for a workspace from the database
func loadPortsFromDB(workspaceID string) (map[string]int, error) {
	database, err := db.Get()
	if err != nil {
		return nil, err
	}

	rows, err := database.Query(`
		SELECT port_name, port_number FROM port_allocations WHERE workspace_id = ?
	`, workspaceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	ports := make(map[string]int)
	for rows.Next() {
		var name string
		var port int
		if err := rows.Scan(&name, &port); err != nil {
			continue
		}
		ports[name] = port
	}

	return ports, rows.Err()
}

// Count returns the number of workspaces
func Count(cfg *config.Config) (int, error) {
	workspaces, err := List(cfg, ListOptions{})
	if err != nil {
		return 0, err
	}
	return len(workspaces), nil
}

// GetByID returns a workspace by ID
func GetByID(cfg *config.Config, id string) (*Workspace, error) {
	return ResolveByID(id)
}
