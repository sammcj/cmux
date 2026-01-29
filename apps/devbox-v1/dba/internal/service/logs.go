// internal/service/logs.go
package service

import (
	"context"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"time"
)

// LogEntry represents a single log entry
type LogEntry struct {
	Timestamp string `json:"timestamp"`
	Service   string `json:"service"`
	Message   string `json:"message"`
}

// LogsOptions are options for getting logs
type LogsOptions struct {
	Service string
	Tail    int
	Since   string
	Follow  bool
}

// Logs retrieves logs for services
func (m *Manager) Logs(ctx context.Context, opts LogsOptions) ([]LogEntry, error) {
	if m.workspace == nil {
		return nil, fmt.Errorf("workspace is nil")
	}
	args := []string{"run", "--", "process-compose", "process", "logs"}

	if opts.Service != "" {
		args = append(args, opts.Service)
	}

	if opts.Tail > 0 {
		args = append(args, "--tail", fmt.Sprintf("%d", opts.Tail))
	}

	cmd := exec.CommandContext(ctx, "devbox", args...)
	cmd.Dir = m.workspace.Path
	cmd.Env = m.buildEnv()

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get logs: %w", err)
	}

	// Parse log output
	// Format depends on process-compose version
	// Simple parsing: each line is a log entry
	lines := strings.Split(string(output), "\n")
	entries := make([]LogEntry, 0, len(lines))

	for _, line := range lines {
		if line == "" {
			continue
		}

		entry := LogEntry{
			Timestamp: time.Now().Format(time.RFC3339),
			Message:   line,
		}

		// Try to extract service name from line
		// Format often: "service_name | message"
		if parts := strings.SplitN(line, "|", 2); len(parts) == 2 {
			entry.Service = strings.TrimSpace(parts[0])
			entry.Message = strings.TrimSpace(parts[1])
		}

		entries = append(entries, entry)
	}

	return entries, nil
}

// FollowLogs streams logs to a writer
func (m *Manager) FollowLogs(ctx context.Context, service string, w io.Writer) error {
	if m.workspace == nil {
		return fmt.Errorf("workspace is nil")
	}
	args := []string{"run", "--", "process-compose", "process", "logs", "-f"}

	if service != "" {
		args = append(args, service)
	}

	cmd := exec.CommandContext(ctx, "devbox", args...)
	cmd.Dir = m.workspace.Path
	cmd.Env = m.buildEnv()
	cmd.Stdout = w
	cmd.Stderr = w

	return cmd.Run()
}

// GetLogPath returns the path to the logs directory
func (m *Manager) GetLogPath() string {
	if m.workspace == nil {
		return ""
	}
	return m.workspace.Path + "/.dba/logs"
}

// LogsResult represents the result of a logs query
type LogsResult struct {
	Service   string     `json:"service,omitempty"`
	Lines     []LogEntry `json:"lines"`
	Count     int        `json:"count"`
	Truncated bool       `json:"truncated,omitempty"`
}

// GetLogs is a convenience method that returns a LogsResult
func (m *Manager) GetLogs(ctx context.Context, service string, tail int) (*LogsResult, error) {
	entries, err := m.Logs(ctx, LogsOptions{
		Service: service,
		Tail:    tail,
	})
	if err != nil {
		return nil, err
	}

	return &LogsResult{
		Service: service,
		Lines:   entries,
		Count:   len(entries),
	}, nil
}

// TailLogs retrieves the last N lines of logs
func (m *Manager) TailLogs(ctx context.Context, service string, n int) ([]LogEntry, error) {
	return m.Logs(ctx, LogsOptions{
		Service: service,
		Tail:    n,
	})
}

// GetServiceLogs returns logs for a specific service
func (m *Manager) GetServiceLogs(ctx context.Context, service string, opts LogsOptions) ([]LogEntry, error) {
	opts.Service = service
	return m.Logs(ctx, opts)
}

// StreamLogs streams logs for a service to a callback function
func (m *Manager) StreamLogs(ctx context.Context, service string, callback func(LogEntry)) error {
	if m.workspace == nil {
		return fmt.Errorf("workspace is nil")
	}
	r, w := io.Pipe()

	go func() {
		defer w.Close()
		m.FollowLogs(ctx, service, w)
	}()

	buf := make([]byte, 4096)
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			n, err := r.Read(buf)
			if err != nil {
				if err == io.EOF {
					return nil
				}
				return err
			}

			line := strings.TrimSpace(string(buf[:n]))
			if line != "" {
				entry := LogEntry{
					Timestamp: time.Now().Format(time.RFC3339),
					Message:   line,
				}

				// Try to extract service name
				if parts := strings.SplitN(line, "|", 2); len(parts) == 2 {
					entry.Service = strings.TrimSpace(parts[0])
					entry.Message = strings.TrimSpace(parts[1])
				}

				callback(entry)
			}
		}
	}
}
