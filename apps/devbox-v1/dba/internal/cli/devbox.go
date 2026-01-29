// internal/cli/devbox.go
package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/dba-cli/dba/internal/devbox"
)

// DevboxAddResult is the result of adding a package
type DevboxAddResult struct {
	Package    string `json:"package"`
	Success    bool   `json:"success"`
	Verified   bool   `json:"verified"`
	DurationMs int64  `json:"duration_ms"`
}

// DevboxSearchResult is the result of searching packages
type DevboxSearchResult struct {
	Query    string              `json:"query"`
	Packages []devbox.PackageInfo `json:"packages"`
	Count    int                 `json:"count"`
}

// DevboxSyncResult is the result of syncing packages
type DevboxSyncResult struct {
	Added      []string `json:"added,omitempty"`
	Removed    []string `json:"removed,omitempty"`
	Synced     bool     `json:"synced"`
	Verified   bool     `json:"verified"`
	DurationMs int64    `json:"duration_ms"`
}

// DevboxListResult is the result of listing packages
type DevboxListResult struct {
	Packages []string `json:"packages"`
	Count    int      `json:"count"`
}

// devboxCmd is the parent command for devbox operations
var devboxCmd = &cobra.Command{
	Use:   "devbox",
	Short: "Manage devbox packages and environment",
	Long: `Manage devbox packages with synchronization and verification.

These commands interact with the devbox package manager to install,
remove, search, and manage development environment packages.`,
}

// devboxAddCmd adds a package
var devboxAddCmd = &cobra.Command{
	Use:   "add <package>",
	Short: "Add a package to the workspace",
	Long: `Add a package to the devbox environment with verification.

This command adds a package and waits for Nix store synchronization
to complete before returning. The package binary is verified to be
available after installation.

Examples:
  dba devbox add nodejs@20
  dba devbox add python@3.11
  dba devbox add --pin pnpm@latest`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		// Check for devbox
		if err := devbox.EnsureDevbox(); err != nil {
			OutputError(err)
			return err
		}

		ctx, err := NewCLIContext()
		if err != nil {
			OutputError(err)
			return err
		}
		defer ctx.Cancel()

		if err := ctx.RequireWorkspace(); err != nil {
			OutputError(err)
			return err
		}

		pin, _ := cmd.Flags().GetBool("pin")
		pkg := args[0]

		pm := devbox.NewPackageManager(ctx.Workspace)
		result, err := pm.Add(ctx.Context, pkg, pin)
		if err != nil {
			OutputError(err)
			return err
		}

		if flagJSON {
			return OutputResult(DevboxAddResult{
				Package:    result.Package,
				Success:    result.Success,
				Verified:   result.Verified,
				DurationMs: result.Duration.Milliseconds(),
			})
		}

		if result.Success {
			fmt.Printf("Added package: %s\n", result.Package)
			if result.Verified {
				fmt.Println("Package verified and ready to use")
			} else if result.VerifyError != "" {
				fmt.Printf("Warning: verification failed: %s\n", result.VerifyError)
			}
		}

		return nil
	},
}

// devboxRemoveCmd removes a package
var devboxRemoveCmd = &cobra.Command{
	Use:   "remove <package>",
	Short: "Remove a package from the workspace",
	Long: `Remove a package from the devbox environment.

Examples:
  dba devbox remove nodejs
  dba devbox remove python`,
	Aliases: []string{"rm"},
	Args:    cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := devbox.EnsureDevbox(); err != nil {
			OutputError(err)
			return err
		}

		ctx, err := NewCLIContext()
		if err != nil {
			OutputError(err)
			return err
		}
		defer ctx.Cancel()

		if err := ctx.RequireWorkspace(); err != nil {
			OutputError(err)
			return err
		}

		pm := devbox.NewPackageManager(ctx.Workspace)
		if err := pm.Remove(ctx.Context, args[0]); err != nil {
			OutputError(err)
			return err
		}

		if flagJSON {
			return OutputResult(map[string]interface{}{
				"removed": args[0],
				"success": true,
			})
		}

		fmt.Printf("Removed package: %s\n", args[0])
		return nil
	},
}

// devboxSearchCmd searches for packages
var devboxSearchCmd = &cobra.Command{
	Use:   "search <query>",
	Short: "Search for packages",
	Long: `Search for available packages in the Nix store.

Examples:
  dba devbox search nodejs
  dba devbox search python
  dba devbox search "database"`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := devbox.EnsureDevbox(); err != nil {
			OutputError(err)
			return err
		}

		ctx, err := NewCLIContext()
		if err != nil {
			OutputError(err)
			return err
		}
		defer ctx.Cancel()

		// Search doesn't require a workspace context
		query := args[0]

		// Use a temporary package manager (doesn't need workspace for search)
		pm := devbox.NewPackageManager(ctx.Workspace)
		packages, err := pm.Search(ctx.Context, query)
		if err != nil {
			OutputError(err)
			return err
		}

		if flagJSON {
			return OutputResult(DevboxSearchResult{
				Query:    query,
				Packages: packages,
				Count:    len(packages),
			})
		}

		if len(packages) == 0 {
			fmt.Printf("No packages found matching '%s'\n", query)
			return nil
		}

		fmt.Printf("Found %d packages matching '%s':\n\n", len(packages), query)
		for _, pkg := range packages {
			if pkg.Version != "" {
				fmt.Printf("  %s (%s)\n", pkg.Name, pkg.Version)
			} else {
				fmt.Printf("  %s\n", pkg.Name)
			}
			if pkg.Description != "" {
				fmt.Printf("    %s\n", pkg.Description)
			}
		}

		return nil
	},
}

// devboxSyncCmd synchronizes packages
var devboxSyncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Synchronize packages with Nix store",
	Long: `Synchronize all packages with the Nix store.

This command runs 'devbox install' and verifies all package binaries
are available before returning. Use this after pulling changes that
modified devbox.json.

Examples:
  dba devbox sync`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := devbox.EnsureDevbox(); err != nil {
			OutputError(err)
			return err
		}

		ctx, err := NewCLIContext()
		if err != nil {
			OutputError(err)
			return err
		}
		defer ctx.Cancel()

		if err := ctx.RequireWorkspace(); err != nil {
			OutputError(err)
			return err
		}

		pm := devbox.NewPackageManager(ctx.Workspace)
		result, err := pm.SyncPackages(ctx.Context)
		if err != nil {
			OutputError(err)
			return err
		}

		if flagJSON {
			return OutputResult(DevboxSyncResult{
				Added:      result.Added,
				Removed:    result.Removed,
				Synced:     result.Synced,
				Verified:   result.Verified,
				DurationMs: result.Duration.Milliseconds(),
			})
		}

		fmt.Println("Packages synchronized successfully")
		if len(result.Added) > 0 {
			fmt.Printf("  Added: %s\n", strings.Join(result.Added, ", "))
		}
		if len(result.Removed) > 0 {
			fmt.Printf("  Removed: %s\n", strings.Join(result.Removed, ", "))
		}
		if result.Verified {
			fmt.Println("  All packages verified")
		}
		fmt.Printf("  Duration: %dms\n", result.Duration.Milliseconds())

		return nil
	},
}

// devboxListCmd lists installed packages
var devboxListCmd = &cobra.Command{
	Use:   "list",
	Short: "List installed packages",
	Long: `List all packages installed in the workspace.

Examples:
  dba devbox list
  dba devbox list --json`,
	Aliases: []string{"ls"},
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := devbox.EnsureDevbox(); err != nil {
			OutputError(err)
			return err
		}

		ctx, err := NewCLIContext()
		if err != nil {
			OutputError(err)
			return err
		}
		defer ctx.Cancel()

		if err := ctx.RequireWorkspace(); err != nil {
			OutputError(err)
			return err
		}

		pm := devbox.NewPackageManager(ctx.Workspace)
		packages, err := pm.GetInstalledPackages()
		if err != nil {
			OutputError(err)
			return err
		}

		if flagJSON {
			return OutputResult(DevboxListResult{
				Packages: packages,
				Count:    len(packages),
			})
		}

		if len(packages) == 0 {
			fmt.Println("No packages installed")
			return nil
		}

		fmt.Printf("Installed packages (%d):\n", len(packages))
		for _, pkg := range packages {
			fmt.Printf("  %s\n", pkg)
		}

		return nil
	},
}

// devboxUpdateCmd updates packages
var devboxUpdateCmd = &cobra.Command{
	Use:   "update [package]",
	Short: "Update packages to latest versions",
	Long: `Update packages to their latest versions.

If a package name is provided, only that package is updated.
Without a package name, all packages are updated.

Examples:
  dba devbox update           # Update all packages
  dba devbox update nodejs    # Update only nodejs`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := devbox.EnsureDevbox(); err != nil {
			OutputError(err)
			return err
		}

		ctx, err := NewCLIContext()
		if err != nil {
			OutputError(err)
			return err
		}
		defer ctx.Cancel()

		if err := ctx.RequireWorkspace(); err != nil {
			OutputError(err)
			return err
		}

		pm := devbox.NewPackageManager(ctx.Workspace)

		if len(args) > 0 {
			// Update specific package
			if err := pm.Update(ctx.Context, args[0]); err != nil {
				OutputError(err)
				return err
			}

			if flagJSON {
				return OutputResult(map[string]interface{}{
					"updated": args[0],
					"success": true,
				})
			}

			fmt.Printf("Updated package: %s\n", args[0])
		} else {
			// Update all packages
			result, err := pm.SyncPackages(ctx.Context)
			if err != nil {
				OutputError(err)
				return err
			}

			if flagJSON {
				return OutputResult(DevboxSyncResult{
					Added:      result.Added,
					Removed:    result.Removed,
					Synced:     result.Synced,
					Verified:   result.Verified,
					DurationMs: result.Duration.Milliseconds(),
				})
			}

			fmt.Println("All packages updated")
		}

		return nil
	},
}

// devboxValidateCmd validates the devbox configuration
var devboxValidateCmd = &cobra.Command{
	Use:   "validate",
	Short: "Validate devbox configuration",
	Long: `Validate the devbox.json and devbox.lock files.

This command checks that the configuration files are valid
and consistent with each other.

Examples:
  dba devbox validate`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := devbox.EnsureDevbox(); err != nil {
			OutputError(err)
			return err
		}

		ctx, err := NewCLIContext()
		if err != nil {
			OutputError(err)
			return err
		}
		defer ctx.Cancel()

		if err := ctx.RequireWorkspace(); err != nil {
			OutputError(err)
			return err
		}

		pm := devbox.NewPackageManager(ctx.Workspace)
		if err := pm.ValidateLockFile(); err != nil {
			OutputError(err)
			return err
		}

		if flagJSON {
			return OutputResult(map[string]interface{}{
				"valid":   true,
				"message": "Configuration is valid",
			})
		}

		fmt.Println("Devbox configuration is valid")
		return nil
	},
}

func init() {
	// Add flags
	devboxAddCmd.Flags().Bool("pin", false, "Pin package to current version")

	// Add subcommands
	devboxCmd.AddCommand(devboxAddCmd)
	devboxCmd.AddCommand(devboxRemoveCmd)
	devboxCmd.AddCommand(devboxSearchCmd)
	devboxCmd.AddCommand(devboxSyncCmd)
	devboxCmd.AddCommand(devboxListCmd)
	devboxCmd.AddCommand(devboxUpdateCmd)
	devboxCmd.AddCommand(devboxValidateCmd)

	// Register with root
	rootCmd.AddCommand(devboxCmd)
}

// TextOutput implementations for human-readable output
func (r DevboxSearchResult) TextOutput() string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Found %d packages matching '%s':\n", r.Count, r.Query))
	for _, pkg := range r.Packages {
		if pkg.Version != "" {
			sb.WriteString(fmt.Sprintf("  %s (%s)\n", pkg.Name, pkg.Version))
		} else {
			sb.WriteString(fmt.Sprintf("  %s\n", pkg.Name))
		}
	}
	return sb.String()
}

func (r DevboxListResult) TextOutput() string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Installed packages (%d):\n", r.Count))
	for _, pkg := range r.Packages {
		sb.WriteString(fmt.Sprintf("  %s\n", pkg))
	}
	return sb.String()
}
