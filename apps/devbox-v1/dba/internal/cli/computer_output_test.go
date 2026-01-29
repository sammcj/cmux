// internal/cli/computer_output_test.go
package cli

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/spf13/cobra"

	"github.com/dba-cli/dba/internal/workspace"
)

// TestOutputJSONMorphState tests JSON output of MorphState
func TestOutputJSONMorphState(t *testing.T) {
	state := workspace.MorphState{
		InstanceID: "inst-123",
		SnapshotID: "snap-abc",
		Status:     "running",
		BaseURL:    "https://example.morph.so",
		CodeURL:    "https://example.morph.so/code/",
		VNCURL:     "https://example.morph.so/vnc/",
		AppURL:     "https://example.morph.so/app/",
		CDPURL:     "https://example.morph.so/cdp/",
		CDPPort:    9222,
		StartedAt:  time.Now(),
	}

	// Marshal to JSON
	data, err := json.Marshal(state)
	if err != nil {
		t.Fatalf("failed to marshal MorphState: %v", err)
	}

	// Verify JSON contains expected fields
	jsonStr := string(data)
	if !strings.Contains(jsonStr, "instance_id") {
		t.Error("JSON should contain instance_id field")
	}
	if !strings.Contains(jsonStr, "inst-123") {
		t.Error("JSON should contain instance ID value")
	}
	if !strings.Contains(jsonStr, "status") {
		t.Error("JSON should contain status field")
	}
	if !strings.Contains(jsonStr, "running") {
		t.Error("JSON should contain running status value")
	}
}

// TestOutputJSONMorphStateEmpty tests JSON output of empty MorphState
func TestOutputJSONMorphStateEmpty(t *testing.T) {
	state := workspace.MorphState{}

	data, err := json.Marshal(state)
	if err != nil {
		t.Fatalf("failed to marshal empty MorphState: %v", err)
	}

	// Empty state should still be valid JSON
	var parsed workspace.MorphState
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Errorf("failed to unmarshal empty MorphState JSON: %v", err)
	}
}

// TestOutputJSONSavedSnapshot tests JSON output of SavedSnapshot
func TestOutputJSONSavedSnapshot(t *testing.T) {
	snap := workspace.SavedSnapshot{
		ID:        "snap-123",
		Name:      "my-checkpoint",
		CreatedAt: time.Now(),
	}

	data, err := json.Marshal(snap)
	if err != nil {
		t.Fatalf("failed to marshal SavedSnapshot: %v", err)
	}

	jsonStr := string(data)
	if !strings.Contains(jsonStr, "snap-123") {
		t.Error("JSON should contain snapshot ID")
	}
	if !strings.Contains(jsonStr, "my-checkpoint") {
		t.Error("JSON should contain snapshot name")
	}
}

// TestOutputJSONMorphStateWithSnapshots tests JSON output with saved snapshots
func TestOutputJSONMorphStateWithSnapshots(t *testing.T) {
	now := time.Now()
	state := workspace.MorphState{
		InstanceID: "inst-123",
		Status:     "running",
		SavedSnapshots: []workspace.SavedSnapshot{
			{ID: "snap-1", Name: "checkpoint-1", CreatedAt: now},
			{ID: "snap-2", Name: "checkpoint-2", CreatedAt: now},
		},
	}

	data, err := json.Marshal(state)
	if err != nil {
		t.Fatalf("failed to marshal MorphState with snapshots: %v", err)
	}

	jsonStr := string(data)
	if !strings.Contains(jsonStr, "saved_snapshots") {
		t.Error("JSON should contain saved_snapshots field")
	}
	if !strings.Contains(jsonStr, "checkpoint-1") {
		t.Error("JSON should contain first checkpoint")
	}
	if !strings.Contains(jsonStr, "checkpoint-2") {
		t.Error("JSON should contain second checkpoint")
	}
}

// TestOutputJSONRoundTrip tests JSON marshal/unmarshal round trip
func TestOutputJSONRoundTrip(t *testing.T) {
	original := workspace.MorphState{
		InstanceID: "inst-xyz",
		SnapshotID: "snap-abc",
		Status:     "running",
		BaseURL:    "https://test.morph.so",
		CodeURL:    "https://test.morph.so/code/",
		VNCURL:     "https://test.morph.so/vnc/",
		AppURL:     "https://test.morph.so/app/",
		CDPURL:     "https://test.morph.so/cdp/",
		CDPPort:    9222,
		StartedAt:  time.Now().Truncate(time.Second),
		SavedSnapshots: []workspace.SavedSnapshot{
			{ID: "snap-1", Name: "test", CreatedAt: time.Now().Truncate(time.Second)},
		},
	}

	// Marshal
	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	// Unmarshal
	var parsed workspace.MorphState
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	// Verify fields
	if parsed.InstanceID != original.InstanceID {
		t.Errorf("InstanceID mismatch: %s != %s", parsed.InstanceID, original.InstanceID)
	}
	if parsed.Status != original.Status {
		t.Errorf("Status mismatch: %s != %s", parsed.Status, original.Status)
	}
	if parsed.BaseURL != original.BaseURL {
		t.Errorf("BaseURL mismatch: %s != %s", parsed.BaseURL, original.BaseURL)
	}
	if parsed.CDPPort != original.CDPPort {
		t.Errorf("CDPPort mismatch: %d != %d", parsed.CDPPort, original.CDPPort)
	}
	if len(parsed.SavedSnapshots) != len(original.SavedSnapshots) {
		t.Errorf("SavedSnapshots count mismatch: %d != %d",
			len(parsed.SavedSnapshots), len(original.SavedSnapshots))
	}
}

// TestComputerStatusJSONFlag tests that status command has --json flag
func TestComputerStatusJSONFlag(t *testing.T) {
	root := GetRootCmd()

	var statusCmd *cobra.Command
	for _, cmd := range root.Commands() {
		if cmd.Use == "computer" {
			for _, sub := range cmd.Commands() {
				if sub.Name() == "status" {
					statusCmd = sub
					break
				}
			}
			break
		}
	}

	if statusCmd == nil {
		t.Fatal("status command not found")
	}

	// Check for --json flag
	jsonFlag := statusCmd.Flag("json")
	if jsonFlag == nil {
		t.Error("status command should have --json flag")
	}

	// Verify flag default is false
	if jsonFlag != nil && jsonFlag.DefValue != "false" {
		t.Errorf("--json flag default should be false, got %s", jsonFlag.DefValue)
	}
}

// TestComputerStatusHelpShowsJSONFlag tests help output mentions JSON flag
func TestComputerStatusHelpShowsJSONFlag(t *testing.T) {
	root := GetRootCmd()
	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs([]string{"computer", "status", "--help"})

	err := root.Execute()
	if err != nil {
		t.Errorf("help should not return error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "json") {
		t.Error("status help should mention json flag")
	}
}

// TestOutputJSONPrettyPrint tests JSON can be pretty printed
func TestOutputJSONPrettyPrint(t *testing.T) {
	state := workspace.MorphState{
		InstanceID: "inst-123",
		Status:     "running",
	}

	// Pretty print with indent
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		t.Fatalf("failed to marshal with indent: %v", err)
	}

	jsonStr := string(data)
	// Pretty printed JSON should have newlines
	if !strings.Contains(jsonStr, "\n") {
		t.Error("pretty printed JSON should have newlines")
	}
}

// TestOutputTextVsJSON tests that text and JSON outputs differ
func TestOutputTextVsJSON(t *testing.T) {
	ws := &workspace.Workspace{
		ID:   "ws_test",
		Name: "test-workspace",
		Morph: workspace.MorphState{
			InstanceID: "inst-123",
			Status:     "running",
			CodeURL:    "https://example.morph.so/code/",
		},
	}

	// Get text output
	textOutput := ws.TextOutput()

	// Get JSON output
	jsonData, _ := json.Marshal(ws)
	jsonOutput := string(jsonData)

	// Text output should be human readable
	if !strings.Contains(textOutput, "Workspace:") {
		t.Error("text output should contain 'Workspace:' label")
	}

	// JSON output should be structured
	if !strings.HasPrefix(jsonOutput, "{") {
		t.Error("JSON output should start with '{'")
	}

	// They should be different
	if textOutput == jsonOutput {
		t.Error("text and JSON outputs should differ")
	}
}

// TestOutputJSONErrorFormat tests JSON error format
func TestOutputJSONErrorFormat(t *testing.T) {
	// Test that errors can be formatted as JSON
	errMsg := "VM is not running"

	type ErrorResponse struct {
		Error struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}

	resp := ErrorResponse{}
	resp.Error.Code = "VM_NOT_RUNNING"
	resp.Error.Message = errMsg

	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("failed to marshal error response: %v", err)
	}

	jsonStr := string(data)
	if !strings.Contains(jsonStr, "error") {
		t.Error("error JSON should contain 'error' field")
	}
	if !strings.Contains(jsonStr, "VM_NOT_RUNNING") {
		t.Error("error JSON should contain error code")
	}
	if !strings.Contains(jsonStr, errMsg) {
		t.Error("error JSON should contain error message")
	}
}

// TestOutputJSONSpecialCharacters tests JSON with special characters
func TestOutputJSONSpecialCharacters(t *testing.T) {
	state := workspace.MorphState{
		InstanceID: "inst-with-\"quotes\"",
		BaseURL:    "https://example.com/path?query=value&other=test",
	}

	data, err := json.Marshal(state)
	if err != nil {
		t.Fatalf("failed to marshal state with special chars: %v", err)
	}

	// Verify it's valid JSON
	var parsed workspace.MorphState
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Errorf("should be able to parse JSON with special chars: %v", err)
	}

	// Quotes should be escaped
	jsonStr := string(data)
	if strings.Contains(jsonStr, "\"quotes\"") && !strings.Contains(jsonStr, "\\\"quotes\\\"") {
		t.Error("quotes should be escaped in JSON")
	}
}

// TestOutputJSONUnicode tests JSON with unicode characters
func TestOutputJSONUnicode(t *testing.T) {
	ws := &workspace.Workspace{
		Name: "日本語ワークスペース",
		Morph: workspace.MorphState{
			SavedSnapshots: []workspace.SavedSnapshot{
				{ID: "snap-1", Name: "检查点-1"},
				{ID: "snap-2", Name: "контрольная-точка"},
			},
		},
	}

	data, err := json.Marshal(ws)
	if err != nil {
		t.Fatalf("failed to marshal workspace with unicode: %v", err)
	}

	// Unmarshal and verify unicode is preserved
	var parsed workspace.Workspace
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if parsed.Name != ws.Name {
		t.Errorf("unicode name not preserved: %s != %s", parsed.Name, ws.Name)
	}
	if len(parsed.Morph.SavedSnapshots) != 2 {
		t.Fatal("snapshots not preserved")
	}
	if parsed.Morph.SavedSnapshots[0].Name != "检查点-1" {
		t.Errorf("chinese unicode not preserved: %s", parsed.Morph.SavedSnapshots[0].Name)
	}
	if parsed.Morph.SavedSnapshots[1].Name != "контрольная-точка" {
		t.Errorf("russian unicode not preserved: %s", parsed.Morph.SavedSnapshots[1].Name)
	}
}

// TestOutputTimestampFormat tests timestamp format in JSON output
func TestOutputTimestampFormat(t *testing.T) {
	now := time.Now()
	state := workspace.MorphState{
		StartedAt: now,
	}

	data, err := json.Marshal(state)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	jsonStr := string(data)
	// Time should be in RFC3339 format
	if !strings.Contains(jsonStr, "T") {
		t.Error("timestamp should contain 'T' separator (RFC3339 format)")
	}
}

// TestOutputEmptyFields tests JSON output with empty/zero fields
func TestOutputEmptyFields(t *testing.T) {
	state := workspace.MorphState{
		InstanceID: "inst-123",
		// Other fields are zero/empty
	}

	data, err := json.Marshal(state)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	// With omitempty, empty fields should be omitted
	jsonStr := string(data)
	// InstanceID should be present
	if !strings.Contains(jsonStr, "instance_id") {
		t.Error("instance_id should be present")
	}
	// Empty SnapshotID should be omitted due to omitempty
	// (depends on struct tags)
}
