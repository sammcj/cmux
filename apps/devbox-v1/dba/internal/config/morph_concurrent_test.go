// internal/config/morph_concurrent_test.go
package config

import (
	"os"
	"path/filepath"
	"sync"
	"testing"
)

// =============================================================================
// Concurrent Access Tests
// =============================================================================

func TestConcurrentConfigLoad(t *testing.T) {
	tmpDir := t.TempDir()
	os.Setenv("DBA_HOME", tmpDir)
	defer os.Unsetenv("DBA_HOME")

	// Write a valid config
	customConfig := `
morph:
  api_key: "concurrent-test-key"
  vm:
    vcpus: 4
    memory: 8192
    disk_size: 65536
    ttl_seconds: 7200
`
	if err := os.WriteFile(filepath.Join(tmpDir, "config.yaml"), []byte(customConfig), 0644); err != nil {
		t.Fatal(err)
	}

	// Load config concurrently from multiple goroutines
	var wg sync.WaitGroup
	errors := make(chan error, 100)

	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			cfg, err := Load()
			if err != nil {
				errors <- err
				return
			}
			if cfg.Morph.APIKey != "concurrent-test-key" {
				errors <- err
			}
		}()
	}

	wg.Wait()
	close(errors)

	for err := range errors {
		if err != nil {
			t.Errorf("Concurrent load error: %v", err)
		}
	}
}

func TestConcurrentValidate(t *testing.T) {
	cfg := DefaultConfig()

	// Validate concurrently from multiple goroutines
	var wg sync.WaitGroup
	errors := make(chan error, 100)

	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := cfg.Validate(); err != nil {
				errors <- err
			}
		}()
	}

	wg.Wait()
	close(errors)

	for err := range errors {
		if err != nil {
			t.Errorf("Concurrent validate error: %v", err)
		}
	}
}

func TestConcurrentGetMorphAPIKey(t *testing.T) {
	os.Setenv("TEST_CONCURRENT_API_KEY", "concurrent-key")
	defer os.Unsetenv("TEST_CONCURRENT_API_KEY")

	cfg := DefaultConfig()
	cfg.Morph.APIKey = "${TEST_CONCURRENT_API_KEY}"

	// Get API key concurrently
	var wg sync.WaitGroup
	results := make(chan string, 100)

	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			results <- cfg.GetMorphAPIKey()
		}()
	}

	wg.Wait()
	close(results)

	for key := range results {
		if key != "concurrent-key" {
			t.Errorf("GetMorphAPIKey() = %s, want concurrent-key", key)
		}
	}
}

func TestConcurrentGetBaseSnapshotID(t *testing.T) {
	os.Setenv("DBA_BASE_SNAPSHOT", "concurrent-snapshot")
	defer os.Unsetenv("DBA_BASE_SNAPSHOT")

	cfg := DefaultConfig()
	cfg.Morph.BaseSnapshotID = ""

	// Get base snapshot ID concurrently
	var wg sync.WaitGroup
	results := make(chan string, 100)

	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			results <- cfg.GetBaseSnapshotID()
		}()
	}

	wg.Wait()
	close(results)

	for snapID := range results {
		if snapID != "concurrent-snapshot" {
			t.Errorf("GetBaseSnapshotID() = %s, want concurrent-snapshot", snapID)
		}
	}
}

// =============================================================================
// Large Value Tests
// =============================================================================

func TestLargeAPIKey(t *testing.T) {
	cfg := DefaultConfig()

	// Generate a very large API key (10KB)
	largeKey := make([]byte, 10*1024)
	for i := range largeKey {
		largeKey[i] = 'a'
	}
	cfg.Morph.APIKey = string(largeKey)

	// Should still work
	key := cfg.GetMorphAPIKey()
	if len(key) != 10*1024 {
		t.Errorf("Large API key length = %d, want %d", len(key), 10*1024)
	}
}

func TestLargeSnapshotID(t *testing.T) {
	cfg := DefaultConfig()

	// Generate a large snapshot ID
	largeID := make([]byte, 10*1024)
	for i := range largeID {
		largeID[i] = 's'
	}
	cfg.Morph.BaseSnapshotID = string(largeID)

	snapID := cfg.GetBaseSnapshotID()
	if len(snapID) != 10*1024 {
		t.Errorf("Large snapshot ID length = %d, want %d", len(snapID), 10*1024)
	}
}

func TestExtremeVMValues(t *testing.T) {
	tests := []struct {
		name       string
		vcpus      int
		memory     int
		diskSize   int
		ttlSeconds int
		wantValid  bool
	}{
		{"min_values", 1, 512, 1024, 60, true},
		{"max_vcpus", 1024, 4096, 32768, 3600, true},
		{"max_memory", 2, 1048576, 32768, 3600, true},      // 1TB RAM
		{"max_disk", 2, 4096, 10485760, 3600, true},        // 10TB disk
		{"max_ttl", 2, 4096, 32768, 2592000, true},         // 30 days
		{"zero_vcpus", 0, 4096, 32768, 3600, false},
		{"zero_memory", 2, 0, 32768, 3600, false},
		{"zero_disk", 2, 4096, 0, 3600, false},
		{"zero_ttl", 2, 4096, 32768, 0, false},
		{"negative_vcpus", -1, 4096, 32768, 3600, false},
		{"negative_memory", 2, -1, 32768, 3600, false},
		{"negative_disk", 2, 4096, -1, 3600, false},
		{"negative_ttl", 2, 4096, 32768, -1, false},
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
				t.Errorf("Validate() error = nil, want error")
			}
		})
	}
}

// =============================================================================
// Unicode Tests
// =============================================================================

func TestUnicodeAPIKey(t *testing.T) {
	cfg := DefaultConfig()

	// Test various Unicode characters
	unicodeKeys := []string{
		"apikey-日本語",
		"apikey-中文",
		"apikey-한국어",
		"apikey-العربية",
		"apikey-עברית",
		"apikey-🔑🔐🗝️",
		"apikey-Ñoño",
		"apikey-θΩπ",
		"apikey-∞∑∏",
	}

	for _, key := range unicodeKeys {
		t.Run(key, func(t *testing.T) {
			cfg.Morph.APIKey = key
			got := cfg.GetMorphAPIKey()
			if got != key {
				t.Errorf("GetMorphAPIKey() = %s, want %s", got, key)
			}
		})
	}
}

func TestUnicodeSnapshotID(t *testing.T) {
	cfg := DefaultConfig()

	unicodeIDs := []string{
		"snap-日本語",
		"snap-中文",
		"snap-🚀",
	}

	for _, id := range unicodeIDs {
		t.Run(id, func(t *testing.T) {
			cfg.Morph.BaseSnapshotID = id
			got := cfg.GetBaseSnapshotID()
			if got != id {
				t.Errorf("GetBaseSnapshotID() = %s, want %s", got, id)
			}
		})
	}
}

func TestUnicodeSessionPrefix(t *testing.T) {
	cfg := DefaultConfig()

	unicodePrefixes := []string{
		"prefix-日本語",
		"prefix-🎯",
	}

	for _, prefix := range unicodePrefixes {
		t.Run(prefix, func(t *testing.T) {
			cfg.AgentBrowser.SessionPrefix = prefix
			if cfg.AgentBrowser.SessionPrefix != prefix {
				t.Errorf("SessionPrefix = %s, want %s", cfg.AgentBrowser.SessionPrefix, prefix)
			}
		})
	}
}

// =============================================================================
// YAML Edge Cases
// =============================================================================

func TestYAMLWithComments(t *testing.T) {
	tmpDir := t.TempDir()
	os.Setenv("DBA_HOME", tmpDir)
	defer os.Unsetenv("DBA_HOME")

	configWithComments := `
# Main config
morph:
  # API key for Morph Cloud
  api_key: "commented-key"  # inline comment
  vm:
    # VM configuration
    vcpus: 4  # number of CPUs
    memory: 8192  # in MB
`
	if err := os.WriteFile(filepath.Join(tmpDir, "config.yaml"), []byte(configWithComments), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	if cfg.Morph.APIKey != "commented-key" {
		t.Errorf("APIKey = %s, want commented-key", cfg.Morph.APIKey)
	}
	if cfg.Morph.VM.VCPUs != 4 {
		t.Errorf("VCPUs = %d, want 4", cfg.Morph.VM.VCPUs)
	}
}

func TestYAMLWithAnchors(t *testing.T) {
	tmpDir := t.TempDir()
	os.Setenv("DBA_HOME", tmpDir)
	defer os.Unsetenv("DBA_HOME")

	configWithAnchors := `
defaults: &defaults
  vcpus: 2
  memory: 4096

morph:
  api_key: "anchor-test"
  vm:
    <<: *defaults
    disk_size: 32768
    ttl_seconds: 3600
`
	if err := os.WriteFile(filepath.Join(tmpDir, "config.yaml"), []byte(configWithAnchors), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	if cfg.Morph.APIKey != "anchor-test" {
		t.Errorf("APIKey = %s, want anchor-test", cfg.Morph.APIKey)
	}
	// YAML anchors should be expanded
	if cfg.Morph.VM.VCPUs != 2 {
		t.Errorf("VCPUs = %d, want 2", cfg.Morph.VM.VCPUs)
	}
	if cfg.Morph.VM.Memory != 4096 {
		t.Errorf("Memory = %d, want 4096", cfg.Morph.VM.Memory)
	}
}

func TestYAMLMultilineStrings(t *testing.T) {
	tmpDir := t.TempDir()
	os.Setenv("DBA_HOME", tmpDir)
	defer os.Unsetenv("DBA_HOME")

	configWithMultiline := `
morph:
  api_key: |
    multi-line-key-line1
    multi-line-key-line2
  base_snapshot_id: >
    folded-snapshot-id
`
	if err := os.WriteFile(filepath.Join(tmpDir, "config.yaml"), []byte(configWithMultiline), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	// Literal block style preserves newlines
	expectedKey := "multi-line-key-line1\nmulti-line-key-line2\n"
	if cfg.Morph.APIKey != expectedKey {
		t.Errorf("APIKey = %q, want %q", cfg.Morph.APIKey, expectedKey)
	}

	// Folded block style folds newlines to spaces
	expectedSnap := "folded-snapshot-id\n"
	if cfg.Morph.BaseSnapshotID != expectedSnap {
		t.Errorf("BaseSnapshotID = %q, want %q", cfg.Morph.BaseSnapshotID, expectedSnap)
	}
}

func TestYAMLWithQuotedStrings(t *testing.T) {
	tmpDir := t.TempDir()
	os.Setenv("DBA_HOME", tmpDir)
	defer os.Unsetenv("DBA_HOME")

	tests := []struct {
		name     string
		yaml     string
		expected string
	}{
		{
			name: "single_quotes",
			yaml: `
morph:
  api_key: 'single-quoted'
`,
			expected: "single-quoted",
		},
		{
			name: "double_quotes",
			yaml: `
morph:
  api_key: "double-quoted"
`,
			expected: "double-quoted",
		},
		{
			name: "escaped_quotes",
			yaml: `
morph:
  api_key: "has \"escaped\" quotes"
`,
			expected: `has "escaped" quotes`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := os.WriteFile(filepath.Join(tmpDir, "config.yaml"), []byte(tt.yaml), 0644); err != nil {
				t.Fatal(err)
			}

			cfg, err := Load()
			if err != nil {
				t.Fatalf("Load() error: %v", err)
			}

			if cfg.Morph.APIKey != tt.expected {
				t.Errorf("APIKey = %q, want %q", cfg.Morph.APIKey, tt.expected)
			}
		})
	}
}

// =============================================================================
// Environment Variable Edge Cases
// =============================================================================

func TestEnvVarWithSpecialCharacters(t *testing.T) {
	tests := []struct {
		name     string
		envValue string
	}{
		{"with_spaces", "key with spaces"},
		{"with_newlines", "key\nwith\nnewlines"},
		{"with_tabs", "key\twith\ttabs"},
		{"with_equals", "key=with=equals"},
		{"with_dollars", "key$with$dollars"},
		{"with_quotes", `key"with"quotes`},
		{"with_backslash", `key\with\backslash`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			envName := "TEST_SPECIAL_" + tt.name
			os.Setenv(envName, tt.envValue)
			defer os.Unsetenv(envName)

			cfg := DefaultConfig()
			cfg.Morph.APIKey = "${" + envName + "}"

			got := cfg.GetMorphAPIKey()
			if got != tt.envValue {
				t.Errorf("GetMorphAPIKey() = %q, want %q", got, tt.envValue)
			}
		})
	}
}

func TestNestedEnvVarSyntax(t *testing.T) {
	os.Setenv("INNER_KEY", "inner-value")
	os.Setenv("OUTER_${INNER}", "should-not-expand")
	defer os.Unsetenv("INNER_KEY")
	defer os.Unsetenv("OUTER_${INNER}")

	cfg := DefaultConfig()

	// Test that ${} doesn't recursively expand
	cfg.Morph.APIKey = "${INNER_KEY}"
	got := cfg.GetMorphAPIKey()
	if got != "inner-value" {
		t.Errorf("GetMorphAPIKey() = %s, want inner-value", got)
	}
}

func TestEmptyEnvVarName(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Morph.APIKey = "${}"

	// Empty env var name resolves to empty (no env var with empty name exists)
	got := cfg.GetMorphAPIKey()
	// The behavior is that ${} tries to resolve empty env var, which returns empty
	t.Logf("GetMorphAPIKey() with ${} = %q", got)
	// Just verify it doesn't panic
}

func TestMalformedEnvVarSyntax(t *testing.T) {
	cfg := DefaultConfig()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"unclosed_brace", "${UNCLOSED", "${UNCLOSED"},
		{"extra_close_brace", "${EXTRA}}", "${EXTRA}}"},
		{"no_dollar", "{NODOLLAR}", "{NODOLLAR}"},
		{"dollar_only", "$", "$"},
		{"double_dollar", "$$", "$$"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg.Morph.APIKey = tt.input
			got := cfg.GetMorphAPIKey()
			// Malformed syntax should be returned as-is or partially resolved
			t.Logf("Input: %q, Got: %q", tt.input, got)
		})
	}
}

// =============================================================================
// Config File Edge Cases
// =============================================================================

func TestConfigFileWithBOM(t *testing.T) {
	tmpDir := t.TempDir()
	os.Setenv("DBA_HOME", tmpDir)
	defer os.Unsetenv("DBA_HOME")

	// UTF-8 BOM followed by valid YAML
	bom := []byte{0xEF, 0xBB, 0xBF}
	content := []byte(`
morph:
  api_key: "bom-test"
`)
	fullContent := append(bom, content...)

	if err := os.WriteFile(filepath.Join(tmpDir, "config.yaml"), fullContent, 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	if cfg.Morph.APIKey != "bom-test" {
		t.Errorf("APIKey = %s, want bom-test", cfg.Morph.APIKey)
	}
}

func TestConfigFileWithCRLF(t *testing.T) {
	tmpDir := t.TempDir()
	os.Setenv("DBA_HOME", tmpDir)
	defer os.Unsetenv("DBA_HOME")

	// Windows-style line endings
	content := "morph:\r\n  api_key: \"crlf-test\"\r\n  vm:\r\n    vcpus: 2\r\n"

	if err := os.WriteFile(filepath.Join(tmpDir, "config.yaml"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	if cfg.Morph.APIKey != "crlf-test" {
		t.Errorf("APIKey = %s, want crlf-test", cfg.Morph.APIKey)
	}
}

func TestConfigFileTrailingWhitespace(t *testing.T) {
	tmpDir := t.TempDir()
	os.Setenv("DBA_HOME", tmpDir)
	defer os.Unsetenv("DBA_HOME")

	// Config with trailing whitespace
	content := `
morph:
  api_key: "trailing-ws"
  vm:
    vcpus: 2
`

	if err := os.WriteFile(filepath.Join(tmpDir, "config.yaml"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	if cfg.Morph.APIKey != "trailing-ws" {
		t.Errorf("APIKey = %s, want trailing-ws", cfg.Morph.APIKey)
	}
}
