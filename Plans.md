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

## ✅ Completed (Phases 1–16)

> Archived to [ARCHIVE.md](ARCHIVE.md). All tasks build-verified (`go build ./...` ✅).

| Phase | 내용 |
|-------|------|
| 1–13 | Go binary, DB layer, 15 MCP tools, dual transport, CI/CD, Docker |
| 14 | SQLite vector search (sqlite-vec, cosine similarity) |
| 15–16 | Dashboard (React SPA), Docker multi-platform CI/CD |

---

## Phase 17: Level 1 — memory_search 버그 수정 [cc:완료]

> 현재 실행 중인 `~/.local/bin/mnemo` 바이너리는 Phase 14 FTS5 제거 이전 빌드.
> `memory_search` 호출 시 `no such table: memories_fts` 에러 발생.
> 소스 코드는 이미 수정됨 — 재빌드·재설치로 해결.

- [x] `cc:완료` **Task 41** — 바이너리 재빌드 및 재설치
  - `make install-local` (`CGO_ENABLED=1` 포함)
  - 설치 경로: `~/.local/bin/mnemo`

- [x] `cc:완료` **Task 42** — memory_search 동작 검증
  - `mnemo search "test"` CLI 호출로 에러 없음 확인
  - `no such table: memories_fts` 에러 없음 확인
  - 벡터 검색(embedding 설정 시) 및 LIKE fallback 모두 검증

---

## Phase 18: Level 2 — CLI Subcommand 체계 구축 [cc:완료]

> 상시 실행 서버 없이 동작하는 아키텍처.
>
> - 평상시: `mnemo hook <event>` 가 DB에 직접 접근 (서버 불필요)
> - MCP: `TRANSPORT=stdio` — Claude Code가 세션 중 spawn, 종료 시 자동 소멸
> - 대시보드: `mnemo dashboard` — 필요할 때만 실행, 확인 후 Ctrl+C
>
> Python 훅 파일·REST API 추가·SSE 상시 실행 모두 불필요.
> 기존 `~/.claude/hooks/session-end.py` 는 Phase 18 완료 후 제거.

### 18-A: `mnemo hook` subcommand 구현

- [x] `cc:완료` **Task 43** — `mnemo hook session-start` 구현
  - 파일: `cmd/mnemo/main.go` (subcommand 라우팅), `internal/cli/hook.go` (신규)
  - stdin: Claude Code hook JSON `{"session_id", "transcript_path"}`
  - 동작: DB에서 최근 notes 3개 + memories(category=decision) 5개 조회
  - stdout: `{"additionalContext": "## Recent mnemo context\n..."}` (JSON)
  - DB 미존재 또는 접근 실패 시: `{}` 출력 (세션 차단 없음)
  - `~/.claude/settings.json` SessionStart 훅 등록:
    `"command": "mnemo hook session-start"`, `matcher: "startup|resume"`

- [x] `cc:완료` **Task 44** — `mnemo hook session-end` 구현
  - stdin: Claude Code Stop hook JSON `{"session_id", "transcript_path"}`
  - 동작: transcript에서 사용자 메시지 추출 → note_save (project 자동 감지)
  - note 형식: `title = "Session YYYY-MM-DD: <project>"`, `tags = ["session", "auto"]`
  - stdout: `{"continue": true}`
  - 기존 `~/.claude/hooks/session-end.py` 대체
  - `~/.claude/settings.json` Stop 훅을 `mnemo hook session-end` 로 교체

- [x] `cc:완료` **Task 45** — `mnemo hook observe` 구현
  - stdin: Claude Code PostToolUse hook JSON `{"tool_name", "tool_input", "session_id"}`
  - 추적 대상: `Edit`, `Write` (파일 경로 추출), `Bash` (빌드 성공 패턴 감지)
  - 동작: `memory_save(category="observation", key="edited:<path>", ...)` — DB 직접 쓰기
  - stdout: `{"continue": true}`
  - `~/.claude/settings.json` PostToolUse 훅 등록:
    `"command": "mnemo hook observe"`, `matcher: "Edit|Write|Bash"`

### 18-B: `mnemo search` / `mnemo save` CLI 구현

- [x] `cc:완료` **Task 46** — interactive CLI subcommand 구현
  - 파일: `internal/cli/search.go`, `internal/cli/save.go` (신규)
  - `mnemo search <query> [--limit N] [--category C]`
    - DB 벡터/LIKE 검색 → JSON 출력 (stdout)
    - Claude가 Bash tool로 호출 가능
  - `mnemo save <category> <key> <value> [--project P]`
    - DB에 memory upsert → `{"id": N, "ok": true}` 출력
  - 두 명령 모두 서버 불필요, DB 직접 접근

### 18-C: `mnemo dashboard` subcommand 구현

- [x] `cc:완료` **Task 47** — on-demand dashboard subcommand 구현
  - 파일: `internal/cli/dashboard.go` (신규)
  - `mnemo dashboard [--port 8765]`
    - SSE 서버 시작 (기존 `internal/transport/sse.go` 재사용)
    - 브라우저 자동 오픈 (`open http://localhost:<port>`)
    - Ctrl+C 로 종료
  - MCP SSE transport는 `TRANSPORT=sse` 환경변수로 여전히 사용 가능 (하위 호환)

### 18-D: MCP stdio 설정 및 문서 업데이트

- [x] `cc:완료` **Task 48** — MCP stdio 설정 및 문서 정리
  - `AGENT_INSTRUCTIONS.md.example` 업데이트:
    - MCP 설정을 stdio 방식으로 변경:
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
    - Hook 등록 예시 추가 (`session-start`, `session-end`, `observe`)
    - `mnemo dashboard` 사용법 추가
  - `README.md` Quick Start 업데이트: stdio 방식 우선 안내

---

## Phase 19: Level 3 — 자동화 강화 [cc:완료]

> mnemo opt-in 자동화 기능 활성화.
> Phase 18 완료 후 진행 (subcommand 체계 위에서 동작).

### 19-A: Auto-Extract — hook observe 연동

- [x] `cc:완료` **Task 49** — `mnemo hook session-end` 에 auto-extract 통합
  - `ENABLE_AUTO_EXTRACT=true` + `EXTRACT_LLM_*` 설정 시:
    세션 종료 시 transcript 전체를 LLM에 전달 → 핵심 사실 자동 추출 → memory_save
  - 설정 없으면 기존 단순 note_save만 수행 (graceful degradation)
  - `~/.mnemo/.env` 예시:
    ```
    ENABLE_AUTO_EXTRACT=true
    EXTRACT_LLM_BASE_URL=http://localhost:11434/v1
    EXTRACT_LLM_MODEL=qwen2.5:7b
    ```
  - 구현: `internal/cli/hook.go` `runSessionEnd()` + `internal/cli/init.go` `svcs.extractSvc`

### 19-B: Git Context 자동 태깅

- [x] `cc:완료` **Task 50** — `mnemo hook` 전체에 git context 통합
  - `ENABLE_GIT_CONTEXT=true` 설정 시:
    `session-start` / `session-end` / `observe` 모두 `git remote get-url origin` → `project` 자동 설정
  - 미설정 시 `git rev-parse --show-toplevel` 또는 cwd (기존 방식 유지)
  - 구현: `detectProjectForHook(cfg)` + `projectFromGitRemote()` in `internal/cli/hook.go`

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
