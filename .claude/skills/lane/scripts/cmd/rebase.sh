# rebase: fetch origin; when origin/master moved, rebase <tree> onto it and move <src> to the new base.

rebase_help() {
  cat <<EOF
lane.sh rebase <work>
  git fetch origin (needs the sandbox disabled). When the upstream ref of state.md (default origin/master)
  moved: rebase <tree> onto it, move <src> to the new commit, update base in state.md. A conflict aborts the
  rebase and reports FAIL:conflict.
  Prints BASE= MOVED=yes|no OVERLAP=<n>: files changed upstream since the old base that the branch also touches.
EOF
}

cmd_rebase() {
  case "${1:-}" in -h|--help) rebase_help; return 0;; esac
  work_dir "${1:-}"
  LOG=$WORK/logs/rebase-$(ts).log; : >"$LOG"
  local upstream; upstream=$(state_get upstream); [ -n "$upstream" ] || upstream=origin/master
  if [ "$DRY" = 1 ]; then
    say "+ git -C $ROOT fetch origin; git -C $TREE rebase $upstream; git -C $SRC checkout --detach $upstream"
    return 0
  fi
  local start=$SECONDS
  git -C "$ROOT" fetch origin >>"$LOG" 2>&1 || fail fetch "git fetch origin failed; log: $LOG"
  local new; new=$(git -C "$ROOT" rev-parse --verify "$upstream^{commit}" 2>/dev/null) || fail upstream "unknown upstream ref: $upstream"
  if [ "$new" = "$BASE" ]; then
    ledger_row "$(current_step)" script:rebase - $(( (SECONDS - start) * 1000 )) - -
    say "BASE=$BASE"; say "MOVED=no"; say "OVERLAP=0"; return 0
  fi
  [ -z "$(git -C "$TREE" status --porcelain --untracked-files=no)" ] || stop dirty "$TREE has uncommitted changes"

  git -C "$TREE" diff --name-only "$BASE" "$new" | sort >"$WORK/logs/rebase-master.txt"
  git -C "$TREE" diff --name-only "$BASE" HEAD | sort >"$WORK/logs/rebase-branch.txt"
  local overlap; overlap=$(comm -12 "$WORK/logs/rebase-master.txt" "$WORK/logs/rebase-branch.txt" | wc -l | tr -d ' ')

  if ! git -C "$TREE" rebase "$new" >>"$LOG" 2>&1; then
    local conflicted; conflicted=$(git -C "$TREE" diff --name-only --diff-filter=U | tr '\n' ' ')
    git -C "$TREE" rebase --abort >>"$LOG" 2>&1 || true
    fail conflict "rebase onto $new conflicts in: $conflicted; aborted; log: $LOG"
  fi
  git -C "$SRC" checkout --detach "$new" >>"$LOG" 2>&1 || fail worktree "could not move $SRC to $new; log: $LOG"
  state_set base "$new"
  ledger_row "$(current_step)" script:rebase - $(( (SECONDS - start) * 1000 )) - -
  say "BASE=$new"
  say "MOVED=yes"
  say "OVERLAP=$overlap"
  say "log: $LOG"
}
