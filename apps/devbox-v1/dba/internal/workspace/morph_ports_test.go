// internal/workspace/morph_ports_test.go
package workspace

import (
	"encoding/json"
	"testing"
	"time"
)

// =============================================================================
// ActivePort Type Tests
// =============================================================================

func TestActivePortBasicFields(t *testing.T) {
	port := ActivePort{
		Port:      5173,
		Protocol:  "tcp",
		Service:   "vite",
		Container: "myapp-web",
		LocalPort: 5173,
		URL:       "http://localhost:5173",
		IsHTTP:    true,
	}

	if port.Port != 5173 {
		t.Errorf("Port = %d, want 5173", port.Port)
	}
	if port.Service != "vite" {
		t.Errorf("Service = %s, want vite", port.Service)
	}
	if !port.IsHTTP {
		t.Error("IsHTTP should be true")
	}
}

func TestActivePortJSONSerialization(t *testing.T) {
	original := ActivePort{
		Port:         5173,
		Protocol:     "tcp",
		Service:      "vite",
		Container:    "myapp-web",
		LocalPort:    5173,
		URL:          "http://localhost:5173",
		IsHTTP:       true,
		DiscoveredAt: time.Date(2024, 6, 15, 10, 30, 0, 0, time.UTC),
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}

	var loaded ActivePort
	if err := json.Unmarshal(data, &loaded); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}

	if loaded.Port != original.Port {
		t.Errorf("Port = %d, want %d", loaded.Port, original.Port)
	}
	if loaded.Service != original.Service {
		t.Errorf("Service = %s, want %s", loaded.Service, original.Service)
	}
	if loaded.IsHTTP != original.IsHTTP {
		t.Errorf("IsHTTP = %v, want %v", loaded.IsHTTP, original.IsHTTP)
	}
}

// =============================================================================
// AddActivePort Tests
// =============================================================================

func TestAddActivePort(t *testing.T) {
	w := &Workspace{}

	w.AddActivePort(ActivePort{
		Port:    5173,
		Service: "vite",
		IsHTTP:  true,
	})

	if len(w.Morph.ActivePorts) != 1 {
		t.Errorf("ActivePorts count = %d, want 1", len(w.Morph.ActivePorts))
	}
	if w.Morph.ActivePorts[0].Port != 5173 {
		t.Errorf("Port = %d, want 5173", w.Morph.ActivePorts[0].Port)
	}
}

func TestAddActivePortSetsDiscoveredAt(t *testing.T) {
	w := &Workspace{}

	before := time.Now()
	w.AddActivePort(ActivePort{Port: 8080})
	after := time.Now()

	if w.Morph.ActivePorts[0].DiscoveredAt.Before(before) {
		t.Error("DiscoveredAt should be >= test start time")
	}
	if w.Morph.ActivePorts[0].DiscoveredAt.After(after) {
		t.Error("DiscoveredAt should be <= test end time")
	}
}

func TestAddActivePortUpdatesExisting(t *testing.T) {
	w := &Workspace{}

	// Add initial port
	w.AddActivePort(ActivePort{
		Port:    5173,
		Service: "vite",
		IsHTTP:  false,
	})

	// Update same port
	w.AddActivePort(ActivePort{
		Port:    5173,
		Service: "vite-updated",
		IsHTTP:  true,
	})

	if len(w.Morph.ActivePorts) != 1 {
		t.Errorf("ActivePorts count = %d, want 1 (should update, not add)", len(w.Morph.ActivePorts))
	}
	if w.Morph.ActivePorts[0].Service != "vite-updated" {
		t.Errorf("Service = %s, want vite-updated", w.Morph.ActivePorts[0].Service)
	}
	if !w.Morph.ActivePorts[0].IsHTTP {
		t.Error("IsHTTP should be true after update")
	}
}

func TestAddMultipleActivePorts(t *testing.T) {
	w := &Workspace{}

	w.AddActivePort(ActivePort{Port: 5173, Service: "vite"})
	w.AddActivePort(ActivePort{Port: 5432, Service: "postgres"})
	w.AddActivePort(ActivePort{Port: 6379, Service: "redis"})

	if len(w.Morph.ActivePorts) != 3 {
		t.Errorf("ActivePorts count = %d, want 3", len(w.Morph.ActivePorts))
	}
}

// =============================================================================
// RemoveActivePort Tests
// =============================================================================

func TestRemoveActivePort(t *testing.T) {
	w := &Workspace{}

	w.AddActivePort(ActivePort{Port: 5173, Service: "vite"})
	w.AddActivePort(ActivePort{Port: 5432, Service: "postgres"})

	w.RemoveActivePort(5173)

	if len(w.Morph.ActivePorts) != 1 {
		t.Errorf("ActivePorts count = %d, want 1", len(w.Morph.ActivePorts))
	}
	if w.Morph.ActivePorts[0].Port != 5432 {
		t.Errorf("Remaining port = %d, want 5432", w.Morph.ActivePorts[0].Port)
	}
}

func TestRemoveActivePortNonExistent(t *testing.T) {
	w := &Workspace{}

	w.AddActivePort(ActivePort{Port: 5173, Service: "vite"})

	// Remove non-existent port should not panic
	w.RemoveActivePort(9999)

	if len(w.Morph.ActivePorts) != 1 {
		t.Errorf("ActivePorts count = %d, want 1", len(w.Morph.ActivePorts))
	}
}

func TestRemoveActivePortFromEmpty(t *testing.T) {
	w := &Workspace{}

	// Should not panic
	w.RemoveActivePort(5173)

	if len(w.Morph.ActivePorts) != 0 {
		t.Errorf("ActivePorts count = %d, want 0", len(w.Morph.ActivePorts))
	}
}

// =============================================================================
// GetActivePort Tests
// =============================================================================

func TestGetActivePort(t *testing.T) {
	w := &Workspace{}

	w.AddActivePort(ActivePort{Port: 5173, Service: "vite"})
	w.AddActivePort(ActivePort{Port: 5432, Service: "postgres"})

	port := w.GetActivePort(5173)
	if port == nil {
		t.Fatal("GetActivePort returned nil")
	}
	if port.Service != "vite" {
		t.Errorf("Service = %s, want vite", port.Service)
	}
}

func TestGetActivePortNotFound(t *testing.T) {
	w := &Workspace{}

	w.AddActivePort(ActivePort{Port: 5173, Service: "vite"})

	port := w.GetActivePort(9999)
	if port != nil {
		t.Error("GetActivePort should return nil for non-existent port")
	}
}

func TestGetActivePortFromEmpty(t *testing.T) {
	w := &Workspace{}

	port := w.GetActivePort(5173)
	if port != nil {
		t.Error("GetActivePort should return nil for empty list")
	}
}

// =============================================================================
// GetHTTPPorts Tests
// =============================================================================

func TestGetHTTPPorts(t *testing.T) {
	w := &Workspace{}

	w.AddActivePort(ActivePort{Port: 5173, Service: "vite", IsHTTP: true})
	w.AddActivePort(ActivePort{Port: 8080, Service: "api", IsHTTP: true})
	w.AddActivePort(ActivePort{Port: 5432, Service: "postgres", IsHTTP: false})
	w.AddActivePort(ActivePort{Port: 6379, Service: "redis", IsHTTP: false})

	httpPorts := w.GetHTTPPorts()

	if len(httpPorts) != 2 {
		t.Errorf("HTTP ports count = %d, want 2", len(httpPorts))
	}
}

func TestGetHTTPPortsNone(t *testing.T) {
	w := &Workspace{}

	w.AddActivePort(ActivePort{Port: 5432, Service: "postgres", IsHTTP: false})

	httpPorts := w.GetHTTPPorts()

	if len(httpPorts) != 0 {
		t.Errorf("HTTP ports count = %d, want 0", len(httpPorts))
	}
}

func TestGetHTTPPortsEmpty(t *testing.T) {
	w := &Workspace{}

	httpPorts := w.GetHTTPPorts()

	if httpPorts != nil && len(httpPorts) != 0 {
		t.Errorf("HTTP ports should be empty")
	}
}

// =============================================================================
// GetPrimaryAppPort Tests
// =============================================================================

func TestGetPrimaryAppPort(t *testing.T) {
	w := &Workspace{}

	w.AddActivePort(ActivePort{Port: 5432, Service: "postgres", IsHTTP: false})
	w.AddActivePort(ActivePort{Port: 5173, Service: "vite", IsHTTP: true})
	w.AddActivePort(ActivePort{Port: 8080, Service: "api", IsHTTP: true})

	port := w.GetPrimaryAppPort()
	if port == nil {
		t.Fatal("GetPrimaryAppPort returned nil")
	}
	// Should return first HTTP port (5173)
	if port.Port != 5173 {
		t.Errorf("Primary port = %d, want 5173 (first HTTP)", port.Port)
	}
}

func TestGetPrimaryAppPortNoHTTP(t *testing.T) {
	w := &Workspace{}

	w.AddActivePort(ActivePort{Port: 5432, Service: "postgres", IsHTTP: false})
	w.AddActivePort(ActivePort{Port: 6379, Service: "redis", IsHTTP: false})

	port := w.GetPrimaryAppPort()
	if port == nil {
		t.Fatal("GetPrimaryAppPort returned nil")
	}
	// Should return first port (5432)
	if port.Port != 5432 {
		t.Errorf("Primary port = %d, want 5432 (first port)", port.Port)
	}
}

func TestGetPrimaryAppPortEmpty(t *testing.T) {
	w := &Workspace{}

	port := w.GetPrimaryAppPort()
	if port != nil {
		t.Error("GetPrimaryAppPort should return nil for empty list")
	}
}

// =============================================================================
// ClearActivePorts Tests
// =============================================================================

func TestClearActivePorts(t *testing.T) {
	w := &Workspace{}

	w.AddActivePort(ActivePort{Port: 5173})
	w.AddActivePort(ActivePort{Port: 5432})
	w.AddActivePort(ActivePort{Port: 6379})

	w.ClearActivePorts()

	if len(w.Morph.ActivePorts) != 0 {
		t.Errorf("ActivePorts count = %d, want 0", len(w.Morph.ActivePorts))
	}
}

func TestClearActivePortsEmpty(t *testing.T) {
	w := &Workspace{}

	// Should not panic
	w.ClearActivePorts()

	if w.Morph.ActivePorts != nil && len(w.Morph.ActivePorts) != 0 {
		t.Error("ActivePorts should be empty")
	}
}

// =============================================================================
// SetActivePorts Tests
// =============================================================================

func TestSetActivePorts(t *testing.T) {
	w := &Workspace{}

	w.AddActivePort(ActivePort{Port: 9999}) // Should be replaced

	ports := []ActivePort{
		{Port: 5173, Service: "vite"},
		{Port: 5432, Service: "postgres"},
	}
	w.SetActivePorts(ports)

	if len(w.Morph.ActivePorts) != 2 {
		t.Errorf("ActivePorts count = %d, want 2", len(w.Morph.ActivePorts))
	}
	if w.Morph.ActivePorts[0].Port != 5173 {
		t.Errorf("First port = %d, want 5173", w.Morph.ActivePorts[0].Port)
	}
}

func TestSetActivePortsSetsDiscoveredAt(t *testing.T) {
	w := &Workspace{}

	before := time.Now()
	ports := []ActivePort{
		{Port: 5173, Service: "vite"},
	}
	w.SetActivePorts(ports)
	after := time.Now()

	if w.Morph.ActivePorts[0].DiscoveredAt.Before(before) {
		t.Error("DiscoveredAt should be >= test start time")
	}
	if w.Morph.ActivePorts[0].DiscoveredAt.After(after) {
		t.Error("DiscoveredAt should be <= test end time")
	}
}

func TestSetActivePortsPreservesExistingDiscoveredAt(t *testing.T) {
	w := &Workspace{}

	existingTime := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	ports := []ActivePort{
		{Port: 5173, Service: "vite", DiscoveredAt: existingTime},
	}
	w.SetActivePorts(ports)

	if !w.Morph.ActivePorts[0].DiscoveredAt.Equal(existingTime) {
		t.Errorf("DiscoveredAt = %v, want %v", w.Morph.ActivePorts[0].DiscoveredAt, existingTime)
	}
}

// =============================================================================
// Integration Tests
// =============================================================================

func TestActivePortsWithMorphState(t *testing.T) {
	w := &Workspace{}

	// Set up Morph instance
	w.SetMorphInstance("inst-123", "snap-456", "https://example.com")

	// Add active ports
	w.AddActivePort(ActivePort{
		Port:      5173,
		Service:   "vite",
		Container: "myapp-web",
		IsHTTP:    true,
		URL:       "http://localhost:5173",
	})
	w.AddActivePort(ActivePort{
		Port:      5432,
		Service:   "postgres",
		Container: "myapp-db",
		IsHTTP:    false,
	})

	// Verify state
	if len(w.Morph.ActivePorts) != 2 {
		t.Errorf("ActivePorts count = %d, want 2", len(w.Morph.ActivePorts))
	}
	if w.Morph.Status != "running" {
		t.Errorf("Status = %s, want running", w.Morph.Status)
	}

	// Get primary app port
	appPort := w.GetPrimaryAppPort()
	if appPort == nil {
		t.Fatal("GetPrimaryAppPort returned nil")
	}
	if appPort.URL != "http://localhost:5173" {
		t.Errorf("App URL = %s, want http://localhost:5173", appPort.URL)
	}
}

func TestActivePortsAfterClearMorphInstance(t *testing.T) {
	w := &Workspace{}

	w.SetMorphInstance("inst-123", "snap-456", "https://example.com")
	w.AddActivePort(ActivePort{Port: 5173})

	w.ClearMorphInstance()

	// ActivePorts are NOT cleared by ClearMorphInstance
	// (they may be needed for reference or reconnection)
	if len(w.Morph.ActivePorts) != 1 {
		t.Errorf("ActivePorts count = %d, want 1 (should be preserved)", len(w.Morph.ActivePorts))
	}
}

func TestActivePortsJSONRoundTrip(t *testing.T) {
	original := &Workspace{
		ID:   "ws-ports",
		Name: "ports-test",
		Morph: MorphState{
			InstanceID: "inst-123",
			Status:     "running",
			ActivePorts: []ActivePort{
				{Port: 5173, Service: "vite", IsHTTP: true},
				{Port: 5432, Service: "postgres", IsHTTP: false},
			},
		},
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}

	var loaded Workspace
	if err := json.Unmarshal(data, &loaded); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}

	if len(loaded.Morph.ActivePorts) != 2 {
		t.Errorf("ActivePorts count = %d, want 2", len(loaded.Morph.ActivePorts))
	}
	if loaded.Morph.ActivePorts[0].Port != 5173 {
		t.Errorf("First port = %d, want 5173", loaded.Morph.ActivePorts[0].Port)
	}
}

// =============================================================================
// Edge Cases
// =============================================================================

func TestActivePortWithZeroPort(t *testing.T) {
	w := &Workspace{}

	w.AddActivePort(ActivePort{Port: 0, Service: "zero"})

	if len(w.Morph.ActivePorts) != 1 {
		t.Errorf("ActivePorts count = %d, want 1", len(w.Morph.ActivePorts))
	}
}

func TestActivePortWithNegativePort(t *testing.T) {
	w := &Workspace{}

	// Negative ports are technically invalid but shouldn't crash
	w.AddActivePort(ActivePort{Port: -1, Service: "negative"})

	if len(w.Morph.ActivePorts) != 1 {
		t.Errorf("ActivePorts count = %d, want 1", len(w.Morph.ActivePorts))
	}
}

func TestActivePortWithLargePort(t *testing.T) {
	w := &Workspace{}

	w.AddActivePort(ActivePort{Port: 65535, Service: "max"})

	if len(w.Morph.ActivePorts) != 1 {
		t.Errorf("ActivePorts count = %d, want 1", len(w.Morph.ActivePorts))
	}
	if w.Morph.ActivePorts[0].Port != 65535 {
		t.Errorf("Port = %d, want 65535", w.Morph.ActivePorts[0].Port)
	}
}

func TestActivePortWithSpecialCharactersInService(t *testing.T) {
	services := []string{
		"service-with-dash",
		"service_with_underscore",
		"service.with.dots",
		"service/with/slashes",
		"service:with:colons",
		"日本語サービス",
		"🚀emoji-service",
	}

	w := &Workspace{}

	for i, service := range services {
		w.AddActivePort(ActivePort{Port: 8000 + i, Service: service})
	}

	if len(w.Morph.ActivePorts) != len(services) {
		t.Errorf("ActivePorts count = %d, want %d", len(w.Morph.ActivePorts), len(services))
	}
}

func TestActivePortUpdatePreservesOrder(t *testing.T) {
	w := &Workspace{}

	w.AddActivePort(ActivePort{Port: 1000, Service: "first"})
	w.AddActivePort(ActivePort{Port: 2000, Service: "second"})
	w.AddActivePort(ActivePort{Port: 3000, Service: "third"})

	// Update middle port
	w.AddActivePort(ActivePort{Port: 2000, Service: "second-updated"})

	// Order should be preserved
	if w.Morph.ActivePorts[1].Service != "second-updated" {
		t.Errorf("Middle port service = %s, want second-updated", w.Morph.ActivePorts[1].Service)
	}
	if w.Morph.ActivePorts[0].Port != 1000 || w.Morph.ActivePorts[2].Port != 3000 {
		t.Error("Order was not preserved after update")
	}
}
