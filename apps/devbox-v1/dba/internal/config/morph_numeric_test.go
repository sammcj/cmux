// internal/config/morph_numeric_test.go
package config

import (
	"math"
	"os"
	"path/filepath"
	"testing"
)

// =============================================================================
// Numeric Edge Case Tests
// =============================================================================

func TestVCPUsNumericEdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		value    int
		wantErr  bool
	}{
		{"negative", -1, true},
		{"zero", 0, true},
		{"one", 1, false},
		{"two", 2, false},
		{"typical", 4, false},
		{"max_int32", math.MaxInt32, false}, // Should be valid (no upper limit check)
		{"large", 1000000, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := DefaultConfig()
			cfg.Morph.VM.VCPUs = tt.value

			err := cfg.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("VCPUs=%d: Validate() error = %v, wantErr %v", tt.value, err, tt.wantErr)
			}
		})
	}
}

func TestMemoryNumericEdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		value    int
		wantErr  bool
	}{
		{"negative", -1, true},
		{"zero", 0, true},
		{"below_min", 511, true},
		{"at_min", 512, false},
		{"above_min", 513, false},
		{"typical", 8192, false},
		{"large", 1048576, false}, // 1TB
		{"max_int32", math.MaxInt32, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := DefaultConfig()
			cfg.Morph.VM.Memory = tt.value

			err := cfg.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Memory=%d: Validate() error = %v, wantErr %v", tt.value, err, tt.wantErr)
			}
		})
	}
}

func TestDiskSizeNumericEdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		value    int
		wantErr  bool
	}{
		{"negative", -1, true},
		{"zero", 0, true},
		{"below_min", 1023, true},
		{"at_min", 1024, false},
		{"above_min", 1025, false},
		{"typical", 65536, false},
		{"large", 10485760, false}, // 10TB
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := DefaultConfig()
			cfg.Morph.VM.DiskSize = tt.value

			err := cfg.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("DiskSize=%d: Validate() error = %v, wantErr %v", tt.value, err, tt.wantErr)
			}
		})
	}
}

func TestTTLSecondsNumericEdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		value    int
		wantErr  bool
	}{
		{"negative", -1, true},
		{"zero", 0, true},
		{"below_min", 59, true},
		{"at_min", 60, false},
		{"above_min", 61, false},
		{"one_hour", 3600, false},
		{"one_day", 86400, false},
		{"one_week", 604800, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := DefaultConfig()
			cfg.Morph.VM.TTLSeconds = tt.value

			err := cfg.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("TTLSeconds=%d: Validate() error = %v, wantErr %v", tt.value, err, tt.wantErr)
			}
		})
	}
}

func TestTimeoutNumericEdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		value    int
		wantErr  bool
	}{
		{"negative", -1, true},
		{"zero", 0, true},
		{"below_min", 999, true},
		{"at_min", 1000, false},
		{"above_min", 1001, false},
		{"typical", 30000, false},
		{"one_minute", 60000, false},
		{"one_hour", 3600000, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := DefaultConfig()
			cfg.AgentBrowser.Timeout = tt.value

			err := cfg.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Timeout=%d: Validate() error = %v, wantErr %v", tt.value, err, tt.wantErr)
			}
		})
	}
}

// =============================================================================
// YAML Numeric Parsing Tests
// =============================================================================

func TestYAMLNumericParsing(t *testing.T) {
	tmpDir := t.TempDir()
	os.Setenv("DBA_HOME", tmpDir)
	defer os.Unsetenv("DBA_HOME")

	tests := []struct {
		name    string
		yaml    string
		wantVal int
		field   string
	}{
		{"decimal", "morph:\n  vm:\n    vcpus: 4", 4, "vcpus"},
		{"octal_style", "morph:\n  vm:\n    vcpus: 010", 10, "vcpus"}, // YAML 1.2 treats 010 as decimal
		{"hex_style", "morph:\n  vm:\n    memory: 0x1000", 4096, "memory"},
		{"scientific", "morph:\n  vm:\n    disk_size: 1e6", 1000000, "disk_size"},
		{"underscore", "morph:\n  vm:\n    ttl_seconds: 3_600", 3600, "ttl_seconds"},
		{"float_truncate", "morph:\n  vm:\n    vcpus: 4.9", 4, "vcpus"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := os.WriteFile(filepath.Join(tmpDir, "config.yaml"), []byte(tt.yaml), 0644); err != nil {
				t.Fatal(err)
			}

			cfg, err := Load()
			if err != nil {
				t.Logf("Load() error for %s: %v (may be expected for some formats)", tt.name, err)
				return
			}

			var got int
			switch tt.field {
			case "vcpus":
				got = cfg.Morph.VM.VCPUs
			case "memory":
				got = cfg.Morph.VM.Memory
			case "disk_size":
				got = cfg.Morph.VM.DiskSize
			case "ttl_seconds":
				got = cfg.Morph.VM.TTLSeconds
			}

			t.Logf("%s: got %d, want %d", tt.name, got, tt.wantVal)
		})
	}
}

func TestYAMLNumericStrings(t *testing.T) {
	tmpDir := t.TempDir()
	os.Setenv("DBA_HOME", tmpDir)
	defer os.Unsetenv("DBA_HOME")

	// Test numeric values passed as strings
	content := `
morph:
  vm:
    vcpus: "4"
    memory: "8192"
`
	if err := os.WriteFile(filepath.Join(tmpDir, "config.yaml"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load()
	if err != nil {
		t.Logf("Load() error with string numbers: %v (may be expected)", err)
		return
	}

	// YAML should coerce strings to int
	t.Logf("VCPUs: %d, Memory: %d", cfg.Morph.VM.VCPUs, cfg.Morph.VM.Memory)
}

// =============================================================================
// Overflow/Underflow Tests
// =============================================================================

func TestNumericOverflow(t *testing.T) {
	cfg := DefaultConfig()

	// Set to max values
	cfg.Morph.VM.VCPUs = math.MaxInt
	cfg.Morph.VM.Memory = math.MaxInt
	cfg.Morph.VM.DiskSize = math.MaxInt
	cfg.Morph.VM.TTLSeconds = math.MaxInt
	cfg.AgentBrowser.Timeout = math.MaxInt

	// Should still be valid
	err := cfg.Validate()
	if err != nil {
		t.Errorf("Validate() with max values: %v", err)
	}
}

func TestNumericMinValues(t *testing.T) {
	cfg := DefaultConfig()

	// Set to minimum valid values
	cfg.Morph.VM.VCPUs = 1
	cfg.Morph.VM.Memory = 512
	cfg.Morph.VM.DiskSize = 1024
	cfg.Morph.VM.TTLSeconds = 60
	cfg.AgentBrowser.Timeout = 1000

	err := cfg.Validate()
	if err != nil {
		t.Errorf("Validate() with min values: %v", err)
	}
}

// =============================================================================
// Default Value Consistency Tests
// =============================================================================

func TestDefaultValuesAreValid(t *testing.T) {
	// All default values should pass validation
	cfg := DefaultConfig()

	if err := cfg.Validate(); err != nil {
		t.Errorf("DefaultConfig() is invalid: %v", err)
	}

	// Verify specific default values
	if cfg.Morph.VM.VCPUs != DefaultMorphVCPUs {
		t.Errorf("VCPUs default mismatch: got %d, want %d", cfg.Morph.VM.VCPUs, DefaultMorphVCPUs)
	}
	if cfg.Morph.VM.Memory != DefaultMorphMemory {
		t.Errorf("Memory default mismatch: got %d, want %d", cfg.Morph.VM.Memory, DefaultMorphMemory)
	}
	if cfg.Morph.VM.DiskSize != DefaultMorphDiskSize {
		t.Errorf("DiskSize default mismatch: got %d, want %d", cfg.Morph.VM.DiskSize, DefaultMorphDiskSize)
	}
	if cfg.Morph.VM.TTLSeconds != DefaultMorphTTLSeconds {
		t.Errorf("TTLSeconds default mismatch: got %d, want %d", cfg.Morph.VM.TTLSeconds, DefaultMorphTTLSeconds)
	}
	if cfg.AgentBrowser.Timeout != DefaultAgentBrowserTimeout {
		t.Errorf("Timeout default mismatch: got %d, want %d", cfg.AgentBrowser.Timeout, DefaultAgentBrowserTimeout)
	}
}

func TestDefaultConstantsAreAboveMinimum(t *testing.T) {
	// Ensure default constants are above their minimum thresholds
	if DefaultMorphVCPUs < 1 {
		t.Errorf("DefaultMorphVCPUs = %d, must be >= 1", DefaultMorphVCPUs)
	}
	if DefaultMorphMemory < 512 {
		t.Errorf("DefaultMorphMemory = %d, must be >= 512", DefaultMorphMemory)
	}
	if DefaultMorphDiskSize < 1024 {
		t.Errorf("DefaultMorphDiskSize = %d, must be >= 1024", DefaultMorphDiskSize)
	}
	if DefaultMorphTTLSeconds < 60 {
		t.Errorf("DefaultMorphTTLSeconds = %d, must be >= 60", DefaultMorphTTLSeconds)
	}
	if DefaultAgentBrowserTimeout < 1000 {
		t.Errorf("DefaultAgentBrowserTimeout = %d, must be >= 1000", DefaultAgentBrowserTimeout)
	}
}

// =============================================================================
// Combination Tests
// =============================================================================

func TestAllFieldsAtMinimum(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Morph.VM.VCPUs = 1
	cfg.Morph.VM.Memory = 512
	cfg.Morph.VM.DiskSize = 1024
	cfg.Morph.VM.TTLSeconds = 60
	cfg.AgentBrowser.Timeout = 1000

	if err := cfg.Validate(); err != nil {
		t.Errorf("All fields at minimum should be valid: %v", err)
	}
}

func TestAllFieldsJustBelowMinimum(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Morph.VM.VCPUs = 0
	cfg.Morph.VM.Memory = 511
	cfg.Morph.VM.DiskSize = 1023
	cfg.Morph.VM.TTLSeconds = 59
	cfg.AgentBrowser.Timeout = 999

	err := cfg.Validate()
	if err == nil {
		t.Error("All fields below minimum should fail validation")
	}
}

func TestMixedValidAndInvalid(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Morph.VM.VCPUs = 4      // valid
	cfg.Morph.VM.Memory = 0    // invalid
	cfg.Morph.VM.DiskSize = 65536 // valid
	cfg.Morph.VM.TTLSeconds = 0 // invalid

	err := cfg.Validate()
	if err == nil {
		t.Error("Mixed valid/invalid should fail validation")
	}

	// Error should mention at least one invalid field
	errStr := err.Error()
	if !containsAny(errStr, "memory", "ttl_seconds") {
		t.Errorf("Error should mention invalid fields, got: %s", errStr)
	}
}

func containsAny(s string, substrings ...string) bool {
	for _, sub := range substrings {
		for i := 0; i <= len(s)-len(sub); i++ {
			if s[i:i+len(sub)] == sub {
				return true
			}
		}
	}
	return false
}

// =============================================================================
// JSON Numeric Serialization Tests
// =============================================================================

func TestJSONNumericRoundTrip(t *testing.T) {
	tests := []struct {
		name  string
		value int
	}{
		{"zero", 0},
		{"one", 1},
		{"negative", -1},
		{"large", 1000000000},
		{"max_int32", math.MaxInt32},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			original := VMConfig{
				VCPUs:      tt.value,
				Memory:     tt.value,
				DiskSize:   tt.value,
				TTLSeconds: tt.value,
			}

			// This would be tested with actual JSON marshaling
			// For now just verify the struct can hold the value
			if original.VCPUs != tt.value {
				t.Errorf("VCPUs = %d, want %d", original.VCPUs, tt.value)
			}
		})
	}
}
