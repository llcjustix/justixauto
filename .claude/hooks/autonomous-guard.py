#!/usr/bin/env python3
"""
Additional permission checks for Claude Code.

This hook can deny a known unsafe operation or ask about a destructive one.
It NEVER approves a tool call: ordinary and unrecognized operations continue
through Claude's native permission system. It is not a sandbox or a complete
shell parser. Remote branch protection is still required.

No development tests, builds, lint or review agents are launched by this hook.
"""

import json
import os
import re
import shlex
import sys

# --- Configurable roots: Edit/Write is allowed only inside these -------------
ALLOWED_WRITE_ROOTS = [
    os.path.expanduser("~/startups/justixauto"),  # this project only (NOT the enclosing ~/startups repo)
    os.path.expanduser("~/.claude/projects"), # auto-memory + session artifacts (NOT ~/.claude/settings.json)
    os.path.expanduser("~/.claude/plans"),    # plan-mode plan files
    "/private/tmp/claude-501",                # session scratchpads
    "/tmp",
    "/var/folders",                           # macOS temp
]

# Paths that must never be written, even inside an allowed root.
DENY_WRITE_SUBSTRINGS = [
    "/.ssh/", "/.aws/credentials", "/.gnupg/", "/etc/", "id_rsa", "id_ed25519",
]

# --- Secret-bearing paths: denied for READS too (never "ask", so no prompts) --
SECRET_DIR_PARTS = frozenset({".ssh", ".aws", ".gnupg"})
SECRET_MARKERS = (
    "/.ssh/", "/.aws/", "/.gnupg/", "/etc/",
    "id_rsa", "id_ed25519", "id_ecdsa", "id_dsa",
    ".npmrc", ".pypirc", ".netrc",
    "private-key", "settings.local.json",
    "application_default_credentials.json", "service-account.json",
    "aws_secret_access_key", "kubeconfig",
)
SECRET_SUFFIXES = (".pem", ".key", ".p12", ".pfx", ".jks")
ENV_EXAMPLE_SUFFIXES = (".example", ".sample", ".template")
WILDCARD_RE = re.compile(r"[*?\[\]{}]")

# Whole-command deny checks (span pipes / structure) — evaluated before splitting.
WHOLE_DENY = [
    (r"(curl|wget)\b[^\n]*\|\s*(sudo\s+)?(bash|sh|zsh)\b", "pipe-to-shell"),
    (r":\s*\(\s*\)\s*\{[^}]*\|\s*:", "fork bomb"),
    (r">\s*[^\s|;&]*(/\.ssh/|/etc/|id_rsa|id_ed25519)", "redirect into sensitive path"),
]

SAFE_RM_MARKERS = ("scratch", "/private/tmp/", "/tmp/", "node_modules", "/dist", "/build", "coverage")

def emit(decision, reason):
    if decision is None:
        sys.exit(0)  # No decision: preserve native permission checks.
    print(json.dumps({"hookSpecificOutput": {
        "hookEventName": "PreToolUse",
        "permissionDecision": decision,
        "permissionDecisionReason": reason,
    }}))
    sys.exit(0)


def path_in_roots(path):
    # realpath on both sides: without it a symlink inside an allowed root escapes
    # to anywhere, and a bare prefix match lets `~/flexobo-evil` pass as `~/flexobo`.
    resolved = os.path.realpath(os.path.expanduser(path))
    for root in ALLOWED_WRITE_ROOTS:
        resolved_root = os.path.realpath(root)
        if resolved == resolved_root or resolved.startswith(resolved_root + os.sep):
            return True
    return False


def is_secret_path(value):
    """True when a path expression points at credential material.

    Deliberately textual: no filesystem resolution, no workspace-root check —
    those produced the permission prompts this guard exists to avoid. Only
    genuinely secret-bearing shapes are matched.
    """
    if not value:
        return False
    normalized = os.path.expanduser(value).lower().replace("\\", "/")
    if any(normalized.endswith(suffix) for suffix in SECRET_SUFFIXES):
        return True
    if any(marker in normalized for marker in SECRET_MARKERS):
        return True
    if set(normalized.split("/")) & SECRET_DIR_PARTS:
        return True
    filename = normalized.rsplit("/", 1)[-1]
    if not filename.startswith(".env"):
        return False
    # .env.example / .env.sample / .env.template are committed and safe; a
    # wildcard basename such as `.env*` can still expand onto the real file.
    if WILDCARD_RE.search(filename):
        return True
    return not filename.endswith(ENV_EXAMPLE_SUFFIXES)


def segment_references_secret(seg):
    for token in seg.split():
        candidate = token.split("=", 1)[1] if token.startswith("-") and "=" in token else token
        if candidate.startswith("-"):
            continue
        if is_secret_path(candidate.strip("'\"")):
            return True
    return False


def segment_program(seg):
    """Return the leading program name of a shell segment, or '' if none."""
    s = seg.strip()
    s = re.sub(r"^[(){}\s]+", "", s)                 # strip leading ( { whitespace
    s = re.sub(r"^(?:sudo|command|nohup|time|exec)\s+", "", s)
    s = re.sub(r"^(?:\w+=(?:\"[^\"]*\"|'[^']*'|\S*)\s+)+", "", s)  # strip leading VAR=val assignments
    m = re.match(r"([\w./-]+)", s)
    if not m:
        return ""
    return m.group(1).rsplit("/", 1)[-1]             # basename


def push_denied(seg):
    """Accept only explicit origin pushes into agent-owned branch namespaces.

    This examines literal Git argv, not executable scripts or Git credentials.
    A supported push still goes through native permissions.
    """
    try:
        words = shlex.split(seg)
    except ValueError:
        return "unparseable Git command"
    if "push" not in words:
        return None
    args = words[words.index("push") + 1:]
    allowed_flags = {"-u", "--set-upstream", "-n", "--dry-run", "-v", "--verbose", "--porcelain"}
    operands = []
    for arg in args:
        if arg.startswith("-"):
            if arg not in allowed_flags:
                return "unsupported push option (force, deletion and bulk pushes are forbidden)"
        else:
            operands.append(arg)
    if len(operands) != 2 or operands[0] != "origin":
        return "push must name origin and one explicit task/feature/fix/infra destination"
    ref = operands[1]
    source, _, destination = ref.rpartition(":")
    if not source and ":" in ref:
        return "branch deletion is forbidden"
    if ref.startswith("+") or any(c in ref for c in "*?[$\x60"):
        return "forced, wildcard or computed push ref"
    destination = destination.removeprefix("refs/heads/")
    if destination in {"dev", "main", "master"}:
        return "protected branch integration is human-controlled"
    if not destination.startswith(("task/", "feature/", "fix/", "infra/")):
        return "push destination must be an agent-owned task/feature/fix/infra branch"
    return None


def segment_denied(prog, seg):
    """Program-scoped dangerous checks. Returns a reason string or None."""
    low = seg.lower()
    # Additional checks only; no positive permission grants.
    if segment_references_secret(seg):
        return "credential-bearing path"
    if re.search(r"(^|\s)sudo\s", seg):
        return "sudo (privilege escalation)"
    if prog == "dd" and re.search(r"\bof=", seg):
        return "dd raw write"
    if prog.startswith("mkfs") or prog in ("fdisk", "parted", "diskutil"):
        return "disk format/partition"
    if re.search(r">\s*/dev/(sd|disk|rdisk|nvme|hd|vd|mem|kmem)", seg):
        return "write to raw block device"
    if prog in ("nc", "ncat", "netcat"):
        return "netcat (possible exfil/backdoor)"
    if prog in ("scp", "rsync") and re.search(r"\s[\w.-]+@[\w.-]+:|\s[\w.-]+:/", seg):
        return "scp/rsync to remote (possible exfiltration)"
    if prog == "git":
        reason = push_denied(seg)
        if reason:
            return reason
        if re.search(r"\breset\s+--hard\b[^\n]*origin/(dev|main|master)\b", seg):
            return "git hard-reset to protected branch"
        if re.search(r"\bclean\b[^\n]*-\w*[fdx]", seg):
            return "git clean (destroys untracked work)"
        if re.search(r"--(ext-diff|textconv|exec-path|config-env|upload-pack|receive-pack)(\s|=|$)", seg):
            return "git option that executes outside the repository"
        # AGENTS.md: no separate development worktrees; never touch the parent repo.
        if re.search(r"\bworktree\s+add\b", seg):
            return "git worktree add (development worktrees are forbidden)"
        if re.search(r"(^|\s)-C\s+['\"]?(~|/Users/[^/\s]+)/startups/?['\"]?(\s|$)", seg):
            return "git against the enclosing ~/startups repository"
    if prog == "agent-git" or "tools/agent-git" in seg:
        if re.search(r"\b(create-worktree|commit)\b", seg):
            return "legacy agent-git worktree operation (forbidden by AGENTS.md)"
    if prog == "gh":
        if re.search(r"\bpr\s+merge\b", seg):
            return "gh pr merge"
        if re.search(r"\bapi\b[^\n]*(?:-X|--method)\s*(POST|PUT|PATCH|DELETE)\b", seg, re.IGNORECASE):
            return "mutating GitHub API request"
        if re.search(r"\bapi\b[^\n]*(?:-f|--field|--raw-field|--input)(\s|=)", seg):
            return "mutating GitHub API request"
    if "values-prod" in seg:
        return "production Helm values (production is human-only)"
    if prog == "helm" and re.search(r"\b(uninstall|delete|rollback)\b", seg):
        return "helm uninstall/rollback"
    if prog == "kubectl":
        if re.search(r"\b(get|describe)\s+(secret|secrets)\b", seg):
            return "kubectl secret read"
        if re.search(r"\bconfig\s+view\b[^\n]*--raw\b", seg):
            return "kubectl raw kubeconfig read"
        if re.search(r"\bdelete\b", seg):
            return "kubectl delete"
        if re.search(r"(-n|--namespace)[\s=]+\S*prod\b|--context[\s=]+(?!kind-)\S+", seg):
            return "kubectl outside the local kind cluster"
        if re.search(r"\b(drain|cordon)\b", seg):
            return "kubectl drain/cordon"
    if prog in ("aws", "eksctl"):
        if "delete-cluster" in low:
            return "cluster deletion"
        if re.search(r"\bdelete[- ]", seg):
            return "aws delete-*"
    if prog == "rm" and re.search(r"\s-\w*r", seg, re.IGNORECASE):
        if not any(mark in low for mark in SAFE_RM_MARKERS):
            return "recursive rm outside safe paths"
    if prog in ("curl", "wget"):
        if re.search(r"(\s-d\b|--data\b|--data-\w+|\s-F\b|--form\b|\s-T\b|--upload-file\b|-X\s*(POST|PUT|DELETE)|--post-data|--post-file|--method\s*(POST|PUT|DELETE))", seg):
            return "curl/wget upload/POST (possible exfiltration)"
    return None


def segment_ask_reason(prog, seg):
    """Destructive operations requiring confirmation in addition to native permissions."""
    if prog == "kubectl" and re.search(
        r"\b(apply|patch|scale|edit|replace|annotate|label|set|create)\b", seg
    ):
        return "kubectl cluster mutation"
    if prog == "git" and re.search(
        r"\b(reset\s+--hard|checkout\s+--\s|restore\b|branch\s+-D|stash\s+(drop|clear))", seg
    ):
        return "discards uncommitted or unmerged work"
    if prog == "make" and re.search(r"\b(db-reset|k8s-down)\b", seg):
        return "wipes the local database/cluster"
    if prog == "docker" and re.search(r"\bcompose\b[^\n]*\bdown\b[^\n]*(-v\b|--volumes)|\bvolume\s+(rm|prune)\b", seg):
        return "deletes local Docker volumes"
    return None


def split_segments(cmd):
    return [s for s in re.split(r"\|\||&&|\||;|\n|\$\(|`", cmd) if s.strip()]


def check_bash(cmd):
    for pat, why in WHOLE_DENY:
        if re.search(pat, cmd, re.IGNORECASE):
            emit("deny", f"blocked: {why}")
    # A heredoc body is stdin DATA, not commands — analyze only the command line
    # before `<<` so body text can't be mis-split into fake segments.
    analysis = cmd.split("<<", 1)[0] if "<<" in cmd else cmd
    segs = split_segments(analysis)
    if not segs:
        emit("ask", "empty/unparseable command")
    for seg in segs:
        prog = segment_program(seg)
        why = segment_denied(prog, seg)
        if why:
            emit("deny", f"blocked dangerous command: {why} (in `{seg.strip()[:60]}`)")
        confirm = segment_ask_reason(prog, seg)
        if confirm:
            emit("ask", f"{confirm} needs confirmation (in `{seg.strip()[:60]}`)")
    emit(None, "continue through native permissions")


def main():
    data = json.loads(sys.stdin.read())
    tool = data.get("tool_name", "")
    ti = data.get("tool_input", {}) or {}

    if tool.startswith("mcp__figma__") or tool.startswith("mcp__claude-in-chrome__"):
        emit(None, f"trusted MCP tool {tool}")

    if tool in ("Edit", "Write", "MultiEdit", "NotebookEdit"):
        p = ti.get("file_path") or ti.get("notebook_path") or ""
        if not p:
            emit("ask", "no file path in tool input")
        if any(s in p for s in DENY_WRITE_SUBSTRINGS):
            emit("deny", f"write to sensitive path blocked: {p}")
        if path_in_roots(p):
            emit(None, f"write within allowed root: {p}")
        emit("deny", f"write outside allowed roots: {p}")

    if tool in ("Read", "Glob", "Grep"):
        # Reads are unrestricted except for credential material — a workspace
        # containment check here is what made ordinary reads prompt.
        for value in (
            ti.get("file_path"),
            ti.get("path"),
            ti.get("pattern") if tool == "Glob" else None,
            ti.get("glob"),
        ):
            if isinstance(value, str) and is_secret_path(value):
                emit("deny", f"read from a credential-bearing path blocked: {value}")
        emit(None, f"{tool} does not touch credential material")

    if tool == "Bash":
        check_bash(ti.get("command", "") or "")

    emit("ask", f"unhandled tool {tool}")


if __name__ == "__main__":
    try:
        main()
    except Exception as e:  # noqa: BLE001 — fail safe
        print(json.dumps({"hookSpecificOutput": {
            "hookEventName": "PreToolUse",
            "permissionDecision": "ask",
            "permissionDecisionReason": f"guard error, asking to be safe: {e}",
        }}))
        sys.exit(0)
