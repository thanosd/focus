# Focus

A personal task manager in the spirit of OmniFocus, built for the Cosmic
Teacups Kubernetes cluster.

- **Inbox** for fast capture, then clean up later
- **Projects** nested up to three levels (bucket > project > sub-project), sequential or parallel, with
  **review mode** so nothing goes stale
- **Tags** that cut across projects
- **Flagged** (urgent) tasks that stand out
- **Defer** with quick buttons (+1 day / week / month) or free text
  ("next monday", "mid october", "in 3 days") — rules first, Claude as the
  fallback for anything unusual
- **Due dates** and **repeats** (from completion or from the due date)
- **Google sign-in** with an email allowlist
- **MCP server** so Claude Code / Claude Desktop can read and edit your tasks

## Quick links

- [Local development](LOCAL_DEVELOPMENT.md)
- [Architecture](docs/architecture.md) · [Data model](docs/data-model.md) ·
  [Security](docs/security.md) · [MCP](docs/mcp.md)
- [Production setup checklist](docs/deploy/setup-checklist.md) — every
  secret, DNS record and cloud resource that has to exist before the first
  deploy
- [Agent guidance](CLAUDE.md)

## At a glance

| Layer     | Choice                                                     |
| --------- | ---------------------------------------------------------- |
| Backend   | Go, `net/http`, Postgres via `lib/pq`, hexagonal layout    |
| Frontend  | Next.js 16 (App Router), React 18, Tailwind, openapi-fetch |
| Contract  | OpenAPI 3.1 in `api/` → generated Go + TS types            |
| Auth      | Google OIDC → server-side sessions (HttpOnly cookie)       |
| MCP       | Go MCP SDK, streamable HTTP at `/mcp`, bearer API tokens   |
| AI        | Claude Opus 5 via the Anthropic Go SDK (date parsing only) |
| Workflows | Temporal (nightly session/state cleanup)                   |
| Runtime   | EKS (`cosmic-cluster`), Helm chart in `k8s/focus/`         |
| IaC       | Pulumi in `infra/`                                         |
| Deploy    | GitHub Actions: push to `main` → build → `helm upgrade`    |
