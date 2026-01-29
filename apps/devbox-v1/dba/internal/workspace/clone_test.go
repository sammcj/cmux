// internal/workspace/clone_test.go
package workspace

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestCopyDir(t *testing.T) {
	// Create temp directories
	srcDir, err := os.MkdirTemp("", "copy_src")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(srcDir)

	dstDir, err := os.MkdirTemp("", "copy_dst")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dstDir)

	// Create source files
	if err := os.WriteFile(filepath.Join(srcDir, "file1.txt"), []byte("content1"), 0644); err != nil {
		t.Fatal(err)
	}

	// Create subdirectory with file
	subDir := filepath.Join(srcDir, "subdir")
	if err := os.MkdirAll(subDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(subDir, "file2.txt"), []byte("content2"), 0644); err != nil {
		t.Fatal(err)
	}

	// Create .dba directory (should be skipped)
	dbaDir := filepath.Join(srcDir, ".dba")
	if err := os.MkdirAll(dbaDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dbaDir, "state.json"), []byte("{}"), 0644); err != nil {
		t.Fatal(err)
	}

	// Create .git directory (should be skipped)
	gitDir := filepath.Join(srcDir, ".git")
	if err := os.MkdirAll(gitDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(gitDir, "config"), []byte("git config"), 0644); err != nil {
		t.Fatal(err)
	}

	// Copy
	if err := copyDir(srcDir, dstDir); err != nil {
		t.Fatalf("copyDir failed: %v", err)
	}

	// Verify files were copied
	if _, err := os.Stat(filepath.Join(dstDir, "file1.txt")); os.IsNotExist(err) {
		t.Error("file1.txt should be copied")
	}

	if _, err := os.Stat(filepath.Join(dstDir, "subdir", "file2.txt")); os.IsNotExist(err) {
		t.Error("subdir/file2.txt should be copied")
	}

	// Verify .dba was NOT copied
	if _, err := os.Stat(filepath.Join(dstDir, ".dba")); !os.IsNotExist(err) {
		t.Error(".dba directory should NOT be copied")
	}

	// Verify .git was NOT copied
	if _, err := os.Stat(filepath.Join(dstDir, ".git")); !os.IsNotExist(err) {
		t.Error(".git directory should NOT be copied")
	}
}

func TestCopyDirSkipsNodeModules(t *testing.T) {
	// Create temp directories
	srcDir, err := os.MkdirTemp("", "copy_nm_src")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(srcDir)

	dstDir, err := os.MkdirTemp("", "copy_nm_dst")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dstDir)

	// Create node_modules directory
	nodeModulesDir := filepath.Join(srcDir, "node_modules")
	if err := os.MkdirAll(nodeModulesDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(nodeModulesDir, "package.json"), []byte("{}"), 0644); err != nil {
		t.Fatal(err)
	}

	// Create regular file
	if err := os.WriteFile(filepath.Join(srcDir, "package.json"), []byte("{}"), 0644); err != nil {
		t.Fatal(err)
	}

	// Copy
	if err := copyDir(srcDir, dstDir); err != nil {
		t.Fatalf("copyDir failed: %v", err)
	}

	// Verify node_modules was NOT copied
	if _, err := os.Stat(filepath.Join(dstDir, "node_modules")); !os.IsNotExist(err) {
		t.Error("node_modules directory should NOT be copied")
	}

	// Verify package.json was copied
	if _, err := os.Stat(filepath.Join(dstDir, "package.json")); os.IsNotExist(err) {
		t.Error("package.json should be copied")
	}
}

func TestCloneOptions(t *testing.T) {
	opts := CloneOptions{}

	// Default name should be empty
	if opts.Name != "" {
		t.Error("default Name should be empty")
	}

	// PortNames should be nil
	if opts.PortNames != nil {
		t.Error("default PortNames should be nil")
	}

	// Test with values
	opts = CloneOptions{
		Name:      "my-clone",
		PortNames: []string{"PORT", "API_PORT"},
	}

	if opts.Name != "my-clone" {
		t.Errorf("expected Name = my-clone, got %s", opts.Name)
	}

	if len(opts.PortNames) != 2 {
		t.Errorf("expected 2 PortNames, got %d", len(opts.PortNames))
	}
}

func TestWorkspaceWithGitInfo(t *testing.T) {
	ws := &Workspace{
		ID:          "ws_git_test",
		Name:        "git-test",
		Path:        "/test/path",
		ProjectPath: "/test/path/project",
		Template:    "node",
		Status:      "ready",
		BasePort:    10000,
		Ports:       map[string]int{"PORT": 10000},
		CreatedAt:   time.Now(),
		LastActive:  time.Now(),
		Git: &GitInfo{
			Remote: "https://github.com/test/repo",
			Branch: "main",
			Commit: "abc123",
		},
	}

	if ws.Git == nil {
		t.Error("Git info should not be nil")
	}

	if ws.Git.Remote != "https://github.com/test/repo" {
		t.Errorf("expected Remote = https://github.com/test/repo, got %s", ws.Git.Remote)
	}

	if ws.Git.Branch != "main" {
		t.Errorf("expected Branch = main, got %s", ws.Git.Branch)
	}

	if ws.Git.Commit != "abc123" {
		t.Errorf("expected Commit = abc123, got %s", ws.Git.Commit)
	}
}
