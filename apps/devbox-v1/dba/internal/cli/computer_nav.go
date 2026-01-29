// internal/cli/computer_nav.go
package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

var computerOpenCmd = &cobra.Command{
	Use:   "open <url>",
	Short: "Navigate to a URL",
	Example: `  dba computer open "https://example.com"
  dba computer open "http://localhost:10000"`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx, err := NewCLIContext()
		if err != nil {
			return err
		}
		defer ctx.Cancel()

		if err := ctx.RequireWorkspace(); err != nil {
			return err
		}

		client, err := getBrowserClient(ctx.Workspace, ctx.Config, ctx.Context)
		if err != nil {
			return err
		}

		url := args[0]
		if err := client.Open(ctx.Context, url); err != nil {
			return fmt.Errorf("navigation failed: %w", err)
		}

		fmt.Printf("Navigated to: %s\n", url)
		return nil
	},
}

var computerBackCmd = &cobra.Command{
	Use:   "back",
	Short: "Go back in browser history",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx, err := NewCLIContext()
		if err != nil {
			return err
		}
		defer ctx.Cancel()

		if err := ctx.RequireWorkspace(); err != nil {
			return err
		}

		client, err := getBrowserClient(ctx.Workspace, ctx.Config, ctx.Context)
		if err != nil {
			return err
		}

		if err := client.Back(ctx.Context); err != nil {
			return fmt.Errorf("back failed: %w", err)
		}

		fmt.Println("Navigated back")
		return nil
	},
}

var computerForwardCmd = &cobra.Command{
	Use:   "forward",
	Short: "Go forward in browser history",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx, err := NewCLIContext()
		if err != nil {
			return err
		}
		defer ctx.Cancel()

		if err := ctx.RequireWorkspace(); err != nil {
			return err
		}

		client, err := getBrowserClient(ctx.Workspace, ctx.Config, ctx.Context)
		if err != nil {
			return err
		}

		if err := client.Forward(ctx.Context); err != nil {
			return fmt.Errorf("forward failed: %w", err)
		}

		fmt.Println("Navigated forward")
		return nil
	},
}

var computerReloadCmd = &cobra.Command{
	Use:   "reload",
	Short: "Reload the current page",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx, err := NewCLIContext()
		if err != nil {
			return err
		}
		defer ctx.Cancel()

		if err := ctx.RequireWorkspace(); err != nil {
			return err
		}

		client, err := getBrowserClient(ctx.Workspace, ctx.Config, ctx.Context)
		if err != nil {
			return err
		}

		if err := client.Reload(ctx.Context); err != nil {
			return fmt.Errorf("reload failed: %w", err)
		}

		fmt.Println("Page reloaded")
		return nil
	},
}

func init() {
	computerCmd.AddCommand(computerOpenCmd)
	computerCmd.AddCommand(computerBackCmd)
	computerCmd.AddCommand(computerForwardCmd)
	computerCmd.AddCommand(computerReloadCmd)
}
