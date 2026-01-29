// internal/workspace/morph_stress_test.go
package workspace

import (
	"encoding/json"
	"sync"
	"testing"
	"time"
)

// =============================================================================
// Stress Tests
// =============================================================================

func TestRapidSetClearCycle(t *testing.T) {
	w := &Workspace{}

	for i := 0; i < 10000; i++ {
		w.SetMorphInstance("inst-"+string(rune('0'+i%10)), "snap-"+string(rune('0'+i%10)), "https://example.com")
		if !w.IsMorphRunning() {
			t.Fatalf("Expected running at iteration %d", i)
		}
		w.ClearMorphInstance()
		if w.IsMorphRunning() {
			t.Fatalf("Expected not running at iteration %d after clear", i)
		}
	}
}

func TestRapidSnapshotAdd(t *testing.T) {
	w := &Workspace{}

	for i := 0; i < 10000; i++ {
		w.AddSavedSnapshot("snap-"+string(rune('0'+i%10)), "snapshot-"+string(rune('0'+i%10)))
	}

	if len(w.Morph.SavedSnapshots) != 10000 {
		t.Errorf("SavedSnapshots count = %d, want 10000", len(w.Morph.SavedSnapshots))
	}
}

func TestRapidIsMorphRunningCheck(t *testing.T) {
	w := &Workspace{}
	w.SetMorphInstance("inst-123", "snap-456", "https://example.com")

	for i := 0; i < 100000; i++ {
		if !w.IsMorphRunning() {
			t.Fatalf("Expected running at iteration %d", i)
		}
	}
}

func TestRapidGetMorphURLs(t *testing.T) {
	w := &Workspace{}
	w.SetMorphInstance("inst-123", "snap-456", "https://example.com")

	for i := 0; i < 100000; i++ {
		urls := w.GetMorphURLs()
		if len(urls) != 4 { // code, vnc, app, cdp
			t.Fatalf("Expected 4 URLs at iteration %d, got %d", i, len(urls))
		}
	}
}

func TestConcurrentMixedOperations(t *testing.T) {
	// NOTE: Workspace is NOT thread-safe. This test verifies that
	// operations don't panic, but the data may be inconsistent.
	// In production, callers must synchronize access.
	// Skip this test with race detector as it will (correctly) report races.
	if raceDetectorEnabled() {
		t.Skip("Skipping concurrent test with race detector (Workspace is not thread-safe)")
	}

	w := &Workspace{}
	var wg sync.WaitGroup

	// Mix of operations running concurrently
	for i := 0; i < 100; i++ {
		wg.Add(5)

		// Set instance
		go func(idx int) {
			defer wg.Done()
			w.SetMorphInstance("inst-"+string(rune('0'+idx%10)), "snap-"+string(rune('0'+idx%10)), "https://example.com")
		}(i)

		// Clear instance
		go func() {
			defer wg.Done()
			w.ClearMorphInstance()
		}()

		// Check running
		go func() {
			defer wg.Done()
			_ = w.IsMorphRunning()
		}()

		// Get URLs
		go func() {
			defer wg.Done()
			_ = w.GetMorphURLs()
		}()

		// Add snapshot
		go func(idx int) {
			defer wg.Done()
			w.AddSavedSnapshot("snap-"+string(rune('0'+idx%10)), "snapshot-"+string(rune('0'+idx%10)))
		}(i)
	}

	wg.Wait()
	t.Logf("Final state: InstanceID=%q, Status=%q, Snapshots=%d",
		w.Morph.InstanceID, w.Morph.Status, len(w.Morph.SavedSnapshots))
}


// =============================================================================
// JSON Serialization Stress Tests
// =============================================================================

func TestMorphStateJSONRoundTripStress(t *testing.T) {
	state := MorphState{
		InstanceID: "inst-stress",
		SnapshotID: "snap-stress",
		Status:     "running",
		BaseURL:    "https://example.com",
		CodeURL:    "https://example.com/code/",
		VNCURL:     "https://example.com/vnc/",
		AppURL:     "https://example.com/app/",
		CDPURL:     "https://example.com/cdp/",
		CDPPort:    9222,
		StartedAt:  time.Now(),
	}

	for i := 0; i < 10000; i++ {
		data, err := json.Marshal(state)
		if err != nil {
			t.Fatalf("Marshal failed at iteration %d: %v", i, err)
		}

		var loaded MorphState
		if err := json.Unmarshal(data, &loaded); err != nil {
			t.Fatalf("Unmarshal failed at iteration %d: %v", i, err)
		}

		if loaded.InstanceID != state.InstanceID {
			t.Fatalf("InstanceID mismatch at iteration %d", i)
		}
	}
}

func TestConcurrentJSONSerialization(t *testing.T) {
	w := &Workspace{
		ID:   "ws-123",
		Name: "test",
		Morph: MorphState{
			InstanceID: "inst-concurrent",
			Status:     "running",
		},
	}

	var wg sync.WaitGroup
	errors := make(chan error, 1000)

	for i := 0; i < 1000; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			data, err := json.Marshal(w)
			if err != nil {
				errors <- err
				return
			}

			var loaded Workspace
			if err := json.Unmarshal(data, &loaded); err != nil {
				errors <- err
				return
			}
		}()
	}

	wg.Wait()
	close(errors)

	for err := range errors {
		if err != nil {
			t.Errorf("JSON error: %v", err)
		}
	}
}

// =============================================================================
// Memory and State Tests
// =============================================================================

func TestManySnapshotsMemory(t *testing.T) {
	w := &Workspace{}

	// Add 100000 snapshots
	for i := 0; i < 100000; i++ {
		w.AddSavedSnapshot("snap-"+string(rune('0'+i%10))+string(rune('0'+(i/10)%10)), "snapshot-"+string(rune('0'+i%10)))
	}

	if len(w.Morph.SavedSnapshots) != 100000 {
		t.Errorf("SavedSnapshots count = %d, want 100000", len(w.Morph.SavedSnapshots))
	}

	// Verify we can still serialize
	data, err := json.Marshal(w.Morph)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}

	var loaded MorphState
	if err := json.Unmarshal(data, &loaded); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}

	if len(loaded.SavedSnapshots) != 100000 {
		t.Errorf("Loaded snapshots count = %d, want 100000", len(loaded.SavedSnapshots))
	}
}

func TestRepeatedSetMorphInstance(t *testing.T) {
	w := &Workspace{}

	// Set the same instance many times
	for i := 0; i < 10000; i++ {
		w.SetMorphInstance("inst-same", "snap-same", "https://example.com")
	}

	// Should still work correctly
	if !w.IsMorphRunning() {
		t.Error("Expected running after repeated SetMorphInstance")
	}
	if w.Morph.InstanceID != "inst-same" {
		t.Errorf("InstanceID = %s, want inst-same", w.Morph.InstanceID)
	}
}

func TestRepeatedClearMorphInstance(t *testing.T) {
	w := &Workspace{}
	w.SetMorphInstance("inst-123", "snap-456", "https://example.com")

	// Clear many times
	for i := 0; i < 10000; i++ {
		w.ClearMorphInstance()
	}

	// Should still be cleared correctly
	if w.IsMorphRunning() {
		t.Error("Expected not running after repeated ClearMorphInstance")
	}
	if w.Morph.InstanceID != "" {
		t.Errorf("InstanceID = %s, want empty", w.Morph.InstanceID)
	}
	if w.Morph.Status != "stopped" {
		t.Errorf("Status = %s, want stopped", w.Morph.Status)
	}
}

// =============================================================================
// Edge Case Combinations
// =============================================================================

func TestSetThenImmediateCheck(t *testing.T) {
	w := &Workspace{}

	for i := 0; i < 1000; i++ {
		w.SetMorphInstance("inst-"+string(rune('0'+i%10)), "snap-"+string(rune('0'+i%10)), "https://example.com")

		// Immediately check
		if !w.IsMorphRunning() {
			t.Fatalf("Expected running immediately after SetMorphInstance at iteration %d", i)
		}
		if w.Morph.InstanceID != "inst-"+string(rune('0'+i%10)) {
			t.Fatalf("InstanceID mismatch at iteration %d", i)
		}
	}
}

func TestClearThenImmediateCheck(t *testing.T) {
	w := &Workspace{}

	for i := 0; i < 1000; i++ {
		w.SetMorphInstance("inst-"+string(rune('0'+i%10)), "snap-"+string(rune('0'+i%10)), "https://example.com")
		w.ClearMorphInstance()

		// Immediately check
		if w.IsMorphRunning() {
			t.Fatalf("Expected not running immediately after ClearMorphInstance at iteration %d", i)
		}
		if w.Morph.InstanceID != "" {
			t.Fatalf("InstanceID should be empty at iteration %d", i)
		}
	}
}

func TestTextOutputStress(t *testing.T) {
	w := &Workspace{
		ID:   "ws-123",
		Name: "test",
		Morph: MorphState{
			InstanceID: "inst-456",
			Status:     "running",
			CodeURL:    "https://example.com/code/",
		},
	}

	for i := 0; i < 10000; i++ {
		output := w.TextOutput()
		if output == "" {
			t.Fatalf("Empty output at iteration %d", i)
		}
	}
}

func TestGetSavedSnapshotStress(t *testing.T) {
	w := &Workspace{}

	// Add 100 snapshots
	for i := 0; i < 100; i++ {
		w.AddSavedSnapshot("snap-"+string(rune('0'+i%10)), "snapshot-"+string(rune('0'+i%10)))
	}

	// Get snapshots 100000 times
	for i := 0; i < 100000; i++ {
		name := "snapshot-" + string(rune('0'+i%10))
		snap := w.GetSavedSnapshot(name)
		if snap == nil {
			t.Fatalf("GetSavedSnapshot(%s) returned nil at iteration %d", name, i)
		}
	}
}

// =============================================================================
// Zero Value and Nil Tests
// =============================================================================

func TestZeroValueWorkspace(t *testing.T) {
	var w Workspace

	// All Morph fields should be zero
	if w.Morph.InstanceID != "" {
		t.Errorf("Zero InstanceID = %q, want empty", w.Morph.InstanceID)
	}
	if w.IsMorphRunning() {
		t.Error("Zero value workspace should not be running")
	}
	if len(w.GetMorphURLs()) != 0 {
		t.Errorf("Zero value workspace should have no URLs")
	}
}

func TestZeroValueMorphState(t *testing.T) {
	var state MorphState

	if state.InstanceID != "" {
		t.Errorf("Zero InstanceID = %q, want empty", state.InstanceID)
	}
	if state.Status != "" {
		t.Errorf("Zero Status = %q, want empty", state.Status)
	}
	if state.CDPPort != 0 {
		t.Errorf("Zero CDPPort = %d, want 0", state.CDPPort)
	}
	if !state.StartedAt.IsZero() {
		t.Errorf("Zero StartedAt should be zero")
	}
	if state.SavedSnapshots != nil {
		t.Errorf("Zero SavedSnapshots should be nil")
	}
}

func TestZeroValueSavedSnapshot(t *testing.T) {
	var snap SavedSnapshot

	if snap.ID != "" {
		t.Errorf("Zero ID = %q, want empty", snap.ID)
	}
	if snap.Name != "" {
		t.Errorf("Zero Name = %q, want empty", snap.Name)
	}
	if !snap.CreatedAt.IsZero() {
		t.Errorf("Zero CreatedAt should be zero")
	}
}

// =============================================================================
// Immutability Tests
// =============================================================================

func TestGetMorphURLsImmutability(t *testing.T) {
	w := &Workspace{}
	w.SetMorphInstance("inst-123", "snap-456", "https://example.com")

	urls1 := w.GetMorphURLs()
	urls2 := w.GetMorphURLs()

	// Modify urls1
	urls1["code"] = "modified"

	// urls2 should be unchanged
	if urls2["code"] == "modified" {
		t.Error("GetMorphURLs should return independent maps")
	}
}

func TestSavedSnapshotsSliceGrowth(t *testing.T) {
	w := &Workspace{}

	// Track capacity growth
	var lastCap int
	capacityChanges := 0

	for i := 0; i < 10000; i++ {
		w.AddSavedSnapshot("snap-"+string(rune('0'+i%10)), "snapshot-"+string(rune('0'+i%10)))
		currentCap := cap(w.Morph.SavedSnapshots)
		if currentCap != lastCap {
			capacityChanges++
			lastCap = currentCap
		}
	}

	t.Logf("Final length: %d, Final capacity: %d, Capacity changes: %d",
		len(w.Morph.SavedSnapshots), cap(w.Morph.SavedSnapshots), capacityChanges)
}

// =============================================================================
// Time-Related Tests
// =============================================================================

func TestStartedAtPrecision(t *testing.T) {
	w := &Workspace{}

	for i := 0; i < 100; i++ {
		before := time.Now()
		w.SetMorphInstance("inst-"+string(rune('0'+i%10)), "snap-"+string(rune('0'+i%10)), "https://example.com")
		after := time.Now()

		if w.Morph.StartedAt.Before(before) {
			t.Errorf("StartedAt is before the operation at iteration %d", i)
		}
		if w.Morph.StartedAt.After(after) {
			t.Errorf("StartedAt is after the operation at iteration %d", i)
		}
	}
}

func TestSnapshotCreatedAtPrecision(t *testing.T) {
	w := &Workspace{}

	for i := 0; i < 100; i++ {
		before := time.Now()
		w.AddSavedSnapshot("snap-"+string(rune('0'+i%10)), "snapshot-"+string(rune('0'+i%10)))
		after := time.Now()

		snap := w.Morph.SavedSnapshots[len(w.Morph.SavedSnapshots)-1]
		if snap.CreatedAt.Before(before) {
			t.Errorf("CreatedAt is before the operation at iteration %d", i)
		}
		if snap.CreatedAt.After(after) {
			t.Errorf("CreatedAt is after the operation at iteration %d", i)
		}
	}
}
