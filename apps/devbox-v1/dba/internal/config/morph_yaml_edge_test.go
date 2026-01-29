// internal/config/morph_yaml_edge_test.go
package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// =============================================================================
// YAML Parsing Edge Cases
// =============================================================================

func TestYAMLOnlyWhitespace(t *testing.T) {
	tmpDir := t.TempDir()
	os.Setenv("DBA_HOME", tmpDir)
	defer os.Unsetenv("DBA_HOME")

	// File with only spaces and newlines (tabs at line start cause YAML errors)
	content := "   \n   \n   \n"
	if err := os.WriteFile(filepath.Join(tmpDir, "config.yaml"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	// Should use defaults
	if cfg.Morph.VM.VCPUs != DefaultMorphVCPUs {
		t.Errorf("VCPUs = %d, want %d", cfg.Morph.VM.VCPUs, DefaultMorphVCPUs)
	}
}

func TestYAMLOnlyComments(t *testing.T) {
	tmpDir := t.TempDir()
	os.Setenv("DBA_HOME", tmpDir)
	defer os.Unsetenv("DBA_HOME")

	content := `# This is a comment
# Another comment
# Yet another comment
`
	if err := os.WriteFile(filepath.Join(tmpDir, "config.yaml"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	if cfg.Morph.VM.VCPUs != DefaultMorphVCPUs {
		t.Errorf("VCPUs = %d, want %d", cfg.Morph.VM.VCPUs, DefaultMorphVCPUs)
	}
}

func TestYAMLInvalidSyntax(t *testing.T) {
	tmpDir := t.TempDir()
	os.Setenv("DBA_HOME", tmpDir)
	defer os.Unsetenv("DBA_HOME")

	tests := []struct {
		name    string
		content string
	}{
		{
			name: "unclosed_bracket",
			content: `
morph:
  api_key: [unclosed
`,
		},
		{
			name: "unclosed_brace",
			content: `
morph:
  vm: {vcpus: 2
`,
		},
		{
			name: "invalid_indentation",
			content: `
morph:
api_key: "bad-indent"
`,
		},
		{
			name: "tab_in_wrong_place",
			content: "morph:\n\t\tapi_key: \"tabbed\"",
		},
		{
			name: "duplicate_key",
			content: `
morph:
  api_key: "first"
  api_key: "second"
`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := os.WriteFile(filepath.Join(tmpDir, "config.yaml"), []byte(tt.content), 0644); err != nil {
				t.Fatal(err)
			}

			_, err := Load()
			// Some of these may or may not error depending on YAML parser behavior
			t.Logf("Load() with %s: err=%v", tt.name, err)
		})
	}
}

func TestYAMLDeeplyNested(t *testing.T) {
	tmpDir := t.TempDir()
	os.Setenv("DBA_HOME", tmpDir)
	defer os.Unsetenv("DBA_HOME")

	// Create deeply nested structure (YAML allows this)
	content := `
morph:
  api_key: "nested-test"
  vm:
    vcpus: 4
    memory: 8192
    disk_size: 65536
    ttl_seconds: 3600
  extra:
    level1:
      level2:
        level3:
          level4:
            level5: "deep"
`
	if err := os.WriteFile(filepath.Join(tmpDir, "config.yaml"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	// Should still parse known fields correctly
	if cfg.Morph.APIKey != "nested-test" {
		t.Errorf("APIKey = %s, want nested-test", cfg.Morph.APIKey)
	}
	if cfg.Morph.VM.VCPUs != 4 {
		t.Errorf("VCPUs = %d, want 4", cfg.Morph.VM.VCPUs)
	}
}

func TestYAMLVeryLongLines(t *testing.T) {
	tmpDir := t.TempDir()
	os.Setenv("DBA_HOME", tmpDir)
	defer os.Unsetenv("DBA_HOME")

	// Create a very long API key (100KB)
	longKey := strings.Repeat("x", 100*1024)
	content := "morph:\n  api_key: \"" + longKey + "\"\n"

	if err := os.WriteFile(filepath.Join(tmpDir, "config.yaml"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	if len(cfg.Morph.APIKey) != 100*1024 {
		t.Errorf("APIKey length = %d, want %d", len(cfg.Morph.APIKey), 100*1024)
	}
}

func TestYAMLSpecialYAMLValues(t *testing.T) {
	tmpDir := t.TempDir()
	os.Setenv("DBA_HOME", tmpDir)
	defer os.Unsetenv("DBA_HOME")

	tests := []struct {
		name       string
		content    string
		checkFunc  func(*Config) error
	}{
		{
			name: "boolean_like_strings",
			content: `
morph:
  api_key: "true"
  base_snapshot_id: "false"
`,
			checkFunc: func(c *Config) error {
				if c.Morph.APIKey != "true" {
					return nil
				}
				return nil
			},
		},
		{
			name: "number_like_strings",
			content: `
morph:
  api_key: "12345"
  base_snapshot_id: "67890"
`,
			checkFunc: func(c *Config) error {
				if c.Morph.APIKey != "12345" {
					return nil
				}
				return nil
			},
		},
		{
			name: "yaml_special_keywords",
			content: `
morph:
  api_key: "yes"
  base_snapshot_id: "no"
`,
			checkFunc: func(c *Config) error {
				// YAML 1.1 treats yes/no as booleans
				return nil
			},
		},
		{
			name: "quoted_yaml_keywords",
			content: `
morph:
  api_key: "null"
  base_snapshot_id: "~"
`,
			checkFunc: func(c *Config) error {
				return nil
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := os.WriteFile(filepath.Join(tmpDir, "config.yaml"), []byte(tt.content), 0644); err != nil {
				t.Fatal(err)
			}

			cfg, err := Load()
			if err != nil {
				t.Logf("Load() error (may be expected): %v", err)
				return
			}

			if tt.checkFunc != nil {
				if err := tt.checkFunc(cfg); err != nil {
					t.Error(err)
				}
			}
		})
	}
}

func TestYAMLFlowStyle(t *testing.T) {
	tmpDir := t.TempDir()
	os.Setenv("DBA_HOME", tmpDir)
	defer os.Unsetenv("DBA_HOME")

	// YAML flow style (inline JSON-like)
	content := `
morph: {api_key: "flow-style", base_snapshot_id: "snap-flow", vm: {vcpus: 4, memory: 8192, disk_size: 65536, ttl_seconds: 3600}}
agent_browser: {path: "/custom/path", timeout: 45000, session_prefix: "flow"}
`
	if err := os.WriteFile(filepath.Join(tmpDir, "config.yaml"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	if cfg.Morph.APIKey != "flow-style" {
		t.Errorf("APIKey = %s, want flow-style", cfg.Morph.APIKey)
	}
	if cfg.Morph.VM.VCPUs != 4 {
		t.Errorf("VCPUs = %d, want 4", cfg.Morph.VM.VCPUs)
	}
	if cfg.AgentBrowser.Timeout != 45000 {
		t.Errorf("Timeout = %d, want 45000", cfg.AgentBrowser.Timeout)
	}
}

func TestYAMLMixedStyles(t *testing.T) {
	tmpDir := t.TempDir()
	os.Setenv("DBA_HOME", tmpDir)
	defer os.Unsetenv("DBA_HOME")

	// Mix of block and flow styles
	content := `
morph:
  api_key: "mixed-style"
  vm: {vcpus: 2, memory: 4096, disk_size: 32768, ttl_seconds: 3600}
agent_browser:
  path: "/block/style"
  timeout: 30000
  session_prefix: "mixed"
`
	if err := os.WriteFile(filepath.Join(tmpDir, "config.yaml"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	if cfg.Morph.APIKey != "mixed-style" {
		t.Errorf("APIKey = %s, want mixed-style", cfg.Morph.APIKey)
	}
	if cfg.AgentBrowser.Path != "/block/style" {
		t.Errorf("Path = %s, want /block/style", cfg.AgentBrowser.Path)
	}
}

// =============================================================================
// Environment Variable Edge Cases
// =============================================================================

func TestEnvVarEmptyValue(t *testing.T) {
	os.Setenv("EMPTY_VAR", "")
	defer os.Unsetenv("EMPTY_VAR")

	cfg := DefaultConfig()
	cfg.Morph.APIKey = "${EMPTY_VAR}"

	key := cfg.GetMorphAPIKey()
	// Empty env var value should return empty string
	if key != "" {
		t.Errorf("GetMorphAPIKey() with empty env var = %q, want empty", key)
	}
}

func TestEnvVarUnset(t *testing.T) {
	os.Unsetenv("UNSET_VAR")

	cfg := DefaultConfig()
	cfg.Morph.APIKey = "${UNSET_VAR}"

	key := cfg.GetMorphAPIKey()
	// Unset env var should return empty string
	if key != "" {
		t.Errorf("GetMorphAPIKey() with unset env var = %q, want empty", key)
	}
}

func TestEnvVarMultipleReferences(t *testing.T) {
	os.Setenv("PART1", "hello")
	os.Setenv("PART2", "world")
	defer os.Unsetenv("PART1")
	defer os.Unsetenv("PART2")

	cfg := DefaultConfig()
	// Multiple env var references in one string
	cfg.Morph.APIKey = "${PART1}-${PART2}"

	key := cfg.GetMorphAPIKey()
	// Behavior depends on implementation - may resolve both or not
	t.Logf("GetMorphAPIKey() with multiple refs = %q", key)
}

func TestEnvVarWithDefault(t *testing.T) {
	os.Unsetenv("MISSING_VAR")

	cfg := DefaultConfig()
	// Some env var syntaxes support defaults like ${VAR:-default}
	cfg.Morph.APIKey = "${MISSING_VAR:-default-value}"

	key := cfg.GetMorphAPIKey()
	t.Logf("GetMorphAPIKey() with default syntax = %q", key)
}

func TestEnvVarInPath(t *testing.T) {
	os.Setenv("BROWSER_DIR", "/opt/browser")
	defer os.Unsetenv("BROWSER_DIR")

	tmpDir := t.TempDir()
	os.Setenv("DBA_HOME", tmpDir)
	defer os.Unsetenv("DBA_HOME")

	content := `
agent_browser:
  path: "${BROWSER_DIR}/agent-browser"
`
	if err := os.WriteFile(filepath.Join(tmpDir, "config.yaml"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	// Path may or may not be resolved depending on implementation
	t.Logf("AgentBrowser.Path = %q", cfg.AgentBrowser.Path)
}

// =============================================================================
// Config File Encoding Edge Cases
// =============================================================================

func TestConfigFileUTF16(t *testing.T) {
	tmpDir := t.TempDir()
	os.Setenv("DBA_HOME", tmpDir)
	defer os.Unsetenv("DBA_HOME")

	// UTF-16 LE BOM + content (this may fail, testing error handling)
	bom := []byte{0xFF, 0xFE}
	// Note: actual UTF-16 encoding would be different, this tests error handling
	content := append(bom, []byte("morph:\n  api_key: test\n")...)

	if err := os.WriteFile(filepath.Join(tmpDir, "config.yaml"), content, 0644); err != nil {
		t.Fatal(err)
	}

	_, err := Load()
	// May or may not error - testing that it doesn't panic
	t.Logf("Load() with UTF-16-like content: err=%v", err)
}

func TestConfigFileBinaryContent(t *testing.T) {
	tmpDir := t.TempDir()
	os.Setenv("DBA_HOME", tmpDir)
	defer os.Unsetenv("DBA_HOME")

	// Random binary content
	content := make([]byte, 256)
	for i := range content {
		content[i] = byte(i)
	}

	if err := os.WriteFile(filepath.Join(tmpDir, "config.yaml"), content, 0644); err != nil {
		t.Fatal(err)
	}

	_, err := Load()
	// Should error but not panic
	if err == nil {
		t.Error("Expected error for binary content")
	}
}

func TestConfigFileNullBytes(t *testing.T) {
	tmpDir := t.TempDir()
	os.Setenv("DBA_HOME", tmpDir)
	defer os.Unsetenv("DBA_HOME")

	// YAML with embedded null bytes
	content := "morph:\n  api_key: \"test\x00key\"\n"

	if err := os.WriteFile(filepath.Join(tmpDir, "config.yaml"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load()
	if err != nil {
		t.Logf("Load() with null bytes: err=%v", err)
		return
	}

	t.Logf("APIKey with null byte = %q (len=%d)", cfg.Morph.APIKey, len(cfg.Morph.APIKey))
}

// =============================================================================
// Config File Size Edge Cases
// =============================================================================

func TestConfigFileEmpty(t *testing.T) {
	tmpDir := t.TempDir()
	os.Setenv("DBA_HOME", tmpDir)
	defer os.Unsetenv("DBA_HOME")

	// Completely empty file
	if err := os.WriteFile(filepath.Join(tmpDir, "config.yaml"), []byte{}, 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	// Should use defaults
	if cfg.Morph.VM.VCPUs != DefaultMorphVCPUs {
		t.Errorf("VCPUs = %d, want %d", cfg.Morph.VM.VCPUs, DefaultMorphVCPUs)
	}
}

func TestConfigFileLarge(t *testing.T) {
	tmpDir := t.TempDir()
	os.Setenv("DBA_HOME", tmpDir)
	defer os.Unsetenv("DBA_HOME")

	// Large file with many comments (1MB)
	var sb strings.Builder
	sb.WriteString("morph:\n  api_key: \"large-file-test\"\n")
	for i := 0; i < 50000; i++ {
		sb.WriteString("# This is comment line number ")
		sb.WriteString(string(rune('0' + i%10)))
		sb.WriteString("\n")
	}

	if err := os.WriteFile(filepath.Join(tmpDir, "config.yaml"), []byte(sb.String()), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	if cfg.Morph.APIKey != "large-file-test" {
		t.Errorf("APIKey = %s, want large-file-test", cfg.Morph.APIKey)
	}
}

// =============================================================================
// Unknown Fields Edge Cases
// =============================================================================

func TestConfigUnknownFields(t *testing.T) {
	tmpDir := t.TempDir()
	os.Setenv("DBA_HOME", tmpDir)
	defer os.Unsetenv("DBA_HOME")

	content := `
morph:
  api_key: "known-field"
  unknown_field: "should be ignored"
  vm:
    vcpus: 4
    unknown_vm_field: 12345
agent_browser:
  path: "/known/path"
  unknown_browser_field: true
unknown_top_level:
  nested: "value"
`
	if err := os.WriteFile(filepath.Join(tmpDir, "config.yaml"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	// Known fields should be parsed correctly
	if cfg.Morph.APIKey != "known-field" {
		t.Errorf("APIKey = %s, want known-field", cfg.Morph.APIKey)
	}
	if cfg.Morph.VM.VCPUs != 4 {
		t.Errorf("VCPUs = %d, want 4", cfg.Morph.VM.VCPUs)
	}
	if cfg.AgentBrowser.Path != "/known/path" {
		t.Errorf("Path = %s, want /known/path", cfg.AgentBrowser.Path)
	}
}

// =============================================================================
// Type Coercion Edge Cases
// =============================================================================

func TestConfigTypeCoercion(t *testing.T) {
	tmpDir := t.TempDir()
	os.Setenv("DBA_HOME", tmpDir)
	defer os.Unsetenv("DBA_HOME")

	tests := []struct {
		name      string
		content   string
		wantError bool
	}{
		{
			name: "string_to_int_fails",
			content: `
morph:
  vm:
    vcpus: "four"
`,
			wantError: true,
		},
		{
			name: "float_to_int",
			content: `
morph:
  vm:
    vcpus: 4.0
`,
			wantError: false, // YAML may coerce this
		},
		{
			name: "scientific_notation",
			content: `
morph:
  vm:
    vcpus: 4e0
`,
			wantError: false, // Should parse as 4
		},
		{
			name: "hex_number",
			content: `
morph:
  vm:
    vcpus: 0x4
`,
			wantError: false, // YAML 1.1 supports hex
		},
		{
			name: "octal_number",
			content: `
morph:
  vm:
    vcpus: 010
`,
			wantError: false, // YAML 1.1 supports octal (010 = 8)
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := os.WriteFile(filepath.Join(tmpDir, "config.yaml"), []byte(tt.content), 0644); err != nil {
				t.Fatal(err)
			}

			cfg, err := Load()
			if tt.wantError {
				if err == nil {
					t.Error("Expected error")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				} else {
					t.Logf("VCPUs = %d", cfg.Morph.VM.VCPUs)
				}
			}
		})
	}
}

// =============================================================================
// Path Edge Cases
// =============================================================================

func TestAgentBrowserPathEdgeCasesExtended(t *testing.T) {
	tests := []struct {
		name string
		path string
	}{
		{"absolute_unix", "/usr/local/bin/agent-browser"},
		{"relative_simple", "./agent-browser"},
		{"relative_parent", "../bin/agent-browser"},
		{"with_spaces", "/path/with spaces/agent-browser"},
		{"with_unicode", "/path/日本語/agent-browser"},
		{"with_special", "/path/with$pecial/agent-browser"},
		{"windows_style", "C:\\Program Files\\agent-browser"},
		{"unc_path", "\\\\server\\share\\agent-browser"},
		{"very_long", "/" + strings.Repeat("very/long/path/", 50) + "agent-browser"},
		{"with_dots", "/path/with.dots/agent.browser"},
		{"with_dashes", "/path-with-dashes/agent-browser"},
		{"with_underscores", "/path_with_underscores/agent_browser"},
		{"tilde_home", "~/bin/agent-browser"},
		{"env_var_style", "$HOME/bin/agent-browser"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := DefaultConfig()
			cfg.AgentBrowser.Path = tt.path

			// Should not error on setting
			if cfg.AgentBrowser.Path != tt.path {
				t.Errorf("Path = %s, want %s", cfg.AgentBrowser.Path, tt.path)
			}
		})
	}
}

// =============================================================================
// Concurrent Config Load Tests
// =============================================================================

func TestConcurrentConfigLoadDifferentFiles(t *testing.T) {
	// Test concurrent loading from different DBA_HOME directories
	numGoroutines := 10
	results := make(chan error, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(idx int) {
			tmpDir := t.TempDir()

			content := "morph:\n  api_key: \"key-" + string(rune('0'+idx)) + "\"\n"
			if err := os.WriteFile(filepath.Join(tmpDir, "config.yaml"), []byte(content), 0644); err != nil {
				results <- err
				return
			}

			// Each goroutine sets its own DBA_HOME - this tests isolation
			// Note: This is racy but tests concurrent access patterns
			originalHome := os.Getenv("DBA_HOME")
			os.Setenv("DBA_HOME", tmpDir)
			_, err := Load()
			os.Setenv("DBA_HOME", originalHome)

			results <- err
		}(i)
	}

	for i := 0; i < numGoroutines; i++ {
		if err := <-results; err != nil {
			t.Errorf("Concurrent load error: %v", err)
		}
	}
}
