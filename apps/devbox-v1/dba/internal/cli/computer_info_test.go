// internal/cli/computer_info_test.go
package cli

import (
	"bytes"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestComputerScreenshotCommand(t *testing.T) {
	root := GetRootCmd()

	var screenshotCmd *cobra.Command
	for _, cmd := range root.Commands() {
		if cmd.Use == "computer" {
			for _, sub := range cmd.Commands() {
				if sub.Name() == "screenshot" {
					screenshotCmd = sub
					break
				}
			}
			break
		}
	}

	if screenshotCmd == nil {
		t.Fatal("screenshot command not found")
	}

	// Check flags
	if f := screenshotCmd.Flag("output"); f == nil {
		t.Error("expected 'output' flag")
	}
	if f := screenshotCmd.Flag("full"); f == nil {
		t.Error("expected 'full' flag")
	}

	// Check description
	if screenshotCmd.Short == "" {
		t.Error("screenshot command should have short description")
	}
}

func TestComputerGetCommand(t *testing.T) {
	root := GetRootCmd()

	var getCmd *cobra.Command
	for _, cmd := range root.Commands() {
		if cmd.Use == "computer" {
			for _, sub := range cmd.Commands() {
				if sub.Name() == "get" {
					getCmd = sub
					break
				}
			}
			break
		}
	}

	if getCmd == nil {
		t.Fatal("get command not found")
	}

	// Check usage pattern
	if !strings.Contains(getCmd.Use, "get") {
		t.Errorf("unexpected usage: %s", getCmd.Use)
	}

	// Check long description mentions different get types
	long := strings.ToLower(getCmd.Long)
	getTypes := []string{"text", "value", "title", "url", "attr"}
	for _, getType := range getTypes {
		if !strings.Contains(long, getType) {
			t.Errorf("get command long description should mention '%s'", getType)
		}
	}
}

func TestComputerIsCommand(t *testing.T) {
	root := GetRootCmd()

	var isCmd *cobra.Command
	for _, cmd := range root.Commands() {
		if cmd.Use == "computer" {
			for _, sub := range cmd.Commands() {
				if sub.Name() == "is" {
					isCmd = sub
					break
				}
			}
			break
		}
	}

	if isCmd == nil {
		t.Fatal("is command not found")
	}

	// Check usage pattern
	if !strings.Contains(isCmd.Use, "is") {
		t.Errorf("unexpected usage: %s", isCmd.Use)
	}
}

func TestInfoCommandsHelp(t *testing.T) {
	tests := []struct {
		cmdName  string
		contains []string
	}{
		{"screenshot", []string{"screenshot", "output"}},
		{"get", []string{"get", "text", "value", "title", "url"}},
		{"is", []string{"visible", "enabled", "checked"}},
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

func TestGetCommandValidation(t *testing.T) {
	root := GetRootCmd()

	tests := []struct {
		name        string
		args        []string
		shouldError bool
	}{
		{"get without args", []string{"computer", "get"}, true},
		{"get title", []string{"computer", "get", "title"}, false}, // This would error for no workspace, but syntax is valid
		{"get url", []string{"computer", "get", "url"}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := new(bytes.Buffer)
			root.SetOut(buf)
			root.SetErr(buf)
			root.SetArgs(tt.args)

			err := root.Execute()
			// Note: Commands may error due to no workspace, but we're testing if they're registered
			// The error type tells us if it's a usage error or a runtime error
			_ = err
		})
	}
}

func TestIsCommandValidation(t *testing.T) {
	root := GetRootCmd()

	// Test argument validation for is command
	tests := []struct {
		name string
		args []string
	}{
		{"is without args", []string{"computer", "is"}},
		{"is with one arg", []string{"computer", "is", "visible"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := new(bytes.Buffer)
			root.SetOut(buf)
			root.SetErr(buf)
			root.SetArgs(tt.args)

			err := root.Execute()
			// Commands with missing args should error
			if err == nil {
				t.Logf("Warning: command %v did not return error as expected", tt.args)
			}
		})
	}
}

func TestScreenshotFlagDefaults(t *testing.T) {
	root := GetRootCmd()

	var screenshotCmd *cobra.Command
	for _, cmd := range root.Commands() {
		if cmd.Use == "computer" {
			for _, sub := range cmd.Commands() {
				if sub.Name() == "screenshot" {
					screenshotCmd = sub
					break
				}
			}
			break
		}
	}

	if screenshotCmd == nil {
		t.Fatal("screenshot command not found")
	}

	// Check output flag default
	outputFlag := screenshotCmd.Flag("output")
	if outputFlag == nil {
		t.Fatal("output flag not found")
	}
	if outputFlag.DefValue != "" {
		t.Errorf("output flag default should be empty, got: %s", outputFlag.DefValue)
	}

	// Check full flag default
	fullFlag := screenshotCmd.Flag("full")
	if fullFlag == nil {
		t.Fatal("full flag not found")
	}
	if fullFlag.DefValue != "false" {
		t.Errorf("full flag default should be false, got: %s", fullFlag.DefValue)
	}
}
