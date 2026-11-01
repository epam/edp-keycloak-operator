# intake: preflight, claim the slug, create the worktrees, build the shared tools.
# Rerun for an item that has state.md with the same branch: the item checks (branch, worktree, namespace, PR)
# are skipped and existing worktrees and tools are kept. --adopt: accept matching leftovers of an old run.

. "$HERE/cmd/state.sh"

intake_help() {
  cat <<EOF
lane.sh intake <slug> <branch> [--issue N] [--waive-pr] [--adopt] [--base REF] [--source KEY] [--task TEXT]
  slug: [a-z0-9-], 1 to 20 chars, no leading or trailing -.   Needs the sandbox disabled (fetch, gh).
  --adopt: reuse an existing branch, worktree or namespace of that name instead of stopping.
  --base REF: the upstream ref the item starts from and rebases onto (default origin/master; a release
  branch, or a commit to replay a fixed defect).
  STOP (exit 2): slug, deps, kubelock, origin, slug-taken, branch, worktree, namespace, testbed, pr, issue
  kubelock: KUBECONFIG must be the krci-kubelock pin of the context in env.sh (/config -> krci-kubelock).
  FAIL (exit 1): worktree, tools; read the log, fix, rerun the same command (idempotent).
  Prints: BASE= UPSTREAM= WORK= SRC= TREE= BRANCH= NS= RELEASE= ISSUE= CRD_ITEMS= LOG=
  CRD_ITEMS: live scope:crd items (information only).
EOF
}

cmd_intake() {
  local slug="" branch="" issue="" waive=0 adopt=0 source="" task="" resume=0 upstream=origin/master start=$SECONDS
  while [ $# -gt 0 ]; do
    case "$1" in
      --issue) issue=${2:?}; shift 2;;
      --waive-pr) waive=1; shift;;
      --adopt) adopt=1; shift;;
      --base) upstream=${2:?}; shift 2;;
      --source) source=${2:?}; shift 2;;
      --task) task=${2:?}; shift 2;;
      -h|--help) intake_help; return 0;;
      -*) usage "unknown option $1";;
      *) if [ -z "$slug" ]; then slug=$1; elif [ -z "$branch" ]; then branch=$1; else usage "too many arguments"; fi; shift;;
    esac
  done
  [ -n "$slug" ] && [ -n "$branch" ] || usage "lane.sh intake <slug> <branch> [options]"
  printf '%s' "$slug" | grep -Eq '^[a-z0-9]([a-z0-9-]{0,18}[a-z0-9])?$' || stop slug "slug must match [a-z0-9]([a-z0-9-]{0,18}[a-z0-9])?"

  need "${DEPS_CMDS[@]}"
  local plugins=$HOME/.claude/plugins/installed_plugins.json missing="" p
  for p in "${DEPS_PLUGINS[@]}"; do grep -q "\"$p@" "$plugins" 2>/dev/null || missing="$missing plugin:$p"; done
  for p in "${DEPS_SKILLS[@]}"; do [ -f "$ROOT_DIR/.claude/skills/$p/SKILL.md" ] || missing="$missing skill:$p"; done
  [ -z "$missing" ] || stop deps "missing:$missing"
  # krci-kubelock (plugin krci-triage) pins KUBECONFIG to one context; its file name is the context with
  # characters outside [A-Za-z0-9_.-] replaced by _.
  [ "${KUBECONFIG:-}" = "$HOME/.kube/krci-kubelock/${CONTEXT//[^A-Za-z0-9_.-]/_}.yaml" ] \
    || stop kubelock "kube access is not pinned to $CONTEXT; /config -> krci-kubelock: on, context $CONTEXT"
  git -C "$ROOT_DIR" remote get-url origin >/dev/null 2>&1 || stop origin "no origin remote in $ROOT_DIR"
  local repo; repo=$(repo_slug)

  local work=$ROOT_DIR/$WORK_DIR/$slug src=$ROOT_DIR/$WORK_DIR/$slug/master-src tree=$ROOT_DIR/$WORK_DIR/$slug/tree
  if [ -d "$work" ]; then
    [ -f "$work/state.md" ] && [ "$(sed -n 's/^branch: //p' "$work/state.md")" = "$branch" ] \
      || stop slug-taken "$work exists; choose another slug, or tear the old item down"
    resume=1
  fi
  local skip_item_checks=0
  [ $resume = 1 ] || [ $adopt = 1 ] && skip_item_checks=1

  if [ "$DRY" = 1 ]; then say "+ git -C $ROOT_DIR fetch origin"; else git -C "$ROOT_DIR" fetch origin >/dev/null 2>&1 || stop origin "git fetch origin failed"; fi
  [ $resume = 0 ] || upstream=$(sed -n 's/^upstream: //p' "$work/state.md")
  local base; base=$(git -C "$ROOT_DIR" rev-parse --verify "$upstream^{commit}" 2>/dev/null) || stop origin "unknown base ref: $upstream"

  local branch_exists=0; [ -z "$(git -C "$ROOT_DIR" branch --list "$branch")" ] || branch_exists=1
  if [ $skip_item_checks = 0 ]; then
    [ $branch_exists = 0 ] || stop branch "branch $branch exists (--adopt reuses it)"
    ! git -C "$ROOT_DIR" worktree list --porcelain | grep -q "^worktree $work" || stop worktree "a worktree under $work exists (--adopt reuses it)"
    ! kube get ns "$slug" >/dev/null 2>&1 || stop namespace "namespace $slug exists on $CONTEXT (--adopt reuses it)"
  fi
  local d
  for d in "${TESTBED_DEPLOYS[@]}"; do
    kube -n "${d%/*}" rollout status "deploy/${d#*/}" --timeout=60s >/dev/null 2>&1 || stop testbed "deploy/${d#*/} in ${d%/*} is not ready"
  done
  if [ -n "$issue" ] && [ $waive = 0 ] && [ $skip_item_checks = 0 ]; then
    local prs; prs=$(gh pr list -R "$repo" --state open --search "#$issue" --json number,url --jq '.[] | "\(.number) \(.url)"' 2>/dev/null || true)
    [ -z "$prs" ] || stop pr "open PR for #$issue: $prs (pass --waive-pr to accept it)"
  fi

  if [ "$DRY" = 1 ]; then
    local existing=""; [ $branch_exists = 0 ] || existing=" (existing branch)"
    say "+ mkdir $work; state init; worktree add --detach $src $base; worktree add$existing $tree $branch at $base"
    say "+ make -C $src LOCALBIN=$ROOT_DIR/bin ${TOOLS[*]}; setup-envtest use"
    say "BASE=$base"
    say "CRD_ITEMS=$(crd_items)"
    return 0
  fi

  if [ $resume = 0 ]; then
    mkdir -p "$ROOT_DIR/$WORK_DIR"
    mkdir "$work" || stop slug-taken "$work appeared; choose another slug"
    state_init "$work" "slug=$slug" "root=$ROOT_DIR" "work=$work" "branch=$branch" "upstream=$upstream" "base=$base" "src=$src" "tree=$tree" >/dev/null
  fi
  WORK=$work
  mkdir -p "$WORK/logs"
  LOG=$WORK/logs/intake.log
  [ -z "$source" ] || state_set source "$source"
  [ -z "$task" ] || state_set task "$task"
  [ $resume = 1 ] || state_step 1 doing "base=$base" >/dev/null

  local issue_file=-
  if [ -n "$issue" ]; then
    issue_file=$WORK/issue.json
    gh issue view "$issue" -R "$repo" --json number,title,body,labels,comments >"$issue_file" 2>>"$LOG" || stop issue "gh issue view $issue failed; log: $LOG"
    local waived=""; [ $waive = 0 ] || waived=" (open PR waived)"
    state_set issue "#$issue $repo$waived"
  fi

  if [ ! -d "$src" ]; then
    runlog git -C "$ROOT_DIR" worktree add --detach "$src" "$base" || fail worktree "worktree add $src failed; log: $LOG"
  fi
  if [ -d "$tree" ]; then
    [ "$(git -C "$tree" branch --show-current)" = "$branch" ] || fail worktree "$tree is not on $branch"
  elif [ $branch_exists = 1 ]; then
    runlog git -C "$ROOT_DIR" worktree add "$tree" "$branch" || fail worktree "worktree add $tree on existing $branch failed; log: $LOG"
  else
    runlog git -C "$ROOT_DIR" worktree add -b "$branch" "$tree" "$base" || fail worktree "worktree add $tree failed; log: $LOG"
  fi
  grep -qx "$WORK_DIR/" "$ROOT_DIR/.git/info/exclude" 2>/dev/null || printf '%s/\n' "$WORK_DIR" >>"$ROOT_DIR/.git/info/exclude"

  local attempt=1
  while :; do
    if make -C "$src" LOCALBIN="$ROOT_DIR/bin" "${TOOLS[@]}" >>"$LOG" 2>&1; then break; fi
    if [ $attempt -lt 2 ] && tail -20 "$LOG" | grep -qE 'text file busy|rename|index\.lock'; then attempt=2; sleep 10; continue; fi
    fail tools "tool build failed; log: $LOG"
  done
  local ver; ver=$(envtest_version "$src")
  "$ROOT_DIR/bin/setup-envtest" use "$ver" --bin-dir "$ROOT_DIR/bin" -p path >>"$LOG" 2>&1 || fail tools "setup-envtest use $ver failed; log: $LOG"

  [ $resume = 1 ] || state_set base "$base"
  ledger_row 1 script:intake - $(( (SECONDS - start) * 1000 )) - -
  say "BASE=$(state_get base)"
  say "UPSTREAM=$upstream"
  say "WORK=$WORK"
  say "SRC=$src"
  say "TREE=$tree"
  say "BRANCH=$branch"
  say "NS=$slug"
  say "RELEASE=$RELEASE_PREFIX$slug"
  say "ISSUE=$issue_file"
  say "CRD_ITEMS=$(crd_items)"
  say "LOG=$LOG"
}
