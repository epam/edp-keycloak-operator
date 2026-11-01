# lock: the crd-lock, a list of live `scope: crd` items in .work/.crd-lock. Information only; nothing waits on it.
# `state scope` registers and releases the item; teardown releases it.

cmd_lock() {
  local op=${1:-} slug=${2:-}
  case "$op" in
    take)
      [ -n "$slug" ] || usage "lane.sh lock take <slug>"
      crd_register "$slug"
      crd_show
      ;;
    release)
      [ -n "$slug" ] || usage "lane.sh lock release <slug>"
      crd_release "$slug"
      crd_show
      ;;
    show|"")
      crd_show
      ;;
    -h|--help) say "lane.sh lock take|release <slug> | show   live scope:crd items; information only";;
    *) usage "lane.sh lock take|release <slug> | show";;
  esac
}
