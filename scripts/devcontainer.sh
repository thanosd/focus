#!/usr/bin/env bash
# Devcontainer management script for focus.
# Usage:
#   ./scripts/devcontainer.sh up          Start the devcontainer
#   ./scripts/devcontainer.sh down        Stop the devcontainer
#   ./scripts/devcontainer.sh recreate    Recreate the devcontainer
#   ./scripts/devcontainer.sh frontend    Run the frontend (inside the container)
#   ./scripts/devcontainer.sh backend     Run the backend (inside the container)
#   ./scripts/devcontainer.sh migrate     Run database migrations (inside the container)
#   ./scripts/devcontainer.sh shell       Open an interactive shell
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
COMPOSE_PROJECT="$(basename "${PROJECT_ROOT}")_devcontainer"

# ── Helpers ──────────────────────────────────────────────────────────

info() { echo "▸ $*"; }
error() {
	echo "✖ $*" >&2
	exit 1
}

ensure_docker() {
	command -v docker >/dev/null 2>&1 || error "Docker is not installed or not in PATH"
	docker info >/dev/null 2>&1 || error "Docker daemon is not running"
}

# ── Container Lifecycle ──────────────────────────────────────────────

cmd_up() {
	ensure_docker
	info "Starting devcontainer…"
	npx --yes @devcontainers/cli up --workspace-folder "${PROJECT_ROOT}"
	info "Done! Container is running."
	info "  Run 'make dc-frontend' or 'make dc-backend' to start services."
}

cmd_down() {
	ensure_docker
	info "Stopping devcontainer…"
	docker compose -f "${PROJECT_ROOT}/.devcontainer/docker-compose.yml" --project-name "${COMPOSE_PROJECT}" down
	info "Done!"
}

cmd_recreate() {
	ensure_docker
	info "Recreating devcontainer (full rebuild)…"
	docker compose -f "${PROJECT_ROOT}/.devcontainer/docker-compose.yml" --project-name "${COMPOSE_PROJECT}" down -v
	npx --yes @devcontainers/cli up \
		--workspace-folder "${PROJECT_ROOT}" \
		--remove-existing-container \
		--build-no-cache
	info "Done! Container has been rebuilt from scratch."
	info "  Run 'make dc-frontend' or 'make dc-backend' to start services."
}

# ── Run Services (exec into the running container) ───────────────────

cmd_frontend() {
	ensure_docker
	info "Installing dependencies and starting frontend…"
	npx --yes @devcontainers/cli exec --workspace-folder "${PROJECT_ROOT}" \
		bash -c "cd /workspace/frontend && npm install && npx next dev -p 3000"
}

cmd_backend() {
	ensure_docker
	info "Starting backend with hot-reloading (air)…"
	npx --yes @devcontainers/cli exec --workspace-folder "${PROJECT_ROOT}" \
		bash -c "command -v air >/dev/null 2>&1 || go install github.com/air-verse/air@latest && cd /workspace/backend && air"
}

cmd_migrate() {
	ensure_docker
	info "Running database migrations…"
	npx --yes @devcontainers/cli exec --workspace-folder "${PROJECT_ROOT}" \
		bash -c "cd /workspace/backend && go run ./cmd/focus migrate"
}

cmd_shell() {
	ensure_docker
	info "Opening shell in devcontainer…"
	npx --yes @devcontainers/cli exec --workspace-folder "${PROJECT_ROOT}" \
		bash -c "cd /workspace && exec bash"
}

# ── Main ─────────────────────────────────────────────────────────────

usage() {
	cat <<EOF
Usage: $(basename "$0") <command>

Lifecycle:
  up          Start the devcontainer (Postgres + Temporal)
  down        Stop and remove containers
  recreate    Stop, remove volumes, and recreate

Run Services (inside the container):
  frontend    Start the Next.js dev server
  backend     Start the Go API server (with air hot-reload)
  migrate     Run database migrations
  shell       Open an interactive shell
EOF
}

case "${1-}" in
up) cmd_up ;;
down) cmd_down ;;
recreate) cmd_recreate ;;
frontend) cmd_frontend ;;
backend) cmd_backend ;;
migrate) cmd_migrate ;;
shell) cmd_shell ;;
-h | --help | help) usage ;;
*)
	usage
	exit 1
	;;
esac
