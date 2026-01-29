// Package daemon provides health check framework for the DBA daemon.
package daemon

import (
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
)

// HealthChecker defines the interface for health checks
// Other agents can implement this interface to add their own health checks
type HealthChecker interface {
	// Name returns the name of this health check
	Name() string
	// Check performs the health check and returns any error
	Check() error
	// IsHealthy returns true if the last check was healthy
	IsHealthy() bool
}

// HealthStatus represents the status of a health check
type HealthStatus struct {
	Name      string    `json:"name"`
	Healthy   bool      `json:"healthy"`
	Message   string    `json:"message,omitempty"`
	LastCheck time.Time `json:"last_check"`
	Duration  int64     `json:"duration_ms"`
}

// HealthManager manages all health checks for the daemon
type HealthManager struct {
	mu       sync.RWMutex
	checkers map[string]HealthChecker
	statuses map[string]*HealthStatus
}

// NewHealthManager creates a new health manager
func NewHealthManager() *HealthManager {
	return &HealthManager{
		checkers: make(map[string]HealthChecker),
		statuses: make(map[string]*HealthStatus),
	}
}

// Register registers a new health checker
func (hm *HealthManager) Register(checker HealthChecker) {
	hm.mu.Lock()
	defer hm.mu.Unlock()

	name := checker.Name()
	hm.checkers[name] = checker
	hm.statuses[name] = &HealthStatus{
		Name:    name,
		Healthy: true,
	}
}

// Unregister removes a health checker
func (hm *HealthManager) Unregister(name string) {
	hm.mu.Lock()
	defer hm.mu.Unlock()

	delete(hm.checkers, name)
	delete(hm.statuses, name)
}

// RunChecks runs all registered health checks
func (hm *HealthManager) RunChecks() {
	hm.mu.Lock()
	defer hm.mu.Unlock()

	for name, checker := range hm.checkers {
		start := time.Now()
		err := checker.Check()
		duration := time.Since(start)

		status := hm.statuses[name]
		status.LastCheck = time.Now()
		status.Duration = duration.Milliseconds()

		if err != nil {
			status.Healthy = false
			status.Message = err.Error()
		} else {
			status.Healthy = true
			status.Message = ""
		}
	}
}

// GetStatus returns the status of a specific health check
func (hm *HealthManager) GetStatus(name string) *HealthStatus {
	hm.mu.RLock()
	defer hm.mu.RUnlock()

	if status, ok := hm.statuses[name]; ok {
		// Return a copy to avoid race conditions
		copy := *status
		return &copy
	}
	return nil
}

// GetAllStatuses returns the status of all health checks
func (hm *HealthManager) GetAllStatuses() []*HealthStatus {
	hm.mu.RLock()
	defer hm.mu.RUnlock()

	statuses := make([]*HealthStatus, 0, len(hm.statuses))
	for _, status := range hm.statuses {
		copy := *status
		statuses = append(statuses, &copy)
	}
	return statuses
}

// IsAllHealthy returns true if all health checks are healthy
func (hm *HealthManager) IsAllHealthy() bool {
	hm.mu.RLock()
	defer hm.mu.RUnlock()

	for _, status := range hm.statuses {
		if !status.Healthy {
			return false
		}
	}
	return true
}

// PortHealthChecker checks if a port is responding
type PortHealthChecker struct {
	name    string
	host    string
	port    int
	timeout time.Duration
	healthy bool
}

// NewPortHealthChecker creates a new port health checker
func NewPortHealthChecker(name, host string, port int, timeout time.Duration) *PortHealthChecker {
	return &PortHealthChecker{
		name:    name,
		host:    host,
		port:    port,
		timeout: timeout,
		healthy: false,
	}
}

// Name returns the name of this health check
func (p *PortHealthChecker) Name() string {
	return p.name
}

// Check performs the health check
func (p *PortHealthChecker) Check() error {
	addr := fmt.Sprintf("%s:%d", p.host, p.port)
	conn, err := net.DialTimeout("tcp", addr, p.timeout)
	if err != nil {
		p.healthy = false
		return fmt.Errorf("port %d not responding: %w", p.port, err)
	}
	conn.Close()
	p.healthy = true
	return nil
}

// IsHealthy returns true if the last check was healthy
func (p *PortHealthChecker) IsHealthy() bool {
	return p.healthy
}

// ProcessHealthChecker checks if a process is running
// Implemented by Agent #6 (Services)
type ProcessHealthChecker struct {
	name       string
	pidFile    string
	healthy    bool
	lastPID    int
	lastError  string
	lastCheck  time.Time
}

// NewProcessHealthChecker creates a new process health checker
func NewProcessHealthChecker(name, pidFile string) *ProcessHealthChecker {
	return &ProcessHealthChecker{
		name:    name,
		pidFile: pidFile,
		healthy: false,
	}
}

// Name returns the name of this health check
func (p *ProcessHealthChecker) Name() string {
	return p.name
}

// Check performs the health check
// Implemented by Agent #6 (Services)
func (p *ProcessHealthChecker) Check() error {
	p.lastCheck = time.Now()

	// Read PID file
	pidBytes, err := readPIDFile(p.pidFile)
	if err != nil {
		p.healthy = false
		p.lastPID = 0
		p.lastError = fmt.Sprintf("failed to read PID file: %v", err)
		return fmt.Errorf("PID file error: %w", err)
	}

	pid := pidBytes
	p.lastPID = pid

	// Check if process exists
	if !isProcessRunning(pid) {
		p.healthy = false
		p.lastError = fmt.Sprintf("process %d is not running", pid)
		return fmt.Errorf("process %d is not running", pid)
	}

	p.healthy = true
	p.lastError = ""
	return nil
}

// IsHealthy returns true if the last check was healthy
func (p *ProcessHealthChecker) IsHealthy() bool {
	return p.healthy
}

// GetLastPID returns the last known PID
func (p *ProcessHealthChecker) GetLastPID() int {
	return p.lastPID
}

// GetLastError returns the last error message
func (p *ProcessHealthChecker) GetLastError() string {
	return p.lastError
}

// readPIDFile reads and parses a PID file
func readPIDFile(path string) (int, error) {
	data, err := readFile(path)
	if err != nil {
		return 0, err
	}

	pidStr := trimSpace(string(data))
	if pidStr == "" {
		return 0, fmt.Errorf("empty PID file")
	}

	pid, err := parseInt(pidStr)
	if err != nil {
		return 0, fmt.Errorf("invalid PID: %w", err)
	}

	if pid <= 0 {
		return 0, fmt.Errorf("invalid PID: %d", pid)
	}

	return pid, nil
}

// isProcessRunning checks if a process with the given PID is running
func isProcessRunning(pid int) bool {
	// On Unix systems, sending signal 0 checks if process exists
	// without actually sending a signal
	process, err := findProcess(pid)
	if err != nil {
		return false
	}

	// Try to send signal 0 (existence check)
	err = process.Signal(signalZero())
	return err == nil
}

// WorkspaceHealthChecker checks the health of a workspace
type WorkspaceHealthChecker struct {
	name        string
	workspaceID string
	daemon      *Daemon
	healthy     bool
}

// NewWorkspaceHealthChecker creates a new workspace health checker
func NewWorkspaceHealthChecker(name, workspaceID string, daemon *Daemon) *WorkspaceHealthChecker {
	return &WorkspaceHealthChecker{
		name:        name,
		workspaceID: workspaceID,
		daemon:      daemon,
		healthy:     false,
	}
}

// Name returns the name of this health check
func (w *WorkspaceHealthChecker) Name() string {
	return w.name
}

// Check performs the health check
func (w *WorkspaceHealthChecker) Check() error {
	state := w.daemon.GetWorkspaceState(w.workspaceID)
	if state == nil {
		w.healthy = false
		return fmt.Errorf("workspace %s not registered", w.workspaceID)
	}

	// Check if workspace has been active recently (within 24 hours)
	if time.Since(state.LastActive) > 24*time.Hour {
		w.healthy = false
		return fmt.Errorf("workspace %s inactive for over 24 hours", w.workspaceID)
	}

	w.healthy = true
	return nil
}

// IsHealthy returns true if the last check was healthy
func (w *WorkspaceHealthChecker) IsHealthy() bool {
	return w.healthy
}

// CompositeHealthChecker combines multiple health checkers
type CompositeHealthChecker struct {
	name     string
	checkers []HealthChecker
	healthy  bool
}

// NewCompositeHealthChecker creates a new composite health checker
func NewCompositeHealthChecker(name string, checkers ...HealthChecker) *CompositeHealthChecker {
	return &CompositeHealthChecker{
		name:     name,
		checkers: checkers,
		healthy:  false,
	}
}

// Name returns the name of this health check
func (c *CompositeHealthChecker) Name() string {
	return c.name
}

// Check performs all health checks
func (c *CompositeHealthChecker) Check() error {
	var errors []string
	allHealthy := true

	for _, checker := range c.checkers {
		if err := checker.Check(); err != nil {
			errors = append(errors, fmt.Sprintf("%s: %v", checker.Name(), err))
			allHealthy = false
		}
	}

	c.healthy = allHealthy
	if !allHealthy {
		return fmt.Errorf("health checks failed: %v", errors)
	}
	return nil
}

// IsHealthy returns true if all checks were healthy
func (c *CompositeHealthChecker) IsHealthy() bool {
	return c.healthy
}

// ═══════════════════════════════════════════════════════════════════════════════
// Helper functions for ProcessHealthChecker - Added by Agent #6
// ═══════════════════════════════════════════════════════════════════════════════

// readFile reads a file and returns its contents
func readFile(path string) ([]byte, error) {
	return os.ReadFile(path)
}

// trimSpace removes leading and trailing whitespace
func trimSpace(s string) string {
	return strings.TrimSpace(s)
}

// parseInt parses a string as an integer
func parseInt(s string) (int, error) {
	return strconv.Atoi(s)
}

// findProcess finds a process by PID
func findProcess(pid int) (*os.Process, error) {
	return os.FindProcess(pid)
}

// signalZero returns signal 0 for process existence checks
func signalZero() syscall.Signal {
	return syscall.Signal(0)
}

// ServiceHealthChecker checks the health of a specific service
// Added by Agent #6 (Services)
type ServiceHealthChecker struct {
	name        string
	workspaceID string
	serviceName string
	port        int
	daemon      *Daemon
	healthy     bool
	lastError   string
}

// NewServiceHealthChecker creates a new service health checker
func NewServiceHealthChecker(name, workspaceID, serviceName string, port int, daemon *Daemon) *ServiceHealthChecker {
	return &ServiceHealthChecker{
		name:        name,
		workspaceID: workspaceID,
		serviceName: serviceName,
		port:        port,
		daemon:      daemon,
		healthy:     false,
	}
}

// Name returns the name of this health check
func (s *ServiceHealthChecker) Name() string {
	return s.name
}

// Check performs the health check
func (s *ServiceHealthChecker) Check() error {
	// Check if port is responding
	if s.port > 0 {
		addr := fmt.Sprintf("localhost:%d", s.port)
		conn, err := net.DialTimeout("tcp", addr, 2*time.Second)
		if err != nil {
			s.healthy = false
			s.lastError = fmt.Sprintf("service port %d not responding: %v", s.port, err)
			return fmt.Errorf("service %s not healthy: %w", s.serviceName, err)
		}
		conn.Close()
	}

	s.healthy = true
	s.lastError = ""

	// Update daemon state if available
	if s.daemon != nil {
		s.daemon.SetServiceHealth(s.workspaceID, s.serviceName, true, "")
	}

	return nil
}

// IsHealthy returns true if the last check was healthy
func (s *ServiceHealthChecker) IsHealthy() bool {
	return s.healthy
}

// GetLastError returns the last error message
func (s *ServiceHealthChecker) GetLastError() string {
	return s.lastError
}
