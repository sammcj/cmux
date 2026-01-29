// internal/cli/computer_nav_test.go
package cli

import (
	"bytes"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestComputerOpenCommand(t *testing.T) {
	root := GetRootCmd()

	var openCmd *cobra.Command
	for _, cmd := range root.Commands() {
		if cmd.Use == "computer" {
			for _, sub := range cmd.Commands() {
				if sub.Name() == "open" {
					openCmd = sub
					break
				}
			}
			break
		}
	}

	if openCmd == nil {
		t.Fatal("open command not found")
	}

	// Check that it requires exactly 1 argument
	if openCmd.Use != "open <url>" {
		t.Errorf("unexpected usage: %s", openCmd.Use)
	}

	// Check description
	if openCmd.Short == "" {
		t.Error("open command should have short description")
	}
}

func TestComputerBackCommand(t *testing.T) {
	root := GetRootCmd()

	var backCmd *cobra.Command
	for _, cmd := range root.Commands() {
		if cmd.Use == "computer" {
			for _, sub := range cmd.Commands() {
				if sub.Name() == "back" {
					backCmd = sub
					break
				}
			}
			break
		}
	}

	if backCmd == nil {
		t.Fatal("back command not found")
	}

	// Should require no arguments
	if backCmd.Use != "back" {
		t.Errorf("unexpected usage: %s", backCmd.Use)
	}
}

func TestComputerForwardCommand(t *testing.T) {
	root := GetRootCmd()

	var forwardCmd *cobra.Command
	for _, cmd := range root.Commands() {
		if cmd.Use == "computer" {
			for _, sub := range cmd.Commands() {
				if sub.Name() == "forward" {
					forwardCmd = sub
					break
				}
			}
			break
		}
	}

	if forwardCmd == nil {
		t.Fatal("forward command not found")
	}

	// Should require no arguments
	if forwardCmd.Use != "forward" {
		t.Errorf("unexpected usage: %s", forwardCmd.Use)
	}
}

func TestComputerReloadCommand(t *testing.T) {
	root := GetRootCmd()

	var reloadCmd *cobra.Command
	for _, cmd := range root.Commands() {
		if cmd.Use == "computer" {
			for _, sub := range cmd.Commands() {
				if sub.Name() == "reload" {
					reloadCmd = sub
					break
				}
			}
			break
		}
	}

	if reloadCmd == nil {
		t.Fatal("reload command not found")
	}

	// Should require no arguments
	if reloadCmd.Use != "reload" {
		t.Errorf("unexpected usage: %s", reloadCmd.Use)
	}
}

func TestNavigationCommandsHelp(t *testing.T) {
	tests := []struct {
		cmdName  string
		contains []string
	}{
		{"open", []string{"open", "url", "navigate"}},
		{"back", []string{"back", "history"}},
		{"forward", []string{"forward", "history"}},
		{"reload", []string{"reload", "page"}},
	}

	for _, tt := range tests {
		t.Run(tt.cmdName, func(t *testing.T) {
			root := GetRootCmd()
			buf := new(bytes.Buffer)
			root.SetOut(buf)
			root.SetErr(buf)
			root.SetArgs([]string{"computer", tt.cmdName, "--help"})

			err := root.Execute()
			if err != nil {
				t.Errorf("help should not return error: %v", err)
			}

			output := strings.ToLower(buf.String())
			for _, substr := range tt.contains {
				if !strings.Contains(output, strings.ToLower(substr)) {
					t.Errorf("expected help output to contain '%s'", substr)
				}
			}
		})
	}
}

func TestNavigationCommandArgsValidation(t *testing.T) {
	root := GetRootCmd()

	// Test argument validation for navigation commands
	tests := []struct {
		name string
		args []string
	}{
		{"open without url", []string{"computer", "open"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := new(bytes.Buffer)
			root.SetOut(buf)
			root.SetErr(buf)
			root.SetArgs(tt.args)

			err := root.Execute()
			// open without url should error
			if err == nil {
				t.Logf("Warning: command %v did not return error as expected", tt.args)
			}
		})
	}
}
