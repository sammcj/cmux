// internal/cli/computer_app.go
package cli

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
)

// computerAppCmd opens the app in browser and shows interactive elements
var computerAppCmd = &cobra.Command{
	Use:   "app",
	Short: "Open the app in browser and show interactive elements",
	Long: `Opens the primary app port in the browser and displays interactive elements.
Automatically detects the primary HTTP port from running containers.

This command:
1. Discovers running containers and their exposed ports
2. Opens the primary app URL in the browser
3. Takes a snapshot showing interactive elements (@e1, @e2, ...)`,
	Example: `  dba computer app
  dba computer app --port 3000
  dba computer app --no-browser`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx, err := NewCLIContext()
		if err != nil {
			return err
		}
		defer ctx.Cancel()

		if err := ctx.RequireWorkspace(); err != nil {
			return err
		}

		if !ctx.Workspace.IsMorphRunning() {
			return fmt.Errorf("VM is not running. Start it with: dba computer start")
		}

		// Get flags
		port, _ := cmd.Flags().GetInt("port")
		noBrowser, _ := cmd.Flags().GetBool("no-browser")
		jsonOutput, _ := cmd.Flags().GetBool("json")

		// Find the app URL
		var appURL string
		if port > 0 {
			// Use specified port
			appPort := ctx.Workspace.GetActivePort(port)
			if appPort != nil && appPort.URL != "" {
				appURL = appPort.URL
			} else {
				// Construct URL from base URL
				appURL = fmt.Sprintf("%s:%d/", ctx.Workspace.Morph.BaseURL, port)
			}
		} else {
			// Auto-detect primary app port
			primaryPort := ctx.Workspace.GetPrimaryAppPort()
			if primaryPort != nil {
				if primaryPort.URL != "" {
					appURL = primaryPort.URL
				} else if primaryPort.LocalPort > 0 {
					appURL = fmt.Sprintf("http://localhost:%d", primaryPort.LocalPort)
				} else {
					appURL = fmt.Sprintf("http://localhost:%d", primaryPort.Port)
				}
			} else {
				// Fall back to the Morph AppURL
				appURL = ctx.Workspace.Morph.AppURL
			}
		}

		if appURL == "" {
			return fmt.Errorf("no app URL available. Start a dev server or specify --port")
		}

		// Open in browser (unless --no-browser)
		if !noBrowser {
			if err := openBrowser(appURL); err != nil {
				// Non-fatal: just warn
				fmt.Printf("Warning: could not open browser: %v\n", err)
			} else {
				fmt.Printf("Opened: %s\n", appURL)
			}
		}

		// Take a snapshot to show interactive elements
		client, err := getBrowserClient(ctx.Workspace, ctx.Config, ctx.Context)
		if err != nil {
			// If browser client fails, just show the URL
			if jsonOutput {
				output := map[string]interface{}{
					"url":      appURL,
					"elements": []interface{}{},
				}
				data, _ := json.MarshalIndent(output, "", "  ")
				fmt.Println(string(data))
			} else {
				fmt.Printf("App URL: %s\n", appURL)
				fmt.Println("(Could not get interactive elements)")
			}
			return nil
		}

		snapshot, err := client.Snapshot(ctx.Context, true)
		if err != nil {
			// If snapshot fails, just show the URL
			if jsonOutput {
				output := map[string]interface{}{
					"url":      appURL,
					"elements": []interface{}{},
					"error":    err.Error(),
				}
				data, _ := json.MarshalIndent(output, "", "  ")
				fmt.Println(string(data))
			} else {
				fmt.Printf("App URL: %s\n", appURL)
				fmt.Printf("Warning: could not get snapshot: %v\n", err)
			}
			return nil
		}

		if jsonOutput {
			output := map[string]interface{}{
				"url":      appURL,
				"title":    snapshot.Title,
				"elements": snapshot.Elements,
			}
			data, _ := json.MarshalIndent(output, "", "  ")
			fmt.Println(string(data))
		} else {
			fmt.Printf("Opening: %s\n", appURL)
			if snapshot.Title != "" {
				fmt.Printf("Title: %s\n", snapshot.Title)
			}
			fmt.Println(snapshot.Raw)
		}

		return nil
	},
}

// computerPortsCmd lists active ports
var computerPortsCmd = &cobra.Command{
	Use:   "ports",
	Short: "List active ports in the VM",
	Long: `Lists all active ports exposed by containers and services in the Morph VM.

Shows port numbers, protocols, services, and containers for each active port.`,
	Example: `  dba computer ports
  dba computer ports --json`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx, err := NewCLIContext()
		if err != nil {
			return err
		}
		defer ctx.Cancel()

		if err := ctx.RequireWorkspace(); err != nil {
			return err
		}

		if !ctx.Workspace.IsMorphRunning() {
			return fmt.Errorf("VM is not running. Start it with: dba computer start")
		}

		jsonOutput, _ := cmd.Flags().GetBool("json")
		ports := ctx.Workspace.Morph.ActivePorts

		if jsonOutput {
			data, _ := json.MarshalIndent(ports, "", "  ")
			fmt.Println(string(data))
			return nil
		}

		if len(ports) == 0 {
			fmt.Println("No active ports found.")
			fmt.Println("Run 'dba up' to start services, then ports will be auto-discovered.")
			return nil
		}

		// Print in a nice table format
		fmt.Println("Active Ports:")
		fmt.Println()
		for _, p := range ports {
			line := fmt.Sprintf("  Port %d", p.Port)
			if p.LocalPort > 0 && p.LocalPort != p.Port {
				line += fmt.Sprintf(" (forwarded to %d)", p.LocalPort)
			}
			if p.Protocol != "" && p.Protocol != "tcp" {
				line += fmt.Sprintf(" [%s]", p.Protocol)
			}
			fmt.Println(line)

			if p.Service != "" {
				fmt.Printf("    Service:   %s\n", p.Service)
			}
			if p.Container != "" {
				fmt.Printf("    Container: %s\n", p.Container)
			}
			if p.URL != "" {
				fmt.Printf("    URL:       %s\n", p.URL)
			}
			if p.IsHTTP {
				fmt.Println("    Type:      HTTP")
			}
			fmt.Println()
		}

		return nil
	},
}

func init() {
	computerAppCmd.Flags().IntP("port", "p", 0, "Specific port to open (auto-detects if not specified)")
	computerAppCmd.Flags().Bool("no-browser", false, "Don't open browser, just show URL and elements")
	// Note: --json flag is inherited from root command

	// Note: --json flag is inherited from root command for ports command

	computerCmd.AddCommand(computerAppCmd)
	computerCmd.AddCommand(computerPortsCmd)
}
