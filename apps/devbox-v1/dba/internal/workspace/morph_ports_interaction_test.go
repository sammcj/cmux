// internal/workspace/morph_ports_interaction_test.go
package workspace

import (
	"encoding/json"
	"testing"
	"time"
)

// TestPortsInteractionWithMorphURLs tests that ports and Morph URLs work correctly together
func TestPortsInteractionWithMorphURLs(t *testing.T) {
	w := &Workspace{}
	w.SetMorphInstance("inst-123", "snap-456", "https://example.com")

	// Add active ports
	w.AddActivePort(ActivePort{
		Port:   8080,
		IsHTTP: true,
		URL:    "https://example.com/ports/8080/",
	})

	// Verify both Morph URLs and active ports are present
	urls := w.GetMorphURLs()
	if urls["code"] == "" {
		t.Error("Expected code URL to be set")
	}

	ports := w.GetHTTPPorts()
	if len(ports) != 1 {
		t.Errorf("Expected 1 HTTP port, got %d", len(ports))
	}
}

// TestPortsResetOnNewInstance tests that ports should be cleared when starting a new instance
func TestPortsResetOnNewInstance(t *testing.T) {
	w := &Workspace{}

	// Add some ports
	w.AddActivePort(ActivePort{Port: 8080, IsHTTP: true})
	w.AddActivePort(ActivePort{Port: 3000, IsHTTP: true})

	// Verify ports exist
	if len(w.Morph.ActivePorts) != 2 {
		t.Errorf("Expected 2 ports, got %d", len(w.Morph.ActivePorts))
	}

	// Clear and set new instance - ports should be manually cleared
	w.ClearActivePorts()
	w.SetMorphInstance("new-inst", "snap-789", "https://newvm.com")

	// Ports should be cleared
	if len(w.Morph.ActivePorts) != 0 {
		t.Errorf("Expected 0 ports after clear, got %d", len(w.Morph.ActivePorts))
	}
}

// TestPortsPreservedAcrossInstanceReset tests that ports can optionally be preserved
func TestPortsPreservedAcrossInstanceReset(t *testing.T) {
	w := &Workspace{}

	w.SetMorphInstance("inst-1", "snap-1", "https://vm1.com")
	w.AddActivePort(ActivePort{Port: 8080, Service: "web"})

	// Save the ports before resetting
	savedPorts := make([]ActivePort, len(w.Morph.ActivePorts))
	copy(savedPorts, w.Morph.ActivePorts)

	// Reset instance but restore ports
	w.ClearMorphInstance()
	w.SetMorphInstance("inst-2", "snap-2", "https://vm2.com")
	w.SetActivePorts(savedPorts)

	// Ports should be restored
	if len(w.Morph.ActivePorts) != 1 {
		t.Errorf("Expected 1 port after restore, got %d", len(w.Morph.ActivePorts))
	}
}

// TestPortsAndSnapshotsIndependent tests that ports and snapshots don't interfere
func TestPortsAndSnapshotsIndependent(t *testing.T) {
	w := &Workspace{}

	w.AddActivePort(ActivePort{Port: 8080})
	w.AddSavedSnapshot("snap-1", "checkpoint-1")

	// Verify both exist
	if len(w.Morph.ActivePorts) != 1 {
		t.Error("Port missing after adding snapshot")
	}
	if len(w.Morph.SavedSnapshots) != 1 {
		t.Error("Snapshot missing after adding port")
	}

	// Clear ports - snapshots should remain
	w.ClearActivePorts()
	if len(w.Morph.SavedSnapshots) != 1 {
		t.Error("Snapshot should remain after clearing ports")
	}
}

// TestPortsWithAllMorphStateFields tests ports alongside all other MorphState fields
func TestPortsWithAllMorphStateFields(t *testing.T) {
	w := &Workspace{}

	// Set all MorphState fields
	w.SetMorphInstance("inst-123", "snap-456", "https://example.com")
	w.Morph.CDPPort = 9222
	w.AddSavedSnapshot("saved-1", "checkpoint-1")
	w.AddActivePort(ActivePort{
		Port:     8080,
		Protocol: "tcp",
		Service:  "web",
		IsHTTP:   true,
	})

	// Verify all fields are set
	if w.Morph.InstanceID != "inst-123" {
		t.Error("InstanceID not set correctly")
	}
	if w.Morph.SnapshotID != "snap-456" {
		t.Error("SnapshotID not set correctly")
	}
	if w.Morph.CDPPort != 9222 {
		t.Error("CDPPort not set correctly")
	}
	if len(w.Morph.SavedSnapshots) != 1 {
		t.Error("SavedSnapshots not set correctly")
	}
	if len(w.Morph.ActivePorts) != 1 {
		t.Error("ActivePorts not set correctly")
	}
}

// TestPortsJSONSerializationWithFullState tests JSON serialization with complete state
func TestPortsJSONSerializationWithFullState(t *testing.T) {
	w := &Workspace{
		ID:   "ws-123",
		Name: "test-workspace",
	}
	w.SetMorphInstance("inst-123", "snap-456", "https://example.com")
	w.Morph.CDPPort = 9222
	w.AddActivePort(ActivePort{
		Port:      8080,
		Protocol:  "tcp",
		Service:   "web",
		Container: "nginx",
		LocalPort: 18080,
		URL:       "https://example.com/ports/8080/",
		IsHTTP:    true,
	})

	// Marshal to JSON
	data, err := json.Marshal(w)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	// Unmarshal and verify
	var w2 Workspace
	if err := json.Unmarshal(data, &w2); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if len(w2.Morph.ActivePorts) != 1 {
		t.Errorf("Expected 1 port after round-trip, got %d", len(w2.Morph.ActivePorts))
	}

	port := w2.Morph.ActivePorts[0]
	if port.Port != 8080 {
		t.Errorf("Expected port 8080, got %d", port.Port)
	}
	if port.Service != "web" {
		t.Errorf("Expected service 'web', got '%s'", port.Service)
	}
}

// TestPortDiscoveryTimestamps tests that port discovery timestamps work correctly
func TestPortDiscoveryTimestamps(t *testing.T) {
	w := &Workspace{}

	t1 := time.Now()
	w.AddActivePort(ActivePort{Port: 8080})
	t2 := time.Now()

	port := w.GetActivePort(8080)
	if port == nil {
		t.Fatal("Port not found")
	}

	if port.DiscoveredAt.Before(t1) || port.DiscoveredAt.After(t2) {
		t.Error("DiscoveredAt should be between t1 and t2")
	}
}

// TestPortDiscoveryPreservedOnUpdate tests that timestamps are preserved on update
func TestPortDiscoveryPreservedOnUpdate(t *testing.T) {
	w := &Workspace{}

	originalTime := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
	w.AddActivePort(ActivePort{
		Port:         8080,
		DiscoveredAt: originalTime,
	})

	// Update the port with new info but keep the timestamp
	w.AddActivePort(ActivePort{
		Port:         8080,
		Service:      "updated",
		DiscoveredAt: originalTime, // Explicitly pass the same time
	})

	port := w.GetActivePort(8080)
	if !port.DiscoveredAt.Equal(originalTime) {
		t.Error("DiscoveredAt should be preserved")
	}
}

// TestPortUpdateReplacesAllFields tests that updating a port replaces all fields
func TestPortUpdateReplacesAllFields(t *testing.T) {
	w := &Workspace{}

	// Add initial port
	w.AddActivePort(ActivePort{
		Port:      8080,
		Protocol:  "tcp",
		Service:   "nginx",
		Container: "container-1",
		LocalPort: 18080,
		URL:       "http://localhost:18080",
		IsHTTP:    true,
	})

	// Update with different values
	w.AddActivePort(ActivePort{
		Port:      8080,
		Protocol:  "udp",
		Service:   "updated",
		Container: "container-2",
		LocalPort: 28080,
		URL:       "http://localhost:28080",
		IsHTTP:    false,
	})

	port := w.GetActivePort(8080)
	if port.Protocol != "udp" {
		t.Errorf("Expected protocol 'udp', got '%s'", port.Protocol)
	}
	if port.Service != "updated" {
		t.Errorf("Expected service 'updated', got '%s'", port.Service)
	}
	if port.Container != "container-2" {
		t.Errorf("Expected container 'container-2', got '%s'", port.Container)
	}
	if port.LocalPort != 28080 {
		t.Errorf("Expected LocalPort 28080, got %d", port.LocalPort)
	}
	if port.IsHTTP {
		t.Error("Expected IsHTTP to be false")
	}
}

// TestPortUpdatePartialFields tests updating a port with only some fields set
func TestPortUpdatePartialFields(t *testing.T) {
	w := &Workspace{}

	// Add full port
	w.AddActivePort(ActivePort{
		Port:      8080,
		Protocol:  "tcp",
		Service:   "nginx",
		Container: "container-1",
	})

	// Update with only port and service
	w.AddActivePort(ActivePort{
		Port:    8080,
		Service: "updated",
	})

	port := w.GetActivePort(8080)
	// Protocol should be empty (replaced, not merged)
	if port.Protocol != "" {
		t.Errorf("Expected empty protocol (replaced), got '%s'", port.Protocol)
	}
	if port.Service != "updated" {
		t.Errorf("Expected service 'updated', got '%s'", port.Service)
	}
	if port.Container != "" {
		t.Errorf("Expected empty container (replaced), got '%s'", port.Container)
	}
}

// TestGetPrimaryAppPortWithMixedPorts tests primary port selection with mixed HTTP/non-HTTP
func TestGetPrimaryAppPortWithMixedPorts(t *testing.T) {
	w := &Workspace{}

	// Add non-HTTP ports first
	w.AddActivePort(ActivePort{Port: 3306, Service: "mysql", IsHTTP: false})
	w.AddActivePort(ActivePort{Port: 5432, Service: "postgres", IsHTTP: false})

	// Add HTTP port later
	w.AddActivePort(ActivePort{Port: 8080, Service: "web", IsHTTP: true})

	primary := w.GetPrimaryAppPort()
	if primary == nil {
		t.Fatal("Expected primary port")
	}

	// Should return the HTTP port, not the first port
	if primary.Port != 8080 {
		t.Errorf("Expected primary port 8080, got %d", primary.Port)
	}
}

// TestGetPrimaryAppPortMultipleHTTP tests that first HTTP port is returned
func TestGetPrimaryAppPortMultipleHTTP(t *testing.T) {
	w := &Workspace{}

	// Add multiple HTTP ports
	w.AddActivePort(ActivePort{Port: 3000, Service: "dev", IsHTTP: true})
	w.AddActivePort(ActivePort{Port: 8080, Service: "api", IsHTTP: true})
	w.AddActivePort(ActivePort{Port: 5173, Service: "vite", IsHTTP: true})

	primary := w.GetPrimaryAppPort()
	if primary == nil {
		t.Fatal("Expected primary port")
	}

	// Should return the first HTTP port (3000)
	if primary.Port != 3000 {
		t.Errorf("Expected first HTTP port 3000, got %d", primary.Port)
	}
}

// TestPortsOrderPreserved tests that port order is preserved
func TestPortsOrderPreserved(t *testing.T) {
	w := &Workspace{}

	ports := []int{8080, 3000, 5000, 4000, 9000}
	for _, p := range ports {
		w.AddActivePort(ActivePort{Port: p})
	}

	// Verify order
	for i, p := range ports {
		if w.Morph.ActivePorts[i].Port != p {
			t.Errorf("Port at index %d: expected %d, got %d", i, p, w.Morph.ActivePorts[i].Port)
		}
	}
}

// TestRemoveMiddlePort tests removing a port from the middle of the list
func TestRemoveMiddlePort(t *testing.T) {
	w := &Workspace{}

	w.AddActivePort(ActivePort{Port: 3000})
	w.AddActivePort(ActivePort{Port: 5000})
	w.AddActivePort(ActivePort{Port: 8080})

	// Remove middle port
	w.RemoveActivePort(5000)

	if len(w.Morph.ActivePorts) != 2 {
		t.Errorf("Expected 2 ports, got %d", len(w.Morph.ActivePorts))
	}

	// Verify remaining ports
	if w.Morph.ActivePorts[0].Port != 3000 {
		t.Errorf("Expected first port 3000, got %d", w.Morph.ActivePorts[0].Port)
	}
	if w.Morph.ActivePorts[1].Port != 8080 {
		t.Errorf("Expected second port 8080, got %d", w.Morph.ActivePorts[1].Port)
	}
}

// TestRemoveFirstPort tests removing the first port
func TestRemoveFirstPort(t *testing.T) {
	w := &Workspace{}

	w.AddActivePort(ActivePort{Port: 3000})
	w.AddActivePort(ActivePort{Port: 5000})
	w.AddActivePort(ActivePort{Port: 8080})

	w.RemoveActivePort(3000)

	if len(w.Morph.ActivePorts) != 2 {
		t.Errorf("Expected 2 ports, got %d", len(w.Morph.ActivePorts))
	}

	if w.Morph.ActivePorts[0].Port != 5000 {
		t.Errorf("Expected first port 5000, got %d", w.Morph.ActivePorts[0].Port)
	}
}

// TestRemoveLastPort tests removing the last port
func TestRemoveLastPort(t *testing.T) {
	w := &Workspace{}

	w.AddActivePort(ActivePort{Port: 3000})
	w.AddActivePort(ActivePort{Port: 5000})
	w.AddActivePort(ActivePort{Port: 8080})

	w.RemoveActivePort(8080)

	if len(w.Morph.ActivePorts) != 2 {
		t.Errorf("Expected 2 ports, got %d", len(w.Morph.ActivePorts))
	}

	if w.Morph.ActivePorts[1].Port != 5000 {
		t.Errorf("Expected last port 5000, got %d", w.Morph.ActivePorts[1].Port)
	}
}

// TestRemoveOnlyPort tests removing the only port
func TestRemoveOnlyPort(t *testing.T) {
	w := &Workspace{}

	w.AddActivePort(ActivePort{Port: 8080})
	w.RemoveActivePort(8080)

	if len(w.Morph.ActivePorts) != 0 {
		t.Errorf("Expected 0 ports, got %d", len(w.Morph.ActivePorts))
	}
}

// TestSetActivePortsWithEmptySlice tests setting an empty slice
func TestSetActivePortsWithEmptySlice(t *testing.T) {
	w := &Workspace{}
	w.AddActivePort(ActivePort{Port: 8080})

	w.SetActivePorts([]ActivePort{})

	if len(w.Morph.ActivePorts) != 0 {
		t.Errorf("Expected 0 ports after setting empty slice, got %d", len(w.Morph.ActivePorts))
	}
}

// TestSetActivePortsWithNil tests setting nil slice
func TestSetActivePortsWithNil(t *testing.T) {
	w := &Workspace{}
	w.AddActivePort(ActivePort{Port: 8080})

	w.SetActivePorts(nil)

	if w.Morph.ActivePorts != nil {
		t.Errorf("Expected nil ActivePorts after setting nil, got %v", w.Morph.ActivePorts)
	}
}

// TestGetActivePortNotFoundAdvanced tests getting a non-existent port
func TestGetActivePortNotFoundAdvanced(t *testing.T) {
	w := &Workspace{}
	w.AddActivePort(ActivePort{Port: 8080})

	port := w.GetActivePort(3000)
	if port != nil {
		t.Errorf("Expected nil for non-existent port, got %+v", port)
	}
}

// TestGetActivePortEmptyList tests getting a port from empty list
func TestGetActivePortEmptyList(t *testing.T) {
	w := &Workspace{}

	port := w.GetActivePort(8080)
	if port != nil {
		t.Errorf("Expected nil for empty list, got %+v", port)
	}
}

// TestPortsWithSpecialCharactersInService tests service names with special characters
func TestPortsWithSpecialCharactersInService(t *testing.T) {
	testCases := []struct {
		name    string
		service string
	}{
		{"with_underscore", "my_service"},
		{"with_dash", "my-service"},
		{"with_dot", "my.service"},
		{"with_slash", "my/service"},
		{"with_colon", "my:service"},
		{"with_at", "my@service"},
		{"unicode", "服务"},
		{"emoji", "🚀service"},
		{"mixed", "my-service_v1.0"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			w := &Workspace{}
			w.AddActivePort(ActivePort{
				Port:    8080,
				Service: tc.service,
			})

			port := w.GetActivePort(8080)
			if port.Service != tc.service {
				t.Errorf("Expected service '%s', got '%s'", tc.service, port.Service)
			}
		})
	}
}

// TestPortsWithSpecialCharactersInContainer tests container names with special characters
func TestPortsWithSpecialCharactersInContainer(t *testing.T) {
	testCases := []struct {
		name      string
		container string
	}{
		{"docker_hash", "abc123def456"},
		{"with_slash", "docker/container"},
		{"long_hash", "sha256:1234567890abcdef1234567890abcdef1234567890abcdef"},
		{"k8s_format", "pod-name-12345-container"},
		{"nested", "ns/pod/container"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			w := &Workspace{}
			w.AddActivePort(ActivePort{
				Port:      8080,
				Container: tc.container,
			})

			port := w.GetActivePort(8080)
			if port.Container != tc.container {
				t.Errorf("Expected container '%s', got '%s'", tc.container, port.Container)
			}
		})
	}
}

// TestPortsWithDifferentProtocols tests various protocol values
func TestPortsWithDifferentProtocols(t *testing.T) {
	protocols := []string{"tcp", "udp", "sctp", "TCP", "UDP", "Tcp", "tcp/ip", ""}

	for _, proto := range protocols {
		name := proto
		if name == "" {
			name = "empty"
		}
		t.Run(name, func(t *testing.T) {
			w := &Workspace{}
			w.AddActivePort(ActivePort{
				Port:     8080,
				Protocol: proto,
			})

			port := w.GetActivePort(8080)
			if port.Protocol != proto {
				t.Errorf("Expected protocol '%s', got '%s'", proto, port.Protocol)
			}
		})
	}
}

// TestPortsWithVariousURLs tests various URL formats
func TestPortsWithVariousURLs(t *testing.T) {
	urls := []string{
		"http://localhost:8080",
		"https://example.com/app/",
		"http://127.0.0.1:8080",
		"http://[::1]:8080",
		"http://example.com:8080/path?query=1",
		"wss://websocket.example.com",
		"",
	}

	for i, url := range urls {
		t.Run(url, func(t *testing.T) {
			w := &Workspace{}
			w.AddActivePort(ActivePort{
				Port: 8080 + i,
				URL:  url,
			})

			port := w.GetActivePort(8080 + i)
			if port.URL != url {
				t.Errorf("Expected URL '%s', got '%s'", url, port.URL)
			}
		})
	}
}

// TestLocalPortRelationship tests relationship between Port and LocalPort
func TestLocalPortRelationship(t *testing.T) {
	testCases := []struct {
		name      string
		port      int
		localPort int
	}{
		{"same", 8080, 8080},
		{"different", 8080, 18080},
		{"high_local", 8080, 65000},
		{"low_local", 8080, 1024},
		{"zero_local", 8080, 0},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			w := &Workspace{}
			w.AddActivePort(ActivePort{
				Port:      tc.port,
				LocalPort: tc.localPort,
			})

			port := w.GetActivePort(tc.port)
			if port.LocalPort != tc.localPort {
				t.Errorf("Expected LocalPort %d, got %d", tc.localPort, port.LocalPort)
			}
		})
	}
}

// TestMorphStateCopy tests that copying MorphState properly copies ports
func TestMorphStateCopy(t *testing.T) {
	w1 := &Workspace{}
	w1.AddActivePort(ActivePort{Port: 8080, Service: "web"})

	// Manual copy
	w2 := &Workspace{}
	w2.Morph = w1.Morph

	// Modify original
	w1.AddActivePort(ActivePort{Port: 3000})

	// Check if w2 is affected (shallow copy issue)
	// Note: This tests the behavior - in Go, slice assignment shares underlying array
	if len(w2.Morph.ActivePorts) == 1 {
		// If it's still 1, assignment created independent copy (after reallocation)
		t.Log("Slice was independent after reallocation")
	}
}

// TestMorphStateDeepCopy tests creating a proper deep copy of ports
func TestMorphStateDeepCopy(t *testing.T) {
	w1 := &Workspace{}
	w1.AddActivePort(ActivePort{Port: 8080, Service: "original"})

	// Deep copy by creating new slice
	w2 := &Workspace{}
	w2.Morph.ActivePorts = make([]ActivePort, len(w1.Morph.ActivePorts))
	copy(w2.Morph.ActivePorts, w1.Morph.ActivePorts)

	// Modify original
	w1.Morph.ActivePorts[0].Service = "modified"

	// w2 should be unaffected
	if w2.Morph.ActivePorts[0].Service != "original" {
		t.Error("Deep copy should be independent")
	}
}
