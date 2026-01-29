// internal/workspace/integration_test.go
package workspace

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/dba-cli/dba/internal/config"
	"github.com/dba-cli/dba/internal/db"
)

// TestFullWorkspaceLifecycle tests the complete workspace lifecycle:
// create -> verify -> update -> destroy
func TestFullWorkspaceLifecycle(t *testing.T) {
	// Skip if running short tests
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Create temp directory for DBA_HOME
	tmpHome, err := os.MkdirTemp("", "dba_integration_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpHome)

	// Set DBA_HOME
	oldHome := os.Getenv("DBA_HOME")
	os.Setenv("DBA_HOME", tmpHome)
	defer os.Setenv("DBA_HOME", oldHome)

	// Create necessary directories
	os.MkdirAll(filepath.Join(tmpHome, "workspaces"), 0755)

	// Load config
	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Step 1: Create workspace
	t.Log("Step 1: Creating workspace")
	ws, err := Create(cfg, CreateOptions{
		Name:     "integration-test",
		Template: "node",
	})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// Verify workspace was created
	if ws.ID == "" {
		t.Error("Workspace ID should not be empty")
	}
	if !strings.HasPrefix(ws.ID, "ws_") {
		t.Errorf("Workspace ID should start with 'ws_', got %s", ws.ID)
	}
	if ws.Name != "integration-test" {
		t.Errorf("Expected name 'integration-test', got %s", ws.Name)
	}
	if ws.Template != "node" {
		t.Errorf("Expected template 'node', got %s", ws.Template)
	}
	if ws.Status != "ready" {
		t.Errorf("Expected status 'ready', got %s", ws.Status)
	}

	// Step 2: Verify files were created
	t.Log("Step 2: Verifying files")

	// Check .dba directory
	dbaDir := filepath.Join(ws.Path, ".dba")
	if _, err := os.Stat(dbaDir); os.IsNotExist(err) {
		t.Error(".dba directory should exist")
	}

	// Check ID file
	idFile := filepath.Join(dbaDir, "id")
	idContent, err := os.ReadFile(idFile)
	if err != nil {
		t.Errorf("Failed to read ID file: %v", err)
	}
	if strings.TrimSpace(string(idContent)) != ws.ID {
		t.Errorf("ID file content mismatch: expected %s, got %s", ws.ID, string(idContent))
	}

	// Check state.json
	stateFile := filepath.Join(dbaDir, "state.json")
	if _, err := os.Stat(stateFile); os.IsNotExist(err) {
		t.Error("state.json should exist")
	}

	// Check devbox.json
	devboxFile := filepath.Join(ws.Path, "devbox.json")
	if _, err := os.Stat(devboxFile); os.IsNotExist(err) {
		t.Error("devbox.json should exist")
	}

	// Check process-compose.yaml
	pcFile := filepath.Join(ws.Path, "process-compose.yaml")
	if _, err := os.Stat(pcFile); os.IsNotExist(err) {
		t.Error("process-compose.yaml should exist")
	}

	// Check project directory
	projectDir := filepath.Join(ws.Path, "project")
	if _, err := os.Stat(projectDir); os.IsNotExist(err) {
		t.Error("project directory should exist")
	}

	// Check logs directory
	logsDir := filepath.Join(dbaDir, "logs")
	if _, err := os.Stat(logsDir); os.IsNotExist(err) {
		t.Error("logs directory should exist")
	}

	// Step 3: Verify ports were allocated
	t.Log("Step 3: Verifying ports")
	if len(ws.Ports) == 0 {
		t.Error("Ports should be allocated")
	}
	if _, ok := ws.Ports["PORT"]; !ok {
		t.Error("PORT should be allocated")
	}
	if _, ok := ws.Ports["CODE_PORT"]; !ok {
		t.Error("CODE_PORT should be allocated")
	}

	// Step 4: Resolve workspace
	t.Log("Step 4: Testing resolution")

	// Resolve by ID
	resolvedWs, err := ResolveByID(ws.ID)
	if err != nil {
		t.Errorf("ResolveByID failed: %v", err)
	}
	if resolvedWs.ID != ws.ID {
		t.Errorf("Resolved workspace ID mismatch: expected %s, got %s", ws.ID, resolvedWs.ID)
	}

	// Resolve by path
	resolvedWs, err = ResolveByPath(ws.Path)
	if err != nil {
		t.Errorf("ResolveByPath failed: %v", err)
	}
	if resolvedWs.ID != ws.ID {
		t.Errorf("Resolved workspace ID mismatch: expected %s, got %s", ws.ID, resolvedWs.ID)
	}

	// Step 5: Update workspace
	t.Log("Step 5: Testing updates")
	oldLastActive := ws.LastActive
	time.Sleep(10 * time.Millisecond)
	if err := ws.UpdateLastActive(); err != nil {
		t.Errorf("UpdateLastActive failed: %v", err)
	}
	if !ws.LastActive.After(oldLastActive) {
		t.Error("LastActive should be updated")
	}

	if err := ws.SetStatus("running"); err != nil {
		t.Errorf("SetStatus failed: %v", err)
	}
	if ws.Status != "running" {
		t.Errorf("Status should be 'running', got %s", ws.Status)
	}

	// Verify persistence
	reloaded, err := Load(ws.Path)
	if err != nil {
		t.Errorf("Load failed: %v", err)
	}
	if reloaded.Status != "running" {
		t.Errorf("Persisted status should be 'running', got %s", reloaded.Status)
	}

	// Step 6: List workspaces
	t.Log("Step 6: Testing list")
	workspaces, err := List(cfg, ListOptions{})
	if err != nil {
		t.Errorf("List failed: %v", err)
	}
	found := false
	for _, w := range workspaces {
		if w.ID == ws.ID {
			found = true
			break
		}
	}
	if !found {
		t.Error("Created workspace should appear in list")
	}

	// Step 7: Destroy workspace
	t.Log("Step 7: Destroying workspace")
	if err := Destroy(cfg, ws, DestroyOptions{}); err != nil {
		t.Errorf("Destroy failed: %v", err)
	}

	// Verify workspace was destroyed
	if _, err := os.Stat(ws.Path); !os.IsNotExist(err) {
		t.Error("Workspace directory should be deleted")
	}

	// Verify workspace no longer in list
	workspaces, err = List(cfg, ListOptions{})
	if err != nil {
		t.Errorf("List after destroy failed: %v", err)
	}
	for _, w := range workspaces {
		if w.ID == ws.ID {
			t.Error("Destroyed workspace should not appear in list")
		}
	}

	t.Log("Integration test completed successfully")
}

// TestMultipleWorkspaces tests creating and managing multiple workspaces
func TestMultipleWorkspaces(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Create temp directory for DBA_HOME
	tmpHome, err := os.MkdirTemp("", "dba_multi_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpHome)

	oldHome := os.Getenv("DBA_HOME")
	os.Setenv("DBA_HOME", tmpHome)
	defer os.Setenv("DBA_HOME", oldHome)

	// Reset database singleton for test isolation
	db.ResetForTesting()

	os.MkdirAll(filepath.Join(tmpHome, "workspaces"), 0755)

	cfg, err := config.Load()
	if err != nil {
		t.Fatal(err)
	}

	// Create multiple workspaces
	workspaces := make([]*Workspace, 0)
	templates := []string{"node", "python", "go"}

	for i, tmpl := range templates {
		ws, err := Create(cfg, CreateOptions{
			Name:     "multi-test-" + tmpl,
			Template: tmpl,
		})
		if err != nil {
			t.Fatalf("Failed to create workspace %d: %v", i, err)
		}
		workspaces = append(workspaces, ws)
	}

	// Verify all workspaces have unique IDs
	ids := make(map[string]bool)
	for _, ws := range workspaces {
		if ids[ws.ID] {
			t.Errorf("Duplicate workspace ID: %s", ws.ID)
		}
		ids[ws.ID] = true
	}

	// Verify all workspaces have different base ports
	basePorts := make(map[int]bool)
	for _, ws := range workspaces {
		if basePorts[ws.BasePort] {
			t.Errorf("Duplicate base port: %d", ws.BasePort)
		}
		basePorts[ws.BasePort] = true
	}

	// Verify list returns all workspaces
	listed, err := List(cfg, ListOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if len(listed) != len(workspaces) {
		t.Errorf("Expected %d workspaces in list, got %d", len(workspaces), len(listed))
	}

	// Clean up
	for _, ws := range workspaces {
		Destroy(cfg, ws, DestroyOptions{})
	}
}

// TestDevboxJsonContent verifies the generated devbox.json content
func TestDevboxJsonContent(t *testing.T) {
	tmpHome, err := os.MkdirTemp("", "dba_devbox_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpHome)

	oldHome := os.Getenv("DBA_HOME")
	os.Setenv("DBA_HOME", tmpHome)
	defer os.Setenv("DBA_HOME", oldHome)

	os.MkdirAll(filepath.Join(tmpHome, "workspaces"), 0755)

	cfg, err := config.Load()
	if err != nil {
		t.Fatal(err)
	}

	ws, err := Create(cfg, CreateOptions{
		Name:     "devbox-test",
		Template: "node",
		Packages: []string{"typescript", "eslint"},
	})
	if err != nil {
		t.Fatal(err)
	}
	defer Destroy(cfg, ws, DestroyOptions{})

	// Read devbox.json
	devboxContent, err := os.ReadFile(ws.DevboxPath())
	if err != nil {
		t.Fatal(err)
	}

	var devbox map[string]interface{}
	if err := json.Unmarshal(devboxContent, &devbox); err != nil {
		t.Fatalf("Failed to parse devbox.json: %v", err)
	}

	// Verify $schema
	if schema, ok := devbox["$schema"].(string); !ok || schema == "" {
		t.Error("devbox.json should have $schema")
	}

	// Verify packages
	packages, ok := devbox["packages"].([]interface{})
	if !ok {
		t.Fatal("devbox.json should have packages array")
	}

	// Check for expected packages
	packageStrings := make([]string, len(packages))
	for i, p := range packages {
		packageStrings[i] = p.(string)
	}

	hasOpenVSCodeServer := false
	hasProcessCompose := false
	hasNodejs := false
	hasTypescript := false
	hasEslint := false

	for _, p := range packageStrings {
		if strings.HasPrefix(p, "openvscode-server@") {
			hasOpenVSCodeServer = true
		}
		if strings.HasPrefix(p, "process-compose@") {
			hasProcessCompose = true
		}
		if strings.HasPrefix(p, "nodejs@") {
			hasNodejs = true
		}
		if p == "typescript" {
			hasTypescript = true
		}
		if p == "eslint" {
			hasEslint = true
		}
	}

	if !hasOpenVSCodeServer {
		t.Error("devbox.json should include openvscode-server")
	}
	if !hasProcessCompose {
		t.Error("devbox.json should include process-compose")
	}
	if !hasNodejs {
		t.Error("devbox.json should include nodejs for node template")
	}
	if !hasTypescript {
		t.Error("devbox.json should include typescript custom package")
	}
	if !hasEslint {
		t.Error("devbox.json should include eslint custom package")
	}

	// Verify env
	env, ok := devbox["env"].(map[string]interface{})
	if !ok {
		t.Fatal("devbox.json should have env object")
	}

	// Check port environment variables
	if _, ok := env["PORT"]; !ok {
		t.Error("env should include PORT")
	}
	if _, ok := env["CODE_PORT"]; !ok {
		t.Error("env should include CODE_PORT")
	}
	if _, ok := env["DBA_WORKSPACE_ID"]; !ok {
		t.Error("env should include DBA_WORKSPACE_ID")
	}
	if _, ok := env["DBA_WORKSPACE_PATH"]; !ok {
		t.Error("env should include DBA_WORKSPACE_PATH")
	}

	// Verify DBA_WORKSPACE_ID matches
	if env["DBA_WORKSPACE_ID"] != ws.ID {
		t.Errorf("DBA_WORKSPACE_ID should be %s, got %s", ws.ID, env["DBA_WORKSPACE_ID"])
	}

	// Verify shell config
	shell, ok := devbox["shell"].(map[string]interface{})
	if !ok {
		t.Fatal("devbox.json should have shell config")
	}

	if _, ok := shell["init_hook"]; !ok {
		t.Error("shell should have init_hook")
	}
	if _, ok := shell["scripts"]; !ok {
		t.Error("shell should have scripts")
	}
}

// TestProcessComposeYamlContent verifies the generated process-compose.yaml content
func TestProcessComposeYamlContent(t *testing.T) {
	tmpHome, err := os.MkdirTemp("", "dba_pc_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpHome)

	oldHome := os.Getenv("DBA_HOME")
	os.Setenv("DBA_HOME", tmpHome)
	defer os.Setenv("DBA_HOME", oldHome)

	os.MkdirAll(filepath.Join(tmpHome, "workspaces"), 0755)

	cfg, err := config.Load()
	if err != nil {
		t.Fatal(err)
	}

	templates := []string{"node", "python", "go", "nextjs", "react", "rust"}

	for _, tmpl := range templates {
		t.Run(tmpl, func(t *testing.T) {
			ws, err := Create(cfg, CreateOptions{
				Name:     "pc-test-" + tmpl,
				Template: tmpl,
			})
			if err != nil {
				t.Fatal(err)
			}
			defer Destroy(cfg, ws, DestroyOptions{})

			// Read process-compose.yaml
			pcContent, err := os.ReadFile(ws.ProcessComposePath())
			if err != nil {
				t.Fatal(err)
			}

			content := string(pcContent)

			// Verify version
			if !strings.Contains(content, "version:") {
				t.Error("process-compose.yaml should have version")
			}

			// Verify processes section
			if !strings.Contains(content, "processes:") {
				t.Error("process-compose.yaml should have processes section")
			}

			// Verify vscode process (openvscode-server)
			if !strings.Contains(content, "vscode:") {
				t.Error("process-compose.yaml should have vscode process")
			}

			// Verify environment variable placeholders
			if !strings.Contains(content, "${CODE_PORT}") {
				t.Error("process-compose.yaml should use ${CODE_PORT}")
			}
			if !strings.Contains(content, "${PORT}") {
				t.Error("process-compose.yaml should use ${PORT}")
			}
			if !strings.Contains(content, "${DBA_WORKSPACE_PATH}") {
				t.Error("process-compose.yaml should use ${DBA_WORKSPACE_PATH}")
			}
		})
	}
}

// TestInitInExistingDirectory tests initializing workspace in an existing directory
func TestInitInExistingDirectory(t *testing.T) {
	tmpHome, err := os.MkdirTemp("", "dba_init_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpHome)

	oldHome := os.Getenv("DBA_HOME")
	os.Setenv("DBA_HOME", tmpHome)
	defer os.Setenv("DBA_HOME", oldHome)

	os.MkdirAll(filepath.Join(tmpHome, "workspaces"), 0755)

	cfg, err := config.Load()
	if err != nil {
		t.Fatal(err)
	}

	// Create a project directory with some files
	projectDir, err := os.MkdirTemp("", "existing_project")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(projectDir)

	// Create some existing files
	os.WriteFile(filepath.Join(projectDir, "package.json"), []byte(`{"name": "test"}`), 0644)
	os.WriteFile(filepath.Join(projectDir, "index.js"), []byte(`console.log("hello")`), 0644)
	os.MkdirAll(filepath.Join(projectDir, "src"), 0755)
	os.WriteFile(filepath.Join(projectDir, "src", "app.js"), []byte(`module.exports = {}`), 0644)

	// Initialize workspace
	ws, err := Init(cfg, projectDir, InitOptions{
		Name:     "existing-project",
		Template: "node",
	})
	if err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	// Verify workspace was created
	if ws.ID == "" {
		t.Error("Workspace ID should not be empty")
	}

	// Verify .dba directory was created
	dbaDir := filepath.Join(projectDir, ".dba")
	if _, err := os.Stat(dbaDir); os.IsNotExist(err) {
		t.Error(".dba directory should be created")
	}

	// Verify existing files are untouched
	pkgContent, _ := os.ReadFile(filepath.Join(projectDir, "package.json"))
	if string(pkgContent) != `{"name": "test"}` {
		t.Error("Existing files should not be modified")
	}

	// Verify devbox.json was created
	if _, err := os.Stat(filepath.Join(projectDir, "devbox.json")); os.IsNotExist(err) {
		t.Error("devbox.json should be created")
	}

	// Verify project path is the directory itself (not a subdirectory)
	if ws.ProjectPath != projectDir {
		t.Errorf("ProjectPath should be %s, got %s", projectDir, ws.ProjectPath)
	}

	// Clean up - just remove .dba, keep the project
	os.RemoveAll(dbaDir)
}

// TestInitWithExistingDevbox tests initializing workspace when devbox.json already exists
func TestInitWithExistingDevbox(t *testing.T) {
	tmpHome, err := os.MkdirTemp("", "dba_init_devbox_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpHome)

	oldHome := os.Getenv("DBA_HOME")
	os.Setenv("DBA_HOME", tmpHome)
	defer os.Setenv("DBA_HOME", oldHome)

	os.MkdirAll(filepath.Join(tmpHome, "workspaces"), 0755)

	cfg, err := config.Load()
	if err != nil {
		t.Fatal(err)
	}

	// Create a project directory with existing devbox.json
	projectDir, err := os.MkdirTemp("", "existing_devbox")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(projectDir)

	// Create existing devbox.json
	existingDevbox := `{
  "packages": ["nodejs@18", "yarn@latest"],
  "env": {
    "CUSTOM_VAR": "custom_value"
  }
}`
	os.WriteFile(filepath.Join(projectDir, "devbox.json"), []byte(existingDevbox), 0644)

	// Initialize workspace
	ws, err := Init(cfg, projectDir, InitOptions{
		Name:     "devbox-exists",
		Template: "node",
	})
	if err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	// Verify workspace was created
	if ws.ID == "" {
		t.Error("Workspace ID should not be empty")
	}

	// Verify existing devbox.json was NOT overwritten
	devboxContent, _ := os.ReadFile(filepath.Join(projectDir, "devbox.json"))
	if !strings.Contains(string(devboxContent), "nodejs@18") {
		t.Error("Existing devbox.json should not be overwritten")
	}
	if !strings.Contains(string(devboxContent), "CUSTOM_VAR") {
		t.Error("Existing devbox.json should preserve custom env vars")
	}

	// Clean up
	os.RemoveAll(filepath.Join(projectDir, ".dba"))
}
