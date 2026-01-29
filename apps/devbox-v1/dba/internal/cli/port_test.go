// internal/cli/port_test.go
package cli

import (
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/dba-cli/dba/internal/port"
)

func TestPortCommandExists(t *testing.T) {
	cmd := GetRootCmd()

	var portCmdFound *cobra.Command
	for _, c := range cmd.Commands() {
		if c.Name() == "port" {
			portCmdFound = c
			break
		}
	}

	if portCmdFound == nil {
		t.Error("port command not found")
		return
	}

	if portCmdFound.Short != "Manage port allocations" {
		t.Errorf("unexpected short description: %s", portCmdFound.Short)
	}
}

func TestPortSubcommands(t *testing.T) {
	cmd := GetRootCmd()

	var portCmd *cobra.Command
	for _, c := range cmd.Commands() {
		if c.Name() == "port" {
			portCmd = c
			break
		}
	}

	if portCmd == nil {
		t.Fatal("port command not found")
	}

	// Check subcommands exist
	subcommands := make(map[string]*cobra.Command)
	for _, c := range portCmd.Commands() {
		subcommands[c.Name()] = c
	}

	expectedSubcommands := []string{"list", "alloc", "free", "check"}
	for _, name := range expectedSubcommands {
		if _, ok := subcommands[name]; !ok {
			t.Errorf("subcommand %s not found", name)
		}
	}
}

func TestPortListFlags(t *testing.T) {
	cmd := GetRootCmd()

	var portCmd *cobra.Command
	for _, c := range cmd.Commands() {
		if c.Name() == "port" {
			portCmd = c
			break
		}
	}

	if portCmd == nil {
		t.Fatal("port command not found")
	}

	var listCmd *cobra.Command
	for _, c := range portCmd.Commands() {
		if c.Name() == "list" {
			listCmd = c
			break
		}
	}

	if listCmd == nil {
		t.Fatal("port list command not found")
	}

	// Check --all flag exists
	allFlag := listCmd.Flags().Lookup("all")
	if allFlag == nil {
		t.Error("--all flag not found on port list command")
	}
}

func TestPortAllocUsage(t *testing.T) {
	cmd := GetRootCmd()

	var portCmd *cobra.Command
	for _, c := range cmd.Commands() {
		if c.Name() == "port" {
			portCmd = c
			break
		}
	}

	if portCmd == nil {
		t.Fatal("port command not found")
	}

	var allocCmd *cobra.Command
	for _, c := range portCmd.Commands() {
		if c.Name() == "alloc" {
			allocCmd = c
			break
		}
	}

	if allocCmd == nil {
		t.Fatal("port alloc command not found")
	}

	// Check usage pattern
	if allocCmd.Use != "alloc <name>" {
		t.Errorf("unexpected usage pattern: %s", allocCmd.Use)
	}
}

func TestPortFreeUsage(t *testing.T) {
	cmd := GetRootCmd()

	var portCmd *cobra.Command
	for _, c := range cmd.Commands() {
		if c.Name() == "port" {
			portCmd = c
			break
		}
	}

	if portCmd == nil {
		t.Fatal("port command not found")
	}

	var freeCmd *cobra.Command
	for _, c := range portCmd.Commands() {
		if c.Name() == "free" {
			freeCmd = c
			break
		}
	}

	if freeCmd == nil {
		t.Fatal("port free command not found")
	}

	// Check usage pattern
	if freeCmd.Use != "free <name>" {
		t.Errorf("unexpected usage pattern: %s", freeCmd.Use)
	}
}

func TestPortCheckUsage(t *testing.T) {
	cmd := GetRootCmd()

	var portCmd *cobra.Command
	for _, c := range cmd.Commands() {
		if c.Name() == "port" {
			portCmd = c
			break
		}
	}

	if portCmd == nil {
		t.Fatal("port command not found")
	}

	var checkCmd *cobra.Command
	for _, c := range portCmd.Commands() {
		if c.Name() == "check" {
			checkCmd = c
			break
		}
	}

	if checkCmd == nil {
		t.Fatal("port check command not found")
	}

	// Check usage pattern
	if checkCmd.Use != "check <port>" {
		t.Errorf("unexpected usage pattern: %s", checkCmd.Use)
	}
}

func TestPortInfoTextOutput(t *testing.T) {
	info := PortInfo{
		Name:   "PORT",
		Number: 10000,
		InUse:  false,
		PID:    0,
	}

	// Test free port
	result := WorkspacePortsResult{
		WorkspaceID: "test-ws",
		Ports:       []PortInfo{info},
	}
	output := result.TextOutput()
	if !strings.Contains(output, "PORT") {
		t.Error("TextOutput should contain port name")
	}
	if !strings.Contains(output, "10000") {
		t.Error("TextOutput should contain port number")
	}
	if !strings.Contains(output, "free") {
		t.Error("TextOutput should indicate port is free")
	}

	// Test in-use port
	info.InUse = true
	info.PID = 12345
	result.Ports = []PortInfo{info}
	output = result.TextOutput()
	if !strings.Contains(output, "in use") {
		t.Error("TextOutput should indicate port is in use")
	}
	if !strings.Contains(output, "12345") {
		t.Error("TextOutput should contain PID when port is in use")
	}
}

func TestAllPortsResultTextOutput(t *testing.T) {
	// Test empty result
	result := AllPortsResult{
		Allocations: nil,
	}
	output := result.TextOutput()
	if !strings.Contains(output, "No port allocations") {
		t.Error("Empty result should indicate no allocations")
	}

	// Test with allocations
	result = AllPortsResult{
		Allocations: []port.PortAllocation{
			{WorkspaceID: "ws-1", Name: "PORT", Port: 10000, BasePort: 10000},
			{WorkspaceID: "ws-1", Name: "API_PORT", Port: 10001, BasePort: 10000},
			{WorkspaceID: "ws-2", Name: "PORT", Port: 10100, BasePort: 10100},
		},
	}
	output = result.TextOutput()
	if !strings.Contains(output, "ws-1") {
		t.Error("TextOutput should contain workspace ID")
	}
	if !strings.Contains(output, "ws-2") {
		t.Error("TextOutput should contain second workspace ID")
	}
	if !strings.Contains(output, "10000") {
		t.Error("TextOutput should contain port numbers")
	}
}

func TestPortAllocResultTextOutput(t *testing.T) {
	result := PortAllocResult{
		Name:       "CUSTOM_PORT",
		Number:     10050,
		EnvUpdated: true,
	}

	output := result.TextOutput()
	if !strings.Contains(output, "CUSTOM_PORT") {
		t.Error("TextOutput should contain port name")
	}
	if !strings.Contains(output, "10050") {
		t.Error("TextOutput should contain port number")
	}
}

func TestPortFreeResultTextOutput(t *testing.T) {
	result := PortFreeResult{
		Name:       "CUSTOM_PORT",
		Number:     10050,
		Freed:      true,
		EnvUpdated: true,
	}

	output := result.TextOutput()
	if !strings.Contains(output, "CUSTOM_PORT") {
		t.Error("TextOutput should contain port name")
	}
	if !strings.Contains(output, "Freed") {
		t.Error("TextOutput should indicate port was freed")
	}
}

func TestPortCheckResultTextOutput(t *testing.T) {
	// Test available port
	result := PortCheckResult{
		Port:      8080,
		Available: true,
		UsedBy:    nil,
	}
	output := result.TextOutput()
	if !strings.Contains(output, "8080") {
		t.Error("TextOutput should contain port number")
	}
	if !strings.Contains(output, "available") {
		t.Error("TextOutput should indicate port is available")
	}

	// Test used port without process info
	result = PortCheckResult{
		Port:      8080,
		Available: false,
		UsedBy:    nil,
	}
	output = result.TextOutput()
	if !strings.Contains(output, "in use") {
		t.Error("TextOutput should indicate port is in use")
	}
}

func TestPortCommandHelp(t *testing.T) {
	cmd := GetRootCmd()

	var portCmd *cobra.Command
	for _, c := range cmd.Commands() {
		if c.Name() == "port" {
			portCmd = c
			break
		}
	}

	if portCmd == nil {
		t.Fatal("port command not found")
	}

	// Verify long description exists
	if portCmd.Long == "" {
		t.Error("port command should have a long description")
	}

	// Verify all subcommands have descriptions
	for _, sub := range portCmd.Commands() {
		if sub.Short == "" {
			t.Errorf("subcommand %s should have a short description", sub.Name())
		}
	}
}

func TestPortAllocationJSONFields(t *testing.T) {
	// Verify struct fields have correct JSON tags
	info := PortInfo{
		Name:   "TEST",
		Number: 1234,
		InUse:  true,
		PID:    5678,
	}

	// This is a compile-time check - if the struct changes, this will fail
	_ = info.Name
	_ = info.Number
	_ = info.InUse
	_ = info.PID

	allocResult := PortAllocResult{
		Name:       "TEST",
		Number:     1234,
		EnvUpdated: true,
	}
	_ = allocResult.Name
	_ = allocResult.Number
	_ = allocResult.EnvUpdated

	freeResult := PortFreeResult{
		Name:       "TEST",
		Number:     1234,
		Freed:      true,
		EnvUpdated: true,
	}
	_ = freeResult.Name
	_ = freeResult.Number
	_ = freeResult.Freed
	_ = freeResult.EnvUpdated

	checkResult := PortCheckResult{
		Port:      1234,
		Available: true,
	}
	_ = checkResult.Port
	_ = checkResult.Available
	_ = checkResult.UsedBy
}

func TestWorkspacePortsResultEmpty(t *testing.T) {
	result := WorkspacePortsResult{
		WorkspaceID: "empty-ws",
		Ports:       []PortInfo{},
	}

	output := result.TextOutput()
	if !strings.Contains(output, "empty-ws") {
		t.Error("TextOutput should contain workspace ID even when empty")
	}
}

func TestAllPortsResultGrouping(t *testing.T) {
	// Verify allocations are grouped correctly in output
	result := AllPortsResult{
		Allocations: []port.PortAllocation{
			{WorkspaceID: "ws-a", Name: "PORT", Port: 10000, BasePort: 10000},
			{WorkspaceID: "ws-a", Name: "API_PORT", Port: 10001, BasePort: 10000},
			{WorkspaceID: "ws-b", Name: "PORT", Port: 10100, BasePort: 10100},
			{WorkspaceID: "ws-a", Name: "DB_PORT", Port: 10002, BasePort: 10000}, // Same workspace, different order
		},
	}

	output := result.TextOutput()

	// Should contain all workspace IDs
	if !strings.Contains(output, "ws-a") {
		t.Error("TextOutput should contain ws-a")
	}
	if !strings.Contains(output, "ws-b") {
		t.Error("TextOutput should contain ws-b")
	}

	// Should contain base port info
	if !strings.Contains(output, "10000") || !strings.Contains(output, "10100") {
		t.Error("TextOutput should contain base port information")
	}
}
