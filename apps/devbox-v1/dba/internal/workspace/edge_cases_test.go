// internal/workspace/edge_cases_test.go
package workspace

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/dba-cli/dba/internal/config"
)

// TestIDFormat verifies generated IDs have correct format
func TestIDFormat(t *testing.T) {
	for i := 0; i < 100; i++ {
		id := generateID()

		// Must start with "ws_"
		if !strings.HasPrefix(id, "ws_") {
			t.Errorf("ID doesn't start with 'ws_': %s", id)
		}

		// Must be exactly 11 characters (ws_ + 8 hex)
		if len(id) != 11 {
			t.Errorf("ID has wrong length: %s (len=%d)", id, len(id))
		}

		// Hex part must be valid hex
		hexPart := id[3:]
		for _, c := range hexPart {
			if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f')) {
				t.Errorf("ID contains non-hex character: %s", id)
				break
			}
		}
	}
}

// TestWorkspaceWithLongName tests workspace with very long name
func TestWorkspaceWithLongName(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "long_name_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	oldHome := os.Getenv("DBA_HOME")
	os.Setenv("DBA_HOME", tmpDir)
	defer os.Setenv("DBA_HOME", oldHome)

	cfg := &config.Config{
		Defaults: config.DefaultsConfig{
			Ports: []string{"PORT"},
		},
		Ports: config.PortConfig{
			RangeStart: 30000,
			RangeEnd:   39999,
			BlockSize:  100,
		},
	}

	// Create workspace with very long name
	longName := strings.Repeat("a", 256)
	opts := CreateOptions{
		Name:     longName,
		Template: "node",
		Dir:      tmpDir,
	}

	ws, err := Create(cfg, opts)
	if err != nil {
		t.Fatalf("Create failed with long name: %v", err)
	}

	if ws.Name != longName {
		t.Errorf("Long name was truncated or modified")
	}
}

// TestWorkspaceWithSpecialCharsInName tests workspace with special characters
func TestWorkspaceWithSpecialCharsInName(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "special_chars_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	oldHome := os.Getenv("DBA_HOME")
	os.Setenv("DBA_HOME", tmpDir)
	defer os.Setenv("DBA_HOME", oldHome)

	cfg := &config.Config{
		Defaults: config.DefaultsConfig{
			Ports: []string{"PORT"},
		},
		Ports: config.PortConfig{
			RangeStart: 30000,
			RangeEnd:   39999,
			BlockSize:  100,
		},
	}

	specialNames := []string{
		"my-app",
		"my_app",
		"my.app",
		"app 123",
		"app@test",
		"app#1",
		"日本語アプリ",
		"emoji-🚀-test",
	}

	for _, name := range specialNames {
		t.Run(name, func(t *testing.T) {
			opts := CreateOptions{
				Name:     name,
				Template: "node",
				Dir:      tmpDir,
			}

			ws, err := Create(cfg, opts)
			if err != nil {
				t.Fatalf("Create failed with name '%s': %v", name, err)
			}

			if ws.Name != name {
				t.Errorf("Name was modified: expected '%s', got '%s'", name, ws.Name)
			}
		})
	}
}

// TestWorkspaceWithManyPorts tests workspace with many ports
func TestWorkspaceWithManyPorts(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "many_ports_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	oldHome := os.Getenv("DBA_HOME")
	os.Setenv("DBA_HOME", tmpDir)
	defer os.Setenv("DBA_HOME", oldHome)

	// Create many port names
	numPorts := 50
	portNames := make([]string, numPorts)
	for i := 0; i < numPorts; i++ {
		portNames[i] = "PORT_" + string(rune('A'+i%26)) + string(rune('0'+i/26))
	}

	cfg := &config.Config{
		Defaults: config.DefaultsConfig{
			Ports: portNames,
		},
		Ports: config.PortConfig{
			RangeStart: 30000,
			RangeEnd:   39999,
			BlockSize:  100,
		},
	}

	opts := CreateOptions{
		Name:     "many-ports",
		Template: "node",
		Dir:      tmpDir,
	}

	ws, err := Create(cfg, opts)
	if err != nil {
		t.Fatalf("Create failed with many ports: %v", err)
	}

	if len(ws.Ports) != numPorts {
		t.Errorf("expected %d ports, got %d", numPorts, len(ws.Ports))
	}
}

// TestWorkspaceStateRoundTrip tests that state survives save/load cycle
func TestWorkspaceStateRoundTrip(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "roundtrip_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	dbaDir := filepath.Join(tmpDir, ".dba")
	if err := os.MkdirAll(dbaDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create workspace with all fields populated
	now := time.Now()
	original := &Workspace{
		ID:          "ws_roundtrip",
		Name:        "roundtrip-test",
		Path:        tmpDir,
		ProjectPath: filepath.Join(tmpDir, "project"),
		Template:    "python",
		Status:      "running",
		BasePort:    12345,
		Ports: map[string]int{
			"PORT":      12345,
			"CODE_PORT": 12346,
			"API_PORT":  12347,
		},
		Packages:   []string{"pkg1", "pkg2", "pkg3"},
		CreatedAt:  now.Add(-24 * time.Hour),
		LastActive: now,
		Git: &GitInfo{
			Remote: "https://github.com/test/repo",
			Branch: "main",
			Commit: "abc123def456",
		},
	}

	// Save
	if err := original.SaveState(); err != nil {
		t.Fatalf("SaveState failed: %v", err)
	}

	// Load
	loaded, err := Load(tmpDir)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	// Compare fields
	if loaded.ID != original.ID {
		t.Errorf("ID mismatch: %s vs %s", loaded.ID, original.ID)
	}
	if loaded.Name != original.Name {
		t.Errorf("Name mismatch: %s vs %s", loaded.Name, original.Name)
	}
	if loaded.Template != original.Template {
		t.Errorf("Template mismatch: %s vs %s", loaded.Template, original.Template)
	}
	if loaded.Status != original.Status {
		t.Errorf("Status mismatch: %s vs %s", loaded.Status, original.Status)
	}
	if loaded.BasePort != original.BasePort {
		t.Errorf("BasePort mismatch: %d vs %d", loaded.BasePort, original.BasePort)
	}
	if len(loaded.Ports) != len(original.Ports) {
		t.Errorf("Ports count mismatch: %d vs %d", len(loaded.Ports), len(original.Ports))
	}
	for k, v := range original.Ports {
		if loaded.Ports[k] != v {
			t.Errorf("Port %s mismatch: %d vs %d", k, loaded.Ports[k], v)
		}
	}
	if len(loaded.Packages) != len(original.Packages) {
		t.Errorf("Packages count mismatch: %d vs %d", len(loaded.Packages), len(original.Packages))
	}

	// Check Git info
	if loaded.Git == nil {
		t.Error("Git info was lost in roundtrip")
	} else {
		if loaded.Git.Remote != original.Git.Remote {
			t.Errorf("Git.Remote mismatch: %s vs %s", loaded.Git.Remote, original.Git.Remote)
		}
		if loaded.Git.Branch != original.Git.Branch {
			t.Errorf("Git.Branch mismatch: %s vs %s", loaded.Git.Branch, original.Git.Branch)
		}
		if loaded.Git.Commit != original.Git.Commit {
			t.Errorf("Git.Commit mismatch: %s vs %s", loaded.Git.Commit, original.Git.Commit)
		}
	}
}

// TestExtractRepoName tests repo name extraction from URLs
func TestExtractRepoName(t *testing.T) {
	tests := []struct {
		url      string
		expected string
	}{
		{"https://github.com/user/repo.git", "repo"},
		{"https://github.com/user/repo", "repo"},
		{"git@github.com:user/repo.git", "repo"},
		{"https://gitlab.com/group/subgroup/project.git", "project"},
		{"https://example.com/my-app.git", "my-app"},
		{"https://github.com/user/my_repo.git", "my_repo"},
	}

	for _, tc := range tests {
		t.Run(tc.url, func(t *testing.T) {
			result := extractRepoName(tc.url)
			if result != tc.expected {
				t.Errorf("extractRepoName(%s) = %s, expected %s", tc.url, result, tc.expected)
			}
		})
	}
}

// TestFindBasePort tests finding minimum port from map
func TestFindBasePort(t *testing.T) {
	tests := []struct {
		name     string
		ports    map[string]int
		expected int
	}{
		{
			name:     "single port",
			ports:    map[string]int{"PORT": 10000},
			expected: 10000,
		},
		{
			name:     "multiple ports",
			ports:    map[string]int{"PORT": 10000, "CODE_PORT": 10001, "API_PORT": 10002},
			expected: 10000,
		},
		{
			name:     "unordered ports",
			ports:    map[string]int{"A": 10005, "B": 10001, "C": 10003},
			expected: 10001,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := findBasePort(tc.ports)
			if result != tc.expected {
				t.Errorf("findBasePort() = %d, expected %d", result, tc.expected)
			}
		})
	}
}

// TestWorkspaceHelperMethods tests workspace helper methods
func TestWorkspaceHelperMethods(t *testing.T) {
	ws := &Workspace{
		ID:   "ws_helpers",
		Path: "/test/path",
	}

	// Test StateDir
	stateDir := ws.StateDir()
	expected := "/test/path/.dba"
	if stateDir != expected {
		t.Errorf("StateDir() = %s, expected %s", stateDir, expected)
	}

	// Test DevboxPath
	devboxPath := ws.DevboxPath()
	expected = "/test/path/devbox.json"
	if devboxPath != expected {
		t.Errorf("DevboxPath() = %s, expected %s", devboxPath, expected)
	}

	// Test ProcessComposePath
	pcPath := ws.ProcessComposePath()
	expected = "/test/path/process-compose.yaml"
	if pcPath != expected {
		t.Errorf("ProcessComposePath() = %s, expected %s", pcPath, expected)
	}
}

// TestWorkspaceURLGeneration tests URL generation for standard ports
func TestWorkspaceURLGeneration(t *testing.T) {
	ws := &Workspace{
		ID: "ws_urls",
		Ports: map[string]int{
			"PORT":      8080,
			"CODE_PORT": 9000,
			"VNC_PORT":  5900,
		},
	}

	urls := ws.URLs()

	// URLs() maps: PORT->app, CODE_PORT->code, VNC_PORT->vnc
	expectedURLs := map[string]string{
		"app":  "http://localhost:8080",
		"code": "http://localhost:9000",
		"vnc":  "vnc://localhost:5900",
	}

	if len(urls) != len(expectedURLs) {
		t.Errorf("expected %d URLs, got %d", len(expectedURLs), len(urls))
	}

	for name, expectedURL := range expectedURLs {
		if urls[name] != expectedURL {
			t.Errorf("URL for %s: expected %s, got %s", name, expectedURL, urls[name])
		}
	}
}

// TestListBuiltinTemplatesComplete tests that all expected templates are listed
func TestListBuiltinTemplatesComplete(t *testing.T) {
	templates := ListBuiltinTemplates()

	expectedTemplates := []string{"node", "nextjs", "python", "go", "react", "rust"}

	if len(templates) != len(expectedTemplates) {
		t.Errorf("expected %d templates, got %d", len(expectedTemplates), len(templates))
	}

	for _, expected := range expectedTemplates {
		found := false
		for _, tmpl := range templates {
			if tmpl == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("template '%s' not found in ListBuiltinTemplates()", expected)
		}
	}
}

// TestTemplateHasRequiredFields tests that all templates have required fields
func TestTemplateHasRequiredFields(t *testing.T) {
	templates := ListBuiltinTemplates()

	for _, name := range templates {
		t.Run(name, func(t *testing.T) {
			tmpl, err := LoadTemplate(name)
			if err != nil {
				t.Fatalf("LoadTemplate(%s) failed: %v", name, err)
			}

			if tmpl.Name != name {
				t.Errorf("Template name mismatch: expected %s, got %s", name, tmpl.Name)
			}

			if tmpl.ProcessCompose == "" {
				t.Errorf("Template %s has empty ProcessCompose", name)
			}

			if len(tmpl.Packages) == 0 {
				t.Errorf("Template %s has no packages", name)
			}
		})
	}
}

// TestStateExistsVariations tests StateExists with various scenarios
func TestStateExistsVariations(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "state_exists_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Empty directory
	emptyDir := filepath.Join(tmpDir, "empty")
	os.MkdirAll(emptyDir, 0755)
	if StateExists(emptyDir) {
		t.Error("StateExists should return false for empty directory")
	}

	// Directory with .dba but no state.json
	dbaDirOnly := filepath.Join(tmpDir, "dba_only")
	os.MkdirAll(filepath.Join(dbaDirOnly, ".dba"), 0755)
	if StateExists(dbaDirOnly) {
		t.Error("StateExists should return false when .dba exists but no state.json")
	}

	// Directory with .dba/state.json
	fullState := filepath.Join(tmpDir, "full_state")
	os.MkdirAll(filepath.Join(fullState, ".dba"), 0755)
	os.WriteFile(filepath.Join(fullState, ".dba", "state.json"), []byte("{}"), 0644)
	if !StateExists(fullState) {
		t.Error("StateExists should return true when state.json exists")
	}
}

// TestClonePreservesPackages tests that clone copies package list
func TestClonePreservesPackages(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "clone_packages_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	oldHome := os.Getenv("DBA_HOME")
	os.Setenv("DBA_HOME", tmpDir)
	defer os.Setenv("DBA_HOME", oldHome)

	cfg := &config.Config{
		Defaults: config.DefaultsConfig{
			Ports: []string{"PORT"},
		},
		Ports: config.PortConfig{
			RangeStart: 30000,
			RangeEnd:   39999,
			BlockSize:  100,
		},
	}

	// Create source workspace with packages
	opts := CreateOptions{
		Name:     "source",
		Template: "node",
		Dir:      tmpDir,
		Packages: []string{"custom-pkg-1", "custom-pkg-2"},
	}

	sourceWs, err := Create(cfg, opts)
	if err != nil {
		t.Fatalf("Create source failed: %v", err)
	}

	// Clone
	clonedWs, err := Clone(cfg, sourceWs, CloneOptions{Name: "cloned"})
	if err != nil {
		t.Fatalf("Clone failed: %v", err)
	}

	// Verify packages were copied
	if len(clonedWs.Packages) != len(sourceWs.Packages) {
		t.Errorf("Package count mismatch: source=%d, clone=%d", len(sourceWs.Packages), len(clonedWs.Packages))
	}

	for i, pkg := range sourceWs.Packages {
		if clonedWs.Packages[i] != pkg {
			t.Errorf("Package %d mismatch: source=%s, clone=%s", i, pkg, clonedWs.Packages[i])
		}
	}
}

// TestCloneHasNewBasePort tests that clone gets new port allocation
func TestCloneHasNewBasePort(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "clone_ports_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	oldHome := os.Getenv("DBA_HOME")
	os.Setenv("DBA_HOME", tmpDir)
	defer os.Setenv("DBA_HOME", oldHome)

	cfg := &config.Config{
		Defaults: config.DefaultsConfig{
			Ports: []string{"PORT", "CODE_PORT"},
		},
		Ports: config.PortConfig{
			RangeStart: 30000,
			RangeEnd:   39999,
			BlockSize:  100,
		},
	}

	// Create source workspace
	opts := CreateOptions{
		Name:     "source",
		Template: "node",
		Dir:      tmpDir,
	}

	sourceWs, err := Create(cfg, opts)
	if err != nil {
		t.Fatalf("Create source failed: %v", err)
	}

	// Clone
	clonedWs, err := Clone(cfg, sourceWs, CloneOptions{Name: "cloned"})
	if err != nil {
		t.Fatalf("Clone failed: %v", err)
	}

	// Verify ports are different
	if clonedWs.BasePort == sourceWs.BasePort {
		t.Error("Clone should have different base port than source")
	}

	for name, sourcePort := range sourceWs.Ports {
		if clonedPort, exists := clonedWs.Ports[name]; exists {
			if clonedPort == sourcePort {
				t.Errorf("Clone port %s (%d) should be different from source", name, sourcePort)
			}
		}
	}
}
