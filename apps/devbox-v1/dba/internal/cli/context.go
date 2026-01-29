// internal/cli/context.go
package cli

import (
	"context"
	"time"

	"github.com/dba-cli/dba/internal/config"
	"github.com/dba-cli/dba/internal/workspace"
)

// CLIContext holds the context for a CLI command
type CLIContext struct {
	Context   context.Context
	Cancel    context.CancelFunc
	Config    *config.Config
	Workspace *workspace.Workspace // May be nil if not in a workspace
}

// NewCLIContext creates a new CLI context
func NewCLIContext() (*CLIContext, error) {
	// Parse timeout
	timeout, err := time.ParseDuration(flagTimeout)
	if err != nil || timeout < 0 {
		// Use default timeout for invalid or negative values
		timeout = 5 * time.Minute
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)

	cliCtx := &CLIContext{
		Context: ctx,
		Cancel:  cancel,
		Config:  cfg,
	}

	// Resolve workspace if flag provided or in workspace directory
	if flagWorkspace != "" {
		ws, err := workspace.Resolve(flagWorkspace)
		if err != nil {
			cancel()
			return nil, err
		}
		cliCtx.Workspace = ws
	} else {
		// Try to detect from current directory
		ws, _ := workspace.ResolveFromCwd()
		cliCtx.Workspace = ws // May be nil, that's OK
	}

	return cliCtx, nil
}

// RequireWorkspace returns error if no workspace is resolved
func (c *CLIContext) RequireWorkspace() error {
	if c.Workspace == nil {
		return NewWorkspaceError("not in a DBA workspace (use --workspace or cd to workspace directory)")
	}
	return nil
}

// GetConfig returns the global configuration
func GetConfig() *config.Config {
	return cfg
}

// IsJSONOutput returns true if JSON output mode is enabled
func IsJSONOutput() bool {
	return flagJSON
}

// IsVerbose returns true if verbose mode is enabled
func IsVerbose() bool {
	return flagVerbose
}

// GetWorkspaceFlag returns the workspace flag value
func GetWorkspaceFlag() string {
	return flagWorkspace
}
