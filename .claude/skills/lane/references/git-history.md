# Commits and history

Every commit message except a fixup's comes from `/krci-general:commit`, called by the Lead in its own turn.
The commit is the summary a reader of `git log` needs; the detail lives in `<work>/pr-body.md`.

## Message

```
fix(keycloakclient): skip the client update when Keycloak matches the spec

- A converged client got one update and one admin event on each reconcile.
- The operator compares Keycloak with the spec and writes only on a difference.
- A spec edit, a Secret rotation or an upgrade always causes one write.
- The next reconcile repairs manual changes in Keycloak, except masked values.
```

- Subject `type(scope): summary`. `fix` for a defect task, `feat` for a feature task. `scope` is the
  lowercase kind. Imperative, no period, at most 80 characters. The PR title is the first subject.
- Body: 2 to 5 `- ` bullets, no section headers, lines at most 100 characters. First bullet: what went
  wrong as a user sees it (feature: what was missing). Then the change, the user-visible behaviour change,
  and a limit only when a user must act on it.
- Words: user-facing terms only: kinds, spec fields, Keycloak objects, observable effects. No function,
  type, variable or file names, no error texts, no test names, no counts of calls. One fact per bullet,
  stated as it is now.
- Footer: `BREAKING CHANGE: <what the user must do>` when omitting a field changes behaviour or an exported
  Go type changes. `Signed-off-by` comes from `git commit -s`. No co-author trailer.

## Procedure (Lead)

1. Stage by path: `git -C <tree> add <paths>`. Never `.work/`.
2. Skill `krci-general:commit`, args: `worktree <tree>: run every git command as git -C <tree>; diff:
   git -C <tree> diff --cached; summary: <the commit summary from the Developer or Fixer report>`.
3. Write it to `<work>/commit-msg.txt` with the Write tool. Check it against the rules above;
   `awk 'length > (NR == 1 ? 80 : 100)' <work>/commit-msg.txt` prints every line over its limit. Cut, never add.
4. `git -C <tree> commit -s -F <work>/commit-msg.txt`. Skip the skill's closing offer.

**Fixup** (step 7): `git -C <tree> commit -s --fixup=<hash> -- <paths>`, no skill call and no body (the
squash discards it; the verdict table names the earlier targets). One staged index feeds several fixups.

**Reword** (commit-message rows, step 7): squash first. Per commit with accepted rows, call the skill with
diff `git -C <tree> show <hash>` and the corrected summary; write each message to `<work>/commit-msg-<n>.txt`;
reword in one rebase with an `exec git commit --amend -s -F <file>` line after each pick.

## History edits

- Always in `<tree>`, never with prompts: `GIT_SEQUENCE_EDITOR=<script> GIT_EDITOR=<script> git -C <tree> rebase -i`.
  Todo lines read `pick <hash> # <subject>`: match on `# `. Use perl `s{}{}` because paths contain `/`.
- `exec git checkout <final> -- <file> && git commit --amend --no-edit` after a pick moves a file's change
  into that commit. Compare `HEAD^{tree}` before and after a squash: the tree QA tested must not change.
- `git rebase -i --autosquash --autostash` runs with an unrelated dirty tracked file and restores it.
- Two commits touch one function: a rename found in review lands in the later commit; the body says so.
- After splitting fixups across commits, build or vet the first commit on its own in a temporary
  `git worktree` under `<work>`; remove it afterwards.
- Evidence file lists use `git show --name-only`, not `--stat` (it truncates paths).
- Name commits by subject, not hash, in messages to agents and the user: squash and reword change hashes.
