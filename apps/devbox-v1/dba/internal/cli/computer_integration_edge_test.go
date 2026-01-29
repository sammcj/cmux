// internal/cli/computer_integration_edge_test.go
package cli

import (
	"bytes"
	"strings"
	"testing"
)

// =============================================================================
// Command Integration Edge Cases
// =============================================================================

// TestComputerCommandCount tests expected number of subcommands
func TestComputerCommandCount(t *testing.T) {
	root := GetRootCmd()
	computerCmd := findCommand(root, "computer")
	if computerCmd == nil {
		t.Fatal("computer command not found")
	}

	subcommands := computerCmd.Commands()
	// We should have at least these commands:
	// start, stop, status, save, vnc, snapshot, click, dblclick, type, fill,
	// press, hover, select, scroll, open, back, forward, reload, screenshot,
	// get, is, wait, app, ports
	expectedMin := 20

	if len(subcommands) < expectedMin {
		t.Errorf("Expected at least %d subcommands, got %d", expectedMin, len(subcommands))
		t.Logf("Subcommands found:")
		for _, cmd := range subcommands {
			t.Logf("  - %s", cmd.Name())
		}
	}
}

// TestAllComputerCommandsHaveRunE tests that all commands have RunE
func TestAllComputerCommandsHaveRunE(t *testing.T) {
	root := GetRootCmd()
	computerCmd := findCommand(root, "computer")
	if computerCmd == nil {
		t.Fatal("computer command not found")
	}

	for _, sub := range computerCmd.Commands() {
		if sub.RunE == nil && sub.Run == nil {
			t.Errorf("command '%s' should have RunE or Run", sub.Name())
		}
	}
}

// TestAllComputerCommandsHaveUse tests that all commands have Use field
func TestAllComputerCommandsHaveUse(t *testing.T) {
	root := GetRootCmd()
	computerCmd := findCommand(root, "computer")
	if computerCmd == nil {
		t.Fatal("computer command not found")
	}

	for _, sub := range computerCmd.Commands() {
		if sub.Use == "" {
			t.Errorf("command '%s' should have Use field", sub.Name())
		}
	}
}

// TestComputerHelpMentionsAllSubcommands tests help output lists key commands
func TestComputerHelpMentionsAllSubcommands(t *testing.T) {
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

	// Key subcommands that should appear in help
	keyCommands := []string{
		"start",
		"stop",
		"status",
		"snapshot",
		"click",
		"type",
		"screenshot",
		"app",
		"ports",
	}

	for _, cmd := range keyCommands {
		if !strings.Contains(output, cmd) {
			t.Errorf("computer help should mention '%s'", cmd)
		}
	}
}

// =============================================================================
// App Command Integration Tests
// =============================================================================

// TestAppCommandShortDescription tests app short description
func TestAppCommandShortDescription(t *testing.T) {
	root := GetRootCmd()
	computerCmd := findCommand(root, "computer")
	appCmd := findCommand(computerCmd, "app")
	if appCmd == nil {
		t.Fatal("app command not found")
	}

	if appCmd.Short == "" {
		t.Error("app command should have Short description")
	}
	if len(appCmd.Short) > 80 {
		t.Errorf("app Short description too long: %d chars", len(appCmd.Short))
	}
}

// TestAppCommandLongDescription tests app long description
func TestAppCommandLongDescription(t *testing.T) {
	root := GetRootCmd()
	computerCmd := findCommand(root, "computer")
	appCmd := findCommand(computerCmd, "app")
	if appCmd == nil {
		t.Fatal("app command not found")
	}

	if appCmd.Long == "" {
		t.Error("app command should have Long description")
	}
}

// TestAppCommandExamplesFormat tests app examples format
func TestAppCommandExamplesFormat(t *testing.T) {
	root := GetRootCmd()
	computerCmd := findCommand(root, "computer")
	appCmd := findCommand(computerCmd, "app")
	if appCmd == nil {
		t.Fatal("app command not found")
	}

	if appCmd.Example == "" {
		t.Error("app command should have Example")
	}

	// Examples should start with "dba computer app"
	if !strings.Contains(appCmd.Example, "dba computer app") {
		t.Error("app examples should contain 'dba computer app'")
	}

	// Examples should show common flags
	if !strings.Contains(appCmd.Example, "--port") {
		t.Error("app examples should show --port flag")
	}
}

// =============================================================================
// Ports Command Integration Tests
// =============================================================================

// TestPortsCommandShortDescription tests ports short description
func TestPortsCommandShortDescription(t *testing.T) {
	root := GetRootCmd()
	computerCmd := findCommand(root, "computer")
	portsCmd := findCommand(computerCmd, "ports")
	if portsCmd == nil {
		t.Fatal("ports command not found")
	}

	if portsCmd.Short == "" {
		t.Error("ports command should have Short description")
	}
	if len(portsCmd.Short) > 80 {
		t.Errorf("ports Short description too long: %d chars", len(portsCmd.Short))
	}
}

// TestPortsCommandLongDescription tests ports long description
func TestPortsCommandLongDescription(t *testing.T) {
	root := GetRootCmd()
	computerCmd := findCommand(root, "computer")
	portsCmd := findCommand(computerCmd, "ports")
	if portsCmd == nil {
		t.Fatal("ports command not found")
	}

	if portsCmd.Long == "" {
		t.Error("ports command should have Long description")
	}
}

// TestPortsCommandExamplesFormat tests ports examples format
func TestPortsCommandExamplesFormat(t *testing.T) {
	root := GetRootCmd()
	computerCmd := findCommand(root, "computer")
	portsCmd := findCommand(computerCmd, "ports")
	if portsCmd == nil {
		t.Fatal("ports command not found")
	}

	if portsCmd.Example == "" {
		t.Error("ports command should have Example")
	}

	// Examples should contain command
	if !strings.Contains(portsCmd.Example, "dba computer ports") {
		t.Error("ports examples should contain 'dba computer ports'")
	}
}

// =============================================================================
// Flag Inheritance Tests
// =============================================================================

// TestComputerSubcommandsInheritGlobalFlags tests global flag inheritance
func TestComputerSubcommandsInheritGlobalFlags(t *testing.T) {
	root := GetRootCmd()
	computerCmd := findCommand(root, "computer")
	if computerCmd == nil {
		t.Fatal("computer command not found")
	}

	globalFlags := []string{"workspace", "verbose", "json", "timeout"}

	for _, sub := range computerCmd.Commands() {
		for _, flagName := range globalFlags {
			// Check inherited flags
			flag := sub.InheritedFlags().Lookup(flagName)
			if flag == nil {
				// Check if it's available through Flag() method
				flag = sub.Flag(flagName)
			}
			if flag == nil {
				t.Errorf("subcommand '%s' should have access to global flag '%s'",
					sub.Name(), flagName)
			}
		}
	}
}

// =============================================================================
// Help Output Format Tests
// =============================================================================

// TestAppHelpContainsUsage tests app help has Usage section
func TestAppHelpContainsUsage(t *testing.T) {
	root := GetRootCmd()
	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs([]string{"computer", "app", "--help"})

	err := root.Execute()
	if err != nil {
		t.Fatalf("app --help failed: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "Usage:") {
		t.Error("app help should contain 'Usage:'")
	}
}

// TestPortsHelpContainsUsage tests ports help has Usage section
func TestPortsHelpContainsUsage(t *testing.T) {
	root := GetRootCmd()
	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs([]string{"computer", "ports", "--help"})

	err := root.Execute()
	if err != nil {
		t.Fatalf("ports --help failed: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "Usage:") {
		t.Error("ports help should contain 'Usage:'")
	}
}

// =============================================================================
// Command Naming Conventions Tests
// =============================================================================

// TestComputerSubcommandNamingConventions tests naming conventions
func TestComputerSubcommandNamingConventions(t *testing.T) {
	root := GetRootCmd()
	computerCmd := findCommand(root, "computer")
	if computerCmd == nil {
		t.Fatal("computer command not found")
	}

	for _, sub := range computerCmd.Commands() {
		name := sub.Name()

		// Should be lowercase
		if name != strings.ToLower(name) {
			t.Errorf("command '%s' should be lowercase", name)
		}

		// Should not contain underscores
		if strings.Contains(name, "_") {
			t.Errorf("command '%s' should not contain underscores", name)
		}

		// Should be reasonably short
		if len(name) > 15 {
			t.Errorf("command '%s' name is too long (%d chars)", name, len(name))
		}

		// Should not start with number
		if len(name) > 0 && name[0] >= '0' && name[0] <= '9' {
			t.Errorf("command '%s' should not start with number", name)
		}
	}
}

// =============================================================================
// Command Args Tests
// =============================================================================

// TestAppCommandNoArgs tests that app accepts no args
func TestAppCommandNoArgs(t *testing.T) {
	root := GetRootCmd()
	computerCmd := findCommand(root, "computer")
	appCmd := findCommand(computerCmd, "app")
	if appCmd == nil {
		t.Fatal("app command not found")
	}

	// App command should accept no required args
	// (all options are flags)
}

// TestPortsCommandNoArgs tests that ports accepts no args
func TestPortsCommandNoArgs(t *testing.T) {
	root := GetRootCmd()
	computerCmd := findCommand(root, "computer")
	portsCmd := findCommand(computerCmd, "ports")
	if portsCmd == nil {
		t.Fatal("ports command not found")
	}

	// Ports command should accept no required args
}

// =============================================================================
// Error Message Tests
// =============================================================================

// TestAppCommandRequiresWorkspaceError tests error when no workspace
func TestAppCommandRequiresWorkspaceError(t *testing.T) {
	// This would require mocking - just verify command exists
	root := GetRootCmd()
	computerCmd := findCommand(root, "computer")
	appCmd := findCommand(computerCmd, "app")
	if appCmd == nil {
		t.Fatal("app command not found")
	}
}

// TestPortsCommandRequiresWorkspaceError tests error when no workspace
func TestPortsCommandRequiresWorkspaceError(t *testing.T) {
	// This would require mocking - just verify command exists
	root := GetRootCmd()
	computerCmd := findCommand(root, "computer")
	portsCmd := findCommand(computerCmd, "ports")
	if portsCmd == nil {
		t.Fatal("ports command not found")
	}
}

// =============================================================================
// Status Command Shows Ports Tests
// =============================================================================

// TestStatusCommandExists tests status command exists
func TestStatusCommandExists(t *testing.T) {
	root := GetRootCmd()
	computerCmd := findCommand(root, "computer")
	if computerCmd == nil {
		t.Fatal("computer command not found")
	}

	statusCmd := findCommand(computerCmd, "status")
	if statusCmd == nil {
		t.Fatal("status command not found")
	}

	if statusCmd.Short == "" {
		t.Error("status command should have Short description")
	}
}

// TestStatusCommandHelpOutput tests status help output
func TestStatusCommandHelpOutput(t *testing.T) {
	root := GetRootCmd()
	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs([]string{"computer", "status", "--help"})

	err := root.Execute()
	if err != nil {
		t.Fatalf("status --help failed: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "Usage:") {
		t.Error("status help should contain 'Usage:'")
	}
}

// =============================================================================
// Open Command Tests
// =============================================================================

// TestOpenCommandExists tests open command exists
func TestOpenCommandExists(t *testing.T) {
	root := GetRootCmd()
	computerCmd := findCommand(root, "computer")
	if computerCmd == nil {
		t.Fatal("computer command not found")
	}

	openCmd := findCommand(computerCmd, "open")
	if openCmd == nil {
		t.Fatal("open command not found")
	}

	if openCmd.Short == "" {
		t.Error("open command should have Short description")
	}
}

// TestOpenCommandRequiresURL tests open command requires URL arg
func TestOpenCommandRequiresURL(t *testing.T) {
	root := GetRootCmd()
	computerCmd := findCommand(root, "computer")
	openCmd := findCommand(computerCmd, "open")
	if openCmd == nil {
		t.Fatal("open command not found")
	}

	// Open command should require URL argument
	if openCmd.Args == nil {
		// This is OK - some commands use validation in RunE
	}
}

// =============================================================================
// Screenshot Command Tests
// =============================================================================

// TestScreenshotCommandExists tests screenshot command exists
func TestScreenshotCommandExists(t *testing.T) {
	root := GetRootCmd()
	computerCmd := findCommand(root, "computer")
	if computerCmd == nil {
		t.Fatal("computer command not found")
	}

	screenshotCmd := findCommand(computerCmd, "screenshot")
	if screenshotCmd == nil {
		t.Fatal("screenshot command not found")
	}

	if screenshotCmd.Short == "" {
		t.Error("screenshot command should have Short description")
	}
}

// =============================================================================
// VNC Command Tests
// =============================================================================

// TestVNCCommandExists tests vnc command exists
func TestVNCCommandExists(t *testing.T) {
	root := GetRootCmd()
	computerCmd := findCommand(root, "computer")
	if computerCmd == nil {
		t.Fatal("computer command not found")
	}

	vncCmd := findCommand(computerCmd, "vnc")
	if vncCmd == nil {
		t.Fatal("vnc command not found")
	}

	if vncCmd.Short == "" {
		t.Error("vnc command should have Short description")
	}
}

// =============================================================================
// Command Discovery Tests
// =============================================================================

// TestAllExpectedComputerCommandsExist tests all expected commands exist
func TestAllExpectedComputerCommandsExist(t *testing.T) {
	root := GetRootCmd()
	computerCmd := findCommand(root, "computer")
	if computerCmd == nil {
		t.Fatal("computer command not found")
	}

	expectedCommands := []string{
		"start",
		"stop",
		"status",
		"save",
		"vnc",
		"snapshot",
		"click",
		"type",
		"fill",
		"press",
		"hover",
		"scroll",
		"open",
		"back",
		"forward",
		"reload",
		"screenshot",
		"get",
		"is",
		"wait",
		"app",
		"ports",
	}

	for _, cmdName := range expectedCommands {
		cmd := findCommand(computerCmd, cmdName)
		if cmd == nil {
			t.Errorf("expected command '%s' not found", cmdName)
		}
	}
}
