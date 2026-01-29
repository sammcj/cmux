// internal/cli/root.go
package cli

import (
	"os"

	"github.com/spf13/cobra"

	"github.com/dba-cli/dba/internal/config"
)

var (
	// Global flags
	flagJSON      bool
	flagWorkspace string
	flagVerbose   bool
	flagTimeout   string

	// Global config
	cfg *config.Config
)

var rootCmd = &cobra.Command{
	Use:   "dba",
	Short: "DevBox Agent - Development environments for AI agents",
	Long: `DBA (DevBox Agent) creates isolated development environments
with browser automation, VS Code, and full computer use capabilities.

All commands support --json for machine-readable output.`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// Load config
		var err error
		cfg, err = config.Load()
		if err != nil {
			return err
		}

		// Auto-detect JSON mode if stdout is not a TTY
		if !isTerminal(os.Stdout) && !flagJSON {
			// Only auto-enable if not explicitly disabled
			// Check if the flag was explicitly set
			if !cmd.Flags().Changed("json") {
				flagJSON = true
			}
		}

		return nil
	},
	// Silence usage and errors - we handle our own error output
	SilenceUsage:  true,
	SilenceErrors: true,
}

func init() {
	// Global flags available to all commands
	rootCmd.PersistentFlags().BoolVar(&flagJSON, "json", false,
		"Output as JSON")
	rootCmd.PersistentFlags().StringVarP(&flagWorkspace, "workspace", "w", "",
		"Workspace ID or path (default: detect from current directory)")
	rootCmd.PersistentFlags().BoolVarP(&flagVerbose, "verbose", "v", false,
		"Verbose output")
	rootCmd.PersistentFlags().StringVar(&flagTimeout, "timeout", "5m",
		"Command timeout")

	// Add subcommands (other agents will implement these)
	// rootCmd.AddCommand(workspaceCmd)  // Agent #4
	// rootCmd.AddCommand(serviceCmd)    // Agent #6
	// rootCmd.AddCommand(fsCmd)         // Agent #7
	// rootCmd.AddCommand(codeCmd)       // Agent #10
	// rootCmd.AddCommand(computerCmd)   // Agent #9
	// rootCmd.AddCommand(portCmd)       // Agent #2
	// rootCmd.AddCommand(daemonCmd)     // Agent #3

	// Version command
	rootCmd.AddCommand(versionCmd)
}

// Execute runs the root command
func Execute() error {
	return rootCmd.Execute()
}

// AddCommand adds a subcommand to root - for use by other agents
func AddCommand(cmd *cobra.Command) {
	rootCmd.AddCommand(cmd)
}

// GetRootCmd returns the root command - for use by other agents
func GetRootCmd() *cobra.Command {
	return rootCmd
}

// Helper to check if output is a terminal
func isTerminal(f *os.File) bool {
	fi, err := f.Stat()
	if err != nil {
		return false
	}
	return (fi.Mode() & os.ModeCharDevice) != 0
}
