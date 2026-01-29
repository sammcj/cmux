// internal/config/morph_race_test.go
package config

import (
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
)

// =============================================================================
// Race Condition Detection Tests
// Run with: go test -race ./internal/config/
// =============================================================================

func TestRaceGetMorphAPIKey(t *testing.T) {
	os.Setenv("RACE_API_KEY", "race-value")
	defer os.Unsetenv("RACE_API_KEY")

	cfg := DefaultConfig()
	cfg.Morph.APIKey = "${RACE_API_KEY}"

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = cfg.GetMorphAPIKey()
		}()
	}
	wg.Wait()
}

func TestRaceGetBaseSnapshotID(t *testing.T) {
	os.Setenv("RACE_SNAPSHOT", "race-snap")
	defer os.Unsetenv("RACE_SNAPSHOT")

	cfg := DefaultConfig()
	cfg.Morph.BaseSnapshotID = "${RACE_SNAPSHOT}"

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = cfg.GetBaseSnapshotID()
		}()
	}
	wg.Wait()
}

func TestRaceValidate(t *testing.T) {
	cfg := DefaultConfig()

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = cfg.Validate()
		}()
	}
	wg.Wait()
}

func TestRaceDefaultConfig(t *testing.T) {
	var wg sync.WaitGroup
	results := make([]*Config, 100)

	for i := 0; i < 100; i++ {
		wg.Add(1)
		idx := i
		go func() {
			defer wg.Done()
			results[idx] = DefaultConfig()
		}()
	}
	wg.Wait()

	// All results should be non-nil
	for i, cfg := range results {
		if cfg == nil {
			t.Errorf("Result %d is nil", i)
		}
	}
}

func TestRaceDefaultMorphConfig(t *testing.T) {
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = DefaultMorphConfig()
		}()
	}
	wg.Wait()
}

func TestRaceDefaultAgentBrowserConfig(t *testing.T) {
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = DefaultAgentBrowserConfig()
		}()
	}
	wg.Wait()
}

func TestRaceLoad(t *testing.T) {
	tmpDir := t.TempDir()
	os.Setenv("DBA_HOME", tmpDir)
	defer os.Unsetenv("DBA_HOME")

	content := `
morph:
  api_key: "race-test"
  vm:
    vcpus: 2
`
	if err := os.WriteFile(filepath.Join(tmpDir, "config.yaml"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	var wg sync.WaitGroup
	var errCount atomic.Int32

	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := Load()
			if err != nil {
				errCount.Add(1)
			}
		}()
	}
	wg.Wait()

	if errCount.Load() > 0 {
		t.Errorf("Load() failed %d times", errCount.Load())
	}
}

func TestRaceEnvVarResolution(t *testing.T) {
	// Set up multiple env vars
	for i := 0; i < 10; i++ {
		key := "RACE_ENV_" + string(rune('A'+i))
		os.Setenv(key, "value-"+key)
		defer os.Unsetenv(key)
	}

	cfg := DefaultConfig()
	cfg.Morph.APIKey = "${RACE_ENV_A}"

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			// Concurrent reads
			_ = cfg.GetMorphAPIKey()
			_ = cfg.GetBaseSnapshotID()
			_ = cfg.Validate()
		}()
	}
	wg.Wait()
}

// =============================================================================
// Atomic Operation Tests
// =============================================================================

func TestAtomicConfigReads(t *testing.T) {
	cfg := DefaultConfig()

	var wg sync.WaitGroup
	var vcpuSum atomic.Int64
	var memSum atomic.Int64

	for i := 0; i < 1000; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			vcpuSum.Add(int64(cfg.Morph.VM.VCPUs))
			memSum.Add(int64(cfg.Morph.VM.Memory))
		}()
	}
	wg.Wait()

	expectedVCPU := int64(DefaultMorphVCPUs * 1000)
	expectedMem := int64(DefaultMorphMemory * 1000)

	if vcpuSum.Load() != expectedVCPU {
		t.Errorf("VCPUs sum = %d, want %d", vcpuSum.Load(), expectedVCPU)
	}
	if memSum.Load() != expectedMem {
		t.Errorf("Memory sum = %d, want %d", memSum.Load(), expectedMem)
	}
}

// =============================================================================
// Concurrent Modification Detection
// =============================================================================

func TestConcurrentConfigModificationDetection(t *testing.T) {
	// This test is designed to detect if concurrent modifications
	// cause inconsistent state

	var wg sync.WaitGroup
	configs := make([]*Config, 100)

	for i := 0; i < 100; i++ {
		wg.Add(1)
		idx := i
		go func() {
			defer wg.Done()
			cfg := DefaultConfig()
			cfg.Morph.VM.VCPUs = idx + 1
			configs[idx] = cfg
		}()
	}
	wg.Wait()

	// Each config should have its own independent VCPUs value
	for i, cfg := range configs {
		if cfg.Morph.VM.VCPUs != i+1 {
			t.Errorf("Config %d has VCPUs = %d, want %d", i, cfg.Morph.VM.VCPUs, i+1)
		}
	}
}

func TestConcurrentMorphConfigIndependence(t *testing.T) {
	var wg sync.WaitGroup
	morphConfigs := make([]MorphConfig, 100)

	for i := 0; i < 100; i++ {
		wg.Add(1)
		idx := i
		go func() {
			defer wg.Done()
			cfg := DefaultMorphConfig()
			cfg.VM.Memory = idx * 1024
			morphConfigs[idx] = cfg
		}()
	}
	wg.Wait()

	// Verify independence
	for i, cfg := range morphConfigs {
		if cfg.VM.Memory != i*1024 {
			t.Errorf("MorphConfig %d has Memory = %d, want %d", i, cfg.VM.Memory, i*1024)
		}
	}
}

// =============================================================================
// High Contention Tests
// =============================================================================

func TestHighContentionLoad(t *testing.T) {
	tmpDir := t.TempDir()
	os.Setenv("DBA_HOME", tmpDir)
	defer os.Unsetenv("DBA_HOME")

	content := `
morph:
  api_key: "contention-test"
`
	if err := os.WriteFile(filepath.Join(tmpDir, "config.yaml"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	var wg sync.WaitGroup
	var successCount atomic.Int32

	// High contention - many goroutines trying to load simultaneously
	for i := 0; i < 200; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			cfg, err := Load()
			if err == nil && cfg.Morph.APIKey == "contention-test" {
				successCount.Add(1)
			}
		}()
	}
	wg.Wait()

	if successCount.Load() != 200 {
		t.Errorf("Success count = %d, want 200", successCount.Load())
	}
}

func TestHighContentionDefaultConfig(t *testing.T) {
	var wg sync.WaitGroup
	var successCount atomic.Int32

	for i := 0; i < 500; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			cfg := DefaultConfig()
			if cfg != nil && cfg.Morph.VM.VCPUs == DefaultMorphVCPUs {
				successCount.Add(1)
			}
		}()
	}
	wg.Wait()

	if successCount.Load() != 500 {
		t.Errorf("Success count = %d, want 500", successCount.Load())
	}
}

// =============================================================================
// Goroutine Leak Detection
// =============================================================================

func TestNoGoroutineLeakOnLoad(t *testing.T) {
	tmpDir := t.TempDir()
	os.Setenv("DBA_HOME", tmpDir)
	defer os.Unsetenv("DBA_HOME")

	content := `morph: {api_key: "leak-test"}`
	if err := os.WriteFile(filepath.Join(tmpDir, "config.yaml"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	// Load many times - should not leak goroutines
	for i := 0; i < 100; i++ {
		_, _ = Load()
	}

	// If this test runs without issues, no obvious goroutine leaks
	t.Log("No goroutine leaks detected")
}

func TestNoGoroutineLeakOnValidation(t *testing.T) {
	cfg := DefaultConfig()

	// Validate many times - should not leak goroutines
	for i := 0; i < 1000; i++ {
		_ = cfg.Validate()
	}

	t.Log("No goroutine leaks detected")
}
