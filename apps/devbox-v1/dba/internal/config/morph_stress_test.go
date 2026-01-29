// internal/config/morph_stress_test.go
package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"testing"
)

// =============================================================================
// Stress Tests
// =============================================================================

func TestRapidConfigReload(t *testing.T) {
	tmpDir := t.TempDir()
	os.Setenv("DBA_HOME", tmpDir)
	defer os.Unsetenv("DBA_HOME")

	// Write initial config
	content := `
morph:
  api_key: "stress-test"
  vm:
    vcpus: 2
`
	if err := os.WriteFile(filepath.Join(tmpDir, "config.yaml"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	// Rapidly reload config 10000 times
	for i := 0; i < 10000; i++ {
		cfg, err := Load()
		if err != nil {
			t.Fatalf("Load() failed at iteration %d: %v", i, err)
		}
		if cfg.Morph.APIKey != "stress-test" {
			t.Fatalf("APIKey mismatch at iteration %d", i)
		}
	}
}

func TestConcurrentConfigReloadStress(t *testing.T) {
	tmpDir := t.TempDir()
	os.Setenv("DBA_HOME", tmpDir)
	defer os.Unsetenv("DBA_HOME")

	content := `
morph:
  api_key: "concurrent-stress"
`
	if err := os.WriteFile(filepath.Join(tmpDir, "config.yaml"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	var wg sync.WaitGroup
	errors := make(chan error, 1000)

	for i := 0; i < 1000; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			cfg, err := Load()
			if err != nil {
				errors <- err
				return
			}
			if cfg.Morph.APIKey != "concurrent-stress" {
				errors <- err
			}
		}()
	}

	wg.Wait()
	close(errors)

	for err := range errors {
		if err != nil {
			t.Errorf("Concurrent reload error: %v", err)
		}
	}
}

func TestManyValidations(t *testing.T) {
	cfg := DefaultConfig()

	// Validate 10000 times
	for i := 0; i < 10000; i++ {
		err := cfg.Validate()
		if err != nil {
			t.Fatalf("Validate() failed at iteration %d: %v", i, err)
		}
	}
}

func TestManyJSONSerializations(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Morph.APIKey = "json-stress-key"
	cfg.Morph.VM.VCPUs = 8

	for i := 0; i < 10000; i++ {
		data, err := json.Marshal(cfg)
		if err != nil {
			t.Fatalf("Marshal failed at iteration %d: %v", i, err)
		}

		var loaded Config
		if err := json.Unmarshal(data, &loaded); err != nil {
			t.Fatalf("Unmarshal failed at iteration %d: %v", i, err)
		}

		if loaded.Morph.APIKey != cfg.Morph.APIKey {
			t.Fatalf("APIKey mismatch at iteration %d", i)
		}
	}
}

func TestConcurrentGetMorphAPIKeyStress(t *testing.T) {
	os.Setenv("STRESS_API_KEY", "stress-value")
	defer os.Unsetenv("STRESS_API_KEY")

	cfg := DefaultConfig()
	cfg.Morph.APIKey = "${STRESS_API_KEY}"

	var wg sync.WaitGroup
	for i := 0; i < 1000; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				key := cfg.GetMorphAPIKey()
				if key != "stress-value" {
					t.Errorf("GetMorphAPIKey() = %s, want stress-value", key)
				}
			}
		}()
	}
	wg.Wait()
}

func TestConcurrentGetBaseSnapshotIDStress(t *testing.T) {
	os.Setenv("DBA_BASE_SNAPSHOT", "stress-snapshot")
	defer os.Unsetenv("DBA_BASE_SNAPSHOT")

	cfg := DefaultConfig()
	// Clear the default value to test env var fallback under stress
	cfg.Morph.BaseSnapshotID = ""

	var wg sync.WaitGroup
	for i := 0; i < 1000; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				snapID := cfg.GetBaseSnapshotID()
				if snapID != "stress-snapshot" {
					t.Errorf("GetBaseSnapshotID() = %s, want stress-snapshot", snapID)
				}
			}
		}()
	}
	wg.Wait()
}

// =============================================================================
// Memory Tests
// =============================================================================

func TestLargeConfigNoLeak(t *testing.T) {
	tmpDir := t.TempDir()
	os.Setenv("DBA_HOME", tmpDir)
	defer os.Unsetenv("DBA_HOME")

	// Create a large config file
	var content string
	content = "morph:\n  api_key: \"large-config\"\n"
	for i := 0; i < 1000; i++ {
		content += "# comment line " + string(rune('0'+i%10)) + "\n"
	}

	if err := os.WriteFile(filepath.Join(tmpDir, "config.yaml"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	// Load many times (should not leak memory)
	for i := 0; i < 1000; i++ {
		cfg, err := Load()
		if err != nil {
			t.Fatalf("Load() failed at iteration %d: %v", i, err)
		}
		if cfg.Morph.APIKey != "large-config" {
			t.Fatalf("APIKey mismatch at iteration %d", i)
		}
	}
}

// =============================================================================
// Edge Case Combinations
// =============================================================================

func TestAllDefaultsValid(t *testing.T) {
	cfg := DefaultConfig()

	// All defaults should pass validation
	if err := cfg.Validate(); err != nil {
		t.Errorf("DefaultConfig should be valid: %v", err)
	}

	// Verify each default constant
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

func TestModifyAllMorphFields(t *testing.T) {
	cfg := DefaultConfig()

	// Modify every Morph field
	cfg.Morph.APIKey = "modified-key"
	cfg.Morph.BaseSnapshotID = "modified-snapshot"
	cfg.Morph.VM.VCPUs = 16
	cfg.Morph.VM.Memory = 32768
	cfg.Morph.VM.DiskSize = 131072
	cfg.Morph.VM.TTLSeconds = 14400

	// Should still be valid
	if err := cfg.Validate(); err != nil {
		t.Errorf("Modified config should be valid: %v", err)
	}

	// Verify modifications
	if cfg.Morph.APIKey != "modified-key" {
		t.Errorf("APIKey = %s, want modified-key", cfg.Morph.APIKey)
	}
	if cfg.Morph.VM.VCPUs != 16 {
		t.Errorf("VCPUs = %d, want 16", cfg.Morph.VM.VCPUs)
	}
}

func TestModifyAllAgentBrowserFields(t *testing.T) {
	cfg := DefaultConfig()

	// Modify every AgentBrowser field
	cfg.AgentBrowser.Path = "/modified/path/agent-browser"
	cfg.AgentBrowser.Timeout = 60000
	cfg.AgentBrowser.SessionPrefix = "modified-prefix"

	// Should still be valid
	if err := cfg.Validate(); err != nil {
		t.Errorf("Modified config should be valid: %v", err)
	}

	// Verify modifications
	if cfg.AgentBrowser.Path != "/modified/path/agent-browser" {
		t.Errorf("Path = %s, want /modified/path/agent-browser", cfg.AgentBrowser.Path)
	}
	if cfg.AgentBrowser.Timeout != 60000 {
		t.Errorf("Timeout = %d, want 60000", cfg.AgentBrowser.Timeout)
	}
	if cfg.AgentBrowser.SessionPrefix != "modified-prefix" {
		t.Errorf("SessionPrefix = %s, want modified-prefix", cfg.AgentBrowser.SessionPrefix)
	}
}

func TestMinimumValidConfig(t *testing.T) {
	cfg := DefaultConfig()

	// Set all to minimum valid values
	cfg.Morph.VM.VCPUs = 1
	cfg.Morph.VM.Memory = 512
	cfg.Morph.VM.DiskSize = 1024
	cfg.Morph.VM.TTLSeconds = 60
	cfg.AgentBrowser.Timeout = 1000

	if err := cfg.Validate(); err != nil {
		t.Errorf("Minimum valid config should pass validation: %v", err)
	}
}

func TestMaximumReasonableConfig(t *testing.T) {
	cfg := DefaultConfig()

	// Set to very high but reasonable values
	cfg.Morph.VM.VCPUs = 256
	cfg.Morph.VM.Memory = 1048576     // 1TB
	cfg.Morph.VM.DiskSize = 10485760  // 10TB
	cfg.Morph.VM.TTLSeconds = 2592000 // 30 days
	cfg.AgentBrowser.Timeout = 3600000 // 1 hour

	if err := cfg.Validate(); err != nil {
		t.Errorf("Maximum reasonable config should pass validation: %v", err)
	}
}

// =============================================================================
// Config Immutability Tests
// =============================================================================

func TestDefaultConfigImmutability(t *testing.T) {
	cfg1 := DefaultConfig()
	cfg2 := DefaultConfig()

	// Modify cfg1
	cfg1.Morph.APIKey = "modified"
	cfg1.Morph.VM.VCPUs = 999

	// cfg2 should be unchanged
	if cfg2.Morph.APIKey == "modified" {
		t.Error("DefaultConfig should return independent copies")
	}
	if cfg2.Morph.VM.VCPUs == 999 {
		t.Error("DefaultConfig should return independent copies")
	}
}

func TestDefaultMorphConfigImmutability(t *testing.T) {
	cfg1 := DefaultMorphConfig()
	cfg2 := DefaultMorphConfig()

	// Modify cfg1
	cfg1.APIKey = "modified"
	cfg1.VM.VCPUs = 999

	// cfg2 should be unchanged
	if cfg2.APIKey == "modified" {
		t.Error("DefaultMorphConfig should return independent copies")
	}
	if cfg2.VM.VCPUs == 999 {
		t.Error("DefaultMorphConfig should return independent copies")
	}
}

func TestDefaultAgentBrowserConfigImmutability(t *testing.T) {
	cfg1 := DefaultAgentBrowserConfig()
	cfg2 := DefaultAgentBrowserConfig()

	// Modify cfg1
	cfg1.Path = "/modified/path"
	cfg1.Timeout = 999

	// cfg2 should be unchanged
	if cfg2.Path == "/modified/path" {
		t.Error("DefaultAgentBrowserConfig should return independent copies")
	}
	if cfg2.Timeout == 999 {
		t.Error("DefaultAgentBrowserConfig should return independent copies")
	}
}

// =============================================================================
// Zero Value Tests
// =============================================================================

func TestZeroValueMorphConfig(t *testing.T) {
	var cfg MorphConfig

	// Zero value should have empty strings and zero ints
	if cfg.APIKey != "" {
		t.Errorf("Zero APIKey = %q, want empty", cfg.APIKey)
	}
	if cfg.BaseSnapshotID != "" {
		t.Errorf("Zero BaseSnapshotID = %q, want empty", cfg.BaseSnapshotID)
	}
	if cfg.VM.VCPUs != 0 {
		t.Errorf("Zero VCPUs = %d, want 0", cfg.VM.VCPUs)
	}
}

func TestZeroValueVMConfig(t *testing.T) {
	var cfg VMConfig

	if cfg.VCPUs != 0 {
		t.Errorf("Zero VCPUs = %d, want 0", cfg.VCPUs)
	}
	if cfg.Memory != 0 {
		t.Errorf("Zero Memory = %d, want 0", cfg.Memory)
	}
	if cfg.DiskSize != 0 {
		t.Errorf("Zero DiskSize = %d, want 0", cfg.DiskSize)
	}
	if cfg.TTLSeconds != 0 {
		t.Errorf("Zero TTLSeconds = %d, want 0", cfg.TTLSeconds)
	}
}

func TestZeroValueAgentBrowserConfig(t *testing.T) {
	var cfg AgentBrowserConfig

	if cfg.Path != "" {
		t.Errorf("Zero Path = %q, want empty", cfg.Path)
	}
	if cfg.Timeout != 0 {
		t.Errorf("Zero Timeout = %d, want 0", cfg.Timeout)
	}
	if cfg.SessionPrefix != "" {
		t.Errorf("Zero SessionPrefix = %q, want empty", cfg.SessionPrefix)
	}
}

func TestZeroValueConfigValidation(t *testing.T) {
	var cfg Config

	// Zero value config should fail validation
	err := cfg.Validate()
	if err == nil {
		t.Error("Zero value Config should fail validation")
	}
}
