# mnemo

[![CI](https://github.com/kyungw00k/mnemo/actions/workflows/ci.yml/badge.svg)](https://github.com/kyungw00k/mnemo/actions/workflows/ci.yml)
[![Latest Release](https://img.shields.io/github/v/release/kyungw00k/mnemo)](https://github.com/kyungw00k/mnemo/releases)
[![Go 1.22+](https://img.shields.io/badge/go-1.22+-00ADD8?logo=go)](https://go.dev/)
[![License](https://img.shields.io/github/license/kyungw00k/mnemo)](LICENSE)
[![SQLite](https://img.shields.io/badge/SQLite-supported-003B57?logo=sqlite)](https://sqlite.org/)
[![PostgreSQL](https://img.shields.io/badge/PostgreSQL-supported-4169E1?logo=postgresql)](https://www.postgresql.org/)

> Persistent MCP memory server for Claude Code and opencode —
> single Go binary, zero setup with SQLite, full vector search with PostgreSQL.

---

## Overview

AI coding tools like Claude Code and opencode lose all context between sessions. **mnemo** solves this by providing a persistent, context-aware memory store that survives restarts and is accessible across multiple tools simultaneously.

Named after **Mnemosyne** — the Greek goddess of memory.

### Key Features

- **Dual database support**: SQLite (zero setup, FTS5 full-text search) or PostgreSQL (pgvector HNSW vector search)
- **OpenAI Compatible embeddings**: works with Ollama, LM Studio, OpenAI, or any `/v1/embeddings` endpoint
- **Dual transport**: stdio (per-session) and SSE (persistent HTTP server) — run both simultaneously
- **11 MCP tools**: memory and note management, raw DB access with security guardrails
- **Host isolation**: memories are scoped per machine via `HOST_ID`
- **Single static binary**: no JVM, no runtime dependencies, CGO-free

---

## Architecture

```
Claude Code ──┐
              ├── MCP (stdio) ──┐
opencode ─────┤                 ├── mnemo ──── SQLite  (FTS5)
              └── MCP (SSE)  ───┘         └── PostgreSQL (pgvector)
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

```bash
docker pull ghcr.io/kyungw00k/mnemo:latest
```

### Build from source

```bash
git clone https://github.com/kyungw00k/mnemo.git
cd mnemo
make build
# binary at ./dist/mnemo
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

**Note**: In SQLite mode, embedding calls are skipped entirely. FTS5 full-text search is used instead. Embedding variables only take effect with PostgreSQL.

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

mnemo exposes 11 tools over MCP.

### Memory tools

| Tool | Description | Key Parameters |
|------|-------------|----------------|
| `memory_save` | Save or update a key-value memory entry | `category`, `key`, `value`, `metadata` (optional) |
| `memory_search` | Search memories by semantic similarity (vector) or full-text (FTS5) | `query`, `category` (optional), `limit` (optional) |
| `memory_list` | List memory entries for a category | `category`, `limit` (optional) |
| `memory_delete` | Soft-delete a memory entry | `id` or (`category` + `key`) |
| `memory_categories` | List all distinct memory categories for the current host | _(none)_ |

### Note tools

| Tool | Description | Key Parameters |
|------|-------------|----------------|
| `note_save` | Save a structured note with tags | `title`, `content`, `project` (optional), `tags` (string array, optional) |
| `note_search` | Search notes by semantic similarity or full-text | `query`, `project` (optional), `limit` (optional) |
| `note_list` | List notes for a project | `project`, `limit` (optional) |
| `note_delete` | Soft-delete a note by ID | `id` |

### Database tools

| Tool | Description | Key Parameters |
|------|-------------|----------------|
| `db_query` | Run a read-only SELECT query (max 100 rows) | `sql` |
| `db_execute` | Run a write statement (INSERT, UPDATE, or DELETE only) | `sql` |

`db_execute` uses an allowlist: only `INSERT`, `UPDATE`, and `DELETE` statements are accepted. `CREATE`, `DROP`, `TRUNCATE`, `ALTER`, `GRANT`, and `SELECT` are rejected. Semicolon injection is not possible (single statement enforced).

---

## Docker

```bash
docker run -d \
  --name mnemo \
  -p 8765:8765 \
  -e DB_URL=sqlite:///data/memory.db \
  -v ~/.mnemo:/data \
  ghcr.io/kyungw00k/mnemo:latest
```

With PostgreSQL and Ollama embeddings:

```bash
docker run -d \
  --name mnemo \
  -p 8765:8765 \
  -e DB_URL="postgres://user:password@host.docker.internal/mnemo" \
  -e EMBEDDING_BASE_URL="http://host.docker.internal:11434/v1" \
  -e EMBEDDING_MODEL="nomic-embed-text" \
  -e EMBEDDING_DIMENSIONS=768 \
  -e HOST_ID="my-workstation" \
  ghcr.io/kyungw00k/mnemo:latest
```

Set `HOST_ID` explicitly in containers so that memories are consistently scoped even if the container hostname changes.

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
