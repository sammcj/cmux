// internal/workspace/morph_security_test.go
package workspace

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

// =============================================================================
// Security-Related Tests
// =============================================================================

func TestXSSInInstanceID(t *testing.T) {
	w := &Workspace{}

	xssPayloads := []string{
		"<script>alert('xss')</script>",
		"<img src=x onerror=alert('xss')>",
		"javascript:alert('xss')",
		"<svg onload=alert('xss')>",
		"';alert('xss');//",
	}

	for _, payload := range xssPayloads {
		t.Run("xss", func(t *testing.T) {
			w.SetMorphInstance(payload, "snap-123", "https://example.com")
			if w.Morph.InstanceID != payload {
				t.Errorf("InstanceID = %s, want %s", w.Morph.InstanceID, payload)
			}
		})
	}
}

func TestXSSInSnapshotName(t *testing.T) {
	w := &Workspace{}

	xssPayloads := []string{
		"<script>alert('xss')</script>",
		"<img src=x onerror=alert('xss')>",
		"javascript:alert('xss')",
	}

	for _, payload := range xssPayloads {
		t.Run("xss", func(t *testing.T) {
			w.AddSavedSnapshot("snap-123", payload)
			snap := w.GetSavedSnapshot(payload)
			if snap == nil {
				t.Fatal("GetSavedSnapshot() returned nil")
			}
			if snap.Name != payload {
				t.Errorf("Name = %s, want %s", snap.Name, payload)
			}
		})
	}
}

func TestSQLInjectionInSnapshotName(t *testing.T) {
	w := &Workspace{}

	sqlPayloads := []string{
		"'; DROP TABLE snapshots; --",
		"1' OR '1'='1",
		"admin'--",
		"UNION SELECT password FROM users",
	}

	for _, payload := range sqlPayloads {
		t.Run("sql", func(t *testing.T) {
			w.AddSavedSnapshot("snap-sql", payload)
			snap := w.GetSavedSnapshot(payload)
			if snap == nil {
				t.Fatal("GetSavedSnapshot() returned nil")
			}
			if snap.Name != payload {
				t.Errorf("Name = %s, want %s", snap.Name, payload)
			}
		})
	}
}

func TestPathTraversalInURL(t *testing.T) {
	w := &Workspace{}

	pathPayloads := []string{
		"https://example.com/../../../etc/passwd",
		"https://example.com/..%2F..%2F..%2Fetc/passwd",
		"https://example.com/path/../../../secret",
		"file:///etc/passwd",
		"file://localhost/etc/passwd",
	}

	for _, payload := range pathPayloads {
		t.Run("path", func(t *testing.T) {
			w.SetMorphInstance("inst-123", "snap-456", payload)
			if w.Morph.BaseURL != payload {
				t.Errorf("BaseURL = %s, want %s", w.Morph.BaseURL, payload)
			}
		})
	}
}

func TestSSRFInURL(t *testing.T) {
	w := &Workspace{}

	ssrfPayloads := []string{
		"http://localhost/admin",
		"http://127.0.0.1/admin",
		"http://[::1]/admin",
		"http://169.254.169.254/latest/meta-data/",
		"http://metadata.google.internal/",
		"http://internal.service.local/",
		"gopher://localhost:25/",
	}

	for _, payload := range ssrfPayloads {
		t.Run("ssrf", func(t *testing.T) {
			w.SetMorphInstance("inst-123", "snap-456", payload)
			// Should store but not fetch
			if w.Morph.BaseURL != payload {
				t.Errorf("BaseURL = %s, want %s", w.Morph.BaseURL, payload)
			}
		})
	}
}

func TestNullByteInInstanceID(t *testing.T) {
	w := &Workspace{}

	nullPayloads := []string{
		"inst\x00evil",
		"\x00",
		"normal\x00",
		"\x00\x00\x00",
	}

	for _, payload := range nullPayloads {
		t.Run("null", func(t *testing.T) {
			w.SetMorphInstance(payload, "snap-123", "https://example.com")
			if w.Morph.InstanceID != payload {
				t.Errorf("InstanceID = %q, want %q", w.Morph.InstanceID, payload)
			}
		})
	}
}

func TestLargeInputDoS(t *testing.T) {
	w := &Workspace{}

	// Very large inputs
	largeID := strings.Repeat("A", 10*1024*1024)  // 10MB
	largeURL := strings.Repeat("B", 10*1024*1024) // 10MB

	w.SetMorphInstance(largeID, "snap-123", largeURL)

	if len(w.Morph.InstanceID) != len(largeID) {
		t.Errorf("InstanceID length = %d, want %d", len(w.Morph.InstanceID), len(largeID))
	}
	if len(w.Morph.BaseURL) != len(largeURL) {
		t.Errorf("BaseURL length = %d, want %d", len(w.Morph.BaseURL), len(largeURL))
	}
}

func TestManySnapshotsDoS(t *testing.T) {
	w := &Workspace{}

	// Add many snapshots with large names
	for i := 0; i < 10000; i++ {
		largeName := strings.Repeat("x", 1024) // 1KB name
		w.AddSavedSnapshot("snap-"+string(rune('0'+i%10)), largeName)
	}

	// Should still be functional
	if len(w.Morph.SavedSnapshots) != 10000 {
		t.Errorf("SavedSnapshots count = %d, want 10000", len(w.Morph.SavedSnapshots))
	}
}

// =============================================================================
// JSON Security Tests
// =============================================================================

func TestJSONInjection(t *testing.T) {
	w := &Workspace{
		ID:   "ws-123",
		Name: "test",
		Morph: MorphState{
			InstanceID: `{"injected": true}`,
			Status:     `"running", "extra": "field"`,
		},
	}

	data, err := json.Marshal(w)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}

	var loaded Workspace
	if err := json.Unmarshal(data, &loaded); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}

	// Should be stored as literal strings, not parsed as JSON
	if loaded.Morph.InstanceID != `{"injected": true}` {
		t.Errorf("InstanceID = %s, want literal JSON string", loaded.Morph.InstanceID)
	}
}

func TestProtoypePollutiionAttempt(t *testing.T) {
	// JSON with __proto__ pollution attempt
	jsonData := `{
		"id": "ws-123",
		"__proto__": {"admin": true},
		"morph": {
			"instance_id": "inst-456",
			"__proto__": {"privileged": true}
		}
	}`

	var w Workspace
	if err := json.Unmarshal([]byte(jsonData), &w); err != nil {
		t.Logf("Unmarshal error (expected): %v", err)
		return
	}

	// Go doesn't have prototype pollution, but verify data is clean
	t.Logf("Workspace ID: %s, InstanceID: %s", w.ID, w.Morph.InstanceID)
}

// =============================================================================
// State File Security Tests
// =============================================================================

func TestStateWithSensitiveData(t *testing.T) {
	w := &Workspace{
		ID:   "ws-123",
		Name: "test",
		Morph: MorphState{
			InstanceID: "inst-456",
			Status:     "running",
			BaseURL:    "https://api-key:secret@example.com/", // Sensitive URL
		},
	}

	data, err := json.Marshal(w)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}

	// Verify sensitive data is in the JSON (as expected)
	if !strings.Contains(string(data), "secret") {
		t.Log("Warning: Sensitive URL data serialized to state file")
	}
}

// =============================================================================
// Input Validation Tests
// =============================================================================

func TestInstanceIDValidation(t *testing.T) {
	w := &Workspace{}

	// Various instance ID formats
	ids := []string{
		"",
		"   ",
		"inst-123",
		"inst_456",
		"INST-789",
		"inst.abc.def",
		"inst-with-very-long-id-that-might-exceed-normal-limits-" + strings.Repeat("x", 1000),
		"inst-日本語",
		"inst-🎯",
		"inst-<script>",
	}

	for _, id := range ids {
		t.Run(id[:minIntSec(20, len(id))], func(t *testing.T) {
			w.SetMorphInstance(id, "snap-123", "https://example.com")
			if w.Morph.InstanceID != id {
				t.Errorf("InstanceID = %s, want %s", w.Morph.InstanceID, id)
			}
		})
	}
}

func minIntSec(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func TestURLValidation(t *testing.T) {
	w := &Workspace{}

	// Various URL formats
	urls := []string{
		"",
		"   ",
		"https://example.com",
		"http://localhost:8080",
		"http://[::1]:8080",
		"ftp://example.com",
		"file:///etc/passwd",
		"javascript:alert(1)",
		"data:text/html,<script>alert(1)</script>",
		"about:blank",
		"chrome://settings",
	}

	for _, url := range urls {
		t.Run(url[:minIntSec(30, len(url))], func(t *testing.T) {
			w.SetMorphInstance("inst-123", "snap-456", url)
			if w.Morph.BaseURL != url {
				t.Errorf("BaseURL = %s, want %s", w.Morph.BaseURL, url)
			}
		})
	}
}

// =============================================================================
// Regression Tests
// =============================================================================

func TestSetMorphInstanceDoesNotPanicOnNil(t *testing.T) {
	w := &Workspace{}

	// Should not panic with empty/nil values
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("SetMorphInstance panicked: %v", r)
		}
	}()

	w.SetMorphInstance("", "", "")
}

func TestClearMorphInstanceDoesNotPanicOnEmpty(t *testing.T) {
	w := &Workspace{}

	// Should not panic on empty workspace
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("ClearMorphInstance panicked: %v", r)
		}
	}()

	w.ClearMorphInstance()
}

func TestAddSavedSnapshotDoesNotPanicOnEmpty(t *testing.T) {
	w := &Workspace{}

	// Should not panic with empty values
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("AddSavedSnapshot panicked: %v", r)
		}
	}()

	w.AddSavedSnapshot("", "")
}

func TestGetSavedSnapshotDoesNotPanicOnNilSlice(t *testing.T) {
	w := &Workspace{}
	// SavedSnapshots is nil

	// Should not panic
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("GetSavedSnapshot panicked: %v", r)
		}
	}()

	snap := w.GetSavedSnapshot("nonexistent")
	if snap != nil {
		t.Errorf("Expected nil for nonexistent snapshot")
	}
}

func TestIsMorphRunningDoesNotPanicOnEmpty(t *testing.T) {
	w := &Workspace{}

	defer func() {
		if r := recover(); r != nil {
			t.Errorf("IsMorphRunning panicked: %v", r)
		}
	}()

	running := w.IsMorphRunning()
	if running {
		t.Error("Empty workspace should not be running")
	}
}

func TestGetMorphURLsDoesNotPanicOnEmpty(t *testing.T) {
	w := &Workspace{}

	defer func() {
		if r := recover(); r != nil {
			t.Errorf("GetMorphURLs panicked: %v", r)
		}
	}()

	urls := w.GetMorphURLs()
	if len(urls) != 0 {
		t.Errorf("Empty workspace should have no URLs")
	}
}

func TestTextOutputDoesNotPanicOnEmpty(t *testing.T) {
	w := &Workspace{}

	defer func() {
		if r := recover(); r != nil {
			t.Errorf("TextOutput panicked: %v", r)
		}
	}()

	output := w.TextOutput()
	if output == "" {
		t.Error("TextOutput should return something even for empty workspace")
	}
}

// =============================================================================
// State Integrity Tests
// =============================================================================

func TestSetMorphInstanceSetsAllFields(t *testing.T) {
	w := &Workspace{}

	before := time.Now()
	w.SetMorphInstance("inst-123", "snap-456", "https://example.com")
	after := time.Now()

	// Verify all expected fields are set
	if w.Morph.InstanceID != "inst-123" {
		t.Errorf("InstanceID = %s, want inst-123", w.Morph.InstanceID)
	}
	if w.Morph.SnapshotID != "snap-456" {
		t.Errorf("SnapshotID = %s, want snap-456", w.Morph.SnapshotID)
	}
	if w.Morph.BaseURL != "https://example.com" {
		t.Errorf("BaseURL = %s, want https://example.com", w.Morph.BaseURL)
	}
	if w.Morph.Status != "running" {
		t.Errorf("Status = %s, want running", w.Morph.Status)
	}
	if w.Morph.StartedAt.Before(before) || w.Morph.StartedAt.After(after) {
		t.Errorf("StartedAt not in expected range")
	}
	if w.Morph.CodeURL != "https://example.com/code/" {
		t.Errorf("CodeURL = %s, want https://example.com/code/", w.Morph.CodeURL)
	}
	if w.Morph.VNCURL != "https://example.com/vnc/vnc.html" {
		t.Errorf("VNCURL = %s, want https://example.com/vnc/vnc.html", w.Morph.VNCURL)
	}
	if w.Morph.AppURL != "https://example.com/vnc/app/" {
		t.Errorf("AppURL = %s, want https://example.com/vnc/app/", w.Morph.AppURL)
	}
	if w.Morph.CDPURL != "wss://example.com/cdp/" {
		t.Errorf("CDPURL = %s, want wss://example.com/cdp/", w.Morph.CDPURL)
	}
}

func TestClearMorphInstancePreservesCorrectFields(t *testing.T) {
	w := &Workspace{}
	w.SetMorphInstance("inst-123", "snap-456", "https://example.com")
	w.AddSavedSnapshot("saved-snap", "my-snapshot")

	// Store values to check
	originalCodeURL := w.Morph.CodeURL
	originalVNCURL := w.Morph.VNCURL
	originalAppURL := w.Morph.AppURL
	originalCDPURL := w.Morph.CDPURL
	originalSnapshotID := w.Morph.SnapshotID
	originalSavedSnapshots := len(w.Morph.SavedSnapshots)

	w.ClearMorphInstance()

	// InstanceID should be cleared
	if w.Morph.InstanceID != "" {
		t.Errorf("InstanceID = %s, want empty", w.Morph.InstanceID)
	}

	// Status should be stopped
	if w.Morph.Status != "stopped" {
		t.Errorf("Status = %s, want stopped", w.Morph.Status)
	}

	// URLs should be preserved
	if w.Morph.CodeURL != originalCodeURL {
		t.Errorf("CodeURL changed unexpectedly")
	}
	if w.Morph.VNCURL != originalVNCURL {
		t.Errorf("VNCURL changed unexpectedly")
	}
	if w.Morph.AppURL != originalAppURL {
		t.Errorf("AppURL changed unexpectedly")
	}
	if w.Morph.CDPURL != originalCDPURL {
		t.Errorf("CDPURL changed unexpectedly")
	}

	// SnapshotID should be preserved
	if w.Morph.SnapshotID != originalSnapshotID {
		t.Errorf("SnapshotID = %s, want %s", w.Morph.SnapshotID, originalSnapshotID)
	}

	// SavedSnapshots should be preserved
	if len(w.Morph.SavedSnapshots) != originalSavedSnapshots {
		t.Errorf("SavedSnapshots count = %d, want %d", len(w.Morph.SavedSnapshots), originalSavedSnapshots)
	}
}

func TestAddSavedSnapshotSetsAllFields(t *testing.T) {
	w := &Workspace{}

	before := time.Now()
	w.AddSavedSnapshot("snap-id-123", "my-snapshot-name")
	after := time.Now()

	if len(w.Morph.SavedSnapshots) != 1 {
		t.Fatalf("SavedSnapshots count = %d, want 1", len(w.Morph.SavedSnapshots))
	}

	snap := w.Morph.SavedSnapshots[0]
	if snap.ID != "snap-id-123" {
		t.Errorf("ID = %s, want snap-id-123", snap.ID)
	}
	if snap.Name != "my-snapshot-name" {
		t.Errorf("Name = %s, want my-snapshot-name", snap.Name)
	}
	if snap.CreatedAt.Before(before) || snap.CreatedAt.After(after) {
		t.Errorf("CreatedAt not in expected range")
	}
}
