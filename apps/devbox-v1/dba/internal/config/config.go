// internal/config/config.go
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// Config is the global DBA configuration
type Config struct {
	// DBA home directory
	Home string `yaml:"-"`

	// Port allocation settings
	Ports PortConfig `yaml:"ports"`

	// Daemon settings
	Daemon DaemonConfig `yaml:"daemon"`

	// Computer use settings
	ComputerUse ComputerUseConfig `yaml:"computer_use"`

	// Sync settings
	Sync SyncConfig `yaml:"sync"`

	// Default settings
	Defaults DefaultsConfig `yaml:"defaults"`

	// Morph holds Morph Cloud configuration
	Morph MorphConfig `yaml:"morph" json:"morph"`

	// AgentBrowser holds agent-browser configuration
	AgentBrowser AgentBrowserConfig `yaml:"agent_browser" json:"agent_browser"`
}

type PortConfig struct {
	RangeStart      int            `yaml:"range_start"`
	RangeEnd        int            `yaml:"range_end"`
	BlockSize       int            `yaml:"block_size"`
	StandardOffsets map[string]int `yaml:"standard_offsets"`
}

type DaemonConfig struct {
	Socket   string `yaml:"socket"`
	PIDFile  string `yaml:"pid_file"`
	LogFile  string `yaml:"log_file"`
	LogLevel string `yaml:"log_level"`
}

type ComputerUseConfig struct {
	Image      string `yaml:"image"`
	Resolution string `yaml:"resolution"`
	Memory     string `yaml:"memory"`
	CPUs       int    `yaml:"cpus"`
}

type SyncConfig struct {
	BarrierTimeout string   `yaml:"barrier_timeout"`
	DebounceMs     int      `yaml:"debounce_ms"`
	IgnorePatterns []string `yaml:"ignore_patterns"`
}

type DefaultsConfig struct {
	Packages []string `yaml:"packages"`
	Services []string `yaml:"services"`
	Ports    []string `yaml:"ports"`
}

// MorphConfig holds Morph Cloud settings
type MorphConfig struct {
	// APIKey is the Morph Cloud API key (can use env var reference like ${MORPH_API_KEY})
	APIKey string `yaml:"api_key" json:"api_key"`

	// BaseSnapshotID is the ID of the base snapshot with all tools installed
	BaseSnapshotID string `yaml:"base_snapshot_id" json:"base_snapshot_id"`

	// VM resource configuration
	VM VMConfig `yaml:"vm" json:"vm"`
}

// VMConfig holds VM resource settings
type VMConfig struct {
	// VCPUs is the number of virtual CPUs
	VCPUs int `yaml:"vcpus" json:"vcpus"`

	// Memory in MB
	Memory int `yaml:"memory" json:"memory"`

	// DiskSize in MB
	DiskSize int `yaml:"disk_size" json:"disk_size"`

	// TTLSeconds is how long the VM stays running before auto-stop
	TTLSeconds int `yaml:"ttl_seconds" json:"ttl_seconds"`
}

// AgentBrowserConfig holds agent-browser settings
type AgentBrowserConfig struct {
	// Path to agent-browser binary (default: "agent-browser")
	Path string `yaml:"path" json:"path"`

	// Timeout in milliseconds for commands
	Timeout int `yaml:"timeout" json:"timeout"`

	// SessionPrefix for isolating browser sessions
	SessionPrefix string `yaml:"session_prefix" json:"session_prefix"`
}

// DBAHome returns the DBA home directory
func DBAHome() string {
	if home := os.Getenv("DBA_HOME"); home != "" {
		return home
	}
	homeDir, _ := os.UserHomeDir()
	return filepath.Join(homeDir, ".dba")
}

// Load loads the configuration from file or returns defaults
func Load() (*Config, error) {
	cfg := DefaultConfig()

	configPath := filepath.Join(DBAHome(), "config.yaml")

	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			// Use defaults but still expand paths
			cfg = expandConfigPaths(cfg)
			return cfg, nil
		}
		return nil, err
	}

	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, err
	}

	// Expand paths
	cfg = expandConfigPaths(cfg)

	return cfg, nil
}

// expandConfigPaths expands all path fields in the config
func expandConfigPaths(cfg *Config) *Config {
	cfg.Home = DBAHome()
	cfg.Daemon.Socket = expandPath(cfg.Daemon.Socket, cfg.Home)
	cfg.Daemon.PIDFile = expandPath(cfg.Daemon.PIDFile, cfg.Home)
	cfg.Daemon.LogFile = expandPath(cfg.Daemon.LogFile, cfg.Home)
	return cfg
}

func expandPath(path, home string) string {
	if path == "" {
		return path
	}
	if path[0] == '~' {
		homeDir, _ := os.UserHomeDir()
		return filepath.Join(homeDir, path[1:])
	}
	return path
}

// GetMorphAPIKey returns the Morph API key from config or environment
func (c *Config) GetMorphAPIKey() string {
	if c.Morph.APIKey != "" {
		// Check if it's an env var reference
		if strings.HasPrefix(c.Morph.APIKey, "${") && strings.HasSuffix(c.Morph.APIKey, "}") {
			envVar := strings.TrimSuffix(strings.TrimPrefix(c.Morph.APIKey, "${"), "}")
			return os.Getenv(envVar)
		}
		return c.Morph.APIKey
	}
	return os.Getenv("MORPH_API_KEY")
}

// GetBaseSnapshotID returns the base snapshot ID from config or environment
func (c *Config) GetBaseSnapshotID() string {
	if c.Morph.BaseSnapshotID != "" {
		return c.Morph.BaseSnapshotID
	}
	return os.Getenv("DBA_BASE_SNAPSHOT")
}

// Validate checks the configuration for errors
func (c *Config) Validate() error {
	var errs []string

	// Validate Morph config (only if Morph values are set - they might be using defaults)
	if c.Morph.VM.VCPUs < 1 {
		errs = append(errs, "morph.vm.vcpus must be at least 1")
	}
	if c.Morph.VM.Memory < 512 {
		errs = append(errs, "morph.vm.memory must be at least 512 MB")
	}
	if c.Morph.VM.DiskSize < 1024 {
		errs = append(errs, "morph.vm.disk_size must be at least 1024 MB")
	}
	if c.Morph.VM.TTLSeconds < 60 {
		errs = append(errs, "morph.vm.ttl_seconds must be at least 60")
	}

	// Validate agent-browser config
	if c.AgentBrowser.Timeout < 1000 {
		errs = append(errs, "agent_browser.timeout must be at least 1000 ms")
	}

	if len(errs) > 0 {
		return fmt.Errorf("config validation errors:\n  - %s", strings.Join(errs, "\n  - "))
	}

	return nil
}
