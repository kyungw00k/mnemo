# mnemo — Implementation Plans

> Persistent MCP memory server for Claude Code & opencode
> Go binary · SQLite + PostgreSQL · OpenAI Compatible embeddings · Dual transport (stdio + SSE)

## Status Legend

| Marker | State |
|--------|-------|
| `cc:TODO` | Not started |
| `cc:WIP` | In progress |
| `cc:완료` | Done |

---

## Phase 1: Repository Initialization ✅

- [x] `cc:완료` **Task 1** — `go.mod` init, core dependencies added
- [x] `cc:완료` **Task 2** — `.gitignore` created

## Phase 2: Configuration Layer ✅

- [x] `cc:완료` **Task 3** — `internal/config/config.go`: all env vars + Phase 13 feature flags

## Phase 3: Database Layer ✅

- [x] `cc:완료` **Task 4** — `internal/db/driver.go`: unified `*sql.DB` interface
- [x] `cc:완료` **Task 5** — `internal/db/postgres.go`: pgxpool + stdlib adapter + pgvector
- [x] `cc:완료` **Task 6** — `internal/db/sqlite.go`: modernc.org/sqlite, WAL mode, CGO-free
- [x] `cc:완료` **Task 7** — `internal/db/migrate.go`: embed.FS-based migration runner
- [x] `cc:완료` **Task 8** — `migrations/postgres/001_initial.sql`
- [x] `cc:완료` **Task 9** — `migrations/postgres/002_indexes.sql` (HNSW)
- [x] `cc:완료` **Task 10** — `migrations/sqlite/001_initial.sql` (FTS5)
- [x] `cc:완료` **Task 11** — `migrations/sqlite/002_indexes.sql`

## Phase 4: Embedding Service ✅

- [x] `cc:완료` **Task 12** — `internal/service/embedding.go`: OpenAI Compatible `/v1/embeddings`

## Phase 5: Repository Layer ✅

- [x] `cc:완료` **Task 13** — `internal/repository/memory.go`: Upsert, VectorSearch, TextSearch, List, Categories, Delete, ExportAll, BulkUpsert, HardDeleteExpired
- [x] `cc:완료` **Task 14** — `internal/repository/note.go`: Insert, VectorSearch, TextSearch, List, Delete, ExportAll, BulkInsert, HardDeleteExpired

## Phase 6: Service Layer ✅

- [x] `cc:완료` **Task 15** — `internal/service/memory.go`: Save (TTL), Search, List, Categories, Delete, Cleanup, ExportAll, BulkImport
- [x] `cc:완료` **Task 16** — `internal/service/note.go`: Save (TTL), Search, List, Delete, Cleanup, ExportAll

## Phase 7: MCP Tools ✅

- [x] `cc:완료` **Task 17** — `internal/mcp/tools_memory.go`: memory_save, memory_search, memory_list, memory_delete, memory_categories
- [x] `cc:완료` **Task 18** — `internal/mcp/tools_note.go`: note_save, note_search, note_list, note_delete
- [x] `cc:완료` **Task 19** — `internal/mcp/tools_db.go`: db_query (SELECT+100row limit), db_execute (allowlist)
- [x] `cc:완료` **Task 20** — `internal/mcp/server.go`: mcp-go server, 11 base tools registered

## Phase 8: Transport Layer ✅

- [x] `cc:완료` **Task 21** — `internal/transport/stdio.go`
- [x] `cc:완료` **Task 22** — `internal/transport/sse.go` + `/health` endpoint

## Phase 9: Entry Point ✅

- [x] `cc:완료` **Task 23** — `cmd/mnemo/main.go`: config → DB → migrate → services → transports → graceful shutdown

## Phase 10: Build Infrastructure ✅

- [x] `cc:완료` **Task 24** — `Makefile`: build, test, lint, clean, run-stdio, run-sse, docker-build, docker-run, install
- [x] `cc:완료` **Task 25** — `Dockerfile`: golang:1.22-alpine → alpine multi-stage

## Phase 11: GitHub Actions ✅

- [x] `cc:완료` **Task 26** — `.github/workflows/ci.yml`: Go 1.22/1.23 matrix, gotestsum
- [x] `cc:완료` **Task 27** — `.github/workflows/release-please.yml`
- [x] `cc:완료` **Task 28** — `release-please-config.json`: go release type, conventional commit sections
- [x] `cc:완료` **Task 29** — `.release-please-manifest.json`: v0.1.0

## Phase 12: Documentation ✅

- [x] `cc:완료` **Task 30** — `README.md`: English, 6 badges, quickstart, config table, Claude Code + opencode guide

## Phase 13: Differentiation Features ✅

- [x] `cc:완료` **Task 31** — `memory_extract` tool (opt-in: `ENABLE_AUTO_EXTRACT=true`)
  - `internal/service/extract.go`: OpenAI Compatible chat completions → JSON extraction
  - `internal/mcp/tools_extra.go`: tool registered only when enabled
- [x] `cc:완료` **Task 32** — Git project auto-tagging (opt-in: `ENABLE_GIT_CONTEXT=true`)
  - Auto-detects git repo root → sets `project` field on memory_save/note_save
- [x] `cc:완료` **Task 33** — Memory TTL (opt-in: `MEMORY_TTL_DAYS=N`)
  - `migrations/postgres/003_ttl.sql`, `migrations/sqlite/003_ttl.sql`
  - `memory_cleanup` tool + background cleanup goroutine
- [x] `cc:완료` **Task 34** — Export/Import (always on)
  - `memory_export`: JSON or Markdown output
  - `memory_import`: bulk upsert from JSON

---

## Phase 14: sqlite-vec (Vector Search in SQLite) [cc:TODO]

> **Background**: SQLite's FTS5 is keyword-only. sqlite-vec adds real cosine similarity search to SQLite via CGO.
> CGO is needed at **build time only** — the final output is still a single binary.
> On macOS: `xcode-select --install` is sufficient (Apple Clang built-in).
> On Linux: `gcc` + `-ldflags="-extldflags=-static"` for fully static binary.

- [ ] `cc:TODO` **Task 35** — Replace `modernc.org/sqlite` with `mattn/go-sqlite3` + `sqlite-vec`
  - Add dependency: `github.com/asg017/sqlite-vec-go-bindings/cgo`
  - Remove: `modernc.org/sqlite`
  - Update `internal/db/sqlite.go`: register sqlite-vec extension on connection open
  - `CGO_ENABLED=1` required at build time

- [ ] `cc:TODO` **Task 36** — SQLite vector storage schema
  - Update `migrations/sqlite/001_initial.sql`:
    - Add `embedding BLOB` column to memories + notes (stores `[]float32` as raw bytes)
    - Add `vec_memories` and `vec_notes` virtual tables using `vec0`
  - New `migrations/sqlite/004_vec.sql`: create vec0 virtual tables (separate migration for upgraders)

- [ ] `cc:TODO` **Task 37** — SQLite vector repository implementation
  - `internal/repository/memory.go`: implement `VectorSearch` for SQLite using `vec_distance_cosine`
  - `internal/repository/note.go`: same
  - Keep FTS5 as fallback when embedding is nil

- [ ] `cc:TODO` **Task 38** — Embedding service: enable for SQLite mode
  - `internal/config/config.go`: remove SQLite-specific skip logic
  - `internal/service/memory.go` + `note.go`: call embedding even when SQLite
  - Fallback: if embedding unavailable → FTS5 (graceful degradation preserved)

- [ ] `cc:TODO` **Task 39** — Build pipeline updates for CGO
  - `Makefile`:
    - `build`: add `CGO_ENABLED=1`
    - `build-linux-static`: `CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build -ldflags="-extldflags=-static"`
  - `Dockerfile`: add `RUN apk add --no-cache gcc musl-dev` in builder stage, use `-ldflags="-extldflags=-static"`
  - `.github/workflows/ci.yml`: add `sudo apt-get install -y gcc` before build steps

- [ ] `cc:TODO` **Task 40** — README update
  - Update "SQLite" section: note that vector search is now supported natively
  - Add build requirements note (C compiler at build time)
  - Update feature comparison table

---

## Phase 15: AI Tool Integration Guide [cc:TODO]

> AI coding tools use different instruction files, but the content is identical.
> mnemo should provide a tool-agnostic example that works for all.

| Tool | Instruction File |
|------|-----------------|
| Claude Code | `CLAUDE.md` |
| OpenAI Codex CLI | `AGENTS.md` |
| opencode | `AGENTS.md` |
| Cursor | `.cursorrules` |
| GitHub Copilot | `.github/copilot-instructions.md` |

- [ ] `cc:TODO` **Task 41** — Create `AGENT_INSTRUCTIONS.md.example`
  - Tool-agnostic template users copy to `CLAUDE.md` / `AGENTS.md` / `.cursorrules` etc.
  - Sections:
    - **Session Start Protocol**: which tools to call first (`memory_search`, `memory_list`, `note_list`)
    - **What to Remember**: categories (decision, bug, convention, config, preference) + when to save
    - **Note-Taking Rules**: when to use notes vs memories, `project` field usage
    - **Session End Protocol**: how to summarize and save before ending
  - Include concrete examples for each MCP tool call

- [ ] `cc:TODO` **Task 42** — README: "AI Tool Integration" section
  - Explain `CLAUDE.md` vs `AGENTS.md` vs `.cursorrules` — same content, different filenames
  - Show the copy command: `cp AGENT_INSTRUCTIONS.md.example CLAUDE.md`
  - Per-tool subsections: Claude Code, Codex CLI, opencode, Cursor, Copilot
  - Include the full recommended instruction template inline

---

## Dependency Graph

```
Phase 1-12, 13: ✅ Complete — go build ./... passes

Phase 14 dependency order:
Task 35 (deps) → Task 36 (schema) → Task 37 (repo) → Task 38 (service)
                                                              ↓
                                                        Task 39 (build)
                                                        Task 40 (docs)

Phase 15: independent — can run any time
Task 41 (AGENT_INSTRUCTIONS.md.example) → Task 42 (README section)
```
