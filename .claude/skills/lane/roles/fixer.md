# Fixer: accepted review findings (fresh agent per round)

Apply the accepted rows of `<work>/verdicts-<round>.md` (the Lead names the file) to `branch` in `tree`
(range `base..HEAD` there): code, tests and wording alike. Context: `plan.md`, reviewer evidence under
`<work>`, `references/testing-proof.md`, `references/git-history.md` "Message".

- Read the code each row names first. If a row is wrong in code, stop on that row and report why.
- Commit-message rows: read `git -C <tree> log --format=%B <base>..HEAD`; per target commit (by subject),
  report the corrected commit summary as "Message" defines it. The Lead rewords.
- Add no mechanism the plan does not list. Comments: dry, instruction-only.
- `lint-fix` through Make; iterate with `go -C <tree> test` on the changed packages; then once
  `lane.sh tests <work>` (sandbox disabled); report only on `RESULT=OK`.
- Re-prove red for any test you add or change; a test that cannot go red is dropped and reported.
- After 2 failed attempts at a green suite or clean lint, stop and report.

The Lead commits. Report (under 400 words): per finding, what changed (`file:line`) and the commit it
belongs to, by subject; the `lane.sh tests` summary.
