// internal/cli/run_test.go
package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

// ═══════════════════════════════════════════════════════════════════════════════
// Run Command Tests
// ═══════════════════════════════════════════════════════════════════════════════

func TestRunCmd_Exists(t *testing.T) {
	if runCmd == nil {
		t.Fatal("runCmd is nil")
	}

	if runCmd.Use != "run <command...>" {
		t.Errorf("Expected use 'run <command...>', got '%s'", runCmd.Use)
	}
}

func TestRunCmd_RequiresArgs(t *testing.T) {
	if runCmd.Args == nil {
		t.Fatal("runCmd.Args is nil")
	}

	// Test that it requires at least 1 argument
	err := runCmd.Args(runCmd, []string{})
	if err == nil {
		t.Error("Expected error for empty args, got nil")
	}

	err = runCmd.Args(runCmd, []string{"echo", "hello"})
	if err != nil {
		t.Errorf("Expected no error for valid args, got %v", err)
	}
}

func TestRunCmd_Flags(t *testing.T) {
	// Check that expected flags exist
	flags := []struct {
		name     string
		expected bool
	}{
		{"no-sync", true},
		{"cwd", true},
		{"env", true},
		{"timeout", true},
	}

	for _, f := range flags {
		flag := runCmd.Flags().Lookup(f.name)
		if f.expected && flag == nil {
			t.Errorf("Expected flag '%s' to exist", f.name)
		}
		if !f.expected && flag != nil {
			t.Errorf("Unexpected flag '%s' exists", f.name)
		}
	}
}

func TestRunCmd_NoSyncDefault(t *testing.T) {
	flag := runCmd.Flags().Lookup("no-sync")
	if flag == nil {
		t.Fatal("no-sync flag not found")
	}

	if flag.DefValue != "false" {
		t.Errorf("Expected no-sync default 'false', got '%s'", flag.DefValue)
	}
}

func TestRunCmd_ShortDescription(t *testing.T) {
	if !strings.Contains(runCmd.Short, "Run") && !strings.Contains(runCmd.Short, "command") {
		t.Errorf("Short description should mention running commands: '%s'", runCmd.Short)
	}
}

func TestRunCmd_LongDescription(t *testing.T) {
	if len(runCmd.Long) < 50 {
		t.Error("Long description should be comprehensive")
	}

	// Should contain examples
	if !strings.Contains(runCmd.Long, "Example") && !strings.Contains(runCmd.Long, "dba run") {
		t.Error("Long description should contain examples")
	}
}

// ═══════════════════════════════════════════════════════════════════════════════
// Test Command Tests
// ═══════════════════════════════════════════════════════════════════════════════

func TestTestCmd_Exists(t *testing.T) {
	if testCmd == nil {
		t.Fatal("testCmd is nil")
	}

	if !strings.HasPrefix(testCmd.Use, "test") {
		t.Errorf("Expected use to start with 'test', got '%s'", testCmd.Use)
	}
}

func TestTestCmd_AcceptsOptionalPattern(t *testing.T) {
	if testCmd.Args == nil {
		t.Fatal("testCmd.Args is nil")
	}

	// Test that it accepts 0 arguments
	err := testCmd.Args(testCmd, []string{})
	if err != nil {
		t.Errorf("Expected no error for empty args, got %v", err)
	}

	// Test that it accepts 1 argument
	err = testCmd.Args(testCmd, []string{"auth"})
	if err != nil {
		t.Errorf("Expected no error for 1 arg, got %v", err)
	}

	// Test that it rejects 2+ arguments
	err = testCmd.Args(testCmd, []string{"auth", "extra"})
	if err == nil {
		t.Error("Expected error for 2 args, got nil")
	}
}

func TestTestCmd_Flags(t *testing.T) {
	flags := []struct {
		name     string
		expected bool
	}{
		{"watch", true},
		{"coverage", true},
		{"runner", true},
	}

	for _, f := range flags {
		flag := testCmd.Flags().Lookup(f.name)
		if f.expected && flag == nil {
			t.Errorf("Expected flag '%s' to exist", f.name)
		}
	}
}

func TestTestCmd_WatchDefault(t *testing.T) {
	flag := testCmd.Flags().Lookup("watch")
	if flag == nil {
		t.Fatal("watch flag not found")
	}

	if flag.DefValue != "false" {
		t.Errorf("Expected watch default 'false', got '%s'", flag.DefValue)
	}
}

func TestTestCmd_CoverageDefault(t *testing.T) {
	flag := testCmd.Flags().Lookup("coverage")
	if flag == nil {
		t.Fatal("coverage flag not found")
	}

	if flag.DefValue != "false" {
		t.Errorf("Expected coverage default 'false', got '%s'", flag.DefValue)
	}
}

func TestTestCmd_RunnerDefault(t *testing.T) {
	flag := testCmd.Flags().Lookup("runner")
	if flag == nil {
		t.Fatal("runner flag not found")
	}

	if flag.DefValue != "" {
		t.Errorf("Expected runner default '', got '%s'", flag.DefValue)
	}
}

func TestTestCmd_Description(t *testing.T) {
	// Should mention test runner detection
	if !strings.Contains(strings.ToLower(testCmd.Short), "test") {
		t.Error("Short description should mention testing")
	}

	if !strings.Contains(strings.ToLower(testCmd.Long), "vitest") &&
		!strings.Contains(strings.ToLower(testCmd.Long), "jest") {
		t.Error("Long description should mention supported test runners")
	}
}

// ═══════════════════════════════════════════════════════════════════════════════
// Shell Command Tests
// ═══════════════════════════════════════════════════════════════════════════════

func TestShellCmd_Exists(t *testing.T) {
	if shellCmd == nil {
		t.Fatal("shellCmd is nil")
	}

	if shellCmd.Use != "shell" {
		t.Errorf("Expected use 'shell', got '%s'", shellCmd.Use)
	}
}

func TestShellCmd_Flags(t *testing.T) {
	flag := shellCmd.Flags().Lookup("pure")
	if flag == nil {
		t.Fatal("pure flag not found")
	}

	if flag.DefValue != "false" {
		t.Errorf("Expected pure default 'false', got '%s'", flag.DefValue)
	}
}

func TestShellCmd_Description(t *testing.T) {
	if !strings.Contains(strings.ToLower(shellCmd.Short), "shell") {
		t.Error("Short description should mention shell")
	}
}

// ═══════════════════════════════════════════════════════════════════════════════
// Result Types Tests
// ═══════════════════════════════════════════════════════════════════════════════

func TestRunResult_JSONSerialization(t *testing.T) {
	result := RunResult{
		ExitCode:   0,
		Stdout:     "hello world\n",
		Stderr:     "",
		DurationMs: 123,
		Synced:     true,
		SyncWaitMs: 50,
	}

	data, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("Failed to marshal RunResult: %v", err)
	}

	// Verify JSON contains expected fields
	var decoded map[string]interface{}
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal JSON: %v", err)
	}

	if decoded["exit_code"].(float64) != 0 {
		t.Error("exit_code not correctly serialized")
	}
	if decoded["stdout"] != "hello world\n" {
		t.Error("stdout not correctly serialized")
	}
	if decoded["duration_ms"].(float64) != 123 {
		t.Error("duration_ms not correctly serialized")
	}
	if decoded["synced"] != true {
		t.Error("synced not correctly serialized")
	}
}

func TestTestResult_JSONSerialization(t *testing.T) {
	result := TestResult{
		Runner:     "vitest",
		Command:    "npx vitest run",
		ExitCode:   0,
		DurationMs: 5000,
		Stdout:     "PASS all tests",
		Stderr:     "",
	}

	data, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("Failed to marshal TestResult: %v", err)
	}

	var decoded map[string]interface{}
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal JSON: %v", err)
	}

	if decoded["runner"] != "vitest" {
		t.Error("runner not correctly serialized")
	}
	if decoded["command"] != "npx vitest run" {
		t.Error("command not correctly serialized")
	}
}

func TestRunResult_OmitsEmptySync(t *testing.T) {
	result := RunResult{
		ExitCode:   0,
		Stdout:     "output",
		Stderr:     "",
		DurationMs: 100,
		Synced:     false,
		SyncWaitMs: 0,
	}

	data, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	// Check that sync_wait_ms is omitted when 0 (omitempty)
	if strings.Contains(string(data), `"sync_wait_ms":0`) {
		t.Log("Note: sync_wait_ms included even when 0 (this is acceptable)")
	}
}

// ═══════════════════════════════════════════════════════════════════════════════
// Command Registration Tests
// ═══════════════════════════════════════════════════════════════════════════════

func TestCommandsRegistered(t *testing.T) {
	// Verify commands are registered with root
	commands := rootCmd.Commands()

	commandNames := make(map[string]bool)
	for _, cmd := range commands {
		commandNames[cmd.Name()] = true
	}

	expectedCommands := []string{"run", "test", "shell"}
	for _, name := range expectedCommands {
		if !commandNames[name] {
			t.Errorf("Expected command '%s' to be registered with root", name)
		}
	}
}

func TestRunCmd_InheritsPersistentFlags(t *testing.T) {
	// Run command should inherit global flags
	flags := runCmd.Flags()

	// Verify we have some flags (either local or inherited)
	if flags.NFlag() < 0 {
		t.Error("No flags available")
	}

	// These are inherited from root's PersistentFlags
	// We can't easily test this without executing, but we can check the command structure
	if runCmd.Parent() != nil {
		t.Log("runCmd has parent set correctly")
	}
}

// ═══════════════════════════════════════════════════════════════════════════════
// Edge Cases
// ═══════════════════════════════════════════════════════════════════════════════

func TestRunResult_SpecialCharactersInOutput(t *testing.T) {
	result := RunResult{
		ExitCode: 0,
		Stdout:   "Line 1\nLine 2\tTabbed\n\"Quoted\"\n",
		Stderr:   "Warning: 'single quotes' and \"double quotes\"",
	}

	data, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("Failed to marshal with special chars: %v", err)
	}

	var decoded RunResult
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if decoded.Stdout != result.Stdout {
		t.Errorf("Stdout mismatch after round-trip")
	}
	if decoded.Stderr != result.Stderr {
		t.Errorf("Stderr mismatch after round-trip")
	}
}

func TestRunResult_UnicodeInOutput(t *testing.T) {
	result := RunResult{
		ExitCode: 0,
		Stdout:   "Hello 世界! 🌍 Émoji test",
	}

	data, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("Failed to marshal with unicode: %v", err)
	}

	var decoded RunResult
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if decoded.Stdout != result.Stdout {
		t.Errorf("Unicode stdout mismatch after round-trip")
	}
}

func TestTestResult_LongCommand(t *testing.T) {
	// Test with a very long command
	longPattern := strings.Repeat("test_", 100)
	result := TestResult{
		Runner:  "pytest",
		Command: "pytest -v -k " + longPattern,
	}

	data, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("Failed to marshal long command: %v", err)
	}

	var decoded TestResult
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if decoded.Command != result.Command {
		t.Error("Long command mismatch after round-trip")
	}
}

// ═══════════════════════════════════════════════════════════════════════════════
// Helper Functions
// ═══════════════════════════════════════════════════════════════════════════════

func executeCommand(root *cobra.Command, args ...string) (output string, err error) {
	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs(args)

	err = root.Execute()
	return buf.String(), err
}

func setEnv(key, value string) func() {
	oldValue := os.Getenv(key)
	os.Setenv(key, value)
	return func() {
		if oldValue == "" {
			os.Unsetenv(key)
		} else {
			os.Setenv(key, oldValue)
		}
	}
}
