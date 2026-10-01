.PHONY: all fast format trunk generate types lint-frontend build test dc-up dc-down dc-recreate dc-frontend dc-backend dc-migrate dc-shell setup release draft-release

PROJECT_ROOT := $(shell cd "$(dir $(lastword $(MAKEFILE_LIST)))" && pwd)
DC_EXEC = npx --yes @devcontainers/cli exec --workspace-folder "$(PROJECT_ROOT)" bash -c

# ── CI / Host Targets (no devcontainer required) ──────────────────────
# These run directly on the host, matching what CI and pre-push hooks expect.

# Run all checks (format, lint, build, test, typecheck, eslint)
all: format trunk build test types lint-frontend

format:
	trunk fmt

# Same linters CI runs (osv-scanner included; it reads go.mod's toolchain line)
trunk:
	trunk check --all

# Build Go backend (host)
build:
	go build ./backend/...

# Run Go tests (host)
test:
	go test ./backend/...

# TypeScript type checking (host)
types:
	cd frontend && npm ci --silent && npx tsc --noEmit

# ESLint (host)
lint-frontend:
	cd frontend && npm run lint

# Fast gate: format + lint + types + build (~1 min)
fast: format trunk types build

# ── Setup ──────────────────────────────────────────────────────────────

# Configure git hooks and project settings
setup:
	git config core.hooksPath .githooks 2>/dev/null || true
	@echo "✓ Project setup complete"

# ── API Type Generation ───────────────────────────────────────────────
# Runs on the host: needs node (redocly + openapi-typescript via npx) and
# oapi-codegen (go install github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@latest).

generate:
	./api/generate-types.sh
	trunk fmt backend/internal/api/ api/openapi-bundled.yaml frontend/src/lib/api-types.ts

# ── Devcontainer Management ────────────────────────────────────────────

# Start the devcontainer (Postgres + Temporal)
dc-up:
	./scripts/devcontainer.sh up

# Stop and remove containers
dc-down:
	./scripts/devcontainer.sh down

# Recreate containers from scratch
dc-recreate:
	./scripts/devcontainer.sh recreate

# Run the frontend inside the devcontainer
dc-frontend:
	./scripts/devcontainer.sh frontend

# Run the backend (server + worker) inside the devcontainer
dc-backend:
	./scripts/devcontainer.sh backend

# Run database migrations inside the devcontainer
dc-migrate:
	./scripts/devcontainer.sh migrate

# Open an interactive shell inside the devcontainer
dc-shell:
	./scripts/devcontainer.sh shell

# ── Release Management ─────────────────────────────────────────────────

# Publish a release with auto-generated notes (triggers production deploy)
release:
	scripts/create-release.sh --publish

# Create a draft release for review before publishing
draft-release:
	scripts/create-release.sh
