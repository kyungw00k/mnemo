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

## ✅ Completed (Phases 1–13, 15–16)

> Archived to [ARCHIVE.md](ARCHIVE.md). All tasks build-verified (`go build ./...` ✅).

**Summary of completed work:**
- Go binary: config, DB layer (SQLite/PostgreSQL), migrations, repositories, services
- 15 MCP tools: memory CRUD, note CRUD, db tools, export/import, auto-extract, TTL cleanup
- Dual transport: stdio + SSE with `/health` endpoint
- CI/CD: GitHub Actions (Go 1.22/1.23 matrix) + release-please (conventional commits)
- Docker: multi-platform image → `ghcr.io/kyungw00k/mnemo` (`:dev` on main, `:vX.Y.Z` on tag)
- Docs: README (EN), AGENT_INSTRUCTIONS.md.example (CLAUDE.md / AGENTS.md / .cursorrules)

---

## Phase 14: sqlite-vec (Vector Search in SQLite) [cc:TODO]

> Replace FTS5 keyword search with real cosine similarity in SQLite mode.
> CGO required at **build time only** — final output is still a single binary.
> macOS: `xcode-select --install` is sufficient. Linux: `gcc` + `-extldflags=-static`.

- [ ] `cc:TODO` **Task 35** — Replace `modernc.org/sqlite` → `mattn/go-sqlite3` + `sqlite-vec`
  - Add: `github.com/asg017/sqlite-vec-go-bindings/cgo`
  - Remove: `modernc.org/sqlite`
  - `internal/db/sqlite.go`: register sqlite-vec extension on open

- [ ] `cc:TODO` **Task 36** — SQLite vector schema
  - `migrations/sqlite/001_initial.sql`: add `embedding BLOB` to memories + notes
  - `migrations/sqlite/004_vec.sql`: `vec_memories` / `vec_notes` virtual tables (`vec0`)

- [ ] `cc:TODO` **Task 37** — SQLite vector repository
  - `internal/repository/memory.go`: `VectorSearch` via `vec_distance_cosine`
  - `internal/repository/note.go`: same; FTS5 fallback when embedding is nil

- [ ] `cc:TODO` **Task 38** — Enable embedding for SQLite mode
  - Remove SQLite-specific skip logic in config/service
  - Fallback: embedding unavailable → FTS5 (graceful degradation preserved)

- [ ] `cc:TODO` **Task 39** — Build pipeline updates
  - `Makefile`: `CGO_ENABLED=1`; add `build-linux-static` target
  - `Dockerfile`: `apk add gcc musl-dev` + `-ldflags="-extldflags=-static"`
  - `.github/workflows/ci.yml`: `sudo apt-get install -y gcc`
  - `.github/workflows/docker.yml`: add gcc to builder stage

- [ ] `cc:TODO` **Task 40** — Docs update
  - README: SQLite now supports vector search; build requirements (C compiler)
  - Update feature comparison table

---

## Dependency Graph

```
Phase 14:
Task 35 → Task 36 → Task 37 → Task 38
                                  ↓
                            Task 39, Task 40
```
