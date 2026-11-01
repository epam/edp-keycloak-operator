# Testbed: the decisions a cluster run needs

The commands live in `scripts/lane.sh` (`build`, `install`, `upgrade`, `kc`, `tests`, `teardown`, `rebase`).
This file holds what the scripts cannot decide. Constants (context, namespaces, image repository, flags) are
in `scripts/lib/env.sh`; read it once instead of guessing a value. The cluster is the `try-kuberocketci`
testbed (its `CLAUDE.md` and Makefile stand it up; the kube context is pinned by krci-kubelock); cert-manager,
the operator CRDs and the shared Keycloak stay installed between runs. "The user fixes the testbed" means that repo.

## Shared cluster

- The kind cluster hosts other components and other items at the same time. An item works only in its
  namespace `<ns>` (= `<slug>`, labelled by `lane.sh install`). Several items run from one checkout; each
  owns `<work>`, `<tree>`, `<branch>`, `<ns>`, `<release>`, its image tags and its `<slug>-*` realms, and
  touches nothing of another item. Items share `<root>` (read-only), `<root>/bin`, the cluster, its CRDs and
  the shared Keycloak.
- The operator is installed with cluster-wide reconciliation off (the install flag in `env.sh`); never turn
  it on: the webhook configurations lose their `namespaceSelector`, three of four have `failurePolicy: Fail`,
  and the operator reconciles every namespace. A scenario that needs cluster-wide reconciliation stops and
  goes to the Approver.
- Cluster-scoped objects an item may create or change, all named with `<ns>` or `<slug>`: the chart's two
  webhook configurations, its webhook ClusterRole and binding, the metrics-scraper ClusterRole and binding,
  and the namespace itself. Nothing else. Never touch the protected namespaces listed in `env.sh`, apart from
  reading the Keycloak admin secret.
- Deleting the namespace is the user's command: `! kubectl --context <ctx> delete ns <ns>`. The hook denies
  it to the skill so no item can remove another's.

## CRDs are shared by every item

- `scope: code` (no change under the CRD paths of `env.sh`): the scripts apply a CRD only when it is missing
  and never overwrite one; teardown leaves them.
- `scope: crd`: `state scope` registers the item in the crd-lock (`lane.sh lock show`), information only; no
  item waits on it. The master run still adds only missing CRDs (the installed set may be newer than `<src>`,
  and a forced base apply would prune fields other operators use); the fix run applies the CRDs from `<tree>`
  server-side, and teardown restores them from `<src>` only when no other live item namespace exists. While
  another crd item is live, the installed CRDs may be theirs: QA names them in the evidence.
- Helm never upgrades CRDs; the chart is installed with `--skip-crds` for that reason.
- KUTTL (`make e2e`) installs the chart with its CRDs and applies `Namespace keycloak`: it is not isolated.
  Run it only as a `crd` item while no other item namespace is live (`lane.sh status`), with the sandbox
  disabled; with other items live the Approver decides. It
  needs the repo root as working directory (step 00 uses `../../../deploy-templates`):
  `env -C <tree> env E2E_IMAGE_REPOSITORY=<image name> CONTAINER_REGISTRY_URL=<registry> CONTAINER_REGISTRY_SPACE=<space> E2E_IMAGE_TAG=<tag> kubectl-kuttl test --config <tree>/kuttl-test.yaml`
  (the three image parts split `IMAGE_REPO` of `env.sh` as `<registry>/<space>/<image name>`). Namespaces
  are auto-deleted; capture CR YAML with a `kubectl get <kind> -A -o yaml -w` watcher started beforehand
  (`-w` takes one resource type). KUTTL tells `[]` from an absent key in asserts. Every CR with a finalizer
  added to an e2e step also goes into `99-cleanup.yaml`. KUTTL leaves cluster-scoped
  `*-webhook-configuration-kuttl-<ns>` objects; delete them.

## Images and releases

- Images come only from `lane.sh build`: the native architecture through the Makefile; never `make e2e`
  (amd64, `--no-cache`), QEMU or a raw `go build`. A detached `<src>` has no reachable tag: `git describe`
  prints `No names found`; the build still succeeds with an empty version and the full commit hash. One fresh
  tag per build: kind caches by tag.
- After `lane.sh upgrade`, compare `helm get manifest <release> -n <ns>` with `helm template` of the same
  chart and values before measuring; a difference means the release did not take the chart you think.

## Keycloak

- Default `keycloak: shared`: the Keycloak CR in `<ns>` points at the shared service; QA reads admin events
  per realm. The Analyst marks the scenario `realm-local` or `server-global` in `cause.md`. `server-global`
  (master realm settings, server features, a Keycloak version, a restart) means `keycloak: dedicated`:
  `lane.sh install` then starts an own Keycloak in `<ns>` from the e2e manifest without its Namespace document.
  Never `kubectl delete -f` the original manifest, never restart or delete the shared deployment.
- The shared Keycloak is also the fixture of `lane.sh tests` for every item: the tests name their realms with a
  pid and time suffix (`pkg/testutils/realm.go`), so parallel runs share it. Never `make start-keycloak`
  (a Docker container on port 8086).
- Every realm, user, client or master-realm object an item creates is named `<slug>-<purpose>`
  (`<slug>-main`, `<slug>-viewer`). Never list, read or change another name. `lane.sh kc` enforces the
  prefix on realm arguments; `master` needs `--master`.
- Verify Keycloak behaviour on the cluster or in the source, never from memory. Source clone per tag at
  `/tmp/claude/keycloak-src/<tag>`, shared by every item; tag = `KEYCLOAK_TEST_VERSION` from the Makefile.
  When it is missing: `git clone --depth 1 --branch <tag> https://github.com/keycloak/keycloak.git
  <work>/keycloak-src-<tag>` (sandbox disabled), then `mv` it into place if the target is still missing, else
  delete the copy. Admin REST attribute rules:
  `server-spi-private/src/main/java/org/keycloak/userprofile/DefaultAttributes.java`, `DefaultUserProfile.java`.
- Admin metadata endpoints (`/admin/serverinfo`, provider and config descriptions) declare only part of what a
  provider reads or stores. For the rest read the source, or `docker cp` the jars from the running container
  and use `javap -c` or `strings`.
- GET masks secret values, so it cannot show what was written. Count writes with admin events:
  `lane.sh kc <work> update events/config -r <slug>-<purpose> -s adminEventsEnabled=true`, then
  `lane.sh kc <work> get admin-events -r <slug>-<purpose> -q resourceTypes=<TYPE>`. Admin events cannot be
  cleared: count from a timestamp written before each phase.
- Not every admin write emits an admin event. Zero events prove zero writes only for a call known to emit
  one; otherwise read the state back and check the operator DEBUG `Sending request` lines (method and path,
  no query, no body).
- Requests by method, path and query, GETs included: the access log of container `nginx` in the Keycloak
  deployment (`kubectl --context <ctx> -n <keycloak namespace> logs deploy/keycloak -c nginx --since-time=<t>`).
  Every item writes to it: filter by the `<slug>-` realm.
- View-only user for negative tests: a master user `<slug>-viewer` with only client role `view-realm` of
  client `master-realm`. Recovery: grant it master realm role `admin`.

## Measurements

- **Fast feedback:** keep measurement windows at 2 to 3 minutes. A tunable timing may not drive the
  controller under test: check that controller's own requeue first. Lower tunable timings for the run
  (`successReconcileTimeout` / `SUCCESS_RECONCILE_TIMEOUT`, test-only constants), write
  `timing: <name> original=<value>` to `INVENTORY.txt`, restore afterwards and verify (`helm get values`,
  deployment env). Nudge reconciles with an annotation instead of waiting for a backoff.
- **Reconcile rate:** create the CR, wait for `status.failureCount`, then over a fixed window diff
  `failureCount` and `grep -c` of `Reconciling <Kind>` and the `non-zero result and a non-nil error` warning
  in the operator log; recreate the CR after switching images.
- **Webhook A/B:** patch `clientConfig.service.path` of one webhook to an unserved path; restore it and
  compare with a saved copy.
- **Metrics** on :8443 need authn/authz:
  ```
  kubectl --context <ctx> -n <ns> create serviceaccount metrics-scraper
  kubectl --context <ctx> create clusterrole keycloak-operator-metrics-scraper-<slug> --verb=get --non-resource-url=/metrics
  kubectl --context <ctx> create clusterrolebinding keycloak-operator-metrics-scraper-<slug> \
    --clusterrole=keycloak-operator-metrics-scraper-<slug> --serviceaccount=<ns>:metrics-scraper
  kubectl --context <ctx> -n <ns> port-forward deploy/keycloak-operator <free-port>:8443
  curl -sk -H "Authorization: Bearer $(kubectl --context <ctx> -n <ns> create token metrics-scraper)" https://localhost:<free-port>/metrics
  ```
  Teardown removes the ClusterRole and binding by that name.
- Capture operator logs before any `rollout restart`; the pod is gone afterwards. After an upgrade wait for
  the ready log line (`lane.sh upgrade` does) before measuring.
- `kubectl apply` of a manifest whose last-applied holds `[]` restores `[]` over a value the operator rewrote.
- Port-forwards: a free local port picked at run time, never a fixed one; stop every background job you
  started (port-forwards, watchers, log tails) with `TaskStop` before you report. A job alive at hand-back
  blocks your resume. Keep `kubectl delete` in a separate call from installs.

## Inventory and teardown

- `<work>/evidence/master/INVENTORY.txt` is the teardown list. The scripts append what they install; QA
  appends every CR, secret, realm, master-realm object, port and timing it creates or changes by hand, as
  it goes, never at the end.
- `lane.sh teardown <work>` removes everything in the documented order and prints the leftovers and the
  user's namespace command. Timings are restored by hand from the `timing:` lines it prints.

## Permissions and sandbox

- The `krci-kubelock` mod of the `krci-triage` plugin pins `KUBECONFIG` to the context in `env.sh`: every
  `kubectl`, `helm`, `make` target and script sees only that context, and a command that names another
  context, `--kubeconfig`, a `KUBECONFIG=` override or a `~/.kube` path is denied before it runs. Its
  settings are the user's (`/config`); intake stops with `STOP:kubelock` while the pin is off. The scripts
  still pass `--context` and `--kube-context` explicitly, so a run without the lock cannot hit another cluster.
- The hook `.claude/hooks/lane-allow.py` allows the lane script, Make through `LOCALBIN`, `go test` and
  read-only git in the lane worktrees, and denies namespace deletes, deletes in protected namespaces, CRD
  deletes, a push from a lane tree and `gh pr create`; its docstring is the list. `.claude/settings.json`
  allows every `kubectl` on the pinned context without a prompt: the hook denies are the only brake, and
  isolation between items rests on the rules in this file. Everything else prompts by design
  (`git add|commit|rebase|fetch|worktree remove|branch -D`, `helm uninstall`, `docker rmi`, `gh`, the Make
  targets in `PROMPT_TARGETS`, `lane.sh teardown <work> --all --discard`). A command with a shell separator (`;`, `|`, `&&`) never matches an
  allow rule: one command per call.
- `make build`, `vet`, `lint-fix`, `generate`, `manifests` and `validate-docs` run inside the sandbox.
  `validate-docs` compares regenerated `docs/api.md` and `deploy-templates/README.md` with the index: run it
  after `git add` or after the commit, not with unstaged changes to those files.
- Envtest starts a local API server only and never touches the cluster. It needs the sandbox disabled because
  it removes its port files under `~/Library/Caches/kubebuilder-envtest`; the user can instead allow that
  path for writes in the sandbox settings.
- `$TMPDIR` differs between sandboxed and unsandboxed shells: write and read a file in the same mode.
- Process substitution `<(...)` and `/dev/fd` fail; an empty `diff` is then false evidence. Write temp files.
- `kill`, `ps`, `pgrep` fail with "operation not permitted"; stop background work with `TaskStop`.
- Shell writes into `.claude/` and into files outside the checkout (a tracker file in another repository)
  fail; use the Write and Edit tools there.
- Tools in `<root>/bin` are built with the host `go` and not rebuilt when it changes. After a Go upgrade,
  `make lint` fails with typecheck errors such as `could not import math/rand/v2`: check with
  `go version -m <root>/bin/<tool>`, delete `<root>/bin/<tool>` and `<root>/bin/<tool>-<version>`, and the
  Lead reruns `lane.sh intake <slug> <branch>` for the item (sandbox disabled); the worktrees share `<root>/bin`.
- `git worktree remove` refuses a dirty worktree: inspect `git -C <tree> status` and ask before `--force`.
- The sandbox protects `.vscode/settings.json` and `.vscode/launch.json` in every directory, and the repo
  commits both: a worktree removal inside the sandbox drops the git entry and leaves the directory.
  `lane.sh teardown` therefore runs with the sandbox disabled and removes such a leftover directory itself.
  A worktree directory removed by hand leaves a stale entry: `git -C <root> worktree prune`.
