# mnemo — Project Context

> This file exists to preserve implementation context across Claude Code sessions.
> If context is compressed, read this file first before continuing work.

---

## What is mnemo?

**mnemo** is a persistent MCP (Model Context Protocol) memory server written in Go.
It solves the session-based memory limitation of AI coding tools (Claude Code, opencode)
by providing persistent, context-aware memory storage backed by PostgreSQL or SQLite.

Named after **Mnemosyne** — the Greek goddess of memory.

## Repository

- GitHub: https://github.com/kyungw00k/mnemo
- Local: /Users/humphrey.park/Sandbox/mnemo
- Module: `github.com/kyungw00k/mnemo`
- Go version: 1.22+

## Origin

Rewrite of: `/Users/humphrey.park/Sandbox/claude-memory-mcp-standalone` (Java/Spring Boot)
Reasons for rewrite:
1. JVM dependency removed → single static Go binary
2. Ollama-specific embedding → OpenAI Compatible endpoint (standard)
3. stdio-only → dual transport (stdio + SSE) for multi-client access
4. Hard-coded PostgreSQL only → SQLite (default) + PostgreSQL
5. Fixed vector(768) → configurable EMBEDDING_DIMENSIONS
6. Security fixes in db_execute (blocklist → allowlist)
7. Missing tools added: note_delete, note_list, memory_categories
8. memory_list DB-level LIMIT (was in-memory truncation)

## Architecture Decisions

### DB Strategy
- **SQLite** (default, Phase 14 pending): `modernc.org/sqlite` → will migrate to `mattn/go-sqlite3` + `sqlite-vec`
  - Current: FTS5 keyword search only
  - Phase 14: real cosine similarity via `vec0` virtual tables (CGO at build time, single binary at runtime)
  - Default path: `~/.mnemo/memory.db`
- **PostgreSQL**: pgx/v5 + pgvector-go, HNSW indexes, cosine similarity
  - Requires pgvector extension

### Embedding Strategy
- **OpenAI Compatible API** (`/v1/embeddings`)
  - `EMBEDDING_BASE_URL` (e.g., `http://localhost:11434/v1` for Ollama, `https://api.openai.com/v1`)
  - `EMBEDDING_API_KEY` (empty = no Authorization header, works for local Ollama/LM Studio)
  - `EMBEDDING_MODEL` (e.g., `nomic-embed-text`, `text-embedding-3-small`)
  - `EMBEDDING_DIMENSIONS` (default: 768, must match model output)
- **SQLite mode (current)**: embedding calls skipped, FTS5 used instead
- **SQLite mode (Phase 14)**: embedding called, stored as BLOB, cosine similarity via sqlite-vec
- **Fallback**: if embedding fails → FTS5 / LIKE text search

### Phase 14: sqlite-vec Plan
- Replace `modernc.org/sqlite` with `mattn/go-sqlite3` + `github.com/asg017/sqlite-vec-go-bindings/cgo`
- CGO required at **build time only** — runtime binary is still single file
- macOS: `xcode-select --install` sufficient (Apple Clang built-in, CGO_ENABLED=1 is default)
- Linux: `gcc` + `-ldflags="-extldflags=-static"` for fully static binary
- Docker: add `apk add --no-cache gcc musl-dev` to builder stage
- CI: add `sudo apt-get install -y gcc` to workflow

### Transport Strategy
- **stdio**: Claude Code integration (spawned process per session)
- **SSE** (`/sse` endpoint): HTTP server for opencode + multi-client access
- **TRANSPORT=both** (default): runs both simultaneously
  - Claude Code uses stdio OR SSE
  - opencode uses SSE
  - Both connect to same in-process state → shared memory

### Host Isolation
- `HOST_ID` env var (explicit override for Docker/containers)
- Falls back to `os.Hostname()` auto-detection
- All memories scoped per host → separate machines = separate memories

### Phase 13: Differentiation Features vs mem0

All features are **opt-in via env vars** (default off/disabled):

| Feature | Env Var | Default | Description |
|---------|---------|---------|-------------|
| Auto memory extraction | `ENABLE_AUTO_EXTRACT` | `false` | LLM extracts key memories from conversation text via `memory_extract` tool |
| Git project auto-tagging | `ENABLE_GIT_CONTEXT` | `false` | Auto-detects git repo → sets `project` field on save |
| Memory TTL | `MEMORY_TTL_DAYS` | `0` (off) | Auto-expires old memories; `memory_cleanup` tool for manual purge |
| Export/Import | always on | — | `memory_export` (JSON/Markdown) + `memory_import` (bulk upsert) |

Auto extraction uses OpenAI Compatible endpoint (separate from embedding):
- `EXTRACT_LLM_BASE_URL`, `EXTRACT_LLM_API_KEY`, `EXTRACT_LLM_MODEL`

TTL adds `expires_at` column via migrations 003 (postgres + sqlite).

## MCP Tools (11 base + up to 4 optional)

| Tool | Description | Note |
|------|-------------|------|
| `memory_save` | Upsert key-value memory | |
| `memory_search` | Vector/FTS5 search with score | |
| `memory_list` | List by category (DB LIMIT) | Fixed: was in-memory |
| `memory_delete` | Soft delete by id or category+key | |
| `memory_categories` | List distinct categories | New |
| `note_save` | Save structured note with tags | tags as []string |
| `note_search` | Vector/FTS5 search notes | |
| `note_list` | List notes by project | New |
| `note_delete` | Soft delete note by id | New |
| `db_query` | SELECT only, max 100 rows | |
| `db_execute` | INSERT/UPDATE/DELETE only (allowlist) | Security fix |
| `memory_extract` | LLM auto-extract from conversation | opt-in: `ENABLE_AUTO_EXTRACT=true` |
| `memory_cleanup` | Hard-delete expired memories | opt-in: `MEMORY_TTL_DAYS>0` |
| `memory_export` | Export as JSON or Markdown | always on |
| `memory_import` | Bulk import JSON memories | always on |

### Security Improvements over Original
- `db_execute`: allowlist approach (only INSERT/UPDATE/DELETE), rejects CREATE/DROP/TRUNCATE/ALTER/SELECT/GRANT etc.
- `db_query`: hard limit 100 rows enforced in SQL
- `db_execute`: no semicolon injection vulnerability (single statement enforced)

## Key Dependencies

```go
// go.mod dependencies (current)
github.com/mark3labs/mcp-go          // MCP server (stdio + SSE)
github.com/jackc/pgx/v5              // PostgreSQL driver
github.com/pgvector/pgvector-go      // pgvector type support
modernc.org/sqlite                   // SQLite (pure Go, CGO-free) — Phase 14에서 교체 예정

// Phase 14 추가 예정
github.com/mattn/go-sqlite3                       // SQLite CGO driver
github.com/asg017/sqlite-vec-go-bindings/cgo      // sqlite-vec extension
```

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `DB_URL` | `sqlite://~/.mnemo/memory.db` | DB connection string |
| `EMBEDDING_BASE_URL` | `http://localhost:11434/v1` | OpenAI Compatible endpoint |
| `EMBEDDING_API_KEY` | `` | API key (empty = no auth header) |
| `EMBEDDING_MODEL` | `nomic-embed-text` | Embedding model name |
| `EMBEDDING_DIMENSIONS` | `768` | Vector dimensions |
| `TRANSPORT` | `both` | `stdio` \| `sse` \| `both` |
| `SSE_PORT` | `8765` | SSE server port |
| `HOST_ID` | `<os.Hostname()>` | Memory isolation key |

## Claude Code Integration

### SSE mode (recommended — shared with opencode)
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

### stdio mode
```json
{
  "mcpServers": {
    "mnemo": {
      "command": "/usr/local/bin/mnemo",
      "env": { "TRANSPORT": "stdio", "DB_URL": "sqlite:///Users/you/.mnemo/memory.db" }
    }
  }
}
```

## opencode Integration
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

## GitHub Actions

- **CI** (`ci.yml`): Go 1.22/1.23 matrix, gotestsum, build+vet+unit tests
  - Pattern: from `github.com/kyungw00k/seleniumBase-go`
- **Release** (`release-please.yml`): Conventional Commit → auto changelog + GitHub Release
  - Initial version: `0.1.0`

## File Structure

```
mnemo/
├── cmd/mnemo/main.go
├── internal/
│   ├── config/config.go
│   ├── db/driver.go, postgres.go, sqlite.go, migrate.go
│   ├── migrations/embed.go            ← embed.FS wrapper
│   ├── repository/memory.go, note.go
│   ├── service/memory.go, note.go, embedding.go, extract.go
│   ├── mcp/server.go, tools_memory.go, tools_note.go, tools_db.go, tools_extra.go
│   └── transport/stdio.go, sse.go
├── migrations/
│   ├── postgres/001_initial.sql, 002_indexes.sql, 003_ttl.sql
│   └── sqlite/001_initial.sql, 002_indexes.sql, 003_ttl.sql
├── .github/workflows/ci.yml, release-please.yml
├── release-please-config.json
├── .release-please-manifest.json
├── Makefile
├── Dockerfile
├── .env.example
├── .gitignore
├── Plans.md      ← task tracking
├── CONTEXT.md    ← this file
└── README.md     (English; multilingual planned: zh/ja/ko)
```

## Resuming Work

1. Read this file (CONTEXT.md)
2. Read Plans.md for current task status
3. Check `git log --oneline -10` for recent commits
4. Continue from first `cc:TODO` task in Plans.md
