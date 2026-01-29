// Package daemon implements the DBA daemon HTTP server and API routes.
package daemon

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"
)

// createRouter creates the HTTP router with all API endpoints
func (d *Daemon) createRouter() http.Handler {
	mux := http.NewServeMux()

	// Health check endpoint
	mux.HandleFunc("/health", d.handleHealth)

	// Status endpoint (detailed daemon status)
	mux.HandleFunc("/status", d.handleStatus)

	// Workspace registration endpoints (called by Agent #4)
	mux.HandleFunc("/workspace/register", d.handleWorkspaceRegister)
	mux.HandleFunc("/workspace/unregister", d.handleWorkspaceUnregister)
	mux.HandleFunc("/workspace/state", d.handleWorkspaceState)
	mux.HandleFunc("/workspace/list", d.handleWorkspaceList)
	mux.HandleFunc("/workspace/activity", d.handleWorkspaceActivity)

	// Sync barrier stub (Agent #14 will implement the actual sync logic)
	mux.HandleFunc("/sync/wait", d.handleSyncWait)

	return mux
}

// HealthResponse represents the response from the health endpoint
type HealthResponse struct {
	Status string `json:"status"`
}

// handleHealth handles health check requests
func (d *Daemon) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(HealthResponse{
		Status: "ok",
	})
}

// StatusResponse represents the response from the status endpoint
type StatusResponse struct {
	Running          bool   `json:"running"`
	PID              int    `json:"pid"`
	Socket           string `json:"socket"`
	WorkspacesActive int    `json:"workspaces_active"`
	UptimeSeconds    int64  `json:"uptime_seconds"`
}

// handleStatus handles status requests
func (d *Daemon) handleStatus(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	response := StatusResponse{
		Running:          true,
		PID:              os.Getpid(),
		Socket:           d.config.Daemon.Socket,
		WorkspacesActive: d.GetWorkspaceCount(),
		UptimeSeconds:    int64(d.Uptime().Seconds()),
	}

	json.NewEncoder(w).Encode(response)
}

// WorkspaceRegisterRequest represents a workspace registration request
type WorkspaceRegisterRequest struct {
	ID   string `json:"id"`
	Path string `json:"path"`
}

// WorkspaceRegisterResponse represents a workspace registration response
type WorkspaceRegisterResponse struct {
	Registered bool `json:"registered"`
}

// handleWorkspaceRegister handles workspace registration
func (d *Daemon) handleWorkspaceRegister(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req WorkspaceRegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if req.ID == "" || req.Path == "" {
		http.Error(w, "id and path are required", http.StatusBadRequest)
		return
	}

	d.RegisterWorkspace(req.ID, req.Path)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(WorkspaceRegisterResponse{Registered: true})
}

// WorkspaceUnregisterRequest represents a workspace unregister request
type WorkspaceUnregisterRequest struct {
	ID string `json:"id"`
}

// WorkspaceUnregisterResponse represents a workspace unregister response
type WorkspaceUnregisterResponse struct {
	Unregistered bool `json:"unregistered"`
}

// handleWorkspaceUnregister handles workspace unregistration
func (d *Daemon) handleWorkspaceUnregister(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req WorkspaceUnregisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if req.ID == "" {
		http.Error(w, "id is required", http.StatusBadRequest)
		return
	}

	d.UnregisterWorkspace(req.ID)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(WorkspaceUnregisterResponse{Unregistered: true})
}

// handleWorkspaceState handles workspace state requests
func (d *Daemon) handleWorkspaceState(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	if id == "" {
		http.Error(w, "missing id parameter", http.StatusBadRequest)
		return
	}

	state := d.GetWorkspaceState(id)
	if state == nil {
		http.Error(w, "workspace not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(state)
}

// WorkspaceListResponse represents a list of workspaces
type WorkspaceListResponse struct {
	Workspaces []*WorkspaceState `json:"workspaces"`
}

// handleWorkspaceList handles workspace list requests
func (d *Daemon) handleWorkspaceList(w http.ResponseWriter, r *http.Request) {
	workspaces := d.GetAllWorkspaces()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(WorkspaceListResponse{Workspaces: workspaces})
}

// WorkspaceActivityRequest represents a workspace activity update request
type WorkspaceActivityRequest struct {
	ID string `json:"id"`
}

// WorkspaceActivityResponse represents a workspace activity update response
type WorkspaceActivityResponse struct {
	Updated bool `json:"updated"`
}

// handleWorkspaceActivity handles workspace activity updates
func (d *Daemon) handleWorkspaceActivity(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req WorkspaceActivityRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if req.ID == "" {
		http.Error(w, "id is required", http.StatusBadRequest)
		return
	}

	d.UpdateWorkspaceActivity(req.ID)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(WorkspaceActivityResponse{Updated: true})
}

// SyncWaitResponse represents a sync wait response
type SyncWaitResponse struct {
	Synced      bool   `json:"synced"`
	WaitMs      int64  `json:"wait_ms"`
	WorkspaceID string `json:"workspace_id"`
	Timeout     string `json:"timeout"`
	Message     string `json:"message,omitempty"` // Informational message
}

// SyncBarrier is the interface for sync barrier implementations
// Agent #14 should implement this interface using watchman or equivalent
type SyncBarrier interface {
	// Wait waits for all pending file changes to be synced
	// Returns the time waited in milliseconds and any error
	Wait(workspaceID string, timeout time.Duration) (waitMs int64, err error)
}

// syncBarrier is the current sync barrier implementation
// This will be set by Agent #14 when they implement the sync functionality
var syncBarrier SyncBarrier

// SetSyncBarrier sets the sync barrier implementation
// Agent #14 should call this to register their implementation
func SetSyncBarrier(sb SyncBarrier) {
	syncBarrier = sb
}

// handleSyncWait handles sync barrier requests
// This endpoint allows clients to wait for file system changes to be synced.
//
// IMPLEMENTATION NOTE (for Agent #14):
// This is currently a stub that returns immediately. To implement proper sync:
// 1. Integrate with watchman (or equivalent) to track file changes
// 2. Implement the SyncBarrier interface
// 3. Call SetSyncBarrier() to register your implementation
// 4. The sync should wait until all pending changes are written to disk
//
// Query parameters:
// - id: workspace ID (optional, for workspace-specific sync)
// - timeout: maximum time to wait (default: 10s)
func (d *Daemon) handleSyncWait(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	timeout := r.URL.Query().Get("timeout")

	if timeout == "" {
		timeout = "10s"
	}

	// Parse timeout to validate it
	timeoutDuration, err := time.ParseDuration(timeout)
	if err != nil {
		http.Error(w, "invalid timeout format", http.StatusBadRequest)
		return
	}

	var waitMs int64
	var synced bool
	var message string

	// Check if a real sync barrier is registered
	if syncBarrier != nil {
		waitMs, err = syncBarrier.Wait(id, timeoutDuration)
		if err != nil {
			message = fmt.Sprintf("sync error: %v", err)
			synced = false
		} else {
			synced = true
		}
	} else {
		// No sync barrier registered - return immediately with a note
		synced = true
		waitMs = 0
		message = "sync barrier not implemented (stub response)"
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(SyncWaitResponse{
		Synced:      synced,
		WaitMs:      waitMs,
		WorkspaceID: id,
		Timeout:     timeout,
		Message:     message,
	})
}
