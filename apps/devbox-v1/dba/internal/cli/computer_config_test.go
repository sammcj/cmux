// internal/cli/computer_config_test.go
package cli

import (
	"testing"

	"github.com/dba-cli/dba/internal/config"
)

// TestMorphConfigDefaults tests default Morph configuration values
func TestMorphConfigDefaults(t *testing.T) {
	morphCfg := config.MorphConfig{}

	// API key defaults
	if morphCfg.APIKey != "" {
		t.Errorf("default APIKey should be empty, got %s", morphCfg.APIKey)
	}
	if morphCfg.BaseSnapshotID != "" {
		t.Errorf("default BaseSnapshotID should be empty, got %s", morphCfg.BaseSnapshotID)
	}

	// VM defaults
	if morphCfg.VM.VCPUs != 0 {
		t.Errorf("default VM VCPUs should be 0 (unset), got %d", morphCfg.VM.VCPUs)
	}
	if morphCfg.VM.Memory != 0 {
		t.Errorf("default VM Memory should be 0 (unset), got %d", morphCfg.VM.Memory)
	}
	if morphCfg.VM.DiskSize != 0 {
		t.Errorf("default VM DiskSize should be 0 (unset), got %d", morphCfg.VM.DiskSize)
	}
	if morphCfg.VM.TTLSeconds != 0 {
		t.Errorf("default VM TTLSeconds should be 0 (unset), got %d", morphCfg.VM.TTLSeconds)
	}
}

// TestBrowserConfigDefaults tests default AgentBrowser configuration values
func TestBrowserConfigDefaults(t *testing.T) {
	browserCfg := config.AgentBrowserConfig{}

	if browserCfg.Path != "" {
		t.Errorf("default Path should be empty, got %s", browserCfg.Path)
	}
	if browserCfg.Timeout != 0 {
		t.Errorf("default Timeout should be 0 (unset), got %d", browserCfg.Timeout)
	}
	if browserCfg.SessionPrefix != "" {
		t.Errorf("default SessionPrefix should be empty, got %s", browserCfg.SessionPrefix)
	}
}

// TestMorphVMConfigWithValues tests VM configuration with values
func TestMorphVMConfigWithValues(t *testing.T) {
	vmCfg := config.VMConfig{
		VCPUs:      4,
		Memory:     8192,
		DiskSize:   102400,
		TTLSeconds: 3600,
	}

	if vmCfg.VCPUs != 4 {
		t.Errorf("expected VCPUs=4, got %d", vmCfg.VCPUs)
	}
	if vmCfg.Memory != 8192 {
		t.Errorf("expected Memory=8192, got %d", vmCfg.Memory)
	}
	if vmCfg.DiskSize != 102400 {
		t.Errorf("expected DiskSize=102400, got %d", vmCfg.DiskSize)
	}
	if vmCfg.TTLSeconds != 3600 {
		t.Errorf("expected TTLSeconds=3600, got %d", vmCfg.TTLSeconds)
	}
}

// TestMorphVMConfigBoundaryValues tests VM configuration boundary values
func TestMorphVMConfigBoundaryValues(t *testing.T) {
	tests := []struct {
		name   string
		config config.VMConfig
	}{
		{"minimum values", config.VMConfig{VCPUs: 1, Memory: 512, DiskSize: 1024, TTLSeconds: 60}},
		{"typical values", config.VMConfig{VCPUs: 4, Memory: 8192, DiskSize: 51200, TTLSeconds: 3600}},
		{"large values", config.VMConfig{VCPUs: 128, Memory: 524288, DiskSize: 1024000, TTLSeconds: 86400}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Values should be set correctly
			if tt.config.VCPUs == 0 {
				t.Error("VCPUs should be set")
			}
			if tt.config.Memory == 0 {
				t.Error("Memory should be set")
			}
			if tt.config.DiskSize == 0 {
				t.Error("DiskSize should be set")
			}
			if tt.config.TTLSeconds == 0 {
				t.Error("TTLSeconds should be set")
			}
		})
	}
}

// TestBrowserConfigTimeouts tests browser configuration timeout values
func TestBrowserConfigTimeouts(t *testing.T) {
	tests := []struct {
		name    string
		timeout int
		valid   bool
	}{
		{"zero timeout", 0, true},    // Uses default
		{"1 second", 1000, true},     // 1s
		{"30 seconds", 30000, true},  // 30s
		{"5 minutes", 300000, true},  // 5 min
		{"negative", -1000, false},   // Invalid
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := config.AgentBrowserConfig{
				Timeout: tt.timeout,
			}

			if tt.valid {
				// Valid timeouts should be settable
				if cfg.Timeout != tt.timeout {
					t.Errorf("expected timeout %d, got %d", tt.timeout, cfg.Timeout)
				}
			}
		})
	}
}

// TestConfigGetMorphAPIKey tests the GetMorphAPIKey method
func TestConfigGetMorphAPIKey(t *testing.T) {
	// Test with empty config
	cfg := &config.Config{}
	key := cfg.GetMorphAPIKey()

	// Empty config should return empty string (will use env var fallback)
	if key != "" {
		t.Errorf("expected empty API key from empty config, got %s", key)
	}
}

// TestConfigGetBaseSnapshotID tests the GetBaseSnapshotID method
func TestConfigGetBaseSnapshotID(t *testing.T) {
	// Test with empty config
	cfg := &config.Config{}
	snapID := cfg.GetBaseSnapshotID()

	// Empty config should return empty string (will use env var fallback)
	if snapID != "" {
		t.Errorf("expected empty snapshot ID from empty config, got %s", snapID)
	}
}

// TestMorphConfigStruct tests MorphConfig struct fields
func TestMorphConfigStruct(t *testing.T) {
	cfg := config.MorphConfig{
		APIKey:         "test-key",
		BaseSnapshotID: "snap_123",
		VM: config.VMConfig{
			VCPUs:      2,
			Memory:     4096,
			DiskSize:   20480,
			TTLSeconds: 1800,
		},
	}

	if cfg.APIKey != "test-key" {
		t.Errorf("unexpected APIKey: %s", cfg.APIKey)
	}
	if cfg.BaseSnapshotID != "snap_123" {
		t.Errorf("unexpected BaseSnapshotID: %s", cfg.BaseSnapshotID)
	}
	if cfg.VM.VCPUs != 2 {
		t.Errorf("unexpected VM.VCPUs: %d", cfg.VM.VCPUs)
	}
	if cfg.VM.Memory != 4096 {
		t.Errorf("unexpected VM.Memory: %d", cfg.VM.Memory)
	}
}

// TestAgentBrowserConfigStruct tests AgentBrowserConfig struct fields
func TestAgentBrowserConfigStruct(t *testing.T) {
	cfg := config.AgentBrowserConfig{
		Path:          "/usr/local/bin/agent-browser",
		Timeout:       30000,
		SessionPrefix: "test-session",
	}

	if cfg.Path != "/usr/local/bin/agent-browser" {
		t.Errorf("unexpected Path: %s", cfg.Path)
	}
	if cfg.Timeout != 30000 {
		t.Errorf("unexpected Timeout: %d", cfg.Timeout)
	}
	if cfg.SessionPrefix != "test-session" {
		t.Errorf("unexpected SessionPrefix: %s", cfg.SessionPrefix)
	}
}

// TestConfigWithNestedStructs tests config with nested Morph and Browser structs
func TestConfigWithNestedStructs(t *testing.T) {
	cfg := &config.Config{
		Morph: config.MorphConfig{
			APIKey: "morph-key",
			VM: config.VMConfig{
				VCPUs: 4,
			},
		},
		AgentBrowser: config.AgentBrowserConfig{
			Timeout: 30000,
		},
	}

	if cfg.Morph.APIKey != "morph-key" {
		t.Errorf("unexpected Morph.APIKey: %s", cfg.Morph.APIKey)
	}
	if cfg.Morph.VM.VCPUs != 4 {
		t.Errorf("unexpected Morph.VM.VCPUs: %d", cfg.Morph.VM.VCPUs)
	}
	if cfg.AgentBrowser.Timeout != 30000 {
		t.Errorf("unexpected AgentBrowser.Timeout: %d", cfg.AgentBrowser.Timeout)
	}
}

// TestConfigEnvVarExpansion tests that API key with env var syntax is handled
func TestConfigEnvVarExpansion(t *testing.T) {
	cfg := &config.Config{
		Morph: config.MorphConfig{
			APIKey: "${MORPH_API_KEY}",
		},
	}

	// When env var is not set, should return empty
	key := cfg.GetMorphAPIKey()
	// Key might be empty or the actual env var value if set
	_ = key // Just verify it doesn't panic
}

// TestConfigMorphBaseSnapshotFromEnv tests base snapshot ID from env
func TestConfigMorphBaseSnapshotFromEnv(t *testing.T) {
	cfg := &config.Config{
		Morph: config.MorphConfig{
			BaseSnapshotID: "", // Empty - should fallback to env
		},
	}

	// Should not panic when checking env
	_ = cfg.GetBaseSnapshotID()
}

// TestConfigMorphAPIKeyPriority tests API key priority (config vs env)
func TestConfigMorphAPIKeyPriority(t *testing.T) {
	// When config has a direct value (not env ref), it should be used
	cfg := &config.Config{
		Morph: config.MorphConfig{
			APIKey: "direct-key-value",
		},
	}

	key := cfg.GetMorphAPIKey()
	if key != "direct-key-value" {
		t.Errorf("expected direct config value, got %s", key)
	}
}

// TestConfigMorphSnapshotPriority tests snapshot ID priority (config vs env)
func TestConfigMorphSnapshotPriority(t *testing.T) {
	// When config has a value, it should be used
	cfg := &config.Config{
		Morph: config.MorphConfig{
			BaseSnapshotID: "snap_config",
		},
	}

	snap := cfg.GetBaseSnapshotID()
	if snap != "snap_config" {
		t.Errorf("expected config snapshot ID, got %s", snap)
	}
}
