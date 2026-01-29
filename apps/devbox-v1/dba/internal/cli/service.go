// internal/cli/service.go
package cli

import (
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"

	"github.com/dba-cli/dba/internal/service"
)

// ═══════════════════════════════════════════════════════════════════════════════
// dba up
// ═══════════════════════════════════════════════════════════════════════════════

var upCmd = &cobra.Command{
	Use:   "up [services...]",
	Short: "Start services",
	Long:  `Start services defined in process-compose.yaml`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx, err := NewCLIContext()
		if err != nil {
			return err
		}
		defer ctx.Cancel()

		if err := ctx.RequireWorkspace(); err != nil {
			return err
		}

		wait, _ := cmd.Flags().GetBool("wait")
		timeoutStr, _ := cmd.Flags().GetString("timeout")
		timeout, _ := time.ParseDuration(timeoutStr)
		if timeout == 0 {
			timeout = 60 * time.Second
		}

		mgr := service.NewManager(ctx.Workspace)
		result, err := mgr.Up(ctx.Context, args, wait, timeout)
		if err != nil {
			return err
		}

		return OutputResult(result)
	},
}

// ═══════════════════════════════════════════════════════════════════════════════
// dba down
// ═══════════════════════════════════════════════════════════════════════════════

var downCmd = &cobra.Command{
	Use:   "down [services...]",
	Short: "Stop services",
	Long:  `Stop services. If no services are specified, stops all services.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx, err := NewCLIContext()
		if err != nil {
			return err
		}
		defer ctx.Cancel()

		if err := ctx.RequireWorkspace(); err != nil {
			return err
		}

		timeoutStr, _ := cmd.Flags().GetString("timeout")
		timeout, _ := time.ParseDuration(timeoutStr)

		mgr := service.NewManager(ctx.Workspace)
		if err := mgr.Down(ctx.Context, args, timeout); err != nil {
			return err
		}

		// Build result
		result := map[string]interface{}{
			"stopped": args,
		}
		if len(args) == 0 {
			result["stopped"] = "all"
		}

		return OutputResult(result)
	},
}

// ═══════════════════════════════════════════════════════════════════════════════
// dba ps
// ═══════════════════════════════════════════════════════════════════════════════

var psCmd = &cobra.Command{
	Use:   "ps",
	Short: "List running services",
	Long:  `List all services and their current status.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx, err := NewCLIContext()
		if err != nil {
			return err
		}
		defer ctx.Cancel()

		if err := ctx.RequireWorkspace(); err != nil {
			return err
		}

		all, _ := cmd.Flags().GetBool("all")

		mgr := service.NewManager(ctx.Workspace)
		statuses, err := mgr.List(ctx.Context, all)
		if err != nil {
			return err
		}

		result := map[string]interface{}{
			"services": statuses,
		}

		return OutputResult(result)
	},
}

// ═══════════════════════════════════════════════════════════════════════════════
// dba logs
// ═══════════════════════════════════════════════════════════════════════════════

var logsCmd = &cobra.Command{
	Use:   "logs [service]",
	Short: "View service logs",
	Long:  `View logs for a specific service or all services.`,
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx, err := NewCLIContext()
		if err != nil {
			return err
		}
		defer ctx.Cancel()

		if err := ctx.RequireWorkspace(); err != nil {
			return err
		}

		svc := ""
		if len(args) > 0 {
			svc = args[0]
		}

		tail, _ := cmd.Flags().GetInt("tail")
		follow, _ := cmd.Flags().GetBool("follow")
		since, _ := cmd.Flags().GetString("since")

		mgr := service.NewManager(ctx.Workspace)

		if follow {
			// Follow mode - stream to stdout
			return mgr.FollowLogs(ctx.Context, svc, os.Stdout)
		}

		entries, err := mgr.Logs(ctx.Context, service.LogsOptions{
			Service: svc,
			Tail:    tail,
			Since:   since,
		})
		if err != nil {
			return err
		}

		if flagJSON {
			result := map[string]interface{}{
				"service": svc,
				"lines":   entries,
			}
			return OutputResult(result)
		}

		// Human-readable output
		for _, entry := range entries {
			if entry.Service != "" {
				fmt.Printf("[%s] %s\n", entry.Service, entry.Message)
			} else {
				fmt.Println(entry.Message)
			}
		}

		return nil
	},
}

// ═══════════════════════════════════════════════════════════════════════════════
// dba restart
// ═══════════════════════════════════════════════════════════════════════════════

var restartCmd = &cobra.Command{
	Use:   "restart [services...]",
	Short: "Restart services",
	Long:  `Restart services. If no services are specified, restarts all services.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx, err := NewCLIContext()
		if err != nil {
			return err
		}
		defer ctx.Cancel()

		if err := ctx.RequireWorkspace(); err != nil {
			return err
		}

		hard, _ := cmd.Flags().GetBool("hard")

		mgr := service.NewManager(ctx.Workspace)
		if err := mgr.Restart(ctx.Context, args, hard); err != nil {
			return err
		}

		// Build result
		result := map[string]interface{}{
			"restarted": args,
		}
		if len(args) == 0 {
			result["restarted"] = "all"
		}

		return OutputResult(result)
	},
}

// ═══════════════════════════════════════════════════════════════════════════════
// dba service (parent command)
// ═══════════════════════════════════════════════════════════════════════════════

var serviceCmd = &cobra.Command{
	Use:   "service",
	Short: "Manage services",
	Long:  `Manage services defined in process-compose.yaml`,
}

// ═══════════════════════════════════════════════════════════════════════════════
// dba service add
// ═══════════════════════════════════════════════════════════════════════════════

var serviceAddCmd = &cobra.Command{
	Use:   "add <name>",
	Short: "Add a new service",
	Long:  `Add a new service to the process-compose.yaml configuration.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx, err := NewCLIContext()
		if err != nil {
			return err
		}
		defer ctx.Cancel()

		if err := ctx.RequireWorkspace(); err != nil {
			return err
		}

		name := args[0]
		command, _ := cmd.Flags().GetString("command")
		template, _ := cmd.Flags().GetString("template")
		portVar, _ := cmd.Flags().GetString("port")
		dependsOn, _ := cmd.Flags().GetStringSlice("depends-on")
		readyLine, _ := cmd.Flags().GetString("ready-line")
		isDaemon, _ := cmd.Flags().GetBool("daemon")

		mgr := service.NewManager(ctx.Workspace)

		var result *service.AddResult
		if template != "" {
			// Add from template
			result, err = mgr.AddFromTemplate(ctx.Context, template, name)
		} else {
			if command == "" {
				return fmt.Errorf("--command is required when not using --template")
			}
			config := service.ServiceConfig{
				Name:         name,
				Command:      command,
				Port:         portVar,
				DependsOn:    dependsOn,
				ReadyLogLine: readyLine,
				IsDaemon:     isDaemon,
			}
			result, err = mgr.Add(ctx.Context, config)
		}

		if err != nil {
			return err
		}

		return OutputResult(result)
	},
}

// ═══════════════════════════════════════════════════════════════════════════════
// dba service rm
// ═══════════════════════════════════════════════════════════════════════════════

var serviceRmCmd = &cobra.Command{
	Use:   "rm <name>",
	Short: "Remove a service",
	Long:  `Remove a service from the process-compose.yaml configuration.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx, err := NewCLIContext()
		if err != nil {
			return err
		}
		defer ctx.Cancel()

		if err := ctx.RequireWorkspace(); err != nil {
			return err
		}

		name := args[0]
		mgr := service.NewManager(ctx.Workspace)

		if err := mgr.Remove(ctx.Context, name); err != nil {
			return err
		}

		result := map[string]interface{}{
			"name":    name,
			"removed": true,
		}

		return OutputResult(result)
	},
}

// ═══════════════════════════════════════════════════════════════════════════════
// dba service templates
// ═══════════════════════════════════════════════════════════════════════════════

var serviceTemplatesCmd = &cobra.Command{
	Use:   "templates",
	Short: "List available service templates",
	Long:  `List available service templates that can be used with 'dba service add --template'.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		templates := service.GetServiceTemplates()

		type TemplateInfo struct {
			Name        string   `json:"name"`
			Description string   `json:"description"`
			Command     string   `json:"command"`
			Port        string   `json:"port,omitempty"`
			DependsOn   []string `json:"depends_on,omitempty"`
		}

		result := struct {
			Templates []TemplateInfo `json:"templates"`
		}{
			Templates: make([]TemplateInfo, 0, len(templates)),
		}

		for name, tmpl := range templates {
			result.Templates = append(result.Templates, TemplateInfo{
				Name:        name,
				Description: tmpl.Description,
				Command:     tmpl.Command,
				Port:        tmpl.Port,
				DependsOn:   tmpl.DependsOn,
			})
		}

		return OutputResult(result)
	},
}

// ═══════════════════════════════════════════════════════════════════════════════
// dba service order
// ═══════════════════════════════════════════════════════════════════════════════

var serviceOrderCmd = &cobra.Command{
	Use:   "order",
	Short: "Show service startup order",
	Long:  `Show the order in which services will be started based on dependencies.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx, err := NewCLIContext()
		if err != nil {
			return err
		}
		defer ctx.Cancel()

		if err := ctx.RequireWorkspace(); err != nil {
			return err
		}

		mgr := service.NewManager(ctx.Workspace)
		order, err := mgr.GetDependencyOrder(ctx.Context)
		if err != nil {
			return err
		}

		result := map[string]interface{}{
			"order": order,
		}

		return OutputResult(result)
	},
}

// ═══════════════════════════════════════════════════════════════════════════════
// dba service list - Added by Agent #6 to fix B04
// ═══════════════════════════════════════════════════════════════════════════════

var serviceListCmd = &cobra.Command{
	Use:   "list",
	Short: "List services and their status",
	Long:  `List all services defined in process-compose.yaml and their current status.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx, err := NewCLIContext()
		if err != nil {
			return err
		}
		defer ctx.Cancel()

		if err := ctx.RequireWorkspace(); err != nil {
			return err
		}

		all, _ := cmd.Flags().GetBool("all")

		mgr := service.NewManager(ctx.Workspace)
		statuses, err := mgr.List(ctx.Context, all)
		if err != nil {
			return err
		}

		result := map[string]interface{}{
			"services": statuses,
		}

		return OutputResult(result)
	},
}

func init() {
	// up flags
	upCmd.Flags().Bool("wait", true, "Wait for services to be healthy")
	upCmd.Flags().String("timeout", "60s", "Health check timeout")

	// down flags
	downCmd.Flags().String("timeout", "30s", "Shutdown timeout")

	// ps flags
	psCmd.Flags().BoolP("all", "a", false, "Include stopped services")

	// logs flags
	logsCmd.Flags().IntP("tail", "n", 100, "Number of lines")
	logsCmd.Flags().BoolP("follow", "f", false, "Follow log output")
	logsCmd.Flags().String("since", "", "Show logs since timestamp")

	// restart flags
	restartCmd.Flags().Bool("hard", false, "Kill and restart")

	// service add flags
	serviceAddCmd.Flags().StringP("command", "c", "", "Command to run")
	serviceAddCmd.Flags().StringP("template", "t", "", "Use a predefined template")
	serviceAddCmd.Flags().StringP("port", "p", "", "Port environment variable name")
	serviceAddCmd.Flags().StringSlice("depends-on", nil, "Services this service depends on")
	serviceAddCmd.Flags().String("ready-line", "", "Log line indicating service is ready")
	serviceAddCmd.Flags().Bool("daemon", false, "Run as daemon process")

	// service list flags
	serviceListCmd.Flags().BoolP("all", "a", false, "Include stopped services")

	// Add subcommands to service
	serviceCmd.AddCommand(serviceAddCmd)
	serviceCmd.AddCommand(serviceRmCmd)
	serviceCmd.AddCommand(serviceTemplatesCmd)
	serviceCmd.AddCommand(serviceOrderCmd)
	serviceCmd.AddCommand(serviceListCmd) // Added by Agent #6 to fix B04

	// Register commands with root
	rootCmd.AddCommand(upCmd)
	rootCmd.AddCommand(downCmd)
	rootCmd.AddCommand(psCmd)
	rootCmd.AddCommand(logsCmd)
	rootCmd.AddCommand(restartCmd)
	rootCmd.AddCommand(serviceCmd)
}
