# mnemo

[![CI](https://github.com/kyungw00k/mnemo/actions/workflows/ci.yml/badge.svg)](https://github.com/kyungw00k/mnemo/actions/workflows/ci.yml)
[![Latest Release](https://img.shields.io/github/v/release/kyungw00k/mnemo)](https://github.com/kyungw00k/mnemo/releases)
[![Go 1.22+](https://img.shields.io/badge/go-1.22+-00ADD8?logo=go)](https://go.dev/)
[![License](https://img.shields.io/github/license/kyungw00k/mnemo)](LICENSE)
[![SQLite](https://img.shields.io/badge/SQLite-sqlite--vec-003B57?logo=sqlite)](https://sqlite.org/)
[![PostgreSQL](https://img.shields.io/badge/PostgreSQL-pgvector-4169E1?logo=postgresql)](https://www.postgresql.org/)

> Persistent MCP memory server for Claude Code and opencode —
> single Go binary, vector search in both SQLite and PostgreSQL.

---

## Overview

AI coding tools like Claude Code and opencode lose all context between sessions. **mnemo** solves this by providing a persistent, context-aware memory store that survives restarts and is accessible across multiple tools simultaneously.

Named after **Mnemosyne** — the Greek goddess of memory.

### Key Features

- **Dual database support**: SQLite (zero setup, [sqlite-vec](https://github.com/asg017/sqlite-vec) cosine similarity + FTS5 fallback) or PostgreSQL (pgvector HNSW vector search)
- **OpenAI Compatible embeddings**: works with Ollama, LM Studio, OpenAI, or any `/v1/embeddings` endpoint — for **both** SQLite and PostgreSQL
- **Dual transport**: stdio (per-session) and SSE (persistent HTTP server) — run both simultaneously
- **11 MCP tools** (+ 4 optional): memory and note management, raw DB access with security guardrails
- **Host isolation**: memories are scoped per machine via `HOST_ID`
- **Single static binary**: no JVM, no runtime dependencies (CGO at build time only)

---

## Architecture

```
Claude Code ──┐
              ├── MCP (stdio) ──┐
opencode ─────┤                 ├── mnemo ──── SQLite  (sqlite-vec cosine similarity)
              └── MCP (SSE)  ───┘         └── PostgreSQL (pgvector HNSW)
```

By default (`TRANSPORT=both`), mnemo runs stdio and SSE simultaneously. Claude Code and opencode both connect to the same in-process state, giving them shared memory.

---

## Quickstart (SQLite — no setup required)

### 1. Download the binary

Download the latest binary from [GitHub Releases](https://github.com/kyungw00k/mnemo/releases) and place it in your `$PATH`:

```bash
# macOS (arm64 example)
curl -L https://github.com/kyungw00k/mnemo/releases/latest/download/mnemo_darwin_arm64.tar.gz | tar xz
sudo mv mnemo /usr/local/bin/
```

Or install with Go:

```bash
go install github.com/kyungw00k/mnemo/cmd/mnemo@latest
```

### 2. Start the server

```bash
# Runs both stdio and SSE on port 8765 (default)
mnemo

# SSE only
TRANSPORT=sse mnemo
```

Memory is stored at `~/.mnemo/memory.db` by default.

### 3. Configure your AI tool

See the [Usage](#usage) section below for Claude Code and opencode configuration.

---

## Installation

### Pre-built binary (recommended)

Download from [GitHub Releases](https://github.com/kyungw00k/mnemo/releases). Binaries are available for Linux, macOS, and Windows.

### go install

```bash
go install github.com/kyungw00k/mnemo/cmd/mnemo@latest
```

### Docker

Pre-built multi-platform images (`linux/amd64`, `linux/arm64`) are published to GHCR automatically:

| Tag | When published |
|-----|---------------|
| `latest` | On each release tag (`v*`) |
| `vX.Y.Z` | On each release tag |
| `dev` | On every push to `main` |

```bash
# Run with SQLite (zero setup)
docker run -d --name mnemo \
  -p 8765:8765 \
  -v ~/.mnemo:/root/.mnemo \
  -e TRANSPORT=sse \
  ghcr.io/kyungw00k/mnemo:latest

# Run with SQLite + embeddings (vector search)
docker run -d --name mnemo \
  -p 8765:8765 \
  -v ~/.mnemo:/root/.mnemo \
  -e EMBEDDING_BASE_URL=http://host.docker.internal:11434/v1 \
  -e TRANSPORT=sse \
  -e HOST_ID=mymachine \
  ghcr.io/kyungw00k/mnemo:latest

# Run with PostgreSQL + embeddings
docker run -d --name mnemo \
  -p 8765:8765 \
  -e DB_URL=postgres://postgres:postgres@host.docker.internal:5432/mnemo \
  -e EMBEDDING_BASE_URL=http://host.docker.internal:11434/v1 \
  -e TRANSPORT=sse \
  ghcr.io/kyungw00k/mnemo:latest
```

> **Docker + SQLite**: use `-e HOST_ID=mymachine` to fix the hostname so memories persist across container restarts.

### Build from source

mnemo uses [sqlite-vec](https://github.com/asg017/sqlite-vec) for vector search in SQLite mode, which requires a C compiler at build time. The final output is still a single static binary.

| Platform | Requirement |
|----------|-------------|
| macOS | Xcode Command Line Tools (`xcode-select --install`) |
| Linux | `gcc` (e.g. `apt-get install gcc`) |
| Docker | Handled automatically in the Dockerfile |

```bash
git clone https://github.com/kyungw00k/mnemo.git
cd mnemo
make build
# binary at ./dist/mnemo

# Linux static binary (for deployment without libc)
make build-linux-static
```

---

## Configuration

All configuration is via environment variables. See `.env.example` for a template.

| Variable | Default | Description |
|----------|---------|-------------|
| `DB_URL` | `sqlite://~/.mnemo/memory.db` | Database connection string. Use `postgres://user:pass@host/db` for PostgreSQL. |
| `TRANSPORT` | `both` | Transport mode: `stdio`, `sse`, or `both` |
| `SSE_PORT` | `8765` | HTTP port for SSE server |
| `HOST_ID` | `<os.Hostname()>` | Memory isolation key. Override for Docker/containers. |
| `EMBEDDING_BASE_URL` | `http://localhost:11434/v1` | OpenAI Compatible embeddings endpoint |
| `EMBEDDING_API_KEY` | _(empty)_ | API key. Empty = no `Authorization` header (works with local Ollama/LM Studio) |
| `EMBEDDING_MODEL` | `nomic-embed-text` | Embedding model name |
| `EMBEDDING_DIMENSIONS` | `768` | Vector dimensions. Must match the model's output size. |

**Embedding variables work with both SQLite and PostgreSQL.** In SQLite mode, embeddings are stored via sqlite-vec for cosine similarity search. If no embedding endpoint is configured, FTS5 full-text search is used as a fallback.

### Optional features

| Variable | Default | Description |
|----------|---------|-------------|
| `ENABLE_AUTO_EXTRACT` | `false` | Enable `memory_extract` tool — LLM auto-extracts key memories from conversation text |
| `ENABLE_GIT_CONTEXT` | `false` | Auto-detect git repo and tag memories with the project name |
| `MEMORY_TTL_DAYS` | `0` (off) | Auto-expire memories after N days; enables `memory_cleanup` tool |
| `EXTRACT_LLM_BASE_URL` | _(same as EMBEDDING_BASE_URL)_ | LLM endpoint for extraction (OpenAI Compatible) |
| `EXTRACT_LLM_API_KEY` | _(empty)_ | API key for extraction LLM |
| `EXTRACT_LLM_MODEL` | `llama3` | Model name for extraction |

---

## Usage

### With Claude Code

#### SSE mode (recommended — shared memory across tools)

Start the server first (`mnemo` or `TRANSPORT=sse mnemo`), then add to your Claude Code MCP config:

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

#### stdio mode

mnemo is spawned per session. Each session shares the same database.

```json
{
  "mcpServers": {
    "mnemo": {
      "command": "/usr/local/bin/mnemo",
      "env": {
        "TRANSPORT": "stdio"
      }
    }
  }
}
```

### With opencode

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

---

## Using SQLite with Vector Search

SQLite mode supports real cosine similarity search via [sqlite-vec](https://github.com/asg017/sqlite-vec). No separate database setup is required — just configure an embedding endpoint.

```bash
# With Ollama (local)
export EMBEDDING_BASE_URL="http://localhost:11434/v1"
export EMBEDDING_MODEL="nomic-embed-text"
export EMBEDDING_DIMENSIONS=768
mnemo

# With OpenAI
export EMBEDDING_BASE_URL="https://api.openai.com/v1"
export EMBEDDING_API_KEY="sk-..."
export EMBEDDING_MODEL="text-embedding-3-small"
export EMBEDDING_DIMENSIONS=1536
mnemo
```

If no embedding endpoint is available, mnemo falls back to FTS5 full-text search automatically.

---

## Using PostgreSQL with Vector Search

PostgreSQL enables semantic (vector) search using pgvector HNSW indexes. Text search remains available as a fallback if embedding fails.

### 1. Install PostgreSQL and pgvector

```bash
# macOS with Homebrew
brew install postgresql pgvector

# Ubuntu / Debian
sudo apt install postgresql
# then install pgvector: https://github.com/pgvector/pgvector
```

### 2. Create the database and enable the extension

```sql
CREATE DATABASE mnemo;
\c mnemo
CREATE EXTENSION IF NOT EXISTS vector;
```

### 3. Configure mnemo

```bash
export DB_URL="postgres://user:password@localhost/mnemo"
export EMBEDDING_BASE_URL="http://localhost:11434/v1"   # Ollama
# export EMBEDDING_BASE_URL="https://api.openai.com/v1" # OpenAI
export EMBEDDING_API_KEY=""                              # empty for Ollama/LM Studio
export EMBEDDING_MODEL="nomic-embed-text"
export EMBEDDING_DIMENSIONS=768
```

Migrations run automatically on startup. HNSW indexes are created for fast approximate nearest-neighbor search.

### Supported embedding providers

| Provider | `EMBEDDING_BASE_URL` | `EMBEDDING_API_KEY` |
|----------|----------------------|----------------------|
| Ollama (local) | `http://localhost:11434/v1` | _(empty)_ |
| LM Studio (local) | `http://localhost:1234/v1` | _(empty)_ |
| OpenAI | `https://api.openai.com/v1` | your API key |
| Any OpenAI-compatible API | your endpoint | as required |

---

## MCP Tools Reference

mnemo exposes 11 tools over MCP, with up to 4 additional optional tools enabled via environment variables.

### Memory tools

| Tool | Description | Key Parameters |
|------|-------------|----------------|
| `memory_save` | Save or update a key-value memory entry | `category`, `key`, `value`, `metadata` (optional) |
| `memory_search` | Search memories by cosine similarity (vector) or full-text (FTS5) | `query`, `category` (optional), `limit` (optional) |
| `memory_list` | List memory entries for a category | `category`, `limit` (optional) |
| `memory_delete` | Soft-delete a memory entry | `id` or (`category` + `key`) |
| `memory_categories` | List all distinct memory categories for the current host | _(none)_ |

### Note tools

| Tool | Description | Key Parameters |
|------|-------------|----------------|
| `note_save` | Save a structured note with tags | `title`, `content`, `project` (optional), `tags` (string array, optional) |
| `note_search` | Search notes by cosine similarity or full-text | `query`, `project` (optional), `limit` (optional) |
| `note_list` | List notes for a project | `project`, `limit` (optional) |
| `note_delete` | Soft-delete a note by ID | `id` |

### Database tools

| Tool | Description | Key Parameters |
|------|-------------|----------------|
| `db_query` | Run a read-only SELECT query (max 100 rows) | `sql` |
| `db_execute` | Run a write statement (INSERT, UPDATE, or DELETE only) | `sql` |

`db_execute` uses an allowlist: only `INSERT`, `UPDATE`, and `DELETE` statements are accepted. `CREATE`, `DROP`, `TRUNCATE`, `ALTER`, `GRANT`, and `SELECT` are rejected. Semicolon injection is not possible (single statement enforced).

### Optional tools

Enabled via environment variables:

| Tool | Env Var | Description |
|------|---------|-------------|
| `memory_extract` | `ENABLE_AUTO_EXTRACT=true` | LLM auto-extracts key memories from a conversation text block |
| `memory_cleanup` | `MEMORY_TTL_DAYS>0` | Hard-delete all expired memories |
| `memory_export` | _(always on)_ | Export all memories as JSON or Markdown |
| `memory_import` | _(always on)_ | Bulk-import memories from JSON |

---

## AI Tool Integration Guide

mnemo works with any MCP-compatible AI coding tool. Each tool uses a different instruction file, but the content is identical — copy the provided template and rename it:

| Tool | Instruction File | Copy Command |
|------|-----------------|--------------|
| Claude Code | `CLAUDE.md` | `cp AGENT_INSTRUCTIONS.md.example CLAUDE.md` |
| OpenAI Codex CLI | `AGENTS.md` | `cp AGENT_INSTRUCTIONS.md.example AGENTS.md` |
| opencode | `AGENTS.md` | `cp AGENT_INSTRUCTIONS.md.example AGENTS.md` |
| Cursor | `.cursorrules` | `cp AGENT_INSTRUCTIONS.md.example .cursorrules` |
| GitHub Copilot | `.github/copilot-instructions.md` | `cp AGENT_INSTRUCTIONS.md.example .github/copilot-instructions.md` |

The template (`AGENT_INSTRUCTIONS.md.example`) instructs the AI to:
- **Load memories at session start** — search and summarize before working
- **Save decisions and conventions** — structured by category (`decision`, `bug`, `config`, `convention`, `preference`)
- **Write session notes** — longer-form summaries with project + tags
- **Clean up on session end** — save key outcomes before closing

### Recommended Instruction Snippet

Add this to your instruction file to activate memory at every session:

````markdown
## Memory Protocol

At the start of every session:
1. Call memory_search("project setup decisions conventions")
2. Call memory_list(category="decision")
3. Call note_list() to see recent notes
4. Summarize recalled context before starting work

Save to memory when:
- An architectural decision is made → category: "decision"
- A non-obvious bug is fixed → category: "bug"
- Project conventions are established → category: "convention"
- Always set project = "<your-repo-name>"
````

See [`AGENT_INSTRUCTIONS.md.example`](AGENT_INSTRUCTIONS.md.example) for the full template with search tips, session-end protocol, and per-category guidance.

---

## Contributing

Standard Go project conventions apply.

```bash
# Run tests
make test

# Build binary
make build

# Lint (requires golangci-lint)
make lint
```

Commit messages follow [Conventional Commits](https://www.conventionalcommits.org/). Releases are automated via [Release Please](https://github.com/googleapis/release-please).

---

## License

[MIT](LICENSE)
