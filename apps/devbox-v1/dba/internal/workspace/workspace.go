// internal/workspace/workspace.go
package workspace

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"
)

// Workspace represents a DBA workspace
type Workspace struct {
	ID          string         `json:"id"`
	Name        string         `json:"name"`
	Path        string         `json:"path"`
	ProjectPath string         `json:"project_path"`
	Template    string         `json:"template"`
	Status      string         `json:"status"`
	BasePort    int            `json:"base_port"`
	Ports       map[string]int `json:"ports"`
	Packages    []string       `json:"packages"`
	CreatedAt   time.Time      `json:"created_at"`
	LastActive  time.Time      `json:"last_active"`

	Git *GitInfo `json:"git,omitempty"`

	// Morph holds the Morph VM state
	Morph MorphState `json:"morph,omitempty"`
}

// GitInfo contains git repository information
type GitInfo struct {
	Remote string `json:"remote"`
	Branch string `json:"branch"`
	Commit string `json:"commit"`
}

// State is the persisted workspace state
type State struct {
	ID         string         `json:"id"`
	Name       string         `json:"name"`
	Template   string         `json:"template"`
	Status     string         `json:"status"`
	BasePort   int            `json:"base_port"`
	Ports      map[string]int `json:"ports"`
	Packages   []string       `json:"packages"`
	CreatedAt  time.Time      `json:"created_at"`
	LastActive time.Time      `json:"last_active"`
	Git        *GitInfo       `json:"git,omitempty"`

	// Runtime state (not persisted in state.json, populated on load)
	Services map[string]ServiceState `json:"services,omitempty"`
	Computer *ComputerState          `json:"computer,omitempty"`

	// Morph holds the Morph VM state
	Morph MorphState `json:"morph,omitempty"`
}

// ServiceState holds runtime state for a service
type ServiceState struct {
	Status  string `json:"status"`
	PID     int    `json:"pid,omitempty"`
	Port    int    `json:"port,omitempty"`
	Healthy bool   `json:"healthy,omitempty"`
}

// ComputerState holds runtime state for computer use container
type ComputerState struct {
	Status      string `json:"status"`
	ContainerID string `json:"container_id,omitempty"`
}

// MorphState holds the Morph VM state for a workspace
type MorphState struct {
	// InstanceID is the current running Morph instance ID
	InstanceID string `json:"instance_id,omitempty"`

	// SnapshotID is the snapshot this instance was started from
	SnapshotID string `json:"snapshot_id,omitempty"`

	// Status is the current instance status
	Status string `json:"status,omitempty"` // "running", "stopped", "paused"

	// BaseURL is the exposed HTTP URL
	BaseURL string `json:"base_url,omitempty"`

	// URLs for specific services
	CodeURL string `json:"code_url,omitempty"`
	VNCURL  string `json:"vnc_url,omitempty"`
	AppURL  string `json:"app_url,omitempty"`
	CDPURL  string `json:"cdp_url,omitempty"`

	// CDPPort is the local port forwarded to CDP (for agent-browser)
	CDPPort int `json:"cdp_port,omitempty"`

	// StartedAt is when the instance was started
	StartedAt time.Time `json:"started_at,omitempty"`

	// SavedSnapshots is a list of user-saved snapshots for this workspace
	SavedSnapshots []SavedSnapshot `json:"saved_snapshots,omitempty"`

	// ActivePorts tracks ports exposed by containers/services in the VM
	ActivePorts []ActivePort `json:"active_ports,omitempty"`
}

// ActivePort represents a port exposed by a container or service in the Morph VM
type ActivePort struct {
	// Port is the port number
	Port int `json:"port"`

	// Protocol is the protocol (tcp, udp)
	Protocol string `json:"protocol,omitempty"`

	// Service is the service name (e.g., "vite", "postgres", "nginx")
	Service string `json:"service,omitempty"`

	// Container is the container name or ID that exposes this port
	Container string `json:"container,omitempty"`

	// LocalPort is the forwarded local port (if port forwarding is active)
	LocalPort int `json:"local_port,omitempty"`

	// URL is the accessible URL for this port (if HTTP)
	URL string `json:"url,omitempty"`

	// IsHTTP indicates if this port serves HTTP traffic
	IsHTTP bool `json:"is_http,omitempty"`

	// DiscoveredAt is when this port was discovered
	DiscoveredAt time.Time `json:"discovered_at,omitempty"`
}

// SavedSnapshot represents a user-saved workspace snapshot
type SavedSnapshot struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
}

// URLs returns the URLs for this workspace
func (w *Workspace) URLs() map[string]string {
	urls := make(map[string]string)

	if port, ok := w.Ports["PORT"]; ok {
		urls["app"] = fmt.Sprintf("http://localhost:%d", port)
	}
	if port, ok := w.Ports["CODE_PORT"]; ok {
		urls["code"] = fmt.Sprintf("http://localhost:%d", port)
	}
	if port, ok := w.Ports["VNC_PORT"]; ok {
		urls["vnc"] = fmt.Sprintf("vnc://localhost:%d", port)
	}

	return urls
}

// StateDir returns the .dba directory path
func (w *Workspace) StateDir() string {
	return filepath.Join(w.Path, ".dba")
}

// StatePath returns the state.json path
func (w *Workspace) StatePath() string {
	return filepath.Join(w.StateDir(), "state.json")
}

// DevboxPath returns the devbox.json path
func (w *Workspace) DevboxPath() string {
	return filepath.Join(w.Path, "devbox.json")
}

// ProcessComposePath returns the process-compose.yaml path
func (w *Workspace) ProcessComposePath() string {
	return filepath.Join(w.Path, "process-compose.yaml")
}

// LogsDir returns the logs directory path
func (w *Workspace) LogsDir() string {
	return filepath.Join(w.StateDir(), "logs")
}

// TextOutput returns human-readable output for the workspace
func (w *Workspace) TextOutput() string {
	output := fmt.Sprintf("Workspace: %s (%s)\n", w.Name, w.ID)
	output += fmt.Sprintf("  Path:     %s\n", w.Path)
	output += fmt.Sprintf("  Template: %s\n", w.Template)
	output += fmt.Sprintf("  Status:   %s\n", w.Status)
	output += fmt.Sprintf("  Ports:\n")
	for name, port := range w.Ports {
		output += fmt.Sprintf("    %s: %d\n", name, port)
	}
	if urls := w.URLs(); len(urls) > 0 {
		output += fmt.Sprintf("  URLs:\n")
		for name, url := range urls {
			output += fmt.Sprintf("    %s: %s\n", name, url)
		}
	}
	// Add Morph state if present
	if w.Morph.InstanceID != "" {
		output += fmt.Sprintf("  Morph:\n")
		output += fmt.Sprintf("    Instance: %s\n", w.Morph.InstanceID)
		output += fmt.Sprintf("    Status:   %s\n", w.Morph.Status)
		if w.Morph.CodeURL != "" {
			output += fmt.Sprintf("    Code URL: %s\n", w.Morph.CodeURL)
		}
		if w.Morph.VNCURL != "" {
			output += fmt.Sprintf("    VNC URL:  %s\n", w.Morph.VNCURL)
		}
		if w.Morph.AppURL != "" {
			output += fmt.Sprintf("    App URL:  %s\n", w.Morph.AppURL)
		}
	}
	return output
}

// IsMorphRunning returns whether the Morph VM is running
func (w *Workspace) IsMorphRunning() bool {
	// Check for valid status and non-empty/non-whitespace instance ID
	if w.Morph.Status != "running" {
		return false
	}
	// Trim whitespace and check if instance ID is meaningful
	trimmedID := w.Morph.InstanceID
	for len(trimmedID) > 0 && (trimmedID[0] == ' ' || trimmedID[0] == '\t' || trimmedID[0] == '\n' || trimmedID[0] == '\r') {
		trimmedID = trimmedID[1:]
	}
	for len(trimmedID) > 0 && (trimmedID[len(trimmedID)-1] == ' ' || trimmedID[len(trimmedID)-1] == '\t' || trimmedID[len(trimmedID)-1] == '\n' || trimmedID[len(trimmedID)-1] == '\r') {
		trimmedID = trimmedID[:len(trimmedID)-1]
	}
	return trimmedID != ""
}

// SetMorphInstance updates the Morph instance info
func (w *Workspace) SetMorphInstance(instanceID, snapshotID, baseURL string) {
	w.Morph.InstanceID = instanceID
	w.Morph.SnapshotID = snapshotID
	w.Morph.BaseURL = baseURL
	w.Morph.Status = "running"
	w.Morph.StartedAt = time.Now()

	// Derive URLs from base URL (trim trailing slashes to avoid double slashes)
	if baseURL != "" {
		baseURL = strings.TrimSuffix(baseURL, "/")
		w.Morph.CodeURL = baseURL + "/code/"
		w.Morph.VNCURL = baseURL + "/vnc/vnc.html"
		w.Morph.AppURL = baseURL + "/vnc/app/"
		// CDP requires WebSocket protocol
		cdpURL := strings.Replace(baseURL, "https://", "wss://", 1)
		cdpURL = strings.Replace(cdpURL, "http://", "ws://", 1)
		w.Morph.CDPURL = cdpURL + "/cdp/"
	}
}

// ClearMorphInstance clears the Morph instance info (when stopped)
func (w *Workspace) ClearMorphInstance() {
	w.Morph.InstanceID = ""
	w.Morph.Status = "stopped"
	// Keep BaseURL and other URLs for reference
}

// AddSavedSnapshot adds a saved snapshot to the list
func (w *Workspace) AddSavedSnapshot(id, name string) {
	w.Morph.SavedSnapshots = append(w.Morph.SavedSnapshots, SavedSnapshot{
		ID:        id,
		Name:      name,
		CreatedAt: time.Now(),
	})
}

// GetSavedSnapshot finds a saved snapshot by name
func (w *Workspace) GetSavedSnapshot(name string) *SavedSnapshot {
	for i := range w.Morph.SavedSnapshots {
		if w.Morph.SavedSnapshots[i].Name == name {
			return &w.Morph.SavedSnapshots[i]
		}
	}
	return nil
}

// GetMorphURLs returns all Morph URLs for this workspace
func (w *Workspace) GetMorphURLs() map[string]string {
	urls := make(map[string]string)
	if w.Morph.CodeURL != "" {
		urls["code"] = w.Morph.CodeURL
	}
	if w.Morph.VNCURL != "" {
		urls["vnc"] = w.Morph.VNCURL
	}
	if w.Morph.AppURL != "" {
		urls["app"] = w.Morph.AppURL
	}
	if w.Morph.CDPURL != "" {
		urls["cdp"] = w.Morph.CDPURL
	}
	return urls
}

// GetMorphState returns the instance ID, base URL, and status for debugging
func (w *Workspace) GetMorphState() (instanceID, baseURL, status string) {
	return w.Morph.InstanceID, w.Morph.BaseURL, w.Morph.Status
}

// AddActivePort adds or updates an active port in the Morph state
func (w *Workspace) AddActivePort(port ActivePort) {
	if port.DiscoveredAt.IsZero() {
		port.DiscoveredAt = time.Now()
	}

	// Update existing port if found
	for i := range w.Morph.ActivePorts {
		if w.Morph.ActivePorts[i].Port == port.Port {
			w.Morph.ActivePorts[i] = port
			return
		}
	}

	// Add new port
	w.Morph.ActivePorts = append(w.Morph.ActivePorts, port)
}

// RemoveActivePort removes an active port by port number
func (w *Workspace) RemoveActivePort(port int) {
	for i := range w.Morph.ActivePorts {
		if w.Morph.ActivePorts[i].Port == port {
			// Remove by shifting elements
			w.Morph.ActivePorts = append(w.Morph.ActivePorts[:i], w.Morph.ActivePorts[i+1:]...)
			return
		}
	}
}

// GetActivePort finds an active port by port number
func (w *Workspace) GetActivePort(port int) *ActivePort {
	for i := range w.Morph.ActivePorts {
		if w.Morph.ActivePorts[i].Port == port {
			return &w.Morph.ActivePorts[i]
		}
	}
	return nil
}

// GetHTTPPorts returns all active ports that serve HTTP traffic
func (w *Workspace) GetHTTPPorts() []ActivePort {
	var httpPorts []ActivePort
	for _, p := range w.Morph.ActivePorts {
		if p.IsHTTP {
			httpPorts = append(httpPorts, p)
		}
	}
	return httpPorts
}

// GetPrimaryAppPort returns the primary app port (first HTTP port, or first port if none are HTTP)
func (w *Workspace) GetPrimaryAppPort() *ActivePort {
	// First try to find an HTTP port
	for i := range w.Morph.ActivePorts {
		if w.Morph.ActivePorts[i].IsHTTP {
			return &w.Morph.ActivePorts[i]
		}
	}

	// Fall back to first port
	if len(w.Morph.ActivePorts) > 0 {
		return &w.Morph.ActivePorts[0]
	}

	return nil
}

// ClearActivePorts removes all active ports
func (w *Workspace) ClearActivePorts() {
	w.Morph.ActivePorts = nil
}

// SetActivePorts replaces all active ports with the given list
func (w *Workspace) SetActivePorts(ports []ActivePort) {
	now := time.Now()
	for i := range ports {
		if ports[i].DiscoveredAt.IsZero() {
			ports[i].DiscoveredAt = now
		}
	}
	w.Morph.ActivePorts = ports
}
