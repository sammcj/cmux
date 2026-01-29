// internal/cli/run.go
package cli

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/dba-cli/dba/internal/devbox"
)

// RunResult represents the result of a run command for JSON output
type RunResult struct {
	ExitCode   int    `json:"exit_code"`
	Stdout     string `json:"stdout"`
	Stderr     string `json:"stderr"`
	DurationMs int64  `json:"duration_ms"`
	Synced     bool   `json:"synced,omitempty"`
	SyncWaitMs int64  `json:"sync_wait_ms,omitempty"`
}

// TestResult represents the result of a test command for JSON output
type TestResult struct {
	Runner     string `json:"runner"`
	Command    string `json:"command"`
	ExitCode   int    `json:"exit_code"`
	DurationMs int64  `json:"duration_ms"`
	Stdout     string `json:"stdout,omitempty"`
	Stderr     string `json:"stderr,omitempty"`
}

// runCmd is the dba run command
var runCmd = &cobra.Command{
	Use:   "run <command...>",
	Short: "Run a command in the workspace environment",
	Long: `Run a command in the workspace's devbox environment.

The command runs with all workspace environment variables (PORT, API_PORT, etc.)
and waits for the sync barrier before executing (unless --no-sync is specified).

Examples:
  dba run npm install
  dba run pnpm test -- --watch
  dba run "echo \$PORT"  # Uses workspace PORT env var
  dba run --no-sync npm start`,
	Args: cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		// Check for devbox
		if err := devbox.EnsureDevbox(); err != nil {
			OutputError(err)
			return err
		}

		ctx, err := NewCLIContext()
		if err != nil {
			OutputError(err)
			return err
		}
		defer ctx.Cancel()

		if err := ctx.RequireWorkspace(); err != nil {
			OutputError(err)
			return err
		}

		noSync, _ := cmd.Flags().GetBool("no-sync")
		cwd, _ := cmd.Flags().GetString("cwd")
		envFlags, _ := cmd.Flags().GetStringArray("env")

		// Parse env flags into map
		env := make(map[string]string)
		for _, e := range envFlags {
			parts := strings.SplitN(e, "=", 2)
			if len(parts) == 2 {
				env[parts[0]] = parts[1]
			}
		}

		// Build command string from args
		command := strings.Join(args, " ")

		runner := devbox.NewRunner(ctx.Workspace, ctx.Config, noSync)
		result, err := runner.Run(ctx.Context, command, devbox.RunOptions{
			Cwd: cwd,
			Env: env,
		})
		if err != nil {
			OutputError(err)
			return err
		}

		if flagJSON {
			return OutputResult(RunResult{
				ExitCode:   result.ExitCode,
				Stdout:     result.Stdout,
				Stderr:     result.Stderr,
				DurationMs: result.DurationMs,
				Synced:     result.Synced,
				SyncWaitMs: result.SyncWaitMs,
			})
		}

		// Human-readable output: just print stdout/stderr
		fmt.Print(result.Stdout)
		if result.Stderr != "" {
			fmt.Fprint(os.Stderr, result.Stderr)
		}

		// Exit with the command's exit code
		if result.ExitCode != 0 {
			os.Exit(result.ExitCode)
		}

		return nil
	},
}

// testCmd is the dba test command
var testCmd = &cobra.Command{
	Use:   "test [pattern]",
	Short: "Run tests with automatic test runner detection",
	Long: `Run tests with automatic test runner detection.

Supported test runners (in order of detection priority):
- vitest (detected from vitest.config.* or package.json)
- jest (detected from jest.config.* or package.json)
- mocha (detected from package.json)
- pytest (detected from pyproject.toml or pytest.ini)
- go test (detected from go.mod)
- cargo test (detected from Cargo.toml)
- npm test (fallback for Node.js projects)

Examples:
  dba test                      # Run all tests
  dba test auth                 # Run tests matching "auth"
  dba test --watch              # Watch mode
  dba test --coverage           # Generate coverage
  dba test --runner=vitest      # Force specific runner`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		// Check for devbox
		if err := devbox.EnsureDevbox(); err != nil {
			OutputError(err)
			return err
		}

		ctx, err := NewCLIContext()
		if err != nil {
			OutputError(err)
			return err
		}
		defer ctx.Cancel()

		if err := ctx.RequireWorkspace(); err != nil {
			OutputError(err)
			return err
		}

		// Get test pattern if provided
		pattern := ""
		if len(args) > 0 {
			pattern = args[0]
		}

		watch, _ := cmd.Flags().GetBool("watch")
		coverage, _ := cmd.Flags().GetBool("coverage")
		runnerName, _ := cmd.Flags().GetString("runner")

		// Detect or use specified runner
		var runner *devbox.TestRunner
		if runnerName != "" {
			runner = devbox.GetRunnerByName(runnerName)
		} else {
			runner = devbox.DetectTestRunner(ctx.Workspace.ProjectPath)
		}

		// Build test command
		command := runner.BuildTestCommand(pattern, watch, coverage)

		// Run tests
		dbxRunner := devbox.NewRunner(ctx.Workspace, ctx.Config, false)
		result, err := dbxRunner.Run(ctx.Context, command, devbox.RunOptions{})
		if err != nil {
			OutputError(err)
			return err
		}

		if flagJSON {
			return OutputResult(TestResult{
				Runner:     runner.Name,
				Command:    command,
				ExitCode:   result.ExitCode,
				DurationMs: result.DurationMs,
				Stdout:     result.Stdout,
				Stderr:     result.Stderr,
			})
		}

		// Human-readable output
		if flagVerbose {
			fmt.Printf("Running: %s\n\n", command)
		}
		fmt.Print(result.Stdout)
		if result.Stderr != "" {
			fmt.Fprint(os.Stderr, result.Stderr)
		}

		// Exit with the test command's exit code
		if result.ExitCode != 0 {
			os.Exit(result.ExitCode)
		}

		return nil
	},
}

// shellCmd is the dba shell command
var shellCmd = &cobra.Command{
	Use:   "shell",
	Short: "Enter interactive devbox shell",
	Long: `Enter an interactive devbox shell with all workspace environment variables.

The shell provides access to all packages installed via devbox, and has
all workspace environment variables set (PORT, API_PORT, etc.).

Examples:
  dba shell              # Enter shell
  dba shell --pure       # Enter pure shell (minimal host environment)`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Check for devbox
		if err := devbox.EnsureDevbox(); err != nil {
			OutputError(err)
			return err
		}

		ctx, err := NewCLIContext()
		if err != nil {
			OutputError(err)
			return err
		}
		defer ctx.Cancel()

		if err := ctx.RequireWorkspace(); err != nil {
			OutputError(err)
			return err
		}

		pure, _ := cmd.Flags().GetBool("pure")

		dbx := devbox.New(ctx.Workspace)
		return dbx.Shell(ctx.Context, pure)
	},
}

func init() {
	// run command flags
	runCmd.Flags().Bool("no-sync", false, "Skip sync barrier (don't wait for file changes to propagate)")
	runCmd.Flags().String("cwd", "", "Working directory relative to project root")
	runCmd.Flags().StringArray("env", nil, "Additional environment variables (KEY=VALUE, can be repeated)")
	runCmd.Flags().Duration("timeout", 0, "Command timeout (0 = use global timeout)")

	// test command flags
	testCmd.Flags().Bool("watch", false, "Watch mode (re-run tests on file changes)")
	testCmd.Flags().Bool("coverage", false, "Generate test coverage report")
	testCmd.Flags().String("runner", "", "Force specific test runner (vitest, jest, pytest, go, cargo, npm)")

	// shell command flags
	shellCmd.Flags().Bool("pure", false, "Pure shell (minimal host environment)")

	// Register commands with root
	rootCmd.AddCommand(runCmd)
	rootCmd.AddCommand(testCmd)
	rootCmd.AddCommand(shellCmd)
}
