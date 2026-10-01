# API Type Generation

The OpenAPI 3.1 spec in this directory is the contract between the Go backend
and the Next.js frontend. Types are generated for both sides so a renamed
field breaks the build instead of a user session.

## Layout

```text
api/
├── openapi.yaml              # Main spec (references paths/ and schemas/)
├── openapi-bundled.yaml      # Auto-generated bundle (all refs resolved)
├── generate-types.sh         # Regenerates everything
├── paths/                    # One file per resource family
└── schemas/                  # Shared schemas
```

## Regenerating

```bash
make generate
```

That bundles the spec (`@redocly/cli`), writes
`frontend/src/lib/api-types.ts` (`openapi-typescript`) and
`backend/internal/api/types.go` (`oapi-codegen`), then formats the output.

`oapi-codegen` is a Go tool:

```bash
go install github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@latest
```

## Rules

- Every endpoint the frontend calls MUST be in the spec. ESLint bans raw
  `fetch()` in the frontend; use `apiClient` from `@/lib/api-client`.
- Add the path in `paths/`, schemas in `schemas/`, wire both into
  `openapi.yaml`, run `make generate`, then write the handler against the
  generated `api.*` types.
- Errors always use `schemas/common.yaml#/Error` (`{"message": "..."}`).
