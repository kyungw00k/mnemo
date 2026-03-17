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

## ✅ Completed (Phases 1–20)

> Archived to [ARCHIVE.md](ARCHIVE.md). All tasks build-verified (`go build ./...` ✅).

| Phase | 내용 |
|-------|------|
| 1–13 | Go binary, DB layer, 15 MCP tools, dual transport, CI/CD, Docker |
| 14 | SQLite vector search (sqlite-vec, cosine similarity) |
| 15–16 | Dashboard (React SPA), Docker multi-platform CI/CD |
| 17 | memory_search FTS5 버그 수정, 바이너리 재빌드 |
| 18 | CLI subcommand 체계 (hook/search/save/dashboard) |
| 19 | Auto-extract, Git context 자동화 |
| 20 | Memory 품질 개선: PreCompact hook, compact_snapshot, observation 노이즈 제거 |

---

## Dependency Graph

```
Phase 14:
Task 35 → Task 36 → Task 37 → Task 38
                                  ↓
                            Task 39, Task 40

Phase 17 (Level 1 — 버그 수정):
Task 41 → Task 42

Phase 18 (Level 2 — CLI Subcommand):
Task 43 (session-start hook)
Task 44 (session-end hook)    ← 병렬 구현 가능
Task 45 (observe hook)        ←
Task 46 (search/save CLI)     ←
Task 47 (dashboard)           ←
Task 48 (docs/config)         ← Task 43~47 완료 후

Phase 19 (Level 3 — 자동화):
Task 49 (auto-extract)  ← Task 44 선행 필요
Task 50 (git context)   ← Task 43~45 선행 필요
```
