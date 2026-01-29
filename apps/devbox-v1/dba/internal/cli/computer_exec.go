// internal/cli/computer_exec.go
package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/dba-cli/dba/internal/morph"
)

var computerExecCmd = &cobra.Command{
	Use:   "exec <command>",
	Short: "Execute a command on the Morph VM",
	Long:  `Run a shell command on the Morph VM and display the output.`,
	Example: `  dba computer exec "ls -la"
  dba computer exec "systemctl status chrome-cdp"
  dba computer exec "docker ps"`,
	Args: cobra.MinimumNArgs(1),
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

		// Create manager
		manager, err := morph.NewManager(morph.ManagerConfig{
			APIKey:         ctx.Config.GetMorphAPIKey(),
			BaseSnapshotID: ctx.Config.GetBaseSnapshotID(),
		})
		if err != nil {
			return fmt.Errorf("failed to create Morph manager: %w", err)
		}

		// Set the instance in the manager
		manager.SetInstance(ws.ID, &morph.Instance{
			ID:     ws.Morph.InstanceID,
			Status: morph.StatusRunning,
		})

		// Join args into a single command
		command := strings.Join(args, " ")

		// Execute command
		result, err := manager.Exec(ctx.Context, ws.ID, command)
		if err != nil {
			return fmt.Errorf("exec failed: %w", err)
		}

		// Print output
		if result.Stdout != "" {
			fmt.Print(result.Stdout)
		}
		if result.Stderr != "" {
			fmt.Fprint(cmd.ErrOrStderr(), result.Stderr)
		}

		if result.ExitCode != 0 {
			return fmt.Errorf("command exited with code %d", result.ExitCode)
		}

		return nil
	},
}

func init() {
	computerCmd.AddCommand(computerExecCmd)
}
