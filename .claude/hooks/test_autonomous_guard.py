"""Unit tests: simulate tool inputs only; no Git, network or Docker operations."""
import contextlib
import importlib.util
import io
import json
from pathlib import Path
import sys
import unittest
from unittest.mock import patch

spec = importlib.util.spec_from_file_location(
    "guard", Path(__file__).with_name("autonomous-guard.py")
)
guard = importlib.util.module_from_spec(spec)
spec.loader.exec_module(guard)


class PermissionGuardTests(unittest.TestCase):
    def decision(self, tool, **tool_input):
        payload = json.dumps({"tool_name": tool, "tool_input": tool_input})
        output = io.StringIO()
        with patch.object(sys, "stdin", io.StringIO(payload)), contextlib.redirect_stdout(output):
            with self.assertRaises(SystemExit) as result:
                guard.main()
        self.assertEqual(result.exception.code, 0)
        if not output.getvalue():
            return None
        return json.loads(output.getvalue())["hookSpecificOutput"]["permissionDecision"]

    def test_normal_and_unknown_commands_preserve_native_permissions(self):
        for command in [
            "git status --short",
            "python3 script.py",
            "an-unknown-command --flag",
            "bash tools/go.sh test ./tools/devtool/",
            "git push origin task/example",
            "git push -u origin fix/example",
            "git push origin HEAD:refs/heads/feature/example",
            "git push origin abc123:infra/example",
            # User decision 2026-09-27: agents may push to dev.
            "git push origin dev",
            "git push origin HEAD:dev",
            "git -C . push origin HEAD:refs/heads/dev",
        ]:
            with self.subTest(command=command):
                self.assertIsNone(self.decision("Bash", command=command))

    def test_protected_and_ambiguous_pushes_are_denied(self):
        for command in [
            "git push origin 'HEAD:refs/heads/main'",
            "git push origin main",
            "git push origin master",
            "git push origin +HEAD:fix/example",
            "git push --force-with-lease origin fix/example",
            "git push --mirror origin",
            "git push --all origin",
            "git push origin",
            "git push",
            "git push origin :fix/example",
            "git push origin 'refs/heads/*:refs/heads/*'",
            "git push origin fix/one fix/two",
        ]:
            with self.subTest(command=command):
                self.assertEqual(self.decision("Bash", command=command), "deny")

    def test_file_and_mcp_calls_are_never_auto_approved(self):
        self.assertIsNone(self.decision("Read", file_path="CLAUDE.md"))
        self.assertIsNone(self.decision("Write", file_path="/tmp/justix-guard-unit.txt"))
        self.assertIsNone(self.decision("mcp__figma__example"))
        self.assertEqual(self.decision("Read", file_path=".env"), "deny")

    def test_destructive_operation_still_requires_confirmation(self):
        self.assertEqual(self.decision("Bash", command="git reset --hard"), "ask")


if __name__ == "__main__":
    unittest.main()
