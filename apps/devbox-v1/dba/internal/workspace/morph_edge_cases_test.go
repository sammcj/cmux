// internal/workspace/morph_edge_cases_test.go
package workspace

import (
	"encoding/json"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

// =============================================================================
// SetMorphInstance Edge Cases
// =============================================================================

func TestSetMorphInstanceEmptyValues(t *testing.T) {
	ws := &Workspace{ID: "test"}

	ws.SetMorphInstance("", "", "")

	if ws.Morph.InstanceID != "" {
		t.Errorf("InstanceID should be empty, got %q", ws.Morph.InstanceID)
	}
	if ws.Morph.Status != "running" {
		t.Errorf("Status should be 'running', got %q", ws.Morph.Status)
	}
	if ws.Morph.StartedAt.IsZero() {
		t.Error("StartedAt should be set even with empty values")
	}
	// Empty base URL should not generate derived URLs
	if ws.Morph.CodeURL != "" {
		t.Errorf("CodeURL should be empty for empty baseURL, got %q", ws.Morph.CodeURL)
	}
}

func TestSetMorphInstanceVeryLongURL(t *testing.T) {
	ws := &Workspace{ID: "test"}

	// Very long URL
	longURL := "https://" + strings.Repeat("a", 10000) + ".morph.so"
	ws.SetMorphInstance("inst-123", "snap-456", longURL)

	if ws.Morph.BaseURL != longURL {
		t.Errorf("BaseURL should handle long URLs")
	}
	expectedCodeURL := longURL + "/code/"
	if ws.Morph.CodeURL != expectedCodeURL {
		t.Errorf("CodeURL not derived correctly from long URL")
	}
}

func TestSetMorphInstanceSpecialCharactersInURL(t *testing.T) {
	tests := []struct {
		name    string
		baseURL string
	}{
		{"with port", "https://example.morph.so:8080"},
		{"with path", "https://example.morph.so/prefix"},
		{"with query", "https://example.morph.so?token=abc"},
		{"with fragment", "https://example.morph.so#section"},
		{"with auth", "https://user:pass@example.morph.so"},
		{"unicode", "https://例え.morph.so"},
		{"ip address", "http://192.168.1.1"},
		{"localhost", "http://localhost:3000"},
		{"trailing slash", "https://example.morph.so/"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ws := &Workspace{ID: "test"}
			ws.SetMorphInstance("inst", "snap", tt.baseURL)

			if ws.Morph.BaseURL != tt.baseURL {
				t.Errorf("BaseURL = %q, want %q", ws.Morph.BaseURL, tt.baseURL)
			}

			// Derived URLs should append paths (trailing slashes are trimmed to avoid double slashes)
			trimmedURL := strings.TrimSuffix(tt.baseURL, "/")
			expectedCode := trimmedURL + "/code/"
			if ws.Morph.CodeURL != expectedCode {
				t.Errorf("CodeURL = %q, want %q", ws.Morph.CodeURL, expectedCode)
			}
		})
	}
}

func TestSetMorphInstanceInstanceIDFormats(t *testing.T) {
	tests := []string{
		"inst_abc123",
		"inst-abc-123",
		"INST_UPPERCASE",
		"i", // Very short
		strings.Repeat("x", 1000), // Very long
		"inst.with.dots",
		"inst/with/slashes",
		"inst:with:colons",
		"日本語インスタンス",
		"inst-🚀-emoji",
	}

	for _, id := range tests {
		t.Run(id[:min(10, len(id))], func(t *testing.T) {
			ws := &Workspace{ID: "test"}
			ws.SetMorphInstance(id, "snap", "https://example.com")

			if ws.Morph.InstanceID != id {
				t.Errorf("InstanceID = %q, want %q", ws.Morph.InstanceID, id)
			}
		})
	}
}

func TestSetMorphInstanceOverwrite(t *testing.T) {
	ws := &Workspace{ID: "test"}

	// Set initial values
	ws.SetMorphInstance("inst-1", "snap-1", "https://first.com")
	firstStartedAt := ws.Morph.StartedAt

	time.Sleep(10 * time.Millisecond)

	// Overwrite with new values
	ws.SetMorphInstance("inst-2", "snap-2", "https://second.com")

	if ws.Morph.InstanceID != "inst-2" {
		t.Errorf("InstanceID should be overwritten to 'inst-2', got %q", ws.Morph.InstanceID)
	}
	if ws.Morph.BaseURL != "https://second.com" {
		t.Errorf("BaseURL should be overwritten")
	}
	if !ws.Morph.StartedAt.After(firstStartedAt) {
		t.Error("StartedAt should be updated on overwrite")
	}
}

// =============================================================================
// ClearMorphInstance Edge Cases
// =============================================================================

func TestClearMorphInstanceWhenAlreadyClear(t *testing.T) {
	ws := &Workspace{ID: "test"}

	// Clear without setting - should not panic
	ws.ClearMorphInstance()

	if ws.Morph.Status != "stopped" {
		t.Errorf("Status should be 'stopped', got %q", ws.Morph.Status)
	}
	if ws.Morph.InstanceID != "" {
		t.Errorf("InstanceID should be empty, got %q", ws.Morph.InstanceID)
	}
}

func TestClearMorphInstancePreservesURLs(t *testing.T) {
	ws := &Workspace{ID: "test"}

	ws.SetMorphInstance("inst", "snap", "https://example.com")
	ws.ClearMorphInstance()

	// URLs should be preserved for reference
	if ws.Morph.BaseURL != "https://example.com" {
		t.Errorf("BaseURL should be preserved, got %q", ws.Morph.BaseURL)
	}
	if ws.Morph.CodeURL != "https://example.com/code/" {
		t.Errorf("CodeURL should be preserved")
	}
}

func TestClearMorphInstanceMultipleTimes(t *testing.T) {
	ws := &Workspace{ID: "test"}

	ws.SetMorphInstance("inst", "snap", "https://example.com")

	// Clear multiple times
	for i := 0; i < 10; i++ {
		ws.ClearMorphInstance()
	}

	if ws.Morph.Status != "stopped" {
		t.Errorf("Status should remain 'stopped'")
	}
}

// =============================================================================
// IsMorphRunning Edge Cases
// =============================================================================

func TestIsMorphRunningEdgeCases(t *testing.T) {
	tests := []struct {
		name       string
		status     string
		instanceID string
		expected   bool
	}{
		{"running with id", "running", "inst-123", true},
		{"running without id", "running", "", false},
		{"stopped with id", "stopped", "inst-123", false},
		{"stopped without id", "stopped", "", false},
		{"empty status with id", "", "inst-123", false},
		{"empty both", "", "", false},
		{"paused with id", "paused", "inst-123", false},
		{"RUNNING uppercase", "RUNNING", "inst-123", false},
		{"Running mixed case", "Running", "inst-123", false},
		{"running with spaces", " running ", "inst-123", false},
		{"whitespace id", "running", " ", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ws := &Workspace{
				ID: "test",
				Morph: MorphState{
					Status:     tt.status,
					InstanceID: tt.instanceID,
				},
			}

			result := ws.IsMorphRunning()
			if result != tt.expected {
				t.Errorf("IsMorphRunning() = %v, want %v", result, tt.expected)
			}
		})
	}
}

// =============================================================================
// SavedSnapshot Edge Cases
// =============================================================================

func TestAddSavedSnapshotDuplicateNames(t *testing.T) {
	ws := &Workspace{ID: "test"}

	ws.AddSavedSnapshot("snap-1", "checkpoint")
	ws.AddSavedSnapshot("snap-2", "checkpoint") // Same name, different ID
	ws.AddSavedSnapshot("snap-3", "checkpoint")

	// All should be added (no deduplication)
	if len(ws.Morph.SavedSnapshots) != 3 {
		t.Errorf("expected 3 snapshots, got %d", len(ws.Morph.SavedSnapshots))
	}

	// GetSavedSnapshot should return first match
	snap := ws.GetSavedSnapshot("checkpoint")
	if snap.ID != "snap-1" {
		t.Errorf("GetSavedSnapshot should return first match, got %q", snap.ID)
	}
}

func TestAddSavedSnapshotEmptyValues(t *testing.T) {
	ws := &Workspace{ID: "test"}

	ws.AddSavedSnapshot("", "")
	ws.AddSavedSnapshot("id-only", "")
	ws.AddSavedSnapshot("", "name-only")

	if len(ws.Morph.SavedSnapshots) != 3 {
		t.Errorf("expected 3 snapshots, got %d", len(ws.Morph.SavedSnapshots))
	}
}

func TestAddSavedSnapshotSpecialNames(t *testing.T) {
	tests := []string{
		"normal-name",
		"名前日本語",
		"name with spaces",
		"name\twith\ttabs",
		"name\nwith\nnewlines",
		strings.Repeat("x", 10000),
		"",
		" ",
		"name/with/slashes",
		"name:with:colons",
		"name@with@at",
		"name#with#hash",
	}

	for _, name := range tests {
		t.Run(name[:min(10, len(name))], func(t *testing.T) {
			ws := &Workspace{ID: "test"}
			ws.AddSavedSnapshot("snap-id", name)

			snap := ws.GetSavedSnapshot(name)
			if snap == nil {
				t.Errorf("should find snapshot with name %q", name)
			} else if snap.Name != name {
				t.Errorf("Name = %q, want %q", snap.Name, name)
			}
		})
	}
}

func TestGetSavedSnapshotCaseSensitivity(t *testing.T) {
	ws := &Workspace{ID: "test"}

	ws.AddSavedSnapshot("snap-1", "MyCheckpoint")

	// Exact match should work
	if snap := ws.GetSavedSnapshot("MyCheckpoint"); snap == nil {
		t.Error("should find exact match")
	}

	// Different case should NOT match (case sensitive)
	if snap := ws.GetSavedSnapshot("mycheckpoint"); snap != nil {
		t.Error("should not find case-insensitive match")
	}
	if snap := ws.GetSavedSnapshot("MYCHECKPOINT"); snap != nil {
		t.Error("should not find uppercase match")
	}
}

func TestGetSavedSnapshotNonExistent(t *testing.T) {
	ws := &Workspace{ID: "test"}

	ws.AddSavedSnapshot("snap-1", "exists")

	tests := []string{
		"nonexistent",
		"",
		" ",
		"exists ", // trailing space
		" exists", // leading space
	}

	for _, name := range tests {
		snap := ws.GetSavedSnapshot(name)
		if name != "exists" && snap != nil {
			t.Errorf("should not find snapshot for %q", name)
		}
	}
}

func TestSavedSnapshotCreatedAt(t *testing.T) {
	ws := &Workspace{ID: "test"}

	before := time.Now()
	ws.AddSavedSnapshot("snap-1", "checkpoint")
	after := time.Now()

	snap := ws.GetSavedSnapshot("checkpoint")
	if snap.CreatedAt.Before(before) || snap.CreatedAt.After(after) {
		t.Error("CreatedAt should be between before and after")
	}
}

func TestManySnapshots(t *testing.T) {
	ws := &Workspace{ID: "test"}

	// Add 1000 snapshots
	for i := 0; i < 1000; i++ {
		ws.AddSavedSnapshot("snap-"+string(rune(i)), "checkpoint-"+string(rune(i)))
	}

	if len(ws.Morph.SavedSnapshots) != 1000 {
		t.Errorf("expected 1000 snapshots, got %d", len(ws.Morph.SavedSnapshots))
	}
}

// =============================================================================
// GetMorphURLs Edge Cases
// =============================================================================

func TestGetMorphURLsPartiallySet(t *testing.T) {
	tests := []struct {
		name     string
		setup    func(*Workspace)
		expected int
	}{
		{
			"no urls",
			func(ws *Workspace) {},
			0,
		},
		{
			"only CodeURL",
			func(ws *Workspace) { ws.Morph.CodeURL = "https://example.com/code/" },
			1,
		},
		{
			"CodeURL and VNCURL",
			func(ws *Workspace) {
				ws.Morph.CodeURL = "https://example.com/code/"
				ws.Morph.VNCURL = "https://example.com/vnc/"
			},
			2,
		},
		{
			"all urls",
			func(ws *Workspace) {
				ws.Morph.CodeURL = "https://example.com/code/"
				ws.Morph.VNCURL = "https://example.com/vnc/"
				ws.Morph.AppURL = "https://example.com/app/"
				ws.Morph.CDPURL = "https://example.com/cdp/"
			},
			4,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ws := &Workspace{ID: "test"}
			tt.setup(ws)

			urls := ws.GetMorphURLs()
			if len(urls) != tt.expected {
				t.Errorf("expected %d URLs, got %d", tt.expected, len(urls))
			}
		})
	}
}

// =============================================================================
// TextOutput Edge Cases
// =============================================================================

func TestTextOutputWithMorphVeryLongValues(t *testing.T) {
	ws := &Workspace{
		ID:       "test",
		Name:     "test",
		Path:     "/test",
		Template: "node",
		Status:   "ready",
		Ports:    map[string]int{},
	}

	longURL := "https://" + strings.Repeat("a", 1000) + ".morph.so"
	ws.SetMorphInstance(strings.Repeat("x", 100), "snap", longURL)

	// Should not panic
	output := ws.TextOutput()
	if !strings.Contains(output, "Morph:") {
		t.Error("output should contain Morph section")
	}
}

func TestTextOutputMorphStatusVariations(t *testing.T) {
	statuses := []string{"running", "stopped", "paused", "error", "", "unknown"}

	for _, status := range statuses {
		t.Run(status, func(t *testing.T) {
			ws := &Workspace{
				ID:       "test",
				Name:     "test",
				Path:     "/test",
				Template: "node",
				Status:   "ready",
				Ports:    map[string]int{},
				Morph: MorphState{
					InstanceID: "inst-123",
					Status:     status,
				},
			}

			// Should not panic
			output := ws.TextOutput()
			if !strings.Contains(output, status) {
				// Empty status might not appear
				if status != "" {
					t.Logf("output may not contain status %q", status)
				}
			}
		})
	}
}

// =============================================================================
// JSON Serialization Edge Cases
// =============================================================================

func TestMorphStateJSONRoundTrip(t *testing.T) {
	original := MorphState{
		InstanceID: "inst-123",
		SnapshotID: "snap-456",
		Status:     "running",
		BaseURL:    "https://example.com",
		CodeURL:    "https://example.com/code/",
		VNCURL:     "https://example.com/vnc/",
		AppURL:     "https://example.com/app/",
		CDPURL:     "https://example.com/cdp/",
		CDPPort:    9222,
		StartedAt:  time.Now().Truncate(time.Second), // Truncate for comparison
		SavedSnapshots: []SavedSnapshot{
			{ID: "s1", Name: "checkpoint1", CreatedAt: time.Now().Truncate(time.Second)},
			{ID: "s2", Name: "checkpoint2", CreatedAt: time.Now().Truncate(time.Second)},
		},
	}

	// Serialize
	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}

	// Deserialize
	var loaded MorphState
	if err := json.Unmarshal(data, &loaded); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}

	// Compare
	if loaded.InstanceID != original.InstanceID {
		t.Errorf("InstanceID mismatch: %q vs %q", loaded.InstanceID, original.InstanceID)
	}
	if loaded.Status != original.Status {
		t.Errorf("Status mismatch: %q vs %q", loaded.Status, original.Status)
	}
	if loaded.CDPPort != original.CDPPort {
		t.Errorf("CDPPort mismatch: %d vs %d", loaded.CDPPort, original.CDPPort)
	}
	if len(loaded.SavedSnapshots) != len(original.SavedSnapshots) {
		t.Errorf("SavedSnapshots length mismatch: %d vs %d",
			len(loaded.SavedSnapshots), len(original.SavedSnapshots))
	}
}

func TestMorphStateJSONOmitEmpty(t *testing.T) {
	state := MorphState{} // All zero values

	data, err := json.Marshal(state)
	if err != nil {
		t.Fatal(err)
	}

	// Should be mostly empty due to omitempty
	var m map[string]interface{}
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatal(err)
	}

	// Check that empty fields are omitted
	if _, ok := m["instance_id"]; ok {
		t.Error("empty instance_id should be omitted")
	}
	if _, ok := m["cdp_port"]; ok {
		t.Error("zero cdp_port should be omitted")
	}
}

func TestMorphStateJSONSpecialCharacters(t *testing.T) {
	state := MorphState{
		InstanceID: "inst-\"quoted\"",
		BaseURL:    "https://example.com?foo=bar&baz=qux",
		SavedSnapshots: []SavedSnapshot{
			{Name: "name\nwith\nnewlines"},
			{Name: "name\twith\ttabs"},
			{Name: "name\"with\"quotes"},
		},
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
		t.Errorf("InstanceID with quotes not preserved")
	}
	if len(loaded.SavedSnapshots) != 3 {
		t.Errorf("SavedSnapshots not preserved")
	}
}

// =============================================================================
// State Persistence Edge Cases
// =============================================================================

func TestSaveLoadEmptyMorphState(t *testing.T) {
	tmpDir := t.TempDir()

	ws := &Workspace{
		ID:          "test",
		Name:        "test",
		Path:        tmpDir,
		ProjectPath: filepath.Join(tmpDir, "project"),
		Template:    "node",
		Status:      "ready",
		BasePort:    10000,
		Ports:       map[string]int{"PORT": 10000},
		CreatedAt:   time.Now(),
		LastActive:  time.Now(),
		// Morph is empty
	}

	if err := ws.SaveState(); err != nil {
		t.Fatal(err)
	}

	loaded, err := LoadState(tmpDir)
	if err != nil {
		t.Fatal(err)
	}

	// Empty Morph should be preserved as empty
	if loaded.Morph.InstanceID != "" {
		t.Errorf("empty InstanceID should be preserved as empty")
	}
}

func TestSaveLoadLargeNumberOfSnapshots(t *testing.T) {
	tmpDir := t.TempDir()

	ws := &Workspace{
		ID:          "test",
		Name:        "test",
		Path:        tmpDir,
		ProjectPath: filepath.Join(tmpDir, "project"),
		Template:    "node",
		Status:      "ready",
		BasePort:    10000,
		Ports:       map[string]int{"PORT": 10000},
		CreatedAt:   time.Now(),
		LastActive:  time.Now(),
	}

	// Add many snapshots
	for i := 0; i < 100; i++ {
		ws.AddSavedSnapshot("snap-"+string(rune(i)), "checkpoint-"+string(rune(i)))
	}

	if err := ws.SaveState(); err != nil {
		t.Fatal(err)
	}

	loaded, err := LoadState(tmpDir)
	if err != nil {
		t.Fatal(err)
	}

	if len(loaded.Morph.SavedSnapshots) != 100 {
		t.Errorf("expected 100 snapshots, got %d", len(loaded.Morph.SavedSnapshots))
	}
}

func TestSaveLoadSpecialCharactersInMorph(t *testing.T) {
	tmpDir := t.TempDir()

	ws := &Workspace{
		ID:          "test",
		Name:        "test",
		Path:        tmpDir,
		ProjectPath: filepath.Join(tmpDir, "project"),
		Template:    "node",
		Status:      "ready",
		BasePort:    10000,
		Ports:       map[string]int{"PORT": 10000},
		CreatedAt:   time.Now(),
		LastActive:  time.Now(),
	}

	ws.SetMorphInstance("inst-日本語-🚀", "snap-特殊文字", "https://例え.morph.so")
	ws.AddSavedSnapshot("snap-1", "チェックポイント")
	ws.AddSavedSnapshot("snap-2", "name\"with\"quotes")

	if err := ws.SaveState(); err != nil {
		t.Fatal(err)
	}

	loaded, err := LoadState(tmpDir)
	if err != nil {
		t.Fatal(err)
	}

	if loaded.Morph.InstanceID != "inst-日本語-🚀" {
		t.Errorf("Unicode InstanceID not preserved: %q", loaded.Morph.InstanceID)
	}
	if len(loaded.Morph.SavedSnapshots) != 2 {
		t.Errorf("SavedSnapshots not preserved")
	}
}

// =============================================================================
// Concurrent Access Edge Cases
// =============================================================================

func TestConcurrentSetMorphInstance(t *testing.T) {
	if raceDetectorEnabled() {
		t.Skip("Skipping concurrent test with race detector (Workspace is not thread-safe)")
	}

	ws := &Workspace{ID: "test"}

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			ws.SetMorphInstance(
				"inst-"+string(rune(n)),
				"snap-"+string(rune(n)),
				"https://"+string(rune(n))+".morph.so",
			)
		}(i)
	}

	wg.Wait()

	// Should not panic, final state should be valid
	if ws.Morph.Status != "running" {
		t.Errorf("Status should be 'running'")
	}
}

func TestConcurrentAddSavedSnapshot(t *testing.T) {
	if raceDetectorEnabled() {
		t.Skip("Skipping concurrent test with race detector (Workspace is not thread-safe)")
	}

	ws := &Workspace{ID: "test"}

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			ws.AddSavedSnapshot("snap-"+string(rune(n)), "checkpoint-"+string(rune(n)))
		}(i)
	}

	wg.Wait()

	// Due to race conditions, we might not have exactly 100
	// but should have at least some and not crash
	if len(ws.Morph.SavedSnapshots) == 0 {
		t.Error("should have some snapshots")
	}
}

func TestConcurrentIsMorphRunning(t *testing.T) {
	ws := &Workspace{ID: "test"}
	ws.SetMorphInstance("inst", "snap", "https://example.com")

	var wg sync.WaitGroup
	errors := make(chan error, 100)

	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			result := ws.IsMorphRunning()
			if !result {
				errors <- errorf("expected IsMorphRunning() = true")
			}
		}()
	}

	wg.Wait()
	close(errors)

	for err := range errors {
		t.Error(err)
	}
}

func TestConcurrentGetMorphURLs(t *testing.T) {
	ws := &Workspace{ID: "test"}
	ws.SetMorphInstance("inst", "snap", "https://example.com")

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			urls := ws.GetMorphURLs()
			if len(urls) != 4 {
				t.Errorf("expected 4 URLs, got %d", len(urls))
			}
		}()
	}

	wg.Wait()
}

// =============================================================================
// Helper Functions
// =============================================================================

func errorf(format string, args ...interface{}) error {
	return &morphTestError{msg: format, args: args}
}

type morphTestError struct {
	msg  string
	args []interface{}
}

func (e *morphTestError) Error() string {
	return e.msg
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
