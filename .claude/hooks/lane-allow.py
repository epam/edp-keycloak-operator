#!/usr/bin/env python3
"""PreToolUse hook for the lane skill (Bash tool).

Allow: the lane script (not with --discard, which prompts), make, go test and read-only git inside the lane
worktrees and the checkout.
Deny: namespace deletes, deletes in protected namespaces, CRD deletes, a push from a lane tree, gh pr create.
Values: .claude/skills/lane/scripts/lib/env.sh. Root: the repository that holds this file.
A command that matches nothing falls through to the normal permission rules.
"""
import json
import re
import shlex
import sys
from pathlib import Path

ROOT_PATH = Path(__file__).resolve().parents[2]
ROOT = re.escape(str(ROOT_PATH))
ENV_FILE = ROOT_PATH / ".claude/skills/lane/scripts/lib/env.sh"


def env_values(name, default):
    try:
        text = ENV_FILE.read_text()
    except OSError:
        return default
    match = re.search(rf"^{name}=\((.*)\)$", text, re.M)
    if match:
        return match.group(1).split()
    match = re.search(rf"^{name}=(\S+)$", text, re.M)
    return [match.group(1).strip("'\"")] if match else default


ENV_READABLE = ENV_FILE.is_file()
PROMPT_TARGETS = set(env_values("PROMPT_TARGETS", []))
PROTECTED_NS = env_values("PROTECTED_NS", [])
WORK_DIR = re.escape(env_values("WORK_DIR", [".work"])[0])

SLUG = r"[a-z0-9-]{1,20}"
TREE = rf"{ROOT}/{WORK_DIR}/{SLUG}/tree"
SRC = rf"{ROOT}/{WORK_DIR}/{SLUG}/master-src"
ANY_TREE = rf"(?:{TREE}|{SRC}|{ROOT})"
SCRIPT = rf"{ROOT}/\.claude/skills/lane/scripts/lane\.sh"
# Free arguments: no shell separators, substitution or redirection.
ARGS = r"[^;&|`$<>(){}\n]*"
MAKE_WORD = r"[A-Za-z0-9_-]+(?:=[^\s;&|`$<>(){}]*)?"
MAKE = re.compile(rf"make -C {ANY_TREE} LOCALBIN={ROOT}/bin((?: {MAKE_WORD})+)")

ALLOW = [
    rf"bash {SCRIPT}( {ARGS})?",
    rf"(?:KUBEBUILDER_ASSETS=\S+ )?(?:TEST_KEYCLOAK_URL=\S+ )?go -C {TREE} (?:test|vet|build)( {ARGS})?",
    rf"git -C {ANY_TREE} (?:status|log|diff|show|rev-parse|blame|merge-base|describe)( {ARGS})?",
    rf"git -C {ANY_TREE} branch (?:--list|--show-current)( {ARGS})?",
    rf"git -C {ROOT} worktree list( {ARGS})?",
    rf"git -C {ROOT} remote get-url origin",
    r"date \+%Y%m%d%H%M%S",
]

DENY = [
    (r"\bkubectl\b.*\bdelete\s+(?:ns|namespace|namespaces)\b", "a namespace delete is the user's command"),
    (r"\bkubectl\b.*\bdelete\s+crd\b", "CRDs are shared by every item"),
    (rf"\bgit -C {TREE} push\b", "the user pushes"),
    (r"\bgh pr create\b", "the user opens the PR"),
]
for ns in PROTECTED_NS:
    DENY.append((rf"\bkubectl\b(?=.*\bdelete\b).*(?:-n|--namespace)[= ]{re.escape(ns)}\b", f"namespace {ns} is protected"))
    DENY.append((rf"\bhelm\b(?=.*\buninstall\b).*(?:-n|--namespace)[= ]{re.escape(ns)}\b", f"namespace {ns} is protected"))
if not ENV_READABLE:
    # Fail closed: without the protected list every cluster delete is blocked.
    DENY.append((r"\bkubectl\b.*\bdelete\b", f"{ENV_FILE} is unreadable; deletes are blocked"))
    DENY.append((r"\bhelm\b.*\buninstall\b", f"{ENV_FILE} is unreadable; uninstalls are blocked"))


def make_allowed(command):
    match = MAKE.fullmatch(command)
    if not match:
        return False
    targets = [word for word in match.group(1).split() if "=" not in word]
    return bool(targets) and not set(targets) & PROMPT_TARGETS


def discards(command):
    try:
        return "--discard" in shlex.split(command)
    except ValueError:
        return True


def decide(command):
    for pattern, reason in DENY:
        if re.search(pattern, command):
            return "deny", reason
    if re.match(rf"bash {SCRIPT}(?:\s|$)", command) and discards(command):
        return None, None
    if make_allowed(command) or any(re.fullmatch(rule, command) for rule in ALLOW):
        return "allow", "lane worktree command"
    return None, None


def main():
    payload = json.load(sys.stdin)
    if payload.get("tool_name") != "Bash":
        return
    command = payload.get("tool_input", {}).get("command", "").strip()
    decision, reason = decide(command)
    if decision is None:
        return
    print(json.dumps({
        "hookSpecificOutput": {
            "hookEventName": "PreToolUse",
            "permissionDecision": decision,
            "permissionDecisionReason": reason,
        }
    }))


if __name__ == "__main__":
    main()
