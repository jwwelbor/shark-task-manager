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
import time
import unittest


def record_cli():
    directory = Path(os.environ["CASE"])
    argv = sys.argv[1:]
    def record(method, **fields):
        with (directory / "calls").open("a") as output:
            print(json.dumps(dict(method=method, argv=argv, cwd=os.getcwd(), **fields)), file=output)
    if argv[-2:] == ["login","status"] or argv[-1:] == ["--version"]:
        with (directory / "local-checks").open("a") as out:
            print(json.dumps(dict(executable=sys.argv[0],argv=argv,cwd=os.getcwd())),file=out)
        if argv[-1] == "--version":
            print("fixture-version")
        elif os.environ.get("RECORDER_LOGIN") != "authenticated":
            sys.exit(1)
        return
    if "app-server" in argv:
        for line in sys.stdin:
            request = json.loads(line)
            if request.get("method") not in ("initialize", "initialized", "config/read", "model/list"):
                record("unexpected", request_method=request.get("method"))
                sys.exit(1)
            if request["method"] == "initialize":
                print(json.dumps(dict(id=request["id"], result={})), flush=True)
            if request["method"] == "config/read":
                config = json.loads((directory / "response").read_text())
                if any(a.startswith("model=") for a in argv) and (directory / "confirmation").exists():
                    pin = next(a[6:] for a in argv if a.startswith("model="))
                    config = json.loads((directory / "confirmation").read_text())
                    config.setdefault("model", pin)
                record("read", params=request["params"], response=config)
                if os.environ.get("STALL_READ") == "1":
                    time.sleep(30)
                print(json.dumps(dict(id=request["id"], result=dict(config=config))), flush=True)
            if request["method"] == "model/list":
                pages = json.loads((directory / "pages").read_bytes())
                cursor = request["params"].get("cursor") or "0"
                page = pages[int(cursor)]
                record("catalog", params=request["params"], response=page)
                print(json.dumps(dict(id=request["id"], result=page)), flush=True)
        return
    if argv[0] == "exec":
        prompt = sys.stdin.read()
        with sqlite3.connect(Path.cwd() / "shark-tasks.db") as db:
            claim = db.execute("select harness, harness_version, harness_model, session_id from entity_claims where entity_key='B001'").fetchone()
        if os.environ.get("DROP_ARG") == "1":
            capture = Path(os.environ["ARGV_FILE"])
            fields = capture.read_bytes().split(b"\0")
            index = fields.index(b"--model")
            del fields[index:index+2]
            capture.write_bytes(b"\0".join(fields))
        thread, turn = str(uuid4()), str(uuid4())
        record("exec", prompt=prompt, claim=claim[:3], session=claim[3], thread=thread, turn=turn)
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
        # The receiving process applies its own profile and argv. The discovery
        # response may be wrong, so it cannot stand in for the worker observation.
        overrides = dict(argv[i + 1].split("=", 1) for i, arg in enumerate(argv) if arg == "--config")
        provider = overrides.get("model_provider", "other-provider")
        model = argv[argv.index("--model") + 1] if "--model" in argv else os.environ["DEFAULT_MODEL"]
        events = [dict(type="thread.started", thread_id=thread), dict(type="turn.started"),
                  dict(type="turn.completed", usage={"input_tokens":7})]
        for event in events:
            print(json.dumps(event))
        records = [("session_meta", dict(id=thread, model_provider=provider)),
                   ("turn_context", dict(turn_id=turn, model=os.environ.get("OBSERVED") or model,
                                         effort=os.environ.get("OBSERVED_EFFORT", overrides.get("model_reasoning_effort", "xhigh")))),
                   ("event_msg", dict(type="task_complete", turn_id=turn))]
        rollout = Path(os.environ["CODEX_HOME"]) / "sessions" / ("rollout-fixture-" + thread + ".jsonl")
        rollout.parent.mkdir(parents=True, exist_ok=True)
        rollout.write_text("".join(json.dumps(dict(type=t, payload=v)) + "\n" for t,v in records))
        Path(argv[argv.index("--output-last-message")+1]).write_text((directory / "worker-output").read_text())
        return
    record("unexpected")
    raise ValueError(argv)


if __name__ == "__main__" and Path(sys.argv[0]).name == "selected-codex":
    record_cli()
    sys.exit(0)

ROOT = Path(__file__).resolve().parents[3]
RIDER = ROOT / "skills/shark-rider"
MAPPING = "B077-default-bug-anthropic-codex-v1"


def catalog_row(model, description="Fast and affordable model for easier tasks.", **changes):
    return dict(dict(model=model, id=model, description=description, hidden=False, isDefault=False,
                     supportedReasoningEfforts=[{"reasoningEffort": value} for value in ("low", "high", "xhigh")],
                     defaultReasoningEffort="high"), **changes)


def commands(structured=False):
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
    if structured:
        apply_section = run.split("3. Call the shared ingestion", 1)[1].split("4. Release", 1)[0]
        return "\n".join([preflight, claim, block(refresh), launch, block(wait),
                          block(apply_section), 'shark release "$ENTITY_KEY" --session "$SID" --outcome needs_research'])
    return "\n".join([preflight, claim, block(refresh), launch, block(wait),
                      'OUTCOME=$(sed -n "s/^RECOMMENDED OUTCOME: //p" "$RESULT_FILE")',
                      apply.replace("{response.status}", "research"),
                      'shark release "$ENTITY_KEY" --session "$SID" --outcome "$OUTCOME"'])


class RiderFixture(unittest.TestCase):
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
        wrapper.write_text("""#!/bin/sh
set -eu
printf '%s\\n' "$1" >> "$CASE/counts"
if [ "$1" = run ]; then
    printf '%s\\0' "$@" > "$CASE/apply-argv"
fi
if [ "$1" = release ] && [ -f "$CASE/takeover" ]; then
    "$B077_SHARK" claim B001 --force --by successor --session successor --field session_id >/dev/null
fi
exec "$B077_SHARK" "$@"
""")
        wrapper.chmod(0o700)

    def submit(self, request, returned, *, mapped=False, effort="low", default="", observable=True,
               observed="", denied=False, known=False, dropped=False, stall=False, prior_model="stale",
               stage_name="research", pages=None, confirmation=None, observed_effort=None,
               claude_auth="unavailable", claude_eligibility="unavailable", codex_auth="authenticated"):
        path = self.directory
        stage_header = "  " + stage_name + ":\n"
        before, stage = self.template.split(stage_header)

        provider = "anthropic" if mapped else "openai"
        stage = stage.replace("    provider: anthropic", "    provider: " + provider)
        stage = re.sub(r"    model: [^\n]+", lambda match: "    model: " + json.dumps(request)
                              + ("\n    effort: " + effort if effort else ""), stage, count=1)
        self.workflow.write_text(before + stage_header + stage)
        keyed = json.loads(self.shark("next", "B001", "--workdir", str(self.work), "--json").stdout)
        self.assertEqual((provider, request, effort), (keyed["provider"], keyed["model"], keyed.get("effort", "")))
        (path / "worker-output").write_text("RECOMMENDED OUTCOME: pass\n" if stage_name == "research" else
            json.dumps(dict(kind="final", recommended_outcome="needs_research",
                            gate_result=dict(schema_version=1, summary="Revisit fixture research",
                                findings=[dict(severity="blocker", disposition="open", class_key="route-check",
                                    fingerprint="fixture-research", class_statement="Recorded fixture needs research.")],
                                kickbacks=[]))))
        if pages is None:
            pages = [dict(data=[catalog_row(returned.get("model", "fixture"))], nextCursor=None)]
        (path / "pages").write_text(json.dumps(pages))
        if mapped:
            confirm = {k: v for k, v in returned.items() if k != "model"} if confirmation is None else confirmation
            (path / "confirmation").write_text(json.dumps(confirm))
        else:
            (path / "confirmation").unlink(missing_ok=True)
        (path / "request").write_text(json.dumps(keyed))
        (path / "response").write_text(json.dumps(returned))
        self.route_file = path / ("route-" + str(uuid4()) + ".json")
        for name in ("calls", "counts", "local-checks", "apply-argv"):
            (path / name).unlink(missing_ok=True)
        env = dict(os.environ, CASE=str(path), PATH=str(path)+os.pathsep+os.environ["PATH"],
                   CODEX_HOME=str(path/"codex-home"), ENTITY_KEY="B001", CLAUDE_SID="fixture",
                   REPO_ROOT=str(self.work), SELECTED_HARNESS="codex", SELECTED_VERSION="fixture-version",
                   STALL_READ=str(int(stall)), DENIED=str(int(denied)), KNOWN_COST=str(int(known)), DROP_ARG=str(int(dropped)), OBSERVED=observed, OUTCOME="pass", CURRENT_FILE=str(path/"current"),
                   OBSERVATION_FILE=str(path/"observation"), CODEX_BIN=str(self.cli), SELECTED_PROVIDER="openai",
                   REQUESTED_MODEL=request, REQUESTED_EFFORT=effort, SELECTED_MODEL=prior_model,
                   DEFAULT_MODEL=default, MAPPING_REF=MAPPING if mapped else "", OBSERVATION_READY=str(int(observable)),
                   RIDER_ROOT=str(RIDER), REQUEST_FILE=str(path / "request"), SELECTED_PROFILE="fixture",
                   WORKDIR=str(self.work), ROUTE_FILE=str(self.route_file), SELECTED_SANDBOX="read-only",
                   PROMPT_FILE=str(path / "prompt"), ARGV_FILE=str(path / "argv"), RESULT_FILE=str(path / "result"),
                   EVENT_FILE=str(path / "events"))
        env.update(EXACT_CLAUDE_AUTH=claude_auth, EXACT_CLAUDE_ELIGIBILITY=claude_eligibility,
                   RECORDER_LOGIN=codex_auth,
                   EXACT_CLAUDE_REASON="authentication_failed" if claude_auth=="failed" else "host_unavailable", RUN_ID="recording-" + str(uuid4()))
        if observed_effort is not None:
            env["OBSERVED_EFFORT"] = observed_effort
        result = subprocess.run(["bash", "-eu", "-c", commands(stage_name != "research")], env=env, cwd=self.work, capture_output=True)
        calls = [json.loads(line) for line in (path / "calls").read_text().splitlines()] if (path / "calls").exists() else []
        self.assertFalse((path / "decoy-used").exists())
        if calls:
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
        self.assertTrue(self.route_file.stat().st_size, "preflight ended without an attempt receipt")
        return calls, json.loads(self.route_file.read_text()), result.returncode


class RiderCLI(RiderFixture):
    def test_success(self):
        for request, model, mapped, default in (
            ("haiku", "one", True, ""),
            ("haiku", "two", True, ""),
            ("pinned", "pinned", False, ""),
            ("", "default", False, "default"),
        ):
            for effort in ("low", ""):
                with self.subTest(request=request, model=model, effort=effort):
                    config = dict(model_provider="openai", model=model, model_reasoning_effort=effort or "xhigh")
                    calls, receipt, rc = self.submit(request, config, mapped=mapped, default=default, effort=effort)
                    self.assertEqual(0, rc, self.stderr)
                    self.assertEqual(model, receipt["selected_model"])
                    self.assertEqual(["read", "catalog", "read", "exec"] if mapped else ["read", "exec"], [c["method"] for c in calls])
                    current = json.loads((self.directory/"current").read_text())
                    self.assertEqual("repair_plan", json.loads(self.shark("get", "B001", "--json").stdout)["status"])
                    with sqlite3.connect(self.work / "shark-tasks.db") as db:
                        actor = db.execute("select changed_by from entity_history order by id desc limit 1").fetchone()[0]
                    self.assertEqual("researcher@openai/" + model, actor)
                    self.shark("status", "set", "B001", "research", "--force", "--reason", "reset isolated fixture")
                    self.assertEqual(current["prompt"], calls[-1]["prompt"])
                    self.assertEqual(["codex", "fixture-version", model], calls[-1]["claim"])
                    history = json.loads(self.shark("status", "history", "B001", "--json").stdout)
                    self.assertIn("researcher@openai/" + model, [h.get("agent") for h in history["history"]])
                    self.assertEqual(str(self.work), calls[-1]["cwd"])
                    argv = calls[-1]["argv"]
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
                self.assertEqual(["read", "catalog", "read", "exec"], [c["method"] for c in calls])
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
                if dropped:
                    self.assertEqual({"input_tokens": 7}, receipt.get("usage"))
                    self.assertEqual("current", receipt.get("attempt_model"))
                if denied or dropped:
                    self.assertNotIn("effective_model", receipt)
                else:
                    self.assertEqual(observed, receipt["effective_model"])

    def test_denied_after_takeover(self):
        self.shark("bug", "create", "unrelated holder")
        self.shark("claim", "B002", "--by", "unrelated", "--session", "unrelated")
        (self.directory / "takeover").touch()
        calls, _, rc = self.submit("haiku", dict(model_provider="openai", model="found",
                                                 model_reasoning_effort="low"), mapped=True, denied=True)
        self.assertNotEqual(0, rc)
        self.assertEqual(["read", "catalog", "read", "exec"], [c["method"] for c in calls])
        effects = (self.directory / "counts").read_text().splitlines()
        self.assertEqual((1, 1, 0), (effects.count("claim"), effects.count("release"), effects.count("status")))
        with sqlite3.connect(self.work / "shark-tasks.db") as db:
            rows = db.execute("select entity_key, session_id from entity_claims order by entity_key").fetchall()
        self.assertEqual([("B001", "successor"), ("B002", "unrelated")], rows)
        self.assertEqual("research", json.loads(self.shark("get", "B001", "--json").stdout)["status"])

    def test_legacy_worker_notes_require_the_live_session(self):
        # C21. Execute the note commands published by Rider against the note store.
        procedure = (RIDER / "verbs/run.md").read_text()
        section = procedure.split("3. **Persist notes.**", 1)[1].split("4. **", 1)[0]
        commands = re.findall(r"^\s*(shark create note [^\n]+)", section, re.MULTILINE)
        expected_types = ["blocker", "decision", "comment", "comment"]
        self.assertEqual(len(expected_types), len(commands))
        env = dict(os.environ, PATH=str(self.directory) + os.pathsep + os.environ["PATH"],
                   CASE=str(self.directory), SID="author", SHARK_CLAIM_TTL_SECONDS="90")
        for state in ("live", "expired", "no-expiry", "reissued"):
            config_path = self.work / ".sharkconfig.json"
            config = json.loads(config_path.read_bytes())
            config["claim_ttl_seconds"] = 0 if state == "no-expiry" else 90
            config_path.write_text(json.dumps(config))
            allowed = state in ("live", "no-expiry")
            self.shark("claim", "B001", "--by", "parent", "--session", "author", "--force")
            with sqlite3.connect(self.work / "shark-tasks.db") as db:
                if state in ("expired", "no-expiry"):
                    db.execute("update entity_claims set last_heartbeat=datetime('now','-1 day')")
            if state == "reissued":
                self.shark("claim", "B001", "--by", "other", "--session", "successor", "--force")
            for index, command in enumerate(commands):
                with self.subTest(state=state, command=index):
                    marker = f"worker-{state}-{index}"
                    cmd = command.replace("{response.entity_key}", "B001")
                    cmd = re.sub(r'"<[^"]+>"', '"' + marker + '"', cmd)
                    result = subprocess.run(["bash", "-eu", "-c", cmd], env=env, cwd=self.work, capture_output=True)
                    with sqlite3.connect(self.work / "shark-tasks.db") as db:
                        notes = db.execute("select note_type from entity_notes where content=?", (marker,)).fetchall()
                        holder, = db.execute("select session_id from entity_claims where entity_key='B001'").fetchone()
                    self.assertEqual("successor" if state == "reissued" else "author", holder)
                    self.assertEqual([(expected_types[index],)] if allowed else [], notes)
                    self.assertEqual(allowed, result.returncode == 0, result.stderr)
        self.shark("create", "note", "B001", "ordinary direct caller")
        with self.assertRaises(subprocess.CalledProcessError):
            self.shark("create", "note", "B001", "explicit empty session", "--session=")
        with sqlite3.connect(self.work / "shark-tasks.db") as db:
            notes = db.execute("select content from entity_notes where content in (?, ?)",
                               ("ordinary direct caller", "explicit empty session")).fetchall()
        self.assertEqual([("ordinary direct caller",)], notes)

    def test_unanswered_read(self):
        # Keep the real timeout and signals. Shorten the reader deadline in this fixture.
        timeout = shutil.which("timeout")
        wrapper = self.directory / "timeout"
        wrapper.write_text('#!/bin/bash\nif [[ $1 == -k && $3 == 15 ]]; then shift 3; '
                           'exec "' + timeout + '" -k 2 1 "$@"; fi\nexec "' + timeout + '" "$@"\n')
        wrapper.chmod(0o700)
        calls, receipt, rc = self.submit("haiku", {}, mapped=True, stall=True)
        self.assertNotEqual(0, rc)
        self.assertEqual(["read"], [c["method"] for c in calls])
        self.assertFalse((self.directory / "counts").exists())
        self.assertEqual(("preflight", "unverified", "anthropic", "haiku", "codex-cli", MAPPING),
                         tuple(receipt[k] for k in ("phase", "route_status", "requested_provider",
                                                   "requested_model", "host_worker_class", "mapping_ref")))
        self.assertEqual(str(self.route_file), receipt["evidence_ref"])
        self.assertFalse({"selected_provider", "selected_model", "effective_model", "worker_id"} & receipt.keys())

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

    def test_partial_selection_after_success(self):
        config = dict(model_provider="openai", model="first", model_reasoning_effort="low")
        _, first, rc = self.submit("haiku", config, mapped=True)
        self.assertEqual(0, rc, self.stderr)
        self.shark("status", "set", "B001", "research", "--force", "--reason", "reset isolated fixture")
        del config["model"]
        for change in ({}, {"model": ""}, {"model": "second"}):
            with self.subTest(change=change):
                returned = config | change
                calls, receipt, rc = self.submit("haiku", returned, mapped=True,
                                                 observable=change.get("model") != "second",
                                                 prior_model=first["selected_model"])
                self.assertNotEqual(0, rc)
                self.assertEqual(["read", "catalog", "read"] if change.get("model") else ["read"], [call["method"] for call in calls])
                self.assertEqual("unverified", receipt["route_status"])
                self.assertEqual(("anthropic", "haiku", MAPPING, "codex-cli", "openai"),
                                 tuple(receipt[k] for k in ("requested_provider", "requested_model",
                                                           "mapping_ref", "host_worker_class", "selected_provider")))
                reference = json.loads(Path(receipt["evidence_ref"]).read_text())
                self.assertNotEqual(first["evidence_ref"], receipt["evidence_ref"])
                self.assertEqual(first, json.loads(Path(first["evidence_ref"]).read_text()))
                self.assertEqual(returned, reference["readback"])
                self.assertEqual((str(self.cli), "fixture"), (reference["executable"], reference["profile"]))
                if change.get("model"):
                    self.assertEqual("second", receipt["selected_model"])
                else:
                    self.assertNotIn("selected_model", receipt)
                self.assertFalse({"worker_id", "host_session_id", "session_id", "effective_model",
                                  "effective_provider"} & receipt.keys())
                self.assertFalse((self.directory / "counts").exists())


if __name__ == "__main__":
    unittest.main()
