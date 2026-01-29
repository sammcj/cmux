// internal/daemon/health_test.go
package daemon

import (
	"os"
	"path/filepath"
	"strconv"
	"testing"
)

func TestProcessHealthCheckerMissingPIDFile(t *testing.T) {
	checker := NewProcessHealthChecker("test", "/tmp/nonexistent-pid-file-that-does-not-exist")

	if checker.IsHealthy() {
		t.Fatalf("expected initial health to be false")
	}

	// Check should return error when PID file doesn't exist
	if err := checker.Check(); err == nil {
		t.Fatalf("expected error for missing PID file, got nil")
	}

	if checker.IsHealthy() {
		t.Fatalf("expected health to be false after failed Check")
	}
}

func TestProcessHealthCheckerValidPID(t *testing.T) {
	// Create a temp PID file with the current process PID
	tmpDir := t.TempDir()
	pidFile := filepath.Join(tmpDir, "test.pid")

	// Write current process PID
	currentPID := os.Getpid()
	err := os.WriteFile(pidFile, []byte(strconv.Itoa(currentPID)), 0644)
	if err != nil {
		t.Fatalf("Failed to write PID file: %v", err)
	}

	checker := NewProcessHealthChecker("test", pidFile)

	if checker.IsHealthy() {
		t.Fatalf("expected initial health to be false")
	}

	// Check should succeed for valid PID
	if err := checker.Check(); err != nil {
		t.Fatalf("expected no error for valid PID, got %v", err)
	}

	if !checker.IsHealthy() {
		t.Fatalf("expected health to be true after successful Check")
	}
}
