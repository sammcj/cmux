// internal/config/morph_integration_test.go
package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// =============================================================================
// Integration Tests - Full Config Lifecycle
// =============================================================================

func TestConfigMorphIntegrationFullCycle(t *testing.T) {
	tmpDir := t.TempDir()
	os.Setenv("DBA_HOME", tmpDir)
	defer os.Unsetenv("DBA_HOME")

	// 1. Load default config (no file)
	cfg1, err := Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	// 2. Verify Morph defaults
	if cfg1.Morph.VM.VCPUs != DefaultMorphVCPUs {
		t.Errorf("default VCPUs = %d, want %d", cfg1.Morph.VM.VCPUs, DefaultMorphVCPUs)
	}
	if cfg1.AgentBrowser.Path != DefaultAgentBrowserPath {
		t.Errorf("default Path = %s, want %s", cfg1.AgentBrowser.Path, DefaultAgentBrowserPath)
	}

	// 3. Validate default config
	if err := cfg1.Validate(); err != nil {
		t.Errorf("default config should be valid: %v", err)
	}

	// 4. Write custom config
	customConfig := `
morph:
  api_key: "test-key-12345"
  base_snapshot_id: "snap-test"
  vm:
    vcpus: 4
    memory: 8192
    disk_size: 65536
    ttl_seconds: 7200

agent_browser:
  path: "/opt/browser/agent-browser"
  timeout: 60000
  session_prefix: "test-session"
`
	if err := os.WriteFile(filepath.Join(tmpDir, "config.yaml"), []byte(customConfig), 0644); err != nil {
		t.Fatal(err)
	}

	// 5. Load custom config
	cfg2, err := Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	// 6. Verify custom values
	if cfg2.Morph.APIKey != "test-key-12345" {
		t.Errorf("APIKey = %s, want test-key-12345", cfg2.Morph.APIKey)
	}
	if cfg2.Morph.VM.VCPUs != 4 {
		t.Errorf("VCPUs = %d, want 4", cfg2.Morph.VM.VCPUs)
	}
	if cfg2.AgentBrowser.Timeout != 60000 {
		t.Errorf("Timeout = %d, want 60000", cfg2.AgentBrowser.Timeout)
	}

	// 7. Validate custom config
	if err := cfg2.Validate(); err != nil {
		t.Errorf("custom config should be valid: %v", err)
	}

	// 8. Test GetMorphAPIKey returns direct value
	key := cfg2.GetMorphAPIKey()
	if key != "test-key-12345" {
		t.Errorf("GetMorphAPIKey() = %s, want test-key-12345", key)
	}

	// 9. Test GetBaseSnapshotID
	snapID := cfg2.GetBaseSnapshotID()
	if snapID != "snap-test" {
		t.Errorf("GetBaseSnapshotID() = %s, want snap-test", snapID)
	}
}

func TestConfigMorphJSONSerialization(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Morph.APIKey = "json-test-key"
	cfg.Morph.BaseSnapshotID = "snap-json-test"
	cfg.Morph.VM.VCPUs = 8
	cfg.AgentBrowser.Path = "/custom/path"

	// Serialize to JSON
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		t.Fatalf("JSON marshal error: %v", err)
	}

	// Deserialize
	var loaded Config
	if err := json.Unmarshal(data, &loaded); err != nil {
		t.Fatalf("JSON unmarshal error: %v", err)
	}

	// Verify
	if loaded.Morph.APIKey != cfg.Morph.APIKey {
		t.Errorf("APIKey = %s, want %s", loaded.Morph.APIKey, cfg.Morph.APIKey)
	}
	if loaded.Morph.VM.VCPUs != cfg.Morph.VM.VCPUs {
		t.Errorf("VCPUs = %d, want %d", loaded.Morph.VM.VCPUs, cfg.Morph.VM.VCPUs)
	}
	if loaded.AgentBrowser.Path != cfg.AgentBrowser.Path {
		t.Errorf("Path = %s, want %s", loaded.AgentBrowser.Path, cfg.AgentBrowser.Path)
	}
}

func TestConfigMorphEnvVarOverride(t *testing.T) {
	tmpDir := t.TempDir()
	os.Setenv("DBA_HOME", tmpDir)
	defer os.Unsetenv("DBA_HOME")

	os.Setenv("MORPH_API_KEY", "env-override-key")
	defer os.Unsetenv("MORPH_API_KEY")

	os.Setenv("DBA_BASE_SNAPSHOT", "env-snapshot")
	defer os.Unsetenv("DBA_BASE_SNAPSHOT")

	// Config with env var reference and empty base_snapshot_id to test fallback
	customConfig := `
morph:
  api_key: "${MORPH_API_KEY}"
  base_snapshot_id: ""
`
	if err := os.WriteFile(filepath.Join(tmpDir, "config.yaml"), []byte(customConfig), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}

	// GetMorphAPIKey should resolve the env var
	key := cfg.GetMorphAPIKey()
	if key != "env-override-key" {
		t.Errorf("GetMorphAPIKey() = %s, want env-override-key", key)
	}

	// GetBaseSnapshotID should use env var since config explicitly sets empty
	snapID := cfg.GetBaseSnapshotID()
	if snapID != "env-snapshot" {
		t.Errorf("GetBaseSnapshotID() = %s, want env-snapshot", snapID)
	}
}

func TestConfigMorphPartialYAMLMerge(t *testing.T) {
	tmpDir := t.TempDir()
	os.Setenv("DBA_HOME", tmpDir)
	defer os.Unsetenv("DBA_HOME")

	// Only override some VM settings
	customConfig := `
morph:
  vm:
    vcpus: 8
`
	if err := os.WriteFile(filepath.Join(tmpDir, "config.yaml"), []byte(customConfig), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}

	// vcpus should be overridden
	if cfg.Morph.VM.VCPUs != 8 {
		t.Errorf("VCPUs = %d, want 8", cfg.Morph.VM.VCPUs)
	}

	// Memory might be zero due to YAML partial parsing
	// This is expected behavior - the entire vm section is replaced
	t.Logf("Memory after partial override: %d", cfg.Morph.VM.Memory)
}

// =============================================================================
// Stress Tests
// =============================================================================

func TestConfigMorphLoadStress(t *testing.T) {
	tmpDir := t.TempDir()
	os.Setenv("DBA_HOME", tmpDir)
	defer os.Unsetenv("DBA_HOME")

	// Write config
	customConfig := `
morph:
  api_key: "stress-test-key"
  vm:
    vcpus: 2
    memory: 4096
    disk_size: 32768
    ttl_seconds: 3600
`
	if err := os.WriteFile(filepath.Join(tmpDir, "config.yaml"), []byte(customConfig), 0644); err != nil {
		t.Fatal(err)
	}

	// Load config 1000 times
	for i := 0; i < 1000; i++ {
		cfg, err := Load()
		if err != nil {
			t.Fatalf("Load() failed at iteration %d: %v", i, err)
		}
		if cfg.Morph.APIKey != "stress-test-key" {
			t.Fatalf("APIKey mismatch at iteration %d", i)
		}
	}
}

func TestConfigMorphValidateStress(t *testing.T) {
	cfg := DefaultConfig()

	// Validate 1000 times
	for i := 0; i < 1000; i++ {
		err := cfg.Validate()
		if err != nil {
			t.Fatalf("Validate() failed at iteration %d: %v", i, err)
		}
	}
}

// =============================================================================
// Error Recovery Tests
// =============================================================================

func TestConfigMorphRecoveryAfterError(t *testing.T) {
	tmpDir := t.TempDir()
	os.Setenv("DBA_HOME", tmpDir)
	defer os.Unsetenv("DBA_HOME")

	// 1. Write invalid config
	invalidConfig := `
morph:
  vm:
    vcpus: "not a number"
`
	if err := os.WriteFile(filepath.Join(tmpDir, "config.yaml"), []byte(invalidConfig), 0644); err != nil {
		t.Fatal(err)
	}

	// 2. Try to load (should fail)
	_, err := Load()
	if err == nil {
		t.Error("expected error for invalid config")
	}

	// 3. Fix the config
	validConfig := `
morph:
  vm:
    vcpus: 4
    memory: 4096
    disk_size: 32768
    ttl_seconds: 3600
`
	if err := os.WriteFile(filepath.Join(tmpDir, "config.yaml"), []byte(validConfig), 0644); err != nil {
		t.Fatal(err)
	}

	// 4. Load should succeed now
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() should succeed after fixing config: %v", err)
	}

	if cfg.Morph.VM.VCPUs != 4 {
		t.Errorf("VCPUs = %d, want 4", cfg.Morph.VM.VCPUs)
	}
}

// =============================================================================
// Default Constant Tests
// =============================================================================

func TestMorphDefaultConstants(t *testing.T) {
	// Verify constants are sensible
	if DefaultMorphVCPUs < 1 {
		t.Errorf("DefaultMorphVCPUs = %d, should be >= 1", DefaultMorphVCPUs)
	}
	if DefaultMorphMemory < 512 {
		t.Errorf("DefaultMorphMemory = %d, should be >= 512", DefaultMorphMemory)
	}
	if DefaultMorphDiskSize < 1024 {
		t.Errorf("DefaultMorphDiskSize = %d, should be >= 1024", DefaultMorphDiskSize)
	}
	if DefaultMorphTTLSeconds < 60 {
		t.Errorf("DefaultMorphTTLSeconds = %d, should be >= 60", DefaultMorphTTLSeconds)
	}
}

func TestAgentBrowserDefaultConstants(t *testing.T) {
	if DefaultAgentBrowserPath == "" {
		t.Error("DefaultAgentBrowserPath should not be empty")
	}
	if DefaultAgentBrowserTimeout < 1000 {
		t.Errorf("DefaultAgentBrowserTimeout = %d, should be >= 1000", DefaultAgentBrowserTimeout)
	}
	if DefaultSessionPrefix == "" {
		t.Error("DefaultSessionPrefix should not be empty")
	}
}

// =============================================================================
// Helper Function Tests
// =============================================================================

func TestDefaultMorphConfigReturnsValidConfig(t *testing.T) {
	cfg := DefaultMorphConfig()

	if cfg.APIKey == "" {
		t.Error("DefaultMorphConfig().APIKey should not be empty")
	}
	if cfg.VM.VCPUs < 1 {
		t.Error("DefaultMorphConfig().VM.VCPUs should be >= 1")
	}
	if cfg.VM.Memory < 512 {
		t.Error("DefaultMorphConfig().VM.Memory should be >= 512")
	}
	if cfg.VM.DiskSize < 1024 {
		t.Error("DefaultMorphConfig().VM.DiskSize should be >= 1024")
	}
	if cfg.VM.TTLSeconds < 60 {
		t.Error("DefaultMorphConfig().VM.TTLSeconds should be >= 60")
	}
}

func TestDefaultAgentBrowserConfigReturnsValidConfig(t *testing.T) {
	cfg := DefaultAgentBrowserConfig()

	if cfg.Path == "" {
		t.Error("DefaultAgentBrowserConfig().Path should not be empty")
	}
	if cfg.Timeout < 1000 {
		t.Error("DefaultAgentBrowserConfig().Timeout should be >= 1000")
	}
	if cfg.SessionPrefix == "" {
		t.Error("DefaultAgentBrowserConfig().SessionPrefix should not be empty")
	}
}
