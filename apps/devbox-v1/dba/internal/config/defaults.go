// internal/config/defaults.go
package config

// Default Morph Cloud settings
const (
	DefaultMorphVCPUs      = 2
	DefaultMorphMemory     = 4096  // 4GB
	DefaultMorphDiskSize   = 32768 // 32GB
	DefaultMorphTTLSeconds = 3600  // 1 hour
)

// Default agent-browser settings
const (
	DefaultAgentBrowserPath    = "agent-browser"
	DefaultAgentBrowserTimeout = 30000 // 30 seconds
	DefaultSessionPrefix       = "dba"
)

// DefaultMorphConfig returns default Morph configuration
func DefaultMorphConfig() MorphConfig {
	return MorphConfig{
		APIKey:         "${MORPH_API_KEY}",
		BaseSnapshotID: "snapshot_u16ybfyb", // DBA base snapshot v3
		VM: VMConfig{
			VCPUs:      DefaultMorphVCPUs,
			Memory:     DefaultMorphMemory,
			DiskSize:   DefaultMorphDiskSize,
			TTLSeconds: DefaultMorphTTLSeconds,
		},
	}
}

// DefaultAgentBrowserConfig returns default agent-browser configuration
func DefaultAgentBrowserConfig() AgentBrowserConfig {
	return AgentBrowserConfig{
		Path:          DefaultAgentBrowserPath,
		Timeout:       DefaultAgentBrowserTimeout,
		SessionPrefix: DefaultSessionPrefix,
	}
}

// DefaultConfig returns the default configuration
func DefaultConfig() *Config {
	home := DBAHome()

	return &Config{
		Home: home,

		Ports: PortConfig{
			RangeStart: 10000,
			RangeEnd:   60000,
			BlockSize:  100,
			StandardOffsets: map[string]int{
				"PORT":              0,
				"APP_PORT":          0,
				"API_PORT":          1,
				"DB_PORT":           2,
				"REDIS_PORT":        3,
				"HMR_PORT":          4,
				"WS_PORT":           5,
				"STORYBOOK_PORT":    6,
				"DOCS_PORT":         7,
				"CODE_PORT":         80,
				"VNC_PORT":          90,
				"COMPUTER_API_PORT": 91,
			},
		},

		Daemon: DaemonConfig{
			Socket:   "~/.dba/daemon.sock",
			PIDFile:  "~/.dba/daemon.pid",
			LogFile:  "~/.dba/daemon.log",
			LogLevel: "info",
		},

		ComputerUse: ComputerUseConfig{
			Image:      "dba-computer-use:latest",
			Resolution: "1920x1080",
			Memory:     "2g",
			CPUs:       2,
		},

		Sync: SyncConfig{
			BarrierTimeout: "10s",
			DebounceMs:     100,
			IgnorePatterns: []string{
				"node_modules",
				".git",
				"dist",
				"build",
				".next",
				"__pycache__",
				".dba",
			},
		},

		Defaults: DefaultsConfig{
			Packages: []string{
				"openvscode-server@1.85.1",
				"process-compose@latest",
				"ripgrep@latest",
			},
			Services: []string{
				"openvscode-server",
			},
			Ports: []string{
				"PORT",
				"API_PORT",
				"DB_PORT",
				"CODE_PORT",
				"VNC_PORT",
				"COMPUTER_API_PORT",
			},
		},

		Morph:        DefaultMorphConfig(),
		AgentBrowser: DefaultAgentBrowserConfig(),
	}
}
