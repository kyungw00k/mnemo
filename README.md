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

# Start (SSE mode on port 8765)
TRANSPORT=sse mnemo
```

Memory is stored at `~/.mnemo/memory.db`. Dashboard available at `http://localhost:8765`.

### Configure Claude Code

Add to `~/.config/claude/mcp.json`:

```json
{
  "mcpServers": {
    "mnemo": {
      "type": "sse",
      "url": "http://localhost:8765/sse"
    }
  }
}
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
| `ENABLE_AUTO_EXTRACT` | `false` | Enable LLM auto-extraction tool |
| `ENABLE_GIT_CONTEXT` | `false` | Auto-detect git repo for project tagging |
| `MEMORY_TTL_DAYS` | `0` (off) | Auto-expire memories after N days |
| `EXTRACT_LLM_BASE_URL` | _(same as EMBEDDING)_ | LLM endpoint for extraction |
| `EXTRACT_LLM_MODEL` | `llama3` | Model name for extraction |

</details>

---

## Dashboard

Start with SSE transport and visit `http://localhost:8765`:

```bash
TRANSPORT=sse mnemo
```

![mnemo dashboard](docs/screenshot.png)

**Features**: Browse memories/notes, full-text search, markdown rendering, detail view modals, knowledge graph visualization.

---

## Features

- **Dual database**: SQLite (sqlite-vec) or PostgreSQL (pgvector) for vector search
- **OpenAI-compatible embeddings**: Works with Ollama, LM Studio, OpenAI, or any `/v1/embeddings` endpoint
- **Dual transport**: stdio (per-session) and SSE (persistent HTTP) — run both simultaneously
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

**opencode** (`.cursorrules` or `AGENTS.md`):

```json
{
  "mcp": {
    "mnemo": {
      "type": "sse",
      "url": "http://localhost:8765/sse"
    }
  }
}
```

See [`AGENT_INSTRUCTIONS.md.example`](AGENT_INSTRUCTIONS.md.example) for AI memory protocol templates.

---

## License

[MIT](LICENSE)
