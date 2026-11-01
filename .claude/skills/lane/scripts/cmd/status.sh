# status: every item under .work, the live scope:crd items, the live item namespaces.

cmd_status() {
  case "${1:-}" in -h|--help) say "lane.sh status [--source KEY]   --source prints the work dir whose source equals KEY (exit 1: none)"; return 0;; esac
  local wd=$ROOT_DIR/$WORK_DIR f
  if [ "${1:-}" = --source ]; then
    local key=${2:?KEY}
    for f in "$wd"/*/state.md; do
      [ -f "$f" ] || continue
      [ "$(sed -n 's/^source: //p' "$f")" = "$key" ] && { say "$(dirname "$f")"; return 0; }
    done
    return 1
  fi
  say "items ($wd):"
  local any=0 slug srckey scope branch line
  for f in "$wd"/*/state.md; do
    [ -f "$f" ] || continue
    any=1
    slug=$(sed -n 's/^slug: //p' "$f" | head -1); srckey=$(sed -n 's/^source: //p' "$f" | head -1)
    scope=$(sed -n 's/^scope: //p' "$f" | head -1); branch=$(sed -n 's/^branch: //p' "$f" | head -1)
    line=$(grep -E '^\| [0-9]+ .* \| (doing|stopped|parked) \| ' "$f" | head -1 | cut -d'|' -f2,3 | tr -s ' ' || true)
    [ -n "$line" ] || line=" $(grep -c '| done |' "$f" || true) steps done"
    say "  ${slug:-$(basename "$(dirname "$f")")}  source=${srckey:--}  scope=${scope:--}  branch=${branch:--} $line"
  done
  [ $any = 1 ] || say "  none"
  crd_show
  say "live namespaces ($LABEL on $CONTEXT):"
  kube get ns -l "$LABEL" --no-headers -o custom-columns=NAME:.metadata.name,ITEM:.metadata.labels.lane/item 2>/dev/null | sed 's/^/  /' || say "  (cluster unreachable)"
}
