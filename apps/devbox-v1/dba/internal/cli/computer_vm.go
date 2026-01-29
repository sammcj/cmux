// internal/cli/computer_vm.go
package cli

import (
	"fmt"
	"os/exec"
	"runtime"
	"time"

	"github.com/spf13/cobra"

	"github.com/dba-cli/dba/internal/morph"
)

// computerCmd is the parent command for all computer subcommands
var computerCmd = &cobra.Command{
	Use:   "computer",
	Short: "Browser automation and VM management",
	Long:  `Manage Morph Cloud VMs and automate browser interactions using agent-browser.`,
}

// computerStartCmd starts a Morph VM for the workspace
var computerStartCmd = &cobra.Command{
	Use:   "start",
	Short: "Start the Morph VM for browser automation",
	Long: `Start a Morph Cloud VM with browser, code-server, and VNC.
The VM is created from the base snapshot (or a saved workspace snapshot).`,
	Example: `  dba computer start
  dba computer start --snapshot=snap_abc123
  dba computer start --from=my-saved-state`,
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

		// Check if already running
		if ws.IsMorphRunning() {
			fmt.Println("VM is already running")

			// If we don't have URLs, try to refresh from API
			if ws.Morph.BaseURL == "" || ws.Morph.CodeURL == "" {
				manager, err := morph.NewManager(morph.ManagerConfig{
					APIKey:         ctx.Config.GetMorphAPIKey(),
					BaseSnapshotID: ctx.Config.GetBaseSnapshotID(),
				})
				if err == nil {
					if urls, err := manager.RefreshInstanceURLs(ctx.Context, ws.Morph.InstanceID); err == nil && urls.BaseURL != "" {
						ws.SetMorphInstance(ws.Morph.InstanceID, ws.Morph.SnapshotID, urls.BaseURL)
						_ = ws.SaveState()
					}
				}
			}

			printVMStatus(ws)
			return nil
		}

		// Get snapshot ID
		snapshotID, _ := cmd.Flags().GetString("snapshot")
		if snapshotID == "" {
			// Check for saved snapshot by name
			snapshotName, _ := cmd.Flags().GetString("from")
			if snapshotName != "" {
				saved := ws.GetSavedSnapshot(snapshotName)
				if saved == nil {
					return fmt.Errorf("saved snapshot '%s' not found", snapshotName)
				}
				snapshotID = saved.ID
			}
		}

		// Create manager
		manager, err := morph.NewManager(morph.ManagerConfig{
			APIKey:         ctx.Config.GetMorphAPIKey(),
			BaseSnapshotID: ctx.Config.GetBaseSnapshotID(),
			DefaultTTL:     ctx.Config.Morph.VM.TTLSeconds,
		})
		if err != nil {
			return fmt.Errorf("failed to create Morph manager: %w", err)
		}

		// Start instance
		fmt.Println("Starting Morph VM...")
		instance, err := manager.StartInstance(ctx.Context, ws.ID, snapshotID)
		if err != nil {
			return fmt.Errorf("failed to start VM: %w", err)
		}

		// Update workspace state
		ws.SetMorphInstance(instance.ID, instance.SnapshotID, instance.BaseURL)
		if err := ws.SaveState(); err != nil {
			return fmt.Errorf("failed to save workspace state: %w", err)
		}

		fmt.Println("VM started successfully!")
		printVMStatus(ws)

		return nil
	},
}

// computerStopCmd stops the Morph VM
var computerStopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stop the Morph VM",
	Long:  `Stop the Morph Cloud VM for this workspace.`,
	Example: `  dba computer stop
  dba computer stop --save=my-state`,
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

		if !ws.IsMorphRunning() {
			fmt.Println("VM is not running")
			return nil
		}

		// Check for save flag
		saveName, _ := cmd.Flags().GetString("save")

		manager, err := morph.NewManager(morph.ManagerConfig{
			APIKey:         ctx.Config.GetMorphAPIKey(),
			BaseSnapshotID: ctx.Config.GetBaseSnapshotID(),
		})
		if err != nil {
			return err
		}

		// Restore instance to manager for tracking
		manager.SetInstance(ws.ID, &morph.Instance{
			ID:     ws.Morph.InstanceID,
			Status: morph.StatusRunning,
		})

		// Save snapshot if requested
		if saveName != "" {
			fmt.Printf("Saving snapshot '%s'...\n", saveName)
			snapshot, err := manager.SaveSnapshot(ctx.Context, ws.ID, saveName)
			if err != nil {
				return fmt.Errorf("failed to save snapshot: %w", err)
			}
			ws.AddSavedSnapshot(snapshot.ID, saveName)
			fmt.Printf("Snapshot saved: %s\n", snapshot.ID)
		}

		// Stop instance
		fmt.Println("Stopping VM...")
		if err := manager.StopInstance(ctx.Context, ws.ID); err != nil {
			return fmt.Errorf("failed to stop VM: %w", err)
		}

		ws.ClearMorphInstance()
		if err := ws.SaveState(); err != nil {
			return fmt.Errorf("failed to save workspace state: %w", err)
		}

		fmt.Println("VM stopped")
		return nil
	},
}

// computerStatusCmd shows VM status
var computerStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show VM status and connection URLs",
	Long:  `Display the status of the Morph VM and all connection URLs.`,
	Example: `  dba computer status
  dba computer status --json`,
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

		jsonOutput, _ := cmd.Flags().GetBool("json")
		if jsonOutput {
			return OutputJSON(ws.Morph)
		}

		printVMStatus(ws)
		return nil
	},
}

// computerSaveCmd saves VM state as snapshot
var computerSaveCmd = &cobra.Command{
	Use:   "save",
	Short: "Save current VM state as a snapshot",
	Long:  `Save the current state of the VM as a Morph snapshot that can be resumed later.`,
	Example: `  dba computer save
  dba computer save --name=my-checkpoint`,
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

		if !ws.IsMorphRunning() {
			return fmt.Errorf("VM is not running")
		}

		name, _ := cmd.Flags().GetString("name")
		if name == "" {
			name = fmt.Sprintf("%s-%s", ws.Name, time.Now().Format("2006-01-02-150405"))
		}

		manager, err := morph.NewManager(morph.ManagerConfig{
			APIKey:         ctx.Config.GetMorphAPIKey(),
			BaseSnapshotID: ctx.Config.GetBaseSnapshotID(),
		})
		if err != nil {
			return err
		}

		// Restore instance to manager for tracking
		manager.SetInstance(ws.ID, &morph.Instance{
			ID:     ws.Morph.InstanceID,
			Status: morph.StatusRunning,
		})

		fmt.Printf("Saving snapshot '%s'...\n", name)
		snapshot, err := manager.SaveSnapshot(ctx.Context, ws.ID, name)
		if err != nil {
			return fmt.Errorf("failed to save snapshot: %w", err)
		}

		ws.AddSavedSnapshot(snapshot.ID, name)
		if err := ws.SaveState(); err != nil {
			return fmt.Errorf("failed to save workspace state: %w", err)
		}

		fmt.Printf("Snapshot saved!\n")
		fmt.Printf("  ID:   %s\n", snapshot.ID)
		fmt.Printf("  Name: %s\n", name)
		return nil
	},
}

// computerVNCCmd opens VNC in browser
var computerVNCCmd = &cobra.Command{
	Use:   "vnc",
	Short: "Open VNC viewer in browser",
	Long:  `Open the noVNC web viewer in your default browser.`,
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

		if !ws.IsMorphRunning() {
			return fmt.Errorf("VM is not running. Start it with: dba computer start")
		}

		if ws.Morph.VNCURL == "" {
			return fmt.Errorf("VNC URL not available")
		}

		fmt.Printf("Opening VNC: %s\n", ws.Morph.VNCURL)
		return openBrowser(ws.Morph.VNCURL)
	},
}

// Helper functions

func printVMStatus(ws interface {
	IsMorphRunning() bool
	GetMorphURLs() map[string]string
}) {
	// Try to get additional info via type assertion
	type morphStateGetter interface {
		GetMorphState() (instanceID, baseURL, status string)
	}

	// Check if running
	if ws.IsMorphRunning() {
		fmt.Println("Status: running")

		// Try to get instance ID and base URL
		if getter, ok := ws.(morphStateGetter); ok {
			instanceID, baseURL, _ := getter.GetMorphState()
			if instanceID != "" {
				fmt.Printf("Instance: %s\n", instanceID)
			}
			if baseURL != "" {
				fmt.Printf("Base URL: %s\n", baseURL)
			}
		}

		urls := ws.GetMorphURLs()
		if len(urls) > 0 {
			fmt.Println("")
			fmt.Println("Service URLs:")
			if url, ok := urls["code"]; ok {
				fmt.Printf("  VS Code: %s\n", url)
			}
			if url, ok := urls["vnc"]; ok {
				fmt.Printf("  VNC:     %s\n", url)
			}
			if url, ok := urls["app"]; ok {
				fmt.Printf("  App:     %s\n", url)
			}
			if url, ok := urls["cdp"]; ok {
				fmt.Printf("  CDP:     %s\n", url)
			}
		} else {
			fmt.Println("")
			fmt.Println("(No service URLs available yet)")
		}
	} else {
		fmt.Println("Status: stopped")
	}
}

func openBrowser(url string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "linux":
		cmd = exec.Command("xdg-open", url)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	default:
		return fmt.Errorf("unsupported platform")
	}
	return cmd.Start()
}

func init() {
	// Register flags
	computerStartCmd.Flags().StringP("snapshot", "s", "", "Snapshot ID to start from")
	computerStartCmd.Flags().String("from", "", "Name of saved snapshot to start from")

	computerStopCmd.Flags().String("save", "", "Save snapshot with this name before stopping")

	computerSaveCmd.Flags().StringP("name", "n", "", "Name for the snapshot")

	// Note: --json flag is inherited from root command

	// Add subcommands
	computerCmd.AddCommand(computerStartCmd)
	computerCmd.AddCommand(computerStopCmd)
	computerCmd.AddCommand(computerStatusCmd)
	computerCmd.AddCommand(computerSaveCmd)
	computerCmd.AddCommand(computerVNCCmd)

	// Register with root
	rootCmd.AddCommand(computerCmd)
}
