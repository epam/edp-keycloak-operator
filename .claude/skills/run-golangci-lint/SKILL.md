---
name: run-golangci-lint
description: "Run golangci-lint linters and fix any linting errors. Use when fixing lint issues or before committing code."
allowed-tools: Bash(make *), Read, Grep, Glob
---

## Task

Run `make lint-fix` to auto-fix linting errors across the entire project, then fix any remaining errors manually.

## Workflow

1. Run `make lint-fix`
2. Review the output — note what was auto-fixed and what errors remain
3. Fix remaining errors manually (see guidelines below)
4. Run `make lint-fix` to confirm zero errors
5. Run the `run-tests` skill to confirm no regressions — only if the lint fixes involved significant code changes (e.g. refactoring logic, splitting functions). Skip for trivial fixes like renaming, formatting, or comment edits.

## Fix Guidelines

**Line length (`lll`)**: Move each function parameter or return value to its own line.

**Duplicate code (`dupl`)**: First try to refactor the test to eliminate duplication (extract helpers, use table-driven tests). If the test becomes significantly harder to read after refactoring, add `//nolint:dupl // test helper duplication aids readability`. Note: `dupl` is excluded from `api/*` and `internal/*`.

**Other errors**: Fix the root cause directly. Do not add `//nolint` unless the false-positive is clear and add a comment explaining why.

**Complex / subjective issues**: Explain the trade-offs and ask for user approval before making changes.

## Iteration

If fixing one error introduces new errors, address them in order of severity. Re-run `make lint-fix` after each manual pass. Continue until `make lint-fix` exits cleanly and the `run-tests` skill passes.
