// internal/cli/workspace.go
package cli

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/dba-cli/dba/internal/port"
	"github.com/dba-cli/dba/internal/workspace"
	"github.com/spf13/cobra"
)

// ═══════════════════════════════════════════════════════════════════════════════
// dba create
// ═══════════════════════════════════════════════════════════════════════════════

var createCmd = &cobra.Command{
	Use:   "create [name]",
	Short: "Create a new workspace",
	Long: `Create a new isolated development workspace.

Examples:
  dba create my-app --template=nextjs
  dba create api-service --template=node --clone=https://github.com/org/repo
  dba create ml-project --template=python --packages=pytorch,numpy`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := ""
		if len(args) > 0 {
			name = args[0]
		}
		template, _ := cmd.Flags().GetString("template")
		clone, _ := cmd.Flags().GetString("clone")
		branch, _ := cmd.Flags().GetString("branch")
		packages, _ := cmd.Flags().GetString("packages")
		ports, _ := cmd.Flags().GetString("ports")
		dir, _ := cmd.Flags().GetString("dir")

		opts := workspace.CreateOptions{
			Name:     name,
			Template: template,
			Clone:    clone,
			Branch:   branch,
			Dir:      dir,
		}

		if packages != "" {
			opts.Packages = strings.Split(packages, ",")
		}
		if ports != "" {
			opts.PortNames = strings.Split(ports, ",")
		}

		ws, err := workspace.Create(cfg, opts)
		if err != nil {
			return err
		}

		result := map[string]interface{}{
			"id":           ws.ID,
			"name":         ws.Name,
			"path":         ws.Path,
			"project_path": ws.ProjectPath,
			"status":       ws.Status,
			"template":     ws.Template,
			"base_port":    ws.BasePort,
			"ports":        ws.Ports,
			"packages":     ws.Packages,
			"urls":         ws.URLs(),
			"created_at":   ws.CreatedAt.Format(time.RFC3339),
		}

		if ws.Git != nil {
			result["git"] = ws.Git
		}

		return OutputResult(result)
	},
}

// ═══════════════════════════════════════════════════════════════════════════════
// dba init
// ═══════════════════════════════════════════════════════════════════════════════

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize workspace in current directory",
	Long: `Initialize a DBA workspace in the current directory.

This is useful for adding DBA management to an existing project.

Examples:
  cd my-project && dba init --template=node
  dba init --name=my-api --template=python`,
	RunE: func(cmd *cobra.Command, args []string) error {
		cwd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get current directory: %w", err)
		}

		template, _ := cmd.Flags().GetString("template")
		name, _ := cmd.Flags().GetString("name")
		packages, _ := cmd.Flags().GetString("packages")
		ports, _ := cmd.Flags().GetString("ports")

		opts := workspace.InitOptions{
			Name:     name,
			Template: template,
		}

		if packages != "" {
			opts.Packages = strings.Split(packages, ",")
		}
		if ports != "" {
			opts.PortNames = strings.Split(ports, ",")
		}

		ws, err := workspace.Init(cfg, cwd, opts)
		if err != nil {
			return err
		}

		result := map[string]interface{}{
			"id":           ws.ID,
			"name":         ws.Name,
			"path":         ws.Path,
			"project_path": ws.ProjectPath,
			"status":       ws.Status,
			"template":     ws.Template,
			"base_port":    ws.BasePort,
			"ports":        ws.Ports,
			"urls":         ws.URLs(),
			"created_at":   ws.CreatedAt.Format(time.RFC3339),
		}

		if ws.Git != nil {
			result["git"] = ws.Git
		}

		return OutputResult(result)
	},
}

// ═══════════════════════════════════════════════════════════════════════════════
// dba list
// ═══════════════════════════════════════════════════════════════════════════════

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all workspaces",
	Long: `List all DBA workspaces.

Examples:
  dba list
  dba list --status=running --json`,
	RunE: func(cmd *cobra.Command, args []string) error {
		status, _ := cmd.Flags().GetString("status")

		workspaces, err := workspace.List(cfg, workspace.ListOptions{
			Status: status,
		})
		if err != nil {
			return err
		}

		// Build response
		type WorkspaceInfo struct {
			ID         string         `json:"id"`
			Name       string         `json:"name"`
			Path       string         `json:"path"`
			Status     string         `json:"status"`
			Template   string         `json:"template"`
			Ports      map[string]int `json:"ports"`
			CreatedAt  string         `json:"created_at"`
			LastActive string         `json:"last_active"`
		}

		result := struct {
			Workspaces []WorkspaceInfo `json:"workspaces"`
		}{
			Workspaces: make([]WorkspaceInfo, len(workspaces)),
		}

		for i, ws := range workspaces {
			result.Workspaces[i] = WorkspaceInfo{
				ID:         ws.ID,
				Name:       ws.Name,
				Path:       ws.Path,
				Status:     ws.Status,
				Template:   ws.Template,
				Ports:      ws.Ports,
				CreatedAt:  ws.CreatedAt.Format(time.RFC3339),
				LastActive: ws.LastActive.Format(time.RFC3339),
			}
		}

		return OutputResult(result)
	},
}

// ═══════════════════════════════════════════════════════════════════════════════
// dba status
// ═══════════════════════════════════════════════════════════════════════════════

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Get workspace status",
	Long: `Get detailed status of the current workspace.

Examples:
  dba status
  dba -w ws_a1b2c3d4 status --json`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx, err := NewCLIContext()
		if err != nil {
			return err
		}
		defer ctx.Cancel()

		if err := ctx.RequireWorkspace(); err != nil {
			return err
		}

		ws := ctx.Workspace

		// Build port info with usage status
		type PortInfo struct {
			Number int  `json:"number"`
			InUse  bool `json:"in_use"`
			PID    int  `json:"pid,omitempty"`
		}

		portInfo := make(map[string]PortInfo)
		for name, portNum := range ws.Ports {
			info := PortInfo{Number: portNum}
			info.InUse = !port.IsPortFree(portNum)
			if info.InUse {
				if proc, _ := port.GetPortProcess(portNum); proc != nil {
					info.PID = proc.PID
				}
			}
			portInfo[name] = info
		}

		result := map[string]interface{}{
			"id":           ws.ID,
			"name":         ws.Name,
			"status":       ws.Status,
			"path":         ws.Path,
			"project_path": ws.ProjectPath,
			"template":     ws.Template,
			"ports":        portInfo,
			"packages":     ws.Packages,
			"urls":         ws.URLs(),
			"created_at":   ws.CreatedAt.Format(time.RFC3339),
			"last_active":  ws.LastActive.Format(time.RFC3339),
		}

		if ws.Git != nil {
			result["git"] = ws.Git
		}

		return OutputResult(result)
	},
}

// ═══════════════════════════════════════════════════════════════════════════════
// dba destroy
// ═══════════════════════════════════════════════════════════════════════════════

var destroyCmd = &cobra.Command{
	Use:   "destroy [workspace_id]",
	Short: "Destroy a workspace",
	Long: `Destroy a workspace and release all resources.

Examples:
  dba destroy ws_a1b2c3d4
  dba destroy --force
  dba destroy ws_a1b2c3d4 --keep-files`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		force, _ := cmd.Flags().GetBool("force")
		keepFiles, _ := cmd.Flags().GetBool("keep-files")

		// Get workspace
		var ws *workspace.Workspace
		var err error

		if len(args) > 0 {
			ws, err = workspace.Resolve(args[0])
			if err != nil {
				return err
			}
		} else {
			ctx, ctxErr := NewCLIContext()
			if ctxErr != nil {
				return ctxErr
			}
			defer ctx.Cancel()
			if err := ctx.RequireWorkspace(); err != nil {
				return err
			}
			ws = ctx.Workspace
		}

		// Confirm unless --force
		if !force && !flagJSON {
			fmt.Printf("Destroy workspace %s (%s)? [y/N] ", ws.ID, ws.Name)
			var response string
			fmt.Scanln(&response)
			if strings.ToLower(response) != "y" {
				return fmt.Errorf("aborted")
			}
		}

		// Get port info before destroying
		portNumbers := make([]int, 0, len(ws.Ports))
		for _, p := range ws.Ports {
			portNumbers = append(portNumbers, p)
		}

		// Destroy
		if err := workspace.Destroy(cfg, ws, workspace.DestroyOptions{
			KeepFiles: keepFiles,
		}); err != nil {
			return err
		}

		result := map[string]interface{}{
			"destroyed":      ws.ID,
			"ports_released": portNumbers,
		}

		if !keepFiles {
			result["path_deleted"] = ws.Path
		}

		return OutputResult(result)
	},
}

// ═══════════════════════════════════════════════════════════════════════════════
// dba clone
// ═══════════════════════════════════════════════════════════════════════════════

var cloneCmd = &cobra.Command{
	Use:   "clone <workspace_id> [new_name]",
	Short: "Clone a workspace",
	Long: `Clone an existing workspace with new port allocations.

The clone will copy project files but generate new devbox.json and process-compose.yaml
with fresh port allocations.

Examples:
  dba clone ws_a1b2c3d4
  dba clone ws_a1b2c3d4 my-clone
  dba clone ws_a1b2c3d4 --name=my-clone`,
	Args: cobra.RangeArgs(1, 2),
	RunE: func(cmd *cobra.Command, args []string) error {
		sourceID := args[0]
		name, _ := cmd.Flags().GetString("name")

		// If name provided as positional arg
		if len(args) > 1 && name == "" {
			name = args[1]
		}

		// Resolve source workspace
		sourceWs, err := workspace.Resolve(sourceID)
		if err != nil {
			return fmt.Errorf("source workspace not found: %w", err)
		}

		opts := workspace.CloneOptions{
			Name: name,
		}

		ws, err := workspace.Clone(cfg, sourceWs, opts)
		if err != nil {
			return err
		}

		result := map[string]interface{}{
			"id":           ws.ID,
			"name":         ws.Name,
			"path":         ws.Path,
			"project_path": ws.ProjectPath,
			"status":       ws.Status,
			"template":     ws.Template,
			"base_port":    ws.BasePort,
			"ports":        ws.Ports,
			"urls":         ws.URLs(),
			"created_at":   ws.CreatedAt.Format(time.RFC3339),
			"cloned_from":  sourceID,
		}

		return OutputResult(result)
	},
}

func init() {
	// create flags
	createCmd.Flags().StringP("template", "t", "", "Template (node, nextjs, python, go, react, rust)")
	createCmd.Flags().StringP("clone", "c", "", "Git URL to clone")
	createCmd.Flags().StringP("branch", "b", "main", "Git branch")
	createCmd.Flags().StringP("packages", "p", "", "Additional packages (comma-separated)")
	createCmd.Flags().String("ports", "", "Port names (comma-separated)")
	createCmd.Flags().String("dir", "", "Parent directory")

	// init flags
	initCmd.Flags().StringP("template", "t", "", "Template (node, nextjs, python, go, react, rust)")
	initCmd.Flags().StringP("name", "n", "", "Workspace name")
	initCmd.Flags().StringP("packages", "p", "", "Additional packages (comma-separated)")
	initCmd.Flags().String("ports", "", "Port names (comma-separated)")

	// list flags
	listCmd.Flags().String("status", "", "Filter by status")

	// destroy flags
	destroyCmd.Flags().BoolP("force", "f", false, "Skip confirmation")
	destroyCmd.Flags().Bool("keep-files", false, "Don't delete files")

	// clone flags
	cloneCmd.Flags().StringP("name", "n", "", "Name for the cloned workspace")

	// Register commands
	rootCmd.AddCommand(createCmd)
	rootCmd.AddCommand(initCmd)
	rootCmd.AddCommand(listCmd)
	rootCmd.AddCommand(statusCmd)
	rootCmd.AddCommand(destroyCmd)
	rootCmd.AddCommand(cloneCmd)
}
