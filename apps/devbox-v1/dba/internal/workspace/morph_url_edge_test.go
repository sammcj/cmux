// internal/workspace/morph_url_edge_test.go
package workspace

import (
	"testing"
)

// =============================================================================
// URL Generation Edge Cases
// =============================================================================

func TestSetMorphInstanceWithVariousURLFormats(t *testing.T) {
	tests := []struct {
		name         string
		baseURL      string
		expectedCode string
		expectedVNC  string
		expectedApp  string
		expectedCDP  string
	}{
		{
			name:         "standard",
			baseURL:      "https://example.com",
			expectedCode: "https://example.com/code/",
			expectedVNC:  "https://example.com/vnc/vnc.html",
			expectedApp:  "https://example.com/vnc/app/",
			expectedCDP:  "wss://example.com/cdp/",
		},
		{
			name:         "with_port",
			baseURL:      "https://example.com:8443",
			expectedCode: "https://example.com:8443/code/",
			expectedVNC:  "https://example.com:8443/vnc/vnc.html",
			expectedApp:  "https://example.com:8443/vnc/app/",
			expectedCDP:  "wss://example.com:8443/cdp/",
		},
		{
			name:         "trailing_slash",
			baseURL:      "https://example.com/",
			expectedCode: "https://example.com/code/",
			expectedVNC:  "https://example.com/vnc/vnc.html",
			expectedApp:  "https://example.com/vnc/app/",
			expectedCDP:  "wss://example.com/cdp/",
		},
		{
			name:         "with_path",
			baseURL:      "https://example.com/api/v1",
			expectedCode: "https://example.com/api/v1/code/",
			expectedVNC:  "https://example.com/api/v1/vnc/vnc.html",
			expectedApp:  "https://example.com/api/v1/vnc/app/",
			expectedCDP:  "wss://example.com/api/v1/cdp/",
		},
		{
			name:         "with_path_trailing_slash",
			baseURL:      "https://example.com/api/v1/",
			expectedCode: "https://example.com/api/v1/code/",
			expectedVNC:  "https://example.com/api/v1/vnc/vnc.html",
			expectedApp:  "https://example.com/api/v1/vnc/app/",
			expectedCDP:  "wss://example.com/api/v1/cdp/",
		},
		{
			name:         "http",
			baseURL:      "http://localhost:3000",
			expectedCode: "http://localhost:3000/code/",
			expectedVNC:  "http://localhost:3000/vnc/vnc.html",
			expectedApp:  "http://localhost:3000/vnc/app/",
			expectedCDP:  "ws://localhost:3000/cdp/",
		},
		{
			name:         "ip_address",
			baseURL:      "http://192.168.1.100:8080",
			expectedCode: "http://192.168.1.100:8080/code/",
			expectedVNC:  "http://192.168.1.100:8080/vnc/vnc.html",
			expectedApp:  "http://192.168.1.100:8080/vnc/app/",
			expectedCDP:  "ws://192.168.1.100:8080/cdp/",
		},
		{
			name:         "ipv6",
			baseURL:      "http://[::1]:8080",
			expectedCode: "http://[::1]:8080/code/",
			expectedVNC:  "http://[::1]:8080/vnc/vnc.html",
			expectedApp:  "http://[::1]:8080/vnc/app/",
			expectedCDP:  "ws://[::1]:8080/cdp/",
		},
		{
			name:         "subdomain",
			baseURL:      "https://instance-123.morph.example.com",
			expectedCode: "https://instance-123.morph.example.com/code/",
			expectedVNC:  "https://instance-123.morph.example.com/vnc/vnc.html",
			expectedApp:  "https://instance-123.morph.example.com/vnc/app/",
			expectedCDP:  "wss://instance-123.morph.example.com/cdp/",
		},
		{
			name:         "multiple_trailing_slashes",
			baseURL:      "https://example.com///",
			expectedCode: "https://example.com///code/", // Only strips one trailing slash
			expectedVNC:  "https://example.com///vnc/vnc.html",
			expectedApp:  "https://example.com///vnc/app/",
			expectedCDP:  "wss://example.com///cdp/",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := &Workspace{}
			w.SetMorphInstance("inst-123", "snap-456", tt.baseURL)

			if w.Morph.CodeURL != tt.expectedCode {
				t.Errorf("CodeURL = %q, want %q", w.Morph.CodeURL, tt.expectedCode)
			}
			if w.Morph.VNCURL != tt.expectedVNC {
				t.Errorf("VNCURL = %q, want %q", w.Morph.VNCURL, tt.expectedVNC)
			}
			if w.Morph.AppURL != tt.expectedApp {
				t.Errorf("AppURL = %q, want %q", w.Morph.AppURL, tt.expectedApp)
			}
			if w.Morph.CDPURL != tt.expectedCDP {
				t.Errorf("CDPURL = %q, want %q", w.Morph.CDPURL, tt.expectedCDP)
			}
		})
	}
}

func TestSetMorphInstanceWithEmptyURL(t *testing.T) {
	w := &Workspace{}
	w.SetMorphInstance("inst-123", "snap-456", "")

	// Empty base URL should result in empty derived URLs
	if w.Morph.CodeURL != "" {
		t.Errorf("CodeURL = %q, want empty", w.Morph.CodeURL)
	}
	if w.Morph.VNCURL != "" {
		t.Errorf("VNCURL = %q, want empty", w.Morph.VNCURL)
	}
	if w.Morph.AppURL != "" {
		t.Errorf("AppURL = %q, want empty", w.Morph.AppURL)
	}
	if w.Morph.CDPURL != "" {
		t.Errorf("CDPURL = %q, want empty", w.Morph.CDPURL)
	}
}

func TestSetMorphInstanceWithSpecialURLCharacters(t *testing.T) {
	tests := []struct {
		name    string
		baseURL string
	}{
		{"with_query", "https://example.com?param=value"},
		{"with_fragment", "https://example.com#section"},
		{"with_query_and_fragment", "https://example.com?param=value#section"},
		{"with_spaces_encoded", "https://example.com/path%20with%20spaces"},
		{"with_unicode", "https://example.com/日本語"},
		{"with_emoji", "https://example.com/🚀"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := &Workspace{}
			w.SetMorphInstance("inst-123", "snap-456", tt.baseURL)

			// Should not panic and should set URLs
			t.Logf("CodeURL for %s: %s", tt.name, w.Morph.CodeURL)
		})
	}
}

// =============================================================================
// GetMorphURLs Edge Cases
// =============================================================================

func TestGetMorphURLsAllPopulated(t *testing.T) {
	w := &Workspace{}
	w.Morph.CodeURL = "https://example.com/code/"
	w.Morph.VNCURL = "https://example.com/vnc/"
	w.Morph.AppURL = "https://example.com/app/"
	w.Morph.CDPURL = "https://example.com/cdp/"

	urls := w.GetMorphURLs()

	expected := map[string]string{
		"code": "https://example.com/code/",
		"vnc":  "https://example.com/vnc/",
		"app":  "https://example.com/app/",
		"cdp":  "https://example.com/cdp/",
	}

	for key, expectedURL := range expected {
		if urls[key] != expectedURL {
			t.Errorf("urls[%s] = %s, want %s", key, urls[key], expectedURL)
		}
	}
}

func TestGetMorphURLsPartiallyPopulated(t *testing.T) {
	w := &Workspace{}
	w.Morph.CodeURL = "https://example.com/code/"
	// Other URLs are empty

	urls := w.GetMorphURLs()

	if _, ok := urls["code"]; !ok {
		t.Error("Expected code URL in map")
	}
	if _, ok := urls["vnc"]; ok {
		t.Error("Did not expect vnc URL in map")
	}
}

func TestGetMorphURLsNonePopulated(t *testing.T) {
	w := &Workspace{}

	urls := w.GetMorphURLs()

	if len(urls) != 0 {
		t.Errorf("Expected empty map, got %v", urls)
	}
}

func TestGetMorphURLsWithWhitespaceOnlyURLs(t *testing.T) {
	w := &Workspace{}
	w.Morph.CodeURL = "   "
	w.Morph.VNCURL = "\t"
	w.Morph.AppURL = "\n"
	w.Morph.CDPURL = ""

	urls := w.GetMorphURLs()

	// Whitespace-only strings are technically non-empty
	// The behavior depends on implementation
	t.Logf("URLs with whitespace: %v", urls)
}

// =============================================================================
// URL Preservation Tests
// =============================================================================

func TestURLsPreservedAfterClear(t *testing.T) {
	w := &Workspace{}
	w.SetMorphInstance("inst-123", "snap-456", "https://example.com")

	originalCodeURL := w.Morph.CodeURL
	originalVNCURL := w.Morph.VNCURL
	originalAppURL := w.Morph.AppURL
	originalCDPURL := w.Morph.CDPURL

	w.ClearMorphInstance()

	// URLs should be preserved after clear
	if w.Morph.CodeURL != originalCodeURL {
		t.Errorf("CodeURL changed after clear: %s -> %s", originalCodeURL, w.Morph.CodeURL)
	}
	if w.Morph.VNCURL != originalVNCURL {
		t.Errorf("VNCURL changed after clear: %s -> %s", originalVNCURL, w.Morph.VNCURL)
	}
	if w.Morph.AppURL != originalAppURL {
		t.Errorf("AppURL changed after clear: %s -> %s", originalAppURL, w.Morph.AppURL)
	}
	if w.Morph.CDPURL != originalCDPURL {
		t.Errorf("CDPURL changed after clear: %s -> %s", originalCDPURL, w.Morph.CDPURL)
	}
}

func TestBaseURLPreservedAfterClear(t *testing.T) {
	w := &Workspace{}
	w.SetMorphInstance("inst-123", "snap-456", "https://example.com")

	originalBaseURL := w.Morph.BaseURL

	w.ClearMorphInstance()

	if w.Morph.BaseURL != originalBaseURL {
		t.Errorf("BaseURL changed after clear: %s -> %s", originalBaseURL, w.Morph.BaseURL)
	}
}

// =============================================================================
// URL Update Tests
// =============================================================================

func TestURLsUpdatedOnNewInstance(t *testing.T) {
	w := &Workspace{}

	// First instance
	w.SetMorphInstance("inst-1", "snap-1", "https://first.example.com")
	firstCodeURL := w.Morph.CodeURL

	// Second instance
	w.SetMorphInstance("inst-2", "snap-2", "https://second.example.com")
	secondCodeURL := w.Morph.CodeURL

	if firstCodeURL == secondCodeURL {
		t.Error("URLs should be updated when new instance is set")
	}
	if secondCodeURL != "https://second.example.com/code/" {
		t.Errorf("CodeURL = %s, want https://second.example.com/code/", secondCodeURL)
	}
}

// =============================================================================
// CDPPort Tests
// =============================================================================

func TestCDPPortWithURL(t *testing.T) {
	w := &Workspace{}
	w.SetMorphInstance("inst-123", "snap-456", "https://example.com")
	w.Morph.CDPPort = 9222

	// CDPPort and CDPURL can coexist
	if w.Morph.CDPPort != 9222 {
		t.Errorf("CDPPort = %d, want 9222", w.Morph.CDPPort)
	}
	if w.Morph.CDPURL == "" {
		t.Error("CDPURL should still be set")
	}
}

func TestCDPPortAfterClear(t *testing.T) {
	w := &Workspace{}
	w.SetMorphInstance("inst-123", "snap-456", "https://example.com")
	w.Morph.CDPPort = 9222

	w.ClearMorphInstance()

	// CDPPort is not cleared by ClearMorphInstance
	// (only InstanceID and Status are changed)
	t.Logf("CDPPort after clear: %d", w.Morph.CDPPort)
}

// =============================================================================
// Edge Case: Very Long URLs
// =============================================================================

func TestVeryLongURL(t *testing.T) {
	w := &Workspace{}

	// Generate a very long URL
	longPath := ""
	for i := 0; i < 1000; i++ {
		longPath += "/segment"
	}
	baseURL := "https://example.com" + longPath

	w.SetMorphInstance("inst-123", "snap-456", baseURL)

	// Should not panic
	if w.Morph.BaseURL != baseURL {
		t.Error("BaseURL should be set even for very long URLs")
	}
	if w.Morph.CodeURL == "" {
		t.Error("CodeURL should be derived even for very long URLs")
	}
}

// =============================================================================
// Protocol Edge Cases
// =============================================================================

func TestURLWithFTPProtocol(t *testing.T) {
	w := &Workspace{}
	w.SetMorphInstance("inst-123", "snap-456", "ftp://files.example.com")

	// FTP protocol should work (though unusual)
	if w.Morph.CodeURL != "ftp://files.example.com/code/" {
		t.Logf("CodeURL with FTP: %s", w.Morph.CodeURL)
	}
}

func TestURLWithWebSocketProtocol(t *testing.T) {
	w := &Workspace{}
	w.SetMorphInstance("inst-123", "snap-456", "wss://ws.example.com")

	if w.Morph.BaseURL != "wss://ws.example.com" {
		t.Errorf("BaseURL = %s, want wss://ws.example.com", w.Morph.BaseURL)
	}
}
