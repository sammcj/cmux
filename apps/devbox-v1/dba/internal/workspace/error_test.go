// internal/workspace/error_test.go
package workspace

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/dba-cli/dba/internal/config"
	"github.com/dba-cli/dba/internal/db"
)

// TestCreateWithUnknownTemplateFallsBackToNode tests that unknown templates fall back to node
func TestCreateWithUnknownTemplateFallsBackToNode(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "error_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Setup test environment
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

	// Create with unknown template - should fall back to node
	opts := CreateOptions{
		Name:     "unknown-template-test",
		Template: "nonexistent_template",
		Dir:      tmpDir,
	}

	ws, err := Create(cfg, opts)
	if err != nil {
		t.Fatalf("Create should succeed with fallback to node: %v", err)
	}

	// Template should still be recorded as requested
	if ws.Template != "nonexistent_template" {
		t.Errorf("expected Template = nonexistent_template, got %s", ws.Template)
	}
}

// TestDestroyNonExistentWorkspace tests destroy on non-existent workspace
func TestDestroyNonExistentWorkspace(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "destroy_error_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	cfg := &config.Config{}

	// Try to destroy non-existent workspace
	ws := &Workspace{
		ID:   "ws_nonexistent",
		Path: "/path/that/does/not/exist",
	}

	// Destroy should not panic, but may return an error
	err = Destroy(cfg, ws, DestroyOptions{KeepFiles: false})
	// We just want to make sure it doesn't panic
	_ = err
}

// TestLoadFromInvalidPath tests loading workspace from invalid path
func TestLoadFromInvalidPath(t *testing.T) {
	_, err := Load("/nonexistent/path")
	if err == nil {
		t.Error("Load should fail for non-existent path")
	}
}

// TestSaveStateToReadOnlyDir tests saving state when directory is read-only
func TestSaveStateToReadOnlyDir(t *testing.T) {
	// Skip on CI where permissions might not work as expected
	if os.Getenv("CI") != "" {
		t.Skip("Skipping permission test in CI")
	}

	tmpDir, err := os.MkdirTemp("", "readonly_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create .dba directory
	dbaDir := filepath.Join(tmpDir, ".dba")
	if err := os.MkdirAll(dbaDir, 0755); err != nil {
		t.Fatal(err)
	}

	ws := &Workspace{
		ID:         "ws_readonly_test",
		Name:       "readonly-test",
		Path:       tmpDir,
		Template:   "node",
		Status:     "ready",
		Ports:      map[string]int{"PORT": 10000},
		CreatedAt:  time.Now(),
		LastActive: time.Now(),
	}

	// Make directory read-only
	if err := os.Chmod(dbaDir, 0444); err != nil {
		t.Fatal(err)
	}
	defer os.Chmod(dbaDir, 0755) // Restore for cleanup

	// Try to save state - should fail
	err = ws.SaveState()
	if err == nil {
		t.Error("SaveState should fail for read-only directory")
	}
}

// TestLoadCorruptedState tests loading workspace with corrupted state.json
func TestLoadCorruptedState(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "corrupted_state_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create .dba directory with corrupted state.json
	dbaDir := filepath.Join(tmpDir, ".dba")
	if err := os.MkdirAll(dbaDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Write corrupted JSON
	statePath := filepath.Join(dbaDir, "state.json")
	if err := os.WriteFile(statePath, []byte("{ corrupted json }"), 0644); err != nil {
		t.Fatal(err)
	}

	// Load should fail with corrupted state
	_, err = Load(tmpDir)
	if err == nil {
		t.Error("Load should fail for corrupted state.json")
	}
}

// TestInitAlreadyInitialized tests init on already-initialized directory
func TestInitAlreadyInitialized(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "already_init_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Setup test environment
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

	// Create initial workspace
	projectDir := filepath.Join(tmpDir, "project")
	if err := os.MkdirAll(projectDir, 0755); err != nil {
		t.Fatal(err)
	}

	// First init should succeed
	ws1, err := Init(cfg, projectDir, InitOptions{Name: "first-init"})
	if err != nil {
		t.Fatalf("First Init failed: %v", err)
	}
	t.Logf("First workspace created: %s", ws1.ID)

	// Second init on same directory should fail
	_, err = Init(cfg, projectDir, InitOptions{Name: "second-init"})
	if err == nil {
		t.Error("Init should fail on already-initialized directory")
	}
}

// TestCloneNonExistentSource tests cloning from non-existent source
func TestCloneNonExistentSource(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "clone_error_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	oldHome := os.Getenv("DBA_HOME")
	os.Setenv("DBA_HOME", tmpDir)
	defer os.Setenv("DBA_HOME", oldHome)

	cfg := &config.Config{
		Ports: config.PortConfig{
			RangeStart: 30000,
			RangeEnd:   39999,
			BlockSize:  100,
		},
	}

	// Try to clone from non-existent source workspace
	sourceWs := &Workspace{
		ID:          "ws_nonexistent_source",
		Name:        "nonexistent",
		Path:        "/nonexistent/source/path",
		ProjectPath: "/nonexistent/source/path/project",
		Template:    "node",
		Ports:       map[string]int{"PORT": 10000},
	}

	// Clone should handle non-existent source gracefully
	_, err = Clone(cfg, sourceWs, CloneOptions{Name: "clone"})
	// Clone may fail when trying to copy files - that's expected
	// Just make sure it doesn't panic
	t.Logf("Clone error (expected): %v", err)
}

// TestStateExistsForNonExistentPath tests StateExists with non-existent path
func TestStateExistsForNonExistentPath(t *testing.T) {
	result := StateExists("/nonexistent/path")
	if result {
		t.Error("StateExists should return false for non-existent path")
	}
}

// TestStateExistsWithoutStateFile tests StateExists when state.json doesn't exist
func TestStateExistsWithoutStateFile(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "no_state_file_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create .dba directory but no state.json
	dbaDir := filepath.Join(tmpDir, ".dba")
	if err := os.MkdirAll(dbaDir, 0755); err != nil {
		t.Fatal(err)
	}

	result := StateExists(tmpDir)
	if result {
		t.Error("StateExists should return false when state.json doesn't exist")
	}
}

// TestListWithNoWorkspaces tests List when there are no workspaces
func TestListWithNoWorkspaces(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "list_empty_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	oldHome := os.Getenv("DBA_HOME")
	os.Setenv("DBA_HOME", tmpDir)
	defer os.Setenv("DBA_HOME", oldHome)

	// Reset database singleton for test isolation
	db.ResetForTesting()

	cfg := &config.Config{}

	// Create workspaces directory
	workspacesDir := filepath.Join(tmpDir, "workspaces")
	if err := os.MkdirAll(workspacesDir, 0755); err != nil {
		t.Fatal(err)
	}

	workspaces, err := List(cfg, ListOptions{})
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}

	if len(workspaces) != 0 {
		t.Errorf("expected 0 workspaces, got %d", len(workspaces))
	}
}

// TestListWithStatusFilter tests List with status filter that matches nothing
func TestListWithStatusFilter(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "list_filter_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	oldHome := os.Getenv("DBA_HOME")
	os.Setenv("DBA_HOME", tmpDir)
	defer os.Setenv("DBA_HOME", oldHome)

	cfg := &config.Config{}

	// Create workspaces directory with a workspace
	workspacesDir := filepath.Join(tmpDir, "workspaces")
	wsPath := filepath.Join(workspacesDir, "ws_filter_test")
	dbaDir := filepath.Join(wsPath, ".dba")
	if err := os.MkdirAll(dbaDir, 0755); err != nil {
		t.Fatal(err)
	}

	ws := &Workspace{
		ID:         "ws_filter_test",
		Name:       "filter-test",
		Path:       wsPath,
		Template:   "node",
		Status:     "ready",
		Ports:      map[string]int{"PORT": 10000},
		CreatedAt:  time.Now(),
		LastActive: time.Now(),
	}
	if err := ws.SaveState(); err != nil {
		t.Fatal(err)
	}

	// List with filter that doesn't match
	workspaces, err := List(cfg, ListOptions{Status: "running"})
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}

	if len(workspaces) != 0 {
		t.Errorf("expected 0 workspaces with status 'running', got %d", len(workspaces))
	}
}

// TestResolveFromCwdWithSymlinks tests resolution with symlinked paths
func TestResolveFromCwdWithSymlinks(t *testing.T) {
	// Skip on Windows where symlinks might not work
	if os.Getenv("CI") != "" && os.Getenv("RUNNER_OS") == "Windows" {
		t.Skip("Skipping symlink test on Windows CI")
	}

	tmpDir, err := os.MkdirTemp("", "symlink_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create workspace
	wsPath := filepath.Join(tmpDir, "workspace")
	dbaDir := filepath.Join(wsPath, ".dba")
	if err := os.MkdirAll(dbaDir, 0755); err != nil {
		t.Fatal(err)
	}

	wsID := "ws_symlink_test"
	if err := os.WriteFile(filepath.Join(dbaDir, "id"), []byte(wsID), 0644); err != nil {
		t.Fatal(err)
	}

	ws := &Workspace{
		ID:         wsID,
		Name:       "symlink-test",
		Path:       wsPath,
		Template:   "node",
		Status:     "ready",
		Ports:      map[string]int{"PORT": 10000},
		CreatedAt:  time.Now(),
		LastActive: time.Now(),
	}
	if err := ws.SaveState(); err != nil {
		t.Fatal(err)
	}

	// Create symlink to workspace
	symlinkPath := filepath.Join(tmpDir, "workspace-link")
	if err := os.Symlink(wsPath, symlinkPath); err != nil {
		t.Skip("Cannot create symlink, skipping test")
	}

	// Resolve from symlink path
	resolved, err := ResolveByPath(symlinkPath)
	if err != nil {
		t.Fatalf("ResolveByPath(symlink) failed: %v", err)
	}

	if resolved.ID != wsID {
		t.Errorf("expected ID = %s, got %s", wsID, resolved.ID)
	}
}

// TestWorkspaceURLsWithNoPorts tests URLs() method when no ports are allocated
func TestWorkspaceURLsWithNoPorts(t *testing.T) {
	ws := &Workspace{
		ID:     "ws_no_ports",
		Name:   "no-ports-test",
		Ports:  nil,
		Status: "ready",
	}

	urls := ws.URLs()
	if urls == nil {
		t.Error("URLs() should return non-nil map even when no ports")
	}
	if len(urls) != 0 {
		t.Errorf("expected 0 URLs, got %d", len(urls))
	}
}

// TestWorkspaceURLsWithEmptyPorts tests URLs() method when ports map is empty
func TestWorkspaceURLsWithEmptyPorts(t *testing.T) {
	ws := &Workspace{
		ID:     "ws_empty_ports",
		Name:   "empty-ports-test",
		Ports:  map[string]int{},
		Status: "ready",
	}

	urls := ws.URLs()
	if urls == nil {
		t.Error("URLs() should return non-nil map when ports is empty")
	}
	if len(urls) != 0 {
		t.Errorf("expected 0 URLs, got %d", len(urls))
	}
}

// TestWorkspaceDevboxPathWithEmptyPath tests DevboxPath() when Path is empty
func TestWorkspaceDevboxPathWithEmptyPath(t *testing.T) {
	ws := &Workspace{
		ID:   "ws_empty_path",
		Path: "",
	}

	// Should return path relative to empty string (just devbox.json)
	result := ws.DevboxPath()
	expected := filepath.Join("", "devbox.json")
	if result != expected {
		t.Errorf("expected %s, got %s", expected, result)
	}
}

// TestEmptyPortNames tests creating workspace with empty port names slice
func TestEmptyPortNames(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "empty_ports_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	oldHome := os.Getenv("DBA_HOME")
	os.Setenv("DBA_HOME", tmpDir)
	defer os.Setenv("DBA_HOME", oldHome)

	cfg := &config.Config{
		Defaults: config.DefaultsConfig{
			Ports: []string{}, // Empty ports
		},
		Ports: config.PortConfig{
			RangeStart: 30000,
			RangeEnd:   39999,
			BlockSize:  100,
		},
	}

	opts := CreateOptions{
		Name:      "empty-ports",
		Template:  "node",
		Dir:       tmpDir,
		PortNames: []string{}, // Empty
	}

	// Create may still succeed with no ports
	ws, err := Create(cfg, opts)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// Should have no ports
	if len(ws.Ports) != 0 {
		t.Errorf("expected 0 ports, got %d", len(ws.Ports))
	}
}
