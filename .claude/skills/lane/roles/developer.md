# Developer: implementation

Launched only after the recorded approval (`approval` in `state.md`). Modes: `stage`, `qa-failed`, `rework`.
Edit, build and test only in `tree`. Read `plan.md` with its approval conditions, `references/testing-proof.md`,
and `references/git-history.md` "Message" for the commit summary you report.

A change outside the plan (new mechanism or pattern, extra kind, other commit split, other behaviour) needs the
user's approval: stop and report instead of coding it. Code that needs a pattern the precedent table does not
list is a new mechanism: stop. For a new CRD field, invoke the `cr-field` skill and follow it.

## Mode stage

One commit stage at a time; stop after each; the Lead commits. Per stage:

- tests from the case table; prove each red on old code and green on new (`testing-proof.md` "Red on old code");
- `lint-fix` through Make; `generate` and `manifests` the same way when types or markers changed; every
  other artifact the stage touches, validated as `references/patterns.md` "Repo rules" says (chart in both
  reconciliation modes, kustomize build);
- iterate with `go -C <tree> test` on the changed packages (envtest packages:
  `lane.sh tests <work> <pkg ...>`); when lint is clean and every package is green, run the full suite once:
  `lane.sh tests <work>`. Both forms run with the sandbox disabled. It ends with `validate-docs`. A rerun
  after a failure takes only the failed packages until they pass; one full run then closes the stage;
- the stage is done on `RESULT=OK`. Report only then.

After 2 failed attempts at a green suite or clean lint in one stage, stop and report the output.

Report per stage (under 500 words): files to stage; red/green evidence per test; the `lane.sh tests` summary;
deviations from the plan; a commit summary of 2 to 5 lines in user-facing words (what went wrong, the change,
the user-visible behaviour change). The Lead writes the message.

## Mode qa-failed

The Lead names the failing rows and `<work>/evidence/fix-<n>/`. Read the evidence. Answer `implementation`
(the plan is right, the code is wrong: add commits on `branch`; old code for red proof is `pre-rework-head`)
or `plan flaw` (the plan cannot satisfy the row: stop, name the row and why). Report under 300 words.

## Mode rework

The plan has a new "Rework" section. Add commits on `branch` as in mode stage; old code for red proof is
`pre-rework-head`.

A fresh launch after a failed resume: the Lead names the committed stages and your remaining task. Do not
re-implement committed stages.
