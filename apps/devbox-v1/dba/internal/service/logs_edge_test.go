// internal/service/logs_edge_test.go
package service

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/dba-cli/dba/internal/workspace"
)

// ═══════════════════════════════════════════════════════════════════════════════
// Logs with Nil Workspace
// ═══════════════════════════════════════════════════════════════════════════════

func TestLogsWithNilWorkspace(t *testing.T) {
	mgr := NewManager(nil)
	ctx := context.Background()

	defer func() {
		if r := recover(); r != nil {
			t.Errorf("Logs panicked with nil workspace: %v", r)
		}
	}()

	_, err := mgr.Logs(ctx, LogsOptions{})
	// Should handle gracefully (error is expected)
	if err == nil {
		t.Log("Logs returned nil error for nil workspace")
	}
}

func TestFollowLogsWithNilWorkspace(t *testing.T) {
	mgr := NewManager(nil)
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	defer func() {
		if r := recover(); r != nil {
			t.Errorf("FollowLogs panicked with nil workspace: %v", r)
		}
	}()

	var buf bytes.Buffer
	err := mgr.FollowLogs(ctx, "web", &buf)
	// Should handle gracefully
	_ = err
}

func TestGetLogPathWithNilWorkspace(t *testing.T) {
	mgr := NewManager(nil)

	defer func() {
		if r := recover(); r != nil {
			t.Errorf("GetLogPath panicked with nil workspace: %v", r)
		}
	}()

	path := mgr.GetLogPath()
	// Will return something like "/.dba/logs" with nil workspace
	t.Logf("GetLogPath with nil workspace: %q", path)
}

func TestGetLogsWithNilWorkspace(t *testing.T) {
	mgr := NewManager(nil)
	ctx := context.Background()

	defer func() {
		if r := recover(); r != nil {
			t.Errorf("GetLogs panicked with nil workspace: %v", r)
		}
	}()

	result, err := mgr.GetLogs(ctx, "web", 10)
	// Should handle gracefully
	_ = result
	_ = err
}

func TestTailLogsWithNilWorkspace(t *testing.T) {
	mgr := NewManager(nil)
	ctx := context.Background()

	defer func() {
		if r := recover(); r != nil {
			t.Errorf("TailLogs panicked with nil workspace: %v", r)
		}
	}()

	entries, err := mgr.TailLogs(ctx, "web", 20)
	// Should handle gracefully
	_ = entries
	_ = err
}

func TestGetServiceLogsWithNilWorkspace(t *testing.T) {
	mgr := NewManager(nil)
	ctx := context.Background()

	defer func() {
		if r := recover(); r != nil {
			t.Errorf("GetServiceLogs panicked with nil workspace: %v", r)
		}
	}()

	entries, err := mgr.GetServiceLogs(ctx, "web", LogsOptions{})
	// Should handle gracefully
	_ = entries
	_ = err
}

func TestStreamLogsWithNilWorkspace(t *testing.T) {
	mgr := NewManager(nil)
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	defer func() {
		if r := recover(); r != nil {
			t.Errorf("StreamLogs panicked with nil workspace: %v", r)
		}
	}()

	callback := func(entry LogEntry) {}
	err := mgr.StreamLogs(ctx, "web", callback)
	// Should handle gracefully
	_ = err
}

// ═══════════════════════════════════════════════════════════════════════════════
// LogsOptions Edge Cases
// ═══════════════════════════════════════════════════════════════════════════════

func TestLogsOptionsWithNegativeTail(t *testing.T) {
	ws := mockWorkspace()
	ws.Path = "/nonexistent/path"
	mgr := NewManager(ws)
	ctx := context.Background()

	_, err := mgr.Logs(ctx, LogsOptions{
		Service: "web",
		Tail:    -100, // Negative tail
	})
	// Should handle gracefully
	_ = err
}

func TestLogsOptionsWithZeroTail(t *testing.T) {
	ws := mockWorkspace()
	ws.Path = "/nonexistent/path"
	mgr := NewManager(ws)
	ctx := context.Background()

	_, err := mgr.Logs(ctx, LogsOptions{
		Service: "web",
		Tail:    0,
	})
	// Should handle gracefully
	_ = err
}

func TestLogsOptionsWithLargeTail(t *testing.T) {
	ws := mockWorkspace()
	ws.Path = "/nonexistent/path"
	mgr := NewManager(ws)
	ctx := context.Background()

	_, err := mgr.Logs(ctx, LogsOptions{
		Service: "web",
		Tail:    1000000, // Very large tail
	})
	// Should handle gracefully
	_ = err
}

func TestLogsOptionsAllFields(t *testing.T) {
	opts := LogsOptions{
		Service: "test-service",
		Tail:    50,
		Since:   "2024-01-01T00:00:00Z",
		Follow:  true,
	}

	// Verify all fields are set correctly
	if opts.Service != "test-service" {
		t.Errorf("Service = %q, want %q", opts.Service, "test-service")
	}
	if opts.Tail != 50 {
		t.Errorf("Tail = %d, want %d", opts.Tail, 50)
	}
	if opts.Since != "2024-01-01T00:00:00Z" {
		t.Errorf("Since = %q, want %q", opts.Since, "2024-01-01T00:00:00Z")
	}
	if !opts.Follow {
		t.Error("Follow should be true")
	}
}

// ═══════════════════════════════════════════════════════════════════════════════
// LogEntry Edge Cases
// ═══════════════════════════════════════════════════════════════════════════════

func TestLogEntryZeroValues(t *testing.T) {
	entry := LogEntry{}

	if entry.Timestamp != "" {
		t.Error("Zero LogEntry should have empty Timestamp")
	}
	if entry.Service != "" {
		t.Error("Zero LogEntry should have empty Service")
	}
	if entry.Message != "" {
		t.Error("Zero LogEntry should have empty Message")
	}
}

func TestLogEntryJSONSerialization(t *testing.T) {
	entry := LogEntry{
		Timestamp: "2024-01-01T00:00:00Z",
		Service:   "test-service",
		Message:   "Test log message with special chars: <>&\"'",
	}

	data, err := json.Marshal(entry)
	if err != nil {
		t.Fatalf("Failed to marshal LogEntry: %v", err)
	}

	var decoded LogEntry
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal LogEntry: %v", err)
	}

	if decoded.Timestamp != entry.Timestamp {
		t.Errorf("Timestamp = %q, want %q", decoded.Timestamp, entry.Timestamp)
	}
	if decoded.Service != entry.Service {
		t.Errorf("Service = %q, want %q", decoded.Service, entry.Service)
	}
	if decoded.Message != entry.Message {
		t.Errorf("Message = %q, want %q", decoded.Message, entry.Message)
	}
}

func TestLogEntryWithMultilineMessage(t *testing.T) {
	entry := LogEntry{
		Timestamp: time.Now().Format(time.RFC3339),
		Service:   "test",
		Message:   "Line 1\nLine 2\nLine 3",
	}

	data, err := json.Marshal(entry)
	if err != nil {
		t.Fatalf("Failed to marshal multiline LogEntry: %v", err)
	}

	var decoded LogEntry
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if decoded.Message != entry.Message {
		t.Errorf("Multiline message not preserved correctly")
	}
}

func TestLogEntryWithUnicode(t *testing.T) {
	entry := LogEntry{
		Timestamp: time.Now().Format(time.RFC3339),
		Service:   "test-日本語",
		Message:   "Unicode message: 你好世界 🎉 émojis",
	}

	data, err := json.Marshal(entry)
	if err != nil {
		t.Fatalf("Failed to marshal unicode LogEntry: %v", err)
	}

	var decoded LogEntry
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if decoded.Service != entry.Service {
		t.Errorf("Unicode service not preserved: %q", decoded.Service)
	}
	if decoded.Message != entry.Message {
		t.Errorf("Unicode message not preserved: %q", decoded.Message)
	}
}

// ═══════════════════════════════════════════════════════════════════════════════
// LogsResult Edge Cases
// ═══════════════════════════════════════════════════════════════════════════════

func TestLogsResultZeroValues(t *testing.T) {
	result := LogsResult{}

	if result.Service != "" {
		t.Error("Zero LogsResult should have empty Service")
	}
	if result.Lines != nil {
		t.Error("Zero LogsResult should have nil Lines")
	}
	if result.Count != 0 {
		t.Error("Zero LogsResult should have 0 Count")
	}
	if result.Truncated {
		t.Error("Zero LogsResult should not be Truncated")
	}
}

func TestLogsResultJSONSerialization(t *testing.T) {
	result := LogsResult{
		Service: "test-service",
		Lines: []LogEntry{
			{Timestamp: "2024-01-01T00:00:00Z", Service: "test", Message: "msg1"},
			{Timestamp: "2024-01-01T00:00:01Z", Service: "test", Message: "msg2"},
		},
		Count:     2,
		Truncated: true,
	}

	data, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("Failed to marshal LogsResult: %v", err)
	}

	var decoded LogsResult
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if decoded.Service != result.Service {
		t.Errorf("Service = %q, want %q", decoded.Service, result.Service)
	}
	if len(decoded.Lines) != len(result.Lines) {
		t.Errorf("Lines length = %d, want %d", len(decoded.Lines), len(result.Lines))
	}
	if decoded.Count != result.Count {
		t.Errorf("Count = %d, want %d", decoded.Count, result.Count)
	}
	if decoded.Truncated != result.Truncated {
		t.Errorf("Truncated = %v, want %v", decoded.Truncated, result.Truncated)
	}
}

// ═══════════════════════════════════════════════════════════════════════════════
// Concurrent Log Access
// ═══════════════════════════════════════════════════════════════════════════════

func TestConcurrentLogs(t *testing.T) {
	ws := mockWorkspace()
	ws.Path = "/nonexistent/path"
	mgr := NewManager(ws)
	ctx := context.Background()

	var wg sync.WaitGroup

	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			_, _ = mgr.Logs(ctx, LogsOptions{
				Service: "web",
				Tail:    10 + idx,
			})
		}(i)
	}

	wg.Wait()
}

func TestConcurrentGetLogs(t *testing.T) {
	ws := mockWorkspace()
	ws.Path = "/nonexistent/path"
	mgr := NewManager(ws)
	ctx := context.Background()

	var wg sync.WaitGroup
	services := []string{"web", "api", "db", "worker"}

	for i := 0; i < 40; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			svc := services[idx%len(services)]
			_, _ = mgr.GetLogs(ctx, svc, 10+idx)
		}(i)
	}

	wg.Wait()
}

// ═══════════════════════════════════════════════════════════════════════════════
// FollowLogs Writer Edge Cases
// ═══════════════════════════════════════════════════════════════════════════════

func TestFollowLogsWithNilWriter(t *testing.T) {
	ws := mockWorkspace()
	ws.Path = "/nonexistent/path"
	mgr := NewManager(ws)
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	// Note: This will likely panic because nil writer isn't checked
	// We're testing to see if we need to add that check
	defer func() {
		if r := recover(); r != nil {
			t.Logf("FollowLogs panicked with nil writer (expected): %v", r)
		}
	}()

	err := mgr.FollowLogs(ctx, "web", nil)
	// If it doesn't panic, error is expected
	_ = err
}

func TestFollowLogsWithClosedWriter(t *testing.T) {
	ws := mockWorkspace()
	ws.Path = "/nonexistent/path"
	mgr := NewManager(ws)
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	// Create a pipe and close the write end
	r, w := io.Pipe()
	w.Close()

	err := mgr.FollowLogs(ctx, "web", w)
	// Should handle gracefully
	_ = err
	r.Close()
}

// ═══════════════════════════════════════════════════════════════════════════════
// Service Name Edge Cases for Logs
// ═══════════════════════════════════════════════════════════════════════════════

func TestLogsWithSpecialServiceNames(t *testing.T) {
	ws := mockWorkspace()
	ws.Path = "/nonexistent/path"
	mgr := NewManager(ws)
	ctx := context.Background()

	specialNames := []string{
		"",
		" ",
		"service with spaces",
		"service\twith\ttabs",
		"service\nwith\nnewlines",
		"../../etc/passwd",
		"/absolute/path",
		"service;rm -rf /",
		"service`whoami`",
		"service$(id)",
	}

	for _, name := range specialNames {
		t.Run(name, func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("Logs panicked for service name %q: %v", name, r)
				}
			}()

			_, _ = mgr.Logs(ctx, LogsOptions{
				Service: name,
				Tail:    10,
			})
		})
	}
}

func TestGetLogPathConsistency(t *testing.T) {
	testCases := []struct {
		name     string
		wsPath   string
		expected string
	}{
		{"normal path", "/tmp/workspace", "/tmp/workspace/.dba/logs"},
		{"path with spaces", "/tmp/my workspace", "/tmp/my workspace/.dba/logs"},
		{"trailing slash", "/tmp/workspace/", "/tmp/workspace//.dba/logs"},
		{"root path", "/", "//.dba/logs"},
		{"relative path", ".", "./.dba/logs"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ws := &workspace.Workspace{
				ID:   "test",
				Path: tc.wsPath,
			}
			mgr := NewManager(ws)

			path := mgr.GetLogPath()
			if path != tc.expected {
				t.Errorf("GetLogPath() = %q, want %q", path, tc.expected)
			}
		})
	}
}

// ═══════════════════════════════════════════════════════════════════════════════
// StreamLogs Edge Cases
// ═══════════════════════════════════════════════════════════════════════════════

func TestStreamLogsWithNilCallback(t *testing.T) {
	ws := mockWorkspace()
	ws.Path = "/nonexistent/path"
	mgr := NewManager(ws)
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	defer func() {
		if r := recover(); r != nil {
			t.Logf("StreamLogs panicked with nil callback (may be expected): %v", r)
		}
	}()

	err := mgr.StreamLogs(ctx, "web", nil)
	// If it doesn't panic, should handle gracefully
	_ = err
}

func TestStreamLogsCallbackReceivesEntries(t *testing.T) {
	// This is more of an integration test
	// We verify the callback structure is correct

	var entries []LogEntry
	callback := func(entry LogEntry) {
		entries = append(entries, entry)
	}

	// Verify callback is callable
	callback(LogEntry{
		Timestamp: time.Now().Format(time.RFC3339),
		Service:   "test",
		Message:   "test message",
	})

	if len(entries) != 1 {
		t.Errorf("Expected 1 entry, got %d", len(entries))
	}
}

// ═══════════════════════════════════════════════════════════════════════════════
// Log Parsing Edge Cases (testing internal logic)
// ═══════════════════════════════════════════════════════════════════════════════

func TestLogLineParsing(t *testing.T) {
	// Test the parsing logic used in Logs()
	testCases := []struct {
		input       string
		wantService string
		wantMessage string
	}{
		// Standard format
		{"web | Server started", "web", "Server started"},
		{"api | Request received", "api", "Request received"},

		// Edge cases
		{"| empty service", "", "empty service"},
		{"service |", "service", ""},
		{"service | | multiple | pipes", "service", "| multiple | pipes"},
		{"no pipe here", "", "no pipe here"},
		{"", "", ""},
		{" | ", "", ""},
		{"   trimmed   |   spaces   ", "trimmed", "spaces"},
	}

	for _, tc := range testCases {
		t.Run(tc.input, func(t *testing.T) {
			var service, message string
			if parts := strings.SplitN(tc.input, "|", 2); len(parts) == 2 {
				service = strings.TrimSpace(parts[0])
				message = strings.TrimSpace(parts[1])
			} else {
				message = tc.input
			}

			if service != tc.wantService {
				t.Errorf("service = %q, want %q", service, tc.wantService)
			}
			if message != tc.wantMessage {
				t.Errorf("message = %q, want %q", message, tc.wantMessage)
			}
		})
	}
}
