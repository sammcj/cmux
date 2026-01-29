// internal/cli/computer_edge_test.go
package cli

import (
	"testing"

	"github.com/spf13/cobra"
)

// TestAllComputerSubcommandsHaveDescriptions ensures all subcommands have proper documentation
func TestAllComputerSubcommandsHaveDescriptions(t *testing.T) {
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

	for _, sub := range computerCmd.Commands() {
		if sub.Short == "" {
			t.Errorf("subcommand '%s' should have short description", sub.Name())
		}
	}
}

// TestComputerCommandsHaveUse ensures all commands have proper Use field
func TestComputerCommandsHaveUse(t *testing.T) {
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

	for _, sub := range computerCmd.Commands() {
		if sub.Use == "" {
			t.Errorf("subcommand '%s' should have Use field", sub.Name())
		}
	}
}

// TestComputerCommandsAreNotHidden ensures browser commands are visible
func TestComputerCommandsAreNotHidden(t *testing.T) {
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

	// Most commands should be visible (not hidden)
	visibleCmds := []string{
		"start", "stop", "status", "snapshot", "click",
		"type", "fill", "press", "open", "screenshot",
		"get", "wait", "scroll", "back", "forward", "reload",
	}

	for _, cmdName := range visibleCmds {
		for _, sub := range computerCmd.Commands() {
			if sub.Name() == cmdName {
				if sub.Hidden {
					t.Errorf("command '%s' should not be hidden", cmdName)
				}
				break
			}
		}
	}
}

// TestComputerCommandsGroupedLogically tests that similar commands exist
func TestComputerCommandsGroupedLogically(t *testing.T) {
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

	// Group 1: VM lifecycle
	vmCmds := []string{"start", "stop", "status", "save", "vnc"}
	for _, cmdName := range vmCmds {
		found := false
		for _, sub := range computerCmd.Commands() {
			if sub.Name() == cmdName {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("VM lifecycle command '%s' not found", cmdName)
		}
	}

	// Group 2: Browser interaction
	interactionCmds := []string{"click", "dblclick", "type", "fill", "press", "hover", "select", "scroll"}
	for _, cmdName := range interactionCmds {
		found := false
		for _, sub := range computerCmd.Commands() {
			if sub.Name() == cmdName {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("browser interaction command '%s' not found", cmdName)
		}
	}

	// Group 3: Navigation
	navCmds := []string{"open", "back", "forward", "reload"}
	for _, cmdName := range navCmds {
		found := false
		for _, sub := range computerCmd.Commands() {
			if sub.Name() == cmdName {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("navigation command '%s' not found", cmdName)
		}
	}

	// Group 4: Information
	infoCmds := []string{"screenshot", "get", "is", "snapshot"}
	for _, cmdName := range infoCmds {
		found := false
		for _, sub := range computerCmd.Commands() {
			if sub.Name() == cmdName {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("information command '%s' not found", cmdName)
		}
	}

	// Group 5: Wait
	waitCmds := []string{"wait"}
	for _, cmdName := range waitCmds {
		found := false
		for _, sub := range computerCmd.Commands() {
			if sub.Name() == cmdName {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("wait command '%s' not found", cmdName)
		}
	}
}

// TestNoCircularDependencies ensures there are no circular command references
func TestNoCircularDependencies(t *testing.T) {
	root := GetRootCmd()

	visited := make(map[*cobra.Command]bool)

	var visit func(*cobra.Command) bool
	visit = func(cmd *cobra.Command) bool {
		if visited[cmd] {
			return true // circular
		}
		visited[cmd] = true

		for _, sub := range cmd.Commands() {
			if visit(sub) {
				return true
			}
		}
		return false
	}

	if visit(root) {
		t.Error("circular dependency detected in command tree")
	}
}

// TestComputerCommandHasParent ensures computer command is properly attached
func TestComputerCommandHasParent(t *testing.T) {
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

	if computerCmd.Parent() != root {
		t.Error("computer command should have root as parent")
	}
}

// TestAllSubcommandsHaveParent ensures all subcommands have computer as parent
func TestAllSubcommandsHaveParent(t *testing.T) {
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

	for _, sub := range computerCmd.Commands() {
		if sub.Parent() != computerCmd {
			t.Errorf("subcommand '%s' should have computer as parent", sub.Name())
		}
	}
}

// TestComputerCommandCountConsistency ensures we have the expected number of commands
func TestComputerCommandCountConsistency(t *testing.T) {
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

	// We expect at least 20 subcommands
	count := len(computerCmd.Commands())
	if count < 20 {
		t.Errorf("expected at least 20 computer subcommands, got %d", count)
	}
}

// TestGlobalFlagsAvailable ensures global flags are inherited
func TestGlobalFlagsAvailable(t *testing.T) {
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

	// Check persistent flags from root
	globalFlags := []string{"json", "verbose", "workspace", "timeout"}

	for _, sub := range computerCmd.Commands() {
		for _, flag := range globalFlags {
			if sub.Flag(flag) == nil && sub.InheritedFlags().Lookup(flag) == nil {
				t.Errorf("subcommand '%s' should have access to global flag '%s'", sub.Name(), flag)
			}
		}
	}
}
