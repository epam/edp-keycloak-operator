# Common rules for every lane agent

Read `<work>/state.md` first. It holds the paths and names below. The Lead passes none of them in the prompt.

- `root`: the checkout. Read-only for you: `git -C <root>` with `log`, `show`, `blame`, `worktree list` only.
  Other items run from the same root in other `.work/*/` directories: never read or touch them.
- `src`: a detached worktree at `base`. Read code, CRDs and the chart there; history with `git -C <src> log|show|blame`.
- `tree`: the worktree on `branch`. Every code change, build, test and commit happens there.
- `work`: the item's files. Evidence, logs and your output files go there; your hand-back cites the paths.
- Scripts: `bash <root>/.claude/skills/lane/scripts/lane.sh <command>`; `help` lists them. Constants such
  as the kube context, the image repository and the Keycloak namespace: `scripts/lib/env.sh`.
- Make: `make -C <tree> LOCALBIN=<root>/bin <target>` (tools and envtest assets live in `<root>/bin`).
  Go: `go -C <tree> ...`. Absolute paths everywhere. Never `cd`. Never a bare `git`: always `git -C <tree|src|root>`.
- Project rules: `<root>/AGENTS.md`. Comments: dry, instruction-only.
- Verify each claim in code or on the cluster; mark the rest unverified. Keycloak behaviour never comes
  from memory (`references/testbed.md` "Keycloak").
- Background jobs (port-forwards, watchers, tails): load `TaskStop` first (`ToolSearch select:TaskStop`) and
  stop every one before you report. A job alive at hand-back blocks your resume.
- Disable the sandbox only on "operation not permitted" or a TLS error, for that command only.
- Without the Agent tool, do delegated lookups yourself and say so.
- Report within the word cap of your role file. Logs and evidence go to `<work>`, not into the report.
