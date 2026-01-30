# dba CLI

DevBox Agent (DBA) - Cloud VMs for development.

## Installation

```bash
cd packages/dba
make build
./bin/dba --help
```

## Quick Start

```bash
# 1. Login
dba auth login

# 2. Create a VM
dba start                     # Creates VM, returns ID (e.g., dba_abc123)
dba start ./my-project        # Creates VM and syncs directory

# 3. Access the VM
dba code dba_abc123           # Open VS Code in browser
dba ssh dba_abc123            # SSH into VM
dba vnc dba_abc123            # Open VNC desktop in browser

# 4. Work with the VM
dba exec dba_abc123 "npm install"    # Run commands
dba sync dba_abc123 ./my-project     # Sync files to VM

# 5. Manage VM lifecycle
dba pause dba_abc123          # Pause (preserves state, saves cost)
dba resume dba_abc123         # Resume paused VM
dba delete dba_abc123         # Delete VM permanently

# 6. List VMs
dba ls                        # List all your VMs
```

## Commands

### Authentication

| Command | Description |
|---------|-------------|
| `dba auth login` | Login via browser (opens auth URL) |
| `dba auth logout` | Logout and clear credentials |
| `dba auth status` | Show authentication status |
| `dba auth whoami` | Alias for status |

### VM Lifecycle

| Command | Description |
|---------|-------------|
| `dba start [path]` | Create new VM, optionally sync directory |
| `dba start --snapshot <id>` | Create VM from specific snapshot |
| `dba delete <id>` | Delete VM permanently |
| `dba pause <id>` | Pause VM (preserves state) |
| `dba resume <id>` | Resume paused VM |

### Accessing VMs

| Command | Description |
|---------|-------------|
| `dba code <id>` | Open VS Code in browser |
| `dba vnc <id>` | Open VNC desktop in browser |
| `dba ssh <id>` | SSH into VM |

### Working with VMs

| Command | Description |
|---------|-------------|
| `dba exec <id> "<command>"` | Run a command in VM |
| `dba sync <id> <path>` | Sync local directory to VM |
| `dba sync <id> <path> --pull` | Pull files from VM to local |

### Listing and Status

| Command | Description |
|---------|-------------|
| `dba ls` | List all VMs (aliases: `list`, `ps`) |
| `dba status <id>` | Show VM status and URLs |

### Other

| Command | Description |
|---------|-------------|
| `dba version` | Show version info |
| `dba completion <shell>` | Generate shell autocompletions (bash/zsh/fish/powershell) |

## Global Flags

| Flag | Description |
|------|-------------|
| `-h, --help` | Show help for a command |
| `-v, --verbose` | Verbose output |

## Command Details

### `dba start [path]`

Create a new VM. Optionally sync a local directory.

```bash
dba start                       # Create VM (no sync)
dba start .                     # Create VM, sync current directory
dba start ./my-project          # Create VM, sync specific directory
dba start --snapshot=snap_xxx   # Create from specific snapshot
```

**Output:**
```
Creating VM...
VM created: dba_abc123
Waiting for VM to be ready...

✓ VM is ready!
  ID:       dba_abc123
  VS Code:  https://vscode-morphvm-xxx.http.cloud.morph.so
  VNC:      https://vnc-morphvm-xxx.http.cloud.morph.so
```

### `dba exec <id> "<command>"`

Execute a command in a VM.

```bash
dba exec dba_abc123 "ls -la"
dba exec dba_abc123 "npm install"
dba exec dba_abc123 "whoami && pwd && uname -a"
```

**Output:**
```
root
/root
Linux morphvm 5.10.225 #1 SMP Sun Dec 15 19:32:42 EST 2024 x86_64 GNU/Linux
```

### `dba sync <id> <path>`

Sync a local directory to/from a VM. Files are synced to `/home/user/project/` in the VM.

```bash
dba sync dba_abc123 .                  # Push current directory to VM
dba sync dba_abc123 ./my-project       # Push specific directory to VM
dba sync dba_abc123 ./output --pull    # Pull from VM to local
```

**Excluded by default:** `.git`, `node_modules`, `.next`, `dist`, `build`, `__pycache__`, `.venv`, `venv`, `target`

### `dba ls`

List all your VMs. Aliases: `list`, `ps`

```bash
dba ls
```

**Output:**
```
ID                   STATUS     VS CODE URL
-------------------- ---------- ----------------------------------------
dba_abc123           running
dba_def456           paused
```

### `dba status <id>`

Show detailed status of a VM.

```bash
dba status dba_abc123
```

**Output:**
```
ID:       dba_abc123
Status:   running
VS Code:  https://vscode-morphvm-xxx.http.cloud.morph.so
VNC:      https://vnc-morphvm-xxx.http.cloud.morph.so
```

## Examples

### Typical Development Workflow

```bash
# Start of day: create or resume a VM
dba start ./my-project
# → dba_abc123

# Work on your code
dba code dba_abc123        # Opens VS Code in browser

# Run commands
dba exec dba_abc123 "npm run dev"

# Sync changes
dba sync dba_abc123 ./my-project

# End of day: pause to save costs
dba pause dba_abc123

# Next day: resume where you left off
dba resume dba_abc123
```

### Multiple VMs

```bash
# Create multiple VMs for different tasks
dba start ./frontend    # → dba_frontend1
dba start ./backend     # → dba_backend1

# Work on them independently
dba code dba_frontend1
dba code dba_backend1

# List all
dba ls
```

### Pull Files from VM

```bash
# After building/generating files in VM
dba exec dba_abc123 "npm run build"

# Pull the output
dba sync dba_abc123 ./dist --pull
```

### Shell Completion

```bash
# Bash
dba completion bash > /etc/bash_completion.d/dba

# Zsh
dba completion zsh > "${fpath[1]}/_dba"

# Fish
dba completion fish > ~/.config/fish/completions/dba.fish
```

## Environment Variables

| Variable | Description |
|----------|-------------|
| `DBA_DEV=1` | Use development environment |

## Development

```bash
# Build
make build

# Run directly
go run ./cmd/dba --help

# Build with race detector
make build-race
```
