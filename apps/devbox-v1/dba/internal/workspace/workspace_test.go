// internal/workspace/workspace_test.go
package workspace

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestWorkspaceURLs(t *testing.T) {
	tests := []struct {
		name     string
		ports    map[string]int
		expected map[string]string
	}{
		{
			name: "all standard ports",
			ports: map[string]int{
				"PORT":      10000,
				"CODE_PORT": 10080,
				"VNC_PORT":  10090,
			},
			expected: map[string]string{
				"app":  "http://localhost:10000",
				"code": "http://localhost:10080",
				"vnc":  "vnc://localhost:10090",
			},
		},
		{
			name: "only PORT",
			ports: map[string]int{
				"PORT": 3000,
			},
			expected: map[string]string{
				"app": "http://localhost:3000",
			},
		},
		{
			name:     "no ports",
			ports:    map[string]int{},
			expected: map[string]string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ws := &Workspace{Ports: tt.ports}
			urls := ws.URLs()

			if len(urls) != len(tt.expected) {
				t.Errorf("expected %d URLs, got %d", len(tt.expected), len(urls))
			}

			for key, expectedURL := range tt.expected {
				if urls[key] != expectedURL {
					t.Errorf("expected URL[%s] = %s, got %s", key, expectedURL, urls[key])
				}
			}
		})
	}
}

func TestWorkspacePaths(t *testing.T) {
	ws := &Workspace{
		Path: "/test/workspace",
	}

	if ws.StateDir() != "/test/workspace/.dba" {
		t.Errorf("expected StateDir = /test/workspace/.dba, got %s", ws.StateDir())
	}

	if ws.StatePath() != "/test/workspace/.dba/state.json" {
		t.Errorf("expected StatePath = /test/workspace/.dba/state.json, got %s", ws.StatePath())
	}

	if ws.DevboxPath() != "/test/workspace/devbox.json" {
		t.Errorf("expected DevboxPath = /test/workspace/devbox.json, got %s", ws.DevboxPath())
	}

	if ws.ProcessComposePath() != "/test/workspace/process-compose.yaml" {
		t.Errorf("expected ProcessComposePath = /test/workspace/process-compose.yaml, got %s", ws.ProcessComposePath())
	}

	if ws.LogsDir() != "/test/workspace/.dba/logs" {
		t.Errorf("expected LogsDir = /test/workspace/.dba/logs, got %s", ws.LogsDir())
	}
}

func TestWorkspaceTextOutput(t *testing.T) {
	ws := &Workspace{
		ID:       "ws_test123",
		Name:     "test-workspace",
		Path:     "/test/workspace",
		Template: "node",
		Status:   "ready",
		Ports: map[string]int{
			"PORT":      10000,
			"CODE_PORT": 10080,
		},
	}

	output := ws.TextOutput()

	// Check that output contains key information
	if output == "" {
		t.Error("TextOutput should not be empty")
	}

	// Should contain workspace name and ID
	if !contains(output, "test-workspace") {
		t.Error("TextOutput should contain workspace name")
	}

	if !contains(output, "ws_test123") {
		t.Error("TextOutput should contain workspace ID")
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func TestStateExists(t *testing.T) {
	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "workspace_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Initially should not exist
	if StateExists(tmpDir) {
		t.Error("StateExists should return false for non-existent state")
	}

	// Create .dba directory and state.json
	dbaDir := filepath.Join(tmpDir, ".dba")
	if err := os.MkdirAll(dbaDir, 0755); err != nil {
		t.Fatal(err)
	}

	statePath := filepath.Join(dbaDir, "state.json")
	if err := os.WriteFile(statePath, []byte("{}"), 0644); err != nil {
		t.Fatal(err)
	}

	// Now should exist
	if !StateExists(tmpDir) {
		t.Error("StateExists should return true for existing state")
	}
}

func TestSaveAndLoadState(t *testing.T) {
	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "workspace_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create workspace
	ws := &Workspace{
		ID:          "ws_test456",
		Name:        "test-save-load",
		Path:        tmpDir,
		ProjectPath: filepath.Join(tmpDir, "project"),
		Template:    "python",
		Status:      "ready",
		BasePort:    10000,
		Ports: map[string]int{
			"PORT":      10000,
			"CODE_PORT": 10080,
		},
		Packages:   []string{"python@3.12"},
		CreatedAt:  time.Now(),
		LastActive: time.Now(),
		Git: &GitInfo{
			Remote: "https://github.com/test/repo",
			Branch: "main",
			Commit: "abc123",
		},
	}

	// Save state
	if err := ws.SaveState(); err != nil {
		t.Fatalf("SaveState failed: %v", err)
	}

	// Verify state file exists
	if !StateExists(tmpDir) {
		t.Error("State file should exist after SaveState")
	}

	// Load state
	loadedState, err := LoadState(tmpDir)
	if err != nil {
		t.Fatalf("LoadState failed: %v", err)
	}

	// Verify loaded state
	if loadedState.ID != ws.ID {
		t.Errorf("expected ID = %s, got %s", ws.ID, loadedState.ID)
	}
	if loadedState.Name != ws.Name {
		t.Errorf("expected Name = %s, got %s", ws.Name, loadedState.Name)
	}
	if loadedState.Template != ws.Template {
		t.Errorf("expected Template = %s, got %s", ws.Template, loadedState.Template)
	}
	if loadedState.BasePort != ws.BasePort {
		t.Errorf("expected BasePort = %d, got %d", ws.BasePort, loadedState.BasePort)
	}
	if len(loadedState.Ports) != len(ws.Ports) {
		t.Errorf("expected %d ports, got %d", len(ws.Ports), len(loadedState.Ports))
	}
	if loadedState.Git == nil {
		t.Error("Git info should not be nil")
	} else if loadedState.Git.Remote != ws.Git.Remote {
		t.Errorf("expected Git.Remote = %s, got %s", ws.Git.Remote, loadedState.Git.Remote)
	}

	// Test Load function
	loadedWs, err := Load(tmpDir)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if loadedWs.ID != ws.ID {
		t.Errorf("Load: expected ID = %s, got %s", ws.ID, loadedWs.ID)
	}
	if loadedWs.Path != tmpDir {
		t.Errorf("Load: expected Path = %s, got %s", tmpDir, loadedWs.Path)
	}
}

func TestLoadStateNonExistent(t *testing.T) {
	_, err := LoadState("/nonexistent/path")
	if err == nil {
		t.Error("LoadState should fail for non-existent path")
	}
}

func TestUpdateLastActive(t *testing.T) {
	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "workspace_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	ws := &Workspace{
		ID:          "ws_test789",
		Name:        "test-update",
		Path:        tmpDir,
		ProjectPath: filepath.Join(tmpDir, "project"),
		Template:    "node",
		Status:      "ready",
		BasePort:    10000,
		Ports:       map[string]int{"PORT": 10000},
		CreatedAt:   time.Now().Add(-1 * time.Hour),
		LastActive:  time.Now().Add(-1 * time.Hour),
	}

	// Save initial state
	if err := ws.SaveState(); err != nil {
		t.Fatal(err)
	}

	oldLastActive := ws.LastActive

	// Wait a bit and update
	time.Sleep(10 * time.Millisecond)
	if err := ws.UpdateLastActive(); err != nil {
		t.Fatalf("UpdateLastActive failed: %v", err)
	}

	if !ws.LastActive.After(oldLastActive) {
		t.Error("LastActive should be updated to a newer time")
	}

	// Verify persistence
	loadedState, err := LoadState(tmpDir)
	if err != nil {
		t.Fatal(err)
	}

	if loadedState.LastActive.Before(oldLastActive) || loadedState.LastActive.Equal(oldLastActive) {
		t.Error("Persisted LastActive should be updated")
	}
}

func TestSetStatus(t *testing.T) {
	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "workspace_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	ws := &Workspace{
		ID:          "ws_status_test",
		Name:        "test-status",
		Path:        tmpDir,
		ProjectPath: filepath.Join(tmpDir, "project"),
		Template:    "node",
		Status:      "ready",
		BasePort:    10000,
		Ports:       map[string]int{"PORT": 10000},
		CreatedAt:   time.Now(),
		LastActive:  time.Now(),
	}

	// Save initial state
	if err := ws.SaveState(); err != nil {
		t.Fatal(err)
	}

	// Change status
	if err := ws.SetStatus("running"); err != nil {
		t.Fatalf("SetStatus failed: %v", err)
	}

	if ws.Status != "running" {
		t.Errorf("expected Status = running, got %s", ws.Status)
	}

	// Verify persistence
	loadedState, err := LoadState(tmpDir)
	if err != nil {
		t.Fatal(err)
	}

	if loadedState.Status != "running" {
		t.Errorf("Persisted Status should be 'running', got %s", loadedState.Status)
	}
}

// TestMorphState tests the Morph state functionality
func TestMorphState(t *testing.T) {
	ws := &Workspace{
		ID:   "ws_morph_test",
		Name: "test-morph",
	}

	// Initially not running
	if ws.IsMorphRunning() {
		t.Error("IsMorphRunning should return false for new workspace")
	}

	// Set Morph instance
	ws.SetMorphInstance("inst-abc123", "snap-xyz789", "https://example.morph.so")

	// Should be running now
	if !ws.IsMorphRunning() {
		t.Error("IsMorphRunning should return true after SetMorphInstance")
	}

	// Verify instance ID
	if ws.Morph.InstanceID != "inst-abc123" {
		t.Errorf("expected InstanceID = inst-abc123, got %s", ws.Morph.InstanceID)
	}

	// Verify snapshot ID
	if ws.Morph.SnapshotID != "snap-xyz789" {
		t.Errorf("expected SnapshotID = snap-xyz789, got %s", ws.Morph.SnapshotID)
	}

	// Verify status
	if ws.Morph.Status != "running" {
		t.Errorf("expected Status = running, got %s", ws.Morph.Status)
	}

	// Verify URLs are derived
	if ws.Morph.CodeURL != "https://example.morph.so/code/" {
		t.Errorf("expected CodeURL = https://example.morph.so/code/, got %s", ws.Morph.CodeURL)
	}
	if ws.Morph.VNCURL != "https://example.morph.so/vnc/vnc.html" {
		t.Errorf("expected VNCURL = https://example.morph.so/vnc/vnc.html, got %s", ws.Morph.VNCURL)
	}
	if ws.Morph.AppURL != "https://example.morph.so/vnc/app/" {
		t.Errorf("expected AppURL = https://example.morph.so/vnc/app/, got %s", ws.Morph.AppURL)
	}
	if ws.Morph.CDPURL != "wss://example.morph.so/cdp/" {
		t.Errorf("expected CDPURL = wss://example.morph.so/cdp/, got %s", ws.Morph.CDPURL)
	}

	// Verify StartedAt is set
	if ws.Morph.StartedAt.IsZero() {
		t.Error("StartedAt should be set")
	}

	// Clear instance
	ws.ClearMorphInstance()

	// Should not be running anymore
	if ws.IsMorphRunning() {
		t.Error("IsMorphRunning should return false after ClearMorphInstance")
	}

	// Instance ID should be cleared
	if ws.Morph.InstanceID != "" {
		t.Errorf("expected empty InstanceID, got %s", ws.Morph.InstanceID)
	}

	// Status should be stopped
	if ws.Morph.Status != "stopped" {
		t.Errorf("expected Status = stopped, got %s", ws.Morph.Status)
	}

	// URLs should be preserved
	if ws.Morph.BaseURL != "https://example.morph.so" {
		t.Errorf("BaseURL should be preserved, got %s", ws.Morph.BaseURL)
	}
}

// TestSavedSnapshots tests the saved snapshot functionality
func TestSavedSnapshots(t *testing.T) {
	ws := &Workspace{
		ID:   "ws_snapshot_test",
		Name: "test-snapshots",
	}

	// Initially no saved snapshots
	if len(ws.Morph.SavedSnapshots) != 0 {
		t.Error("should have no saved snapshots initially")
	}

	// Add a saved snapshot
	ws.AddSavedSnapshot("snap-001", "initial-state")

	// Should have one snapshot
	if len(ws.Morph.SavedSnapshots) != 1 {
		t.Errorf("expected 1 snapshot, got %d", len(ws.Morph.SavedSnapshots))
	}

	// Get saved snapshot by name
	snapshot := ws.GetSavedSnapshot("initial-state")
	if snapshot == nil {
		t.Fatal("GetSavedSnapshot should find initial-state")
	}
	if snapshot.ID != "snap-001" {
		t.Errorf("expected snapshot ID = snap-001, got %s", snapshot.ID)
	}
	if snapshot.Name != "initial-state" {
		t.Errorf("expected snapshot Name = initial-state, got %s", snapshot.Name)
	}
	if snapshot.CreatedAt.IsZero() {
		t.Error("snapshot CreatedAt should be set")
	}

	// Add another snapshot
	ws.AddSavedSnapshot("snap-002", "logged-in")

	// Should have two snapshots
	if len(ws.Morph.SavedSnapshots) != 2 {
		t.Errorf("expected 2 snapshots, got %d", len(ws.Morph.SavedSnapshots))
	}

	// Get second snapshot
	snapshot2 := ws.GetSavedSnapshot("logged-in")
	if snapshot2 == nil {
		t.Fatal("GetSavedSnapshot should find logged-in")
	}
	if snapshot2.ID != "snap-002" {
		t.Errorf("expected snapshot ID = snap-002, got %s", snapshot2.ID)
	}

	// Try to get non-existent snapshot
	notFound := ws.GetSavedSnapshot("non-existent")
	if notFound != nil {
		t.Error("GetSavedSnapshot should return nil for non-existent snapshot")
	}
}

// TestGetMorphURLs tests the GetMorphURLs helper
func TestGetMorphURLs(t *testing.T) {
	ws := &Workspace{
		ID:   "ws_urls_test",
		Name: "test-urls",
	}

	// No URLs initially
	urls := ws.GetMorphURLs()
	if len(urls) != 0 {
		t.Errorf("expected 0 URLs, got %d", len(urls))
	}

	// Set instance with URLs
	ws.SetMorphInstance("inst-123", "snap-456", "https://test.morph.so")

	// Should have all URLs
	urls = ws.GetMorphURLs()
	if len(urls) != 4 {
		t.Errorf("expected 4 URLs, got %d", len(urls))
	}

	if urls["code"] != "https://test.morph.so/code/" {
		t.Errorf("expected code URL, got %s", urls["code"])
	}
	if urls["vnc"] != "https://test.morph.so/vnc/vnc.html" {
		t.Errorf("expected vnc URL, got %s", urls["vnc"])
	}
	if urls["app"] != "https://test.morph.so/vnc/app/" {
		t.Errorf("expected app URL, got %s", urls["app"])
	}
	if urls["cdp"] != "wss://test.morph.so/cdp/" {
		t.Errorf("expected cdp URL, got %s", urls["cdp"])
	}
}

// TestMorphStateTextOutput tests that Morph state appears in TextOutput
func TestMorphStateTextOutput(t *testing.T) {
	ws := &Workspace{
		ID:       "ws_text_test",
		Name:     "test-text",
		Path:     "/test/path",
		Template: "node",
		Status:   "ready",
		Ports:    map[string]int{"PORT": 10000},
	}

	// Without Morph state
	output := ws.TextOutput()
	if contains(output, "Morph:") {
		t.Error("TextOutput should not contain Morph section when no instance")
	}

	// With Morph state
	ws.SetMorphInstance("inst-xyz", "snap-abc", "https://ws-test.morph.so")
	output = ws.TextOutput()

	if !contains(output, "Morph:") {
		t.Error("TextOutput should contain Morph section when instance is set")
	}
	if !contains(output, "inst-xyz") {
		t.Error("TextOutput should contain instance ID")
	}
	if !contains(output, "running") {
		t.Error("TextOutput should contain status")
	}
}

// TestMorphStatePersistence tests that Morph state is properly serialized
func TestMorphStatePersistence(t *testing.T) {
	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "workspace_morph_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create workspace with Morph state
	ws := &Workspace{
		ID:          "ws_persist_morph",
		Name:        "test-persist-morph",
		Path:        tmpDir,
		ProjectPath: filepath.Join(tmpDir, "project"),
		Template:    "node",
		Status:      "ready",
		BasePort:    10000,
		Ports:       map[string]int{"PORT": 10000},
		CreatedAt:   time.Now(),
		LastActive:  time.Now(),
	}

	// Set Morph state
	ws.SetMorphInstance("morph-inst-123", "snap-base-456", "https://persist-test.morph.so")
	ws.AddSavedSnapshot("snap-saved-001", "my-checkpoint")
	ws.Morph.CDPPort = 9222

	// Save state
	if err := ws.SaveState(); err != nil {
		t.Fatalf("SaveState failed: %v", err)
	}

	// Load state and verify Morph data
	loadedState, err := LoadState(tmpDir)
	if err != nil {
		t.Fatalf("LoadState failed: %v", err)
	}

	if loadedState.Morph.InstanceID != "morph-inst-123" {
		t.Errorf("expected InstanceID = morph-inst-123, got %s", loadedState.Morph.InstanceID)
	}
	if loadedState.Morph.SnapshotID != "snap-base-456" {
		t.Errorf("expected SnapshotID = snap-base-456, got %s", loadedState.Morph.SnapshotID)
	}
	if loadedState.Morph.Status != "running" {
		t.Errorf("expected Status = running, got %s", loadedState.Morph.Status)
	}
	if loadedState.Morph.BaseURL != "https://persist-test.morph.so" {
		t.Errorf("expected BaseURL = https://persist-test.morph.so, got %s", loadedState.Morph.BaseURL)
	}
	if loadedState.Morph.CodeURL != "https://persist-test.morph.so/code/" {
		t.Errorf("expected CodeURL to be derived, got %s", loadedState.Morph.CodeURL)
	}
	if loadedState.Morph.CDPPort != 9222 {
		t.Errorf("expected CDPPort = 9222, got %d", loadedState.Morph.CDPPort)
	}
	if len(loadedState.Morph.SavedSnapshots) != 1 {
		t.Errorf("expected 1 saved snapshot, got %d", len(loadedState.Morph.SavedSnapshots))
	}
	if loadedState.Morph.SavedSnapshots[0].Name != "my-checkpoint" {
		t.Errorf("expected snapshot name = my-checkpoint, got %s", loadedState.Morph.SavedSnapshots[0].Name)
	}
}
