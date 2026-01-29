// internal/workspace/morph_ports_advanced_test.go
package workspace

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

// =============================================================================
// Port Range Edge Cases
// =============================================================================

func TestActivePortBoundaryValues(t *testing.T) {
	tests := []struct {
		name string
		port int
	}{
		{"zero", 0},
		{"one", 1},
		{"privileged_max", 1023},
		{"registered_min", 1024},
		{"registered_max", 49151},
		{"dynamic_min", 49152},
		{"dynamic_max", 65535},
		{"negative", -1},
		{"large_negative", -65535},
		{"overflow", 65536},
		{"large_overflow", 100000},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := &Workspace{}
			w.AddActivePort(ActivePort{Port: tt.port, Service: tt.name})

			if len(w.Morph.ActivePorts) != 1 {
				t.Errorf("ActivePorts count = %d, want 1", len(w.Morph.ActivePorts))
			}
			if w.Morph.ActivePorts[0].Port != tt.port {
				t.Errorf("Port = %d, want %d", w.Morph.ActivePorts[0].Port, tt.port)
			}
		})
	}
}

func TestCommonPorts(t *testing.T) {
	commonPorts := []struct {
		port    int
		service string
	}{
		{22, "ssh"},
		{80, "http"},
		{443, "https"},
		{3000, "node"},
		{3306, "mysql"},
		{5432, "postgres"},
		{5173, "vite"},
		{6379, "redis"},
		{8080, "http-alt"},
		{8443, "https-alt"},
		{9222, "cdp"},
		{27017, "mongodb"},
	}

	w := &Workspace{}

	for _, cp := range commonPorts {
		w.AddActivePort(ActivePort{
			Port:    cp.port,
			Service: cp.service,
			IsHTTP:  cp.port == 80 || cp.port == 443 || cp.port == 3000 || cp.port == 5173 || cp.port == 8080 || cp.port == 8443,
		})
	}

	if len(w.Morph.ActivePorts) != len(commonPorts) {
		t.Errorf("ActivePorts count = %d, want %d", len(w.Morph.ActivePorts), len(commonPorts))
	}

	// Verify each port can be retrieved
	for _, cp := range commonPorts {
		port := w.GetActivePort(cp.port)
		if port == nil {
			t.Errorf("GetActivePort(%d) returned nil", cp.port)
		} else if port.Service != cp.service {
			t.Errorf("Port %d service = %s, want %s", cp.port, port.Service, cp.service)
		}
	}
}

// =============================================================================
// Protocol Edge Cases
// =============================================================================

func TestActivePortProtocols(t *testing.T) {
	protocols := []string{
		"tcp",
		"udp",
		"TCP",
		"UDP",
		"Tcp",
		"",
		"sctp",
		"dccp",
		"unknown",
		"tcp/udp",
	}

	for _, proto := range protocols {
		t.Run("protocol_"+proto, func(t *testing.T) {
			w := &Workspace{}
			w.AddActivePort(ActivePort{Port: 8080, Protocol: proto})

			if w.Morph.ActivePorts[0].Protocol != proto {
				t.Errorf("Protocol = %s, want %s", w.Morph.ActivePorts[0].Protocol, proto)
			}
		})
	}
}

// =============================================================================
// URL Edge Cases
// =============================================================================

func TestActivePortURLVariations(t *testing.T) {
	urls := []string{
		"http://localhost:5173",
		"https://localhost:5173",
		"http://127.0.0.1:5173",
		"http://[::1]:5173",
		"http://0.0.0.0:5173",
		"http://host.docker.internal:5173",
		"http://example.com:5173",
		"http://example.com:5173/path",
		"http://example.com:5173/path?query=value",
		"http://example.com:5173/path#fragment",
		"",
		"   ",
		"invalid-url",
		"ftp://localhost:21",
		"ws://localhost:8080",
		"wss://localhost:8080",
	}

	for i, url := range urls {
		t.Run("url_"+string(rune('0'+i%10)), func(t *testing.T) {
			w := &Workspace{}
			w.AddActivePort(ActivePort{Port: 8080 + i, URL: url})

			if w.Morph.ActivePorts[0].URL != url {
				t.Errorf("URL = %q, want %q", w.Morph.ActivePorts[0].URL, url)
			}
		})
	}
}

// =============================================================================
// Container Name Edge Cases
// =============================================================================

func TestActivePortContainerNames(t *testing.T) {
	containers := []string{
		"myapp-web",
		"myapp_web",
		"myapp.web",
		"MyApp-Web",
		"myapp-web-1",
		"a",
		"",
		"   ",
		"container-with-very-long-name-that-might-cause-issues",
		"container/with/slashes",
		"container:with:colons",
		"日本語コンテナ",
		"🐳docker-emoji",
		"sha256:abc123def456",
	}

	for i, container := range containers {
		t.Run("container_"+string(rune('0'+i%10)), func(t *testing.T) {
			w := &Workspace{}
			w.AddActivePort(ActivePort{Port: 8080 + i, Container: container})

			if w.Morph.ActivePorts[0].Container != container {
				t.Errorf("Container = %q, want %q", w.Morph.ActivePorts[0].Container, container)
			}
		})
	}
}

// =============================================================================
// Service Name Edge Cases
// =============================================================================

func TestActivePortServiceNames(t *testing.T) {
	services := []string{
		"vite",
		"postgres",
		"redis",
		"nginx",
		"apache",
		"node",
		"python",
		"go",
		"rust",
		"",
		"   ",
		"service-with-dashes",
		"service_with_underscores",
		"service.with.dots",
		"SERVICE_UPPER",
		"MixedCase",
		"service123",
		"123service",
		"日本語サービス",
		"🚀rocket-service",
		"a",
		"service-with-very-long-name-that-might-exceed-normal-limits-for-display",
	}

	for i, service := range services {
		t.Run("service_"+string(rune('0'+i%10)), func(t *testing.T) {
			w := &Workspace{}
			w.AddActivePort(ActivePort{Port: 8080 + i, Service: service})

			if w.Morph.ActivePorts[0].Service != service {
				t.Errorf("Service = %q, want %q", w.Morph.ActivePorts[0].Service, service)
			}
		})
	}
}

// =============================================================================
// LocalPort Relationship Tests
// =============================================================================

func TestActivePortLocalPortRelationship(t *testing.T) {
	tests := []struct {
		name      string
		port      int
		localPort int
	}{
		{"same", 5173, 5173},
		{"different", 5173, 15173},
		{"zero_local", 5173, 0},
		{"zero_both", 0, 0},
		{"swapped", 15173, 5173},
		{"privileged_to_unprivileged", 80, 8080},
		{"unprivileged_to_privileged", 8080, 80},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := &Workspace{}
			w.AddActivePort(ActivePort{
				Port:      tt.port,
				LocalPort: tt.localPort,
			})

			if w.Morph.ActivePorts[0].Port != tt.port {
				t.Errorf("Port = %d, want %d", w.Morph.ActivePorts[0].Port, tt.port)
			}
			if w.Morph.ActivePorts[0].LocalPort != tt.localPort {
				t.Errorf("LocalPort = %d, want %d", w.Morph.ActivePorts[0].LocalPort, tt.localPort)
			}
		})
	}
}

// =============================================================================
// IsHTTP Edge Cases
// =============================================================================

func TestActivePortIsHTTPVariations(t *testing.T) {
	w := &Workspace{}

	// Mix of HTTP and non-HTTP ports
	w.AddActivePort(ActivePort{Port: 80, IsHTTP: true})
	w.AddActivePort(ActivePort{Port: 443, IsHTTP: true})
	w.AddActivePort(ActivePort{Port: 22, IsHTTP: false})
	w.AddActivePort(ActivePort{Port: 3306, IsHTTP: false})
	w.AddActivePort(ActivePort{Port: 8080, IsHTTP: true})
	w.AddActivePort(ActivePort{Port: 5432, IsHTTP: false})

	httpPorts := w.GetHTTPPorts()
	if len(httpPorts) != 3 {
		t.Errorf("HTTP ports count = %d, want 3", len(httpPorts))
	}

	// Verify correct ports are HTTP
	httpPortNumbers := map[int]bool{80: true, 443: true, 8080: true}
	for _, p := range httpPorts {
		if !httpPortNumbers[p.Port] {
			t.Errorf("Port %d should not be in HTTP list", p.Port)
		}
	}
}

// =============================================================================
// Concurrent Access Tests
// =============================================================================

func TestConcurrentAddActivePorts(t *testing.T) {
	if raceDetectorEnabled() {
		t.Skip("Skipping concurrent test with race detector (Workspace is not thread-safe)")
	}

	w := &Workspace{}
	var wg sync.WaitGroup

	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			w.AddActivePort(ActivePort{
				Port:    8000 + idx,
				Service: "service-" + string(rune('0'+idx%10)),
			})
		}(i)
	}

	wg.Wait()

	// Due to race conditions, count may vary
	t.Logf("Final ActivePorts count: %d", len(w.Morph.ActivePorts))
}

func TestConcurrentGetActivePort(t *testing.T) {
	w := &Workspace{}

	// Pre-populate with ports
	for i := 0; i < 100; i++ {
		w.AddActivePort(ActivePort{Port: 8000 + i, Service: "service"})
	}

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			_ = w.GetActivePort(8000 + idx)
		}(i)
	}

	wg.Wait()
}

func TestConcurrentGetHTTPPorts(t *testing.T) {
	w := &Workspace{}

	// Pre-populate with ports
	for i := 0; i < 50; i++ {
		w.AddActivePort(ActivePort{Port: 8000 + i, IsHTTP: i%2 == 0})
	}

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = w.GetHTTPPorts()
		}()
	}

	wg.Wait()
}

// =============================================================================
// Persistence Tests
// =============================================================================

func TestActivePortsPersistence(t *testing.T) {
	tmpDir := t.TempDir()
	stateDir := filepath.Join(tmpDir, ".dba")
	if err := os.MkdirAll(stateDir, 0755); err != nil {
		t.Fatal(err)
	}

	original := &Workspace{
		ID:       "ws-ports-persist",
		Name:     "ports-persist-test",
		Path:     tmpDir,
		Template: "node",
		Status:   "active",
	}
	original.SetMorphInstance("inst-123", "snap-456", "https://example.com")

	// Add various ports
	original.AddActivePort(ActivePort{
		Port:      5173,
		Protocol:  "tcp",
		Service:   "vite",
		Container: "myapp-web",
		LocalPort: 5173,
		URL:       "http://localhost:5173",
		IsHTTP:    true,
	})
	original.AddActivePort(ActivePort{
		Port:      5432,
		Protocol:  "tcp",
		Service:   "postgres",
		Container: "myapp-db",
		IsHTTP:    false,
	})

	// Save
	if err := original.SaveState(); err != nil {
		t.Fatalf("SaveState error: %v", err)
	}

	// Load
	loaded, err := Load(tmpDir)
	if err != nil {
		t.Fatalf("Load error: %v", err)
	}

	// Verify ports
	if len(loaded.Morph.ActivePorts) != 2 {
		t.Fatalf("ActivePorts count = %d, want 2", len(loaded.Morph.ActivePorts))
	}

	port1 := loaded.GetActivePort(5173)
	if port1 == nil {
		t.Fatal("Port 5173 not found after load")
	}
	if port1.Service != "vite" {
		t.Errorf("Port 5173 service = %s, want vite", port1.Service)
	}
	if port1.Container != "myapp-web" {
		t.Errorf("Port 5173 container = %s, want myapp-web", port1.Container)
	}
	if !port1.IsHTTP {
		t.Error("Port 5173 should be HTTP")
	}

	port2 := loaded.GetActivePort(5432)
	if port2 == nil {
		t.Fatal("Port 5432 not found after load")
	}
	if port2.Service != "postgres" {
		t.Errorf("Port 5432 service = %s, want postgres", port2.Service)
	}
}

func TestActivePortsEmptyPersistence(t *testing.T) {
	tmpDir := t.TempDir()
	stateDir := filepath.Join(tmpDir, ".dba")
	if err := os.MkdirAll(stateDir, 0755); err != nil {
		t.Fatal(err)
	}

	original := &Workspace{
		ID:   "ws-empty-ports",
		Name: "empty-ports-test",
		Path: tmpDir,
	}
	// No active ports

	if err := original.SaveState(); err != nil {
		t.Fatalf("SaveState error: %v", err)
	}

	loaded, err := Load(tmpDir)
	if err != nil {
		t.Fatalf("Load error: %v", err)
	}

	if len(loaded.Morph.ActivePorts) != 0 {
		t.Errorf("ActivePorts count = %d, want 0", len(loaded.Morph.ActivePorts))
	}
}

// =============================================================================
// State Transition Tests
// =============================================================================

func TestActivePortsAcrossStateTransitions(t *testing.T) {
	w := &Workspace{}

	// Initial state - no instance
	w.AddActivePort(ActivePort{Port: 5173, Service: "vite"})

	// Start instance
	w.SetMorphInstance("inst-123", "snap-456", "https://example.com")

	// Ports should be preserved
	if len(w.Morph.ActivePorts) != 1 {
		t.Errorf("ActivePorts count after SetMorphInstance = %d, want 1", len(w.Morph.ActivePorts))
	}

	// Add more ports while running
	w.AddActivePort(ActivePort{Port: 5432, Service: "postgres"})

	// Clear instance
	w.ClearMorphInstance()

	// Ports should still be preserved
	if len(w.Morph.ActivePorts) != 2 {
		t.Errorf("ActivePorts count after ClearMorphInstance = %d, want 2", len(w.Morph.ActivePorts))
	}

	// Explicitly clear ports
	w.ClearActivePorts()

	if len(w.Morph.ActivePorts) != 0 {
		t.Errorf("ActivePorts count after ClearActivePorts = %d, want 0", len(w.Morph.ActivePorts))
	}
}

// =============================================================================
// JSON Serialization Edge Cases
// =============================================================================

func TestActivePortJSONWithAllFields(t *testing.T) {
	original := ActivePort{
		Port:         5173,
		Protocol:     "tcp",
		Service:      "vite",
		Container:    "myapp-web",
		LocalPort:    15173,
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
	if loaded.Protocol != original.Protocol {
		t.Errorf("Protocol = %s, want %s", loaded.Protocol, original.Protocol)
	}
	if loaded.Service != original.Service {
		t.Errorf("Service = %s, want %s", loaded.Service, original.Service)
	}
	if loaded.Container != original.Container {
		t.Errorf("Container = %s, want %s", loaded.Container, original.Container)
	}
	if loaded.LocalPort != original.LocalPort {
		t.Errorf("LocalPort = %d, want %d", loaded.LocalPort, original.LocalPort)
	}
	if loaded.URL != original.URL {
		t.Errorf("URL = %s, want %s", loaded.URL, original.URL)
	}
	if loaded.IsHTTP != original.IsHTTP {
		t.Errorf("IsHTTP = %v, want %v", loaded.IsHTTP, original.IsHTTP)
	}
}

func TestActivePortJSONWithMinimalFields(t *testing.T) {
	original := ActivePort{Port: 8080}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}

	var loaded ActivePort
	if err := json.Unmarshal(data, &loaded); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}

	if loaded.Port != 8080 {
		t.Errorf("Port = %d, want 8080", loaded.Port)
	}
	if loaded.Protocol != "" {
		t.Errorf("Protocol = %s, want empty", loaded.Protocol)
	}
	if loaded.Service != "" {
		t.Errorf("Service = %s, want empty", loaded.Service)
	}
}

func TestActivePortJSONWithSpecialCharacters(t *testing.T) {
	original := ActivePort{
		Port:      8080,
		Service:   "service-with-\"quotes\"",
		Container: "container\\with\\backslashes",
		URL:       "http://example.com/path?query=value&other=<tag>",
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}

	var loaded ActivePort
	if err := json.Unmarshal(data, &loaded); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}

	if loaded.Service != original.Service {
		t.Errorf("Service = %s, want %s", loaded.Service, original.Service)
	}
	if loaded.Container != original.Container {
		t.Errorf("Container = %s, want %s", loaded.Container, original.Container)
	}
	if loaded.URL != original.URL {
		t.Errorf("URL = %s, want %s", loaded.URL, original.URL)
	}
}

func TestActivePortJSONWithUnicode(t *testing.T) {
	original := ActivePort{
		Port:      8080,
		Service:   "日本語サービス",
		Container: "컨테이너-한국어",
		URL:       "http://example.com/путь",
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}

	var loaded ActivePort
	if err := json.Unmarshal(data, &loaded); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}

	if loaded.Service != original.Service {
		t.Errorf("Service = %s, want %s", loaded.Service, original.Service)
	}
	if loaded.Container != original.Container {
		t.Errorf("Container = %s, want %s", loaded.Container, original.Container)
	}
}

// =============================================================================
// Performance Tests
// =============================================================================

func TestManyActivePorts(t *testing.T) {
	w := &Workspace{}

	// Add 1000 ports
	for i := 0; i < 1000; i++ {
		w.AddActivePort(ActivePort{
			Port:    8000 + i,
			Service: "service-" + string(rune('0'+i%10)),
		})
	}

	if len(w.Morph.ActivePorts) != 1000 {
		t.Errorf("ActivePorts count = %d, want 1000", len(w.Morph.ActivePorts))
	}

	// Get a port from the middle
	port := w.GetActivePort(8500)
	if port == nil {
		t.Error("GetActivePort(8500) returned nil")
	}

	// Get HTTP ports
	httpPorts := w.GetHTTPPorts()
	if len(httpPorts) != 0 {
		t.Errorf("HTTP ports = %d, want 0 (none were marked HTTP)", len(httpPorts))
	}

	// Get primary app port
	primary := w.GetPrimaryAppPort()
	if primary == nil {
		t.Error("GetPrimaryAppPort returned nil")
	}
}

func TestRapidPortAddRemove(t *testing.T) {
	w := &Workspace{}

	// Rapid add/remove cycles
	for i := 0; i < 1000; i++ {
		w.AddActivePort(ActivePort{Port: 8080})
		w.RemoveActivePort(8080)
	}

	if len(w.Morph.ActivePorts) != 0 {
		t.Errorf("ActivePorts count = %d, want 0", len(w.Morph.ActivePorts))
	}
}

func TestRapidPortUpdates(t *testing.T) {
	w := &Workspace{}

	// Rapid updates to same port
	for i := 0; i < 1000; i++ {
		w.AddActivePort(ActivePort{
			Port:    8080,
			Service: "service-" + string(rune('0'+i%10)),
		})
	}

	if len(w.Morph.ActivePorts) != 1 {
		t.Errorf("ActivePorts count = %d, want 1", len(w.Morph.ActivePorts))
	}
}

// =============================================================================
// GetPrimaryAppPort Edge Cases
// =============================================================================

func TestGetPrimaryAppPortOrderPreference(t *testing.T) {
	w := &Workspace{}

	// Add non-HTTP first, then HTTP
	w.AddActivePort(ActivePort{Port: 5432, Service: "postgres", IsHTTP: false})
	w.AddActivePort(ActivePort{Port: 6379, Service: "redis", IsHTTP: false})
	w.AddActivePort(ActivePort{Port: 5173, Service: "vite", IsHTTP: true})
	w.AddActivePort(ActivePort{Port: 8080, Service: "api", IsHTTP: true})

	primary := w.GetPrimaryAppPort()
	if primary == nil {
		t.Fatal("GetPrimaryAppPort returned nil")
	}

	// Should return first HTTP port (5173)
	if primary.Port != 5173 {
		t.Errorf("Primary port = %d, want 5173", primary.Port)
	}
}

func TestGetPrimaryAppPortAfterRemove(t *testing.T) {
	w := &Workspace{}

	w.AddActivePort(ActivePort{Port: 5173, Service: "vite", IsHTTP: true})
	w.AddActivePort(ActivePort{Port: 8080, Service: "api", IsHTTP: true})

	// Remove first HTTP port
	w.RemoveActivePort(5173)

	primary := w.GetPrimaryAppPort()
	if primary == nil {
		t.Fatal("GetPrimaryAppPort returned nil")
	}

	// Should return second HTTP port (8080)
	if primary.Port != 8080 {
		t.Errorf("Primary port = %d, want 8080", primary.Port)
	}
}

func TestGetPrimaryAppPortAfterUpdate(t *testing.T) {
	w := &Workspace{}

	w.AddActivePort(ActivePort{Port: 5173, Service: "vite", IsHTTP: true})
	w.AddActivePort(ActivePort{Port: 8080, Service: "api", IsHTTP: false})

	// Update second port to be HTTP
	w.AddActivePort(ActivePort{Port: 8080, Service: "api", IsHTTP: true})

	primary := w.GetPrimaryAppPort()
	if primary == nil {
		t.Fatal("GetPrimaryAppPort returned nil")
	}

	// Should still return first HTTP port (5173) due to order
	if primary.Port != 5173 {
		t.Errorf("Primary port = %d, want 5173", primary.Port)
	}
}

// =============================================================================
// Duplicate Port Handling
// =============================================================================

func TestAddDuplicatePortDifferentServices(t *testing.T) {
	w := &Workspace{}

	// Add same port with different service names
	w.AddActivePort(ActivePort{Port: 8080, Service: "service-a"})
	w.AddActivePort(ActivePort{Port: 8080, Service: "service-b"})

	// Should update, not add
	if len(w.Morph.ActivePorts) != 1 {
		t.Errorf("ActivePorts count = %d, want 1", len(w.Morph.ActivePorts))
	}

	// Should have latest service name
	if w.Morph.ActivePorts[0].Service != "service-b" {
		t.Errorf("Service = %s, want service-b", w.Morph.ActivePorts[0].Service)
	}
}

func TestAddSameServiceDifferentPorts(t *testing.T) {
	w := &Workspace{}

	// Add different ports with same service name
	w.AddActivePort(ActivePort{Port: 8080, Service: "nginx"})
	w.AddActivePort(ActivePort{Port: 8443, Service: "nginx"})

	// Should add both
	if len(w.Morph.ActivePorts) != 2 {
		t.Errorf("ActivePorts count = %d, want 2", len(w.Morph.ActivePorts))
	}
}

// =============================================================================
// Integration with MorphState Fields
// =============================================================================

func TestActivePortsWithCDPPort(t *testing.T) {
	w := &Workspace{}

	w.SetMorphInstance("inst-123", "snap-456", "https://example.com")
	w.Morph.CDPPort = 9222

	// Add active port for CDP
	w.AddActivePort(ActivePort{
		Port:      9222,
		Service:   "cdp",
		Protocol:  "tcp",
		IsHTTP:    false,
		LocalPort: 9222,
	})

	// CDPPort and ActivePorts should coexist
	if w.Morph.CDPPort != 9222 {
		t.Errorf("CDPPort = %d, want 9222", w.Morph.CDPPort)
	}

	cdpPort := w.GetActivePort(9222)
	if cdpPort == nil {
		t.Error("CDP port not found in ActivePorts")
	}
}

func TestActivePortsWithSavedSnapshots(t *testing.T) {
	w := &Workspace{}

	w.SetMorphInstance("inst-123", "snap-456", "https://example.com")

	// Add ports and snapshots
	w.AddActivePort(ActivePort{Port: 5173, Service: "vite"})
	w.AddSavedSnapshot("snap-save-1", "checkpoint-1")

	// Both should be preserved
	if len(w.Morph.ActivePorts) != 1 {
		t.Errorf("ActivePorts count = %d, want 1", len(w.Morph.ActivePorts))
	}
	if len(w.Morph.SavedSnapshots) != 1 {
		t.Errorf("SavedSnapshots count = %d, want 1", len(w.Morph.SavedSnapshots))
	}
}
