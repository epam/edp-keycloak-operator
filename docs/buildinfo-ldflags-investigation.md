# LDFLAGS vs `debug/buildinfo` — investigation

**Question**: can the `-ldflags -X` stamping in [Makefile:35-43](../Makefile#L35-L43) be removed in favor of Go's built-in `debug/buildinfo`? Can `kubectlVersion` be dropped?

**Short answer**: mostly yes. `gitCommit` and `kubectlVersion` are cleanly replaceable today. `buildDate` is replaceable if commit-time is acceptable instead of wall-clock build-time. `version` / `gitTag` (based on `git describe --tags`) cannot be fully reproduced from `debug.ReadBuildInfo()` alone; they'd require either a kept-minimal `-X`, a `//go:embed VERSION` file, or switching to `go install module@tag` semantics.

---

## Current state

Five package-level variables in [`github.com/epam/edp-common/pkg/config`](../Makefile#L1) are injected at link time:

| Var               | Makefile source                                                 | Log key in [cmd/main.go:113-121](../cmd/main.go#L113-L121) |
|-------------------|------------------------------------------------------------------|-----------------------------------------------------------|
| `version`         | `git describe --tags`                                            | `version`                                                  |
| `buildDate`       | `date -u +'%Y-%m-%dT%H:%M:%SZ'`                                  | `build-date`                                               |
| `gitCommit`       | `git rev-parse HEAD`                                             | `git-commit`                                               |
| `gitTag`          | `git describe --exact-match --tags HEAD` (only when worktree clean) | `git-tag`                                                |
| `kubectlVersion`  | `go list -m all \| grep k8s.io/client-go \| cut -d' ' -f2`      | `go-client`                                                |

Two other fields in `BuildInfo` — `Go` and `Platform` — are already filled from `runtime.Version()` / `runtime.GOOS,GOARCH`, no ldflags needed.

The binary is built on the host (see [Makefile:144-146](../Makefile#L144-L146)) and copied into a distroless image ([Dockerfile:6](../Dockerfile#L6)). The build happens inside the git worktree, so `-buildvcs=true` (the default) will stamp VCS info — the Dockerfile is not an obstacle.

**Verified empirically** with `make build && go version -m dist/manager-$(go env GOARCH)`:

```
path    github.com/epam/edp-keycloak-operator/cmd
mod     github.com/epam/edp-keycloak-operator v1.3.0-alpha-81.0.20260418121102-bb0490407faf+dirty
dep     k8s.io/client-go v0.33.0
build   vcs=git
build   vcs.revision=bb0490407faffab8bd5bf9e4b2d6d0ebb63d5094
build   vcs.time=2026-04-18T12:11:02Z
build   vcs.modified=true
```

So `gitCommit`, commit timestamp, dirty flag, and client-go version are all present without any `-X`. Note the `mod` line: Go already synthesizes a pseudo-version (`v1.3.0-alpha-81.0.<date>-<sha>+dirty`) from the nearest tag — see the "aggressive" option below.

## What `debug/buildinfo` gives you for free (Go 1.18+)

`runtime/debug.ReadBuildInfo()` returns:

- `Main.Path` / `Main.Version` — module path + version. For `go build` in a local checkout, `Main.Version` is `(devel)` (useless). For `go install module@v1.2.3`, it's the semver.
- `GoVersion` — already covered.
- `Settings []BuildSetting` — key/value pairs, including:
  - `vcs` (e.g. `git`)
  - `vcs.revision` — full commit SHA — **replaces `gitCommit`**
  - `vcs.time` — commit timestamp (RFC3339) — approximates `buildDate` (see caveat)
  - `vcs.modified` — `true`/`false` dirty flag
  - `GOOS`, `GOARCH`, `-ldflags`, `-tags`, `CGO_ENABLED`, etc.
- `Deps []*Module` — every dependency with its version — **replaces `kubectlVersion`** (just iterate and find `k8s.io/client-go`; authoritative and less fragile than `go list -m all | grep`).

Sources: [Go 1.18 release notes](https://go.dev/doc/go1.18#debug_buildinfo), Stapelberg's "Stamp it all" article.

## Field-by-field verdict

### `gitCommit` — drop cleanly ✅
`vcs.revision` in `ReadBuildInfo().Settings` is the full SHA. No caveats for this build setup.

### `kubectlVersion` — drop cleanly ✅
Two wins: (a) no more `-X`, (b) replace the fragile `go list -m all | grep k8s.io/client-go | cut -d' ' -f2` shellout with direct module lookup in `Deps`. Also, the variable is misnamed — it has always been the `client-go` version, not `kubectl`'s (the log key already acknowledges this: `"go-client"`). Worth fixing while changing.

### `buildDate` — trade-off ⚠️
`vcs.time` is **commit** time, not **build** time. Pros of switching: builds become reproducible (same commit → same timestamp). Cons: if ops uses `build-date` to tell "when did someone rebuild the image from this commit", they lose that signal. Most teams accept commit-time; confirm with whoever reads the log line.

### `version` — partial ❌
`git describe --tags` produces `v1.28.0-3-gabcdef1`. `debug/buildinfo` has no git-describe equivalent. Options:
1. **Keep one `-X` for `version`, drop the other four.** Smallest diff, preserves existing log output.
2. **Embed a `VERSION` file** via `//go:embed`. Release bumps edit the file; dev builds use its current contents. Works without ldflags but adds a manual bump step.
3. **Rely on `Main.Version` from buildinfo.** Works for consumers running `go install .../cmd@v1.2.3`. For local `make build` in a checkout, you get `(devel)` — likely unacceptable.
4. **Compute from buildinfo**: `vcs.revision` short SHA + `vcs.modified` flag, accepting that no "last-tag" info is available. Useful for dev builds only.

### `gitTag` — drop (but it's not strictly redundant) ❌
Only populated when HEAD is exactly at a tag AND the worktree is clean. It carries a signal `version` does **not**: "this is a clean release build." `git describe --tags` on a dirty checkout that's on a tag still returns the tag name (no `-dirty` without `--dirty`), so `version=v1.28.0` and `gitTag=""` can coexist. If nobody consumes `git-tag` in logs, drop it. If someone does (alerting, dashboards keying off clean-release builds), replace with a derivation from `vcs.modified` at runtime.

## Caveats (general, not specific to this repo)

- **Docker builds that `COPY` only `*.go` files** lose VCS stamping silently. Not an issue here — the binary is built on the host before `COPY`.
- **`-buildvcs=false`**, `-trimpath` with certain flag combos, or running `go build` outside any VCS root will disable VCS stamping. Current Makefile doesn't use these.
- **`vcs.modified=true`** triggers on any dirty file, including untracked. The current `gitTag` logic already gates on `git status --porcelain`, so behavior stays consistent.
- **`edp-common/pkg/config` is a shared library** across EDP operators. Three clean paths:
  1. Keep `edp-common` but stop feeding it (the vars fall back to their defaults: `"XXXX"`, `"1970-01-01..."`, empty strings). Unacceptable — logs become useless.
  2. Replace the call in `cmd/main.go` with a local `buildinfo.Get()` helper that reads `debug.ReadBuildInfo()`, bypassing the shared package. Cleanest for this repo; other EDP operators unaffected.
  3. Upstream a `debug/buildinfo`-based implementation into `edp-common` so all operators benefit. Highest value, highest coordination cost.

## Recommended path

**Minimal, low-risk**: drop `gitCommit`, `gitTag`, `kubectlVersion`, and `buildDate` from the Makefile; keep only `version` via `-X`. In `cmd/main.go`, add a small helper that reads `debug.ReadBuildInfo()` to populate the remaining fields before logging (or replace the `buildInfo.Get()` call entirely). Net result:

- Makefile loses ~10 lines plus the `KUBECTL_VERSION` shellout.
- Binary gains authoritative, free-to-compute VCS/dep info.
- Log output preserved (`git-commit`, `go-client`, `git-tag` derivable from buildinfo; `build-date` replaced by commit time — confirm acceptability).

**Aggressive**: also drop `version`. Note that simply deleting the `-X` line makes the logged value fall back to the string literal `"XXXX"` defined in [`edp-common/pkg/config/buildinfo.go`](https://github.com/epam/edp-common/blob/main/pkg/config/buildinfo.go) — not to a sensible default. To get `(devel)` / the module semver, you must also replace the `buildInfo.Get()` call site with a `debug.ReadBuildInfo()` reader in `cmd/main.go`. Combine with embedding a `VERSION` file for release builds. Requires changing release tooling; weigh against the benefit of a fully flag-free `go build`.

## TODOs the Makefile already flagged

- ~~"Investigate whether we can get rid off these flags and use debug/buildinfo package"~~ — answered above.
- ~~"Do we need kubectlVersion?"~~ — no; derive from `Deps` at runtime, or drop entirely if nobody reads `go-client` in logs.
