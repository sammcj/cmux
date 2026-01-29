// internal/service/health.go
package service

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/dba-cli/dba/internal/port"
)

// WaitForHealthy waits for services to be healthy
func (m *Manager) WaitForHealthy(ctx context.Context, services []string, timeout time.Duration) error {
	if m.workspace == nil {
		return fmt.Errorf("workspace is nil")
	}
	if timeout == 0 {
		timeout = 60 * time.Second
	}

	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// Get current status
		statuses, err := m.List(ctx, false)
		if err != nil {
			time.Sleep(500 * time.Millisecond)
			continue
		}

		// Check if all requested services are healthy
		allHealthy := true
		serviceSet := make(map[string]bool)
		for _, s := range services {
			serviceSet[s] = true
		}

		for _, status := range statuses {
			// Skip if not in requested set (if set is non-empty)
			if len(serviceSet) > 0 && !serviceSet[status.Name] {
				continue
			}

			if status.Status == "running" && !status.Healthy {
				allHealthy = false
				break
			}
		}

		if allHealthy {
			return nil
		}

		time.Sleep(500 * time.Millisecond)
	}

	return fmt.Errorf("services not healthy within %s", timeout)
}

// CheckServiceHealth checks if a specific service is healthy
func (m *Manager) CheckServiceHealth(ctx context.Context, name string) (bool, error) {
	portNum, ok := m.getServicePort(name)
	if !ok {
		return false, fmt.Errorf("unknown service: %s", name)
	}

	// For HTTP services, try to make a request
	if name == "web" || name == "api" || name == "vscode" {
		client := &http.Client{Timeout: 2 * time.Second}
		resp, err := client.Get(fmt.Sprintf("http://localhost:%d/", portNum))
		if err != nil {
			return false, nil
		}
		resp.Body.Close()
		return resp.StatusCode < 500, nil
	}

	// For other services, check if port is in use
	return !port.IsPortFree(portNum), nil
}

// WaitForService waits for a specific service to be ready
func (m *Manager) WaitForService(ctx context.Context, name string, timeout time.Duration) error {
	portNum, ok := m.getServicePort(name)
	if !ok {
		return fmt.Errorf("unknown service: %s", name)
	}

	return port.WaitForPort(portNum, timeout)
}

// HealthCheckResult represents the result of a health check
type HealthCheckResult struct {
	Service   string `json:"service"`
	Healthy   bool   `json:"healthy"`
	Port      int    `json:"port,omitempty"`
	Message   string `json:"message,omitempty"`
	CheckedAt string `json:"checked_at"`
}

// CheckAllHealth checks the health of all services
func (m *Manager) CheckAllHealth(ctx context.Context) ([]HealthCheckResult, error) {
	if m.workspace == nil {
		return []HealthCheckResult{}, nil
	}
	statuses, err := m.List(ctx, false)
	if err != nil {
		return nil, err
	}

	results := make([]HealthCheckResult, len(statuses))
	for i, status := range statuses {
		result := HealthCheckResult{
			Service:   status.Name,
			Healthy:   status.Healthy,
			Port:      status.Port,
			CheckedAt: time.Now().Format(time.RFC3339),
		}

		if status.Status != "running" {
			result.Message = "service not running"
		} else if !status.Healthy {
			result.Message = "health check failed"
		} else {
			result.Message = "healthy"
		}

		results[i] = result
	}

	return results, nil
}

// IsServiceRunning checks if a service is running
func (m *Manager) IsServiceRunning(ctx context.Context, name string) (bool, error) {
	if m.workspace == nil {
		return false, fmt.Errorf("workspace is nil")
	}
	statuses, err := m.List(ctx, false)
	if err != nil {
		return false, err
	}

	for _, status := range statuses {
		if status.Name == name {
			return status.Status == "running", nil
		}
	}

	return false, nil
}

// WaitForAllHealthy waits for all services to be healthy
func (m *Manager) WaitForAllHealthy(ctx context.Context, timeout time.Duration) error {
	return m.WaitForHealthy(ctx, nil, timeout)
}
