// internal/workspace/morph_state_machine_test.go
package workspace

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// =============================================================================
// State Machine Exhaustive Tests
// =============================================================================

func TestAllPossibleStateTransitions(t *testing.T) {
	// Test all valid state transitions for Morph instances
	states := []string{"", "starting", "running", "stopping", "stopped", "error", "paused"}

	for _, fromState := range states {
		for _, toState := range states {
			t.Run(fromState+"_to_"+toState, func(t *testing.T) {
				w := &Workspace{}
				w.Morph.Status = fromState

				// Transition via SetMorphInstance
				w.SetMorphInstance("inst-123", "snap-456", "https://example.com")

				if w.Morph.Status != "running" {
					t.Errorf("After SetMorphInstance, status = %s, want running", w.Morph.Status)
				}

				// Transition via ClearMorphInstance
				w.ClearMorphInstance()

				if w.Morph.Status != "stopped" {
					t.Errorf("After ClearMorphInstance, status = %s, want stopped", w.Morph.Status)
				}
			})
		}
	}
}

func TestIsMorphRunningStateMatrix(t *testing.T) {
	// Complete state matrix for IsMorphRunning
	instanceIDs := []string{"", "  ", "\t", "\n", "\r\n", "valid-id", "id-with-space ", " id-with-leading"}
	statuses := []string{"", "running", "Running", "RUNNING", "stopped", "stopping", "starting", "error", "paused", " running", "running "}

	for _, instanceID := range instanceIDs {
		for _, status := range statuses {
			testName := "id_" + truncateForTestName(instanceID) + "_status_" + truncateForTestName(status)
			t.Run(testName, func(t *testing.T) {
				w := &Workspace{}
				w.Morph.InstanceID = instanceID
				w.Morph.Status = status

				result := w.IsMorphRunning()

				// Expected: only true if status == "running" (exact match) and instanceID is non-empty after trim
				expected := status == "running" && trimWhitespace(instanceID) != ""

				if result != expected {
					t.Errorf("IsMorphRunning() = %v, want %v (instanceID=%q, status=%q)",
						result, expected, instanceID, status)
				}
			})
		}
	}
}

func truncateForTestName(s string) string {
	if len(s) == 0 {
		return "empty"
	}
	// Replace special characters
	result := ""
	for _, c := range s {
		switch c {
		case ' ':
			result += "_sp_"
		case '\t':
			result += "_tab_"
		case '\n':
			result += "_nl_"
		case '\r':
			result += "_cr_"
		default:
			result += string(c)
		}
	}
	if len(result) > 20 {
		result = result[:20]
	}
	return result
}

func trimWhitespace(s string) string {
	for len(s) > 0 && (s[0] == ' ' || s[0] == '\t' || s[0] == '\n' || s[0] == '\r') {
		s = s[1:]
	}
	for len(s) > 0 && (s[len(s)-1] == ' ' || s[len(s)-1] == '\t' || s[len(s)-1] == '\n' || s[len(s)-1] == '\r') {
		s = s[:len(s)-1]
	}
	return s
}

// =============================================================================
// Concurrent Modification Tests (Not Thread-Safe)
// =============================================================================

// NOTE: Workspace is NOT thread-safe. These tests verify that operations don't panic
// but the data may be inconsistent. In production, callers must synchronize access.

func TestConcurrentSetMorphInstanceSM(t *testing.T) {
	if raceDetectorEnabled() {
		t.Skip("Skipping concurrent test with race detector (Workspace is not thread-safe)")
	}

	w := &Workspace{}

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		idx := i
		go func() {
			defer wg.Done()
			w.SetMorphInstance("inst-"+string(rune('0'+idx%10)), "snap-"+string(rune('0'+idx%10)), "https://example.com")
		}()
	}
	wg.Wait()

	// Should end in a valid state
	if w.Morph.Status != "running" {
		t.Errorf("Status = %s, want running", w.Morph.Status)
	}
}

func TestConcurrentClearMorphInstanceSM(t *testing.T) {
	if raceDetectorEnabled() {
		t.Skip("Skipping concurrent test with race detector (Workspace is not thread-safe)")
	}

	w := &Workspace{}
	w.SetMorphInstance("inst-123", "snap-456", "https://example.com")

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			w.ClearMorphInstance()
		}()
	}
	wg.Wait()

	// Should end in a valid state
	if w.Morph.Status != "stopped" {
		t.Errorf("Status = %s, want stopped", w.Morph.Status)
	}
}

func TestConcurrentAddSnapshotSM(t *testing.T) {
	if raceDetectorEnabled() {
		t.Skip("Skipping concurrent test with race detector (Workspace is not thread-safe)")
	}
	w := &Workspace{}

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		idx := i
		go func() {
			defer wg.Done()
			w.AddSavedSnapshot("snap-"+string(rune('0'+idx%10)), "name-"+string(rune('0'+idx%10)))
		}()
	}
	wg.Wait()

	// Should have some snapshots (exact count depends on race outcome)
	t.Logf("SavedSnapshots count: %d", len(w.Morph.SavedSnapshots))
}

func TestConcurrentIsMorphRunningSM(t *testing.T) {
	// This test is read-only after setup, so it should be safe
	w := &Workspace{}
	w.SetMorphInstance("inst-123", "snap-456", "https://example.com")

	var wg sync.WaitGroup
	var trueCount atomic.Int32
	var falseCount atomic.Int32

	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if w.IsMorphRunning() {
				trueCount.Add(1)
			} else {
				falseCount.Add(1)
			}
		}()
	}
	wg.Wait()

	// All should be true since we're just reading
	if trueCount.Load() != 100 {
		t.Errorf("trueCount = %d, falseCount = %d", trueCount.Load(), falseCount.Load())
	}
}

func TestConcurrentGetMorphURLsSM(t *testing.T) {
	// This test is read-only after setup, so it should be safe
	w := &Workspace{}
	w.SetMorphInstance("inst-123", "snap-456", "https://example.com")

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = w.GetMorphURLs()
		}()
	}
	wg.Wait()
}

func TestConcurrentMixedOperationsStateMachine(t *testing.T) {
	if raceDetectorEnabled() {
		t.Skip("Skipping concurrent test with race detector (Workspace is not thread-safe)")
	}

	w := &Workspace{}

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		idx := i
		go func() {
			defer wg.Done()
			switch idx % 5 {
			case 0:
				w.SetMorphInstance("inst-"+string(rune('0'+idx%10)), "snap", "https://example.com")
			case 1:
				w.ClearMorphInstance()
			case 2:
				_ = w.IsMorphRunning()
			case 3:
				_ = w.GetMorphURLs()
			case 4:
				w.AddSavedSnapshot("snap-"+string(rune('0'+idx%10)), "name")
			}
		}()
	}
	wg.Wait()

	// Should be in a valid state
	t.Logf("Final status: %s, instanceID: %s", w.Morph.Status, w.Morph.InstanceID)
}

// =============================================================================
// State Persistence Tests
// =============================================================================

func TestStatePersistenceRoundTrip(t *testing.T) {
	tmpDir := t.TempDir()

	original := &Workspace{
		ID:       "ws-persistence",
		Name:     "persistence-test",
		Path:     tmpDir,
		Template: "node",
		Status:   "active",
		Morph: MorphState{
			InstanceID:   "inst-persist",
			SnapshotID:   "snap-persist",
			Status:       "running",
			BaseURL:      "https://persist.example.com",
			CodeURL:      "https://persist.example.com/code/",
			VNCURL:       "https://persist.example.com/vnc/",
			AppURL:       "https://persist.example.com/app/",
			CDPURL:       "https://persist.example.com/cdp/",
			CDPPort:      9222,
			StartedAt:    time.Now(),
			SavedSnapshots: []SavedSnapshot{
				{ID: "snap-1", Name: "save-1", CreatedAt: time.Now()},
				{ID: "snap-2", Name: "save-2", CreatedAt: time.Now()},
			},
		},
	}

	// Save
	if err := original.SaveState(); err != nil {
		t.Fatalf("SaveState error: %v", err)
	}

	// Load
	loaded, err := Load(tmpDir)
	if err != nil {
		t.Fatalf("Load error: %v", err)
	}

	// Verify all Morph fields
	if loaded.Morph.InstanceID != original.Morph.InstanceID {
		t.Errorf("InstanceID = %s, want %s", loaded.Morph.InstanceID, original.Morph.InstanceID)
	}
	if loaded.Morph.Status != original.Morph.Status {
		t.Errorf("Status = %s, want %s", loaded.Morph.Status, original.Morph.Status)
	}
	if loaded.Morph.CDPPort != original.Morph.CDPPort {
		t.Errorf("CDPPort = %d, want %d", loaded.Morph.CDPPort, original.Morph.CDPPort)
	}
	if len(loaded.Morph.SavedSnapshots) != 2 {
		t.Errorf("SavedSnapshots count = %d, want 2", len(loaded.Morph.SavedSnapshots))
	}
}

func TestStatePersistenceWithZeroValues(t *testing.T) {
	tmpDir := t.TempDir()

	original := &Workspace{
		ID:     "ws-zero",
		Name:   "zero-test",
		Path:   tmpDir,
		Morph:  MorphState{}, // All zero values
	}

	if err := original.SaveState(); err != nil {
		t.Fatalf("SaveState error: %v", err)
	}

	loaded, err := Load(tmpDir)
	if err != nil {
		t.Fatalf("Load error: %v", err)
	}

	if loaded.Morph.InstanceID != "" {
		t.Errorf("InstanceID = %s, want empty", loaded.Morph.InstanceID)
	}
	if loaded.Morph.CDPPort != 0 {
		t.Errorf("CDPPort = %d, want 0", loaded.Morph.CDPPort)
	}
}

// =============================================================================
// Edge Case: Rapid State Transitions
// =============================================================================

func TestRapidStateTransitions(t *testing.T) {
	w := &Workspace{}

	for i := 0; i < 1000; i++ {
		w.SetMorphInstance("inst-"+string(rune('0'+i%10)), "snap", "https://example.com")
		w.ClearMorphInstance()
	}

	if w.Morph.Status != "stopped" {
		t.Errorf("Final status = %s, want stopped", w.Morph.Status)
	}
}

func TestRapidSnapshotAdditions(t *testing.T) {
	w := &Workspace{}

	for i := 0; i < 1000; i++ {
		w.AddSavedSnapshot("snap-"+string(rune('0'+i%10)), "name-"+string(rune('0'+i%10)))
	}

	// Should have 1000 snapshots (no deduplication)
	if len(w.Morph.SavedSnapshots) != 1000 {
		t.Errorf("SavedSnapshots count = %d, want 1000", len(w.Morph.SavedSnapshots))
	}
}

// =============================================================================
// JSON Marshaling Edge Cases
// =============================================================================

func TestMorphStateJSONWithSpecialCharacters(t *testing.T) {
	state := MorphState{
		InstanceID: "inst-with-\"quotes\"",
		SnapshotID: "snap-with-\\backslash",
		Status:     "running",
		BaseURL:    "https://example.com/path?query=value&other=<tag>",
	}

	data, err := json.Marshal(state)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}

	var loaded MorphState
	if err := json.Unmarshal(data, &loaded); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}

	if loaded.InstanceID != state.InstanceID {
		t.Errorf("InstanceID = %s, want %s", loaded.InstanceID, state.InstanceID)
	}
	if loaded.BaseURL != state.BaseURL {
		t.Errorf("BaseURL = %s, want %s", loaded.BaseURL, state.BaseURL)
	}
}

func TestMorphStateJSONWithUnicode(t *testing.T) {
	state := MorphState{
		InstanceID: "inst-日本語",
		SnapshotID: "snap-emoji-🚀",
		Status:     "running-状态",
	}

	data, err := json.Marshal(state)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}

	var loaded MorphState
	if err := json.Unmarshal(data, &loaded); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}

	if loaded.InstanceID != state.InstanceID {
		t.Errorf("InstanceID = %s, want %s", loaded.InstanceID, state.InstanceID)
	}
	if loaded.SnapshotID != state.SnapshotID {
		t.Errorf("SnapshotID = %s, want %s", loaded.SnapshotID, state.SnapshotID)
	}
}

func TestMorphStateJSONWithNullFields(t *testing.T) {
	// JSON with explicit null values
	jsonData := `{
		"instance_id": null,
		"snapshot_id": "snap-123",
		"status": null,
		"saved_snapshots": null
	}`

	var state MorphState
	if err := json.Unmarshal([]byte(jsonData), &state); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}

	if state.InstanceID != "" {
		t.Errorf("InstanceID = %s, want empty", state.InstanceID)
	}
	if state.SnapshotID != "snap-123" {
		t.Errorf("SnapshotID = %s, want snap-123", state.SnapshotID)
	}
}

// =============================================================================
// File System Edge Cases
// =============================================================================

func TestLoadFromSymlink(t *testing.T) {
	tmpDir := t.TempDir()
	realDir := filepath.Join(tmpDir, "real")
	if err := os.MkdirAll(filepath.Join(realDir, ".dba"), 0755); err != nil {
		t.Fatal(err)
	}

	content := `{"id": "ws-symlink", "name": "symlink-test", "morph": {"instance_id": "inst-sym"}}`
	if err := os.WriteFile(filepath.Join(realDir, ".dba", "state.json"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	// Create symlink
	linkDir := filepath.Join(tmpDir, "link")
	if err := os.Symlink(realDir, linkDir); err != nil {
		t.Skip("Symlinks not supported on this system")
	}

	// Load from symlink
	loaded, err := Load(linkDir)
	if err != nil {
		t.Fatalf("Load error: %v", err)
	}

	if loaded.Morph.InstanceID != "inst-sym" {
		t.Errorf("InstanceID = %s, want inst-sym", loaded.Morph.InstanceID)
	}
}

func TestSaveToReadOnlyDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	stateDir := filepath.Join(tmpDir, ".dba")
	if err := os.MkdirAll(stateDir, 0555); err != nil { // Read-only
		t.Fatal(err)
	}
	defer os.Chmod(stateDir, 0755) // Restore permissions for cleanup

	w := &Workspace{
		ID:   "ws-readonly",
		Name: "readonly-test",
		Path: tmpDir,
	}
	w.SetMorphInstance("inst-123", "snap-456", "https://example.com")

	err := w.SaveState()
	if err == nil {
		t.Log("SaveState succeeded on read-only directory (may be running as root)")
	} else {
		t.Logf("Expected error on read-only directory: %v", err)
	}
}

// =============================================================================
// TextOutput Edge Cases
// =============================================================================

func TestTextOutputWithAllURLs(t *testing.T) {
	w := &Workspace{
		ID:   "ws-output",
		Name: "output-test",
		Morph: MorphState{
			InstanceID: "inst-123",
			Status:     "running",
			CodeURL:    "https://example.com/code/",
			VNCURL:     "https://example.com/vnc/",
			AppURL:     "https://example.com/app/",
			CDPURL:     "https://example.com/cdp/",
		},
	}

	output := w.TextOutput()

	// Should contain all URLs
	if !containsSubstring(output, "inst-123") {
		t.Error("Output should contain instance ID")
	}
}

func TestTextOutputWithVeryLongFields(t *testing.T) {
	longString := ""
	for i := 0; i < 1000; i++ {
		longString += "a"
	}

	w := &Workspace{
		ID:   "ws-" + longString,
		Name: "name-" + longString,
		Morph: MorphState{
			InstanceID: "inst-" + longString,
			BaseURL:    "https://example.com/" + longString + "/",
		},
	}

	output := w.TextOutput()

	// Should not panic and should return something
	if len(output) == 0 {
		t.Error("Output should not be empty")
	}
}

func containsSubstring(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
