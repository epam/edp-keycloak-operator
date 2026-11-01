---
name: lane
description: "Fix one operator defect or add one small feature end to end, in its own lane: analysis, reproduction on kind, a design the user approves, development, QA A/B on kind, code review, PR body. A lane is a git worktree, a cluster namespace and a Keycloak realm prefix under .work/<slug>/, so several lanes run at once from one checkout, one session each."
argument-hint: "<description | file[#item] | GitHub issue #N or URL>"
disable-model-invocation: true
---

# lane

Task: `$ARGUMENTS`

The **Lead** (this session) runs eight steps and talks to the **Approver** (the user). It writes verdict
tables, `state.md`, the "Approval" section of `plan.md`, and commits. It writes no code: every code, test and
comment change goes through the Developer or the Fixer. Report each **done** line in a sentence or two.

**Requires:** `/krci-general:commit`, `/mattpocock-skills:pr`, `/code-review` (built in), the
`krci-godev:go-dev` agent, the `cr-field` skill, the `krci-triage` plugin with `krci-kubelock` on (`/config`),
`gh`, `kubectl`, `helm`, `docker`, `kind`. Intake names a missing one.

## Files

| Path | Holds |
|---|---|
| `scripts/lane.sh` | the mechanical jobs; `bash <root>/.claude/skills/lane/scripts/lane.sh help` |
| `scripts/lib/env.sh` | every repo constant, once |
| `roles/*.md` | one file per agent role; every agent reads `_common.md` |
| `references/` | `lead.md`: the Lead's procedures, named by the steps below; the rest: role files say who reads what |
| `tests/` | `check.sh` (static checks, hook tests), `trace.py`; the README has the smoke run |

## Terms

`<root>`: the checkout. `<slug>`: `issue<N>`, `item<N>`, or the first
words of inline text; `[a-z0-9-]`, at most 20 characters. `<work>` = `<root>/.work/<slug>/`. `<src>` =
`<work>/master-src`, detached at `<base>` (the upstream ref at intake, `origin/master` unless `--base`;
`lane.sh rebase` moves both). `<tree>`
= `<work>/tree` on `<branch>` (`fix/<slug>-<summary>`; `feat/` for a feature). `<ns>` = `<slug>`;
`<release>` = prefix plus `<slug>`. `<n>`: QA run. `<k>`: design-review round. `<round>`: code-review
table. All three count up across the item. Defect task: a wrong behaviour to reproduce. Feature task: a
missing behaviour. Production code: a diff that is not test, generated, comment-only or docs.

## Source and tracker

| Source | Key (`--source`) | Read | Tracker |
|---|---|---|---|
| Inline text | `text:<first words, kebab case>` | `$ARGUMENTS` | ask once in step 1: a file, GitHub issues, or none (gaps go into the final report) |
| File (`path#item` or heading) | `file:<path>#<item>` | the file, the item | the same file, updated in place, in its format |
| GitHub issue of the origin repo (`#N`, `issueN`, URL) | `issue:<N>` | `<work>/issue.json`, written by intake | the issue; comments, labels and new issues go to `<work>/github-drafts.md` as `gh` commands the user posts |

## Agents

Launch prompt, every role: `Role <role>, mode <mode>. Work: <work>. Read <root>/.claude/skills/lane/roles/_common.md,
roles/<role>.md and <work>/state.md first.` (no mode for a role without modes) plus the "Lead names" cell.
Nothing else.

| Role | Modes | Agent | Model | Lifetime | Lead names |
|---|---|---|---|---|---|
| analyst | defect, feature | `general-purpose` | sonnet | fresh | - |
| architect | plan, after-review, after-qa-fail | `general-purpose` | opus | step 3; resumed in steps 4 and 6 | after-review: `verdicts-design-<k>.md`, the evidence path; after-qa-fail: the failing rows, `evidence/fix-<n>/` |
| design-reviewer | first, rework | `krci-godev:go-dev` | opus | fresh per round | rework: `evidence/fix-<n>/` |
| developer | stage, qa-failed, rework | `general-purpose` | sonnet | step 5; resumed per stage, after a QA failure and for a rework | qa-failed: the failing rows, `evidence/fix-<n>/`; a fresh relaunch: the committed stages |
| fixer | - | `general-purpose` | sonnet | fresh per round | `verdicts-<round>.md` |
| qa | repro, verify, recheck | `general-purpose` | sonnet | defect: step 3, resumed in 6 and 7. Feature: fresh in 6 | `<n>`; recheck: the rows |
| deep review | - | `/code-review xhigh`, a skill call by the Lead | session | per call | `lead.md` "Deep review call" |
| polish-reviewer | - | `general-purpose` | sonnet | fresh | - |

Resume = `SendMessage` to the same agent (load it with `ToolSearch select:SendMessage,TaskStop`). A failed
resume: a fresh agent of the same role. At every launch: `lane.sh state agent <work> <role>=<id>`. When the
task notification arrives: `lane.sh state ledger <work> <step> <role> <model> <duration_ms> <tool_uses> <tokens>`;
no placeholder rows; a resume is its own row, its tokens figure is cumulative (record the difference); a skill
call: model `n/a`, the other figures from its task notification, else wall-clock ms and `-`. Step 8 reports `lane.sh state totals <work>`.

## Invariants

- **Isolation.** An item touches only its own `<work>`, `<tree>`, `<branch>`, `<ns>`, `<release>`, images
  and `<slug>-*` realms. `<root>` is read-only, except `fetch`, `worktree` and the step 8 rebase. Never a
  bare `git`; never `cd`. Stage commits by path; never `.work/`.
- **Verdicts.** Every finding gets a row: finding, verdict (accept / accept as text only /
  reject), reason. Verify in code first. Reject findings already in the tracker or present before the change;
  state them in the PR body as known limits when they bound the fix. A row that adds or changes a test names
  its red path; no red path: reject.
- **Precedent first** (`references/patterns.md`). A new pattern needs the Approver's explicit yes or no,
  one by one, in step 4.
- **Freeze before fan-out.** Commit before any cluster run of the fix or any code review; amend afterwards
  while unpushed. One concern per commit; generated drift outside the task's kinds stays out.
- **Caps.** Design review: 2 rounds per entry to step 4. QA failure: 2 in total, re-checks included.
  Developer and Fixer: 2 failed attempts per stage. At a cap: stop, report the evidence, Cleanup.
- **Cleanup needs an answer.** At every stop and in step 8 after the PR URL: report and ask: keep, tear down the
  cluster side, or also discard tree and branch. Teardown = `lane.sh teardown <work>` (sandbox disabled); hand
  the user the namespace command it prints. `--all` (tree and branch) runs after the push; `--all --discard`
  skips both checks and prompts; the Approver orders it.
- **User gates.** Design approval (step 4). The user runs every outward command: the push, `gh pr create`,
  `gh issue comment`. The Lead drafts, never runs, not even on an approval in chat.
- **State.** `lane.sh state step <work> <n> doing|done|stopped|parked [values]` at each transition;
  `state set` for every value (`lane.sh state --help` lists the keys). `parked` = put aside by the Approver;
  the resume check treats it like `stopped`.

## Steps

### 1. Intake

Resume check first (`lead.md` "Resume check"). New run: resolve the source; restate the task (symptom or
missing behaviour, expected behaviour, scope; defect or feature). Inline text: ask the tracker question; a
feature text missing type, default, validation or scope: ask in the same message, else state the assumptions
in the plan. Pick `<slug>` and `<branch>`; run `lane.sh intake <slug> <branch> [--issue N] --source <key>
--task "<line>"` (sandbox disabled; `--base <ref>` for a release branch); a `STOP:` or `FAIL:` goes by
`lead.md` "Intake stops". Then
`state set variant= tracker=`; settle the tracker.

**Done:** task, variant, source, tracker, slug, branch, base, paths, ns, release stated; `state.md` current.

### 2. Analysis

Launch the analyst (mode = variant). From `cause.md`: `lane.sh state scope <work> <the artifacts paths>` and
`state set keycloak=shared|dedicated` (`server-global` → dedicated). "Decision, not a bug" and `scope: crd`:
`lead.md`.

**Done:** cause (or the unresolved path) with `file:line`, introducing commit, verdict, sibling table, pattern
survey, cluster scenario; scope and keycloak named to the Approver; a crd item names the other live crd items (`lead.md` "CRD scope").

### 3. Reproduction and design

Defect: launch qa (mode repro) and architect (mode plan) together. Feature: launch the architect; no repro;
write `evidence/master/INVENTORY.txt` (`<utc> nothing installed`); the Analyst's table in `cause.md` is the
evidence. Wait for every task notification, then `lane.sh gate <work> 3`: `FAIL:gate` = resume the role that
owes the piece, or write it yourself (variant, feature inventory). Not reproduced: stop, report, Cleanup.

**Done:** `gate` OK; evidence table stated.

### 4. Design review and approval

Review rounds: `lead.md` "Design review rounds". Then stop for the Approver. Present, in this order: "Issue
brief" and "Before / after" verbatim from `plan.md`; cause and verdict; reproduction table; decision and
rejected alternatives; precedent table; **new patterns**, each in three lines (question it answers; what it
does; what is lost with "no") beside its alternative; commit split and behaviour changes; case table (once:
the Before / after results block, when it covers every row) and tests; review verdicts; options; the plan path. End with "Approve the plan, or request changes? New patterns
need an explicit yes or no each." Replies: `lead.md` "Approval replies".

**Done:** every finding has a verdict; approval in plain words; recorded in `plan.md` and `state.md`.

### 5. Development

After the recorded approval only. Check `git -C <tree> status --porcelain --untracked-files=no` is empty,
`branch --show-current` is `<branch>`, `rev-parse HEAD` is `<base>` (after a rework: stay on the branch).
Launch the developer (mode stage), one stop per planned commit; after a plan rework resume it with mode
rework. Commit each stage by `references/git-history.md` "Procedure". A deviation from the plan goes to the
Approver as in step 4. After the last stage: `lane.sh rebase <work>` (sandbox disabled); `MOVED=yes` →
`lane.sh tests <work>` again.

**Done:** tests red on old code (shim when needed), green on new; lint clean; `lane.sh tests` OK on the
latest `origin/master`; commit hashes stated.

### 6. QA verification

`state set qa-run=<n>`. Defect: resume qa; feature: launch it fresh. Mode verify, background, on the frozen commit and start step 7 on the
same commit at once; wait for QA and both reviews before the verdict table. Pass: keep the setup. Fail:
`lead.md` "QA failure".

**Done:** every case-table row passes in `evidence/fix-<n>/`; or the rework entered; or stopped.

### 7. Code review

On the frozen commits: polish-reviewer in the background; the deep review as a skill call in the Lead's turn
(`lead.md` "Deep review call"). Verdict table, fixer, fixup commits, re-review, re-check and squash:
`lead.md` "Fix round".

**Done:** one verdict table per round; fixes applied; `lane.sh tests` OK; re-review and re-check done or not
needed; open findings reported; final hashes stated.

### 8. Release

Tracker: the task status and one entry per gap the reviews surfaced (GitHub: drafts in `github-drafts.md`;
the `gh` command takes a here-doc on `--body-file -`). PR body: `/mattpocock-skills:pr`, filled as
`lead.md` "PR body" says. Base refresh: `lead.md`. Link `[pr-body.md](<work>/pr-body.md)` for review, then hand
over both commands, `<owner/repo>` from `git -C <root> remote get-url origin`:

```
! git -C <tree> push -u origin <branch>
! gh pr create -R <owner/repo> --head <branch> --title "<first commit subject>" --body-file <work>/pr-body.md
```

After the PR URL, and later the merge: `lead.md` "Base refresh and merge".

**Done:** tracker current; `pr-body.md` written and linked; both commands handed over; cleanup answer applied; ledger
totals reported.
