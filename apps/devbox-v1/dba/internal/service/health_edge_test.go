// internal/service/health_edge_test.go
package service

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/dba-cli/dba/internal/workspace"
)

// ═══════════════════════════════════════════════════════════════════════════════
// Health Checking Edge Cases
// ═══════════════════════════════════════════════════════════════════════════════

func TestCheckServiceHealthWithNilWorkspace(t *testing.T) {
	mgr := NewManager(nil)
	ctx := context.Background()

	defer func() {
		if r := recover(); r != nil {
			t.Errorf("CheckServiceHealth panicked with nil workspace: %v", r)
		}
	}()

	healthy, err := mgr.CheckServiceHealth(ctx, "web")
	if err == nil {
		t.Error("CheckServiceHealth should return error for nil workspace")
	}
	if healthy {
		t.Error("CheckServiceHealth should return false for nil workspace")
	}
}

func TestWaitForServiceWithNilWorkspace(t *testing.T) {
	mgr := NewManager(nil)
	ctx := context.Background()

	defer func() {
		if r := recover(); r != nil {
			t.Errorf("WaitForService panicked with nil workspace: %v", r)
		}
	}()

	err := mgr.WaitForService(ctx, "web", 100*time.Millisecond)
	if err == nil {
		t.Error("WaitForService should return error for nil workspace")
	}
}

func TestWaitForHealthyWithNilWorkspace(t *testing.T) {
	mgr := NewManager(nil)
	ctx := context.Background()

	defer func() {
		if r := recover(); r != nil {
			t.Errorf("WaitForHealthy panicked with nil workspace: %v", r)
		}
	}()

	err := mgr.WaitForHealthy(ctx, []string{"web"}, 100*time.Millisecond)
	// Should handle gracefully
	_ = err
}

func TestIsServiceRunningWithNilWorkspace(t *testing.T) {
	mgr := NewManager(nil)
	ctx := context.Background()

	defer func() {
		if r := recover(); r != nil {
			t.Errorf("IsServiceRunning panicked with nil workspace: %v", r)
		}
	}()

	running, err := mgr.IsServiceRunning(ctx, "web")
	if running {
		t.Error("IsServiceRunning should return false for nil workspace")
	}
	_ = err
}

func TestCheckAllHealthWithNilWorkspace(t *testing.T) {
	mgr := NewManager(nil)
	ctx := context.Background()

	defer func() {
		if r := recover(); r != nil {
			t.Errorf("CheckAllHealth panicked with nil workspace: %v", r)
		}
	}()

	results, err := mgr.CheckAllHealth(ctx)
	// Should handle gracefully
	_ = results
	_ = err
}

// ═══════════════════════════════════════════════════════════════════════════════
// Concurrent Health Check Tests
// ═══════════════════════════════════════════════════════════════════════════════

func TestConcurrentCheckServiceHealth(t *testing.T) {
	ws := mockWorkspace()
	mgr := NewManager(ws)
	ctx := context.Background()

	var wg sync.WaitGroup
	services := []string{"web", "api", "vscode", "db"}

	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			svc := services[idx%len(services)]
			_, _ = mgr.CheckServiceHealth(ctx, svc)
		}(i)
	}

	wg.Wait()
}

func TestConcurrentIsServiceRunning(t *testing.T) {
	ws := mockWorkspace()
	ws.Path = "/nonexistent/path"
	mgr := NewManager(ws)
	ctx := context.Background()

	var wg sync.WaitGroup

	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, _ = mgr.IsServiceRunning(ctx, "web")
		}()
	}

	wg.Wait()
}

// ═══════════════════════════════════════════════════════════════════════════════
// Timeout Edge Cases
// ═══════════════════════════════════════════════════════════════════════════════

func TestWaitForHealthyWithVeryShortTimeout(t *testing.T) {
	ws := mockWorkspace()
	ws.Path = "/nonexistent/path"
	mgr := NewManager(ws)
	ctx := context.Background()

	start := time.Now()
	err := mgr.WaitForHealthy(ctx, []string{"web"}, 1*time.Nanosecond)
	elapsed := time.Since(start)

	// Should return quickly
	if elapsed > 1*time.Second {
		t.Errorf("WaitForHealthy with tiny timeout took too long: %v", elapsed)
	}
	// Error is expected
	_ = err
}

func TestWaitForHealthyWithNegativeTimeout(t *testing.T) {
	ws := mockWorkspace()
	ws.Path = "/nonexistent/path"
	mgr := NewManager(ws)
	ctx := context.Background()

	// Negative timeout - should use default or return immediately
	err := mgr.WaitForHealthy(ctx, nil, -1*time.Second)
	// Should handle gracefully
	_ = err
}

func TestWaitForServiceWithVeryShortTimeout(t *testing.T) {
	ws := mockWorkspace()
	mgr := NewManager(ws)
	ctx := context.Background()

	start := time.Now()
	err := mgr.WaitForService(ctx, "web", 1*time.Nanosecond)
	elapsed := time.Since(start)

	// Should return quickly with error
	if elapsed > 1*time.Second {
		t.Errorf("WaitForService with tiny timeout took too long: %v", elapsed)
	}
	if err == nil {
		t.Log("Service was already available")
	}
}

// ═══════════════════════════════════════════════════════════════════════════════
// HealthCheckResult Edge Cases
// ═══════════════════════════════════════════════════════════════════════════════

func TestHealthCheckResultZeroValues(t *testing.T) {
	result := HealthCheckResult{}

	if result.Service != "" {
		t.Error("Zero HealthCheckResult should have empty Service")
	}
	if result.Healthy {
		t.Error("Zero HealthCheckResult should not be Healthy")
	}
	if result.Port != 0 {
		t.Error("Zero HealthCheckResult should have 0 Port")
	}
	if result.Message != "" {
		t.Error("Zero HealthCheckResult should have empty Message")
	}
	if result.CheckedAt != "" {
		t.Error("Zero HealthCheckResult should have empty CheckedAt")
	}
}

func TestHealthCheckResultWithAllFields(t *testing.T) {
	result := HealthCheckResult{
		Service:   "test-service",
		Healthy:   true,
		Port:      8080,
		Message:   "Service is healthy and responding",
		CheckedAt: time.Now().Format(time.RFC3339),
	}

	if result.Service != "test-service" {
		t.Errorf("Service = %q, want %q", result.Service, "test-service")
	}
	if !result.Healthy {
		t.Error("Healthy should be true")
	}
	if result.Port != 8080 {
		t.Errorf("Port = %d, want %d", result.Port, 8080)
	}
}

// ═══════════════════════════════════════════════════════════════════════════════
// Service Health Check for Different Service Types
// ═══════════════════════════════════════════════════════════════════════════════

func TestCheckServiceHealthForAllKnownServices(t *testing.T) {
	ws := &workspace.Workspace{
		ID:   "ws_test",
		Path: "/tmp/test",
		Ports: map[string]int{
			"PORT":      10000,
			"CODE_PORT": 10080,
			"API_PORT":  10001,
			"DB_PORT":   10002,
		},
	}
	mgr := NewManager(ws)
	ctx := context.Background()

	services := []string{"web", "vscode", "api", "db"}
	for _, svc := range services {
		t.Run(svc, func(t *testing.T) {
			healthy, err := mgr.CheckServiceHealth(ctx, svc)
			if err != nil {
				t.Errorf("CheckServiceHealth(%q) returned unexpected error: %v", svc, err)
			}
			// Services are likely not running, so healthy should be false
			t.Logf("CheckServiceHealth(%q) = %v", svc, healthy)
		})
	}
}

// ═══════════════════════════════════════════════════════════════════════════════
// Empty Service List Edge Cases
// ═══════════════════════════════════════════════════════════════════════════════

func TestWaitForHealthyWithEmptyServiceList(t *testing.T) {
	ws := mockWorkspace()
	ws.Path = "/nonexistent/path"
	mgr := NewManager(ws)
	ctx := context.Background()

	// Empty service list should check all services
	err := mgr.WaitForHealthy(ctx, []string{}, 100*time.Millisecond)
	// Should handle gracefully
	_ = err
}

func TestWaitForHealthyWithNilServiceList(t *testing.T) {
	ws := mockWorkspace()
	ws.Path = "/nonexistent/path"
	mgr := NewManager(ws)
	ctx := context.Background()

	// Nil service list should check all services
	err := mgr.WaitForHealthy(ctx, nil, 100*time.Millisecond)
	// Should handle gracefully
	_ = err
}

// ═══════════════════════════════════════════════════════════════════════════════
// Multiple Context Cancellation Scenarios
// ═══════════════════════════════════════════════════════════════════════════════

func TestWaitForHealthyWithMultipleCancellations(t *testing.T) {
	ws := mockWorkspace()
	ws.Path = "/nonexistent/path"
	mgr := NewManager(ws)

	var wg sync.WaitGroup

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			ctx, cancel := context.WithCancel(context.Background())

			// Cancel after a random short time
			go func() {
				time.Sleep(time.Duration(10+i*5) * time.Millisecond)
				cancel()
			}()

			_ = mgr.WaitForHealthy(ctx, nil, 5*time.Second)
		}()
	}

	wg.Wait()
}

// ═══════════════════════════════════════════════════════════════════════════════
// WaitForAllHealthy Tests
// ═══════════════════════════════════════════════════════════════════════════════

func TestWaitForAllHealthyWithNilWorkspace(t *testing.T) {
	mgr := NewManager(nil)
	ctx := context.Background()

	defer func() {
		if r := recover(); r != nil {
			t.Errorf("WaitForAllHealthy panicked: %v", r)
		}
	}()

	err := mgr.WaitForAllHealthy(ctx, 100*time.Millisecond)
	// Should handle gracefully
	_ = err
}

func TestWaitForAllHealthyDelegation(t *testing.T) {
	ws := mockWorkspace()
	ws.Path = "/nonexistent/path"
	mgr := NewManager(ws)

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	// WaitForAllHealthy should delegate to WaitForHealthy with nil services
	err := mgr.WaitForAllHealthy(ctx, 50*time.Millisecond)
	// Should complete without panic
	_ = err
}

// ═══════════════════════════════════════════════════════════════════════════════
// Service Name Edge Cases
// ═══════════════════════════════════════════════════════════════════════════════

func TestCheckServiceHealthWithSpecialNames(t *testing.T) {
	ws := mockWorkspace()
	mgr := NewManager(ws)
	ctx := context.Background()

	specialNames := []string{
		"",
		" ",
		"service with spaces",
		"service\twith\ttabs",
		"service\nwith\nnewlines",
		"very-long-service-name-that-exceeds-typical-length-limits-for-testing-purposes",
		"unicode-服务",
		"./relative/path",
		"../parent/path",
		"/absolute/path",
	}

	for _, name := range specialNames {
		t.Run(name, func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("CheckServiceHealth panicked for name %q: %v", name, r)
				}
			}()

			healthy, err := mgr.CheckServiceHealth(ctx, name)
			// Should return error for unknown service, not panic
			if err == nil && name != "web" && name != "api" && name != "db" && name != "vscode" {
				t.Logf("CheckServiceHealth(%q) returned nil error (unexpected)", name)
			}
			_ = healthy
		})
	}
}

func TestIsServiceRunningWithSpecialNames(t *testing.T) {
	ws := mockWorkspace()
	ws.Path = "/nonexistent/path"
	mgr := NewManager(ws)
	ctx := context.Background()

	specialNames := []string{"", " ", "nonexistent", "special!@#$%"}

	for _, name := range specialNames {
		t.Run(name, func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("IsServiceRunning panicked for name %q: %v", name, r)
				}
			}()

			running, err := mgr.IsServiceRunning(ctx, name)
			// Should handle gracefully
			_ = running
			_ = err
		})
	}
}
