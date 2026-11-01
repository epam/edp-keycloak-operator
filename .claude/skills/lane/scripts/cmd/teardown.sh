# teardown: remove the item from the cluster and the base worktree. The namespace itself is the user's command.
# Order: CRs (the operator deletes the realms), leftover realms and master users, helm release, namespace objects,
# cluster-scoped objects named with <ns> or <slug>, images by tag, CRDs by the shared rule, crd-lock, worktrees.

teardown_help() {
  cat <<EOF
lane.sh teardown <work> [--all [--discard]]
  Removes everything INVENTORY.txt and state.md name, except the namespace (printed as the user's command).
  Images: every tag <slug>-* on docker and on the kind node.
  --all also removes <tree> and <branch>. STOP:unsaved while <tree> has uncommitted or untracked files;
  STOP:unpushed while refs/remotes/origin/<branch> does not contain the branch tip. Both checks run before
  anything is removed. --discard skips them and forces the worktree removal; the hook leaves it to a prompt.
  CRDs: a crd item restores them from <src> only when no other live item namespace exists.
  Needs the sandbox disabled: the worktree removal deletes .vscode/*.json, which the sandbox protects.
  Leftovers: exit 1 (FAIL:leftovers); fix the cause named in the log and rerun.
EOF
}

# --all without --discard: STOP unless <tree> is clean and its HEAD and <branch> are contained in origin/<branch>.
teardown_guard() {
  local st tip head
  if [ -d "$TREE" ]; then
    st=$(git -C "$TREE" status --porcelain --untracked-files=all) || stop unsaved "cannot read the status of $TREE"
    [ -z "$st" ] || stop unsaved "$TREE has uncommitted or untracked files; commit them, or rerun with --discard"
    if head=$(git -C "$TREE" rev-parse HEAD 2>/dev/null); then
      git -C "$ROOT" merge-base --is-ancestor "$head" "refs/remotes/origin/$BRANCH" 2>/dev/null \
        || stop unpushed "origin/$BRANCH does not contain the HEAD of $TREE ($head); push first, or rerun with --discard"
    fi
  fi
  if tip=$(git -C "$ROOT" rev-parse --verify "refs/heads/$BRANCH" 2>/dev/null); then
    git -C "$ROOT" merge-base --is-ancestor "$tip" "refs/remotes/origin/$BRANCH" 2>/dev/null \
      || stop unpushed "origin/$BRANCH does not contain $BRANCH ($tip); push first, or rerun with --discard"
  fi
}

cmd_teardown() {
  case "${1:-}" in -h|--help) teardown_help; return 0;; esac
  work_dir "${1:-}"; shift
  local all=0 discard=0 a
  for a in "$@"; do
    case "$a" in --all) all=1;; --discard) discard=1;; *) usage "lane.sh teardown <work> [--all [--discard]]";; esac
  done
  [ $discard = 0 ] || [ $all = 1 ] || usage "--discard needs --all"
  [ $all = 0 ] || [ $discard = 1 ] || teardown_guard
  LOG=$WORK/logs/teardown-$(ts).log; : >"$LOG"
  local start=$SECONDS left="" removed="" kcns=$KC_NS kind name obj img r id
  [ "$KEYCLOAK" != dedicated ] || kcns=$NS
  [ -f "$INV" ] || say "note: no $INV; cluster steps use the names in state.md"

  if kube get ns "$NS" >/dev/null 2>&1; then
    for kind in "${CR_DELETE_ORDER[@]}"; do
      kube get crd "$kind.$CRD_GROUP" >/dev/null 2>&1 || continue
      [ "$(kube -n "$NS" get "$kind" -o name 2>/dev/null | wc -l | tr -d ' ' || true)" != 0 ] || continue
      runlog kube -n "$NS" delete "$kind" --all --wait=true --timeout=180s && removed="$removed $kind" || left="$left $kind"
    done
    if [ "$KEYCLOAK" != dedicated ] && [ "$DRY" != 1 ] && kube -n "$kcns" get "deploy/$KC_DEPLOY" >/dev/null 2>&1; then
      for r in $(kc_item_realms "$kcns"); do
        kc_exec "$kcns" delete "realms/$r" >>"$LOG" 2>&1 && removed="$removed realm:$r" || left="$left realm:$r"
      done
      for id in $(kc_exec "$kcns" get users -r master -q "username=$SLUG-" --fields id,username 2>>"$LOG" | jq -r --arg p "$SLUG-" '.[] | select(.username | startswith($p)) | .id' 2>/dev/null || true); do
        kc_exec "$kcns" delete "users/$id" -r master >>"$LOG" 2>&1 && removed="$removed master-user:$id" || left="$left master-user:$id"
      done
    fi
    if helm list --kube-context "$CONTEXT" -n "$NS" --short 2>/dev/null | grep -qx "$RELEASE"; then
      runlog helm uninstall "$RELEASE" --kube-context "$CONTEXT" -n "$NS" --wait && removed="$removed helm:$RELEASE" || left="$left helm:$RELEASE"
    fi
    runlog kube -n "$NS" delete all,secret,configmap,serviceaccount,role,rolebinding --all --wait=true --timeout=120s || left="$left objects-in-$NS"
    if kube get crd certificates.cert-manager.io >/dev/null 2>&1; then
      runlog kube -n "$NS" delete certificate,issuer --all --wait=true --timeout=60s || left="$left certs-in-$NS"
    fi
  else
    say "namespace $NS: absent"
  fi

  for obj in \
    "mutatingwebhookconfiguration/$DEPLOY_NAME-mutating-webhook-configuration-$NS" \
    "validatingwebhookconfiguration/$DEPLOY_NAME-validating-webhook-configuration-$NS" \
    "clusterrole/edp-$NS-webhook-clusterrole" "clusterrolebinding/edp-$NS-webhook-clusterrolebinding" \
    "clusterrole/edp-$NS-clusterrole" "clusterrolebinding/edp-$NS-servicebindings" \
    "clusterrole/$DEPLOY_NAME-metrics-scraper-$SLUG" "clusterrolebinding/$DEPLOY_NAME-metrics-scraper-$SLUG"; do
    kube get "$obj" >/dev/null 2>&1 || continue
    runlog kube delete "$obj" && removed="$removed $obj" || left="$left $obj"
  done
  for kind in clusterkeycloakrealms clusterkeycloaks; do
    kube get crd "$kind.$CRD_GROUP" >/dev/null 2>&1 || continue
    for name in $(kube get "$kind" -o name 2>/dev/null | grep "/$SLUG-" || true); do
      runlog kube delete "$name" && removed="$removed $name" || left="$left $name"
    done
  done

  # Images by tag prefix on both sides plus INVENTORY: a build that failed after `kind load` has no inventory row.
  local imgs
  imgs=$( {
    docker image ls "$IMAGE_REPO" --format '{{.Repository}}:{{.Tag}}' 2>/dev/null | grep ":$SLUG-" || true
    docker exec "$KIND_CLUSTER-control-plane" crictl images -o json 2>/dev/null | jq -r '.images[].repoTags[]' \
      | sed 's#^docker.io/##' | grep "^$IMAGE_REPO:$SLUG-" || true
    [ ! -f "$INV" ] || sed -n 's/^.* image: \([^ ]*\).*/\1/p' "$INV"
  } | sort -u )
  for img in $imgs; do
    if docker exec "$KIND_CLUSTER-control-plane" crictl inspecti "$img" >/dev/null 2>&1; then
      runlog docker exec "$KIND_CLUSTER-control-plane" crictl rmi "$img" && removed="$removed node:$img" || left="$left node:$img"
    fi
    if docker image inspect "$img" >/dev/null 2>&1; then
      runlog docker rmi "$img" && removed="$removed docker:$img" || left="$left docker:$img"
    fi
  done
  [ ! -f "$INV" ] || { grep ' timing: ' "$INV" | sed 's/^/manual restore: /' || true; }

  if [ "$SCOPE" = crd ]; then
    local others
    others=$(kube get ns -l "$LABEL" -o name 2>/dev/null | grep -v "^namespace/$NS\$" || true)
    if [ -z "$others" ]; then
      runlog kube apply --server-side --force-conflicts -f "$SRC/$CRD_DIR" && say "crds: restored from $SRC" || left="$left crds"
    else
      say "crds: left as installed; other live items: $(printf '%s ' $others)"
    fi
  fi
  case " $(crd_items) " in
    *" $SLUG "*)
      crd_release "$SLUG"
      say "crd-lock: released"
      [ "$DRY" = 1 ] || state_set crd-lock -
      ;;
  esac

  # A directory git no longer lists is the remains of a denied delete: git drops the entry before the files go.
  local force=""; [ $discard = 0 ] || force=--force
  remove_worktree() {
    local dir=$1
    [ -d "$dir" ] || return 0
    case "$dir" in "$WORK"/*) ;; *) left="$left outside-work:$dir"; return 0;; esac
    if git -C "$ROOT" worktree list --porcelain | grep -qxF "worktree $dir"; then
      runlog git -C "$ROOT" worktree remove ${force:+"$force"} "$dir" && removed="$removed worktree:$dir" || left="$left worktree:$dir"
    else
      runlog rm -rf "$dir" && removed="$removed dir:$dir" || left="$left dir:$dir"
    fi
  }
  remove_worktree "$SRC"
  if [ $all = 1 ]; then
    [ $discard = 1 ] || teardown_guard
    remove_worktree "$TREE"
    if [ -n "$(git -C "$ROOT" branch --list "$BRANCH")" ]; then
      runlog git -C "$ROOT" branch -D "$BRANCH" && removed="$removed branch:$BRANCH" || left="$left branch:$BRANCH"
    fi
  fi
  local suffix=""; [ $all = 0 ] || suffix=" (--all)"
  [ "$DRY" = 1 ] || state_set installed "torn down $(utc)$suffix"
  ledger_row "$(current_step)" "script:teardown" - $(( (SECONDS - start) * 1000 )) - -

  say "removed:${removed:- nothing}"
  [ -z "$left" ] || say "leftovers:$left (see $LOG)"
  say "log: $LOG"
  if kube get ns "$NS" >/dev/null 2>&1; then say "user command: ! kubectl --context $CONTEXT delete ns $NS"; fi
  [ -z "$left" ] || fail leftovers "a worktree or dir leftover with 'Operation not permitted' in the log: rerun with the sandbox disabled"
}
