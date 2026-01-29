// internal/cli/computer_interaction_test.go
package cli

import (
	"strings"
	"testing"
	"time"

	"github.com/dba-cli/dba/internal/workspace"
)

// =============================================================================
// Workspace-Morph Interaction Tests
// =============================================================================

// TestWorkspaceMorphStateTransitions tests workspace Morph state transitions
func TestWorkspaceMorphStateTransitions(t *testing.T) {
	tests := []struct {
		name              string
		initialStatus     string
		initialInstanceID string
		action            string
		expectedStatus    string
		expectedRunning   bool
	}{
		{
			name:              "stopped to running",
			initialStatus:     "stopped",
			initialInstanceID: "",
			action:            "start",
			expectedStatus:    "running",
			expectedRunning:   true,
		},
		{
			name:              "running to stopped",
			initialStatus:     "running",
			initialInstanceID: "inst-123",
			action:            "stop",
			expectedStatus:    "stopped",
			expectedRunning:   false,
		},
		{
			name:              "empty to running",
			initialStatus:     "",
			initialInstanceID: "",
			action:            "start",
			expectedStatus:    "running",
			expectedRunning:   true,
		},
		{
			name:              "paused to running",
			initialStatus:     "paused",
			initialInstanceID: "inst-456",
			action:            "resume",
			expectedStatus:    "running",
			expectedRunning:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ws := &workspace.Workspace{
				Morph: workspace.MorphState{
					Status:     tt.initialStatus,
					InstanceID: tt.initialInstanceID,
				},
			}

			switch tt.action {
			case "start":
				ws.SetMorphInstance("new-inst-id", "snap-123", "https://example.morph.so")
			case "stop":
				ws.ClearMorphInstance()
			case "resume":
				ws.SetMorphInstance(ws.Morph.InstanceID, ws.Morph.SnapshotID, "https://example.morph.so")
			}

			if ws.Morph.Status != tt.expectedStatus {
				t.Errorf("expected status %q, got %q", tt.expectedStatus, ws.Morph.Status)
			}

			if ws.IsMorphRunning() != tt.expectedRunning {
				t.Errorf("expected IsMorphRunning() = %v, got %v", tt.expectedRunning, ws.IsMorphRunning())
			}
		})
	}
}

// TestWorkspaceMorphURLConsistency tests that URLs are consistently formatted
func TestWorkspaceMorphURLConsistency(t *testing.T) {
	baseURLs := []string{
		"https://example.morph.so",
		"https://example.morph.so/",
		"http://localhost:8080",
		"http://localhost:8080/",
		"https://sub.domain.morph.so:443",
		"https://sub.domain.morph.so:443/",
	}

	for _, baseURL := range baseURLs {
		t.Run(baseURL, func(t *testing.T) {
			ws := &workspace.Workspace{}
			ws.SetMorphInstance("inst-123", "snap-456", baseURL)

			urls := ws.GetMorphURLs()

			// All URLs should exist
			if _, ok := urls["code"]; !ok {
				t.Error("code URL should exist")
			}
			if _, ok := urls["vnc"]; !ok {
				t.Error("vnc URL should exist")
			}
			if _, ok := urls["app"]; !ok {
				t.Error("app URL should exist")
			}
			if _, ok := urls["cdp"]; !ok {
				t.Error("cdp URL should exist")
			}

			// No double slashes in URLs (except protocol prefixes)
			for name, url := range urls {
				if strings.Contains(url, "//") &&
					!strings.HasPrefix(url, "https://") &&
					!strings.HasPrefix(url, "http://") &&
					!strings.HasPrefix(url, "wss://") &&
					!strings.HasPrefix(url, "ws://") {
					t.Errorf("%s URL has unexpected double slashes: %s", name, url)
				}
			}

			// URLs should end with / or .html (for vnc)
			for name, url := range urls {
				if !strings.HasSuffix(url, "/") && !strings.HasSuffix(url, ".html") {
					t.Errorf("%s URL should end with / or .html: %s", name, url)
				}
			}
		})
	}
}

// TestWorkspaceSnapshotManagement tests snapshot management
func TestWorkspaceSnapshotManagement(t *testing.T) {
	ws := &workspace.Workspace{}

	// Add multiple snapshots
	ws.AddSavedSnapshot("snap-1", "checkpoint-1")
	ws.AddSavedSnapshot("snap-2", "checkpoint-2")
	ws.AddSavedSnapshot("snap-3", "checkpoint-3")

	// Verify count
	if len(ws.Morph.SavedSnapshots) != 3 {
		t.Errorf("expected 3 snapshots, got %d", len(ws.Morph.SavedSnapshots))
	}

	// Find by name
	snap := ws.GetSavedSnapshot("checkpoint-2")
	if snap == nil {
		t.Error("should find checkpoint-2")
	}
	if snap.ID != "snap-2" {
		t.Errorf("expected snap-2, got %s", snap.ID)
	}

	// Not found
	notFound := ws.GetSavedSnapshot("nonexistent")
	if notFound != nil {
		t.Error("should not find nonexistent snapshot")
	}
}

// TestWorkspaceSnapshotEdgeCases tests snapshot edge cases
func TestWorkspaceSnapshotEdgeCases(t *testing.T) {
	t.Run("duplicate names", func(t *testing.T) {
		ws := &workspace.Workspace{}
		ws.AddSavedSnapshot("snap-1", "checkpoint")
		ws.AddSavedSnapshot("snap-2", "checkpoint") // Same name

		// GetSavedSnapshot returns first match
		snap := ws.GetSavedSnapshot("checkpoint")
		if snap.ID != "snap-1" {
			t.Errorf("expected first match snap-1, got %s", snap.ID)
		}
	})

	t.Run("empty name", func(t *testing.T) {
		ws := &workspace.Workspace{}
		ws.AddSavedSnapshot("snap-1", "")

		snap := ws.GetSavedSnapshot("")
		if snap == nil {
			t.Error("should find snapshot with empty name")
		}
	})

	t.Run("whitespace name", func(t *testing.T) {
		ws := &workspace.Workspace{}
		ws.AddSavedSnapshot("snap-1", "  ")

		snap := ws.GetSavedSnapshot("  ")
		if snap == nil {
			t.Error("should find snapshot with whitespace name")
		}
	})

	t.Run("unicode name", func(t *testing.T) {
		ws := &workspace.Workspace{}
		ws.AddSavedSnapshot("snap-1", "日本語スナップショット")

		snap := ws.GetSavedSnapshot("日本語スナップショット")
		if snap == nil {
			t.Error("should find snapshot with unicode name")
		}
	})
}

// TestWorkspaceMorphStateTimestamps tests timestamp behavior
func TestWorkspaceMorphStateTimestamps(t *testing.T) {
	ws := &workspace.Workspace{}

	before := time.Now()
	ws.SetMorphInstance("inst-123", "snap-456", "https://example.morph.so")
	after := time.Now()

	// StartedAt should be between before and after
	if ws.Morph.StartedAt.Before(before) {
		t.Error("StartedAt should be after test start")
	}
	if ws.Morph.StartedAt.After(after) {
		t.Error("StartedAt should be before test end")
	}

	// Add snapshot and check timestamp
	before = time.Now()
	ws.AddSavedSnapshot("snap-save", "save-point")
	after = time.Now()

	snap := ws.GetSavedSnapshot("save-point")
	if snap.CreatedAt.Before(before) {
		t.Error("CreatedAt should be after snapshot add start")
	}
	if snap.CreatedAt.After(after) {
		t.Error("CreatedAt should be before snapshot add end")
	}
}

// TestWorkspaceTextOutputWithMorph tests text output includes Morph info
func TestWorkspaceTextOutputWithMorph(t *testing.T) {
	t.Run("with running morph", func(t *testing.T) {
		ws := &workspace.Workspace{
			ID:     "ws-test",
			Name:   "test-workspace",
			Path:   "/path/to/workspace",
			Status: "running",
			Ports:  map[string]int{"PORT": 8080},
		}
		ws.SetMorphInstance("inst-abc123", "snap-xyz789", "https://test.morph.so")

		output := ws.TextOutput()

		// Should include Morph section
		if !strings.Contains(output, "Morph:") {
			t.Error("output should contain Morph section")
		}
		if !strings.Contains(output, "inst-abc123") {
			t.Error("output should contain instance ID")
		}
		if !strings.Contains(output, "running") {
			t.Error("output should contain status")
		}
	})

	t.Run("without morph", func(t *testing.T) {
		ws := &workspace.Workspace{
			ID:     "ws-test",
			Name:   "test-workspace",
			Path:   "/path/to/workspace",
			Status: "running",
			Ports:  map[string]int{},
		}

		output := ws.TextOutput()

		// Should NOT include Morph section when not running
		if strings.Contains(output, "Morph:") {
			t.Error("output should not contain Morph section when not running")
		}
	})
}

// TestWorkspaceMorphCDPPortManagement tests CDP port tracking
func TestWorkspaceMorphCDPPortManagement(t *testing.T) {
	ws := &workspace.Workspace{}
	ws.SetMorphInstance("inst-123", "snap-456", "https://example.morph.so")

	// Set CDP port
	ws.Morph.CDPPort = 9222

	if ws.Morph.CDPPort != 9222 {
		t.Errorf("expected CDPPort 9222, got %d", ws.Morph.CDPPort)
	}

	// CDP URL should be set
	if ws.Morph.CDPURL == "" {
		t.Error("CDPURL should be set")
	}

	// Clear instance keeps URLs but clears port
	ws.ClearMorphInstance()

	// CDPURL should still be present (per implementation)
	if ws.Morph.CDPURL == "" {
		t.Error("CDPURL should be preserved after clear")
	}
}

// TestWorkspaceMorphURLPaths tests URL path formatting
func TestWorkspaceMorphURLPaths(t *testing.T) {
	tests := []struct {
		name          string
		baseURL       string
		expectedCode  string
		expectedVNC   string
		expectedApp   string
		expectedCDP   string
	}{
		{
			name:         "simple base URL",
			baseURL:      "https://example.morph.so",
			expectedCode: "https://example.morph.so/code/",
			expectedVNC:  "https://example.morph.so/vnc/vnc.html",
			expectedApp:  "https://example.morph.so/vnc/app/",
			expectedCDP:  "wss://example.morph.so/cdp/",
		},
		{
			name:         "base URL with trailing slash",
			baseURL:      "https://example.morph.so/",
			expectedCode: "https://example.morph.so/code/",
			expectedVNC:  "https://example.morph.so/vnc/vnc.html",
			expectedApp:  "https://example.morph.so/vnc/app/",
			expectedCDP:  "wss://example.morph.so/cdp/",
		},
		{
			name:         "localhost URL",
			baseURL:      "http://localhost:8080",
			expectedCode: "http://localhost:8080/code/",
			expectedVNC:  "http://localhost:8080/vnc/vnc.html",
			expectedApp:  "http://localhost:8080/vnc/app/",
			expectedCDP:  "ws://localhost:8080/cdp/",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ws := &workspace.Workspace{}
			ws.SetMorphInstance("inst-123", "snap-456", tt.baseURL)

			if ws.Morph.CodeURL != tt.expectedCode {
				t.Errorf("CodeURL: expected %q, got %q", tt.expectedCode, ws.Morph.CodeURL)
			}
			if ws.Morph.VNCURL != tt.expectedVNC {
				t.Errorf("VNCURL: expected %q, got %q", tt.expectedVNC, ws.Morph.VNCURL)
			}
			if ws.Morph.AppURL != tt.expectedApp {
				t.Errorf("AppURL: expected %q, got %q", tt.expectedApp, ws.Morph.AppURL)
			}
			if ws.Morph.CDPURL != tt.expectedCDP {
				t.Errorf("CDPURL: expected %q, got %q", tt.expectedCDP, ws.Morph.CDPURL)
			}
		})
	}
}

// TestEmptyBaseURLHandling tests empty base URL handling
func TestEmptyBaseURLHandling(t *testing.T) {
	ws := &workspace.Workspace{}
	ws.SetMorphInstance("inst-123", "snap-456", "")

	// Instance ID should be set
	if ws.Morph.InstanceID != "inst-123" {
		t.Errorf("InstanceID should be set, got %q", ws.Morph.InstanceID)
	}

	// URLs should be empty
	if ws.Morph.CodeURL != "" {
		t.Errorf("CodeURL should be empty, got %q", ws.Morph.CodeURL)
	}
	if ws.Morph.VNCURL != "" {
		t.Errorf("VNCURL should be empty, got %q", ws.Morph.VNCURL)
	}

	// GetMorphURLs should return empty map
	urls := ws.GetMorphURLs()
	if len(urls) != 0 {
		t.Errorf("GetMorphURLs should return empty map, got %d entries", len(urls))
	}
}

// =============================================================================
// State Consistency Tests
// =============================================================================

// TestStateConsistencyAfterOperations tests state consistency
func TestStateConsistencyAfterOperations(t *testing.T) {
	ws := &workspace.Workspace{}

	// Initial state
	if ws.IsMorphRunning() {
		t.Error("initial state should not be running")
	}

	// Set instance
	ws.SetMorphInstance("inst-1", "snap-1", "https://test.morph.so")
	if !ws.IsMorphRunning() {
		t.Error("should be running after SetMorphInstance")
	}

	// Update with new instance
	ws.SetMorphInstance("inst-2", "snap-2", "https://test2.morph.so")
	if !ws.IsMorphRunning() {
		t.Error("should still be running after updating instance")
	}
	if ws.Morph.InstanceID != "inst-2" {
		t.Errorf("instance ID should be updated, got %q", ws.Morph.InstanceID)
	}

	// Clear instance
	ws.ClearMorphInstance()
	if ws.IsMorphRunning() {
		t.Error("should not be running after ClearMorphInstance")
	}

	// Instance ID should be cleared
	if ws.Morph.InstanceID != "" {
		t.Errorf("instance ID should be empty, got %q", ws.Morph.InstanceID)
	}
}

// TestMorphStateIndependence tests that workspaces are independent
func TestMorphStateIndependence(t *testing.T) {
	ws1 := &workspace.Workspace{}
	ws2 := &workspace.Workspace{}

	ws1.SetMorphInstance("inst-1", "snap-1", "https://ws1.morph.so")
	ws2.SetMorphInstance("inst-2", "snap-2", "https://ws2.morph.so")

	// Both should be running
	if !ws1.IsMorphRunning() {
		t.Error("ws1 should be running")
	}
	if !ws2.IsMorphRunning() {
		t.Error("ws2 should be running")
	}

	// Clear ws1
	ws1.ClearMorphInstance()

	// ws1 should be stopped, ws2 should still run
	if ws1.IsMorphRunning() {
		t.Error("ws1 should not be running after clear")
	}
	if !ws2.IsMorphRunning() {
		t.Error("ws2 should still be running")
	}

	// ws2 URLs should be intact
	if ws2.Morph.InstanceID != "inst-2" {
		t.Errorf("ws2 instance ID should be intact, got %q", ws2.Morph.InstanceID)
	}
}
