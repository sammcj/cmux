// internal/daemon/process_health_test.go
// Tests for ProcessHealthChecker - Added by Agent #6

package daemon

import (
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/dba-cli/dba/internal/config"
)

func TestProcessHealthCheckerWithValidPID(t *testing.T) {
	// Create a temp PID file with the current process PID
	tmpDir := t.TempDir()
	pidFile := filepath.Join(tmpDir, "test.pid")

	// Write current process PID
	currentPID := os.Getpid()
	err := os.WriteFile(pidFile, []byte(strconv.Itoa(currentPID)), 0644)
	if err != nil {
		t.Fatalf("Failed to write PID file: %v", err)
	}

	checker := NewProcessHealthChecker("test-process", pidFile)
	err = checker.Check()
	if err != nil {
		t.Errorf("expected no error for current process, got %v", err)
	}

	if !checker.IsHealthy() {
		t.Error("expected checker to report healthy for current process")
	}

	if checker.GetLastPID() != currentPID {
		t.Errorf("expected last PID %d, got %d", currentPID, checker.GetLastPID())
	}
}

func TestProcessHealthCheckerWithNonExistentPID(t *testing.T) {
	// Create a temp PID file with a non-existent PID
	tmpDir := t.TempDir()
	pidFile := filepath.Join(tmpDir, "test.pid")

	// Use a very high PID that's unlikely to exist
	fakePID := 999999
	err := os.WriteFile(pidFile, []byte(strconv.Itoa(fakePID)), 0644)
	if err != nil {
		t.Fatalf("Failed to write PID file: %v", err)
	}

	checker := NewProcessHealthChecker("test-process", pidFile)
	err = checker.Check()
	if err == nil {
		t.Error("expected error for non-existent process")
	}

	if checker.IsHealthy() {
		t.Error("expected checker to report unhealthy for non-existent process")
	}
}

func TestProcessHealthCheckerWithMissingPIDFile(t *testing.T) {
	checker := NewProcessHealthChecker("test-process", "/nonexistent/path/test.pid")
	err := checker.Check()
	if err == nil {
		t.Error("expected error for missing PID file")
	}

	if checker.IsHealthy() {
		t.Error("expected checker to report unhealthy for missing PID file")
	}
}

func TestProcessHealthCheckerWithEmptyPIDFile(t *testing.T) {
	tmpDir := t.TempDir()
	pidFile := filepath.Join(tmpDir, "empty.pid")

	// Write empty file
	err := os.WriteFile(pidFile, []byte(""), 0644)
	if err != nil {
		t.Fatalf("Failed to write PID file: %v", err)
	}

	checker := NewProcessHealthChecker("test-process", pidFile)
	err = checker.Check()
	if err == nil {
		t.Error("expected error for empty PID file")
	}

	if checker.IsHealthy() {
		t.Error("expected checker to report unhealthy for empty PID file")
	}
}

func TestProcessHealthCheckerWithInvalidPIDFormat(t *testing.T) {
	tmpDir := t.TempDir()
	pidFile := filepath.Join(tmpDir, "invalid.pid")

	// Write invalid PID
	err := os.WriteFile(pidFile, []byte("not-a-number"), 0644)
	if err != nil {
		t.Fatalf("Failed to write PID file: %v", err)
	}

	checker := NewProcessHealthChecker("test-process", pidFile)
	err = checker.Check()
	if err == nil {
		t.Error("expected error for invalid PID format")
	}

	if checker.IsHealthy() {
		t.Error("expected checker to report unhealthy for invalid PID")
	}
}

func TestProcessHealthCheckerWithNegativePID(t *testing.T) {
	tmpDir := t.TempDir()
	pidFile := filepath.Join(tmpDir, "negative.pid")

	// Write negative PID
	err := os.WriteFile(pidFile, []byte("-1"), 0644)
	if err != nil {
		t.Fatalf("Failed to write PID file: %v", err)
	}

	checker := NewProcessHealthChecker("test-process", pidFile)
	err = checker.Check()
	if err == nil {
		t.Error("expected error for negative PID")
	}

	if checker.IsHealthy() {
		t.Error("expected checker to report unhealthy for negative PID")
	}
}

func TestProcessHealthCheckerWithWhitespacePID(t *testing.T) {
	tmpDir := t.TempDir()
	pidFile := filepath.Join(tmpDir, "whitespace.pid")

	// Write PID with whitespace
	currentPID := os.Getpid()
	err := os.WriteFile(pidFile, []byte("  \n"+strconv.Itoa(currentPID)+"\n  "), 0644)
	if err != nil {
		t.Fatalf("Failed to write PID file: %v", err)
	}

	checker := NewProcessHealthChecker("test-process", pidFile)
	err = checker.Check()
	if err != nil {
		t.Errorf("expected no error for PID with whitespace, got %v", err)
	}

	if !checker.IsHealthy() {
		t.Error("expected checker to report healthy for valid PID with whitespace")
	}
}

func TestProcessHealthCheckerName(t *testing.T) {
	checker := NewProcessHealthChecker("my-test-process", "/tmp/test.pid")
	if checker.Name() != "my-test-process" {
		t.Errorf("expected name 'my-test-process', got '%s'", checker.Name())
	}
}

func TestServiceHealthChecker(t *testing.T) {
	// Test ServiceHealthChecker struct
	checker := NewServiceHealthChecker("web-health", "ws-123", "web", 8080, nil)

	if checker.Name() != "web-health" {
		t.Errorf("expected name 'web-health', got '%s'", checker.Name())
	}

	// Health check will fail because no server is running on port 8080 (usually)
	// But we can test the error handling
	err := checker.Check()
	// We expect an error because nothing is listening on port 8080
	if err == nil {
		t.Log("Note: Port 8080 is responding (test server might be running)")
	}

	// Test that GetLastError returns something when unhealthy
	if !checker.IsHealthy() && checker.GetLastError() == "" {
		t.Error("expected error message when unhealthy")
	}
}

func TestPortHealthCheckerDetailed(t *testing.T) {
	// Test PortHealthChecker with a port that's unlikely to be in use
	checker := NewPortHealthChecker("test-port", "localhost", 59999, 0)

	if checker.Name() != "test-port" {
		t.Errorf("expected name 'test-port', got '%s'", checker.Name())
	}

	// Check should fail because nothing is listening
	err := checker.Check()
	if err == nil {
		t.Log("Note: Port 59999 is responding unexpectedly")
	}

	if checker.IsHealthy() {
		t.Log("Note: Port checker reports healthy (something might be listening on 59999)")
	}
}

func TestWorkspaceHealthChecker(t *testing.T) {
	cfg := &config.Config{
		Daemon: config.DaemonConfig{
			Socket:  "/tmp/test-ws-health.sock",
			PIDFile: "/tmp/test-ws-health.pid",
			LogFile: "/tmp/test-ws-health.log",
		},
	}

	daemon, err := New(cfg)
	if err != nil {
		t.Fatalf("Failed to create daemon: %v", err)
	}

	// Register workspace
	daemon.RegisterWorkspace("ws-test", "/tmp/test-workspace")

	// Create checker
	checker := NewWorkspaceHealthChecker("ws-health", "ws-test", daemon)

	if checker.Name() != "ws-health" {
		t.Errorf("expected name 'ws-health', got '%s'", checker.Name())
	}

	// Check should pass (workspace was just registered)
	err = checker.Check()
	if err != nil {
		t.Errorf("expected no error for recently registered workspace, got %v", err)
	}

	if !checker.IsHealthy() {
		t.Error("expected checker to be healthy for recently registered workspace")
	}
}

func TestWorkspaceHealthCheckerNotFound(t *testing.T) {
	cfg := &config.Config{
		Daemon: config.DaemonConfig{
			Socket:  "/tmp/test-ws-notfound.sock",
			PIDFile: "/tmp/test-ws-notfound.pid",
			LogFile: "/tmp/test-ws-notfound.log",
		},
	}

	daemon, err := New(cfg)
	if err != nil {
		t.Fatalf("Failed to create daemon: %v", err)
	}

	// Create checker for non-existent workspace
	checker := NewWorkspaceHealthChecker("ws-health", "nonexistent", daemon)

	// Check should fail
	err = checker.Check()
	if err == nil {
		t.Error("expected error for non-existent workspace")
	}

	if checker.IsHealthy() {
		t.Error("expected checker to be unhealthy for non-existent workspace")
	}
}

func TestCompositeHealthCheckerWithProcesses(t *testing.T) {
	tmpDir := t.TempDir()

	// Create two valid PID files
	pidFile1 := filepath.Join(tmpDir, "proc1.pid")
	pidFile2 := filepath.Join(tmpDir, "proc2.pid")

	currentPID := os.Getpid()
	os.WriteFile(pidFile1, []byte(strconv.Itoa(currentPID)), 0644)
	os.WriteFile(pidFile2, []byte(strconv.Itoa(currentPID)), 0644)

	checker1 := NewProcessHealthChecker("proc1", pidFile1)
	checker2 := NewProcessHealthChecker("proc2", pidFile2)

	composite := NewCompositeHealthChecker("all-processes", checker1, checker2)

	if composite.Name() != "all-processes" {
		t.Errorf("expected name 'all-processes', got '%s'", composite.Name())
	}

	err := composite.Check()
	if err != nil {
		t.Errorf("expected no error for all valid processes, got %v", err)
	}

	if !composite.IsHealthy() {
		t.Error("expected composite checker to be healthy when all sub-checkers are healthy")
	}
}

func TestCompositeHealthCheckerWithFailure(t *testing.T) {
	tmpDir := t.TempDir()

	// Create one valid and one invalid PID file
	pidFile1 := filepath.Join(tmpDir, "proc1.pid")
	pidFile2 := filepath.Join(tmpDir, "proc2.pid")

	currentPID := os.Getpid()
	os.WriteFile(pidFile1, []byte(strconv.Itoa(currentPID)), 0644)
	os.WriteFile(pidFile2, []byte("999999"), 0644) // Invalid PID

	checker1 := NewProcessHealthChecker("proc1", pidFile1)
	checker2 := NewProcessHealthChecker("proc2", pidFile2)

	composite := NewCompositeHealthChecker("mixed-processes", checker1, checker2)

	err := composite.Check()
	if err == nil {
		t.Error("expected error when one sub-checker fails")
	}

	if composite.IsHealthy() {
		t.Error("expected composite checker to be unhealthy when one sub-checker fails")
	}
}

