// internal/cli/context_test.go
package cli

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/dba-cli/dba/internal/config"
	"github.com/dba-cli/dba/internal/workspace"
)

func TestCLIContextCreation(t *testing.T) {
	// Setup: ensure config is loaded
	var err error
	cfg, err = config.Load()
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	// Reset flags to defaults
	flagTimeout = "5m"
	flagWorkspace = ""
	flagJSON = false
	flagVerbose = false

	ctx, err := NewCLIContext()
	if err != nil {
		t.Fatalf("NewCLIContext() error = %v", err)
	}
	defer ctx.Cancel()

	// Verify context is created
	if ctx.Context == nil {
		t.Error("Context should not be nil")
	}
	if ctx.Cancel == nil {
		t.Error("Cancel should not be nil")
	}
	if ctx.Config == nil {
		t.Error("Config should not be nil")
	}
	// Workspace may be nil if not in a workspace directory, that's OK
}

func TestCLIContextTimeoutParsing(t *testing.T) {
	// Setup
	cfg, _ = config.Load()

	tests := []struct {
		name           string
		timeout        string
		expectDefault  bool
	}{
		{"valid duration 30s", "30s", false},
		{"valid duration 1m", "1m", false},
		{"valid duration 2h", "2h", false},
		{"invalid duration", "invalid", true},
		{"empty string", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			flagTimeout = tt.timeout
			flagWorkspace = ""

			ctx, err := NewCLIContext()
			if err != nil {
				t.Fatalf("NewCLIContext() error = %v", err)
			}
			defer ctx.Cancel()

			// Context should be created regardless of timeout parsing
			if ctx.Context == nil {
				t.Error("Context should be created even with invalid timeout")
			}
		})
	}
}

func TestCLIContextWithWorkspaceFlag(t *testing.T) {
	// Setup
	cfg, _ = config.Load()
	flagTimeout = "5m"

	// Create a temporary workspace
	tmpDir := t.TempDir()
	wsDir := filepath.Join(tmpDir, "test-workspace")
	dbaDir := filepath.Join(wsDir, ".dba")
	if err := os.MkdirAll(dbaDir, 0755); err != nil {
		t.Fatalf("failed to create .dba dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dbaDir, "id"), []byte("ws_test123"), 0644); err != nil {
		t.Fatalf("failed to write id file: %v", err)
	}

	// Test with workspace flag
	flagWorkspace = wsDir

	ctx, err := NewCLIContext()
	if err != nil {
		t.Fatalf("NewCLIContext() error = %v", err)
	}
	defer ctx.Cancel()

	if ctx.Workspace == nil {
		t.Error("Workspace should be resolved when flag is set")
	}
	if ctx.Workspace != nil && ctx.Workspace.ID != "ws_test123" {
		t.Errorf("Workspace ID = %s, want ws_test123", ctx.Workspace.ID)
	}
}

func TestCLIContextWithInvalidWorkspaceFlag(t *testing.T) {
	// Setup
	cfg, _ = config.Load()
	flagTimeout = "5m"
	flagWorkspace = "/nonexistent/workspace/path"

	ctx, err := NewCLIContext()
	if err == nil {
		ctx.Cancel()
		t.Error("NewCLIContext() should return error for nonexistent workspace")
	}
}

func TestRequireWorkspace(t *testing.T) {
	t.Run("nil workspace", func(t *testing.T) {
		ctx := &CLIContext{Workspace: nil}
		err := ctx.RequireWorkspace()
		if err == nil {
			t.Error("RequireWorkspace() should return error for nil workspace")
		}
	})

	t.Run("non-nil workspace", func(t *testing.T) {
		ctx := &CLIContext{
			Workspace: &workspace.Workspace{ID: "ws_test123"},
		}
		err := ctx.RequireWorkspace()
		if err != nil {
			t.Errorf("RequireWorkspace() error = %v, wantErr false", err)
		}
	})
}

func TestRequireWorkspaceErrorMessage(t *testing.T) {
	ctx := &CLIContext{Workspace: nil}
	err := ctx.RequireWorkspace()
	if err == nil {
		t.Fatal("expected error for nil workspace")
	}

	expectedMsg := "not in a DBA workspace (use --workspace or cd to workspace directory)"
	if err.Error() != expectedMsg {
		t.Errorf("error message = %q, want %q", err.Error(), expectedMsg)
	}
}

func TestGetConfig(t *testing.T) {
	// Set up config
	testCfg := &config.Config{
		Home: "/test/home",
	}
	cfg = testCfg

	result := GetConfig()
	if result != testCfg {
		t.Error("GetConfig() should return the global config")
	}
}

func TestIsJSONOutput(t *testing.T) {
	tests := []struct {
		name     string
		flagVal  bool
		expected bool
	}{
		{"flag true", true, true},
		{"flag false", false, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			flagJSON = tt.flagVal
			if got := IsJSONOutput(); got != tt.expected {
				t.Errorf("IsJSONOutput() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestIsVerbose(t *testing.T) {
	tests := []struct {
		name     string
		flagVal  bool
		expected bool
	}{
		{"flag true", true, true},
		{"flag false", false, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			flagVerbose = tt.flagVal
			if got := IsVerbose(); got != tt.expected {
				t.Errorf("IsVerbose() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestGetWorkspaceFlag(t *testing.T) {
	tests := []struct {
		name     string
		flagVal  string
		expected string
	}{
		{"empty", "", ""},
		{"workspace id", "ws_test123", "ws_test123"},
		{"path", "/path/to/workspace", "/path/to/workspace"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			flagWorkspace = tt.flagVal
			if got := GetWorkspaceFlag(); got != tt.expected {
				t.Errorf("GetWorkspaceFlag() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestCLIContextCancellation(t *testing.T) {
	cfg, _ = config.Load()
	flagTimeout = "1s"
	flagWorkspace = ""

	ctx, err := NewCLIContext()
	if err != nil {
		t.Fatalf("NewCLIContext() error = %v", err)
	}

	// Verify context is not cancelled initially
	select {
	case <-ctx.Context.Done():
		t.Error("Context should not be done immediately")
	default:
		// OK
	}

	// Cancel the context
	ctx.Cancel()

	// Verify context is now cancelled
	select {
	case <-ctx.Context.Done():
		// OK - context is cancelled
	case <-time.After(100 * time.Millisecond):
		t.Error("Context should be done after Cancel()")
	}
}

func TestCLIContextTimeout(t *testing.T) {
	cfg, _ = config.Load()
	flagTimeout = "100ms"
	flagWorkspace = ""

	ctx, err := NewCLIContext()
	if err != nil {
		t.Fatalf("NewCLIContext() error = %v", err)
	}
	defer ctx.Cancel()

	// Wait for timeout
	select {
	case <-ctx.Context.Done():
		// OK - context timed out
	case <-time.After(500 * time.Millisecond):
		t.Error("Context should timeout after 100ms")
	}
}

func TestRequireWorkspaceErrorType(t *testing.T) {
	ctx := &CLIContext{Workspace: nil}
	err := ctx.RequireWorkspace()
	if err == nil {
		t.Fatal("expected error for nil workspace")
	}

	// The error should be a WorkspaceError
	var wsErr *WorkspaceError
	if !errors.As(err, &wsErr) {
		t.Error("RequireWorkspace should return a WorkspaceError type")
	}
}

func TestRequireWorkspaceErrorCode(t *testing.T) {
	ctx := &CLIContext{Workspace: nil}
	err := ctx.RequireWorkspace()
	if err == nil {
		t.Fatal("expected error for nil workspace")
	}

	// The error code should be WORKSPACE_NOT_FOUND
	code := getErrorCode(err)
	if code != ErrCodeWorkspaceNotFound {
		t.Errorf("error code = %s, want %s", code, ErrCodeWorkspaceNotFound)
	}
}

func TestWorkspaceErrorType(t *testing.T) {
	// Test creating a WorkspaceError
	err := NewWorkspaceError("test workspace error")
	if err == nil {
		t.Fatal("NewWorkspaceError should not return nil")
	}

	if err.Error() != "test workspace error" {
		t.Errorf("Error() = %q, want %q", err.Error(), "test workspace error")
	}

	// Test that it's properly detected
	var wsErr *WorkspaceError
	if !errors.As(err, &wsErr) {
		t.Error("NewWorkspaceError should return a WorkspaceError type")
	}
}

func TestWorkspaceErrorCodeDetection(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		wantCode string
	}{
		{
			name:     "WorkspaceError type",
			err:      NewWorkspaceError("test error"),
			wantCode: ErrCodeWorkspaceNotFound,
		},
		{
			name:     "not in a DBA workspace message",
			err:      errors.New("not in a DBA workspace"),
			wantCode: ErrCodeWorkspaceNotFound,
		},
		{
			name:     "workspace not found message",
			err:      errors.New("workspace ws_123 not found"),
			wantCode: ErrCodeWorkspaceNotFound,
		},
		{
			name:     "generic error",
			err:      errors.New("something went wrong"),
			wantCode: ErrCodeInternal,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			code := getErrorCode(tt.err)
			if code != tt.wantCode {
				t.Errorf("getErrorCode() = %s, want %s", code, tt.wantCode)
			}
		})
	}
}

// TestConcurrentContextCreation tests creating multiple contexts concurrently
func TestConcurrentContextCreation(t *testing.T) {
	cfg, _ = config.Load()
	flagTimeout = "5m"
	flagWorkspace = ""

	const numGoroutines = 10
	errors := make(chan error, numGoroutines)
	contexts := make(chan *CLIContext, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func() {
			ctx, err := NewCLIContext()
			if err != nil {
				errors <- err
				return
			}
			contexts <- ctx
		}()
	}

	// Collect results
	var createdContexts []*CLIContext
	for i := 0; i < numGoroutines; i++ {
		select {
		case err := <-errors:
			t.Errorf("concurrent context creation error: %v", err)
		case ctx := <-contexts:
			createdContexts = append(createdContexts, ctx)
		case <-time.After(5 * time.Second):
			t.Error("timeout waiting for context creation")
		}
	}

	// Clean up
	for _, ctx := range createdContexts {
		ctx.Cancel()
	}

	if len(createdContexts) != numGoroutines {
		t.Errorf("expected %d contexts, got %d", numGoroutines, len(createdContexts))
	}
}

// TestCLIContextWithVeryShortTimeout tests context with very short timeout
func TestCLIContextWithVeryShortTimeout(t *testing.T) {
	cfg, _ = config.Load()
	flagTimeout = "1ms"
	flagWorkspace = ""

	ctx, err := NewCLIContext()
	if err != nil {
		t.Fatalf("NewCLIContext() error = %v", err)
	}
	defer ctx.Cancel()

	// Context should timeout almost immediately
	select {
	case <-ctx.Context.Done():
		// OK - context timed out as expected
	case <-time.After(100 * time.Millisecond):
		t.Error("Context should have timed out by now")
	}
}

// TestCLIContextWithLongTimeout tests context with long timeout value
func TestCLIContextWithLongTimeout(t *testing.T) {
	cfg, _ = config.Load()
	flagTimeout = "24h"
	flagWorkspace = ""

	ctx, err := NewCLIContext()
	if err != nil {
		t.Fatalf("NewCLIContext() error = %v", err)
	}
	defer ctx.Cancel()

	// Context should not be done immediately
	select {
	case <-ctx.Context.Done():
		t.Error("Context should not be done with 24h timeout")
	case <-time.After(10 * time.Millisecond):
		// OK - context is still active
	}
}

// TestCLIContextWithZeroTimeout tests handling of zero timeout
func TestCLIContextWithZeroTimeout(t *testing.T) {
	cfg, _ = config.Load()
	flagTimeout = "0s"
	flagWorkspace = ""

	ctx, err := NewCLIContext()
	if err != nil {
		t.Fatalf("NewCLIContext() error = %v", err)
	}
	defer ctx.Cancel()

	// Zero timeout: context should expire very quickly
	// Note: context.WithTimeout with 0 duration creates a context that
	// expires immediately but may take a moment to propagate
	select {
	case <-ctx.Context.Done():
		// OK - context done as expected
	case <-time.After(500 * time.Millisecond):
		t.Error("Context with 0 timeout should be done very quickly")
	}
}

// TestCLIContextWithNegativeTimeout tests handling of negative timeout
func TestCLIContextWithNegativeTimeout(t *testing.T) {
	cfg, _ = config.Load()
	flagTimeout = "-5m"
	flagWorkspace = ""

	ctx, err := NewCLIContext()
	if err != nil {
		t.Fatalf("NewCLIContext() error = %v", err)
	}
	defer ctx.Cancel()

	// Negative timeout should be treated as invalid and use default
	select {
	case <-ctx.Context.Done():
		t.Error("Context should not be immediately done with negative timeout (should use default)")
	case <-time.After(10 * time.Millisecond):
		// OK - context is active with default timeout
	}
}

// TestFlagAccessors tests all flag accessor functions
func TestFlagAccessors(t *testing.T) {
	// Save original values
	origJSON := flagJSON
	origVerbose := flagVerbose
	origWorkspace := flagWorkspace
	defer func() {
		flagJSON = origJSON
		flagVerbose = origVerbose
		flagWorkspace = origWorkspace
	}()

	// Test IsJSONOutput
	flagJSON = true
	if !IsJSONOutput() {
		t.Error("IsJSONOutput() should return true when flagJSON is true")
	}
	flagJSON = false
	if IsJSONOutput() {
		t.Error("IsJSONOutput() should return false when flagJSON is false")
	}

	// Test IsVerbose
	flagVerbose = true
	if !IsVerbose() {
		t.Error("IsVerbose() should return true when flagVerbose is true")
	}
	flagVerbose = false
	if IsVerbose() {
		t.Error("IsVerbose() should return false when flagVerbose is false")
	}

	// Test GetWorkspaceFlag
	flagWorkspace = "test-workspace"
	if GetWorkspaceFlag() != "test-workspace" {
		t.Error("GetWorkspaceFlag() should return the flag value")
	}
	flagWorkspace = ""
	if GetWorkspaceFlag() != "" {
		t.Error("GetWorkspaceFlag() should return empty string when not set")
	}
}

// TestCLIContextConfigNotNil tests that config is never nil
func TestCLIContextConfigNotNil(t *testing.T) {
	// Load config first
	var err error
	cfg, err = config.Load()
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	flagTimeout = "5m"
	flagWorkspace = ""

	ctx, err := NewCLIContext()
	if err != nil {
		t.Fatalf("NewCLIContext() error = %v", err)
	}
	defer ctx.Cancel()

	if ctx.Config == nil {
		t.Error("CLIContext.Config should never be nil")
	}
}

// TestCLIContextCancelMultipleTimes tests that Cancel can be called multiple times safely
func TestCLIContextCancelMultipleTimes(t *testing.T) {
	cfg, _ = config.Load()
	flagTimeout = "5m"
	flagWorkspace = ""

	ctx, err := NewCLIContext()
	if err != nil {
		t.Fatalf("NewCLIContext() error = %v", err)
	}

	// Call Cancel multiple times - should not panic
	ctx.Cancel()
	ctx.Cancel()
	ctx.Cancel()

	// Context should be done
	select {
	case <-ctx.Context.Done():
		// OK
	default:
		t.Error("Context should be done after Cancel")
	}
}

// TestRequireWorkspaceWithValidWorkspace tests RequireWorkspace with various valid workspaces
func TestRequireWorkspaceWithValidWorkspace(t *testing.T) {
	tests := []struct {
		name      string
		workspace *workspace.Workspace
	}{
		{
			name:      "basic workspace",
			workspace: &workspace.Workspace{ID: "ws_basic"},
		},
		{
			name: "full workspace",
			workspace: &workspace.Workspace{
				ID:       "ws_full",
				Name:     "full-workspace",
				Path:     "/tmp/full-workspace",
				Template: "node",
				Status:   "active",
				BasePort: 10000,
			},
		},
		{
			name: "workspace with ports",
			workspace: &workspace.Workspace{
				ID:       "ws_ports",
				Ports:    map[string]int{"PORT": 3000, "API_PORT": 3001},
				Packages: []string{"nodejs@18"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := &CLIContext{Workspace: tt.workspace}
			err := ctx.RequireWorkspace()
			if err != nil {
				t.Errorf("RequireWorkspace() should not error for valid workspace, got: %v", err)
			}
		})
	}
}

// TestCLIContextWithWorkspaceByID tests creating context with workspace ID format
func TestCLIContextWithWorkspaceByID(t *testing.T) {
	cfg, _ = config.Load()
	flagTimeout = "5m"

	// Test with non-existent workspace ID
	flagWorkspace = "ws_nonexistent_12345"

	ctx, err := NewCLIContext()
	if err == nil {
		ctx.Cancel()
		t.Error("NewCLIContext() should return error for non-existent workspace ID")
	}

	// Error should mention workspace
	if err != nil && !errors.As(err, new(*WorkspaceError)) {
		// Check if error message contains workspace-related text
		errMsg := err.Error()
		if !containsAny(errMsg, []string{"not found", "workspace"}) {
			t.Errorf("error should be workspace-related, got: %v", err)
		}
	}
}

// Helper function to check if string contains any of the substrings
func containsAny(s string, substrs []string) bool {
	for _, substr := range substrs {
		if containsIgnoreCase(s, substr) {
			return true
		}
	}
	return false
}

// Helper function for case-insensitive contains
func containsIgnoreCase(s, substr string) bool {
	return len(s) >= len(substr) &&
		(s == substr ||
			len(s) > 0 && len(substr) > 0 &&
				(s[0] == substr[0] || s[0]^32 == substr[0]) &&
				containsIgnoreCase(s[1:], substr[1:]) ||
			len(s) > 0 && containsIgnoreCase(s[1:], substr))
}

// TestTimeoutDurationParsing tests various timeout duration formats
func TestTimeoutDurationParsing(t *testing.T) {
	cfg, _ = config.Load()
	flagWorkspace = ""

	tests := []struct {
		name    string
		timeout string
		valid   bool
	}{
		{"seconds", "30s", true},
		{"minutes", "5m", true},
		{"hours", "2h", true},
		{"milliseconds", "500ms", true},
		{"microseconds", "1000us", true},
		{"nanoseconds", "1000000ns", true},
		{"combined", "1h30m", true},
		{"invalid format", "abc", false},
		{"missing unit", "100", false},
		{"empty", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			flagTimeout = tt.timeout
			ctx, err := NewCLIContext()
			if err != nil {
				t.Fatalf("NewCLIContext() error = %v", err)
			}
			defer ctx.Cancel()

			// Context should be created regardless of timeout validity
			if ctx.Context == nil {
				t.Error("Context should be created")
			}
		})
	}
}
