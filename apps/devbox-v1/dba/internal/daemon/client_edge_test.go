// Package daemon provides edge case tests for the daemon client.
package daemon

import (
	"context"
	"encoding/json"
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

// TestClientConnectionRetry tests client behavior with connection failures
func TestClientConnectionRetry(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "dba-client-retry-*")
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

	// Test IsRunning with no daemon
	t.Run("IsRunningNoDaemon", func(t *testing.T) {
		if client.IsRunning() {
			t.Error("Should report not running when no daemon exists")
		}
	})

	// Test Status with no daemon
	t.Run("StatusNoDaemon", func(t *testing.T) {
		_, err := client.Status()
		if err == nil {
			t.Error("Should return error when no daemon exists")
		}
	})

	// Test RegisterWorkspace with no daemon
	t.Run("RegisterWorkspaceNoDaemon", func(t *testing.T) {
		err := client.RegisterWorkspace("test", "/path")
		if err == nil {
			t.Error("Should return error when no daemon exists")
		}
	})

	// Test UnregisterWorkspace with no daemon
	t.Run("UnregisterWorkspaceNoDaemon", func(t *testing.T) {
		err := client.UnregisterWorkspace("test")
		if err == nil {
			t.Error("Should return error when no daemon exists")
		}
	})

	// Test GetWorkspaceState with no daemon
	t.Run("GetWorkspaceStateNoDaemon", func(t *testing.T) {
		_, err := client.GetWorkspaceState("test")
		if err == nil {
			t.Error("Should return error when no daemon exists")
		}
	})

	// Test ListWorkspaces with no daemon
	t.Run("ListWorkspacesNoDaemon", func(t *testing.T) {
		_, err := client.ListWorkspaces()
		if err == nil {
			t.Error("Should return error when no daemon exists")
		}
	})

	// Test WaitForSync with no daemon
	t.Run("WaitForSyncNoDaemon", func(t *testing.T) {
		_, err := client.WaitForSync("test", 1*time.Second)
		if err == nil {
			t.Error("Should return error when no daemon exists")
		}
	})
}

// TestClientWithSlowServer tests client behavior with slow server responses
func TestClientWithSlowServer(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping slow server test in short mode")
	}

	tmpDir, err := os.MkdirTemp("", "dba-client-slow-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	socketPath := filepath.Join(tmpDir, "daemon.sock")

	// Create a slow server
	listener, err := net.Listen("unix", socketPath)
	if err != nil {
		t.Fatalf("Failed to create listener: %v", err)
	}
	defer listener.Close()

	slowHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Simulate slow response
		time.Sleep(200 * time.Millisecond)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"status": "ok"})
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

	client := NewClient(cfg)

	// Test that slow responses work
	t.Run("SlowHealthCheck", func(t *testing.T) {
		start := time.Now()
		running := client.IsRunning()
		elapsed := time.Since(start)

		if !running {
			t.Error("Should report running")
		}

		if elapsed < 200*time.Millisecond {
			t.Errorf("Should have waited for slow response, elapsed: %v", elapsed)
		}
	})
}

// TestClientConcurrentRequests tests concurrent client requests
func TestClientConcurrentRequests(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "dba-client-concurrent-*")
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

	client := NewClient(cfg)

	// Run many concurrent requests
	t.Run("ManyConcurrentRequests", func(t *testing.T) {
		var wg sync.WaitGroup
		var successCount, errorCount int64
		numRequests := 100

		for i := 0; i < numRequests; i++ {
			wg.Add(1)
			go func(n int) {
				defer wg.Done()

				// Mix of different operations
				switch n % 4 {
				case 0:
					if client.IsRunning() {
						atomic.AddInt64(&successCount, 1)
					} else {
						atomic.AddInt64(&errorCount, 1)
					}
				case 1:
					_, err := client.Status()
					if err == nil {
						atomic.AddInt64(&successCount, 1)
					} else {
						atomic.AddInt64(&errorCount, 1)
					}
				case 2:
					err := client.RegisterWorkspace(fmt.Sprintf("ws_%d", n), "/path")
					if err == nil {
						atomic.AddInt64(&successCount, 1)
					} else {
						atomic.AddInt64(&errorCount, 1)
					}
				case 3:
					_, err := client.ListWorkspaces()
					if err == nil {
						atomic.AddInt64(&successCount, 1)
					} else {
						atomic.AddInt64(&errorCount, 1)
					}
				}
			}(i)
		}

		wg.Wait()

		t.Logf("Concurrent requests: %d success, %d errors", successCount, errorCount)

		if successCount == 0 {
			t.Error("Expected some successful requests")
		}

		// Allow some errors due to timing, but not too many
		errorRate := float64(errorCount) / float64(numRequests)
		if errorRate > 0.1 {
			t.Errorf("Error rate too high: %.2f%%", errorRate*100)
		}
	})
}

// TestClientWithBadJSON tests client handling of malformed JSON responses
func TestClientWithBadJSON(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "dba-client-badjson-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	socketPath := filepath.Join(tmpDir, "daemon.sock")

	// Create a server that returns bad JSON
	listener, err := net.Listen("unix", socketPath)
	if err != nil {
		t.Fatalf("Failed to create listener: %v", err)
	}
	defer listener.Close()

	badJSONHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte("{invalid json"))
	})

	server := &http.Server{Handler: badJSONHandler}
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

	client := NewClient(cfg)

	// Test Status with bad JSON
	t.Run("StatusBadJSON", func(t *testing.T) {
		_, err := client.Status()
		if err == nil {
			t.Error("Should return error for bad JSON")
		}
	})
}

// TestClientWithHTTPErrors tests client handling of HTTP errors
func TestClientWithHTTPErrors(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "dba-client-httperr-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	socketPath := filepath.Join(tmpDir, "daemon.sock")

	// Create a server that returns various HTTP errors
	listener, err := net.Listen("unix", socketPath)
	if err != nil {
		t.Fatalf("Failed to create listener: %v", err)
	}
	defer listener.Close()

	var errorCode int = 500

	errorHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "server error", errorCode)
	})

	server := &http.Server{Handler: errorHandler}
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

	client := NewClient(cfg)

	// Test with 500 error
	t.Run("InternalServerError", func(t *testing.T) {
		running := client.IsRunning()
		if running {
			t.Error("Should report not running on 500 error")
		}
	})
}

// TestGetDaemonPIDEdgeCases tests edge cases for GetDaemonPID
func TestGetDaemonPIDEdgeCases(t *testing.T) {
	// Test with valid current process PID
	t.Run("CurrentProcessPID", func(t *testing.T) {
		tmpDir, err := os.MkdirTemp("", "dba-pid-current-*")
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

		// Write current process PID
		currentPID := os.Getpid()
		os.WriteFile(cfg.Daemon.PIDFile, []byte(fmt.Sprintf("%d", currentPID)), 0644)

		pid := GetDaemonPID(cfg)
		if pid != currentPID {
			t.Errorf("Expected PID %d, got %d", currentPID, pid)
		}
	})

	// Test with PID 1 (init process)
	t.Run("InitProcessPID", func(t *testing.T) {
		tmpDir, err := os.MkdirTemp("", "dba-pid-init-*")
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

		// PID 1 exists (init/systemd) but isn't our daemon
		os.WriteFile(cfg.Daemon.PIDFile, []byte("1"), 0644)

		pid := GetDaemonPID(cfg)
		// Should return 1 since process exists (even though it's not our daemon)
		// The actual daemon check happens elsewhere
		if pid != 1 {
			t.Logf("PID 1 check returned %d (may vary by system)", pid)
		}
	})

	// Test with negative PID
	t.Run("NegativePID", func(t *testing.T) {
		tmpDir, err := os.MkdirTemp("", "dba-pid-negative-*")
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

		os.WriteFile(cfg.Daemon.PIDFile, []byte("-123"), 0644)

		pid := GetDaemonPID(cfg)
		if pid != 0 {
			t.Errorf("Expected PID 0 for negative value, got %d", pid)
		}
	})

	// Test with whitespace around PID
	t.Run("WhitespacePID", func(t *testing.T) {
		tmpDir, err := os.MkdirTemp("", "dba-pid-whitespace-*")
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

		currentPID := os.Getpid()
		os.WriteFile(cfg.Daemon.PIDFile, []byte(fmt.Sprintf("  %d  \n", currentPID)), 0644)

		pid := GetDaemonPID(cfg)
		if pid != currentPID {
			t.Errorf("Expected PID %d, got %d (whitespace handling)", currentPID, pid)
		}
	})
}

// TestStopDaemonEdgeCases tests edge cases for StopDaemon
func TestStopDaemonEdgeCases(t *testing.T) {
	// Test stopping non-existent daemon
	t.Run("StopNonExistent", func(t *testing.T) {
		tmpDir, err := os.MkdirTemp("", "dba-stop-nonexistent-*")
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

		err = StopDaemon(cfg)
		if err == nil {
			t.Error("Should return error when no daemon running")
		}
	})

	// Test stopping with stale PID file
	t.Run("StopStalePID", func(t *testing.T) {
		tmpDir, err := os.MkdirTemp("", "dba-stop-stale-*")
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

		// Write stale PID
		os.WriteFile(cfg.Daemon.PIDFile, []byte("999999999"), 0644)

		err = StopDaemon(cfg)
		// Should handle stale PID gracefully
		// Either returns nil (cleaned up) or error (couldn't find process)
		t.Logf("StopDaemon with stale PID result: %v", err)
	})
}

// TestEnsureRunningEdgeCases tests edge cases for EnsureRunning
func TestEnsureRunningEdgeCases(t *testing.T) {
	// Test when daemon already running
	t.Run("AlreadyRunning", func(t *testing.T) {
		tmpDir, err := os.MkdirTemp("", "dba-ensure-running-*")
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

		// Start a mock daemon
		listener, err := net.Listen("unix", socketPath)
		if err != nil {
			t.Fatalf("Failed to create listener: %v", err)
		}
		defer listener.Close()

		server := &http.Server{Handler: d.createRouter()}
		go server.Serve(listener)
		defer server.Close()

		time.Sleep(50 * time.Millisecond)

		// EnsureRunning should succeed quickly when daemon already running
		start := time.Now()
		err = EnsureRunning(cfg)
		elapsed := time.Since(start)

		if err != nil {
			t.Errorf("EnsureRunning should succeed: %v", err)
		}

		if elapsed > 1*time.Second {
			t.Errorf("EnsureRunning took too long when daemon running: %v", elapsed)
		}
	})
}

// TestClientUpdateActivity tests UpdateWorkspaceActivity
func TestClientUpdateActivity(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "dba-client-activity-*")
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

	// Register a workspace
	d.RegisterWorkspace("ws_activity_test", "/path/test")

	listener, err := net.Listen("unix", socketPath)
	if err != nil {
		t.Fatalf("Failed to create listener: %v", err)
	}
	defer listener.Close()

	server := &http.Server{Handler: d.createRouter()}
	go server.Serve(listener)
	defer server.Close()

	time.Sleep(50 * time.Millisecond)

	client := NewClient(cfg)

	// Test UpdateWorkspaceActivity
	t.Run("UpdateActivity", func(t *testing.T) {
		err := client.UpdateWorkspaceActivity("ws_activity_test")
		if err != nil {
			t.Errorf("UpdateWorkspaceActivity failed: %v", err)
		}
	})

	// Test with non-existent workspace (should not error)
	t.Run("UpdateActivityNonExistent", func(t *testing.T) {
		err := client.UpdateWorkspaceActivity("ws_nonexistent")
		// This might not error, just doesn't update anything
		if err != nil {
			t.Logf("UpdateWorkspaceActivity for non-existent workspace: %v", err)
		}
	})
}

// TestSyncBarrierInterface tests the SyncBarrier interface
func TestSyncBarrierInterface(t *testing.T) {
	// Test with no sync barrier set
	t.Run("NoSyncBarrier", func(t *testing.T) {
		// Save current sync barrier
		oldBarrier := syncBarrier
		defer func() { syncBarrier = oldBarrier }()

		// Clear sync barrier
		syncBarrier = nil

		tmpDir, err := os.MkdirTemp("", "dba-syncbarrier-*")
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

		client := NewClient(cfg)

		result, err := client.WaitForSync("test", 1*time.Second)
		if err != nil {
			t.Fatalf("WaitForSync failed: %v", err)
		}

		if !result.Synced {
			t.Error("Expected synced=true with no barrier")
		}
	})

	// Test with custom sync barrier
	t.Run("CustomSyncBarrier", func(t *testing.T) {
		// Save current sync barrier
		oldBarrier := syncBarrier
		defer func() { syncBarrier = oldBarrier }()

		// Set custom sync barrier
		customBarrier := &mockSyncBarrier{waitMs: 50}
		SetSyncBarrier(customBarrier)

		tmpDir, err := os.MkdirTemp("", "dba-syncbarrier-custom-*")
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

		client := NewClient(cfg)

		result, err := client.WaitForSync("test", 1*time.Second)
		if err != nil {
			t.Fatalf("WaitForSync failed: %v", err)
		}

		if result.WaitMs != 50 {
			t.Errorf("Expected waitMs=50, got %d", result.WaitMs)
		}
	})

	// Test with failing sync barrier
	t.Run("FailingSyncBarrier", func(t *testing.T) {
		// Save current sync barrier
		oldBarrier := syncBarrier
		defer func() { syncBarrier = oldBarrier }()

		// Set failing sync barrier
		failingBarrier := &mockSyncBarrier{err: fmt.Errorf("sync failed")}
		SetSyncBarrier(failingBarrier)

		tmpDir, err := os.MkdirTemp("", "dba-syncbarrier-fail-*")
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

		client := NewClient(cfg)

		result, err := client.WaitForSync("test", 1*time.Second)
		if err != nil {
			t.Fatalf("WaitForSync failed: %v", err)
		}

		if result.Synced {
			t.Error("Expected synced=false with failing barrier")
		}
	})
}

// mockSyncBarrier is a mock implementation of SyncBarrier for testing
type mockSyncBarrier struct {
	waitMs int64
	err    error
}

func (m *mockSyncBarrier) Wait(workspaceID string, timeout time.Duration) (int64, error) {
	if m.err != nil {
		return 0, m.err
	}
	if m.waitMs > 0 {
		time.Sleep(time.Duration(m.waitMs) * time.Millisecond)
	}
	return m.waitMs, nil
}

// TestClientContextCancellation tests client behavior with cancelled contexts
func TestClientContextCancellation(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "dba-client-context-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	socketPath := filepath.Join(tmpDir, "daemon.sock")

	// Create a slow server
	listener, err := net.Listen("unix", socketPath)
	if err != nil {
		t.Fatalf("Failed to create listener: %v", err)
	}
	defer listener.Close()

	slowHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(5 * time.Second)
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

	// Create client with very short timeout
	client := &Client{
		socketPath: cfg.Daemon.Socket,
		httpClient: &http.Client{
			Transport: &http.Transport{
				DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
					var d net.Dialer
					return d.DialContext(ctx, "unix", cfg.Daemon.Socket)
				},
			},
			Timeout: 100 * time.Millisecond,
		},
	}

	// Test that timeout works
	t.Run("RequestTimeout", func(t *testing.T) {
		start := time.Now()
		_, err := client.Status()
		elapsed := time.Since(start)

		if err == nil {
			t.Error("Expected timeout error")
		}

		if elapsed > 500*time.Millisecond {
			t.Errorf("Timeout took too long: %v", elapsed)
		}
	})
}
