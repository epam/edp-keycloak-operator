---
name: cr-field
description: "Use when adding a new field to an existing Custom Resource. Guides the full workflow: API types, code generation, CRD examples, client investigation, handler mapping, and tests."
allowed-tools: Bash(make *), Read, Grep, Glob
---

## Your task

Add a new field to an existing Custom Resource following the project's established conventions.
Work through all steps in order. Do not skip steps.

---

## Step 1 — Add field to API types

File: `api/v1/{kind}_types.go` (or `api/v1alpha1/` for alpha resources).

Add the struct field with a godoc comment and kubebuilder markers. Choose markers that match the field's semantics:

- `// +required` / `// +optional`
- `// +nullable` — for pointer slices or maps that can be explicitly null
- `// +kubebuilder:validation:Enum=val1;val2` — restrict to allowed values
- `// +kubebuilder:default=value` — set a default when field is omitted
- `// +kubebuilder:example=value` — example shown in generated docs
- `// +kubebuilder:validation:XValidation:rule=...,message=...` — CEL validation (e.g. immutability)

JSON tag: `json:"fieldName,omitempty"` for optional fields, `json:"fieldName"` for required.

See `api/v1/keycloakclient_types.go` for the full variety of marker patterns in use.

---

## Step 2 — Run code generation

```
make generate && make manifests
```

This regenerates DeepCopy methods and CRD YAMLs in `config/crd/bases/` and `deploy-templates/crds/`.

---

## Step 3 — Update CRD examples

Add the new field with a meaningful example value to:
- `config/samples/v1_v1_{kind}.yaml`
- `deploy-templates/_crd_examples/{kind}.yaml`

---

## Step 4 — Identify the client and find the Keycloak representation type

This is the critical investigation step before touching any handler code.

### 4a. Which client does this controller use?

Read `internal/controller/{resource}/chain/chain.go` and the relevant handler file and check the imports:
- `pkg/client/keycloak/` → **legacy gocloak client**
- `pkg/client/keycloakv2/` → **new keycloakv2 client**

### 4b. Find the Keycloak representation struct

**keycloakv2**: Check `pkg/client/keycloakv2/contracts.go` for the relevant client interface
(`GroupsClient`, `ClientsClient`, `RolesClient`, etc.).
Representation types are type aliases — grep `pkg/client/keycloakv2/generated/client_generated.go`
for the struct name (e.g. `GroupRepresentation`, `ClientRepresentation`, `RoleRepresentation`)
to see which fields are available.

**legacy gocloak**: The representation structs live in the `gocloak` package
(`gocloak.Group`, `gocloak.Client`, etc.). Read the handler to see which struct is being built.

### 4c. Confirm the field exists — then follow this decision tree

- **keycloakv2 + field exists** → proceed to Step 5.
- **legacy gocloak + field exists in gocloak struct** → map the field in the legacy handler and proceed to Step 5. Note: this controller is also a candidate for keycloakv2 migration per project conventions.
- **legacy gocloak + field missing from gocloak** → check if the field exists in the keycloakv2 representation. If yes, **propose migrating the controller to keycloakv2** as the path to support this field. Use the `KeycloakRealmGroup` controller migration as the reference pattern. Do not patch the legacy client.
- **Field missing from both clients** → **stop**. Warn the user that the field is not exposed by either client. `openapi.yaml` is auto-generated and must not be edited manually. Ask the user how to proceed before making any further changes.

---

## Step 5 — Map the field in the chain handler

In `internal/controller/{resource}/chain/` find the handler that creates/updates the resource
(usually `create_or_update_{resource}.go` or `put_{resource}.go`).

Map the field in **both** paths:
- **Create path**: include the field when building the representation struct before the Create call.
- **Update path**: assign the field on the fetched existing representation before the Update call.

Reference: `internal/controller/keycloakrealmgroup/chain/create_or_update_group.go`

---

## Step 6 — Update unit tests

File: `internal/controller/{resource}/chain/*_test.go`

Add the field to the spec setup and to the mock expectations for all test cases:
create path, update path, and error paths.

Reference: `internal/controller/keycloakrealmgroup/chain/create_or_update_group_test.go`

---

## Step 7 — Update integration tests

File: `internal/controller/{resource}/*_controller_integration_test.go`

Add the field to the CR creation spec. Assert the correct value is persisted in Keycloak
using `Eventually()` + `g.Expect()`.

Reference: `internal/controller/keycloakrealmgroup/keycloakrealmgroup_controller_integration_test.go`

---

## Step 8 — Validate

Invoke the `run-golangci-lint` skill, then the `run-tests` skill.
