#!/usr/bin/env bash
# lane: the mechanical jobs of the lane skill.
# Usage: bash lane.sh [--dry-run] <command> [args]   (`bash lane.sh help` lists the commands)
set -euo pipefail

HERE=$(cd "$(dirname "$0")" && pwd -P)
ROOT_DIR=$(cd "$HERE/../../../.." && pwd -P)
. "$HERE/lib/env.sh"
. "$HERE/lib/common.sh"

DRY=0
if [ "${1:-}" = "--dry-run" ]; then DRY=1; shift; fi
CMD=${1:-help}
[ $# -eq 0 ] || shift

case "$CMD" in
  intake|status|state|lock|gate|tests|build|install|kc|teardown|rebase)
    . "$HERE/cmd/$CMD.sh"
    "cmd_$CMD" "$@"
    ;;
  upgrade)
    . "$HERE/cmd/install.sh"
    cmd_upgrade "$@"
    ;;
  help|-h|--help)
    cat <<EOF
lane.sh [--dry-run] <command> [args]      root: $ROOT_DIR

  intake <slug> <branch> [--issue N] [--waive-pr] [--adopt] [--base REF] [--source KEY] [--task TEXT]
                                   preflight, claim .work/<slug>, worktrees, tools; prints BASE=
  rebase <work>                    fetch; rebase <tree> onto origin/master and move <src>; prints BASE= MOVED= OVERLAP=
  status [--source KEY]            items, live scope:crd items, live namespaces; --source prints the work dir
  state <op> <work> ...            init | get | set | step | agent | ledger | totals | scope | show
  lock take|release|show [slug]    live scope:crd items (information only; state scope maintains it)
  gate <work> 3                    step 3 is complete: cause, evidence, inventory, plan sections
  tests <work> [pkg ...]           Keycloak fixture, full suite or packages, validate-docs
  build <work> master|fix [--run N]
                                   make build, docker build, kind load, verify; prints TAG=
  install <work> master|fix <tag>  namespace, CRDs by scope, helm install, Keycloak CR, realm <slug>-main
  upgrade <work> <tag>             helm upgrade of the item release to a fix image
  kc <work> [--master] realms | <kcadm args>
                                   kcadm in the Keycloak pod; realm args must be <slug>-*
  teardown <work> [--all [--discard]]
                                   remove the item from the cluster; --all also removes tree and branch
                                   (STOP:unsaved|unpushed; --discard skips both checks)

Exit codes: 0 OK, 1 FAIL, 2 STOP (precondition), 3 usage. Logs: <work>/logs/.
Sandbox: intake and rebase (SSH fetch, gh), tests (envtest cache) and teardown (worktree removal) need it
disabled; the others run inside it.
A command that fails with "operation not permitted" is rerun with the sandbox disabled.
EOF
    ;;
  *)
    usage "unknown command: $CMD (bash lane.sh help)"
    ;;
esac
