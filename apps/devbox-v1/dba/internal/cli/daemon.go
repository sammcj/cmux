// Package cli provides daemon CLI commands for managing the DBA daemon.
package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/dba-cli/dba/internal/daemon"
)

var daemonCmd = &cobra.Command{
	Use:   "daemon",
	Short: "Manage the DBA daemon",
	Long: `Manage the DBA background daemon process.

The daemon manages:
- Port registry and allocation
- Workspace registration and state
- Service health monitoring
- Sync barriers for file changes

The daemon is automatically started when needed by other commands.`,
}

var daemonStartCmd = &cobra.Command{
	Use:   "start",
	Short: "Start the daemon",
	Long: `Start the DBA daemon process.

By default, the daemon runs in the background. Use --foreground to run
in the foreground (useful for debugging).`,
	RunE: func(cmd *cobra.Command, args []string) error {
		foreground, _ := cmd.Flags().GetBool("foreground")

		if foreground {
			// Run daemon in foreground
			d, err := daemon.New(cfg)
			if err != nil {
				return fmt.Errorf("failed to create daemon: %w", err)
			}
			return d.Start()
		}

		// Check if already running
		client := daemon.NewClient(cfg)
		if client.IsRunning() {
			result := map[string]interface{}{
				"status": "already_running",
				"socket": cfg.Daemon.Socket,
			}

			if pid := daemon.GetDaemonPID(cfg); pid > 0 {
				result["pid"] = pid
			}

			return OutputResult(result)
		}

		// Start in background
		if err := daemon.StartInBackground(cfg); err != nil {
			return fmt.Errorf("failed to start daemon: %w", err)
		}

		// Get PID after starting
		pid := daemon.GetDaemonPID(cfg)

		result := map[string]interface{}{
			"status": "started",
			"pid":    pid,
			"socket": cfg.Daemon.Socket,
		}

		return OutputResult(result)
	},
}

var daemonStopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stop the daemon",
	Long:  `Stop the DBA daemon process gracefully.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Get PID before stopping
		pid := daemon.GetDaemonPID(cfg)
		if pid == 0 {
			return OutputResult(map[string]interface{}{
				"status":  "not_running",
				"message": "daemon is not running",
			})
		}

		if err := daemon.StopDaemon(cfg); err != nil {
			return fmt.Errorf("failed to stop daemon: %w", err)
		}

		result := map[string]interface{}{
			"status": "stopped",
			"pid":    pid,
		}

		return OutputResult(result)
	},
}

var daemonStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Get daemon status",
	Long:  `Get the current status of the DBA daemon.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		client := daemon.NewClient(cfg)

		// Check if daemon is running
		status, err := client.Status()
		if err != nil {
			// Daemon not running
			return OutputResult(map[string]interface{}{
				"running": false,
				"message": "daemon is not running",
			})
		}

		// Build response
		result := map[string]interface{}{
			"running":           status.Running,
			"pid":               status.PID,
			"socket":            status.Socket,
			"workspaces_active": status.WorkspacesActive,
			"uptime_seconds":    status.UptimeSeconds,
		}

		return OutputResult(result)
	},
}

var daemonRestartCmd = &cobra.Command{
	Use:   "restart",
	Short: "Restart the daemon",
	Long:  `Stop and start the DBA daemon.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Stop if running
		if daemon.GetDaemonPID(cfg) > 0 {
			if err := daemon.StopDaemon(cfg); err != nil {
				// Ignore errors, try to start anyway
				fmt.Fprintf(os.Stderr, "Warning: %v\n", err)
			}
		}

		// Start daemon
		if err := daemon.StartInBackground(cfg); err != nil {
			return fmt.Errorf("failed to start daemon: %w", err)
		}

		pid := daemon.GetDaemonPID(cfg)

		result := map[string]interface{}{
			"status": "restarted",
			"pid":    pid,
			"socket": cfg.Daemon.Socket,
		}

		return OutputResult(result)
	},
}

var daemonLogsCmd = &cobra.Command{
	Use:   "logs",
	Short: "View daemon logs",
	Long:  `View the DBA daemon log file.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		tail, _ := cmd.Flags().GetInt("tail")
		follow, _ := cmd.Flags().GetBool("follow")

		logFile := cfg.Daemon.LogFile

		// Check if log file exists
		if _, err := os.Stat(logFile); os.IsNotExist(err) {
			return fmt.Errorf("log file not found: %s", logFile)
		}

		if flagJSON {
			// Read log file and return as JSON
			content, err := os.ReadFile(logFile)
			if err != nil {
				return fmt.Errorf("failed to read log file: %w", err)
			}

			return OutputResult(map[string]interface{}{
				"path":    logFile,
				"content": string(content),
			})
		}

		// For non-JSON output, use tail command
		tailArgs := []string{}
		if tail > 0 {
			tailArgs = append(tailArgs, "-n", fmt.Sprintf("%d", tail))
		}
		if follow {
			tailArgs = append(tailArgs, "-f")
		}
		tailArgs = append(tailArgs, logFile)

		// Print log file path and contents
		fmt.Printf("Log file: %s\n\n", logFile)
		content, err := os.ReadFile(logFile)
		if err != nil {
			return fmt.Errorf("failed to read log file: %w", err)
		}

		// Simple tail implementation
		lines := splitLines(string(content))
		if tail > 0 && len(lines) > tail {
			lines = lines[len(lines)-tail:]
		}

		for _, line := range lines {
			fmt.Println(line)
		}

		return nil
	},
}

// splitLines splits a string into lines
func splitLines(s string) []string {
	var lines []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			lines = append(lines, s[start:i])
			start = i + 1
		}
	}
	if start < len(s) {
		lines = append(lines, s[start:])
	}
	return lines
}

func init() {
	// Add flags to start command
	daemonStartCmd.Flags().Bool("foreground", false, "Run in foreground (for debugging)")

	// Add flags to logs command
	daemonLogsCmd.Flags().IntP("tail", "n", 50, "Number of lines to show")
	daemonLogsCmd.Flags().BoolP("follow", "f", false, "Follow log output")

	// Add subcommands
	daemonCmd.AddCommand(daemonStartCmd)
	daemonCmd.AddCommand(daemonStopCmd)
	daemonCmd.AddCommand(daemonStatusCmd)
	daemonCmd.AddCommand(daemonRestartCmd)
	daemonCmd.AddCommand(daemonLogsCmd)

	// Register with root command
	rootCmd.AddCommand(daemonCmd)
}
