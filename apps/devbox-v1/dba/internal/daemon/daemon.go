// Package daemon implements the DBA background daemon process that manages
// long-running state, workspace registration, and provides IPC for CLI commands.
package daemon

import (
	"context"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/dba-cli/dba/internal/config"
	"github.com/dba-cli/dba/internal/db"
)

// ShutdownCallback is a function called during graceful shutdown
// Agent #6 (Services) will register StopAllServices via this
type ShutdownCallback func() error

// Daemon is the main daemon process that manages workspaces and provides IPC
type Daemon struct {
	config   *config.Config
	listener net.Listener
	server   *http.Server

	// Subsystems (other agents will add to this)
	mu         sync.RWMutex
	workspaces map[string]*WorkspaceState

	// Health manager for monitoring
	healthManager *HealthManager

	// Shutdown callbacks
	shutdownCallbacks []ShutdownCallback
	callbackMu        sync.Mutex

	// Shutdown coordination
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup

	// Started timestamp
	startedAt time.Time

	// Logger
	logger *log.Logger
}

// ServiceState holds runtime state for a single service
// Added by Agent #6 (Services)
type ServiceState struct {
	Name      string    `json:"name"`
	Status    string    `json:"status"` // running, stopped, starting, stopping, error
	PID       int       `json:"pid,omitempty"`
	Port      int       `json:"port,omitempty"`
	Healthy   bool      `json:"healthy"`
	Restarts  int       `json:"restarts"`
	StartedAt time.Time `json:"started_at,omitempty"`
	LastCheck time.Time `json:"last_check,omitempty"`
	Error     string    `json:"error,omitempty"`
}

// WorkspaceState holds runtime state for a workspace
// Other agents will extend this
type WorkspaceState struct {
	ID         string    `json:"id"`
	Path       string    `json:"path"`
	LastActive time.Time `json:"last_active"`
	Status     string    `json:"status"` // running, stopped, error

	// Agent #6: Service state tracking
	ServiceStates map[string]*ServiceState `json:"service_states,omitempty"`

	// Agent #10 will add: LSPClient *lsp.Client
	// Agent #14 will add: WatchmanClock string
}

// New creates a new daemon instance
func New(cfg *config.Config) (*Daemon, error) {
	ctx, cancel := context.WithCancel(context.Background())

	d := &Daemon{
		config:            cfg,
		workspaces:        make(map[string]*WorkspaceState),
		healthManager:     NewHealthManager(),
		shutdownCallbacks: make([]ShutdownCallback, 0),
		ctx:               ctx,
		cancel:            cancel,
		startedAt:         time.Now(),
		logger:            log.Default(),
	}

	return d, nil
}

// Start starts the daemon
func (d *Daemon) Start() error {
	// Setup logging first
	if err := d.setupLogging(); err != nil {
		return fmt.Errorf("failed to setup logging: %w", err)
	}

	d.logger.Println("Starting DBA daemon...")

	// Clean up orphaned processes from previous daemon runs
	if err := d.cleanupOrphanedProcesses(); err != nil {
		d.logger.Printf("Warning: failed to clean up orphaned processes: %v", err)
	}

	// Initialize database
	if _, err := db.Get(); err != nil {
		return fmt.Errorf("failed to initialize database: %w", err)
	}

	// Create socket directory
	socketDir := filepath.Dir(d.config.Daemon.Socket)
	if err := os.MkdirAll(socketDir, 0755); err != nil {
		return fmt.Errorf("failed to create socket directory: %w", err)
	}

	// Check for stale socket file
	if err := d.checkAndCleanStaleSocket(); err != nil {
		return fmt.Errorf("failed to handle stale socket: %w", err)
	}

	// Create listener
	listener, err := net.Listen("unix", d.config.Daemon.Socket)
	if err != nil {
		return fmt.Errorf("failed to listen on socket: %w", err)
	}
	d.listener = listener

	// Set socket permissions (only user can access)
	if err := os.Chmod(d.config.Daemon.Socket, 0600); err != nil {
		d.logger.Printf("Warning: failed to set socket permissions: %v", err)
	}

	// Write PID file
	if err := d.writePIDFile(); err != nil {
		return fmt.Errorf("failed to write PID file: %w", err)
	}

	// Create HTTP server with router
	d.server = &http.Server{
		Handler: d.createRouter(),
	}

	// Start background health check loop
	d.wg.Add(1)
	go d.healthCheckLoop()

	// Handle signals for graceful shutdown
	go d.handleSignals()

	d.logger.Printf("DBA daemon started on %s (PID: %d)", d.config.Daemon.Socket, os.Getpid())

	// Serve (blocks until shutdown)
	if err := d.server.Serve(listener); err != http.ErrServerClosed {
		return err
	}

	return nil
}

// Stop stops the daemon gracefully
func (d *Daemon) Stop() error {
	d.logger.Println("DBA daemon shutting down...")

	// Cancel context to stop background goroutines
	d.cancel()

	// Run shutdown callbacks (e.g., StopAllServices from Agent #6)
	d.runShutdownCallbacks()

	// Shutdown HTTP server with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if d.server != nil {
		if err := d.server.Shutdown(ctx); err != nil {
			d.logger.Printf("HTTP server shutdown error: %v", err)
		}
	}

	// Wait for background tasks to complete
	d.wg.Wait()

	// Cleanup resources
	d.cleanup()

	d.logger.Println("DBA daemon stopped")
	return nil
}

// RegisterShutdownCallback registers a callback to be called during shutdown
// Agent #6 will use this to register StopAllServices
func (d *Daemon) RegisterShutdownCallback(cb ShutdownCallback) {
	d.callbackMu.Lock()
	defer d.callbackMu.Unlock()
	d.shutdownCallbacks = append(d.shutdownCallbacks, cb)
}

// runShutdownCallbacks executes all registered shutdown callbacks
func (d *Daemon) runShutdownCallbacks() {
	d.callbackMu.Lock()
	callbacks := make([]ShutdownCallback, len(d.shutdownCallbacks))
	copy(callbacks, d.shutdownCallbacks)
	d.callbackMu.Unlock()

	for i, cb := range callbacks {
		if err := cb(); err != nil {
			d.logger.Printf("Shutdown callback %d error: %v", i, err)
		}
	}
}

// setupLogging configures the daemon's logging to use the log file
func (d *Daemon) setupLogging() error {
	// Ensure log directory exists
	logDir := filepath.Dir(d.config.Daemon.LogFile)
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return err
	}

	// Open log file (append mode)
	logFile, err := os.OpenFile(d.config.Daemon.LogFile,
		os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return err
	}

	// Create a multi-writer to write to both file and stderr
	multiWriter := io.MultiWriter(os.Stderr, logFile)
	d.logger = log.New(multiWriter, "[dba-daemon] ", log.LstdFlags|log.Lshortfile)

	return nil
}

// cleanupOrphanedProcesses cleans up any orphaned processes from previous daemon runs
func (d *Daemon) cleanupOrphanedProcesses() error {
	// Check if there's a stale PID file
	pidFile := d.config.Daemon.PIDFile
	if pidBytes, err := os.ReadFile(pidFile); err == nil {
		oldPID, err := strconv.Atoi(strings.TrimSpace(string(pidBytes)))
		if err != nil || oldPID <= 0 {
			// Invalid PID file content, remove it
			d.logger.Printf("Removing invalid PID file: %s", pidFile)
			os.Remove(pidFile)
		} else {
			// Check if process still exists
			process, err := os.FindProcess(oldPID)
			if err == nil {
				// Try to send signal 0 to check if process exists
				if err := process.Signal(syscall.Signal(0)); err == nil {
					// Process exists - might be old daemon or unrelated process
					// Try to determine if it's actually our daemon
					d.logger.Printf("Found existing process with PID %d", oldPID)

					// Check if socket exists and is responsive
					if d.isOldDaemonAlive(oldPID) {
						return fmt.Errorf("another daemon instance is already running (PID: %d)", oldPID)
					}

					// Old daemon is not responding, try to kill it
					d.logger.Printf("Killing unresponsive old daemon (PID: %d)", oldPID)
					if err := process.Signal(syscall.SIGTERM); err != nil {
						d.logger.Printf("Warning: failed to terminate old daemon: %v", err)
					}

					// Give it time to shut down
					time.Sleep(500 * time.Millisecond)

					// Force kill if still running
					if err := process.Signal(syscall.Signal(0)); err == nil {
						process.Signal(syscall.SIGKILL)
						time.Sleep(100 * time.Millisecond)
					}
				}
			}

			// Remove stale PID file
			os.Remove(pidFile)
		}
	}

	// Remove stale socket file
	socketFile := d.config.Daemon.Socket
	if _, err := os.Stat(socketFile); err == nil {
		// Socket file exists, try to remove it
		os.Remove(socketFile)
	}

	return nil
}

// isOldDaemonAlive checks if an old daemon at the given PID is still responsive
func (d *Daemon) isOldDaemonAlive(pid int) bool {
	// Try to connect to the socket
	conn, err := net.DialTimeout("unix", d.config.Daemon.Socket, 1*time.Second)
	if err != nil {
		return false
	}
	conn.Close()
	return true
}

// checkAndCleanStaleSocket checks for and cleans up stale socket files
func (d *Daemon) checkAndCleanStaleSocket() error {
	socketPath := d.config.Daemon.Socket

	// Check if socket file exists
	if _, err := os.Stat(socketPath); os.IsNotExist(err) {
		return nil // No socket file, nothing to clean
	}

	// Try to connect to see if there's an active daemon
	conn, err := net.DialTimeout("unix", socketPath, 1*time.Second)
	if err == nil {
		conn.Close()
		return fmt.Errorf("another daemon instance is already running")
	}

	// Connection failed, socket is stale
	d.logger.Printf("Removing stale socket file: %s", socketPath)
	return os.Remove(socketPath)
}

// writePIDFile writes the daemon's PID to the configured PID file
func (d *Daemon) writePIDFile() error {
	pidDir := filepath.Dir(d.config.Daemon.PIDFile)
	if err := os.MkdirAll(pidDir, 0755); err != nil {
		return err
	}
	return os.WriteFile(d.config.Daemon.PIDFile,
		[]byte(fmt.Sprintf("%d", os.Getpid())), 0644)
}

// cleanup removes the socket and PID files
func (d *Daemon) cleanup() {
	os.Remove(d.config.Daemon.Socket)
	os.Remove(d.config.Daemon.PIDFile)
	db.Close()
}

// handleSignals handles OS signals for graceful shutdown
func (d *Daemon) handleSignals() {
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)

	for {
		select {
		case sig := <-sigCh:
			switch sig {
			case syscall.SIGHUP:
				// Reload configuration (future enhancement)
				d.logger.Println("Received SIGHUP, reload not implemented yet")
			case syscall.SIGINT, syscall.SIGTERM:
				d.logger.Printf("Received signal: %v", sig)
				d.Stop()
				return
			}
		case <-d.ctx.Done():
			// Context cancelled, daemon is stopping
			return
		}
	}
}

// healthCheckLoop periodically performs health checks on registered workspaces
func (d *Daemon) healthCheckLoop() {
	defer d.wg.Done()

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-d.ctx.Done():
			return
		case <-ticker.C:
			d.performHealthChecks()
		}
	}
}

// performHealthChecks runs health checks on all registered workspaces
func (d *Daemon) performHealthChecks() {
	// Run global health manager checks
	d.healthManager.RunChecks()

	// Check workspace-specific health
	d.mu.Lock()
	defer d.mu.Unlock()

	for id, ws := range d.workspaces {
		// Check if workspace path still exists
		if _, err := os.Stat(ws.Path); os.IsNotExist(err) {
			d.logger.Printf("Warning: workspace %s path no longer exists: %s", id, ws.Path)
			ws.Status = "error"
			continue
		}

		// Agent #6: Perform service health checks for this workspace
		d.checkWorkspaceServicesHealth(ws)
	}
}

// checkWorkspaceServicesHealth checks the health of all services in a workspace
// Added by Agent #6 (Services)
func (d *Daemon) checkWorkspaceServicesHealth(ws *WorkspaceState) {
	if ws.ServiceStates == nil {
		return
	}

	now := time.Now()
	allHealthy := true

	for name, svc := range ws.ServiceStates {
		if svc.Status != "running" {
			continue
		}

		// Check if service port is responding
		if svc.Port > 0 {
			healthy := d.checkPortHealth(svc.Port)
			svc.Healthy = healthy
			svc.LastCheck = now
			if !healthy {
				allHealthy = false
			}
		}

		// Log status changes
		if !svc.Healthy && svc.Status == "running" {
			d.logger.Printf("Service %s in workspace %s is unhealthy", name, ws.ID)
		}
	}

	// Update workspace status based on service health
	if len(ws.ServiceStates) > 0 && !allHealthy {
		ws.Status = "degraded"
	} else if ws.Status == "degraded" && allHealthy {
		ws.Status = "running"
	}
}

// checkPortHealth checks if a port is responding
// Added by Agent #6 (Services)
func (d *Daemon) checkPortHealth(port int) bool {
	addr := fmt.Sprintf("localhost:%d", port)
	conn, err := net.DialTimeout("tcp", addr, 2*time.Second)
	if err != nil {
		return false
	}
	conn.Close()
	return true
}

// GetHealthManager returns the daemon's health manager
func (d *Daemon) GetHealthManager() *HealthManager {
	return d.healthManager
}

// RegisterWorkspace registers a workspace with the daemon
func (d *Daemon) RegisterWorkspace(id, path string) {
	d.mu.Lock()
	defer d.mu.Unlock()

	d.workspaces[id] = &WorkspaceState{
		ID:            id,
		Path:          path,
		LastActive:    time.Now(),
		Status:        "running",
		ServiceStates: make(map[string]*ServiceState), // Agent #6: Initialize service states
	}

	d.logger.Printf("Registered workspace: %s at %s", id, path)
}

// UnregisterWorkspace removes a workspace from the daemon
func (d *Daemon) UnregisterWorkspace(id string) {
	d.mu.Lock()
	defer d.mu.Unlock()

	if _, exists := d.workspaces[id]; exists {
		delete(d.workspaces, id)
		d.logger.Printf("Unregistered workspace: %s", id)
	}
}

// GetWorkspaceState returns the state for a workspace
func (d *Daemon) GetWorkspaceState(id string) *WorkspaceState {
	d.mu.RLock()
	defer d.mu.RUnlock()

	if ws, exists := d.workspaces[id]; exists {
		// Return a copy to avoid race conditions
		copy := *ws
		return &copy
	}
	return nil
}

// GetWorkspaceCount returns the number of registered workspaces
func (d *Daemon) GetWorkspaceCount() int {
	d.mu.RLock()
	defer d.mu.RUnlock()

	return len(d.workspaces)
}

// UpdateWorkspaceActivity updates the last active timestamp for a workspace
func (d *Daemon) UpdateWorkspaceActivity(id string) {
	d.mu.Lock()
	defer d.mu.Unlock()

	if ws, exists := d.workspaces[id]; exists {
		ws.LastActive = time.Now()
	}
}

// UpdateWorkspaceStatus updates the status for a workspace
func (d *Daemon) UpdateWorkspaceStatus(id, status string) {
	d.mu.Lock()
	defer d.mu.Unlock()

	if ws, exists := d.workspaces[id]; exists {
		ws.Status = status
		ws.LastActive = time.Now()
	}
}

// GetAllWorkspaces returns all registered workspaces
func (d *Daemon) GetAllWorkspaces() []*WorkspaceState {
	d.mu.RLock()
	defer d.mu.RUnlock()

	workspaces := make([]*WorkspaceState, 0, len(d.workspaces))
	for _, ws := range d.workspaces {
		// Return copies to avoid race conditions
		copy := *ws
		workspaces = append(workspaces, &copy)
	}
	return workspaces
}

// Uptime returns the daemon uptime duration
func (d *Daemon) Uptime() time.Duration {
	return time.Since(d.startedAt)
}

// Context returns the daemon's context for use by subsystems
func (d *Daemon) Context() context.Context {
	return d.ctx
}

// Logger returns the daemon's logger for use by subsystems
func (d *Daemon) Logger() *log.Logger {
	return d.logger
}

// ═══════════════════════════════════════════════════════════════════════════════
// Service State Management - Added by Agent #6 (Services)
// ═══════════════════════════════════════════════════════════════════════════════

// UpdateServiceState updates the state of a service in a workspace
func (d *Daemon) UpdateServiceState(workspaceID, serviceName string, state *ServiceState) {
	d.mu.Lock()
	defer d.mu.Unlock()

	ws, exists := d.workspaces[workspaceID]
	if !exists {
		return
	}

	if ws.ServiceStates == nil {
		ws.ServiceStates = make(map[string]*ServiceState)
	}

	ws.ServiceStates[serviceName] = state
	ws.LastActive = time.Now()
}

// GetServiceState returns the state of a specific service
func (d *Daemon) GetServiceState(workspaceID, serviceName string) *ServiceState {
	d.mu.RLock()
	defer d.mu.RUnlock()

	ws, exists := d.workspaces[workspaceID]
	if !exists || ws.ServiceStates == nil {
		return nil
	}

	if svc, ok := ws.ServiceStates[serviceName]; ok {
		// Return a copy
		copy := *svc
		return &copy
	}
	return nil
}

// GetAllServiceStates returns all service states for a workspace
func (d *Daemon) GetAllServiceStates(workspaceID string) map[string]*ServiceState {
	d.mu.RLock()
	defer d.mu.RUnlock()

	ws, exists := d.workspaces[workspaceID]
	if !exists || ws.ServiceStates == nil {
		return nil
	}

	// Return copies
	result := make(map[string]*ServiceState, len(ws.ServiceStates))
	for name, svc := range ws.ServiceStates {
		copy := *svc
		result[name] = &copy
	}
	return result
}

// RemoveServiceState removes a service state from a workspace
func (d *Daemon) RemoveServiceState(workspaceID, serviceName string) {
	d.mu.Lock()
	defer d.mu.Unlock()

	ws, exists := d.workspaces[workspaceID]
	if !exists || ws.ServiceStates == nil {
		return
	}

	delete(ws.ServiceStates, serviceName)
}

// SetServiceHealth updates the health status of a service
func (d *Daemon) SetServiceHealth(workspaceID, serviceName string, healthy bool, message string) {
	d.mu.Lock()
	defer d.mu.Unlock()

	ws, exists := d.workspaces[workspaceID]
	if !exists || ws.ServiceStates == nil {
		return
	}

	if svc, ok := ws.ServiceStates[serviceName]; ok {
		svc.Healthy = healthy
		svc.LastCheck = time.Now()
		if !healthy {
			svc.Error = message
		} else {
			svc.Error = ""
		}
	}
}

// IncrementServiceRestarts increments the restart counter for a service
func (d *Daemon) IncrementServiceRestarts(workspaceID, serviceName string) {
	d.mu.Lock()
	defer d.mu.Unlock()

	ws, exists := d.workspaces[workspaceID]
	if !exists || ws.ServiceStates == nil {
		return
	}

	if svc, ok := ws.ServiceStates[serviceName]; ok {
		svc.Restarts++
	}
}

// StopAllServicesCallback creates a shutdown callback that stops all services
// This should be registered with RegisterShutdownCallback during daemon startup
func (d *Daemon) StopAllServicesCallback() ShutdownCallback {
	return func() error {
		d.logger.Println("Stopping all services in all workspaces...")

		d.mu.Lock()
		workspaces := make([]*WorkspaceState, 0, len(d.workspaces))
		for _, ws := range d.workspaces {
			workspaces = append(workspaces, ws)
		}
		d.mu.Unlock()

		var lastErr error
		for _, ws := range workspaces {
			if ws.ServiceStates == nil || len(ws.ServiceStates) == 0 {
				continue
			}

			d.logger.Printf("Stopping services in workspace %s", ws.ID)

			// Update service states to stopping
			d.mu.Lock()
			for name, svc := range ws.ServiceStates {
				if svc.Status == "running" {
					svc.Status = "stopping"
					d.logger.Printf("  Stopping service: %s", name)
				}
			}
			d.mu.Unlock()

			// Note: Actual service stopping is done via process-compose
			// This callback just updates the state tracking
		}

		return lastErr
	}
}

// GetWorkspaceServicesSummary returns a summary of services for a workspace
func (d *Daemon) GetWorkspaceServicesSummary(workspaceID string) (running, stopped, unhealthy int) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	ws, exists := d.workspaces[workspaceID]
	if !exists || ws.ServiceStates == nil {
		return 0, 0, 0
	}

	for _, svc := range ws.ServiceStates {
		switch svc.Status {
		case "running":
			running++
			if !svc.Healthy {
				unhealthy++
			}
		case "stopped":
			stopped++
		}
	}

	return running, stopped, unhealthy
}
