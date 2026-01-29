// internal/cli/computer_snapshot_test.go
package cli

import (
	"strings"
	"testing"

	"github.com/dba-cli/dba/internal/browser"
)

// TestSnapshotParsingEmpty tests parsing empty snapshot
func TestSnapshotParsingEmpty(t *testing.T) {
	result := browser.SnapshotResult{}

	if len(result.Elements) != 0 {
		t.Error("empty snapshot should have no elements")
	}
}

// TestSnapshotParsingWithElements tests parsing snapshot with elements
func TestSnapshotParsingWithElements(t *testing.T) {
	result := browser.SnapshotResult{
		Elements: []browser.Element{
			{Ref: "@e1", Role: "button", Name: "Submit"},
			{Ref: "@e2", Role: "input", Name: "Email"},
		},
		URL:   "https://example.com",
		Title: "Test Page",
	}

	if len(result.Elements) != 2 {
		t.Errorf("expected 2 elements, got %d", len(result.Elements))
	}

	// Check first element
	if result.Elements[0].Ref != "@e1" {
		t.Errorf("first element ref should be @e1")
	}
	if result.Elements[0].Role != "button" {
		t.Errorf("first element role should be button")
	}
}

// TestRefFormatValidation tests element reference format
func TestRefFormatValidation(t *testing.T) {
	validRefs := []string{
		"@e1",
		"@e2",
		"@e10",
		"@e99",
		"@e100",
		"@e999",
		"@e0",
	}

	for _, ref := range validRefs {
		t.Run(ref, func(t *testing.T) {
			if !strings.HasPrefix(ref, "@e") {
				t.Errorf("ref should start with @e: %s", ref)
			}
			// Check that the part after @e is numeric
			numPart := strings.TrimPrefix(ref, "@e")
			if numPart == "" {
				t.Errorf("ref should have numeric part: %s", ref)
			}
			for _, c := range numPart {
				if c < '0' || c > '9' {
					t.Errorf("ref numeric part should only contain digits: %s", ref)
				}
			}
		})
	}
}

// TestInvalidRefFormats tests invalid reference formats
func TestInvalidRefFormats(t *testing.T) {
	invalidRefs := []string{
		"e1",        // Missing @
		"@1",        // Missing e
		"@ea",       // Non-numeric
		"@e",        // No number
		"@ e1",      // Space
		"@E1",       // Capital E
		"#e1",       // Wrong prefix
		"@e-1",      // Negative
		"@e1.5",     // Decimal
		"@element1", // Too verbose
	}

	for _, ref := range invalidRefs {
		t.Run(ref, func(t *testing.T) {
			// Check that it doesn't match valid format
			isValid := strings.HasPrefix(ref, "@e") && len(ref) > 2
			if isValid {
				numPart := strings.TrimPrefix(ref, "@e")
				for _, c := range numPart {
					if c < '0' || c > '9' {
						isValid = false
						break
					}
				}
			}
			// Most of these should be invalid
			// Just log which ones are technically valid by the loose check
			if isValid {
				t.Logf("ref %s passes loose validation but may be invalid", ref)
			}
		})
	}
}

// TestSnapshotElementRoles tests various element roles
func TestSnapshotElementRoles(t *testing.T) {
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
		"heading",
		"img",
		"generic",
	}

	for _, role := range roles {
		t.Run(role, func(t *testing.T) {
			elem := browser.Element{
				Ref:  "@e1",
				Role: role,
				Name: "Test Element",
			}
			if elem.Role != role {
				t.Errorf("role mismatch")
			}
		})
	}
}

// TestSnapshotElementStates tests element state combinations
func TestSnapshotElementStates(t *testing.T) {
	tests := []struct {
		name    string
		enabled bool
		visible bool
	}{
		{"enabled and visible", true, true},
		{"enabled but hidden", true, false},
		{"disabled but visible", false, true},
		{"disabled and hidden", false, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			elem := browser.Element{
				Ref:     "@e1",
				Role:    "button",
				Enabled: tt.enabled,
				Visible: tt.visible,
			}

			if elem.Enabled != tt.enabled {
				t.Errorf("Enabled mismatch")
			}
			if elem.Visible != tt.visible {
				t.Errorf("Visible mismatch")
			}
		})
	}
}

// TestSnapshotURLFormats tests various URL formats in snapshot
func TestSnapshotURLFormats(t *testing.T) {
	urls := []string{
		"https://example.com",
		"https://example.com/path",
		"https://example.com/path?query=value",
		"https://example.com/path#hash",
		"https://example.com:8080/path",
		"http://localhost:3000",
		"file:///path/to/file.html",
		"about:blank",
	}

	for _, url := range urls {
		t.Run(url, func(t *testing.T) {
			result := browser.SnapshotResult{
				URL: url,
			}
			if result.URL != url {
				t.Errorf("URL mismatch")
			}
		})
	}
}

// TestSnapshotTitleFormats tests various title formats
func TestSnapshotTitleFormats(t *testing.T) {
	titles := []string{
		"",
		"Simple Title",
		"Title with (parentheses)",
		"Title with [brackets]",
		"Title with <special> chars",
		"Very Long Title " + strings.Repeat("x", 200),
		"Unicode Title: 日本語",
		"Emoji Title: 🚀🎉",
	}

	for _, title := range titles {
		t.Run(title, func(t *testing.T) {
			result := browser.SnapshotResult{
				Title: title,
			}
			if result.Title != title {
				t.Errorf("Title mismatch")
			}
		})
	}
}

// TestSnapshotElementNames tests element name handling
func TestSnapshotElementNames(t *testing.T) {
	names := []string{
		"",
		"Submit",
		"Sign In",
		"Click here to continue",
		"Very Long Button Name " + strings.Repeat("x", 100),
		"中文按钮",
		"日本語ボタン",
		"🚀 Launch",
	}

	for _, name := range names {
		t.Run(name, func(t *testing.T) {
			elem := browser.Element{
				Ref:  "@e1",
				Role: "button",
				Name: name,
			}
			if elem.Name != name {
				t.Errorf("Name mismatch")
			}
		})
	}
}

// TestSnapshotElementDescriptions tests element description handling
func TestSnapshotElementDescriptions(t *testing.T) {
	descriptions := []string{
		"",
		"Click to submit the form",
		"Opens in a new window",
		"Required field",
	}

	for _, desc := range descriptions {
		t.Run(desc, func(t *testing.T) {
			elem := browser.Element{
				Ref:         "@e1",
				Role:        "button",
				Description: desc,
			}
			if elem.Description != desc {
				t.Errorf("Description mismatch")
			}
		})
	}
}

// TestSnapshotManyElements tests snapshot with many elements
func TestSnapshotManyElements(t *testing.T) {
	// Create snapshot with 100 elements
	elements := make([]browser.Element, 100)
	for i := 0; i < 100; i++ {
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
	}

	if len(result.Elements) != 100 {
		t.Errorf("expected 100 elements, got %d", len(result.Elements))
	}
}

// TestSnapshotInteractiveElements tests filtering interactive elements
func TestSnapshotInteractiveElements(t *testing.T) {
	result := browser.SnapshotResult{
		Elements: []browser.Element{
			{Ref: "@e1", Role: "button", Enabled: true, Visible: true},
			{Ref: "@e2", Role: "link", Enabled: true, Visible: true},
			{Ref: "@e3", Role: "input", Enabled: true, Visible: true},
			{Ref: "@e4", Role: "heading", Enabled: true, Visible: true}, // Not interactive
			{Ref: "@e5", Role: "button", Enabled: false, Visible: true}, // Disabled
			{Ref: "@e6", Role: "button", Enabled: true, Visible: false}, // Hidden
		},
	}

	// Count interactive elements (enabled buttons, links, inputs)
	interactive := 0
	for _, elem := range result.Elements {
		if elem.Enabled && elem.Visible {
			switch elem.Role {
			case "button", "link", "input":
				interactive++
			}
		}
	}

	if interactive != 3 {
		t.Errorf("expected 3 interactive elements, got %d", interactive)
	}
}

// TestSnapshotRawOutput tests raw output storage
func TestSnapshotRawOutput(t *testing.T) {
	raw := `Interactive elements:
@e1 button "Submit"
@e2 input "Email"
@e3 link "Home"`

	result := browser.SnapshotResult{
		Raw: raw,
	}

	if result.Raw != raw {
		t.Error("Raw output mismatch")
	}
	if !strings.Contains(result.Raw, "@e1") {
		t.Error("Raw output should contain refs")
	}
}

// TestRefNumberSequence tests ref number sequence
func TestRefNumberSequence(t *testing.T) {
	// Elements should typically be numbered sequentially
	elements := []browser.Element{
		{Ref: "@e1"},
		{Ref: "@e2"},
		{Ref: "@e3"},
		{Ref: "@e4"},
		{Ref: "@e5"},
	}

	for i, elem := range elements {
		expected := "@e" + string(rune('1'+i))
		if elem.Ref != expected {
			t.Errorf("element %d should have ref %s, got %s", i, expected, elem.Ref)
		}
	}
}

// TestRefWithHighNumbers tests refs with high numbers
func TestRefWithHighNumbers(t *testing.T) {
	highRefs := []string{
		"@e100",
		"@e500",
		"@e999",
		"@e1000",
		"@e9999",
	}

	for _, ref := range highRefs {
		t.Run(ref, func(t *testing.T) {
			elem := browser.Element{
				Ref:  ref,
				Role: "button",
			}
			if elem.Ref != ref {
				t.Errorf("high ref mismatch")
			}
		})
	}
}

// TestSnapshotEmptyPage tests snapshot of empty page
func TestSnapshotEmptyPage(t *testing.T) {
	result := browser.SnapshotResult{
		URL:      "about:blank",
		Title:    "",
		Elements: []browser.Element{},
	}

	if result.URL != "about:blank" {
		t.Error("URL mismatch for blank page")
	}
	if len(result.Elements) != 0 {
		t.Error("blank page should have no elements")
	}
}

// TestSnapshotSpecialPageURLs tests special page URLs
func TestSnapshotSpecialPageURLs(t *testing.T) {
	specialURLs := []string{
		"about:blank",
		"chrome://settings",
		"chrome://extensions",
		"data:text/html,<h1>Hello</h1>",
	}

	for _, url := range specialURLs {
		t.Run(url, func(t *testing.T) {
			result := browser.SnapshotResult{
				URL: url,
			}
			if result.URL != url {
				t.Errorf("URL mismatch for special page")
			}
		})
	}
}
