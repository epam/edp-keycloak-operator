---
name: run-tests
description: "Run tests for the edp-keycloak-operator. By default runs unit + integration tests. Use /run-tests e2e for end-to-end tests."
argument-hint: "[unit|integration|e2e]"
allowed-tools: Bash(make *), Bash(docker *), Read
---

## Your task

Dispatch based on `$ARGUMENTS`:

- **no argument** (default): run unit + integration tests
- **`unit`**: run unit tests only
- **`integration`**: run unit + integration tests
- **`e2e`**: run e2e tests only

## Unit Tests

```
make test
```

No external dependencies. `TEST_KEYCLOAK_URL` must NOT be set — integration tests will be skipped automatically.

## Integration Tests

Requires a running Keycloak instance on port 8086.

1. Check if Keycloak is running:
   ```
   docker ps --filter name=keycloak-test --format '{{.Names}}'
   ```
2. If not running, start it and wait ~10 seconds for initialization:
   ```
   make start-keycloak
   ```
3. Run unit + integration tests:
   ```
   TEST_KEYCLOAK_URL=http://localhost:8086 make test
   ```

## E2E Tests

Long-running. Requires kind cluster, kuttl, and Docker.

Prerequisites — verify before running:
- kind cluster is running (`make start-kind` if not)
- CRDs are installed (`make install` if not)

```
make e2e
```

## Interpreting Results

- Coverage report is written to `coverage.out`
- `PASS` with coverage summary = all tests passed
- `FAIL` lines indicate failures — read the error output carefully

## On Failure

- Do NOT retry the same failing test blindly
- Isolate failures to a specific package: `go test -v ./pkg/client/keycloakv2/...`
- Investigate the root cause before making changes
- After fixing, re-run the full suite to confirm no regressions
