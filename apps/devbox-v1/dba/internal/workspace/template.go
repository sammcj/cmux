// internal/workspace/template.go
package workspace

import (
	"os"
	"path/filepath"

	"github.com/dba-cli/dba/internal/config"
	"gopkg.in/yaml.v3"
)

// Template represents a workspace template
type Template struct {
	Name           string            `yaml:"name"`
	Packages       []string          `yaml:"packages"`
	Env            map[string]string `yaml:"env"`
	ProcessCompose string            `yaml:"process_compose"`
	WebCommand     string            `yaml:"web_command"`
	WebReadyLine   string            `yaml:"web_ready_line"`
}

// LoadTemplate loads a template by name
func LoadTemplate(name string) (*Template, error) {
	// Try custom template first
	customPath := filepath.Join(config.DBAHome(), "templates", name, "template.yaml")
	if _, err := os.Stat(customPath); err == nil {
		return loadTemplateFile(customPath)
	}

	// Use built-in
	return getBuiltinTemplate(name)
}

func loadTemplateFile(path string) (*Template, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var tmpl Template
	if err := yaml.Unmarshal(data, &tmpl); err != nil {
		return nil, err
	}

	// Load process-compose.yaml from same directory
	dir := filepath.Dir(path)
	pcPath := filepath.Join(dir, "process-compose.yaml")
	if pcData, err := os.ReadFile(pcPath); err == nil {
		tmpl.ProcessCompose = string(pcData)
	}

	return &tmpl, nil
}

func getBuiltinTemplate(name string) (*Template, error) {
	templates := map[string]*Template{
		"node": {
			Name:     "node",
			Packages: []string{"nodejs@20", "pnpm@latest"},
			Env:      map[string]string{"NODE_ENV": "development"},
			ProcessCompose: `version: "0.5"
processes:
  vscode:
    command: openvscode-server --port ${CODE_PORT} --host 0.0.0.0 --without-connection-token
    ready_log_line: "HTTP server listening"

  web:
    command: pnpm dev --port ${PORT}
    working_dir: ${DBA_WORKSPACE_PATH}/project
    ready_log_line: "ready"
    depends_on:
      vscode:
        condition: process_started
`,
			WebCommand:   "pnpm dev --port ${PORT}",
			WebReadyLine: "ready",
		},
		"nextjs": {
			Name:     "nextjs",
			Packages: []string{"nodejs@20", "pnpm@latest"},
			Env:      map[string]string{"NODE_ENV": "development", "NEXT_TELEMETRY_DISABLED": "1"},
			ProcessCompose: `version: "0.5"
processes:
  vscode:
    command: openvscode-server --port ${CODE_PORT} --host 0.0.0.0 --without-connection-token
    ready_log_line: "HTTP server listening"

  web:
    command: pnpm next dev --port ${PORT}
    working_dir: ${DBA_WORKSPACE_PATH}/project
    ready_log_line: "Ready"
    depends_on:
      vscode:
        condition: process_started
`,
			WebCommand:   "pnpm next dev --port ${PORT}",
			WebReadyLine: "Ready",
		},
		"python": {
			Name:     "python",
			Packages: []string{"python@3.12", "uv@latest"},
			Env:      map[string]string{"PYTHONDONTWRITEBYTECODE": "1"},
			ProcessCompose: `version: "0.5"
processes:
  vscode:
    command: openvscode-server --port ${CODE_PORT} --host 0.0.0.0 --without-connection-token
    ready_log_line: "HTTP server listening"

  web:
    command: python -m uvicorn main:app --reload --port ${PORT}
    working_dir: ${DBA_WORKSPACE_PATH}/project
    ready_log_line: "Uvicorn running"
    depends_on:
      vscode:
        condition: process_started
`,
			WebCommand:   "python -m uvicorn main:app --reload --port ${PORT}",
			WebReadyLine: "Uvicorn running",
		},
		"go": {
			Name:     "go",
			Packages: []string{"go@1.22"},
			Env:      map[string]string{},
			ProcessCompose: `version: "0.5"
processes:
  vscode:
    command: openvscode-server --port ${CODE_PORT} --host 0.0.0.0 --without-connection-token
    ready_log_line: "HTTP server listening"

  web:
    command: go run .
    working_dir: ${DBA_WORKSPACE_PATH}/project
    environment:
      - PORT=${PORT}
    depends_on:
      vscode:
        condition: process_started
`,
			WebCommand:   "go run .",
			WebReadyLine: "listening",
		},
		"react": {
			Name:     "react",
			Packages: []string{"nodejs@20", "pnpm@latest"},
			Env:      map[string]string{"NODE_ENV": "development"},
			ProcessCompose: `version: "0.5"
processes:
  vscode:
    command: openvscode-server --port ${CODE_PORT} --host 0.0.0.0 --without-connection-token
    ready_log_line: "HTTP server listening"

  web:
    command: pnpm dev --port ${PORT}
    working_dir: ${DBA_WORKSPACE_PATH}/project
    ready_log_line: "Local:"
    depends_on:
      vscode:
        condition: process_started
`,
			WebCommand:   "pnpm dev --port ${PORT}",
			WebReadyLine: "Local:",
		},
		"rust": {
			Name:     "rust",
			Packages: []string{"rustup@latest", "cargo@latest"},
			Env:      map[string]string{},
			ProcessCompose: `version: "0.5"
processes:
  vscode:
    command: openvscode-server --port ${CODE_PORT} --host 0.0.0.0 --without-connection-token
    ready_log_line: "HTTP server listening"

  web:
    command: cargo run
    working_dir: ${DBA_WORKSPACE_PATH}/project
    environment:
      - PORT=${PORT}
    depends_on:
      vscode:
        condition: process_started
`,
			WebCommand:   "cargo run",
			WebReadyLine: "Listening",
		},
	}

	tmpl, ok := templates[name]
	if !ok {
		return templates["node"], nil // Default to node
	}
	return tmpl, nil
}

// ListBuiltinTemplates returns a list of built-in template names
func ListBuiltinTemplates() []string {
	return []string{"node", "nextjs", "python", "go", "react", "rust"}
}
