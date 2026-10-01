# Architecture

Focus is a single-tenant-per-user task manager: every row is scoped by
`user_id`, users sign in with Google, and only allowlisted emails get in.
The repo layout and tooling deliberately mirror `team-metrics` so the two
projects feel the same to work on.

## Stack

| Layer      | Choice                                                                  |
| ---------- | ----------------------------------------------------------------------- |
| Backend    | Go 1.26, `net/http` (Go 1.22+ method/path routing), `database/sql` + `lib/pq` |
| Frontend   | Next.js 16 (App Router), React 18, Tailwind 3, `openapi-fetch`          |
| Contract   | OpenAPI 3.1 (`api/`), generated with `oapi-codegen` and `openapi-typescript` |
| Auth       | Google OIDC (`coreos/go-oidc` + `golang.org/x/oauth2`), server-side sessions |
| CSRF       | `filippo.io/csrf/gorilla` (Fetch-metadata based)                        |
| AI         | Anthropic Go SDK, Claude Opus 5, only for natural-language date parsing |
| MCP        | `modelcontextprotocol/go-sdk`, streamable HTTP, bearer tokens           |
| Workflows  | Temporal (`go.temporal.io/sdk`) — one nightly maintenance workflow      |
| Database   | Postgres on the shared Cosmic RDS                                       |
| Runtime    | EKS `cosmic-cluster`, namespace `focus`, arm64 (Graviton) node pool     |
| IaC        | Pulumi (`infra/`), Helm (`k8s/focus/`)                                   |
| CI/CD      | GitHub Actions; push to `main` → trunk/go/ts checks → ECR → `helm upgrade` |

## Repository layout

```text
api/        OpenAPI spec. openapi.yaml references paths/*.yaml and schemas/*.yaml.
backend/    Go module root is the repo root (go.mod); code lives in backend/.
frontend/   The app. src/lib/api-types.ts is generated — never edit.
infra/      Pulumi: ECR, deploy role, Secrets Manager entry, namespace, Route 53.
k8s/        Helm chart (focus/) + Temporal install script and values template.
deploy/     bootstrap-db.sh — creates the RDS databases/roles once.
docs/       This directory.
```

## Backend (hexagonal)

Under `backend/internal/`:

- `domain/` — pure types and sentinel errors (`ErrNotFound`, `ErrValidation`)
- `ports/` — repository interfaces
- `adapters/` — Postgres implementations; `integration_test.go` runs the
  real SQL against an embedded Postgres
- `services/` — `AuthService` (Google sign-in, sessions, API tokens),
  `TaskService` (capture, edit, complete + repeat, defer), `ProjectService`
  (nesting, status, reviews), `TagService`; `services/dateparse/` is the
  natural-language date parser (rules → Claude fallback)
- `handlers/` — HTTP handlers, one per resource family, plus `router.go`
- `mcpserver/` — MCP tools that call the same services
- `temporal/` — `MaintenanceWorkflow` (delete expired sessions and OAuth
  states), its activities, the schedule manager and the worker entrypoint
- `database/` — migrator over `backend/migrations/*.sql`
- `testdb/` — embedded-Postgres harness used by the integration tests

### CLI

`backend/cmd/focus` multiplexes:

- `server` — HTTP API + `/mcp`
- `worker` — Temporal worker (registers the nightly schedule on start)
- `migrate` — apply pending migrations (run as a Helm pre-upgrade Job)

`config.MustHave(...)` at the top of each subcommand fails fast on missing
env vars (the AgencyHQ rule).

## Request flow

```text
browser ──cookie──▶ CORS → CSRF → RequireSession → handler → service → repo
Claude  ──bearer──▶ /mcp (auth.RequireBearerToken) → tool → service → repo
```

Session cookies are host-only on `focus-api.cosmicteacups.com`; the app at
`focus.cosmicteacups.com` is same-site, so `SameSite=Lax` + `credentials:
"include"` works without a cookie domain.

### Google sign-in

1. Frontend navigates to `GET /api/auth/google/login?redirect=/inbox`.
2. Backend stores a state nonce (`oauth_states`, 10 min) and redirects to
   Google with scopes `openid email profile`.
3. Google calls `GET /api/auth/google/callback?state&code`.
4. Backend consumes the nonce, exchanges the code, verifies the ID token,
   checks `email_verified` and `AUTH_ALLOWED_EMAILS`, upserts the user,
   creates a 30-day session, sets the cookie, redirects to
   `FRONTEND_URL + redirect`. Failures redirect to `/login?error=<code>`.

### Natural-language dates

`POST /api/tasks/{id}/defer {input}` and `POST /api/dates/parse` go through
`dateparse.Parser`: `ParseRules` handles the everyday vocabulary
deterministically (relative offsets, weekdays, month/day, "mid october",
"eod", clock times…); only phrases it rejects go to Claude, which returns a
JSON `{datetime, interpretation}` that is validated before use. Without
`CLAUDE_API_KEY` the fallback is disabled and the API returns 400 with the
phrase it couldn't understand.

## Frontend

Routes under `frontend/src/app/`: `/login`, `/inbox`, `/projects`,
`/projects/[id]`, `/tags`, `/tags/[id]`, `/flagged`, `/review`, `/settings`.
`RequireAuth` gates app pages; `AuthContext` holds the user (and timezone);
`CountsContext` refreshes the sidebar badges after mutations. All API calls
go through the typed `apiClient` (ESLint bans raw `fetch`).

## Deploy

`deploy-production.yml` (push to `main`): trunk + Go + TS checks → build
multi-arch images (`focus-backend`, `focus-frontend`) → verify the arm64
binary is really aarch64 → fetch `cosmic/focus/production` from Secrets
Manager, camelCase the keys into Helm values → `helm upgrade --atomic`
(pre-upgrade hook runs `focus migrate`) → wait for rollouts.

There is no staging environment. Temporal is installed separately with
`k8s/install-temporal.sh` into the `focus-temporal` namespace.
