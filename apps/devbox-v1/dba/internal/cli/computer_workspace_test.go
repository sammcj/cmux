// internal/cli/computer_workspace_test.go
package cli

import (
	"testing"
	"time"

	"github.com/dba-cli/dba/internal/workspace"
)

// TestWorkspaceMorphStateDefaults tests default values of MorphState
func TestWorkspaceMorphStateDefaults(t *testing.T) {
	state := workspace.MorphState{}

	if state.InstanceID != "" {
		t.Error("default InstanceID should be empty")
	}
	if state.SnapshotID != "" {
		t.Error("default SnapshotID should be empty")
	}
	if state.Status != "" {
		t.Error("default Status should be empty")
	}
	if state.BaseURL != "" {
		t.Error("default BaseURL should be empty")
	}
	if state.CodeURL != "" {
		t.Error("default CodeURL should be empty")
	}
	if state.VNCURL != "" {
		t.Error("default VNCURL should be empty")
	}
	if state.AppURL != "" {
		t.Error("default AppURL should be empty")
	}
	if state.CDPURL != "" {
		t.Error("default CDPURL should be empty")
	}
	if state.CDPPort != 0 {
		t.Error("default CDPPort should be 0")
	}
	if !state.StartedAt.IsZero() {
		t.Error("default StartedAt should be zero")
	}
	if len(state.SavedSnapshots) != 0 {
		t.Error("default SavedSnapshots should be empty")
	}
}

// TestWorkspaceMorphStateWithValues tests MorphState with populated values
func TestWorkspaceMorphStateWithValues(t *testing.T) {
	now := time.Now()
	state := workspace.MorphState{
		InstanceID: "morph-inst-123",
		SnapshotID: "snap-abc",
		Status:     "running",
		BaseURL:    "https://example.morph.so",
		CodeURL:    "https://example.morph.so/code/",
		VNCURL:     "https://example.morph.so/vnc/",
		AppURL:     "https://example.morph.so/app/",
		CDPURL:     "https://example.morph.so/cdp/",
		CDPPort:    9222,
		StartedAt:  now,
		SavedSnapshots: []workspace.SavedSnapshot{
			{ID: "snap-1", Name: "checkpoint-1", CreatedAt: now},
		},
	}

	if state.InstanceID != "morph-inst-123" {
		t.Errorf("unexpected InstanceID: %s", state.InstanceID)
	}
	if state.Status != "running" {
		t.Errorf("unexpected Status: %s", state.Status)
	}
	if len(state.SavedSnapshots) != 1 {
		t.Errorf("expected 1 saved snapshot, got %d", len(state.SavedSnapshots))
	}
}

// TestWorkspaceIsMorphRunning tests the IsMorphRunning method
func TestWorkspaceIsMorphRunning(t *testing.T) {
	tests := []struct {
		name       string
		state      workspace.MorphState
		wantResult bool
	}{
		{
			name:       "empty state",
			state:      workspace.MorphState{},
			wantResult: false,
		},
		{
			name: "status running but no instance ID",
			state: workspace.MorphState{
				Status: "running",
			},
			wantResult: false,
		},
		{
			name: "instance ID but not running status",
			state: workspace.MorphState{
				InstanceID: "inst-123",
				Status:     "stopped",
			},
			wantResult: false,
		},
		{
			name: "both running and instance ID",
			state: workspace.MorphState{
				InstanceID: "inst-123",
				Status:     "running",
			},
			wantResult: true,
		},
		{
			name: "paused status",
			state: workspace.MorphState{
				InstanceID: "inst-123",
				Status:     "paused",
			},
			wantResult: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ws := &workspace.Workspace{
				Morph: tt.state,
			}
			got := ws.IsMorphRunning()
			if got != tt.wantResult {
				t.Errorf("IsMorphRunning() = %v, want %v", got, tt.wantResult)
			}
		})
	}
}

// TestWorkspaceSetMorphInstance tests SetMorphInstance
func TestWorkspaceSetMorphInstance(t *testing.T) {
	ws := &workspace.Workspace{}

	ws.SetMorphInstance("inst-123", "snap-abc", "https://example.morph.so")

	if ws.Morph.InstanceID != "inst-123" {
		t.Errorf("unexpected InstanceID: %s", ws.Morph.InstanceID)
	}
	if ws.Morph.SnapshotID != "snap-abc" {
		t.Errorf("unexpected SnapshotID: %s", ws.Morph.SnapshotID)
	}
	if ws.Morph.BaseURL != "https://example.morph.so" {
		t.Errorf("unexpected BaseURL: %s", ws.Morph.BaseURL)
	}
	if ws.Morph.Status != "running" {
		t.Errorf("unexpected Status: %s", ws.Morph.Status)
	}
	if ws.Morph.StartedAt.IsZero() {
		t.Error("StartedAt should be set")
	}

	// Check derived URLs
	if ws.Morph.CodeURL != "https://example.morph.so/code/" {
		t.Errorf("unexpected CodeURL: %s", ws.Morph.CodeURL)
	}
	if ws.Morph.VNCURL != "https://example.morph.so/vnc/vnc.html" {
		t.Errorf("unexpected VNCURL: %s", ws.Morph.VNCURL)
	}
	if ws.Morph.AppURL != "https://example.morph.so/vnc/app/" {
		t.Errorf("unexpected AppURL: %s", ws.Morph.AppURL)
	}
	if ws.Morph.CDPURL != "wss://example.morph.so/cdp/" {
		t.Errorf("unexpected CDPURL: %s", ws.Morph.CDPURL)
	}
}

// TestWorkspaceSetMorphInstanceEmptyBaseURL tests SetMorphInstance with empty base URL
func TestWorkspaceSetMorphInstanceEmptyBaseURL(t *testing.T) {
	ws := &workspace.Workspace{}

	ws.SetMorphInstance("inst-123", "snap-abc", "")

	if ws.Morph.InstanceID != "inst-123" {
		t.Errorf("unexpected InstanceID: %s", ws.Morph.InstanceID)
	}
	// Derived URLs should not be set with empty base URL
	if ws.Morph.CodeURL != "" {
		t.Errorf("CodeURL should be empty, got: %s", ws.Morph.CodeURL)
	}
}

// TestWorkspaceClearMorphInstance tests ClearMorphInstance
func TestWorkspaceClearMorphInstance(t *testing.T) {
	ws := &workspace.Workspace{
		Morph: workspace.MorphState{
			InstanceID: "inst-123",
			Status:     "running",
			BaseURL:    "https://example.morph.so",
		},
	}

	ws.ClearMorphInstance()

	if ws.Morph.InstanceID != "" {
		t.Errorf("InstanceID should be cleared, got: %s", ws.Morph.InstanceID)
	}
	if ws.Morph.Status != "stopped" {
		t.Errorf("Status should be 'stopped', got: %s", ws.Morph.Status)
	}
	// BaseURL should be preserved for reference
	if ws.Morph.BaseURL != "https://example.morph.so" {
		t.Errorf("BaseURL should be preserved, got: %s", ws.Morph.BaseURL)
	}
}

// TestWorkspaceAddSavedSnapshot tests AddSavedSnapshot
func TestWorkspaceAddSavedSnapshot(t *testing.T) {
	ws := &workspace.Workspace{}

	ws.AddSavedSnapshot("snap-1", "checkpoint-1")
	ws.AddSavedSnapshot("snap-2", "checkpoint-2")

	if len(ws.Morph.SavedSnapshots) != 2 {
		t.Errorf("expected 2 saved snapshots, got %d", len(ws.Morph.SavedSnapshots))
	}

	snap1 := ws.Morph.SavedSnapshots[0]
	if snap1.ID != "snap-1" || snap1.Name != "checkpoint-1" {
		t.Errorf("unexpected snapshot 1: %+v", snap1)
	}

	snap2 := ws.Morph.SavedSnapshots[1]
	if snap2.ID != "snap-2" || snap2.Name != "checkpoint-2" {
		t.Errorf("unexpected snapshot 2: %+v", snap2)
	}
}

// TestWorkspaceGetSavedSnapshot tests GetSavedSnapshot
func TestWorkspaceGetSavedSnapshot(t *testing.T) {
	ws := &workspace.Workspace{}
	ws.AddSavedSnapshot("snap-1", "checkpoint-1")
	ws.AddSavedSnapshot("snap-2", "checkpoint-2")

	// Find existing snapshot
	found := ws.GetSavedSnapshot("checkpoint-1")
	if found == nil {
		t.Fatal("should find checkpoint-1")
	}
	if found.ID != "snap-1" {
		t.Errorf("unexpected ID: %s", found.ID)
	}

	// Find non-existing snapshot
	notFound := ws.GetSavedSnapshot("nonexistent")
	if notFound != nil {
		t.Error("should not find nonexistent snapshot")
	}
}

// TestWorkspaceGetMorphURLs tests GetMorphURLs
func TestWorkspaceGetMorphURLs(t *testing.T) {
	ws := &workspace.Workspace{
		Morph: workspace.MorphState{
			CodeURL: "https://example.morph.so/code/",
			VNCURL:  "https://example.morph.so/vnc/",
			AppURL:  "https://example.morph.so/app/",
			CDPURL:  "https://example.morph.so/cdp/",
		},
	}

	urls := ws.GetMorphURLs()

	if urls["code"] != "https://example.morph.so/code/" {
		t.Errorf("unexpected code URL: %s", urls["code"])
	}
	if urls["vnc"] != "https://example.morph.so/vnc/" {
		t.Errorf("unexpected vnc URL: %s", urls["vnc"])
	}
	if urls["app"] != "https://example.morph.so/app/" {
		t.Errorf("unexpected app URL: %s", urls["app"])
	}
	if urls["cdp"] != "https://example.morph.so/cdp/" {
		t.Errorf("unexpected cdp URL: %s", urls["cdp"])
	}
}

// TestWorkspaceGetMorphURLsEmpty tests GetMorphURLs with no URLs set
func TestWorkspaceGetMorphURLsEmpty(t *testing.T) {
	ws := &workspace.Workspace{}

	urls := ws.GetMorphURLs()

	if len(urls) != 0 {
		t.Errorf("expected 0 URLs, got %d", len(urls))
	}
}

// TestWorkspaceGetMorphURLsPartial tests GetMorphURLs with partial URLs
func TestWorkspaceGetMorphURLsPartial(t *testing.T) {
	ws := &workspace.Workspace{
		Morph: workspace.MorphState{
			CodeURL: "https://example.morph.so/code/",
			// Other URLs not set
		},
	}

	urls := ws.GetMorphURLs()

	if len(urls) != 1 {
		t.Errorf("expected 1 URL, got %d", len(urls))
	}
	if urls["code"] != "https://example.morph.so/code/" {
		t.Errorf("unexpected code URL: %s", urls["code"])
	}
}

// TestSavedSnapshotStruct tests SavedSnapshot struct
func TestSavedSnapshotStruct(t *testing.T) {
	now := time.Now()
	snap := workspace.SavedSnapshot{
		ID:        "snap-123",
		Name:      "my-checkpoint",
		CreatedAt: now,
	}

	if snap.ID != "snap-123" {
		t.Errorf("unexpected ID: %s", snap.ID)
	}
	if snap.Name != "my-checkpoint" {
		t.Errorf("unexpected Name: %s", snap.Name)
	}
	if !snap.CreatedAt.Equal(now) {
		t.Error("unexpected CreatedAt")
	}
}

// TestMorphStateMultipleSnapshots tests adding many snapshots
func TestMorphStateMultipleSnapshots(t *testing.T) {
	ws := &workspace.Workspace{}

	// Add many snapshots
	for i := 0; i < 100; i++ {
		ws.AddSavedSnapshot(
			"snap-"+string(rune('a'+i%26)),
			"checkpoint-"+string(rune('a'+i%26)),
		)
	}

	if len(ws.Morph.SavedSnapshots) != 100 {
		t.Errorf("expected 100 snapshots, got %d", len(ws.Morph.SavedSnapshots))
	}
}

// TestMorphStateUnicodeNames tests unicode in snapshot names
func TestMorphStateUnicodeNames(t *testing.T) {
	ws := &workspace.Workspace{}

	ws.AddSavedSnapshot("snap-1", "检查点-1")
	ws.AddSavedSnapshot("snap-2", "チェックポイント")
	ws.AddSavedSnapshot("snap-3", "контрольная-точка")

	found := ws.GetSavedSnapshot("検査点-1") // Not found - different characters
	if found != nil {
		t.Error("should not find different unicode string")
	}

	found = ws.GetSavedSnapshot("检查点-1") // Found - exact match
	if found == nil {
		t.Error("should find exact unicode match")
	}
}

// TestMorphStateSpecialCharNames tests special characters in snapshot names
func TestMorphStateSpecialCharNames(t *testing.T) {
	ws := &workspace.Workspace{}

	specialNames := []string{
		"snapshot with spaces",
		"snapshot-with-dashes",
		"snapshot_with_underscores",
		"snapshot.with.dots",
		"snapshot/with/slashes",
		"snapshot@with@at",
		"snapshot#with#hash",
	}

	for i, name := range specialNames {
		ws.AddSavedSnapshot("snap-"+string(rune('a'+i)), name)
	}

	for _, name := range specialNames {
		found := ws.GetSavedSnapshot(name)
		if found == nil {
			t.Errorf("should find snapshot with name: %s", name)
		}
	}
}
