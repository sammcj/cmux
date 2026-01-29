// internal/cli/computer_app_edge_test.go
package cli

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/dba-cli/dba/internal/workspace"
)

// =============================================================================
// App Command Edge Cases
// =============================================================================

// TestAppCommandWithVariousPortFlags tests app command with various port flag values
func TestAppCommandWithVariousPortFlags(t *testing.T) {
	root := GetRootCmd()
	computerCmd := findCommand(root, "computer")
	if computerCmd == nil {
		t.Fatal("computer command not found")
	}

	appCmd := findCommand(computerCmd, "app")
	if appCmd == nil {
		t.Fatal("app command not found")
	}

	// Test port flag parsing
	portFlag := appCmd.Flags().Lookup("port")
	if portFlag == nil {
		t.Fatal("port flag not found")
	}

	// Test default value
	if portFlag.DefValue != "0" {
		t.Errorf("port flag default should be '0', got %s", portFlag.DefValue)
	}

	// Test shorthand
	if portFlag.Shorthand != "p" {
		t.Errorf("port flag shorthand should be 'p', got %s", portFlag.Shorthand)
	}
}

// TestAppCommandNoBrowserFlag tests the --no-browser flag
func TestAppCommandNoBrowserFlag(t *testing.T) {
	root := GetRootCmd()
	computerCmd := findCommand(root, "computer")
	appCmd := findCommand(computerCmd, "app")
	if appCmd == nil {
		t.Fatal("app command not found")
	}

	noBrowserFlag := appCmd.Flags().Lookup("no-browser")
	if noBrowserFlag == nil {
		t.Fatal("no-browser flag not found")
	}

	if noBrowserFlag.DefValue != "false" {
		t.Errorf("no-browser default should be 'false', got %s", noBrowserFlag.DefValue)
	}
}

// TestAppCommandHelpOutput tests app command help output
func TestAppCommandHelpOutput(t *testing.T) {
	root := GetRootCmd()
	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs([]string{"computer", "app", "--help"})

	err := root.Execute()
	if err != nil {
		t.Fatalf("app --help failed: %v", err)
	}

	output := buf.String()

	// Should contain key sections
	expectedContent := []string{
		"Usage:",
		"--port",
		"--no-browser",
		"interactive elements",
	}

	for _, expected := range expectedContent {
		if !strings.Contains(output, expected) {
			t.Errorf("app help should contain '%s'", expected)
		}
	}
}

// =============================================================================
// Ports Command Edge Cases
// =============================================================================

// TestPortsCommandHelpOutput tests ports command help output
func TestPortsCommandHelpOutput(t *testing.T) {
	root := GetRootCmd()
	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs([]string{"computer", "ports", "--help"})

	err := root.Execute()
	if err != nil {
		t.Fatalf("ports --help failed: %v", err)
	}

	output := buf.String()

	if !strings.Contains(output, "Usage:") {
		t.Error("ports help should contain 'Usage:'")
	}
	if !strings.Contains(output, "ports") {
		t.Error("ports help should mention 'ports'")
	}
}

// =============================================================================
// ActivePort Edge Cases
// =============================================================================

// TestActivePortAddDuplicate tests adding duplicate port updates existing
func TestActivePortAddDuplicate(t *testing.T) {
	ws := &workspace.Workspace{}

	// Add initial port
	ws.AddActivePort(workspace.ActivePort{
		Port:    5173,
		Service: "vite",
	})

	// Add duplicate with different service
	ws.AddActivePort(workspace.ActivePort{
		Port:    5173,
		Service: "webpack",
	})

	// Should only have one port
	if len(ws.Morph.ActivePorts) != 1 {
		t.Errorf("Expected 1 port, got %d", len(ws.Morph.ActivePorts))
	}

	// Service should be updated
	port := ws.GetActivePort(5173)
	if port.Service != "webpack" {
		t.Errorf("Service should be updated to 'webpack', got '%s'", port.Service)
	}
}

// TestActivePortRemoveNonExistent tests removing non-existent port
func TestActivePortRemoveNonExistent(t *testing.T) {
	ws := &workspace.Workspace{}
	ws.AddActivePort(workspace.ActivePort{Port: 5173})

	// Remove non-existent port - should not panic
	ws.RemoveActivePort(9999)

	// Original port should still exist
	if len(ws.Morph.ActivePorts) != 1 {
		t.Errorf("Expected 1 port, got %d", len(ws.Morph.ActivePorts))
	}
}

// TestActivePortZeroPort tests port number 0
func TestActivePortZeroPort(t *testing.T) {
	ws := &workspace.Workspace{}
	ws.AddActivePort(workspace.ActivePort{Port: 0, Service: "test"})

	port := ws.GetActivePort(0)
	if port == nil {
		t.Error("Should be able to add port 0")
	}
}

// TestActivePortLargePortNumber tests very large port numbers
func TestActivePortLargePortNumber(t *testing.T) {
	ws := &workspace.Workspace{}

	// Max valid port is 65535
	ws.AddActivePort(workspace.ActivePort{Port: 65535, Service: "max"})
	port := ws.GetActivePort(65535)
	if port == nil {
		t.Error("Should be able to add port 65535")
	}

	// Port beyond max (technically invalid but we don't validate)
	ws.AddActivePort(workspace.ActivePort{Port: 99999, Service: "invalid"})
	port = ws.GetActivePort(99999)
	if port == nil {
		t.Error("Should store port even if invalid number")
	}
}

// TestActivePortDiscoveredAtAutoSet tests DiscoveredAt auto-set
func TestActivePortDiscoveredAtAutoSet(t *testing.T) {
	ws := &workspace.Workspace{}

	before := time.Now()
	ws.AddActivePort(workspace.ActivePort{Port: 5173})
	after := time.Now()

	port := ws.GetActivePort(5173)
	if port.DiscoveredAt.Before(before) || port.DiscoveredAt.After(after) {
		t.Error("DiscoveredAt should be set to current time")
	}
}

// TestActivePortDiscoveredAtPreserved tests DiscoveredAt preserved if set
func TestActivePortDiscoveredAtPreserved(t *testing.T) {
	ws := &workspace.Workspace{}

	customTime := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	ws.AddActivePort(workspace.ActivePort{
		Port:         5173,
		DiscoveredAt: customTime,
	})

	port := ws.GetActivePort(5173)
	if !port.DiscoveredAt.Equal(customTime) {
		t.Error("DiscoveredAt should be preserved if already set")
	}
}

// TestActivePortAllFields tests all ActivePort fields
func TestActivePortAllFields(t *testing.T) {
	ws := &workspace.Workspace{}

	now := time.Now()
	ws.AddActivePort(workspace.ActivePort{
		Port:         5173,
		Protocol:     "tcp",
		Service:      "vite",
		Container:    "myapp-web",
		LocalPort:    15173,
		URL:          "http://localhost:15173",
		IsHTTP:       true,
		DiscoveredAt: now,
	})

	port := ws.GetActivePort(5173)
	if port == nil {
		t.Fatal("Port should exist")
	}

	if port.Port != 5173 {
		t.Errorf("Port = %d, want 5173", port.Port)
	}
	if port.Protocol != "tcp" {
		t.Errorf("Protocol = %s, want 'tcp'", port.Protocol)
	}
	if port.Service != "vite" {
		t.Errorf("Service = %s, want 'vite'", port.Service)
	}
	if port.Container != "myapp-web" {
		t.Errorf("Container = %s, want 'myapp-web'", port.Container)
	}
	if port.LocalPort != 15173 {
		t.Errorf("LocalPort = %d, want 15173", port.LocalPort)
	}
	if port.URL != "http://localhost:15173" {
		t.Errorf("URL = %s, want 'http://localhost:15173'", port.URL)
	}
	if !port.IsHTTP {
		t.Error("IsHTTP should be true")
	}
}

// TestActivePortSpecialCharacters tests special characters in service/container names
func TestActivePortSpecialCharacters(t *testing.T) {
	tests := []struct {
		name      string
		service   string
		container string
	}{
		{"with spaces", "my service", "my container"},
		{"with dashes", "my-service", "my-container"},
		{"with underscores", "my_service", "my_container"},
		{"with dots", "my.service", "my.container"},
		{"unicode", "服务", "容器"},
		{"emoji", "🚀service", "📦container"},
		{"empty", "", ""},
		{"long name", strings.Repeat("x", 1000), strings.Repeat("y", 1000)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ws := &workspace.Workspace{}
			ws.AddActivePort(workspace.ActivePort{
				Port:      5173,
				Service:   tt.service,
				Container: tt.container,
			})

			port := ws.GetActivePort(5173)
			if port.Service != tt.service {
				t.Errorf("Service mismatch")
			}
			if port.Container != tt.container {
				t.Errorf("Container mismatch")
			}
		})
	}
}

// TestActivePortURLFormats tests various URL formats
func TestActivePortURLFormats(t *testing.T) {
	urls := []string{
		"",
		"http://localhost:5173",
		"https://localhost:5173",
		"http://127.0.0.1:5173",
		"http://[::1]:5173",
		"http://example.com:5173",
		"http://localhost:5173/path",
		"http://localhost:5173/path?query=value",
		"http://localhost:5173/path#hash",
		"ws://localhost:5173",
		"wss://localhost:5173",
	}

	for _, url := range urls {
		t.Run(url, func(t *testing.T) {
			ws := &workspace.Workspace{}
			ws.AddActivePort(workspace.ActivePort{
				Port: 5173,
				URL:  url,
			})

			port := ws.GetActivePort(5173)
			if port.URL != url {
				t.Errorf("URL = %s, want %s", port.URL, url)
			}
		})
	}
}

// TestActivePortProtocols tests various protocol values
func TestActivePortProtocols(t *testing.T) {
	protocols := []string{"", "tcp", "udp", "TCP", "UDP", "sctp"}

	for _, proto := range protocols {
		t.Run(proto, func(t *testing.T) {
			ws := &workspace.Workspace{}
			ws.AddActivePort(workspace.ActivePort{
				Port:     5173,
				Protocol: proto,
			})

			port := ws.GetActivePort(5173)
			if port.Protocol != proto {
				t.Errorf("Protocol = %s, want %s", port.Protocol, proto)
			}
		})
	}
}

// =============================================================================
// GetPrimaryAppPort Edge Cases
// =============================================================================

// TestGetPrimaryAppPortMultipleHTTP tests with multiple HTTP ports
func TestGetPrimaryAppPortMultipleHTTP(t *testing.T) {
	ws := &workspace.Workspace{}
	ws.AddActivePort(workspace.ActivePort{Port: 8080, IsHTTP: true})
	ws.AddActivePort(workspace.ActivePort{Port: 3000, IsHTTP: true})
	ws.AddActivePort(workspace.ActivePort{Port: 5173, IsHTTP: true})

	// Should return first HTTP port added
	port := ws.GetPrimaryAppPort()
	if port == nil {
		t.Fatal("Should return a port")
	}
	if port.Port != 8080 {
		t.Errorf("Should return first HTTP port (8080), got %d", port.Port)
	}
}

// TestGetPrimaryAppPortMixedIsHTTP tests with mixed IsHTTP values
func TestGetPrimaryAppPortMixedIsHTTP(t *testing.T) {
	ws := &workspace.Workspace{}
	ws.AddActivePort(workspace.ActivePort{Port: 5432, IsHTTP: false}) // postgres
	ws.AddActivePort(workspace.ActivePort{Port: 6379, IsHTTP: false}) // redis
	ws.AddActivePort(workspace.ActivePort{Port: 5173, IsHTTP: true})  // vite

	port := ws.GetPrimaryAppPort()
	if port == nil {
		t.Fatal("Should return a port")
	}
	if port.Port != 5173 {
		t.Errorf("Should return HTTP port (5173), got %d", port.Port)
	}
}

// TestGetPrimaryAppPortAllNonHTTP tests when all ports are non-HTTP
func TestGetPrimaryAppPortAllNonHTTP(t *testing.T) {
	ws := &workspace.Workspace{}
	ws.AddActivePort(workspace.ActivePort{Port: 5432, IsHTTP: false})
	ws.AddActivePort(workspace.ActivePort{Port: 6379, IsHTTP: false})
	ws.AddActivePort(workspace.ActivePort{Port: 27017, IsHTTP: false})

	port := ws.GetPrimaryAppPort()
	if port == nil {
		t.Fatal("Should return first port as fallback")
	}
	if port.Port != 5432 {
		t.Errorf("Should return first port (5432), got %d", port.Port)
	}
}

// =============================================================================
// GetHTTPPorts Edge Cases
// =============================================================================

// TestGetHTTPPortsEmpty tests with no ports
func TestGetHTTPPortsEmpty(t *testing.T) {
	ws := &workspace.Workspace{}
	ports := ws.GetHTTPPorts()
	if len(ports) != 0 {
		t.Errorf("Expected 0 HTTP ports, got %d", len(ports))
	}
}

// TestGetHTTPPortsNoHTTP tests with no HTTP ports
func TestGetHTTPPortsNoHTTP(t *testing.T) {
	ws := &workspace.Workspace{}
	ws.AddActivePort(workspace.ActivePort{Port: 5432, IsHTTP: false})
	ws.AddActivePort(workspace.ActivePort{Port: 6379, IsHTTP: false})

	ports := ws.GetHTTPPorts()
	if len(ports) != 0 {
		t.Errorf("Expected 0 HTTP ports, got %d", len(ports))
	}
}

// TestGetHTTPPortsAllHTTP tests with all HTTP ports
func TestGetHTTPPortsAllHTTP(t *testing.T) {
	ws := &workspace.Workspace{}
	ws.AddActivePort(workspace.ActivePort{Port: 3000, IsHTTP: true})
	ws.AddActivePort(workspace.ActivePort{Port: 5173, IsHTTP: true})
	ws.AddActivePort(workspace.ActivePort{Port: 8080, IsHTTP: true})

	ports := ws.GetHTTPPorts()
	if len(ports) != 3 {
		t.Errorf("Expected 3 HTTP ports, got %d", len(ports))
	}
}

// =============================================================================
// SetActivePorts Edge Cases
// =============================================================================

// TestSetActivePortsEmpty tests setting empty ports list
func TestSetActivePortsEmpty(t *testing.T) {
	ws := &workspace.Workspace{}
	ws.AddActivePort(workspace.ActivePort{Port: 5173})

	ws.SetActivePorts([]workspace.ActivePort{})

	if len(ws.Morph.ActivePorts) != 0 {
		t.Errorf("Expected 0 ports after SetActivePorts([]), got %d", len(ws.Morph.ActivePorts))
	}
}

// TestSetActivePortsNil tests setting nil ports list
func TestSetActivePortsNil(t *testing.T) {
	ws := &workspace.Workspace{}
	ws.AddActivePort(workspace.ActivePort{Port: 5173})

	ws.SetActivePorts(nil)

	if ws.Morph.ActivePorts != nil && len(ws.Morph.ActivePorts) != 0 {
		t.Errorf("Expected nil/empty ports after SetActivePorts(nil)")
	}
}

// TestSetActivePortsSetsDiscoveredAt tests that SetActivePorts sets DiscoveredAt
func TestSetActivePortsSetsDiscoveredAt(t *testing.T) {
	ws := &workspace.Workspace{}

	before := time.Now()
	ws.SetActivePorts([]workspace.ActivePort{
		{Port: 5173},
		{Port: 3000},
	})
	after := time.Now()

	for _, port := range ws.Morph.ActivePorts {
		if port.DiscoveredAt.Before(before) || port.DiscoveredAt.After(after) {
			t.Error("DiscoveredAt should be set for new ports")
		}
	}
}

// TestSetActivePortsPreservesDiscoveredAt tests that SetActivePorts preserves existing DiscoveredAt
func TestSetActivePortsPreservesDiscoveredAt(t *testing.T) {
	ws := &workspace.Workspace{}

	customTime := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	ws.SetActivePorts([]workspace.ActivePort{
		{Port: 5173, DiscoveredAt: customTime},
	})

	port := ws.GetActivePort(5173)
	if !port.DiscoveredAt.Equal(customTime) {
		t.Error("DiscoveredAt should be preserved")
	}
}

// =============================================================================
// ClearActivePorts Edge Cases
// =============================================================================

// TestClearActivePortsEmpty tests clearing empty ports
func TestClearActivePortsEmpty(t *testing.T) {
	ws := &workspace.Workspace{}
	ws.ClearActivePorts() // Should not panic
	if len(ws.Morph.ActivePorts) != 0 {
		t.Error("Should have 0 ports")
	}
}

// TestClearActivePortsMultiple tests clearing multiple ports
func TestClearActivePortsMultiple(t *testing.T) {
	ws := &workspace.Workspace{}
	for i := 0; i < 100; i++ {
		ws.AddActivePort(workspace.ActivePort{Port: 5000 + i})
	}

	ws.ClearActivePorts()

	if len(ws.Morph.ActivePorts) != 0 {
		t.Errorf("Expected 0 ports after clear, got %d", len(ws.Morph.ActivePorts))
	}
}

// =============================================================================
// GetActivePort Edge Cases
// =============================================================================

// TestGetActivePortNotFound tests getting non-existent port
func TestGetActivePortNotFound(t *testing.T) {
	ws := &workspace.Workspace{}
	ws.AddActivePort(workspace.ActivePort{Port: 5173})

	port := ws.GetActivePort(9999)
	if port != nil {
		t.Error("Should return nil for non-existent port")
	}
}

// TestGetActivePortFromEmpty tests getting port from empty list
func TestGetActivePortFromEmpty(t *testing.T) {
	ws := &workspace.Workspace{}
	port := ws.GetActivePort(5173)
	if port != nil {
		t.Error("Should return nil for empty port list")
	}
}

// =============================================================================
// ManyPorts Edge Cases
// =============================================================================

// TestManyPorts tests handling many ports
func TestManyPorts(t *testing.T) {
	ws := &workspace.Workspace{}

	// Add 1000 ports
	for i := 0; i < 1000; i++ {
		ws.AddActivePort(workspace.ActivePort{
			Port:    1000 + i,
			Service: "service",
		})
	}

	if len(ws.Morph.ActivePorts) != 1000 {
		t.Errorf("Expected 1000 ports, got %d", len(ws.Morph.ActivePorts))
	}

	// Get random port
	port := ws.GetActivePort(1500)
	if port == nil {
		t.Error("Should find port 1500")
	}

	// Get HTTP ports (none)
	httpPorts := ws.GetHTTPPorts()
	if len(httpPorts) != 0 {
		t.Errorf("Expected 0 HTTP ports, got %d", len(httpPorts))
	}

	// Remove port
	ws.RemoveActivePort(1500)
	if ws.GetActivePort(1500) != nil {
		t.Error("Port 1500 should be removed")
	}
	if len(ws.Morph.ActivePorts) != 999 {
		t.Errorf("Expected 999 ports after removal, got %d", len(ws.Morph.ActivePorts))
	}
}

// =============================================================================
// Command Registration Tests
// =============================================================================

// TestAppAndPortsInComputerSubcommands tests commands are in computer subcommands
func TestAppAndPortsInComputerSubcommands(t *testing.T) {
	root := GetRootCmd()
	computerCmd := findCommand(root, "computer")
	if computerCmd == nil {
		t.Fatal("computer command not found")
	}

	subcommands := computerCmd.Commands()
	foundApp := false
	foundPorts := false

	for _, cmd := range subcommands {
		if cmd.Name() == "app" {
			foundApp = true
		}
		if cmd.Name() == "ports" {
			foundPorts = true
		}
	}

	if !foundApp {
		t.Error("app command not found in computer subcommands")
	}
	if !foundPorts {
		t.Error("ports command not found in computer subcommands")
	}
}

// TestAppAndPortsExamples tests that commands have examples
func TestAppAndPortsExamples(t *testing.T) {
	root := GetRootCmd()
	computerCmd := findCommand(root, "computer")

	appCmd := findCommand(computerCmd, "app")
	if appCmd.Example == "" {
		t.Error("app command should have examples")
	}
	if !strings.Contains(appCmd.Example, "dba computer app") {
		t.Error("app examples should show full command")
	}

	portsCmd := findCommand(computerCmd, "ports")
	if portsCmd.Example == "" {
		t.Error("ports command should have examples")
	}
	if !strings.Contains(portsCmd.Example, "dba computer ports") {
		t.Error("ports examples should show full command")
	}
}
