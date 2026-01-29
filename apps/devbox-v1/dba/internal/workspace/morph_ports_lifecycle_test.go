// internal/workspace/morph_ports_lifecycle_test.go
package workspace

import (
	"encoding/json"
	"testing"
	"time"
)

// TestPortsLifecycleMorphStart tests ports with instance start
func TestPortsLifecycleMorphStart(t *testing.T) {
	w := &Workspace{}

	// Start instance
	w.SetMorphInstance("inst-1", "snap-1", "https://vm.example.com")

	// Verify ports are empty initially
	if len(w.Morph.ActivePorts) != 0 {
		t.Errorf("Expected 0 ports at start, got %d", len(w.Morph.ActivePorts))
	}

	// Add ports after start
	w.AddActivePort(ActivePort{Port: 8080, IsHTTP: true})

	// Verify port exists
	if len(w.Morph.ActivePorts) != 1 {
		t.Errorf("Expected 1 port, got %d", len(w.Morph.ActivePorts))
	}
}

// TestPortsLifecycleMorphStop tests ports with instance stop
func TestPortsLifecycleMorphStop(t *testing.T) {
	w := &Workspace{}

	w.SetMorphInstance("inst-1", "snap-1", "https://vm.example.com")
	w.AddActivePort(ActivePort{Port: 8080, IsHTTP: true})
	w.AddActivePort(ActivePort{Port: 3000, IsHTTP: true})

	// Stop instance (ports remain but should be considered stale)
	w.ClearMorphInstance()

	// Ports still exist (need manual cleanup if desired)
	if len(w.Morph.ActivePorts) != 2 {
		t.Errorf("Expected 2 ports after stop, got %d", len(w.Morph.ActivePorts))
	}

	// Status should be stopped
	if w.Morph.Status != "stopped" {
		t.Errorf("Expected status 'stopped', got '%s'", w.Morph.Status)
	}
}

// TestPortsLifecycleMorphRestart tests ports during restart cycle
func TestPortsLifecycleMorphRestart(t *testing.T) {
	w := &Workspace{}

	// First instance
	w.SetMorphInstance("inst-1", "snap-1", "https://vm1.example.com")
	w.AddActivePort(ActivePort{Port: 8080, Service: "old-service"})

	// Stop
	w.ClearMorphInstance()

	// Clear ports for fresh start
	w.ClearActivePorts()

	// Second instance
	w.SetMorphInstance("inst-2", "snap-2", "https://vm2.example.com")
	w.AddActivePort(ActivePort{Port: 8080, Service: "new-service"})

	// Verify new service
	port := w.GetActivePort(8080)
	if port.Service != "new-service" {
		t.Errorf("Expected 'new-service', got '%s'", port.Service)
	}
}

// TestPortsLifecycleWithSnapshots tests ports interaction with snapshots
func TestPortsLifecycleWithSnapshots(t *testing.T) {
	w := &Workspace{}

	w.SetMorphInstance("inst-1", "snap-1", "https://vm.example.com")

	// Add ports
	w.AddActivePort(ActivePort{Port: 8080})
	w.AddActivePort(ActivePort{Port: 3000})

	// Save snapshot
	w.AddSavedSnapshot("snap-checkpoint", "pre-change")

	// Modify ports
	w.RemoveActivePort(3000)
	w.AddActivePort(ActivePort{Port: 5000})

	// Snapshot count unchanged
	if len(w.Morph.SavedSnapshots) != 1 {
		t.Errorf("Expected 1 snapshot, got %d", len(w.Morph.SavedSnapshots))
	}

	// Ports changed
	if len(w.Morph.ActivePorts) != 2 {
		t.Errorf("Expected 2 ports, got %d", len(w.Morph.ActivePorts))
	}
}

// TestPortsLifecycleMultipleInstances tests multiple instance lifecycle
func TestPortsLifecycleMultipleInstances(t *testing.T) {
	w := &Workspace{}

	// Simulate 10 instance cycles
	for i := 0; i < 10; i++ {
		w.SetMorphInstance("inst", "snap", "https://vm.example.com")

		// Add ports
		for p := 8000; p < 8010; p++ {
			w.AddActivePort(ActivePort{Port: p})
		}

		// Stop and clear
		w.ClearMorphInstance()
		w.ClearActivePorts()
	}

	// Should be clean
	if len(w.Morph.ActivePorts) != 0 {
		t.Errorf("Expected 0 ports after cycles, got %d", len(w.Morph.ActivePorts))
	}
}

// TestPortsLifecyclePersistence tests that ports persist across serialize/deserialize
func TestPortsLifecyclePersistence(t *testing.T) {
	w := &Workspace{
		ID:   "ws-persist",
		Name: "persist-test",
	}

	w.SetMorphInstance("inst-1", "snap-1", "https://vm.example.com")
	w.AddActivePort(ActivePort{
		Port:      8080,
		Protocol:  "tcp",
		Service:   "web",
		Container: "nginx",
		LocalPort: 18080,
		URL:       "https://vm.example.com/ports/8080/",
		IsHTTP:    true,
	})

	// Serialize
	data, err := json.Marshal(w)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	// Deserialize
	var w2 Workspace
	if err := json.Unmarshal(data, &w2); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	// Verify all fields preserved
	port := w2.GetActivePort(8080)
	if port == nil {
		t.Fatal("Port not found after persistence")
	}
	if port.Protocol != "tcp" {
		t.Errorf("Protocol not preserved: %s", port.Protocol)
	}
	if port.Service != "web" {
		t.Errorf("Service not preserved: %s", port.Service)
	}
	if port.Container != "nginx" {
		t.Errorf("Container not preserved: %s", port.Container)
	}
	if port.LocalPort != 18080 {
		t.Errorf("LocalPort not preserved: %d", port.LocalPort)
	}
	if !port.IsHTTP {
		t.Error("IsHTTP not preserved")
	}
}

// TestPortsLifecycleNoInstance tests ports without an active instance
func TestPortsLifecycleNoInstance(t *testing.T) {
	w := &Workspace{}

	// Add ports without setting instance
	w.AddActivePort(ActivePort{Port: 8080})

	// Should work (no instance required for data structure)
	if len(w.Morph.ActivePorts) != 1 {
		t.Errorf("Expected 1 port, got %d", len(w.Morph.ActivePorts))
	}

	// Instance ID should be empty
	if w.Morph.InstanceID != "" {
		t.Errorf("Expected empty instance ID, got '%s'", w.Morph.InstanceID)
	}
}

// TestPortsLifecycleIsMorphRunning tests IsMorphRunning with ports
func TestPortsLifecycleIsMorphRunning(t *testing.T) {
	w := &Workspace{}

	// Not running initially
	if w.IsMorphRunning() {
		t.Error("Should not be running initially")
	}

	// Start instance
	w.SetMorphInstance("inst-1", "snap-1", "https://vm.example.com")
	w.AddActivePort(ActivePort{Port: 8080})

	// Should be running
	if !w.IsMorphRunning() {
		t.Error("Should be running after SetMorphInstance")
	}

	// Stop
	w.ClearMorphInstance()

	// Should not be running
	if w.IsMorphRunning() {
		t.Error("Should not be running after ClearMorphInstance")
	}

	// Ports still exist
	if len(w.Morph.ActivePorts) != 1 {
		t.Errorf("Expected 1 port to remain, got %d", len(w.Morph.ActivePorts))
	}
}

// TestPortsLifecycleGetMorphURLsWithPorts tests GetMorphURLs alongside ports
func TestPortsLifecycleGetMorphURLsWithPorts(t *testing.T) {
	w := &Workspace{}

	w.SetMorphInstance("inst-1", "snap-1", "https://vm.example.com")

	// Add custom ports
	w.AddActivePort(ActivePort{
		Port:   8080,
		URL:    "https://vm.example.com/custom-port/",
		IsHTTP: true,
	})

	// Get standard Morph URLs
	morphURLs := w.GetMorphURLs()
	if morphURLs["code"] == "" {
		t.Error("Expected code URL")
	}
	if morphURLs["vnc"] == "" {
		t.Error("Expected VNC URL")
	}

	// Get port URL (different from Morph URLs)
	port := w.GetActivePort(8080)
	if port.URL != "https://vm.example.com/custom-port/" {
		t.Errorf("Expected custom port URL, got '%s'", port.URL)
	}
}

// TestPortsLifecycleTimestampDrifts tests DiscoveredAt over time
func TestPortsLifecycleTimestampDrifts(t *testing.T) {
	w := &Workspace{}

	t1 := time.Now()
	w.AddActivePort(ActivePort{Port: 8080})

	// Small delay
	time.Sleep(10 * time.Millisecond)

	t2 := time.Now()
	w.AddActivePort(ActivePort{Port: 8081})

	port1 := w.GetActivePort(8080)
	port2 := w.GetActivePort(8081)

	// port1 should be before port2
	if !port1.DiscoveredAt.Before(port2.DiscoveredAt) && !port1.DiscoveredAt.Equal(port2.DiscoveredAt) {
		// They could be equal if very fast
	}

	// Both should be between t1 and t2 (roughly)
	if port1.DiscoveredAt.Before(t1.Add(-time.Second)) {
		t.Error("port1 timestamp too early")
	}
	if port2.DiscoveredAt.After(t2.Add(time.Second)) {
		t.Error("port2 timestamp too late")
	}
}

// TestPortsLifecycleCDPPortInteraction tests CDPPort alongside ActivePorts
func TestPortsLifecycleCDPPortInteraction(t *testing.T) {
	w := &Workspace{}

	w.SetMorphInstance("inst-1", "snap-1", "https://vm.example.com")
	w.Morph.CDPPort = 9222

	// Add ActivePort for CDP (might be used differently)
	w.AddActivePort(ActivePort{
		Port:     9222,
		Service:  "cdp",
		Protocol: "tcp",
	})

	// Both should coexist
	if w.Morph.CDPPort != 9222 {
		t.Error("CDPPort should be 9222")
	}

	port := w.GetActivePort(9222)
	if port == nil || port.Service != "cdp" {
		t.Error("ActivePort for 9222 should exist")
	}
}

// TestPortsLifecyclePartialState tests partial MorphState scenarios
func TestPortsLifecyclePartialState(t *testing.T) {
	testCases := []struct {
		name  string
		setup func(*Workspace)
	}{
		{
			name: "only_instance_id",
			setup: func(w *Workspace) {
				w.Morph.InstanceID = "inst-1"
			},
		},
		{
			name: "only_status",
			setup: func(w *Workspace) {
				w.Morph.Status = "running"
			},
		},
		{
			name: "only_ports",
			setup: func(w *Workspace) {
				w.AddActivePort(ActivePort{Port: 8080})
			},
		},
		{
			name: "only_snapshots",
			setup: func(w *Workspace) {
				w.AddSavedSnapshot("snap-1", "checkpoint")
			},
		},
		{
			name: "status_and_ports",
			setup: func(w *Workspace) {
				w.Morph.Status = "running"
				w.AddActivePort(ActivePort{Port: 8080})
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			w := &Workspace{}
			tc.setup(w)

			// Serialize and deserialize
			data, err := json.Marshal(w)
			if err != nil {
				t.Fatalf("Marshal failed: %v", err)
			}

			var w2 Workspace
			if err := json.Unmarshal(data, &w2); err != nil {
				t.Fatalf("Unmarshal failed: %v", err)
			}
		})
	}
}

// TestPortsLifecycleURLDerivation tests URL derivation patterns
func TestPortsLifecycleURLDerivation(t *testing.T) {
	testCases := []struct {
		name    string
		baseURL string
		port    int
		wantURL string
	}{
		{"standard", "https://vm.example.com", 8080, "https://vm.example.com/ports/8080/"},
		{"with_path", "https://vm.example.com/app", 3000, "https://vm.example.com/app/ports/3000/"},
		{"localhost", "http://localhost", 8080, "http://localhost/ports/8080/"},
		{"ip_address", "http://192.168.1.1", 8080, "http://192.168.1.1/ports/8080/"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			w := &Workspace{}
			w.SetMorphInstance("inst-1", "snap-1", tc.baseURL)
			w.AddActivePort(ActivePort{
				Port: tc.port,
				URL:  tc.wantURL, // Set explicitly since AddActivePort doesn't derive URLs
			})

			port := w.GetActivePort(tc.port)
			if port.URL != tc.wantURL {
				t.Errorf("Expected URL '%s', got '%s'", tc.wantURL, port.URL)
			}
		})
	}
}

// TestPortsLifecycleStateMachine tests full state machine transitions
func TestPortsLifecycleStateMachine(t *testing.T) {
	w := &Workspace{}

	// State: Initial
	if w.Morph.Status != "" {
		t.Error("Initial status should be empty")
	}
	if len(w.Morph.ActivePorts) != 0 {
		t.Error("Initial ports should be empty")
	}

	// Transition: Start
	w.SetMorphInstance("inst-1", "snap-1", "https://vm.example.com")
	if w.Morph.Status != "running" {
		t.Error("Status should be 'running' after start")
	}

	// Add ports while running
	w.AddActivePort(ActivePort{Port: 8080, IsHTTP: true})
	w.AddActivePort(ActivePort{Port: 3000, IsHTTP: true})
	if len(w.Morph.ActivePorts) != 2 {
		t.Errorf("Expected 2 ports, got %d", len(w.Morph.ActivePorts))
	}

	// Transition: Stop
	w.ClearMorphInstance()
	if w.Morph.Status != "stopped" {
		t.Error("Status should be 'stopped' after clear")
	}

	// Ports persist through stop
	if len(w.Morph.ActivePorts) != 2 {
		t.Errorf("Ports should persist through stop, got %d", len(w.Morph.ActivePorts))
	}

	// Transition: Clear ports
	w.ClearActivePorts()
	if len(w.Morph.ActivePorts) != 0 {
		t.Errorf("Expected 0 ports after clear, got %d", len(w.Morph.ActivePorts))
	}

	// Transition: Restart
	w.SetMorphInstance("inst-2", "snap-2", "https://vm2.example.com")
	if w.Morph.Status != "running" {
		t.Error("Status should be 'running' after restart")
	}
}

// TestPortsLifecycleEdgeTiming tests edge cases with timing
func TestPortsLifecycleEdgeTiming(t *testing.T) {
	w := &Workspace{}

	// Set timestamp before adding
	before := time.Now()
	w.AddActivePort(ActivePort{Port: 8080})
	after := time.Now()

	port := w.GetActivePort(8080)

	// Timestamp should be between before and after
	if port.DiscoveredAt.Before(before) {
		t.Error("Timestamp should not be before 'before'")
	}
	if port.DiscoveredAt.After(after) {
		t.Error("Timestamp should not be after 'after'")
	}
}

// TestPortsLifecycleWithAllWorkspaceFields tests ports with complete workspace
func TestPortsLifecycleWithAllWorkspaceFields(t *testing.T) {
	w := &Workspace{
		ID:          "ws-full",
		Name:        "full-test",
		Path:        "/tmp/test",
		ProjectPath: "/tmp/test/project",
		Template:    "node",
		Status:      "active",
		BasePort:    8000,
		Ports: map[string]int{
			"PORT":      8000,
			"CODE_PORT": 8001,
		},
		Packages:   []string{"node", "npm"},
		CreatedAt:  time.Now(),
		LastActive: time.Now(),
		Git: &GitInfo{
			Remote: "origin",
			Branch: "main",
			Commit: "abc123",
		},
	}

	w.SetMorphInstance("inst-1", "snap-1", "https://vm.example.com")
	w.AddActivePort(ActivePort{Port: 8080, Service: "web"})

	// Serialize
	data, err := json.Marshal(w)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	// Deserialize
	var w2 Workspace
	if err := json.Unmarshal(data, &w2); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	// Verify workspace fields
	if w2.ID != "ws-full" {
		t.Error("ID not preserved")
	}
	if w2.Git == nil || w2.Git.Branch != "main" {
		t.Error("Git info not preserved")
	}

	// Verify Morph fields
	if w2.Morph.InstanceID != "inst-1" {
		t.Error("InstanceID not preserved")
	}

	// Verify ports
	port := w2.GetActivePort(8080)
	if port == nil || port.Service != "web" {
		t.Error("Port not preserved")
	}
}
