// Package daemon provides tests for shutdown and lifecycle functionality.
package daemon

import (
	"errors"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/dba-cli/dba/internal/config"
)

// TestShutdownCallbacks tests the shutdown callback mechanism
func TestShutdownCallbacks(t *testing.T) {
	cfg := setupTestConfig(t)

	d, err := New(cfg)
	if err != nil {
		t.Fatalf("Failed to create daemon: %v", err)
	}

	// Test single callback
	t.Run("SingleCallback", func(t *testing.T) {
		var called bool
		d.RegisterShutdownCallback(func() error {
			called = true
			return nil
		})

		d.runShutdownCallbacks()

		if !called {
			t.Error("Shutdown callback was not called")
		}
	})

	// Test multiple callbacks
	t.Run("MultipleCallbacks", func(t *testing.T) {
		d2, _ := New(cfg)
		var callOrder []int
		var mu sync.Mutex

		for i := 1; i <= 3; i++ {
			num := i
			d2.RegisterShutdownCallback(func() error {
				mu.Lock()
				callOrder = append(callOrder, num)
				mu.Unlock()
				return nil
			})
		}

		d2.runShutdownCallbacks()

		if len(callOrder) != 3 {
			t.Errorf("Expected 3 callbacks called, got %d", len(callOrder))
		}

		// Verify order (should be 1, 2, 3)
		for i, v := range callOrder {
			if v != i+1 {
				t.Errorf("Expected callback %d at position %d, got %d", i+1, i, v)
			}
		}
	})

	// Test callback error handling
	t.Run("CallbackErrors", func(t *testing.T) {
		d3, _ := New(cfg)
		var callCount int32

		// First callback fails
		d3.RegisterShutdownCallback(func() error {
			atomic.AddInt32(&callCount, 1)
			return errors.New("callback 1 failed")
		})

		// Second callback should still be called
		d3.RegisterShutdownCallback(func() error {
			atomic.AddInt32(&callCount, 1)
			return nil
		})

		// Third callback fails
		d3.RegisterShutdownCallback(func() error {
			atomic.AddInt32(&callCount, 1)
			return errors.New("callback 3 failed")
		})

		d3.runShutdownCallbacks()

		// All callbacks should be called despite errors
		if atomic.LoadInt32(&callCount) != 3 {
			t.Errorf("Expected all 3 callbacks called, got %d", callCount)
		}
	})

	// Test concurrent callback registration
	t.Run("ConcurrentRegistration", func(t *testing.T) {
		d4, _ := New(cfg)
		var wg sync.WaitGroup
		var callCount int32

		// Register callbacks concurrently
		for i := 0; i < 10; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				d4.RegisterShutdownCallback(func() error {
					atomic.AddInt32(&callCount, 1)
					return nil
				})
			}()
		}

		wg.Wait()

		d4.runShutdownCallbacks()

		if atomic.LoadInt32(&callCount) != 10 {
			t.Errorf("Expected 10 callbacks called, got %d", callCount)
		}
	})
}

// TestLoggingSetup tests the logging setup functionality
func TestLoggingSetup(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "dba-logging-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	logFile := filepath.Join(tmpDir, "subdir", "daemon.log")

	cfg := &config.Config{
		Home: tmpDir,
		Daemon: config.DaemonConfig{
			Socket:   filepath.Join(tmpDir, "daemon.sock"),
			PIDFile:  filepath.Join(tmpDir, "daemon.pid"),
			LogFile:  logFile,
			LogLevel: "info",
		},
	}

	d, err := New(cfg)
	if err != nil {
		t.Fatalf("Failed to create daemon: %v", err)
	}

	// Test logging setup creates directory and file
	t.Run("CreatesLogDirectory", func(t *testing.T) {
		err := d.setupLogging()
		if err != nil {
			t.Fatalf("Failed to setup logging: %v", err)
		}

		// Verify directory was created
		if _, err := os.Stat(filepath.Dir(logFile)); os.IsNotExist(err) {
			t.Error("Log directory was not created")
		}
	})

	// Test logger writes to file
	t.Run("WritesToLogFile", func(t *testing.T) {
		d.logger.Println("test message")

		// Give time for write
		time.Sleep(10 * time.Millisecond)

		content, err := os.ReadFile(logFile)
		if err != nil {
			t.Fatalf("Failed to read log file: %v", err)
		}

		if len(content) == 0 {
			t.Error("Log file is empty")
		}
	})
}

// TestOrphanedProcessCleanup tests cleanup of orphaned processes
func TestOrphanedProcessCleanup(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "dba-orphan-test-*")
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

	// Test cleanup with non-existent PID file
	t.Run("NoPIDFile", func(t *testing.T) {
		d, _ := New(cfg)
		err := d.cleanupOrphanedProcesses()
		if err != nil {
			t.Errorf("Should not error with no PID file: %v", err)
		}
	})

	// Test cleanup with invalid PID file content
	t.Run("InvalidPIDContent", func(t *testing.T) {
		os.WriteFile(cfg.Daemon.PIDFile, []byte("not a number"), 0644)

		d, _ := New(cfg)
		err := d.cleanupOrphanedProcesses()
		if err != nil {
			t.Errorf("Should handle invalid PID content gracefully: %v", err)
		}

		// PID file should be removed
		if _, err := os.Stat(cfg.Daemon.PIDFile); !os.IsNotExist(err) {
			t.Error("Invalid PID file should be removed")
		}
	})

	// Test cleanup with stale PID (non-existent process)
	t.Run("StalePID", func(t *testing.T) {
		// Use a very high PID that likely doesn't exist
		os.WriteFile(cfg.Daemon.PIDFile, []byte("999999999"), 0644)

		d, _ := New(cfg)
		err := d.cleanupOrphanedProcesses()
		if err != nil {
			t.Errorf("Should handle stale PID gracefully: %v", err)
		}

		// PID file should be removed
		if _, err := os.Stat(cfg.Daemon.PIDFile); !os.IsNotExist(err) {
			t.Error("Stale PID file should be removed")
		}
	})

	// Test cleanup removes stale socket file
	t.Run("StaleSocket", func(t *testing.T) {
		// Create a regular file (not a socket) to simulate stale socket
		os.WriteFile(cfg.Daemon.Socket, []byte("dummy"), 0644)

		d, _ := New(cfg)
		err := d.cleanupOrphanedProcesses()
		if err != nil {
			t.Errorf("Should handle stale socket gracefully: %v", err)
		}

		// Socket file should be removed
		if _, err := os.Stat(cfg.Daemon.Socket); !os.IsNotExist(err) {
			t.Error("Stale socket file should be removed")
		}
	})
}

// TestStaleSocketCleanup tests the checkAndCleanStaleSocket functionality
func TestStaleSocketCleanup(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "dba-socket-test-*")
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

	// Test with no socket file
	t.Run("NoSocketFile", func(t *testing.T) {
		d, _ := New(cfg)
		err := d.checkAndCleanStaleSocket()
		if err != nil {
			t.Errorf("Should not error with no socket file: %v", err)
		}
	})

	// Test with stale socket file (regular file, not a socket)
	t.Run("StaleSocketFile", func(t *testing.T) {
		os.WriteFile(cfg.Daemon.Socket, []byte("dummy"), 0644)

		d, _ := New(cfg)
		err := d.checkAndCleanStaleSocket()
		if err != nil {
			t.Errorf("Should clean stale socket: %v", err)
		}

		if _, err := os.Stat(cfg.Daemon.Socket); !os.IsNotExist(err) {
			t.Error("Stale socket should be removed")
		}
	})
}

// TestWorkspaceStatusUpdates tests workspace status management
func TestWorkspaceStatusUpdates(t *testing.T) {
	cfg := setupTestConfig(t)

	d, err := New(cfg)
	if err != nil {
		t.Fatalf("Failed to create daemon: %v", err)
	}

	// Register workspace
	d.RegisterWorkspace("ws_status_test", "/path/test")

	// Verify initial status
	state := d.GetWorkspaceState("ws_status_test")
	if state == nil {
		t.Fatal("Workspace should exist")
	}
	if state.Status != "running" {
		t.Errorf("Expected initial status 'running', got '%s'", state.Status)
	}

	// Update status
	d.UpdateWorkspaceStatus("ws_status_test", "stopped")

	state = d.GetWorkspaceState("ws_status_test")
	if state.Status != "stopped" {
		t.Errorf("Expected status 'stopped', got '%s'", state.Status)
	}

	// Update status for non-existent workspace (should not panic)
	d.UpdateWorkspaceStatus("ws_nonexistent", "error")
}

// TestHealthManagerIntegration tests health manager integration with daemon
func TestHealthManagerIntegration(t *testing.T) {
	cfg := setupTestConfig(t)

	d, err := New(cfg)
	if err != nil {
		t.Fatalf("Failed to create daemon: %v", err)
	}

	// Verify health manager is created
	if d.GetHealthManager() == nil {
		t.Fatal("Health manager should be initialized")
	}

	// Register a health checker
	checker := &testHealthChecker{name: "test-integration", healthy: true}
	d.GetHealthManager().Register(checker)

	// Run health checks
	d.performHealthChecks()

	// Verify status
	status := d.GetHealthManager().GetStatus("test-integration")
	if status == nil {
		t.Fatal("Health status should exist")
	}
	if !status.Healthy {
		t.Error("Health check should be healthy")
	}
}

// TestDaemonContext tests the daemon context functionality
func TestDaemonContext(t *testing.T) {
	cfg := setupTestConfig(t)

	d, err := New(cfg)
	if err != nil {
		t.Fatalf("Failed to create daemon: %v", err)
	}

	// Verify context is not nil
	ctx := d.Context()
	if ctx == nil {
		t.Fatal("Context should not be nil")
	}

	// Context should not be cancelled initially
	select {
	case <-ctx.Done():
		t.Error("Context should not be cancelled initially")
	default:
		// Good
	}

	// Cancel and verify
	d.cancel()

	select {
	case <-ctx.Done():
		// Good
	case <-time.After(100 * time.Millisecond):
		t.Error("Context should be cancelled after cancel()")
	}
}

// TestDaemonLogger tests the daemon logger functionality
func TestDaemonLogger(t *testing.T) {
	cfg := setupTestConfig(t)

	d, err := New(cfg)
	if err != nil {
		t.Fatalf("Failed to create daemon: %v", err)
	}

	// Verify logger is not nil
	logger := d.Logger()
	if logger == nil {
		t.Fatal("Logger should not be nil")
	}

	// Logger should work without panicking
	logger.Println("test message")
}

// TestWorkspaceStateReturn tests that GetWorkspaceState returns a copy
func TestWorkspaceStateReturn(t *testing.T) {
	cfg := setupTestConfig(t)

	d, err := New(cfg)
	if err != nil {
		t.Fatalf("Failed to create daemon: %v", err)
	}

	d.RegisterWorkspace("ws_copy_test", "/path/test")

	// Get state
	state1 := d.GetWorkspaceState("ws_copy_test")
	state2 := d.GetWorkspaceState("ws_copy_test")

	// Modify state1
	state1.Status = "modified"

	// state2 should not be affected
	if state2.Status == "modified" {
		t.Error("GetWorkspaceState should return a copy, not the original")
	}

	// Internal state should not be affected
	internalState := d.GetWorkspaceState("ws_copy_test")
	if internalState.Status == "modified" {
		t.Error("Internal state should not be affected by modifying returned copy")
	}
}

// TestGetAllWorkspacesReturn tests that GetAllWorkspaces returns copies
func TestGetAllWorkspacesReturn(t *testing.T) {
	cfg := setupTestConfig(t)

	d, err := New(cfg)
	if err != nil {
		t.Fatalf("Failed to create daemon: %v", err)
	}

	d.RegisterWorkspace("ws_all_1", "/path/1")
	d.RegisterWorkspace("ws_all_2", "/path/2")

	// Get all workspaces
	workspaces := d.GetAllWorkspaces()

	// Modify one
	for _, ws := range workspaces {
		ws.Status = "modified"
	}

	// Internal state should not be affected
	internalState := d.GetWorkspaceState("ws_all_1")
	if internalState.Status == "modified" {
		t.Error("GetAllWorkspaces should return copies")
	}
}
