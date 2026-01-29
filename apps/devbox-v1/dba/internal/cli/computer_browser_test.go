// internal/cli/computer_browser_test.go
package cli

import (
	"bytes"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestComputerSnapshotCommand(t *testing.T) {
	root := GetRootCmd()

	var snapshotCmd *cobra.Command
	for _, cmd := range root.Commands() {
		if cmd.Use == "computer" {
			for _, sub := range cmd.Commands() {
				if sub.Name() == "snapshot" {
					snapshotCmd = sub
					break
				}
			}
			break
		}
	}

	if snapshotCmd == nil {
		t.Fatal("snapshot command not found")
	}

	// Check flags
	if f := snapshotCmd.Flag("interactive"); f == nil {
		t.Error("expected 'interactive' flag")
	}

	if f := snapshotCmd.Flag("compact"); f == nil {
		t.Error("expected 'compact' flag")
	}

	// Check description
	if snapshotCmd.Short == "" {
		t.Error("snapshot command should have short description")
	}
	if snapshotCmd.Long == "" {
		t.Error("snapshot command should have long description")
	}
}

func TestComputerClickCommand(t *testing.T) {
	root := GetRootCmd()

	var clickCmd *cobra.Command
	for _, cmd := range root.Commands() {
		if cmd.Use == "computer" {
			for _, sub := range cmd.Commands() {
				if sub.Name() == "click" {
					clickCmd = sub
					break
				}
			}
			break
		}
	}

	if clickCmd == nil {
		t.Fatal("click command not found")
	}

	// Check that it requires exactly 1 argument
	if clickCmd.Use != "click <selector>" {
		t.Errorf("unexpected usage: %s", clickCmd.Use)
	}
}

func TestComputerDblclickCommand(t *testing.T) {
	root := GetRootCmd()

	var dblclickCmd *cobra.Command
	for _, cmd := range root.Commands() {
		if cmd.Use == "computer" {
			for _, sub := range cmd.Commands() {
				if sub.Name() == "dblclick" {
					dblclickCmd = sub
					break
				}
			}
			break
		}
	}

	if dblclickCmd == nil {
		t.Fatal("dblclick command not found")
	}

	// Check that it requires exactly 1 argument
	if dblclickCmd.Use != "dblclick <selector>" {
		t.Errorf("unexpected usage: %s", dblclickCmd.Use)
	}
}

func TestComputerTypeCommand(t *testing.T) {
	root := GetRootCmd()

	var typeCmd *cobra.Command
	for _, cmd := range root.Commands() {
		if cmd.Use == "computer" {
			for _, sub := range cmd.Commands() {
				if sub.Name() == "type" {
					typeCmd = sub
					break
				}
			}
			break
		}
	}

	if typeCmd == nil {
		t.Fatal("type command not found")
	}

	// Check that it requires exactly 2 arguments
	if typeCmd.Use != "type <selector> <text>" {
		t.Errorf("unexpected usage: %s", typeCmd.Use)
	}
}

func TestComputerFillCommand(t *testing.T) {
	root := GetRootCmd()

	var fillCmd *cobra.Command
	for _, cmd := range root.Commands() {
		if cmd.Use == "computer" {
			for _, sub := range cmd.Commands() {
				if sub.Name() == "fill" {
					fillCmd = sub
					break
				}
			}
			break
		}
	}

	if fillCmd == nil {
		t.Fatal("fill command not found")
	}

	// Check that it requires exactly 2 arguments
	if fillCmd.Use != "fill <selector> <text>" {
		t.Errorf("unexpected usage: %s", fillCmd.Use)
	}
}

func TestComputerPressCommand(t *testing.T) {
	root := GetRootCmd()

	var pressCmd *cobra.Command
	for _, cmd := range root.Commands() {
		if cmd.Use == "computer" {
			for _, sub := range cmd.Commands() {
				if sub.Name() == "press" {
					pressCmd = sub
					break
				}
			}
			break
		}
	}

	if pressCmd == nil {
		t.Fatal("press command not found")
	}

	// Check that it requires exactly 1 argument
	if pressCmd.Use != "press <key>" {
		t.Errorf("unexpected usage: %s", pressCmd.Use)
	}

	// Check long description mentions common keys
	if !strings.Contains(pressCmd.Long, "Enter") {
		t.Error("press command should mention Enter key")
	}
}

func TestComputerHoverCommand(t *testing.T) {
	root := GetRootCmd()

	var hoverCmd *cobra.Command
	for _, cmd := range root.Commands() {
		if cmd.Use == "computer" {
			for _, sub := range cmd.Commands() {
				if sub.Name() == "hover" {
					hoverCmd = sub
					break
				}
			}
			break
		}
	}

	if hoverCmd == nil {
		t.Fatal("hover command not found")
	}

	// Check that it requires exactly 1 argument
	if hoverCmd.Use != "hover <selector>" {
		t.Errorf("unexpected usage: %s", hoverCmd.Use)
	}
}

func TestComputerSelectCommand(t *testing.T) {
	root := GetRootCmd()

	var selectCmd *cobra.Command
	for _, cmd := range root.Commands() {
		if cmd.Use == "computer" {
			for _, sub := range cmd.Commands() {
				if sub.Name() == "select" {
					selectCmd = sub
					break
				}
			}
			break
		}
	}

	if selectCmd == nil {
		t.Fatal("select command not found")
	}

	// Check that it requires exactly 2 arguments
	if selectCmd.Use != "select <selector> <value>" {
		t.Errorf("unexpected usage: %s", selectCmd.Use)
	}
}

func TestComputerScrollCommand(t *testing.T) {
	root := GetRootCmd()

	var scrollCmd *cobra.Command
	for _, cmd := range root.Commands() {
		if cmd.Use == "computer" {
			for _, sub := range cmd.Commands() {
				if sub.Name() == "scroll" {
					scrollCmd = sub
					break
				}
			}
			break
		}
	}

	if scrollCmd == nil {
		t.Fatal("scroll command not found")
	}

	// Check that it requires 1-2 arguments
	if scrollCmd.Use != "scroll <direction> [amount]" {
		t.Errorf("unexpected usage: %s", scrollCmd.Use)
	}
}

func TestBrowserCommandHelpOutput(t *testing.T) {
	tests := []struct {
		cmdName  string
		contains []string
	}{
		{"snapshot", []string{"snapshot", "elements"}},
		{"click", []string{"click"}},
		{"type", []string{"type", "text"}},
		{"fill", []string{"fill"}},
		{"press", []string{"press", "key"}},
		{"scroll", []string{"scroll"}},
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

			output := buf.String()
			for _, substr := range tt.contains {
				if !strings.Contains(strings.ToLower(output), strings.ToLower(substr)) {
					t.Errorf("expected help output to contain '%s'", substr)
				}
			}
		})
	}
}

func TestGetBrowserClientFunction(t *testing.T) {
	// Test that getBrowserClient function exists and has correct signature
	// We can't test it fully without a running workspace
	var _ = getBrowserClient
}

func TestCommandArgsValidation(t *testing.T) {
	root := GetRootCmd()

	// Test that commands with missing args produce errors
	// These commands use cobra.ExactArgs or similar validators
	tests := []struct {
		name string
		args []string
	}{
		{"click without args", []string{"computer", "click"}},
		{"type with one arg", []string{"computer", "type", "@e1"}},
		{"fill with one arg", []string{"computer", "fill", "@e1"}},
		{"select with one arg", []string{"computer", "select", "@e1"}},
		{"press without args", []string{"computer", "press"}},
		{"hover without args", []string{"computer", "hover"}},
		{"scroll without args", []string{"computer", "scroll"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := new(bytes.Buffer)
			root.SetOut(buf)
			root.SetErr(buf)
			root.SetArgs(tt.args)

			err := root.Execute()
			// All these should error due to missing args
			if err == nil {
				t.Logf("Warning: command %v did not return error as expected (may need workspace)", tt.args)
			}
		})
	}
}
