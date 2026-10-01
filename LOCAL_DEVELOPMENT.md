# Local Development

## Prerequisites

- Go 1.26+, Node.js 22+
- Docker (for the devcontainer: Postgres 18 + Temporal), or a Postgres
  you can point `DATABASE_URL` at
- A Google OAuth 2.0 **Web application** client
  (<https://console.cloud.google.com/apis/credentials>) with
  `http://localhost:8000/api/auth/google/callback` as an authorized
  redirect URI
- Optional: a Claude API key for the AI date-parsing fallback

## 1. Configure

```bash
cp .env.sample backend/.env       # fill in Google client, allowlist, Claude key
echo 'NEXT_PUBLIC_API_URL=http://localhost:8000' > frontend/.env.local
```

`AUTH_ALLOWED_EMAILS` must contain the Google account you'll sign in with.

## 2. Database + Temporal

With Docker:

```bash
make dc-up        # Postgres on 5403, Temporal on 7234, Temporal UI on 8083
make dc-migrate
```

Without Docker, point `DATABASE_URL` at any Postgres and run:

```bash
go run ./backend/cmd/focus migrate
```

Temporal is only needed for the worker (nightly cleanup); the server runs
without it.

## 3. Run

```bash
make dc-backend    # server (8000) + worker with hot reload, inside the devcontainer
make dc-frontend   # Next.js on 3000
```

Or on the host:

```bash
go run ./backend/cmd/focus server
cd frontend && npm install && npm run dev
```

Open <http://localhost:3000>, sign in with Google, and you're in the inbox.

## 4. MCP from Claude Code

Create a token under **Settings → API tokens**, then:

```bash
claude mcp add --transport http focus http://localhost:8000/mcp \
  --header "Authorization: Bearer fcs_..."
```

## Tests

```bash
make test
```

Integration tests start an embedded Postgres automatically (binaries are
cached in `~/.cache/focus-embedded-pg-bin`). Use `TEST_DATABASE_URL` to
run them against an existing server, or `FOCUS_SKIP_DB_TESTS=1` to skip.

## Regenerating API types

Edit `api/openapi.yaml` (+ `paths/`, `schemas/`), then:

```bash
make generate
```

Requires `oapi-codegen`:
`go install github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@latest`.

## Environment variables

| Variable               | Purpose                                              |
| ---------------------- | ---------------------------------------------------- |
| `DATABASE_URL`         | Postgres connection string                           |
| `BACKEND_CORS_ORIGINS` | Allowed browser origins (comma-separated)            |
| `FRONTEND_URL`         | Where the API redirects after sign-in                |
| `API_URL`              | Public API URL (builds the Google redirect URL)      |
| `GOOGLE_CLIENT_ID`     | Google OAuth client ID                               |
| `GOOGLE_CLIENT_SECRET` | Google OAuth client secret                           |
| `AUTH_ALLOWED_EMAILS`  | Sign-in allowlist (comma-separated, case-insensitive)|
| `CSRF_AUTH_KEY`        | 32+ byte random secret for the CSRF middleware       |
| `CLAUDE_API_KEY`       | Optional; enables the AI date-parsing fallback       |
| `CLAUDE_MODEL`         | Optional; defaults to `claude-opus-5`                |
| `TEMPORAL_HOST`        | Temporal frontend host:port (worker only)            |
| `TEMPORAL_NAMESPACE`   | Temporal namespace (worker only)                     |
| `SESSION_EXPIRE_HOURS` | Session lifetime (default 720 = 30 days)             |
