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

## ✅ Completed (Phases 1–13, 14, 15–16)

> Archived to [ARCHIVE.md](ARCHIVE.md). All tasks build-verified (`go build ./...` ✅).

**Summary of completed work:**
- Go binary: config, DB layer (SQLite/PostgreSQL), migrations, repositories, services
- 15 MCP tools: memory CRUD, note CRUD, db tools, export/import, auto-extract, TTL cleanup
- Dual transport: stdio + SSE with `/health` endpoint
- CI/CD: GitHub Actions (Go 1.22/1.23 matrix, CGO_ENABLED=1) + release-please (conventional commits)
- Docker: multi-platform image → `ghcr.io/kyungw00k/mnemo` (`:dev` on main, `:vX.Y.Z` on tag)
- Docs: README (EN), AGENT_INSTRUCTIONS.md.example (CLAUDE.md / AGENTS.md / .cursorrules)
- **Phase 14**: SQLite vector search via sqlite-vec (cosine similarity), FTS5 fallback

---

## Phase 14: sqlite-vec (Vector Search in SQLite) [cc:완료]

> Replace FTS5 keyword search with real cosine similarity in SQLite mode.
> CGO required at **build time only** — final output is still a single binary.
> macOS: `xcode-select --install` is sufficient. Linux: `gcc` + `-extldflags=-static`.

- [x] `cc:완료` **Task 35** — Replace `modernc.org/sqlite` → `mattn/go-sqlite3` + `sqlite-vec`
  - Add: `github.com/asg017/sqlite-vec-go-bindings/cgo`
  - Remove: `modernc.org/sqlite`
  - `internal/db/sqlite.go`: `init() { sqlite_vec.Auto() }`, driver name `"sqlite3"`

- [x] `cc:완료` **Task 36** — SQLite vector schema
  - `internal/migrations/sqlite/004_vec.sql`: `vec_memories` / `vec_notes` virtual tables (`vec0`)
  - `internal/db/migrate.go`: `$DIMENSIONS` replacement applies to both PostgreSQL and SQLite

- [x] `cc:완료` **Task 37** — SQLite vector repository
  - `internal/repository/memory.go`: `sqliteVectorSearch()` via `vec_memories`; `Upsert` writes to `vec_memories` when embedding != nil
  - `internal/repository/note.go`: same for `vec_notes`

- [x] `cc:완료` **Task 38** — Enable embedding for SQLite mode
  - Removed `isSQLite` guard in `service/memory.go` and `service/note.go`
  - Fallback: embedding unavailable → FTS5 (graceful degradation preserved)

- [x] `cc:완료` **Task 39** — Build pipeline updates
  - `Makefile`: `CGO_ENABLED=1`; added `build-linux-static` target
  - `Dockerfile`: `apk add gcc musl-dev` + `-ldflags="-extldflags=-static"` + `FROM scratch`
  - `.github/workflows/ci.yml`: `sudo apt-get install -y gcc`

- [x] `cc:완료` **Task 40** — Docs update
  - README: SQLite now supports vector search via sqlite-vec; build requirements table added

---

## Dependency Graph

```
Phase 14:
Task 35 → Task 36 → Task 37 → Task 38
                                  ↓
                            Task 39, Task 40
```
