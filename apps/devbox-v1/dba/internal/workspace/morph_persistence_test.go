// internal/workspace/morph_persistence_test.go
package workspace

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

// =============================================================================
// State File Persistence Edge Cases
// =============================================================================

func TestSaveStateWithMorphThenReload(t *testing.T) {
	tmpDir := t.TempDir()
	stateDir := filepath.Join(tmpDir, ".dba")
	if err := os.MkdirAll(stateDir, 0755); err != nil {
		t.Fatal(err)
	}

	ws := &Workspace{
		ID:       "ws-persistence-test",
		Name:     "persistence-test",
		Path:     tmpDir,
		Template: "node",
		Status:   "active",
	}

	// Set Morph state
	ws.SetMorphInstance("inst-persist-123", "snap-persist-456", "https://persist.example.com")
	ws.AddSavedSnapshot("saved-snap-1", "first-snapshot")
	ws.AddSavedSnapshot("saved-snap-2", "second-snapshot")

	// Save
	if err := ws.SaveState(); err != nil {
		t.Fatalf("SaveState error: %v", err)
	}

	// Reload
	loaded, err := Load(tmpDir)
	if err != nil {
		t.Fatalf("Load error: %v", err)
	}

	// Verify all Morph fields persisted
	if loaded.Morph.InstanceID != "inst-persist-123" {
		t.Errorf("InstanceID = %s, want inst-persist-123", loaded.Morph.InstanceID)
	}
	if loaded.Morph.SnapshotID != "snap-persist-456" {
		t.Errorf("SnapshotID = %s, want snap-persist-456", loaded.Morph.SnapshotID)
	}
	if loaded.Morph.Status != "running" {
		t.Errorf("Status = %s, want running", loaded.Morph.Status)
	}
	if loaded.Morph.BaseURL != "https://persist.example.com" {
		t.Errorf("BaseURL = %s, want https://persist.example.com", loaded.Morph.BaseURL)
	}
	if len(loaded.Morph.SavedSnapshots) != 2 {
		t.Errorf("SavedSnapshots count = %d, want 2", len(loaded.Morph.SavedSnapshots))
	}
}

func TestSaveStateClearedMorphThenReload(t *testing.T) {
	tmpDir := t.TempDir()
	stateDir := filepath.Join(tmpDir, ".dba")
	if err := os.MkdirAll(stateDir, 0755); err != nil {
		t.Fatal(err)
	}

	ws := &Workspace{
		ID:       "ws-cleared-test",
		Name:     "cleared-test",
		Path:     tmpDir,
		Template: "node",
		Status:   "active",
	}

	// Set then clear Morph state
	ws.SetMorphInstance("inst-temp", "snap-temp", "https://temp.example.com")
	ws.ClearMorphInstance()

	// Save
	if err := ws.SaveState(); err != nil {
		t.Fatalf("SaveState error: %v", err)
	}

	// Reload
	loaded, err := Load(tmpDir)
	if err != nil {
		t.Fatalf("Load error: %v", err)
	}

	// Verify cleared state persisted
	if loaded.Morph.InstanceID != "" {
		t.Errorf("InstanceID = %s, want empty", loaded.Morph.InstanceID)
	}
	if loaded.Morph.Status != "stopped" {
		t.Errorf("Status = %s, want stopped", loaded.Morph.Status)
	}
}

func TestStateFileLargeSnapshots(t *testing.T) {
	tmpDir := t.TempDir()
	stateDir := filepath.Join(tmpDir, ".dba")
	if err := os.MkdirAll(stateDir, 0755); err != nil {
		t.Fatal(err)
	}

	ws := &Workspace{
		ID:       "ws-large-snap",
		Name:     "large-snap-test",
		Path:     tmpDir,
		Template: "node",
		Status:   "active",
	}

	// Add 1000 snapshots
	for i := 0; i < 1000; i++ {
		ws.AddSavedSnapshot("snap-"+strings.Repeat("x", 100), "snapshot-"+strings.Repeat("y", 200))
	}

	// Save
	if err := ws.SaveState(); err != nil {
		t.Fatalf("SaveState error: %v", err)
	}

	// Check file size
	statePath := filepath.Join(stateDir, "state.json")
	info, err := os.Stat(statePath)
	if err != nil {
		t.Fatalf("Stat error: %v", err)
	}
	t.Logf("State file size with 1000 snapshots: %d bytes", info.Size())

	// Reload
	loaded, err := Load(tmpDir)
	if err != nil {
		t.Fatalf("Load error: %v", err)
	}

	if len(loaded.Morph.SavedSnapshots) != 1000 {
		t.Errorf("SavedSnapshots count = %d, want 1000", len(loaded.Morph.SavedSnapshots))
	}
}

func TestStateFileAtomicity(t *testing.T) {
	if raceDetectorEnabled() {
		t.Skip("Skipping concurrent test with race detector (Workspace is not thread-safe)")
	}

	tmpDir := t.TempDir()
	stateDir := filepath.Join(tmpDir, ".dba")
	if err := os.MkdirAll(stateDir, 0755); err != nil {
		t.Fatal(err)
	}

	ws := &Workspace{
		ID:       "ws-atomic",
		Name:     "atomic-test",
		Path:     tmpDir,
		Template: "node",
		Status:   "active",
	}
	ws.SetMorphInstance("inst-atomic", "snap-atomic", "https://atomic.example.com")

	// Save multiple times concurrently
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			ws.Morph.InstanceID = "inst-atomic-" + string(rune('0'+idx))
			_ = ws.SaveState()
		}(i)
	}
	wg.Wait()

	// Reload should get a valid state (not corrupted)
	loaded, err := Load(tmpDir)
	if err != nil {
		t.Fatalf("Load error after concurrent saves: %v", err)
	}

	if !strings.HasPrefix(loaded.Morph.InstanceID, "inst-atomic") {
		t.Errorf("InstanceID = %s, want inst-atomic-*", loaded.Morph.InstanceID)
	}
}

// =============================================================================
// State File Format Edge Cases
// =============================================================================

func TestStateWithExtraFields(t *testing.T) {
	tmpDir := t.TempDir()
	stateDir := filepath.Join(tmpDir, ".dba")
	if err := os.MkdirAll(stateDir, 0755); err != nil {
		t.Fatal(err)
	}

	// State with extra/future fields
	content := `{
		"id": "ws-extra",
		"name": "extra-test",
		"template": "node",
		"status": "active",
		"future_field": "some value",
		"morph": {
			"instance_id": "inst-extra",
			"status": "running",
			"future_morph_field": 12345
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

	// Known fields should be parsed
	if loaded.Morph.InstanceID != "inst-extra" {
		t.Errorf("InstanceID = %s, want inst-extra", loaded.Morph.InstanceID)
	}
}

func TestStateWithMissingFields(t *testing.T) {
	tmpDir := t.TempDir()
	stateDir := filepath.Join(tmpDir, ".dba")
	if err := os.MkdirAll(stateDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Minimal state
	content := `{
		"id": "ws-minimal",
		"name": "minimal-test"
	}`

	statePath := filepath.Join(stateDir, "state.json")
	if err := os.WriteFile(statePath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	loaded, err := Load(tmpDir)
	if err != nil {
		t.Fatalf("Load error: %v", err)
	}

	// Morph should be zero value
	if loaded.Morph.InstanceID != "" {
		t.Errorf("InstanceID = %s, want empty", loaded.Morph.InstanceID)
	}
	if loaded.IsMorphRunning() {
		t.Error("Minimal state should not be running")
	}
}

func TestStateWithUnicodeContent(t *testing.T) {
	tmpDir := t.TempDir()
	stateDir := filepath.Join(tmpDir, ".dba")
	if err := os.MkdirAll(stateDir, 0755); err != nil {
		t.Fatal(err)
	}

	ws := &Workspace{
		ID:       "ws-日本語",
		Name:     "テスト-workspace",
		Path:     tmpDir,
		Template: "node",
		Status:   "active",
	}
	ws.SetMorphInstance("inst-中文", "snap-한국어", "https://例え.jp")
	ws.AddSavedSnapshot("snap-🎯", "スナップショット-📸")

	if err := ws.SaveState(); err != nil {
		t.Fatalf("SaveState error: %v", err)
	}

	loaded, err := Load(tmpDir)
	if err != nil {
		t.Fatalf("Load error: %v", err)
	}

	if loaded.ID != "ws-日本語" {
		t.Errorf("ID = %s, want ws-日本語", loaded.ID)
	}
	if loaded.Morph.InstanceID != "inst-中文" {
		t.Errorf("InstanceID = %s, want inst-中文", loaded.Morph.InstanceID)
	}
}

func TestStateWithSpecialCharactersInValues(t *testing.T) {
	tmpDir := t.TempDir()
	stateDir := filepath.Join(tmpDir, ".dba")
	if err := os.MkdirAll(stateDir, 0755); err != nil {
		t.Fatal(err)
	}

	specialValues := []struct {
		name       string
		instanceID string
	}{
		{"backslash", `inst-with\backslash`},
		{"quotes", `inst-with"quotes"`},
		{"newlines", "inst-with\nnewlines"},
		{"tabs", "inst-with\ttabs"},
		{"unicode_escape", `inst-with\u0000nulls`},
	}

	for _, sv := range specialValues {
		t.Run(sv.name, func(t *testing.T) {
			ws := &Workspace{
				ID:       "ws-special",
				Name:     "special-test",
				Path:     tmpDir,
				Template: "node",
				Status:   "active",
			}
			ws.SetMorphInstance(sv.instanceID, "snap-special", "https://example.com")

			if err := ws.SaveState(); err != nil {
				t.Fatalf("SaveState error: %v", err)
			}

			loaded, err := Load(tmpDir)
			if err != nil {
				t.Fatalf("Load error: %v", err)
			}

			if loaded.Morph.InstanceID != sv.instanceID {
				t.Errorf("InstanceID = %q, want %q", loaded.Morph.InstanceID, sv.instanceID)
			}
		})
	}
}

// =============================================================================
// State Directory Edge Cases
// =============================================================================

func TestStateDirectoryPermissions(t *testing.T) {
	tmpDir := t.TempDir()
	stateDir := filepath.Join(tmpDir, ".dba")
	if err := os.MkdirAll(stateDir, 0755); err != nil {
		t.Fatal(err)
	}

	ws := &Workspace{
		ID:       "ws-perm",
		Name:     "perm-test",
		Path:     tmpDir,
		Template: "node",
		Status:   "active",
	}
	ws.SetMorphInstance("inst-perm", "snap-perm", "https://example.com")

	// Save first
	if err := ws.SaveState(); err != nil {
		t.Fatalf("SaveState error: %v", err)
	}

	// Make directory read-only
	if err := os.Chmod(stateDir, 0444); err != nil {
		t.Skip("Cannot change directory permissions")
	}
	defer os.Chmod(stateDir, 0755) // Restore for cleanup

	// Try to save again - should fail
	ws.Morph.InstanceID = "inst-modified"
	err := ws.SaveState()
	if err == nil {
		t.Log("SaveState succeeded on read-only directory (may be running as root)")
	} else {
		t.Logf("SaveState failed as expected: %v", err)
	}
}

func TestStateDirectorySymlink(t *testing.T) {
	tmpDir := t.TempDir()
	realStateDir := filepath.Join(tmpDir, "real-state")
	symlinkStateDir := filepath.Join(tmpDir, ".dba")

	if err := os.MkdirAll(realStateDir, 0755); err != nil {
		t.Fatal(err)
	}

	if err := os.Symlink(realStateDir, symlinkStateDir); err != nil {
		t.Skip("Symlinks not supported")
	}

	ws := &Workspace{
		ID:       "ws-symlink-state",
		Name:     "symlink-state-test",
		Path:     tmpDir,
		Template: "node",
		Status:   "active",
	}
	ws.SetMorphInstance("inst-symlink", "snap-symlink", "https://example.com")

	if err := ws.SaveState(); err != nil {
		t.Fatalf("SaveState error: %v", err)
	}

	// Verify file was created in real directory
	realStatePath := filepath.Join(realStateDir, "state.json")
	if _, err := os.Stat(realStatePath); os.IsNotExist(err) {
		t.Error("State file not created in real directory")
	}

	// Load should work
	loaded, err := Load(tmpDir)
	if err != nil {
		t.Fatalf("Load error: %v", err)
	}

	if loaded.Morph.InstanceID != "inst-symlink" {
		t.Errorf("InstanceID = %s, want inst-symlink", loaded.Morph.InstanceID)
	}
}

// =============================================================================
// Workspace Path Edge Cases
// =============================================================================

func TestWorkspaceWithVeryLongPath(t *testing.T) {
	// Create a very long path (may fail on some systems)
	tmpDir := t.TempDir()
	longPath := tmpDir
	for i := 0; i < 20; i++ {
		longPath = filepath.Join(longPath, "longsubdir")
	}

	if err := os.MkdirAll(filepath.Join(longPath, ".dba"), 0755); err != nil {
		t.Skip("Cannot create very long path")
	}

	ws := &Workspace{
		ID:       "ws-long-path",
		Name:     "long-path-test",
		Path:     longPath,
		Template: "node",
		Status:   "active",
	}
	ws.SetMorphInstance("inst-long", "snap-long", "https://example.com")

	if err := ws.SaveState(); err != nil {
		t.Fatalf("SaveState error: %v", err)
	}

	loaded, err := Load(longPath)
	if err != nil {
		t.Fatalf("Load error: %v", err)
	}

	if loaded.Morph.InstanceID != "inst-long" {
		t.Errorf("InstanceID = %s, want inst-long", loaded.Morph.InstanceID)
	}
}

func TestWorkspaceWithSpacesInPath(t *testing.T) {
	tmpDir := t.TempDir()
	spacePath := filepath.Join(tmpDir, "path with spaces", "and more spaces")
	if err := os.MkdirAll(filepath.Join(spacePath, ".dba"), 0755); err != nil {
		t.Fatal(err)
	}

	ws := &Workspace{
		ID:       "ws-spaces",
		Name:     "spaces-test",
		Path:     spacePath,
		Template: "node",
		Status:   "active",
	}
	ws.SetMorphInstance("inst-spaces", "snap-spaces", "https://example.com")

	if err := ws.SaveState(); err != nil {
		t.Fatalf("SaveState error: %v", err)
	}

	loaded, err := Load(spacePath)
	if err != nil {
		t.Fatalf("Load error: %v", err)
	}

	if loaded.Morph.InstanceID != "inst-spaces" {
		t.Errorf("InstanceID = %s, want inst-spaces", loaded.Morph.InstanceID)
	}
}

// =============================================================================
// Timestamp Persistence Tests
// =============================================================================

func TestStartedAtPersistenceAdvanced(t *testing.T) {
	tmpDir := t.TempDir()
	stateDir := filepath.Join(tmpDir, ".dba")
	if err := os.MkdirAll(stateDir, 0755); err != nil {
		t.Fatal(err)
	}

	ws := &Workspace{
		ID:       "ws-timestamp",
		Name:     "timestamp-test",
		Path:     tmpDir,
		Template: "node",
		Status:   "active",
	}

	before := time.Now()
	ws.SetMorphInstance("inst-time", "snap-time", "https://example.com")
	after := time.Now()

	if err := ws.SaveState(); err != nil {
		t.Fatalf("SaveState error: %v", err)
	}

	loaded, err := Load(tmpDir)
	if err != nil {
		t.Fatalf("Load error: %v", err)
	}

	if loaded.Morph.StartedAt.Before(before) {
		t.Error("StartedAt is before operation")
	}
	if loaded.Morph.StartedAt.After(after) {
		t.Error("StartedAt is after operation")
	}
}

func TestSnapshotCreatedAtPersistence(t *testing.T) {
	tmpDir := t.TempDir()
	stateDir := filepath.Join(tmpDir, ".dba")
	if err := os.MkdirAll(stateDir, 0755); err != nil {
		t.Fatal(err)
	}

	ws := &Workspace{
		ID:       "ws-snap-time",
		Name:     "snap-time-test",
		Path:     tmpDir,
		Template: "node",
		Status:   "active",
	}

	before := time.Now()
	time.Sleep(10 * time.Millisecond)
	ws.AddSavedSnapshot("snap-timed", "timed-snapshot")
	time.Sleep(10 * time.Millisecond)
	after := time.Now()

	if err := ws.SaveState(); err != nil {
		t.Fatalf("SaveState error: %v", err)
	}

	loaded, err := Load(tmpDir)
	if err != nil {
		t.Fatalf("Load error: %v", err)
	}

	if len(loaded.Morph.SavedSnapshots) != 1 {
		t.Fatalf("SavedSnapshots count = %d, want 1", len(loaded.Morph.SavedSnapshots))
	}

	snap := loaded.Morph.SavedSnapshots[0]
	if snap.CreatedAt.Before(before) {
		t.Error("CreatedAt is before operation")
	}
	if snap.CreatedAt.After(after) {
		t.Error("CreatedAt is after operation")
	}
}

// =============================================================================
// JSON Serialization Edge Cases
// =============================================================================

func TestMorphStateJSONPrettyPrint(t *testing.T) {
	state := MorphState{
		InstanceID: "inst-pretty",
		Status:     "running",
		SavedSnapshots: []SavedSnapshot{
			{ID: "snap-1", Name: "first"},
			{ID: "snap-2", Name: "second"},
		},
	}

	// Pretty print
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		t.Fatalf("MarshalIndent error: %v", err)
	}

	// Should be readable
	if !strings.Contains(string(data), "\n") {
		t.Error("Pretty print should contain newlines")
	}

	// Should round-trip
	var loaded MorphState
	if err := json.Unmarshal(data, &loaded); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}

	if loaded.InstanceID != state.InstanceID {
		t.Errorf("InstanceID = %s, want %s", loaded.InstanceID, state.InstanceID)
	}
}

func TestMorphStateJSONCompact(t *testing.T) {
	state := MorphState{
		InstanceID: "inst-compact",
		Status:     "running",
	}

	// Compact
	data, err := json.Marshal(state)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}

	// Should not have extra whitespace
	if strings.Contains(string(data), "\n") {
		t.Error("Compact should not contain newlines")
	}

	// Should round-trip
	var loaded MorphState
	if err := json.Unmarshal(data, &loaded); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}

	if loaded.InstanceID != state.InstanceID {
		t.Errorf("InstanceID = %s, want %s", loaded.InstanceID, state.InstanceID)
	}
}
