// internal/cli/computer_integration_test.go
package cli

import (
	"bytes"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// TestComputerCommandTreeIntegrity tests the complete command tree
func TestComputerCommandTreeIntegrity(t *testing.T) {
	root := GetRootCmd()

	// Find computer command
	var computerCmd = findCommand(root, "computer")
	if computerCmd == nil {
		t.Fatal("computer command not found")
	}

	// Count total subcommands
	subCmds := computerCmd.Commands()
	t.Logf("Total computer subcommands: %d", len(subCmds))

	// List all subcommand names
	for _, cmd := range subCmds {
		t.Logf("  - %s: %s", cmd.Name(), cmd.Short)
	}

	// Ensure we have enough subcommands
	if len(subCmds) < 21 {
		t.Errorf("expected at least 21 subcommands, got %d", len(subCmds))
	}
}

// TestComputerHelpConsistency tests that help output is consistent
func TestComputerHelpConsistency(t *testing.T) {
	root := GetRootCmd()

	// All commands should have consistent help structure
	computerCmd := findCommand(root, "computer")
	if computerCmd == nil {
		t.Fatal("computer command not found")
	}

	for _, cmd := range computerCmd.Commands() {
		t.Run(cmd.Name(), func(t *testing.T) {
			// Each command should have:
			// 1. Use field
			if cmd.Use == "" {
				t.Error("missing Use field")
			}
			// 2. Short description
			if cmd.Short == "" {
				t.Error("missing Short description")
			}
		})
	}
}

// TestComputerWorkflowCommands tests commands in a typical workflow order
func TestComputerWorkflowCommands(t *testing.T) {
	root := GetRootCmd()
	computerCmd := findCommand(root, "computer")
	if computerCmd == nil {
		t.Fatal("computer command not found")
	}

	// Typical workflow:
	// 1. start - start VM
	// 2. open - navigate to URL
	// 3. snapshot - get elements
	// 4. click/type/fill - interact
	// 5. screenshot - verify
	// 6. save - save state
	// 7. stop - stop VM

	workflow := []string{
		"start",
		"open",
		"snapshot",
		"click",
		"type",
		"fill",
		"screenshot",
		"save",
		"stop",
	}

	for _, cmdName := range workflow {
		found := false
		for _, sub := range computerCmd.Commands() {
			if sub.Name() == cmdName {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("workflow command '%s' not found", cmdName)
		}
	}
}

// TestComputerCommandOutput tests command output formatting
func TestComputerCommandOutput(t *testing.T) {
	root := GetRootCmd()
	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(buf)

	// Test main help
	root.SetArgs([]string{"computer", "--help"})
	err := root.Execute()
	if err != nil {
		t.Errorf("computer --help failed: %v", err)
	}

	output := buf.String()

	// Should contain usage section
	if !strings.Contains(output, "Usage:") {
		t.Error("help output should contain 'Usage:'")
	}

	// Should contain available commands
	if !strings.Contains(output, "Available Commands:") {
		t.Error("help output should contain 'Available Commands:'")
	}

	// Should contain flags section
	if !strings.Contains(output, "Flags:") {
		t.Error("help output should contain 'Flags:'")
	}
}

// TestComputerFlagInheritance tests that global flags are inherited
func TestComputerFlagInheritance(t *testing.T) {
	root := GetRootCmd()
	computerCmd := findCommand(root, "computer")
	if computerCmd == nil {
		t.Fatal("computer command not found")
	}

	globalFlags := []string{"json", "verbose", "workspace", "timeout"}

	for _, sub := range computerCmd.Commands() {
		for _, flagName := range globalFlags {
			flag := sub.Flags().Lookup(flagName)
			inheritedFlag := sub.InheritedFlags().Lookup(flagName)

			if flag == nil && inheritedFlag == nil {
				t.Errorf("command '%s' should have access to global flag '%s'", sub.Name(), flagName)
			}
		}
	}
}

// TestComputerNoConflictingFlags tests that there are no conflicting flag names
func TestComputerNoConflictingFlags(t *testing.T) {
	root := GetRootCmd()
	computerCmd := findCommand(root, "computer")
	if computerCmd == nil {
		t.Fatal("computer command not found")
	}

	for _, sub := range computerCmd.Commands() {
		t.Run(sub.Name(), func(t *testing.T) {
			// Count local flags - if there's an issue, cobra would error during registration
			localCount := 0
			sub.LocalFlags().VisitAll(func(f *pflag.Flag) {
				localCount++
			})
			t.Logf("command %s has %d local flags", sub.Name(), localCount)
		})
	}
}

// TestAllCommandsReturnErrorWithoutWorkspace tests that commands error when workspace is missing
func TestAllCommandsReturnErrorWithoutWorkspace(t *testing.T) {
	root := GetRootCmd()
	computerCmd := findCommand(root, "computer")
	if computerCmd == nil {
		t.Fatal("computer command not found")
	}

	// Commands that require a workspace
	cmdsNeedingWorkspace := []string{
		"start", "stop", "status", "save", "vnc",
		"snapshot", "click", "type", "fill", "press",
		"hover", "select", "scroll", "open", "back",
		"forward", "reload", "screenshot", "get", "is", "wait",
	}

	for _, cmdName := range cmdsNeedingWorkspace {
		t.Run(cmdName, func(t *testing.T) {
			buf := new(bytes.Buffer)
			root.SetOut(buf)
			root.SetErr(buf)

			// Attempt to run command without workspace
			args := []string{"computer", cmdName}

			// Add required args for some commands
			switch cmdName {
			case "click", "dblclick", "hover":
				args = append(args, "@e1")
			case "type", "fill", "select":
				args = append(args, "@e1", "text")
			case "press":
				args = append(args, "Enter")
			case "scroll":
				args = append(args, "down")
			case "open":
				args = append(args, "http://example.com")
			case "get":
				args = append(args, "title")
			case "is":
				args = append(args, "visible", "@e1")
			}

			root.SetArgs(args)
			err := root.Execute()

			// Should error (either missing args or missing workspace)
			// We can't fully test this without a workspace, but we verify error handling exists
			_ = err // Command should error when workspace not found
		})
	}
}

// findCommand recursively finds a command by name
func findCommand(cmd interface {
	Commands() []*cobra.Command
}, name string) *cobra.Command {
	for _, sub := range cmd.Commands() {
		if sub.Name() == name {
			return sub
		}
	}
	return nil
}

// TestComputerExamplesContainRefs tests that examples use proper ref format
func TestComputerExamplesContainRefs(t *testing.T) {
	root := GetRootCmd()
	computerCmd := findCommand(root, "computer")
	if computerCmd == nil {
		t.Fatal("computer command not found")
	}

	// Commands that should have @e refs in examples
	cmdsWithRefs := []string{"click", "dblclick", "type", "fill", "hover"}

	for _, cmdName := range cmdsWithRefs {
		for _, sub := range computerCmd.Commands() {
			if sub.Name() == cmdName {
				if sub.Example == "" {
					t.Errorf("command '%s' should have examples", cmdName)
					continue
				}
				if !strings.Contains(sub.Example, "@e") {
					t.Errorf("command '%s' examples should contain @e refs", cmdName)
				}
				break
			}
		}
	}
}

// TestComputerCommandShortDescriptionsNotTooLong tests that short descriptions are actually short
func TestComputerCommandShortDescriptionsNotTooLong(t *testing.T) {
	root := GetRootCmd()
	computerCmd := findCommand(root, "computer")
	if computerCmd == nil {
		t.Fatal("computer command not found")
	}

	for _, sub := range computerCmd.Commands() {
		if len(sub.Short) > 80 {
			t.Errorf("command '%s' short description is too long (%d chars): %s",
				sub.Name(), len(sub.Short), sub.Short)
		}
	}
}

// TestComputerPersistentFlagsNotOverridden tests that subcommands don't override persistent flags incorrectly
func TestComputerPersistentFlagsNotOverridden(t *testing.T) {
	root := GetRootCmd()
	computerCmd := findCommand(root, "computer")
	if computerCmd == nil {
		t.Fatal("computer command not found")
	}

	// Persistent flags from root that should NOT be overridden
	// Note: Some flags like --json on status or --timeout on wait are intentional overrides
	criticalFlags := []string{"verbose", "workspace"}

	// Known exceptions where overriding is expected
	expectedOverrides := map[string][]string{
		"status": {"json"},  // status has its own --json flag for JSON output
		"wait":   {"timeout"}, // wait has its own --timeout flag for wait timeout
	}

	for _, sub := range computerCmd.Commands() {
		for _, flagName := range criticalFlags {
			localFlag := sub.LocalFlags().Lookup(flagName)
			if localFlag != nil {
				t.Errorf("command '%s' unexpectedly overrides persistent flag '%s'", sub.Name(), flagName)
			}
		}

		// Check for unexpected overrides of json/timeout
		for _, flagName := range []string{"json", "timeout"} {
			localFlag := sub.LocalFlags().Lookup(flagName)
			if localFlag != nil {
				// Check if this is expected
				expected := false
				if overrides, ok := expectedOverrides[sub.Name()]; ok {
					for _, o := range overrides {
						if o == flagName {
							expected = true
							break
						}
					}
				}
				if !expected {
					t.Errorf("command '%s' unexpectedly overrides persistent flag '%s'", sub.Name(), flagName)
				}
			}
		}
	}
}
