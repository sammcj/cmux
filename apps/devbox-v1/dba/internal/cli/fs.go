// internal/cli/fs.go
package cli

import (
	"io"
	"os"

	"github.com/spf13/cobra"

	"github.com/dba-cli/dba/internal/fs"
)

var fsCmd = &cobra.Command{
	Use:   "fs",
	Short: "File system operations",
	Long: `File system operations for reading, writing, and manipulating files.

Commands:
  read    - Read file contents
  write   - Write file contents
  list    - List directory contents
  search  - Search file contents
  cp      - Copy file or directory
  mv      - Move file or directory
  rm      - Remove file or directory
  mkdir   - Create directory`,
}

// ═══════════════════════════════════════════════════════════════════════════════
// dba fs read
// ═══════════════════════════════════════════════════════════════════════════════

var fsReadCmd = &cobra.Command{
	Use:   "read <path>",
	Short: "Read file contents",
	Long: `Read the contents of a file.

Examples:
  dba fs read src/app.tsx
  dba fs read src/app.tsx --lines=10:20
  dba fs read image.png --base64`,
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

		base64, _ := cmd.Flags().GetBool("base64")
		lines, _ := cmd.Flags().GetString("lines")
		followSymlinks, _ := cmd.Flags().GetBool("follow-symlinks")

		result, err := fs.Read(ctx.Workspace.ProjectPath, args[0], fs.ReadOptions{
			Base64:         base64,
			LineRange:      lines,
			FollowSymlinks: followSymlinks,
		})
		if err != nil {
			return err
		}

		return OutputResult(result)
	},
}

// ═══════════════════════════════════════════════════════════════════════════════
// dba fs write
// ═══════════════════════════════════════════════════════════════════════════════

var fsWriteCmd = &cobra.Command{
	Use:   "write <path>",
	Short: "Write file contents",
	Long: `Write content to a file. Content can be provided via --content flag or stdin.

Examples:
  dba fs write test.txt --content="Hello, World!"
  echo "Hello" | dba fs write test.txt
  dba fs write src/new/file.txt --content="..." --mkdir`,
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

		content, _ := cmd.Flags().GetString("content")
		base64Flag, _ := cmd.Flags().GetBool("base64")
		appendFlag, _ := cmd.Flags().GetBool("append")
		mkdirFlag, _ := cmd.Flags().GetBool("mkdir")

		// If no content flag, read from stdin
		if content == "" {
			data, err := io.ReadAll(os.Stdin)
			if err != nil {
				return err
			}
			content = string(data)
		}

		result, err := fs.Write(ctx.Workspace.ProjectPath, args[0], fs.WriteOptions{
			Content: content,
			Base64:  base64Flag,
			Append:  appendFlag,
			MkdirP:  mkdirFlag,
		})
		if err != nil {
			return err
		}

		return OutputResult(result)
	},
}

// ═══════════════════════════════════════════════════════════════════════════════
// dba fs list
// ═══════════════════════════════════════════════════════════════════════════════

var fsListCmd = &cobra.Command{
	Use:   "list [path]",
	Short: "List directory contents",
	Long: `List files and directories.

Examples:
  dba fs list
  dba fs list src
  dba fs list --recursive
  dba fs list --pattern="*.tsx"`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx, err := NewCLIContext()
		if err != nil {
			return err
		}
		defer ctx.Cancel()

		if err := ctx.RequireWorkspace(); err != nil {
			return err
		}

		path := ""
		if len(args) > 0 {
			path = args[0]
		}

		recursive, _ := cmd.Flags().GetBool("recursive")
		hidden, _ := cmd.Flags().GetBool("hidden")
		pattern, _ := cmd.Flags().GetString("pattern")
		maxDepth, _ := cmd.Flags().GetInt("max-depth")

		result, err := fs.List(ctx.Workspace.ProjectPath, path, fs.ListOptions{
			Recursive: recursive,
			Hidden:    hidden,
			Pattern:   pattern,
			GitIgnore: true,
			MaxDepth:  maxDepth,
		})
		if err != nil {
			return err
		}

		return OutputResult(result)
	},
}

// ═══════════════════════════════════════════════════════════════════════════════
// dba fs search
// ═══════════════════════════════════════════════════════════════════════════════

var fsSearchCmd = &cobra.Command{
	Use:   "search <query>",
	Short: "Search file contents",
	Long: `Search for a pattern in files using ripgrep.

Examples:
  dba fs search "TODO"
  dba fs search "function" --pattern="*.ts"
  dba fs search "error" --regex --context=2`,
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

		pattern, _ := cmd.Flags().GetString("pattern")
		regex, _ := cmd.Flags().GetBool("regex")
		caseSensitive, _ := cmd.Flags().GetBool("case-sensitive")
		context, _ := cmd.Flags().GetInt("context")
		maxResults, _ := cmd.Flags().GetInt("max-results")
		noIgnore, _ := cmd.Flags().GetBool("no-ignore")
		followSymlinks, _ := cmd.Flags().GetBool("follow-symlinks")

		result, err := fs.Search(ctx.Workspace.ProjectPath, args[0], fs.SearchOptions{
			Pattern:        pattern,
			Regex:          regex,
			CaseSensitive:  caseSensitive,
			Context:        context,
			MaxResults:     maxResults,
			NoIgnore:       noIgnore,
			FollowSymlinks: followSymlinks,
		})
		if err != nil {
			return err
		}

		return OutputResult(result)
	},
}

// ═══════════════════════════════════════════════════════════════════════════════
// dba fs cp
// ═══════════════════════════════════════════════════════════════════════════════

var fsCpCmd = &cobra.Command{
	Use:   "cp <src> <dst>",
	Short: "Copy file or directory",
	Long: `Copy a file or directory.

Examples:
  dba fs cp src/app.tsx src/app.backup.tsx
  dba fs cp src/ backup/src/ --recursive`,
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

		recursive, _ := cmd.Flags().GetBool("recursive")

		result, err := fs.Copy(ctx.Workspace.ProjectPath, args[0], args[1], recursive)
		if err != nil {
			return err
		}

		return OutputResult(result)
	},
}

// ═══════════════════════════════════════════════════════════════════════════════
// dba fs mv
// ═══════════════════════════════════════════════════════════════════════════════

var fsMvCmd = &cobra.Command{
	Use:   "mv <src> <dst>",
	Short: "Move file or directory",
	Long: `Move (rename) a file or directory.

Examples:
  dba fs mv src/old.tsx src/new.tsx
  dba fs mv src/components src/ui`,
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

		result, err := fs.Move(ctx.Workspace.ProjectPath, args[0], args[1])
		if err != nil {
			return err
		}

		return OutputResult(result)
	},
}

// ═══════════════════════════════════════════════════════════════════════════════
// dba fs rm
// ═══════════════════════════════════════════════════════════════════════════════

var fsRmCmd = &cobra.Command{
	Use:   "rm <path>",
	Short: "Remove file or directory",
	Long: `Remove a file or directory.

Examples:
  dba fs rm src/temp.tsx
  dba fs rm src/old/ --recursive`,
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

		recursive, _ := cmd.Flags().GetBool("recursive")
		force, _ := cmd.Flags().GetBool("force")

		// Force flag just suppresses not-found errors
		result, err := fs.Remove(ctx.Workspace.ProjectPath, args[0], recursive)
		if err != nil {
			if force {
				// With force, return success even if not found
				return OutputResult(&fs.RemoveResult{
					Path:    args[0],
					Success: true,
				})
			}
			return err
		}

		return OutputResult(result)
	},
}

// ═══════════════════════════════════════════════════════════════════════════════
// dba fs mkdir
// ═══════════════════════════════════════════════════════════════════════════════

var fsMkdirCmd = &cobra.Command{
	Use:   "mkdir <path>",
	Short: "Create directory",
	Long: `Create a new directory.

Examples:
  dba fs mkdir src/components
  dba fs mkdir src/a/b/c --parents`,
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

		parents, _ := cmd.Flags().GetBool("parents")

		result, err := fs.Mkdir(ctx.Workspace.ProjectPath, args[0], parents)
		if err != nil {
			return err
		}

		return OutputResult(result)
	},
}

func init() {
	// read flags
	fsReadCmd.Flags().Bool("base64", false, "Output as base64 (for binary files)")
	fsReadCmd.Flags().String("lines", "", "Line range (e.g., 10:20)")
	fsReadCmd.Flags().BoolP("follow-symlinks", "L", true, "Follow symbolic links")

	// write flags
	fsWriteCmd.Flags().String("content", "", "Content to write (if not provided, reads from stdin)")
	fsWriteCmd.Flags().Bool("base64", false, "Content is base64 encoded")
	fsWriteCmd.Flags().Bool("append", false, "Append to file instead of overwriting")
	fsWriteCmd.Flags().Bool("mkdir", false, "Create parent directories if needed")

	// list flags
	fsListCmd.Flags().BoolP("recursive", "r", false, "List recursively")
	fsListCmd.Flags().BoolP("hidden", "a", false, "Include hidden files")
	fsListCmd.Flags().String("pattern", "", "Glob pattern to filter files")
	fsListCmd.Flags().Int("max-depth", 0, "Maximum depth for recursive listing (0 = unlimited)")

	// search flags
	fsSearchCmd.Flags().String("pattern", "", "File pattern (glob) to search in")
	fsSearchCmd.Flags().Bool("regex", false, "Treat query as regular expression")
	fsSearchCmd.Flags().Bool("case-sensitive", false, "Case sensitive search")
	fsSearchCmd.Flags().IntP("context", "C", 0, "Lines of context around matches")
	fsSearchCmd.Flags().Int("max-results", 100, "Maximum number of results")
	fsSearchCmd.Flags().Bool("no-ignore", false, "Don't respect .gitignore patterns")
	fsSearchCmd.Flags().BoolP("follow-symlinks", "L", false, "Follow symbolic links")

	// cp flags
	fsCpCmd.Flags().BoolP("recursive", "r", false, "Copy directories recursively")

	// rm flags
	fsRmCmd.Flags().BoolP("recursive", "r", false, "Remove directories recursively")
	fsRmCmd.Flags().BoolP("force", "f", false, "Ignore errors if file doesn't exist")

	// mkdir flags
	fsMkdirCmd.Flags().BoolP("parents", "p", false, "Create parent directories as needed")

	// Add subcommands to fs command
	fsCmd.AddCommand(fsReadCmd)
	fsCmd.AddCommand(fsWriteCmd)
	fsCmd.AddCommand(fsListCmd)
	fsCmd.AddCommand(fsSearchCmd)
	fsCmd.AddCommand(fsCpCmd)
	fsCmd.AddCommand(fsMvCmd)
	fsCmd.AddCommand(fsRmCmd)
	fsCmd.AddCommand(fsMkdirCmd)

	// Add fs command to root
	rootCmd.AddCommand(fsCmd)
}
