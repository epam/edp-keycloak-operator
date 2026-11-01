# lane tests

Run after a change to the skill, the hook or `.claude/settings.json`. `<root>` is the checkout.

```
bash <root>/.claude/skills/lane/tests/check.sh
python3 -I <root>/.claude/skills/lane/tests/trace.py <transcript.jsonl> --root <root> [--full]
```

`check.sh`: word budgets, no machine or person strings, repo constants in `scripts/lib/env.sh` only,
stated-once phrases, script syntax and `--help`, shellcheck when installed, the hook parses, the hook
decisions of `hook_test.py` (allow, deny and fall-through per sample command, fail-closed without `env.sh`).

`trace.py`: one sub-agent transcript: duration, turns, tokens, tool counts, skill files read with repeat
counts, other files read, failed or denied Bash commands, text lines that signal a gap in the skill.
Transcripts: `~/.claude/projects/<project>/<session>/subagents/agent-<id>.jsonl`; the ids are the `agents`
value of the item's `state.md`.

## Smoke run of every script

On the kind testbed with a scratch slug; about 15 minutes. `L` = `bash <root>/.claude/skills/lane/scripts/lane.sh`,
`W` = `<root>/.work/lane-smoke`. Sandbox disabled where marked (D). Stop at the first unexpected result line.

```
L lock show                                                      expect: crd-lock: - (or the live crd items)
L status                                                         expect: no live namespaces, or known items only
L --dry-run intake lane-smoke fix/lane-smoke-x --task smoke      expect: BASE=<hash>; STOP:testbed = shared Keycloak or cert-manager not ready
L intake lane-smoke fix/lane-smoke-x --source text:smoke --task "script smoke run"     (D)
L state set W scope=code keycloak=shared
L build W master                                                 expect: TAG=lane-smoke-master-<ts>
L install W master <tag>                                         expect: RESULT=OK, realm lane-smoke-main
L kc W realms                                                    expect: lane-smoke-main
L build W fix --run 1                                            expect: TAG=lane-smoke-fix-1-<ts>
L upgrade W <tag>                                                expect: RESULT=OK
L tests W ./internal/controller/keycloakrealmuser/...            (D)  expect: RESULT=OK, validate-docs: 0
L rebase W                                                       (D)  expect: MOVED=no
L --dry-run teardown W                                           read the plan
L gate W 3                                                       expect: FAIL:gate, missing variant(state: pending) cause.md plan.md
L teardown W --all                                               (D)  expect: STOP:unpushed
L teardown W --all --discard                                     (D)  expect: removed: ..., the namespace command printed
```

After the run: `W/evidence/master/INVENTORY.txt` lists namespace, CRDs, secret, release, Keycloak CR, realm
and image; `W/agent-timings.md` has one `script:*` row per build, install, upgrade, tests, rebase and teardown run; `git -C <root> worktree list` has no
lane-smoke entry; `docker image ls` has no lane-smoke tag; the user runs the printed namespace delete.
