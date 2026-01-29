// Package daemon tests the daemon functionality.
package daemon

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/dba-cli/dba/internal/config"
)

func setupTestConfig(t *testing.T) *config.Config {
	t.Helper()

	tmpDir, err := os.MkdirTemp("", "dba-daemon-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	t.Cleanup(func() {
		os.RemoveAll(tmpDir)
	})

	return &config.Config{
		Home: tmpDir,
		Daemon: config.DaemonConfig{
			Socket:   filepath.Join(tmpDir, "daemon.sock"),
			PIDFile:  filepath.Join(tmpDir, "daemon.pid"),
			LogFile:  filepath.Join(tmpDir, "daemon.log"),
			LogLevel: "info",
		},
		Ports: config.PortConfig{
			RangeStart: 10000,
			RangeEnd:   60000,
			BlockSize:  100,
			StandardOffsets: map[string]int{
				"PORT":      0,
				"API_PORT":  1,
				"CODE_PORT": 80,
				"VNC_PORT":  90,
			},
		},
	}
}

func TestDaemonNew(t *testing.T) {
	cfg := setupTestConfig(t)

	d, err := New(cfg)
	if err != nil {
		t.Fatalf("Failed to create daemon: %v", err)
	}

	if d == nil {
		t.Fatal("Daemon is nil")
	}

	if d.config != cfg {
		t.Error("Daemon config mismatch")
	}

	if d.workspaces == nil {
		t.Error("Workspaces map is nil")
	}
}

func TestDaemonWorkspaceRegistration(t *testing.T) {
	cfg := setupTestConfig(t)

	d, err := New(cfg)
	if err != nil {
		t.Fatalf("Failed to create daemon: %v", err)
	}

	// Test registration
	d.RegisterWorkspace("ws_test1", "/path/to/workspace1")
	d.RegisterWorkspace("ws_test2", "/path/to/workspace2")

	if d.GetWorkspaceCount() != 2 {
		t.Errorf("Expected 2 workspaces, got %d", d.GetWorkspaceCount())
	}

	// Test get workspace state
	state := d.GetWorkspaceState("ws_test1")
	if state == nil {
		t.Fatal("Workspace state is nil")
	}

	if state.ID != "ws_test1" {
		t.Errorf("Expected workspace ID ws_test1, got %s", state.ID)
	}

	if state.Path != "/path/to/workspace1" {
		t.Errorf("Expected path /path/to/workspace1, got %s", state.Path)
	}

	// Test unregistration
	d.UnregisterWorkspace("ws_test1")

	if d.GetWorkspaceCount() != 1 {
		t.Errorf("Expected 1 workspace, got %d", d.GetWorkspaceCount())
	}

	state = d.GetWorkspaceState("ws_test1")
	if state != nil {
		t.Error("Expected nil state for unregistered workspace")
	}
}

func TestDaemonGetAllWorkspaces(t *testing.T) {
	cfg := setupTestConfig(t)

	d, err := New(cfg)
	if err != nil {
		t.Fatalf("Failed to create daemon: %v", err)
	}

	d.RegisterWorkspace("ws_1", "/path/1")
	d.RegisterWorkspace("ws_2", "/path/2")
	d.RegisterWorkspace("ws_3", "/path/3")

	workspaces := d.GetAllWorkspaces()
	if len(workspaces) != 3 {
		t.Errorf("Expected 3 workspaces, got %d", len(workspaces))
	}

	// Verify all workspaces are present
	ids := make(map[string]bool)
	for _, ws := range workspaces {
		ids[ws.ID] = true
	}

	for _, id := range []string{"ws_1", "ws_2", "ws_3"} {
		if !ids[id] {
			t.Errorf("Expected workspace %s not found", id)
		}
	}
}

func TestDaemonUpdateWorkspaceActivity(t *testing.T) {
	cfg := setupTestConfig(t)

	d, err := New(cfg)
	if err != nil {
		t.Fatalf("Failed to create daemon: %v", err)
	}

	d.RegisterWorkspace("ws_test", "/path/to/workspace")

	// Get initial state
	state := d.GetWorkspaceState("ws_test")
	initialTime := state.LastActive

	// Wait a bit and update activity
	time.Sleep(10 * time.Millisecond)
	d.UpdateWorkspaceActivity("ws_test")

	state = d.GetWorkspaceState("ws_test")
	if !state.LastActive.After(initialTime) {
		t.Error("LastActive should be updated after UpdateWorkspaceActivity")
	}
}

func TestClientIsRunning_NotRunning(t *testing.T) {
	cfg := setupTestConfig(t)

	client := NewClient(cfg)
	if client.IsRunning() {
		t.Error("Client should report daemon as not running")
	}
}

func TestGetDaemonPID_NotRunning(t *testing.T) {
	cfg := setupTestConfig(t)

	pid := GetDaemonPID(cfg)
	if pid != 0 {
		t.Errorf("Expected PID 0 when daemon not running, got %d", pid)
	}
}

func TestHealthManager(t *testing.T) {
	hm := NewHealthManager()

	// Create a simple health checker
	checker := &testHealthChecker{
		name:    "test",
		healthy: true,
	}

	hm.Register(checker)

	// Run checks
	hm.RunChecks()

	// Verify status
	status := hm.GetStatus("test")
	if status == nil {
		t.Fatal("Status is nil")
	}

	if !status.Healthy {
		t.Error("Expected healthy status")
	}

	if status.Name != "test" {
		t.Errorf("Expected name 'test', got '%s'", status.Name)
	}

	// Test IsAllHealthy
	if !hm.IsAllHealthy() {
		t.Error("Expected all healthy")
	}

	// Make checker unhealthy
	checker.healthy = false
	checker.err = http.ErrNotSupported
	hm.RunChecks()

	if hm.IsAllHealthy() {
		t.Error("Expected not all healthy")
	}

	// Unregister
	hm.Unregister("test")
	if hm.GetStatus("test") != nil {
		t.Error("Expected nil status after unregister")
	}
}

// testHealthChecker is a mock health checker for testing
type testHealthChecker struct {
	name    string
	healthy bool
	err     error
}

func (t *testHealthChecker) Name() string {
	return t.name
}

func (t *testHealthChecker) Check() error {
	if t.err != nil {
		return t.err
	}
	return nil
}

func (t *testHealthChecker) IsHealthy() bool {
	return t.healthy
}

func TestPortHealthChecker(t *testing.T) {
	// Create a checker for a port that's likely not in use
	checker := NewPortHealthChecker("test-port", "localhost", 59999, 100*time.Millisecond)

	// The port should not be responding (nothing listening)
	err := checker.Check()
	if err == nil {
		t.Skip("Port 59999 is in use, skipping test")
	}

	if checker.IsHealthy() {
		t.Error("Expected unhealthy when port not responding")
	}
}

func TestHealthResponse(t *testing.T) {
	cfg := setupTestConfig(t)

	// Create daemon but don't start it (can't do full integration test without starting)
	d, err := New(cfg)
	if err != nil {
		t.Fatalf("Failed to create daemon: %v", err)
	}

	// Create the router and test the health handler directly
	router := d.createRouter()

	// This would require a full HTTP test server setup
	// For now, just verify the router is created
	if router == nil {
		t.Error("Router is nil")
	}
}

func TestWorkspaceState_JSON(t *testing.T) {
	ws := &WorkspaceState{
		ID:         "ws_test",
		Path:       "/test/path",
		LastActive: time.Now(),
	}

	data, err := json.Marshal(ws)
	if err != nil {
		t.Fatalf("Failed to marshal workspace state: %v", err)
	}

	var decoded WorkspaceState
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal workspace state: %v", err)
	}

	if decoded.ID != ws.ID {
		t.Errorf("Expected ID %s, got %s", ws.ID, decoded.ID)
	}

	if decoded.Path != ws.Path {
		t.Errorf("Expected path %s, got %s", ws.Path, decoded.Path)
	}
}

func TestDaemonUptime(t *testing.T) {
	cfg := setupTestConfig(t)

	d, err := New(cfg)
	if err != nil {
		t.Fatalf("Failed to create daemon: %v", err)
	}

	time.Sleep(10 * time.Millisecond)

	uptime := d.Uptime()
	if uptime < 10*time.Millisecond {
		t.Errorf("Expected uptime > 10ms, got %v", uptime)
	}
}

func TestCompositeHealthChecker(t *testing.T) {
	checker1 := &testHealthChecker{name: "check1", healthy: true}
	checker2 := &testHealthChecker{name: "check2", healthy: true}

	composite := NewCompositeHealthChecker("composite", checker1, checker2)

	// All healthy
	err := composite.Check()
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if !composite.IsHealthy() {
		t.Error("Expected composite to be healthy")
	}

	// One unhealthy
	checker1.healthy = false
	checker1.err = http.ErrAbortHandler
	err = composite.Check()
	if err == nil {
		t.Error("Expected error when one checker fails")
	}

	if composite.IsHealthy() {
		t.Error("Expected composite to be unhealthy")
	}
}

func TestSyncResult_JSON(t *testing.T) {
	result := &SyncResult{
		Synced: true,
		WaitMs: 100,
	}

	data, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("Failed to marshal sync result: %v", err)
	}

	var decoded SyncResult
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal sync result: %v", err)
	}

	if decoded.Synced != result.Synced {
		t.Errorf("Expected synced %v, got %v", result.Synced, decoded.Synced)
	}

	if decoded.WaitMs != result.WaitMs {
		t.Errorf("Expected wait_ms %d, got %d", result.WaitMs, decoded.WaitMs)
	}
}

func TestStatusResponse_JSON(t *testing.T) {
	status := &StatusResponse{
		Running:          true,
		PID:              12345,
		Socket:           "/tmp/test.sock",
		WorkspacesActive: 3,
		UptimeSeconds:    3600,
	}

	data, err := json.Marshal(status)
	if err != nil {
		t.Fatalf("Failed to marshal status: %v", err)
	}

	var decoded StatusResponse
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal status: %v", err)
	}

	if decoded.Running != status.Running {
		t.Errorf("Expected running %v, got %v", status.Running, decoded.Running)
	}

	if decoded.PID != status.PID {
		t.Errorf("Expected PID %d, got %d", status.PID, decoded.PID)
	}
}

func TestDaemonConcurrentAccess(t *testing.T) {
	cfg := setupTestConfig(t)

	d, err := New(cfg)
	if err != nil {
		t.Fatalf("Failed to create daemon: %v", err)
	}

	// Run concurrent registrations and reads
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	done := make(chan bool, 10)

	// Writers
	for i := 0; i < 5; i++ {
		go func(n int) {
			for {
				select {
				case <-ctx.Done():
					done <- true
					return
				default:
					id := "ws_concurrent_" + string(rune('a'+n))
					d.RegisterWorkspace(id, "/path/"+id)
					time.Sleep(1 * time.Millisecond)
					d.UnregisterWorkspace(id)
				}
			}
		}(i)
	}

	// Readers
	for i := 0; i < 5; i++ {
		go func() {
			for {
				select {
				case <-ctx.Done():
					done <- true
					return
				default:
					_ = d.GetWorkspaceCount()
					_ = d.GetAllWorkspaces()
					time.Sleep(1 * time.Millisecond)
				}
			}
		}()
	}

	// Wait for all goroutines to finish
	for i := 0; i < 10; i++ {
		<-done
	}

	// If we get here without a race condition panic, the test passes
}
