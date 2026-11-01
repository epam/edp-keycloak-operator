# build: operator image from <src> (master) or <tree> (fix), loaded into the kind node and verified by id.

build_help() {
  cat <<EOF
lane.sh build <work> master|fix [--run N]
  Tag: <slug>-master-<ts> or <slug>-fix-<n>-<ts>; n from --run or state qa-run. Arch from go env GOARCH.
  Appends the image to INVENTORY.txt. Prints TAG= IMAGE= log.
EOF
}

cmd_build() {
  case "${1:-}" in -h|--help) build_help; return 0;; esac
  work_dir "${1:-}"; shift
  local which=${1:-}; [ $# -eq 0 ] || shift
  local runn=""
  [ "${1:-}" != --run ] || runn=${2:?N}
  local bt tag arch start=$SECONDS
  arch=$(go env GOARCH)
  case "$which" in
    master) bt=$SRC; tag="$SLUG-master-$(ts)";;
    fix) bt=$TREE; [ -n "$runn" ] || runn=$(state_get qa-run); tag="$SLUG-fix-$runn-$(ts)";;
    *) usage "lane.sh build <work> master|fix [--run N]";;
  esac
  [ -d "$bt" ] || stop worktree "no build tree at $bt"
  LOG=$WORK/logs/build-$tag.log; : >"$LOG"

  runlog make -C "$bt" LOCALBIN="$ROOT/bin" build HOST_OS=linux HOST_ARCH="$arch" || fail build "make build failed; log: $LOG"
  runlog docker build --build-arg TARGETARCH="$arch" -t "$IMAGE_REPO:$tag" "$bt" || fail build "docker build failed; log: $LOG"
  runlog kind load docker-image "$IMAGE_REPO:$tag" --name "$KIND_CLUSTER" || fail build "kind load failed; log: $LOG"
  if [ "$DRY" != 1 ]; then
    local want got
    want=$(docker image inspect "$IMAGE_REPO:$tag" --format '{{.Id}}')
    got=$(docker exec "$KIND_CLUSTER-control-plane" crictl inspecti -o json "$IMAGE_REPO:$tag" | jq -r '.status.id')
    [ "$want" = "$got" ] || fail build "image id differs: docker $want, node $got"
  fi
  inv_append "image: $IMAGE_REPO:$tag ($which${runn:+ run $runn}) from $bt"
  ledger_row "$(current_step)" "script:build-$which" - $(( (SECONDS - start) * 1000 )) - -
  say "TAG=$tag"
  say "IMAGE=$IMAGE_REPO:$tag"
  say "log: $LOG"
}
