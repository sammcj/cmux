// internal/cli/cli_test.go
package cli

import (
	"fmt"
	"testing"

	"github.com/spf13/cobra"
)

func TestSetVersionInfo(t *testing.T) {
	SetVersionInfo("1.0.0", "abc123", "2024-01-01T00:00:00Z")

	if version != "1.0.0" {
		t.Errorf("expected version 1.0.0, got %s", version)
	}
	if commit != "abc123" {
		t.Errorf("expected commit abc123, got %s", commit)
	}
	if buildTime != "2024-01-01T00:00:00Z" {
		t.Errorf("expected buildTime 2024-01-01T00:00:00Z, got %s", buildTime)
	}
}

func TestVersionInfoTextOutput(t *testing.T) {
	info := VersionInfo{
		Version:   "1.0.0",
		Commit:    "abc123",
		BuildTime: "2024-01-01T00:00:00Z",
		GoVersion: "go1.22",
		OS:        "darwin",
		Arch:      "amd64",
	}

	output := info.TextOutput()
	if output == "" {
		t.Error("TextOutput should not be empty")
	}

	// Check that key info is present
	if len(output) < 20 {
		t.Errorf("TextOutput seems too short: %s", output)
	}
}

func TestErrorCodes(t *testing.T) {
	// Verify error codes are defined
	if ErrCodeWorkspaceNotFound != "WORKSPACE_NOT_FOUND" {
		t.Errorf("unexpected ErrCodeWorkspaceNotFound: %s", ErrCodeWorkspaceNotFound)
	}
	if ErrCodeInvalidInput != "INVALID_INPUT" {
		t.Errorf("unexpected ErrCodeInvalidInput: %s", ErrCodeInvalidInput)
	}
	if ErrCodeTimeout != "TIMEOUT" {
		t.Errorf("unexpected ErrCodeTimeout: %s", ErrCodeTimeout)
	}
	if ErrCodeInternal != "INTERNAL_ERROR" {
		t.Errorf("unexpected ErrCodeInternal: %s", ErrCodeInternal)
	}
}

func TestGetRootCmd(t *testing.T) {
	cmd := GetRootCmd()
	if cmd == nil {
		t.Error("GetRootCmd returned nil")
	}
	if cmd.Use != "dba" {
		t.Errorf("expected command name 'dba', got '%s'", cmd.Use)
	}
}

func TestGlobalFlags(t *testing.T) {
	cmd := GetRootCmd()

	// Check --json flag
	jsonFlag := cmd.PersistentFlags().Lookup("json")
	if jsonFlag == nil {
		t.Error("--json flag not found")
	}

	// Check --workspace flag
	wsFlag := cmd.PersistentFlags().Lookup("workspace")
	if wsFlag == nil {
		t.Error("--workspace flag not found")
	}
	if wsFlag.Shorthand != "w" {
		t.Errorf("expected -w shorthand for workspace, got %s", wsFlag.Shorthand)
	}

	// Check --verbose flag
	verboseFlag := cmd.PersistentFlags().Lookup("verbose")
	if verboseFlag == nil {
		t.Error("--verbose flag not found")
	}
	if verboseFlag.Shorthand != "v" {
		t.Errorf("expected -v shorthand for verbose, got %s", verboseFlag.Shorthand)
	}

	// Check --timeout flag
	timeoutFlag := cmd.PersistentFlags().Lookup("timeout")
	if timeoutFlag == nil {
		t.Error("--timeout flag not found")
	}
	if timeoutFlag.DefValue != "5m" {
		t.Errorf("expected default timeout of 5m, got %s", timeoutFlag.DefValue)
	}
}

func TestVersionCommand(t *testing.T) {
	// Find version subcommand
	cmd := GetRootCmd()
	var versionCmdFound *cobra.Command
	for _, c := range cmd.Commands() {
		if c.Name() == "version" {
			versionCmdFound = c
			break
		}
	}

	if versionCmdFound == nil {
		t.Error("version command not found")
		return
	}

	if versionCmdFound.Short != "Print version information" {
		t.Errorf("unexpected short description: %s", versionCmdFound.Short)
	}

	// Check --check flag exists
	checkFlag := versionCmdFound.Flags().Lookup("check")
	if checkFlag == nil {
		t.Error("--check flag not found on version command")
	}
}

func TestExitCodes(t *testing.T) {
	// Test exit code constants
	if ExitCodeSuccess != 0 {
		t.Errorf("expected ExitCodeSuccess = 0, got %d", ExitCodeSuccess)
	}
	if ExitCodeError != 1 {
		t.Errorf("expected ExitCodeError = 1, got %d", ExitCodeError)
	}
	if ExitCodeUsage != 2 {
		t.Errorf("expected ExitCodeUsage = 2, got %d", ExitCodeUsage)
	}
}

func TestGetExitCode(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected int
	}{
		{"nil error", nil, ExitCodeSuccess},
		{"usage error", NewUsageError("test"), ExitCodeUsage},
		{"unknown command", fmt.Errorf("unknown command \"foo\" for \"dba\""), ExitCodeUsage},
		{"unknown flag", fmt.Errorf("unknown flag: --foo"), ExitCodeUsage},
		{"general error", fmt.Errorf("something went wrong"), ExitCodeError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetExitCode(tt.err)
			if got != tt.expected {
				t.Errorf("GetExitCode(%v) = %d, want %d", tt.err, got, tt.expected)
			}
		})
	}
}

func TestUsageError(t *testing.T) {
	err := NewUsageError("invalid argument")
	if err.Error() != "invalid argument" {
		t.Errorf("expected 'invalid argument', got '%s'", err.Error())
	}

	// Verify it returns exit code 2
	if GetExitCode(err) != ExitCodeUsage {
		t.Errorf("expected ExitCodeUsage for UsageError")
	}
}

func TestVersionCheckResult(t *testing.T) {
	// Test with update available
	result := VersionCheckResult{
		CurrentVersion: "1.0.0",
		LatestVersion:  "1.1.0",
		UpdateAvail:    true,
		DownloadURL:    "https://example.com/download",
	}
	output := result.TextOutput()
	if output == "" {
		t.Error("TextOutput should not be empty")
	}

	// Test with no update
	result = VersionCheckResult{
		CurrentVersion: "1.0.0",
		UpdateAvail:    false,
	}
	output = result.TextOutput()
	if output == "" {
		t.Error("TextOutput should not be empty for no update case")
	}

	// Test with error
	result = VersionCheckResult{
		CurrentVersion: "1.0.0",
		Error:          "network error",
	}
	output = result.TextOutput()
	if output == "" {
		t.Error("TextOutput should not be empty for error case")
	}
}

func TestGetVersion(t *testing.T) {
	SetVersionInfo("2.0.0", "def456", "2025-01-01T00:00:00Z")
	if GetVersion() != "2.0.0" {
		t.Errorf("expected GetVersion() = '2.0.0', got '%s'", GetVersion())
	}
}
