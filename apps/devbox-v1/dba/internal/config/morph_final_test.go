// internal/config/morph_final_test.go
package config

import (
	"os"
	"path/filepath"
	"sync"
	"testing"
)

// =============================================================================
// Final Edge Case Tests
// =============================================================================

func TestDefaultConfigNeverChanges(t *testing.T) {
	// Get default values to compare
	expected := DefaultConfig()

	// Get defaults 1000 times and verify consistency
	for i := 0; i < 1000; i++ {
		cfg := DefaultConfig()

		if cfg.Morph.VM.VCPUs != expected.Morph.VM.VCPUs {
			t.Fatalf("VCPUs changed at iteration %d", i)
		}
		if cfg.Morph.VM.Memory != expected.Morph.VM.Memory {
			t.Fatalf("Memory changed at iteration %d", i)
		}
		if cfg.AgentBrowser.Timeout != expected.AgentBrowser.Timeout {
			t.Fatalf("Timeout changed at iteration %d", i)
		}
	}
}

func TestConcurrentDefaultConfig(t *testing.T) {
	var wg sync.WaitGroup
	results := make(chan *Config, 1000)

	for i := 0; i < 1000; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			results <- DefaultConfig()
		}()
	}

	wg.Wait()
	close(results)

	var first *Config
	for cfg := range results {
		if first == nil {
			first = cfg
			continue
		}

		// All defaults should be the same
		if cfg.Morph.VM.VCPUs != first.Morph.VM.VCPUs {
			t.Error("VCPUs inconsistent across concurrent calls")
		}
		if cfg.AgentBrowser.Timeout != first.AgentBrowser.Timeout {
			t.Error("Timeout inconsistent across concurrent calls")
		}
	}
}

func TestValidateDoesNotModifyDefaults(t *testing.T) {
	cfg := DefaultConfig()
	original := *cfg

	// Validate many times
	for i := 0; i < 100; i++ {
		_ = cfg.Validate()
	}

	// Verify unchanged
	if cfg.Morph.VM.VCPUs != original.Morph.VM.VCPUs {
		t.Error("VCPUs changed after Validate")
	}
	if cfg.AgentBrowser.Timeout != original.AgentBrowser.Timeout {
		t.Error("Timeout changed after Validate")
	}
}

func TestGetMorphAPIKeyDoesNotModifyDefaults(t *testing.T) {
	os.Setenv("TEST_FINAL_KEY", "final-key")
	defer os.Unsetenv("TEST_FINAL_KEY")

	cfg := DefaultConfig()
	cfg.Morph.APIKey = "${TEST_FINAL_KEY}"
	original := cfg.Morph.APIKey

	// Get API key many times
	for i := 0; i < 100; i++ {
		_ = cfg.GetMorphAPIKey()
	}

	// APIKey reference should be unchanged
	if cfg.Morph.APIKey != original {
		t.Error("APIKey reference changed after GetMorphAPIKey")
	}
}

func TestLoadWithSameConfigMultipleTimes(t *testing.T) {
	tmpDir := t.TempDir()
	os.Setenv("DBA_HOME", tmpDir)
	defer os.Unsetenv("DBA_HOME")

	content := `
morph:
  api_key: "consistency-test"
  vm:
    vcpus: 4
`
	if err := os.WriteFile(filepath.Join(tmpDir, "config.yaml"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	// Load multiple times and verify consistency
	for i := 0; i < 100; i++ {
		cfg, err := Load()
		if err != nil {
			t.Fatalf("Load() error at iteration %d: %v", i, err)
		}

		if cfg.Morph.APIKey != "consistency-test" {
			t.Fatalf("APIKey inconsistent at iteration %d", i)
		}
		if cfg.Morph.VM.VCPUs != 4 {
			t.Fatalf("VCPUs inconsistent at iteration %d", i)
		}
	}
}

func TestConfigWithAllFieldsSet(t *testing.T) {
	tmpDir := t.TempDir()
	os.Setenv("DBA_HOME", tmpDir)
	defer os.Unsetenv("DBA_HOME")

	content := `
morph:
  api_key: "all-fields-key"
  base_snapshot_id: "all-fields-snap"
  vm:
    vcpus: 8
    memory: 16384
    disk_size: 131072
    ttl_seconds: 7200

agent_browser:
  path: "/all/fields/path"
  timeout: 60000
  session_prefix: "all-fields-prefix"
`
	if err := os.WriteFile(filepath.Join(tmpDir, "config.yaml"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	// Verify all fields
	if cfg.Morph.APIKey != "all-fields-key" {
		t.Errorf("APIKey = %s, want all-fields-key", cfg.Morph.APIKey)
	}
	if cfg.Morph.BaseSnapshotID != "all-fields-snap" {
		t.Errorf("BaseSnapshotID = %s, want all-fields-snap", cfg.Morph.BaseSnapshotID)
	}
	if cfg.Morph.VM.VCPUs != 8 {
		t.Errorf("VCPUs = %d, want 8", cfg.Morph.VM.VCPUs)
	}
	if cfg.Morph.VM.Memory != 16384 {
		t.Errorf("Memory = %d, want 16384", cfg.Morph.VM.Memory)
	}
	if cfg.Morph.VM.DiskSize != 131072 {
		t.Errorf("DiskSize = %d, want 131072", cfg.Morph.VM.DiskSize)
	}
	if cfg.Morph.VM.TTLSeconds != 7200 {
		t.Errorf("TTLSeconds = %d, want 7200", cfg.Morph.VM.TTLSeconds)
	}
	if cfg.AgentBrowser.Path != "/all/fields/path" {
		t.Errorf("Path = %s, want /all/fields/path", cfg.AgentBrowser.Path)
	}
	if cfg.AgentBrowser.Timeout != 60000 {
		t.Errorf("Timeout = %d, want 60000", cfg.AgentBrowser.Timeout)
	}
	if cfg.AgentBrowser.SessionPrefix != "all-fields-prefix" {
		t.Errorf("SessionPrefix = %s, want all-fields-prefix", cfg.AgentBrowser.SessionPrefix)
	}
}

func TestConfigValidationAllFields(t *testing.T) {
	cfg := &Config{
		Morph: MorphConfig{
			APIKey:         "validation-key",
			BaseSnapshotID: "validation-snap",
			VM: VMConfig{
				VCPUs:      4,
				Memory:     8192,
				DiskSize:   65536,
				TTLSeconds: 3600,
			},
		},
		AgentBrowser: AgentBrowserConfig{
			Path:          "/validation/path",
			Timeout:       30000,
			SessionPrefix: "validation",
		},
	}

	if err := cfg.Validate(); err != nil {
		t.Errorf("Valid config failed validation: %v", err)
	}
}

func TestConfigValidationWithJustAboveMinimum(t *testing.T) {
	cfg := DefaultConfig()

	// Set all to just above minimum
	cfg.Morph.VM.VCPUs = 2
	cfg.Morph.VM.Memory = 513
	cfg.Morph.VM.DiskSize = 1025
	cfg.Morph.VM.TTLSeconds = 61
	cfg.AgentBrowser.Timeout = 1001

	if err := cfg.Validate(); err != nil {
		t.Errorf("Just-above-minimum config failed: %v", err)
	}
}

func TestConfigValidationWithAllInvalid(t *testing.T) {
	cfg := &Config{
		Morph: MorphConfig{
			VM: VMConfig{
				VCPUs:      0,
				Memory:     0,
				DiskSize:   0,
				TTLSeconds: 0,
			},
		},
		AgentBrowser: AgentBrowserConfig{
			Path:    "",
			Timeout: 0,
		},
	}

	err := cfg.Validate()
	if err == nil {
		t.Error("Expected validation error for all-invalid config")
	}

	// Error should mention multiple fields
	errStr := err.Error()
	fieldsToCheck := []string{"vcpus", "memory", "disk_size", "ttl_seconds", "timeout"}
	for _, field := range fieldsToCheck {
		found := false
		for i := 0; i <= len(errStr)-len(field); i++ {
			if errStr[i:i+len(field)] == field {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Error should mention %s, got: %s", field, errStr)
		}
	}
}

func TestEnvVarResolutionOrder(t *testing.T) {
	// Set env var
	os.Setenv("RESOLUTION_TEST", "env-value")
	defer os.Unsetenv("RESOLUTION_TEST")

	cfg := DefaultConfig()

	// Direct value should not resolve env var
	cfg.Morph.APIKey = "direct-value"
	got := cfg.GetMorphAPIKey()
	if got != "direct-value" {
		t.Errorf("Direct value = %s, want direct-value", got)
	}

	// Reference should resolve
	cfg.Morph.APIKey = "${RESOLUTION_TEST}"
	got = cfg.GetMorphAPIKey()
	if got != "env-value" {
		t.Errorf("Resolved value = %s, want env-value", got)
	}
}

func TestSnapshotIDFallbackToEnv(t *testing.T) {
	os.Setenv("DBA_BASE_SNAPSHOT", "fallback-snapshot")
	defer os.Unsetenv("DBA_BASE_SNAPSHOT")

	cfg := DefaultConfig()

	// Empty config value should fall back to env
	cfg.Morph.BaseSnapshotID = ""
	got := cfg.GetBaseSnapshotID()
	if got != "fallback-snapshot" {
		t.Errorf("Fallback = %s, want fallback-snapshot", got)
	}

	// Config value takes precedence
	cfg.Morph.BaseSnapshotID = "config-snapshot"
	got = cfg.GetBaseSnapshotID()
	if got != "config-snapshot" {
		t.Errorf("Precedence = %s, want config-snapshot", got)
	}
}

// =============================================================================
// Constants Verification
// =============================================================================

func TestAllDefaultConstantsAreReasonable(t *testing.T) {
	// VCPUs
	if DefaultMorphVCPUs < 1 || DefaultMorphVCPUs > 128 {
		t.Errorf("DefaultMorphVCPUs = %d, should be reasonable (1-128)", DefaultMorphVCPUs)
	}

	// Memory (in MB)
	if DefaultMorphMemory < 512 || DefaultMorphMemory > 1048576 {
		t.Errorf("DefaultMorphMemory = %d, should be reasonable (512MB-1TB)", DefaultMorphMemory)
	}

	// Disk (in MB)
	if DefaultMorphDiskSize < 1024 || DefaultMorphDiskSize > 10485760 {
		t.Errorf("DefaultMorphDiskSize = %d, should be reasonable (1GB-10TB)", DefaultMorphDiskSize)
	}

	// TTL (in seconds)
	if DefaultMorphTTLSeconds < 60 || DefaultMorphTTLSeconds > 604800 {
		t.Errorf("DefaultMorphTTLSeconds = %d, should be reasonable (1min-1week)", DefaultMorphTTLSeconds)
	}

	// Browser timeout (in ms)
	if DefaultAgentBrowserTimeout < 1000 || DefaultAgentBrowserTimeout > 3600000 {
		t.Errorf("DefaultAgentBrowserTimeout = %d, should be reasonable (1s-1hr)", DefaultAgentBrowserTimeout)
	}

	// Path should not be empty
	if DefaultAgentBrowserPath == "" {
		t.Error("DefaultAgentBrowserPath should not be empty")
	}

	// Session prefix should not be empty
	if DefaultSessionPrefix == "" {
		t.Error("DefaultSessionPrefix should not be empty")
	}
}

func TestConstantsMatchDefaults(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.Morph.VM.VCPUs != DefaultMorphVCPUs {
		t.Errorf("VCPUs: default=%d, constant=%d", cfg.Morph.VM.VCPUs, DefaultMorphVCPUs)
	}
	if cfg.Morph.VM.Memory != DefaultMorphMemory {
		t.Errorf("Memory: default=%d, constant=%d", cfg.Morph.VM.Memory, DefaultMorphMemory)
	}
	if cfg.Morph.VM.DiskSize != DefaultMorphDiskSize {
		t.Errorf("DiskSize: default=%d, constant=%d", cfg.Morph.VM.DiskSize, DefaultMorphDiskSize)
	}
	if cfg.Morph.VM.TTLSeconds != DefaultMorphTTLSeconds {
		t.Errorf("TTLSeconds: default=%d, constant=%d", cfg.Morph.VM.TTLSeconds, DefaultMorphTTLSeconds)
	}
	if cfg.AgentBrowser.Path != DefaultAgentBrowserPath {
		t.Errorf("Path: default=%s, constant=%s", cfg.AgentBrowser.Path, DefaultAgentBrowserPath)
	}
	if cfg.AgentBrowser.Timeout != DefaultAgentBrowserTimeout {
		t.Errorf("Timeout: default=%d, constant=%d", cfg.AgentBrowser.Timeout, DefaultAgentBrowserTimeout)
	}
	if cfg.AgentBrowser.SessionPrefix != DefaultSessionPrefix {
		t.Errorf("SessionPrefix: default=%s, constant=%s", cfg.AgentBrowser.SessionPrefix, DefaultSessionPrefix)
	}
}
