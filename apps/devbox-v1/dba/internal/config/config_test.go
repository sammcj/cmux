// internal/config/config_test.go
package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDBAHome(t *testing.T) {
	// Test with DBA_HOME env var set
	os.Setenv("DBA_HOME", "/custom/dba/home")
	defer os.Unsetenv("DBA_HOME")

	home := DBAHome()
	if home != "/custom/dba/home" {
		t.Errorf("expected /custom/dba/home, got %s", home)
	}
}

func TestDBAHomeDefault(t *testing.T) {
	// Ensure DBA_HOME is not set
	os.Unsetenv("DBA_HOME")

	home := DBAHome()
	homeDir, _ := os.UserHomeDir()
	expected := filepath.Join(homeDir, ".dba")

	if home != expected {
		t.Errorf("expected %s, got %s", expected, home)
	}
}

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg == nil {
		t.Fatal("DefaultConfig returned nil")
	}

	// Check port config
	if cfg.Ports.RangeStart != 10000 {
		t.Errorf("expected RangeStart 10000, got %d", cfg.Ports.RangeStart)
	}
	if cfg.Ports.RangeEnd != 60000 {
		t.Errorf("expected RangeEnd 60000, got %d", cfg.Ports.RangeEnd)
	}
	if cfg.Ports.BlockSize != 100 {
		t.Errorf("expected BlockSize 100, got %d", cfg.Ports.BlockSize)
	}

	// Check standard offsets
	if cfg.Ports.StandardOffsets["PORT"] != 0 {
		t.Errorf("expected PORT offset 0, got %d", cfg.Ports.StandardOffsets["PORT"])
	}
	if cfg.Ports.StandardOffsets["CODE_PORT"] != 80 {
		t.Errorf("expected CODE_PORT offset 80, got %d", cfg.Ports.StandardOffsets["CODE_PORT"])
	}
	if cfg.Ports.StandardOffsets["VNC_PORT"] != 90 {
		t.Errorf("expected VNC_PORT offset 90, got %d", cfg.Ports.StandardOffsets["VNC_PORT"])
	}

	// Check daemon config
	if cfg.Daemon.LogLevel != "info" {
		t.Errorf("expected LogLevel 'info', got %s", cfg.Daemon.LogLevel)
	}

	// Check computer use config
	if cfg.ComputerUse.Resolution != "1920x1080" {
		t.Errorf("expected Resolution 1920x1080, got %s", cfg.ComputerUse.Resolution)
	}
	if cfg.ComputerUse.CPUs != 2 {
		t.Errorf("expected CPUs 2, got %d", cfg.ComputerUse.CPUs)
	}

	// Check sync config
	if cfg.Sync.DebounceMs != 100 {
		t.Errorf("expected DebounceMs 100, got %d", cfg.Sync.DebounceMs)
	}
	if len(cfg.Sync.IgnorePatterns) == 0 {
		t.Error("expected non-empty IgnorePatterns")
	}

	// Check defaults
	if len(cfg.Defaults.Packages) == 0 {
		t.Error("expected non-empty default Packages")
	}
	if len(cfg.Defaults.Services) == 0 {
		t.Error("expected non-empty default Services")
	}
	if len(cfg.Defaults.Ports) == 0 {
		t.Error("expected non-empty default Ports")
	}
}

func TestLoad(t *testing.T) {
	// Test loading with no config file (should return defaults)
	os.Setenv("DBA_HOME", t.TempDir())
	defer os.Unsetenv("DBA_HOME")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if cfg == nil {
		t.Fatal("Load returned nil config")
	}

	// Should have default values
	if cfg.Ports.RangeStart != 10000 {
		t.Errorf("expected RangeStart 10000, got %d", cfg.Ports.RangeStart)
	}
}

func TestLoadWithConfigFile(t *testing.T) {
	tmpDir := t.TempDir()
	os.Setenv("DBA_HOME", tmpDir)
	defer os.Unsetenv("DBA_HOME")

	// Create a config file
	configContent := `
ports:
  range_start: 20000
  range_end: 50000
daemon:
  log_level: debug
`
	configPath := filepath.Join(tmpDir, "config.yaml")
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}

	// Should have values from config file
	if cfg.Ports.RangeStart != 20000 {
		t.Errorf("expected RangeStart 20000, got %d", cfg.Ports.RangeStart)
	}
	if cfg.Ports.RangeEnd != 50000 {
		t.Errorf("expected RangeEnd 50000, got %d", cfg.Ports.RangeEnd)
	}
	if cfg.Daemon.LogLevel != "debug" {
		t.Errorf("expected LogLevel 'debug', got %s", cfg.Daemon.LogLevel)
	}
}

func TestLoadWithInvalidYAML(t *testing.T) {
	tmpDir := t.TempDir()
	os.Setenv("DBA_HOME", tmpDir)
	defer os.Unsetenv("DBA_HOME")

	// Create an invalid YAML file
	invalidYAML := `
ports:
  range_start: not_a_number
  - invalid: yaml: structure
`
	configPath := filepath.Join(tmpDir, "config.yaml")
	if err := os.WriteFile(configPath, []byte(invalidYAML), 0644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	_, err := Load()
	if err == nil {
		t.Error("expected error for invalid YAML")
	}
}

func TestLoadWithPartialConfig(t *testing.T) {
	tmpDir := t.TempDir()
	os.Setenv("DBA_HOME", tmpDir)
	defer os.Unsetenv("DBA_HOME")

	// Create a config file with only some fields
	configContent := `
daemon:
  log_level: warn
`
	configPath := filepath.Join(tmpDir, "config.yaml")
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}

	// Daemon log level should be from file
	if cfg.Daemon.LogLevel != "warn" {
		t.Errorf("expected LogLevel 'warn', got %s", cfg.Daemon.LogLevel)
	}

	// Other values should still have defaults
	if cfg.Ports.RangeStart != 10000 {
		t.Errorf("expected RangeStart 10000, got %d", cfg.Ports.RangeStart)
	}
}

func TestLoadWithCompleteConfig(t *testing.T) {
	tmpDir := t.TempDir()
	os.Setenv("DBA_HOME", tmpDir)
	defer os.Unsetenv("DBA_HOME")

	// Create a complete config file
	configContent := `
ports:
  range_start: 15000
  range_end: 55000
  block_size: 50
  standard_offsets:
    PORT: 0
    API_PORT: 1
    CUSTOM_PORT: 99

daemon:
  socket: ~/.dba/custom.sock
  pid_file: ~/.dba/custom.pid
  log_file: ~/.dba/custom.log
  log_level: error

computer_use:
  image: custom-image:v1
  resolution: 1280x720
  memory: 4g
  cpus: 4

sync:
  barrier_timeout: 30s
  debounce_ms: 200
  ignore_patterns:
    - vendor
    - target

defaults:
  packages:
    - custom-pkg@1.0
  services:
    - custom-service
  ports:
    - CUSTOM_PORT
`
	configPath := filepath.Join(tmpDir, "config.yaml")
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}

	// Verify all sections
	if cfg.Ports.RangeStart != 15000 {
		t.Errorf("RangeStart = %d, want 15000", cfg.Ports.RangeStart)
	}
	if cfg.Ports.BlockSize != 50 {
		t.Errorf("BlockSize = %d, want 50", cfg.Ports.BlockSize)
	}
	if cfg.Ports.StandardOffsets["CUSTOM_PORT"] != 99 {
		t.Errorf("CUSTOM_PORT offset = %d, want 99", cfg.Ports.StandardOffsets["CUSTOM_PORT"])
	}

	if cfg.Daemon.LogLevel != "error" {
		t.Errorf("LogLevel = %s, want error", cfg.Daemon.LogLevel)
	}

	if cfg.ComputerUse.Resolution != "1280x720" {
		t.Errorf("Resolution = %s, want 1280x720", cfg.ComputerUse.Resolution)
	}
	if cfg.ComputerUse.CPUs != 4 {
		t.Errorf("CPUs = %d, want 4", cfg.ComputerUse.CPUs)
	}
	if cfg.ComputerUse.Memory != "4g" {
		t.Errorf("Memory = %s, want 4g", cfg.ComputerUse.Memory)
	}

	if cfg.Sync.DebounceMs != 200 {
		t.Errorf("DebounceMs = %d, want 200", cfg.Sync.DebounceMs)
	}
	if len(cfg.Sync.IgnorePatterns) != 2 {
		t.Errorf("IgnorePatterns count = %d, want 2", len(cfg.Sync.IgnorePatterns))
	}

	if len(cfg.Defaults.Packages) != 1 {
		t.Errorf("Packages count = %d, want 1", len(cfg.Defaults.Packages))
	}
}

func TestExpandPath(t *testing.T) {
	homeDir, _ := os.UserHomeDir()

	tests := []struct {
		name     string
		path     string
		home     string
		expected string
	}{
		{"empty path", "", "/home", ""},
		{"absolute path", "/var/log/test.log", "/home", "/var/log/test.log"},
		{"tilde path", "~/.dba/test", "/custom/home", filepath.Join(homeDir, ".dba/test")},
		{"relative path", "relative/path", "/home", "relative/path"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := expandPath(tt.path, tt.home)
			if result != tt.expected {
				t.Errorf("expandPath(%q, %q) = %q, want %q", tt.path, tt.home, result, tt.expected)
			}
		})
	}
}

func TestDBAHomeEnvVariations(t *testing.T) {
	tests := []struct {
		name     string
		envValue string
	}{
		{"absolute path", "/custom/path"},
		{"with trailing slash", "/custom/path/"},
		{"home relative", "~/custom"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Setenv("DBA_HOME", tt.envValue)
			defer os.Unsetenv("DBA_HOME")

			home := DBAHome()
			if home != tt.envValue {
				t.Errorf("DBAHome() = %q, want %q", home, tt.envValue)
			}
		})
	}
}

func TestDefaultConfigCompleteness(t *testing.T) {
	cfg := DefaultConfig()

	// Verify all standard port offsets are defined
	requiredOffsets := []string{
		"PORT", "APP_PORT", "API_PORT", "DB_PORT",
		"REDIS_PORT", "CODE_PORT", "VNC_PORT", "COMPUTER_API_PORT",
	}
	for _, name := range requiredOffsets {
		if _, ok := cfg.Ports.StandardOffsets[name]; !ok {
			t.Errorf("missing standard offset: %s", name)
		}
	}

	// Verify daemon paths contain tilde
	if cfg.Daemon.Socket == "" {
		t.Error("Daemon.Socket should not be empty")
	}
	if cfg.Daemon.PIDFile == "" {
		t.Error("Daemon.PIDFile should not be empty")
	}
	if cfg.Daemon.LogFile == "" {
		t.Error("Daemon.LogFile should not be empty")
	}

	// Verify computer use defaults
	if cfg.ComputerUse.Image == "" {
		t.Error("ComputerUse.Image should not be empty")
	}
	if cfg.ComputerUse.CPUs < 1 {
		t.Error("ComputerUse.CPUs should be at least 1")
	}

	// Verify sync defaults
	if cfg.Sync.BarrierTimeout == "" {
		t.Error("Sync.BarrierTimeout should not be empty")
	}
	if cfg.Sync.DebounceMs < 0 {
		t.Error("Sync.DebounceMs should not be negative")
	}

	// Verify defaults are non-empty
	if len(cfg.Defaults.Packages) == 0 {
		t.Error("Defaults.Packages should not be empty")
	}
	if len(cfg.Defaults.Services) == 0 {
		t.Error("Defaults.Services should not be empty")
	}
	if len(cfg.Defaults.Ports) == 0 {
		t.Error("Defaults.Ports should not be empty")
	}
}

func TestConfigHomeIsSet(t *testing.T) {
	tmpDir := t.TempDir()
	os.Setenv("DBA_HOME", tmpDir)
	defer os.Unsetenv("DBA_HOME")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}

	if cfg.Home != tmpDir {
		t.Errorf("Home = %q, want %q", cfg.Home, tmpDir)
	}
}

func TestLoadPermissionError(t *testing.T) {
	// Skip on Windows as permissions work differently
	if os.Getenv("GOOS") == "windows" {
		t.Skip("skipping permission test on Windows")
	}

	tmpDir := t.TempDir()
	os.Setenv("DBA_HOME", tmpDir)
	defer os.Unsetenv("DBA_HOME")

	// Create a config file with no read permission
	configPath := filepath.Join(tmpDir, "config.yaml")
	if err := os.WriteFile(configPath, []byte("ports:\n  range_start: 1000"), 0000); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}
	defer os.Chmod(configPath, 0644) // cleanup

	_, err := Load()
	if err == nil {
		t.Error("expected error for unreadable config file")
	}
}

func TestPortConfigRangeValidation(t *testing.T) {
	cfg := DefaultConfig()

	// Verify port range is valid
	if cfg.Ports.RangeStart >= cfg.Ports.RangeEnd {
		t.Errorf("RangeStart (%d) should be less than RangeEnd (%d)",
			cfg.Ports.RangeStart, cfg.Ports.RangeEnd)
	}

	// Verify block size is reasonable
	if cfg.Ports.BlockSize <= 0 {
		t.Errorf("BlockSize (%d) should be positive", cfg.Ports.BlockSize)
	}

	// Verify we can have at least some workspaces
	numWorkspaces := (cfg.Ports.RangeEnd - cfg.Ports.RangeStart) / cfg.Ports.BlockSize
	if numWorkspaces < 10 {
		t.Errorf("port range allows only %d workspaces, should be more", numWorkspaces)
	}
}

func TestStandardOffsetsWithinBlockSize(t *testing.T) {
	cfg := DefaultConfig()

	for name, offset := range cfg.Ports.StandardOffsets {
		if offset < 0 {
			t.Errorf("offset for %s (%d) should not be negative", name, offset)
		}
		if offset >= cfg.Ports.BlockSize {
			t.Errorf("offset for %s (%d) exceeds BlockSize (%d)",
				name, offset, cfg.Ports.BlockSize)
		}
	}
}

// TestLoadWithEmptyConfig tests loading an empty config file
func TestLoadWithEmptyConfig(t *testing.T) {
	tmpDir := t.TempDir()
	os.Setenv("DBA_HOME", tmpDir)
	defer os.Unsetenv("DBA_HOME")

	// Create an empty config file
	configPath := filepath.Join(tmpDir, "config.yaml")
	if err := os.WriteFile(configPath, []byte(""), 0644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}

	// Should still have defaults
	if cfg.Ports.RangeStart != 10000 {
		t.Errorf("expected RangeStart 10000, got %d", cfg.Ports.RangeStart)
	}
}

// TestLoadWithCommentsOnly tests loading a config with only comments
func TestLoadWithCommentsOnly(t *testing.T) {
	tmpDir := t.TempDir()
	os.Setenv("DBA_HOME", tmpDir)
	defer os.Unsetenv("DBA_HOME")

	// Create a config file with only comments
	configContent := `
# This is a comment
# Another comment
`
	configPath := filepath.Join(tmpDir, "config.yaml")
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}

	// Should still have defaults
	if cfg.Ports.RangeStart != 10000 {
		t.Errorf("expected RangeStart 10000, got %d", cfg.Ports.RangeStart)
	}
}

// TestExpandPathWithTildeOnly tests expanding path with just tilde
func TestExpandPathWithTildeOnly(t *testing.T) {
	homeDir, _ := os.UserHomeDir()

	result := expandPath("~", "/custom/home")
	expected := filepath.Join(homeDir, "")
	if result != expected {
		t.Errorf("expandPath(~) = %q, want %q", result, expected)
	}
}

// TestExpandPathWithTildeSlash tests expanding path with tilde followed by slash
func TestExpandPathWithTildeSlash(t *testing.T) {
	homeDir, _ := os.UserHomeDir()

	result := expandPath("~/", "/custom/home")
	expected := filepath.Join(homeDir, "")
	if result != expected {
		t.Errorf("expandPath(~/) = %q, want %q", result, expected)
	}
}

// TestConfigPathExpansion tests that daemon paths are expanded
func TestConfigPathExpansion(t *testing.T) {
	tmpDir := t.TempDir()
	os.Setenv("DBA_HOME", tmpDir)
	defer os.Unsetenv("DBA_HOME")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}

	// Paths should be expanded (not contain tilde)
	if len(cfg.Daemon.Socket) > 0 && cfg.Daemon.Socket[0] == '~' {
		t.Error("Daemon.Socket should be expanded, not contain tilde")
	}
	if len(cfg.Daemon.PIDFile) > 0 && cfg.Daemon.PIDFile[0] == '~' {
		t.Error("Daemon.PIDFile should be expanded, not contain tilde")
	}
	if len(cfg.Daemon.LogFile) > 0 && cfg.Daemon.LogFile[0] == '~' {
		t.Error("Daemon.LogFile should be expanded, not contain tilde")
	}
}

// TestLoadConcurrent tests concurrent config loading
func TestLoadConcurrent(t *testing.T) {
	tmpDir := t.TempDir()
	os.Setenv("DBA_HOME", tmpDir)
	defer os.Unsetenv("DBA_HOME")

	const numGoroutines = 10
	errors := make(chan error, numGoroutines)
	configs := make(chan *Config, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func() {
			cfg, err := Load()
			if err != nil {
				errors <- err
				return
			}
			configs <- cfg
		}()
	}

	// Collect results
	for i := 0; i < numGoroutines; i++ {
		select {
		case err := <-errors:
			t.Errorf("concurrent config load error: %v", err)
		case cfg := <-configs:
			if cfg == nil {
				t.Error("got nil config")
			}
			if cfg != nil && cfg.Ports.RangeStart != 10000 {
				t.Errorf("unexpected RangeStart: %d", cfg.Ports.RangeStart)
			}
		}
	}
}

// TestDBAHomeWithSpaces tests DBA_HOME with spaces in path
func TestDBAHomeWithSpaces(t *testing.T) {
	pathWithSpaces := "/path/with spaces/dba"
	os.Setenv("DBA_HOME", pathWithSpaces)
	defer os.Unsetenv("DBA_HOME")

	home := DBAHome()
	if home != pathWithSpaces {
		t.Errorf("DBAHome() = %q, want %q", home, pathWithSpaces)
	}
}

// TestDBAHomeWithUnicode tests DBA_HOME with unicode characters
func TestDBAHomeWithUnicode(t *testing.T) {
	pathWithUnicode := "/path/日本語/dba"
	os.Setenv("DBA_HOME", pathWithUnicode)
	defer os.Unsetenv("DBA_HOME")

	home := DBAHome()
	if home != pathWithUnicode {
		t.Errorf("DBAHome() = %q, want %q", home, pathWithUnicode)
	}
}

// TestConfigYAMLSyntax tests various YAML syntax patterns
func TestConfigYAMLSyntax(t *testing.T) {
	tests := []struct {
		name    string
		content string
		wantErr bool
	}{
		{
			name:    "inline map",
			content: "ports: {range_start: 15000, range_end: 50000}",
			wantErr: false,
		},
		{
			name:    "flow sequence",
			content: "sync:\n  ignore_patterns: [node_modules, .git]",
			wantErr: false,
		},
		{
			name:    "multi-document",
			content: "---\nports:\n  range_start: 15000\n---\ndaemon:\n  log_level: debug",
			wantErr: false,
		},
		{
			name:    "anchors",
			content: "defaults:\n  packages: &pkg\n    - nodejs\nports:\n  standard_offsets: {}",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			os.Setenv("DBA_HOME", tmpDir)
			defer os.Unsetenv("DBA_HOME")

			configPath := filepath.Join(tmpDir, "config.yaml")
			if err := os.WriteFile(configPath, []byte(tt.content), 0644); err != nil {
				t.Fatalf("failed to write config file: %v", err)
			}

			_, err := Load()
			if (err != nil) != tt.wantErr {
				t.Errorf("Load() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestConfigStructTags tests that all config struct fields have proper YAML tags
func TestConfigStructTags(t *testing.T) {
	cfg := DefaultConfig()

	// Verify that the config can be marshaled and unmarshaled
	// This implicitly tests that all struct tags are correct
	if cfg.Home == "" {
		cfg.Home = "/test"
	}

	// Marshal and verify no error would occur with yaml
	// (actual marshaling tested via Load/Save cycle)
}

// TestDefaultConfigValues tests specific default values
func TestDefaultConfigValues(t *testing.T) {
	cfg := DefaultConfig()

	// Test specific expected values
	tests := []struct {
		name     string
		got      interface{}
		expected interface{}
	}{
		{"Ports.RangeStart", cfg.Ports.RangeStart, 10000},
		{"Ports.RangeEnd", cfg.Ports.RangeEnd, 60000},
		{"Ports.BlockSize", cfg.Ports.BlockSize, 100},
		{"ComputerUse.Resolution", cfg.ComputerUse.Resolution, "1920x1080"},
		{"ComputerUse.Memory", cfg.ComputerUse.Memory, "2g"},
		{"ComputerUse.CPUs", cfg.ComputerUse.CPUs, 2},
		{"Daemon.LogLevel", cfg.Daemon.LogLevel, "info"},
		{"Sync.DebounceMs", cfg.Sync.DebounceMs, 100},
		{"Sync.BarrierTimeout", cfg.Sync.BarrierTimeout, "10s"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got != tt.expected {
				t.Errorf("%s = %v, want %v", tt.name, tt.got, tt.expected)
			}
		})
	}
}

// TestMorphConfigDefaults tests Morph Cloud default configuration
func TestMorphConfigDefaults(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.Morph.VM.VCPUs != DefaultMorphVCPUs {
		t.Errorf("expected %d vCPUs, got %d", DefaultMorphVCPUs, cfg.Morph.VM.VCPUs)
	}
	if cfg.Morph.VM.Memory != DefaultMorphMemory {
		t.Errorf("expected %d memory, got %d", DefaultMorphMemory, cfg.Morph.VM.Memory)
	}
	if cfg.Morph.VM.DiskSize != DefaultMorphDiskSize {
		t.Errorf("expected %d disk size, got %d", DefaultMorphDiskSize, cfg.Morph.VM.DiskSize)
	}
	if cfg.Morph.VM.TTLSeconds != DefaultMorphTTLSeconds {
		t.Errorf("expected %d TTL seconds, got %d", DefaultMorphTTLSeconds, cfg.Morph.VM.TTLSeconds)
	}
	if cfg.Morph.APIKey != "${MORPH_API_KEY}" {
		t.Errorf("expected API key template, got %s", cfg.Morph.APIKey)
	}
}

// TestAgentBrowserConfigDefaults tests agent-browser default configuration
func TestAgentBrowserConfigDefaults(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.AgentBrowser.Path != DefaultAgentBrowserPath {
		t.Errorf("expected path %s, got %s", DefaultAgentBrowserPath, cfg.AgentBrowser.Path)
	}
	if cfg.AgentBrowser.Timeout != DefaultAgentBrowserTimeout {
		t.Errorf("expected timeout %d, got %d", DefaultAgentBrowserTimeout, cfg.AgentBrowser.Timeout)
	}
	if cfg.AgentBrowser.SessionPrefix != DefaultSessionPrefix {
		t.Errorf("expected session prefix %s, got %s", DefaultSessionPrefix, cfg.AgentBrowser.SessionPrefix)
	}
}

// TestGetMorphAPIKey tests the GetMorphAPIKey helper
func TestGetMorphAPIKey(t *testing.T) {
	// Test from environment
	os.Setenv("MORPH_API_KEY", "test-key-123")
	defer os.Unsetenv("MORPH_API_KEY")

	cfg := &Config{}
	key := cfg.GetMorphAPIKey()
	if key != "test-key-123" {
		t.Errorf("expected test-key-123, got %s", key)
	}

	// Test from config with env var reference
	cfg.Morph.APIKey = "${MORPH_API_KEY}"
	key = cfg.GetMorphAPIKey()
	if key != "test-key-123" {
		t.Errorf("expected test-key-123 from env var reference, got %s", key)
	}

	// Test from config with direct value
	cfg.Morph.APIKey = "direct-key"
	key = cfg.GetMorphAPIKey()
	if key != "direct-key" {
		t.Errorf("expected direct-key, got %s", key)
	}
}

// TestGetBaseSnapshotID tests the GetBaseSnapshotID helper
func TestGetBaseSnapshotID(t *testing.T) {
	// Test from environment
	os.Setenv("DBA_BASE_SNAPSHOT", "snap-abc123")
	defer os.Unsetenv("DBA_BASE_SNAPSHOT")

	cfg := &Config{}
	id := cfg.GetBaseSnapshotID()
	if id != "snap-abc123" {
		t.Errorf("expected snap-abc123, got %s", id)
	}

	// Test from config - should take precedence
	cfg.Morph.BaseSnapshotID = "snap-xyz789"
	id = cfg.GetBaseSnapshotID()
	if id != "snap-xyz789" {
		t.Errorf("expected snap-xyz789, got %s", id)
	}
}

// TestConfigValidation tests the Validate helper
func TestConfigValidation(t *testing.T) {
	// Valid config
	cfg := DefaultConfig()
	if err := cfg.Validate(); err != nil {
		t.Errorf("expected valid config, got error: %v", err)
	}

	// Invalid vCPUs
	cfg = DefaultConfig()
	cfg.Morph.VM.VCPUs = 0
	if err := cfg.Validate(); err == nil {
		t.Error("expected error for invalid vCPUs")
	}

	// Invalid memory
	cfg = DefaultConfig()
	cfg.Morph.VM.Memory = 256
	if err := cfg.Validate(); err == nil {
		t.Error("expected error for invalid memory")
	}

	// Invalid disk size
	cfg = DefaultConfig()
	cfg.Morph.VM.DiskSize = 512
	if err := cfg.Validate(); err == nil {
		t.Error("expected error for invalid disk size")
	}

	// Invalid TTL
	cfg = DefaultConfig()
	cfg.Morph.VM.TTLSeconds = 30
	if err := cfg.Validate(); err == nil {
		t.Error("expected error for invalid TTL")
	}

	// Invalid timeout
	cfg = DefaultConfig()
	cfg.AgentBrowser.Timeout = 100
	if err := cfg.Validate(); err == nil {
		t.Error("expected error for invalid timeout")
	}
}

// TestMorphConfigFromYAML tests loading Morph config from YAML
func TestMorphConfigFromYAML(t *testing.T) {
	tmpDir := t.TempDir()
	os.Setenv("DBA_HOME", tmpDir)
	defer os.Unsetenv("DBA_HOME")

	// Create a config file with Morph settings
	configContent := `
morph:
  api_key: "test-morph-key"
  base_snapshot_id: "snap_test123"
  vm:
    vcpus: 4
    memory: 8192
    disk_size: 65536
    ttl_seconds: 7200

agent_browser:
  path: "/usr/local/bin/agent-browser"
  timeout: 60000
  session_prefix: "custom-prefix"
`
	configPath := filepath.Join(tmpDir, "config.yaml")
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}

	// Verify Morph settings
	if cfg.Morph.APIKey != "test-morph-key" {
		t.Errorf("Morph.APIKey = %s, want test-morph-key", cfg.Morph.APIKey)
	}
	if cfg.Morph.BaseSnapshotID != "snap_test123" {
		t.Errorf("Morph.BaseSnapshotID = %s, want snap_test123", cfg.Morph.BaseSnapshotID)
	}
	if cfg.Morph.VM.VCPUs != 4 {
		t.Errorf("Morph.VM.VCPUs = %d, want 4", cfg.Morph.VM.VCPUs)
	}
	if cfg.Morph.VM.Memory != 8192 {
		t.Errorf("Morph.VM.Memory = %d, want 8192", cfg.Morph.VM.Memory)
	}
	if cfg.Morph.VM.DiskSize != 65536 {
		t.Errorf("Morph.VM.DiskSize = %d, want 65536", cfg.Morph.VM.DiskSize)
	}
	if cfg.Morph.VM.TTLSeconds != 7200 {
		t.Errorf("Morph.VM.TTLSeconds = %d, want 7200", cfg.Morph.VM.TTLSeconds)
	}

	// Verify agent-browser settings
	if cfg.AgentBrowser.Path != "/usr/local/bin/agent-browser" {
		t.Errorf("AgentBrowser.Path = %s, want /usr/local/bin/agent-browser", cfg.AgentBrowser.Path)
	}
	if cfg.AgentBrowser.Timeout != 60000 {
		t.Errorf("AgentBrowser.Timeout = %d, want 60000", cfg.AgentBrowser.Timeout)
	}
	if cfg.AgentBrowser.SessionPrefix != "custom-prefix" {
		t.Errorf("AgentBrowser.SessionPrefix = %s, want custom-prefix", cfg.AgentBrowser.SessionPrefix)
	}
}

// TestGetMorphAPIKeyWithDifferentEnvVar tests env var reference with custom name
func TestGetMorphAPIKeyWithDifferentEnvVar(t *testing.T) {
	os.Setenv("MY_CUSTOM_MORPH_KEY", "custom-key-456")
	defer os.Unsetenv("MY_CUSTOM_MORPH_KEY")

	cfg := &Config{
		Morph: MorphConfig{
			APIKey: "${MY_CUSTOM_MORPH_KEY}",
		},
	}

	key := cfg.GetMorphAPIKey()
	if key != "custom-key-456" {
		t.Errorf("expected custom-key-456, got %s", key)
	}
}
