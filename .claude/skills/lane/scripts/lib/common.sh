# Shared helpers. Sourced by lane.sh after env.sh. Bash 3.2 compatible.
# Exit codes: 0 OK, 1 FAIL (the job ran and failed), 2 STOP (a precondition), 3 usage.

DRY=${DRY:-0}
LOG=${LOG:-/dev/null}

say() { printf '%s\n' "$*"; }
err() { printf '%s\n' "$*" >&2; }
stop() { say "STOP:$1"; [ -z "${2:-}" ] || say "$2"; exit 2; }
fail() { say "FAIL:$1"; [ -z "${2:-}" ] || say "$2"; exit 1; }
usage() { err "usage: $*"; exit 3; }

# Mutating command. Printed under --dry-run, run otherwise.
run() {
  if [ "$DRY" = 1 ]; then say "+ $*"; return 0; fi
  "$@"
}
# Mutating command whose output goes to $LOG.
runlog() {
  if [ "$DRY" = 1 ]; then say "+ $*"; return 0; fi
  printf '+ %s\n' "$*" >>"$LOG"
  "$@" >>"$LOG" 2>&1
}

kube() { kubectl --context "$CONTEXT" "$@"; }
need() { local c; for c in "$@"; do command -v "$c" >/dev/null 2>&1 || stop deps "missing command: $c"; done; }
ts() { date +%Y%m%d%H%M%S; }
utc() { date -u +%Y-%m-%dT%H:%M:%SZ; }
free_port() { python3 -c 'import socket; s=socket.socket(); s.bind(("127.0.0.1",0)); print(s.getsockname()[1]); s.close()'; }
envtest_version() { go -C "$1" list -m -f '{{.Version}}' k8s.io/api | awk -F'[v.]' '{printf "1.%d", $3}'; }
repo_slug() { git -C "$ROOT_DIR" remote get-url origin | sed -E 's#^(git@github\.com:|https://github\.com/)##; s#\.git$##'; }

# Loads <work>/state.md into SLUG ROOT SRC TREE BRANCH BASE NS RELEASE SCOPE KEYCLOAK.
work_dir() {
  [ -n "${1:-}" ] || usage "<work> is required"
  [ -d "$1" ] || stop state "no such directory: $1"
  WORK=$(realpath "$1")
  [ -f "$WORK/state.md" ] || stop state "no state.md in $WORK"
  SLUG=$(state_get slug); ROOT=$(state_get root); SRC=$(state_get src); TREE=$(state_get tree)
  BRANCH=$(state_get branch); BASE=$(state_get base); NS=$(state_get ns); RELEASE=$(state_get release)
  SCOPE=$(state_get scope); KEYCLOAK=$(state_get keycloak)
  INV=$WORK/evidence/master/INVENTORY.txt
  mkdir -p "$WORK/logs"
}

state_get() { sed -n "s/^$1: //p" "$WORK/state.md" | head -1; }
state_set() {
  grep -q "^$1: " "$WORK/state.md" || fail state "unknown key: $1"
  k="$1" v="$2" awk 'index($0, ENVIRON["k"] ": ") == 1 { print ENVIRON["k"] ": " ENVIRON["v"]; next } { print }' \
    "$WORK/state.md" >"$WORK/state.md.tmp" && mv "$WORK/state.md.tmp" "$WORK/state.md"
}
# Number of the step marked doing, or -.
current_step() { grep -E '^\| [0-9]+ .* \| doing \| ' "$WORK/state.md" | head -1 | sed -E 's/^\| ([0-9]+) .*/\1/' | grep . || say -; }

# crd-lock: slugs of live scope:crd items, one per line. Information only; nothing waits on it.
crd_items() {
  local f=$ROOT_DIR/$LOCK_FILE
  [ -f "$f" ] || return 0
  grep -v '^$' "$f" | tr '\n' ' ' | sed 's/ $//' || true
}
crd_show() { local i; i=$(crd_items); say "crd-lock: ${i:--}"; }
# mkdir is the mutex (no flock on macOS); a holder older than 5 s is taken as dead.
crd_mutex_take() {
  local d=$ROOT_DIR/$LOCK_FILE.d i=0
  mkdir -p "$(dirname "$d")"
  until mkdir "$d" 2>/dev/null; do
    i=$((i + 1))
    if [ $i -ge 100 ]; then rmdir "$d" 2>/dev/null || true; i=0; fi
    sleep 0.05
  done
}
crd_mutex_drop() { rmdir "$ROOT_DIR/$LOCK_FILE.d" 2>/dev/null || true; }
crd_register() {
  local f=$ROOT_DIR/$LOCK_FILE rc=0
  if [ "$DRY" = 1 ]; then say "+ register $1 in $f"; return 0; fi
  crd_mutex_take
  grep -qxF -- "$1" "$f" 2>/dev/null || printf '%s\n' "$1" >>"$f" || rc=$?
  crd_mutex_drop
  return $rc
}
crd_release() {
  local f=$ROOT_DIR/$LOCK_FILE rc=0 tmp
  [ -f "$f" ] || return 0
  if [ "$DRY" = 1 ]; then say "+ release $1 from $f"; return 0; fi
  crd_mutex_take
  tmp=$(mktemp "$f.XXXXXX") || rc=$?
  if [ $rc = 0 ]; then
    grep -vxF -- "$1" "$f" >"$tmp" || true
    if [ -s "$tmp" ]; then mv "$tmp" "$f" || rc=$?; else rm -f "$tmp" "$f" || rc=$?; fi
  fi
  crd_mutex_drop
  return $rc
}

inv_append() {
  if [ "$DRY" = 1 ]; then say "+ inventory: $*"; return 0; fi
  mkdir -p "$(dirname "$INV")"
  printf '%s %s\n' "$(utc)" "$*" >>"$INV"
}
# ledger_row <step> <role> <model> <duration_ms> <tool_uses> <tokens>
ledger_row() {
  [ "$DRY" = 1 ] && return 0
  local f=$WORK/agent-timings.md
  [ -f "$f" ] || printf '| when | step | role | model | duration_ms | tool_uses | tokens |\n|---|---|---|---|---|---|---|\n' >"$f"
  printf '| %s | %s | %s | %s | %s | %s | %s |\n' "$(utc)" "$1" "$2" "$3" "$4" "$5" "$6" >>"$f"
}

# kcadm inside the Keycloak pod. Args: <keycloak namespace> <kcadm args...>. Output on stdout.
kc_exec() {
  local kcns=$1; shift
  local user pass cfg args
  user=$(kube -n "$kcns" get secret "$KC_SECRET" -o jsonpath='{.data.username}' | base64 -d)
  pass=$(kube -n "$kcns" get secret "$KC_SECRET" -o jsonpath='{.data.password}' | base64 -d)
  cfg="/tmp/kcadm-$SLUG-$$.config"
  args=$(printf ' %q' "$@")
  kube -n "$kcns" exec "deploy/$KC_DEPLOY" -c "$KC_CONTAINER" -- sh -c \
    "C=$cfg; K=/opt/keycloak/bin/kcadm.sh; \$K config credentials --config \$C --server http://localhost:8080 --realm master --user $(printf %q "$user") --password $(printf %q "$pass") >/dev/null 2>&1 && \$K$args --config \$C; rc=\$?; rm -f \$C; exit \$rc"
}
# Realms of this item on the shared or dedicated Keycloak, one per line.
kc_item_realms() {
  kc_exec "$1" get realms --fields realm | grep -o "\"$SLUG-[^\"]*\"" | tr -d '"' || true
}
