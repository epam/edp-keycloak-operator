# Lead procedures at the branch points

`SKILL.md` holds the flow. This file holds what the Lead does when a step leaves the straight path. Read
the section the step names, when it names it.

## Resume check (step 1)

`lane.sh status --source <key>` with the key of the Source table; a bare label (`F1`) is matched against the
`source=` column of `lane.sh status` first. Nothing found (exit 1, no output): a new run. Found:

- every step `done`: report the outcome (the `pr` value); stop.
- a `stopped` or `parked` step: show the state; ask the Approver to continue or to restart under a new slug.
- otherwise restate the state and continue at the first step not `done`. A `doing` step restarts, except
  step 4 with `approval: pending`: re-present the stored plan and verdict tables without a new review.
  After a teardown of a crd item, rerun `lane.sh state scope <work> <artifacts>`: teardown removes it from
  the crd-lock. Before restarting step 3 or 5 run `lane.sh status`, `helm list -n <ns>` and
  `git -C <tree> status --porcelain --untracked-files=no`; ask: adopt the leftovers, or tear down and start again.

## Intake stops (step 1)

`STOP:<reason>`: ask the Approver once, rerun with the answer. `slug`: pick a valid slug. `deps`: install the
named plugin, skill or tool. `kubelock`: the krci-kubelock pin is off or on another context; the user turns
it on in `/config` (plugin `krci-triage`). `origin`: no remote, or the fetch failed (sandbox, network). `slug-taken`: a
work dir of that slug exists for another branch; pick a new slug or tear the old item down. `branch`,
`worktree`, `namespace`: leftovers of an old run; `--adopt` reuses them, or pick a new slug. `testbed`: the
shared Keycloak or cert-manager is not ready; the user fixes the testbed. `pr`: `--waive-pr` accepts the open PR; intake records it in the `issue` value. `issue`: the
`gh issue view` failed; check the number and `gh auth status`.
`FAIL:worktree|tools`: read the log it names, fix, rerun the same command; intake is idempotent.

## CRD scope (step 2)

`scope: crd` means the item changes CRDs shared by every operator on the cluster. `state scope` registers
the item in the crd-lock, a list of live crd items; it is information and nothing waits on it. At the done line
say: "This item changes CRDs, shared by every operator on the cluster. Other live crd items: <`lane.sh lock show`, minus this slug>.
Their QA can run against this item's CRDs, and this item's QA against theirs." No question follows. The
master repro adds only missing CRDs whatever the scope; `install fix` and `upgrade` apply the item's CRDs from `<tree>`.
QA names the live crd items in its evidence. A failing row that traces to another item's CRDs is reported as
that, not as a defect of this item. Teardown releases the entry.

## Decision, not a bug (step 2)

Stop. Report the evidence. Offer two options: close as designed, or restate as a feature and restart at
step 1 under a new slug. Cleanup.

## Design review rounds (step 4)

A round runs on the first pass and on every rework entry; a rework entry reviews the plan's "Rework" section
against the failing `evidence/fix-<n>/` and needs approval again. One design-reviewer per round; table in
`verdicts-design-<k>.md`; resume the architect (mode after-review) once per round, in one message with every
accepted finding, the evidence path and any Approver change collected so far. A second round runs only when
the verdict was "disagree", or the mechanism or compare rules changed afterwards. Two rounds per entry is the cap.

## Approval replies (step 4)

- Approved (an approval naming no pattern approves the alternatives): append an "Approval" section to
  `plan.md` with the date, the conditions and yes/no per pattern (the Lead's only write into the plan);
  `state set approval=<date; conditions; patterns>`; step 5.
- Mixed reply (some patterns rejected): approved; each rejected pattern switches to its alternative, already
  in the plan; resume the architect to update; record.
- Changes requested: resume the architect with the user's words; present again.
- Rejected: stop; Cleanup.

Never infer a missing answer: the next message that is not an approval is a change request. When the session
ends first, `approval` stays `pending`.

## QA failure (step 6)

QA writes the failing rows, observed against expected, to `evidence/fix-<n>/` and keeps the setup. The
review results for the old commit are set aside; findings that still apply carry over.
`state set pre-rework-head=<HEAD>`; resume the developer (mode qa-failed) naming the rows and the evidence
path. Its answer:

- `implementation`: it adds commits; the Lead commits; `state set qa-run=<n + 1>`; step 6 again.
- `plan flaw`: resume the architect (mode after-qa-fail) with the rows and the path; it appends "Rework";
  then step 4 (design-reviewer mode rework), step 5 (resume the developer, mode rework),
  `state set qa-run=<n + 1>`, step 6.

The second QA failure stops the flow, whichever route: report, Cleanup.

## Deep review call (step 7)

`/code-review xhigh <base>..<branch> — code is in the worktree <tree> (the session cwd is the main checkout,
whose HEAD is not this branch; read files there, not in the cwd); <what was implemented: change, kinds,
behaviour>; plan: <work>/plan.md; approved new patterns: <list or none>; any other mechanism without a
precedent in the repo is a finding`. Never `--fix` or `--comment`. Name the range, never `HEAD`. If the
skill rejects the range, pass `<branch>`; if the call fails, tell the Approver.

Re-review, after a fix round that changed production code: `/code-review xhigh <pre-fix-head>..<branch>`
plus `fixed in this range: <list>; rejected, do not re-report: <list>; report critical defects only (broken
reconcile, data loss beyond stated limits, tests green on broken code)`.

## Fix round (step 7)

One verdict table `verdicts-<round>.md` per review round; the first round holds every review of the frozen
commit, the re-review is the next round. A fresh fixer applies every accepted row, wording included. Commit
the round as one `fixup!` per target commit when the rows touch disjoint files; when a file changed in
several target commits, one `fixup!` for the latest of them, and the verdict table names the earlier targets
(a review fix lands in the later commit; the squash discards fixup bodies). `state set pre-fix-head=<HEAD
before the round>`. Then:

- production code changed: one re-review (above; its accepted findings go to a fresh fixer, whose commits
  get no third review), then `state set qa-run=<n + 1>` and one QA re-check (mode recheck) on the HEAD after
  that fixer's commits: the rows the round touched, named from the verdict tables, plus a smoke set (one
  error row, one OK row, one regression row). A re-check failure counts toward the QA cap: at the cap stop,
  else route it as a QA failure and repeat step 7 on the new commits.
- no production-code change: the step 6 run stands; the A/B table names both hashes.

Squash the fixups after the re-review and the re-check (`git-history.md` "History edits"); the tree stays
equal to the one QA tested. Commit-message rows are applied after the squash, as a reword. Findings still
open after the re-review go to the Approver.

## PR body (step 8)

Gather before the `/mattpocock-skills:pr` call, so nothing comes from memory: the range `<base>..<branch>` in
`<tree>`; the task in one line and the source (`Fixes #N` for a GitHub source); the cause with `file:line`;
the fix and its behaviour changes; the A/B table of the last QA run (master hash, fix hash, rows); red/green
per test and the suite result; known limits; `bundle/` impact. Evidence uses the template's **Before:** /
**After:** bullets with the A/B table below them. Order: the skill's sections first, the cause in the opening
lines of Summary; then the sections of the repo's `pull_request_template.md` the skill's sections do not
cover. A template that asks for one commit: leave the box unchecked and name the commit count.

Style: the body holds the detail the commits leave out. One fact per line, stated as it is now; no narrative
connectives ("so", "already", "now", "still"). Name what is mirrored and what is not, the behaviour lost with
the fix, and the precise scope of each claim ("no execution write", not "no Keycloak call"). Write
`<work>/pr-body.md`; give the Approver its path as a link, not the pasted text.

## Base refresh and merge (step 8)

Before handing over the push commands: `lane.sh rebase <work>` (sandbox disabled). `MOVED=yes` with
`OVERLAP=0`: `lane.sh tests <work> <the branch's packages>`; with an overlap: the full suite. After any
history rewrite (rebase, reword, squash) update every hash in `state.md`, the tracker and `pr-body.md`.

After the PR URL: `state set pr=<url>`; tracker; the Cleanup question. When the user reports the merge:
confirm it with `git -C <root> fetch origin` and the merged commit's patch-id against the final hashes in
`state.md`; mark the task resolved in the
tracker and `state.md`; when the `<root>` working tree is clean, `git -C <root> rebase origin/master`
(sandbox disabled) so local commits stay on top, and check their diff is unchanged.
