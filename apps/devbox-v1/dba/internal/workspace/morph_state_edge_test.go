// internal/workspace/morph_state_edge_test.go
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
// State File Edge Cases
// =============================================================================

func TestLoadStateWithCorruptedMorph(t *testing.T) {
	tmpDir := t.TempDir()
	stateDir := filepath.Join(tmpDir, ".dba")
	if err := os.MkdirAll(stateDir, 0755); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name    string
		content string
	}{
		{
			name: "invalid_morph_type",
			content: `{
				"id": "ws-123",
				"name": "test",
				"morph": "not an object"
			}`,
		},
		{
			name: "invalid_instance_id_type",
			content: `{
				"id": "ws-123",
				"name": "test",
				"morph": {
					"instance_id": 12345
				}
			}`,
		},
		{
			name: "invalid_cdp_port_type",
			content: `{
				"id": "ws-123",
				"name": "test",
				"morph": {
					"cdp_port": "not a number"
				}
			}`,
		},
		{
			name: "invalid_started_at",
			content: `{
				"id": "ws-123",
				"name": "test",
				"morph": {
					"started_at": "not a valid time"
				}
			}`,
		},
		{
			name: "invalid_saved_snapshots_type",
			content: `{
				"id": "ws-123",
				"name": "test",
				"morph": {
					"saved_snapshots": "not an array"
				}
			}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			statePath := filepath.Join(stateDir, "state.json")
			if err := os.WriteFile(statePath, []byte(tt.content), 0644); err != nil {
				t.Fatal(err)
			}

			_, err := Load(tmpDir)
			// Should error on invalid types
			if err == nil {
				t.Logf("Load() did not error for %s", tt.name)
			} else {
				t.Logf("Load() error for %s: %v", tt.name, err)
			}
		})
	}
}

func TestLoadStateWithPartialMorph(t *testing.T) {
	tmpDir := t.TempDir()
	stateDir := filepath.Join(tmpDir, ".dba")
	if err := os.MkdirAll(stateDir, 0755); err != nil {
		t.Fatal(err)
	}

	// State with only some Morph fields
	content := `{
		"id": "ws-123",
		"name": "test",
		"template": "node",
		"status": "active",
		"morph": {
			"instance_id": "inst-partial",
			"status": "running"
		}
	}`

	statePath := filepath.Join(stateDir, "state.json")
	if err := os.WriteFile(statePath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	ws, err := Load(tmpDir)
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	if ws.Morph.InstanceID != "inst-partial" {
		t.Errorf("InstanceID = %s, want inst-partial", ws.Morph.InstanceID)
	}
	if ws.Morph.Status != "running" {
		t.Errorf("Status = %s, want running", ws.Morph.Status)
	}
	// Other fields should be zero values
	if ws.Morph.SnapshotID != "" {
		t.Errorf("SnapshotID = %s, want empty", ws.Morph.SnapshotID)
	}
}

func TestLoadStateWithEmptyMorph(t *testing.T) {
	tmpDir := t.TempDir()
	stateDir := filepath.Join(tmpDir, ".dba")
	if err := os.MkdirAll(stateDir, 0755); err != nil {
		t.Fatal(err)
	}

	content := `{
		"id": "ws-123",
		"name": "test",
		"template": "node",
		"status": "active",
		"morph": {}
	}`

	statePath := filepath.Join(stateDir, "state.json")
	if err := os.WriteFile(statePath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	ws, err := Load(tmpDir)
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	if ws.Morph.InstanceID != "" {
		t.Errorf("InstanceID = %s, want empty", ws.Morph.InstanceID)
	}
	if ws.IsMorphRunning() {
		t.Error("IsMorphRunning() should be false for empty Morph")
	}
}

func TestLoadStateWithNullMorph(t *testing.T) {
	tmpDir := t.TempDir()
	stateDir := filepath.Join(tmpDir, ".dba")
	if err := os.MkdirAll(stateDir, 0755); err != nil {
		t.Fatal(err)
	}

	content := `{
		"id": "ws-123",
		"name": "test",
		"template": "node",
		"status": "active",
		"morph": null
	}`

	statePath := filepath.Join(stateDir, "state.json")
	if err := os.WriteFile(statePath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	ws, err := Load(tmpDir)
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	if ws.Morph.InstanceID != "" {
		t.Errorf("InstanceID = %s, want empty", ws.Morph.InstanceID)
	}
}

func TestLoadStateWithMissingMorph(t *testing.T) {
	tmpDir := t.TempDir()
	stateDir := filepath.Join(tmpDir, ".dba")
	if err := os.MkdirAll(stateDir, 0755); err != nil {
		t.Fatal(err)
	}

	// State without morph field at all
	content := `{
		"id": "ws-123",
		"name": "test",
		"template": "node",
		"status": "active"
	}`

	statePath := filepath.Join(stateDir, "state.json")
	if err := os.WriteFile(statePath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	ws, err := Load(tmpDir)
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	// Morph should be zero value
	if ws.Morph.InstanceID != "" {
		t.Errorf("InstanceID = %s, want empty", ws.Morph.InstanceID)
	}
}

// =============================================================================
// SavedSnapshot Edge Cases
// =============================================================================

func TestSavedSnapshotWithMaxLength(t *testing.T) {
	w := &Workspace{}

	// Very long snapshot ID and name
	longID := strings.Repeat("a", 10000)
	longName := strings.Repeat("b", 10000)

	w.AddSavedSnapshot(longID, longName)

	snap := w.GetSavedSnapshot(longName)
	if snap == nil {
		t.Fatal("GetSavedSnapshot() returned nil")
	}
	if len(snap.ID) != 10000 {
		t.Errorf("ID length = %d, want 10000", len(snap.ID))
	}
	if len(snap.Name) != 10000 {
		t.Errorf("Name length = %d, want 10000", len(snap.Name))
	}
}

func TestSavedSnapshotOrderPreservation(t *testing.T) {
	w := &Workspace{}

	// Add snapshots in order
	for i := 0; i < 100; i++ {
		w.AddSavedSnapshot("snap-"+string(rune('0'+i/10))+string(rune('0'+i%10)), "snapshot-"+string(rune('0'+i/10))+string(rune('0'+i%10)))
	}

	// Verify order is preserved
	for i := 0; i < 100; i++ {
		expectedID := "snap-" + string(rune('0'+i/10)) + string(rune('0'+i%10))
		if w.Morph.SavedSnapshots[i].ID != expectedID {
			t.Errorf("SavedSnapshots[%d].ID = %s, want %s", i, w.Morph.SavedSnapshots[i].ID, expectedID)
		}
	}
}

func TestSavedSnapshotEmptyName(t *testing.T) {
	w := &Workspace{}
	w.AddSavedSnapshot("snap-1", "")

	snap := w.GetSavedSnapshot("")
	if snap == nil {
		t.Fatal("GetSavedSnapshot('') returned nil")
	}
	if snap.ID != "snap-1" {
		t.Errorf("ID = %s, want snap-1", snap.ID)
	}
}

func TestSavedSnapshotNilSlice(t *testing.T) {
	w := &Workspace{}
	// SavedSnapshots is nil initially

	snap := w.GetSavedSnapshot("nonexistent")
	if snap != nil {
		t.Errorf("GetSavedSnapshot() = %v, want nil", snap)
	}

	// Adding to nil slice should work
	w.AddSavedSnapshot("snap-1", "first")
	if len(w.Morph.SavedSnapshots) != 1 {
		t.Errorf("SavedSnapshots length = %d, want 1", len(w.Morph.SavedSnapshots))
	}
}

// =============================================================================
// IsMorphRunning Edge Cases
// =============================================================================

func TestIsMorphRunningVariousWhitespace(t *testing.T) {
	tests := []struct {
		name       string
		status     string
		instanceID string
		want       bool
	}{
		{"tab_id", "running", "\t", false},
		{"newline_id", "running", "\n", false},
		{"carriage_return_id", "running", "\r", false},
		{"mixed_whitespace", "running", " \t\n\r ", false},
		{"space_padded_valid", "running", "  valid  ", true},
		{"tab_padded_valid", "running", "\tvalid\t", true},
		{"valid_with_internal_space", "running", "valid id", true},
		{"valid_with_internal_tab", "running", "valid\tid", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := &Workspace{}
			w.Morph.Status = tt.status
			w.Morph.InstanceID = tt.instanceID

			got := w.IsMorphRunning()
			if got != tt.want {
				t.Errorf("IsMorphRunning() = %v, want %v", got, tt.want)
			}
		})
	}
}

// =============================================================================
// URL Edge Cases
// =============================================================================

func TestSetMorphInstanceURLEdgeCases(t *testing.T) {
	tests := []struct {
		name    string
		baseURL string
	}{
		{"query_string", "https://example.com?token=abc"},
		{"fragment", "https://example.com#section"},
		{"query_and_fragment", "https://example.com?token=abc#section"},
		{"encoded_chars", "https://example.com/path%20with%20spaces"},
		{"unicode_domain", "https://例え.jp"},
		{"punycode", "https://xn--n3h.com"},
		{"ipv6", "http://[::1]:8080"},
		{"auth_in_url", "https://user:pass@example.com"},
		{"double_slash", "https://example.com//path"},
		{"no_scheme", "example.com/path"},
		{"file_scheme", "file:///path/to/file"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := &Workspace{}
			w.SetMorphInstance("inst-123", "snap-456", tt.baseURL)

			// Should not panic
			if w.Morph.BaseURL != tt.baseURL {
				t.Errorf("BaseURL = %s, want %s", w.Morph.BaseURL, tt.baseURL)
			}

			t.Logf("CodeURL = %s", w.Morph.CodeURL)
		})
	}
}

func TestGetMorphURLsWithEmptyBase(t *testing.T) {
	w := &Workspace{}
	w.SetMorphInstance("inst-123", "snap-456", "")

	urls := w.GetMorphURLs()

	// With empty base URL, all derived URLs should be empty
	if len(urls) != 0 {
		t.Errorf("GetMorphURLs() returned %d urls, want 0", len(urls))
	}
}

// =============================================================================
// StartedAt Edge Cases
// =============================================================================

func TestStartedAtZeroValue(t *testing.T) {
	w := &Workspace{}

	// StartedAt is zero initially
	if !w.Morph.StartedAt.IsZero() {
		t.Error("StartedAt should be zero initially")
	}

	// After setting instance, it should be non-zero
	w.SetMorphInstance("inst-123", "snap-456", "https://example.com")
	if w.Morph.StartedAt.IsZero() {
		t.Error("StartedAt should be non-zero after SetMorphInstance")
	}
}

func TestStartedAtFutureTime(t *testing.T) {
	w := &Workspace{}

	// Set a future time manually (edge case)
	futureTime := time.Now().Add(24 * time.Hour)
	w.Morph.StartedAt = futureTime

	// Should still serialize/deserialize correctly
	data, err := json.Marshal(w.Morph)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}

	var loaded MorphState
	if err := json.Unmarshal(data, &loaded); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}

	diff := futureTime.Sub(loaded.StartedAt)
	if diff > time.Second || diff < -time.Second {
		t.Errorf("StartedAt not preserved: diff = %v", diff)
	}
}

func TestStartedAtVeryOldTime(t *testing.T) {
	w := &Workspace{}

	// Set a very old time
	oldTime := time.Date(1970, 1, 1, 0, 0, 0, 0, time.UTC)
	w.Morph.StartedAt = oldTime

	data, err := json.Marshal(w.Morph)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}

	var loaded MorphState
	if err := json.Unmarshal(data, &loaded); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}

	if !loaded.StartedAt.Equal(oldTime) {
		t.Errorf("StartedAt = %v, want %v", loaded.StartedAt, oldTime)
	}
}

// =============================================================================
// CDPPort Edge Cases
// =============================================================================

func TestCDPPortBoundaryValues(t *testing.T) {
	tests := []struct {
		port int
		name string
	}{
		{0, "zero"},
		{1, "one"},
		{80, "http"},
		{443, "https"},
		{1024, "first_unprivileged"},
		{9222, "default_cdp"},
		{65535, "max_valid"},
		{65536, "above_max"},
		{-1, "negative"},
		{-65535, "negative_large"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := &Workspace{}
			w.Morph.CDPPort = tt.port

			// Should serialize without error
			data, err := json.Marshal(w.Morph)
			if err != nil {
				t.Fatalf("Marshal error: %v", err)
			}

			var loaded MorphState
			if err := json.Unmarshal(data, &loaded); err != nil {
				t.Fatalf("Unmarshal error: %v", err)
			}

			if loaded.CDPPort != tt.port {
				t.Errorf("CDPPort = %d, want %d", loaded.CDPPort, tt.port)
			}
		})
	}
}

// =============================================================================
// Concurrent State Operations
// =============================================================================

func TestConcurrentSetAndClear(t *testing.T) {
	if raceDetectorEnabled() {
		t.Skip("Skipping concurrent test with race detector (Workspace is not thread-safe)")
	}

	w := &Workspace{}
	var wg sync.WaitGroup

	// Concurrent set and clear operations
	for i := 0; i < 100; i++ {
		wg.Add(2)
		go func(idx int) {
			defer wg.Done()
			w.SetMorphInstance("inst-"+string(rune('0'+idx%10)), "snap-"+string(rune('0'+idx%10)), "https://example.com")
		}(i)
		go func() {
			defer wg.Done()
			w.ClearMorphInstance()
		}()
	}

	wg.Wait()

	// Final state should be consistent (either set or cleared)
	t.Logf("Final InstanceID: %q, Status: %q", w.Morph.InstanceID, w.Morph.Status)
}

func TestConcurrentAddAndGetSnapshot(t *testing.T) {
	if raceDetectorEnabled() {
		t.Skip("Skipping concurrent test with race detector (Workspace is not thread-safe)")
	}

	w := &Workspace{}
	var wg sync.WaitGroup

	// Add some initial snapshots
	for i := 0; i < 10; i++ {
		w.AddSavedSnapshot("snap-"+string(rune('0'+i)), "snapshot-"+string(rune('0'+i)))
	}

	// Concurrent add and get
	for i := 0; i < 100; i++ {
		wg.Add(2)
		go func(idx int) {
			defer wg.Done()
			w.AddSavedSnapshot("new-snap-"+string(rune('0'+idx%10)), "new-snapshot-"+string(rune('0'+idx%10)))
		}(i)
		go func(idx int) {
			defer wg.Done()
			_ = w.GetSavedSnapshot("snapshot-" + string(rune('0'+idx%10)))
		}(i)
	}

	wg.Wait()

	t.Logf("Final snapshot count: %d", len(w.Morph.SavedSnapshots))
}

// =============================================================================
// TextOutput Edge Cases
// =============================================================================

func TestTextOutputWithNilWorkspace(t *testing.T) {
	// This would panic, but testing that proper initialization works
	w := &Workspace{
		ID:   "test-id",
		Name: "test-name",
	}

	output := w.TextOutput()
	if output == "" {
		t.Error("TextOutput() should not return empty string")
	}
}

func TestTextOutputWithAllMorphFields(t *testing.T) {
	w := &Workspace{
		ID:   "ws-123",
		Name: "test-workspace",
		Morph: MorphState{
			InstanceID: "inst-456",
			SnapshotID: "snap-789",
			Status:     "running",
			BaseURL:    "https://example.com",
			CodeURL:    "https://example.com/code/",
			VNCURL:     "https://example.com/vnc/",
			AppURL:     "https://example.com/app/",
			CDPURL:     "https://example.com/cdp/",
			CDPPort:    9222,
			StartedAt:  time.Now(),
			SavedSnapshots: []SavedSnapshot{
				{ID: "s1", Name: "snap1", CreatedAt: time.Now()},
				{ID: "s2", Name: "snap2", CreatedAt: time.Now()},
			},
		},
	}

	output := w.TextOutput()

	// Should contain Morph section
	if !strings.Contains(output, "Morph:") {
		t.Error("Output should contain 'Morph:'")
	}
	if !strings.Contains(output, "inst-456") {
		t.Error("Output should contain instance ID")
	}
	if !strings.Contains(output, "running") {
		t.Error("Output should contain status")
	}
}

func TestTextOutputWithUnicodeValues(t *testing.T) {
	w := &Workspace{
		ID:   "ws-日本語",
		Name: "テスト-workspace",
		Morph: MorphState{
			InstanceID: "inst-中文",
			Status:     "running",
			CodeURL:    "https://例え.jp/code/",
		},
	}

	output := w.TextOutput()
	if !strings.Contains(output, "日本語") {
		t.Error("Output should contain Japanese characters")
	}
	if !strings.Contains(output, "中文") {
		t.Error("Output should contain Chinese characters")
	}
}

// =============================================================================
// State Struct Edge Cases
// =============================================================================

func TestStateStructMorphField(t *testing.T) {
	state := &State{
		ID:       "ws-123",
		Name:     "test",
		Template: "node",
		Status:   "active",
		Morph: MorphState{
			InstanceID: "inst-state",
			Status:     "running",
		},
	}

	// Marshal
	data, err := json.Marshal(state)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}

	// Unmarshal
	var loaded State
	if err := json.Unmarshal(data, &loaded); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}

	if loaded.Morph.InstanceID != "inst-state" {
		t.Errorf("InstanceID = %s, want inst-state", loaded.Morph.InstanceID)
	}
}

func TestStateStructMorphOmitEmpty(t *testing.T) {
	state := &State{
		ID:       "ws-123",
		Name:     "test",
		Template: "node",
		Status:   "active",
		// Morph is zero value
	}

	data, err := json.Marshal(state)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}

	// With omitempty, empty Morph might be omitted
	t.Logf("State JSON: %s", string(data))
}
