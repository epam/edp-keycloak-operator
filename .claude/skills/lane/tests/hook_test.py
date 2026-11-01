#!/usr/bin/env python3
"""Feed sample Bash commands to the lane hook and compare the decision with the expectation.

Usage: python3 -I hook_test.py <root>. Exit 1 on any mismatch. The expectations double as the hook's contract.
A copy of the hook without env.sh next to it checks the fail-closed path.
"""
import json
import shutil
import subprocess
import sys
import tempfile
from pathlib import Path

ROOT = Path(sys.argv[1]).resolve()
HOOK = ROOT / ".claude/hooks/lane-allow.py"
R = str(ROOT)
TREE = f"{R}/.work/abc/tree"
SRC = f"{R}/.work/abc/master-src"
SCRIPT = f"{R}/.claude/skills/lane/scripts/lane.sh"

CASES = [
    ("allow", f"bash {SCRIPT} status"),
    ("allow", f"bash {SCRIPT} --dry-run teardown {R}/.work/abc"),
    ("none", f"bash {SCRIPT} status; rm -rf /"),
    ("allow", f"bash {SCRIPT} teardown {R}/.work/abc --all"),
    ("none", f"bash {SCRIPT} teardown {R}/.work/abc --all --discard"),
    ("none", f"bash {SCRIPT} teardown {R}/.work/abc --all\t--discard"),
    ("none", f"bash {SCRIPT} teardown {R}/.work/abc --all \"--discard\""),
    ("none", f"bash {SCRIPT} teardown {R}/.work/abc --all '--discard'"),
    ("none", f"bash {SCRIPT} teardown {R}/.work/abc --all --dis''card"),
    ("none", f"bash {SCRIPT} teardown {R}/.work/abc --all \\--discard"),
    ("allow", f"bash {SCRIPT} intake abc fix/abc --task \"mention --discard in text\""),
    ("allow", f"make -C {TREE} LOCALBIN={R}/bin lint-fix"),
    ("allow", f"make -C {SRC} LOCALBIN={R}/bin golangci-lint setup-envtest"),
    ("allow", f"make -C {R} LOCALBIN={R}/bin controller-gen"),
    ("none", f"make -C {TREE} LOCALBIN={R}/bin e2e"),
    ("none", f"make -C {TREE} test"),
    ("allow", f"KUBEBUILDER_ASSETS=/x TEST_KEYCLOAK_URL=http://localhost:1 go -C {TREE} test ./internal/... -run TestX"),
    ("allow", f"git -C {SRC} log --oneline -5"),
    ("allow", f"git -C {R} worktree list"),
    ("allow", f"git -C {TREE} branch --show-current"),
    ("deny", f"git -C {TREE} push -u origin fix/abc"),
    ("none", f"git -C {TREE} commit -s -F msg"),
    ("deny", "gh pr create -R epam/x --head fix/abc"),
    ("deny", "kubectl --context kind-krci delete ns abc"),
    ("deny", "kubectl --context kind-krci delete namespace abc"),
    ("deny", "kubectl --context kind-krci -n keycloak delete deploy keycloak"),
    ("deny", "kubectl --context kind-krci delete deploy keycloak -n keycloak"),
    ("deny", "kubectl --context kind-krci delete deploy keycloak --namespace=cert-manager"),
    ("none", "kubectl --context kind-krci -n abc delete keycloakrealms --all"),
    ("deny", "kubectl --context kind-krci delete crd keycloaks.v1.edp.epam.com"),
    ("deny", "helm uninstall x -n keycloak"),
    ("none", "helm uninstall keycloak-operator-abc -n abc"),
    ("none", "kubectl --context kind-krci -n keycloak get secret keycloak-secret -o yaml"),
    ("allow", "date +%Y%m%d%H%M%S"),
]
NO_ENV_CASES = [
    ("deny", "kubectl --context kind-krci -n abc delete keycloakrealms --all"),
    ("deny", "helm uninstall x -n abc"),
    ("none", "kubectl --context kind-krci get ns"),
]


def decide(hook, command):
    payload = json.dumps({"tool_name": "Bash", "tool_input": {"command": command}})
    out = subprocess.run([sys.executable, "-I", str(hook)], input=payload, capture_output=True, text=True).stdout.strip()
    return json.loads(out)["hookSpecificOutput"]["permissionDecision"] if out else "none"


def run(hook, cases, label):
    bad = 0
    for want, command in cases:
        got = decide(hook, command)
        mark = "ok  " if got == want else "FAIL"
        bad += got != want
        print(f"{mark} [{label}] want {want:5} got {got:5}  {command}")
    return bad


failures = run(HOOK, CASES, "env")
with tempfile.TemporaryDirectory() as tmp:
    copy = Path(tmp) / ".claude/hooks/lane-allow.py"
    copy.parent.mkdir(parents=True)
    shutil.copy(HOOK, copy)
    failures += run(copy, NO_ENV_CASES, "no-env")
for name in (".claude/settings.json", ".claude/settings.local.json"):
    if not (ROOT / name).exists():
        print(f"ok   {name} absent")
        continue
    json.load(open(ROOT / name))
    print(f"ok   {name} parses")
print("all hook checks passed" if failures == 0 else f"{failures} hook check(s) failed")
sys.exit(1 if failures else 0)
