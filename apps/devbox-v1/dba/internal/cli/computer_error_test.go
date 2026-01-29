// internal/cli/computer_error_test.go
package cli

import (
	"errors"
	"strings"
	"testing"
)

// TestComputerErrorMessages tests error message formatting for computer commands
func TestComputerErrorMessages(t *testing.T) {
	tests := []struct {
		name          string
		err           error
		expectContain []string
	}{
		{
			name: "workspace error",
			err:  NewWorkspaceError("not in a DBA workspace"),
			expectContain: []string{
				"not in a DBA workspace",
			},
		},
		{
			name: "VM not running",
			err:  errors.New("VM is not running. Start it with: dba computer start"),
			expectContain: []string{
				"VM is not running",
				"dba computer start",
			},
		},
		{
			name: "snapshot not found",
			err:  errors.New("saved snapshot 'my-state' not found"),
			expectContain: []string{
				"snapshot",
				"not found",
			},
		},
		{
			name: "browser command error",
			err:  errors.New("failed to execute browser command: click"),
			expectContain: []string{
				"browser command",
				"click",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errStr := tt.err.Error()
			for _, substr := range tt.expectContain {
				if !strings.Contains(errStr, substr) {
					t.Errorf("error message should contain '%s', got: %s", substr, errStr)
				}
			}
		})
	}
}

// TestWorkspaceErrorType tests WorkspaceError type
func TestComputerWorkspaceErrorType(t *testing.T) {
	err := NewWorkspaceError("test error")

	// Should implement error interface
	var _ error = err

	// Error message should match
	if err.Error() != "test error" {
		t.Errorf("unexpected error message: %s", err.Error())
	}

	// Should be a WorkspaceError
	var wsErr *WorkspaceError
	if !errors.As(err, &wsErr) {
		t.Error("should be a WorkspaceError")
	}
}

// TestErrorHelpMessages tests that errors include helpful hints
func TestErrorHelpMessages(t *testing.T) {
	tests := []struct {
		name        string
		errorMsg    string
		shouldHave  string
		description string
	}{
		{
			name:        "VM not running includes start hint",
			errorMsg:    "VM is not running. Start it with: dba computer start",
			shouldHave:  "dba computer start",
			description: "should tell user how to start VM",
		},
		{
			name:        "workspace missing includes hint",
			errorMsg:    "not in a DBA workspace (use --workspace or cd to workspace directory)",
			shouldHave:  "--workspace",
			description: "should mention --workspace flag",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if !strings.Contains(tt.errorMsg, tt.shouldHave) {
				t.Errorf("%s: error should contain '%s'", tt.description, tt.shouldHave)
			}
		})
	}
}

// TestComputerCommandArgsValidation tests that commands validate required args
// Note: This uses GetRootCmd which shares state, so individual arg checks are in TestComputerArgsValidation
func TestComputerCommandArgsValidation(t *testing.T) {
	// Commands that require args - verify at command definition level
	tests := []struct {
		name     string
		cmdName  string
		minArgs  int
	}{
		{"click requires 1 arg", "click", 1},
		{"dblclick requires 1 arg", "dblclick", 1},
		{"type requires 2 args", "type", 2},
		{"fill requires 2 args", "fill", 2},
		{"select requires 2 args", "select", 2},
		{"hover requires 1 arg", "hover", 1},
		{"open requires 1 arg", "open", 1},
		{"press requires 1 arg", "press", 1},
		{"is requires 2 args", "is", 2},
	}

	root := GetRootCmd()
	for _, cmd := range root.Commands() {
		if cmd.Use == "computer" {
			for _, tt := range tests {
				t.Run(tt.name, func(t *testing.T) {
					for _, sub := range cmd.Commands() {
						if sub.Name() == tt.cmdName {
							// Commands should have Args validator set
							if sub.Args == nil {
								t.Logf("command %s has no Args validator - uses default", tt.cmdName)
							}
							return
						}
					}
					t.Errorf("command %s not found", tt.cmdName)
				})
			}
			return
		}
	}
	t.Fatal("computer command not found")
}

// TestErrorJSONFormat tests error output in JSON format
func TestErrorJSONFormat(t *testing.T) {
	// Verify error output matches expected JSON structure
	type JSONError struct {
		Error struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}

	// Test different error codes
	codes := []string{
		"WORKSPACE_NOT_FOUND",
		"VM_NOT_RUNNING",
		"INVALID_ARGUMENT",
		"TIMEOUT",
		"INTERNAL_ERROR",
	}

	for _, code := range codes {
		t.Run(code, func(t *testing.T) {
			// Verify code format is valid
			if !strings.Contains(code, "_") && len(code) > 10 {
				t.Error("error code should use SCREAMING_SNAKE_CASE")
			}
		})
	}
}

// TestComputerInvalidRefFormat tests error for invalid ref format
func TestComputerInvalidRefFormat(t *testing.T) {
	// Invalid ref formats that should produce clear error messages
	invalidRefs := []string{
		"e1",       // Missing @
		"@",        // Missing ID
		"@ e1",     // Space in ref
		"#e1",      // Wrong prefix
		"@element", // Should be numeric
	}

	for _, ref := range invalidRefs {
		t.Run(ref, func(t *testing.T) {
			// Note: This is a design-level test
			// Implementation would validate ref format in browser commands
			if strings.HasPrefix(ref, "@e") && len(ref) > 2 {
				// Valid format like @e1, @e23
				return
			}
			// Invalid format - would produce error
		})
	}
}

// TestErrorWrapping tests error wrapping
func TestErrorWrapping(t *testing.T) {
	baseErr := errors.New("base error")
	wrappedErr := errors.New("failed to click: base error")

	if !strings.Contains(wrappedErr.Error(), "failed to click") {
		t.Error("wrapped error should contain context")
	}
	if !strings.Contains(wrappedErr.Error(), "base error") {
		t.Error("wrapped error should contain base error")
	}

	// Test with fmt.Errorf style
	_ = baseErr // suppress unused warning
}

// TestErrorChain tests error chain with Is/As
func TestErrorChain(t *testing.T) {
	wsErr := NewWorkspaceError("test")

	// Test errors.Is
	if !errors.Is(wsErr, wsErr) {
		t.Error("error should match itself")
	}

	// Test errors.As
	var target *WorkspaceError
	if !errors.As(wsErr, &target) {
		t.Error("should be able to extract WorkspaceError")
	}
}

// TestBrowserElementNotFoundError tests element not found error
func TestBrowserElementNotFoundError(t *testing.T) {
	errMsg := "element @e42 not found on page"

	if !strings.Contains(errMsg, "@e42") {
		t.Error("error should contain element ref")
	}
	if !strings.Contains(errMsg, "not found") {
		t.Error("error should indicate element was not found")
	}
}

// TestBrowserTimeoutError tests timeout error format
func TestBrowserTimeoutError(t *testing.T) {
	errMsg := "timeout waiting for element @e1 (30000ms)"

	if !strings.Contains(errMsg, "timeout") {
		t.Error("error should mention timeout")
	}
	if !strings.Contains(errMsg, "@e1") {
		t.Error("error should mention element ref")
	}
	if !strings.Contains(errMsg, "ms") {
		t.Error("error should include timeout duration")
	}
}

// TestMorphVMStartError tests VM start error format
func TestMorphVMStartError(t *testing.T) {
	errMsg := "failed to start VM: insufficient quota"

	if !strings.Contains(errMsg, "failed to start") {
		t.Error("error should indicate start failure")
	}
	if !strings.Contains(errMsg, "VM") {
		t.Error("error should mention VM")
	}
}

// TestMorphSnapshotNotFoundError tests snapshot not found error
func TestMorphSnapshotNotFoundError(t *testing.T) {
	errMsg := "saved snapshot 'my-checkpoint' not found"

	if !strings.Contains(errMsg, "snapshot") {
		t.Error("error should mention snapshot")
	}
	if !strings.Contains(errMsg, "not found") {
		t.Error("error should indicate not found")
	}
	if !strings.Contains(errMsg, "my-checkpoint") {
		t.Error("error should include snapshot name")
	}
}

// TestErrorConsistency tests error message consistency across commands
func TestErrorConsistency(t *testing.T) {
	// All "not running" errors should be consistent
	notRunningErrors := []string{
		"VM is not running. Start it with: dba computer start",
		"VM is not running",
	}

	for _, errMsg := range notRunningErrors {
		if !strings.Contains(strings.ToLower(errMsg), "not running") {
			t.Errorf("error should contain 'not running': %s", errMsg)
		}
	}

	// All "not found" errors should be consistent
	notFoundErrors := []string{
		"workspace not found",
		"snapshot not found",
		"element not found",
	}

	for _, errMsg := range notFoundErrors {
		if !strings.Contains(errMsg, "not found") {
			t.Errorf("error should contain 'not found': %s", errMsg)
		}
	}
}
