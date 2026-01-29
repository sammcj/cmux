// internal/config/morph_encoding_test.go
package config

import (
	"os"
	"path/filepath"
	"testing"
)

// =============================================================================
// File Encoding Edge Cases
// =============================================================================

func TestYAMLWithBOM(t *testing.T) {
	tmpDir := t.TempDir()
	os.Setenv("DBA_HOME", tmpDir)
	defer os.Unsetenv("DBA_HOME")

	// UTF-8 BOM followed by valid YAML
	bom := []byte{0xEF, 0xBB, 0xBF}
	content := append(bom, []byte(`
morph:
  api_key: "bom-test"
  vm:
    vcpus: 2
`)...)

	if err := os.WriteFile(filepath.Join(tmpDir, "config.yaml"), content, 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load()
	if err != nil {
		t.Logf("Load with BOM error: %v (may be expected)", err)
		return
	}

	t.Logf("Config loaded with BOM: APIKey=%s", cfg.Morph.APIKey)
}

func TestYAMLWithWindowsLineEndings(t *testing.T) {
	tmpDir := t.TempDir()
	os.Setenv("DBA_HOME", tmpDir)
	defer os.Unsetenv("DBA_HOME")

	// CRLF line endings
	content := "morph:\r\n  api_key: \"windows-test\"\r\n  vm:\r\n    vcpus: 2\r\n"

	if err := os.WriteFile(filepath.Join(tmpDir, "config.yaml"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load with CRLF error: %v", err)
	}

	if cfg.Morph.APIKey != "windows-test" {
		t.Errorf("APIKey = %s, want windows-test", cfg.Morph.APIKey)
	}
}

func TestYAMLWithMacLineEndings(t *testing.T) {
	tmpDir := t.TempDir()
	os.Setenv("DBA_HOME", tmpDir)
	defer os.Unsetenv("DBA_HOME")

	// Old Mac CR line endings (unlikely but possible)
	content := "morph:\r  api_key: \"mac-test\"\r  vm:\r    vcpus: 2\r"

	if err := os.WriteFile(filepath.Join(tmpDir, "config.yaml"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load()
	if err != nil {
		t.Logf("Load with CR-only line endings error: %v (may be expected)", err)
		return
	}

	t.Logf("Config loaded with CR: APIKey=%s", cfg.Morph.APIKey)
}

func TestYAMLWithMixedLineEndings(t *testing.T) {
	tmpDir := t.TempDir()
	os.Setenv("DBA_HOME", tmpDir)
	defer os.Unsetenv("DBA_HOME")

	// Mixed line endings
	content := "morph:\n  api_key: \"mixed-test\"\r\n  vm:\r    vcpus: 2\n"

	if err := os.WriteFile(filepath.Join(tmpDir, "config.yaml"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load()
	if err != nil {
		t.Logf("Load with mixed line endings error: %v (may be expected)", err)
		return
	}

	t.Logf("Config loaded with mixed endings: APIKey=%s", cfg.Morph.APIKey)
}

// =============================================================================
// Unicode Edge Cases
// =============================================================================

func TestYAMLWithUnicodeAPIKey(t *testing.T) {
	tmpDir := t.TempDir()
	os.Setenv("DBA_HOME", tmpDir)
	defer os.Unsetenv("DBA_HOME")

	content := `
morph:
  api_key: "日本語のAPIキー"
  vm:
    vcpus: 2
`
	if err := os.WriteFile(filepath.Join(tmpDir, "config.yaml"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load error: %v", err)
	}

	if cfg.Morph.APIKey != "日本語のAPIキー" {
		t.Errorf("APIKey = %s, want 日本語のAPIキー", cfg.Morph.APIKey)
	}
}

func TestYAMLWithEmojiAPIKey(t *testing.T) {
	tmpDir := t.TempDir()
	os.Setenv("DBA_HOME", tmpDir)
	defer os.Unsetenv("DBA_HOME")

	content := `
morph:
  api_key: "🔑secret-key🔐"
  vm:
    vcpus: 2
`
	if err := os.WriteFile(filepath.Join(tmpDir, "config.yaml"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load error: %v", err)
	}

	if cfg.Morph.APIKey != "🔑secret-key🔐" {
		t.Errorf("APIKey = %s, want 🔑secret-key🔐", cfg.Morph.APIKey)
	}
}

func TestYAMLWithZeroWidthCharacters(t *testing.T) {
	tmpDir := t.TempDir()
	os.Setenv("DBA_HOME", tmpDir)
	defer os.Unsetenv("DBA_HOME")

	// Zero-width space and zero-width joiner
	content := "morph:\n  api_key: \"test\u200Bkey\u200Dvalue\"\n"

	if err := os.WriteFile(filepath.Join(tmpDir, "config.yaml"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load error: %v", err)
	}

	// The zero-width characters should be preserved
	if cfg.Morph.APIKey != "test\u200Bkey\u200Dvalue" {
		t.Errorf("APIKey doesn't match (may have zero-width chars stripped)")
	}
}

// =============================================================================
// Whitespace Edge Cases
// =============================================================================

func TestYAMLWithLeadingWhitespace(t *testing.T) {
	tmpDir := t.TempDir()
	os.Setenv("DBA_HOME", tmpDir)
	defer os.Unsetenv("DBA_HOME")

	// Leading whitespace before YAML content
	content := "\n\n\n  \nmorph:\n  api_key: \"leading-ws\"\n"

	if err := os.WriteFile(filepath.Join(tmpDir, "config.yaml"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load()
	if err != nil {
		t.Logf("Load with leading whitespace error: %v", err)
		return
	}

	t.Logf("Config loaded with leading whitespace: APIKey=%s", cfg.Morph.APIKey)
}

func TestYAMLWithTrailingWhitespace(t *testing.T) {
	tmpDir := t.TempDir()
	os.Setenv("DBA_HOME", tmpDir)
	defer os.Unsetenv("DBA_HOME")

	content := "morph:\n  api_key: \"trailing-ws\"\n\n\n  \n"

	if err := os.WriteFile(filepath.Join(tmpDir, "config.yaml"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load error: %v", err)
	}

	if cfg.Morph.APIKey != "trailing-ws" {
		t.Errorf("APIKey = %s, want trailing-ws", cfg.Morph.APIKey)
	}
}

func TestYAMLWithTabsVsSpaces(t *testing.T) {
	tmpDir := t.TempDir()
	os.Setenv("DBA_HOME", tmpDir)
	defer os.Unsetenv("DBA_HOME")

	// YAML spec requires spaces for indentation, but some parsers accept tabs
	content := "morph:\n\tapi_key: \"tab-indent\"\n"

	if err := os.WriteFile(filepath.Join(tmpDir, "config.yaml"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load()
	if err != nil {
		t.Logf("Load with tab indentation error: %v (expected - YAML requires spaces)", err)
		return
	}

	t.Logf("Config loaded with tabs: APIKey=%s", cfg.Morph.APIKey)
}

// =============================================================================
// Special YAML Values
// =============================================================================

func TestYAMLNullValues(t *testing.T) {
	tmpDir := t.TempDir()
	os.Setenv("DBA_HOME", tmpDir)
	defer os.Unsetenv("DBA_HOME")

	content := `
morph:
  api_key: null
  base_snapshot_id: ~
  vm:
    vcpus: 2
`
	if err := os.WriteFile(filepath.Join(tmpDir, "config.yaml"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load error: %v", err)
	}

	// When YAML has null/~, defaults may be applied during merge
	// The exact behavior depends on the YAML library and Load implementation
	t.Logf("APIKey with null YAML value = %q", cfg.Morph.APIKey)
	t.Logf("BaseSnapshotID with ~ YAML value = %q", cfg.Morph.BaseSnapshotID)
}

func TestYAMLBooleanStrings(t *testing.T) {
	tmpDir := t.TempDir()
	os.Setenv("DBA_HOME", tmpDir)
	defer os.Unsetenv("DBA_HOME")

	// "yes", "no", "on", "off" can be interpreted as booleans in YAML 1.1
	content := `
morph:
  api_key: "yes"
  base_snapshot_id: "no"
  vm:
    vcpus: 2
`
	if err := os.WriteFile(filepath.Join(tmpDir, "config.yaml"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load error: %v", err)
	}

	// Should be preserved as strings due to quotes
	t.Logf("APIKey = %q, BaseSnapshotID = %q", cfg.Morph.APIKey, cfg.Morph.BaseSnapshotID)
}

func TestYAMLLiteralBlockScalar(t *testing.T) {
	tmpDir := t.TempDir()
	os.Setenv("DBA_HOME", tmpDir)
	defer os.Unsetenv("DBA_HOME")

	// Literal block scalar
	content := `
morph:
  api_key: |
    line1
    line2
    line3
  vm:
    vcpus: 2
`
	if err := os.WriteFile(filepath.Join(tmpDir, "config.yaml"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load error: %v", err)
	}

	// Should contain newlines
	expected := "line1\nline2\nline3\n"
	if cfg.Morph.APIKey != expected {
		t.Errorf("APIKey = %q, want %q", cfg.Morph.APIKey, expected)
	}
}

func TestYAMLFoldedStrings(t *testing.T) {
	tmpDir := t.TempDir()
	os.Setenv("DBA_HOME", tmpDir)
	defer os.Unsetenv("DBA_HOME")

	// Folded block scalar
	content := `
morph:
  api_key: >
    this is a very
    long api key that
    spans multiple lines
  vm:
    vcpus: 2
`
	if err := os.WriteFile(filepath.Join(tmpDir, "config.yaml"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load error: %v", err)
	}

	// Folded strings replace newlines with spaces
	t.Logf("APIKey (folded) = %q", cfg.Morph.APIKey)
}

// =============================================================================
// Large File Edge Cases
// =============================================================================

func TestYAMLWithManyComments(t *testing.T) {
	tmpDir := t.TempDir()
	os.Setenv("DBA_HOME", tmpDir)
	defer os.Unsetenv("DBA_HOME")

	// Build a file with many comments
	content := "# Comment line 1\n"
	for i := 0; i < 100; i++ {
		content += "# Comment line " + string(rune('0'+i%10)) + "\n"
	}
	content += "morph:\n  api_key: \"after-comments\"\n  vm:\n    vcpus: 2\n"

	if err := os.WriteFile(filepath.Join(tmpDir, "config.yaml"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load error: %v", err)
	}

	if cfg.Morph.APIKey != "after-comments" {
		t.Errorf("APIKey = %s, want after-comments", cfg.Morph.APIKey)
	}
}

func TestYAMLWithVeryDeepNesting(t *testing.T) {
	tmpDir := t.TempDir()
	os.Setenv("DBA_HOME", tmpDir)
	defer os.Unsetenv("DBA_HOME")

	// Very deep nesting (though not in morph section)
	content := `
morph:
  api_key: "deep-test"
  vm:
    vcpus: 2
# Deep nesting in another section (should be ignored)
extra:
  level1:
    level2:
      level3:
        level4:
          level5:
            level6:
              level7:
                value: "deep"
`
	if err := os.WriteFile(filepath.Join(tmpDir, "config.yaml"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load error: %v", err)
	}

	if cfg.Morph.APIKey != "deep-test" {
		t.Errorf("APIKey = %s, want deep-test", cfg.Morph.APIKey)
	}
}

// =============================================================================
// Anchor and Alias Tests
// =============================================================================

func TestYAMLWithAnchorsAndAliases(t *testing.T) {
	tmpDir := t.TempDir()
	os.Setenv("DBA_HOME", tmpDir)
	defer os.Unsetenv("DBA_HOME")

	content := `
defaults: &defaults
  vcpus: 4
  memory: 8192

morph:
  api_key: "anchor-test"
  vm:
    <<: *defaults
    disk_size: 65536
`
	if err := os.WriteFile(filepath.Join(tmpDir, "config.yaml"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load()
	if err != nil {
		t.Logf("Load with anchors error: %v (may not be supported)", err)
		return
	}

	t.Logf("Config with anchors: VCPUs=%d, Memory=%d", cfg.Morph.VM.VCPUs, cfg.Morph.VM.Memory)
}
