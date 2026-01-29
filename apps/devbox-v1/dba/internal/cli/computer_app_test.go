// internal/cli/computer_app_test.go
package cli

import (
	"strings"
	"testing"

	"github.com/dba-cli/dba/internal/workspace"
)

// TestAppCommandExists tests that app command is registered
func TestAppCommandExists(t *testing.T) {
	root := GetRootCmd()
	computerCmd := findCommand(root, "computer")
	if computerCmd == nil {
		t.Fatal("computer command not found")
	}

	appCmd := findCommand(computerCmd, "app")
	if appCmd == nil {
		t.Fatal("app command not found")
	}

	// Check basic properties
	if appCmd.Use != "app" {
		t.Errorf("app command Use = %s, want 'app'", appCmd.Use)
	}
	if appCmd.Short == "" {
		t.Error("app command should have Short description")
	}
}

// TestPortsCommandExists tests that ports command is registered
func TestPortsCommandExists(t *testing.T) {
	root := GetRootCmd()
	computerCmd := findCommand(root, "computer")
	if computerCmd == nil {
		t.Fatal("computer command not found")
	}

	portsCmd := findCommand(computerCmd, "ports")
	if portsCmd == nil {
		t.Fatal("ports command not found")
	}

	// Check basic properties
	if portsCmd.Use != "ports" {
		t.Errorf("ports command Use = %s, want 'ports'", portsCmd.Use)
	}
	if portsCmd.Short == "" {
		t.Error("ports command should have Short description")
	}
}

// TestAppCommandFlags tests app command flags
func TestAppCommandFlags(t *testing.T) {
	root := GetRootCmd()
	computerCmd := findCommand(root, "computer")
	if computerCmd == nil {
		t.Fatal("computer command not found")
	}

	appCmd := findCommand(computerCmd, "app")
	if appCmd == nil {
		t.Fatal("app command not found")
	}

	// Check for expected flags
	portFlag := appCmd.Flags().Lookup("port")
	if portFlag == nil {
		t.Error("app command should have --port flag")
	}

	noBrowserFlag := appCmd.Flags().Lookup("no-browser")
	if noBrowserFlag == nil {
		t.Error("app command should have --no-browser flag")
	}
}

// TestAppCommandUsesInheritedJSONFlag tests that app uses inherited --json flag
func TestAppCommandUsesInheritedJSONFlag(t *testing.T) {
	root := GetRootCmd()
	computerCmd := findCommand(root, "computer")
	if computerCmd == nil {
		t.Fatal("computer command not found")
	}

	appCmd := findCommand(computerCmd, "app")
	if appCmd == nil {
		t.Fatal("app command not found")
	}

	// Should NOT have local --json flag (it's inherited from root)
	localFlag := appCmd.LocalFlags().Lookup("json")
	if localFlag != nil {
		t.Error("app command should not have local --json flag (should inherit from root)")
	}

	// Should be able to access inherited --json flag
	inheritedFlag := appCmd.InheritedFlags().Lookup("json")
	if inheritedFlag == nil {
		t.Error("app command should inherit --json flag from root")
	}
}

// TestPortsCommandUsesInheritedJSONFlag tests that ports uses inherited --json flag
func TestPortsCommandUsesInheritedJSONFlag(t *testing.T) {
	root := GetRootCmd()
	computerCmd := findCommand(root, "computer")
	if computerCmd == nil {
		t.Fatal("computer command not found")
	}

	portsCmd := findCommand(computerCmd, "ports")
	if portsCmd == nil {
		t.Fatal("ports command not found")
	}

	// Should NOT have local --json flag (it's inherited from root)
	localFlag := portsCmd.LocalFlags().Lookup("json")
	if localFlag != nil {
		t.Error("ports command should not have local --json flag (should inherit from root)")
	}

	// Should be able to access inherited --json flag
	inheritedFlag := portsCmd.InheritedFlags().Lookup("json")
	if inheritedFlag == nil {
		t.Error("ports command should inherit --json flag from root")
	}
}

// TestGetPrimaryAppPortEmptyPorts tests GetPrimaryAppPort with no ports
func TestGetPrimaryAppPortEmptyPorts(t *testing.T) {
	ws := &workspace.Workspace{}
	port := ws.GetPrimaryAppPort()
	if port != nil {
		t.Error("GetPrimaryAppPort should return nil when no ports")
	}
}

// TestGetPrimaryAppPortHTTP tests GetPrimaryAppPort prefers HTTP
func TestGetPrimaryAppPortHTTP(t *testing.T) {
	ws := &workspace.Workspace{}
	ws.AddActivePort(workspace.ActivePort{Port: 5432, Service: "postgres"})
	ws.AddActivePort(workspace.ActivePort{Port: 5173, Service: "vite", IsHTTP: true})
	ws.AddActivePort(workspace.ActivePort{Port: 6379, Service: "redis"})

	port := ws.GetPrimaryAppPort()
	if port == nil {
		t.Fatal("GetPrimaryAppPort should return a port")
	}
	if port.Port != 5173 {
		t.Errorf("GetPrimaryAppPort should prefer HTTP port, got %d", port.Port)
	}
}

// TestGetPrimaryAppPortFallback tests GetPrimaryAppPort falls back to first port
func TestGetPrimaryAppPortFallback(t *testing.T) {
	ws := &workspace.Workspace{}
	ws.AddActivePort(workspace.ActivePort{Port: 5432, Service: "postgres"})
	ws.AddActivePort(workspace.ActivePort{Port: 6379, Service: "redis"})

	port := ws.GetPrimaryAppPort()
	if port == nil {
		t.Fatal("GetPrimaryAppPort should return a port")
	}
	if port.Port != 5432 {
		t.Errorf("GetPrimaryAppPort should fall back to first port, got %d", port.Port)
	}
}

// TestAppCommandHelpContainsUsage tests that app help contains expected content
func TestAppCommandHelpContainsUsage(t *testing.T) {
	root := GetRootCmd()
	computerCmd := findCommand(root, "computer")
	if computerCmd == nil {
		t.Fatal("computer command not found")
	}

	appCmd := findCommand(computerCmd, "app")
	if appCmd == nil {
		t.Fatal("app command not found")
	}

	// Check long description mentions key features
	if !strings.Contains(appCmd.Long, "interactive elements") {
		t.Error("app Long description should mention interactive elements")
	}
}

// TestPortsCommandHelpContainsUsage tests that ports help contains expected content
func TestPortsCommandHelpContainsUsage(t *testing.T) {
	root := GetRootCmd()
	computerCmd := findCommand(root, "computer")
	if computerCmd == nil {
		t.Fatal("computer command not found")
	}

	portsCmd := findCommand(computerCmd, "ports")
	if portsCmd == nil {
		t.Fatal("ports command not found")
	}

	// Check long description mentions key features
	if !strings.Contains(portsCmd.Long, "ports") {
		t.Error("ports Long description should mention ports")
	}
}

// TestActivePortSetAndGet tests setting and getting active ports
func TestActivePortSetAndGet(t *testing.T) {
	ws := &workspace.Workspace{}

	// Add port
	ws.AddActivePort(workspace.ActivePort{
		Port:      5173,
		Service:   "vite",
		Container: "myapp-web",
		IsHTTP:    true,
		URL:       "http://localhost:5173",
	})

	// Get port
	port := ws.GetActivePort(5173)
	if port == nil {
		t.Fatal("GetActivePort should return the added port")
	}
	if port.Service != "vite" {
		t.Errorf("Service = %s, want 'vite'", port.Service)
	}
	if port.Container != "myapp-web" {
		t.Errorf("Container = %s, want 'myapp-web'", port.Container)
	}
}

// TestActivePortRemove tests removing active ports
func TestActivePortRemove(t *testing.T) {
	ws := &workspace.Workspace{}
	ws.AddActivePort(workspace.ActivePort{Port: 5173})
	ws.AddActivePort(workspace.ActivePort{Port: 5432})

	ws.RemoveActivePort(5173)

	if ws.GetActivePort(5173) != nil {
		t.Error("Port 5173 should have been removed")
	}
	if ws.GetActivePort(5432) == nil {
		t.Error("Port 5432 should still exist")
	}
}

// TestActivePortClear tests clearing all active ports
func TestActivePortClear(t *testing.T) {
	ws := &workspace.Workspace{}
	ws.AddActivePort(workspace.ActivePort{Port: 5173})
	ws.AddActivePort(workspace.ActivePort{Port: 5432})

	ws.ClearActivePorts()

	if len(ws.Morph.ActivePorts) != 0 {
		t.Error("All ports should have been cleared")
	}
}

// TestActivePortSet tests replacing all active ports
func TestActivePortSet(t *testing.T) {
	ws := &workspace.Workspace{}
	ws.AddActivePort(workspace.ActivePort{Port: 1234})

	newPorts := []workspace.ActivePort{
		{Port: 5173, Service: "vite"},
		{Port: 5432, Service: "postgres"},
	}
	ws.SetActivePorts(newPorts)

	if len(ws.Morph.ActivePorts) != 2 {
		t.Errorf("Expected 2 ports, got %d", len(ws.Morph.ActivePorts))
	}
	if ws.GetActivePort(1234) != nil {
		t.Error("Old port 1234 should be replaced")
	}
	if ws.GetActivePort(5173) == nil {
		t.Error("New port 5173 should exist")
	}
}

// TestGetHTTPPorts tests getting only HTTP ports
func TestGetHTTPPorts(t *testing.T) {
	ws := &workspace.Workspace{}
	ws.AddActivePort(workspace.ActivePort{Port: 5173, IsHTTP: true})
	ws.AddActivePort(workspace.ActivePort{Port: 5432, IsHTTP: false})
	ws.AddActivePort(workspace.ActivePort{Port: 3000, IsHTTP: true})
	ws.AddActivePort(workspace.ActivePort{Port: 6379, IsHTTP: false})

	httpPorts := ws.GetHTTPPorts()
	if len(httpPorts) != 2 {
		t.Errorf("Expected 2 HTTP ports, got %d", len(httpPorts))
	}
}
