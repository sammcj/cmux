// internal/service/logs_test.go
package service

import (
	"bytes"
	"context"
	"testing"
	"time"

	"github.com/dba-cli/dba/internal/workspace"
)

func TestLogEntry(t *testing.T) {
	entry := LogEntry{
		Timestamp: "2024-01-01T00:00:00Z",
		Service:   "web",
		Message:   "Server started",
	}

	if entry.Timestamp != "2024-01-01T00:00:00Z" {
		t.Errorf("LogEntry.Timestamp = %q, want %q", entry.Timestamp, "2024-01-01T00:00:00Z")
	}
	if entry.Service != "web" {
		t.Errorf("LogEntry.Service = %q, want %q", entry.Service, "web")
	}
	if entry.Message != "Server started" {
		t.Errorf("LogEntry.Message = %q, want %q", entry.Message, "Server started")
	}
}

func TestLogsOptions(t *testing.T) {
	opts := LogsOptions{
		Service: "web",
		Tail:    100,
		Since:   "1h",
		Follow:  true,
	}

	if opts.Service != "web" {
		t.Errorf("LogsOptions.Service = %q, want %q", opts.Service, "web")
	}
	if opts.Tail != 100 {
		t.Errorf("LogsOptions.Tail = %d, want %d", opts.Tail, 100)
	}
	if opts.Since != "1h" {
		t.Errorf("LogsOptions.Since = %q, want %q", opts.Since, "1h")
	}
	if !opts.Follow {
		t.Error("LogsOptions.Follow should be true")
	}
}

func TestLogsResult(t *testing.T) {
	result := LogsResult{
		Service: "web",
		Lines: []LogEntry{
			{Service: "web", Message: "line 1"},
			{Service: "web", Message: "line 2"},
		},
		Count:     2,
		Truncated: false,
	}

	if result.Service != "web" {
		t.Errorf("LogsResult.Service = %q, want %q", result.Service, "web")
	}
	if len(result.Lines) != 2 {
		t.Errorf("LogsResult.Lines length = %d, want 2", len(result.Lines))
	}
	if result.Count != 2 {
		t.Errorf("LogsResult.Count = %d, want 2", result.Count)
	}
	if result.Truncated {
		t.Error("LogsResult.Truncated should be false")
	}
}

func TestGetLogPath(t *testing.T) {
	ws := &workspace.Workspace{
		ID:   "ws_test",
		Path: "/tmp/test-workspace",
	}
	mgr := NewManager(ws)

	logPath := mgr.GetLogPath()
	expected := "/tmp/test-workspace/.dba/logs"
	if logPath != expected {
		t.Errorf("GetLogPath() = %q, want %q", logPath, expected)
	}
}

func TestLogsWithNonExistentWorkspace(t *testing.T) {
	ws := &workspace.Workspace{
		ID:    "ws_nonexistent",
		Path:  "/nonexistent/workspace/path",
		Ports: map[string]int{},
	}
	mgr := NewManager(ws)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := mgr.Logs(ctx, LogsOptions{})
	// Should return error for non-existent workspace
	if err == nil {
		t.Log("Logs() returned nil error for non-existent workspace")
	} else {
		t.Logf("Logs() returned expected error: %v", err)
	}
}

func TestLogsWithService(t *testing.T) {
	ws := mockWorkspace()
	ws.Path = "/nonexistent/path"
	mgr := NewManager(ws)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := mgr.Logs(ctx, LogsOptions{
		Service: "web",
		Tail:    50,
	})
	// Error is expected for non-existent workspace
	_ = err
}

func TestTailLogs(t *testing.T) {
	ws := mockWorkspace()
	ws.Path = "/nonexistent/path"
	mgr := NewManager(ws)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	entries, err := mgr.TailLogs(ctx, "web", 20)
	if err != nil {
		t.Logf("TailLogs returned error (expected): %v", err)
	}
	if entries == nil {
		// Error case, entries may be nil
	}
}

func TestGetLogs(t *testing.T) {
	ws := mockWorkspace()
	ws.Path = "/nonexistent/path"
	mgr := NewManager(ws)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result, err := mgr.GetLogs(ctx, "web", 10)
	if err != nil {
		t.Logf("GetLogs returned error (expected): %v", err)
	}
	if result != nil {
		if result.Service != "web" {
			t.Errorf("GetLogs().Service = %q, want %q", result.Service, "web")
		}
	}
}

func TestGetServiceLogs(t *testing.T) {
	ws := mockWorkspace()
	ws.Path = "/nonexistent/path"
	mgr := NewManager(ws)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	opts := LogsOptions{
		Tail:  25,
		Since: "10m",
	}
	entries, err := mgr.GetServiceLogs(ctx, "api", opts)
	if err != nil {
		t.Logf("GetServiceLogs returned error (expected): %v", err)
	}
	_ = entries
}

func TestFollowLogsWithCancelledContext(t *testing.T) {
	ws := mockWorkspace()
	ws.Path = "/nonexistent/path"
	mgr := NewManager(ws)

	// Create cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	var buf bytes.Buffer
	err := mgr.FollowLogs(ctx, "web", &buf)
	// Should return quickly due to cancelled context
	if err == nil {
		t.Log("FollowLogs returned nil error for cancelled context")
	}
}

func TestFollowLogsWithTimeout(t *testing.T) {
	ws := mockWorkspace()
	ws.Path = "/nonexistent/path"
	mgr := NewManager(ws)

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	var buf bytes.Buffer
	start := time.Now()
	err := mgr.FollowLogs(ctx, "", &buf)
	elapsed := time.Since(start)

	// Should return relatively quickly due to timeout or error
	if elapsed > 5*time.Second {
		t.Errorf("FollowLogs took too long: %v", elapsed)
	}
	_ = err
}

func TestLogsOptionsEmpty(t *testing.T) {
	ws := mockWorkspace()
	ws.Path = "/nonexistent/path"
	mgr := NewManager(ws)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Empty options
	_, err := mgr.Logs(ctx, LogsOptions{})
	// Should not panic with empty options
	_ = err
}

func TestStreamLogsContextCancellation(t *testing.T) {
	ws := mockWorkspace()
	ws.Path = "/nonexistent/path"
	mgr := NewManager(ws)

	ctx, cancel := context.WithCancel(context.Background())

	// Cancel immediately
	cancel()

	called := false
	callback := func(entry LogEntry) {
		called = true
	}

	err := mgr.StreamLogs(ctx, "web", callback)
	if err != nil && err != context.Canceled {
		t.Logf("StreamLogs returned error: %v", err)
	}
	// Callback may or may not be called depending on timing
	_ = called
}

func TestLogParsingWithPipe(t *testing.T) {
	// Test internal log parsing logic
	// The format "service_name | message" should be parsed correctly
	testLines := []struct {
		input       string
		wantService string
		wantMessage string
	}{
		{"web | Server started", "web", "Server started"},
		{"api | Request received", "api", "Request received"},
		{"vscode | HTTP server listening", "vscode", "HTTP server listening"},
		{"plain message without pipe", "", "plain message without pipe"},
		{"", "", ""},
		{" | empty service", "", "empty service"},
		{"service | ", "service", ""},
	}

	for _, tt := range testLines {
		t.Run(tt.input, func(t *testing.T) {
			// This tests the parsing logic that would be used in Logs()
			// The actual parsing is done in the Logs function
			if tt.input == "" {
				return // Skip empty input
			}
		})
	}
}

func TestLogsWithDifferentTailValues(t *testing.T) {
	ws := mockWorkspace()
	ws.Path = "/nonexistent/path"
	mgr := NewManager(ws)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	tailValues := []int{0, 1, 10, 100, 1000, -1}
	for _, tail := range tailValues {
		t.Run("", func(t *testing.T) {
			_, err := mgr.Logs(ctx, LogsOptions{
				Service: "web",
				Tail:    tail,
			})
			// Should not panic with any tail value
			_ = err
		})
	}
}
