// internal/config/morph_copy_test.go
package config

import (
	"testing"
)

// =============================================================================
// Copy and Independence Tests
// =============================================================================

func TestDefaultConfigIndependence(t *testing.T) {
	// Multiple calls should return independent configs
	cfg1 := DefaultConfig()
	cfg2 := DefaultConfig()

	// Modify cfg1
	cfg1.Morph.VM.VCPUs = 999
	cfg1.Morph.APIKey = "modified"
	cfg1.AgentBrowser.Timeout = 1

	// cfg2 should be unchanged
	if cfg2.Morph.VM.VCPUs == 999 {
		t.Error("cfg2.VCPUs was modified when cfg1 was changed")
	}
	if cfg2.Morph.APIKey == "modified" {
		t.Error("cfg2.APIKey was modified when cfg1 was changed")
	}
	if cfg2.AgentBrowser.Timeout == 1 {
		t.Error("cfg2.Timeout was modified when cfg1 was changed")
	}
}

func TestDefaultMorphConfigIndependence(t *testing.T) {
	m1 := DefaultMorphConfig()
	m2 := DefaultMorphConfig()

	m1.VM.VCPUs = 999
	m1.APIKey = "modified"

	if m2.VM.VCPUs == 999 {
		t.Error("m2.VCPUs was modified when m1 was changed")
	}
	if m2.APIKey == "modified" {
		t.Error("m2.APIKey was modified when m1 was changed")
	}
}

func TestDefaultAgentBrowserConfigIndependence(t *testing.T) {
	b1 := DefaultAgentBrowserConfig()
	b2 := DefaultAgentBrowserConfig()

	b1.Timeout = 999
	b1.Path = "modified"

	if b2.Timeout == 999 {
		t.Error("b2.Timeout was modified when b1 was changed")
	}
	if b2.Path == "modified" {
		t.Error("b2.Path was modified when b1 was changed")
	}
}

// =============================================================================
// Struct Assignment Tests
// =============================================================================

func TestVMConfigAssignment(t *testing.T) {
	original := VMConfig{
		VCPUs:      4,
		Memory:     8192,
		DiskSize:   65536,
		TTLSeconds: 3600,
	}

	copy := original
	copy.VCPUs = 999

	if original.VCPUs == 999 {
		t.Error("Original was modified after copy assignment")
	}
}

func TestMorphConfigAssignment(t *testing.T) {
	original := MorphConfig{
		APIKey: "original-key",
		VM: VMConfig{
			VCPUs: 4,
		},
	}

	copy := original
	copy.APIKey = "modified-key"
	copy.VM.VCPUs = 999

	if original.APIKey == "modified-key" {
		t.Error("Original APIKey was modified")
	}
	// Note: VMConfig is a value type, so this should also be independent
	if original.VM.VCPUs == 999 {
		t.Error("Original VM.VCPUs was modified")
	}
}

func TestAgentBrowserConfigAssignment(t *testing.T) {
	original := AgentBrowserConfig{
		Path:          "/original/path",
		Timeout:       30000,
		SessionPrefix: "original",
	}

	copy := original
	copy.Path = "/modified/path"
	copy.Timeout = 999

	if original.Path == "/modified/path" {
		t.Error("Original Path was modified")
	}
	if original.Timeout == 999 {
		t.Error("Original Timeout was modified")
	}
}

func TestConfigAssignment(t *testing.T) {
	original := DefaultConfig()
	copy := *original

	copy.Morph.APIKey = "modified-key"
	copy.AgentBrowser.Path = "/modified/path"

	if original.Morph.APIKey == "modified-key" {
		t.Error("Original Morph.APIKey was modified")
	}
	if original.AgentBrowser.Path == "/modified/path" {
		t.Error("Original AgentBrowser.Path was modified")
	}
}

// =============================================================================
// Zero Value Tests
// =============================================================================

func TestZeroVMConfig(t *testing.T) {
	var cfg VMConfig

	if cfg.VCPUs != 0 {
		t.Errorf("Zero VCPUs = %d, want 0", cfg.VCPUs)
	}
	if cfg.Memory != 0 {
		t.Errorf("Zero Memory = %d, want 0", cfg.Memory)
	}
}

func TestZeroMorphConfig(t *testing.T) {
	var cfg MorphConfig

	if cfg.APIKey != "" {
		t.Errorf("Zero APIKey = %q, want empty", cfg.APIKey)
	}
	if cfg.VM.VCPUs != 0 {
		t.Errorf("Zero VM.VCPUs = %d, want 0", cfg.VM.VCPUs)
	}
}

func TestZeroAgentBrowserConfig(t *testing.T) {
	var cfg AgentBrowserConfig

	if cfg.Path != "" {
		t.Errorf("Zero Path = %q, want empty", cfg.Path)
	}
	if cfg.Timeout != 0 {
		t.Errorf("Zero Timeout = %d, want 0", cfg.Timeout)
	}
}

func TestZeroConfig(t *testing.T) {
	var cfg Config

	if cfg.Morph.APIKey != "" {
		t.Errorf("Zero Morph.APIKey = %q, want empty", cfg.Morph.APIKey)
	}
	if cfg.AgentBrowser.Path != "" {
		t.Errorf("Zero AgentBrowser.Path = %q, want empty", cfg.AgentBrowser.Path)
	}
}

// =============================================================================
// Nil Pointer Tests
// =============================================================================

func TestNilConfigValidation(t *testing.T) {
	var cfg *Config = nil

	defer func() {
		if r := recover(); r != nil {
			t.Logf("Validate on nil panics as expected: %v", r)
		}
	}()

	// This may panic
	_ = cfg.Validate()
	t.Log("Validate on nil did not panic")
}

func TestNilConfigGetters(t *testing.T) {
	var cfg *Config = nil

	defer func() {
		if r := recover(); r != nil {
			t.Logf("Getter on nil panics as expected: %v", r)
		}
	}()

	// This may panic
	_ = cfg.GetMorphAPIKey()
	t.Log("GetMorphAPIKey on nil did not panic")
}

// =============================================================================
// Equality Tests
// =============================================================================

func TestVMConfigEquality(t *testing.T) {
	cfg1 := VMConfig{VCPUs: 4, Memory: 8192}
	cfg2 := VMConfig{VCPUs: 4, Memory: 8192}

	if cfg1 != cfg2 {
		t.Error("Equal VMConfigs are not equal")
	}

	cfg2.VCPUs = 8
	if cfg1 == cfg2 {
		t.Error("Unequal VMConfigs are equal")
	}
}

func TestAgentBrowserConfigEquality(t *testing.T) {
	cfg1 := AgentBrowserConfig{Path: "/test", Timeout: 30000}
	cfg2 := AgentBrowserConfig{Path: "/test", Timeout: 30000}

	if cfg1 != cfg2 {
		t.Error("Equal AgentBrowserConfigs are not equal")
	}

	cfg2.Timeout = 60000
	if cfg1 == cfg2 {
		t.Error("Unequal AgentBrowserConfigs are equal")
	}
}

// =============================================================================
// Method Chaining Tests
// =============================================================================

func TestMultipleGetMorphAPIKeyCalls(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Morph.APIKey = "test-key"

	// Multiple calls should return the same value
	result1 := cfg.GetMorphAPIKey()
	result2 := cfg.GetMorphAPIKey()
	result3 := cfg.GetMorphAPIKey()

	if result1 != result2 || result2 != result3 {
		t.Errorf("Inconsistent results: %q, %q, %q", result1, result2, result3)
	}
}

func TestMultipleValidateCalls(t *testing.T) {
	cfg := DefaultConfig()

	// Multiple validation calls should give same result
	err1 := cfg.Validate()
	err2 := cfg.Validate()
	err3 := cfg.Validate()

	if (err1 != nil) != (err2 != nil) || (err2 != nil) != (err3 != nil) {
		t.Error("Inconsistent validation results")
	}
}

// =============================================================================
// Empty vs Default Tests
// =============================================================================

func TestEmptyVsDefault(t *testing.T) {
	empty := &Config{}
	def := DefaultConfig()

	// Empty config should have different values from default
	if empty.Morph.VM.VCPUs == def.Morph.VM.VCPUs && def.Morph.VM.VCPUs != 0 {
		t.Error("Empty config has same VCPUs as default (both non-zero)")
	}
	if empty.AgentBrowser.Timeout == def.AgentBrowser.Timeout && def.AgentBrowser.Timeout != 0 {
		t.Error("Empty config has same Timeout as default (both non-zero)")
	}
}

func TestEmptyConfigValidation(t *testing.T) {
	cfg := &Config{}

	err := cfg.Validate()
	if err == nil {
		t.Error("Empty config should fail validation")
	}
}
