#!/bin/sh
set -eu

pids=""

cleanup() {
	for pid in ${pids}; do
		kill "${pid}" 2>/dev/null || true
	done
	wait 2>/dev/null || true
}

trap cleanup EXIT INT TERM

go run ./cmd/focus server &
pids="${pids} $!"

go run ./cmd/focus worker &
pids="${pids} $!"

wait
