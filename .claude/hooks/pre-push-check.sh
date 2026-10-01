#!/bin/bash
# Pre-push hook: runs make all before allowing git push
# Input: JSON on stdin with tool_input.command

COMMAND=$(jq -r '.tool_input.command // empty')

# Only gate on git push commands
if [[ ${COMMAND} == *"git push"* ]]; then
	REPO_ROOT="$(git rev-parse --show-toplevel 2>/dev/null)"
	if [[ -z ${REPO_ROOT} ]]; then
		exit 0
	fi

	# Run make all
	if ! make -C "${REPO_ROOT}" all >/dev/null 2>&1; then
		jq -n '{
      hookSpecificOutput: {
        hookEventName: "PreToolUse",
        permissionDecision: "deny",
        permissionDecisionReason: "make all failed — fix build/test/lint/typecheck errors before pushing"
      }
    }'
		exit 0
	fi
fi

# Allow everything else
exit 0
