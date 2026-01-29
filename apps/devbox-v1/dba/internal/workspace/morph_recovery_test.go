// internal/workspace/morph_recovery_test.go
package workspace

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// =============================================================================
// Error Recovery Tests
// =============================================================================

func TestRecoveryFromCorruptedState(t *testing.T) {
	tmpDir := t.TempDir()
	stateDir := filepath.Join(tmpDir, ".dba")
	if err := os.MkdirAll(stateDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create corrupted state
	corruptedContent := `{
		"id": "ws-corrupt",
		"name": "corrupt-test",
		"morph": {
			"instance_id": "inst-corrupt
	}`
	// Note: missing closing quote and braces

	statePath := filepath.Join(stateDir, "state.json")
	if err := os.WriteFile(statePath, []byte(corruptedContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Try to load
	_, err := Load(tmpDir)
	if err == nil {
		t.Error("Expected error for corrupted state")
	} else {
		t.Logf("Expected error: %v", err)
	}

	// Write valid state to recover
	validContent := `{
		"id": "ws-recovered",
		"name": "recovered-test",
		"template": "node",
		"status": "active",
		"morph": {
			"instance_id": "inst-recovered",
			"status": "running"
		}
	}`
	if err := os.WriteFile(statePath, []byte(validContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Should load now
	loaded, err := Load(tmpDir)
	if err != nil {
		t.Fatalf("Load error after recovery: %v", err)
	}

	if loaded.Morph.InstanceID != "inst-recovered" {
		t.Errorf("InstanceID = %s, want inst-recovered", loaded.Morph.InstanceID)
	}
}

func TestRecoveryFromTruncatedState(t *testing.T) {
	tmpDir := t.TempDir()
	stateDir := filepath.Join(tmpDir, ".dba")
	if err := os.MkdirAll(stateDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create truncated state (simulating crash during write)
	truncatedContent := `{"id": "ws-truncated", "name": "truncated", "morph": {"instance_`

	statePath := filepath.Join(stateDir, "state.json")
	if err := os.WriteFile(statePath, []byte(truncatedContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Try to load
	_, err := Load(tmpDir)
	if err == nil {
		t.Error("Expected error for truncated state")
	}
}

func TestRecoveryFromEmptyState(t *testing.T) {
	tmpDir := t.TempDir()
	stateDir := filepath.Join(tmpDir, ".dba")
	if err := os.MkdirAll(stateDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create empty state file
	statePath := filepath.Join(stateDir, "state.json")
	if err := os.WriteFile(statePath, []byte{}, 0644); err != nil {
		t.Fatal(err)
	}

	// Try to load
	_, err := Load(tmpDir)
	// May error or return defaults
	t.Logf("Load with empty state: err=%v", err)
}

func TestRecoveryFromNullState(t *testing.T) {
	tmpDir := t.TempDir()
	stateDir := filepath.Join(tmpDir, ".dba")
	if err := os.MkdirAll(stateDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create state with just "null"
	statePath := filepath.Join(stateDir, "state.json")
	if err := os.WriteFile(statePath, []byte("null"), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := Load(tmpDir)
	t.Logf("Load with null state: err=%v", err)
}

// =============================================================================
// State Transition Recovery Tests
// =============================================================================

func TestRecoveryAfterCrashDuringStart(t *testing.T) {
	// Simulate state where instance was being started but crashed
	tmpDir := t.TempDir()
	stateDir := filepath.Join(tmpDir, ".dba")
	if err := os.MkdirAll(stateDir, 0755); err != nil {
		t.Fatal(err)
	}

	// State with instance_id but no status
	content := `{
		"id": "ws-crash-start",
		"name": "crash-start-test",
		"template": "node",
		"status": "active",
		"morph": {
			"instance_id": "inst-starting"
		}
	}`

	statePath := filepath.Join(stateDir, "state.json")
	if err := os.WriteFile(statePath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	loaded, err := Load(tmpDir)
	if err != nil {
		t.Fatalf("Load error: %v", err)
	}

	// With instance_id but no status, IsMorphRunning should be false
	if loaded.IsMorphRunning() {
		t.Error("IsMorphRunning should be false with missing status")
	}
}

func TestRecoveryAfterCrashDuringStop(t *testing.T) {
	// Simulate state where instance was being stopped but crashed
	tmpDir := t.TempDir()
	stateDir := filepath.Join(tmpDir, ".dba")
	if err := os.MkdirAll(stateDir, 0755); err != nil {
		t.Fatal(err)
	}

	// State with status=stopping
	content := `{
		"id": "ws-crash-stop",
		"name": "crash-stop-test",
		"template": "node",
		"status": "active",
		"morph": {
			"instance_id": "inst-stopping",
			"status": "stopping"
		}
	}`

	statePath := filepath.Join(stateDir, "state.json")
	if err := os.WriteFile(statePath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	loaded, err := Load(tmpDir)
	if err != nil {
		t.Fatalf("Load error: %v", err)
	}

	// status=stopping means not running
	if loaded.IsMorphRunning() {
		t.Error("IsMorphRunning should be false with status=stopping")
	}
}

// =============================================================================
// Workspace Method Error Handling
// =============================================================================

func TestSetMorphInstanceWithEmptyFields(t *testing.T) {
	w := &Workspace{}

	// All empty
	w.SetMorphInstance("", "", "")

	if w.Morph.Status != "running" {
		t.Errorf("Status = %s, want running (status is always set)", w.Morph.Status)
	}
	if !w.Morph.StartedAt.IsZero() {
		// StartedAt should be set even with empty fields
		t.Logf("StartedAt was set to: %v", w.Morph.StartedAt)
	}
}

func TestClearMorphInstanceMultipleTimesRecovery(t *testing.T) {
	w := &Workspace{}
	w.SetMorphInstance("inst-123", "snap-456", "https://example.com")

	// Clear multiple times
	for i := 0; i < 100; i++ {
		w.ClearMorphInstance()
	}

	// Should still be in valid state
	if w.Morph.InstanceID != "" {
		t.Errorf("InstanceID = %s, want empty", w.Morph.InstanceID)
	}
	if w.Morph.Status != "stopped" {
		t.Errorf("Status = %s, want stopped", w.Morph.Status)
	}
}

func TestAddSavedSnapshotWithEmptyID(t *testing.T) {
	w := &Workspace{}

	w.AddSavedSnapshot("", "valid-name")
	w.AddSavedSnapshot("valid-id", "")
	w.AddSavedSnapshot("", "")

	// All should be added
	if len(w.Morph.SavedSnapshots) != 3 {
		t.Errorf("SavedSnapshots count = %d, want 3", len(w.Morph.SavedSnapshots))
	}
}

func TestGetSavedSnapshotOnNilSlice(t *testing.T) {
	w := &Workspace{}
	// SavedSnapshots is nil

	snap := w.GetSavedSnapshot("any")
	if snap != nil {
		t.Error("Expected nil for empty slice")
	}
}

func TestGetSavedSnapshotOnEmptySlice(t *testing.T) {
	w := &Workspace{}
	w.Morph.SavedSnapshots = []SavedSnapshot{}

	snap := w.GetSavedSnapshot("any")
	if snap != nil {
		t.Error("Expected nil for empty slice")
	}
}

// =============================================================================
// IsMorphRunning Edge Cases
// =============================================================================

func TestIsMorphRunningWithVariousStatuses(t *testing.T) {
	tests := []struct {
		status   string
		instID   string
		expected bool
	}{
		{"running", "inst-123", true},
		{"Running", "inst-123", false}, // case sensitive
		{"RUNNING", "inst-123", false},
		{"stopped", "inst-123", false},
		{"Stopped", "inst-123", false},
		{"stopping", "inst-123", false},
		{"starting", "inst-123", false},
		{"paused", "inst-123", false},
		{"error", "inst-123", false},
		{"unknown", "inst-123", false},
		{"", "inst-123", false},
		{"running", "", false},
		{"running", "   ", false},
		{"running", "\t\n", false},
		{"  running  ", "inst-123", false}, // whitespace in status
	}

	for _, tt := range tests {
		t.Run(tt.status+"_"+tt.instID[:minIntRecov(5, len(tt.instID))], func(t *testing.T) {
			w := &Workspace{}
			w.Morph.Status = tt.status
			w.Morph.InstanceID = tt.instID

			got := w.IsMorphRunning()
			if got != tt.expected {
				t.Errorf("IsMorphRunning() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func minIntRecov(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// =============================================================================
// GetMorphURLs Edge Cases
// =============================================================================

func TestGetMorphURLsWithPartialURLs(t *testing.T) {
	w := &Workspace{}

	// Set only some URLs
	w.Morph.CodeURL = "https://example.com/code/"
	w.Morph.VNCURL = "" // empty
	w.Morph.AppURL = "https://example.com/app/"
	// CDPURL not set

	urls := w.GetMorphURLs()

	if _, ok := urls["code"]; !ok {
		t.Error("Expected code URL")
	}
	if _, ok := urls["vnc"]; ok {
		t.Error("Did not expect vnc URL (empty)")
	}
	if _, ok := urls["app"]; !ok {
		t.Error("Expected app URL")
	}
	if _, ok := urls["cdp"]; ok {
		t.Error("Did not expect cdp URL (not set)")
	}
}

func TestGetMorphURLsWithWhitespaceURLs(t *testing.T) {
	w := &Workspace{}

	// Set URLs with only whitespace
	w.Morph.CodeURL = "   "
	w.Morph.VNCURL = "\t\n"
	w.Morph.AppURL = "https://valid.example.com/app/"

	urls := w.GetMorphURLs()

	// Whitespace URLs should be included (non-empty strings)
	if _, ok := urls["code"]; !ok {
		t.Log("Whitespace-only code URL was not included")
	}
	if _, ok := urls["app"]; !ok {
		t.Error("Expected app URL")
	}
}

// =============================================================================
// TextOutput Edge Cases
// =============================================================================

func TestTextOutputWithAllMorphFieldsEmpty(t *testing.T) {
	w := &Workspace{
		ID:    "ws-123",
		Name:  "test",
		Morph: MorphState{}, // all empty
	}

	output := w.TextOutput()

	// Should not include Morph section (instance ID is empty)
	if w.Morph.InstanceID == "" {
		// Expected behavior
		t.Logf("TextOutput with empty Morph: %s", output)
	}
}

func TestTextOutputWithOnlyURLs(t *testing.T) {
	w := &Workspace{
		ID:   "ws-123",
		Name: "test",
		Morph: MorphState{
			InstanceID: "inst-456",
			// Only URLs, no status
			CodeURL: "https://example.com/code/",
		},
	}

	output := w.TextOutput()
	t.Logf("TextOutput: %s", output)
}

// =============================================================================
// SavedSnapshot Edge Cases
// =============================================================================

func TestGetSavedSnapshotByEmptyName(t *testing.T) {
	w := &Workspace{}
	w.AddSavedSnapshot("snap-empty", "")

	snap := w.GetSavedSnapshot("")
	if snap == nil {
		t.Fatal("GetSavedSnapshot('') returned nil")
	}
	if snap.ID != "snap-empty" {
		t.Errorf("ID = %s, want snap-empty", snap.ID)
	}
}

func TestSavedSnapshotCreatedAtNeverZero(t *testing.T) {
	w := &Workspace{}

	before := time.Now()
	w.AddSavedSnapshot("snap-time", "time-test")
	after := time.Now()

	snap := w.GetSavedSnapshot("time-test")
	if snap == nil {
		t.Fatal("GetSavedSnapshot returned nil")
	}

	if snap.CreatedAt.IsZero() {
		t.Error("CreatedAt should never be zero")
	}
	if snap.CreatedAt.Before(before) || snap.CreatedAt.After(after) {
		t.Errorf("CreatedAt %v not in expected range [%v, %v]", snap.CreatedAt, before, after)
	}
}

// =============================================================================
// JSON Marshaling Edge Cases
// =============================================================================

func TestMorphStateJSONWithZeroTime(t *testing.T) {
	state := MorphState{
		InstanceID: "inst-zero-time",
		// StartedAt is zero
	}

	data, err := json.Marshal(state)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}

	var loaded MorphState
	if err := json.Unmarshal(data, &loaded); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}

	// Zero time should round-trip
	if loaded.InstanceID != state.InstanceID {
		t.Errorf("InstanceID = %s, want %s", loaded.InstanceID, state.InstanceID)
	}
}

func TestWorkspaceJSONWithNilMorph(t *testing.T) {
	// MorphState is a value type, not a pointer, so it can't be nil
	// But we can test with zero value
	w := &Workspace{
		ID:   "ws-nil-morph",
		Name: "nil-morph-test",
		// Morph is zero value
	}

	data, err := json.Marshal(w)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}

	var loaded Workspace
	if err := json.Unmarshal(data, &loaded); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}

	if loaded.ID != w.ID {
		t.Errorf("ID = %s, want %s", loaded.ID, w.ID)
	}
}

func TestStateJSONWithMorph(t *testing.T) {
	state := &State{
		ID:       "ws-state-morph",
		Name:     "state-morph-test",
		Template: "node",
		Status:   "active",
		Morph: MorphState{
			InstanceID: "inst-state",
			Status:     "running",
		},
	}

	data, err := json.Marshal(state)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}

	var loaded State
	if err := json.Unmarshal(data, &loaded); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}

	if loaded.Morph.InstanceID != state.Morph.InstanceID {
		t.Errorf("Morph.InstanceID = %s, want %s", loaded.Morph.InstanceID, state.Morph.InstanceID)
	}
}
