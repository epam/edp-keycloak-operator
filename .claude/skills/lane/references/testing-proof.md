# Tests that prove something

## Design from a table

- Write the **case table** before coding a handler: empty name, duplicates, self reference, dependency listed
  later, missing dependency, cycle, existing object, failure at every write step, every event emitted and its
  predicate effect, unreachable branches. Derive the tests from it; re-run the whole table after every fix.
- A new status field is a state machine: set on, cleared on, stale when, consumer contract. Derive unit tests
  and cluster scenarios from that table, not from the implementation.
- A bug found mid-implementation: first ask whether ordering or structure removes it, then patch.
- Hand-maintained files and tables need a test that ties them to their source of truth (generated manifests,
  Keycloak metadata), or they drift silently.
- Timing bounds come from the minimum guaranteed count, not from the observed run. Keep `Consistently`
  windows well below the first backoff and `Eventually` timeouts above the next one.

## Red on old code

- Every new test runs against the old code and fails there. Swap the file in from the base commit, or remove
  the new term, run, restore. A test that passes on old code does not test the bug.
- When the new test cannot compile against the old code, run it against a shim that keeps the old behaviour
  behind the new signature.
- Rework commits: "old code" is `pre-rework-head` from `state.md`, not `<base>`.
- A value lost in several layers (converter, the operator's own full-object `Update`, a defaulting webhook)
  passes unit tests and fails on the cluster with a one-layer fix. Prove each layer with a variant that fixes
  only that layer.
- A test for a forced write uses a value the plain compare cannot repair (a masked key); prove red by deleting
  the force term.
- A test about two objects colliding on one lookup creates them in sequence: the second only after the first
  holds its ID. Created together, both may find nothing and old code passes.
- A test rewritten to drop a removed API keeps each assertion able to fail: capture an identity once, never
  refresh it in a helper that later steps call, or the check compares a value to itself.
- A test that cannot go red is dropped and reported.

## Clients and fakes

- Typed clients drop `[]` on plain slices at marshal: create the CR as unstructured for old-code red runs.
  The fake client JSON round-trips on `Get`, so it reproduces omitempty erasure without Keycloak.
- `require.Empty(*p)` passes for `&[]string(nil)`, which marshals as `null`; assert
  `require.Equal(t, ptr.To([]string{}), p)`.
- Fake Keycloak HTTP server for handler tests: `pkg/fakehttp` (`NewServerBuilder().AddKeycloakAuthResponders()`).
  Mocks come from `.mockery.yml` through `make mocks`.

## Ginkgo, Gomega, envtest

- A global `Expect` inside `Eventually(func(g Gomega))` panics instead of retrying; pass `g` into helpers.
- An `Ordered` suite skips every spec after the first failure. Put cleanup in `AfterAll` or `DeferCleanup`,
  never in a final `It`. Prove the first spec red and state which later specs are only implied.
- A spec that patches status and then acts cannot prove a path that needs that status empty: the status
  write enqueues a reconcile that refills it. Leave such a path to unit tests and do not name the spec after it.
- Nudge a reconcile in envtest with an annotation patch through `client.MergeFrom`, not `Update`
  (optimistic-lock 409). Tests that read Secrets through the cached client re-nudge inside `Eventually`
  and tolerate one repeated write from a status 409.
- `go test <pkg>` without `-v` prints only `ok` for a Ginkgo package. To see specs:
  `go test <pkg> -v -run <TestName> -ginkgo.focus="<describe text>"` into a file. `-ginkgo.v` with `./...`
  fails packages without Ginkgo; pass it to the Ginkgo package only.
- Envtest packages need `KUBEBUILDER_ASSETS` (the `<root>/bin/k8s/<version>-<os>-<arch>` directory that
  `make test` prints) and `TEST_KEYCLOAK_URL`; `lane.sh tests <work> <pkg ...>` sets both. A direct
  `go -C <tree> test` of an envtest package needs the sandbox disabled: envtest removes its port files
  under `~/Library/Caches/kubebuilder-envtest`, which the sandbox blocks.
- Integration tests run against every Keycloak version in the `.github/workflows/pr.yaml` matrix, not only
  `KEYCLOAK_TEST_VERSION`. A test against the live Keycloak passes on each: prove it on a temporary container
  of the oldest matrix version on a free port, then remove it.

## controller-runtime facts

- `RequeueAfter` is ignored when the error is non-nil; the error path uses the rate limiter (5 ms doubling,
  1000 s cap). Each occurrence logs `Reconciler returned both a non-zero result and a non-nil error`;
  counting that line is a cheap loop detector.
- Without a predicate, the controller's own status write enqueues the next reconcile.
- Several `RequeueAfter` for one key collapse to the earliest timer.
- `helper.IsFailuresUpdated` returns true when the count is UNCHANGED (the name is inverted): it drops only
  updates that change `failureCount`, and returns false for types that do not implement `FailureCountable`.
  Mid-chain status writes leave `failureCount` unchanged and pass it.
