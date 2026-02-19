---
name: generate-mocks
description: "Use when generating testify mocks for an interface. Ensures the interface is registered in .mockery.yml, runs make mocks, and verifies the output file."
allowed-tools: Read, Grep, Glob, Bash(make *)
---

## Your task

Generate mocks for one or more interfaces. `$ARGUMENTS` contains the interface name(s) to add/verify (may be empty — in that case just regenerate all existing mocks).

Work through all steps in order.

---

## Step 1 — Read .mockery.yml

Read `.mockery.yml` to understand the current package/interface registrations.

Key structure:
```yaml
packages:
  github.com/epam/edp-keycloak-operator/<pkg-path>:
    interfaces:
      InterfaceName: {}
```

Generated filename pattern: `{{ .InterfaceName | snakecase }}_generated.mock.go`
Output directory: `<package_dir>/mocks/`

---

## Step 2 — Locate the interface in source (skip if `$ARGUMENTS` is empty)

Use Grep to find `type <InterfaceName> interface` in `pkg/` and `internal/`.

Derive the Go import path from the file path:
- Use the **directory** of the file (not the file itself)
- Prepend `github.com/epam/edp-keycloak-operator/`
- Example: `pkg/client/keycloakv2/contracts.go` → package `github.com/epam/edp-keycloak-operator/pkg/client/keycloakv2`

---

## Step 3 — Update .mockery.yml if needed (skip if `$ARGUMENTS` is empty)

Use **targeted Edit** (never rewrite the whole file) to add only what is missing:

- Interface already registered → nothing to do, skip to Step 4.
- Package present, interface missing → insert `      InterfaceName: {}` under its `interfaces:` block.
- Package not present → append a new package entry at the end of the `packages:` block.

---

## Step 4 — Run make mocks

```
make mocks
```

---

## Step 5 — Verify output

Use Glob to confirm the generated file exists at the expected path.

Expected path: `<package_filesystem_dir>/mocks/<interface_name_snakecase>_generated.mock.go`

Example: `UsersClient` in `pkg/client/keycloakv2/` → `pkg/client/keycloakv2/mocks/users_client_generated.mock.go`

Report the verified path to the user.
