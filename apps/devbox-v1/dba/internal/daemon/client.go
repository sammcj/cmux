// Package daemon provides the client for communicating with the DBA daemon.
package daemon

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/dba-cli/dba/internal/config"
)

// Client is a client for communicating with the daemon
type Client struct {
	socketPath string
	httpClient *http.Client
}

// NewClient creates a new daemon client
func NewClient(cfg *config.Config) *Client {
	return &Client{
		socketPath: cfg.Daemon.Socket,
		httpClient: &http.Client{
			Transport: &http.Transport{
				DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
					var d net.Dialer
					return d.DialContext(ctx, "unix", cfg.Daemon.Socket)
				},
			},
			Timeout: 30 * time.Second,
		},
	}
}

// IsRunning checks if the daemon is running
func (c *Client) IsRunning() bool {
	resp, err := c.httpClient.Get("http://unix/health")
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}

// Status returns the daemon status
func (c *Client) Status() (*StatusResponse, error) {
	resp, err := c.httpClient.Get("http://unix/status")
	if err != nil {
		return nil, fmt.Errorf("daemon not running: %w", err)
	}
	defer resp.Body.Close()

	var status StatusResponse
	if err := json.NewDecoder(resp.Body).Decode(&status); err != nil {
		return nil, fmt.Errorf("failed to decode status: %w", err)
	}

	return &status, nil
}

// RegisterWorkspace registers a workspace with the daemon
func (c *Client) RegisterWorkspace(id, path string) error {
	data, err := json.Marshal(WorkspaceRegisterRequest{
		ID:   id,
		Path: path,
	})
	if err != nil {
		return err
	}

	resp, err := c.httpClient.Post("http://unix/workspace/register",
		"application/json", bytes.NewReader(data))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to register workspace: %s", body)
	}

	return nil
}

// UnregisterWorkspace unregisters a workspace from the daemon
func (c *Client) UnregisterWorkspace(id string) error {
	data, err := json.Marshal(WorkspaceUnregisterRequest{ID: id})
	if err != nil {
		return err
	}

	resp, err := c.httpClient.Post("http://unix/workspace/unregister",
		"application/json", bytes.NewReader(data))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to unregister workspace: %s", body)
	}

	return nil
}

// GetWorkspaceState returns the state of a workspace
func (c *Client) GetWorkspaceState(id string) (*WorkspaceState, error) {
	resp, err := c.httpClient.Get(fmt.Sprintf("http://unix/workspace/state?id=%s", id))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("workspace not found: %s", id)
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to get workspace state: %s", body)
	}

	var state WorkspaceState
	if err := json.NewDecoder(resp.Body).Decode(&state); err != nil {
		return nil, err
	}

	return &state, nil
}

// ListWorkspaces returns all registered workspaces
func (c *Client) ListWorkspaces() ([]*WorkspaceState, error) {
	resp, err := c.httpClient.Get("http://unix/workspace/list")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var listResp WorkspaceListResponse
	if err := json.NewDecoder(resp.Body).Decode(&listResp); err != nil {
		return nil, err
	}

	return listResp.Workspaces, nil
}

// UpdateWorkspaceActivity updates the activity timestamp for a workspace
func (c *Client) UpdateWorkspaceActivity(id string) error {
	data, err := json.Marshal(WorkspaceActivityRequest{ID: id})
	if err != nil {
		return err
	}

	resp, err := c.httpClient.Post("http://unix/workspace/activity",
		"application/json", bytes.NewReader(data))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return nil
}

// SyncResult represents the result of a sync wait operation
type SyncResult struct {
	Synced bool  `json:"synced"`
	WaitMs int64 `json:"wait_ms"`
}

// WaitForSync waits for the sync barrier
func (c *Client) WaitForSync(workspaceID string, timeout time.Duration) (*SyncResult, error) {
	url := fmt.Sprintf("http://unix/sync/wait?id=%s&timeout=%s",
		workspaceID, timeout.String())

	resp, err := c.httpClient.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result SyncWaitResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return &SyncResult{
		Synced: result.Synced,
		WaitMs: result.WaitMs,
	}, nil
}

// EnsureRunning starts the daemon if it's not running
func EnsureRunning(cfg *config.Config) error {
	client := NewClient(cfg)

	if client.IsRunning() {
		return nil
	}

	// Start daemon in background
	return StartInBackground(cfg)
}

// StartInBackground starts the daemon as a background process
func StartInBackground(cfg *config.Config) error {
	// Find the dba executable
	executable, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to find executable: %w", err)
	}

	// Start daemon process with --foreground flag
	cmd := exec.Command(executable, "daemon", "start", "--foreground")
	cmd.Stdout = nil
	cmd.Stderr = nil
	cmd.Stdin = nil

	// Detach from parent process
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true,
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start daemon: %w", err)
	}

	// Wait for daemon to be ready
	client := NewClient(cfg)
	deadline := time.Now().Add(10 * time.Second)

	for time.Now().Before(deadline) {
		if client.IsRunning() {
			return nil
		}
		time.Sleep(100 * time.Millisecond)
	}

	return fmt.Errorf("daemon failed to start within timeout")
}

// StopDaemon stops the running daemon
func StopDaemon(cfg *config.Config) error {
	// Read PID file
	pidBytes, err := os.ReadFile(cfg.Daemon.PIDFile)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("daemon not running (no PID file)")
		}
		return fmt.Errorf("failed to read PID file: %w", err)
	}

	pid, err := strconv.Atoi(strings.TrimSpace(string(pidBytes)))
	if err != nil {
		return fmt.Errorf("invalid PID file content")
	}

	// Send SIGTERM to the daemon process
	process, err := os.FindProcess(pid)
	if err != nil {
		return fmt.Errorf("failed to find process %d: %w", pid, err)
	}

	if err := process.Signal(syscall.SIGTERM); err != nil {
		// Process might already be dead
		if err.Error() == "os: process already finished" {
			// Clean up stale files
			os.Remove(cfg.Daemon.Socket)
			os.Remove(cfg.Daemon.PIDFile)
			return nil
		}
		return fmt.Errorf("failed to send signal: %w", err)
	}

	// Wait for daemon to stop
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		// Check if socket still exists
		if _, err := os.Stat(cfg.Daemon.Socket); os.IsNotExist(err) {
			return nil
		}
		time.Sleep(100 * time.Millisecond)
	}

	// Force kill if still running
	if err := process.Signal(syscall.SIGKILL); err != nil {
		// Ignore errors here
	}

	// Clean up any remaining files
	os.Remove(cfg.Daemon.Socket)
	os.Remove(cfg.Daemon.PIDFile)

	return nil
}

// GetDaemonPID returns the PID of the running daemon, or 0 if not running
func GetDaemonPID(cfg *config.Config) int {
	pidBytes, err := os.ReadFile(cfg.Daemon.PIDFile)
	if err != nil {
		return 0
	}

	pid, err := strconv.Atoi(strings.TrimSpace(string(pidBytes)))
	if err != nil {
		return 0
	}

	// Verify process is still running
	process, err := os.FindProcess(pid)
	if err != nil {
		return 0
	}

	// On Unix, this doesn't actually check if process exists
	// We need to send signal 0 to check
	if err := process.Signal(syscall.Signal(0)); err != nil {
		return 0
	}

	return pid
}
