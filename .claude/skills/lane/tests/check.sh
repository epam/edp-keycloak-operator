#!/usr/bin/env bash
# Static checks and hook tests for the lane skill. Usage: bash check.sh [<root>]. Exit 1 on any failure.
set -uo pipefail
ROOT=${1:-$(cd "$(dirname "$0")/../../../.." && pwd -P)}
SKILL=$ROOT/.claude/skills/lane
S=$SKILL/scripts
fails=0
ok() { printf 'ok    %s\n' "$*"; }
bad() { printf 'FAIL  %s\n' "$*"; fails=$((fails + 1)); }

# Budgets.
w=$(wc -w <"$SKILL/SKILL.md" | tr -d ' '); [ "$w" -le 1800 ] && ok "SKILL.md $w words (<= 1800)" || bad "SKILL.md $w words (> 1800)"
for f in "$SKILL"/roles/*.md; do w=$(wc -w <"$f" | tr -d ' '); [ "$w" -le 500 ] && ok "$(basename "$f") $w words" || bad "$(basename "$f") $w words (> 500)"; done
total=$(cat "$SKILL"/SKILL.md "$SKILL"/roles/*.md "$SKILL"/references/*.md | wc -w | tr -d ' '); ok "markdown total $total words"

# Machine and person specific strings.
for p in '/Users/' '/home/' 'arm64' 'Apple Silicon' 'darwin-arm64'; do
  if grep -rq --exclude=check.sh -- "$p" "$SKILL"; then bad "'$p' found: $(grep -rl --exclude=check.sh -- "$p" "$SKILL" | sed "s#$SKILL/##" | tr '\n' ' ')"; else ok "no '$p'"; fi
done

# Repo constants only in env.sh; the hook tests spell them out in the commands they check.
for p in 'kind-krci' 'repo/edp/keycloak-image' 'keycloak-secret' 'clusterReconciliationEnabled=' 'krci-control-plane'; do
  hits=$(grep -rl --exclude-dir=tests -- "$p" "$SKILL" | grep -v 'lib/env.sh' || true)
  [ -z "$hits" ] && ok "'$p' only in env.sh" || bad "'$p' outside env.sh: $(echo "$hits" | sed "s#$SKILL/##" | tr '\n' ' ')"
done

# Stated once across the markdown.
for p in 'LOCALBIN=<root>/bin' 'never a bare' 'Never a bare' '--reset-then-reuse-values' 'Starting workers' 'adminEventsEnabled' 'kcadm' 'clusterReconciliationEnabled'; do
  n=$(cat "$SKILL"/SKILL.md "$SKILL"/roles/*.md "$SKILL"/references/*.md | grep -o -F -- "$p" | wc -l | tr -d ' ')
  [ "$n" -le 2 ] && ok "'$p' $n times" || bad "'$p' $n times (> 2)"
done

# Scripts.
for f in "$S"/lane.sh "$S"/lib/*.sh "$S"/cmd/*.sh; do bash -n "$f" && ok "syntax $(basename "$f")" || bad "syntax $(basename "$f")"; done
for c in intake status state lock gate tests build install kc teardown rebase; do
  bash "$S/lane.sh" "$c" --help >/dev/null 2>&1 && ok "--help $c" || bad "--help $c"
done
SC=$(command -v shellcheck || true); [ -n "$SC" ] || [ ! -x "$ROOT/.venv/bin/shellcheck" ] || SC=$ROOT/.venv/bin/shellcheck
if [ -n "$SC" ]; then
  "$SC" -x -s bash -S warning -e SC2034,SC1090,SC1091 "$S"/lane.sh "$S"/lib/*.sh "$S"/cmd/*.sh "$SKILL/tests/check.sh" && ok shellcheck || bad shellcheck
else ok "shellcheck not installed (skipped)"; fi
python3 -I -c "import ast,sys; ast.parse(open(sys.argv[1]).read())" "$ROOT/.claude/hooks/lane-allow.py" && ok "hook parses" || bad "hook parses"
python3 -I -c "import ast,sys; ast.parse(open(sys.argv[1]).read())" "$SKILL/tests/trace.py" && ok "trace.py parses" || bad "trace.py parses"
python3 -I "$SKILL/tests/hook_test.py" "$ROOT" >/dev/null && ok "hook tests" || bad "hook tests: python3 -I $SKILL/tests/hook_test.py $ROOT"

# Every env.sh constant the hook reads is present.
for k in PROMPT_TARGETS PROTECTED_NS WORK_DIR; do grep -q "^$k=" "$S/lib/env.sh" && ok "env.sh has $k" || bad "env.sh lacks $k"; done

[ $fails -eq 0 ] && { echo "all checks passed"; exit 0; } || { echo "$fails check(s) failed"; exit 1; }
