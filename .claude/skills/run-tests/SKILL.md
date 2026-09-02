---
name: run-tests
description: "Run tests for the edp-keycloak-operator. By default runs unit + integration tests. Use /run-tests e2e for end-to-end tests."
argument-hint: "[unit|integration|e2e]"
allowed-tools: Bash(make *), Bash(docker *), Bash(curl *), Read
context: fork
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
2. If not running, start it, then wait until it answers on port 8086:
   ```
   make start-keycloak
   until curl -sf http://localhost:8086/realms/master >/dev/null; do sleep 2; done
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

## On Failure

- Isolate failures to a specific package: `go test -v ./pkg/client/keycloakapi/...`
- After fixing, re-run the full suite to confirm no regressions
