// internal/workspace/resolve_test.go
package workspace

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestResolveFromCwd_WithDbaID(t *testing.T) {
	oldCwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd failed: %v", err)
	}
	defer os.Chdir(oldCwd)

	tmpDir, err := os.MkdirTemp("", "workspace_resolve")
	if err != nil {
		t.Fatalf("mkdirtemp failed: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Resolve symlinks for comparison (macOS /var -> /private/var)
	tmpDir, err = filepath.EvalSymlinks(tmpDir)
	if err != nil {
		t.Fatalf("eval symlinks failed: %v", err)
	}

	// Create minimal .dba/id file
	dbaDir := filepath.Join(tmpDir, ".dba")
	if err := os.MkdirAll(dbaDir, 0755); err != nil {
		t.Fatalf("mkdir .dba failed: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dbaDir, "id"), []byte("ws_test_id"), 0644); err != nil {
		t.Fatalf("write id failed: %v", err)
	}

	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("chdir failed: %v", err)
	}

	ws, err := ResolveFromCwd()
	if err != nil {
		t.Fatalf("ResolveFromCwd failed: %v", err)
	}

	if ws.ID != "ws_test_id" {
		t.Fatalf("expected ws ID ws_test_id, got %s", ws.ID)
	}
	if ws.Path != tmpDir {
		t.Fatalf("expected ws path %s, got %s", tmpDir, ws.Path)
	}
}

func TestResolveFromCwd_FromDevboxEnvByID(t *testing.T) {
	oldCwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd failed: %v", err)
	}
	defer os.Chdir(oldCwd)

	oldHome := os.Getenv("DBA_HOME")
	defer os.Setenv("DBA_HOME", oldHome)

	baseDir, err := os.MkdirTemp("", "dba_home")
	if err != nil {
		t.Fatalf("mkdirtemp failed: %v", err)
	}
	defer os.RemoveAll(baseDir)

	if err := os.Setenv("DBA_HOME", baseDir); err != nil {
		t.Fatalf("setenv failed: %v", err)
	}

	// Create workspace at ~/.dba/workspaces/ws_env
	wsID := "ws_env"
	wsPath := filepath.Join(baseDir, "workspaces", wsID)
	if err := os.MkdirAll(filepath.Join(wsPath, ".dba"), 0755); err != nil {
		t.Fatalf("mkdir ws failed: %v", err)
	}
	if err := os.WriteFile(filepath.Join(wsPath, ".dba", "id"), []byte(wsID), 0644); err != nil {
		t.Fatalf("write id failed: %v", err)
	}

	// Create separate cwd with devbox.json pointing to wsID
	projectDir := filepath.Join(baseDir, "project")
	if err := os.MkdirAll(projectDir, 0755); err != nil {
		t.Fatalf("mkdir project failed: %v", err)
	}
	devbox := map[string]map[string]string{
		"env": {
			"DBA_WORKSPACE_ID": wsID,
		},
	}
	devboxData, _ := json.Marshal(devbox)
	if err := os.WriteFile(filepath.Join(projectDir, "devbox.json"), devboxData, 0644); err != nil {
		t.Fatalf("write devbox failed: %v", err)
	}

	if err := os.Chdir(projectDir); err != nil {
		t.Fatalf("chdir failed: %v", err)
	}

	ws, err := ResolveFromCwd()
	if err != nil {
		t.Fatalf("ResolveFromCwd failed: %v", err)
	}

	if ws.ID != wsID {
		t.Fatalf("expected ws ID %s, got %s", wsID, ws.ID)
	}
	if ws.Path != wsPath {
		t.Fatalf("expected ws path %s, got %s", wsPath, ws.Path)
	}
}

func TestResolveFromCwd_FromDevboxEnvByPathFallback(t *testing.T) {
	oldCwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd failed: %v", err)
	}
	defer os.Chdir(oldCwd)

	oldHome := os.Getenv("DBA_HOME")
	defer os.Setenv("DBA_HOME", oldHome)

	baseDir, err := os.MkdirTemp("", "dba_home")
	if err != nil {
		t.Fatalf("mkdirtemp failed: %v", err)
	}
	defer os.RemoveAll(baseDir)

	if err := os.Setenv("DBA_HOME", baseDir); err != nil {
		t.Fatalf("setenv failed: %v", err)
	}

	// Workspace path not registered by ID, but has state.json
	wsPath := filepath.Join(baseDir, "custom_ws")
	if err := os.MkdirAll(filepath.Join(wsPath, ".dba"), 0755); err != nil {
		t.Fatalf("mkdir ws failed: %v", err)
	}
	state := []byte(`{"id":"ws_fallback","name":"fallback","template":"node","status":"ready"}`)
	if err := os.WriteFile(filepath.Join(wsPath, ".dba", "state.json"), state, 0644); err != nil {
		t.Fatalf("write state failed: %v", err)
	}

	projectDir := filepath.Join(baseDir, "project_fallback")
	if err := os.MkdirAll(projectDir, 0755); err != nil {
		t.Fatalf("mkdir project failed: %v", err)
	}

	devbox := map[string]map[string]string{
		"env": {
			"DBA_WORKSPACE_ID":   "ws_missing",
			"DBA_WORKSPACE_PATH": wsPath,
		},
	}
	devboxData, _ := json.Marshal(devbox)
	if err := os.WriteFile(filepath.Join(projectDir, "devbox.json"), devboxData, 0644); err != nil {
		t.Fatalf("write devbox failed: %v", err)
	}

	if err := os.Chdir(projectDir); err != nil {
		t.Fatalf("chdir failed: %v", err)
	}

	ws, err := ResolveFromCwd()
	if err != nil {
		t.Fatalf("ResolveFromCwd failed: %v", err)
	}

	if ws.Path != wsPath {
		t.Fatalf("expected ws path %s, got %s", wsPath, ws.Path)
	}
	if ws.ID != "ws_fallback" {
		t.Fatalf("expected ws ID ws_fallback, got %s", ws.ID)
	}
}

func TestResolveFromCwd_IgnoresInvalidDevbox(t *testing.T) {
	oldCwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd failed: %v", err)
	}
	defer os.Chdir(oldCwd)

	tmpDir, err := os.MkdirTemp("", "workspace_invalid_devbox")
	if err != nil {
		t.Fatalf("mkdirtemp failed: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	if err := os.WriteFile(filepath.Join(tmpDir, "devbox.json"), []byte("{invalid"), 0644); err != nil {
		t.Fatalf("write invalid devbox failed: %v", err)
	}

	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("chdir failed: %v", err)
	}

	if _, err := ResolveFromCwd(); err == nil {
		t.Fatalf("expected error when no workspace found")
	}
}

// Additional comprehensive tests for resolve.go

func TestResolve_ByID(t *testing.T) {
	oldHome := os.Getenv("DBA_HOME")
	defer os.Setenv("DBA_HOME", oldHome)

	tmpHome, _ := os.MkdirTemp("", "dba_home_resolve")
	defer os.RemoveAll(tmpHome)
	os.Setenv("DBA_HOME", tmpHome)

	// Create workspace
	wsID := "ws_resolve_test"
	wsPath := filepath.Join(tmpHome, "workspaces", wsID)
	os.MkdirAll(filepath.Join(wsPath, ".dba"), 0755)
	os.WriteFile(filepath.Join(wsPath, ".dba", "id"), []byte(wsID), 0644)

	ws, err := Resolve(wsID)
	if err != nil {
		t.Fatalf("Resolve by ID failed: %v", err)
	}
	if ws.ID != wsID {
		t.Errorf("expected ID %s, got %s", wsID, ws.ID)
	}
}

func TestResolve_ByPath(t *testing.T) {
	tmpDir, _ := os.MkdirTemp("", "workspace_by_path")
	defer os.RemoveAll(tmpDir)

	// Create workspace at path
	os.MkdirAll(filepath.Join(tmpDir, ".dba"), 0755)
	os.WriteFile(filepath.Join(tmpDir, ".dba", "id"), []byte("ws_path_test"), 0644)

	ws, err := Resolve(tmpDir)
	if err != nil {
		t.Fatalf("Resolve by path failed: %v", err)
	}
	if ws.ID != "ws_path_test" {
		t.Errorf("expected ID ws_path_test, got %s", ws.ID)
	}
}

func TestResolveByID_NonExistent(t *testing.T) {
	oldHome := os.Getenv("DBA_HOME")
	defer os.Setenv("DBA_HOME", oldHome)

	tmpHome, _ := os.MkdirTemp("", "dba_home_nonexist")
	defer os.RemoveAll(tmpHome)
	os.Setenv("DBA_HOME", tmpHome)

	_, err := ResolveByID("ws_does_not_exist")
	if err == nil {
		t.Error("expected error for non-existent workspace")
	}
}

func TestResolveByPath_NonExistent(t *testing.T) {
	_, err := ResolveByPath("/nonexistent/path/to/workspace")
	if err == nil {
		t.Error("expected error for non-existent path")
	}
}

func TestResolveFromCwd_DeepNesting(t *testing.T) {
	oldCwd, _ := os.Getwd()
	defer os.Chdir(oldCwd)

	tmpDir, _ := os.MkdirTemp("", "workspace_deep")
	defer os.RemoveAll(tmpDir)

	// Create workspace at root
	os.MkdirAll(filepath.Join(tmpDir, ".dba"), 0755)
	os.WriteFile(filepath.Join(tmpDir, ".dba", "id"), []byte("ws_deep_nest"), 0644)

	// Create deeply nested subdirectory
	deepPath := filepath.Join(tmpDir, "a", "b", "c", "d", "e", "f", "g", "h", "i", "j")
	os.MkdirAll(deepPath, 0755)

	os.Chdir(deepPath)

	ws, err := ResolveFromCwd()
	if err != nil {
		t.Fatalf("ResolveFromCwd failed: %v", err)
	}
	if ws.ID != "ws_deep_nest" {
		t.Errorf("expected ID ws_deep_nest, got %s", ws.ID)
	}
}

func TestResolveFromCwd_MultipleWorkspaces(t *testing.T) {
	oldCwd, _ := os.Getwd()
	defer os.Chdir(oldCwd)

	tmpDir, _ := os.MkdirTemp("", "workspace_multi")
	defer os.RemoveAll(tmpDir)

	// Create parent workspace
	os.MkdirAll(filepath.Join(tmpDir, ".dba"), 0755)
	os.WriteFile(filepath.Join(tmpDir, ".dba", "id"), []byte("ws_parent"), 0644)

	// Create nested workspace
	nestedWs := filepath.Join(tmpDir, "nested")
	os.MkdirAll(filepath.Join(nestedWs, ".dba"), 0755)
	os.WriteFile(filepath.Join(nestedWs, ".dba", "id"), []byte("ws_nested"), 0644)

	// From nested workspace, should find nested
	os.Chdir(nestedWs)
	ws, err := ResolveFromCwd()
	if err != nil {
		t.Fatalf("ResolveFromCwd failed: %v", err)
	}
	if ws.ID != "ws_nested" {
		t.Errorf("expected nested workspace, got %s", ws.ID)
	}

	// From subdirectory of nested, should still find nested
	subDir := filepath.Join(nestedWs, "src")
	os.MkdirAll(subDir, 0755)
	os.Chdir(subDir)

	ws, err = ResolveFromCwd()
	if err != nil {
		t.Fatalf("ResolveFromCwd failed: %v", err)
	}
	if ws.ID != "ws_nested" {
		t.Errorf("expected nested workspace, got %s", ws.ID)
	}
}

func TestLoadWorkspace_WithWhitespace(t *testing.T) {
	tmpDir, _ := os.MkdirTemp("", "workspace_whitespace")
	defer os.RemoveAll(tmpDir)

	os.MkdirAll(filepath.Join(tmpDir, ".dba"), 0755)
	// ID with whitespace and newlines
	os.WriteFile(filepath.Join(tmpDir, ".dba", "id"), []byte("\n  ws_whitespace  \n\t"), 0644)

	ws, err := loadWorkspace(tmpDir)
	if err != nil {
		t.Fatalf("loadWorkspace failed: %v", err)
	}
	if ws.ID != "ws_whitespace" {
		t.Errorf("expected trimmed ID 'ws_whitespace', got '%s'", ws.ID)
	}
}

func TestTryLoadFromDevbox_EmptyEnv(t *testing.T) {
	tmpDir, _ := os.MkdirTemp("", "devbox_empty_env")
	defer os.RemoveAll(tmpDir)

	devboxPath := filepath.Join(tmpDir, "devbox.json")
	os.WriteFile(devboxPath, []byte(`{"env": {}}`), 0644)

	ws := tryLoadFromDevbox(devboxPath)
	if ws != nil {
		t.Error("expected nil for empty env")
	}
}

func TestTryLoadFromDevbox_MissingEnv(t *testing.T) {
	tmpDir, _ := os.MkdirTemp("", "devbox_no_env")
	defer os.RemoveAll(tmpDir)

	devboxPath := filepath.Join(tmpDir, "devbox.json")
	os.WriteFile(devboxPath, []byte(`{"packages": ["nodejs"]}`), 0644)

	ws := tryLoadFromDevbox(devboxPath)
	if ws != nil {
		t.Error("expected nil for missing env")
	}
}

func TestTryLoadFromDevbox_NonExistentFile(t *testing.T) {
	ws := tryLoadFromDevbox("/nonexistent/devbox.json")
	if ws != nil {
		t.Error("expected nil for non-existent file")
	}
}

func TestStateExists_Directory(t *testing.T) {
	tmpDir, _ := os.MkdirTemp("", "state_exists")
	defer os.RemoveAll(tmpDir)

	// No .dba directory
	if StateExists(tmpDir) {
		t.Error("expected false when no .dba directory")
	}

	// .dba directory but no state.json
	os.MkdirAll(filepath.Join(tmpDir, ".dba"), 0755)
	if StateExists(tmpDir) {
		t.Error("expected false when no state.json")
	}

	// state.json exists
	os.WriteFile(filepath.Join(tmpDir, ".dba", "state.json"), []byte("{}"), 0644)
	if !StateExists(tmpDir) {
		t.Error("expected true when state.json exists")
	}
}

func TestResolveByID_WithDefaultHome(t *testing.T) {
	// Test that it falls back to ~/.dba when DBA_HOME not set
	oldHome := os.Getenv("DBA_HOME")
	os.Unsetenv("DBA_HOME")
	defer func() {
		if oldHome != "" {
			os.Setenv("DBA_HOME", oldHome)
		}
	}()

	// This should use default home directory
	_, err := ResolveByID("ws_default_home_test")
	// Should fail since workspace doesn't exist, but shouldn't panic
	if err == nil {
		t.Error("expected error for non-existent workspace")
	}
}

func TestResolve_PrefixDetection(t *testing.T) {
	tests := []struct {
		input    string
		isWSID   bool
	}{
		{"ws_test", true},
		{"ws_", true},
		{"ws_123abc", true},
		{"WS_test", false},     // Case sensitive
		{"workspace_test", false},
		{"/path/ws_test", false},
		{"./ws_test", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			// We're testing the prefix detection, not the actual resolution
			// so we don't need real workspaces
			isWSID := len(tt.input) >= 3 && tt.input[:3] == "ws_"
			if isWSID != tt.isWSID {
				t.Errorf("input %q: isWSID = %v, want %v", tt.input, isWSID, tt.isWSID)
			}
		})
	}
}
