# Security

Snapshot as of 2026-09-30. Update in place when things change.

## Authentication

- **Google only.** No passwords, no password column. The backend runs the
  OAuth 2.0 / OIDC authorization-code flow server-side; ID tokens are
  verified against Google's JWKS (`coreos/go-oidc`).
- **Allowlist.** `AUTH_ALLOWED_EMAILS` is the only gate: an account not on
  it is bounced to `/login?error=not_allowed` and nothing is written. The
  server refuses to start without the variable.
- **Sessions** are server-side rows referenced by an `HttpOnly`,
  `SameSite=Lax`, `Secure` (outside development) cookie; 30-day lifetime.
  Expired rows are deleted nightly by the Temporal maintenance workflow.
- **OAuth state** nonces live 10 minutes and are consumed atomically.

## API tokens (MCP)

- Created in Settings; prefixed `fcs_`; 256 bits of randomness.
- Only the SHA-256 hash is stored; the plaintext is shown once.
- Sent as `Authorization: Bearer` to `/mcp`; the endpoint is outside the
  cookie/CSRF middleware. Revocation is immediate (soft delete).
- Tokens carry the full rights of the user. There are no scopes yet.

## CSRF

`filippo.io/csrf/gorilla` on every `/api` route: Fetch-metadata
(`Sec-Fetch-Site`) and `Origin` checks against the CORS origins. Non-browser
clients without those headers are allowed, which is the designed behavior.

## Transport

TLS terminated at ingress-nginx with cert-manager (`letsencrypt-prod`).

## Secrets at rest

- AWS Secrets Manager (`cosmic/focus/production`), KMS-encrypted by default.
- Kubernetes `Secret` objects are base64 only (cluster-level encryption is
  the cluster stack's concern).
- RDS encryption is provisioned by the cluster stack.

## Data

Every query is scoped by `user_id`. There is no admin surface and no
cross-user access path.

## Third parties

- Google receives the sign-in; profile name/picture are stored.
- Anthropic receives date phrases (e.g. "after the launch") plus the current
  time and timezone — never task titles or notes.

## Not yet done

- Rate limiting on `/api/auth/google/login` and `/mcp`
- Scoped / read-only API tokens
- Audit log of MCP tool calls
