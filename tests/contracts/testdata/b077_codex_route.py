#!/usr/bin/env python3
"""Exercise the published Rider commands. The recording CLI supplies its own config."""
import json
import os
import re
from pathlib import Path
import shutil
import sqlite3
from uuid import uuid4
import subprocess
import sys
import tempfile
import unittest


def record_cli():
    directory = Path(os.environ["CASE"])
    argv = sys.argv[1:]
    def record(method, **fields):
        with (directory / "calls").open("a") as output:
            print(json.dumps(dict(method=method, argv=argv, cwd=os.getcwd(), **fields)), file=output)
    if "app-server" in argv:
        for line in sys.stdin:
            request = json.loads(line)
            if request["method"] == "initialize":
                print(json.dumps(dict(id=request["id"], result={})), flush=True)
            if request["method"] == "config/read":
                config = json.loads((directory / "response").read_text())
                record("read", params=request["params"], response=config)
                print(json.dumps(dict(id=request["id"], result=dict(config=config))), flush=True)
        return
    if argv[0] == "exec":
        prompt = sys.stdin.read()
        with sqlite3.connect(Path.cwd() / "shark-tasks.db") as db:
            claim = db.execute("select harness, harness_version, harness_model from entity_claims where entity_key='B001'").fetchone()
        if os.environ.get("DROP_ARG") == "1":
            capture = Path(os.environ["ARGV_FILE"])
            fields = capture.read_bytes().split(b"\0")
            index = fields.index(b"--model")
            del fields[index:index+2]
            capture.write_bytes(b"\0".join(fields))
        thread, turn = str(uuid4()), str(uuid4())
        record("exec", prompt=prompt, claim=claim, thread=thread)
        if os.environ.get("DENIED") == "1":
            event = dict(type="turn.failed", error={"message":"model access denied"})
            if os.environ.get("KNOWN_COST") == "1":
                event.update(usage={"input_tokens": 11}, cost=0.25)
                print(json.dumps(dict(type="thread.started", thread_id=thread)))
                print(json.dumps(dict(type="turn.started")))
                session = Path(os.environ["CODEX_HOME"]) / "sessions"
                session.mkdir(parents=True, exist_ok=True)
                observed = [dict(type="session_meta", payload=dict(id=thread, model_provider="openai")),
                            dict(type="turn_context", payload=dict(turn_id=turn, model="denied-observation"))]
                (session / ("rollout-fixture-" + thread + ".jsonl")).write_text(
                    "".join(json.dumps(item) + "\n" for item in observed))
            print(json.dumps(event))
            sys.exit(1)
        config = json.loads((directory / "response").read_text())
        events = [dict(type="thread.started", thread_id=thread), dict(type="turn.started"),
                  dict(type="turn.completed", usage={"input_tokens":7})]
        for event in events:
            print(json.dumps(event))
        records = [("session_meta", dict(id=thread, model_provider="openai")),
                   ("turn_context", dict(turn_id=turn, model=os.environ.get("OBSERVED") or config["model"],
                                         effort=config.get("model_reasoning_effort"))),
                   ("event_msg", dict(type="task_complete", turn_id=turn))]
        rollout = Path(os.environ["CODEX_HOME"]) / "sessions" / ("rollout-fixture-" + thread + ".jsonl")
        rollout.parent.mkdir(parents=True, exist_ok=True)
        rollout.write_text("".join(json.dumps(dict(type=t, payload=v)) + "\n" for t,v in records))
        Path(argv[argv.index("--output-last-message")+1]).write_text("RECOMMENDED OUTCOME: pass\n")
        return
    raise ValueError(argv)


if __name__ == "__main__" and Path(sys.argv[0]).name == "selected-codex":
    record_cli()
    sys.exit(0)

ROOT = Path(__file__).resolve().parents[3]
RIDER = ROOT / "skills/shark-rider"
MAPPING = "B077-default-bug-anthropic-codex-v1"


def commands():
    run = (RIDER / "verbs/run.md").read_text()
    preclaim = run.split("### B077 default bug-stage", 1)[1].split("Claim the concrete entity", 1)[0]
    preflight = preclaim.split("```bash\n", 1)[1].split("\n```", 1)[0] if "```bash\n" in preclaim else ""
    launch = run.split("### Long-running Codex CLI adapter", 1)[1].split("```bash\n", 1)[1].split("\n```", 1)[0]
    def block(section):
        return re.search(r"(?ms)^ *```bash\n(.*?)^ *```", section)[1]
    claim = block(run.split("Claim the concrete entity returned by Shark:", 1)[1])
    refresh = run.split("For every initial or replacement bug, refresh", 1)[1]
    wait = run.split("At the Codex completion boundary", 1)[1]
    apply = block(run.split("5. **Advance by the resolved outcome**", 1)[1])
    return "\n".join([preflight, claim, block(refresh), launch, block(wait),
                      'OUTCOME=$(sed -n "s/^RECOMMENDED OUTCOME: //p" "$RESULT_FILE")',
                      apply.replace("{response.status}", "research"),
                      'shark release "$ENTITY_KEY" --session "$SID" --outcome "$OUTCOME"'])


class RiderCLI(unittest.TestCase):
    def setUp(self):
        temp = tempfile.TemporaryDirectory()
        self.addCleanup(temp.cleanup)
        self.directory = Path(temp.name)
        self.work = self.directory / "worker directory"
        self.work.mkdir()
        decoy = self.directory / "codex"
        decoy.write_text('#!/bin/sh\ntouch "$CASE/decoy-used"\nexit 90\n')
        decoy.chmod(0o700)
        self.cli = self.directory / "selected-codex"
        shutil.copy(__file__, self.cli)
        self.cli.chmod(0o700)
        def run(*args):
            return subprocess.run(args, cwd=self.work, check=True, capture_output=True)
        run("git", "init", "-q")
        run("git", "-c", "user.name=Fixture", "-c", "user.email=test@example.invalid",
            "commit", "--allow-empty", "-qm", "baseline")
        self.shark = lambda *args: run(os.environ["B077_SHARK"], *args)
        self.shark("admin", "init", "--non-interactive")
        self.shark("admin", "install-shark-data")
        self.shark("bug", "create", "route fixture")
        self.shark("status", "set", "B001", "research", "--force", "--reason", "fixture")
        report = "docs/plan/bugs/B001.research-report.md"
        (self.work / report).write_text("""---
research_schema: 2
rigor: simple
categories: [backend]
related_work: false
---
# Route fixture
## Scope
Record the selected route.
## Research checklist
- [x] `scope_vocabulary` — Evidence: `app`
- [x] `affected_implementation_or_contract` — Evidence: `app`
## Findings
The receiver records the submitted model.
## Decisions
Verify each selection.
## Sources
- `app`
""")
        run("git", "add", "-f", report)
        run("git", "-c", "user.name=Fixture", "-c", "user.email=test@example.invalid",
            "commit", "-qm", "record research")
        self.shark("related-docs", "add", "Research report", report, "--bug=B001")
        self.workflow = self.work / "shark-data/workflow/bug.yaml"
        self.template = self.workflow.read_text()
        wrapper = self.directory / "shark"
        wrapper.write_text('#!/bin/sh\nprintf "%s\\n" "$1" >> "$CASE/counts"\nexec "$B077_SHARK" "$@"\n')
        wrapper.chmod(0o700)

    def submit(self, request, returned, *, mapped=False, effort="low", default="", observable=True,
               observed="", denied=False, known=False, dropped=False):
        path = self.directory
        before, stage = self.template.split("  research:\n")
        provider = "anthropic" if mapped else "openai"
        stage = stage.replace("    provider: anthropic", "    provider: " + provider)
        stage = stage.replace("    model: haiku", "    model: " + json.dumps(request)
                              + ("\n    effort: " + effort if effort else ""))
        self.workflow.write_text(before + "  research:\n" + stage)
        keyed = json.loads(self.shark("next", "B001", "--workdir", str(self.work), "--json").stdout)
        self.assertEqual((provider, request, effort), (keyed["provider"], keyed["model"], keyed.get("effort", "")))
        (path / "request").write_text(json.dumps(keyed))
        (path / "response").write_text(json.dumps(returned))
        for name in ("calls", "counts", "route"):
            (path / name).unlink(missing_ok=True)
        env = dict(os.environ, CASE=str(path), PATH=str(path)+os.pathsep+os.environ["PATH"],
                   CODEX_HOME=str(path/"codex-home"), ENTITY_KEY="B001", CLAUDE_SID="fixture",
                   REPO_ROOT=str(self.work), SELECTED_HARNESS="codex", SELECTED_VERSION="fixture-version",
                   DENIED=str(int(denied)), KNOWN_COST=str(int(known)), DROP_ARG=str(int(dropped)), OBSERVED=observed, OUTCOME="pass", CURRENT_FILE=str(path/"current"),
                   OBSERVATION_FILE=str(path/"observation"), CODEX_BIN=str(self.cli), SELECTED_PROVIDER="openai",
                   REQUESTED_MODEL=request, REQUESTED_EFFORT=effort, SELECTED_MODEL="stale",
                   DEFAULT_MODEL=default, MAPPING_REF=MAPPING if mapped else "", OBSERVATION_READY=str(int(observable)),
                   RIDER_ROOT=str(RIDER), REQUEST_FILE=str(path / "request"), SELECTED_PROFILE="fixture",
                   WORKDIR=str(self.work), ROUTE_FILE=str(path / "route"), SELECTED_SANDBOX="read-only",
                   PROMPT_FILE=str(path / "prompt"), ARGV_FILE=str(path / "argv"), RESULT_FILE=str(path / "result"),
                   EVENT_FILE=str(path / "events"))
        result = subprocess.run(["bash", "-eu", "-c", commands()], env=env, cwd=self.work, capture_output=True)
        calls = [json.loads(line) for line in (path / "calls").read_text().splitlines()] if (path / "calls").exists() else []
        self.assertFalse((path / "decoy-used").exists())
        self.assertTrue(calls, result.stderr)
        self.assertEqual("read", calls[0]["method"], "first boundary must be fresh local discovery")
        self.assertEqual(str(self.work), calls[0]["params"]["cwd"])
        self.assertEqual(str(self.work), calls[0]["cwd"])
        self.assertIn("fixture", calls[0]["argv"])
        self.assertIn("model_provider=openai", calls[0]["argv"])
        self.assertEqual(["model=" + request] if request and not mapped else [],
                         [x for x in calls[0]["argv"] if x.startswith("model=")])
        expected_effort = ["model_reasoning_effort=" + effort] if effort else []
        self.assertEqual(expected_effort, [x for x in calls[0]["argv"] if x.startswith("model_reasoning_effort=")])
        self.stderr = (result.stdout + result.stderr).decode()
        return calls, json.loads((path / "route").read_text()), result.returncode

    def test_success(self):
        for request, model, mapped, default in (("haiku", "one", True, ""), ("haiku", "two", True, ""),
                                                ("pinned", "pinned", False, ""), ("", "default", False, "default")):
            for effort in ("low", ""):
                with self.subTest(request=request, model=model, effort=effort):
                    config = dict(model_provider="openai", model=model, model_reasoning_effort=effort or "xhigh")
                    calls, receipt, rc = self.submit(request, config, mapped=mapped, default=default, effort=effort)
                    self.assertEqual(0, rc, self.stderr)
                    self.assertEqual(model, receipt["selected_model"])
                    self.assertEqual(["read", "exec"], [c["method"] for c in calls])
                    current = json.loads((self.directory/"current").read_text())
                    self.assertEqual("repair_plan", json.loads(self.shark("get", "B001", "--json").stdout)["status"])
                    with sqlite3.connect(self.work / "shark-tasks.db") as db:
                        actor = db.execute("select changed_by from entity_history order by id desc limit 1").fetchone()[0]
                    self.assertEqual("researcher@openai/" + model, actor)
                    self.shark("status", "set", "B001", "research", "--force", "--reason", "reset isolated fixture")
                    self.assertEqual(current["prompt"], calls[1]["prompt"])
                    self.assertEqual(["codex", "fixture-version", model], calls[1]["claim"])
                    history = json.loads(self.shark("status", "history", "B001", "--json").stdout)
                    self.assertIn("researcher@openai/" + model, [h.get("agent") for h in history["history"]])
                    self.assertEqual(str(self.work), calls[1]["cwd"])
                    argv = calls[1]["argv"]
                    capture = (self.directory / "argv").read_bytes().decode().split("\0")
                    self.assertEqual([str(self.cli), *argv, ""], capture)
                    self.assertIn("model_provider=openai", argv)
                    self.assertIn("fixture", argv)
                    self.assertEqual(1 if request else 0, argv.count("--model"))
                    if request:
                        self.assertEqual(model, argv[argv.index("--model") + 1])
                    self.assertEqual(["model_reasoning_effort=low"] if effort else [],
                                     [x for x in argv if x.startswith("model_reasoning_effort=")])

    def test_post_dispatch_rejection(self):
        for denied, observed, known, dropped in ((True, "", True, False), (True, "", False, False),
                                                (False, "previous-selection", False, False), (False, "", False, True)):
            with self.subTest(denied=denied):
                stale = self.directory / "result"
                stale.write_text("RECOMMENDED OUTCOME: pass\n")
                calls, _, rc = self.submit("haiku", dict(model_provider="openai", model="current",
                                           model_reasoning_effort="low"), mapped=True, denied=denied,
                                           observed=observed, known=known, dropped=dropped)
                self.assertNotEqual(0, rc)
                self.assertEqual(["read", "exec"], [c["method"] for c in calls])
                commands = (self.directory / "counts").read_text().splitlines()
                self.assertEqual(1, commands.count("claim"))
                self.assertEqual(1, commands.count("release"))
                self.assertNotIn("status", commands)
                self.assertEqual("research", json.loads(self.shark("get", "B001", "--json").stdout)["status"])
                with sqlite3.connect(self.work / "shark-tasks.db") as db:
                    self.assertEqual(0, db.execute("select count(*) from entity_claims").fetchone()[0])
                receipt = json.loads((self.directory / "observation").read_text())
                self.assertEqual("unverified" if denied or dropped else "mismatch", receipt["route_status"])
                if known:
                    self.assertEqual("denied-observation", receipt.get("attempt_model"))
                    self.assertEqual(({"input_tokens": 11}, 0.25), (receipt["usage"], receipt["cost"]))
                elif denied:
                    self.assertFalse({"usage", "cost"} & receipt.keys())
                if denied or dropped:
                    self.assertNotIn("effective_model", receipt)
                else:
                    self.assertEqual(observed, receipt["effective_model"])

    def test_refusal(self):
        for request, model, mapped in (("haiku", "found", True), ("pin", "pin", False), ("", "default", False)):
            good = dict(model=model, model_provider="openai", model_reasoning_effort="low")
            conflicts = [dict(model_provider="other"), dict(model_reasoning_effort="xhigh"),
                         dict(model_reasoning_effort=None), dict(model=None), dict(model="")]
            if not mapped:
                conflicts.append(dict(model="different"))
            for conflict in conflicts:
                with self.subTest(request=request, conflict=conflict):
                    config = {k:v for k,v in (good | conflict).items() if v is not None}
                    calls, receipt, rc = self.submit(request, config, mapped=mapped, default="default")
                    self.assertNotEqual(0, rc)
                    self.assertEqual(["read"], [c["method"] for c in calls])
                    self.assertFalse((self.directory / "counts").exists())
                    self.assertEqual("unverified", receipt["route_status"])
                    self.assertEqual(request, receipt["requested_model"])
                    self.assertFalse({"worker_id", "effective_model", "effective_provider", "session_id"} & receipt.keys())
                    if mapped and not config.get("model"):
                        self.assertEqual("openai", receipt["selected_provider"])
                        self.assertNotIn("selected_model", receipt)
        _, receipt, rc = self.submit("haiku", good, mapped=True, observable=False)
        self.assertNotEqual(0, rc)
        self.assertEqual(good["model"], receipt["selected_model"])


if __name__ == "__main__":
    unittest.main()
