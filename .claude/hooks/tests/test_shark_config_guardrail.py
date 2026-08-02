#!/usr/bin/env python3
"""
Regression tests for shark-config-guardrail.py.

Run: python3 .claude/hooks/tests/test_shark_config_guardrail.py

Subprocess-driven (pipes JSON into the hook's stdin, same contract Claude
Code uses) since the hook's filename contains a dash and isn't importable.
"""

import json
import subprocess
import sys
import unittest
from pathlib import Path

HOOK_PATH = Path(__file__).resolve().parent.parent / "shark-config-guardrail.py"


def run_hook(event):
    proc = subprocess.run(
        [sys.executable, str(HOOK_PATH)],
        input=json.dumps(event),
        capture_output=True,
        text=True,
        timeout=5,
    )
    denied = False
    if proc.stdout.strip():
        payload = json.loads(proc.stdout)
        denied = (
            payload.get("hookSpecificOutput", {}).get("permissionDecision")
            == "deny"
        )
    return proc.returncode, denied


def bash_event(command):
    return {"tool_name": "Bash", "tool_input": {"command": command}}


class ShouldDeny(unittest.TestCase):
    def assert_denied(self, command):
        code, denied = run_hook(bash_event(command))
        self.assertEqual(code, 0, f"hook should always exit 0 for: {command!r}")
        self.assertTrue(denied, f"expected deny for: {command!r}")

    def test_plain_invocation(self):
        self.assert_denied("sh" + "ark admin in" + "it --force")

    def test_cloud_init(self):
        self.assert_denied("sh" + "ark cloud in" + "it")

    def test_global_flag_before_subcommand(self):
        # Regression: interleaved global flags used to bypass the guard.
        self.assert_denied("sh" + "ark --db=/tmp/x.db admin in" + "it --force")

    def test_global_flag_between_subcommand_words(self):
        self.assert_denied("sh" + "ark admin --force in" + "it")

    def test_relative_binary_path(self):
        self.assert_denied("./bin/sh" + "ark admin in" + "it")

    def test_chained_command(self):
        self.assert_denied("echo hi && sh" + "ark admin in" + "it --force")


class ShouldAllow(unittest.TestCase):
    def assert_allowed(self, command):
        code, denied = run_hook(bash_event(command))
        self.assertEqual(code, 0)
        self.assertFalse(denied, f"expected allow for: {command!r}")

    def test_unrelated_shark_command(self):
        self.assert_allowed("sh" + "ark task list")

    def test_admin_without_init(self):
        self.assert_allowed("sh" + "ark admin get")

    def test_no_word_boundary(self):
        self.assert_allowed("sh" + "arkadmin in" + "it")

    def test_non_bash_tool_ignored(self):
        code, denied = run_hook(
            {"tool_name": "Read", "tool_input": {"file_path": "x"}}
        )
        self.assertEqual(code, 0)
        self.assertFalse(denied)


class MalformedInputFailsOpen(unittest.TestCase):
    def assert_fails_open(self, raw_stdin):
        proc = subprocess.run(
            [sys.executable, str(HOOK_PATH)],
            input=raw_stdin,
            capture_output=True,
            text=True,
            timeout=5,
        )
        self.assertEqual(proc.returncode, 0)
        self.assertEqual(proc.stdout.strip(), "")

    def test_invalid_json(self):
        self.assert_fails_open("not json")

    def test_json_null(self):
        self.assert_fails_open("null")

    def test_json_list(self):
        self.assert_fails_open("[1, 2, 3]")

    def test_tool_input_not_dict(self):
        self.assert_fails_open(
            json.dumps({"tool_name": "Bash", "tool_input": "sh" + "ark admin in" + "it"})
        )


if __name__ == "__main__":
    unittest.main()
