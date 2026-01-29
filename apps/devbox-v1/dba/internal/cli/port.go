// internal/cli/port.go
package cli

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/dba-cli/dba/internal/port"
)

var portCmd = &cobra.Command{
	Use:   "port",
	Short: "Manage port allocations",
	Long:  `Manage port allocations for workspaces. List, allocate, and free ports.`,
}

var portListCmd = &cobra.Command{
	Use:   "list",
	Short: "List allocated ports",
	Long:  `List all allocated ports for the current workspace or all workspaces.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx, err := NewCLIContext()
		if err != nil {
			OutputError(err)
			return err
		}
		defer ctx.Cancel()

		allocator, err := port.NewAllocator(ctx.Config.Ports)
		if err != nil {
			OutputError(err)
			return err
		}

		showAll, _ := cmd.Flags().GetBool("all")

		if showAll {
			return listAllPorts(allocator)
		}

		// List ports for current workspace
		if err := ctx.RequireWorkspace(); err != nil {
			OutputError(err)
			return err
		}

		return listWorkspacePorts(allocator, ctx.Workspace.ID)
	},
}

// PortInfo represents port information for output
type PortInfo struct {
	Name   string `json:"name"`
	Number int    `json:"number"`
	InUse  bool   `json:"in_use"`
	PID    int    `json:"pid,omitempty"`
}

// WorkspacePortsResult is the result of listing workspace ports
type WorkspacePortsResult struct {
	WorkspaceID string     `json:"workspace_id"`
	Ports       []PortInfo `json:"ports"`
}

// TextOutput returns human-readable output
func (r WorkspacePortsResult) TextOutput() string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Ports for workspace %s:\n", r.WorkspaceID))
	for _, p := range r.Ports {
		status := "free"
		if p.InUse {
			status = fmt.Sprintf("in use (PID: %d)", p.PID)
		}
		sb.WriteString(fmt.Sprintf("  %s: %d (%s)\n", p.Name, p.Number, status))
	}
	return sb.String()
}

func listWorkspacePorts(allocator *port.Allocator, workspaceID string) error {
	ports, err := allocator.GetWorkspacePorts(workspaceID)
	if err != nil {
		OutputError(err)
		return err
	}

	result := WorkspacePortsResult{
		WorkspaceID: workspaceID,
		Ports:       make([]PortInfo, 0, len(ports)),
	}

	for name, portNum := range ports {
		info := PortInfo{
			Name:   name,
			Number: portNum,
			InUse:  !port.IsPortFree(portNum),
		}

		if info.InUse {
			if proc, _ := port.GetPortProcess(portNum); proc != nil {
				info.PID = proc.PID
			}
		}

		result.Ports = append(result.Ports, info)
	}

	return OutputResult(result)
}

// AllPortsResult is the result of listing all ports
type AllPortsResult struct {
	Allocations []port.PortAllocation `json:"allocations"`
}

// TextOutput returns human-readable output
func (r AllPortsResult) TextOutput() string {
	if len(r.Allocations) == 0 {
		return "No port allocations found."
	}

	var sb strings.Builder
	sb.WriteString("All port allocations:\n")

	currentWorkspace := ""
	for _, alloc := range r.Allocations {
		if alloc.WorkspaceID != currentWorkspace {
			currentWorkspace = alloc.WorkspaceID
			sb.WriteString(fmt.Sprintf("\nWorkspace: %s (base: %d)\n", alloc.WorkspaceID, alloc.BasePort))
		}
		sb.WriteString(fmt.Sprintf("  %s: %d\n", alloc.Name, alloc.Port))
	}
	return sb.String()
}

func listAllPorts(allocator *port.Allocator) error {
	allocations, err := allocator.ListAllAllocations()
	if err != nil {
		OutputError(err)
		return err
	}

	result := AllPortsResult{
		Allocations: allocations,
	}

	return OutputResult(result)
}

var portAllocCmd = &cobra.Command{
	Use:   "alloc <name>",
	Short: "Allocate a new port",
	Long:  `Allocate a new named port for the current workspace.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx, err := NewCLIContext()
		if err != nil {
			OutputError(err)
			return err
		}
		defer ctx.Cancel()

		if err := ctx.RequireWorkspace(); err != nil {
			OutputError(err)
			return err
		}

		name := args[0]

		allocator, err := port.NewAllocator(ctx.Config.Ports)
		if err != nil {
			OutputError(err)
			return err
		}

		portNum, err := allocator.AllocateAdditional(ctx.Workspace.ID, name)
		if err != nil {
			OutputError(err)
			return err
		}

		// Update devbox.json with the new port
		devboxPath := port.GetDevboxPath(ctx.Workspace.Path)
		envUpdated := false
		if _, statErr := os.Stat(devboxPath); statErr == nil {
			if updateErr := port.UpdateDevboxEnv(devboxPath, name, portNum); updateErr == nil {
				envUpdated = true
			}
		}

		result := PortAllocResult{
			Name:       name,
			Number:     portNum,
			EnvUpdated: envUpdated,
		}

		return OutputResult(result)
	},
}

// PortAllocResult is the result of allocating a port
type PortAllocResult struct {
	Name       string `json:"name"`
	Number     int    `json:"number"`
	EnvUpdated bool   `json:"env_updated"`
}

// TextOutput returns human-readable output
func (r PortAllocResult) TextOutput() string {
	return fmt.Sprintf("Allocated port %s: %d", r.Name, r.Number)
}

var portFreeCmd = &cobra.Command{
	Use:   "free <name>",
	Short: "Free an allocated port",
	Long:  `Free a named port allocation for the current workspace.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx, err := NewCLIContext()
		if err != nil {
			OutputError(err)
			return err
		}
		defer ctx.Cancel()

		if err := ctx.RequireWorkspace(); err != nil {
			OutputError(err)
			return err
		}

		name := args[0]

		allocator, err := port.NewAllocator(ctx.Config.Ports)
		if err != nil {
			OutputError(err)
			return err
		}

		// Get port number before freeing
		ports, err := allocator.GetWorkspacePorts(ctx.Workspace.ID)
		if err != nil {
			OutputError(err)
			return err
		}

		portNum, ok := ports[name]
		if !ok {
			err := fmt.Errorf("port %s not allocated", name)
			OutputError(err)
			return err
		}

		if err := allocator.Free(ctx.Workspace.ID, name); err != nil {
			OutputError(err)
			return err
		}

		// Remove from devbox.json
		devboxPath := port.GetDevboxPath(ctx.Workspace.Path)
		envUpdated := false
		if _, statErr := os.Stat(devboxPath); statErr == nil {
			if removeErr := port.RemoveDevboxEnv(devboxPath, name); removeErr == nil {
				envUpdated = true
			}
		}

		result := PortFreeResult{
			Name:       name,
			Number:     portNum,
			Freed:      true,
			EnvUpdated: envUpdated,
		}

		return OutputResult(result)
	},
}

// PortFreeResult is the result of freeing a port
type PortFreeResult struct {
	Name       string `json:"name"`
	Number     int    `json:"number"`
	Freed      bool   `json:"freed"`
	EnvUpdated bool   `json:"env_updated"`
}

// TextOutput returns human-readable output
func (r PortFreeResult) TextOutput() string {
	return fmt.Sprintf("Freed port %s: %d", r.Name, r.Number)
}

var portCheckCmd = &cobra.Command{
	Use:   "check <port>",
	Short: "Check if a port is available",
	Long:  `Check if a specific port number is available or in use.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		portNum, err := strconv.Atoi(args[0])
		if err != nil {
			err := fmt.Errorf("invalid port number: %s", args[0])
			OutputError(err)
			return err
		}

		available := port.IsPortFree(portNum)

		result := PortCheckResult{
			Port:      portNum,
			Available: available,
		}

		if !available {
			result.UsedBy, _ = port.GetPortProcess(portNum)
		}

		return OutputResult(result)
	},
}

// PortCheckResult is the result of checking a port
type PortCheckResult struct {
	Port      int               `json:"port"`
	Available bool              `json:"available"`
	UsedBy    *port.ProcessInfo `json:"used_by,omitempty"`
}

// TextOutput returns human-readable output
func (r PortCheckResult) TextOutput() string {
	if r.Available {
		return fmt.Sprintf("Port %d is available", r.Port)
	}
	if r.UsedBy != nil {
		return fmt.Sprintf("Port %d is in use by %s (PID: %d)", r.Port, r.UsedBy.Process, r.UsedBy.PID)
	}
	return fmt.Sprintf("Port %d is in use", r.Port)
}

func init() {
	portListCmd.Flags().Bool("all", false, "Show all workspaces")

	portCmd.AddCommand(portListCmd)
	portCmd.AddCommand(portAllocCmd)
	portCmd.AddCommand(portFreeCmd)
	portCmd.AddCommand(portCheckCmd)

	// Register with root command
	AddCommand(portCmd)
}
