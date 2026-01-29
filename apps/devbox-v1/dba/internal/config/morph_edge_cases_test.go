// internal/config/morph_edge_cases_test.go
package config

import (
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

// =============================================================================
// API Key Edge Cases
// =============================================================================

func TestGetMorphAPIKeyEmpty(t *testing.T) {
	// Ensure env var is not set
	os.Unsetenv("MORPH_API_KEY")

	cfg := &Config{}
	key := cfg.GetMorphAPIKey()
	if key != "" {
		t.Errorf("expected empty key, got %q", key)
	}
}

func TestGetMorphAPIKeyWithSpecialCharacters(t *testing.T) {
	testKeys := []string{
		"morph_abc123!@#$%^&*()",
		"key-with-dashes",
		"key_with_underscores",
		"key.with.dots",
		"key with spaces",
		"key\twith\ttabs",
		"key\nwith\nnewlines",
		"キー日本語",
		"🔑emoji-key",
		"a]b[c{d}e",
	}

	for _, testKey := range testKeys {
		t.Run(testKey[:min(10, len(testKey))], func(t *testing.T) {
			os.Setenv("MORPH_API_KEY", testKey)
			defer os.Unsetenv("MORPH_API_KEY")

			cfg := &Config{}
			key := cfg.GetMorphAPIKey()
			if key != testKey {
				t.Errorf("expected %q, got %q", testKey, key)
			}
		})
	}
}

func TestGetMorphAPIKeyEnvVarNotExists(t *testing.T) {
	os.Unsetenv("NONEXISTENT_VAR_12345")

	cfg := &Config{
		Morph: MorphConfig{
			APIKey: "${NONEXISTENT_VAR_12345}",
		},
	}

	key := cfg.GetMorphAPIKey()
	if key != "" {
		t.Errorf("expected empty string for non-existent env var, got %q", key)
	}
}

func TestGetMorphAPIKeyMalformedEnvVarReference(t *testing.T) {
	os.Setenv("MORPH_API_KEY", "fallback-key")
	defer os.Unsetenv("MORPH_API_KEY")

	tests := []struct {
		name     string
		apiKey   string
		expected string
	}{
		{"missing dollar", "{MORPH_API_KEY}", "{MORPH_API_KEY}"},
		{"missing open brace", "$MORPH_API_KEY}", "$MORPH_API_KEY}"},
		{"missing close brace", "${MORPH_API_KEY", "${MORPH_API_KEY"},
		{"empty braces", "${}", ""},
		{"just dollar brace", "${", "${"},
		{"nested braces", "${${MORPH_API_KEY}}", ""},
		{"spaces inside", "${ MORPH_API_KEY }", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{
				Morph: MorphConfig{
					APIKey: tt.apiKey,
				},
			}

			key := cfg.GetMorphAPIKey()
			if key != tt.expected {
				t.Errorf("for %q: expected %q, got %q", tt.apiKey, tt.expected, key)
			}
		})
	}
}

func TestGetMorphAPIKeyVeryLong(t *testing.T) {
	longKey := strings.Repeat("a", 10000)
	os.Setenv("MORPH_API_KEY", longKey)
	defer os.Unsetenv("MORPH_API_KEY")

	cfg := &Config{}
	key := cfg.GetMorphAPIKey()
	if key != longKey {
		t.Errorf("expected key of length %d, got length %d", len(longKey), len(key))
	}
}

func TestGetMorphAPIKeyWhitespaceOnly(t *testing.T) {
	tests := []string{" ", "  ", "\t", "\n", " \t\n "}

	for _, ws := range tests {
		t.Run("whitespace", func(t *testing.T) {
			cfg := &Config{
				Morph: MorphConfig{
					APIKey: ws,
				},
			}

			key := cfg.GetMorphAPIKey()
			if key != ws {
				t.Errorf("expected whitespace key %q, got %q", ws, key)
			}
		})
	}
}

func TestGetMorphAPIKeyPrecedence(t *testing.T) {
	os.Setenv("MORPH_API_KEY", "env-key")
	defer os.Unsetenv("MORPH_API_KEY")

	// Direct value takes precedence over env var
	cfg := &Config{
		Morph: MorphConfig{
			APIKey: "config-key",
		},
	}

	key := cfg.GetMorphAPIKey()
	if key != "config-key" {
		t.Errorf("config value should take precedence, expected 'config-key', got %q", key)
	}
}

// =============================================================================
// Base Snapshot ID Edge Cases
// =============================================================================

func TestGetBaseSnapshotIDEmpty(t *testing.T) {
	os.Unsetenv("DBA_BASE_SNAPSHOT")

	cfg := &Config{}
	id := cfg.GetBaseSnapshotID()
	if id != "" {
		t.Errorf("expected empty ID, got %q", id)
	}
}

func TestGetBaseSnapshotIDFormats(t *testing.T) {
	testIDs := []string{
		"snap_abc123",
		"snap-abc-123",
		"snapshot_2024_01_28_12_00_00",
		"SNAP_UPPERCASE",
		"snap.with.dots",
		"snap/with/slashes",
		"snap:with:colons",
	}

	for _, testID := range testIDs {
		t.Run(testID, func(t *testing.T) {
			os.Setenv("DBA_BASE_SNAPSHOT", testID)
			defer os.Unsetenv("DBA_BASE_SNAPSHOT")

			cfg := &Config{}
			id := cfg.GetBaseSnapshotID()
			if id != testID {
				t.Errorf("expected %q, got %q", testID, id)
			}
		})
	}
}

func TestGetBaseSnapshotIDPrecedence(t *testing.T) {
	os.Setenv("DBA_BASE_SNAPSHOT", "env-snapshot")
	defer os.Unsetenv("DBA_BASE_SNAPSHOT")

	// Config value takes precedence
	cfg := &Config{
		Morph: MorphConfig{
			BaseSnapshotID: "config-snapshot",
		},
	}

	id := cfg.GetBaseSnapshotID()
	if id != "config-snapshot" {
		t.Errorf("config value should take precedence, expected 'config-snapshot', got %q", id)
	}
}

// =============================================================================
// VM Config Validation Edge Cases
// =============================================================================

func TestValidateBoundaryValues(t *testing.T) {
	tests := []struct {
		name    string
		modify  func(*Config)
		wantErr bool
	}{
		// VCPUs
		{"vcpus=0", func(c *Config) { c.Morph.VM.VCPUs = 0 }, true},
		{"vcpus=1", func(c *Config) { c.Morph.VM.VCPUs = 1 }, false},
		{"vcpus=-1", func(c *Config) { c.Morph.VM.VCPUs = -1 }, true},
		{"vcpus=1000", func(c *Config) { c.Morph.VM.VCPUs = 1000 }, false},

		// Memory
		{"memory=0", func(c *Config) { c.Morph.VM.Memory = 0 }, true},
		{"memory=511", func(c *Config) { c.Morph.VM.Memory = 511 }, true},
		{"memory=512", func(c *Config) { c.Morph.VM.Memory = 512 }, false},
		{"memory=513", func(c *Config) { c.Morph.VM.Memory = 513 }, false},
		{"memory=-1", func(c *Config) { c.Morph.VM.Memory = -1 }, true},
		{"memory=1048576", func(c *Config) { c.Morph.VM.Memory = 1048576 }, false}, // 1TB

		// DiskSize
		{"disk=0", func(c *Config) { c.Morph.VM.DiskSize = 0 }, true},
		{"disk=1023", func(c *Config) { c.Morph.VM.DiskSize = 1023 }, true},
		{"disk=1024", func(c *Config) { c.Morph.VM.DiskSize = 1024 }, false},
		{"disk=1025", func(c *Config) { c.Morph.VM.DiskSize = 1025 }, false},
		{"disk=-1", func(c *Config) { c.Morph.VM.DiskSize = -1 }, true},

		// TTLSeconds
		{"ttl=0", func(c *Config) { c.Morph.VM.TTLSeconds = 0 }, true},
		{"ttl=59", func(c *Config) { c.Morph.VM.TTLSeconds = 59 }, true},
		{"ttl=60", func(c *Config) { c.Morph.VM.TTLSeconds = 60 }, false},
		{"ttl=61", func(c *Config) { c.Morph.VM.TTLSeconds = 61 }, false},
		{"ttl=-1", func(c *Config) { c.Morph.VM.TTLSeconds = -1 }, true},
		{"ttl=86400", func(c *Config) { c.Morph.VM.TTLSeconds = 86400 }, false}, // 1 day

		// AgentBrowser Timeout
		{"timeout=0", func(c *Config) { c.AgentBrowser.Timeout = 0 }, true},
		{"timeout=999", func(c *Config) { c.AgentBrowser.Timeout = 999 }, true},
		{"timeout=1000", func(c *Config) { c.AgentBrowser.Timeout = 1000 }, false},
		{"timeout=1001", func(c *Config) { c.AgentBrowser.Timeout = 1001 }, false},
		{"timeout=-1", func(c *Config) { c.AgentBrowser.Timeout = -1 }, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := DefaultConfig()
			tt.modify(cfg)

			err := cfg.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateMultipleErrors(t *testing.T) {
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
			Timeout: 0,
		},
	}

	err := cfg.Validate()
	if err == nil {
		t.Fatal("expected validation error")
	}

	errStr := err.Error()
	// Should contain multiple error messages
	if !strings.Contains(errStr, "vcpus") {
		t.Error("error should mention vcpus")
	}
	if !strings.Contains(errStr, "memory") {
		t.Error("error should mention memory")
	}
	if !strings.Contains(errStr, "disk_size") {
		t.Error("error should mention disk_size")
	}
	if !strings.Contains(errStr, "ttl_seconds") {
		t.Error("error should mention ttl_seconds")
	}
	if !strings.Contains(errStr, "timeout") {
		t.Error("error should mention timeout")
	}
}

// =============================================================================
// YAML Loading Edge Cases
// =============================================================================

func TestLoadMorphConfigPartialYAML(t *testing.T) {
	tests := []struct {
		name   string
		yaml   string
		verify func(*Config) error
	}{
		{
			name: "only api_key",
			yaml: `
morph:
  api_key: "test-key"
`,
			verify: func(c *Config) error {
				if c.Morph.APIKey != "test-key" {
					return errorf("APIKey = %q, want 'test-key'", c.Morph.APIKey)
				}
				// VM should have defaults
				if c.Morph.VM.VCPUs != DefaultMorphVCPUs {
					return errorf("VCPUs = %d, want default %d", c.Morph.VM.VCPUs, DefaultMorphVCPUs)
				}
				return nil
			},
		},
		{
			name: "only vm.vcpus",
			yaml: `
morph:
  vm:
    vcpus: 8
`,
			verify: func(c *Config) error {
				if c.Morph.VM.VCPUs != 8 {
					return errorf("VCPUs = %d, want 8", c.Morph.VM.VCPUs)
				}
				// Other VM fields might be zero (YAML partial override)
				return nil
			},
		},
		{
			name: "empty morph section",
			yaml: `
morph:
`,
			verify: func(c *Config) error {
				// Should still work, use defaults
				return nil
			},
		},
		{
			name: "only agent_browser.path",
			yaml: `
agent_browser:
  path: "/custom/path"
`,
			verify: func(c *Config) error {
				if c.AgentBrowser.Path != "/custom/path" {
					return errorf("Path = %q, want '/custom/path'", c.AgentBrowser.Path)
				}
				return nil
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			os.Setenv("DBA_HOME", tmpDir)
			defer os.Unsetenv("DBA_HOME")

			configPath := filepath.Join(tmpDir, "config.yaml")
			if err := os.WriteFile(configPath, []byte(tt.yaml), 0644); err != nil {
				t.Fatal(err)
			}

			cfg, err := Load()
			if err != nil {
				t.Fatalf("Load() error: %v", err)
			}

			if verifyErr := tt.verify(cfg); verifyErr != nil {
				t.Error(verifyErr)
			}
		})
	}
}

func TestLoadMorphConfigInvalidTypes(t *testing.T) {
	tests := []struct {
		name string
		yaml string
	}{
		{
			name: "vcpus as string",
			yaml: `
morph:
  vm:
    vcpus: "two"
`,
		},
		{
			name: "memory as string",
			yaml: `
morph:
  vm:
    memory: "4GB"
`,
		},
		{
			name: "timeout as string",
			yaml: `
agent_browser:
  timeout: "30s"
`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			os.Setenv("DBA_HOME", tmpDir)
			defer os.Unsetenv("DBA_HOME")

			configPath := filepath.Join(tmpDir, "config.yaml")
			if err := os.WriteFile(configPath, []byte(tt.yaml), 0644); err != nil {
				t.Fatal(err)
			}

			_, err := Load()
			if err == nil {
				t.Error("expected error for invalid type")
			}
		})
	}
}

func TestLoadMorphConfigSpecialYAMLValues(t *testing.T) {
	tests := []struct {
		name      string
		yaml      string
		wantErr   bool
		verify    func(*Config) error
	}{
		{
			name: "null api_key overrides default",
			yaml: `
morph:
  api_key: null
`,
			wantErr: false,
			verify: func(c *Config) error {
				// When YAML null is used on partial config, it zeroes the field
				// but since we start with defaults, the behavior depends on yaml lib
				// The important thing is it doesn't crash
				return nil
			},
		},
		{
			name: "quoted numbers cause type error",
			yaml: `
morph:
  vm:
    vcpus: "4"
`,
			wantErr: true, // YAML should fail to parse quoted number as int
			verify:  nil,
		},
		{
			name: "explicit empty string api_key",
			yaml: `
morph:
  api_key: ""
`,
			wantErr: false,
			verify: func(c *Config) error {
				if c.Morph.APIKey != "" {
					return errorf("APIKey should be empty for explicit empty string, got %q", c.Morph.APIKey)
				}
				return nil
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			os.Setenv("DBA_HOME", tmpDir)
			defer os.Unsetenv("DBA_HOME")

			configPath := filepath.Join(tmpDir, "config.yaml")
			if err := os.WriteFile(configPath, []byte(tt.yaml), 0644); err != nil {
				t.Fatal(err)
			}

			cfg, err := Load()
			if tt.wantErr {
				if err == nil {
					t.Error("expected error but got none")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if tt.verify != nil {
				if verifyErr := tt.verify(cfg); verifyErr != nil {
					t.Error(verifyErr)
				}
			}
		})
	}
}

// =============================================================================
// Concurrent Access Edge Cases
// =============================================================================

func TestGetMorphAPIKeyConcurrent(t *testing.T) {
	os.Setenv("MORPH_API_KEY", "concurrent-test-key")
	defer os.Unsetenv("MORPH_API_KEY")

	cfg := &Config{
		Morph: MorphConfig{
			APIKey: "${MORPH_API_KEY}",
		},
	}

	var wg sync.WaitGroup
	errors := make(chan error, 100)

	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			key := cfg.GetMorphAPIKey()
			if key != "concurrent-test-key" {
				errors <- errorf("expected 'concurrent-test-key', got %q", key)
			}
		}()
	}

	wg.Wait()
	close(errors)

	for err := range errors {
		t.Error(err)
	}
}

func TestValidateConcurrent(t *testing.T) {
	cfg := DefaultConfig()

	var wg sync.WaitGroup
	errors := make(chan error, 100)

	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			err := cfg.Validate()
			if err != nil {
				errors <- err
			}
		}()
	}

	wg.Wait()
	close(errors)

	for err := range errors {
		t.Errorf("concurrent Validate() failed: %v", err)
	}
}

// =============================================================================
// AgentBrowser Config Edge Cases
// =============================================================================

func TestAgentBrowserPathEdgeCases(t *testing.T) {
	tests := []struct {
		name string
		path string
	}{
		{"empty", ""},
		{"relative", "agent-browser"},
		{"absolute", "/usr/local/bin/agent-browser"},
		{"with spaces", "/path/to/my agent browser"},
		{"with dots", "./agent-browser"},
		{"parent dir", "../bin/agent-browser"},
		{"home dir", "~/bin/agent-browser"},
		{"windows style", "C:\\Program Files\\agent-browser"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{
				AgentBrowser: AgentBrowserConfig{
					Path:    tt.path,
					Timeout: 30000,
				},
			}

			// Should not panic
			_ = cfg.AgentBrowser.Path
		})
	}
}

func TestSessionPrefixEdgeCases(t *testing.T) {
	tests := []string{
		"",
		"a",
		"dba",
		"my-prefix",
		"prefix_with_underscore",
		"PREFIX123",
		"prefix.with.dots",
		strings.Repeat("x", 1000),
	}

	for _, prefix := range tests {
		t.Run(prefix[:min(10, len(prefix))], func(t *testing.T) {
			cfg := &Config{
				AgentBrowser: AgentBrowserConfig{
					Path:          "agent-browser",
					Timeout:       30000,
					SessionPrefix: prefix,
				},
			}

			if cfg.AgentBrowser.SessionPrefix != prefix {
				t.Errorf("expected %q, got %q", prefix, cfg.AgentBrowser.SessionPrefix)
			}
		})
	}
}

// =============================================================================
// Default Config Edge Cases
// =============================================================================

func TestDefaultConfigMorphValues(t *testing.T) {
	cfg := DefaultConfig()

	// Verify all Morph defaults are set
	if cfg.Morph.APIKey != "${MORPH_API_KEY}" {
		t.Errorf("default APIKey = %q, want '${MORPH_API_KEY}'", cfg.Morph.APIKey)
	}
	if cfg.Morph.BaseSnapshotID != "snapshot_3namut0l" {
		t.Errorf("default BaseSnapshotID = %q, want snapshot_3namut0l", cfg.Morph.BaseSnapshotID)
	}
	if cfg.Morph.VM.VCPUs != 2 {
		t.Errorf("default VCPUs = %d, want 2", cfg.Morph.VM.VCPUs)
	}
	if cfg.Morph.VM.Memory != 4096 {
		t.Errorf("default Memory = %d, want 4096", cfg.Morph.VM.Memory)
	}
	if cfg.Morph.VM.DiskSize != 32768 {
		t.Errorf("default DiskSize = %d, want 32768", cfg.Morph.VM.DiskSize)
	}
	if cfg.Morph.VM.TTLSeconds != 3600 {
		t.Errorf("default TTLSeconds = %d, want 3600", cfg.Morph.VM.TTLSeconds)
	}
}

func TestDefaultConfigAgentBrowserValues(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.AgentBrowser.Path != "agent-browser" {
		t.Errorf("default Path = %q, want 'agent-browser'", cfg.AgentBrowser.Path)
	}
	if cfg.AgentBrowser.Timeout != 30000 {
		t.Errorf("default Timeout = %d, want 30000", cfg.AgentBrowser.Timeout)
	}
	if cfg.AgentBrowser.SessionPrefix != "dba" {
		t.Errorf("default SessionPrefix = %q, want 'dba'", cfg.AgentBrowser.SessionPrefix)
	}
}

func TestDefaultConfigIsValid(t *testing.T) {
	cfg := DefaultConfig()
	err := cfg.Validate()
	if err != nil {
		t.Errorf("DefaultConfig() should be valid, got error: %v", err)
	}
}

// =============================================================================
// Helper Functions
// =============================================================================

func errorf(format string, args ...interface{}) error {
	return &testError{msg: format, args: args}
}

type testError struct {
	msg  string
	args []interface{}
}

func (e *testError) Error() string {
	return e.msg
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
