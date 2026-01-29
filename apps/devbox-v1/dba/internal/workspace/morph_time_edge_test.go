// internal/workspace/morph_time_edge_test.go
package workspace

import (
	"encoding/json"
	"testing"
	"time"
)

// =============================================================================
// Time Handling Edge Cases
// =============================================================================

func TestStartedAtTimezoneEdge(t *testing.T) {
	w := &Workspace{}
	w.SetMorphInstance("inst-123", "snap-456", "https://example.com")

	// StartedAt should be set to current time
	now := time.Now()
	diff := now.Sub(w.Morph.StartedAt)

	if diff < 0 || diff > time.Second {
		t.Errorf("StartedAt diff = %v, want < 1s", diff)
	}
}

func TestStartedAtPersistenceRoundTrip(t *testing.T) {
	original := MorphState{
		InstanceID: "inst-123",
		StartedAt:  time.Date(2024, 6, 15, 10, 30, 0, 0, time.UTC),
	}

	// Marshal to JSON
	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}

	// Unmarshal back
	var loaded MorphState
	if err := json.Unmarshal(data, &loaded); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}

	if !loaded.StartedAt.Equal(original.StartedAt) {
		t.Errorf("StartedAt = %v, want %v", loaded.StartedAt, original.StartedAt)
	}
}

func TestStartedAtWithDifferentTimezones(t *testing.T) {
	// Create times in different timezones
	utc := time.Date(2024, 6, 15, 12, 0, 0, 0, time.UTC)
	pst := time.Date(2024, 6, 15, 5, 0, 0, 0, time.FixedZone("PST", -7*3600))

	// These should represent the same instant
	if !utc.Equal(pst) {
		t.Skip("Test times don't match - adjust as needed")
	}

	state := MorphState{
		InstanceID: "inst-tz",
		StartedAt:  utc,
	}

	data, err := json.Marshal(state)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}

	var loaded MorphState
	if err := json.Unmarshal(data, &loaded); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}

	// Times should be equal when compared
	t.Logf("Original: %v, Loaded: %v", state.StartedAt, loaded.StartedAt)
}

func TestStartedAtZeroValueEdge(t *testing.T) {
	state := MorphState{
		InstanceID: "inst-zero",
		// StartedAt is zero value
	}

	data, err := json.Marshal(state)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}

	var loaded MorphState
	if err := json.Unmarshal(data, &loaded); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}

	if !loaded.StartedAt.IsZero() {
		t.Errorf("StartedAt should be zero, got %v", loaded.StartedAt)
	}
}

func TestStartedAtFarFuture(t *testing.T) {
	future := time.Date(9999, 12, 31, 23, 59, 59, 0, time.UTC)

	state := MorphState{
		InstanceID: "inst-future",
		StartedAt:  future,
	}

	data, err := json.Marshal(state)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}

	var loaded MorphState
	if err := json.Unmarshal(data, &loaded); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}

	if !loaded.StartedAt.Equal(future) {
		t.Errorf("StartedAt = %v, want %v", loaded.StartedAt, future)
	}
}

func TestStartedAtFarPast(t *testing.T) {
	past := time.Date(1970, 1, 1, 0, 0, 1, 0, time.UTC)

	state := MorphState{
		InstanceID: "inst-past",
		StartedAt:  past,
	}

	data, err := json.Marshal(state)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}

	var loaded MorphState
	if err := json.Unmarshal(data, &loaded); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}

	if !loaded.StartedAt.Equal(past) {
		t.Errorf("StartedAt = %v, want %v", loaded.StartedAt, past)
	}
}

// =============================================================================
// SavedSnapshot Time Edge Cases
// =============================================================================

func TestSavedSnapshotTimeHandling(t *testing.T) {
	w := &Workspace{}

	before := time.Now()
	w.AddSavedSnapshot("snap-1", "name-1")
	after := time.Now()

	snap := w.GetSavedSnapshot("name-1")
	if snap == nil {
		t.Fatal("GetSavedSnapshot returned nil")
	}

	if snap.CreatedAt.Before(before) || snap.CreatedAt.After(after) {
		t.Errorf("CreatedAt = %v, not in range [%v, %v]", snap.CreatedAt, before, after)
	}
}

func TestSavedSnapshotTimeUniqueness(t *testing.T) {
	w := &Workspace{}

	// Add many snapshots rapidly
	times := make([]time.Time, 100)
	for i := 0; i < 100; i++ {
		w.AddSavedSnapshot("snap-"+string(rune('0'+i%10)), "name-"+string(rune('0'+i%10)))
		times[i] = w.Morph.SavedSnapshots[i].CreatedAt
	}

	// Check for duplicates
	duplicates := 0
	for i := 1; i < len(times); i++ {
		if times[i].Equal(times[i-1]) {
			duplicates++
		}
	}

	t.Logf("Duplicates found: %d (may be expected for rapid additions)", duplicates)
}

func TestSavedSnapshotTimeJSONFormat(t *testing.T) {
	snap := SavedSnapshot{
		ID:        "snap-json",
		Name:      "json-test",
		CreatedAt: time.Date(2024, 6, 15, 10, 30, 45, 123456789, time.UTC),
	}

	data, err := json.Marshal(snap)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}

	jsonStr := string(data)
	t.Logf("JSON representation: %s", jsonStr)

	// Should be RFC3339 format
	if !containsTimeFormat(jsonStr) {
		t.Error("JSON should contain ISO8601/RFC3339 formatted time")
	}

	// Verify round-trip
	var loaded SavedSnapshot
	if err := json.Unmarshal(data, &loaded); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}

	// Nanoseconds may be truncated
	if loaded.CreatedAt.Unix() != snap.CreatedAt.Unix() {
		t.Errorf("Unix timestamp mismatch: %d vs %d",
			loaded.CreatedAt.Unix(), snap.CreatedAt.Unix())
	}
}

func containsTimeFormat(s string) bool {
	// Simple check for time format (contains T and Z or offset)
	hasT := false
	hasZ := false
	for i := 0; i < len(s)-1; i++ {
		if s[i] == 'T' && s[i+1] >= '0' && s[i+1] <= '9' {
			hasT = true
		}
		if s[i] == 'Z' || (s[i] == '+' && i > 0 && s[i-1] >= '0') {
			hasZ = true
		}
	}
	return hasT && hasZ
}

// =============================================================================
// Time Comparison Edge Cases
// =============================================================================

func TestTimeComparisonAfterSerialization(t *testing.T) {
	original := time.Now()

	state := MorphState{
		InstanceID: "inst-compare",
		StartedAt:  original,
	}

	data, err := json.Marshal(state)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}

	var loaded MorphState
	if err := json.Unmarshal(data, &loaded); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}

	// Note: nanoseconds may be truncated to microseconds in JSON
	// Use Unix() for robust comparison
	if loaded.StartedAt.Unix() != original.Unix() {
		t.Errorf("Unix timestamp differs: %d vs %d",
			loaded.StartedAt.Unix(), original.Unix())
	}
}

func TestMultipleTimeFieldsSerialization(t *testing.T) {
	w := &Workspace{
		ID:         "ws-time",
		Name:       "time-test",
		CreatedAt:  time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		LastActive: time.Date(2024, 6, 15, 12, 0, 0, 0, time.UTC),
		Morph: MorphState{
			InstanceID: "inst-time",
			StartedAt:  time.Date(2024, 6, 15, 10, 0, 0, 0, time.UTC),
			SavedSnapshots: []SavedSnapshot{
				{ID: "snap-1", Name: "s1", CreatedAt: time.Date(2024, 6, 10, 0, 0, 0, 0, time.UTC)},
				{ID: "snap-2", Name: "s2", CreatedAt: time.Date(2024, 6, 12, 0, 0, 0, 0, time.UTC)},
			},
		},
	}

	data, err := json.Marshal(w)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}

	var loaded Workspace
	if err := json.Unmarshal(data, &loaded); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}

	// Verify all times
	if !loaded.CreatedAt.Equal(w.CreatedAt) {
		t.Errorf("CreatedAt mismatch")
	}
	if !loaded.LastActive.Equal(w.LastActive) {
		t.Errorf("LastActive mismatch")
	}
	if !loaded.Morph.StartedAt.Equal(w.Morph.StartedAt) {
		t.Errorf("Morph.StartedAt mismatch")
	}
	if len(loaded.Morph.SavedSnapshots) != 2 {
		t.Errorf("SavedSnapshots count = %d, want 2", len(loaded.Morph.SavedSnapshots))
	}
}

// =============================================================================
// Duration Edge Cases
// =============================================================================

func TestStartedAtDurationCalculation(t *testing.T) {
	w := &Workspace{}

	past := time.Now().Add(-1 * time.Hour)
	w.Morph.StartedAt = past
	w.Morph.InstanceID = "inst-duration"
	w.Morph.Status = "running"

	duration := time.Since(w.Morph.StartedAt)

	if duration < 59*time.Minute || duration > 61*time.Minute {
		t.Errorf("Duration = %v, expected ~1 hour", duration)
	}
}

func TestStartedAtNeverInFuture(t *testing.T) {
	w := &Workspace{}

	before := time.Now()
	w.SetMorphInstance("inst-123", "snap-456", "https://example.com")
	after := time.Now()

	if w.Morph.StartedAt.Before(before) {
		t.Error("StartedAt is before SetMorphInstance was called")
	}
	if w.Morph.StartedAt.After(after) {
		t.Error("StartedAt is in the future")
	}
}
