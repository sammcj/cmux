package cli

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os/exec"
	"runtime"

	"github.com/cmux-cli/cmux-devbox-2/internal/api"
	"github.com/spf13/cobra"
)

// buildAuthURL builds a URL with token authentication
// E2B gives each port its own subdomain, so we use query params for auth
func buildAuthURL(baseURL, token string, isVNC bool) (string, error) {
	parsed, err := url.Parse(baseURL)
	if err != nil {
		return "", fmt.Errorf("invalid URL: %w", err)
	}
	query := parsed.Query()
	if isVNC {
		// noVNC uses 'password' param, first 8 chars of token
		if len(token) >= 8 {
			query.Set("password", token[:8])
		}
		// Add default noVNC params
		query.Set("resize", "scale")
		query.Set("quality", "9")
		query.Set("compression", "0")
	} else {
		// VSCode uses 'tkn' param
		query.Set("tkn", token)
		// Set default folder
		query.Set("folder", "/home/user/workspace")
	}
	parsed.RawQuery = query.Encode()
	return parsed.String(), nil
}

var codeCmd = &cobra.Command{
	Use:   "code <id>",
	Short: "Open VS Code in browser",
	Long: `Open VS Code for a sandbox in your browser.

Examples:
  cmux code cmux_abc123`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		teamSlug, err := getTeamSlug()
		if err != nil {
			return fmt.Errorf("failed to get team: %w", err)
		}

		client := api.NewClient()
		inst, err := client.GetInstance(teamSlug, args[0])
		if err != nil {
			return err
		}

		if inst.VSCodeURL == "" {
			return fmt.Errorf("VSCode URL not available")
		}

		// Fetch auth token from the sandbox
		token, err := client.GetAuthToken(teamSlug, args[0])
		if err != nil {
			return fmt.Errorf("failed to get auth token: %w", err)
		}

		authURL, err := buildAuthURL(inst.VSCodeURL, token, false)
		if err != nil {
			return err
		}

		fmt.Println("Opening VS Code...")
		return openBrowser(authURL)
	},
}

var vncCmd = &cobra.Command{
	Use:   "vnc <id>",
	Short: "Open VNC desktop in browser",
	Long: `Open VNC desktop for a sandbox in your browser.

Examples:
  cmux vnc cmux_abc123`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		teamSlug, err := getTeamSlug()
		if err != nil {
			return fmt.Errorf("failed to get team: %w", err)
		}

		client := api.NewClient()
		inst, err := client.GetInstance(teamSlug, args[0])
		if err != nil {
			return err
		}

		if inst.VNCURL == "" {
			return fmt.Errorf("VNC URL not available")
		}

		// Fetch auth token from the sandbox
		token, err := client.GetAuthToken(teamSlug, args[0])
		if err != nil {
			return fmt.Errorf("failed to get auth token: %w", err)
		}

		authURL, err := buildAuthURL(inst.VNCURL, token, true)
		if err != nil {
			return err
		}

		fmt.Println("Opening VNC...")
		return openBrowser(authURL)
	},
}

var statusCmd = &cobra.Command{
	Use:   "status <id>",
	Short: "Show sandbox status",
	Long: `Show the status of a sandbox.

Examples:
  cmux status cmux_abc123`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		teamSlug, err := getTeamSlug()
		if err != nil {
			return fmt.Errorf("failed to get team: %w", err)
		}

		client := api.NewClient()
		inst, err := client.GetInstance(teamSlug, args[0])
		if err != nil {
			return err
		}

		if flagJSON {
			data, _ := json.MarshalIndent(inst, "", "  ")
			fmt.Println(string(data))
			return nil
		}

		fmt.Printf("ID:       %s\n", inst.DevboxID)
		fmt.Printf("Status:   %s\n", inst.Status)
		if inst.Name != "" {
			fmt.Printf("Name:     %s\n", inst.Name)
		}

		// Try to get authenticated URLs
		if inst.Status == "running" {
			token, err := client.GetAuthToken(teamSlug, args[0])
			if err == nil && token != "" {
				if inst.VSCodeURL != "" {
					codeURL, _ := buildAuthURL(inst.VSCodeURL, token, false)
					fmt.Printf("VS Code:  %s\n", codeURL)
				}
				if inst.VNCURL != "" {
					vncURL, _ := buildAuthURL(inst.VNCURL, token, true)
					fmt.Printf("VNC:      %s\n", vncURL)
				}
			} else {
				if inst.VSCodeURL != "" {
					fmt.Printf("VS Code:  %s\n", inst.VSCodeURL)
				}
				if inst.VNCURL != "" {
					fmt.Printf("VNC:      %s\n", inst.VNCURL)
				}
			}
		}

		return nil
	},
}

func openBrowser(url string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "linux":
		cmd = exec.Command("xdg-open", url)
	case "windows":
		cmd = exec.Command("cmd", "/c", "start", url)
	default:
		return fmt.Errorf("unsupported platform")
	}
	return cmd.Start()
}

// Keep openURL as alias for backward compatibility
func openURL(url string) error {
	return openBrowser(url)
}
