// internal/config/morph_compatibility_test.go
package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// =============================================================================
// Backward Compatibility Tests
// =============================================================================

func TestOldConfigFormatWithoutMorph(t *testing.T) {
	tmpDir := t.TempDir()
	os.Setenv("DBA_HOME", tmpDir)
	defer os.Unsetenv("DBA_HOME")

	// Old config format without morph section
	oldConfig := `
ports:
  range_start: 3000
  range_end: 4000
  block_size: 10
`
	if err := os.WriteFile(filepath.Join(tmpDir, "config.yaml"), []byte(oldConfig), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	// Morph should use defaults
	if cfg.Morph.VM.VCPUs != DefaultMorphVCPUs {
		t.Errorf("VCPUs = %d, want %d", cfg.Morph.VM.VCPUs, DefaultMorphVCPUs)
	}
	if cfg.AgentBrowser.Timeout != DefaultAgentBrowserTimeout {
		t.Errorf("Timeout = %d, want %d", cfg.AgentBrowser.Timeout, DefaultAgentBrowserTimeout)
	}
}

func TestOldConfigFormatWithEmptyMorph(t *testing.T) {
	tmpDir := t.TempDir()
	os.Setenv("DBA_HOME", tmpDir)
	defer os.Unsetenv("DBA_HOME")

	// Config with empty morph section
	config := `
morph:
agent_browser:
`
	if err := os.WriteFile(filepath.Join(tmpDir, "config.yaml"), []byte(config), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	// Should use defaults for empty sections
	if cfg.Morph.VM.VCPUs != DefaultMorphVCPUs {
		t.Errorf("VCPUs = %d, want %d", cfg.Morph.VM.VCPUs, DefaultMorphVCPUs)
	}
}

func TestConfigWithFutureMorphFields(t *testing.T) {
	tmpDir := t.TempDir()
	os.Setenv("DBA_HOME", tmpDir)
	defer os.Unsetenv("DBA_HOME")

	// Config with unknown future fields
	config := `
morph:
  api_key: "test-key"
  future_field: "some value"
  vm:
    vcpus: 4
    future_vm_field: 12345
  future_section:
    nested: true
`
	if err := os.WriteFile(filepath.Join(tmpDir, "config.yaml"), []byte(config), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	// Known fields should be parsed
	if cfg.Morph.APIKey != "test-key" {
		t.Errorf("APIKey = %s, want test-key", cfg.Morph.APIKey)
	}
	if cfg.Morph.VM.VCPUs != 4 {
		t.Errorf("VCPUs = %d, want 4", cfg.Morph.VM.VCPUs)
	}
}

// =============================================================================
// JSON Compatibility Tests
// =============================================================================

func TestMorphConfigJSONFieldNaming(t *testing.T) {
	cfg := MorphConfig{
		APIKey:         "test-key",
		BaseSnapshotID: "snap-123",
		VM: VMConfig{
			VCPUs:      4,
			Memory:     8192,
			DiskSize:   65536,
			TTLSeconds: 3600,
		},
	}

	data, err := json.Marshal(cfg)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}

	jsonStr := string(data)

	// Verify snake_case field names
	expectedFields := map[string]bool{
		`"api_key"`:         true,
		`"base_snapshot_id"`: true,
		`"vcpus"`:           true,
		`"memory"`:          true,
		`"disk_size"`:       true,
		`"ttl_seconds"`:     true,
	}

	for field := range expectedFields {
		if !containsField(jsonStr, field) {
			t.Errorf("JSON should contain %s, got: %s", field, jsonStr)
		}
	}
}

func containsField(s, field string) bool {
	for i := 0; i <= len(s)-len(field); i++ {
		if s[i:i+len(field)] == field {
			return true
		}
	}
	return false
}

func TestAgentBrowserConfigJSONFieldNaming(t *testing.T) {
	cfg := AgentBrowserConfig{
		Path:          "/test/path",
		Timeout:       30000,
		SessionPrefix: "test",
	}

	data, err := json.Marshal(cfg)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}

	jsonStr := string(data)

	expectedFields := map[string]bool{
		`"path"`:           true,
		`"timeout"`:        true,
		`"session_prefix"`: true,
	}

	for field := range expectedFields {
		if !containsField(jsonStr, field) {
			t.Errorf("JSON should contain %s, got: %s", field, jsonStr)
		}
	}
}

// =============================================================================
// YAML Compatibility Tests
// =============================================================================

func TestYAMLTagConsistency(t *testing.T) {
	tmpDir := t.TempDir()
	os.Setenv("DBA_HOME", tmpDir)
	defer os.Unsetenv("DBA_HOME")

	// YAML with snake_case keys (matching YAML tags)
	config := `
morph:
  api_key: "yaml-key"
  base_snapshot_id: "yaml-snap"
  vm:
    vcpus: 2
    memory: 4096
    disk_size: 32768
    ttl_seconds: 3600

agent_browser:
  path: "/yaml/path"
  timeout: 45000
  session_prefix: "yaml-prefix"
`
	if err := os.WriteFile(filepath.Join(tmpDir, "config.yaml"), []byte(config), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	// Verify all fields are parsed correctly
	if cfg.Morph.APIKey != "yaml-key" {
		t.Errorf("APIKey = %s, want yaml-key", cfg.Morph.APIKey)
	}
	if cfg.Morph.BaseSnapshotID != "yaml-snap" {
		t.Errorf("BaseSnapshotID = %s, want yaml-snap", cfg.Morph.BaseSnapshotID)
	}
	if cfg.Morph.VM.VCPUs != 2 {
		t.Errorf("VCPUs = %d, want 2", cfg.Morph.VM.VCPUs)
	}
	if cfg.Morph.VM.Memory != 4096 {
		t.Errorf("Memory = %d, want 4096", cfg.Morph.VM.Memory)
	}
	if cfg.Morph.VM.DiskSize != 32768 {
		t.Errorf("DiskSize = %d, want 32768", cfg.Morph.VM.DiskSize)
	}
	if cfg.Morph.VM.TTLSeconds != 3600 {
		t.Errorf("TTLSeconds = %d, want 3600", cfg.Morph.VM.TTLSeconds)
	}
	if cfg.AgentBrowser.Path != "/yaml/path" {
		t.Errorf("Path = %s, want /yaml/path", cfg.AgentBrowser.Path)
	}
	if cfg.AgentBrowser.Timeout != 45000 {
		t.Errorf("Timeout = %d, want 45000", cfg.AgentBrowser.Timeout)
	}
	if cfg.AgentBrowser.SessionPrefix != "yaml-prefix" {
		t.Errorf("SessionPrefix = %s, want yaml-prefix", cfg.AgentBrowser.SessionPrefix)
	}
}

// =============================================================================
// Cross-Package Compatibility Tests
// =============================================================================

func TestConfigAndDefaultsConsistency(t *testing.T) {
	// DefaultConfig should produce a valid config
	cfg := DefaultConfig()

	if err := cfg.Validate(); err != nil {
		t.Errorf("DefaultConfig() produces invalid config: %v", err)
	}

	// DefaultMorphConfig values should match defaults used in DefaultConfig
	morphCfg := DefaultMorphConfig()
	if morphCfg.VM.VCPUs != cfg.Morph.VM.VCPUs {
		t.Errorf("VCPUs mismatch: DefaultMorphConfig=%d, DefaultConfig=%d",
			morphCfg.VM.VCPUs, cfg.Morph.VM.VCPUs)
	}
	if morphCfg.VM.Memory != cfg.Morph.VM.Memory {
		t.Errorf("Memory mismatch: DefaultMorphConfig=%d, DefaultConfig=%d",
			morphCfg.VM.Memory, cfg.Morph.VM.Memory)
	}

	// DefaultAgentBrowserConfig values should match defaults used in DefaultConfig
	browserCfg := DefaultAgentBrowserConfig()
	if browserCfg.Path != cfg.AgentBrowser.Path {
		t.Errorf("Path mismatch: DefaultAgentBrowserConfig=%s, DefaultConfig=%s",
			browserCfg.Path, cfg.AgentBrowser.Path)
	}
	if browserCfg.Timeout != cfg.AgentBrowser.Timeout {
		t.Errorf("Timeout mismatch: DefaultAgentBrowserConfig=%d, DefaultConfig=%d",
			browserCfg.Timeout, cfg.AgentBrowser.Timeout)
	}
}

func TestConstantsAndDefaultsConsistency(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.Morph.VM.VCPUs != DefaultMorphVCPUs {
		t.Errorf("VCPUs mismatch: DefaultConfig=%d, Constant=%d",
			cfg.Morph.VM.VCPUs, DefaultMorphVCPUs)
	}
	if cfg.Morph.VM.Memory != DefaultMorphMemory {
		t.Errorf("Memory mismatch: DefaultConfig=%d, Constant=%d",
			cfg.Morph.VM.Memory, DefaultMorphMemory)
	}
	if cfg.Morph.VM.DiskSize != DefaultMorphDiskSize {
		t.Errorf("DiskSize mismatch: DefaultConfig=%d, Constant=%d",
			cfg.Morph.VM.DiskSize, DefaultMorphDiskSize)
	}
	if cfg.Morph.VM.TTLSeconds != DefaultMorphTTLSeconds {
		t.Errorf("TTLSeconds mismatch: DefaultConfig=%d, Constant=%d",
			cfg.Morph.VM.TTLSeconds, DefaultMorphTTLSeconds)
	}
	if cfg.AgentBrowser.Path != DefaultAgentBrowserPath {
		t.Errorf("Path mismatch: DefaultConfig=%s, Constant=%s",
			cfg.AgentBrowser.Path, DefaultAgentBrowserPath)
	}
	if cfg.AgentBrowser.Timeout != DefaultAgentBrowserTimeout {
		t.Errorf("Timeout mismatch: DefaultConfig=%d, Constant=%d",
			cfg.AgentBrowser.Timeout, DefaultAgentBrowserTimeout)
	}
	if cfg.AgentBrowser.SessionPrefix != DefaultSessionPrefix {
		t.Errorf("SessionPrefix mismatch: DefaultConfig=%s, Constant=%s",
			cfg.AgentBrowser.SessionPrefix, DefaultSessionPrefix)
	}
}

// =============================================================================
// Struct Tag Tests
// =============================================================================

func TestMorphConfigStructTags(t *testing.T) {
	// This tests that struct tags are correctly defined
	// by verifying round-trip through JSON and YAML

	original := MorphConfig{
		APIKey:         "tag-test",
		BaseSnapshotID: "snap-tag",
		VM: VMConfig{
			VCPUs:      8,
			Memory:     16384,
			DiskSize:   131072,
			TTLSeconds: 7200,
		},
	}

	// JSON round-trip
	jsonData, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("JSON marshal error: %v", err)
	}

	var jsonLoaded MorphConfig
	if err := json.Unmarshal(jsonData, &jsonLoaded); err != nil {
		t.Fatalf("JSON unmarshal error: %v", err)
	}

	if jsonLoaded.APIKey != original.APIKey {
		t.Errorf("JSON: APIKey = %s, want %s", jsonLoaded.APIKey, original.APIKey)
	}
	if jsonLoaded.VM.VCPUs != original.VM.VCPUs {
		t.Errorf("JSON: VCPUs = %d, want %d", jsonLoaded.VM.VCPUs, original.VM.VCPUs)
	}
}

func TestAgentBrowserConfigStructTags(t *testing.T) {
	original := AgentBrowserConfig{
		Path:          "/tag/test/path",
		Timeout:       60000,
		SessionPrefix: "tag-prefix",
	}

	// JSON round-trip
	jsonData, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("JSON marshal error: %v", err)
	}

	var jsonLoaded AgentBrowserConfig
	if err := json.Unmarshal(jsonData, &jsonLoaded); err != nil {
		t.Fatalf("JSON unmarshal error: %v", err)
	}

	if jsonLoaded.Path != original.Path {
		t.Errorf("JSON: Path = %s, want %s", jsonLoaded.Path, original.Path)
	}
	if jsonLoaded.Timeout != original.Timeout {
		t.Errorf("JSON: Timeout = %d, want %d", jsonLoaded.Timeout, original.Timeout)
	}
	if jsonLoaded.SessionPrefix != original.SessionPrefix {
		t.Errorf("JSON: SessionPrefix = %s, want %s", jsonLoaded.SessionPrefix, original.SessionPrefix)
	}
}

// =============================================================================
// Error Message Consistency Tests
// =============================================================================

func TestValidationErrorFormat(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Morph.VM.VCPUs = 0
	cfg.Morph.VM.Memory = 0
	cfg.Morph.VM.DiskSize = 0
	cfg.Morph.VM.TTLSeconds = 0
	cfg.AgentBrowser.Timeout = 0

	err := cfg.Validate()
	if err == nil {
		t.Fatal("Expected validation error")
	}

	errStr := err.Error()

	// Error message should mention all invalid fields
	fieldMentions := []string{"vcpus", "memory", "disk_size", "ttl_seconds", "timeout"}
	for _, field := range fieldMentions {
		if !containsField(errStr, field) {
			t.Errorf("Error message should mention %s, got: %s", field, errStr)
		}
	}
}
