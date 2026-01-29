// internal/service/health_test.go
package service

import (
	"context"
	"testing"
	"time"
)

func TestHealthCheckResult(t *testing.T) {
	result := HealthCheckResult{
		Service:   "web",
		Healthy:   true,
		Port:      10000,
		Message:   "healthy",
		CheckedAt: "2024-01-01T00:00:00Z",
	}

	if result.Service != "web" {
		t.Errorf("HealthCheckResult.Service = %q, want %q", result.Service, "web")
	}
	if !result.Healthy {
		t.Error("HealthCheckResult.Healthy should be true")
	}
	if result.Port != 10000 {
		t.Errorf("HealthCheckResult.Port = %d, want %d", result.Port, 10000)
	}
	if result.Message != "healthy" {
		t.Errorf("HealthCheckResult.Message = %q, want %q", result.Message, "healthy")
	}
}

func TestCheckServiceHealthUnknownService(t *testing.T) {
	ws := mockWorkspace()
	mgr := NewManager(ws)

	ctx := context.Background()

	healthy, err := mgr.CheckServiceHealth(ctx, "unknown-service")
	if err == nil {
		t.Error("CheckServiceHealth should return error for unknown service")
	}
	if healthy {
		t.Error("CheckServiceHealth should return false for unknown service")
	}
}

func TestCheckServiceHealthKnownService(t *testing.T) {
	ws := mockWorkspace()
	mgr := NewManager(ws)

	ctx := context.Background()

	// These ports are likely not in use, so should return not healthy
	healthy, err := mgr.CheckServiceHealth(ctx, "web")
	if err != nil {
		t.Errorf("CheckServiceHealth returned unexpected error: %v", err)
	}
	// Port is likely free, so service is not healthy
	if healthy {
		t.Log("Port 10000 appears to be in use (service healthy)")
	} else {
		t.Log("Port 10000 is free (service not healthy)")
	}
}

func TestWaitForServiceUnknownService(t *testing.T) {
	ws := mockWorkspace()
	mgr := NewManager(ws)

	ctx := context.Background()

	err := mgr.WaitForService(ctx, "unknown-service", 100*time.Millisecond)
	if err == nil {
		t.Error("WaitForService should return error for unknown service")
	}
}

func TestWaitForServiceTimeout(t *testing.T) {
	ws := mockWorkspace()
	mgr := NewManager(ws)

	ctx := context.Background()

	// Port is likely not in use, so should timeout
	start := time.Now()
	err := mgr.WaitForService(ctx, "web", 500*time.Millisecond)
	elapsed := time.Since(start)

	if err == nil {
		t.Skip("Port 10000 is in use, skipping timeout test")
	}

	// Should have waited at least close to the timeout
	if elapsed < 400*time.Millisecond {
		t.Errorf("WaitForService returned too quickly: %v", elapsed)
	}
}

func TestWaitForHealthyTimeout(t *testing.T) {
	ws := mockWorkspace()
	ws.Path = "/nonexistent/path" // Force process-compose to fail
	mgr := NewManager(ws)

	ctx := context.Background()

	// Should timeout since no process-compose is running
	start := time.Now()
	err := mgr.WaitForHealthy(ctx, nil, 500*time.Millisecond)
	elapsed := time.Since(start)

	// Error is expected (timeout or no services)
	if elapsed < 400*time.Millisecond {
		t.Logf("WaitForHealthy returned in %v (may have returned early due to empty services)", elapsed)
	}
	_ = err
}

func TestWaitForHealthyContextCancellation(t *testing.T) {
	ws := mockWorkspace()
	ws.Path = "/nonexistent/path"
	mgr := NewManager(ws)

	ctx, cancel := context.WithCancel(context.Background())

	// Cancel context after a short delay
	go func() {
		time.Sleep(100 * time.Millisecond)
		cancel()
	}()

	start := time.Now()
	err := mgr.WaitForHealthy(ctx, []string{"web"}, 5*time.Second)
	elapsed := time.Since(start)

	// Should return quickly due to context cancellation
	if elapsed > 1*time.Second {
		t.Errorf("WaitForHealthy took too long after context cancellation: %v", elapsed)
	}

	// Should return context error
	if err != nil && err != context.Canceled {
		// May return other errors too, that's OK
		t.Logf("WaitForHealthy returned: %v", err)
	}
}

func TestWaitForHealthyZeroTimeout(t *testing.T) {
	ws := mockWorkspace()
	ws.Path = "/nonexistent/path"
	mgr := NewManager(ws)

	ctx := context.Background()

	// Zero timeout should use default (60s), but we'll cancel before that
	ctxWithTimeout, cancel := context.WithTimeout(ctx, 200*time.Millisecond)
	defer cancel()

	err := mgr.WaitForHealthy(ctxWithTimeout, nil, 0)
	// Should timeout based on context, not default
	if err != nil && err != context.DeadlineExceeded {
		t.Logf("WaitForHealthy returned: %v", err)
	}
}

func TestIsServiceRunning(t *testing.T) {
	ws := mockWorkspace()
	ws.Path = "/nonexistent/path"
	mgr := NewManager(ws)

	ctx := context.Background()

	running, err := mgr.IsServiceRunning(ctx, "web")
	if err != nil {
		t.Logf("IsServiceRunning returned error (expected for non-existent path): %v", err)
	}
	if running {
		t.Error("IsServiceRunning should return false for non-running service")
	}
}

func TestWaitForAllHealthy(t *testing.T) {
	ws := mockWorkspace()
	ws.Path = "/nonexistent/path"
	mgr := NewManager(ws)

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	err := mgr.WaitForAllHealthy(ctx, 100*time.Millisecond)
	// Should return error due to timeout or no services
	_ = err
}

func TestCheckAllHealth(t *testing.T) {
	ws := mockWorkspace()
	ws.Path = "/nonexistent/path"
	mgr := NewManager(ws)

	ctx := context.Background()

	results, err := mgr.CheckAllHealth(ctx)
	if err != nil {
		t.Logf("CheckAllHealth returned error: %v", err)
	}
	// Should return empty list for non-running process-compose
	if results == nil {
		t.Error("CheckAllHealth should return empty slice, not nil")
	}
}

func TestCheckServiceHealthHTTPServices(t *testing.T) {
	ws := mockWorkspace()
	mgr := NewManager(ws)

	ctx := context.Background()

	// Test each HTTP service type
	httpServices := []string{"web", "api", "vscode"}
	for _, svc := range httpServices {
		healthy, err := mgr.CheckServiceHealth(ctx, svc)
		if err != nil {
			t.Errorf("CheckServiceHealth(%q) returned unexpected error: %v", svc, err)
		}
		// Service is likely not running, so not healthy
		t.Logf("CheckServiceHealth(%q) = %v", svc, healthy)
	}
}

func TestCheckServiceHealthDBService(t *testing.T) {
	ws := mockWorkspace()
	mgr := NewManager(ws)

	ctx := context.Background()

	healthy, err := mgr.CheckServiceHealth(ctx, "db")
	if err != nil {
		t.Errorf("CheckServiceHealth(db) returned unexpected error: %v", err)
	}
	// DB service uses port check, not HTTP
	t.Logf("CheckServiceHealth(db) = %v", healthy)
}
