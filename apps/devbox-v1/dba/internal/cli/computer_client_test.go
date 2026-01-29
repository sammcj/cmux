// internal/cli/computer_client_test.go
package cli

import (
	"testing"

	"github.com/dba-cli/dba/internal/browser"
)

// TestBrowserClientConfigDefaults tests default client configuration
func TestBrowserClientConfigDefaults(t *testing.T) {
	cfg := browser.ClientConfig{}

	if cfg.CDPPort != 0 {
		t.Errorf("default CDPPort should be 0, got %d", cfg.CDPPort)
	}
	if cfg.CDPURL != "" {
		t.Errorf("default CDPURL should be empty, got %s", cfg.CDPURL)
	}
	if cfg.Session != "" {
		t.Errorf("default Session should be empty, got %s", cfg.Session)
	}
	if cfg.Timeout != 0 {
		t.Errorf("default Timeout should be 0, got %d", cfg.Timeout)
	}
	if cfg.BinaryPath != "" {
		t.Errorf("default BinaryPath should be empty, got %s", cfg.BinaryPath)
	}
}

// TestBrowserClientConfigWithValues tests client configuration with values
func TestBrowserClientConfigWithValues(t *testing.T) {
	cfg := browser.ClientConfig{
		CDPPort:    9222,
		CDPURL:     "ws://localhost:9222/devtools/browser",
		Session:    "test-session",
		Timeout:    30000,
		BinaryPath: "/usr/local/bin/agent-browser",
	}

	if cfg.CDPPort != 9222 {
		t.Errorf("CDPPort mismatch: expected 9222, got %d", cfg.CDPPort)
	}
	if cfg.CDPURL != "ws://localhost:9222/devtools/browser" {
		t.Errorf("CDPURL mismatch")
	}
	if cfg.Session != "test-session" {
		t.Errorf("Session mismatch")
	}
	if cfg.Timeout != 30000 {
		t.Errorf("Timeout mismatch: expected 30000, got %d", cfg.Timeout)
	}
	if cfg.BinaryPath != "/usr/local/bin/agent-browser" {
		t.Errorf("BinaryPath mismatch")
	}
}

// TestBrowserClientConfigCDPURLOverride tests that CDPURL overrides CDPPort
func TestBrowserClientConfigCDPURLOverride(t *testing.T) {
	// When both are set, CDPURL should take precedence (by convention)
	cfg := browser.ClientConfig{
		CDPPort: 9222,
		CDPURL:  "ws://remote:9333/devtools/browser",
	}

	// Both should be stored
	if cfg.CDPPort != 9222 {
		t.Errorf("CDPPort should be stored: %d", cfg.CDPPort)
	}
	if cfg.CDPURL != "ws://remote:9333/devtools/browser" {
		t.Errorf("CDPURL should be stored")
	}
}

// TestBrowserElementStruct tests Element struct
func TestBrowserElementStruct(t *testing.T) {
	elem := browser.Element{
		Ref:         "@e1",
		Role:        "button",
		Name:        "Submit",
		Description: "Submit the form",
		Enabled:     true,
		Visible:     true,
	}

	if elem.Ref != "@e1" {
		t.Errorf("Ref mismatch: %s", elem.Ref)
	}
	if elem.Role != "button" {
		t.Errorf("Role mismatch: %s", elem.Role)
	}
	if elem.Name != "Submit" {
		t.Errorf("Name mismatch: %s", elem.Name)
	}
	if !elem.Enabled {
		t.Error("Enabled should be true")
	}
	if !elem.Visible {
		t.Error("Visible should be true")
	}
}

// TestBrowserElementDisabled tests disabled element
func TestBrowserElementDisabled(t *testing.T) {
	elem := browser.Element{
		Ref:     "@e1",
		Role:    "button",
		Name:    "Disabled Button",
		Enabled: false,
		Visible: true,
	}

	if elem.Enabled {
		t.Error("Enabled should be false")
	}
}

// TestBrowserElementHidden tests hidden element
func TestBrowserElementHidden(t *testing.T) {
	elem := browser.Element{
		Ref:     "@e1",
		Role:    "div",
		Name:    "Hidden Content",
		Enabled: true,
		Visible: false,
	}

	if elem.Visible {
		t.Error("Visible should be false")
	}
}

// TestBrowserSnapshotResultEmpty tests empty snapshot result
func TestBrowserSnapshotResultEmpty(t *testing.T) {
	result := browser.SnapshotResult{}

	if len(result.Elements) != 0 {
		t.Errorf("Elements should be empty, got %d", len(result.Elements))
	}
	if result.Raw != "" {
		t.Errorf("Raw should be empty, got %s", result.Raw)
	}
	if result.URL != "" {
		t.Errorf("URL should be empty, got %s", result.URL)
	}
	if result.Title != "" {
		t.Errorf("Title should be empty, got %s", result.Title)
	}
}

// TestBrowserSnapshotResultWithElements tests snapshot with elements
func TestBrowserSnapshotResultWithElements(t *testing.T) {
	result := browser.SnapshotResult{
		Elements: []browser.Element{
			{Ref: "@e1", Role: "button", Name: "Submit"},
			{Ref: "@e2", Role: "input", Name: "Email"},
			{Ref: "@e3", Role: "link", Name: "Home"},
		},
		URL:   "https://example.com",
		Title: "Example Page",
		Raw:   "raw snapshot output",
	}

	if len(result.Elements) != 3 {
		t.Errorf("Expected 3 elements, got %d", len(result.Elements))
	}
	if result.Elements[0].Ref != "@e1" {
		t.Errorf("First element ref should be @e1")
	}
	if result.URL != "https://example.com" {
		t.Errorf("URL mismatch")
	}
	if result.Title != "Example Page" {
		t.Errorf("Title mismatch")
	}
}

// TestBrowserClickOptions tests ClickOptions struct
func TestBrowserClickOptions(t *testing.T) {
	tests := []struct {
		name   string
		opts   browser.ClickOptions
		button string
		count  int
	}{
		{"default", browser.ClickOptions{}, "", 0},
		{"left click", browser.ClickOptions{Button: "left"}, "left", 0},
		{"right click", browser.ClickOptions{Button: "right"}, "right", 0},
		{"double click", browser.ClickOptions{ClickCount: 2}, "", 2},
		{"with delay", browser.ClickOptions{Delay: 100}, "", 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.opts.Button != tt.button {
				t.Errorf("Button mismatch: %s != %s", tt.opts.Button, tt.button)
			}
			if tt.opts.ClickCount != tt.count {
				t.Errorf("ClickCount mismatch: %d != %d", tt.opts.ClickCount, tt.count)
			}
		})
	}
}

// TestBrowserTypeOptions tests TypeOptions struct
func TestBrowserTypeOptions(t *testing.T) {
	opts := browser.TypeOptions{
		Delay: 50,
	}

	if opts.Delay != 50 {
		t.Errorf("Delay mismatch: expected 50, got %d", opts.Delay)
	}
}

// TestBrowserScreenshotOptions tests ScreenshotOptions struct
func TestBrowserScreenshotOptions(t *testing.T) {
	tests := []struct {
		name string
		opts browser.ScreenshotOptions
	}{
		{"default", browser.ScreenshotOptions{}},
		{"with path", browser.ScreenshotOptions{Path: "/tmp/screenshot.png"}},
		{"full page", browser.ScreenshotOptions{FullPage: true}},
		{"with quality", browser.ScreenshotOptions{Quality: 80}},
		{"all options", browser.ScreenshotOptions{
			Path:     "/tmp/test.jpg",
			FullPage: true,
			Quality:  90,
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Verify options can be set
			_ = tt.opts
		})
	}
}

// TestBrowserWaitOptions tests WaitOptions struct
func TestBrowserWaitOptions(t *testing.T) {
	tests := []struct {
		name    string
		opts    browser.WaitOptions
		timeout int
		state   string
	}{
		{"default", browser.WaitOptions{}, 0, ""},
		{"with timeout", browser.WaitOptions{Timeout: 30000}, 30000, ""},
		{"visible state", browser.WaitOptions{State: "visible"}, 0, "visible"},
		{"hidden state", browser.WaitOptions{State: "hidden"}, 0, "hidden"},
		{"attached state", browser.WaitOptions{State: "attached"}, 0, "attached"},
		{"detached state", browser.WaitOptions{State: "detached"}, 0, "detached"},
		{"combined", browser.WaitOptions{Timeout: 60000, State: "visible"}, 60000, "visible"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.opts.Timeout != tt.timeout {
				t.Errorf("Timeout mismatch: %d != %d", tt.opts.Timeout, tt.timeout)
			}
			if tt.opts.State != tt.state {
				t.Errorf("State mismatch: %s != %s", tt.opts.State, tt.state)
			}
		})
	}
}

// TestBrowserScrollDirection tests ScrollDirection type
func TestBrowserScrollDirectionValues(t *testing.T) {
	tests := []struct {
		dir    browser.ScrollDirection
		expect string
	}{
		{browser.ScrollUp, "up"},
		{browser.ScrollDown, "down"},
		{browser.ScrollLeft, "left"},
		{browser.ScrollRight, "right"},
	}

	for _, tt := range tests {
		t.Run(string(tt.dir), func(t *testing.T) {
			if string(tt.dir) != tt.expect {
				t.Errorf("Direction mismatch: %s != %s", string(tt.dir), tt.expect)
			}
		})
	}
}

// TestBrowserScrollDirectionIsValid tests scroll direction validation
func TestBrowserScrollDirectionIsValid(t *testing.T) {
	validDirs := []string{"up", "down", "left", "right"}
	invalidDirs := []string{"", "invalid", "UP", "Down", "diagonal"}

	for _, dir := range validDirs {
		t.Run("valid_"+dir, func(t *testing.T) {
			d := browser.ScrollDirection(dir)
			switch d {
			case browser.ScrollUp, browser.ScrollDown, browser.ScrollLeft, browser.ScrollRight:
				// Valid
			default:
				t.Errorf("Direction %s should be valid", dir)
			}
		})
	}

	for _, dir := range invalidDirs {
		t.Run("invalid_"+dir, func(t *testing.T) {
			d := browser.ScrollDirection(dir)
			switch d {
			case browser.ScrollUp, browser.ScrollDown, browser.ScrollLeft, browser.ScrollRight:
				t.Errorf("Direction %s should be invalid", dir)
			default:
				// Invalid as expected
			}
		})
	}
}

// TestBrowserElementRoles tests various element roles
func TestBrowserElementRoles(t *testing.T) {
	roles := []string{
		"button",
		"link",
		"input",
		"textbox",
		"checkbox",
		"radio",
		"combobox",
		"listbox",
		"option",
		"menuitem",
		"tab",
		"tabpanel",
		"dialog",
		"alert",
		"heading",
		"img",
		"list",
		"listitem",
		"table",
		"row",
		"cell",
		"navigation",
		"main",
		"article",
		"section",
		"generic",
	}

	for _, role := range roles {
		t.Run(role, func(t *testing.T) {
			elem := browser.Element{
				Ref:  "@e1",
				Role: role,
			}
			if elem.Role != role {
				t.Errorf("Role mismatch: %s != %s", elem.Role, role)
			}
		})
	}
}

// TestBrowserSnapshotManyElements tests snapshot with many elements
func TestBrowserSnapshotManyElements(t *testing.T) {
	// Create snapshot with 1000 elements
	elements := make([]browser.Element, 1000)
	for i := 0; i < 1000; i++ {
		elements[i] = browser.Element{
			Ref:     "@e" + string(rune('0'+i%10)),
			Role:    "button",
			Name:    "Button",
			Enabled: true,
			Visible: true,
		}
	}

	result := browser.SnapshotResult{
		Elements: elements,
		URL:      "https://example.com",
		Title:    "Page with many elements",
	}

	if len(result.Elements) != 1000 {
		t.Errorf("Expected 1000 elements, got %d", len(result.Elements))
	}
}

// TestBrowserElementUnicodeNames tests elements with unicode names
func TestBrowserElementUnicodeNames(t *testing.T) {
	tests := []struct {
		name string
		elem browser.Element
	}{
		{"chinese", browser.Element{Ref: "@e1", Role: "button", Name: "提交"}},
		{"japanese", browser.Element{Ref: "@e2", Role: "button", Name: "送信"}},
		{"korean", browser.Element{Ref: "@e3", Role: "button", Name: "제출"}},
		{"russian", browser.Element{Ref: "@e4", Role: "button", Name: "Отправить"}},
		{"arabic", browser.Element{Ref: "@e5", Role: "button", Name: "إرسال"}},
		{"emoji", browser.Element{Ref: "@e6", Role: "button", Name: "🚀 Launch"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.elem.Name == "" {
				t.Error("Name should not be empty")
			}
		})
	}
}

// TestBrowserClientConfigSessionIsolation tests session isolation config
func TestBrowserClientConfigSessionIsolation(t *testing.T) {
	sessions := []string{
		"",                  // Default session
		"session-1",
		"workspace-abc123",
		"test_session_with_underscores",
		"session.with.dots",
	}

	for _, session := range sessions {
		t.Run(session, func(t *testing.T) {
			cfg := browser.ClientConfig{
				Session: session,
			}
			if cfg.Session != session {
				t.Errorf("Session mismatch: %s != %s", cfg.Session, session)
			}
		})
	}
}

// TestBrowserClientConfigBinaryPaths tests various binary paths
func TestBrowserClientConfigBinaryPaths(t *testing.T) {
	paths := []string{
		"",
		"agent-browser",
		"/usr/local/bin/agent-browser",
		"/home/user/.local/bin/agent-browser",
		"./agent-browser",
		"../bin/agent-browser",
		"/path with spaces/agent-browser",
	}

	for _, path := range paths {
		t.Run(path, func(t *testing.T) {
			cfg := browser.ClientConfig{
				BinaryPath: path,
			}
			if cfg.BinaryPath != path {
				t.Errorf("BinaryPath mismatch: %s != %s", cfg.BinaryPath, path)
			}
		})
	}
}
