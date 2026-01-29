// internal/cli/workspace_test.go
package cli

import (
	"testing"

	"github.com/spf13/cobra"
)

// TestWorkspaceCommands verifies all workspace-related commands are registered
func TestWorkspaceCommands(t *testing.T) {
	cmd := GetRootCmd()

	expectedCommands := []string{"create", "init", "list", "status", "destroy", "clone"}

	for _, name := range expectedCommands {
		found := false
		for _, subCmd := range cmd.Commands() {
			if subCmd.Name() == name {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("command '%s' not found in root command", name)
		}
	}
}

// TestCreateCommandFlags tests that create command has correct flags
func TestCreateCommandFlags(t *testing.T) {
	cmd := GetRootCmd()
	var createCmd *cobra.Command
	for _, c := range cmd.Commands() {
		if c.Name() == "create" {
			createCmd = c
			break
		}
	}

	if createCmd == nil {
		t.Fatal("create command not found")
	}

	// Check required flags
	flags := []struct {
		name      string
		shorthand string
	}{
		{"template", "t"},
		{"clone", "c"},
		{"branch", "b"},
		{"packages", "p"},
		{"ports", ""},
		{"dir", ""},
	}

	for _, f := range flags {
		flag := createCmd.Flags().Lookup(f.name)
		if flag == nil {
			t.Errorf("flag '--%s' not found on create command", f.name)
			continue
		}
		if f.shorthand != "" && flag.Shorthand != f.shorthand {
			t.Errorf("flag '--%s' expected shorthand '-%s', got '-%s'", f.name, f.shorthand, flag.Shorthand)
		}
	}

	// Check that create accepts optional positional arg
	if createCmd.Use != "create [name]" {
		t.Errorf("expected 'create [name]', got '%s'", createCmd.Use)
	}
}

// TestInitCommandFlags tests that init command has correct flags
func TestInitCommandFlags(t *testing.T) {
	cmd := GetRootCmd()
	var initCmd *cobra.Command
	for _, c := range cmd.Commands() {
		if c.Name() == "init" {
			initCmd = c
			break
		}
	}

	if initCmd == nil {
		t.Fatal("init command not found")
	}

	// Check flags
	flags := []struct {
		name      string
		shorthand string
	}{
		{"template", "t"},
		{"name", "n"},
		{"packages", "p"},
		{"ports", ""},
	}

	for _, f := range flags {
		flag := initCmd.Flags().Lookup(f.name)
		if flag == nil {
			t.Errorf("flag '--%s' not found on init command", f.name)
			continue
		}
		if f.shorthand != "" && flag.Shorthand != f.shorthand {
			t.Errorf("flag '--%s' expected shorthand '-%s', got '-%s'", f.name, f.shorthand, flag.Shorthand)
		}
	}

	// Init should not accept positional args
	if initCmd.Use != "init" {
		t.Errorf("expected 'init', got '%s'", initCmd.Use)
	}
}

// TestListCommandFlags tests that list command has correct flags
func TestListCommandFlags(t *testing.T) {
	cmd := GetRootCmd()
	var listCmd *cobra.Command
	for _, c := range cmd.Commands() {
		if c.Name() == "list" {
			listCmd = c
			break
		}
	}

	if listCmd == nil {
		t.Fatal("list command not found")
	}

	// Check --status flag
	statusFlag := listCmd.Flags().Lookup("status")
	if statusFlag == nil {
		t.Error("--status flag not found on list command")
	}
}

// TestDestroyCommandFlags tests that destroy command has correct flags
func TestDestroyCommandFlags(t *testing.T) {
	cmd := GetRootCmd()
	var destroyCmd *cobra.Command
	for _, c := range cmd.Commands() {
		if c.Name() == "destroy" {
			destroyCmd = c
			break
		}
	}

	if destroyCmd == nil {
		t.Fatal("destroy command not found")
	}

	// Check flags
	flags := []struct {
		name      string
		shorthand string
	}{
		{"force", "f"},
		{"keep-files", ""},
	}

	for _, f := range flags {
		flag := destroyCmd.Flags().Lookup(f.name)
		if flag == nil {
			t.Errorf("flag '--%s' not found on destroy command", f.name)
			continue
		}
		if f.shorthand != "" && flag.Shorthand != f.shorthand {
			t.Errorf("flag '--%s' expected shorthand '-%s', got '-%s'", f.name, f.shorthand, flag.Shorthand)
		}
	}

	// Destroy should accept optional positional arg
	if destroyCmd.Use != "destroy [workspace_id]" {
		t.Errorf("expected 'destroy [workspace_id]', got '%s'", destroyCmd.Use)
	}
}

// TestCloneCommandFlags tests that clone command has correct flags
func TestCloneCommandFlags(t *testing.T) {
	cmd := GetRootCmd()
	var cloneCmd *cobra.Command
	for _, c := range cmd.Commands() {
		if c.Name() == "clone" {
			cloneCmd = c
			break
		}
	}

	if cloneCmd == nil {
		t.Fatal("clone command not found")
	}

	// Check --name flag
	nameFlag := cloneCmd.Flags().Lookup("name")
	if nameFlag == nil {
		t.Error("--name flag not found on clone command")
	}
	if nameFlag.Shorthand != "n" {
		t.Errorf("expected -n shorthand for name, got %s", nameFlag.Shorthand)
	}

	// Clone should require source and optionally accept new name
	if cloneCmd.Use != "clone <workspace_id> [new_name]" {
		t.Errorf("expected 'clone <workspace_id> [new_name]', got '%s'", cloneCmd.Use)
	}
}

// TestStatusCommand tests status command exists and has correct structure
func TestStatusCommand(t *testing.T) {
	cmd := GetRootCmd()
	var statusCmd *cobra.Command
	for _, c := range cmd.Commands() {
		if c.Name() == "status" {
			statusCmd = c
			break
		}
	}

	if statusCmd == nil {
		t.Fatal("status command not found")
	}

	if statusCmd.Use != "status" {
		t.Errorf("expected 'status', got '%s'", statusCmd.Use)
	}

	if statusCmd.Short == "" {
		t.Error("status command should have a short description")
	}
}

// TestWorkspaceCommandShortDescriptions verifies all workspace commands have descriptions
func TestWorkspaceCommandShortDescriptions(t *testing.T) {
	cmd := GetRootCmd()

	commands := []string{"create", "init", "list", "status", "destroy", "clone"}

	for _, name := range commands {
		var found *cobra.Command
		for _, c := range cmd.Commands() {
			if c.Name() == name {
				found = c
				break
			}
		}

		if found == nil {
			t.Errorf("command '%s' not found", name)
			continue
		}

		if found.Short == "" {
			t.Errorf("command '%s' has no short description", name)
		}
	}
}

// TestCreateBranchDefaultValue tests that create --branch has correct default
func TestCreateBranchDefaultValue(t *testing.T) {
	cmd := GetRootCmd()
	var createCmd *cobra.Command
	for _, c := range cmd.Commands() {
		if c.Name() == "create" {
			createCmd = c
			break
		}
	}

	if createCmd == nil {
		t.Fatal("create command not found")
	}

	branchFlag := createCmd.Flags().Lookup("branch")
	if branchFlag == nil {
		t.Fatal("--branch flag not found")
	}

	if branchFlag.DefValue != "main" {
		t.Errorf("expected default branch 'main', got '%s'", branchFlag.DefValue)
	}
}

// TestCommandAliases checks that commands could have aliases if defined
func TestCommandAliases(t *testing.T) {
	cmd := GetRootCmd()

	// list should work, check it exists
	var listCmd *cobra.Command
	for _, c := range cmd.Commands() {
		if c.Name() == "list" {
			listCmd = c
			break
		}
	}

	if listCmd == nil {
		t.Error("list command not found")
	}
}

// TestHelpCommandExists verifies help is available
func TestHelpCommandExists(t *testing.T) {
	cmd := GetRootCmd()

	// Run with --help should not error
	cmd.SetArgs([]string{"--help"})
	if err := cmd.Execute(); err != nil {
		t.Errorf("--help failed: %v", err)
	}
}

// TestWorkspaceGlobalFlag tests that -w global flag exists
func TestWorkspaceGlobalFlag(t *testing.T) {
	cmd := GetRootCmd()

	wsFlag := cmd.PersistentFlags().Lookup("workspace")
	if wsFlag == nil {
		t.Error("--workspace flag not found")
	}

	if wsFlag.Shorthand != "w" {
		t.Errorf("expected -w shorthand, got %s", wsFlag.Shorthand)
	}
}

// TestJSONGlobalFlag tests that --json global flag exists
func TestJSONGlobalFlag(t *testing.T) {
	cmd := GetRootCmd()

	jsonFlag := cmd.PersistentFlags().Lookup("json")
	if jsonFlag == nil {
		t.Error("--json flag not found")
	}
}
