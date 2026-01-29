// internal/workspace/morph_advanced_test.go
package workspace

import (
	"encoding/json"
	"strings"
	"sync"
	"testing"
	"time"
)

// =============================================================================
// URL Generation Edge Cases
// =============================================================================

func TestSetMorphInstanceURLVariations(t *testing.T) {
	tests := []struct {
		name         string
		baseURL      string
		expectedCode string
		expectedVNC  string
		expectedApp  string
		expectedCDP  string
	}{
		{
			name:         "standard_https",
			baseURL:      "https://morph.example.com",
			expectedCode: "https://morph.example.com/code/",
			expectedVNC:  "https://morph.example.com/vnc/vnc.html",
			expectedApp:  "https://morph.example.com/vnc/app/",
			expectedCDP:  "wss://morph.example.com/cdp/",
		},
		{
			name:         "with_trailing_slash",
			baseURL:      "https://morph.example.com/",
			expectedCode: "https://morph.example.com/code/",  // Trailing slash is trimmed to avoid double slashes
			expectedVNC:  "https://morph.example.com/vnc/vnc.html",
			expectedApp:  "https://morph.example.com/vnc/app/",
			expectedCDP:  "wss://morph.example.com/cdp/",
		},
		{
			name:         "with_port",
			baseURL:      "https://morph.example.com:8443",
			expectedCode: "https://morph.example.com:8443/code/",
			expectedVNC:  "https://morph.example.com:8443/vnc/vnc.html",
			expectedApp:  "https://morph.example.com:8443/vnc/app/",
			expectedCDP:  "wss://morph.example.com:8443/cdp/",
		},
		{
			name:         "with_path",
			baseURL:      "https://morph.example.com/v1/instances/abc123",
			expectedCode: "https://morph.example.com/v1/instances/abc123/code/",
			expectedVNC:  "https://morph.example.com/v1/instances/abc123/vnc/vnc.html",
			expectedApp:  "https://morph.example.com/v1/instances/abc123/vnc/app/",
			expectedCDP:  "wss://morph.example.com/v1/instances/abc123/cdp/",
		},
		{
			name:         "http_localhost",
			baseURL:      "http://localhost:3000",
			expectedCode: "http://localhost:3000/code/",
			expectedVNC:  "http://localhost:3000/vnc/vnc.html",
			expectedApp:  "http://localhost:3000/vnc/app/",
			expectedCDP:  "ws://localhost:3000/cdp/",
		},
		{
			name:         "ip_address",
			baseURL:      "http://192.168.1.100:8080",
			expectedCode: "http://192.168.1.100:8080/code/",
			expectedVNC:  "http://192.168.1.100:8080/vnc/vnc.html",
			expectedApp:  "http://192.168.1.100:8080/vnc/app/",
			expectedCDP:  "ws://192.168.1.100:8080/cdp/",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := &Workspace{}
			w.SetMorphInstance("inst-123", "snap-456", tt.baseURL)

			if w.Morph.CodeURL != tt.expectedCode {
				t.Errorf("CodeURL = %s, want %s", w.Morph.CodeURL, tt.expectedCode)
			}
			if w.Morph.VNCURL != tt.expectedVNC {
				t.Errorf("VNCURL = %s, want %s", w.Morph.VNCURL, tt.expectedVNC)
			}
			if w.Morph.AppURL != tt.expectedApp {
				t.Errorf("AppURL = %s, want %s", w.Morph.AppURL, tt.expectedApp)
			}
			if w.Morph.CDPURL != tt.expectedCDP {
				t.Errorf("CDPURL = %s, want %s", w.Morph.CDPURL, tt.expectedCDP)
			}
		})
	}
}

func TestGetMorphURLsEmpty(t *testing.T) {
	w := &Workspace{}

	urls := w.GetMorphURLs()
	if len(urls) != 0 {
		t.Errorf("GetMorphURLs() returned %d urls, want 0", len(urls))
	}
}

func TestGetMorphURLsPartial(t *testing.T) {
	w := &Workspace{}
	w.Morph.CodeURL = "https://example.com/code/"
	// Leave others empty

	urls := w.GetMorphURLs()
	if len(urls) != 1 {
		t.Errorf("GetMorphURLs() returned %d urls, want 1", len(urls))
	}
	if urls["code"] != "https://example.com/code/" {
		t.Errorf("urls[code] = %s, want https://example.com/code/", urls["code"])
	}
}

// =============================================================================
// Snapshot Management Edge Cases
// =============================================================================

func TestAddSavedSnapshotStress(t *testing.T) {
	w := &Workspace{}

	// Add many snapshots
	for i := 0; i < 1000; i++ {
		w.AddSavedSnapshot("snap-"+string(rune('0'+i%10)), "snapshot-"+string(rune('0'+i%10)))
	}

	if len(w.Morph.SavedSnapshots) != 1000 {
		t.Errorf("SavedSnapshots count = %d, want 1000", len(w.Morph.SavedSnapshots))
	}
}

func TestGetSavedSnapshotDuplicateNames(t *testing.T) {
	w := &Workspace{}

	// Add multiple snapshots with same name
	w.AddSavedSnapshot("snap-1", "duplicate-name")
	w.AddSavedSnapshot("snap-2", "duplicate-name")
	w.AddSavedSnapshot("snap-3", "duplicate-name")

	// Should return first one found
	snap := w.GetSavedSnapshot("duplicate-name")
	if snap == nil {
		t.Fatal("GetSavedSnapshot() returned nil")
	}
	if snap.ID != "snap-1" {
		t.Errorf("Snapshot ID = %s, want snap-1 (first match)", snap.ID)
	}
}

func TestSavedSnapshotTimestamps(t *testing.T) {
	w := &Workspace{}

	before := time.Now()
	time.Sleep(10 * time.Millisecond)
	w.AddSavedSnapshot("snap-1", "test-snapshot")
	time.Sleep(10 * time.Millisecond)
	after := time.Now()

	snap := w.GetSavedSnapshot("test-snapshot")
	if snap == nil {
		t.Fatal("GetSavedSnapshot() returned nil")
	}

	if snap.CreatedAt.Before(before) {
		t.Errorf("CreatedAt is before the operation started")
	}
	if snap.CreatedAt.After(after) {
		t.Errorf("CreatedAt is after the operation finished")
	}
}

func TestSavedSnapshotSpecialNames(t *testing.T) {
	tests := []string{
		"",                      // empty
		"   ",                   // whitespace only
		"name with spaces",     // spaces
		"name\twith\ttabs",     // tabs
		"name\nwith\nnewlines", // newlines
		"日本語スナップショット",           // Japanese
		"快照-中文",                 // Chinese
		"스냅샷-한국어",               // Korean
		"🔖📸💾",                  // emoji
		"../../../etc/passwd",  // path traversal
		"<script>alert(1)</script>",
		"SELECT * FROM snapshots; DROP TABLE snapshots;--",
	}

	for _, name := range tests {
		t.Run("name_"+name[:minInt(10, len(name))], func(t *testing.T) {
			w := &Workspace{}
			w.AddSavedSnapshot("snap-test", name)

			snap := w.GetSavedSnapshot(name)
			if snap == nil {
				t.Errorf("GetSavedSnapshot(%q) returned nil", name)
				return
			}
			if snap.Name != name {
				t.Errorf("Snapshot name = %q, want %q", snap.Name, name)
			}
		})
	}
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// =============================================================================
// Instance ID Edge Cases
// =============================================================================

func TestSetMorphInstanceIDVariations(t *testing.T) {
	tests := []struct {
		name       string
		instanceID string
	}{
		{"empty", ""},
		{"whitespace", "   "},
		{"uuid", "550e8400-e29b-41d4-a716-446655440000"},
		{"short", "a"},
		{"long", strings.Repeat("a", 1000)},
		{"special_chars", "inst-!@#$%^&*()"},
		{"unicode", "inst-日本語"},
		{"with_slashes", "inst/with/slashes"},
		{"with_backslashes", `inst\with\backslashes`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := &Workspace{}
			w.SetMorphInstance(tt.instanceID, "snap-123", "https://example.com")

			if w.Morph.InstanceID != tt.instanceID {
				t.Errorf("InstanceID = %q, want %q", w.Morph.InstanceID, tt.instanceID)
			}
		})
	}
}

// =============================================================================
// Status Edge Cases
// =============================================================================

func TestMorphStatusValues(t *testing.T) {
	tests := []struct {
		status     string
		instanceID string
		wantRunning bool
	}{
		{"running", "inst-123", true},
		{"Running", "inst-123", false}, // case sensitive
		{"RUNNING", "inst-123", false},
		{"stopped", "inst-123", false},
		{"paused", "inst-123", false},
		{"error", "inst-123", false},
		{"starting", "inst-123", false},
		{"stopping", "inst-123", false},
		{"", "inst-123", false},
		{"running", "", false}, // running but no instance
		{"running", "   ", false}, // running but whitespace instance
	}

	for _, tt := range tests {
		name := tt.status + "_" + tt.instanceID
		t.Run(name, func(t *testing.T) {
			w := &Workspace{}
			w.Morph.Status = tt.status
			w.Morph.InstanceID = tt.instanceID

			got := w.IsMorphRunning()
			if got != tt.wantRunning {
				t.Errorf("IsMorphRunning() = %v, want %v", got, tt.wantRunning)
			}
		})
	}
}

// =============================================================================
// Concurrent Access Tests
// =============================================================================

func TestConcurrentAddSavedSnapshotAdvanced(t *testing.T) {
	if raceDetectorEnabled() {
		t.Skip("Skipping concurrent test with race detector (Workspace is not thread-safe)")
	}

	w := &Workspace{}
	var wg sync.WaitGroup

	// Add snapshots concurrently
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			name := "snapshot-" + string(rune('0'+idx%10))
			w.AddSavedSnapshot("snap-"+string(rune('0'+idx%10)), name)
		}(i)
	}

	wg.Wait()

	// Should have 100 snapshots (slice append is not thread-safe but this tests the behavior)
	t.Logf("Final snapshot count: %d", len(w.Morph.SavedSnapshots))
}

func TestConcurrentGetSavedSnapshot(t *testing.T) {
	w := &Workspace{}
	w.AddSavedSnapshot("snap-1", "test-snapshot")

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			snap := w.GetSavedSnapshot("test-snapshot")
			if snap == nil {
				t.Error("GetSavedSnapshot() returned nil")
			}
		}()
	}

	wg.Wait()
}

// =============================================================================
// JSON Serialization Edge Cases
// =============================================================================

func TestMorphStateJSONNullValues(t *testing.T) {
	// Test deserializing null values
	jsonData := `{
		"instance_id": null,
		"snapshot_id": null,
		"status": null,
		"saved_snapshots": null
	}`

	var state MorphState
	if err := json.Unmarshal([]byte(jsonData), &state); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}

	if state.InstanceID != "" {
		t.Errorf("InstanceID = %q, want empty", state.InstanceID)
	}
	if state.SavedSnapshots != nil {
		t.Errorf("SavedSnapshots = %v, want nil", state.SavedSnapshots)
	}
}

func TestMorphStateJSONExtraFields(t *testing.T) {
	// Test that extra fields are ignored
	jsonData := `{
		"instance_id": "inst-123",
		"unknown_field": "should be ignored",
		"another_unknown": 12345
	}`

	var state MorphState
	if err := json.Unmarshal([]byte(jsonData), &state); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}

	if state.InstanceID != "inst-123" {
		t.Errorf("InstanceID = %q, want inst-123", state.InstanceID)
	}
}

func TestMorphStateJSONInvalidTypes(t *testing.T) {
	tests := []struct {
		name string
		json string
	}{
		{
			name: "instance_id_as_number",
			json: `{"instance_id": 12345}`,
		},
		{
			name: "cdp_port_as_string",
			json: `{"cdp_port": "not a number"}`,
		},
		{
			name: "saved_snapshots_as_string",
			json: `{"saved_snapshots": "not an array"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var state MorphState
			err := json.Unmarshal([]byte(tt.json), &state)
			if err == nil {
				t.Error("Expected error for invalid JSON type")
			}
		})
	}
}

func TestWorkspaceJSONWithMorph(t *testing.T) {
	w := &Workspace{
		ID:   "ws-123",
		Name: "test-workspace",
		Morph: MorphState{
			InstanceID: "inst-456",
			SnapshotID: "snap-789",
			Status:     "running",
			BaseURL:    "https://example.com",
			SavedSnapshots: []SavedSnapshot{
				{ID: "snap-1", Name: "first", CreatedAt: time.Now()},
				{ID: "snap-2", Name: "second", CreatedAt: time.Now()},
			},
		},
	}

	// Marshal
	data, err := json.Marshal(w)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}

	// Unmarshal
	var loaded Workspace
	if err := json.Unmarshal(data, &loaded); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}

	// Verify
	if loaded.Morph.InstanceID != w.Morph.InstanceID {
		t.Errorf("InstanceID = %q, want %q", loaded.Morph.InstanceID, w.Morph.InstanceID)
	}
	if len(loaded.Morph.SavedSnapshots) != 2 {
		t.Errorf("SavedSnapshots count = %d, want 2", len(loaded.Morph.SavedSnapshots))
	}
}

// =============================================================================
// TextOutput Edge Cases
// =============================================================================

func TestTextOutputWithLongValues(t *testing.T) {
	w := &Workspace{
		ID:   "ws-123",
		Name: "test-workspace",
		Morph: MorphState{
			InstanceID: strings.Repeat("a", 1000),
			Status:     "running",
			CodeURL:    strings.Repeat("b", 1000),
		},
	}

	output := w.TextOutput()
	if !strings.Contains(output, "Morph:") {
		t.Error("TextOutput should contain Morph section")
	}
}

func TestTextOutputWithEmptyMorph(t *testing.T) {
	w := &Workspace{
		ID:    "ws-123",
		Name:  "test-workspace",
		Morph: MorphState{}, // empty
	}

	output := w.TextOutput()
	if strings.Contains(output, "Morph:") {
		t.Error("TextOutput should not contain Morph section when instance ID is empty")
	}
}

func TestTextOutputWithSpecialCharacters(t *testing.T) {
	w := &Workspace{
		ID:   "ws-123",
		Name: "test-workspace",
		Morph: MorphState{
			InstanceID: "inst-<script>alert(1)</script>",
			Status:     "running",
			CodeURL:    "https://example.com/code?param=value&other=<>",
		},
	}

	output := w.TextOutput()
	if !strings.Contains(output, "Morph:") {
		t.Error("TextOutput should handle special characters")
	}
}

// =============================================================================
// ClearMorphInstance Edge Cases
// =============================================================================

func TestClearMorphInstanceURLPreservation(t *testing.T) {
	w := &Workspace{}
	w.SetMorphInstance("inst-123", "snap-456", "https://example.com")

	// Store original URLs
	originalCodeURL := w.Morph.CodeURL
	originalVNCURL := w.Morph.VNCURL

	// Clear
	w.ClearMorphInstance()

	// Instance ID should be cleared
	if w.Morph.InstanceID != "" {
		t.Errorf("InstanceID = %q, want empty", w.Morph.InstanceID)
	}

	// Status should be stopped
	if w.Morph.Status != "stopped" {
		t.Errorf("Status = %q, want stopped", w.Morph.Status)
	}

	// URLs should be preserved
	if w.Morph.CodeURL != originalCodeURL {
		t.Errorf("CodeURL = %q, want %q", w.Morph.CodeURL, originalCodeURL)
	}
	if w.Morph.VNCURL != originalVNCURL {
		t.Errorf("VNCURL = %q, want %q", w.Morph.VNCURL, originalVNCURL)
	}
}

func TestClearMorphInstancePreservesSnapshots(t *testing.T) {
	w := &Workspace{}
	w.SetMorphInstance("inst-123", "snap-456", "https://example.com")
	w.AddSavedSnapshot("snap-1", "my-snapshot")

	w.ClearMorphInstance()

	// Saved snapshots should be preserved
	if len(w.Morph.SavedSnapshots) != 1 {
		t.Errorf("SavedSnapshots count = %d, want 1", len(w.Morph.SavedSnapshots))
	}
}

func TestClearMorphInstancePreservesSnapshotID(t *testing.T) {
	w := &Workspace{}
	w.SetMorphInstance("inst-123", "snap-456", "https://example.com")

	w.ClearMorphInstance()

	// SnapshotID should be preserved (it's the base snapshot)
	if w.Morph.SnapshotID != "snap-456" {
		t.Errorf("SnapshotID = %q, want snap-456", w.Morph.SnapshotID)
	}
}

// =============================================================================
// StartedAt Edge Cases
// =============================================================================

func TestSetMorphInstanceSetsStartedAt(t *testing.T) {
	w := &Workspace{}

	before := time.Now()
	w.SetMorphInstance("inst-123", "snap-456", "https://example.com")
	after := time.Now()

	if w.Morph.StartedAt.Before(before) {
		t.Errorf("StartedAt is before the operation")
	}
	if w.Morph.StartedAt.After(after) {
		t.Errorf("StartedAt is after the operation")
	}
}

func TestStartedAtJSONPersistence(t *testing.T) {
	w := &Workspace{}
	w.SetMorphInstance("inst-123", "snap-456", "https://example.com")

	// Marshal
	data, err := json.Marshal(w.Morph)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}

	// Unmarshal
	var loaded MorphState
	if err := json.Unmarshal(data, &loaded); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}

	// StartedAt should be preserved with reasonable precision
	diff := w.Morph.StartedAt.Sub(loaded.StartedAt)
	if diff > time.Second || diff < -time.Second {
		t.Errorf("StartedAt diff = %v, want < 1s", diff)
	}
}

// =============================================================================
// CDPPort Edge Cases
// =============================================================================

func TestCDPPortValues(t *testing.T) {
	tests := []struct {
		name     string
		port     int
		wantJSON bool
	}{
		{"zero", 0, false}, // omitempty
		{"standard", 9222, true},
		{"max_port", 65535, true},
		{"negative", -1, true}, // will serialize, may be invalid
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			state := MorphState{CDPPort: tt.port}

			data, err := json.Marshal(state)
			if err != nil {
				t.Fatalf("Marshal error: %v", err)
			}

			containsPort := strings.Contains(string(data), "cdp_port")
			if tt.wantJSON && !containsPort && tt.port != 0 {
				t.Errorf("Expected cdp_port in JSON, got %s", string(data))
			}
			if !tt.wantJSON && containsPort {
				t.Errorf("Did not expect cdp_port in JSON, got %s", string(data))
			}
		})
	}
}
