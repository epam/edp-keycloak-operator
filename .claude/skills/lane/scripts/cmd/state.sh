# state: the run variables of one item, in <work>/state.md. Fixed key set; `set` rejects unknown keys.

state_help() {
  cat <<EOF
lane.sh state <op> <work> ...
  init  <work> key=value ...     create state.md (slug=, root=, work= required)
  get   <work> <key>             print one value
  set   <work> key=value ...     update values; a value may contain spaces
  step  <work> <n> <status> [values]
                                 status: todo|doing|done|stopped|parked; values keeps the old text when omitted
  agent <work> <role>=<id>       append to the agents list
  ledger <work> <step> <role> <model> <duration_ms> <tool_uses> <tokens>
                                 append a row to agent-timings.md; duration_ms is a positive integer
  totals <work>                  agent-timings.md sums: one line per agent row, then agents and scripts
  scope <work> <path> ...        set scope from the artifact paths: crd when one starts with a CRD path, else code
  show  <work>                   print state.md
Keys: task source variant tracker slug root work src tree upstream base branch ns release scope keycloak
      crd-lock qa-run pre-rework-head pre-fix-head approval installed agents issue pr
EOF
}

state_init() {
  local dir=$1; shift
  local slug="" root="" work="" kv
  for kv in "$@"; do case "$kv" in slug=*) slug=${kv#*=};; root=*) root=${kv#*=};; work=*) work=${kv#*=};; esac; done
  [ -n "$slug" ] && [ -n "$root" ] && [ -n "$work" ] || usage "state init <work> slug= root= work= [key=value ...]"
  [ -f "$dir/state.md" ] && fail state "state.md exists in $dir"
  mkdir -p "$dir"
  cat >"$dir/state.md" <<EOF
# lane: $slug

task: -
source: -
variant: pending
tracker: pending
slug: $slug
root: $root
work: $work
src: -
tree: -
upstream: origin/master
base: -
branch: -
ns: $slug
release: $RELEASE_PREFIX$slug
scope: pending
keycloak: pending
crd-lock: -
qa-run: 0
pre-rework-head: -
pre-fix-head: -
approval: pending
installed: none
agents: -
issue: -
pr: -

## Steps

| step | status | values |
|---|---|---|
| 1 intake | todo | - |
| 2 analysis | todo | - |
| 3 reproduction and design | todo | - |
| 4 design review and approval | todo | - |
| 5 development | todo | - |
| 6 qa verification | todo | - |
| 7 code review | todo | - |
| 8 release | todo | - |
EOF
  WORK=$dir
  for kv in "$@"; do state_set "${kv%%=*}" "${kv#*=}"; done
  say "state: $dir/state.md"
}

state_step() {
  local n=$1 status=$2 values=${3:-} tmp=$WORK/state.md.tmp found=0 line body name tail old
  case "$status" in todo|doing|done|stopped|parked) ;; *) usage "status: todo|doing|done|stopped|parked";; esac
  : >"$tmp"
  while IFS= read -r line || [ -n "$line" ]; do
    if [ "${line#| $n }" != "$line" ]; then
      body=${line#| }; body=${body% |}
      name=${body%% | *}; tail=${body#* | }; old=${tail#* | }
      [ -n "$values" ] || values=$old
      line="| $name | $status | $values |"
      found=1
    fi
    printf '%s\n' "$line" >>"$tmp"
  done <"$WORK/state.md"
  [ "$found" = 1 ] || { rm -f "$tmp"; fail state "no step $n"; }
  mv "$tmp" "$WORK/state.md"
  say "step $n: $status"
}

state_totals() {
  local f=$WORK/agent-timings.md
  [ -f "$f" ] || fail state "no agent-timings.md in $WORK"
  awk -F'|' '
    NR > 2 { for (i = 2; i <= 8; i++) gsub(/^ +| +$/, "", $i) }
    NR > 2 && $6 ~ /^[0-9]+$/ {
      if ($4 ~ /^script:/) { sn++; sd += $6; next }
      an++; ad += $6
      if ($7 ~ /^[0-9]+$/) at += $7
      if ($8 ~ /^[0-9]+$/) ak += $8
      printf "  %-18s %-7s %6.1f min %5s tools %8s tokens\n", $4, $5, $6 / 60000, $7, $8
    }
    END {
      printf "agents: %d rows, %.1f min, %d tool uses, %d tokens\n", an, ad / 60000, at, ak
      printf "scripts: %d rows, %.1f min\n", sn, sd / 60000
    }' "$f"
}

state_scope() {
  local scope=code p c
  for p in "$@"; do
    for c in "${CRD_PATHS[@]}"; do
      case "$p" in "$c"*|"./$c"*) scope=crd;; esac
    done
  done
  state_set scope "$scope"
  if [ "$scope" = crd ]; then
    crd_register "$SLUG"
    [ "$DRY" = 1 ] || state_set crd-lock registered
  else
    crd_release "$SLUG"
    [ "$DRY" = 1 ] || state_set crd-lock -
  fi
  say "scope: $scope"
  say "crd items: $(crd_items)"
}

cmd_state() {
  local op=${1:-}
  [ $# -eq 0 ] || shift
  case "$op" in
    init) [ -n "${1:-}" ] || usage "state init <work> ..."; local d=$1; shift; state_init "$d" "$@";;
    get) work_dir "${1:-}"; state_get "${2:?key}";;
    set) work_dir "${1:-}"; shift; [ $# -gt 0 ] || usage "state set <work> key=value ..."
         local kv; for kv in "$@"; do state_set "${kv%%=*}" "${kv#*=}"; done; say "ok";;
    step) work_dir "${1:-}"; state_step "${2:?n}" "${3:?status}" "${4:-}";;
    agent) work_dir "${1:-}"; local cur; cur=$(state_get agents); [ "$cur" != - ] || cur=""
           state_set agents "${cur:+$cur, }${2:?role=id}"; say "ok";;
    ledger) work_dir "${1:-}"; shift; [ $# -eq 6 ] || usage "state ledger <work> <step> <role> <model> <duration_ms> <tool_uses> <tokens>"
            case "$4" in ''|0|*[!0-9]*) fail state "ledger: duration_ms must be a positive integer; no placeholder rows";; esac
            ledger_row "$@"; say "ok";;
    totals) work_dir "${1:-}"; state_totals;;
    scope) work_dir "${1:-}"; shift; [ $# -gt 0 ] || usage "state scope <work> <path> ..."; state_scope "$@";;
    show) work_dir "${1:-}"; cat "$WORK/state.md";;
    -h|--help|help|"") state_help;;
    *) usage "unknown state op: $op";;
  esac
}
