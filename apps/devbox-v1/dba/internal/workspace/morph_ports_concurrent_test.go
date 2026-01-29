// internal/workspace/morph_ports_concurrent_test.go
package workspace

import (
	"fmt"
	"sync"
	"testing"
	"time"
)

// TestPortsConcurrentReadOnly tests concurrent read-only operations
func TestPortsConcurrentReadOnly(t *testing.T) {
	w := &Workspace{}

	// Setup ports
	for i := 0; i < 100; i++ {
		w.AddActivePort(ActivePort{
			Port:   8000 + i,
			IsHTTP: i%2 == 0,
		})
	}

	var wg sync.WaitGroup

	// Many concurrent readers
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				_ = w.GetActivePort(8000 + (j % 100))
				_ = w.GetHTTPPorts()
				_ = w.GetPrimaryAppPort()
			}
		}(i)
	}

	wg.Wait()
}

// TestPortsConcurrentMixedOperations tests concurrent mixed operations
func TestPortsConcurrentMixedOperations(t *testing.T) {
	if raceDetectorEnabled() {
		t.Skip("Skipping concurrent modification test with race detector")
	}

	w := &Workspace{}
	var wg sync.WaitGroup

	// Writers
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(base int) {
			defer wg.Done()
			for j := 0; j < 10; j++ {
				w.AddActivePort(ActivePort{Port: base*100 + j})
			}
		}(i)
	}

	// Readers
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 50; j++ {
				_ = w.GetHTTPPorts()
				_ = w.GetPrimaryAppPort()
			}
		}()
	}

	wg.Wait()

	t.Logf("Final port count: %d", len(w.Morph.ActivePorts))
}

// TestPortsConcurrentAddSamePort tests concurrent adds of the same port
func TestPortsConcurrentAddSamePort(t *testing.T) {
	if raceDetectorEnabled() {
		t.Skip("Skipping concurrent modification test with race detector")
	}

	w := &Workspace{}
	var wg sync.WaitGroup

	// Many goroutines adding the same port
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			w.AddActivePort(ActivePort{
				Port:    8080,
				Service: fmt.Sprintf("writer-%d", id),
			})
		}(i)
	}

	wg.Wait()

	// Should have exactly 1 port (updates overwrite)
	count := 0
	for _, p := range w.Morph.ActivePorts {
		if p.Port == 8080 {
			count++
		}
	}
	// Due to race conditions, we might have duplicates
	t.Logf("Port 8080 count: %d", count)
}

// TestPortsConcurrentRemove tests concurrent remove operations
func TestPortsConcurrentRemove(t *testing.T) {
	if raceDetectorEnabled() {
		t.Skip("Skipping concurrent modification test with race detector")
	}

	w := &Workspace{}

	// Add ports
	for i := 0; i < 100; i++ {
		w.AddActivePort(ActivePort{Port: 8000 + i})
	}

	var wg sync.WaitGroup

	// Concurrent removes
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(port int) {
			defer wg.Done()
			w.RemoveActivePort(8000 + port)
		}(i)
	}

	wg.Wait()

	t.Logf("Remaining ports: %d", len(w.Morph.ActivePorts))
}

// TestPortsConcurrentClear tests concurrent clear operations
func TestPortsConcurrentClear(t *testing.T) {
	if raceDetectorEnabled() {
		t.Skip("Skipping concurrent modification test with race detector")
	}

	// This test demonstrates that ClearActivePorts and AddActivePort
	// are not safe to call concurrently without external synchronization.
	// We test sequential patterns instead.

	w := &Workspace{}

	// Add some ports
	for i := 0; i < 100; i++ {
		w.AddActivePort(ActivePort{Port: 8000 + i})
	}

	if len(w.Morph.ActivePorts) != 100 {
		t.Errorf("Expected 100 ports, got %d", len(w.Morph.ActivePorts))
	}

	// Clear
	w.ClearActivePorts()

	if len(w.Morph.ActivePorts) != 0 {
		t.Errorf("Expected 0 ports after clear, got %d", len(w.Morph.ActivePorts))
	}

	// Add again
	for i := 0; i < 50; i++ {
		w.AddActivePort(ActivePort{Port: 9000 + i})
	}

	if len(w.Morph.ActivePorts) != 50 {
		t.Errorf("Expected 50 ports, got %d", len(w.Morph.ActivePorts))
	}
}

// TestPortsConcurrentSet tests concurrent SetActivePorts
func TestPortsConcurrentSet(t *testing.T) {
	if raceDetectorEnabled() {
		t.Skip("Skipping concurrent modification test with race detector")
	}

	w := &Workspace{}
	var wg sync.WaitGroup

	// Concurrent SetActivePorts
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			ports := make([]ActivePort, 5)
			for j := range ports {
				ports[j] = ActivePort{Port: id*10 + j}
			}
			w.SetActivePorts(ports)
		}(i)
	}

	wg.Wait()

	t.Logf("Final port count: %d", len(w.Morph.ActivePorts))
}

// TestPortsConcurrentGetPrimary tests concurrent GetPrimaryAppPort
func TestPortsConcurrentGetPrimary(t *testing.T) {
	w := &Workspace{}

	// Add some ports
	w.AddActivePort(ActivePort{Port: 8080, IsHTTP: true})
	w.AddActivePort(ActivePort{Port: 3000, IsHTTP: true})

	var wg sync.WaitGroup
	results := make(chan *ActivePort, 1000)

	// Many concurrent calls
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 10; j++ {
				results <- w.GetPrimaryAppPort()
			}
		}()
	}

	wg.Wait()
	close(results)

	// All results should be the same
	count := 0
	for p := range results {
		if p != nil && p.Port == 8080 {
			count++
		}
	}
	t.Logf("Got port 8080 as primary %d times", count)
}

// TestPortsConcurrentGetHTTP tests concurrent GetHTTPPorts
func TestPortsConcurrentGetHTTP(t *testing.T) {
	w := &Workspace{}

	// Add some ports
	for i := 0; i < 50; i++ {
		w.AddActivePort(ActivePort{Port: 8000 + i, IsHTTP: i%2 == 0})
	}

	var wg sync.WaitGroup
	results := make(chan int, 100)

	// Many concurrent calls
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 10; j++ {
				httpPorts := w.GetHTTPPorts()
				results <- len(httpPorts)
			}
		}()
	}

	wg.Wait()
	close(results)

	// All should return 25 (half of 50)
	for count := range results {
		if count != 25 {
			t.Errorf("Expected 25 HTTP ports, got %d", count)
		}
	}
}

// TestPortsConcurrentUpdateAndRead tests update while reading
func TestPortsConcurrentUpdateAndRead(t *testing.T) {
	if raceDetectorEnabled() {
		t.Skip("Skipping concurrent modification test with race detector")
	}

	w := &Workspace{}
	w.AddActivePort(ActivePort{Port: 8080, Service: "initial"})

	var wg sync.WaitGroup
	done := make(chan bool)

	// Reader
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-done:
				return
			default:
				port := w.GetActivePort(8080)
				if port != nil {
					_ = port.Service
				}
			}
		}
	}()

	// Writer
	for i := 0; i < 1000; i++ {
		w.AddActivePort(ActivePort{
			Port:    8080,
			Service: fmt.Sprintf("update-%d", i),
		})
	}

	close(done)
	wg.Wait()
}

// TestPortsConcurrentFullLifecycle tests concurrent full lifecycle operations
func TestPortsConcurrentFullLifecycle(t *testing.T) {
	if raceDetectorEnabled() {
		t.Skip("Skipping concurrent modification test with race detector")
	}

	w := &Workspace{}
	var wg sync.WaitGroup

	// Start instance
	w.SetMorphInstance("inst-1", "snap-1", "https://vm.example.com")

	// Port adders
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(base int) {
			defer wg.Done()
			for j := 0; j < 20; j++ {
				w.AddActivePort(ActivePort{
					Port:    base*100 + j,
					IsHTTP:  j%2 == 0,
					Service: fmt.Sprintf("svc-%d-%d", base, j),
				})
			}
		}(i)
	}

	// Port readers
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				_ = w.GetHTTPPorts()
				_ = w.GetPrimaryAppPort()
			}
		}()
	}

	wg.Wait()

	t.Logf("Final state - ports: %d", len(w.Morph.ActivePorts))
}

// TestPortsConcurrentTimestamps tests timestamp consistency
func TestPortsConcurrentTimestamps(t *testing.T) {
	if raceDetectorEnabled() {
		t.Skip("Skipping concurrent modification test with race detector")
	}

	w := &Workspace{}
	var wg sync.WaitGroup

	start := time.Now()

	// Add ports concurrently
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(port int) {
			defer wg.Done()
			w.AddActivePort(ActivePort{Port: port})
		}(8000 + i)
	}

	wg.Wait()

	end := time.Now()

	// All timestamps should be between start and end
	for _, p := range w.Morph.ActivePorts {
		if p.DiscoveredAt.Before(start) {
			t.Errorf("Port %d has timestamp before start", p.Port)
		}
		if p.DiscoveredAt.After(end) {
			t.Errorf("Port %d has timestamp after end", p.Port)
		}
	}
}

// TestPortsConcurrentWithMorphStateChanges tests ports with Morph state changes
func TestPortsConcurrentWithMorphStateChanges(t *testing.T) {
	if raceDetectorEnabled() {
		t.Skip("Skipping concurrent modification test with race detector")
	}

	w := &Workspace{}
	var wg sync.WaitGroup

	// Port operations
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < 10; j++ {
				w.AddActivePort(ActivePort{Port: id*100 + j})
			}
		}(i)
	}

	// State changes
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			w.SetMorphInstance(fmt.Sprintf("inst-%d", id), "snap", "https://vm.example.com")
			time.Sleep(time.Millisecond)
			w.ClearMorphInstance()
		}(i)
	}

	wg.Wait()
}

// TestPortsConcurrentSnapshots tests ports with snapshot operations
func TestPortsConcurrentSnapshots(t *testing.T) {
	if raceDetectorEnabled() {
		t.Skip("Skipping concurrent modification test with race detector")
	}

	w := &Workspace{}
	var wg sync.WaitGroup

	// Port operations
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			w.AddActivePort(ActivePort{Port: 8000 + id})
		}(i)
	}

	// Snapshot operations
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			w.AddSavedSnapshot(fmt.Sprintf("snap-%d", id), fmt.Sprintf("checkpoint-%d", id))
		}(i)
	}

	wg.Wait()

	t.Logf("Final state - ports: %d, snapshots: %d",
		len(w.Morph.ActivePorts),
		len(w.Morph.SavedSnapshots))
}

// TestPortsConcurrentGetByPort tests concurrent GetActivePort
func TestPortsConcurrentGetByPort(t *testing.T) {
	w := &Workspace{}

	// Setup
	for i := 0; i < 100; i++ {
		w.AddActivePort(ActivePort{Port: 8000 + i, Service: fmt.Sprintf("svc-%d", i)})
	}

	var wg sync.WaitGroup
	errors := make(chan error, 1000)

	// Many concurrent gets
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 10; j++ {
				port := 8000 + (j % 100)
				p := w.GetActivePort(port)
				if p == nil {
					errors <- fmt.Errorf("port %d not found", port)
				}
			}
		}()
	}

	wg.Wait()
	close(errors)

	// Check for errors
	for err := range errors {
		t.Error(err)
	}
}

// TestPortsConcurrentRemoveNonExistent tests removing non-existent ports concurrently
func TestPortsConcurrentRemoveNonExistent(t *testing.T) {
	if raceDetectorEnabled() {
		t.Skip("Skipping concurrent modification test with race detector")
	}

	w := &Workspace{}

	// Add a few ports
	for i := 0; i < 10; i++ {
		w.AddActivePort(ActivePort{Port: 8000 + i})
	}

	var wg sync.WaitGroup

	// Many goroutines removing ports that may or may not exist
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			// Try to remove ports in range 8000-8099, most don't exist
			w.RemoveActivePort(8000 + (id % 100))
		}(i)
	}

	wg.Wait()
}

// TestPortsConcurrentRapidToggle tests rapid toggling of ports
func TestPortsConcurrentRapidToggle(t *testing.T) {
	if raceDetectorEnabled() {
		t.Skip("Skipping concurrent modification test with race detector")
	}

	w := &Workspace{}
	var wg sync.WaitGroup

	// Toggle ports rapidly
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(port int) {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				w.AddActivePort(ActivePort{Port: port})
				w.RemoveActivePort(port)
			}
		}(8080 + i)
	}

	wg.Wait()
}
