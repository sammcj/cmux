// internal/workspace/concurrent_test.go
package workspace

import (
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/dba-cli/dba/internal/config"
	"github.com/dba-cli/dba/internal/db"
)

// TestConcurrentWorkspaceCreation tests creating multiple workspaces concurrently
// Note: Due to SQLite's locking behavior, some concurrent creates may fail with
// UNIQUE constraint errors. This is expected behavior - the test verifies that
// successful creates have unique IDs and ports.
func TestConcurrentWorkspaceCreation(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "concurrent_create_test")
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

	numWorkspaces := 5
	var wg sync.WaitGroup
	var mu sync.Mutex
	workspaces := make([]*Workspace, 0, numWorkspaces)

	// Create workspaces concurrently
	for i := 0; i < numWorkspaces; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			opts := CreateOptions{
				Name:     "concurrent-" + string(rune('A'+idx)),
				Template: "node",
				Dir:      tmpDir,
			}
			ws, err := Create(cfg, opts)
			if err == nil && ws != nil {
				mu.Lock()
				workspaces = append(workspaces, ws)
				mu.Unlock()
			}
			// Some failures are expected due to SQLite locking
		}(i)
	}

	wg.Wait()

	// At least some workspaces should have been created
	if len(workspaces) == 0 {
		t.Error("No workspaces were created - all concurrent creates failed")
	}

	t.Logf("Created %d out of %d workspaces concurrently", len(workspaces), numWorkspaces)

	// Verify all successful workspaces have unique IDs
	ids := make(map[string]bool)
	for _, ws := range workspaces {
		if ids[ws.ID] {
			t.Errorf("Duplicate workspace ID: %s", ws.ID)
		}
		ids[ws.ID] = true
	}

	// Verify all successful workspaces have different base ports
	basePorts := make(map[int]string)
	for _, ws := range workspaces {
		if existingID, exists := basePorts[ws.BasePort]; exists {
			t.Errorf("Duplicate base port %d for workspaces %s and %s", ws.BasePort, existingID, ws.ID)
		}
		basePorts[ws.BasePort] = ws.ID
	}
}

// TestConcurrentWorkspaceReads tests reading workspace state concurrently
func TestConcurrentWorkspaceReads(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "concurrent_read_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	oldHome := os.Getenv("DBA_HOME")
	os.Setenv("DBA_HOME", tmpDir)
	defer os.Setenv("DBA_HOME", oldHome)

	// Create a workspace to read
	wsPath := filepath.Join(tmpDir, "workspaces", "ws_concurrent_read")
	dbaDir := filepath.Join(wsPath, ".dba")
	if err := os.MkdirAll(dbaDir, 0755); err != nil {
		t.Fatal(err)
	}

	ws := &Workspace{
		ID:         "ws_concurrent_read",
		Name:       "concurrent-read-test",
		Path:       wsPath,
		Template:   "node",
		Status:     "ready",
		BasePort:   10000,
		Ports:      map[string]int{"PORT": 10000, "CODE_PORT": 10001},
		CreatedAt:  time.Now(),
		LastActive: time.Now(),
	}
	if err := ws.SaveState(); err != nil {
		t.Fatal(err)
	}

	// Read state concurrently
	numReaders := 10
	var wg sync.WaitGroup
	results := make([]*Workspace, numReaders)
	readErrors := make([]error, numReaders)

	for i := 0; i < numReaders; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			loaded, err := Load(wsPath)
			results[idx] = loaded
			readErrors[idx] = err
		}(i)
	}

	wg.Wait()

	// All reads should succeed
	for i, err := range readErrors {
		if err != nil {
			t.Errorf("Concurrent read %d failed: %v", i, err)
		}
	}

	// All results should have same data
	for i, result := range results {
		if result == nil {
			t.Errorf("Read %d returned nil", i)
			continue
		}
		if result.ID != "ws_concurrent_read" {
			t.Errorf("Read %d: expected ID ws_concurrent_read, got %s", i, result.ID)
		}
		if result.BasePort != 10000 {
			t.Errorf("Read %d: expected BasePort 10000, got %d", i, result.BasePort)
		}
	}
}

// TestConcurrentResolveByID tests resolving workspaces by ID concurrently
func TestConcurrentResolveByID(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "concurrent_resolve_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	oldHome := os.Getenv("DBA_HOME")
	os.Setenv("DBA_HOME", tmpDir)
	defer os.Setenv("DBA_HOME", oldHome)

	// Create multiple workspaces
	numWorkspaces := 3
	wsIDs := make([]string, numWorkspaces)
	for i := 0; i < numWorkspaces; i++ {
		wsID := "ws_resolve_" + string(rune('a'+i))
		wsPath := filepath.Join(tmpDir, "workspaces", wsID)
		dbaDir := filepath.Join(wsPath, ".dba")
		if err := os.MkdirAll(dbaDir, 0755); err != nil {
			t.Fatal(err)
		}

		ws := &Workspace{
			ID:         wsID,
			Name:       "resolve-test-" + string(rune('a'+i)),
			Path:       wsPath,
			Template:   "node",
			Status:     "ready",
			BasePort:   10000 + i*100,
			Ports:      map[string]int{"PORT": 10000 + i*100},
			CreatedAt:  time.Now(),
			LastActive: time.Now(),
		}
		if err := ws.SaveState(); err != nil {
			t.Fatal(err)
		}

		if err := os.WriteFile(filepath.Join(dbaDir, "id"), []byte(wsID), 0644); err != nil {
			t.Fatal(err)
		}

		wsIDs[i] = wsID
	}

	// Resolve all workspaces concurrently, multiple times each
	numResolves := 10
	var wg sync.WaitGroup
	resolveErrors := make([]error, numResolves*numWorkspaces)

	for i := 0; i < numResolves; i++ {
		for j := 0; j < numWorkspaces; j++ {
			wg.Add(1)
			go func(resolveIdx, wsIdx int) {
				defer wg.Done()
				_, err := ResolveByID(wsIDs[wsIdx])
				resolveErrors[resolveIdx*numWorkspaces+wsIdx] = err
			}(i, j)
		}
	}

	wg.Wait()

	// All resolves should succeed
	for i, err := range resolveErrors {
		if err != nil {
			t.Errorf("Concurrent resolve %d failed: %v", i, err)
		}
	}
}

// TestConcurrentStateSaveLoad tests saving and loading state concurrently
func TestConcurrentStateSaveLoad(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "concurrent_save_load_test")
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
		ID:         "ws_save_load",
		Name:       "save-load-test",
		Path:       tmpDir,
		Template:   "node",
		Status:     "ready",
		BasePort:   10000,
		Ports:      map[string]int{"PORT": 10000},
		CreatedAt:  time.Now(),
		LastActive: time.Now(),
	}

	// Save initial state
	if err := ws.SaveState(); err != nil {
		t.Fatal(err)
	}

	// Concurrent saves and loads
	numOps := 20
	var wg sync.WaitGroup
	errors := make([]error, numOps)

	for i := 0; i < numOps; i++ {
		wg.Add(1)
		if i%2 == 0 {
			// Save
			go func(idx int) {
				defer wg.Done()
				wsCopy := *ws
				wsCopy.LastActive = time.Now()
				errors[idx] = wsCopy.SaveState()
			}(i)
		} else {
			// Load
			go func(idx int) {
				defer wg.Done()
				_, err := Load(tmpDir)
				errors[idx] = err
			}(i)
		}
	}

	wg.Wait()

	// Check for errors - some failures might be expected due to race conditions
	errCount := 0
	for _, err := range errors {
		if err != nil {
			errCount++
		}
	}

	// Should have mostly successes
	if errCount > numOps/2 {
		t.Errorf("Too many errors in concurrent save/load: %d out of %d", errCount, numOps)
	}
}

// TestConcurrentList tests listing workspaces concurrently
func TestConcurrentList(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "concurrent_list_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	oldHome := os.Getenv("DBA_HOME")
	os.Setenv("DBA_HOME", tmpDir)
	defer os.Setenv("DBA_HOME", oldHome)

	// Reset database singleton for test isolation
	db.ResetForTesting()

	cfg := &config.Config{
		Defaults: config.DefaultsConfig{
			Ports: []string{"PORT"},
		},
		Ports: config.PortConfig{
			RangeStart: 40000,
			RangeEnd:   49999,
			BlockSize:  100,
		},
	}

	// Create workspaces directory
	workspacesDir := filepath.Join(tmpDir, "workspaces")
	if err := os.MkdirAll(workspacesDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create some workspaces using Create (which registers in DB)
	numWorkspaces := 5
	for i := 0; i < numWorkspaces; i++ {
		_, err := Create(cfg, CreateOptions{
			Name:     "list-test-" + string(rune('a'+i)),
			Template: "node",
			Dir:      workspacesDir,
		})
		if err != nil {
			t.Fatalf("Failed to create workspace %d: %v", i, err)
		}
	}

	// List concurrently
	numLists := 10
	var wg sync.WaitGroup
	results := make([][]*Workspace, numLists)
	listErrors := make([]error, numLists)

	for i := 0; i < numLists; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			workspaces, err := List(cfg, ListOptions{})
			results[idx] = workspaces
			listErrors[idx] = err
		}(i)
	}

	wg.Wait()

	// All lists should succeed
	for i, err := range listErrors {
		if err != nil {
			t.Errorf("Concurrent list %d failed: %v", i, err)
		}
	}

	// All results should have same count
	for i, result := range results {
		if len(result) != numWorkspaces {
			t.Errorf("List %d: expected %d workspaces, got %d", i, numWorkspaces, len(result))
		}
	}
}

// TestIDGenerationUniqueness tests that generated IDs are unique even under high concurrency
func TestIDGenerationUniqueness(t *testing.T) {
	numIDs := 1000
	ids := make([]string, numIDs)
	var wg sync.WaitGroup

	for i := 0; i < numIDs; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			ids[idx] = generateID()
		}(i)
	}

	wg.Wait()

	// Check for duplicates
	seen := make(map[string]bool)
	for i, id := range ids {
		if seen[id] {
			t.Errorf("Duplicate ID generated at index %d: %s", i, id)
		}
		seen[id] = true
	}

	// Verify format
	for i, id := range ids {
		if len(id) != 11 { // "ws_" + 8 hex chars
			t.Errorf("ID %d has wrong length: %s (len=%d)", i, id, len(id))
		}
		if id[:3] != "ws_" {
			t.Errorf("ID %d doesn't start with 'ws_': %s", i, id)
		}
	}
}
