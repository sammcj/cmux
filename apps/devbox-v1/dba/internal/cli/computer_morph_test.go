// internal/cli/computer_morph_test.go
package cli

import (
	"testing"
	"time"

	"github.com/dba-cli/dba/internal/morph"
)

// TestMorphManagerConfigDefaults tests default manager configuration
func TestMorphManagerConfigDefaults(t *testing.T) {
	cfg := morph.ManagerConfig{}

	if cfg.APIKey != "" {
		t.Errorf("default APIKey should be empty, got %s", cfg.APIKey)
	}
	if cfg.BaseSnapshotID != "" {
		t.Errorf("default BaseSnapshotID should be empty, got %s", cfg.BaseSnapshotID)
	}
	if cfg.DefaultTTL != 0 {
		t.Errorf("default DefaultTTL should be 0, got %d", cfg.DefaultTTL)
	}
}

// TestMorphManagerConfigWithValues tests manager configuration with values
func TestMorphManagerConfigWithValues(t *testing.T) {
	cfg := morph.ManagerConfig{
		APIKey:         "morph_test123",
		BaseSnapshotID: "snap_abc123",
		DefaultTTL:     3600,
	}

	if cfg.APIKey != "morph_test123" {
		t.Errorf("APIKey mismatch")
	}
	if cfg.BaseSnapshotID != "snap_abc123" {
		t.Errorf("BaseSnapshotID mismatch")
	}
	if cfg.DefaultTTL != 3600 {
		t.Errorf("DefaultTTL mismatch: expected 3600, got %d", cfg.DefaultTTL)
	}
}

// TestMorphInstanceStruct tests Instance struct
func TestMorphInstanceStruct(t *testing.T) {
	now := time.Now()
	inst := morph.Instance{
		ID:         "inst_123",
		SnapshotID: "snap_abc",
		Status:     morph.StatusRunning,
		BaseURL:    "https://example.morph.so",
		CreatedAt:  now,
	}

	if inst.ID != "inst_123" {
		t.Errorf("ID mismatch")
	}
	if inst.SnapshotID != "snap_abc" {
		t.Errorf("SnapshotID mismatch")
	}
	if inst.Status != morph.StatusRunning {
		t.Errorf("Status mismatch")
	}
	if inst.BaseURL != "https://example.morph.so" {
		t.Errorf("BaseURL mismatch")
	}
}

// TestMorphStatusValues tests status string values
func TestMorphStatusValues(t *testing.T) {
	tests := []struct {
		status morph.InstanceStatus
		expect string
	}{
		{morph.StatusPending, "pending"},
		{morph.StatusStarting, "starting"},
		{morph.StatusRunning, "running"},
		{morph.StatusStopping, "stopping"},
		{morph.StatusStopped, "stopped"},
		{morph.StatusError, "error"},
	}

	for _, tt := range tests {
		t.Run(tt.expect, func(t *testing.T) {
			if string(tt.status) != tt.expect {
				t.Errorf("Status mismatch: %s != %s", string(tt.status), tt.expect)
			}
		})
	}
}

// TestMorphStatusIsRunning tests running status check
func TestMorphStatusIsRunning(t *testing.T) {
	running := []morph.InstanceStatus{morph.StatusRunning}
	notRunning := []morph.InstanceStatus{
		morph.StatusPending,
		morph.StatusStarting,
		morph.StatusStopping,
		morph.StatusStopped,
		morph.StatusError,
	}

	for _, s := range running {
		if s != morph.StatusRunning {
			t.Errorf("Status %s should be running", s)
		}
	}

	for _, s := range notRunning {
		if s == morph.StatusRunning {
			t.Errorf("Status %s should not be running", s)
		}
	}
}

// TestMorphSnapshotStruct tests Snapshot struct
func TestMorphSnapshotStruct(t *testing.T) {
	now := time.Now()
	snap := morph.Snapshot{
		ID:        "snap_123",
		Digest:    "my-checkpoint", // Digest is the human-readable name
		CreatedAt: now,
	}

	if snap.ID != "snap_123" {
		t.Errorf("ID mismatch")
	}
	if snap.Digest != "my-checkpoint" {
		t.Errorf("Digest mismatch")
	}
	if !snap.CreatedAt.Equal(now) {
		t.Errorf("CreatedAt mismatch")
	}
}

// TestMorphErrorTypes tests error types
func TestMorphErrorTypes(t *testing.T) {
	tests := []struct {
		name string
		err  error
	}{
		{"API key missing", morph.ErrAPIKeyMissing},
		{"not found", morph.ErrNotFound},
		{"not running", morph.ErrNotRunning},
		{"already running", morph.ErrAlreadyRunning},
		{"timeout", morph.ErrTimeout},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.err == nil {
				t.Error("error should not be nil")
			}
			if tt.err.Error() == "" {
				t.Error("error message should not be empty")
			}
		})
	}
}

// TestMorphManagerConfigAPIKeyFormats tests various API key formats
func TestMorphManagerConfigAPIKeyFormats(t *testing.T) {
	keys := []string{
		"morph_test",
		"morph_abc123xyz",
		"morph_longkeyvalue12345678901234567890",
	}

	for _, key := range keys {
		t.Run(key[:10], func(t *testing.T) {
			cfg := morph.ManagerConfig{
				APIKey: key,
			}
			if cfg.APIKey != key {
				t.Errorf("APIKey mismatch")
			}
		})
	}
}

// TestMorphManagerConfigSnapshotIDFormats tests snapshot ID formats
func TestMorphManagerConfigSnapshotIDFormats(t *testing.T) {
	ids := []string{
		"snap_a",
		"snap_abc123",
		"snapshot_123456789",
		"snap_verylongidvalue123456789",
	}

	for _, id := range ids {
		name := id
		if len(name) > 10 {
			name = name[:10]
		}
		t.Run(name, func(t *testing.T) {
			cfg := morph.ManagerConfig{
				BaseSnapshotID: id,
			}
			if cfg.BaseSnapshotID != id {
				t.Errorf("BaseSnapshotID mismatch")
			}
		})
	}
}

// TestMorphManagerConfigTTLValues tests TTL value boundaries
func TestMorphManagerConfigTTLValues(t *testing.T) {
	tests := []struct {
		name string
		ttl  int
	}{
		{"zero", 0},
		{"one minute", 60},
		{"one hour", 3600},
		{"one day", 86400},
		{"one week", 604800},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := morph.ManagerConfig{
				DefaultTTL: tt.ttl,
			}
			if cfg.DefaultTTL != tt.ttl {
				t.Errorf("DefaultTTL mismatch: %d != %d", cfg.DefaultTTL, tt.ttl)
			}
		})
	}
}

// TestMorphInstanceStatusTransitions tests valid status transitions
func TestMorphInstanceStatusTransitions(t *testing.T) {
	transitions := []struct {
		from  morph.InstanceStatus
		to    morph.InstanceStatus
		valid bool
	}{
		// Valid transitions
		{morph.StatusPending, morph.StatusStarting, true},
		{morph.StatusStarting, morph.StatusRunning, true},
		{morph.StatusRunning, morph.StatusStopping, true},
		{morph.StatusStopping, morph.StatusStopped, true},
		{morph.StatusStarting, morph.StatusError, true},
		{morph.StatusRunning, morph.StatusError, true},
	}

	for _, tt := range transitions {
		name := string(tt.from) + "->" + string(tt.to)
		t.Run(name, func(t *testing.T) {
			// Just verify we can represent these transitions
			inst := morph.Instance{Status: tt.from}
			inst.Status = tt.to
			if inst.Status != tt.to {
				t.Errorf("Status should be %s", tt.to)
			}
		})
	}
}

// TestMorphInstanceURLFormats tests various URL formats
func TestMorphInstanceURLFormats(t *testing.T) {
	urls := []string{
		"https://example.morph.so",
		"https://abc123.morph.so",
		"https://instance-123.region.morph.so",
		"http://localhost:8080",
		"http://192.168.1.100:3000",
	}

	for _, url := range urls {
		t.Run(url, func(t *testing.T) {
			inst := morph.Instance{
				ID:      "inst_123",
				BaseURL: url,
			}
			if inst.BaseURL != url {
				t.Errorf("BaseURL mismatch")
			}
		})
	}
}

// TestMorphSnapshotDigestFormats tests snapshot digest (name) formats
func TestMorphSnapshotDigestFormats(t *testing.T) {
	digests := []string{
		"",
		"checkpoint",
		"my-checkpoint-1",
		"checkpoint_with_underscore",
		"Checkpoint With Spaces",
		"checkpoint.with.dots",
		"检查点",
		"チェックポイント",
		"emoji-rocket-checkpoint",
	}

	for _, digest := range digests {
		t.Run(digest, func(t *testing.T) {
			snap := morph.Snapshot{
				ID:     "snap_123",
				Digest: digest,
			}
			if snap.Digest != digest {
				t.Errorf("Digest mismatch")
			}
		})
	}
}

// TestMorphInstanceTimestamps tests timestamp handling
func TestMorphInstanceTimestamps(t *testing.T) {
	tests := []struct {
		name string
		time time.Time
	}{
		{"zero time", time.Time{}},
		{"now", time.Now()},
		{"past", time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)},
		{"future", time.Date(2030, 12, 31, 23, 59, 59, 0, time.UTC)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			inst := morph.Instance{
				ID:        "inst_123",
				CreatedAt: tt.time,
			}
			if !inst.CreatedAt.Equal(tt.time) {
				t.Errorf("CreatedAt mismatch")
			}
		})
	}
}

// TestMorphAPIError tests APIError struct
func TestMorphAPIError(t *testing.T) {
	err := &morph.APIError{
		Code:    "INSTANCE_NOT_FOUND",
		Message: "Instance not found",
		Details: "Instance inst_123 does not exist",
	}

	errStr := err.Error()
	if errStr == "" {
		t.Error("Error message should not be empty")
	}
	if err.Code != "INSTANCE_NOT_FOUND" {
		t.Errorf("Code mismatch")
	}
}

// TestMorphExecResult tests ExecResult struct
func TestMorphExecResult(t *testing.T) {
	result := morph.ExecResult{
		Stdout:   "hello world",
		Stderr:   "",
		ExitCode: 0,
	}

	if result.Stdout != "hello world" {
		t.Errorf("Stdout mismatch")
	}
	if result.ExitCode != 0 {
		t.Errorf("ExitCode mismatch")
	}
}

// TestMorphExecResultWithError tests ExecResult with error
func TestMorphExecResultWithError(t *testing.T) {
	result := morph.ExecResult{
		Stdout:   "",
		Stderr:   "command not found",
		ExitCode: 127,
	}

	if result.Stderr != "command not found" {
		t.Errorf("Stderr mismatch")
	}
	if result.ExitCode != 127 {
		t.Errorf("ExitCode mismatch: expected 127, got %d", result.ExitCode)
	}
}

// TestMorphInstanceAllURLs tests all URL fields
func TestMorphInstanceAllURLs(t *testing.T) {
	inst := morph.Instance{
		ID:      "inst_123",
		BaseURL: "https://example.morph.so",
		CDPURL:  "wss://example.morph.so/cdp/",
		VNCURL:  "https://example.morph.so/vnc/",
		CodeURL: "https://example.morph.so/code/",
		AppURL:  "https://example.morph.so/app/",
	}

	if inst.BaseURL == "" {
		t.Error("BaseURL should be set")
	}
	if inst.CDPURL == "" {
		t.Error("CDPURL should be set")
	}
	if inst.VNCURL == "" {
		t.Error("VNCURL should be set")
	}
	if inst.CodeURL == "" {
		t.Error("CodeURL should be set")
	}
	if inst.AppURL == "" {
		t.Error("AppURL should be set")
	}
}

// TestMorphInstanceMetadata tests metadata field
func TestMorphInstanceMetadata(t *testing.T) {
	inst := morph.Instance{
		ID: "inst_123",
		Metadata: map[string]string{
			"workspace": "ws_abc123",
			"owner":     "user@example.com",
			"created":   "2024-01-01",
		},
	}

	if inst.Metadata["workspace"] != "ws_abc123" {
		t.Errorf("workspace metadata mismatch")
	}
	if inst.Metadata["owner"] != "user@example.com" {
		t.Errorf("owner metadata mismatch")
	}
}

// TestMorphSnapshotFields tests all Snapshot fields
func TestMorphSnapshotFields(t *testing.T) {
	snap := morph.Snapshot{
		ID:       "snap_123",
		Digest:   "my-snapshot",
		ImageID:  "img_456",
		VCPUs:    4,
		Memory:   8192,
		DiskSize: 51200,
		Metadata: map[string]string{
			"name": "checkpoint-1",
		},
	}

	if snap.VCPUs != 4 {
		t.Errorf("VCPUs mismatch")
	}
	if snap.Memory != 8192 {
		t.Errorf("Memory mismatch")
	}
	if snap.DiskSize != 51200 {
		t.Errorf("DiskSize mismatch")
	}
}

// TestMorphWrapError tests WrapError function
func TestMorphWrapError(t *testing.T) {
	// Wrap nil error
	wrapped := morph.WrapError(nil, "context")
	if wrapped != nil {
		t.Error("WrapError(nil) should return nil")
	}

	// Wrap actual error
	err := morph.ErrNotRunning
	wrapped = morph.WrapError(err, "failed to stop")
	if wrapped == nil {
		t.Error("WrapError should not return nil for non-nil error")
	}
}
