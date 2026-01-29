// internal/workspace/morph_state_transitions_test.go
package workspace

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// =============================================================================
// State Machine Transition Tests
// =============================================================================

func TestMorphStateTransitions(t *testing.T) {
	tests := []struct {
		name     string
		actions  []func(*Workspace)
		expected func(*Workspace) bool
	}{
		{
			name: "initial -> running",
			actions: []func(*Workspace){
				func(ws *Workspace) { ws.SetMorphInstance("inst", "snap", "https://a.com") },
			},
			expected: func(ws *Workspace) bool {
				return ws.IsMorphRunning() && ws.Morph.Status == "running"
			},
		},
		{
			name: "initial -> running -> stopped",
			actions: []func(*Workspace){
				func(ws *Workspace) { ws.SetMorphInstance("inst", "snap", "https://a.com") },
				func(ws *Workspace) { ws.ClearMorphInstance() },
			},
			expected: func(ws *Workspace) bool {
				return !ws.IsMorphRunning() && ws.Morph.Status == "stopped"
			},
		},
		{
			name: "running -> running (different instance)",
			actions: []func(*Workspace){
				func(ws *Workspace) { ws.SetMorphInstance("inst1", "snap1", "https://a.com") },
				func(ws *Workspace) { ws.SetMorphInstance("inst2", "snap2", "https://b.com") },
			},
			expected: func(ws *Workspace) bool {
				return ws.IsMorphRunning() && ws.Morph.InstanceID == "inst2"
			},
		},
		{
			name: "stopped -> running",
			actions: []func(*Workspace){
				func(ws *Workspace) { ws.ClearMorphInstance() },
				func(ws *Workspace) { ws.SetMorphInstance("inst", "snap", "https://a.com") },
			},
			expected: func(ws *Workspace) bool {
				return ws.IsMorphRunning()
			},
		},
		{
			name: "running -> stopped -> running",
			actions: []func(*Workspace){
				func(ws *Workspace) { ws.SetMorphInstance("inst1", "snap1", "https://a.com") },
				func(ws *Workspace) { ws.ClearMorphInstance() },
				func(ws *Workspace) { ws.SetMorphInstance("inst2", "snap2", "https://b.com") },
			},
			expected: func(ws *Workspace) bool {
				return ws.IsMorphRunning() && ws.Morph.InstanceID == "inst2"
			},
		},
		{
			name: "multiple stop calls",
			actions: []func(*Workspace){
				func(ws *Workspace) { ws.SetMorphInstance("inst", "snap", "https://a.com") },
				func(ws *Workspace) { ws.ClearMorphInstance() },
				func(ws *Workspace) { ws.ClearMorphInstance() },
				func(ws *Workspace) { ws.ClearMorphInstance() },
			},
			expected: func(ws *Workspace) bool {
				return ws.Morph.Status == "stopped" && ws.Morph.InstanceID == ""
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ws := &Workspace{ID: "test"}

			for _, action := range tt.actions {
				action(ws)
			}

			if !tt.expected(ws) {
				t.Errorf("state transition test failed")
			}
		})
	}
}

// =============================================================================
// Snapshot Lifecycle Tests
// =============================================================================

func TestSnapshotLifecycle(t *testing.T) {
	ws := &Workspace{ID: "test"}

	// 1. Start from base snapshot
	ws.SetMorphInstance("inst-1", "base-snapshot", "https://example.com")

	if ws.Morph.SnapshotID != "base-snapshot" {
		t.Errorf("should start from base snapshot")
	}

	// 2. Add checkpoint
	ws.AddSavedSnapshot("checkpoint-1", "after-login")

	if len(ws.Morph.SavedSnapshots) != 1 {
		t.Errorf("should have 1 checkpoint")
	}

	// 3. Stop
	ws.ClearMorphInstance()

	// 4. Snapshots should be preserved
	if len(ws.Morph.SavedSnapshots) != 1 {
		t.Errorf("checkpoints should be preserved after stop")
	}

	// 5. Start from checkpoint
	ws.SetMorphInstance("inst-2", "checkpoint-1", "https://example.com")

	if ws.Morph.SnapshotID != "checkpoint-1" {
		t.Errorf("should start from checkpoint")
	}

	// 6. Add another checkpoint
	ws.AddSavedSnapshot("checkpoint-2", "after-checkout")

	if len(ws.Morph.SavedSnapshots) != 2 {
		t.Errorf("should have 2 checkpoints now")
	}
}

// =============================================================================
// File I/O Edge Cases
// =============================================================================

func TestSaveLoadCorruptedState(t *testing.T) {
	tmpDir := t.TempDir()
	dbaDir := filepath.Join(tmpDir, ".dba")
	if err := os.MkdirAll(dbaDir, 0755); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name     string
		content  string
		wantErr  bool
	}{
		{"empty file", "", true},
		{"invalid json", "{invalid", true},
		{"just null", "null", false}, // null is valid JSON
		{"just number", "123", true},
		{"just string", `"hello"`, true},
		{"empty object", "{}", false},
		{"valid but minimal", `{"id":"test"}`, false},
		{"array instead of object", "[]", true},
		{"deep nesting", `{"morph":{"saved_snapshots":[{"id":"s1","name":"n1"}]}}`, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			statePath := filepath.Join(dbaDir, "state.json")
			if err := os.WriteFile(statePath, []byte(tt.content), 0644); err != nil {
				t.Fatal(err)
			}

			_, err := LoadState(tmpDir)
			if (err != nil) != tt.wantErr {
				t.Errorf("LoadState() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestSaveLoadWithPermissionIssues(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("skipping permission test when running as root")
	}

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

	// Save initially
	if err := ws.SaveState(); err != nil {
		t.Fatal(err)
	}

	// Make state file read-only
	statePath := filepath.Join(tmpDir, ".dba", "state.json")
	if err := os.Chmod(statePath, 0444); err != nil {
		t.Fatal(err)
	}
	defer os.Chmod(statePath, 0644) // cleanup

	// Try to save again - should fail
	ws.Morph.InstanceID = "new-instance"
	err := ws.SaveState()
	if err == nil {
		t.Error("expected error when saving to read-only file")
	}
}

func TestSaveLoadWithSymlinks(t *testing.T) {
	tmpDir := t.TempDir()
	realDir := filepath.Join(tmpDir, "real")
	linkDir := filepath.Join(tmpDir, "link")

	if err := os.MkdirAll(realDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(realDir, linkDir); err != nil {
		t.Skip("symlinks not supported")
	}

	ws := &Workspace{
		ID:          "test",
		Name:        "test",
		Path:        linkDir, // Use symlink path
		ProjectPath: filepath.Join(linkDir, "project"),
		Template:    "node",
		Status:      "ready",
		BasePort:    10000,
		Ports:       map[string]int{"PORT": 10000},
		CreatedAt:   time.Now(),
		LastActive:  time.Now(),
	}

	ws.SetMorphInstance("inst", "snap", "https://example.com")

	// Save via symlink
	if err := ws.SaveState(); err != nil {
		t.Fatalf("SaveState via symlink failed: %v", err)
	}

	// Load via symlink
	loaded, err := Load(linkDir)
	if err != nil {
		t.Fatalf("Load via symlink failed: %v", err)
	}

	if loaded.Morph.InstanceID != "inst" {
		t.Errorf("Morph.InstanceID not preserved via symlink")
	}

	// Also verify via real path
	loadedReal, err := Load(realDir)
	if err != nil {
		t.Fatalf("Load via real path failed: %v", err)
	}

	if loadedReal.Morph.InstanceID != "inst" {
		t.Errorf("Morph.InstanceID not accessible via real path")
	}
}

func TestSaveLoadLargeState(t *testing.T) {
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

	// Add many snapshots with long names
	for i := 0; i < 1000; i++ {
		longID := "snap-" + string(make([]byte, 100))
		longName := "checkpoint-" + string(make([]byte, 100))
		ws.AddSavedSnapshot(longID, longName)
	}

	// Save
	if err := ws.SaveState(); err != nil {
		t.Fatalf("SaveState with large data failed: %v", err)
	}

	// Check file size
	statePath := filepath.Join(tmpDir, ".dba", "state.json")
	info, err := os.Stat(statePath)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("State file size: %d bytes", info.Size())

	// Load
	loaded, err := Load(tmpDir)
	if err != nil {
		t.Fatalf("Load large state failed: %v", err)
	}

	if len(loaded.Morph.SavedSnapshots) != 1000 {
		t.Errorf("expected 1000 snapshots, got %d", len(loaded.Morph.SavedSnapshots))
	}
}

// =============================================================================
// URL Generation Edge Cases
// =============================================================================

func TestURLGeneration(t *testing.T) {
	tests := []struct {
		name        string
		baseURL     string
		expectedCode string
		expectedVNC  string
		expectedApp  string
		expectedCDP  string
	}{
		{
			name:         "standard https",
			baseURL:      "https://example.morph.so",
			expectedCode: "https://example.morph.so/code/",
			expectedVNC:  "https://example.morph.so/vnc/vnc.html",
			expectedApp:  "https://example.morph.so/vnc/app/",
			expectedCDP:  "wss://example.morph.so/cdp/",
		},
		{
			name:         "with trailing slash",
			baseURL:      "https://example.morph.so/",
			expectedCode: "https://example.morph.so/code/",
			expectedVNC:  "https://example.morph.so/vnc/vnc.html",
			expectedApp:  "https://example.morph.so/vnc/app/",
			expectedCDP:  "wss://example.morph.so/cdp/",
		},
		{
			name:         "with port",
			baseURL:      "https://example.morph.so:8443",
			expectedCode: "https://example.morph.so:8443/code/",
			expectedVNC:  "https://example.morph.so:8443/vnc/vnc.html",
			expectedApp:  "https://example.morph.so:8443/vnc/app/",
			expectedCDP:  "wss://example.morph.so:8443/cdp/",
		},
		{
			name:         "http localhost",
			baseURL:      "http://localhost:3000",
			expectedCode: "http://localhost:3000/code/",
			expectedVNC:  "http://localhost:3000/vnc/vnc.html",
			expectedApp:  "http://localhost:3000/vnc/app/",
			expectedCDP:  "ws://localhost:3000/cdp/",
		},
		{
			name:         "ip address",
			baseURL:      "http://192.168.1.100:8080",
			expectedCode: "http://192.168.1.100:8080/code/",
			expectedVNC:  "http://192.168.1.100:8080/vnc/vnc.html",
			expectedApp:  "http://192.168.1.100:8080/vnc/app/",
			expectedCDP:  "ws://192.168.1.100:8080/cdp/",
		},
		{
			name:         "empty",
			baseURL:      "",
			expectedCode: "",
			expectedVNC:  "",
			expectedApp:  "",
			expectedCDP:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ws := &Workspace{ID: "test"}
			ws.SetMorphInstance("inst", "snap", tt.baseURL)

			if ws.Morph.CodeURL != tt.expectedCode {
				t.Errorf("CodeURL = %q, want %q", ws.Morph.CodeURL, tt.expectedCode)
			}
			if ws.Morph.VNCURL != tt.expectedVNC {
				t.Errorf("VNCURL = %q, want %q", ws.Morph.VNCURL, tt.expectedVNC)
			}
			if ws.Morph.AppURL != tt.expectedApp {
				t.Errorf("AppURL = %q, want %q", ws.Morph.AppURL, tt.expectedApp)
			}
			if ws.Morph.CDPURL != tt.expectedCDP {
				t.Errorf("CDPURL = %q, want %q", ws.Morph.CDPURL, tt.expectedCDP)
			}
		})
	}
}

// =============================================================================
// CDPPort Edge Cases
// =============================================================================

func TestCDPPortEdgeCases(t *testing.T) {
	tests := []int{0, 1, 1024, 9222, 65535, -1, 100000}

	for _, port := range tests {
		t.Run("port_"+string(rune(port)), func(t *testing.T) {
			ws := &Workspace{ID: "test"}
			ws.Morph.CDPPort = port

			// Should not panic
			_ = ws.Morph.CDPPort
		})
	}
}

func TestCDPPortPersistence(t *testing.T) {
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

	ws.Morph.CDPPort = 9222

	if err := ws.SaveState(); err != nil {
		t.Fatal(err)
	}

	loaded, err := LoadState(tmpDir)
	if err != nil {
		t.Fatal(err)
	}

	if loaded.Morph.CDPPort != 9222 {
		t.Errorf("CDPPort = %d, want 9222", loaded.Morph.CDPPort)
	}
}

// =============================================================================
// StartedAt Edge Cases
// =============================================================================

func TestStartedAtTimezone(t *testing.T) {
	ws := &Workspace{ID: "test"}

	// Set instance
	before := time.Now()
	ws.SetMorphInstance("inst", "snap", "https://example.com")
	after := time.Now()

	// StartedAt should be between before and after
	if ws.Morph.StartedAt.Before(before) {
		t.Error("StartedAt is before the call")
	}
	if ws.Morph.StartedAt.After(after) {
		t.Error("StartedAt is after the call")
	}
}

func TestStartedAtPersistence(t *testing.T) {
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

	originalTime := time.Date(2026, 1, 28, 12, 0, 0, 0, time.UTC)
	ws.Morph.StartedAt = originalTime

	if err := ws.SaveState(); err != nil {
		t.Fatal(err)
	}

	loaded, err := LoadState(tmpDir)
	if err != nil {
		t.Fatal(err)
	}

	// JSON marshaling uses RFC3339, so nanoseconds might be lost
	if !loaded.Morph.StartedAt.Equal(originalTime) {
		t.Errorf("StartedAt not preserved: got %v, want %v",
			loaded.Morph.StartedAt, originalTime)
	}
}

// =============================================================================
// Integration Tests
// =============================================================================

func TestMorphFullWorkspaceLifecycle(t *testing.T) {
	tmpDir := t.TempDir()

	// 1. Create workspace
	ws := &Workspace{
		ID:          "ws-lifecycle-test",
		Name:        "lifecycle-test",
		Path:        tmpDir,
		ProjectPath: filepath.Join(tmpDir, "project"),
		Template:    "node",
		Status:      "ready",
		BasePort:    10000,
		Ports:       map[string]int{"PORT": 10000},
		CreatedAt:   time.Now(),
		LastActive:  time.Now(),
	}

	// 2. Start Morph VM
	ws.SetMorphInstance("morph-inst-001", "base-snapshot-v1", "https://ws-001.morph.so")

	// 3. Verify running
	if !ws.IsMorphRunning() {
		t.Error("should be running after SetMorphInstance")
	}

	// 4. Save state
	if err := ws.SaveState(); err != nil {
		t.Fatal(err)
	}

	// 5. Create checkpoint
	ws.AddSavedSnapshot("snapshot-after-setup", "after-setup")

	// 6. Save state again
	if err := ws.SaveState(); err != nil {
		t.Fatal(err)
	}

	// 7. Load state
	loadedWs, err := Load(tmpDir)
	if err != nil {
		t.Fatal(err)
	}

	// 8. Verify loaded state
	if !loadedWs.IsMorphRunning() {
		t.Error("loaded workspace should be running")
	}
	if loadedWs.Morph.InstanceID != "morph-inst-001" {
		t.Errorf("InstanceID mismatch")
	}
	if len(loadedWs.Morph.SavedSnapshots) != 1 {
		t.Errorf("should have 1 snapshot, got %d", len(loadedWs.Morph.SavedSnapshots))
	}

	// 9. Stop VM
	loadedWs.ClearMorphInstance()

	// 10. Save
	if err := loadedWs.SaveState(); err != nil {
		t.Fatal(err)
	}

	// 11. Load again
	finalWs, err := Load(tmpDir)
	if err != nil {
		t.Fatal(err)
	}

	// 12. Verify stopped
	if finalWs.IsMorphRunning() {
		t.Error("final state should be stopped")
	}
	if finalWs.Morph.Status != "stopped" {
		t.Errorf("status should be 'stopped', got %q", finalWs.Morph.Status)
	}

	// 13. Snapshots should be preserved
	if len(finalWs.Morph.SavedSnapshots) != 1 {
		t.Errorf("snapshots should be preserved after stop")
	}
}

// =============================================================================
// JSON Edge Cases
// =============================================================================

func TestJSONFieldNaming(t *testing.T) {
	ws := &Workspace{
		ID: "test",
		Morph: MorphState{
			InstanceID: "inst",
			SnapshotID: "snap",
			Status:     "running",
			BaseURL:    "https://example.com",
			CodeURL:    "https://example.com/code/",
			VNCURL:     "https://example.com/vnc/",
			AppURL:     "https://example.com/app/",
			CDPURL:     "https://example.com/cdp/",
			CDPPort:    9222,
		},
	}

	data, err := json.Marshal(ws)
	if err != nil {
		t.Fatal(err)
	}

	// Verify snake_case field names in JSON
	jsonStr := string(data)

	expectedFields := []string{
		`"instance_id"`,
		`"snapshot_id"`,
		`"base_url"`,
		`"code_url"`,
		`"vnc_url"`,
		`"app_url"`,
		`"cdp_url"`,
		`"cdp_port"`,
	}

	for _, field := range expectedFields {
		if !containsString(jsonStr, field) {
			t.Errorf("JSON should contain %s", field)
		}
	}
}

func containsString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
