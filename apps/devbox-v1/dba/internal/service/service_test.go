// internal/service/service_test.go
package service

import (
	"context"
	"testing"
	"time"

	"github.com/dba-cli/dba/internal/workspace"
)

// mockWorkspace creates a mock workspace for testing
func mockWorkspace() *workspace.Workspace {
	return &workspace.Workspace{
		ID:   "ws_test123",
		Name: "test-workspace",
		Path: "/tmp/dba-test-workspace",
		Ports: map[string]int{
			"PORT":      10000,
			"CODE_PORT": 10080,
			"API_PORT":  10001,
			"DB_PORT":   10002,
		},
	}
}

func TestNewManager(t *testing.T) {
	ws := mockWorkspace()
	mgr := NewManager(ws)

	if mgr == nil {
		t.Fatal("NewManager returned nil")
	}

	if mgr.workspace != ws {
		t.Error("Manager workspace does not match input workspace")
	}
}

func TestGetServicePort(t *testing.T) {
	ws := mockWorkspace()
	mgr := NewManager(ws)

	tests := []struct {
		name        string
		serviceName string
		wantPort    int
		wantOK      bool
	}{
		{"web service", "web", 10000, true},
		{"vscode", "vscode", 10080, true},
		{"api service", "api", 10001, true},
		{"db service", "db", 10002, true},
		{"unknown service", "unknown", 0, false},
		{"empty service", "", 0, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			port, ok := mgr.GetServicePort(tt.serviceName)
			if ok != tt.wantOK {
				t.Errorf("GetServicePort(%q) ok = %v, want %v", tt.serviceName, ok, tt.wantOK)
			}
			if port != tt.wantPort {
				t.Errorf("GetServicePort(%q) port = %d, want %d", tt.serviceName, port, tt.wantPort)
			}
		})
	}
}

func TestGetWorkspace(t *testing.T) {
	ws := mockWorkspace()
	mgr := NewManager(ws)

	got := mgr.GetWorkspace()
	if got != ws {
		t.Error("GetWorkspace() did not return the correct workspace")
	}
}

func TestBuildEnv(t *testing.T) {
	ws := mockWorkspace()
	mgr := NewManager(ws)

	env := mgr.buildEnv()

	// Check that workspace ID and path are set
	foundID := false
	foundPath := false
	foundPort := false

	for _, e := range env {
		if e == "DBA_WORKSPACE_ID=ws_test123" {
			foundID = true
		}
		if e == "DBA_WORKSPACE_PATH=/tmp/dba-test-workspace" {
			foundPath = true
		}
		if e == "PORT=10000" {
			foundPort = true
		}
	}

	if !foundID {
		t.Error("buildEnv() did not include DBA_WORKSPACE_ID")
	}
	if !foundPath {
		t.Error("buildEnv() did not include DBA_WORKSPACE_PATH")
	}
	if !foundPort {
		t.Error("buildEnv() did not include PORT")
	}
}

func TestServiceStatus(t *testing.T) {
	status := ServiceStatus{
		Name:      "web",
		Status:    "running",
		PID:       12345,
		Port:      10000,
		URL:       "http://localhost:10000",
		Healthy:   true,
		Uptime:    3600,
		Restarts:  0,
		StartedAt: "2024-01-01T00:00:00Z",
	}

	if status.Name != "web" {
		t.Errorf("ServiceStatus.Name = %q, want %q", status.Name, "web")
	}
	if status.Status != "running" {
		t.Errorf("ServiceStatus.Status = %q, want %q", status.Status, "running")
	}
	if status.PID != 12345 {
		t.Errorf("ServiceStatus.PID = %d, want %d", status.PID, 12345)
	}
	if status.Port != 10000 {
		t.Errorf("ServiceStatus.Port = %d, want %d", status.Port, 10000)
	}
	if !status.Healthy {
		t.Error("ServiceStatus.Healthy should be true")
	}
}

func TestUpResult(t *testing.T) {
	result := UpResult{
		Services: []ServiceStatus{
			{Name: "web", Status: "running", Healthy: true},
			{Name: "api", Status: "running", Healthy: true},
		},
		AllHealthy: true,
	}

	if len(result.Services) != 2 {
		t.Errorf("UpResult.Services length = %d, want 2", len(result.Services))
	}
	if !result.AllHealthy {
		t.Error("UpResult.AllHealthy should be true")
	}
}

func TestListWithNoProcessCompose(t *testing.T) {
	// Test that List returns empty slice when process-compose is not running
	ws := mockWorkspace()
	ws.Path = "/nonexistent/path" // Use non-existent path
	mgr := NewManager(ws)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	statuses, err := mgr.List(ctx, false)
	if err != nil {
		t.Errorf("List() error = %v, want nil for missing process-compose", err)
	}
	if statuses == nil {
		t.Error("List() returned nil, want empty slice")
	}
}

func TestContextCancellation(t *testing.T) {
	ws := mockWorkspace()
	ws.Path = "/nonexistent/path"
	mgr := NewManager(ws)

	// Create already cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	// These should handle cancelled context gracefully
	_, err := mgr.List(ctx, false)
	// Should not panic, error is acceptable

	_, err = mgr.Up(ctx, nil, false, 0)
	if err == nil {
		// Context cancellation may or may not cause error depending on timing
		// The important thing is no panic
	}

	err = mgr.Down(ctx, nil, 0)
	// Should not panic

	err = mgr.Restart(ctx, nil, false)
	// Should not panic
	_ = err
}

func TestManagerWithEmptyPorts(t *testing.T) {
	ws := &workspace.Workspace{
		ID:    "ws_empty",
		Name:  "empty-workspace",
		Path:  "/tmp/empty",
		Ports: map[string]int{}, // Empty ports
	}
	mgr := NewManager(ws)

	// Should not crash with empty ports
	port, ok := mgr.GetServicePort("web")
	if ok {
		t.Error("GetServicePort should return false for empty ports")
	}
	if port != 0 {
		t.Errorf("GetServicePort should return 0 for empty ports, got %d", port)
	}

	env := mgr.buildEnv()
	// Should still have workspace ID and path
	foundID := false
	for _, e := range env {
		if e == "DBA_WORKSPACE_ID=ws_empty" {
			foundID = true
			break
		}
	}
	if !foundID {
		t.Error("buildEnv() should include workspace ID even with empty ports")
	}
}

func TestManagerWithNilPorts(t *testing.T) {
	ws := &workspace.Workspace{
		ID:    "ws_nil",
		Name:  "nil-workspace",
		Path:  "/tmp/nil",
		Ports: nil, // Nil ports
	}
	mgr := NewManager(ws)

	// Should not panic with nil ports
	port, ok := mgr.GetServicePort("web")
	if ok {
		t.Error("GetServicePort should return false for nil ports")
	}
	if port != 0 {
		t.Errorf("GetServicePort should return 0 for nil ports, got %d", port)
	}
}

func TestServiceConfig(t *testing.T) {
	config := ServiceConfig{
		Name:         "test-service",
		Command:      "npm run dev",
		WorkingDir:   "/app",
		Environment:  map[string]string{"NODE_ENV": "development"},
		Port:         "PORT",
		DependsOn:    []string{"db", "redis"},
		ReadyLogLine: "Server started",
		IsDaemon:     true,
	}

	if config.Name != "test-service" {
		t.Errorf("ServiceConfig.Name = %q, want %q", config.Name, "test-service")
	}
	if config.Command != "npm run dev" {
		t.Errorf("ServiceConfig.Command = %q, want %q", config.Command, "npm run dev")
	}
	if len(config.DependsOn) != 2 {
		t.Errorf("ServiceConfig.DependsOn length = %d, want 2", len(config.DependsOn))
	}
	if !config.IsDaemon {
		t.Error("ServiceConfig.IsDaemon should be true")
	}
}

func TestGetServiceTemplates(t *testing.T) {
	templates := GetServiceTemplates()

	if templates == nil {
		t.Fatal("GetServiceTemplates returned nil")
	}

	// Check that expected templates exist
	expectedTemplates := []string{"redis", "postgres", "nginx", "api", "worker"}
	for _, name := range expectedTemplates {
		if _, ok := templates[name]; !ok {
			t.Errorf("Expected template %q not found", name)
		}
	}

	// Verify redis template
	redis, ok := templates["redis"]
	if !ok {
		t.Fatal("redis template not found")
	}
	if redis.Name != "redis" {
		t.Errorf("redis.Name = %q, want %q", redis.Name, "redis")
	}
	if redis.Port != "REDIS_PORT" {
		t.Errorf("redis.Port = %q, want %q", redis.Port, "REDIS_PORT")
	}
	if !redis.IsDaemon {
		t.Error("redis.IsDaemon should be true")
	}

	// Verify postgres template
	postgres, ok := templates["postgres"]
	if !ok {
		t.Fatal("postgres template not found")
	}
	if postgres.Port != "DB_PORT" {
		t.Errorf("postgres.Port = %q, want %q", postgres.Port, "DB_PORT")
	}
}

func TestServiceTemplateStruct(t *testing.T) {
	tmpl := ServiceTemplate{
		Name:         "custom",
		Description:  "Custom service",
		Command:      "custom-cmd",
		Port:         "CUSTOM_PORT",
		ReadyLogLine: "Ready",
		DependsOn:    []string{"db"},
		IsDaemon:     false,
	}

	if tmpl.Name != "custom" {
		t.Errorf("ServiceTemplate.Name = %q, want %q", tmpl.Name, "custom")
	}
	if tmpl.Description != "Custom service" {
		t.Errorf("ServiceTemplate.Description = %q, want %q", tmpl.Description, "Custom service")
	}
}

func TestAddResult(t *testing.T) {
	result := AddResult{
		Name:    "test-service",
		Added:   true,
		Message: "service added successfully",
	}

	if result.Name != "test-service" {
		t.Errorf("AddResult.Name = %q, want %q", result.Name, "test-service")
	}
	if !result.Added {
		t.Error("AddResult.Added should be true")
	}
	if result.Message != "service added successfully" {
		t.Errorf("AddResult.Message = %q, want %q", result.Message, "service added successfully")
	}
}

func TestTopologicalSort(t *testing.T) {
	tests := []struct {
		name    string
		deps    map[string][]string
		wantLen int
		wantErr bool
	}{
		{
			name:    "empty",
			deps:    map[string][]string{},
			wantLen: 0,
			wantErr: false,
		},
		{
			name: "single service",
			deps: map[string][]string{
				"web": {},
			},
			wantLen: 1,
			wantErr: false,
		},
		{
			name: "linear dependency",
			deps: map[string][]string{
				"web": {"api"},
				"api": {"db"},
				"db":  {},
			},
			wantLen: 3,
			wantErr: false,
		},
		{
			name: "parallel services",
			deps: map[string][]string{
				"web":   {},
				"api":   {},
				"worker": {},
			},
			wantLen: 3,
			wantErr: false,
		},
		{
			name: "circular dependency",
			deps: map[string][]string{
				"a": {"b"},
				"b": {"c"},
				"c": {"a"},
			},
			wantLen: 0,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := topologicalSort(tt.deps)
			if (err != nil) != tt.wantErr {
				t.Errorf("topologicalSort() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && len(result) != tt.wantLen {
				t.Errorf("topologicalSort() length = %d, want %d", len(result), tt.wantLen)
			}
		})
	}
}

func TestAddValidation(t *testing.T) {
	ws := mockWorkspace()
	ws.Path = "/nonexistent/path"
	mgr := NewManager(ws)

	ctx := context.Background()

	// Test empty name
	_, err := mgr.Add(ctx, ServiceConfig{
		Name:    "",
		Command: "test",
	})
	if err == nil {
		t.Error("Add should return error for empty name")
	}

	// Test empty command
	_, err = mgr.Add(ctx, ServiceConfig{
		Name:    "test",
		Command: "",
	})
	if err == nil {
		t.Error("Add should return error for empty command")
	}
}

func TestRemoveValidation(t *testing.T) {
	ws := mockWorkspace()
	ws.Path = "/nonexistent/path"
	mgr := NewManager(ws)

	ctx := context.Background()

	// Test empty name
	err := mgr.Remove(ctx, "")
	if err == nil {
		t.Error("Remove should return error for empty name")
	}
}

func TestAddFromTemplateUnknown(t *testing.T) {
	ws := mockWorkspace()
	ws.Path = "/nonexistent/path"
	mgr := NewManager(ws)

	ctx := context.Background()

	_, err := mgr.AddFromTemplate(ctx, "unknown-template", "test")
	if err == nil {
		t.Error("AddFromTemplate should return error for unknown template")
	}
}
