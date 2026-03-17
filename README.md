<p align="center">
  <img src="logo.svg" alt="mnemo logo" width="96" height="96">
</p>

<h1 align="center">mnemo</h1>

<p align="center">
  <a href="https://github.com/kyungw00k/mnemo/actions/workflows/ci.yml">
    <img src="https://github.com/kyungw00k/mnemo/actions/workflows/ci.yml/badge.svg" alt="CI">
  </a>
  <a href="https://github.com/kyungw00k/mnemo/releases">
    <img src="https://img.shields.io/github/v/release/kyungw00k/mnemo" alt="Latest Release">
  </a>
  <a href="https://go.dev/">
    <img src="https://img.shields.io/badge/go-1.22+-00ADD8?logo=go" alt="Go 1.22+">
  </a>
  <a href="LICENSE">
    <img src="https://img.shields.io/github/license/kyungw00k/mnemo" alt="License">
  </a>
</p>

Persistent MCP memory server for Claude Code and opencode — single Go binary, vector search in SQLite and PostgreSQL.

---

## Quick Start

```bash
# Install
go install github.com/kyungw00k/mnemo/cmd/mnemo@latest
```

Memory is stored at `~/.mnemo/memory.db`.

### Configure Claude Code

**1. MCP (stdio)** — add to `~/.claude/claude_desktop_config.json`:

```json
{
  "mcpServers": {
    "mnemo": {
      "command": "mnemo",
      "env": { "TRANSPORT": "stdio" }
    }
  }
}
```

No persistent server needed — Claude spawns mnemo per session.

**2. Hooks (automatic memory)** — add to `~/.claude/settings.json`:

```json
{
  "hooks": {
    "SessionStart": [
      { "matcher": "startup|resume",
        "hooks": [{ "type": "command", "command": "mnemo hook session-start" }] }
    ],
    "PostToolUse": [
      { "matcher": "Edit|Write|Bash",
        "hooks": [{ "type": "command", "command": "mnemo hook observe" }] }
    ],
    "Stop": [
      { "hooks": [{ "type": "command", "command": "mnemo hook session-end" }] }
    ]
  }
}
```

**3. Dashboard (on-demand)**:

```bash
mnemo dashboard          # opens http://localhost:8765
mnemo dashboard --port 9000
```

---

## Installation

**Pre-built binary** (recommended):

```bash
curl -L https://github.com/kyungw00k/mnemo/releases/latest/download/mnemo_darwin_arm64.tar.gz | tar xz
sudo mv mnemo /usr/local/bin/
```

**Go install**:

```bash
go install github.com/kyungw00k/mnemo/cmd/mnemo@latest
```

**Docker**:

```bash
docker run -d --name mnemo \
  -p 8765:8765 \
  -v ~/.mnemo:/root/.mnemo \
  -e TRANSPORT=sse \
  ghcr.io/kyungw00k/mnemo:latest
```

<details>
<summary>Docker with PostgreSQL + Embeddings</summary>

```bash
docker run -d --name mnemo \
  -p 8765:8765 \
  -e DB_URL=postgres://postgres:postgres@host.docker.internal:5432/mnemo \
  -e EMBEDDING_BASE_URL=http://host.docker.internal:11434/v1 \
  -e TRANSPORT=sse \
  ghcr.io/kyungw00k/mnemo:latest
```
</details>

**Build from source** (requires CGO):

```bash
git clone https://github.com/kyungw00k/mnemo.git
cd mnemo
make build
```

---

## Configuration

| Variable | Default | Description |
|----------|---------|-------------|
| `DB_URL` | `sqlite://~/.mnemo/memory.db` | Database (SQLite or PostgreSQL) |
| `TRANSPORT` | `both` | `stdio`, `sse`, or `both` |
| `SSE_PORT` | `8765` | HTTP port for SSE server |
| `EMBEDDING_BASE_URL` | `http://localhost:11434/v1` | OpenAI-compatible embeddings endpoint |
| `EMBEDDING_API_KEY` | _(empty)_ | API key (empty for local Ollama/LM Studio) |
| `EMBEDDING_MODEL` | `nomic-embed-text` | Embedding model name |
| `EMBEDDING_DIMENSIONS` | `768` | Vector dimensions |

<details>
<summary>Optional: Advanced Configuration</summary>

| Variable | Default | Description |
|----------|---------|-------------|
| `HOST_ID` | `<os.Hostname()>` | Memory isolation key (for Docker) |
| `ENABLE_GIT_CONTEXT` | `true` | Auto-detect git remote for project tagging |
| `ENABLE_AUTO_EXTRACT` | `false` | LLM-based fact extraction on session-end |
| `MEMORY_TTL_DAYS` | `0` (off) | Auto-expire memories after N days |
| `EXTRACT_LLM_BASE_URL` | _(same as EMBEDDING)_ | LLM endpoint for extraction |
| `EXTRACT_LLM_MODEL` | `gpt-4.1-nano` | Model name for extraction |

</details>

---

## Dashboard

```bash
mnemo dashboard          # opens http://localhost:8765
mnemo dashboard --port 9000
```

![mnemo dashboard](docs/screenshot.png)

**Features**: Browse memories/notes, full-text search, markdown rendering, detail view modals, knowledge graph visualization.

---

## CLI Subcommands

No server needed — all subcommands access the DB directly.

```bash
# Hooks (called by Claude Code automatically via settings.json)
mnemo hook session-start   # returns additionalContext with recent decisions + notes
mnemo hook session-end     # saves session note; extracts facts if ENABLE_AUTO_EXTRACT=true
mnemo hook observe         # records Edit/Write/Bash tool usage as observations

# Interactive
mnemo search "authentication strategy"
mnemo search "database" --category decision --limit 5
mnemo save decision auth_strategy "JWT with refresh tokens"

# Dashboard (on-demand SSE server)
mnemo dashboard
mnemo dashboard --port 9000
```

---

## Features

- **Dual database**: SQLite (sqlite-vec) or PostgreSQL (pgvector) for vector search
- **OpenAI-compatible embeddings**: Works with Ollama, LM Studio, OpenAI, or any `/v1/embeddings` endpoint
- **Dual transport**: stdio (per-session) and SSE (persistent HTTP) — run both simultaneously
- **Hook automation**: Session-start context injection, session-end note saving, tool-use observation
- **Git context**: Auto-detects `git remote origin` for project tagging (on by default)
- **Host isolation**: Memories scoped per machine via `HOST_ID`
- **Single static binary**: No runtime dependencies (CGO at build time only)

---

## MCP Tools

| Tool | Description |
|------|-------------|
| `memory_save` | Save key-value memory |
| `memory_search` | Search by vector similarity or text |
| `memory_list` | List memories by category |
| `memory_delete` | Delete memory |
| `note_save` | Save structured note with tags |
| `note_search` | Search notes |
| `note_list` | List notes by project |
| `note_delete` | Delete note |
| `db_query` | Read-only SELECT (max 100 rows) |
| `db_execute` | Write statements (INSERT/UPDATE/DELETE only) |

<details>
<summary>Optional Tools (enable via env vars)</summary>

| Tool | Env Var |
|------|---------|
| `memory_extract` | `ENABLE_AUTO_EXTRACT=true` |
| `memory_cleanup` | `MEMORY_TTL_DAYS>0` |
| `memory_export` | Always available |
| `memory_import` | Always available |

</details>

---

## Usage with Other AI Tools

**opencode / Cursor / Codex** — add MCP to your tool's config:

```json
{
  "mcpServers": {
    "mnemo": {
      "command": "mnemo",
      "env": { "TRANSPORT": "stdio" }
    }
  }
}
```

Or use SSE if a persistent server is preferred:

```json
{
  "mcp": {
    "mnemo": { "type": "sse", "url": "http://localhost:8765/sse" }
  }
}
```

See [`AGENT_INSTRUCTIONS.md.example`](AGENT_INSTRUCTIONS.md.example) for hook setup and AI memory protocol templates.

---

## License

[MIT](LICENSE)
