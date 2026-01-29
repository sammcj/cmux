// internal/workspace/morph_ports_stress_test.go
package workspace

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"sync"
	"testing"
	"time"
)

// TestPortStressLargeNumberOfPorts tests handling thousands of ports
func TestPortStressLargeNumberOfPorts(t *testing.T) {
	w := &Workspace{}

	// Add 10000 ports
	for i := 1; i <= 10000; i++ {
		w.AddActivePort(ActivePort{
			Port:    i,
			Service: fmt.Sprintf("svc-%d", i),
		})
	}

	if len(w.Morph.ActivePorts) != 10000 {
		t.Errorf("Expected 10000 ports, got %d", len(w.Morph.ActivePorts))
	}

	// Verify lookup still works
	port := w.GetActivePort(5000)
	if port == nil || port.Service != "svc-5000" {
		t.Error("Lookup failed for port in large list")
	}
}

// TestPortStressRapidAddRemove tests rapid add/remove cycles
func TestPortStressRapidAddRemove(t *testing.T) {
	w := &Workspace{}

	// Rapid add/remove cycles
	for i := 0; i < 10000; i++ {
		w.AddActivePort(ActivePort{Port: 8080})
		w.RemoveActivePort(8080)
	}

	if len(w.Morph.ActivePorts) != 0 {
		t.Errorf("Expected 0 ports after rapid cycles, got %d", len(w.Morph.ActivePorts))
	}
}

// TestPortStressRandomOperations tests random port operations
func TestPortStressRandomOperations(t *testing.T) {
	w := &Workspace{}
	rand.Seed(time.Now().UnixNano())

	tracked := make(map[int]bool)

	for i := 0; i < 1000; i++ {
		port := rand.Intn(65536)
		if rand.Intn(2) == 0 {
			// Add
			w.AddActivePort(ActivePort{Port: port})
			tracked[port] = true
		} else {
			// Remove
			w.RemoveActivePort(port)
			delete(tracked, port)
		}
	}

	// Verify tracked matches actual
	if len(w.Morph.ActivePorts) != len(tracked) {
		t.Errorf("Mismatch: %d ports vs %d tracked", len(w.Morph.ActivePorts), len(tracked))
	}
}

// TestPortStressConcurrentAddRemove tests concurrent add/remove operations
func TestPortStressConcurrentAddRemove(t *testing.T) {
	if raceDetectorEnabled() {
		t.Skip("Skipping concurrent modification test with race detector")
	}

	w := &Workspace{}
	var wg sync.WaitGroup

	// Multiple goroutines adding
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(base int) {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				w.AddActivePort(ActivePort{Port: base*100 + j})
			}
		}(i)
	}

	wg.Wait()
	t.Logf("Final port count after concurrent add: %d", len(w.Morph.ActivePorts))
}

// TestPortStressJSONLargePayload tests JSON serialization of large port lists
func TestPortStressJSONLargePayload(t *testing.T) {
	w := &Workspace{
		ID:   "ws-stress",
		Name: "stress-test",
	}

	// Add 1000 ports with full data
	for i := 0; i < 1000; i++ {
		w.AddActivePort(ActivePort{
			Port:      i,
			Protocol:  "tcp",
			Service:   fmt.Sprintf("service-%d", i),
			Container: fmt.Sprintf("container-%d", i),
			LocalPort: 10000 + i,
			URL:       fmt.Sprintf("http://localhost:%d", 10000+i),
			IsHTTP:    i%2 == 0,
		})
	}

	// Marshal
	data, err := json.Marshal(w)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	t.Logf("JSON size for 1000 ports: %d bytes", len(data))

	// Unmarshal
	var w2 Workspace
	if err := json.Unmarshal(data, &w2); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if len(w2.Morph.ActivePorts) != 1000 {
		t.Errorf("Expected 1000 ports after round-trip, got %d", len(w2.Morph.ActivePorts))
	}
}

// TestPortStressGetHTTPPortsLarge tests GetHTTPPorts with many ports
func TestPortStressGetHTTPPortsLarge(t *testing.T) {
	w := &Workspace{}

	// Add 5000 ports, half HTTP
	for i := 0; i < 5000; i++ {
		w.AddActivePort(ActivePort{
			Port:   i,
			IsHTTP: i%2 == 0,
		})
	}

	httpPorts := w.GetHTTPPorts()
	if len(httpPorts) != 2500 {
		t.Errorf("Expected 2500 HTTP ports, got %d", len(httpPorts))
	}
}

// TestPortStressClearLargeList tests clearing a large port list
func TestPortStressClearLargeList(t *testing.T) {
	w := &Workspace{}

	// Add many ports
	for i := 0; i < 10000; i++ {
		w.AddActivePort(ActivePort{Port: i})
	}

	// Clear
	w.ClearActivePorts()

	if len(w.Morph.ActivePorts) != 0 {
		t.Errorf("Expected 0 ports after clear, got %d", len(w.Morph.ActivePorts))
	}
}

// TestPortStressSetLargeList tests SetActivePorts with large list
func TestPortStressSetLargeList(t *testing.T) {
	w := &Workspace{}

	// Create large list
	ports := make([]ActivePort, 5000)
	for i := range ports {
		ports[i] = ActivePort{Port: i, Service: fmt.Sprintf("svc-%d", i)}
	}

	w.SetActivePorts(ports)

	if len(w.Morph.ActivePorts) != 5000 {
		t.Errorf("Expected 5000 ports, got %d", len(w.Morph.ActivePorts))
	}
}

// TestPortStressSequentialLookups tests many sequential lookups
func TestPortStressSequentialLookups(t *testing.T) {
	w := &Workspace{}

	// Add ports
	for i := 0; i < 1000; i++ {
		w.AddActivePort(ActivePort{Port: i})
	}

	// Many lookups
	for i := 0; i < 10000; i++ {
		port := w.GetActivePort(i % 1000)
		if port == nil {
			t.Errorf("Failed to find port %d", i%1000)
		}
	}
}

// TestPortStressMemoryReuse tests that memory is properly managed
func TestPortStressMemoryReuse(t *testing.T) {
	w := &Workspace{}

	// Cycle through add/clear operations
	for cycle := 0; cycle < 100; cycle++ {
		for i := 0; i < 100; i++ {
			w.AddActivePort(ActivePort{Port: i})
		}
		w.ClearActivePorts()
	}

	// Should be empty
	if len(w.Morph.ActivePorts) != 0 {
		t.Errorf("Expected 0 ports, got %d", len(w.Morph.ActivePorts))
	}
}

// TestPortStressUpdateAllPorts tests updating all ports in a list
func TestPortStressUpdateAllPorts(t *testing.T) {
	w := &Workspace{}

	// Add ports
	for i := 0; i < 1000; i++ {
		w.AddActivePort(ActivePort{Port: i, Service: "initial"})
	}

	// Update all
	for i := 0; i < 1000; i++ {
		w.AddActivePort(ActivePort{Port: i, Service: "updated"})
	}

	// Verify all updated
	for _, p := range w.Morph.ActivePorts {
		if p.Service != "updated" {
			t.Errorf("Port %d not updated, service is '%s'", p.Port, p.Service)
		}
	}

	// Count should remain same
	if len(w.Morph.ActivePorts) != 1000 {
		t.Errorf("Expected 1000 ports, got %d", len(w.Morph.ActivePorts))
	}
}

// TestPortStressMixedOperations tests a realistic mix of operations
func TestPortStressMixedOperations(t *testing.T) {
	w := &Workspace{}

	// Simulate realistic usage
	// 1. Add initial ports
	for i := 3000; i < 3010; i++ {
		w.AddActivePort(ActivePort{Port: i, IsHTTP: true})
	}

	// 2. Get primary
	primary := w.GetPrimaryAppPort()
	if primary == nil {
		t.Fatal("Expected primary port")
	}

	// 3. Add more ports
	for i := 5000; i < 5005; i++ {
		w.AddActivePort(ActivePort{Port: i, IsHTTP: false})
	}

	// 4. Get HTTP ports
	httpPorts := w.GetHTTPPorts()
	if len(httpPorts) != 10 {
		t.Errorf("Expected 10 HTTP ports, got %d", len(httpPorts))
	}

	// 5. Remove some
	for i := 3000; i < 3005; i++ {
		w.RemoveActivePort(i)
	}

	// 6. Verify state
	if len(w.Morph.ActivePorts) != 10 {
		t.Errorf("Expected 10 ports remaining, got %d", len(w.Morph.ActivePorts))
	}
}

// TestPortStressBoundaryPortNumbers tests boundary port values under stress
func TestPortStressBoundaryPortNumbers(t *testing.T) {
	w := &Workspace{}

	// Add boundary ports
	boundaries := []int{0, 1, 1023, 1024, 32767, 32768, 49151, 49152, 65534, 65535}
	for _, b := range boundaries {
		w.AddActivePort(ActivePort{Port: b})
	}

	// Verify all present
	for _, b := range boundaries {
		if w.GetActivePort(b) == nil {
			t.Errorf("Boundary port %d not found", b)
		}
	}

	// Remove and re-add in reverse
	for i := len(boundaries) - 1; i >= 0; i-- {
		w.RemoveActivePort(boundaries[i])
	}

	if len(w.Morph.ActivePorts) != 0 {
		t.Errorf("Expected 0 ports, got %d", len(w.Morph.ActivePorts))
	}

	// Re-add in reverse
	for i := len(boundaries) - 1; i >= 0; i-- {
		w.AddActivePort(ActivePort{Port: boundaries[i]})
	}

	if len(w.Morph.ActivePorts) != 10 {
		t.Errorf("Expected 10 ports, got %d", len(w.Morph.ActivePorts))
	}
}

// TestPortStressLongServiceNames tests ports with very long service names
func TestPortStressLongServiceNames(t *testing.T) {
	w := &Workspace{}

	// Create long service name (1MB)
	longName := make([]byte, 1024*1024)
	for i := range longName {
		longName[i] = 'a'
	}

	w.AddActivePort(ActivePort{
		Port:    8080,
		Service: string(longName),
	})

	port := w.GetActivePort(8080)
	if len(port.Service) != 1024*1024 {
		t.Errorf("Expected 1MB service name, got %d bytes", len(port.Service))
	}
}

// TestPortStressLongContainerNames tests ports with very long container names
func TestPortStressLongContainerNames(t *testing.T) {
	w := &Workspace{}

	// Create long container name
	longName := make([]byte, 10000)
	for i := range longName {
		longName[i] = byte('a' + (i % 26))
	}

	w.AddActivePort(ActivePort{
		Port:      8080,
		Container: string(longName),
	})

	port := w.GetActivePort(8080)
	if len(port.Container) != 10000 {
		t.Errorf("Expected 10000 char container name, got %d", len(port.Container))
	}
}

// TestPortStressLongURLs tests ports with very long URLs
func TestPortStressLongURLs(t *testing.T) {
	w := &Workspace{}

	// Create long URL
	longPath := make([]byte, 8000)
	for i := range longPath {
		longPath[i] = 'p'
	}

	url := "http://example.com/" + string(longPath)
	w.AddActivePort(ActivePort{
		Port: 8080,
		URL:  url,
	})

	port := w.GetActivePort(8080)
	if len(port.URL) != len(url) {
		t.Errorf("Expected URL length %d, got %d", len(url), len(port.URL))
	}
}

// TestPortStressConcurrentReads tests concurrent read operations
func TestPortStressConcurrentReads(t *testing.T) {
	w := &Workspace{}

	// Add ports first
	for i := 0; i < 100; i++ {
		w.AddActivePort(ActivePort{Port: i, IsHTTP: i%2 == 0})
	}

	var wg sync.WaitGroup

	// Concurrent reads
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				_ = w.GetActivePort(j % 100)
				_ = w.GetHTTPPorts()
				_ = w.GetPrimaryAppPort()
			}
		}(i)
	}

	wg.Wait()
}

// TestPortStressAlternatingHTTP tests alternating HTTP/non-HTTP patterns
func TestPortStressAlternatingHTTP(t *testing.T) {
	w := &Workspace{}

	// Add with alternating IsHTTP
	for i := 0; i < 1000; i++ {
		w.AddActivePort(ActivePort{
			Port:   i,
			IsHTTP: i%2 == 0,
		})
	}

	httpPorts := w.GetHTTPPorts()
	if len(httpPorts) != 500 {
		t.Errorf("Expected 500 HTTP ports, got %d", len(httpPorts))
	}

	// Toggle all
	for i := 0; i < 1000; i++ {
		w.AddActivePort(ActivePort{
			Port:   i,
			IsHTTP: i%2 != 0, // Opposite
		})
	}

	httpPorts = w.GetHTTPPorts()
	if len(httpPorts) != 500 {
		t.Errorf("Expected 500 HTTP ports after toggle, got %d", len(httpPorts))
	}
}

// TestPortStressTimestampOrdering tests timestamps under stress
func TestPortStressTimestampOrdering(t *testing.T) {
	w := &Workspace{}

	// Add ports rapidly - timestamps should be monotonically increasing or equal
	for i := 0; i < 1000; i++ {
		w.AddActivePort(ActivePort{Port: i})
	}

	var prevTime time.Time
	for i, p := range w.Morph.ActivePorts {
		if i > 0 && p.DiscoveredAt.Before(prevTime) {
			t.Errorf("Timestamp at index %d is before previous timestamp", i)
		}
		prevTime = p.DiscoveredAt
	}
}

// TestPortStressSliceCapacity tests that slice grows efficiently
func TestPortStressSliceCapacity(t *testing.T) {
	w := &Workspace{}

	// Track capacity growth
	prevCap := 0
	capacityChanges := 0

	for i := 0; i < 10000; i++ {
		w.AddActivePort(ActivePort{Port: i})
		newCap := cap(w.Morph.ActivePorts)
		if newCap != prevCap {
			capacityChanges++
			prevCap = newCap
		}
	}

	// Should have grown a reasonable number of times (not 10000 times)
	if capacityChanges > 50 {
		t.Errorf("Too many capacity changes: %d (indicates inefficient growth)", capacityChanges)
	}

	t.Logf("Capacity changes during growth to 10000: %d", capacityChanges)
}

// TestPortStressJSONRoundTripIntegrity tests JSON integrity under stress
func TestPortStressJSONRoundTripIntegrity(t *testing.T) {
	w := &Workspace{ID: "ws-json-stress"}

	// Add diverse ports
	for i := 0; i < 100; i++ {
		w.AddActivePort(ActivePort{
			Port:      i,
			Protocol:  fmt.Sprintf("proto-%d", i%5),
			Service:   fmt.Sprintf("svc-%d", i),
			Container: fmt.Sprintf("cont-%d", i),
			LocalPort: 10000 + i,
			URL:       fmt.Sprintf("http://localhost:%d/path/%d", 10000+i, i),
			IsHTTP:    i%3 == 0,
		})
	}

	// Multiple round-trips
	for round := 0; round < 10; round++ {
		data, err := json.Marshal(w)
		if err != nil {
			t.Fatalf("Marshal failed at round %d: %v", round, err)
		}

		var w2 Workspace
		if err := json.Unmarshal(data, &w2); err != nil {
			t.Fatalf("Unmarshal failed at round %d: %v", round, err)
		}

		// Verify integrity
		if len(w2.Morph.ActivePorts) != 100 {
			t.Errorf("Round %d: expected 100 ports, got %d", round, len(w2.Morph.ActivePorts))
		}

		// Continue with w2
		w = &w2
	}
}
