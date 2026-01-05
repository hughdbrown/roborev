# roborev

Automatic code review for git commits using AI agents.

## Install

```bash
curl -fsSL https://raw.githubusercontent.com/wesm/roborev/main/scripts/install.sh | bash
```

Or with Go:

```bash
go install github.com/wesm/roborev/cmd/roborev@latest
go install github.com/wesm/roborev/cmd/roborevd@latest
```

## Quick Start

```bash
cd your-repo
roborev init
```

This installs a post-commit hook. Every commit is now reviewed automatically.

## Usage

```bash
roborev status       # Show queue and daemon status
roborev show         # Show review for HEAD
roborev show abc123  # Show review for specific commit
roborev respond HEAD # Add response to a review
roborev-tui          # Interactive terminal UI
```

## Configuration

Per-repository `.roborev.toml`:

```toml
agent = "claude-code"    # or "codex"
review_context_count = 5
```

Global `~/.roborev/config.toml`:

```toml
server_addr = "127.0.0.1:7373"
max_workers = 4
default_agent = "codex"
```

## Architecture

roborev runs as a local daemon that processes review jobs in parallel.

```
~/.roborev/
├── config.toml    # Configuration
├── daemon.json    # Runtime state (port, PID)
└── reviews.db     # SQLite database
```

The daemon starts automatically when needed and handles port conflicts by finding an available port.

## Agents

- `codex` - OpenAI Codex CLI
- `claude-code` - Anthropic Claude Code CLI

Agent selection priority:
1. `--agent` flag on enqueue
2. Per-repo `.roborev.toml`
3. Global config
4. Default: `codex`

## Commands

| Command | Description |
|---------|-------------|
| `roborev init` | Initialize in current repo |
| `roborev status` | Show daemon and queue status |
| `roborev show [sha]` | Display review |
| `roborev respond <sha>` | Add response |
| `roborev enqueue` | Manually enqueue a commit |
| `roborev daemon start\|stop\|restart` | Manage daemon |
| `roborev install-hook` | Install git hook only |

## Development

```bash
git clone https://github.com/wesm/roborev
cd roborev
go test ./...
go install ./cmd/...
```

## License

MIT
