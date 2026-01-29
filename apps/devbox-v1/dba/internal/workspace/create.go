// internal/workspace/create.go
package workspace

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/dba-cli/dba/internal/config"
	"github.com/dba-cli/dba/internal/daemon"
	"github.com/dba-cli/dba/internal/db"
	"github.com/dba-cli/dba/internal/port"
)

// CreateOptions are options for creating a workspace
type CreateOptions struct {
	Name      string
	Template  string
	Clone     string
	Branch    string
	Packages  []string
	PortNames []string
	Dir       string
}

// Create creates a new workspace
func Create(cfg *config.Config, opts CreateOptions) (*Workspace, error) {
	// Generate workspace ID
	id := generateID()

	// Determine workspace path
	if opts.Dir == "" {
		opts.Dir = filepath.Join(config.DBAHome(), "workspaces")
	}
	wsPath := filepath.Join(opts.Dir, id)

	// Set defaults
	if opts.Name == "" {
		opts.Name = "workspace"
	}
	if opts.Template == "" {
		opts.Template = "node"
	}
	if len(opts.PortNames) == 0 {
		opts.PortNames = cfg.Defaults.Ports
	}
	if opts.Branch == "" {
		opts.Branch = "main"
	}

	// Register workspace in database first (required for foreign key constraint)
	if err := registerWorkspaceInDB(id, opts.Name, wsPath, opts.Template); err != nil {
		return nil, fmt.Errorf("failed to register workspace in database: %w", err)
	}

	// Allocate ports
	allocator, err := port.NewAllocator(cfg.Ports)
	if err != nil {
		unregisterWorkspaceFromDB(id)
		return nil, fmt.Errorf("failed to create port allocator: %w", err)
	}

	ports, err := allocator.AllocateForWorkspace(id, opts.PortNames)
	if err != nil {
		unregisterWorkspaceFromDB(id)
		return nil, fmt.Errorf("failed to allocate ports: %w", err)
	}

	// Create directory structure
	if err := createDirectoryStructure(wsPath); err != nil {
		allocator.ReleaseForWorkspace(id)
		unregisterWorkspaceFromDB(id)
		return nil, fmt.Errorf("failed to create directories: %w", err)
	}

	// Clone repo if specified
	var gitInfo *GitInfo
	if opts.Clone != "" {
		gitInfo, err = cloneRepo(wsPath, opts.Clone, opts.Branch)
		if err != nil {
			os.RemoveAll(wsPath)
			allocator.ReleaseForWorkspace(id)
			unregisterWorkspaceFromDB(id)
			return nil, fmt.Errorf("failed to clone repo: %w", err)
		}

		// Use repo name as workspace name if not specified
		if opts.Name == "workspace" {
			opts.Name = extractRepoName(opts.Clone)
		}
	}

	// Create workspace object
	ws := &Workspace{
		ID:          id,
		Name:        opts.Name,
		Path:        wsPath,
		ProjectPath: filepath.Join(wsPath, "project"),
		Template:    opts.Template,
		Status:      "ready",
		BasePort:    findBasePort(ports),
		Ports:       ports,
		Packages:    opts.Packages,
		CreatedAt:   time.Now(),
		LastActive:  time.Now(),
		Git:         gitInfo,
	}

	// Generate devbox.json from template
	if err := generateDevboxConfig(ws, cfg); err != nil {
		os.RemoveAll(wsPath)
		allocator.ReleaseForWorkspace(id)
		unregisterWorkspaceFromDB(id)
		return nil, fmt.Errorf("failed to generate devbox.json: %w", err)
	}

	// Generate process-compose.yaml from template
	if err := generateProcessCompose(ws, cfg); err != nil {
		os.RemoveAll(wsPath)
		allocator.ReleaseForWorkspace(id)
		unregisterWorkspaceFromDB(id)
		return nil, fmt.Errorf("failed to generate process-compose.yaml: %w", err)
	}

	// Write workspace ID file
	if err := os.WriteFile(filepath.Join(ws.StateDir(), "id"), []byte(id), 0644); err != nil {
		os.RemoveAll(wsPath)
		allocator.ReleaseForWorkspace(id)
		unregisterWorkspaceFromDB(id)
		return nil, err
	}

	// Save workspace state
	if err := ws.SaveState(); err != nil {
		os.RemoveAll(wsPath)
		allocator.ReleaseForWorkspace(id)
		unregisterWorkspaceFromDB(id)
		return nil, err
	}

	// Update workspace in database with final base_port
	updateWorkspaceBasePort(id, ws.BasePort)

	// Auto-start daemon if needed and register workspace
	if err := daemon.EnsureRunning(cfg); err == nil {
		client := daemon.NewClient(cfg)
		client.RegisterWorkspace(id, wsPath)
	}

	return ws, nil
}

func generateID() string {
	bytes := make([]byte, 4)
	rand.Read(bytes)
	return "ws_" + hex.EncodeToString(bytes)
}

func createDirectoryStructure(wsPath string) error {
	dirs := []string{
		wsPath,
		filepath.Join(wsPath, ".dba"),
		filepath.Join(wsPath, ".dba", "logs"),
		filepath.Join(wsPath, "project"),
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
	}

	// Create .gitignore in .dba
	gitignore := filepath.Join(wsPath, ".dba", ".gitignore")
	os.WriteFile(gitignore, []byte("*\n"), 0644)

	return nil
}

func cloneRepo(wsPath, url, branch string) (*GitInfo, error) {
	projectPath := filepath.Join(wsPath, "project")

	// Remove the empty project directory
	os.RemoveAll(projectPath)

	// Clone
	args := []string{"clone", "--branch", branch, "--depth", "1", url, "project"}
	cmd := exec.Command("git", args...)
	cmd.Dir = wsPath

	if output, err := cmd.CombinedOutput(); err != nil {
		return nil, fmt.Errorf("git clone failed: %s", output)
	}

	// Get commit hash
	cmd = exec.Command("git", "rev-parse", "HEAD")
	cmd.Dir = projectPath
	output, _ := cmd.Output()

	return &GitInfo{
		Remote: url,
		Branch: branch,
		Commit: strings.TrimSpace(string(output)),
	}, nil
}

func extractRepoName(url string) string {
	// Extract name from URL like https://github.com/user/repo.git
	parts := strings.Split(url, "/")
	name := parts[len(parts)-1]
	name = strings.TrimSuffix(name, ".git")
	return name
}

func findBasePort(ports map[string]int) int {
	// Find the minimum port (should be base)
	min := 99999
	for _, p := range ports {
		if p < min {
			min = p
		}
	}
	return min
}

func generateDevboxConfig(ws *Workspace, cfg *config.Config) error {
	// Load template
	tmpl, err := LoadTemplate(ws.Template)
	if err != nil {
		return err
	}

	// Build packages list - start with defaults
	packages := make([]string, 0)
	packages = append(packages, cfg.Defaults.Packages...)

	// Add template packages
	if tmpl.Packages != nil {
		packages = append(packages, tmpl.Packages...)
	}

	// Add user-specified packages
	packages = append(packages, ws.Packages...)

	// Build devbox.json
	devbox := map[string]interface{}{
		"$schema":  "https://raw.githubusercontent.com/jetify-com/devbox/main/.schema/devbox.schema.json",
		"packages": packages,
	}

	// Build environment
	env := make(map[string]string)
	for name, portNum := range ws.Ports {
		env[name] = fmt.Sprintf("%d", portNum)
	}
	env["NODE_ENV"] = "development"
	env["DBA_WORKSPACE_ID"] = ws.ID
	env["DBA_WORKSPACE_PATH"] = ws.Path

	// Add template env
	for k, v := range tmpl.Env {
		env[k] = v
	}

	devbox["env"] = env

	// Add shell config
	devbox["shell"] = map[string]interface{}{
		"init_hook": []string{
			fmt.Sprintf("echo 'DBA Workspace: %s'", ws.Name),
			"echo '   App:     http://localhost:'$PORT",
			"echo '   VS Code: http://localhost:'$CODE_PORT",
		},
		"scripts": map[string]string{
			"dev":  "process-compose up",
			"stop": "process-compose down",
		},
	}

	// Write
	data, err := json.MarshalIndent(devbox, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(ws.DevboxPath(), data, 0644)
}

func generateProcessCompose(ws *Workspace, cfg *config.Config) error {
	tmpl, err := LoadTemplate(ws.Template)
	if err != nil {
		return err
	}

	// Write process-compose.yaml from template
	return os.WriteFile(ws.ProcessComposePath(), []byte(tmpl.ProcessCompose), 0644)
}

// InitOptions are options for initializing workspace in current directory
type InitOptions struct {
	Name      string
	Template  string
	Packages  []string
	PortNames []string
}

// Init initializes a workspace in the current directory
func Init(cfg *config.Config, cwd string, opts InitOptions) (*Workspace, error) {
	// Check if already a workspace
	if StateExists(cwd) {
		return nil, fmt.Errorf("directory is already a DBA workspace")
	}

	// Generate workspace ID
	id := generateID()

	// Set defaults
	if opts.Name == "" {
		opts.Name = filepath.Base(cwd)
	}
	if opts.Template == "" {
		opts.Template = "node"
	}
	if len(opts.PortNames) == 0 {
		opts.PortNames = cfg.Defaults.Ports
	}

	// Register workspace in database first (required for foreign key constraint)
	if err := registerWorkspaceInDB(id, opts.Name, cwd, opts.Template); err != nil {
		return nil, fmt.Errorf("failed to register workspace in database: %w", err)
	}

	// Allocate ports
	allocator, err := port.NewAllocator(cfg.Ports)
	if err != nil {
		unregisterWorkspaceFromDB(id)
		return nil, fmt.Errorf("failed to create port allocator: %w", err)
	}

	ports, err := allocator.AllocateForWorkspace(id, opts.PortNames)
	if err != nil {
		unregisterWorkspaceFromDB(id)
		return nil, fmt.Errorf("failed to allocate ports: %w", err)
	}

	// Create .dba directory structure
	dbaDir := filepath.Join(cwd, ".dba")
	if err := os.MkdirAll(dbaDir, 0755); err != nil {
		allocator.ReleaseForWorkspace(id)
		unregisterWorkspaceFromDB(id)
		return nil, fmt.Errorf("failed to create .dba directory: %w", err)
	}

	if err := os.MkdirAll(filepath.Join(dbaDir, "logs"), 0755); err != nil {
		allocator.ReleaseForWorkspace(id)
		unregisterWorkspaceFromDB(id)
		return nil, fmt.Errorf("failed to create logs directory: %w", err)
	}

	// Create .gitignore
	gitignore := filepath.Join(dbaDir, ".gitignore")
	os.WriteFile(gitignore, []byte("*\n"), 0644)

	// Detect git info if in git repo
	var gitInfo *GitInfo
	gitInfo = detectGitInfo(cwd)

	// Create workspace object
	ws := &Workspace{
		ID:          id,
		Name:        opts.Name,
		Path:        cwd,
		ProjectPath: cwd, // For init, project is the current directory
		Template:    opts.Template,
		Status:      "ready",
		BasePort:    findBasePort(ports),
		Ports:       ports,
		Packages:    opts.Packages,
		CreatedAt:   time.Now(),
		LastActive:  time.Now(),
		Git:         gitInfo,
	}

	// Generate devbox.json (unless it exists)
	devboxPath := filepath.Join(cwd, "devbox.json")
	if _, err := os.Stat(devboxPath); os.IsNotExist(err) {
		if err := generateDevboxConfig(ws, cfg); err != nil {
			os.RemoveAll(dbaDir)
			allocator.ReleaseForWorkspace(id)
			unregisterWorkspaceFromDB(id)
			return nil, fmt.Errorf("failed to generate devbox.json: %w", err)
		}
	}

	// Generate process-compose.yaml (unless it exists)
	pcPath := filepath.Join(cwd, "process-compose.yaml")
	if _, err := os.Stat(pcPath); os.IsNotExist(err) {
		if err := generateProcessCompose(ws, cfg); err != nil {
			os.RemoveAll(dbaDir)
			allocator.ReleaseForWorkspace(id)
			unregisterWorkspaceFromDB(id)
			return nil, fmt.Errorf("failed to generate process-compose.yaml: %w", err)
		}
	}

	// Write workspace ID file
	if err := os.WriteFile(filepath.Join(dbaDir, "id"), []byte(id), 0644); err != nil {
		os.RemoveAll(dbaDir)
		allocator.ReleaseForWorkspace(id)
		unregisterWorkspaceFromDB(id)
		return nil, err
	}

	// Save workspace state
	if err := ws.SaveState(); err != nil {
		os.RemoveAll(dbaDir)
		allocator.ReleaseForWorkspace(id)
		unregisterWorkspaceFromDB(id)
		return nil, err
	}

	// Update workspace in database with final base_port
	updateWorkspaceBasePort(id, ws.BasePort)

	// Auto-start daemon if needed and register workspace
	if err := daemon.EnsureRunning(cfg); err == nil {
		client := daemon.NewClient(cfg)
		client.RegisterWorkspace(id, cwd)
	}

	return ws, nil
}

func detectGitInfo(path string) *GitInfo {
	// Check if in git repo
	cmd := exec.Command("git", "rev-parse", "--git-dir")
	cmd.Dir = path
	if err := cmd.Run(); err != nil {
		return nil
	}

	info := &GitInfo{}

	// Get remote
	cmd = exec.Command("git", "remote", "get-url", "origin")
	cmd.Dir = path
	if output, err := cmd.Output(); err == nil {
		info.Remote = strings.TrimSpace(string(output))
	}

	// Get branch
	cmd = exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
	cmd.Dir = path
	if output, err := cmd.Output(); err == nil {
		info.Branch = strings.TrimSpace(string(output))
	}

	// Get commit
	cmd = exec.Command("git", "rev-parse", "HEAD")
	cmd.Dir = path
	if output, err := cmd.Output(); err == nil {
		info.Commit = strings.TrimSpace(string(output))
	}

	return info
}

// registerWorkspaceInDB inserts a workspace record into the database
// This must be called before port allocation due to foreign key constraints
func registerWorkspaceInDB(id, name, path, template string) error {
	database, err := db.Get()
	if err != nil {
		return err
	}

	_, err = database.Exec(`
		INSERT INTO workspaces (id, name, path, template, base_port, status)
		VALUES (?, ?, ?, ?, 0, 'creating')
	`, id, name, path, template)
	return err
}

// unregisterWorkspaceFromDB removes a workspace record from the database
func unregisterWorkspaceFromDB(id string) error {
	database, err := db.Get()
	if err != nil {
		return err
	}

	_, err = database.Exec(`DELETE FROM workspaces WHERE id = ?`, id)
	return err
}

// updateWorkspaceBasePort updates the base_port and status of a workspace
func updateWorkspaceBasePort(id string, basePort int) error {
	database, err := db.Get()
	if err != nil {
		return err
	}

	_, err = database.Exec(`
		UPDATE workspaces SET base_port = ?, status = 'ready' WHERE id = ?
	`, basePort, id)
	return err
}

// CloneOptions are options for cloning a workspace
type CloneOptions struct {
	Name      string
	PortNames []string
}

// Clone creates a copy of an existing workspace with new port allocations
func Clone(cfg *config.Config, sourceWs *Workspace, opts CloneOptions) (*Workspace, error) {
	// Generate new workspace ID
	id := generateID()

	// Determine new workspace path
	wsDir := filepath.Join(config.DBAHome(), "workspaces")
	wsPath := filepath.Join(wsDir, id)

	// Set defaults
	if opts.Name == "" {
		opts.Name = sourceWs.Name + "-clone"
	}
	if len(opts.PortNames) == 0 {
		// Use the same port names as source workspace
		opts.PortNames = make([]string, 0, len(sourceWs.Ports))
		for name := range sourceWs.Ports {
			opts.PortNames = append(opts.PortNames, name)
		}
	}

	// Register workspace in database first
	if err := registerWorkspaceInDB(id, opts.Name, wsPath, sourceWs.Template); err != nil {
		return nil, fmt.Errorf("failed to register workspace in database: %w", err)
	}

	// Allocate NEW ports (don't copy source ports)
	allocator, err := port.NewAllocator(cfg.Ports)
	if err != nil {
		unregisterWorkspaceFromDB(id)
		return nil, fmt.Errorf("failed to create port allocator: %w", err)
	}

	ports, err := allocator.AllocateForWorkspace(id, opts.PortNames)
	if err != nil {
		unregisterWorkspaceFromDB(id)
		return nil, fmt.Errorf("failed to allocate ports: %w", err)
	}

	// Create directory structure
	if err := createDirectoryStructure(wsPath); err != nil {
		allocator.ReleaseForWorkspace(id)
		unregisterWorkspaceFromDB(id)
		return nil, fmt.Errorf("failed to create directories: %w", err)
	}

	// Copy project files from source (if they exist)
	sourceProjectPath := sourceWs.ProjectPath
	destProjectPath := filepath.Join(wsPath, "project")
	if _, err := os.Stat(sourceProjectPath); err == nil {
		if err := copyDir(sourceProjectPath, destProjectPath); err != nil {
			os.RemoveAll(wsPath)
			allocator.ReleaseForWorkspace(id)
			unregisterWorkspaceFromDB(id)
			return nil, fmt.Errorf("failed to copy project files: %w", err)
		}
	}

	// Create new workspace object
	ws := &Workspace{
		ID:          id,
		Name:        opts.Name,
		Path:        wsPath,
		ProjectPath: destProjectPath,
		Template:    sourceWs.Template,
		Status:      "ready",
		BasePort:    findBasePort(ports),
		Ports:       ports,
		Packages:    sourceWs.Packages,
		CreatedAt:   time.Now(),
		LastActive:  time.Now(),
		Git:         nil, // Don't copy git info for clones
	}

	// Generate NEW devbox.json with new ports (don't copy source)
	if err := generateDevboxConfig(ws, cfg); err != nil {
		os.RemoveAll(wsPath)
		allocator.ReleaseForWorkspace(id)
		unregisterWorkspaceFromDB(id)
		return nil, fmt.Errorf("failed to generate devbox.json: %w", err)
	}

	// Generate NEW process-compose.yaml from template (don't copy source)
	if err := generateProcessCompose(ws, cfg); err != nil {
		os.RemoveAll(wsPath)
		allocator.ReleaseForWorkspace(id)
		unregisterWorkspaceFromDB(id)
		return nil, fmt.Errorf("failed to generate process-compose.yaml: %w", err)
	}

	// Write workspace ID file
	if err := os.WriteFile(filepath.Join(ws.StateDir(), "id"), []byte(id), 0644); err != nil {
		os.RemoveAll(wsPath)
		allocator.ReleaseForWorkspace(id)
		unregisterWorkspaceFromDB(id)
		return nil, err
	}

	// Save workspace state
	if err := ws.SaveState(); err != nil {
		os.RemoveAll(wsPath)
		allocator.ReleaseForWorkspace(id)
		unregisterWorkspaceFromDB(id)
		return nil, err
	}

	// Update workspace in database with final base_port
	updateWorkspaceBasePort(id, ws.BasePort)

	// Auto-start daemon if needed and register workspace
	if err := daemon.EnsureRunning(cfg); err == nil {
		client := daemon.NewClient(cfg)
		client.RegisterWorkspace(id, wsPath)
	}

	return ws, nil
}

// copyDir copies a directory recursively
func copyDir(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Calculate relative path
		relPath, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		dstPath := filepath.Join(dst, relPath)

		// Skip .dba directory
		if info.IsDir() && info.Name() == ".dba" {
			return filepath.SkipDir
		}

		// Skip .git directory
		if info.IsDir() && info.Name() == ".git" {
			return filepath.SkipDir
		}

		// Skip node_modules
		if info.IsDir() && info.Name() == "node_modules" {
			return filepath.SkipDir
		}

		if info.IsDir() {
			return os.MkdirAll(dstPath, info.Mode())
		}

		// Copy file
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(dstPath, data, info.Mode())
	})
}
