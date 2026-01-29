// internal/service/edge_cases_test.go
package service

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/dba-cli/dba/internal/workspace"
)

// ═══════════════════════════════════════════════════════════════════════════════
// Edge Cases for Manager Creation
// ═══════════════════════════════════════════════════════════════════════════════

func TestNewManagerWithNilWorkspace(t *testing.T) {
	// This should not panic
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("NewManager panicked with nil workspace: %v", r)
		}
	}()

	mgr := NewManager(nil)
	if mgr == nil {
		t.Error("NewManager returned nil")
	}
}

func TestManagerMethodsWithNilWorkspace(t *testing.T) {
	mgr := NewManager(nil)
	ctx := context.Background()

	// These should not panic, but may return errors
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("Manager method panicked with nil workspace: %v", r)
		}
	}()

	// Test GetServicePort
	_, _ = mgr.GetServicePort("web")

	// Test GetWorkspace
	ws := mgr.GetWorkspace()
	if ws != nil {
		t.Error("GetWorkspace should return nil for nil workspace")
	}

	// Test buildEnv - this might panic without nil check
	// We need to handle this case
	_ = ctx
}

func TestManagerWithEmptyWorkspace(t *testing.T) {
	ws := &workspace.Workspace{}
	mgr := NewManager(ws)

	// Should handle empty workspace gracefully
	port, ok := mgr.GetServicePort("web")
	if ok {
		t.Error("GetServicePort should return false for empty workspace")
	}
	if port != 0 {
		t.Errorf("GetServicePort should return 0, got %d", port)
	}

	env := mgr.buildEnv()
	// Should have workspace ID and path (even if empty)
	if env == nil {
		t.Error("buildEnv should not return nil")
	}
}

// ═══════════════════════════════════════════════════════════════════════════════
// Edge Cases for Port Mapping
// ═══════════════════════════════════════════════════════════════════════════════

func TestGetServicePortWithSpecialCharacters(t *testing.T) {
	ws := mockWorkspace()
	mgr := NewManager(ws)

	testCases := []string{
		"web/api",
		"service.name",
		"service-with-dashes",
		"service_with_underscores",
		"123numeric",
		"",
		" ",
		"  spaces  ",
		"\t\n",
		"unicode-日本語",
	}

	for _, tc := range testCases {
		t.Run(tc, func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("GetServicePort panicked for %q: %v", tc, r)
				}
			}()
			_, _ = mgr.GetServicePort(tc)
		})
	}
}

func TestGetServicePortWithLargePorts(t *testing.T) {
	ws := &workspace.Workspace{
		ID:   "ws_test",
		Path: "/tmp/test",
		Ports: map[string]int{
			"PORT":      65535,         // Max valid port
			"CODE_PORT": 0,             // Min port
			"API_PORT":  -1,            // Invalid negative
			"DB_PORT":   999999,        // Over max
		},
	}
	mgr := NewManager(ws)

	tests := []struct {
		service  string
		wantPort int
		wantOK   bool
	}{
		{"web", 65535, true},
		{"vscode", 0, true},
		{"api", -1, true},
		{"db", 999999, true},
	}

	for _, tt := range tests {
		t.Run(tt.service, func(t *testing.T) {
			port, ok := mgr.GetServicePort(tt.service)
			if ok != tt.wantOK {
				t.Errorf("GetServicePort(%q) ok = %v, want %v", tt.service, ok, tt.wantOK)
			}
			if port != tt.wantPort {
				t.Errorf("GetServicePort(%q) port = %d, want %d", tt.service, port, tt.wantPort)
			}
		})
	}
}

// ═══════════════════════════════════════════════════════════════════════════════
// Edge Cases for Environment Building
// ═══════════════════════════════════════════════════════════════════════════════

func TestBuildEnvWithSpecialValues(t *testing.T) {
	ws := &workspace.Workspace{
		ID:   "ws_special=value",
		Path: "/path/with spaces/and=equals",
		Ports: map[string]int{
			"PORT=SPECIAL": 8080,
			"":             9000,
		},
	}
	mgr := NewManager(ws)

	env := mgr.buildEnv()
	if env == nil {
		t.Fatal("buildEnv returned nil")
	}

	// Verify we got some environment variables
	if len(env) == 0 {
		t.Error("buildEnv returned empty slice")
	}
}

func TestBuildEnvWithManyPorts(t *testing.T) {
	ports := make(map[string]int)
	for i := 0; i < 100; i++ {
		ports[string(rune('A'+i%26))+string(rune('0'+i/26))] = 10000 + i
	}

	ws := &workspace.Workspace{
		ID:    "ws_many_ports",
		Path:  "/tmp/test",
		Ports: ports,
	}
	mgr := NewManager(ws)

	env := mgr.buildEnv()

	// Should have at least the original env vars + ports + workspace vars
	if len(env) < len(ports)+2 {
		t.Errorf("buildEnv returned %d vars, expected at least %d", len(env), len(ports)+2)
	}
}

// ═══════════════════════════════════════════════════════════════════════════════
// Concurrent Access Tests
// ═══════════════════════════════════════════════════════════════════════════════

func TestConcurrentGetServicePort(t *testing.T) {
	ws := mockWorkspace()
	mgr := NewManager(ws)

	var wg sync.WaitGroup
	services := []string{"web", "api", "vscode", "db", "unknown"}

	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			svc := services[idx%len(services)]
			_, _ = mgr.GetServicePort(svc)
		}(i)
	}

	wg.Wait()
}

func TestConcurrentBuildEnv(t *testing.T) {
	ws := mockWorkspace()
	mgr := NewManager(ws)

	var wg sync.WaitGroup

	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			env := mgr.buildEnv()
			if env == nil {
				t.Error("buildEnv returned nil in concurrent access")
			}
		}()
	}

	wg.Wait()
}

func TestConcurrentList(t *testing.T) {
	ws := mockWorkspace()
	ws.Path = "/nonexistent/path"
	mgr := NewManager(ws)

	var wg sync.WaitGroup
	ctx := context.Background()

	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, _ = mgr.List(ctx, false)
		}()
	}

	wg.Wait()
}

// ═══════════════════════════════════════════════════════════════════════════════
// Context Handling Edge Cases
// ═══════════════════════════════════════════════════════════════════════════════

func TestOperationsWithDeadlineExceededContext(t *testing.T) {
	ws := mockWorkspace()
	ws.Path = "/nonexistent/path"
	mgr := NewManager(ws)

	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(-1*time.Second))
	defer cancel()

	// Context is already expired
	_, err := mgr.List(ctx, false)
	// Should handle gracefully
	_ = err

	_, err = mgr.Up(ctx, nil, false, 0)
	_ = err

	err = mgr.Down(ctx, nil, 0)
	_ = err
}

func TestOperationsWithValueContext(t *testing.T) {
	ws := mockWorkspace()
	ws.Path = "/nonexistent/path"
	mgr := NewManager(ws)

	type ctxKey string
	ctx := context.WithValue(context.Background(), ctxKey("test"), "value")

	// Should work with value context
	_, _ = mgr.List(ctx, false)
}

// ═══════════════════════════════════════════════════════════════════════════════
// ServiceStatus Edge Cases
// ═══════════════════════════════════════════════════════════════════════════════

func TestServiceStatusZeroValues(t *testing.T) {
	status := ServiceStatus{}

	if status.Name != "" {
		t.Error("Zero ServiceStatus should have empty Name")
	}
	if status.Status != "" {
		t.Error("Zero ServiceStatus should have empty Status")
	}
	if status.PID != 0 {
		t.Error("Zero ServiceStatus should have 0 PID")
	}
	if status.Healthy {
		t.Error("Zero ServiceStatus should not be Healthy")
	}
}

func TestServiceStatusWithNegativeValues(t *testing.T) {
	status := ServiceStatus{
		Name:     "test",
		Status:   "error",
		PID:      -1,
		Port:     -1,
		Uptime:   -100,
		Restarts: -5,
	}

	// Should not panic when marshaling
	data, err := json.Marshal(status)
	if err != nil {
		t.Errorf("Failed to marshal ServiceStatus with negative values: %v", err)
	}

	// Should unmarshal back correctly
	var decoded ServiceStatus
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Errorf("Failed to unmarshal ServiceStatus: %v", err)
	}

	if decoded.PID != -1 {
		t.Errorf("PID = %d, want -1", decoded.PID)
	}
}

// ═══════════════════════════════════════════════════════════════════════════════
// Topological Sort Edge Cases
// ═══════════════════════════════════════════════════════════════════════════════

func TestTopologicalSortSelfDependency(t *testing.T) {
	deps := map[string][]string{
		"a": {"a"}, // Self-dependency
	}

	_, err := topologicalSort(deps)
	if err == nil {
		t.Error("topologicalSort should detect self-dependency as circular")
	}
}

func TestTopologicalSortDeepChain(t *testing.T) {
	// Create a deep chain: a -> b -> c -> d -> ... -> z
	deps := make(map[string][]string)
	for i := 0; i < 26; i++ {
		name := string(rune('a' + i))
		if i < 25 {
			deps[name] = []string{string(rune('a' + i + 1))}
		} else {
			deps[name] = []string{}
		}
	}

	result, err := topologicalSort(deps)
	if err != nil {
		t.Errorf("topologicalSort failed on deep chain: %v", err)
	}
	if len(result) != 26 {
		t.Errorf("Expected 26 items, got %d", len(result))
	}

	// Verify order: z should come before y, y before x, etc.
	positions := make(map[string]int)
	for i, name := range result {
		positions[name] = i
	}

	for i := 0; i < 25; i++ {
		current := string(rune('a' + i))
		next := string(rune('a' + i + 1))
		if positions[next] >= positions[current] {
			t.Errorf("%s should come before %s in result", next, current)
		}
	}
}

func TestTopologicalSortDiamond(t *testing.T) {
	// Diamond dependency: a -> b, a -> c, b -> d, c -> d
	deps := map[string][]string{
		"a": {"b", "c"},
		"b": {"d"},
		"c": {"d"},
		"d": {},
	}

	result, err := topologicalSort(deps)
	if err != nil {
		t.Errorf("topologicalSort failed on diamond: %v", err)
	}
	if len(result) != 4 {
		t.Errorf("Expected 4 items, got %d", len(result))
	}

	// d must come before b and c, b and c must come before a
	positions := make(map[string]int)
	for i, name := range result {
		positions[name] = i
	}

	if positions["d"] >= positions["b"] {
		t.Error("d should come before b")
	}
	if positions["d"] >= positions["c"] {
		t.Error("d should come before c")
	}
	if positions["b"] >= positions["a"] {
		t.Error("b should come before a")
	}
	if positions["c"] >= positions["a"] {
		t.Error("c should come before a")
	}
}

func TestTopologicalSortWithNilDeps(t *testing.T) {
	deps := map[string][]string{
		"a": nil,
		"b": nil,
	}

	result, err := topologicalSort(deps)
	if err != nil {
		t.Errorf("topologicalSort failed with nil deps: %v", err)
	}
	if len(result) != 2 {
		t.Errorf("Expected 2 items, got %d", len(result))
	}
}

// ═══════════════════════════════════════════════════════════════════════════════
// Service Templates Edge Cases
// ═══════════════════════════════════════════════════════════════════════════════

func TestAllServiceTemplatesHaveRequiredFields(t *testing.T) {
	templates := GetServiceTemplates()

	for name, tmpl := range templates {
		t.Run(name, func(t *testing.T) {
			if tmpl.Name == "" {
				t.Error("Template Name is empty")
			}
			if tmpl.Description == "" {
				t.Error("Template Description is empty")
			}
			if tmpl.Command == "" {
				t.Error("Template Command is empty")
			}
		})
	}
}

func TestServiceTemplatesAreConsistent(t *testing.T) {
	templates := GetServiceTemplates()

	for name, tmpl := range templates {
		t.Run(name, func(t *testing.T) {
			// Name in map should match Name in struct
			if name != tmpl.Name {
				t.Errorf("Map key %q doesn't match template.Name %q", name, tmpl.Name)
			}

			// DependsOn should reference valid templates (if any)
			for _, dep := range tmpl.DependsOn {
				// Check if the dependency is a known template or a standard service
				knownDeps := map[string]bool{
					"db": true, "redis": true, "postgres": true,
					"web": true, "api": true,
				}
				if _, ok := templates[dep]; !ok && !knownDeps[dep] {
					t.Logf("Warning: template %q depends on %q which is not a template", name, dep)
				}
			}
		})
	}
}

// ═══════════════════════════════════════════════════════════════════════════════
// ServiceConfig Edge Cases
// ═══════════════════════════════════════════════════════════════════════════════

func TestServiceConfigJSONSerialization(t *testing.T) {
	config := ServiceConfig{
		Name:         "test-service",
		Command:      "npm run dev",
		WorkingDir:   "/app",
		Environment:  map[string]string{"KEY": "value", "EMPTY": ""},
		Port:         "PORT",
		DependsOn:    []string{"db", "redis"},
		ReadyLogLine: "Server started on port",
		IsDaemon:     true,
	}

	// Serialize
	data, err := json.Marshal(config)
	if err != nil {
		t.Fatalf("Failed to marshal ServiceConfig: %v", err)
	}

	// Deserialize
	var decoded ServiceConfig
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal ServiceConfig: %v", err)
	}

	// Verify
	if decoded.Name != config.Name {
		t.Errorf("Name = %q, want %q", decoded.Name, config.Name)
	}
	if decoded.Command != config.Command {
		t.Errorf("Command = %q, want %q", decoded.Command, config.Command)
	}
	if len(decoded.DependsOn) != len(config.DependsOn) {
		t.Errorf("DependsOn length = %d, want %d", len(decoded.DependsOn), len(config.DependsOn))
	}
	if decoded.IsDaemon != config.IsDaemon {
		t.Errorf("IsDaemon = %v, want %v", decoded.IsDaemon, config.IsDaemon)
	}
}

func TestServiceConfigWithEmptyEnvironment(t *testing.T) {
	config := ServiceConfig{
		Name:        "test",
		Command:     "test",
		Environment: map[string]string{},
	}

	data, err := json.Marshal(config)
	if err != nil {
		t.Fatalf("Failed to marshal with empty environment: %v", err)
	}

	var decoded ServiceConfig
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}
}

func TestServiceConfigWithNilEnvironment(t *testing.T) {
	config := ServiceConfig{
		Name:        "test",
		Command:     "test",
		Environment: nil,
	}

	data, err := json.Marshal(config)
	if err != nil {
		t.Fatalf("Failed to marshal with nil environment: %v", err)
	}

	var decoded ServiceConfig
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}
}

// ═══════════════════════════════════════════════════════════════════════════════
// File System Edge Cases (for Add/Remove)
// ═══════════════════════════════════════════════════════════════════════════════

func TestAddWithNonExistentProcessCompose(t *testing.T) {
	ws := &workspace.Workspace{
		ID:   "ws_test",
		Path: "/nonexistent/path/that/does/not/exist",
	}
	mgr := NewManager(ws)

	ctx := context.Background()
	_, err := mgr.Add(ctx, ServiceConfig{
		Name:    "test",
		Command: "test",
	})

	if err == nil {
		t.Error("Add should fail when process-compose.yaml doesn't exist")
	}
}

func TestRemoveWithNonExistentProcessCompose(t *testing.T) {
	ws := &workspace.Workspace{
		ID:   "ws_test",
		Path: "/nonexistent/path/that/does/not/exist",
	}
	mgr := NewManager(ws)

	ctx := context.Background()
	err := mgr.Remove(ctx, "test")

	if err == nil {
		t.Error("Remove should fail when process-compose.yaml doesn't exist")
	}
}

func TestAddWithTempDirectory(t *testing.T) {
	// Create temp directory with process-compose.yaml
	tmpDir, err := os.MkdirTemp("", "dba-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create minimal process-compose.yaml
	pcPath := filepath.Join(tmpDir, "process-compose.yaml")
	initialContent := `{"processes": {}}`
	if err := os.WriteFile(pcPath, []byte(initialContent), 0644); err != nil {
		t.Fatalf("Failed to write process-compose.yaml: %v", err)
	}

	ws := &workspace.Workspace{
		ID:   "ws_test",
		Path: tmpDir,
	}
	mgr := NewManager(ws)

	ctx := context.Background()
	result, err := mgr.Add(ctx, ServiceConfig{
		Name:    "test-service",
		Command: "echo hello",
	})

	if err != nil {
		t.Fatalf("Add failed: %v", err)
	}

	if !result.Added {
		t.Error("Expected service to be added")
	}

	// Try to add same service again
	result2, err := mgr.Add(ctx, ServiceConfig{
		Name:    "test-service",
		Command: "echo hello",
	})

	if err != nil {
		t.Fatalf("Second Add failed: %v", err)
	}

	if result2.Added {
		t.Error("Should not add duplicate service")
	}
}

func TestRemoveFromTempDirectory(t *testing.T) {
	// Create temp directory with process-compose.yaml
	tmpDir, err := os.MkdirTemp("", "dba-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create process-compose.yaml with a service
	pcPath := filepath.Join(tmpDir, "process-compose.yaml")
	content := `{"processes": {"test-service": {"command": "echo hello"}}}`
	if err := os.WriteFile(pcPath, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write process-compose.yaml: %v", err)
	}

	ws := &workspace.Workspace{
		ID:   "ws_test",
		Path: tmpDir,
	}
	mgr := NewManager(ws)

	ctx := context.Background()

	// Remove the service
	err = mgr.Remove(ctx, "test-service")
	if err != nil {
		t.Fatalf("Remove failed: %v", err)
	}

	// Try to remove again (should fail)
	err = mgr.Remove(ctx, "test-service")
	if err == nil {
		t.Error("Second Remove should fail for non-existent service")
	}
}

// ═══════════════════════════════════════════════════════════════════════════════
// UpResult and AddResult Edge Cases
// ═══════════════════════════════════════════════════════════════════════════════

func TestUpResultWithEmptyServices(t *testing.T) {
	result := UpResult{
		Services:   []ServiceStatus{},
		AllHealthy: true,
	}

	data, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("Failed to marshal empty UpResult: %v", err)
	}

	var decoded UpResult
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if decoded.Services == nil {
		t.Error("Decoded Services should not be nil")
	}
	if len(decoded.Services) != 0 {
		t.Error("Decoded Services should be empty")
	}
}

func TestUpResultWithNilServices(t *testing.T) {
	result := UpResult{
		Services:   nil,
		AllHealthy: false,
	}

	data, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("Failed to marshal UpResult with nil services: %v", err)
	}

	var decoded UpResult
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}
}

func TestAddResultAllFields(t *testing.T) {
	result := AddResult{
		Name:    "",
		Added:   false,
		Message: "",
	}

	// Should handle all empty/zero values
	data, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("Failed to marshal AddResult: %v", err)
	}

	if string(data) == "" {
		t.Error("JSON should not be empty")
	}
}
