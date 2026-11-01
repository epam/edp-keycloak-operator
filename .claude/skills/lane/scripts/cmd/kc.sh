# kc: kcadm.sh inside the Keycloak pod. Every realm argument must be <slug>-*; `master` needs --master.

kc_help() {
  cat <<EOF
lane.sh kc <work> [--master] realms            realms of this item
lane.sh kc <work> [--master] <kcadm args ...>  for example: get admin-events -r <slug>-main -q resourceTypes=USER
  Realm arguments (-r X, --target-realm X, realms/X) must start with <slug>-. --master allows realm master.
  Credentials come from secret $KC_SECRET in the Keycloak namespace. Output is kcadm's stdout.
EOF
}

cmd_kc() {
  case "${1:-}" in -h|--help) kc_help; return 0;; esac
  work_dir "${1:-}"; shift
  local master=0
  [ "${1:-}" != --master ] || { master=1; shift; }
  [ $# -gt 0 ] || usage "lane.sh kc <work> [--master] realms | <kcadm args>"
  local kcns=$KC_NS
  [ "$KEYCLOAK" != dedicated ] || kcns=$NS

  if [ "$1" = realms ]; then
    [ "$DRY" = 1 ] && { say "+ kcadm get realms --fields realm (filtered to $SLUG-)"; return 0; }
    kc_item_realms "$kcns" | grep . || say "no $SLUG- realm"
    return 0
  fi

  check_realm() {
    case "$1" in
      "$SLUG-"*) ;;
      master) [ $master = 1 ] || stop realm "realm master needs --master";;
      *) stop realm "realm $1 is not $SLUG-*";;
    esac
  }
  local prev="" a r
  for a in "$@"; do
    case "$a" in realms/*) r=${a#realms/}; check_realm "${r%%/*}";; esac
    case "$prev" in -r|--target-realm) check_realm "$a";; esac
    prev=$a
  done
  if [ "$DRY" = 1 ]; then say "+ kubectl --context $CONTEXT -n $kcns exec deploy/$KC_DEPLOY -c $KC_CONTAINER -- kcadm.sh $*"; return 0; fi
  kc_exec "$kcns" "$@"
}
