// internal/cli/computer_wait.go
package cli

import (
	"fmt"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/dba-cli/dba/internal/browser"
)

var computerWaitCmd = &cobra.Command{
	Use:   "wait [selector|ms]",
	Short: "Wait for element or fixed time",
	Long: `Wait for an element to appear or for a fixed duration.
  dba computer wait @e1           # Wait for element
  dba computer wait 2000          # Wait 2 seconds
  dba computer wait --text "Done" # Wait for text
  dba computer wait --url "/dashboard" # Wait for URL`,
	Example: `  dba computer wait @e1
  dba computer wait 2000
  dba computer wait --text "Success"
  dba computer wait --url "/login"`,
	Args: cobra.MaximumNArgs(1),
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

		timeout, _ := cmd.Flags().GetInt("timeout")
		text, _ := cmd.Flags().GetString("text")
		url, _ := cmd.Flags().GetString("url")
		ms, _ := cmd.Flags().GetInt("ms")

		// Wait for text
		if text != "" {
			if err := client.WaitForText(ctx.Context, text, timeout); err != nil {
				return fmt.Errorf("wait for text failed: %w", err)
			}
			fmt.Printf("Text found: %s\n", text)
			return nil
		}

		// Wait for URL
		if url != "" {
			if err := client.WaitForURL(ctx.Context, url, timeout); err != nil {
				return fmt.Errorf("wait for URL failed: %w", err)
			}
			fmt.Printf("URL matched: %s\n", url)
			return nil
		}

		// Wait for fixed time
		if ms > 0 {
			if err := client.WaitMs(ctx.Context, ms); err != nil {
				return fmt.Errorf("wait failed: %w", err)
			}
			fmt.Printf("Waited %dms\n", ms)
			return nil
		}

		// Wait for element (from args)
		if len(args) > 0 {
			selector := args[0]
			// Check if it's a number (ms)
			if n, err := strconv.Atoi(selector); err == nil {
				if err := client.WaitMs(ctx.Context, n); err != nil {
					return fmt.Errorf("wait failed: %w", err)
				}
				fmt.Printf("Waited %dms\n", n)
				return nil
			}

			// It's a selector
			if err := client.Wait(ctx.Context, selector, browser.WaitOptions{Timeout: timeout}); err != nil {
				return fmt.Errorf("wait for element failed: %w", err)
			}
			fmt.Printf("Element found: %s\n", selector)
			return nil
		}

		return fmt.Errorf("specify a selector, --ms, --text, or --url")
	},
}

func init() {
	computerWaitCmd.Flags().Int("timeout", 30000, "Timeout in milliseconds")
	computerWaitCmd.Flags().String("text", "", "Wait for text to appear")
	computerWaitCmd.Flags().String("url", "", "Wait for URL to match pattern")
	computerWaitCmd.Flags().Int("ms", 0, "Wait for fixed milliseconds")

	computerCmd.AddCommand(computerWaitCmd)
}
