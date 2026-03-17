# mnemo — Completed Tasks Archive

> Archived from Plans.md. All phases verified with `go build ./... ✅`

## Phase 1–13: Core Implementation ✅

| Phase | Description | Key Files |
|-------|-------------|-----------|
| 1 | Repo init | `go.mod`, `.gitignore` |
| 2 | Config | `internal/config/config.go` |
| 3 | DB layer | `internal/db/{driver,postgres,sqlite,migrate}.go` |
| 4 | Embedding | `internal/service/embedding.go` (OpenAI Compatible) |
| 5 | Repository | `internal/repository/{memory,note}.go` |
| 6 | Service | `internal/service/{memory,note}.go` |
| 7 | MCP tools (11) | `internal/mcp/tools_{memory,note,db}.go` |
| 8 | Transport | `internal/transport/{stdio,sse}.go` |
| 9 | Entry point | `cmd/mnemo/main.go` |
| 10 | Build infra | `Makefile`, `Dockerfile` |
| 11 | GitHub Actions | `ci.yml`, `release-please.yml` |
| 12 | README | `README.md` (English, 6 badges) |
| 13 | Differentiation | `tools_extra.go`, `service/extract.go`, TTL migrations |

### Phase 13 Feature Flags
| Feature | Env Var | Default |
|---------|---------|---------|
| Auto memory extraction | `ENABLE_AUTO_EXTRACT` | `false` |
| Git project tagging | `ENABLE_GIT_CONTEXT` | `false` |
| Memory TTL | `MEMORY_TTL_DAYS` | `0` (off) |
| Export/Import | always on | — |

## Phase 15: AI Tool Integration Guide ✅

- `AGENT_INSTRUCTIONS.md.example` — works for CLAUDE.md / AGENTS.md / .cursorrules / copilot-instructions.md
- README: "AI Tool Integration Guide" section with per-tool copy commands

## Phase 16: Docker CI/CD ✅

- `.github/workflows/docker.yml` — multi-platform (amd64/arm64), GHCR
- Tags: `:dev` on main push, `:vX.Y.Z` + `:latest` on release tag

## Phase 14: sqlite-vec (Vector Search in SQLite) ✅

> Replaced FTS5 keyword search with cosine similarity via sqlite-vec.

- Task 35: `modernc.org/sqlite` → `mattn/go-sqlite3` + `sqlite-vec-go-bindings/cgo`
- Task 36: `internal/migrations/sqlite/004_vec.sql` — `vec_memories`, `vec_notes` vec0 tables
- Task 37: `sqliteVectorSearch()` in `memory.go` / `note.go`
- Task 38: Removed `isSQLite` embedding guard — both DBs attempt embedding
- Task 39: `Makefile` CGO_ENABLED=1, static Linux build, CI gcc install
- Task 40: README updated with sqlite-vec vector search docs
