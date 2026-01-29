// internal/daemon/service_state_test.go
// Tests for service state management - Added by Agent #6

package daemon

import (
	"testing"
	"time"

	"github.com/dba-cli/dba/internal/config"
)

func TestServiceStateStruct(t *testing.T) {
	// Test that ServiceState struct has all required fields
	state := &ServiceState{
		Name:      "web",
		Status:    "running",
		PID:       12345,
		Port:      8080,
		Healthy:   true,
		Restarts:  2,
		StartedAt: time.Now(),
		LastCheck: time.Now(),
		Error:     "",
	}

	if state.Name != "web" {
		t.Errorf("expected name 'web', got '%s'", state.Name)
	}
	if state.Status != "running" {
		t.Errorf("expected status 'running', got '%s'", state.Status)
	}
	if state.PID != 12345 {
		t.Errorf("expected PID 12345, got %d", state.PID)
	}
	if state.Port != 8080 {
		t.Errorf("expected port 8080, got %d", state.Port)
	}
	if !state.Healthy {
		t.Error("expected healthy to be true")
	}
	if state.Restarts != 2 {
		t.Errorf("expected restarts 2, got %d", state.Restarts)
	}
}

func TestWorkspaceStateWithServiceStates(t *testing.T) {
	// Test that WorkspaceState includes ServiceStates map
	ws := &WorkspaceState{
		ID:            "ws-123",
		Path:          "/path/to/workspace",
		LastActive:    time.Now(),
		Status:        "running",
		ServiceStates: make(map[string]*ServiceState),
	}

	// Add a service state
	ws.ServiceStates["web"] = &ServiceState{
		Name:    "web",
		Status:  "running",
		Port:    8080,
		Healthy: true,
	}

	if len(ws.ServiceStates) != 1 {
		t.Errorf("expected 1 service state, got %d", len(ws.ServiceStates))
	}

	if ws.ServiceStates["web"].Port != 8080 {
		t.Errorf("expected port 8080, got %d", ws.ServiceStates["web"].Port)
	}
}

func TestDaemonUpdateServiceState(t *testing.T) {
	cfg := &config.Config{
		Daemon: config.DaemonConfig{
			Socket:  "/tmp/dba-test-service-state.sock",
			PIDFile: "/tmp/dba-test-service-state.pid",
			LogFile: "/tmp/dba-test-service-state.log",
		},
	}

	daemon, err := New(cfg)
	if err != nil {
		t.Fatalf("Failed to create daemon: %v", err)
	}

	// Register a workspace
	daemon.RegisterWorkspace("ws-test", "/path/to/test")

	// Update service state
	state := &ServiceState{
		Name:    "api",
		Status:  "running",
		Port:    3000,
		Healthy: true,
	}
	daemon.UpdateServiceState("ws-test", "api", state)

	// Verify state was saved
	retrieved := daemon.GetServiceState("ws-test", "api")
	if retrieved == nil {
		t.Fatal("expected to get service state, got nil")
	}
	if retrieved.Name != "api" {
		t.Errorf("expected name 'api', got '%s'", retrieved.Name)
	}
	if retrieved.Port != 3000 {
		t.Errorf("expected port 3000, got %d", retrieved.Port)
	}
}

func TestDaemonGetServiceStateNotFound(t *testing.T) {
	cfg := &config.Config{
		Daemon: config.DaemonConfig{
			Socket:  "/tmp/dba-test-service-notfound.sock",
			PIDFile: "/tmp/dba-test-service-notfound.pid",
			LogFile: "/tmp/dba-test-service-notfound.log",
		},
	}

	daemon, err := New(cfg)
	if err != nil {
		t.Fatalf("Failed to create daemon: %v", err)
	}

	// Try to get service state for non-existent workspace
	state := daemon.GetServiceState("nonexistent", "api")
	if state != nil {
		t.Errorf("expected nil for non-existent workspace, got %+v", state)
	}

	// Register workspace but don't add service
	daemon.RegisterWorkspace("ws-test", "/path")
	state = daemon.GetServiceState("ws-test", "nonexistent")
	if state != nil {
		t.Errorf("expected nil for non-existent service, got %+v", state)
	}
}

func TestDaemonGetAllServiceStates(t *testing.T) {
	cfg := &config.Config{
		Daemon: config.DaemonConfig{
			Socket:  "/tmp/dba-test-all-states.sock",
			PIDFile: "/tmp/dba-test-all-states.pid",
			LogFile: "/tmp/dba-test-all-states.log",
		},
	}

	daemon, err := New(cfg)
	if err != nil {
		t.Fatalf("Failed to create daemon: %v", err)
	}

	// Register workspace and add multiple services
	daemon.RegisterWorkspace("ws-test", "/path")
	daemon.UpdateServiceState("ws-test", "web", &ServiceState{Name: "web", Port: 8080})
	daemon.UpdateServiceState("ws-test", "api", &ServiceState{Name: "api", Port: 3000})
	daemon.UpdateServiceState("ws-test", "db", &ServiceState{Name: "db", Port: 5432})

	// Get all service states
	states := daemon.GetAllServiceStates("ws-test")
	if len(states) != 3 {
		t.Errorf("expected 3 service states, got %d", len(states))
	}

	// Verify each service
	if states["web"].Port != 8080 {
		t.Errorf("expected web port 8080, got %d", states["web"].Port)
	}
	if states["api"].Port != 3000 {
		t.Errorf("expected api port 3000, got %d", states["api"].Port)
	}
	if states["db"].Port != 5432 {
		t.Errorf("expected db port 5432, got %d", states["db"].Port)
	}
}

func TestDaemonRemoveServiceState(t *testing.T) {
	cfg := &config.Config{
		Daemon: config.DaemonConfig{
			Socket:  "/tmp/dba-test-remove-state.sock",
			PIDFile: "/tmp/dba-test-remove-state.pid",
			LogFile: "/tmp/dba-test-remove-state.log",
		},
	}

	daemon, err := New(cfg)
	if err != nil {
		t.Fatalf("Failed to create daemon: %v", err)
	}

	// Register workspace and add service
	daemon.RegisterWorkspace("ws-test", "/path")
	daemon.UpdateServiceState("ws-test", "web", &ServiceState{Name: "web", Port: 8080})

	// Verify it exists
	state := daemon.GetServiceState("ws-test", "web")
	if state == nil {
		t.Fatal("expected service state to exist")
	}

	// Remove it
	daemon.RemoveServiceState("ws-test", "web")

	// Verify it's gone
	state = daemon.GetServiceState("ws-test", "web")
	if state != nil {
		t.Errorf("expected service state to be removed, got %+v", state)
	}
}

func TestDaemonSetServiceHealth(t *testing.T) {
	cfg := &config.Config{
		Daemon: config.DaemonConfig{
			Socket:  "/tmp/dba-test-health.sock",
			PIDFile: "/tmp/dba-test-health.pid",
			LogFile: "/tmp/dba-test-health.log",
		},
	}

	daemon, err := New(cfg)
	if err != nil {
		t.Fatalf("Failed to create daemon: %v", err)
	}

	// Register workspace and add service
	daemon.RegisterWorkspace("ws-test", "/path")
	daemon.UpdateServiceState("ws-test", "web", &ServiceState{Name: "web", Port: 8080, Healthy: true})

	// Set health to false with error message
	daemon.SetServiceHealth("ws-test", "web", false, "connection refused")

	// Verify health status
	state := daemon.GetServiceState("ws-test", "web")
	if state.Healthy {
		t.Error("expected healthy to be false")
	}
	if state.Error != "connection refused" {
		t.Errorf("expected error 'connection refused', got '%s'", state.Error)
	}

	// Set health back to true
	daemon.SetServiceHealth("ws-test", "web", true, "")

	state = daemon.GetServiceState("ws-test", "web")
	if !state.Healthy {
		t.Error("expected healthy to be true")
	}
	if state.Error != "" {
		t.Errorf("expected empty error, got '%s'", state.Error)
	}
}

func TestDaemonIncrementServiceRestarts(t *testing.T) {
	cfg := &config.Config{
		Daemon: config.DaemonConfig{
			Socket:  "/tmp/dba-test-restarts.sock",
			PIDFile: "/tmp/dba-test-restarts.pid",
			LogFile: "/tmp/dba-test-restarts.log",
		},
	}

	daemon, err := New(cfg)
	if err != nil {
		t.Fatalf("Failed to create daemon: %v", err)
	}

	// Register workspace and add service with 0 restarts
	daemon.RegisterWorkspace("ws-test", "/path")
	daemon.UpdateServiceState("ws-test", "web", &ServiceState{Name: "web", Restarts: 0})

	// Increment restarts
	daemon.IncrementServiceRestarts("ws-test", "web")
	daemon.IncrementServiceRestarts("ws-test", "web")
	daemon.IncrementServiceRestarts("ws-test", "web")

	// Verify restart count
	state := daemon.GetServiceState("ws-test", "web")
	if state.Restarts != 3 {
		t.Errorf("expected 3 restarts, got %d", state.Restarts)
	}
}

func TestDaemonGetWorkspaceServicesSummary(t *testing.T) {
	cfg := &config.Config{
		Daemon: config.DaemonConfig{
			Socket:  "/tmp/dba-test-summary.sock",
			PIDFile: "/tmp/dba-test-summary.pid",
			LogFile: "/tmp/dba-test-summary.log",
		},
	}

	daemon, err := New(cfg)
	if err != nil {
		t.Fatalf("Failed to create daemon: %v", err)
	}

	// Register workspace and add services with various states
	daemon.RegisterWorkspace("ws-test", "/path")
	daemon.UpdateServiceState("ws-test", "web", &ServiceState{Name: "web", Status: "running", Healthy: true})
	daemon.UpdateServiceState("ws-test", "api", &ServiceState{Name: "api", Status: "running", Healthy: true})
	daemon.UpdateServiceState("ws-test", "db", &ServiceState{Name: "db", Status: "running", Healthy: false}) // unhealthy
	daemon.UpdateServiceState("ws-test", "cache", &ServiceState{Name: "cache", Status: "stopped"})

	// Get summary
	running, stopped, unhealthy := daemon.GetWorkspaceServicesSummary("ws-test")

	if running != 3 {
		t.Errorf("expected 3 running services, got %d", running)
	}
	if stopped != 1 {
		t.Errorf("expected 1 stopped service, got %d", stopped)
	}
	if unhealthy != 1 {
		t.Errorf("expected 1 unhealthy service, got %d", unhealthy)
	}
}

func TestStopAllServicesCallback(t *testing.T) {
	cfg := &config.Config{
		Daemon: config.DaemonConfig{
			Socket:  "/tmp/dba-test-callback.sock",
			PIDFile: "/tmp/dba-test-callback.pid",
			LogFile: "/tmp/dba-test-callback.log",
		},
	}

	daemon, err := New(cfg)
	if err != nil {
		t.Fatalf("Failed to create daemon: %v", err)
	}

	// Register workspace and add running services
	daemon.RegisterWorkspace("ws-test", "/path")
	daemon.UpdateServiceState("ws-test", "web", &ServiceState{Name: "web", Status: "running"})
	daemon.UpdateServiceState("ws-test", "api", &ServiceState{Name: "api", Status: "running"})

	// Get the callback
	callback := daemon.StopAllServicesCallback()

	// Execute the callback
	err = callback()
	if err != nil {
		t.Errorf("expected no error from callback, got %v", err)
	}

	// Verify services are marked as stopping
	state := daemon.GetServiceState("ws-test", "web")
	if state.Status != "stopping" {
		t.Errorf("expected status 'stopping', got '%s'", state.Status)
	}

	state = daemon.GetServiceState("ws-test", "api")
	if state.Status != "stopping" {
		t.Errorf("expected status 'stopping', got '%s'", state.Status)
	}
}
