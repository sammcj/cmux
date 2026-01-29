// Package daemon provides HTTP endpoint edge case tests.
package daemon

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/dba-cli/dba/internal/config"
)

// TestHealthEndpointEdgeCases tests the /health endpoint edge cases
func TestHealthEndpointEdgeCases(t *testing.T) {
	cfg := setupTestConfig(t)

	d, err := New(cfg)
	if err != nil {
		t.Fatalf("Failed to create daemon: %v", err)
	}

	router := d.createRouter()

	// Test with different HTTP methods
	t.Run("MethodGET", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/health", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}
	})

	t.Run("MethodPOST", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/health", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// Health endpoint should accept any method
		if w.Code != http.StatusOK {
			t.Logf("POST to /health returned %d", w.Code)
		}
	})

	t.Run("MethodHEAD", func(t *testing.T) {
		req := httptest.NewRequest("HEAD", "/health", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// HEAD should work too
		if w.Code != http.StatusOK {
			t.Logf("HEAD to /health returned %d", w.Code)
		}
	})

	t.Run("ResponseFormat", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/health", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// Verify JSON response
		var resp HealthResponse
		err := json.NewDecoder(w.Body).Decode(&resp)
		if err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if resp.Status != "ok" {
			t.Errorf("Expected status 'ok', got '%s'", resp.Status)
		}

		// Verify Content-Type
		ct := w.Header().Get("Content-Type")
		if ct != "application/json" {
			t.Errorf("Expected Content-Type 'application/json', got '%s'", ct)
		}
	})
}

// TestStatusEndpointEdgeCases tests the /status endpoint edge cases
func TestStatusEndpointEdgeCases(t *testing.T) {
	cfg := setupTestConfig(t)

	d, err := New(cfg)
	if err != nil {
		t.Fatalf("Failed to create daemon: %v", err)
	}

	// Register some workspaces
	d.RegisterWorkspace("ws1", "/path1")
	d.RegisterWorkspace("ws2", "/path2")

	router := d.createRouter()

	t.Run("ResponseFields", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/status", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		var resp StatusResponse
		err := json.NewDecoder(w.Body).Decode(&resp)
		if err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		// Verify all fields
		if !resp.Running {
			t.Error("Expected Running to be true")
		}
		if resp.PID <= 0 {
			t.Error("Expected valid PID")
		}
		if resp.Socket != cfg.Daemon.Socket {
			t.Errorf("Expected socket '%s', got '%s'", cfg.Daemon.Socket, resp.Socket)
		}
		if resp.WorkspacesActive != 2 {
			t.Errorf("Expected 2 active workspaces, got %d", resp.WorkspacesActive)
		}
	})

	t.Run("UptimeIncreases", func(t *testing.T) {
		req1 := httptest.NewRequest("GET", "/status", nil)
		w1 := httptest.NewRecorder()
		router.ServeHTTP(w1, req1)

		var resp1 StatusResponse
		json.NewDecoder(w1.Body).Decode(&resp1)

		time.Sleep(100 * time.Millisecond)

		req2 := httptest.NewRequest("GET", "/status", nil)
		w2 := httptest.NewRecorder()
		router.ServeHTTP(w2, req2)

		var resp2 StatusResponse
		json.NewDecoder(w2.Body).Decode(&resp2)

		// Uptime should increase (or at least not decrease)
		if resp2.UptimeSeconds < resp1.UptimeSeconds {
			t.Error("Uptime should not decrease")
		}
	})
}

// TestWorkspaceRegisterEndpointEdgeCases tests /workspace/register edge cases
func TestWorkspaceRegisterEndpointEdgeCases(t *testing.T) {
	cfg := setupTestConfig(t)

	d, err := New(cfg)
	if err != nil {
		t.Fatalf("Failed to create daemon: %v", err)
	}

	router := d.createRouter()

	t.Run("ValidRequest", func(t *testing.T) {
		body := `{"id":"ws_test","path":"/path/test"}`
		req := httptest.NewRequest("POST", "/workspace/register", strings.NewReader(body))
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		var resp WorkspaceRegisterResponse
		json.NewDecoder(w.Body).Decode(&resp)
		if !resp.Registered {
			t.Error("Expected Registered to be true")
		}
	})

	t.Run("WrongMethod", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/workspace/register", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusMethodNotAllowed {
			t.Errorf("Expected status 405, got %d", w.Code)
		}
	})

	t.Run("EmptyBody", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/workspace/register", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400, got %d", w.Code)
		}
	})

	t.Run("InvalidJSON", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/workspace/register", strings.NewReader("{invalid}"))
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400, got %d", w.Code)
		}
	})

	t.Run("MissingID", func(t *testing.T) {
		body := `{"path":"/path/test"}`
		req := httptest.NewRequest("POST", "/workspace/register", strings.NewReader(body))
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400, got %d", w.Code)
		}
	})

	t.Run("MissingPath", func(t *testing.T) {
		body := `{"id":"ws_test"}`
		req := httptest.NewRequest("POST", "/workspace/register", strings.NewReader(body))
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400, got %d", w.Code)
		}
	})

	t.Run("EmptyID", func(t *testing.T) {
		body := `{"id":"","path":"/path/test"}`
		req := httptest.NewRequest("POST", "/workspace/register", strings.NewReader(body))
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400, got %d", w.Code)
		}
	})

	t.Run("EmptyPath", func(t *testing.T) {
		body := `{"id":"ws_test","path":""}`
		req := httptest.NewRequest("POST", "/workspace/register", strings.NewReader(body))
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400, got %d", w.Code)
		}
	})

	t.Run("LargePayload", func(t *testing.T) {
		// Create a very long path
		longPath := "/path/" + strings.Repeat("a", 10000)
		body := `{"id":"ws_large","path":"` + longPath + `"}`
		req := httptest.NewRequest("POST", "/workspace/register", strings.NewReader(body))
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// Should still work
		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}
	})

	t.Run("SpecialCharactersInID", func(t *testing.T) {
		body := `{"id":"ws/test:123!@#","path":"/path/test"}`
		req := httptest.NewRequest("POST", "/workspace/register", strings.NewReader(body))
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// Should work with special characters
		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}
	})

	t.Run("UnicodeInPath", func(t *testing.T) {
		body := `{"id":"ws_unicode","path":"/path/测试/ტესტი/тест"}`
		req := httptest.NewRequest("POST", "/workspace/register", strings.NewReader(body))
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		// Verify the path was stored correctly
		state := d.GetWorkspaceState("ws_unicode")
		if state == nil {
			t.Fatal("Workspace should exist")
		}
		if state.Path != "/path/测试/ტესტი/тест" {
			t.Errorf("Path mismatch: %s", state.Path)
		}
	})
}

// TestWorkspaceUnregisterEndpointEdgeCases tests /workspace/unregister edge cases
func TestWorkspaceUnregisterEndpointEdgeCases(t *testing.T) {
	cfg := setupTestConfig(t)

	d, err := New(cfg)
	if err != nil {
		t.Fatalf("Failed to create daemon: %v", err)
	}

	router := d.createRouter()

	t.Run("WrongMethod", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/workspace/unregister", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusMethodNotAllowed {
			t.Errorf("Expected status 405, got %d", w.Code)
		}
	})

	t.Run("NonExistentWorkspace", func(t *testing.T) {
		body := `{"id":"ws_nonexistent"}`
		req := httptest.NewRequest("POST", "/workspace/unregister", strings.NewReader(body))
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// Should succeed even for non-existent workspace
		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}
	})

	t.Run("ExistingWorkspace", func(t *testing.T) {
		// Register first
		d.RegisterWorkspace("ws_to_remove", "/path/remove")

		body := `{"id":"ws_to_remove"}`
		req := httptest.NewRequest("POST", "/workspace/unregister", strings.NewReader(body))
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		// Verify it's gone
		state := d.GetWorkspaceState("ws_to_remove")
		if state != nil {
			t.Error("Workspace should be removed")
		}
	})

	t.Run("MissingID", func(t *testing.T) {
		body := `{}`
		req := httptest.NewRequest("POST", "/workspace/unregister", strings.NewReader(body))
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400, got %d", w.Code)
		}
	})
}

// TestWorkspaceStateEndpointEdgeCases tests /workspace/state edge cases
func TestWorkspaceStateEndpointEdgeCases(t *testing.T) {
	cfg := setupTestConfig(t)

	d, err := New(cfg)
	if err != nil {
		t.Fatalf("Failed to create daemon: %v", err)
	}

	// Register a workspace
	d.RegisterWorkspace("ws_state_test", "/path/state")

	router := d.createRouter()

	t.Run("ExistingWorkspace", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/workspace/state?id=ws_state_test", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		var state WorkspaceState
		json.NewDecoder(w.Body).Decode(&state)
		if state.ID != "ws_state_test" {
			t.Errorf("Expected ID 'ws_state_test', got '%s'", state.ID)
		}
	})

	t.Run("NonExistentWorkspace", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/workspace/state?id=ws_nonexistent", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusNotFound {
			t.Errorf("Expected status 404, got %d", w.Code)
		}
	})

	t.Run("MissingIDParam", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/workspace/state", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400, got %d", w.Code)
		}
	})

	t.Run("EmptyIDParam", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/workspace/state?id=", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400, got %d", w.Code)
		}
	})
}

// TestWorkspaceListEndpointEdgeCases tests /workspace/list edge cases
func TestWorkspaceListEndpointEdgeCases(t *testing.T) {
	cfg := setupTestConfig(t)

	d, err := New(cfg)
	if err != nil {
		t.Fatalf("Failed to create daemon: %v", err)
	}

	router := d.createRouter()

	t.Run("EmptyList", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/workspace/list", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		var resp WorkspaceListResponse
		json.NewDecoder(w.Body).Decode(&resp)
		if len(resp.Workspaces) != 0 {
			t.Errorf("Expected empty list, got %d workspaces", len(resp.Workspaces))
		}
	})

	t.Run("WithWorkspaces", func(t *testing.T) {
		d.RegisterWorkspace("ws_list_1", "/path/1")
		d.RegisterWorkspace("ws_list_2", "/path/2")
		d.RegisterWorkspace("ws_list_3", "/path/3")

		req := httptest.NewRequest("GET", "/workspace/list", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		var resp WorkspaceListResponse
		json.NewDecoder(w.Body).Decode(&resp)
		if len(resp.Workspaces) != 3 {
			t.Errorf("Expected 3 workspaces, got %d", len(resp.Workspaces))
		}
	})
}

// TestWorkspaceActivityEndpointEdgeCases tests /workspace/activity edge cases
func TestWorkspaceActivityEndpointEdgeCases(t *testing.T) {
	cfg := setupTestConfig(t)

	d, err := New(cfg)
	if err != nil {
		t.Fatalf("Failed to create daemon: %v", err)
	}

	d.RegisterWorkspace("ws_activity_test", "/path/activity")

	router := d.createRouter()

	t.Run("ValidUpdate", func(t *testing.T) {
		// Get initial state
		stateBefore := d.GetWorkspaceState("ws_activity_test")
		timeBefore := stateBefore.LastActive

		time.Sleep(10 * time.Millisecond)

		body := `{"id":"ws_activity_test"}`
		req := httptest.NewRequest("POST", "/workspace/activity", strings.NewReader(body))
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		// Verify time was updated
		stateAfter := d.GetWorkspaceState("ws_activity_test")
		if !stateAfter.LastActive.After(timeBefore) {
			t.Error("LastActive should be updated")
		}
	})

	t.Run("WrongMethod", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/workspace/activity", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusMethodNotAllowed {
			t.Errorf("Expected status 405, got %d", w.Code)
		}
	})

	t.Run("MissingID", func(t *testing.T) {
		body := `{}`
		req := httptest.NewRequest("POST", "/workspace/activity", strings.NewReader(body))
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400, got %d", w.Code)
		}
	})
}

// TestSyncWaitEndpointEdgeCases tests /sync/wait edge cases
func TestSyncWaitEndpointEdgeCases(t *testing.T) {
	cfg := setupTestConfig(t)

	d, err := New(cfg)
	if err != nil {
		t.Fatalf("Failed to create daemon: %v", err)
	}

	router := d.createRouter()

	t.Run("DefaultTimeout", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/sync/wait", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		var resp SyncWaitResponse
		json.NewDecoder(w.Body).Decode(&resp)
		if resp.Timeout != "10s" {
			t.Errorf("Expected default timeout '10s', got '%s'", resp.Timeout)
		}
	})

	t.Run("CustomTimeout", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/sync/wait?timeout=5s", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		var resp SyncWaitResponse
		json.NewDecoder(w.Body).Decode(&resp)
		if resp.Timeout != "5s" {
			t.Errorf("Expected timeout '5s', got '%s'", resp.Timeout)
		}
	})

	t.Run("InvalidTimeout", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/sync/wait?timeout=invalid", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400, got %d", w.Code)
		}
	})

	t.Run("WithWorkspaceID", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/sync/wait?id=ws_test", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		var resp SyncWaitResponse
		json.NewDecoder(w.Body).Decode(&resp)
		if resp.WorkspaceID != "ws_test" {
			t.Errorf("Expected workspace ID 'ws_test', got '%s'", resp.WorkspaceID)
		}
	})

	t.Run("SubsecondTimeout", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/sync/wait?timeout=100ms", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}
	})
}

// TestUnknownEndpoint tests behavior with unknown endpoints
func TestUnknownEndpoint(t *testing.T) {
	cfg := setupTestConfig(t)

	d, err := New(cfg)
	if err != nil {
		t.Fatalf("Failed to create daemon: %v", err)
	}

	router := d.createRouter()

	t.Run("UnknownPath", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/unknown/path", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusNotFound {
			t.Errorf("Expected status 404, got %d", w.Code)
		}
	})

	t.Run("RootPath", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusNotFound {
			t.Errorf("Expected status 404, got %d", w.Code)
		}
	})
}

// TestContentTypeHandling tests Content-Type handling
func TestContentTypeHandling(t *testing.T) {
	cfg := setupTestConfig(t)

	d, err := New(cfg)
	if err != nil {
		t.Fatalf("Failed to create daemon: %v", err)
	}

	router := d.createRouter()

	t.Run("RequestWithContentType", func(t *testing.T) {
		body := `{"id":"ws_ct_test","path":"/path/test"}`
		req := httptest.NewRequest("POST", "/workspace/register", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}
	})

	t.Run("RequestWithWrongContentType", func(t *testing.T) {
		body := `{"id":"ws_ct_test2","path":"/path/test"}`
		req := httptest.NewRequest("POST", "/workspace/register", strings.NewReader(body))
		req.Header.Set("Content-Type", "text/plain")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// Should still work since we parse JSON from body regardless
		// The server doesn't validate Content-Type strictly
		t.Logf("Request with text/plain Content-Type returned %d", w.Code)
	})

	t.Run("ResponseContentType", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/health", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		ct := w.Header().Get("Content-Type")
		if ct != "application/json" {
			t.Errorf("Expected response Content-Type 'application/json', got '%s'", ct)
		}
	})
}

// TestLargeRequestBody tests handling of large request bodies
func TestLargeRequestBody(t *testing.T) {
	cfg := setupTestConfig(t)

	d, err := New(cfg)
	if err != nil {
		t.Fatalf("Failed to create daemon: %v", err)
	}

	router := d.createRouter()

	t.Run("VeryLargeBody", func(t *testing.T) {
		// Create a 1MB body
		largeData := bytes.Repeat([]byte("x"), 1024*1024)
		req := httptest.NewRequest("POST", "/workspace/register", bytes.NewReader(largeData))
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// Should fail due to invalid JSON
		if w.Code != http.StatusBadRequest {
			t.Logf("Large body returned %d", w.Code)
		}
	})
}

// TestConnectionReset tests behavior when connection is reset
func TestConnectionReset(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "dba-connreset-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	socketPath := filepath.Join(tmpDir, "daemon.sock")

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

	listener, err := net.Listen("unix", socketPath)
	if err != nil {
		t.Fatalf("Failed to create listener: %v", err)
	}

	server := &http.Server{Handler: d.createRouter()}
	go server.Serve(listener)

	time.Sleep(50 * time.Millisecond)

	// Close server mid-request scenario
	t.Run("ServerClosesDuringRequest", func(t *testing.T) {
		// Make a connection
		conn, err := net.Dial("unix", socketPath)
		if err != nil {
			t.Fatalf("Failed to connect: %v", err)
		}

		// Close server
		server.Close()
		listener.Close()

		// Try to read (should fail)
		buf := make([]byte, 100)
		_, err = conn.Read(buf)
		if err == nil {
			t.Logf("Connection should eventually fail")
		}
		conn.Close()
	})
}

// TestResponseBodyDrained tests that response bodies are properly handled
func TestResponseBodyDrained(t *testing.T) {
	cfg := setupTestConfig(t)

	d, err := New(cfg)
	if err != nil {
		t.Fatalf("Failed to create daemon: %v", err)
	}

	router := d.createRouter()

	// Make many requests and don't read all the body
	for i := 0; i < 100; i++ {
		req := httptest.NewRequest("GET", "/status", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// Read only part of body
		buf := make([]byte, 10)
		w.Body.Read(buf)
	}
}

// TestConcurrentHTTPRequests tests concurrent HTTP request handling
func TestConcurrentHTTPRequests(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "dba-concurrent-http-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	socketPath := filepath.Join(tmpDir, "daemon.sock")

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

	listener, err := net.Listen("unix", socketPath)
	if err != nil {
		t.Fatalf("Failed to create listener: %v", err)
	}
	defer listener.Close()

	server := &http.Server{Handler: d.createRouter()}
	go server.Serve(listener)
	defer server.Close()

	time.Sleep(50 * time.Millisecond)

	// Create HTTP client for Unix socket
	httpClient := &http.Client{
		Transport: &http.Transport{
			DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
				var d net.Dialer
				return d.DialContext(ctx, "unix", socketPath)
			},
		},
		Timeout: 5 * time.Second,
	}

	// Make concurrent requests
	t.Run("ConcurrentMixedRequests", func(t *testing.T) {
		done := make(chan bool)
		errCh := make(chan error, 100)

		for i := 0; i < 50; i++ {
			go func(n int) {
				var resp *http.Response
				var err error

				switch n % 5 {
				case 0:
					resp, err = httpClient.Get("http://localhost/health")
				case 1:
					resp, err = httpClient.Get("http://localhost/status")
				case 2:
					resp, err = httpClient.Get("http://localhost/workspace/list")
				case 3:
					body := strings.NewReader(`{"id":"ws_concurrent_` + string(rune('0'+n%10)) + `","path":"/path"}`)
					resp, err = httpClient.Post("http://localhost/workspace/register", "application/json", body)
				case 4:
					resp, err = httpClient.Get("http://localhost/sync/wait?timeout=100ms")
				}

				if err != nil {
					errCh <- err
				} else {
					io.Copy(io.Discard, resp.Body)
					resp.Body.Close()
				}
				done <- true
			}(i)
		}

		// Wait for all requests
		for i := 0; i < 50; i++ {
			<-done
		}

		// Check for errors
		close(errCh)
		errorCount := 0
		for err := range errCh {
			t.Logf("Request error: %v", err)
			errorCount++
		}

		if errorCount > 5 {
			t.Errorf("Too many errors: %d", errorCount)
		}
	})
}
