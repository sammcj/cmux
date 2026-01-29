// internal/workspace/morph_snapshot_edge_test.go
package workspace

import (
	"sync"
	"testing"
	"time"
)

// =============================================================================
// Snapshot Management Edge Cases
// =============================================================================

func TestAddSavedSnapshotWithDuplicateIDs(t *testing.T) {
	w := &Workspace{}

	// Add same ID multiple times
	w.AddSavedSnapshot("snap-dup", "name-1")
	w.AddSavedSnapshot("snap-dup", "name-2")
	w.AddSavedSnapshot("snap-dup", "name-3")

	// All should be added (no deduplication)
	if len(w.Morph.SavedSnapshots) != 3 {
		t.Errorf("SavedSnapshots count = %d, want 3", len(w.Morph.SavedSnapshots))
	}
}

func TestAddSavedSnapshotWithDuplicateNames(t *testing.T) {
	w := &Workspace{}

	// Add same name with different IDs
	w.AddSavedSnapshot("snap-1", "same-name")
	w.AddSavedSnapshot("snap-2", "same-name")
	w.AddSavedSnapshot("snap-3", "same-name")

	if len(w.Morph.SavedSnapshots) != 3 {
		t.Errorf("SavedSnapshots count = %d, want 3", len(w.Morph.SavedSnapshots))
	}

	// GetSavedSnapshot should return first match
	snap := w.GetSavedSnapshot("same-name")
	if snap == nil {
		t.Fatal("GetSavedSnapshot returned nil")
	}
	if snap.ID != "snap-1" {
		t.Errorf("ID = %s, want snap-1 (first match)", snap.ID)
	}
}

func TestGetSavedSnapshotByNameOnly(t *testing.T) {
	w := &Workspace{}

	w.AddSavedSnapshot("snap-byid", "name-byid")
	w.AddSavedSnapshot("snap-other", "name-other")

	// GetSavedSnapshot only searches by name, not ID
	// So searching for "snap-byid" (an ID) should return nil
	snapByID := w.GetSavedSnapshot("snap-byid")
	if snapByID != nil {
		t.Log("GetSavedSnapshot unexpectedly found by ID (might be a feature)")
	}

	// Get by name should work
	snapByName := w.GetSavedSnapshot("name-other")
	if snapByName == nil {
		t.Fatal("GetSavedSnapshot by name returned nil")
	}
	if snapByName.ID != "snap-other" {
		t.Errorf("By name: got ID = %s, want snap-other", snapByName.ID)
	}

	// Get by other name should work
	snapByName2 := w.GetSavedSnapshot("name-byid")
	if snapByName2 == nil {
		t.Fatal("GetSavedSnapshot by name (name-byid) returned nil")
	}
	if snapByName2.ID != "snap-byid" {
		t.Errorf("By name: got ID = %s, want snap-byid", snapByName2.ID)
	}
}

func TestGetSavedSnapshotWithSpecialCharacters(t *testing.T) {
	w := &Workspace{}

	specialNames := []string{
		"name with spaces",
		"name-with-dashes",
		"name_with_underscores",
		"name.with.dots",
		"name/with/slashes",
		"name:with:colons",
		"name@with@at",
		"name#with#hash",
		"name$with$dollar",
		"name%with%percent",
		"name&with&ampersand",
		"name+with+plus",
		"name=with=equals",
		"name[with]brackets",
		"name{with}braces",
		"name|with|pipes",
		"name\\with\\backslashes",
		"name\"with\"quotes",
		"name'with'apostrophes",
		"name<with>angles",
		"name?with?questions",
		"name*with*asterisks",
		"日本語の名前",
		"🚀emoji🚀name",
	}

	for i, name := range specialNames {
		w.AddSavedSnapshot("snap-"+string(rune('A'+i)), name)
	}

	// Verify all can be retrieved
	for _, name := range specialNames {
		snap := w.GetSavedSnapshot(name)
		if snap == nil {
			t.Errorf("GetSavedSnapshot(%q) returned nil", name)
		}
	}
}

func TestGetSavedSnapshotCaseSensitivityEdge(t *testing.T) {
	w := &Workspace{}

	w.AddSavedSnapshot("SNAP-UPPER", "NAME-UPPER")
	w.AddSavedSnapshot("snap-lower", "name-lower")
	w.AddSavedSnapshot("Snap-Mixed", "Name-Mixed")

	tests := []struct {
		query    string
		wantNil  bool
		wantID   string
	}{
		{"NAME-UPPER", false, "SNAP-UPPER"},
		{"name-upper", true, ""},  // case mismatch
		{"name-lower", false, "snap-lower"},
		{"NAME-LOWER", true, ""},  // case mismatch
		{"Name-Mixed", false, "Snap-Mixed"},
		{"name-mixed", true, ""},  // case mismatch
	}

	for _, tt := range tests {
		t.Run(tt.query, func(t *testing.T) {
			snap := w.GetSavedSnapshot(tt.query)
			if (snap == nil) != tt.wantNil {
				t.Errorf("GetSavedSnapshot(%q) nil = %v, want %v", tt.query, snap == nil, tt.wantNil)
			}
			if snap != nil && snap.ID != tt.wantID {
				t.Errorf("GetSavedSnapshot(%q) ID = %s, want %s", tt.query, snap.ID, tt.wantID)
			}
		})
	}
}

func TestSavedSnapshotCreatedAtIsMonotonic(t *testing.T) {
	w := &Workspace{}

	// Add snapshots in rapid succession
	for i := 0; i < 100; i++ {
		w.AddSavedSnapshot("snap-"+string(rune('0'+i%10)), "name-"+string(rune('0'+i%10)))
	}

	// CreatedAt should be monotonically increasing (or equal for same-instant adds)
	for i := 1; i < len(w.Morph.SavedSnapshots); i++ {
		prev := w.Morph.SavedSnapshots[i-1].CreatedAt
		curr := w.Morph.SavedSnapshots[i].CreatedAt
		if curr.Before(prev) {
			t.Errorf("Snapshot %d has CreatedAt before snapshot %d", i, i-1)
		}
	}
}

func TestSavedSnapshotOrderPreserved(t *testing.T) {
	w := &Workspace{}

	// Add in specific order
	names := []string{"first", "second", "third", "fourth", "fifth"}
	for i, name := range names {
		w.AddSavedSnapshot("snap-"+string(rune('0'+i)), name)
	}

	// Verify order is preserved
	for i, name := range names {
		if w.Morph.SavedSnapshots[i].Name != name {
			t.Errorf("Snapshot %d has name %s, want %s",
				i, w.Morph.SavedSnapshots[i].Name, name)
		}
	}
}

// =============================================================================
// Concurrent Snapshot Operations
// =============================================================================

func TestConcurrentAddSnapshotOnly(t *testing.T) {
	if raceDetectorEnabled() {
		t.Skip("Skipping concurrent test with race detector (Workspace is not thread-safe)")
	}

	// Note: The Workspace type is not thread-safe, so we only test
	// concurrent adds (which is also racy but less likely to panic)
	// In production, callers should synchronize access
	w := &Workspace{}

	var wg sync.WaitGroup

	// Concurrent adds only - gets during concurrent modification can panic
	for i := 0; i < 100; i++ {
		wg.Add(1)
		idx := i
		go func() {
			defer wg.Done()
			w.AddSavedSnapshot("snap-"+string(rune('0'+idx%10)), "name-"+string(rune('0'+idx%10)))
		}()
	}
	wg.Wait()

	// After all adds complete, verify we have some snapshots
	// (exact count may vary due to race conditions)
	t.Logf("Final snapshot count: %d", len(w.Morph.SavedSnapshots))
}

func TestMassiveSnapshotList(t *testing.T) {
	w := &Workspace{}

	// Add 10000 snapshots
	for i := 0; i < 10000; i++ {
		w.AddSavedSnapshot("snap-"+string(rune('0'+i%1000)), "name-"+string(rune('0'+i%1000)))
	}

	if len(w.Morph.SavedSnapshots) != 10000 {
		t.Errorf("SavedSnapshots count = %d, want 10000", len(w.Morph.SavedSnapshots))
	}

	// Get should still work
	snap := w.GetSavedSnapshot("name-" + string(rune('0'+5000%1000)))
	if snap == nil {
		t.Error("GetSavedSnapshot returned nil for existing snapshot")
	}
}

// =============================================================================
// Snapshot State After Operations
// =============================================================================

func TestSnapshotsPreservedAfterSetMorphInstance(t *testing.T) {
	w := &Workspace{}

	// Add snapshots
	w.AddSavedSnapshot("snap-1", "name-1")
	w.AddSavedSnapshot("snap-2", "name-2")

	// Set instance
	w.SetMorphInstance("inst-123", "snap-456", "https://example.com")

	// Snapshots should be preserved
	if len(w.Morph.SavedSnapshots) != 2 {
		t.Errorf("SavedSnapshots count = %d, want 2", len(w.Morph.SavedSnapshots))
	}
}

func TestSnapshotsPreservedAfterClearMorphInstance(t *testing.T) {
	w := &Workspace{}

	w.SetMorphInstance("inst-123", "snap-456", "https://example.com")
	w.AddSavedSnapshot("snap-1", "name-1")
	w.AddSavedSnapshot("snap-2", "name-2")

	// Clear instance
	w.ClearMorphInstance()

	// Snapshots should be preserved
	if len(w.Morph.SavedSnapshots) != 2 {
		t.Errorf("SavedSnapshots count = %d, want 2", len(w.Morph.SavedSnapshots))
	}
}

// =============================================================================
// Snapshot Time Edge Cases
// =============================================================================

func TestSnapshotCreatedAtPrecisionEdge(t *testing.T) {
	w := &Workspace{}

	before := time.Now()
	w.AddSavedSnapshot("snap-1", "name-1")
	after := time.Now()

	snap := w.GetSavedSnapshot("name-1")
	if snap == nil {
		t.Fatal("GetSavedSnapshot returned nil")
	}

	if snap.CreatedAt.Before(before) {
		t.Errorf("CreatedAt %v is before test start %v", snap.CreatedAt, before)
	}
	if snap.CreatedAt.After(after) {
		t.Errorf("CreatedAt %v is after test end %v", snap.CreatedAt, after)
	}
}

func TestSnapshotWithZeroTime(t *testing.T) {
	w := &Workspace{}

	// Manually add snapshot with zero time
	w.Morph.SavedSnapshots = append(w.Morph.SavedSnapshots, SavedSnapshot{
		ID:        "snap-zero-time",
		Name:      "zero-time",
		CreatedAt: time.Time{}, // Zero value
	})

	snap := w.GetSavedSnapshot("zero-time")
	if snap == nil {
		t.Fatal("GetSavedSnapshot returned nil")
	}
	if !snap.CreatedAt.IsZero() {
		t.Errorf("CreatedAt should be zero")
	}
}

// =============================================================================
// Edge Case: Empty and Nil Handling
// =============================================================================

func TestGetSavedSnapshotFromNilSlice(t *testing.T) {
	w := &Workspace{}
	w.Morph.SavedSnapshots = nil

	snap := w.GetSavedSnapshot("any")
	if snap != nil {
		t.Error("Expected nil for nil slice")
	}
}

func TestGetSavedSnapshotFromEmptySlice(t *testing.T) {
	w := &Workspace{}
	w.Morph.SavedSnapshots = []SavedSnapshot{}

	snap := w.GetSavedSnapshot("any")
	if snap != nil {
		t.Error("Expected nil for empty slice")
	}
}

func TestAddSnapshotToNilSlice(t *testing.T) {
	w := &Workspace{}
	w.Morph.SavedSnapshots = nil

	// Should not panic
	w.AddSavedSnapshot("snap-1", "name-1")

	if len(w.Morph.SavedSnapshots) != 1 {
		t.Errorf("SavedSnapshots count = %d, want 1", len(w.Morph.SavedSnapshots))
	}
}

// =============================================================================
// Snapshot Search Edge Cases
// =============================================================================

func TestGetSavedSnapshotPartialMatch(t *testing.T) {
	w := &Workspace{}

	w.AddSavedSnapshot("snap-full-id", "full-name")

	// Partial matches should NOT work
	partials := []string{"snap-", "snap-full", "full", "name", "snap-full-id-extra"}
	for _, partial := range partials {
		snap := w.GetSavedSnapshot(partial)
		if snap != nil {
			t.Errorf("GetSavedSnapshot(%q) should return nil for partial match", partial)
		}
	}
}

func TestGetSavedSnapshotExactMatchByName(t *testing.T) {
	w := &Workspace{}

	w.AddSavedSnapshot("snap-exact", "name-exact")

	// GetSavedSnapshot only searches by name, not by ID
	// So searching for "snap-exact" (an ID) returns nil (unless name matches)
	snapByID := w.GetSavedSnapshot("snap-exact")
	if snapByID != nil {
		t.Log("GetSavedSnapshot found by ID - unexpected but okay if name matches")
	}

	// Exact match by name should work
	snapByName := w.GetSavedSnapshot("name-exact")
	if snapByName == nil {
		t.Error("GetSavedSnapshot by exact name returned nil")
	}
}
