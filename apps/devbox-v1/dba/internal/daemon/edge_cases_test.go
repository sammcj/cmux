// Package daemon provides edge case tests for the daemon.
package daemon

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/dba-cli/dba/internal/config"
)

// TestWorkspaceRegistrationEdgeCases tests edge cases for workspace registration
func TestWorkspaceRegistrationEdgeCases(t *testing.T) {
	cfg := setupTestConfig(t)
	d, err := New(cfg)
	if err != nil {
		t.Fatalf("Failed to create daemon: %v", err)
	}

	// Test registering same workspace twice (should update, not fail)
	t.Run("RegisterTwice", func(t *testing.T) {
		d.RegisterWorkspace("ws_dup", "/path/1")
		d.RegisterWorkspace("ws_dup", "/path/2")

		state := d.GetWorkspaceState("ws_dup")
		if state == nil {
			t.Fatal("Workspace should exist")
		}
		if state.Path != "/path/2" {
			t.Errorf("Expected path /path/2, got %s", state.Path)
		}

		// Count should still be 1
		count := d.GetWorkspaceCount()
		if count != 1 {
			t.Errorf("Expected 1 workspace, got %d", count)
		}

		d.UnregisterWorkspace("ws_dup")
	})

	// Test unregistering non-existent workspace (should not panic)
	t.Run("UnregisterNonExistent", func(t *testing.T) {
		// Should not panic
		d.UnregisterWorkspace("ws_nonexistent")
		d.UnregisterWorkspace("ws_nonexistent") // Double unregister
	})

	// Test very long workspace ID
	t.Run("VeryLongWorkspaceID", func(t *testing.T) {
		longID := "ws_" + strings.Repeat("a", 10000)
		d.RegisterWorkspace(longID, "/path")

		state := d.GetWorkspaceState(longID)
		if state == nil {
			t.Fatal("Workspace with long ID should exist")
		}

		d.UnregisterWorkspace(longID)
	})

	// Test special characters in path
	t.Run("SpecialCharactersInPath", func(t *testing.T) {
		specialPath := "/path/with spaces/and\ttabs/and\nnewlines/and'quotes'"
		d.RegisterWorkspace("ws_special", specialPath)

		state := d.GetWorkspaceState("ws_special")
		if state == nil {
			t.Fatal("Workspace should exist")
		}
		if state.Path != specialPath {
			t.Errorf("Expected path with special chars, got %s", state.Path)
		}

		d.UnregisterWorkspace("ws_special")
	})

	// Test unicode in workspace ID and path
	t.Run("UnicodeSupport", func(t *testing.T) {
		unicodeID := "ws_日本語_émoji_🚀"
		unicodePath := "/path/日本語/émoji/🚀"

		d.RegisterWorkspace(unicodeID, unicodePath)

		state := d.GetWorkspaceState(unicodeID)
		if state == nil {
			t.Fatal("Workspace with unicode should exist")
		}
		if state.Path != unicodePath {
			t.Errorf("Expected unicode path, got %s", state.Path)
		}

		d.UnregisterWorkspace(unicodeID)
	})

	// Test empty values (should handle gracefully)
	t.Run("EmptyValues", func(t *testing.T) {
		// Empty ID - should still work (though not recommended)
		d.RegisterWorkspace("", "/path")
		state := d.GetWorkspaceState("")
		if state == nil {
			t.Error("Empty ID workspace should be retrievable")
		}
		d.UnregisterWorkspace("")

		// Empty path
		d.RegisterWorkspace("ws_empty_path", "")
		state = d.GetWorkspaceState("ws_empty_path")
		if state == nil {
			t.Error("Empty path workspace should exist")
		}
		d.UnregisterWorkspace("ws_empty_path")
	})
}

// TestHTTPEdgeCases tests edge cases for HTTP endpoints
func TestHTTPEdgeCases(t *testing.T) {
	cfg := setupTestConfig(t)
	d, err := New(cfg)
	if err != nil {
		t.Fatalf("Failed to create daemon: %v", err)
	}
	router := d.createRouter()

	// Test empty body for POST requests
	t.Run("EmptyBody", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/workspace/register", nil)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected 400 for empty body, got %d", w.Code)
		}
	})

	// Test very large request body
	t.Run("VeryLargeBody", func(t *testing.T) {
		largeBody := fmt.Sprintf(`{"id": "ws_large", "path": "%s"}`, strings.Repeat("a", 100000))
		req := httptest.NewRequest("POST", "/workspace/register", strings.NewReader(largeBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		// Should still work
		if w.Code != http.StatusOK {
			t.Errorf("Expected 200, got %d", w.Code)
		}
	})

	// Test invalid content type
	t.Run("InvalidContentType", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/workspace/register",
			strings.NewReader(`{"id": "test", "path": "/test"}`))
		req.Header.Set("Content-Type", "text/plain")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		// Should still work (we don't strictly validate content-type)
		// Just checking it doesn't crash
		_ = w.Code
	})

	// Test malformed JSON variations
	t.Run("MalformedJSONVariations", func(t *testing.T) {
		malformedBodies := []string{
			`{`,
			`{"id":}`,
			`{"id": "test", "path": }`,
			`[{"id": "test"}]`, // Array instead of object
			`null`,
			`"just a string"`,
			`12345`,
			`{"id": "test", "path": "/test", "extra": {"nested": "data"}}`, // Extra fields
		}

		for _, body := range malformedBodies {
			req := httptest.NewRequest("POST", "/workspace/register", strings.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)
			// Just verify it doesn't panic
		}
	})

	// Test query parameters edge cases
	t.Run("QueryParameterEdgeCases", func(t *testing.T) {
		// Multiple id parameters
		req := httptest.NewRequest("GET", "/workspace/state?id=one&id=two", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		// Should use first value

		// URL encoded values
		d.RegisterWorkspace("ws_encoded", "/path")
		req = httptest.NewRequest("GET", "/workspace/state?id=ws_encoded", nil)
		w = httptest.NewRecorder()
		router.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Errorf("Expected 200, got %d", w.Code)
		}
		d.UnregisterWorkspace("ws_encoded")

		// Very long query parameter
		longID := strings.Repeat("a", 10000)
		req = httptest.NewRequest("GET", "/workspace/state?id="+longID, nil)
		w = httptest.NewRecorder()
		router.ServeHTTP(w, req)
		// Should return 404 (not found) not crash
	})

	// Test concurrent requests to same endpoint
	t.Run("ConcurrentRequests", func(t *testing.T) {
		var wg sync.WaitGroup
		errors := make(chan error, 100)

		for i := 0; i < 100; i++ {
			wg.Add(1)
			go func(n int) {
				defer wg.Done()

				body := fmt.Sprintf(`{"id": "ws_concurrent_%d", "path": "/path/%d"}`, n, n)
				req := httptest.NewRequest("POST", "/workspace/register", strings.NewReader(body))
				req.Header.Set("Content-Type", "application/json")
				w := httptest.NewRecorder()

				router.ServeHTTP(w, req)

				if w.Code != http.StatusOK {
					errors <- fmt.Errorf("request %d failed with %d", n, w.Code)
				}
			}(i)
		}

		wg.Wait()
		close(errors)

		for err := range errors {
			t.Error(err)
		}
	})

	// Test unknown endpoint
	t.Run("UnknownEndpoint", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/unknown/endpoint", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusNotFound {
			t.Errorf("Expected 404 for unknown endpoint, got %d", w.Code)
		}
	})

	// Test HEAD request
	t.Run("HeadRequest", func(t *testing.T) {
		req := httptest.NewRequest("HEAD", "/health", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		// Should not panic
	})

	// Test OPTIONS request
	t.Run("OptionsRequest", func(t *testing.T) {
		req := httptest.NewRequest("OPTIONS", "/health", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		// Should not panic
	})
}

// TestHealthCheckEdgeCases tests edge cases for health checks
func TestHealthCheckEdgeCases(t *testing.T) {
	// Test health checker that always fails
	t.Run("AlwaysFailingChecker", func(t *testing.T) {
		hm := NewHealthManager()

		failingChecker := &alwaysFailingChecker{name: "failing"}
		hm.Register(failingChecker)

		hm.RunChecks()

		if hm.IsAllHealthy() {
			t.Error("Should not be all healthy with failing checker")
		}

		status := hm.GetStatus("failing")
		if status.Healthy {
			t.Error("Failing checker should report unhealthy")
		}

		if status.Message == "" {
			t.Error("Failing checker should have error message")
		}
	})

	// Test registering checker with same name (should replace)
	t.Run("DuplicateCheckerName", func(t *testing.T) {
		hm := NewHealthManager()

		checker1 := &testHealthChecker{name: "dup", healthy: true}
		checker2 := &testHealthChecker{name: "dup", healthy: false, err: fmt.Errorf("unhealthy")}

		hm.Register(checker1)
		hm.Register(checker2)

		hm.RunChecks()

		// Should use second checker (unhealthy)
		if hm.IsAllHealthy() {
			t.Error("Should use second registered checker")
		}
	})

	// Test running checks with no registered checkers
	t.Run("NoCheckers", func(t *testing.T) {
		hm := NewHealthManager()

		// Should not panic
		hm.RunChecks()

		if !hm.IsAllHealthy() {
			t.Error("No checkers should mean all healthy")
		}

		statuses := hm.GetAllStatuses()
		if len(statuses) != 0 {
			t.Errorf("Expected 0 statuses, got %d", len(statuses))
		}
	})

	// Test unregistering during iteration (via separate goroutine)
	t.Run("ConcurrentUnregister", func(t *testing.T) {
		hm := NewHealthManager()

		for i := 0; i < 10; i++ {
			hm.Register(&testHealthChecker{name: fmt.Sprintf("checker_%d", i), healthy: true})
		}

		var wg sync.WaitGroup
		wg.Add(2)

		// Goroutine 1: Run checks repeatedly
		go func() {
			defer wg.Done()
			for i := 0; i < 100; i++ {
				hm.RunChecks()
			}
		}()

		// Goroutine 2: Unregister checkers
		go func() {
			defer wg.Done()
			for i := 0; i < 10; i++ {
				hm.Unregister(fmt.Sprintf("checker_%d", i))
				time.Sleep(time.Millisecond)
			}
		}()

		wg.Wait()
	})

	// Test slow health checker
	t.Run("SlowHealthChecker", func(t *testing.T) {
		hm := NewHealthManager()

		slowChecker := &slowHealthChecker{
			name:     "slow",
			duration: 100 * time.Millisecond,
		}
		hm.Register(slowChecker)

		start := time.Now()
		hm.RunChecks()
		elapsed := time.Since(start)

		if elapsed < 100*time.Millisecond {
			t.Error("Should have waited for slow checker")
		}

		status := hm.GetStatus("slow")
		if status.Duration < 100 {
			t.Errorf("Expected duration >= 100ms, got %dms", status.Duration)
		}
	})
}

// alwaysFailingChecker is a health checker that always fails
type alwaysFailingChecker struct {
	name string
}

func (a *alwaysFailingChecker) Name() string  { return a.name }
func (a *alwaysFailingChecker) Check() error  { return fmt.Errorf("always fails") }
func (a *alwaysFailingChecker) IsHealthy() bool { return false }

// slowHealthChecker is a health checker that takes time
type slowHealthChecker struct {
	name     string
	duration time.Duration
}

func (s *slowHealthChecker) Name() string { return s.name }
func (s *slowHealthChecker) Check() error {
	time.Sleep(s.duration)
	return nil
}
func (s *slowHealthChecker) IsHealthy() bool { return true }

// TestPIDFileEdgeCases tests edge cases for PID file management
func TestPIDFileEdgeCases(t *testing.T) {
	// Test stale PID file (process no longer exists)
	t.Run("StalePIDFile", func(t *testing.T) {
		tmpDir, err := os.MkdirTemp("", "dba-pid-test-*")
		if err != nil {
			t.Fatalf("Failed to create temp dir: %v", err)
		}
		defer os.RemoveAll(tmpDir)

		cfg := &config.Config{
			Home: tmpDir,
			Daemon: config.DaemonConfig{
				Socket:   filepath.Join(tmpDir, "daemon.sock"),
				PIDFile:  filepath.Join(tmpDir, "daemon.pid"),
				LogFile:  filepath.Join(tmpDir, "daemon.log"),
				LogLevel: "info",
			},
		}

		// Write a stale PID (unlikely to exist)
		os.WriteFile(cfg.Daemon.PIDFile, []byte("999999999"), 0644)

		pid := GetDaemonPID(cfg)
		if pid != 0 {
			t.Error("Stale PID should return 0")
		}
	})

	// Test invalid PID file content
	t.Run("InvalidPIDFileContent", func(t *testing.T) {
		tmpDir, err := os.MkdirTemp("", "dba-pid-test-*")
		if err != nil {
			t.Fatalf("Failed to create temp dir: %v", err)
		}
		defer os.RemoveAll(tmpDir)

		cfg := &config.Config{
			Home: tmpDir,
			Daemon: config.DaemonConfig{
				Socket:   filepath.Join(tmpDir, "daemon.sock"),
				PIDFile:  filepath.Join(tmpDir, "daemon.pid"),
				LogFile:  filepath.Join(tmpDir, "daemon.log"),
				LogLevel: "info",
			},
		}

		// Write invalid content
		invalidContents := []string{
			"not a number",
			"",
			"123abc",
			"-1",
			"1.5",
		}

		for _, content := range invalidContents {
			os.WriteFile(cfg.Daemon.PIDFile, []byte(content), 0644)
			pid := GetDaemonPID(cfg)
			if pid != 0 {
				t.Errorf("Invalid PID content '%s' should return 0, got %d", content, pid)
			}
		}
	})

	// Test PID file in non-existent directory
	t.Run("PIDFileNonExistentDir", func(t *testing.T) {
		tmpDir, err := os.MkdirTemp("", "dba-pid-test-*")
		if err != nil {
			t.Fatalf("Failed to create temp dir: %v", err)
		}
		defer os.RemoveAll(tmpDir)

		cfg := &config.Config{
			Home: tmpDir,
			Daemon: config.DaemonConfig{
				Socket:   filepath.Join(tmpDir, "daemon.sock"),
				PIDFile:  filepath.Join(tmpDir, "nonexistent", "deep", "daemon.pid"),
				LogFile:  filepath.Join(tmpDir, "daemon.log"),
				LogLevel: "info",
			},
		}

		d, err := New(cfg)
		if err != nil {
			t.Fatalf("Failed to create daemon: %v", err)
		}

		// Should create directory and write PID file
		err = d.writePIDFile()
		if err != nil {
			t.Errorf("Should create directory for PID file: %v", err)
		}

		// Verify file was created
		if _, err := os.Stat(cfg.Daemon.PIDFile); os.IsNotExist(err) {
			t.Error("PID file should be created")
		}
	})
}

// TestSocketEdgeCases tests edge cases for socket handling
func TestSocketEdgeCases(t *testing.T) {
	// Test socket file already exists
	t.Run("SocketAlreadyExists", func(t *testing.T) {
		tmpDir, err := os.MkdirTemp("", "dba-socket-test-*")
		if err != nil {
			t.Fatalf("Failed to create temp dir: %v", err)
		}
		defer os.RemoveAll(tmpDir)

		socketPath := filepath.Join(tmpDir, "daemon.sock")

		// Create a regular file at socket path
		os.WriteFile(socketPath, []byte("dummy"), 0644)

		cfg := &config.Config{
			Home: tmpDir,
			Daemon: config.DaemonConfig{
				Socket:   socketPath,
				PIDFile:  filepath.Join(tmpDir, "daemon.pid"),
				LogFile:  filepath.Join(tmpDir, "daemon.log"),
				LogLevel: "info",
			},
		}

		d, err := New(cfg)
		if err != nil {
			t.Fatalf("Failed to create daemon: %v", err)
		}

		// Start should remove old file and create socket
		// We can't fully test Start() without blocking, but we can test the socket removal logic
		os.Remove(socketPath) // Simulate what Start() does

		listener, err := net.Listen("unix", socketPath)
		if err != nil {
			t.Fatalf("Should be able to listen after removing old file: %v", err)
		}
		listener.Close()

		_ = d // Use d to avoid unused variable error
	})

	// Test socket in directory without write permission
	t.Run("SocketNoWritePermission", func(t *testing.T) {
		if os.Getuid() == 0 {
			t.Skip("Skipping permission test when running as root")
		}

		tmpDir, err := os.MkdirTemp("", "dba-socket-test-*")
		if err != nil {
			t.Fatalf("Failed to create temp dir: %v", err)
		}
		defer os.RemoveAll(tmpDir)

		// Create read-only directory
		noWriteDir := filepath.Join(tmpDir, "readonly")
		os.Mkdir(noWriteDir, 0555)
		defer os.Chmod(noWriteDir, 0755) // Restore for cleanup

		socketPath := filepath.Join(noWriteDir, "daemon.sock")

		cfg := &config.Config{
			Home: tmpDir,
			Daemon: config.DaemonConfig{
				Socket:   socketPath,
				PIDFile:  filepath.Join(tmpDir, "daemon.pid"),
				LogFile:  filepath.Join(tmpDir, "daemon.log"),
				LogLevel: "info",
			},
		}

		d, err := New(cfg)
		if err != nil {
			t.Fatalf("Failed to create daemon: %v", err)
		}

		// Start should fail due to permission denied
		err = d.Start()
		if err == nil {
			t.Error("Should fail to create socket in read-only directory")
			d.Stop()
		}
	})
}

// TestClientEdgeCases tests edge cases for the daemon client
func TestClientEdgeCases(t *testing.T) {
	// Test client with daemon not running
	t.Run("ClientDaemonNotRunning", func(t *testing.T) {
		tmpDir, err := os.MkdirTemp("", "dba-client-test-*")
		if err != nil {
			t.Fatalf("Failed to create temp dir: %v", err)
		}
		defer os.RemoveAll(tmpDir)

		cfg := &config.Config{
			Home: tmpDir,
			Daemon: config.DaemonConfig{
				Socket:   filepath.Join(tmpDir, "daemon.sock"),
				PIDFile:  filepath.Join(tmpDir, "daemon.pid"),
				LogFile:  filepath.Join(tmpDir, "daemon.log"),
				LogLevel: "info",
			},
		}

		client := NewClient(cfg)

		// IsRunning should return false
		if client.IsRunning() {
			t.Error("Client should report daemon not running")
		}

		// Status should return error
		_, err = client.Status()
		if err == nil {
			t.Error("Status should fail when daemon not running")
		}

		// RegisterWorkspace should fail
		err = client.RegisterWorkspace("test", "/path")
		if err == nil {
			t.Error("RegisterWorkspace should fail when daemon not running")
		}

		// GetWorkspaceState should fail
		_, err = client.GetWorkspaceState("test")
		if err == nil {
			t.Error("GetWorkspaceState should fail when daemon not running")
		}
	})

	// Test client timeout handling
	t.Run("ClientTimeout", func(t *testing.T) {
		tmpDir, err := os.MkdirTemp("", "dba-client-test-*")
		if err != nil {
			t.Fatalf("Failed to create temp dir: %v", err)
		}
		defer os.RemoveAll(tmpDir)

		socketPath := filepath.Join(tmpDir, "daemon.sock")

		// Create a slow server that delays responses
		listener, err := net.Listen("unix", socketPath)
		if err != nil {
			t.Fatalf("Failed to create listener: %v", err)
		}
		defer listener.Close()

		slowHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			time.Sleep(5 * time.Second) // Very slow
			w.WriteHeader(http.StatusOK)
		})

		server := &http.Server{Handler: slowHandler}
		go server.Serve(listener)
		defer server.Close()

		time.Sleep(50 * time.Millisecond)

		cfg := &config.Config{
			Home: tmpDir,
			Daemon: config.DaemonConfig{
				Socket:   socketPath,
				PIDFile:  filepath.Join(tmpDir, "daemon.pid"),
				LogFile:  filepath.Join(tmpDir, "daemon.log"),
				LogLevel: "info",
			},
		}

		// Create client with short timeout
		client := &Client{
			socketPath: cfg.Daemon.Socket,
			httpClient: &http.Client{
				Transport: &http.Transport{
					DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
						var d net.Dialer
						return d.DialContext(ctx, "unix", cfg.Daemon.Socket)
					},
				},
				Timeout: 100 * time.Millisecond, // Very short timeout
			},
		}

		// This should timeout
		start := time.Now()
		_, err = client.Status()
		elapsed := time.Since(start)

		if err == nil {
			t.Error("Should timeout")
		}

		if elapsed > 1*time.Second {
			t.Errorf("Should have timed out quickly, took %v", elapsed)
		}
	})
}

// TestConcurrencyStress performs stress testing with concurrent operations
func TestConcurrencyStress(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping stress test in short mode")
	}

	cfg := setupTestConfig(t)
	d, err := New(cfg)
	if err != nil {
		t.Fatalf("Failed to create daemon: %v", err)
	}

	// Start an HTTP server
	listener, err := net.Listen("unix", cfg.Daemon.Socket)
	if err != nil {
		t.Fatalf("Failed to create listener: %v", err)
	}
	defer listener.Close()

	server := &http.Server{Handler: d.createRouter()}
	go server.Serve(listener)
	defer server.Close()

	time.Sleep(50 * time.Millisecond)

	client := NewClient(cfg)

	var wg sync.WaitGroup
	var registerCount, unregisterCount, statusCount, listCount int64
	var registerErrors, unregisterErrors, statusErrors, listErrors int64

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Spawn many goroutines doing different operations
	numWorkers := 20

	for i := 0; i < numWorkers; i++ {
		wg.Add(4)

		// Register workers
		go func(n int) {
			defer wg.Done()
			for {
				select {
				case <-ctx.Done():
					return
				default:
					err := client.RegisterWorkspace(
						fmt.Sprintf("ws_stress_%d_%d", n, time.Now().UnixNano()),
						fmt.Sprintf("/path/%d", n),
					)
					if err != nil {
						atomic.AddInt64(&registerErrors, 1)
					} else {
						atomic.AddInt64(&registerCount, 1)
					}
				}
			}
		}(i)

		// Unregister workers
		go func(n int) {
			defer wg.Done()
			for {
				select {
				case <-ctx.Done():
					return
				default:
					// Try to unregister (may fail if doesn't exist)
					client.UnregisterWorkspace(fmt.Sprintf("ws_stress_%d", n))
					atomic.AddInt64(&unregisterCount, 1)
				}
			}
		}(i)

		// Status workers
		go func() {
			defer wg.Done()
			for {
				select {
				case <-ctx.Done():
					return
				default:
					_, err := client.Status()
					if err != nil {
						atomic.AddInt64(&statusErrors, 1)
					} else {
						atomic.AddInt64(&statusCount, 1)
					}
				}
			}
		}()

		// List workers
		go func() {
			defer wg.Done()
			for {
				select {
				case <-ctx.Done():
					return
				default:
					_, err := client.ListWorkspaces()
					if err != nil {
						atomic.AddInt64(&listErrors, 1)
					} else {
						atomic.AddInt64(&listCount, 1)
					}
				}
			}
		}()
	}

	wg.Wait()

	t.Logf("Stress test results:")
	t.Logf("  Registers: %d (errors: %d)", registerCount, registerErrors)
	t.Logf("  Unregisters: %d (errors: %d)", unregisterCount, unregisterErrors)
	t.Logf("  Status: %d (errors: %d)", statusCount, statusErrors)
	t.Logf("  Lists: %d (errors: %d)", listCount, listErrors)

	// Some errors are expected due to timing, but should be minimal
	totalOps := registerCount + statusCount + listCount
	totalErrors := registerErrors + statusErrors + listErrors

	if totalOps == 0 {
		t.Error("No operations completed")
	}

	errorRate := float64(totalErrors) / float64(totalOps+totalErrors)
	if errorRate > 0.01 { // Allow up to 1% error rate
		t.Errorf("Error rate too high: %.2f%%", errorRate*100)
	}
}

// TestResponseFormats verifies all response formats are valid JSON
func TestResponseFormats(t *testing.T) {
	cfg := setupTestConfig(t)
	d, err := New(cfg)
	if err != nil {
		t.Fatalf("Failed to create daemon: %v", err)
	}

	// Register a workspace for testing
	d.RegisterWorkspace("ws_format_test", "/path/test")

	router := d.createRouter()

	endpoints := []struct {
		method string
		path   string
		body   string
	}{
		{"GET", "/health", ""},
		{"GET", "/status", ""},
		{"GET", "/workspace/list", ""},
		{"GET", "/workspace/state?id=ws_format_test", ""},
		{"GET", "/sync/wait?id=ws_format_test&timeout=1s", ""},
		{"POST", "/workspace/register", `{"id": "ws_new", "path": "/new"}`},
		{"POST", "/workspace/unregister", `{"id": "ws_new"}`},
		{"POST", "/workspace/activity", `{"id": "ws_format_test"}`},
	}

	for _, ep := range endpoints {
		t.Run(fmt.Sprintf("%s_%s", ep.method, ep.path), func(t *testing.T) {
			var body io.Reader
			if ep.body != "" {
				body = strings.NewReader(ep.body)
			}

			req := httptest.NewRequest(ep.method, ep.path, body)
			if ep.body != "" {
				req.Header.Set("Content-Type", "application/json")
			}
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			// Skip if not successful
			if w.Code >= 400 {
				return
			}

			// Verify response is valid JSON
			var result interface{}
			err := json.NewDecoder(w.Body).Decode(&result)
			if err != nil {
				t.Errorf("Response is not valid JSON: %v\nBody: %s", err, w.Body.String())
			}

			// Verify Content-Type header
			contentType := w.Header().Get("Content-Type")
			if contentType != "application/json" {
				t.Errorf("Expected Content-Type application/json, got %s", contentType)
			}
		})
	}
}

// TestGracefulShutdown tests graceful shutdown behavior
func TestGracefulShutdown(t *testing.T) {
	cfg := setupTestConfig(t)

	d, err := New(cfg)
	if err != nil {
		t.Fatalf("Failed to create daemon: %v", err)
	}

	// Register some workspaces
	for i := 0; i < 10; i++ {
		d.RegisterWorkspace(fmt.Sprintf("ws_%d", i), fmt.Sprintf("/path/%d", i))
	}

	// Start daemon in goroutine
	listener, err := net.Listen("unix", cfg.Daemon.Socket)
	if err != nil {
		t.Fatalf("Failed to create listener: %v", err)
	}

	d.listener = listener
	d.server = &http.Server{Handler: d.createRouter()}

	serverDone := make(chan struct{})
	go func() {
		d.server.Serve(listener)
		close(serverDone)
	}()

	time.Sleep(50 * time.Millisecond)

	// Start making requests
	client := NewClient(cfg)

	// Verify daemon is running
	if !client.IsRunning() {
		t.Fatal("Daemon should be running")
	}

	// Initiate shutdown
	shutdownDone := make(chan struct{})
	go func() {
		d.Stop()
		close(shutdownDone)
	}()

	// Wait for shutdown to complete
	select {
	case <-shutdownDone:
		// Good
	case <-time.After(15 * time.Second):
		t.Fatal("Shutdown took too long")
	}

	// Verify server stopped
	select {
	case <-serverDone:
		// Good
	case <-time.After(5 * time.Second):
		t.Fatal("Server didn't stop")
	}

	// Verify socket and PID files are cleaned up
	if _, err := os.Stat(cfg.Daemon.Socket); !os.IsNotExist(err) {
		t.Error("Socket file should be removed after shutdown")
	}

	if _, err := os.Stat(cfg.Daemon.PIDFile); !os.IsNotExist(err) {
		t.Error("PID file should be removed after shutdown")
	}
}

// TestWorkspaceStateConsistency tests that workspace state remains consistent
func TestWorkspaceStateConsistency(t *testing.T) {
	cfg := setupTestConfig(t)
	d, err := New(cfg)
	if err != nil {
		t.Fatalf("Failed to create daemon: %v", err)
	}

	// Perform many operations and verify consistency
	for i := 0; i < 100; i++ {
		wsID := fmt.Sprintf("ws_consistency_%d", i)
		path := fmt.Sprintf("/path/%d", i)

		// Register
		d.RegisterWorkspace(wsID, path)

		// Verify state
		state := d.GetWorkspaceState(wsID)
		if state == nil {
			t.Errorf("Workspace %s should exist after registration", wsID)
			continue
		}
		if state.ID != wsID {
			t.Errorf("Expected ID %s, got %s", wsID, state.ID)
		}
		if state.Path != path {
			t.Errorf("Expected path %s, got %s", path, state.Path)
		}

		// Verify in list
		workspaces := d.GetAllWorkspaces()
		found := false
		for _, ws := range workspaces {
			if ws.ID == wsID {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Workspace %s should be in list", wsID)
		}

		// Unregister
		d.UnregisterWorkspace(wsID)

		// Verify removed
		state = d.GetWorkspaceState(wsID)
		if state != nil {
			t.Errorf("Workspace %s should not exist after unregistration", wsID)
		}
	}

	// Final count should be 0
	if d.GetWorkspaceCount() != 0 {
		t.Errorf("Expected 0 workspaces, got %d", d.GetWorkspaceCount())
	}
}

// TestMemoryLeak tests for potential memory leaks with many operations
func TestMemoryLeak(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping memory leak test in short mode")
	}

	cfg := setupTestConfig(t)
	d, err := New(cfg)
	if err != nil {
		t.Fatalf("Failed to create daemon: %v", err)
	}

	// Perform many register/unregister cycles
	for cycle := 0; cycle < 10; cycle++ {
		// Register many workspaces
		for i := 0; i < 1000; i++ {
			d.RegisterWorkspace(fmt.Sprintf("ws_%d_%d", cycle, i), fmt.Sprintf("/path/%d/%d", cycle, i))
		}

		// Verify count
		if d.GetWorkspaceCount() != 1000 {
			t.Errorf("Cycle %d: Expected 1000 workspaces, got %d", cycle, d.GetWorkspaceCount())
		}

		// Unregister all
		for i := 0; i < 1000; i++ {
			d.UnregisterWorkspace(fmt.Sprintf("ws_%d_%d", cycle, i))
		}

		// Verify all removed
		if d.GetWorkspaceCount() != 0 {
			t.Errorf("Cycle %d: Expected 0 workspaces, got %d", cycle, d.GetWorkspaceCount())
		}
	}
}
