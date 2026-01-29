// internal/cli/computer_info.go
package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/dba-cli/dba/internal/browser"
)

var computerScreenshotCmd = &cobra.Command{
	Use:   "screenshot",
	Short: "Take a screenshot",
	Long:  `Take a screenshot of the browser. Saves to file with --output or prints base64 to stdout.`,
	Example: `  dba computer screenshot
  dba computer screenshot --output=screen.png
  dba computer screenshot --full`,
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

		output, _ := cmd.Flags().GetString("output")
		fullPage, _ := cmd.Flags().GetBool("full")

		result, err := client.Screenshot(ctx.Context, browser.ScreenshotOptions{
			Path:     output,
			FullPage: fullPage,
		})
		if err != nil {
			return fmt.Errorf("screenshot failed: %w", err)
		}

		if output != "" {
			// Verify file was created
			if _, err := os.Stat(output); err != nil {
				return fmt.Errorf("screenshot file not created: %w", err)
			}
			info, _ := os.Stat(output)
			fmt.Printf("Screenshot saved: %s (%d bytes)\n", output, info.Size())
		} else {
			fmt.Println(result)
		}

		return nil
	},
}

var computerGetCmd = &cobra.Command{
	Use:   "get <what> [selector] [attr]",
	Short: "Get information (text, value, title, url, attr)",
	Long: `Get information from the page or an element.
  dba computer get text @e1      # Get element text
  dba computer get value @e2     # Get input value
  dba computer get title         # Get page title
  dba computer get url           # Get current URL
  dba computer get attr @e1 href # Get element attribute`,
	Example: `  dba computer get title
  dba computer get url
  dba computer get text @e1
  dba computer get value @e2
  dba computer get attr @e1 href`,
	Args: cobra.RangeArgs(1, 3),
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

		what := args[0]
		var result string

		switch what {
		case "text":
			if len(args) < 2 {
				return fmt.Errorf("usage: get text <selector>")
			}
			result, err = client.GetText(ctx.Context, args[1])
		case "value":
			if len(args) < 2 {
				return fmt.Errorf("usage: get value <selector>")
			}
			result, err = client.GetValue(ctx.Context, args[1])
		case "title":
			result, err = client.GetTitle(ctx.Context)
		case "url":
			result, err = client.GetURL(ctx.Context)
		case "attr":
			if len(args) < 3 {
				return fmt.Errorf("usage: get attr <selector> <attribute>")
			}
			result, err = client.GetAttribute(ctx.Context, args[1], args[2])
		default:
			return fmt.Errorf("unknown get type: %s (use: text, value, title, url, attr)", what)
		}

		if err != nil {
			return err
		}

		fmt.Println(result)
		return nil
	},
}

var computerIsCmd = &cobra.Command{
	Use:   "is <what> <selector>",
	Short: "Check element state (visible, enabled, checked)",
	Example: `  dba computer is visible @e1
  dba computer is enabled @e2
  dba computer is checked @e3`,
	Args:  cobra.ExactArgs(2),
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

		what := args[0]
		selector := args[1]
		var result bool

		switch what {
		case "visible":
			result, err = client.IsVisible(ctx.Context, selector)
		case "enabled":
			result, err = client.IsEnabled(ctx.Context, selector)
		case "checked":
			result, err = client.IsChecked(ctx.Context, selector)
		default:
			return fmt.Errorf("unknown check type: %s (use: visible, enabled, checked)", what)
		}

		if err != nil {
			return err
		}

		fmt.Println(result)
		return nil
	},
}

func init() {
	computerScreenshotCmd.Flags().StringP("output", "o", "", "Output file path")
	computerScreenshotCmd.Flags().Bool("full", false, "Capture full page")

	computerCmd.AddCommand(computerScreenshotCmd)
	computerCmd.AddCommand(computerGetCmd)
	computerCmd.AddCommand(computerIsCmd)
}
