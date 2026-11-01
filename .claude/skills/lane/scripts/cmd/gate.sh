# gate: checks that the files a step promises exist before the Lead moves on. Step 3 only.

gate_help() {
  cat <<EOF
lane.sh gate <work> 3
  Step 3 (reproduction and design) is complete when: variant is defect or feature; cause.md and
  evidence/master/INVENTORY.txt exist; a defect has a non-empty evidence/master/SUMMARY.txt; plan.md holds
  "Issue brief", "Before / after", a precedent table and a case table, each at the start of a line (a heading,
  bold label or list item).
  Prints RESULT=OK, or FAIL:gate with the missing pieces (exit 1). It checks presence, not content.
EOF
}

cmd_gate() {
  case "${1:-}" in -h|--help) gate_help; return 0;; esac
  work_dir "${1:-}"; shift
  [ "${1:-}" = 3 ] || usage "lane.sh gate <work> 3"
  local variant miss="" phrase
  variant=$(state_get variant)
  case "$variant" in defect|feature) ;; *) miss="$miss variant(state: $variant)";; esac
  [ -s "$WORK/cause.md" ] || miss="$miss cause.md"
  [ -s "$INV" ] || miss="$miss evidence/master/INVENTORY.txt"
  if [ "$variant" = defect ] && [ ! -s "$WORK/evidence/master/SUMMARY.txt" ]; then
    miss="$miss evidence/master/SUMMARY.txt"
  fi
  if [ -s "$WORK/plan.md" ]; then
    for phrase in "issue brief" "before / after" "precedent" "case table"; do
      grep -qiE -- "^[#*[:space:]-]*$phrase" "$WORK/plan.md" || miss="$miss plan.md:\"$phrase\""
    done
  else
    miss="$miss plan.md"
  fi
  [ -z "$miss" ] || fail gate "missing:$miss"
  say "RESULT=OK"
  say "variant: $variant"
}
