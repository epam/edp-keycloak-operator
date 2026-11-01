# install: the item namespace, CRDs by scope, the operator release, the Keycloak CR and realm <slug>-main.
# upgrade: the release to a fix image. Both wait for the operator's ready log line.

install_help() {
  cat <<EOF
lane.sh install <work> master|fix <tag>     chart from <src> (master) or <tree> (fix)
lane.sh upgrade <work> <tag>                helm upgrade --reset-then-reuse-values to the fix image
  Needs state scope (code|crd) and keycloak (shared|dedicated).
  CRDs: install master adds only missing CRDs, whatever the scope. scope crd: install fix and upgrade apply
  all from <tree>, server-side; scope code: they add only missing CRDs.
  keycloak shared: copy $KC_SECRET from $KC_NS, Keycloak CR url http://$KC_SVC.$KC_NS:$KC_PORT.
  keycloak dedicated: the e2e manifest without its Namespace document, applied in <ns>.
  Appends every object to INVENTORY.txt.
EOF
}

# apply_crds <dir> <item:0|1>. The base install adds only missing CRDs whatever the scope: the installed set may be
# newer than <src>, and a forced apply of the base would prune fields other operators on the cluster use.
# A crd item's fix install and upgrade apply the whole set from <tree>.
apply_crds() {
  local dir=$1 item=${2:-0} f name applied=""
  ls "$dir"/*.yaml >/dev/null 2>&1 || stop worktree "no CRD files in $dir"
  if [ "$SCOPE" = crd ] && [ "$item" = 1 ]; then
    runlog kube apply --server-side --force-conflicts -f "$dir" || fail crds "CRD apply from $dir failed; log: $LOG"
    inv_append "crds: applied all from $dir (scope crd)"
    return 0
  fi
  for f in "$dir"/*.yaml; do
    name=$(grep -m1 -E '^  name: ' "$f" | awk '{print $2}')
    kube get crd "$name" >/dev/null 2>&1 && continue
    runlog kube apply -f "$f" || fail crds "CRD apply $f failed; log: $LOG"
    applied="$applied $name"
  done
  inv_append "crds:${applied:- present, none applied}"
}

copy_admin_secret() {
  if [ "$DRY" = 1 ]; then say "+ copy secret $KC_SECRET from $KC_NS to $NS"; return 0; fi
  kube -n "$KC_NS" get secret "$KC_SECRET" -o json \
    | jq 'del(.metadata.namespace, .metadata.uid, .metadata.resourceVersion, .metadata.creationTimestamp, .metadata.managedFields, .metadata.ownerReferences)' \
    | kube -n "$NS" apply -f - >>"$LOG" 2>&1 || fail secret "secret copy failed; log: $LOG"
  inv_append "secret: $KC_SECRET (copied from $KC_NS)"
}

install_dedicated_keycloak() {
  local copy=$WORK/evidence/master/manifests/keycloak.yaml
  if [ "$DRY" = 1 ]; then say "+ apply $SRC/$KC_MANIFEST without its Namespace document in $NS"; return 0; fi
  mkdir -p "$(dirname "$copy")"
  python3 -c 'import sys
docs = open(sys.argv[1]).read().split("\n---\n")
print("\n---\n".join(d for d in docs if "kind: Namespace" not in d))' "$SRC/$KC_MANIFEST" >"$copy"
  kube -n "$NS" apply -f "$copy" >>"$LOG" 2>&1 || fail keycloak "dedicated Keycloak apply failed; log: $LOG"
  kube -n "$NS" rollout status "deploy/$KC_DEPLOY" --timeout=300s >>"$LOG" 2>&1 || fail keycloak "dedicated Keycloak not ready; log: $LOG"
  inv_append "dedicated keycloak: deploy/$KC_DEPLOY svc/$KC_SVC secret/$KC_SECRET in $NS (manifest copy $copy)"
}

wait_operator() {
  [ "$DRY" != 1 ] || { say "+ wait for deploy/$DEPLOY_NAME and log line '$READY_LOG'"; return 0; }
  kube -n "$NS" rollout status "deploy/$DEPLOY_NAME" --timeout=180s >>"$LOG" 2>&1 || fail operator "deploy/$DEPLOY_NAME not ready; log: $LOG"
  local i=0
  # grep -q closes the pipe early; under pipefail the SIGPIPE exit of kubectl would fail the check.
  until { kube -n "$NS" logs "deploy/$DEPLOY_NAME" --tail=-1 2>/dev/null || true; } | grep -q "$READY_LOG"; do
    i=$((i + 1)); [ $i -lt 60 ] || fail operator "no '$READY_LOG' in the operator log after 120 s"
    sleep 2
  done
}

apply_keycloak_crs() {
  local url=$1
  if [ "$DRY" = 1 ]; then say "+ apply Keycloak/keycloak url=$url secret=$KC_SECRET; KeycloakRealm/$SLUG-main"; return 0; fi
  kube -n "$NS" apply -f - >>"$LOG" 2>&1 <<EOF || fail crs "Keycloak CR apply failed; log: $LOG"
apiVersion: $CRD_GROUP/v1
kind: Keycloak
metadata:
  name: keycloak
spec:
  url: $url
  secret: $KC_SECRET
---
apiVersion: $CRD_GROUP/v1
kind: KeycloakRealm
metadata:
  name: $SLUG-main
spec:
  realmName: $SLUG-main
  keycloakRef:
    name: keycloak
    kind: Keycloak
EOF
  inv_append "cr: Keycloak/keycloak url=$url"
  inv_append "cr: KeycloakRealm/$SLUG-main"
  inv_append "realm: $SLUG-main"
  kube -n "$NS" wait keycloak/keycloak --for=jsonpath='{.status.connected}'=true --timeout=120s >>"$LOG" 2>&1 || fail crs "Keycloak/keycloak not connected; log: $LOG"
  kube -n "$NS" wait "keycloakrealm/$SLUG-main" --for=jsonpath='{.status.available}'=true --timeout=180s >>"$LOG" 2>&1 || fail crs "KeycloakRealm/$SLUG-main not available; log: $LOG"
}

cmd_install() {
  case "${1:-}" in -h|--help) install_help; return 0;; esac
  work_dir "${1:-}"; shift
  local which=${1:-} tag=${2:-} chart start=$SECONDS
  [ -n "$which" ] && [ -n "$tag" ] || usage "lane.sh install <work> master|fix <tag>"
  case "$which" in master) chart=$SRC/$CHART;; fix) chart=$TREE/$CHART;; *) usage "lane.sh install <work> master|fix <tag>";; esac
  [ "$SCOPE" = code ] || [ "$SCOPE" = crd ] || stop scope "state scope is $SCOPE; set code or crd first"
  [ "$KEYCLOAK" = shared ] || [ "$KEYCLOAK" = dedicated ] || stop keycloak "state keycloak is $KEYCLOAK; set shared or dedicated first"
  LOG=$WORK/logs/install-$(ts).log; : >"$LOG"

  kube get ns "$NS" >/dev/null 2>&1 || runlog kube create ns "$NS" || fail namespace "create ns $NS failed; log: $LOG"
  runlog kube label ns "$NS" "$LABEL=$SLUG" --overwrite || fail namespace "label ns $NS failed; log: $LOG"
  inv_append "namespace: $NS ($LABEL=$SLUG)"
  local item=0; [ "$which" != fix ] || item=1
  apply_crds "$chart/crds" "$item"
  local kc_url
  if [ "$KEYCLOAK" = dedicated ]; then install_dedicated_keycloak; kc_url="http://$KC_SVC.$NS:$KC_PORT"
  else copy_admin_secret; kc_url="http://$KC_SVC.$KC_NS:$KC_PORT"; fi
  runlog helm install "$RELEASE" "$chart" --kube-context "$CONTEXT" -n "$NS" "${INSTALL_FLAGS[@]}" \
    --set image.repository="$IMAGE_REPO" --set image.tag="$tag" --wait --timeout 5m \
    || fail install "helm install failed; log: $LOG"
  inv_append "release: $RELEASE chart=$chart image=$IMAGE_REPO:$tag"
  wait_operator
  apply_keycloak_crs "$kc_url"
  [ "$DRY" = 1 ] || state_set installed "$NS, $RELEASE, $IMAGE_REPO:$tag"
  ledger_row "$(current_step)" "script:install-$which" - $(( (SECONDS - start) * 1000 )) - -
  say "RESULT=OK"
  say "namespace: $NS"
  say "release: $RELEASE"
  say "keycloak: $kc_url ($KEYCLOAK)"
  say "realm: $SLUG-main"
  say "log: $LOG"
}

cmd_upgrade() {
  case "${1:-}" in -h|--help) install_help; return 0;; esac
  work_dir "${1:-}"; shift
  local tag=${1:-} chart=$TREE/$CHART start=$SECONDS
  [ -n "$tag" ] || usage "lane.sh upgrade <work> <tag>"
  LOG=$WORK/logs/upgrade-$(ts).log; : >"$LOG"
  [ "$SCOPE" != crd ] || apply_crds "$chart/crds" 1
  runlog helm upgrade "$RELEASE" "$chart" --kube-context "$CONTEXT" -n "$NS" --reset-then-reuse-values --set image.tag="$tag" --wait --timeout 5m \
    || fail upgrade "helm upgrade failed; log: $LOG"
  inv_append "upgrade: image=$IMAGE_REPO:$tag chart=$chart"
  wait_operator
  [ "$DRY" = 1 ] || state_set installed "$NS, $RELEASE, $IMAGE_REPO:$tag"
  ledger_row "$(current_step)" "script:upgrade" - $(( (SECONDS - start) * 1000 )) - -
  say "RESULT=OK"
  say "release: $RELEASE image=$IMAGE_REPO:$tag"
  say "log: $LOG"
}
