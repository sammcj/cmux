// Package daemon provides integration tests for the daemon HTTP server.
package daemon

import (
	"bytes"
	"context"
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/dba-cli/dba/internal/config"
)

// TestDaemonHTTPEndpoints tests the daemon HTTP API endpoints
func TestDaemonHTTPEndpoints(t *testing.T) {
	cfg := setupTestConfig(t)

	d, err := New(cfg)
	if err != nil {
		t.Fatalf("Failed to create daemon: %v", err)
	}

	router := d.createRouter()

	// Test /health endpoint
	t.Run("Health", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/health", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		var resp HealthResponse
		if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if resp.Status != "ok" {
			t.Errorf("Expected status 'ok', got '%s'", resp.Status)
		}
	})

	// Test /status endpoint
	t.Run("Status", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/status", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		var resp StatusResponse
		if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if !resp.Running {
			t.Error("Expected running to be true")
		}

		if resp.WorkspacesActive != 0 {
			t.Errorf("Expected 0 workspaces, got %d", resp.WorkspacesActive)
		}
	})

	// Test /workspace/register endpoint
	t.Run("WorkspaceRegister", func(t *testing.T) {
		body := `{"id": "ws_test_integration", "path": "/test/path"}`
		req := httptest.NewRequest("POST", "/workspace/register", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		var resp WorkspaceRegisterResponse
		if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if !resp.Registered {
			t.Error("Expected registered to be true")
		}

		// Verify workspace is registered
		if d.GetWorkspaceCount() != 1 {
			t.Errorf("Expected 1 workspace, got %d", d.GetWorkspaceCount())
		}
	})

	// Test /workspace/state endpoint
	t.Run("WorkspaceState", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/workspace/state?id=ws_test_integration", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		var resp WorkspaceState
		if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if resp.ID != "ws_test_integration" {
			t.Errorf("Expected ID 'ws_test_integration', got '%s'", resp.ID)
		}

		if resp.Path != "/test/path" {
			t.Errorf("Expected path '/test/path', got '%s'", resp.Path)
		}
	})

	// Test /workspace/state endpoint - not found
	t.Run("WorkspaceStateNotFound", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/workspace/state?id=ws_nonexistent", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != http.StatusNotFound {
			t.Errorf("Expected status 404, got %d", w.Code)
		}
	})

	// Test /workspace/list endpoint
	t.Run("WorkspaceList", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/workspace/list", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		var resp WorkspaceListResponse
		if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if len(resp.Workspaces) != 1 {
			t.Errorf("Expected 1 workspace, got %d", len(resp.Workspaces))
		}
	})

	// Test /workspace/activity endpoint
	t.Run("WorkspaceActivity", func(t *testing.T) {
		body := `{"id": "ws_test_integration"}`
		req := httptest.NewRequest("POST", "/workspace/activity", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		var resp WorkspaceActivityResponse
		if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if !resp.Updated {
			t.Error("Expected updated to be true")
		}
	})

	// Test /sync/wait endpoint (stub)
	t.Run("SyncWait", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/sync/wait?id=ws_test_integration&timeout=5s", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		var resp SyncWaitResponse
		if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if !resp.Synced {
			t.Error("Expected synced to be true")
		}

		if resp.WorkspaceID != "ws_test_integration" {
			t.Errorf("Expected workspace ID 'ws_test_integration', got '%s'", resp.WorkspaceID)
		}
	})

	// Test /workspace/unregister endpoint
	t.Run("WorkspaceUnregister", func(t *testing.T) {
		body := `{"id": "ws_test_integration"}`
		req := httptest.NewRequest("POST", "/workspace/unregister", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		var resp WorkspaceUnregisterResponse
		if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if !resp.Unregistered {
			t.Error("Expected unregistered to be true")
		}

		// Verify workspace is unregistered
		if d.GetWorkspaceCount() != 0 {
			t.Errorf("Expected 0 workspaces, got %d", d.GetWorkspaceCount())
		}
	})
}

// TestDaemonHTTPErrors tests error handling in HTTP endpoints
func TestDaemonHTTPErrors(t *testing.T) {
	cfg := setupTestConfig(t)

	d, err := New(cfg)
	if err != nil {
		t.Fatalf("Failed to create daemon: %v", err)
	}

	router := d.createRouter()

	// Test invalid method for register
	t.Run("RegisterInvalidMethod", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/workspace/register", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != http.StatusMethodNotAllowed {
			t.Errorf("Expected status 405, got %d", w.Code)
		}
	})

	// Test missing id for state
	t.Run("StateMissingID", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/workspace/state", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400, got %d", w.Code)
		}
	})

	// Test invalid JSON for register
	t.Run("RegisterInvalidJSON", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/workspace/register", bytes.NewBufferString("{invalid}"))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400, got %d", w.Code)
		}
	})

	// Test missing required fields for register
	t.Run("RegisterMissingFields", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/workspace/register", bytes.NewBufferString(`{"id": ""}`))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400, got %d", w.Code)
		}
	})

	// Test invalid timeout for sync/wait
	t.Run("SyncWaitInvalidTimeout", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/sync/wait?id=test&timeout=invalid", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400, got %d", w.Code)
		}
	})
}

// TestDaemonUnixSocket tests the daemon over a real Unix socket
func TestDaemonUnixSocket(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "dba-socket-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	socketPath := filepath.Join(tmpDir, "test.sock")

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

	// Create Unix socket listener
	listener, err := net.Listen("unix", socketPath)
	if err != nil {
		t.Fatalf("Failed to create listener: %v", err)
	}
	defer listener.Close()

	// Create HTTP server with daemon router
	server := &http.Server{
		Handler: d.createRouter(),
	}

	// Start server in background
	go func() {
		server.Serve(listener)
	}()
	defer server.Close()

	// Give server time to start
	time.Sleep(50 * time.Millisecond)

	// Create HTTP client that uses Unix socket
	client := &http.Client{
		Transport: &http.Transport{
			DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
				return net.Dial("unix", socketPath)
			},
		},
	}

	// Test health endpoint through Unix socket
	// Use localhost as the host since we're overriding the dial function
	resp, err := client.Get("http://localhost/health")
	if err != nil {
		t.Fatalf("Failed to get health: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	var healthResp HealthResponse
	if err := json.NewDecoder(resp.Body).Decode(&healthResp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if healthResp.Status != "ok" {
		t.Errorf("Expected status 'ok', got '%s'", healthResp.Status)
	}
}

// TestDaemonClient tests the client library functions
func TestDaemonClient(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "dba-client-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	socketPath := filepath.Join(tmpDir, "test.sock")

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

	// Create Unix socket listener
	listener, err := net.Listen("unix", socketPath)
	if err != nil {
		t.Fatalf("Failed to create listener: %v", err)
	}
	defer listener.Close()

	// Create HTTP server with daemon router
	server := &http.Server{
		Handler: d.createRouter(),
	}

	// Start server in background
	go func() {
		server.Serve(listener)
	}()
	defer server.Close()

	// Give server time to start
	time.Sleep(50 * time.Millisecond)

	// Create client
	client := NewClient(cfg)

	// Test IsRunning
	if !client.IsRunning() {
		t.Error("Expected daemon to be running")
	}

	// Test Status
	status, err := client.Status()
	if err != nil {
		t.Fatalf("Failed to get status: %v", err)
	}

	if !status.Running {
		t.Error("Expected running to be true")
	}

	// Test RegisterWorkspace
	err = client.RegisterWorkspace("ws_client_test", "/test/path")
	if err != nil {
		t.Fatalf("Failed to register workspace: %v", err)
	}

	// Verify registration
	status, err = client.Status()
	if err != nil {
		t.Fatalf("Failed to get status: %v", err)
	}

	if status.WorkspacesActive != 1 {
		t.Errorf("Expected 1 workspace, got %d", status.WorkspacesActive)
	}

	// Test GetWorkspaceState
	state, err := client.GetWorkspaceState("ws_client_test")
	if err != nil {
		t.Fatalf("Failed to get workspace state: %v", err)
	}

	if state.ID != "ws_client_test" {
		t.Errorf("Expected ID 'ws_client_test', got '%s'", state.ID)
	}

	// Test ListWorkspaces
	workspaces, err := client.ListWorkspaces()
	if err != nil {
		t.Fatalf("Failed to list workspaces: %v", err)
	}

	if len(workspaces) != 1 {
		t.Errorf("Expected 1 workspace, got %d", len(workspaces))
	}

	// Test WaitForSync
	result, err := client.WaitForSync("ws_client_test", 5*time.Second)
	if err != nil {
		t.Fatalf("Failed to wait for sync: %v", err)
	}

	if !result.Synced {
		t.Error("Expected synced to be true")
	}

	// Test UnregisterWorkspace
	err = client.UnregisterWorkspace("ws_client_test")
	if err != nil {
		t.Fatalf("Failed to unregister workspace: %v", err)
	}

	// Verify unregistration
	status, err = client.Status()
	if err != nil {
		t.Fatalf("Failed to get status: %v", err)
	}

	if status.WorkspacesActive != 0 {
		t.Errorf("Expected 0 workspaces, got %d", status.WorkspacesActive)
	}
}

// TestPIDFile tests PID file management
func TestPIDFile(t *testing.T) {
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

	d, err := New(cfg)
	if err != nil {
		t.Fatalf("Failed to create daemon: %v", err)
	}

	// Write PID file
	err = d.writePIDFile()
	if err != nil {
		t.Fatalf("Failed to write PID file: %v", err)
	}

	// Verify PID file exists
	if _, err := os.Stat(cfg.Daemon.PIDFile); os.IsNotExist(err) {
		t.Error("PID file should exist")
	}

	// Verify GetDaemonPID returns current PID
	pid := GetDaemonPID(cfg)
	if pid != os.Getpid() {
		t.Errorf("Expected PID %d, got %d", os.Getpid(), pid)
	}

	// Cleanup
	d.cleanup()

	// Verify PID file is removed
	if _, err := os.Stat(cfg.Daemon.PIDFile); !os.IsNotExist(err) {
		t.Error("PID file should be removed after cleanup")
	}
}
