#!/usr/bin/env python3
"""Summarize a subagent transcript (JSONL) for the lane acceptance trace.

usage: python3 -I trace.py <transcript.jsonl> [--root ROOT] [--full]
Prints: duration, turns, token usage, tool counts, skill files read (with repeat counts), other files read,
Bash commands that failed or were denied, and assistant text lines that signal a gap in the skill.
--full lists every Bash command.
"""
import argparse
import collections
import json
import re
import sys
from datetime import datetime
from pathlib import Path

GAP = re.compile(r"(?i)\b(unclear|ambiguous|not (specified|stated|documented|found)|missing|could not|cannot|"
                 r"no such|does not (say|exist|mention)|workaround|instead|assum(e|ing)|skip(ping)?|deviat|unexpected|"
                 r"permission|denied|sandbox|operation not permitted)\b")


def text_of(content):
    if isinstance(content, str):
        return content
    if isinstance(content, list):
        return " ".join(c.get("text", "") for c in content if isinstance(c, dict) and c.get("type") == "text")
    return ""


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("transcript")
    ap.add_argument("--root", default="")
    ap.add_argument("--full", action="store_true")
    a = ap.parse_args()

    ts = []
    usage = collections.Counter()
    tools = collections.Counter()
    reads = collections.Counter()
    bash = []          # (command, is_error, result head)
    pending = {}       # tool_use_id -> (name, input)
    agents = []
    gaps = []
    turns = 0
    for line in open(a.transcript):
        line = line.strip()
        if not line:
            continue
        try:
            m = json.loads(line)
        except json.JSONDecodeError:
            continue
        if m.get("timestamp"):
            ts.append(m["timestamp"])
        msg = m.get("message") or {}
        content = msg.get("content")
        if not isinstance(content, list):
            continue
        if m.get("type") == "assistant":
            turns += 1
            for k, v in (msg.get("usage") or {}).items():
                if isinstance(v, int):
                    usage[k] += v
            for c in content:
                if not isinstance(c, dict):
                    continue
                if c.get("type") == "text":
                    for sent in re.split(r"(?<=[.!?])\s+", c.get("text", "")):
                        if GAP.search(sent) and 20 < len(sent) < 400:
                            gaps.append(sent.strip())
                elif c.get("type") == "tool_use":
                    name = c.get("name", "?")
                    tools[name] += 1
                    inp = c.get("input") or {}
                    pending[c.get("id")] = (name, inp)
                    if name == "Read":
                        reads[inp.get("file_path", "?")] += 1
                    elif name == "Bash":
                        for p in re.findall(r"\S*/\.claude/skills/lane/\S+\.md", inp.get("command", "")):
                            reads[p.rstrip(";)'\"")] += 1
                    elif name == "Agent":
                        agents.append(f"{inp.get('subagent_type', '?')}/{inp.get('model', 'inherit')}: {inp.get('description', '')}")
        else:
            for c in content:
                if isinstance(c, dict) and c.get("type") == "tool_result":
                    name, inp = pending.pop(c.get("tool_use_id"), ("?", {}))
                    if name == "Bash":
                        head = text_of(c.get("content"))[:200].replace("\n", " | ")
                        bash.append((inp.get("command", ""), bool(c.get("is_error")), head))

    def parse(s):
        return datetime.fromisoformat(s.replace("Z", "+00:00"))
    ts.sort()
    dur = (parse(ts[-1]) - parse(ts[0])).total_seconds() if len(ts) > 1 else 0
    print(f"duration {dur / 60:.1f} min, assistant turns {turns}, tools {dict(tools)}")
    print(f"tokens: output {usage['output_tokens']}, cache read {usage['cache_read_input_tokens']}, "
          f"cache write {usage['cache_creation_input_tokens']}")
    root = a.root.rstrip("/")
    skill = [(p, n) for p, n in reads.items() if "/.claude/skills/lane" in p]
    other = [(p, n) for p, n in reads.items() if "/.claude/skills/lane" not in p]
    print("\nskill files read:")
    for p, n in sorted(skill):
        print(f"  {n}x {p.replace(root + '/', '')}")
    print(f"\nother files read: {len(other)} distinct, {sum(n for _, n in other)} reads")
    for p, n in sorted(other, key=lambda x: -x[1])[:12]:
        print(f"  {n}x {p.replace(root + '/', '')}")
    if agents:
        print("\nagents launched:")
        for s in agents:
            print("  " + s)
    failed = [(c, h) for c, e, h in bash if e]
    denied = [(c, h) for c, e, h in bash
              if re.search(r"(?i)operation not permitted|permission denied|denied by|hook (error|denied)|krci-kubelock:", h)]
    print(f"\nbash: {len(bash)} calls, {len(failed)} errors, {len(denied)} denied/permission")
    for c, h in (failed + [d for d in denied if d not in failed])[:15]:
        print(f"  ! {c[:140].replace(chr(10), ' ')}\n      -> {h[:160]}")
    if a.full:
        print("\nall bash commands:")
        for c, e, _ in bash:
            print(("  ! " if e else "    ") + c[:160].replace("\n", " "))
    print(f"\ngap signals in the agent's text ({len(gaps)}):")
    for g in gaps[:25]:
        print("  - " + g[:300])


if __name__ == "__main__":
    sys.exit(main())
