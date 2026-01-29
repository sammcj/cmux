// internal/cli/code.go
package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/dba-cli/dba/internal/lsp"
)

var codeCmd = &cobra.Command{
	Use:   "code",
	Short: "Code intelligence commands",
	Long: `Code intelligence commands for working with code.

Provides LSP-like functionality including:
- Diagnostics (errors and warnings)
- Code actions (quick fixes)
- Symbol search
- Go to definition
- Find references
- Rename symbol
- Format code
- VS Code URL generation`,
}

// ═══════════════════════════════════════════════════════════════════════════════
// dba code diagnostics
// ═══════════════════════════════════════════════════════════════════════════════

var codeDiagnosticsCmd = &cobra.Command{
	Use:   "diagnostics",
	Short: "Get code errors and warnings",
	Long: `Get diagnostics (errors, warnings, etc.) from TypeScript and ESLint.

Examples:
  dba code diagnostics
  dba code diagnostics --file=src/app.tsx
  dba code diagnostics --severity=error
  dba code diagnostics --source=typescript`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx, err := NewCLIContext()
		if err != nil {
			return err
		}
		defer ctx.Cancel()

		if err := ctx.RequireWorkspace(); err != nil {
			return err
		}

		file, _ := cmd.Flags().GetString("file")
		severity, _ := cmd.Flags().GetString("severity")
		source, _ := cmd.Flags().GetString("source")

		// Validate severity
		if !lsp.ValidSeverity(severity) {
			return fmt.Errorf("invalid severity: %s (valid: error, warning, info, hint)", severity)
		}

		// Validate source
		if !lsp.ValidSource(source) {
			return fmt.Errorf("invalid source: %s (valid: typescript, eslint)", source)
		}

		result, err := lsp.GetDiagnostics(ctx.Context, ctx.Workspace.ProjectPath, lsp.DiagnosticsOptions{
			File:     file,
			Severity: severity,
			Source:   source,
		})
		if err != nil {
			return err
		}

		return OutputResult(result)
	},
}

// ═══════════════════════════════════════════════════════════════════════════════
// dba code action
// ═══════════════════════════════════════════════════════════════════════════════

var codeActionCmd = &cobra.Command{
	Use:   "action <file:line:col>",
	Short: "Get or apply code actions",
	Long: `Get available code actions (quick fixes, refactorings) at a location.

Examples:
  dba code action src/app.tsx:42:10
  dba code action src/app.tsx:42:10 --apply --index=0`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx, err := NewCLIContext()
		if err != nil {
			return err
		}
		defer ctx.Cancel()

		if err := ctx.RequireWorkspace(); err != nil {
			return err
		}

		file, line, col, err := lsp.ParseLocation(args[0])
		if err != nil {
			return err
		}

		apply, _ := cmd.Flags().GetBool("apply")
		index, _ := cmd.Flags().GetInt("index")

		if apply {
			result, err := lsp.ApplyCodeAction(ctx.Context, ctx.Workspace.ProjectPath, file, line, col, index)
			if err != nil {
				return err
			}
			return OutputResult(result)
		}

		result, err := lsp.GetCodeActions(ctx.Context, ctx.Workspace.ProjectPath, file, line, col)
		if err != nil {
			return err
		}

		return OutputResult(result)
	},
}

// ═══════════════════════════════════════════════════════════════════════════════
// dba code search
// ═══════════════════════════════════════════════════════════════════════════════

var codeSearchCmd = &cobra.Command{
	Use:   "search <query>",
	Short: "Search code symbols",
	Long: `Search for symbols or text in the codebase.

Examples:
  dba code search "handleSubmit"
  dba code search "useState" --symbols`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx, err := NewCLIContext()
		if err != nil {
			return err
		}
		defer ctx.Cancel()

		if err := ctx.RequireWorkspace(); err != nil {
			return err
		}

		symbolsOnly, _ := cmd.Flags().GetBool("symbols")

		result, err := lsp.SearchSymbols(ctx.Context, ctx.Workspace.ProjectPath, args[0], symbolsOnly)
		if err != nil {
			return err
		}

		return OutputResult(result)
	},
}

// ═══════════════════════════════════════════════════════════════════════════════
// dba code definition
// ═══════════════════════════════════════════════════════════════════════════════

var codeDefinitionCmd = &cobra.Command{
	Use:   "definition <file:line:col>",
	Short: "Go to definition",
	Long: `Find the definition of a symbol at the given location.

Examples:
  dba code definition src/app.tsx:15:10`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx, err := NewCLIContext()
		if err != nil {
			return err
		}
		defer ctx.Cancel()

		if err := ctx.RequireWorkspace(); err != nil {
			return err
		}

		file, line, col, err := lsp.ParseLocation(args[0])
		if err != nil {
			return err
		}

		result, err := lsp.GetDefinition(ctx.Context, ctx.Workspace.ProjectPath, file, line, col)
		if err != nil {
			return err
		}

		return OutputResult(result)
	},
}

// ═══════════════════════════════════════════════════════════════════════════════
// dba code references
// ═══════════════════════════════════════════════════════════════════════════════

var codeReferencesCmd = &cobra.Command{
	Use:   "references <file:line:col>",
	Short: "Find all references",
	Long: `Find all references to a symbol at the given location.

Examples:
  dba code references src/types.ts:10:15
  dba code references src/types.ts:10:15 --include-declaration`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx, err := NewCLIContext()
		if err != nil {
			return err
		}
		defer ctx.Cancel()

		if err := ctx.RequireWorkspace(); err != nil {
			return err
		}

		file, line, col, err := lsp.ParseLocation(args[0])
		if err != nil {
			return err
		}

		includeDecl, _ := cmd.Flags().GetBool("include-declaration")

		result, err := lsp.GetReferences(ctx.Context, ctx.Workspace.ProjectPath, file, line, col, includeDecl)
		if err != nil {
			return err
		}

		return OutputResult(result)
	},
}

// ═══════════════════════════════════════════════════════════════════════════════
// dba code rename
// ═══════════════════════════════════════════════════════════════════════════════

var codeRenameCmd = &cobra.Command{
	Use:   "rename <file:line:col> <new-name>",
	Short: "Rename symbol",
	Long: `Rename a symbol across the codebase.

Examples:
  dba code rename src/types.ts:10:15 UserProfile --dry-run
  dba code rename src/types.ts:10:15 UserProfile`,
	Args: cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx, err := NewCLIContext()
		if err != nil {
			return err
		}
		defer ctx.Cancel()

		if err := ctx.RequireWorkspace(); err != nil {
			return err
		}

		file, line, col, err := lsp.ParseLocation(args[0])
		if err != nil {
			return err
		}

		newName := args[1]
		dryRun, _ := cmd.Flags().GetBool("dry-run")

		result, err := lsp.RenameSymbol(ctx.Context, ctx.Workspace.ProjectPath, file, line, col, newName, dryRun)
		if err != nil {
			return err
		}

		return OutputResult(result)
	},
}

// ═══════════════════════════════════════════════════════════════════════════════
// dba code format
// ═══════════════════════════════════════════════════════════════════════════════

var codeFormatCmd = &cobra.Command{
	Use:   "format [file...]",
	Short: "Format code",
	Long: `Format code using the appropriate formatter (prettier, gofmt, black, etc.).

Examples:
  dba code format
  dba code format src/app.tsx
  dba code format --check`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx, err := NewCLIContext()
		if err != nil {
			return err
		}
		defer ctx.Cancel()

		if err := ctx.RequireWorkspace(); err != nil {
			return err
		}

		check, _ := cmd.Flags().GetBool("check")

		files := args
		if len(files) == 0 {
			files = []string{"."}
		}

		result, err := lsp.FormatFile(ctx.Context, ctx.Workspace.ProjectPath, files, check)
		if err != nil {
			return err
		}

		return OutputResult(result)
	},
}

// ═══════════════════════════════════════════════════════════════════════════════
// dba code hover
// ═══════════════════════════════════════════════════════════════════════════════

var codeHoverCmd = &cobra.Command{
	Use:   "hover <file:line:col>",
	Short: "Get hover information",
	Long: `Get hover information for a symbol at the given location.

Examples:
  dba code hover src/app.tsx:20:10`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx, err := NewCLIContext()
		if err != nil {
			return err
		}
		defer ctx.Cancel()

		if err := ctx.RequireWorkspace(); err != nil {
			return err
		}

		file, line, col, err := lsp.ParseLocation(args[0])
		if err != nil {
			return err
		}

		result, err := lsp.GetHover(ctx.Context, ctx.Workspace.ProjectPath, file, line, col)
		if err != nil {
			return err
		}

		return OutputResult(result)
	},
}

// ═══════════════════════════════════════════════════════════════════════════════
// dba code url
// ═══════════════════════════════════════════════════════════════════════════════

var codeURLCmd = &cobra.Command{
	Use:   "url",
	Short: "Get VS Code URL",
	Long: `Get the URL to open VS Code (code-server) for the workspace.

Examples:
  dba code url
  dba code url --file=src/app.tsx --line=42`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx, err := NewCLIContext()
		if err != nil {
			return err
		}
		defer ctx.Cancel()

		if err := ctx.RequireWorkspace(); err != nil {
			return err
		}

		file, _ := cmd.Flags().GetString("file")
		line, _ := cmd.Flags().GetInt("line")

		codePort, ok := ctx.Workspace.Ports["CODE_PORT"]
		if !ok {
			return fmt.Errorf("CODE_PORT not allocated for this workspace")
		}

		result := lsp.GenerateVSCodeURL(codePort, ctx.Workspace.ProjectPath, file, line)

		return OutputResult(result)
	},
}

func init() {
	// diagnostics flags
	codeDiagnosticsCmd.Flags().String("file", "", "Filter by file")
	codeDiagnosticsCmd.Flags().String("severity", "", "Filter by severity (error, warning, info, hint)")
	codeDiagnosticsCmd.Flags().String("source", "", "Filter by source (typescript, eslint)")

	// action flags
	codeActionCmd.Flags().Bool("apply", false, "Apply the action")
	codeActionCmd.Flags().Int("index", 0, "Action index to apply")

	// search flags
	codeSearchCmd.Flags().Bool("symbols", false, "Search symbol definitions only")

	// references flags
	codeReferencesCmd.Flags().Bool("include-declaration", false, "Include declaration in results")

	// rename flags
	codeRenameCmd.Flags().Bool("dry-run", false, "Preview changes without applying")

	// format flags
	codeFormatCmd.Flags().Bool("check", false, "Check only, don't modify files")

	// url flags
	codeURLCmd.Flags().String("file", "", "File to open")
	codeURLCmd.Flags().Int("line", 0, "Line number")

	// Add subcommands
	codeCmd.AddCommand(codeDiagnosticsCmd)
	codeCmd.AddCommand(codeActionCmd)
	codeCmd.AddCommand(codeSearchCmd)
	codeCmd.AddCommand(codeDefinitionCmd)
	codeCmd.AddCommand(codeReferencesCmd)
	codeCmd.AddCommand(codeRenameCmd)
	codeCmd.AddCommand(codeFormatCmd)
	codeCmd.AddCommand(codeHoverCmd)
	codeCmd.AddCommand(codeURLCmd)

	// Add code command to root
	AddCommand(codeCmd)
}
