// internal/cli/output_test.go
package cli

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"
)

// Test helper to capture stdout
func captureStdout(f func()) string {
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Read in a goroutine to prevent blocking when output is large
	outC := make(chan string)
	go func() {
		var buf bytes.Buffer
		io.Copy(&buf, r)
		outC <- buf.String()
	}()

	f()

	w.Close()
	os.Stdout = old

	return <-outC
}

// Test helper to capture stderr
func captureStderr(f func()) string {
	old := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	// Read in a goroutine to prevent blocking when output is large
	outC := make(chan string)
	go func() {
		var buf bytes.Buffer
		io.Copy(&buf, r)
		outC <- buf.String()
	}()

	f()

	w.Close()
	os.Stderr = old

	return <-outC
}

func TestOutputJSON(t *testing.T) {
	tests := []struct {
		name     string
		input    interface{}
		wantErr  bool
		validate func(string) error
	}{
		{
			name:  "simple struct",
			input: struct{ Name string }{"test"},
			validate: func(s string) error {
				if !strings.Contains(s, `"Name"`) {
					return fmt.Errorf("expected Name field in output")
				}
				return nil
			},
		},
		{
			name:  "map",
			input: map[string]int{"a": 1, "b": 2},
			validate: func(s string) error {
				var m map[string]int
				if err := json.Unmarshal([]byte(s), &m); err != nil {
					return err
				}
				if m["a"] != 1 || m["b"] != 2 {
					return fmt.Errorf("unexpected values in map")
				}
				return nil
			},
		},
		{
			name:  "slice",
			input: []string{"a", "b", "c"},
			validate: func(s string) error {
				var arr []string
				if err := json.Unmarshal([]byte(s), &arr); err != nil {
					return err
				}
				if len(arr) != 3 {
					return fmt.Errorf("expected 3 elements, got %d", len(arr))
				}
				return nil
			},
		},
		{
			name:  "nested struct",
			input: struct{ Inner struct{ Value int } }{struct{ Value int }{42}},
			validate: func(s string) error {
				if !strings.Contains(s, "42") {
					return fmt.Errorf("expected value 42 in output")
				}
				return nil
			},
		},
		{
			name:  "nil value",
			input: nil,
			validate: func(s string) error {
				if strings.TrimSpace(s) != "null" {
					return fmt.Errorf("expected null, got %s", s)
				}
				return nil
			},
		},
		{
			name:  "empty struct",
			input: struct{}{},
			validate: func(s string) error {
				if strings.TrimSpace(s) != "{}" {
					return fmt.Errorf("expected {}, got %s", s)
				}
				return nil
			},
		},
		{
			name:  "unicode characters",
			input: map[string]string{"greeting": "你好世界"},
			validate: func(s string) error {
				if !strings.Contains(s, "你好世界") {
					return fmt.Errorf("expected unicode characters in output")
				}
				return nil
			},
		},
		{
			name:  "special characters",
			input: map[string]string{"text": "line1\nline2\ttab"},
			validate: func(s string) error {
				// JSON should escape newlines and tabs
				if !strings.Contains(s, `\n`) && !strings.Contains(s, `\t`) {
					return fmt.Errorf("expected escaped special characters")
				}
				return nil
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := captureStdout(func() {
				err := OutputJSON(tt.input)
				if (err != nil) != tt.wantErr {
					t.Errorf("OutputJSON() error = %v, wantErr %v", err, tt.wantErr)
				}
			})

			if tt.validate != nil {
				if err := tt.validate(output); err != nil {
					t.Errorf("validation failed: %v, output: %s", err, output)
				}
			}
		})
	}
}

func TestOutputText(t *testing.T) {
	// Test with TextOutput interface
	type testOutput struct {
		msg string
	}
	testOutput1 := testOutput{msg: "test message"}

	// Create a type that implements TextOutput
	type textOutputter struct {
		text string
	}
	// Note: TextOutput interface requires TextOutput() string method

	// Test fallback to JSON when TextOutput is not implemented
	output := captureStdout(func() {
		OutputText(map[string]string{"key": "value"})
	})
	if !strings.Contains(output, "key") {
		t.Errorf("expected JSON fallback for non-TextOutput type")
	}

	// Test with nil
	output = captureStdout(func() {
		OutputText(nil)
	})
	if strings.TrimSpace(output) != "null" {
		t.Errorf("expected null for nil input, got: %s", output)
	}

	// Verify testOutput doesn't implement TextOutput (for coverage)
	_ = testOutput1
}

func TestOutputResult(t *testing.T) {
	// Save and restore flagJSON
	oldFlagJSON := flagJSON
	defer func() { flagJSON = oldFlagJSON }()

	// Test JSON mode
	flagJSON = true
	output := captureStdout(func() {
		OutputResult(map[string]string{"mode": "json"})
	})
	if !strings.Contains(output, `"mode"`) {
		t.Errorf("expected JSON output when flagJSON is true")
	}

	// Test text mode (will fallback to JSON for map)
	flagJSON = false
	output = captureStdout(func() {
		OutputResult(map[string]string{"mode": "text"})
	})
	// Should fallback to JSON since map doesn't implement TextOutput
	if !strings.Contains(output, `"mode"`) {
		t.Errorf("expected JSON fallback for text mode")
	}
}

func TestOutputError(t *testing.T) {
	oldFlagJSON := flagJSON
	defer func() { flagJSON = oldFlagJSON }()

	tests := []struct {
		name      string
		err       error
		jsonMode  bool
		checkFunc func(string) error
	}{
		{
			name:     "simple error - text mode",
			err:      errors.New("something went wrong"),
			jsonMode: false,
			checkFunc: func(s string) error {
				if !strings.Contains(s, "Error: something went wrong") {
					return fmt.Errorf("expected error message, got: %s", s)
				}
				return nil
			},
		},
		{
			name:     "simple error - JSON mode",
			err:      errors.New("something went wrong"),
			jsonMode: true,
			checkFunc: func(s string) error {
				var resp ErrorResponse
				if err := json.Unmarshal([]byte(s), &resp); err != nil {
					return fmt.Errorf("failed to parse JSON: %v", err)
				}
				if resp.Error.Message != "something went wrong" {
					return fmt.Errorf("unexpected message: %s", resp.Error.Message)
				}
				return nil
			},
		},
		{
			name:     "usage error - JSON mode",
			err:      NewUsageError("invalid argument"),
			jsonMode: true,
			checkFunc: func(s string) error {
				var resp ErrorResponse
				if err := json.Unmarshal([]byte(s), &resp); err != nil {
					return fmt.Errorf("failed to parse JSON: %v", err)
				}
				if resp.Error.Code != ErrCodeUsage {
					return fmt.Errorf("expected USAGE_ERROR code, got: %s", resp.Error.Code)
				}
				return nil
			},
		},
		{
			name:     "not found error - JSON mode",
			err:      errors.New("workspace not found"),
			jsonMode: true,
			checkFunc: func(s string) error {
				var resp ErrorResponse
				if err := json.Unmarshal([]byte(s), &resp); err != nil {
					return fmt.Errorf("failed to parse JSON: %v", err)
				}
				if resp.Error.Code != ErrCodeWorkspaceNotFound {
					return fmt.Errorf("expected WORKSPACE_NOT_FOUND code, got: %s", resp.Error.Code)
				}
				return nil
			},
		},
		{
			name:     "timeout error - JSON mode",
			err:      errors.New("operation timed out"),
			jsonMode: true,
			checkFunc: func(s string) error {
				var resp ErrorResponse
				if err := json.Unmarshal([]byte(s), &resp); err != nil {
					return fmt.Errorf("failed to parse JSON: %v", err)
				}
				if resp.Error.Code != ErrCodeTimeout {
					return fmt.Errorf("expected TIMEOUT code, got: %s", resp.Error.Code)
				}
				return nil
			},
		},
		{
			name:     "invalid error - JSON mode",
			err:      errors.New("invalid configuration"),
			jsonMode: true,
			checkFunc: func(s string) error {
				var resp ErrorResponse
				if err := json.Unmarshal([]byte(s), &resp); err != nil {
					return fmt.Errorf("failed to parse JSON: %v", err)
				}
				if resp.Error.Code != ErrCodeInvalidInput {
					return fmt.Errorf("expected INVALID_INPUT code, got: %s", resp.Error.Code)
				}
				return nil
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset error output guard to allow testing the same error message
			ResetErrorOutput()
			flagJSON = tt.jsonMode

			var output string
			if tt.jsonMode {
				output = captureStdout(func() {
					OutputError(tt.err)
				})
			} else {
				output = captureStderr(func() {
					OutputError(tt.err)
				})
			}

			if err := tt.checkFunc(output); err != nil {
				t.Errorf("check failed: %v", err)
			}
		})
	}
}

func TestGetExitCodeComprehensive(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected int
	}{
		{"nil error", nil, ExitCodeSuccess},
		{"usage error type", NewUsageError("bad input"), ExitCodeUsage},
		{"unknown command", errors.New(`unknown command "foo" for "dba"`), ExitCodeUsage},
		{"unknown flag", errors.New("unknown flag: --bar"), ExitCodeUsage},
		{"unknown shorthand", errors.New("unknown shorthand flag: 'x'"), ExitCodeUsage},
		{"requires at least", errors.New("requires at least 1 arg(s)"), ExitCodeUsage},
		{"accepts at most", errors.New("accepts at most 2 arg(s)"), ExitCodeUsage},
		{"invalid argument", errors.New("invalid argument \"foo\" for \"--bar\""), ExitCodeUsage},
		{"generic error", errors.New("something failed"), ExitCodeError},
		{"wrapped error", fmt.Errorf("wrap: %w", errors.New("inner")), ExitCodeError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetExitCode(tt.err)
			if got != tt.expected {
				t.Errorf("GetExitCode() = %d, want %d", got, tt.expected)
			}
		})
	}
}

func TestGetErrorCodeComprehensive(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected string
	}{
		{"usage error", NewUsageError("test"), ErrCodeUsage},
		{"unknown command", errors.New("unknown command"), ErrCodeUsage},
		{"not found", errors.New("workspace not found"), ErrCodeWorkspaceNotFound},
		{"invalid input", errors.New("invalid value"), ErrCodeInvalidInput},
		{"timeout", errors.New("request timeout"), ErrCodeTimeout},
		{"timed out", errors.New("operation timed out"), ErrCodeTimeout},
		{"generic", errors.New("some error"), ErrCodeInternal},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getErrorCode(tt.err)
			if got != tt.expected {
				t.Errorf("getErrorCode() = %s, want %s", got, tt.expected)
			}
		})
	}
}

func TestErrorResponseSerialization(t *testing.T) {
	resp := ErrorResponse{
		Error: ErrorDetail{
			Code:    "TEST_ERROR",
			Message: "Test error message",
			Details: map[string]interface{}{
				"field": "value",
				"count": 42,
			},
		},
	}

	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var parsed ErrorResponse
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if parsed.Error.Code != "TEST_ERROR" {
		t.Errorf("unexpected code: %s", parsed.Error.Code)
	}
	if parsed.Error.Message != "Test error message" {
		t.Errorf("unexpected message: %s", parsed.Error.Message)
	}
	if parsed.Error.Details["field"] != "value" {
		t.Errorf("unexpected details field")
	}
}

func TestErrorResponseOmitEmptyDetails(t *testing.T) {
	resp := ErrorResponse{
		Error: ErrorDetail{
			Code:    "TEST",
			Message: "test",
			// Details is nil
		},
	}

	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	// Details should be omitted
	if strings.Contains(string(data), "details") {
		t.Errorf("expected details to be omitted, got: %s", string(data))
	}
}

func TestUsageErrorImplementsError(t *testing.T) {
	var err error = NewUsageError("test message")

	// Should implement error interface
	if err.Error() != "test message" {
		t.Errorf("Error() = %s, want 'test message'", err.Error())
	}

	// Should be detectable as UsageError
	var usageErr *UsageError
	if !errors.As(err, &usageErr) {
		t.Error("expected errors.As to match UsageError")
	}
}

func TestExitCodeConstants(t *testing.T) {
	// Verify constants match expected values per spec
	if ExitCodeSuccess != 0 {
		t.Errorf("ExitCodeSuccess should be 0, got %d", ExitCodeSuccess)
	}
	if ExitCodeError != 1 {
		t.Errorf("ExitCodeError should be 1, got %d", ExitCodeError)
	}
	if ExitCodeUsage != 2 {
		t.Errorf("ExitCodeUsage should be 2, got %d", ExitCodeUsage)
	}
}

func TestErrorCodeConstants(t *testing.T) {
	// Verify all error codes are defined and non-empty
	codes := []string{
		ErrCodeWorkspaceNotFound,
		ErrCodeInvalidInput,
		ErrCodeUsage,
		ErrCodeTimeout,
		ErrCodeInternal,
	}

	seen := make(map[string]bool)
	for _, code := range codes {
		if code == "" {
			t.Error("error code should not be empty")
		}
		if seen[code] {
			t.Errorf("duplicate error code: %s", code)
		}
		seen[code] = true
	}
}

// Test that JSON output is properly indented
func TestOutputJSONIndentation(t *testing.T) {
	output := captureStdout(func() {
		OutputJSON(map[string]string{"key": "value"})
	})

	// Should have indentation (2 spaces)
	if !strings.Contains(output, "  ") {
		t.Errorf("expected indented JSON output")
	}
}

// Test output with very large data
func TestOutputJSONLargeData(t *testing.T) {
	// Create a large slice
	data := make([]int, 10000)
	for i := range data {
		data[i] = i
	}

	output := captureStdout(func() {
		err := OutputJSON(data)
		if err != nil {
			t.Errorf("failed to output large data: %v", err)
		}
	})

	// Verify it's valid JSON
	var parsed []int
	if err := json.Unmarshal([]byte(output), &parsed); err != nil {
		t.Errorf("output is not valid JSON: %v", err)
	}
	if len(parsed) != 10000 {
		t.Errorf("expected 10000 elements, got %d", len(parsed))
	}
}

// Test output with deeply nested structures
func TestOutputJSONDeepNesting(t *testing.T) {
	type Nested struct {
		Level int     `json:"level"`
		Child *Nested `json:"child,omitempty"`
	}

	// Create a 10-level deep structure
	root := &Nested{Level: 0}
	current := root
	for i := 1; i < 10; i++ {
		current.Child = &Nested{Level: i}
		current = current.Child
	}

	output := captureStdout(func() {
		err := OutputJSON(root)
		if err != nil {
			t.Errorf("failed to output nested data: %v", err)
		}
	})

	// Verify it's valid JSON
	var parsed Nested
	if err := json.Unmarshal([]byte(output), &parsed); err != nil {
		t.Errorf("output is not valid JSON: %v", err)
	}

	// Verify depth
	current = &parsed
	for i := 0; i < 9; i++ {
		if current.Level != i {
			t.Errorf("expected level %d, got %d", i, current.Level)
		}
		current = current.Child
	}
}

// Test output with nil pointer in struct
func TestOutputJSONNilPointers(t *testing.T) {
	type WithPointer struct {
		Name  string  `json:"name"`
		Value *int    `json:"value,omitempty"`
		Inner *struct {
			Field string `json:"field"`
		} `json:"inner,omitempty"`
	}

	data := WithPointer{Name: "test"}

	output := captureStdout(func() {
		err := OutputJSON(data)
		if err != nil {
			t.Errorf("failed to output: %v", err)
		}
	})

	// Should not contain "value" or "inner" keys (omitempty)
	if strings.Contains(output, `"value"`) {
		t.Error("expected value to be omitted")
	}
	if strings.Contains(output, `"inner"`) {
		t.Error("expected inner to be omitted")
	}
}

// Test output with empty collections
func TestOutputJSONEmptyCollections(t *testing.T) {
	tests := []struct {
		name     string
		input    interface{}
		expected string
	}{
		{"empty slice", []string{}, "[]"},
		{"empty map", map[string]int{}, "{}"},
		{"nil slice", []string(nil), "null"},
		{"nil map", map[string]int(nil), "null"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := captureStdout(func() {
				OutputJSON(tt.input)
			})
			trimmed := strings.TrimSpace(output)
			if trimmed != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, trimmed)
			}
		})
	}
}

// Test output with various number types
func TestOutputJSONNumbers(t *testing.T) {
	type Numbers struct {
		Int     int     `json:"int"`
		Int64   int64   `json:"int64"`
		Float32 float32 `json:"float32"`
		Float64 float64 `json:"float64"`
		Uint    uint    `json:"uint"`
	}

	data := Numbers{
		Int:     -42,
		Int64:   9223372036854775807,
		Float32: 3.14159,
		Float64: 2.718281828459045,
		Uint:    4294967295,
	}

	output := captureStdout(func() {
		OutputJSON(data)
	})

	var parsed Numbers
	if err := json.Unmarshal([]byte(output), &parsed); err != nil {
		t.Fatalf("failed to parse: %v", err)
	}

	if parsed.Int != -42 {
		t.Errorf("int mismatch: %d", parsed.Int)
	}
	if parsed.Int64 != 9223372036854775807 {
		t.Errorf("int64 mismatch: %d", parsed.Int64)
	}
}

// Test output with boolean values
func TestOutputJSONBooleans(t *testing.T) {
	data := map[string]bool{
		"true_val":  true,
		"false_val": false,
	}

	output := captureStdout(func() {
		OutputJSON(data)
	})

	if !strings.Contains(output, "true") {
		t.Error("expected true in output")
	}
	if !strings.Contains(output, "false") {
		t.Error("expected false in output")
	}
}

// Test output with time.Time
func TestOutputJSONTime(t *testing.T) {
	type WithTime struct {
		CreatedAt string `json:"created_at"`
	}

	data := WithTime{CreatedAt: "2025-01-27T10:00:00Z"}

	output := captureStdout(func() {
		OutputJSON(data)
	})

	if !strings.Contains(output, "2025-01-27T10:00:00Z") {
		t.Errorf("expected timestamp in output: %s", output)
	}
}

// Test TextOutput with custom type
func TestOutputTextWithTextOutputInterface(t *testing.T) {
	// VersionInfo implements TextOutput
	info := VersionInfo{
		Version:   "1.0.0",
		Commit:    "abc123",
		BuildTime: "now",
		GoVersion: "go1.22",
		OS:        "darwin",
		Arch:      "amd64",
	}

	output := captureStdout(func() {
		OutputText(info)
	})

	// Should contain formatted text, not JSON
	if strings.Contains(output, `"version"`) {
		t.Error("expected text output, got JSON")
	}
	if !strings.Contains(output, "1.0.0") {
		t.Error("expected version in output")
	}
}

// Test OutputResult switches based on flagJSON
func TestOutputResultModeSwitch(t *testing.T) {
	oldFlag := flagJSON
	defer func() { flagJSON = oldFlag }()

	// Test with a type that implements TextOutput
	info := VersionInfo{
		Version: "1.0.0",
		Commit:  "abc",
	}

	// JSON mode
	flagJSON = true
	jsonOutput := captureStdout(func() {
		OutputResult(info)
	})
	if !strings.Contains(jsonOutput, `"version"`) {
		t.Error("expected JSON in JSON mode")
	}

	// Text mode
	flagJSON = false
	textOutput := captureStdout(func() {
		OutputResult(info)
	})
	if strings.Contains(textOutput, `"version"`) {
		t.Error("expected text in text mode")
	}
}

// Test getErrorCode with various error messages
func TestGetErrorCodeEdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected string
	}{
		{"not found (workspace)", errors.New("workspace not found"), ErrCodeWorkspaceNotFound},
		{"not found (generic)", errors.New("NOT FOUND"), ErrCodeInternal},
		{"not found (item)", errors.New("Item Not Found"), ErrCodeInternal},
		{"invalid (lowercase)", errors.New("invalid input"), ErrCodeInvalidInput},
		{"invalid (uppercase)", errors.New("INVALID VALUE"), ErrCodeInvalidInput},
		{"timeout (lowercase)", errors.New("connection timeout"), ErrCodeTimeout},
		{"timeout (phrase)", errors.New("operation timed out"), ErrCodeTimeout},
		{"unknown flag", errors.New("unknown flag: --xyz"), ErrCodeUsage},
		{"requires args", errors.New("requires at least 2 arg(s)"), ErrCodeUsage},
		{"generic error", errors.New("something happened"), ErrCodeInternal},
		{"empty error", errors.New(""), ErrCodeInternal},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getErrorCode(tt.err)
			if got != tt.expected {
				t.Errorf("getErrorCode(%q) = %s, want %s", tt.err.Error(), got, tt.expected)
			}
		})
	}
}

// Test wrapped errors
func TestGetExitCodeWrappedErrors(t *testing.T) {
	// Wrapped UsageError should still return UsageExitCode
	wrapped := fmt.Errorf("wrapper: %w", NewUsageError("inner usage error"))
	if GetExitCode(wrapped) != ExitCodeUsage {
		t.Error("wrapped UsageError should return ExitCodeUsage")
	}
}

// Test NewUsageError with various messages
func TestNewUsageErrorVariants(t *testing.T) {
	tests := []struct {
		msg string
	}{
		{"simple message"},
		{"message with: special chars!"},
		{"multi\nline\nmessage"},
		{""},
		{"unicode: 日本語"},
	}

	for _, tt := range tests {
		t.Run(tt.msg, func(t *testing.T) {
			err := NewUsageError(tt.msg)
			if err.Error() != tt.msg {
				t.Errorf("Error() = %q, want %q", err.Error(), tt.msg)
			}
			if GetExitCode(err) != ExitCodeUsage {
				t.Error("should return ExitCodeUsage")
			}
		})
	}
}

// Test error detail with complex details
func TestErrorDetailComplexDetails(t *testing.T) {
	detail := ErrorDetail{
		Code:    "COMPLEX_ERROR",
		Message: "Complex error occurred",
		Details: map[string]interface{}{
			"string_val":  "test",
			"int_val":     42,
			"float_val":   3.14,
			"bool_val":    true,
			"array_val":   []int{1, 2, 3},
			"nested_val":  map[string]string{"key": "value"},
			"nil_val":     nil,
		},
	}

	data, err := json.Marshal(detail)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var parsed ErrorDetail
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if parsed.Details["string_val"] != "test" {
		t.Error("string_val mismatch")
	}
	// Note: JSON numbers unmarshal as float64
	if parsed.Details["int_val"].(float64) != 42 {
		t.Error("int_val mismatch")
	}
}

// Test OutputError with nil error
func TestOutputErrorWithNil(t *testing.T) {
	ResetErrorOutput()
	oldFlagJSON := flagJSON
	defer func() { flagJSON = oldFlagJSON }()

	flagJSON = true
	output := captureStdout(func() {
		OutputError(nil)
	})

	// Should not output anything for nil error
	if output != "" {
		t.Errorf("expected empty output for nil error, got: %s", output)
	}
}

// Test error output deduplication
func TestOutputErrorDeduplication(t *testing.T) {
	ResetErrorOutput()
	oldFlagJSON := flagJSON
	defer func() { flagJSON = oldFlagJSON }()

	flagJSON = true
	err := errors.New("duplicate test error")

	// First call should output
	output1 := captureStdout(func() {
		OutputError(err)
	})
	if output1 == "" {
		t.Error("first OutputError call should produce output")
	}

	// Second call with same error should NOT output
	output2 := captureStdout(func() {
		OutputError(err)
	})
	if output2 != "" {
		t.Errorf("second OutputError call should not produce output, got: %s", output2)
	}

	// After reset, should output again
	ResetErrorOutput()
	output3 := captureStdout(func() {
		OutputError(err)
	})
	if output3 == "" {
		t.Error("OutputError should produce output after reset")
	}
}

// Test different errors are not deduplicated
func TestOutputErrorDifferentErrors(t *testing.T) {
	ResetErrorOutput()
	oldFlagJSON := flagJSON
	defer func() { flagJSON = oldFlagJSON }()

	flagJSON = true

	err1 := errors.New("first error")
	err2 := errors.New("second error")

	output1 := captureStdout(func() {
		OutputError(err1)
	})
	if output1 == "" {
		t.Error("first error should produce output")
	}

	output2 := captureStdout(func() {
		OutputError(err2)
	})
	if output2 == "" {
		t.Error("different error should produce output")
	}

	// Verify the messages are different
	if !strings.Contains(output1, "first error") {
		t.Error("output1 should contain first error")
	}
	if !strings.Contains(output2, "second error") {
		t.Error("output2 should contain second error")
	}
}

// Test ResetErrorOutput function
func TestResetErrorOutput(t *testing.T) {
	ResetErrorOutput()
	oldFlagJSON := flagJSON
	defer func() { flagJSON = oldFlagJSON }()

	flagJSON = false
	err := errors.New("reset test error")

	// Output error
	captureStderr(func() {
		OutputError(err)
	})

	// Reset
	ResetErrorOutput()

	// Should be able to output same error again
	output := captureStderr(func() {
		OutputError(err)
	})
	if !strings.Contains(output, "reset test error") {
		t.Error("should output error after reset")
	}
}

// Test WorkspaceError type in output module
func TestWorkspaceErrorTypeOutput(t *testing.T) {
	err := NewWorkspaceError("workspace not available")

	// Should implement error interface
	if err.Error() != "workspace not available" {
		t.Errorf("Error() = %q, want %q", err.Error(), "workspace not available")
	}

	// Should be detectable as WorkspaceError
	var wsErr *WorkspaceError
	if !errors.As(err, &wsErr) {
		t.Error("expected errors.As to match WorkspaceError")
	}
}

// Test WorkspaceError returns correct error code
func TestWorkspaceErrorCode(t *testing.T) {
	err := NewWorkspaceError("any message")
	code := getErrorCode(err)
	if code != ErrCodeWorkspaceNotFound {
		t.Errorf("getErrorCode(WorkspaceError) = %s, want %s", code, ErrCodeWorkspaceNotFound)
	}
}

// Test wrapped WorkspaceError
func TestWrappedWorkspaceError(t *testing.T) {
	innerErr := NewWorkspaceError("inner workspace error")
	wrapped := fmt.Errorf("outer: %w", innerErr)

	// Should be detectable
	var wsErr *WorkspaceError
	if !errors.As(wrapped, &wsErr) {
		t.Error("wrapped WorkspaceError should be detectable")
	}

	// Should return correct error code
	code := getErrorCode(wrapped)
	if code != ErrCodeWorkspaceNotFound {
		t.Errorf("getErrorCode(wrapped WorkspaceError) = %s, want %s", code, ErrCodeWorkspaceNotFound)
	}
}

// Test error code detection for various workspace-related messages
func TestWorkspaceErrorMessagePatterns(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		wantCode string
	}{
		{"typed WorkspaceError", NewWorkspaceError("test"), ErrCodeWorkspaceNotFound},
		{"not in a DBA workspace", errors.New("not in a DBA workspace"), ErrCodeWorkspaceNotFound},
		{"workspace not found", errors.New("workspace ws_123 not found"), ErrCodeWorkspaceNotFound},
		{"workspace does not exist", errors.New("workspace does not exist"), ErrCodeWorkspaceNotFound},
		{"not in workspace (uppercase)", errors.New("NOT IN A DBA WORKSPACE"), ErrCodeWorkspaceNotFound},
		{"generic not found", errors.New("resource not found"), ErrCodeInternal},
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

// Test GetExitCode with WorkspaceError
func TestGetExitCodeWithWorkspaceError(t *testing.T) {
	err := NewWorkspaceError("workspace not found")
	exitCode := GetExitCode(err)
	// WorkspaceError should return ExitCodeError (1), not ExitCodeUsage (2)
	if exitCode != ExitCodeError {
		t.Errorf("GetExitCode(WorkspaceError) = %d, want %d", exitCode, ExitCodeError)
	}
}

// Test accepts arg error pattern
func TestAcceptsArgErrorPattern(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		wantCode int
	}{
		{"accepts 1 arg", errors.New("accepts 1 arg(s), received 0"), ExitCodeUsage},
		{"accepts 2 args", errors.New("accepts 2 arg(s), received 3"), ExitCodeUsage},
		{"accepts between", errors.New("accepts between 1 and 3 arg(s), received 5"), ExitCodeUsage},
		{"accepts at most", errors.New("accepts at most 2 arg(s), received 5"), ExitCodeUsage},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			code := GetExitCode(tt.err)
			if code != tt.wantCode {
				t.Errorf("GetExitCode() = %d, want %d", code, tt.wantCode)
			}
		})
	}
}

// Test output with interface{} type that's nil
func TestOutputJSONWithTypedNil(t *testing.T) {
	var ptr *string = nil
	output := captureStdout(func() {
		OutputJSON(ptr)
	})
	if strings.TrimSpace(output) != "null" {
		t.Errorf("expected null for typed nil pointer, got: %s", output)
	}
}

// Test error message with special characters
func TestOutputErrorSpecialCharacters(t *testing.T) {
	ResetErrorOutput()
	oldFlagJSON := flagJSON
	defer func() { flagJSON = oldFlagJSON }()

	tests := []struct {
		name string
		msg  string
	}{
		{"quotes", `error with "quotes"`},
		{"backslash", `error with \ backslash`},
		{"newline", "error with\nnewline"},
		{"tab", "error with\ttab"},
		{"unicode", "error with 日本語"},
		{"emoji", "error with 🚀 emoji"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ResetErrorOutput()
			flagJSON = true
			err := errors.New(tt.msg)

			output := captureStdout(func() {
				OutputError(err)
			})

			// Should be valid JSON
			var resp ErrorResponse
			if jsonErr := json.Unmarshal([]byte(output), &resp); jsonErr != nil {
				t.Errorf("output is not valid JSON: %v, output: %s", jsonErr, output)
			}
			if resp.Error.Message != tt.msg {
				t.Errorf("message = %q, want %q", resp.Error.Message, tt.msg)
			}
		})
	}
}

// Test very long error messages
func TestOutputErrorLongMessage(t *testing.T) {
	ResetErrorOutput()
	oldFlagJSON := flagJSON
	defer func() { flagJSON = oldFlagJSON }()

	// Create a very long error message
	longMsg := strings.Repeat("a", 10000)
	err := errors.New(longMsg)

	flagJSON = true
	output := captureStdout(func() {
		OutputError(err)
	})

	var resp ErrorResponse
	if jsonErr := json.Unmarshal([]byte(output), &resp); jsonErr != nil {
		t.Fatalf("failed to parse JSON: %v", jsonErr)
	}
	if len(resp.Error.Message) != 10000 {
		t.Errorf("message length = %d, want 10000", len(resp.Error.Message))
	}
}

// Test OutputJSON with channel (should error)
func TestOutputJSONWithUnsupportedType(t *testing.T) {
	ch := make(chan int)
	output := captureStdout(func() {
		err := OutputJSON(ch)
		if err == nil {
			t.Error("expected error for channel type")
		}
	})
	// Output should be empty or contain error info
	_ = output
}

// Test OutputJSON with function (should error)
func TestOutputJSONWithFunction(t *testing.T) {
	fn := func() {}
	output := captureStdout(func() {
		err := OutputJSON(fn)
		if err == nil {
			t.Error("expected error for function type")
		}
	})
	_ = output
}

// Test error code priority (typed errors should take precedence)
func TestErrorCodePriority(t *testing.T) {
	// UsageError with "not found" in message should return USAGE_ERROR, not WORKSPACE_NOT_FOUND
	err := NewUsageError("not found in args")
	code := getErrorCode(err)
	if code != ErrCodeUsage {
		t.Errorf("UsageError should return USAGE_ERROR regardless of message, got %s", code)
	}

	// WorkspaceError with "invalid" in message should return WORKSPACE_NOT_FOUND
	err2 := NewWorkspaceError("invalid workspace path")
	code2 := getErrorCode(err2)
	if code2 != ErrCodeWorkspaceNotFound {
		t.Errorf("WorkspaceError should return WORKSPACE_NOT_FOUND regardless of message, got %s", code2)
	}
}
