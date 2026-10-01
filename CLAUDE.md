# Focus

Personal task manager in the spirit of OmniFocus: inbox capture, projects
nested two levels, tags, flagged (urgent) tasks, deferral with
natural-language dates, due dates, repeats, and a periodic review mode.
Google sign-in (email allowlist) and an MCP server so Claude can use it.

## CRITICAL: Pre-Push Checklist

**Before pushing ANY code, you MUST run these checks:**

```bash
trunk fmt --all          # Auto-format all files
trunk check --all        # Lint check — must pass with zero issues
```

If trunk check fails, fix the issues before pushing. Never push code that fails trunk checks.

## Stack

- **Backend**: Go (in `backend/`), `net/http` + `database/sql` + `lib/pq`, hexagonal layout
- **Frontend**: Next.js / TypeScript (in `frontend/`)
- **MCP server**: Go MCP SDK, streamable HTTP at `/mcp` on the backend
- **AI**: Claude (`CLAUDE_API_KEY`) as the fallback natural-language date parser
- **Workflows**: Temporal (nightly maintenance) in `backend/internal/temporal/`
- **Infrastructure**: Pulumi / TypeScript (in `infra/`), Helm chart in `k8s/focus/`
- **Linting**: Trunk (`.trunk/trunk.yaml`)
- **CI**: GitHub Actions (`.github/workflows/ci.yml`); merge to `main` deploys production

## Definition of Done

Before pushing, run:

```bash
make all
```

This runs: `format`, `trunk`, `build`, `test`, `types`, `lint-frontend` on the host.

`make test` runs Go integration tests against an embedded Postgres (first run
downloads the binaries into `~/.cache/focus-embedded-pg-bin`). Set
`TEST_DATABASE_URL` to use an existing database instead, or
`FOCUS_SKIP_DB_TESTS=1` to skip them.

## OpenAPI-First

Every endpoint the frontend calls MUST be in `api/openapi.yaml` BEFORE
handler or client code. Run `make generate` to regenerate
`backend/internal/api/types.go` and `frontend/src/lib/api-types.ts`.
ESLint bans raw `fetch()` in the frontend — use `apiClient`.

## Fail Fast on Missing Required Config

Required env vars are validated at startup via `config.MustHave(...)` in
`backend/cmd/focus/main.go`, not at first use. When you add a new required
env var, add it there AND to the Helm chart (`k8s/focus/templates/secrets.yaml`

- the deployment env block) AND to `docs/deploy/setup-checklist.md`.

## Key Commands

| Command              | Description                                         |
| -------------------- | --------------------------------------------------- |
| `make all`           | Run all checks (format, lint, build, test, types)   |
| `make fast`          | Fast gate: format + lint + types + build            |
| `make build`         | Build Go backend                                    |
| `make test`          | Run Go tests (unit + embedded-Postgres integration) |
| `make types`         | TypeScript type check                               |
| `make lint-frontend` | ESLint                                              |
| `make generate`      | Regenerate API types from the OpenAPI spec          |
| `make dc-up`         | Start devcontainer (Postgres + Temporal)            |
| `make dc-migrate`    | Run database migrations in devcontainer             |
| `make dc-backend`    | Run server + worker with hot reload in devcontainer |
| `make dc-frontend`   | Run the Next.js dev server in devcontainer          |

## Project Structure

```text
focus/
├── api/              # OpenAPI spec (source of truth for API types)
├── backend/          # Go backend
│   ├── cmd/focus/    # Entry point: server | worker | migrate | import-omnifocus
│   ├── internal/
│   │   ├── domain/      # Pure types (Task, Project, Tag, RepeatRule…)
│   │   ├── ports/       # Repository interfaces
│   │   ├── adapters/    # Postgres repositories (+ integration tests)
│   │   ├── services/    # Business logic; services/dateparse = NL dates
│   │   ├── handlers/    # HTTP handlers + router
│   │   ├── mcpserver/   # MCP tools over streamable HTTP
│   │   ├── importer/    # OmniFocus CSV import (focus import-omnifocus)
│   │   └── temporal/    # Workflows, activities, schedules, worker
│   └── migrations/   # Numbered SQL migrations
├── frontend/         # Next.js app
├── infra/            # Pulumi (ECR, deploy role, secret, namespace, DNS)
├── k8s/              # Helm chart + Temporal install
├── deploy/           # One-shot DB bootstrap
└── docs/             # Architecture, data model, security, MCP, deploy checklist
```

## Domain rules worth knowing

- A task with `project_id = NULL` is in the **inbox**.
- **Available** = active, `defer_until` is null or past, project is active, and
  for sequential projects it is the first active task in sort order. Computed
  in SQL (`backend/internal/adapters/postgres_tasks.go`) so every view agrees.
- Projects nest **two levels max**; the service rejects deeper parents.
- Completing a task with a `repeat_rule` creates the next occurrence
  (`from: completion` shifts from now; `from: due` shifts the previous dates).
- Reviews: `next_review_at <= now` on an active/on-hold project means due.
- Dates: `dateparse.ParseRules` first ("1w", "next monday", "mid october",
  "oct 5 at 3pm"…), Claude only for phrases the rules reject.
