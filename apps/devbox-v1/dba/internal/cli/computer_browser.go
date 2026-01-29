// internal/cli/computer_browser.go
package cli

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/dba-cli/dba/internal/browser"
	"github.com/dba-cli/dba/internal/workspace"
)

// computerSnapshotCmd gets interactive elements
var computerSnapshotCmd = &cobra.Command{
	Use:   "snapshot",
	Short: "Get interactive elements with refs (@e1, @e2, ...)",
	Long: `Returns a list of interactive elements on the current page.
Each element has a ref (like @e1) that can be used in click, type, fill commands.`,
	Example: `  dba computer snapshot
  dba computer snapshot -i`,
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

		interactive, _ := cmd.Flags().GetBool("interactive")
		if !cmd.Flags().Changed("interactive") {
			interactive = true // Default to interactive
		}

		snapshot, err := client.Snapshot(ctx.Context, interactive)
		if err != nil {
			return fmt.Errorf("failed to get snapshot: %w", err)
		}

		// Print raw output for agent consumption
		fmt.Println(snapshot.Raw)
		return nil
	},
}

// computerClickCmd clicks an element
var computerClickCmd = &cobra.Command{
	Use:   "click <selector>",
	Short: "Click an element",
	Long: `Click an element using a ref (@e1), CSS selector (#id, .class), or text (text=Submit).
Run 'dba computer snapshot' first to get element refs.`,
	Example: `  dba computer click @e1
  dba computer click "#submit-btn"
  dba computer click "text=Login"`,
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

		selector := args[0]
		if err := client.Click(ctx.Context, selector); err != nil {
			return fmt.Errorf("click failed: %w", err)
		}

		fmt.Printf("Clicked: %s\n", selector)
		return nil
	},
}

// computerDblclickCmd double-clicks an element
var computerDblclickCmd = &cobra.Command{
	Use:   "dblclick <selector>",
	Short: "Double-click an element",
	Example: `  dba computer dblclick @e1`,
	Args:  cobra.ExactArgs(1),
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

		if err := client.DoubleClick(ctx.Context, args[0]); err != nil {
			return fmt.Errorf("double-click failed: %w", err)
		}

		fmt.Printf("Double-clicked: %s\n", args[0])
		return nil
	},
}

// computerTypeCmd types text into an element
var computerTypeCmd = &cobra.Command{
	Use:   "type <selector> <text>",
	Short: "Type text into an element (appends to existing)",
	Example: `  dba computer type @e2 "Hello World"`,
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

		if err := client.Type(ctx.Context, args[0], args[1]); err != nil {
			return fmt.Errorf("type failed: %w", err)
		}

		fmt.Printf("Typed into %s\n", args[0])
		return nil
	},
}

// computerFillCmd clears and fills an element
var computerFillCmd = &cobra.Command{
	Use:   "fill <selector> <text>",
	Short: "Clear and fill an element with text",
	Example: `  dba computer fill @e2 "test@example.com"`,
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

		if err := client.Fill(ctx.Context, args[0], args[1]); err != nil {
			return fmt.Errorf("fill failed: %w", err)
		}

		fmt.Printf("Filled %s\n", args[0])
		return nil
	},
}

// computerPressCmd presses a key
var computerPressCmd = &cobra.Command{
	Use:   "press <key>",
	Short: "Press a keyboard key (Enter, Tab, Escape, etc.)",
	Long: `Press a keyboard key. Common keys: Enter, Tab, Escape, Backspace, Delete,
ArrowUp, ArrowDown, ArrowLeft, ArrowRight, Home, End, PageUp, PageDown,
F1-F12, Control+a, Shift+Tab, etc.`,
	Example: `  dba computer press Enter
  dba computer press Tab
  dba computer press Control+a`,
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

		if err := client.Press(ctx.Context, args[0]); err != nil {
			return fmt.Errorf("press failed: %w", err)
		}

		fmt.Printf("Pressed: %s\n", args[0])
		return nil
	},
}

// computerHoverCmd hovers over an element
var computerHoverCmd = &cobra.Command{
	Use:   "hover <selector>",
	Short: "Hover over an element",
	Example: `  dba computer hover @e1`,
	Args:  cobra.ExactArgs(1),
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

		if err := client.Hover(ctx.Context, args[0]); err != nil {
			return fmt.Errorf("hover failed: %w", err)
		}

		fmt.Printf("Hovering over: %s\n", args[0])
		return nil
	},
}

// computerSelectCmd selects a dropdown option
var computerSelectCmd = &cobra.Command{
	Use:   "select <selector> <value>",
	Short: "Select an option in a dropdown",
	Example: `  dba computer select @e3 "Option 1"`,
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

		if err := client.Select(ctx.Context, args[0], args[1]); err != nil {
			return fmt.Errorf("select failed: %w", err)
		}

		fmt.Printf("Selected '%s' in %s\n", args[1], args[0])
		return nil
	},
}

// computerScrollCmd scrolls the page
var computerScrollCmd = &cobra.Command{
	Use:   "scroll <direction> [amount]",
	Short: "Scroll the page (up, down, left, right)",
	Example: `  dba computer scroll down
  dba computer scroll up 500`,
	Args:  cobra.RangeArgs(1, 2),
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

		direction := browser.ScrollDirection(args[0])
		amount := 0
		if len(args) > 1 {
			fmt.Sscanf(args[1], "%d", &amount)
		}

		if err := client.Scroll(ctx.Context, direction, amount); err != nil {
			return fmt.Errorf("scroll failed: %w", err)
		}

		fmt.Printf("Scrolled %s\n", direction)
		return nil
	},
}

// Helper to get browser client
func getBrowserClient(ws *workspace.Workspace, cfg interface {
	GetMorphAPIKey() string
	GetBaseSnapshotID() string
}, ctx ...context.Context) (*browser.Client, error) {
	if !ws.IsMorphRunning() {
		return nil, fmt.Errorf("VM is not running. Start it with: dba computer start")
	}

	// Get the actual config type to access AgentBrowser settings
	type agentBrowserConfig interface {
		GetAgentBrowserTimeout() int
		GetAgentBrowserSessionPrefix() string
	}

	var timeout int = 30000
	var sessionPrefix string = "dba"

	// Try to get config values if available
	if abc, ok := cfg.(agentBrowserConfig); ok {
		timeout = abc.GetAgentBrowserTimeout()
		sessionPrefix = abc.GetAgentBrowserSessionPrefix()
	}

	clientConfig := browser.ClientConfig{
		CDPPort: ws.Morph.CDPPort,
		CDPURL:  ws.Morph.CDPURL,
		Timeout: timeout,
		Session: fmt.Sprintf("%s-%s", sessionPrefix, ws.ID),
	}

	client, err := browser.NewClient(clientConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create browser client: %w", err)
	}

	// Connect to the browser
	var connectCtx context.Context
	if len(ctx) > 0 && ctx[0] != nil {
		connectCtx = ctx[0]
	} else {
		connectCtx = context.Background()
	}

	if err := client.Connect(connectCtx); err != nil {
		return nil, fmt.Errorf("failed to connect to browser: %w", err)
	}

	return client, nil
}

func init() {
	computerSnapshotCmd.Flags().BoolP("interactive", "i", true, "Only show interactive elements")
	computerSnapshotCmd.Flags().BoolP("compact", "c", false, "Compact output")

	computerCmd.AddCommand(computerSnapshotCmd)
	computerCmd.AddCommand(computerClickCmd)
	computerCmd.AddCommand(computerDblclickCmd)
	computerCmd.AddCommand(computerTypeCmd)
	computerCmd.AddCommand(computerFillCmd)
	computerCmd.AddCommand(computerPressCmd)
	computerCmd.AddCommand(computerHoverCmd)
	computerCmd.AddCommand(computerSelectCmd)
	computerCmd.AddCommand(computerScrollCmd)
}
