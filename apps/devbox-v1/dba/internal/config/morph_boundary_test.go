// internal/config/morph_boundary_test.go
package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// =============================================================================
// Boundary Value Tests
// =============================================================================

func TestVMConfigBoundaryValues(t *testing.T) {
	tests := []struct {
		name       string
		vcpus      int
		memory     int
		diskSize   int
		ttlSeconds int
		wantValid  bool
	}{
		// Exact boundary conditions
		{"vcpus_at_min", 1, 4096, 32768, 3600, true},
		{"vcpus_below_min", 0, 4096, 32768, 3600, false},
		{"memory_at_min", 2, 512, 32768, 3600, true},
		{"memory_below_min", 2, 511, 32768, 3600, false},
		{"disk_at_min", 2, 4096, 1024, 3600, true},
		{"disk_below_min", 2, 4096, 1023, 3600, false},
		{"ttl_at_min", 2, 4096, 32768, 60, true},
		{"ttl_below_min", 2, 4096, 32768, 59, false},

		// Power of 2 values (common in VM config)
		{"vcpus_power_of_2", 8, 4096, 32768, 3600, true},
		{"memory_power_of_2", 2, 8192, 32768, 3600, true},
		{"disk_power_of_2", 2, 4096, 65536, 3600, true},

		// Max int32 values (shouldn't overflow)
		{"vcpus_large", 2147483647, 4096, 32768, 3600, true},
		{"memory_large", 2, 2147483647, 32768, 3600, true},
		{"disk_large", 2, 4096, 2147483647, 3600, true},
		{"ttl_large", 2, 4096, 32768, 2147483647, true},

		// All at boundary
		{"all_at_min", 1, 512, 1024, 60, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := DefaultConfig()
			cfg.Morph.VM.VCPUs = tt.vcpus
			cfg.Morph.VM.Memory = tt.memory
			cfg.Morph.VM.DiskSize = tt.diskSize
			cfg.Morph.VM.TTLSeconds = tt.ttlSeconds

			err := cfg.Validate()
			if tt.wantValid && err != nil {
				t.Errorf("Validate() error = %v, want nil", err)
			}
			if !tt.wantValid && err == nil {
				t.Errorf("Validate() error = nil, want error for %s", tt.name)
			}
		})
	}
}

func TestAgentBrowserTimeoutBoundary(t *testing.T) {
	tests := []struct {
		name    string
		timeout int
		wantErr bool
	}{
		{"min_valid", 1000, false},
		{"below_min", 999, true},
		{"zero", 0, true},
		{"negative", -1, true},
		{"large_valid", 3600000, false}, // 1 hour
		{"typical", 30000, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := DefaultConfig()
			cfg.AgentBrowser.Timeout = tt.timeout

			err := cfg.Validate()
			if tt.wantErr && err == nil {
				t.Errorf("Expected error for timeout=%d", tt.timeout)
			}
			if !tt.wantErr && err != nil {
				t.Errorf("Unexpected error for timeout=%d: %v", tt.timeout, err)
			}
		})
	}
}

// =============================================================================
// Default Value Preservation Tests
// =============================================================================

func TestDefaultsPreservedOnPartialLoad(t *testing.T) {
	tmpDir := t.TempDir()
	os.Setenv("DBA_HOME", tmpDir)
	defer os.Unsetenv("DBA_HOME")

	// Write config with only API key
	customConfig := `
morph:
  api_key: "test-key"
`
	if err := os.WriteFile(filepath.Join(tmpDir, "config.yaml"), []byte(customConfig), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	// API key should be set
	if cfg.Morph.APIKey != "test-key" {
		t.Errorf("APIKey = %s, want test-key", cfg.Morph.APIKey)
	}

	// VM defaults should be preserved
	if cfg.Morph.VM.VCPUs != DefaultMorphVCPUs {
		t.Errorf("VCPUs = %d, want %d", cfg.Morph.VM.VCPUs, DefaultMorphVCPUs)
	}
	if cfg.Morph.VM.Memory != DefaultMorphMemory {
		t.Errorf("Memory = %d, want %d", cfg.Morph.VM.Memory, DefaultMorphMemory)
	}
	if cfg.Morph.VM.DiskSize != DefaultMorphDiskSize {
		t.Errorf("DiskSize = %d, want %d", cfg.Morph.VM.DiskSize, DefaultMorphDiskSize)
	}
	if cfg.Morph.VM.TTLSeconds != DefaultMorphTTLSeconds {
		t.Errorf("TTLSeconds = %d, want %d", cfg.Morph.VM.TTLSeconds, DefaultMorphTTLSeconds)
	}

	// AgentBrowser defaults should be preserved
	if cfg.AgentBrowser.Path != DefaultAgentBrowserPath {
		t.Errorf("Path = %s, want %s", cfg.AgentBrowser.Path, DefaultAgentBrowserPath)
	}
	if cfg.AgentBrowser.Timeout != DefaultAgentBrowserTimeout {
		t.Errorf("Timeout = %d, want %d", cfg.AgentBrowser.Timeout, DefaultAgentBrowserTimeout)
	}
	if cfg.AgentBrowser.SessionPrefix != DefaultSessionPrefix {
		t.Errorf("SessionPrefix = %s, want %s", cfg.AgentBrowser.SessionPrefix, DefaultSessionPrefix)
	}
}

func TestEmptyConfigUsesAllDefaults(t *testing.T) {
	tmpDir := t.TempDir()
	os.Setenv("DBA_HOME", tmpDir)
	defer os.Unsetenv("DBA_HOME")

	// Write empty YAML (just a comment)
	customConfig := `# Empty config`
	if err := os.WriteFile(filepath.Join(tmpDir, "config.yaml"), []byte(customConfig), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	defaults := DefaultConfig()

	if cfg.Morph.VM.VCPUs != defaults.Morph.VM.VCPUs {
		t.Errorf("VCPUs = %d, want %d", cfg.Morph.VM.VCPUs, defaults.Morph.VM.VCPUs)
	}
	if cfg.AgentBrowser.Timeout != defaults.AgentBrowser.Timeout {
		t.Errorf("Timeout = %d, want %d", cfg.AgentBrowser.Timeout, defaults.AgentBrowser.Timeout)
	}
}

// =============================================================================
// JSON Round-Trip Tests
// =============================================================================

func TestMorphConfigJSONRoundTrip(t *testing.T) {
	original := MorphConfig{
		APIKey:         "test-key-123",
		BaseSnapshotID: "snap-abc-456",
		VM: VMConfig{
			VCPUs:      8,
			Memory:     16384,
			DiskSize:   131072,
			TTLSeconds: 7200,
		},
	}

	// Marshal
	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}

	// Unmarshal
	var loaded MorphConfig
	if err := json.Unmarshal(data, &loaded); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}

	// Verify
	if loaded.APIKey != original.APIKey {
		t.Errorf("APIKey = %s, want %s", loaded.APIKey, original.APIKey)
	}
	if loaded.BaseSnapshotID != original.BaseSnapshotID {
		t.Errorf("BaseSnapshotID = %s, want %s", loaded.BaseSnapshotID, original.BaseSnapshotID)
	}
	if loaded.VM.VCPUs != original.VM.VCPUs {
		t.Errorf("VCPUs = %d, want %d", loaded.VM.VCPUs, original.VM.VCPUs)
	}
	if loaded.VM.Memory != original.VM.Memory {
		t.Errorf("Memory = %d, want %d", loaded.VM.Memory, original.VM.Memory)
	}
	if loaded.VM.DiskSize != original.VM.DiskSize {
		t.Errorf("DiskSize = %d, want %d", loaded.VM.DiskSize, original.VM.DiskSize)
	}
	if loaded.VM.TTLSeconds != original.VM.TTLSeconds {
		t.Errorf("TTLSeconds = %d, want %d", loaded.VM.TTLSeconds, original.VM.TTLSeconds)
	}
}

func TestAgentBrowserConfigJSONRoundTrip(t *testing.T) {
	original := AgentBrowserConfig{
		Path:          "/custom/path/agent-browser",
		Timeout:       45000,
		SessionPrefix: "custom-prefix",
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}

	var loaded AgentBrowserConfig
	if err := json.Unmarshal(data, &loaded); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}

	if loaded.Path != original.Path {
		t.Errorf("Path = %s, want %s", loaded.Path, original.Path)
	}
	if loaded.Timeout != original.Timeout {
		t.Errorf("Timeout = %d, want %d", loaded.Timeout, original.Timeout)
	}
	if loaded.SessionPrefix != original.SessionPrefix {
		t.Errorf("SessionPrefix = %s, want %s", loaded.SessionPrefix, original.SessionPrefix)
	}
}

func TestVMConfigJSONFieldNames(t *testing.T) {
	cfg := VMConfig{
		VCPUs:      4,
		Memory:     8192,
		DiskSize:   65536,
		TTLSeconds: 3600,
	}

	data, err := json.Marshal(cfg)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}

	jsonStr := string(data)

	// Verify JSON field names match expected format
	expectedFields := []string{
		`"vcpus"`,
		`"memory"`,
		`"disk_size"`,
		`"ttl_seconds"`,
	}

	for _, field := range expectedFields {
		if !contains(jsonStr, field) {
			t.Errorf("JSON should contain field %s, got: %s", field, jsonStr)
		}
	}
}

func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// =============================================================================
// Error Message Tests
// =============================================================================

func TestValidateErrorMessages(t *testing.T) {
	tests := []struct {
		name           string
		setupFunc      func(*Config)
		expectedSubstr string
	}{
		{
			name: "zero_vcpus_message",
			setupFunc: func(c *Config) {
				c.Morph.VM.VCPUs = 0
			},
			expectedSubstr: "vcpus",
		},
		{
			name: "zero_memory_message",
			setupFunc: func(c *Config) {
				c.Morph.VM.Memory = 0
			},
			expectedSubstr: "memory",
		},
		{
			name: "zero_disk_message",
			setupFunc: func(c *Config) {
				c.Morph.VM.DiskSize = 0
			},
			expectedSubstr: "disk_size",
		},
		{
			name: "zero_ttl_message",
			setupFunc: func(c *Config) {
				c.Morph.VM.TTLSeconds = 0
			},
			expectedSubstr: "ttl_seconds",
		},
		{
			name: "zero_timeout_message",
			setupFunc: func(c *Config) {
				c.AgentBrowser.Timeout = 0
			},
			expectedSubstr: "timeout",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := DefaultConfig()
			tt.setupFunc(cfg)

			err := cfg.Validate()
			if err == nil {
				t.Fatal("Expected validation error")
			}

			if !contains(err.Error(), tt.expectedSubstr) {
				t.Errorf("Error message should contain %q, got: %s", tt.expectedSubstr, err.Error())
			}
		})
	}
}

// =============================================================================
// Config Reload Tests
// =============================================================================

func TestConfigReloadAfterChange(t *testing.T) {
	tmpDir := t.TempDir()
	os.Setenv("DBA_HOME", tmpDir)
	defer os.Unsetenv("DBA_HOME")

	// Write initial config
	config1 := `
morph:
  api_key: "initial-key"
  vm:
    vcpus: 2
`
	if err := os.WriteFile(filepath.Join(tmpDir, "config.yaml"), []byte(config1), 0644); err != nil {
		t.Fatal(err)
	}

	// Load initial
	cfg1, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg1.Morph.APIKey != "initial-key" {
		t.Errorf("Initial APIKey = %s, want initial-key", cfg1.Morph.APIKey)
	}
	if cfg1.Morph.VM.VCPUs != 2 {
		t.Errorf("Initial VCPUs = %d, want 2", cfg1.Morph.VM.VCPUs)
	}

	// Update config file
	config2 := `
morph:
  api_key: "updated-key"
  vm:
    vcpus: 4
`
	if err := os.WriteFile(filepath.Join(tmpDir, "config.yaml"), []byte(config2), 0644); err != nil {
		t.Fatal(err)
	}

	// Load again
	cfg2, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg2.Morph.APIKey != "updated-key" {
		t.Errorf("Updated APIKey = %s, want updated-key", cfg2.Morph.APIKey)
	}
	if cfg2.Morph.VM.VCPUs != 4 {
		t.Errorf("Updated VCPUs = %d, want 4", cfg2.Morph.VM.VCPUs)
	}
}

func TestConfigNoFileUsesDefaults(t *testing.T) {
	tmpDir := t.TempDir()
	os.Setenv("DBA_HOME", tmpDir)
	defer os.Unsetenv("DBA_HOME")

	// Don't create any config file
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	defaults := DefaultConfig()

	if cfg.Morph.VM.VCPUs != defaults.Morph.VM.VCPUs {
		t.Errorf("VCPUs = %d, want %d", cfg.Morph.VM.VCPUs, defaults.Morph.VM.VCPUs)
	}
	if cfg.AgentBrowser.Path != defaults.AgentBrowser.Path {
		t.Errorf("Path = %s, want %s", cfg.AgentBrowser.Path, defaults.AgentBrowser.Path)
	}
}

// =============================================================================
// Env Var Fallback Tests
// =============================================================================

func TestAPIKeyEnvVarFallback(t *testing.T) {
	os.Setenv("MORPH_API_KEY", "env-fallback-key")
	defer os.Unsetenv("MORPH_API_KEY")

	cfg := DefaultConfig()
	// No API key set in config

	key := cfg.GetMorphAPIKey()
	if key != "" {
		// If no ${} reference in config, env var is not checked
		t.Logf("GetMorphAPIKey() without reference = %q", key)
	}

	// Now set with reference
	cfg.Morph.APIKey = "${MORPH_API_KEY}"
	key = cfg.GetMorphAPIKey()
	if key != "env-fallback-key" {
		t.Errorf("GetMorphAPIKey() = %s, want env-fallback-key", key)
	}
}

func TestSnapshotIDEnvVarFallback(t *testing.T) {
	os.Setenv("DBA_BASE_SNAPSHOT", "env-snapshot-id")
	defer os.Unsetenv("DBA_BASE_SNAPSHOT")

	cfg := DefaultConfig()
	// Default config now has a snapshot ID, so config takes precedence
	// First verify the default is used
	snapID := cfg.GetBaseSnapshotID()
	if snapID != "snapshot_3namut0l" {
		t.Errorf("GetBaseSnapshotID() = %s, want snapshot_3namut0l (default)", snapID)
	}

	// Clear the config value to test env var fallback
	cfg.Morph.BaseSnapshotID = ""
	snapID = cfg.GetBaseSnapshotID()
	if snapID != "env-snapshot-id" {
		t.Errorf("GetBaseSnapshotID() with empty config = %s, want env-snapshot-id", snapID)
	}

	// If config has value, it should take precedence
	cfg.Morph.BaseSnapshotID = "config-snapshot-id"
	snapID = cfg.GetBaseSnapshotID()
	if snapID != "config-snapshot-id" {
		t.Errorf("GetBaseSnapshotID() = %s, want config-snapshot-id", snapID)
	}
}

// =============================================================================
// Helper Function Tests
// =============================================================================

func TestDefaultMorphConfigValues(t *testing.T) {
	cfg := DefaultMorphConfig()

	if cfg.VM.VCPUs != DefaultMorphVCPUs {
		t.Errorf("VCPUs = %d, want %d", cfg.VM.VCPUs, DefaultMorphVCPUs)
	}
	if cfg.VM.Memory != DefaultMorphMemory {
		t.Errorf("Memory = %d, want %d", cfg.VM.Memory, DefaultMorphMemory)
	}
	if cfg.VM.DiskSize != DefaultMorphDiskSize {
		t.Errorf("DiskSize = %d, want %d", cfg.VM.DiskSize, DefaultMorphDiskSize)
	}
	if cfg.VM.TTLSeconds != DefaultMorphTTLSeconds {
		t.Errorf("TTLSeconds = %d, want %d", cfg.VM.TTLSeconds, DefaultMorphTTLSeconds)
	}
}

func TestDefaultAgentBrowserConfigValues(t *testing.T) {
	cfg := DefaultAgentBrowserConfig()

	if cfg.Path != DefaultAgentBrowserPath {
		t.Errorf("Path = %s, want %s", cfg.Path, DefaultAgentBrowserPath)
	}
	if cfg.Timeout != DefaultAgentBrowserTimeout {
		t.Errorf("Timeout = %d, want %d", cfg.Timeout, DefaultAgentBrowserTimeout)
	}
	if cfg.SessionPrefix != DefaultSessionPrefix {
		t.Errorf("SessionPrefix = %s, want %s", cfg.SessionPrefix, DefaultSessionPrefix)
	}
}
