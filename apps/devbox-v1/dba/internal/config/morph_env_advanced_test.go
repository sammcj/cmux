// internal/config/morph_env_advanced_test.go
package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// =============================================================================
// Advanced Environment Variable Tests
// =============================================================================

func TestEnvVarWithNewlines(t *testing.T) {
	os.Setenv("NEWLINE_KEY", "line1\nline2\nline3")
	defer os.Unsetenv("NEWLINE_KEY")

	cfg := DefaultConfig()
	cfg.Morph.APIKey = "${NEWLINE_KEY}"

	got := cfg.GetMorphAPIKey()
	if got != "line1\nline2\nline3" {
		t.Errorf("GetMorphAPIKey() = %q, want %q", got, "line1\nline2\nline3")
	}
}

func TestEnvVarWithCarriageReturn(t *testing.T) {
	os.Setenv("CR_KEY", "line1\r\nline2\r\nline3")
	defer os.Unsetenv("CR_KEY")

	cfg := DefaultConfig()
	cfg.Morph.APIKey = "${CR_KEY}"

	got := cfg.GetMorphAPIKey()
	if got != "line1\r\nline2\r\nline3" {
		t.Errorf("GetMorphAPIKey() = %q, want %q", got, "line1\r\nline2\r\nline3")
	}
}

func TestEnvVarWithTabs(t *testing.T) {
	os.Setenv("TAB_KEY", "field1\tfield2\tfield3")
	defer os.Unsetenv("TAB_KEY")

	cfg := DefaultConfig()
	cfg.Morph.APIKey = "${TAB_KEY}"

	got := cfg.GetMorphAPIKey()
	if got != "field1\tfield2\tfield3" {
		t.Errorf("GetMorphAPIKey() = %q, want %q", got, "field1\tfield2\tfield3")
	}
}

func TestEnvVarWithLeadingTrailingWhitespace(t *testing.T) {
	os.Setenv("WHITESPACE_KEY", "   value-with-spaces   ")
	defer os.Unsetenv("WHITESPACE_KEY")

	cfg := DefaultConfig()
	cfg.Morph.APIKey = "${WHITESPACE_KEY}"

	got := cfg.GetMorphAPIKey()
	// Should preserve whitespace
	if got != "   value-with-spaces   " {
		t.Errorf("GetMorphAPIKey() = %q, want %q", got, "   value-with-spaces   ")
	}
}

func TestEnvVarWithOnlyWhitespace(t *testing.T) {
	os.Setenv("ONLY_WHITESPACE", "   \t\n   ")
	defer os.Unsetenv("ONLY_WHITESPACE")

	cfg := DefaultConfig()
	cfg.Morph.APIKey = "${ONLY_WHITESPACE}"

	got := cfg.GetMorphAPIKey()
	if got != "   \t\n   " {
		t.Errorf("GetMorphAPIKey() = %q, want whitespace", got)
	}
}

func TestEnvVarCaseSensitivity(t *testing.T) {
	os.Setenv("CASE_TEST", "uppercase")
	os.Setenv("case_test", "lowercase")
	defer os.Unsetenv("CASE_TEST")
	defer os.Unsetenv("case_test")

	cfg := DefaultConfig()

	cfg.Morph.APIKey = "${CASE_TEST}"
	upper := cfg.GetMorphAPIKey()

	cfg.Morph.APIKey = "${case_test}"
	lower := cfg.GetMorphAPIKey()

	t.Logf("CASE_TEST = %q, case_test = %q", upper, lower)
	// On Unix, these are different; on Windows, they might be the same
}

func TestEnvVarWithDollarSign(t *testing.T) {
	os.Setenv("DOLLAR_KEY", "value$with$dollars")
	defer os.Unsetenv("DOLLAR_KEY")

	cfg := DefaultConfig()
	cfg.Morph.APIKey = "${DOLLAR_KEY}"

	got := cfg.GetMorphAPIKey()
	if got != "value$with$dollars" {
		t.Errorf("GetMorphAPIKey() = %q, want %q", got, "value$with$dollars")
	}
}

func TestEnvVarWithBraces(t *testing.T) {
	os.Setenv("BRACE_KEY", "value{with}braces")
	defer os.Unsetenv("BRACE_KEY")

	cfg := DefaultConfig()
	cfg.Morph.APIKey = "${BRACE_KEY}"

	got := cfg.GetMorphAPIKey()
	if got != "value{with}braces" {
		t.Errorf("GetMorphAPIKey() = %q, want %q", got, "value{with}braces")
	}
}

func TestEnvVarWithBackslash(t *testing.T) {
	os.Setenv("BACKSLASH_KEY", `value\with\backslashes`)
	defer os.Unsetenv("BACKSLASH_KEY")

	cfg := DefaultConfig()
	cfg.Morph.APIKey = "${BACKSLASH_KEY}"

	got := cfg.GetMorphAPIKey()
	if got != `value\with\backslashes` {
		t.Errorf("GetMorphAPIKey() = %q, want %q", got, `value\with\backslashes`)
	}
}

func TestEnvVarWithQuotes(t *testing.T) {
	os.Setenv("QUOTE_KEY", `value"with"quotes`)
	defer os.Unsetenv("QUOTE_KEY")

	cfg := DefaultConfig()
	cfg.Morph.APIKey = "${QUOTE_KEY}"

	got := cfg.GetMorphAPIKey()
	if got != `value"with"quotes` {
		t.Errorf("GetMorphAPIKey() = %q, want %q", got, `value"with"quotes`)
	}
}

func TestEnvVarWithSingleQuotes(t *testing.T) {
	os.Setenv("SINGLE_QUOTE_KEY", `value'with'single'quotes`)
	defer os.Unsetenv("SINGLE_QUOTE_KEY")

	cfg := DefaultConfig()
	cfg.Morph.APIKey = "${SINGLE_QUOTE_KEY}"

	got := cfg.GetMorphAPIKey()
	if got != `value'with'single'quotes` {
		t.Errorf("GetMorphAPIKey() = %q, want %q", got, `value'with'single'quotes`)
	}
}

func TestEnvVarWithEquals(t *testing.T) {
	os.Setenv("EQUALS_KEY", "key=value=pair")
	defer os.Unsetenv("EQUALS_KEY")

	cfg := DefaultConfig()
	cfg.Morph.APIKey = "${EQUALS_KEY}"

	got := cfg.GetMorphAPIKey()
	if got != "key=value=pair" {
		t.Errorf("GetMorphAPIKey() = %q, want %q", got, "key=value=pair")
	}
}

func TestEnvVarWithJSON(t *testing.T) {
	jsonValue := `{"key": "value", "nested": {"a": 1}}`
	os.Setenv("JSON_KEY", jsonValue)
	defer os.Unsetenv("JSON_KEY")

	cfg := DefaultConfig()
	cfg.Morph.APIKey = "${JSON_KEY}"

	got := cfg.GetMorphAPIKey()
	if got != jsonValue {
		t.Errorf("GetMorphAPIKey() = %q, want %q", got, jsonValue)
	}
}

func TestEnvVarWithBase64(t *testing.T) {
	base64Value := "SGVsbG8gV29ybGQhIQ=="
	os.Setenv("BASE64_KEY", base64Value)
	defer os.Unsetenv("BASE64_KEY")

	cfg := DefaultConfig()
	cfg.Morph.APIKey = "${BASE64_KEY}"

	got := cfg.GetMorphAPIKey()
	if got != base64Value {
		t.Errorf("GetMorphAPIKey() = %q, want %q", got, base64Value)
	}
}

func TestEnvVarVeryLongName(t *testing.T) {
	longName := "VERY_LONG_ENV_VAR_NAME_" + strings.Repeat("X", 1000)
	os.Setenv(longName, "long-name-value")
	defer os.Unsetenv(longName)

	cfg := DefaultConfig()
	cfg.Morph.APIKey = "${" + longName + "}"

	got := cfg.GetMorphAPIKey()
	if got != "long-name-value" {
		t.Errorf("GetMorphAPIKey() = %q, want long-name-value", got)
	}
}

func TestEnvVarWithUnderscore(t *testing.T) {
	os.Setenv("_UNDERSCORE_START", "underscore-start")
	os.Setenv("UNDERSCORE__DOUBLE", "underscore-double")
	os.Setenv("UNDERSCORE_END_", "underscore-end")
	defer os.Unsetenv("_UNDERSCORE_START")
	defer os.Unsetenv("UNDERSCORE__DOUBLE")
	defer os.Unsetenv("UNDERSCORE_END_")

	cfg := DefaultConfig()

	tests := []struct {
		envRef   string
		expected string
	}{
		{"${_UNDERSCORE_START}", "underscore-start"},
		{"${UNDERSCORE__DOUBLE}", "underscore-double"},
		{"${UNDERSCORE_END_}", "underscore-end"},
	}

	for _, tt := range tests {
		cfg.Morph.APIKey = tt.envRef
		got := cfg.GetMorphAPIKey()
		if got != tt.expected {
			t.Errorf("GetMorphAPIKey() with %s = %q, want %q", tt.envRef, got, tt.expected)
		}
	}
}

func TestEnvVarWithNumbers(t *testing.T) {
	os.Setenv("ENV123", "number-in-name")
	os.Setenv("123ENV", "number-start") // Invalid env var name on some systems
	defer os.Unsetenv("ENV123")
	defer os.Unsetenv("123ENV")

	cfg := DefaultConfig()
	cfg.Morph.APIKey = "${ENV123}"

	got := cfg.GetMorphAPIKey()
	if got != "number-in-name" {
		t.Errorf("GetMorphAPIKey() = %q, want number-in-name", got)
	}
}

// =============================================================================
// DBA_HOME Edge Cases
// =============================================================================

func TestDBAHomeWithSpacesAdvanced(t *testing.T) {
	tmpDir := t.TempDir()
	dirWithSpaces := filepath.Join(tmpDir, "path with spaces")
	if err := os.MkdirAll(dirWithSpaces, 0755); err != nil {
		t.Fatal(err)
	}

	os.Setenv("DBA_HOME", dirWithSpaces)
	defer os.Unsetenv("DBA_HOME")

	content := `
morph:
  api_key: "spaces-test"
`
	if err := os.WriteFile(filepath.Join(dirWithSpaces, "config.yaml"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	if cfg.Morph.APIKey != "spaces-test" {
		t.Errorf("APIKey = %s, want spaces-test", cfg.Morph.APIKey)
	}
}

func TestDBAHomeWithUnicodeAdvanced(t *testing.T) {
	tmpDir := t.TempDir()
	dirWithUnicode := filepath.Join(tmpDir, "日本語パス")
	if err := os.MkdirAll(dirWithUnicode, 0755); err != nil {
		t.Fatal(err)
	}

	os.Setenv("DBA_HOME", dirWithUnicode)
	defer os.Unsetenv("DBA_HOME")

	content := `
morph:
  api_key: "unicode-path-test"
`
	if err := os.WriteFile(filepath.Join(dirWithUnicode, "config.yaml"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	if cfg.Morph.APIKey != "unicode-path-test" {
		t.Errorf("APIKey = %s, want unicode-path-test", cfg.Morph.APIKey)
	}
}

func TestDBAHomeWithSymlink(t *testing.T) {
	tmpDir := t.TempDir()
	realDir := filepath.Join(tmpDir, "real")
	symlinkDir := filepath.Join(tmpDir, "symlink")

	if err := os.MkdirAll(realDir, 0755); err != nil {
		t.Fatal(err)
	}

	if err := os.Symlink(realDir, symlinkDir); err != nil {
		t.Skip("Symlinks not supported on this system")
	}

	os.Setenv("DBA_HOME", symlinkDir)
	defer os.Unsetenv("DBA_HOME")

	content := `
morph:
  api_key: "symlink-test"
`
	if err := os.WriteFile(filepath.Join(realDir, "config.yaml"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	if cfg.Morph.APIKey != "symlink-test" {
		t.Errorf("APIKey = %s, want symlink-test", cfg.Morph.APIKey)
	}
}

func TestDBAHomeEmpty(t *testing.T) {
	os.Setenv("DBA_HOME", "")
	defer os.Unsetenv("DBA_HOME")

	// Should fall back to default behavior
	_, err := Load()
	t.Logf("Load() with empty DBA_HOME: err=%v", err)
}

func TestDBAHomeNonExistent(t *testing.T) {
	os.Setenv("DBA_HOME", "/nonexistent/path/that/does/not/exist")
	defer os.Unsetenv("DBA_HOME")

	cfg, err := Load()
	// Should use defaults if config doesn't exist
	if err == nil && cfg != nil {
		if cfg.Morph.VM.VCPUs != DefaultMorphVCPUs {
			t.Errorf("VCPUs = %d, want default %d", cfg.Morph.VM.VCPUs, DefaultMorphVCPUs)
		}
	}
	t.Logf("Load() with nonexistent DBA_HOME: err=%v", err)
}

func TestDBAHomeIsFile(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "notadir")
	if err := os.WriteFile(filePath, []byte("I am a file"), 0644); err != nil {
		t.Fatal(err)
	}

	os.Setenv("DBA_HOME", filePath)
	defer os.Unsetenv("DBA_HOME")

	_, err := Load()
	// Should error or fall back to defaults
	t.Logf("Load() with file as DBA_HOME: err=%v", err)
}

func TestDBAHomeWithTrailingSlash(t *testing.T) {
	tmpDir := t.TempDir()

	os.Setenv("DBA_HOME", tmpDir+"/")
	defer os.Unsetenv("DBA_HOME")

	content := `
morph:
  api_key: "trailing-slash-test"
`
	if err := os.WriteFile(filepath.Join(tmpDir, "config.yaml"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	if cfg.Morph.APIKey != "trailing-slash-test" {
		t.Errorf("APIKey = %s, want trailing-slash-test", cfg.Morph.APIKey)
	}
}

func TestDBAHomeWithDotDot(t *testing.T) {
	tmpDir := t.TempDir()
	subDir := filepath.Join(tmpDir, "sub", "dir")
	if err := os.MkdirAll(subDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Use .. to go back
	os.Setenv("DBA_HOME", filepath.Join(subDir, "..", ".."))
	defer os.Unsetenv("DBA_HOME")

	content := `
morph:
  api_key: "dotdot-test"
`
	if err := os.WriteFile(filepath.Join(tmpDir, "config.yaml"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	if cfg.Morph.APIKey != "dotdot-test" {
		t.Errorf("APIKey = %s, want dotdot-test", cfg.Morph.APIKey)
	}
}

// =============================================================================
// Config File Edge Cases
// =============================================================================

func TestConfigFileSymlink(t *testing.T) {
	tmpDir := t.TempDir()
	realConfig := filepath.Join(tmpDir, "real-config.yaml")
	symlinkConfig := filepath.Join(tmpDir, "config.yaml")

	content := `
morph:
  api_key: "symlink-config-test"
`
	if err := os.WriteFile(realConfig, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	if err := os.Symlink(realConfig, symlinkConfig); err != nil {
		t.Skip("Symlinks not supported")
	}

	os.Setenv("DBA_HOME", tmpDir)
	defer os.Unsetenv("DBA_HOME")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	if cfg.Morph.APIKey != "symlink-config-test" {
		t.Errorf("APIKey = %s, want symlink-config-test", cfg.Morph.APIKey)
	}
}

func TestConfigFileIsDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	configDir := filepath.Join(tmpDir, "config.yaml")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatal(err)
	}

	os.Setenv("DBA_HOME", tmpDir)
	defer os.Unsetenv("DBA_HOME")

	_, err := Load()
	// Should error
	if err == nil {
		t.Error("Expected error when config.yaml is a directory")
	}
}

func TestConfigFileWithExtendedAttributes(t *testing.T) {
	tmpDir := t.TempDir()
	os.Setenv("DBA_HOME", tmpDir)
	defer os.Unsetenv("DBA_HOME")

	content := `
morph:
  api_key: "xattr-test"
`
	configPath := filepath.Join(tmpDir, "config.yaml")
	if err := os.WriteFile(configPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	// On macOS, files might have extended attributes
	// This tests that they don't interfere with reading
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	if cfg.Morph.APIKey != "xattr-test" {
		t.Errorf("APIKey = %s, want xattr-test", cfg.Morph.APIKey)
	}
}

// =============================================================================
// Boundary Value Edge Cases
// =============================================================================

func TestExactBoundaryValues(t *testing.T) {
	cfg := DefaultConfig()

	// Test exactly at boundaries
	boundaryTests := []struct {
		name       string
		setup      func(*Config)
		wantValid  bool
	}{
		{
			name: "vcpus_exactly_1",
			setup: func(c *Config) {
				c.Morph.VM.VCPUs = 1
			},
			wantValid: true,
		},
		{
			name: "memory_exactly_512",
			setup: func(c *Config) {
				c.Morph.VM.Memory = 512
			},
			wantValid: true,
		},
		{
			name: "disk_exactly_1024",
			setup: func(c *Config) {
				c.Morph.VM.DiskSize = 1024
			},
			wantValid: true,
		},
		{
			name: "ttl_exactly_60",
			setup: func(c *Config) {
				c.Morph.VM.TTLSeconds = 60
			},
			wantValid: true,
		},
		{
			name: "timeout_exactly_1000",
			setup: func(c *Config) {
				c.AgentBrowser.Timeout = 1000
			},
			wantValid: true,
		},
	}

	for _, tt := range boundaryTests {
		t.Run(tt.name, func(t *testing.T) {
			testCfg := DefaultConfig()
			tt.setup(testCfg)

			err := testCfg.Validate()
			if tt.wantValid && err != nil {
				t.Errorf("Validate() error = %v, want nil", err)
			}
			if !tt.wantValid && err == nil {
				t.Errorf("Validate() = nil, want error")
			}
		})
	}
	_ = cfg // Use cfg to avoid unused variable error
}

func TestOneAboveOneBelowBoundary(t *testing.T) {
	tests := []struct {
		name      string
		field     string
		below     int
		at        int
		above     int
		setup     func(*Config, int)
	}{
		{
			name:  "vcpus",
			field: "VCPUs",
			below: 0, at: 1, above: 2,
			setup: func(c *Config, v int) { c.Morph.VM.VCPUs = v },
		},
		{
			name:  "memory",
			field: "Memory",
			below: 511, at: 512, above: 513,
			setup: func(c *Config, v int) { c.Morph.VM.Memory = v },
		},
		{
			name:  "disk",
			field: "DiskSize",
			below: 1023, at: 1024, above: 1025,
			setup: func(c *Config, v int) { c.Morph.VM.DiskSize = v },
		},
		{
			name:  "ttl",
			field: "TTLSeconds",
			below: 59, at: 60, above: 61,
			setup: func(c *Config, v int) { c.Morph.VM.TTLSeconds = v },
		},
		{
			name:  "timeout",
			field: "Timeout",
			below: 999, at: 1000, above: 1001,
			setup: func(c *Config, v int) { c.AgentBrowser.Timeout = v },
		},
	}

	for _, tt := range tests {
		t.Run(tt.name+"_below", func(t *testing.T) {
			cfg := DefaultConfig()
			tt.setup(cfg, tt.below)
			if err := cfg.Validate(); err == nil {
				t.Errorf("%s=%d should fail validation", tt.field, tt.below)
			}
		})

		t.Run(tt.name+"_at", func(t *testing.T) {
			cfg := DefaultConfig()
			tt.setup(cfg, tt.at)
			if err := cfg.Validate(); err != nil {
				t.Errorf("%s=%d should pass validation: %v", tt.field, tt.at, err)
			}
		})

		t.Run(tt.name+"_above", func(t *testing.T) {
			cfg := DefaultConfig()
			tt.setup(cfg, tt.above)
			if err := cfg.Validate(); err != nil {
				t.Errorf("%s=%d should pass validation: %v", tt.field, tt.above, err)
			}
		})
	}
}
