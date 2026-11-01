# tests: the suite or named packages against the shared Keycloak fixture, then validate-docs.
# Both forms need the sandbox disabled (envtest cache, kill of the port-forward).

tests_help() {
  cat <<EOF
lane.sh tests <work> [pkg ...]
  No packages: make test (full suite). Packages: go test <pkg ...> in <tree>, with envtest assets.
  Both forms need the sandbox disabled. Parallel runs are safe: the tests name their realms with a pid and time suffix.
  Prints: RESULT=OK|FAIL, packages ok/fail, tests-exit, validate-docs exit, first-fail, log, port.
  FAIL when a test fails, validate-docs fails, or validate-docs changes a tracked file.
EOF
}

cmd_tests() {
  case "${1:-}" in -h|--help) tests_help; return 0;; esac
  work_dir "${1:-}"; shift
  local start=$SECONDS n
  n=$(ls "$WORK/logs"/tests-*.log 2>/dev/null | wc -l | tr -d ' ' || true); n=$((n + 1))
  LOG=$WORK/logs/tests-$n.log; : >"$LOG"

  kube -n "$KC_NS" get "deploy/$KC_DEPLOY" >/dev/null 2>&1 || stop fixture "deploy/$KC_DEPLOY missing in namespace $KC_NS"
  kube -n "$KC_NS" rollout status "deploy/$KC_DEPLOY" --timeout=300s >>"$LOG" 2>&1 || stop fixture "deploy/$KC_DEPLOY not ready; log: $LOG"

  if [ "$DRY" = 1 ]; then
    say "+ kubectl --context $CONTEXT -n $KC_NS port-forward svc/$KC_SVC <port>:$KC_PORT &"
    if [ $# -eq 0 ]; then say "+ make -C $TREE LOCALBIN=$ROOT/bin test $TEST_URL_VAR=http://localhost:<port>"
    else say "+ KUBEBUILDER_ASSETS=<envtest> $TEST_URL_VAR=http://localhost:<port> go -C $TREE test $*"; fi
    say "+ make -C $TREE LOCALBIN=$ROOT/bin validate-docs"
    return 0
  fi

  local port pf_pid i=0
  port=$(free_port)
  kube -n "$KC_NS" port-forward "svc/$KC_SVC" "$port:$KC_PORT" >"$WORK/logs/port-forward-$n.log" 2>&1 &
  pf_pid=$!
  trap 'kill $pf_pid 2>/dev/null || true; wait $pf_pid 2>/dev/null || true' EXIT
  until [ "$(curl -s -o /dev/null -w '%{http_code}' "http://localhost:$port/realms/master" || true)" = 200 ]; do
    i=$((i + 1)); [ $i -lt 30 ] || fail fixture "no HTTP 200 from /realms/master on port $port"
    sleep 1
  done

  local url="http://localhost:$port" rc=0
  if [ $# -eq 0 ]; then
    make -C "$TREE" LOCALBIN="$ROOT/bin" test "$TEST_URL_VAR=$url" >>"$LOG" 2>&1 || rc=$?
  else
    local ver assets
    ver=$(envtest_version "$TREE")
    assets=$("$ROOT/bin/setup-envtest" use "$ver" --bin-dir "$ROOT/bin" -p path)
    env KUBEBUILDER_ASSETS="$assets" "$TEST_URL_VAR=$url" go -C "$TREE" test "$@" >>"$LOG" 2>&1 || rc=$?
  fi

  local before after docs_rc=0
  before=$(git -C "$TREE" status --porcelain)
  make -C "$TREE" LOCALBIN="$ROOT/bin" validate-docs >>"$LOG" 2>&1 || docs_rc=$?
  after=$(git -C "$TREE" status --porcelain)
  kill $pf_pid 2>/dev/null || true
  wait $pf_pid 2>/dev/null || true
  trap - EXIT

  local ok failc first result=OK
  ok=$(grep -c '^ok ' "$LOG" || true)
  failc=$(grep -c -E '^(FAIL|--- FAIL)' "$LOG" || true)
  first=$(grep -m1 -E '^--- FAIL|^FAIL[[:space:]]|^FAIL$|^panic:' "$LOG" || true)
  [ $rc -eq 0 ] && [ $docs_rc -eq 0 ] && [ "$before" = "$after" ] || result=FAIL
  ledger_row "$(current_step)" "script:tests-$n" - $(( (SECONDS - start) * 1000 )) - -

  say "RESULT=$result"
  say "packages: ok=$ok fail=$failc"
  say "tests-exit: $rc"
  say "validate-docs: $docs_rc"
  [ "$before" = "$after" ] || say "tree-drift: validate-docs changed tracked files; run it after git add or commit"
  [ -z "$first" ] || say "first-fail: $first"
  say "log: $LOG"
  say "port: $port"
  [ "$result" = OK ]
}
