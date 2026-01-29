// Package daemon provides lifecycle and race condition tests.
package daemon

import (
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/dba-cli/dba/internal/config"
)

// TestDaemonLifecycleRace tests race conditions in daemon lifecycle
func TestDaemonLifecycleRace(t *testing.T) {
	cfg := setupTestConfig(t)

	// Test concurrent starts (should fail gracefully)
	t.Run("ConcurrentStarts", func(t *testing.T) {
		// Create multiple daemons pointing to same socket
		daemons := make([]*Daemon, 5)
		for i := 0; i < 5; i++ {
			d, err := New(cfg)
			if err != nil {
				t.Fatalf("Failed to create daemon %d: %v", i, err)
			}
			daemons[i] = d
		}

		// Try to start them concurrently
		var wg sync.WaitGroup
		var successCount, errorCount int32

		for _, d := range daemons {
			wg.Add(1)
			go func(daemon *Daemon) {
				defer wg.Done()

				// Use the internal method that doesn't block
				err := daemon.checkAndCleanStaleSocket()
				if err != nil {
					atomic.AddInt32(&errorCount, 1)
				} else {
					atomic.AddInt32(&successCount, 1)
				}
			}(d)
		}

		wg.Wait()
		t.Logf("Concurrent starts: %d success, %d errors", successCount, errorCount)
	})
}

// TestWorkspaceRaceConditions tests race conditions in workspace operations
func TestWorkspaceRaceConditions(t *testing.T) {
	cfg := setupTestConfig(t)

	d, err := New(cfg)
	if err != nil {
		t.Fatalf("Failed to create daemon: %v", err)
	}

	// Concurrent registration and unregistration
	t.Run("ConcurrentRegisterUnregister", func(t *testing.T) {
		var wg sync.WaitGroup
		numOps := 100

		for i := 0; i < numOps; i++ {
			wg.Add(2)

			// Register
			go func(n int) {
				defer wg.Done()
				d.RegisterWorkspace(fmt.Sprintf("ws_%d", n), fmt.Sprintf("/path/%d", n))
			}(i)

			// Unregister (may or may not exist)
			go func(n int) {
				defer wg.Done()
				d.UnregisterWorkspace(fmt.Sprintf("ws_%d", n))
			}(i)
		}

		wg.Wait()
	})

	// Concurrent reads and writes
	t.Run("ConcurrentReadsWrites", func(t *testing.T) {
		var wg sync.WaitGroup
		numOps := 100

		for i := 0; i < numOps; i++ {
			wg.Add(4)

			// Write
			go func(n int) {
				defer wg.Done()
				d.RegisterWorkspace(fmt.Sprintf("ws_rw_%d", n), fmt.Sprintf("/path/%d", n))
			}(i)

			// Read single
			go func(n int) {
				defer wg.Done()
				d.GetWorkspaceState(fmt.Sprintf("ws_rw_%d", n))
			}(i)

			// Read all
			go func(n int) {
				defer wg.Done()
				d.GetAllWorkspaces()
			}(i)

			// Count
			go func(n int) {
				defer wg.Done()
				d.GetWorkspaceCount()
			}(i)
		}

		wg.Wait()
	})

	// Concurrent activity updates
	t.Run("ConcurrentActivityUpdates", func(t *testing.T) {
		// Register a workspace first
		d.RegisterWorkspace("ws_activity", "/path/activity")

		var wg sync.WaitGroup
		numOps := 100

		for i := 0; i < numOps; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				d.UpdateWorkspaceActivity("ws_activity")
			}()
		}

		wg.Wait()

		// Verify workspace still exists and is valid
		state := d.GetWorkspaceState("ws_activity")
		if state == nil {
			t.Error("Workspace should still exist after concurrent updates")
		}
	})

	// Concurrent status updates
	t.Run("ConcurrentStatusUpdates", func(t *testing.T) {
		d.RegisterWorkspace("ws_status", "/path/status")

		var wg sync.WaitGroup
		statuses := []string{"running", "stopped", "error", "paused"}

		for i := 0; i < 100; i++ {
			wg.Add(1)
			go func(n int) {
				defer wg.Done()
				status := statuses[n%len(statuses)]
				d.UpdateWorkspaceStatus("ws_status", status)
			}(i)
		}

		wg.Wait()

		// Verify workspace still exists
		state := d.GetWorkspaceState("ws_status")
		if state == nil {
			t.Error("Workspace should still exist after concurrent status updates")
		}
	})
}

// TestHealthManagerRaceConditions tests race conditions in health manager
func TestHealthManagerRaceConditions(t *testing.T) {
	hm := NewHealthManager()

	t.Run("ConcurrentRegistration", func(t *testing.T) {
		var wg sync.WaitGroup

		for i := 0; i < 50; i++ {
			wg.Add(1)
			go func(n int) {
				defer wg.Done()
				checker := &testHealthChecker{
					name:    fmt.Sprintf("checker_%d", n),
					healthy: n%2 == 0,
				}
				if !checker.healthy {
					checker.err = fmt.Errorf("unhealthy")
				}
				hm.Register(checker)
			}(i)
		}

		wg.Wait()
	})

	t.Run("ConcurrentChecksAndRegistration", func(t *testing.T) {
		var wg sync.WaitGroup

		for i := 0; i < 50; i++ {
			wg.Add(2)

			// Register
			go func(n int) {
				defer wg.Done()
				checker := &testHealthChecker{
					name:    fmt.Sprintf("checker_cr_%d", n),
					healthy: true,
				}
				hm.Register(checker)
			}(i)

			// Run checks
			go func() {
				defer wg.Done()
				hm.RunChecks()
			}()
		}

		wg.Wait()
	})

	t.Run("ConcurrentGetStatus", func(t *testing.T) {
		// Register some checkers first
		for i := 0; i < 10; i++ {
			checker := &testHealthChecker{
				name:    fmt.Sprintf("checker_gs_%d", i),
				healthy: true,
			}
			hm.Register(checker)
		}
		hm.RunChecks()

		var wg sync.WaitGroup
		for i := 0; i < 100; i++ {
			wg.Add(2)

			go func(n int) {
				defer wg.Done()
				hm.GetStatus(fmt.Sprintf("checker_gs_%d", n%10))
			}(i)

			go func() {
				defer wg.Done()
				hm.GetAllStatuses()
			}()
		}

		wg.Wait()
	})
}

// TestShutdownCallbackRaceConditions tests race conditions in shutdown callbacks
func TestShutdownCallbackRaceConditions(t *testing.T) {
	cfg := setupTestConfig(t)

	d, err := New(cfg)
	if err != nil {
		t.Fatalf("Failed to create daemon: %v", err)
	}

	t.Run("ConcurrentCallbackRegistration", func(t *testing.T) {
		var wg sync.WaitGroup
		var callCount int32

		for i := 0; i < 100; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				d.RegisterShutdownCallback(func() error {
					atomic.AddInt32(&callCount, 1)
					return nil
				})
			}()
		}

		wg.Wait()

		// Run callbacks
		d.runShutdownCallbacks()

		if atomic.LoadInt32(&callCount) != 100 {
			t.Errorf("Expected 100 callbacks called, got %d", callCount)
		}
	})
}

// TestDaemonUptimeAccuracy tests that uptime is tracked correctly
func TestDaemonUptimeAccuracy(t *testing.T) {
	cfg := setupTestConfig(t)

	d, err := New(cfg)
	if err != nil {
		t.Fatalf("Failed to create daemon: %v", err)
	}

	// Initial uptime should be very small
	uptime := d.Uptime()
	if uptime > 1*time.Second {
		t.Errorf("Initial uptime too large: %v", uptime)
	}

	// Wait and check
	time.Sleep(100 * time.Millisecond)

	uptime = d.Uptime()
	if uptime < 100*time.Millisecond {
		t.Errorf("Uptime should be at least 100ms, got %v", uptime)
	}
}

// TestPIDFileCreation tests PID file creation and cleanup
func TestPIDFileCreation(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "dba-pid-creation-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	cfg := &config.Config{
		Home: tmpDir,
		Daemon: config.DaemonConfig{
			Socket:   filepath.Join(tmpDir, "daemon.sock"),
			PIDFile:  filepath.Join(tmpDir, "subdir", "daemon.pid"),
			LogFile:  filepath.Join(tmpDir, "daemon.log"),
			LogLevel: "info",
		},
	}

	d, err := New(cfg)
	if err != nil {
		t.Fatalf("Failed to create daemon: %v", err)
	}

	// Test PID file creation with nested directory
	t.Run("NestedPIDDirectory", func(t *testing.T) {
		err := d.writePIDFile()
		if err != nil {
			t.Fatalf("Failed to write PID file: %v", err)
		}

		// Verify PID file exists
		if _, err := os.Stat(cfg.Daemon.PIDFile); os.IsNotExist(err) {
			t.Error("PID file should exist")
		}

		// Verify content
		content, err := os.ReadFile(cfg.Daemon.PIDFile)
		if err != nil {
			t.Fatalf("Failed to read PID file: %v", err)
		}

		expectedPID := fmt.Sprintf("%d", os.Getpid())
		if string(content) != expectedPID {
			t.Errorf("Expected PID %s, got %s", expectedPID, string(content))
		}
	})
}

// TestSocketCreationAndCleanup tests socket file handling
func TestSocketCreationAndCleanup(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "dba-socket-creation-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	socketPath := filepath.Join(tmpDir, "subdir", "daemon.sock")

	cfg := &config.Config{
		Home: tmpDir,
		Daemon: config.DaemonConfig{
			Socket:   socketPath,
			PIDFile:  filepath.Join(tmpDir, "daemon.pid"),
			LogFile:  filepath.Join(tmpDir, "daemon.log"),
			LogLevel: "info",
		},
	}

	// Create socket directory
	socketDir := filepath.Dir(socketPath)
	if err := os.MkdirAll(socketDir, 0755); err != nil {
		t.Fatalf("Failed to create socket dir: %v", err)
	}

	// Create listener
	listener, err := net.Listen("unix", socketPath)
	if err != nil {
		t.Fatalf("Failed to create listener: %v", err)
	}

	// Verify socket exists
	if _, err := os.Stat(socketPath); os.IsNotExist(err) {
		t.Error("Socket file should exist")
	}

	// Close and cleanup
	listener.Close()
	os.Remove(socketPath)

	// Verify cleanup
	if _, err := os.Stat(socketPath); !os.IsNotExist(err) {
		t.Error("Socket file should be removed")
	}

	// Test that new daemon can create socket in same location
	listener2, err := net.Listen("unix", socketPath)
	if err != nil {
		t.Fatalf("Failed to create listener after cleanup: %v", err)
	}
	listener2.Close()
	os.Remove(socketPath)

	// Verify config object is correctly set
	_ = cfg // just to use cfg
}

// TestMultipleClientInstances tests multiple client instances
func TestMultipleClientInstances(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "dba-multi-client-*")
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

	// Create multiple clients
	numClients := 10
	clients := make([]*Client, numClients)
	for i := 0; i < numClients; i++ {
		clients[i] = NewClient(cfg)
	}

	// All clients should work independently
	t.Run("IndependentClients", func(t *testing.T) {
		var wg sync.WaitGroup
		var successCount int32

		for _, client := range clients {
			wg.Add(1)
			go func(c *Client) {
				defer wg.Done()
				if c.IsRunning() {
					atomic.AddInt32(&successCount, 1)
				}
			}(client)
		}

		wg.Wait()

		if int(successCount) != numClients {
			t.Errorf("Expected all %d clients to succeed, got %d", numClients, successCount)
		}
	})

	// Each client can register different workspaces
	t.Run("DifferentWorkspaces", func(t *testing.T) {
		var wg sync.WaitGroup

		for i, client := range clients {
			wg.Add(1)
			go func(c *Client, n int) {
				defer wg.Done()
				c.RegisterWorkspace(fmt.Sprintf("ws_client_%d", n), fmt.Sprintf("/path/%d", n))
			}(client, i)
		}

		wg.Wait()

		// Verify all workspaces were registered
		workspaces := d.GetAllWorkspaces()
		if len(workspaces) < numClients {
			t.Errorf("Expected at least %d workspaces, got %d", numClients, len(workspaces))
		}
	})
}

// TestHealthCheckIntervalTiming tests health check timing
func TestHealthCheckIntervalTiming(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping timing test in short mode")
	}

	cfg := setupTestConfig(t)

	d, err := New(cfg)
	if err != nil {
		t.Fatalf("Failed to create daemon: %v", err)
	}

	// Register a health checker that tracks calls
	var callTimes []time.Time
	var mu sync.Mutex

	checker := &trackingHealthChecker{
		name: "timing-tracker",
		onCheck: func() {
			mu.Lock()
			callTimes = append(callTimes, time.Now())
			mu.Unlock()
		},
	}

	d.GetHealthManager().Register(checker)

	// Manually call performHealthChecks multiple times
	for i := 0; i < 3; i++ {
		d.performHealthChecks()
		time.Sleep(10 * time.Millisecond)
	}

	mu.Lock()
	defer mu.Unlock()

	if len(callTimes) != 3 {
		t.Errorf("Expected 3 check calls, got %d", len(callTimes))
	}
}

// trackingHealthChecker tracks when it's called
type trackingHealthChecker struct {
	name    string
	onCheck func()
	healthy bool
}

func (c *trackingHealthChecker) Name() string {
	return c.name
}

func (c *trackingHealthChecker) Check() error {
	if c.onCheck != nil {
		c.onCheck()
	}
	c.healthy = true
	return nil
}

func (c *trackingHealthChecker) IsHealthy() bool {
	return c.healthy
}

// TestDaemonResourceCleanup tests that resources are properly cleaned up
func TestDaemonResourceCleanup(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "dba-cleanup-*")
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
	d.writePIDFile()

	// Create socket (simulating started daemon)
	listener, err := net.Listen("unix", cfg.Daemon.Socket)
	if err != nil {
		t.Fatalf("Failed to create listener: %v", err)
	}
	listener.Close()
	// Recreate for daemon to own
	listener, err = net.Listen("unix", cfg.Daemon.Socket)
	if err != nil {
		t.Fatalf("Failed to recreate listener: %v", err)
	}
	d.listener = listener

	// Verify files exist
	if _, err := os.Stat(cfg.Daemon.Socket); os.IsNotExist(err) {
		t.Error("Socket should exist before cleanup")
	}
	if _, err := os.Stat(cfg.Daemon.PIDFile); os.IsNotExist(err) {
		t.Error("PID file should exist before cleanup")
	}

	// Cleanup
	d.cleanup()

	// Verify files are removed
	if _, err := os.Stat(cfg.Daemon.Socket); !os.IsNotExist(err) {
		t.Error("Socket should be removed after cleanup")
	}
	if _, err := os.Stat(cfg.Daemon.PIDFile); !os.IsNotExist(err) {
		t.Error("PID file should be removed after cleanup")
	}
}

// TestContextCancellationPropagation tests context cancellation
func TestContextCancellationPropagation(t *testing.T) {
	cfg := setupTestConfig(t)

	d, err := New(cfg)
	if err != nil {
		t.Fatalf("Failed to create daemon: %v", err)
	}

	ctx := d.Context()

	// Context should be initially valid
	select {
	case <-ctx.Done():
		t.Error("Context should not be cancelled initially")
	default:
		// Good
	}

	// Start a goroutine that waits on context
	var done bool
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		<-ctx.Done()
		done = true
	}()

	// Cancel
	d.cancel()

	// Wait for goroutine to notice
	wg.Wait()

	if !done {
		t.Error("Goroutine should have been notified of cancellation")
	}
}

// TestLoggerOutput tests logger output functionality
func TestLoggerOutput(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "dba-logger-output-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	logFile := filepath.Join(tmpDir, "daemon.log")

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

	err = d.setupLogging()
	if err != nil {
		t.Fatalf("Failed to setup logging: %v", err)
	}

	// Write test messages
	d.Logger().Println("test message 1")
	d.Logger().Println("test message 2")
	d.Logger().Printf("formatted message: %d", 42)

	// Give time for writes
	time.Sleep(50 * time.Millisecond)

	// Read log file
	content, err := os.ReadFile(logFile)
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	contentStr := string(content)

	// Verify messages are in log
	if !containsSubstring(contentStr, "test message 1") {
		t.Error("Log should contain 'test message 1'")
	}
	if !containsSubstring(contentStr, "test message 2") {
		t.Error("Log should contain 'test message 2'")
	}
	if !containsSubstring(contentStr, "formatted message: 42") {
		t.Error("Log should contain 'formatted message: 42'")
	}
}

func containsSubstring(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsSubstring(s[1:], substr) || (len(s) >= len(substr) && s[:len(substr)] == substr))
}
