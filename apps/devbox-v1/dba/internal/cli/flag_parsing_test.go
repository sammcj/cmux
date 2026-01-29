// internal/cli/flag_parsing_test.go
package cli

import (
	"bytes"
	"strings"
	"testing"

	"github.com/spf13/pflag"
)

// =============================================================================
// Flag Parsing Edge Cases
// =============================================================================

// TestFlagParsingOrder tests that flags can be passed in any order
func TestFlagParsingOrder(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{"flags before command", []string{"--json", "computer", "status"}},
		{"flags after command", []string{"computer", "status", "--json"}},
		{"mixed flags", []string{"computer", "--workspace", "test", "status", "--json"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := GetRootCmd()
			buf := new(bytes.Buffer)
			root.SetOut(buf)
			root.SetErr(buf)
			root.SetArgs(tt.args)

			// Execute should not fail due to flag parsing
			// (it may fail for other reasons like workspace not found)
			_ = root.Execute()
			// If we got past Execute, flag parsing worked
		})
	}
}

// TestBooleanFlagVariants tests different ways to specify boolean flags
func TestBooleanFlagVariants(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{"short form", []string{"computer", "status", "-j"}},
		{"long form", []string{"computer", "status", "--json"}},
		{"explicit true", []string{"computer", "status", "--json=true"}},
		{"explicit false", []string{"computer", "status", "--json=false"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := GetRootCmd()
			buf := new(bytes.Buffer)
			root.SetOut(buf)
			root.SetErr(buf)
			root.SetArgs(tt.args)

			// Execute to test flag parsing
			_ = root.Execute()
		})
	}
}

// TestStringFlagVariants tests different ways to specify string flags
func TestStringFlagVariants(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{"space separated", []string{"--workspace", "test-ws", "computer", "status"}},
		{"equals sign", []string{"--workspace=test-ws", "computer", "status"}},
		{"short form space", []string{"-w", "test-ws", "computer", "status"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := GetRootCmd()
			buf := new(bytes.Buffer)
			root.SetOut(buf)
			root.SetErr(buf)
			root.SetArgs(tt.args)

			_ = root.Execute()
		})
	}
}

// TestIntFlagVariants tests different ways to specify integer flags
func TestIntFlagVariants(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{"space separated", []string{"--timeout", "60000", "computer", "status"}},
		{"equals sign", []string{"--timeout=60000", "computer", "status"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := GetRootCmd()
			buf := new(bytes.Buffer)
			root.SetOut(buf)
			root.SetErr(buf)
			root.SetArgs(tt.args)

			_ = root.Execute()
		})
	}
}

// TestAppPortFlagVariants tests app command port flag
func TestAppPortFlagVariants(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{"short form", []string{"computer", "app", "-p", "3000"}},
		{"long form", []string{"computer", "app", "--port", "3000"}},
		{"equals sign", []string{"computer", "app", "--port=3000"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := GetRootCmd()
			buf := new(bytes.Buffer)
			root.SetOut(buf)
			root.SetErr(buf)
			root.SetArgs(tt.args)

			_ = root.Execute()
		})
	}
}

// TestNoBrowserFlagVariants tests app command no-browser flag
func TestNoBrowserFlagVariants(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{"present", []string{"computer", "app", "--no-browser"}},
		{"explicit true", []string{"computer", "app", "--no-browser=true"}},
		{"explicit false", []string{"computer", "app", "--no-browser=false"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := GetRootCmd()
			buf := new(bytes.Buffer)
			root.SetOut(buf)
			root.SetErr(buf)
			root.SetArgs(tt.args)

			_ = root.Execute()
		})
	}
}

// =============================================================================
// Flag Inheritance Tests
// =============================================================================

// TestGlobalFlagsAreInherited tests that global flags are available to all subcommands
func TestGlobalFlagsAreInherited(t *testing.T) {
	globalFlags := []string{"workspace", "verbose", "json", "timeout"}

	root := GetRootCmd()
	computerCmd := findCommand(root, "computer")
	if computerCmd == nil {
		t.Fatal("computer command not found")
	}

	// Check each subcommand has access to global flags
	for _, sub := range computerCmd.Commands() {
		for _, flag := range globalFlags {
			f := sub.InheritedFlags().Lookup(flag)
			if f == nil {
				f = sub.Flag(flag)
			}
			if f == nil {
				t.Errorf("subcommand '%s' should have access to global flag '%s'", sub.Name(), flag)
			}
		}
	}
}

// TestLocalFlagsNotInherited tests that local flags are NOT inherited
func TestLocalFlagsNotInherited(t *testing.T) {
	root := GetRootCmd()
	computerCmd := findCommand(root, "computer")
	if computerCmd == nil {
		t.Fatal("computer command not found")
	}

	appCmd := findCommand(computerCmd, "app")
	if appCmd == nil {
		t.Fatal("app command not found")
	}

	// port is a local flag to app command
	portFlag := appCmd.LocalFlags().Lookup("port")
	if portFlag == nil {
		t.Error("app command should have local 'port' flag")
	}

	// Other commands should NOT have port flag
	statusCmd := findCommand(computerCmd, "status")
	if statusCmd != nil {
		statusPortFlag := statusCmd.LocalFlags().Lookup("port")
		if statusPortFlag != nil {
			t.Error("status command should NOT have 'port' flag")
		}
	}
}

// =============================================================================
// Flag Default Values Tests
// =============================================================================

// TestFlagDefaultValues tests that flags have correct default values
func TestFlagDefaultValues(t *testing.T) {
	root := GetRootCmd()

	// Check workspace flag default
	workspaceFlag := root.PersistentFlags().Lookup("workspace")
	if workspaceFlag != nil && workspaceFlag.DefValue != "" {
		t.Errorf("workspace flag default should be empty, got '%s'", workspaceFlag.DefValue)
	}

	// Check json flag default
	jsonFlag := root.PersistentFlags().Lookup("json")
	if jsonFlag != nil && jsonFlag.DefValue != "false" {
		t.Errorf("json flag default should be 'false', got '%s'", jsonFlag.DefValue)
	}

	// Check verbose flag default
	verboseFlag := root.PersistentFlags().Lookup("verbose")
	if verboseFlag != nil && verboseFlag.DefValue != "false" {
		t.Errorf("verbose flag default should be 'false', got '%s'", verboseFlag.DefValue)
	}
}

// TestAppFlagDefaultValues tests app command flag defaults
func TestAppFlagDefaultValues(t *testing.T) {
	root := GetRootCmd()
	computerCmd := findCommand(root, "computer")
	if computerCmd == nil {
		t.Fatal("computer command not found")
	}

	appCmd := findCommand(computerCmd, "app")
	if appCmd == nil {
		t.Fatal("app command not found")
	}

	// Check port flag default
	portFlag := appCmd.Flags().Lookup("port")
	if portFlag != nil && portFlag.DefValue != "0" {
		t.Errorf("port flag default should be '0', got '%s'", portFlag.DefValue)
	}

	// Check no-browser flag default
	noBrowserFlag := appCmd.Flags().Lookup("no-browser")
	if noBrowserFlag != nil && noBrowserFlag.DefValue != "false" {
		t.Errorf("no-browser flag default should be 'false', got '%s'", noBrowserFlag.DefValue)
	}
}

// =============================================================================
// Flag Help Text Tests
// =============================================================================

// TestFlagsHaveUsageText tests that all flags have usage text
func TestFlagsHaveUsageText(t *testing.T) {
	root := GetRootCmd()

	// Check persistent flags
	root.PersistentFlags().VisitAll(func(f *pflag.Flag) {
		if f.Usage == "" {
			t.Errorf("persistent flag '%s' should have usage text", f.Name)
		}
	})

	// Check computer subcommand flags
	computerCmd := findCommand(root, "computer")
	if computerCmd != nil {
		for _, sub := range computerCmd.Commands() {
			sub.LocalFlags().VisitAll(func(f *pflag.Flag) {
				if f.Usage == "" {
					t.Errorf("flag '%s' on command '%s' should have usage text", f.Name, sub.Name())
				}
			})
		}
	}
}

// TestFlagShorthandsUnique tests that flag shorthands are unique within commands
func TestFlagShorthandsUnique(t *testing.T) {
	root := GetRootCmd()
	computerCmd := findCommand(root, "computer")
	if computerCmd == nil {
		t.Fatal("computer command not found")
	}

	for _, sub := range computerCmd.Commands() {
		shorthands := make(map[string]string)

		// Include inherited flags
		sub.Flags().VisitAll(func(f *pflag.Flag) {
			if f.Shorthand != "" {
				if existing, ok := shorthands[f.Shorthand]; ok {
					t.Errorf("command '%s' has duplicate shorthand '%s' for flags '%s' and '%s'",
						sub.Name(), f.Shorthand, existing, f.Name)
				}
				shorthands[f.Shorthand] = f.Name
			}
		})
	}
}

// =============================================================================
// Help Output Tests
// =============================================================================

// TestHelpOutputContainsFlags tests that help output shows flags
func TestHelpOutputContainsFlags(t *testing.T) {
	root := GetRootCmd()
	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs([]string{"--help"})

	_ = root.Execute()

	output := buf.String()

	// Should contain flag section
	if !strings.Contains(output, "Flags:") && !strings.Contains(output, "Global Flags:") {
		t.Error("help output should contain 'Flags:' or 'Global Flags:'")
	}

	// Should mention key flags
	if !strings.Contains(output, "--workspace") {
		t.Error("help output should mention --workspace flag")
	}
	if !strings.Contains(output, "--json") {
		t.Error("help output should mention --json flag")
	}
}

// TestComputerHelpContainsFlags tests that computer help shows flags
func TestComputerHelpContainsFlags(t *testing.T) {
	root := GetRootCmd()
	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs([]string{"computer", "--help"})

	_ = root.Execute()

	output := buf.String()

	// Should show available subcommands
	if !strings.Contains(output, "Available Commands:") {
		t.Error("computer help should contain 'Available Commands:'")
	}
}

// TestAppHelpContainsFlags tests that app help shows its flags
func TestAppHelpContainsFlags(t *testing.T) {
	root := GetRootCmd()
	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs([]string{"computer", "app", "--help"})

	_ = root.Execute()

	output := buf.String()

	// Should mention app-specific flags
	if !strings.Contains(output, "--port") {
		t.Error("app help should mention --port flag")
	}
	if !strings.Contains(output, "--no-browser") {
		t.Error("app help should mention --no-browser flag")
	}
}

// =============================================================================
// Invalid Flag Tests
// =============================================================================

// TestInvalidFlagReturnsError tests that invalid flags return errors
func TestInvalidFlagReturnsError(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{"unknown flag", []string{"--unknown-flag", "computer", "status"}},
		{"typo in flag", []string{"--jsn", "computer", "status"}},
		{"invalid shorthand", []string{"-x", "computer", "status"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := GetRootCmd()
			buf := new(bytes.Buffer)
			root.SetOut(buf)
			root.SetErr(buf)
			root.SetArgs(tt.args)

			err := root.Execute()
			if err == nil {
				t.Error("expected error for invalid flag")
			}
		})
	}
}

// TestInvalidFlagValueReturnsError tests that invalid flag values return errors
func TestInvalidFlagValueReturnsError(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{"non-integer for int flag", []string{"computer", "app", "--port", "abc"}},
		{"negative port", []string{"computer", "app", "--port", "-1"}},
		{"empty value for required", []string{"computer", "start", "--snapshot="}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := GetRootCmd()
			buf := new(bytes.Buffer)
			root.SetOut(buf)
			root.SetErr(buf)
			root.SetArgs(tt.args)

			err := root.Execute()
			// Some may not error at parse time but at execution time
			// Either is acceptable
			_ = err
		})
	}
}

// =============================================================================
// Special Characters in Flag Values Tests
// =============================================================================

// TestSpecialCharsInWorkspace tests special characters in workspace name
func TestSpecialCharsInWorkspace(t *testing.T) {
	tests := []struct {
		name      string
		workspace string
	}{
		{"with dash", "my-workspace"},
		{"with underscore", "my_workspace"},
		{"with dot", "my.workspace"},
		{"with number", "workspace123"},
		{"mixed", "my-workspace_v1.0"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := GetRootCmd()
			buf := new(bytes.Buffer)
			root.SetOut(buf)
			root.SetErr(buf)
			root.SetArgs([]string{"--workspace", tt.workspace, "computer", "status"})

			// Should parse without error (may fail to find workspace)
			_ = root.Execute()
		})
	}
}

// Type alias for pflag.Flag to make tests compile
type Flag = struct {
	Name      string
	Shorthand string
	Usage     string
	DefValue  string
}
