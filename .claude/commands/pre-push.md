---
description: Run all pre-push checks (build, test, lint, typecheck) before pushing
allowed-tools: Bash(make:*), Bash(go:*), Bash(trunk:*), Bash(npx:*), Bash(npm:*), Bash(cd:*)
---

Run all quality checks before pushing. This replicates what CI will run.

## Steps

### Step 1: Build

```bash
make build
```

If build fails, report the error and stop.

### Step 2: Test

```bash
make test
```

If tests fail, report failures and stop.

### Step 3: Lint

```bash
make lint
```

If lint fails, try auto-fixing with `make fmt` and re-check.

### Step 4: TypeScript Check

```bash
make typecheck
```

If typecheck fails, report errors and stop.

### Step 5: Report

Report results:

```text
Pre-push checks:
  Build:     PASS/FAIL
  Test:      PASS/FAIL
  Lint:      PASS/FAIL
  Typecheck: PASS/FAIL

Verdict: READY TO PUSH / NOT READY
```
