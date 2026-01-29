// internal/cli/computer_help_test.go
package cli

import (
	"bytes"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// TestAllComputerCommandsHaveShortDescription tests short descriptions
func TestAllComputerCommandsHaveShortDescription(t *testing.T) {
	root := GetRootCmd()
	computerCmd := findCommand(root, "computer")
	if computerCmd == nil {
		t.Fatal("computer command not found")
	}

	for _, sub := range computerCmd.Commands() {
		t.Run(sub.Name(), func(t *testing.T) {
			if sub.Short == "" {
				t.Errorf("command '%s' should have short description", sub.Name())
			}
			if len(sub.Short) > 80 {
				t.Errorf("command '%s' short description too long (%d chars)",
					sub.Name(), len(sub.Short))
			}
		})
	}
}

// TestAllComputerCommandsHaveUsage tests usage strings
func TestAllComputerCommandsHaveUsage(t *testing.T) {
	root := GetRootCmd()
	computerCmd := findCommand(root, "computer")
	if computerCmd == nil {
		t.Fatal("computer command not found")
	}

	for _, sub := range computerCmd.Commands() {
		t.Run(sub.Name(), func(t *testing.T) {
			if sub.Use == "" {
				t.Errorf("command '%s' should have Use field", sub.Name())
			}
			// Use should start with command name
			if !strings.HasPrefix(sub.Use, sub.Name()) {
				t.Errorf("command '%s' Use should start with command name", sub.Name())
			}
		})
	}
}

// TestComputerCommandHelpOutput tests help output format
func TestComputerCommandHelpOutput(t *testing.T) {
	root := GetRootCmd()
	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs([]string{"computer", "--help"})

	err := root.Execute()
	if err != nil {
		t.Errorf("computer --help failed: %v", err)
	}

	output := buf.String()

	// Should contain standard sections
	sections := []string{
		"Usage:",
		"Available Commands:",
		"Flags:",
	}

	for _, section := range sections {
		if !strings.Contains(output, section) {
			t.Errorf("help output should contain '%s'", section)
		}
	}
}

// TestComputerSubcommandHelpOutputs tests each subcommand's help
func TestComputerSubcommandHelpOutputs(t *testing.T) {
	root := GetRootCmd()
	computerCmd := findCommand(root, "computer")
	if computerCmd == nil {
		t.Fatal("computer command not found")
	}

	for _, sub := range computerCmd.Commands() {
		t.Run(sub.Name(), func(t *testing.T) {
			buf := new(bytes.Buffer)
			root.SetOut(buf)
			root.SetErr(buf)
			root.SetArgs([]string{"computer", sub.Name(), "--help"})

			err := root.Execute()
			if err != nil {
				t.Errorf("help for '%s' failed: %v", sub.Name(), err)
				return
			}

			output := buf.String()

			// Should contain Usage section
			if !strings.Contains(output, "Usage:") {
				t.Errorf("help for '%s' should contain 'Usage:'", sub.Name())
			}
		})
	}
}

// TestComputerCommandLongDescriptions tests long descriptions
func TestComputerCommandLongDescriptions(t *testing.T) {
	root := GetRootCmd()
	computerCmd := findCommand(root, "computer")
	if computerCmd == nil {
		t.Fatal("computer command not found")
	}

	// Commands that should have long descriptions (key commands)
	expectLong := []string{
		"start", "stop", "status", "snapshot", "click",
		"screenshot", "get", "wait", "press",
	}

	for _, cmdName := range expectLong {
		for _, sub := range computerCmd.Commands() {
			if sub.Name() == cmdName {
				t.Run(cmdName, func(t *testing.T) {
					if sub.Long == "" {
						t.Errorf("command '%s' should have long description", cmdName)
					}
				})
				break
			}
		}
	}
}

// TestComputerCommandExamples tests command examples
func TestComputerCommandExamples(t *testing.T) {
	root := GetRootCmd()
	computerCmd := findCommand(root, "computer")
	if computerCmd == nil {
		t.Fatal("computer command not found")
	}

	// Commands that should have examples
	expectExamples := []string{
		"start", "click", "type", "fill", "open", "screenshot",
	}

	for _, cmdName := range expectExamples {
		for _, sub := range computerCmd.Commands() {
			if sub.Name() == cmdName {
				t.Run(cmdName, func(t *testing.T) {
					if sub.Example == "" {
						t.Errorf("command '%s' should have examples", cmdName)
					}
				})
				break
			}
		}
	}
}

// TestComputerCommandExampleFormat tests example format
func TestComputerCommandExampleFormat(t *testing.T) {
	root := GetRootCmd()
	computerCmd := findCommand(root, "computer")
	if computerCmd == nil {
		t.Fatal("computer command not found")
	}

	for _, sub := range computerCmd.Commands() {
		if sub.Example != "" {
			t.Run(sub.Name(), func(t *testing.T) {
				// Examples should contain "dba computer"
				if !strings.Contains(sub.Example, "dba computer") {
					t.Errorf("example for '%s' should contain 'dba computer'", sub.Name())
				}
			})
		}
	}
}

// TestComputerCommandFlagDescriptions tests flag descriptions
func TestComputerCommandFlagDescriptions(t *testing.T) {
	root := GetRootCmd()
	computerCmd := findCommand(root, "computer")
	if computerCmd == nil {
		t.Fatal("computer command not found")
	}

	for _, sub := range computerCmd.Commands() {
		t.Run(sub.Name(), func(t *testing.T) {
			sub.LocalFlags().VisitAll(func(flag *pflag.Flag) {
				if flag.Usage == "" {
					t.Errorf("flag '%s' in '%s' should have usage description",
						flag.Name, sub.Name())
				}
			})
		})
	}
}

// TestComputerCommandsConsistentNaming tests consistent command naming
func TestComputerCommandsConsistentNaming(t *testing.T) {
	root := GetRootCmd()
	computerCmd := findCommand(root, "computer")
	if computerCmd == nil {
		t.Fatal("computer command not found")
	}

	for _, sub := range computerCmd.Commands() {
		t.Run(sub.Name(), func(t *testing.T) {
			name := sub.Name()

			// Names should be lowercase
			if name != strings.ToLower(name) {
				t.Errorf("command name '%s' should be lowercase", name)
			}

			// Names should not have underscores
			if strings.Contains(name, "_") {
				t.Errorf("command name '%s' should use hyphens not underscores", name)
			}

			// Names should be reasonably short
			if len(name) > 15 {
				t.Errorf("command name '%s' is too long (%d chars)", name, len(name))
			}
		})
	}
}

// TestComputerCommandAliases tests command aliases if any
func TestComputerCommandAliases(t *testing.T) {
	root := GetRootCmd()
	computerCmd := findCommand(root, "computer")
	if computerCmd == nil {
		t.Fatal("computer command not found")
	}

	// Log aliases for documentation purposes
	for _, sub := range computerCmd.Commands() {
		if len(sub.Aliases) > 0 {
			t.Logf("command '%s' has aliases: %v", sub.Name(), sub.Aliases)
		}
	}
}

// TestHelpTextNoTrailingWhitespace tests no trailing whitespace in help
func TestHelpTextNoTrailingWhitespace(t *testing.T) {
	root := GetRootCmd()
	computerCmd := findCommand(root, "computer")
	if computerCmd == nil {
		t.Fatal("computer command not found")
	}

	for _, sub := range computerCmd.Commands() {
		t.Run(sub.Name(), func(t *testing.T) {
			// Check Short
			if strings.HasSuffix(sub.Short, " ") {
				t.Errorf("Short for '%s' has trailing whitespace", sub.Name())
			}
			// Check Long
			if strings.HasSuffix(sub.Long, " ") {
				t.Errorf("Long for '%s' has trailing whitespace", sub.Name())
			}
		})
	}
}

// TestHelpTextNoLeadingWhitespace tests no leading whitespace
func TestHelpTextNoLeadingWhitespace(t *testing.T) {
	root := GetRootCmd()
	computerCmd := findCommand(root, "computer")
	if computerCmd == nil {
		t.Fatal("computer command not found")
	}

	for _, sub := range computerCmd.Commands() {
		t.Run(sub.Name(), func(t *testing.T) {
			// Check Short
			if strings.HasPrefix(sub.Short, " ") {
				t.Errorf("Short for '%s' has leading whitespace", sub.Name())
			}
			// Check Long (first character)
			if len(sub.Long) > 0 && sub.Long[0] == ' ' {
				t.Errorf("Long for '%s' has leading whitespace", sub.Name())
			}
		})
	}
}

// TestHelpTextCapitalization tests proper capitalization
func TestHelpTextCapitalization(t *testing.T) {
	root := GetRootCmd()
	computerCmd := findCommand(root, "computer")
	if computerCmd == nil {
		t.Fatal("computer command not found")
	}

	for _, sub := range computerCmd.Commands() {
		if sub.Short != "" {
			t.Run(sub.Name()+"_short", func(t *testing.T) {
				// Short description should start with capital letter
				firstChar := sub.Short[0]
				if firstChar >= 'a' && firstChar <= 'z' {
					t.Errorf("Short for '%s' should start with capital letter", sub.Name())
				}
			})
		}
	}
}

// TestHelpTextNoPeriodEnding tests short descriptions don't end with period
func TestHelpTextNoPeriodEnding(t *testing.T) {
	root := GetRootCmd()
	computerCmd := findCommand(root, "computer")
	if computerCmd == nil {
		t.Fatal("computer command not found")
	}

	for _, sub := range computerCmd.Commands() {
		if sub.Short != "" {
			t.Run(sub.Name(), func(t *testing.T) {
				// Short descriptions typically don't end with period
				if strings.HasSuffix(sub.Short, ".") {
					t.Logf("Short for '%s' ends with period (style note)", sub.Name())
				}
			})
		}
	}
}

// TestComputerHelpMentionsSubcommands tests computer help lists subcommands
func TestComputerHelpMentionsSubcommands(t *testing.T) {
	root := GetRootCmd()
	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs([]string{"computer", "--help"})

	err := root.Execute()
	if err != nil {
		t.Fatalf("computer --help failed: %v", err)
	}

	output := buf.String()

	// Important subcommands should be mentioned
	commands := []string{
		"start", "stop", "status", "snapshot", "click",
		"type", "open", "screenshot", "wait",
	}

	for _, cmd := range commands {
		if !strings.Contains(output, cmd) {
			t.Errorf("computer help should mention '%s'", cmd)
		}
	}
}

// TestComputerSubcommandCount tests expected number of subcommands
func TestComputerSubcommandCount(t *testing.T) {
	root := GetRootCmd()
	computerCmd := findCommand(root, "computer")
	if computerCmd == nil {
		t.Fatal("computer command not found")
	}

	count := len(computerCmd.Commands())
	t.Logf("Total computer subcommands: %d", count)

	// We expect at least 20 subcommands
	if count < 20 {
		t.Errorf("expected at least 20 subcommands, got %d", count)
	}
}

// findCommand helper is defined in computer_integration_test.go
// If not found, define it here
func findCommandByName(cmd *cobra.Command, name string) *cobra.Command {
	for _, sub := range cmd.Commands() {
		if sub.Name() == name {
			return sub
		}
	}
	return nil
}
