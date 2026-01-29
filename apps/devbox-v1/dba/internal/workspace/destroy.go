// internal/workspace/destroy.go
package workspace

import (
	"os"

	"github.com/dba-cli/dba/internal/config"
	"github.com/dba-cli/dba/internal/daemon"
	"github.com/dba-cli/dba/internal/db"
	"github.com/dba-cli/dba/internal/port"
)

// DestroyOptions are options for destroying a workspace
type DestroyOptions struct {
	KeepFiles bool
}

// Destroy destroys a workspace
func Destroy(cfg *config.Config, ws *Workspace, opts DestroyOptions) error {
	// Unregister from daemon (only if running, don't auto-start for destroy)
	client := daemon.NewClient(cfg)
	if client.IsRunning() {
		client.UnregisterWorkspace(ws.ID)
	}

	// Release ports
	allocator, err := port.NewAllocator(cfg.Ports)
	if err == nil {
		allocator.ReleaseForWorkspace(ws.ID)
	}

	// Remove from database
	if database, err := db.Get(); err == nil {
		database.Exec(`DELETE FROM workspaces WHERE id = ?`, ws.ID)
	}

	// Delete files
	if !opts.KeepFiles {
		if err := os.RemoveAll(ws.Path); err != nil {
			return err
		}
	} else {
		// Just remove the .dba directory if keeping files
		if err := os.RemoveAll(ws.StateDir()); err != nil {
			return err
		}
	}

	return nil
}

// DestroyByID destroys a workspace by ID
func DestroyByID(cfg *config.Config, id string, opts DestroyOptions) error {
	ws, err := ResolveByID(id)
	if err != nil {
		return err
	}

	return Destroy(cfg, ws, opts)
}
