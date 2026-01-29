// internal/cli/computer_vm_test.go
package cli

import (
	"bytes"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestComputerCommandExists(t *testing.T) {
	root := GetRootCmd()

	// Find computer command
	var computerCmd *cobra.Command
	for _, cmd := range root.Commands() {
		if cmd.Use == "computer" {
			computerCmd = cmd
			break
		}
	}

	if computerCmd == nil {
		t.Fatal("computer command not found in root")
	}

	// Verify description
	if computerCmd.Short == "" {
		t.Error("computer command should have short description")
	}
}

func TestComputerSubcommands(t *testing.T) {
	root := GetRootCmd()

	// Find computer command
	var computerCmd *cobra.Command
	for _, cmd := range root.Commands() {
		if cmd.Use == "computer" {
			computerCmd = cmd
			break
		}
	}

	if computerCmd == nil {
		t.Fatal("computer command not found")
	}

	// Expected subcommands
	expectedCmds := []string{
		"start", "stop", "status", "save", "vnc",
		"snapshot", "click", "dblclick", "type", "fill",
		"press", "hover", "select", "scroll",
		"open", "back", "forward", "reload",
		"screenshot", "get", "is", "wait",
	}

	foundCmds := make(map[string]bool)
	for _, cmd := range computerCmd.Commands() {
		foundCmds[cmd.Name()] = true
	}

	for _, expected := range expectedCmds {
		if !foundCmds[expected] {
			t.Errorf("expected subcommand '%s' not found", expected)
		}
	}
}

func TestComputerStartFlags(t *testing.T) {
	root := GetRootCmd()

	var startCmd *cobra.Command
	for _, cmd := range root.Commands() {
		if cmd.Use == "computer" {
			for _, sub := range cmd.Commands() {
				if sub.Name() == "start" {
					startCmd = sub
					break
				}
			}
			break
		}
	}

	if startCmd == nil {
		t.Fatal("start command not found")
	}

	// Check flags
	tests := []struct {
		flagName string
		hasFlag  bool
	}{
		{"snapshot", true},
		{"from", true},
	}

	for _, tt := range tests {
		f := startCmd.Flag(tt.flagName)
		if tt.hasFlag && f == nil {
			t.Errorf("expected flag '%s' not found", tt.flagName)
		}
	}
}

func TestComputerStopFlags(t *testing.T) {
	root := GetRootCmd()

	var stopCmd *cobra.Command
	for _, cmd := range root.Commands() {
		if cmd.Use == "computer" {
			for _, sub := range cmd.Commands() {
				if sub.Name() == "stop" {
					stopCmd = sub
					break
				}
			}
			break
		}
	}

	if stopCmd == nil {
		t.Fatal("stop command not found")
	}

	// Check save flag
	f := stopCmd.Flag("save")
	if f == nil {
		t.Error("expected 'save' flag not found")
	}
}

func TestComputerSaveFlags(t *testing.T) {
	root := GetRootCmd()

	var saveCmd *cobra.Command
	for _, cmd := range root.Commands() {
		if cmd.Use == "computer" {
			for _, sub := range cmd.Commands() {
				if sub.Name() == "save" {
					saveCmd = sub
					break
				}
			}
			break
		}
	}

	if saveCmd == nil {
		t.Fatal("save command not found")
	}

	// Check name flag
	f := saveCmd.Flag("name")
	if f == nil {
		t.Error("expected 'name' flag not found")
	}
}

func TestComputerStatusFlags(t *testing.T) {
	root := GetRootCmd()

	var statusCmd *cobra.Command
	for _, cmd := range root.Commands() {
		if cmd.Use == "computer" {
			for _, sub := range cmd.Commands() {
				if sub.Name() == "status" {
					statusCmd = sub
					break
				}
			}
			break
		}
	}

	if statusCmd == nil {
		t.Fatal("status command not found")
	}

	// Check json flag
	f := statusCmd.Flag("json")
	if f == nil {
		t.Error("expected 'json' flag not found")
	}
}

func TestComputerCommandHelp(t *testing.T) {
	root := GetRootCmd()
	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs([]string{"computer", "--help"})

	err := root.Execute()
	if err != nil {
		t.Errorf("help should not return error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "computer") {
		t.Error("help output should contain 'computer'")
	}
}

func TestComputerStartHelp(t *testing.T) {
	root := GetRootCmd()
	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs([]string{"computer", "start", "--help"})

	err := root.Execute()
	if err != nil {
		t.Errorf("help should not return error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "start") {
		t.Error("help output should contain 'start'")
	}
	if !strings.Contains(output, "--snapshot") {
		t.Error("help output should contain '--snapshot'")
	}
	if !strings.Contains(output, "--from") {
		t.Error("help output should contain '--from'")
	}
}

func TestOpenBrowserFunction(t *testing.T) {
	// Test that openBrowser returns error for unsupported platforms
	// We can't actually test browser opening, but we can test the function exists
	// The function signature is correct if this compiles
	var _ = openBrowser
}

func TestPrintVMStatusFunction(t *testing.T) {
	// Create a mock workspace interface
	type mockWorkspace struct{}

	// Verify the function exists and compiles
	// We can't fully test without a real workspace
	var _ = printVMStatus
}

func TestCommandExamples(t *testing.T) {
	root := GetRootCmd()

	var computerCmd *cobra.Command
	for _, cmd := range root.Commands() {
		if cmd.Use == "computer" {
			computerCmd = cmd
			break
		}
	}

	if computerCmd == nil {
		t.Fatal("computer command not found")
	}

	// Check that key commands have examples
	cmdsWithExamples := []string{"start", "stop", "snapshot", "click", "type", "fill", "open", "wait"}

	for _, cmdName := range cmdsWithExamples {
		for _, sub := range computerCmd.Commands() {
			if sub.Name() == cmdName {
				if sub.Example == "" {
					t.Errorf("command '%s' should have examples", cmdName)
				}
				break
			}
		}
	}
}

func TestCommandLongDescriptions(t *testing.T) {
	root := GetRootCmd()

	var computerCmd *cobra.Command
	for _, cmd := range root.Commands() {
		if cmd.Use == "computer" {
			computerCmd = cmd
			break
		}
	}

	if computerCmd == nil {
		t.Fatal("computer command not found")
	}

	// Check that key commands have long descriptions
	cmdsWithLong := []string{"start", "stop", "snapshot", "click", "press", "get", "wait"}

	for _, cmdName := range cmdsWithLong {
		for _, sub := range computerCmd.Commands() {
			if sub.Name() == cmdName {
				if sub.Long == "" {
					t.Errorf("command '%s' should have long description", cmdName)
				}
				break
			}
		}
	}
}
