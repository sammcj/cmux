// internal/cli/output.go
package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"
)

// Exit codes per specification
const (
	ExitCodeSuccess = 0
	ExitCodeError   = 1 // General errors
	ExitCodeUsage   = 2 // Usage errors (invalid command, missing args, etc.)
)

// errorOutputOnce ensures errors are only output once to prevent duplicate messages
var (
	errorOutputOnce sync.Once
	lastOutputError error
)

// UsageError represents a usage/input error (exit code 2)
type UsageError struct {
	Message string
}

func (e *UsageError) Error() string {
	return e.Message
}

// NewUsageError creates a new usage error
func NewUsageError(msg string) error {
	return &UsageError{Message: msg}
}

// OutputResult outputs the result as JSON or formatted text
func OutputResult(data interface{}) error {
	if flagJSON {
		return OutputJSON(data)
	}
	return OutputText(data)
}

// OutputJSON outputs data as JSON
func OutputJSON(data interface{}) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(data)
}

// OutputText outputs data as human-readable text
// Each type should implement a TextOutput() string method
func OutputText(data interface{}) error {
	if t, ok := data.(interface{ TextOutput() string }); ok {
		fmt.Println(t.TextOutput())
		return nil
	}
	// Fallback to JSON
	return OutputJSON(data)
}

// OutputError outputs an error in consistent format
// It guards against duplicate output by checking if this error was already output
func OutputError(err error) {
	if err == nil {
		return
	}

	// Check if this exact error was already output
	if lastOutputError != nil && lastOutputError.Error() == err.Error() {
		return
	}
	lastOutputError = err

	if flagJSON {
		OutputJSON(ErrorResponse{
			Error: ErrorDetail{
				Code:    getErrorCode(err),
				Message: err.Error(),
			},
		})
	} else {
		fmt.Fprintf(os.Stderr, "Error: %s\n", err.Error())
	}
}

// ResetErrorOutput resets the error output guard (for testing)
func ResetErrorOutput() {
	lastOutputError = nil
}

// GetExitCode returns the appropriate exit code for an error
func GetExitCode(err error) int {
	if err == nil {
		return ExitCodeSuccess
	}

	// Check for usage errors
	var usageErr *UsageError
	if errors.As(err, &usageErr) {
		return ExitCodeUsage
	}

	// Check for cobra usage errors (unknown command, missing args, etc.)
	errMsg := err.Error()
	if strings.Contains(errMsg, "unknown command") ||
		strings.Contains(errMsg, "unknown flag") ||
		strings.Contains(errMsg, "unknown shorthand flag") ||
		strings.Contains(errMsg, "requires at least") ||
		strings.Contains(errMsg, "accepts at most") ||
		strings.Contains(errMsg, "accepts ") || // Matches "accepts X arg(s), received Y"
		strings.Contains(errMsg, "invalid argument") {
		return ExitCodeUsage
	}

	return ExitCodeError
}

// ErrorResponse is the standard error format
type ErrorResponse struct {
	Error ErrorDetail `json:"error"`
}

type ErrorDetail struct {
	Code    string                 `json:"code"`
	Message string                 `json:"message"`
	Details map[string]interface{} `json:"details,omitempty"`
}

// Common error codes
const (
	ErrCodeWorkspaceNotFound   = "WORKSPACE_NOT_FOUND"
	ErrCodeInvalidInput        = "INVALID_INPUT"
	ErrCodeUsage               = "USAGE_ERROR"
	ErrCodeTimeout             = "TIMEOUT"
	ErrCodeInternal            = "INTERNAL_ERROR"
	ErrCodeDependencyMissing   = "DEPENDENCY_MISSING"   // Added by Agent #6 for B05
	ErrCodeServiceError        = "SERVICE_ERROR"        // Added by Agent #6 for service-related errors
	ErrCodeProcessComposeError = "PROCESS_COMPOSE_ERROR" // Added by Agent #6
)

// WorkspaceError represents a workspace-related error
type WorkspaceError struct {
	Message string
}

func (e *WorkspaceError) Error() string {
	return e.Message
}

// NewWorkspaceError creates a new workspace error
func NewWorkspaceError(msg string) error {
	return &WorkspaceError{Message: msg}
}

func getErrorCode(err error) string {
	// Check for usage errors
	var usageErr *UsageError
	if errors.As(err, &usageErr) {
		return ErrCodeUsage
	}

	// Check for workspace errors (typed)
	var wsErr *WorkspaceError
	if errors.As(err, &wsErr) {
		return ErrCodeWorkspaceNotFound
	}

	// Check for cobra usage errors (case-insensitive)
	errMsg := strings.ToLower(err.Error())
	if strings.Contains(errMsg, "unknown command") ||
		strings.Contains(errMsg, "unknown flag") ||
		strings.Contains(errMsg, "requires at least") ||
		strings.Contains(errMsg, "accepts at most") {
		return ErrCodeUsage
	}

	// Fixed by Agent #6 (B05): Check for dependency errors BEFORE workspace errors
	// to avoid misclassifying "devbox: command not found" as WORKSPACE_NOT_FOUND
	if strings.Contains(errMsg, "devbox") ||
		strings.Contains(errMsg, "process-compose") ||
		strings.Contains(errMsg, "command not found") ||
		strings.Contains(errMsg, "executable file not found") ||
		strings.Contains(errMsg, "devbox is not installed") {
		return ErrCodeDependencyMissing
	}

	// Check for service-related errors
	if strings.Contains(errMsg, "failed to start services") ||
		strings.Contains(errMsg, "failed to stop services") ||
		strings.Contains(errMsg, "failed to restart services") ||
		strings.Contains(errMsg, "failed to get logs") {
		return ErrCodeServiceError
	}

	// Check for process-compose errors
	if strings.Contains(errMsg, "process-compose") ||
		strings.Contains(errMsg, "process exited") {
		return ErrCodeProcessComposeError
	}

	// Check for known error types (case-insensitive)
	// Match workspace errors by message patterns - but exclude dependency errors
	if (strings.Contains(errMsg, "workspace") && strings.Contains(errMsg, "not found")) ||
		strings.Contains(errMsg, "not in a dba workspace") ||
		(strings.Contains(errMsg, "workspace") && strings.Contains(errMsg, "not") && !strings.Contains(errMsg, "devbox") && !strings.Contains(errMsg, "command")) {
		return ErrCodeWorkspaceNotFound
	}
	if strings.Contains(errMsg, "invalid") {
		return ErrCodeInvalidInput
	}
	if strings.Contains(errMsg, "timeout") || strings.Contains(errMsg, "timed out") {
		return ErrCodeTimeout
	}

	return ErrCodeInternal
}
