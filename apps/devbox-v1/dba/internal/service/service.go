// internal/service/service.go
package service

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/dba-cli/dba/internal/port"
	"github.com/dba-cli/dba/internal/workspace"
	"gopkg.in/yaml.v3"
)

// Manager manages services for a workspace
type Manager struct {
	workspace *workspace.Workspace
}

// NewManager creates a new service manager
func NewManager(ws *workspace.Workspace) *Manager {
	return &Manager{workspace: ws}
}

// ServiceStatus represents the status of a service
type ServiceStatus struct {
	Name      string `json:"name"`
	Status    string `json:"status"` // running, stopped, error
	PID       int    `json:"pid,omitempty"`
	Port      int    `json:"port,omitempty"`
	URL       string `json:"url,omitempty"`
	Healthy   bool   `json:"healthy,omitempty"`
	Uptime    int64  `json:"uptime_seconds,omitempty"`
	Restarts  int    `json:"restarts,omitempty"`
	StartedAt string `json:"started_at,omitempty"`
}

// UpResult is the result of starting services
type UpResult struct {
	Services   []ServiceStatus `json:"services"`
	AllHealthy bool            `json:"all_healthy"`
}

// Up starts services
func (m *Manager) Up(ctx context.Context, services []string, wait bool, timeout time.Duration) (*UpResult, error) {
	// Check if workspace is valid
	if m.workspace == nil {
		return nil, fmt.Errorf("workspace is nil")
	}

	// Build process-compose command (-D for detached mode, --tui=false for non-interactive)
	args := []string{"run", "--", "process-compose", "up", "-D", "--tui=false"}
	if len(services) > 0 {
		args = append(args, services...)
	}

	cmd := exec.CommandContext(ctx, "devbox", args...)
	cmd.Dir = m.workspace.Path
	cmd.Env = m.buildEnv()

	// Fixed by Agent #6: Provide better error messages for B01
	output, err := cmd.CombinedOutput()
	if err != nil {
		errMsg := strings.TrimSpace(string(output))
		if errMsg == "" {
			// Provide more context when output is empty
			if exitErr, ok := err.(*exec.ExitError); ok {
				errMsg = fmt.Sprintf("process exited with code %d", exitErr.ExitCode())
			} else {
				errMsg = err.Error()
			}
		}
		return nil, fmt.Errorf("failed to start services: %s", errMsg)
	}

	// Wait for services to be healthy if requested
	if wait {
		if err := m.WaitForHealthy(ctx, services, timeout); err != nil {
			// Don't fail, just note in result
			// We still return the current status
		}
	}

	// Get current status
	statuses, err := m.List(ctx, false)
	if err != nil {
		return nil, err
	}

	// Filter to requested services
	if len(services) > 0 {
		filtered := []ServiceStatus{}
		serviceSet := make(map[string]bool)
		for _, s := range services {
			serviceSet[s] = true
		}
		for _, status := range statuses {
			if serviceSet[status.Name] {
				filtered = append(filtered, status)
			}
		}
		statuses = filtered
	}

	// Check if all healthy
	allHealthy := true
	for _, s := range statuses {
		if s.Status == "running" && !s.Healthy {
			allHealthy = false
			break
		}
	}

	return &UpResult{
		Services:   statuses,
		AllHealthy: allHealthy,
	}, nil
}

// Down stops services
func (m *Manager) Down(ctx context.Context, services []string, timeout time.Duration) error {
	if m.workspace == nil {
		return fmt.Errorf("workspace is nil")
	}

	args := []string{"run", "--", "process-compose", "down"}
	if len(services) > 0 {
		args = append(args, services...)
	}

	cmd := exec.CommandContext(ctx, "devbox", args...)
	cmd.Dir = m.workspace.Path
	cmd.Env = m.buildEnv()

	// Fixed by Agent #6: Provide better error messages
	output, err := cmd.CombinedOutput()
	if err != nil {
		errMsg := strings.TrimSpace(string(output))
		if errMsg == "" {
			if exitErr, ok := err.(*exec.ExitError); ok {
				errMsg = fmt.Sprintf("process exited with code %d", exitErr.ExitCode())
			} else {
				errMsg = err.Error()
			}
		}
		return fmt.Errorf("failed to stop services: %s", errMsg)
	}

	return nil
}

// Restart restarts services
func (m *Manager) Restart(ctx context.Context, services []string, hard bool) error {
	if m.workspace == nil {
		return fmt.Errorf("workspace is nil")
	}

	args := []string{"run", "--", "process-compose", "restart"}
	if len(services) > 0 {
		args = append(args, services...)
	}

	cmd := exec.CommandContext(ctx, "devbox", args...)
	cmd.Dir = m.workspace.Path
	cmd.Env = m.buildEnv()

	// Fixed by Agent #6: Provide better error messages
	output, err := cmd.CombinedOutput()
	if err != nil {
		errMsg := strings.TrimSpace(string(output))
		if errMsg == "" {
			if exitErr, ok := err.(*exec.ExitError); ok {
				errMsg = fmt.Sprintf("process exited with code %d", exitErr.ExitCode())
			} else {
				errMsg = err.Error()
			}
		}
		return fmt.Errorf("failed to restart services: %s", errMsg)
	}

	return nil
}

// List lists all services and their status
func (m *Manager) List(ctx context.Context, includeDisabled bool) ([]ServiceStatus, error) {
	args := []string{"run", "--", "process-compose", "process", "list", "-o", "json"}

	cmd := exec.CommandContext(ctx, "devbox", args...)
	cmd.Dir = m.workspace.Path
	cmd.Env = m.buildEnv()

	output, err := cmd.Output()
	if err != nil {
		// process-compose might not be running, return empty list
		return []ServiceStatus{}, nil
	}

	// Extract JSON from output (devbox adds preamble lines)
	outputStr := string(output)
	jsonStart := strings.Index(outputStr, "[")
	jsonEnd := strings.LastIndex(outputStr, "]")
	if jsonStart == -1 || jsonEnd == -1 || jsonEnd <= jsonStart {
		// No valid JSON array found
		return []ServiceStatus{}, nil
	}
	jsonData := []byte(outputStr[jsonStart : jsonEnd+1])

	// Parse process-compose output
	var processes []struct {
		Name        string `json:"name"`
		Status      string `json:"status"`
		PID         int    `json:"pid"`
		IsRunning   bool   `json:"is_running"`
		Restarts    int    `json:"restarts"`
		UptimeNanos int64  `json:"uptime"`
	}

	if err := json.Unmarshal(jsonData, &processes); err != nil {
		return nil, fmt.Errorf("failed to parse process list: %w", err)
	}

	statuses := make([]ServiceStatus, len(processes))
	for i, p := range processes {
		status := ServiceStatus{
			Name:     p.Name,
			PID:      p.PID,
			Restarts: p.Restarts,
		}

		if p.IsRunning {
			status.Status = "running"
			status.Uptime = p.UptimeNanos / int64(time.Second)
			status.StartedAt = time.Now().Add(-time.Duration(p.UptimeNanos)).Format(time.RFC3339)
		} else {
			status.Status = "stopped"
		}

		// Add port and URL if we know it
		if portNum, ok := m.getServicePort(p.Name); ok {
			status.Port = portNum
			status.URL = fmt.Sprintf("http://localhost:%d", portNum)

			// Check health by trying to connect
			if status.Status == "running" {
				status.Healthy = !port.IsPortFree(portNum)
			}
		}

		statuses[i] = status
	}

	return statuses, nil
}

// getServicePort returns the port for a known service
func (m *Manager) getServicePort(name string) (int, bool) {
	if m.workspace == nil || m.workspace.Ports == nil {
		return 0, false
	}

	// Map service names to port environment variables
	portMapping := map[string]string{
		"web":     "PORT",
		"vscode":  "CODE_PORT",
		"api":     "API_PORT",
		"db":      "DB_PORT",
	}

	if envName, ok := portMapping[name]; ok {
		if portNum, ok := m.workspace.Ports[envName]; ok {
			return portNum, true
		}
	}

	return 0, false
}

func (m *Manager) buildEnv() []string {
	env := os.Environ()
	if m.workspace == nil {
		return env
	}
	if m.workspace.Ports != nil {
		for name, portNum := range m.workspace.Ports {
			env = append(env, fmt.Sprintf("%s=%d", name, portNum))
		}
	}
	env = append(env, fmt.Sprintf("DBA_WORKSPACE_ID=%s", m.workspace.ID))
	env = append(env, fmt.Sprintf("DBA_WORKSPACE_PATH=%s", m.workspace.Path))
	return env
}

// GetServicePort returns the port for a service name (exported version)
func (m *Manager) GetServicePort(name string) (int, bool) {
	return m.getServicePort(name)
}

// GetWorkspace returns the workspace associated with this manager
func (m *Manager) GetWorkspace() *workspace.Workspace {
	return m.workspace
}

// ServiceConfig defines the configuration for a service
type ServiceConfig struct {
	Name         string            `json:"name" yaml:"name"`
	Command      string            `json:"command" yaml:"command"`
	WorkingDir   string            `json:"working_dir,omitempty" yaml:"working_dir,omitempty"`
	Environment  map[string]string `json:"environment,omitempty" yaml:"environment,omitempty"`
	Port         string            `json:"port,omitempty" yaml:"port,omitempty"` // Port env var name
	DependsOn    []string          `json:"depends_on,omitempty" yaml:"depends_on,omitempty"`
	ReadyLogLine string            `json:"ready_log_line,omitempty" yaml:"ready_log_line,omitempty"`
	IsDaemon     bool              `json:"is_daemon,omitempty" yaml:"is_daemon,omitempty"`
}

// AddResult is the result of adding a service
type AddResult struct {
	Name    string `json:"name"`
	Added   bool   `json:"added"`
	Message string `json:"message,omitempty"`
}

// Add adds a new service to the process-compose.yaml
func (m *Manager) Add(ctx context.Context, config ServiceConfig) (*AddResult, error) {
	if config.Name == "" {
		return nil, fmt.Errorf("service name is required")
	}
	if config.Command == "" {
		return nil, fmt.Errorf("service command is required")
	}

	// Read existing process-compose.yaml
	pcPath := m.workspace.ProcessComposePath()
	data, err := os.ReadFile(pcPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read process-compose.yaml: %w", err)
	}

	// Parse YAML - Fixed by Agent #6: use yaml.Unmarshal instead of json.Unmarshal
	var pc map[string]interface{}
	if err := yaml.Unmarshal(data, &pc); err != nil {
		return nil, fmt.Errorf("failed to parse process-compose.yaml: %w", err)
	}

	// Get or create processes map
	processes, ok := pc["processes"].(map[string]interface{})
	if !ok {
		processes = make(map[string]interface{})
		pc["processes"] = processes
	}

	// Check if service already exists
	if _, exists := processes[config.Name]; exists {
		return &AddResult{
			Name:    config.Name,
			Added:   false,
			Message: "service already exists",
		}, nil
	}

	// Build service definition
	svcDef := map[string]interface{}{
		"command": config.Command,
	}

	if config.WorkingDir != "" {
		svcDef["working_dir"] = config.WorkingDir
	}

	if len(config.Environment) > 0 {
		envList := make([]string, 0, len(config.Environment))
		for k, v := range config.Environment {
			envList = append(envList, fmt.Sprintf("%s=%s", k, v))
		}
		svcDef["environment"] = envList
	}

	if config.ReadyLogLine != "" {
		svcDef["ready_log_line"] = config.ReadyLogLine
	}

	if config.IsDaemon {
		svcDef["is_daemon"] = true
	}

	if len(config.DependsOn) > 0 {
		deps := make(map[string]interface{})
		for _, dep := range config.DependsOn {
			deps[dep] = map[string]string{"condition": "process_started"}
		}
		svcDef["depends_on"] = deps
	}

	// Add to processes
	processes[config.Name] = svcDef

	// Write back - Fixed by Agent #6: use yaml.Marshal instead of json.MarshalIndent
	newData, err := yaml.Marshal(pc)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal process-compose.yaml: %w", err)
	}

	if err := os.WriteFile(pcPath, newData, 0644); err != nil {
		return nil, fmt.Errorf("failed to write process-compose.yaml: %w", err)
	}

	return &AddResult{
		Name:    config.Name,
		Added:   true,
		Message: "service added successfully",
	}, nil
}

// Remove removes a service from process-compose.yaml
func (m *Manager) Remove(ctx context.Context, name string) error {
	if name == "" {
		return fmt.Errorf("service name is required")
	}

	// Read existing process-compose.yaml
	pcPath := m.workspace.ProcessComposePath()
	data, err := os.ReadFile(pcPath)
	if err != nil {
		return fmt.Errorf("failed to read process-compose.yaml: %w", err)
	}

	// Parse YAML - Fixed by Agent #6: use yaml.Unmarshal instead of json.Unmarshal
	var pc map[string]interface{}
	if err := yaml.Unmarshal(data, &pc); err != nil {
		return fmt.Errorf("failed to parse process-compose.yaml: %w", err)
	}

	// Get processes map
	processes, ok := pc["processes"].(map[string]interface{})
	if !ok {
		return fmt.Errorf("no processes defined in process-compose.yaml")
	}

	// Check if service exists
	if _, exists := processes[name]; !exists {
		return fmt.Errorf("service %s not found", name)
	}

	// Remove service
	delete(processes, name)

	// Write back - Fixed by Agent #6: use yaml.Marshal instead of json.MarshalIndent
	newData, err := yaml.Marshal(pc)
	if err != nil {
		return fmt.Errorf("failed to marshal process-compose.yaml: %w", err)
	}

	if err := os.WriteFile(pcPath, newData, 0644); err != nil {
		return fmt.Errorf("failed to write process-compose.yaml: %w", err)
	}

	return nil
}

// ServiceTemplate represents a predefined service template
type ServiceTemplate struct {
	Name         string
	Description  string
	Command      string
	Port         string
	ReadyLogLine string
	DependsOn    []string
	IsDaemon     bool
}

// GetServiceTemplates returns available service templates
func GetServiceTemplates() map[string]ServiceTemplate {
	return map[string]ServiceTemplate{
		"redis": {
			Name:         "redis",
			Description:  "Redis in-memory data store",
			Command:      "redis-server --port ${REDIS_PORT:-6379}",
			Port:         "REDIS_PORT",
			ReadyLogLine: "Ready to accept connections",
			IsDaemon:     true,
		},
		"postgres": {
			Name:         "postgres",
			Description:  "PostgreSQL database",
			Command:      "postgres -D ${PGDATA:-/tmp/pgdata} -p ${DB_PORT:-5432}",
			Port:         "DB_PORT",
			ReadyLogLine: "database system is ready to accept connections",
			IsDaemon:     true,
		},
		"nginx": {
			Name:         "nginx",
			Description:  "Nginx web server",
			Command:      "nginx -g 'daemon off;' -p . -c nginx.conf",
			Port:         "NGINX_PORT",
			ReadyLogLine: "start worker processes",
			IsDaemon:     true,
		},
		"api": {
			Name:         "api",
			Description:  "API server (generic)",
			Command:      "npm run dev:api",
			Port:         "API_PORT",
			ReadyLogLine: "Listening on port",
			DependsOn:    []string{"db"},
		},
		"worker": {
			Name:         "worker",
			Description:  "Background worker process",
			Command:      "npm run worker",
			DependsOn:    []string{"redis"},
			IsDaemon:     true,
		},
	}
}

// AddFromTemplate adds a service from a predefined template
func (m *Manager) AddFromTemplate(ctx context.Context, templateName string, serviceName string) (*AddResult, error) {
	templates := GetServiceTemplates()
	tmpl, ok := templates[templateName]
	if !ok {
		return nil, fmt.Errorf("unknown service template: %s", templateName)
	}

	if serviceName == "" {
		serviceName = tmpl.Name
	}

	config := ServiceConfig{
		Name:         serviceName,
		Command:      tmpl.Command,
		Port:         tmpl.Port,
		ReadyLogLine: tmpl.ReadyLogLine,
		DependsOn:    tmpl.DependsOn,
		IsDaemon:     tmpl.IsDaemon,
	}

	return m.Add(ctx, config)
}

// GetDependencyOrder returns the services in proper startup order
func (m *Manager) GetDependencyOrder(ctx context.Context) ([]string, error) {
	// Read process-compose.yaml
	pcPath := m.workspace.ProcessComposePath()
	data, err := os.ReadFile(pcPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read process-compose.yaml: %w", err)
	}

	// Parse YAML - Fixed by Agent #6: use yaml.Unmarshal instead of json.Unmarshal
	var pc map[string]interface{}
	if err := yaml.Unmarshal(data, &pc); err != nil {
		return nil, fmt.Errorf("failed to parse process-compose.yaml: %w", err)
	}

	processes, ok := pc["processes"].(map[string]interface{})
	if !ok {
		return []string{}, nil
	}

	// Build dependency graph
	deps := make(map[string][]string)
	for name, proc := range processes {
		procMap, ok := proc.(map[string]interface{})
		if !ok {
			continue
		}

		deps[name] = []string{}
		if depOn, ok := procMap["depends_on"].(map[string]interface{}); ok {
			for depName := range depOn {
				deps[name] = append(deps[name], depName)
			}
		}
	}

	// Topological sort
	return topologicalSort(deps)
}

// topologicalSort performs a topological sort on the dependency graph
func topologicalSort(deps map[string][]string) ([]string, error) {
	var result []string
	visited := make(map[string]bool)
	visiting := make(map[string]bool)

	var visit func(name string) error
	visit = func(name string) error {
		if visiting[name] {
			return fmt.Errorf("circular dependency detected: %s", name)
		}
		if visited[name] {
			return nil
		}

		visiting[name] = true
		for _, dep := range deps[name] {
			if err := visit(dep); err != nil {
				return err
			}
		}
		visiting[name] = false
		visited[name] = true
		result = append(result, name)
		return nil
	}

	for name := range deps {
		if err := visit(name); err != nil {
			return nil, err
		}
	}

	return result, nil
}
