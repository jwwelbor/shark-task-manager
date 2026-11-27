"""Mapped-catalog cases at Rider's published preflight, worker and result boundaries."""

import copy
import hashlib
import json
import os
from pathlib import Path
import sqlite3
import subprocess
import unittest
from unittest.mock import patch

from b077_codex_route import RiderFixture, catalog_row


class CatalogCases(RiderFixture):
    def test_unsuitable_configured_research_model_never_claims(self):
        config = dict(model_provider="openai", model="opaque", model_reasoning_effort="high")
        pages = [
            dict(
                data=[
                    catalog_row(
                        "opaque",
                        "Our most capable model for complex coding and professional work.",
                    )
                ]
            )
        ]
        calls, receipt, rc = self.submit("haiku", config, mapped=True, effort="high", pages=pages)
        self.assertNotEqual(0, rc, "a known unsuitable research model reached the result writer")
        self.assertEqual("unsupported", receipt["route_status"])
        self.assertEqual(["read", "catalog"], [c["method"] for c in calls])
        self.assertFalse((self.directory / "counts").exists())

    def approve_fixture_plan(self):
        def git(*args):
            return (
                subprocess.check_output(
                    [
                        "git",
                        "-c",
                        "user.name=Fixture",
                        "-c",
                        "user.email=test@example.invalid",
                        *args,
                    ],
                    cwd=self.work,
                )
                .decode()
                .strip()
            )

        plan = Path("docs/plan/bugs/B001.repair-plan.md")
        report = Path("docs/review/plan-review.md")
        baseline = git("rev-parse", "HEAD")
        (self.work / plan).write_text("Fixture plan for the isolated route observation.\n")
        git("add", "-f", str(plan))
        git("commit", "-qm", "fixture plan")
        digest = "sha256:" + hashlib.sha256((self.work / plan).read_bytes()).hexdigest()
        (self.work / report).parent.mkdir(parents=True, exist_ok=True)
        (self.work / report).write_text(f"plan_path: {plan}\nplan_sha256: {digest}\nFixture approval.\n")
        self.shark("related-docs", "add", "Plan review", str(report), "--bug=B001")
        proof = [
            dict(kind="bug_repair_plan", pointer=str(plan), summary=digest),
            dict(kind="bug_review_baseline", pointer=baseline),
            dict(
                kind="plan_review_report",
                pointer=str(report),
                summary="sha256:" + hashlib.sha256((self.work / report).read_bytes()).hexdigest(),
            ),
        ]
        result = self.work / ".shark/runs/fixture-input/result.json"
        result.parent.mkdir(parents=True, exist_ok=True)
        result.write_text(
            json.dumps(
                dict(
                    kind="final",
                    recommended_outcome="pass",
                    evidence=proof,
                    gate_result=dict(schema_version=1, summary="Fixture approval"),
                )
            )
        )
        self.shark(
            "status",
            "set",
            "B001",
            "plan_review",
            "--force",
            "--reason",
            "isolated plan review",
        )
        self.shark("claim", "B001", "--by", "fixture", "--session", "fixture-plan")
        self.shark(
            "run",
            "B001",
            "--apply-result",
            str(result),
            "--run-id",
            "fixture-plan",
            "--session",
            "fixture-plan",
        )
        self.shark("release", "B001", "--session", "fixture-plan", "--outcome", "pass")
        with (
            sqlite3.connect(self.work / "shark-tasks.db") as db,
            sqlite3.connect(self.directory / "approved.db") as snapshot,
        ):
            db.backup(snapshot)

    def catalog_case(
        self,
        stage,
        requested,
        pages,
        expected,
        *,
        configured="B",
        chosen="B",
        effort="high",
        profile="high",
        selected_effort="high",
        confirmation=None,
        observed_effort=None,
        reason=None,
    ):
        with (
            sqlite3.connect(self.directory / "approved.db") as snapshot,
            sqlite3.connect(self.work / "shark-tasks.db") as db,
        ):
            snapshot.backup(db)
        self.shark(
            "status",
            "set",
            "B001",
            stage,
            "--force",
            "--reason",
            "isolated catalog case",
        )
        config = dict(model_provider="openai", model=configured, notify=["/fixture/initial-only"])
        if profile:
            config["model_reasoning_effort"] = profile
        pinned = dict(model_provider="openai", notify=["/fixture/confirmation-only"])
        if profile:
            pinned["model_reasoning_effort"] = profile
        if confirmation:
            pinned.update(confirmation)
        wire_pages = [
            dict(data=rows, nextCursor=str(i + 1) if i + 1 < len(pages) else None)
            for i, rows in enumerate(pages)
        ]
        output = self.directory / "observation"
        output.unlink(missing_ok=True)
        calls, decision, rc = self.submit(
            requested,
            config,
            mapped=True,
            effort=effort,
            stage_name=stage,
            pages=wire_pages,
            confirmation=pinned,
            observed_effort=selected_effort if observed_effort is None else observed_effort,
        )
        self.assertEqual(wire_pages, [c["response"] for c in calls if c["method"] == "catalog"])
        counts_path = self.directory / "counts"
        counts = counts_path.read_text().splitlines() if counts_path.exists() else []
        workers = [c for c in calls if c["method"] == "exec"]
        final = json.loads(output.read_bytes())
        if trace := os.environ.get("B077_CATALOG_TRACE"):
            with Path(trace).open("a") as stream:
                print(
                    json.dumps(
                        dict(
                            stage=stage,
                            expected=expected,
                            raw_pages=wire_pages,
                            actual=final,
                            decision=decision,
                            calls=[{k: v for k, v in c.items() if k != "prompt"} for c in calls],
                            commands=counts,
                            submitted_prompt_sha256=[
                                hashlib.sha256(c["prompt"].encode()).hexdigest() for c in workers
                            ],
                            requested_effort=effort,
                            exit=rc,
                            graph={
                                str(file): file.read_text()
                                for file in (
                                    self.route_file,
                                    Path(str(self.route_file) + ".decision.json"),
                                    output,
                                )
                                if file.exists()
                            },
                        )
                    ),
                    file=stream,
                )
        self.assertTrue(all(c["method"] in {"read","catalog","exec"} for c in calls))
        self.assertTrue(all(c["cwd"] == str(self.work) and "fixture" in c["argv"] for c in calls))
        self.assertEqual(expected, final["route_status"], self.stderr)
        self.assertNotIn("/fixture/", json.dumps([decision, final]))
        self.assertEqual(str(self.route_file) + ".decision.json", final["decision_ref"])
        self.assertEqual(decision["decision_ref"], final["decision_ref"])
        decision_bytes = Path(final["decision_ref"]).read_bytes()
        self.assertEqual(hashlib.sha256(decision_bytes).hexdigest(), final["decision_sha256"])
        self.assertNotIn(b"/fixture/", decision_bytes)
        self.assertEqual(
            decision["comparison"] if "comparison" in decision else None,
            json.loads(decision_bytes).get("comparison"),
        )
        self.assertEqual(decision.get("reason"), final["decision_reason"])
        if reason:
            self.assertIn(reason, decision["reason"])
        entered = expected in ("verified", "mismatch") or observed_effort == ""
        self.assertEqual(
            {"auth": "unavailable", "eligibility": "unavailable", "reason": "host_unavailable"},
            decision["exact_claude"],
        )
        self.assertEqual("authenticated", decision["codex_auth"])
        self.assertEqual("codex-catalog-description-v1", decision["policy"])
        self.assertEqual(
            (stage, requested, "anthropic", effort or None),
            tuple(decision["request"].get(k) for k in ("status", "model", "provider", "effort")),
        )
        self.assertEqual(configured, decision["readback"]["model"])
        confirmations = [c for c in calls if c["method"] == "read"][1:]
        if entered or confirmation:
            self.assertEqual(1, len(confirmations))
            self.assertIn("model=" + chosen, confirmations[0]["argv"])
            self.assertEqual(
                ["model_reasoning_effort=" + effort] if effort else [],
                [a for a in confirmations[0]["argv"] if a.startswith("model_reasoning_effort=")],
            )
            comparison = decision["comparison"]
            self.assertEqual(chosen, comparison["model"])
            self.assertEqual(
                "requested" if effort else "profile" if profile else "row default",
                comparison["effort_source"],
            )
            self.assertEqual(selected_effort, comparison["effort"])
            self.assertEqual(
                "configured" if chosen == configured else "alternative",
                comparison["branch"],
            )
        else:
            self.assertFalse(confirmations)
        self.assertEqual(int(entered), counts.count("claim"))
        self.assertEqual(int(entered), len(workers))
        with sqlite3.connect(self.work / "shark-tasks.db") as db:
            self.assertEqual(0, db.execute("select count(*) from entity_claims").fetchone()[0])
        if entered:
            (worker,) = workers
            self.assertEqual(["codex", "fixture-version", chosen], worker["claim"])
            self.assertEqual(1, worker["argv"].count("--model"))
            self.assertEqual(chosen, worker["argv"][worker["argv"].index("--model") + 1])
            self.assertEqual(
                ["model_reasoning_effort=" + effort] if effort else [],
                [a for a in worker["argv"] if a.startswith("model_reasoning_effort=")],
            )
            self.assertEqual(
                json.loads((self.directory / "current").read_bytes())["prompt"],
                worker["prompt"],
            )
            if expected == "unverified":
                self.assertNotIn("effective_model", final)
            else:
                self.assertEqual(
                    (
                        worker["thread"],
                        worker["turn"],
                        "openai",
                        chosen,
                        observed_effort or selected_effort,
                    ),
                    tuple(
                        final[k]
                        for k in (
                            "worker_id",
                            "turn_id",
                            "effective_provider",
                            "effective_model",
                            "effective_effort",
                        )
                    ),
                )
        else:
            self.assertFalse(
                {"worker_id", "selected_model", "effective_model", "selected_effort"} & final.keys()
            )
            if expected == "unsupported":
                self.assertNotIn("selected_provider", final)
        self.assertEqual(expected == "verified", rc == 0, self.stderr)
        self.assertEqual(int(expected == "verified" and stage != "research"), counts.count("run"))
        self.assertEqual(int(expected == "verified" and stage == "research"), counts.count("status"))
        target = ("repair_plan" if stage == "research" else "research") if expected == "verified" else stage
        self.assertEqual(target, json.loads(self.shark("get", "B001", "--json").stdout)["status"])
        if expected == "verified" and stage != "research":
            argv = (self.directory/"apply-argv").read_bytes().decode().split("\0")
            self.assertEqual(workers[0]["session"], argv[argv.index("--session")+1])
            self.assertEqual(str(self.work), argv[argv.index("--workdir")+1])
            with sqlite3.connect(self.work/"shark-tasks.db") as db:
                metadata, = db.execute("select metadata from entity_notes where note_type='review' order by id desc limit 1").fetchone()
            summary = json.loads(metadata)
            self.assertEqual(("needs_research",workers[0]["session"]), (summary["outcome_key"],summary["parent_session"]))
        if expected == "verified" and stage == "research":
            history = json.loads(self.shark("status", "history", "B001", "--json").stdout)["history"]
            self.assertEqual(1, sum(h.get("agent") == "researcher@openai/" + chosen for h in history))
        return decision, final

    def test_catalog_stage_matrix(self):
        self.approve_fixture_plan()
        stages = [
            ("research", "haiku"),
            ("repair_plan", "sonnet"),
            ("plan_review", "opus"),
            ("development", "sonnet"),
            ("code_review", "opus"),
            ("qa", "sonnet"),
        ]
        cases = [
            ("Fast and affordable model for easier tasks.", "ARRRRR"),
            ("Legacy fast model for simpler tasks.", "ARRRRR"),
            ("Fast, go-to model for coding and everyday work.", "AARARA"),
            ("Legacy balanced model for agentic coding.", "RARARA"),
            ("Previous generation model for coding and everyday work.", "RARARA"),
            ("Legacy model for coding and everyday work.", "RARARA"),
            (
                "Previous generation model for complex coding and professional work.",
                "RARARA",
            ),
            (
                "Our most capable model for complex coding and professional work.",
                "RAAAAA",
            ),
        ]
        self.assertEqual(8, len({description for description, _ in cases}))
        self.assertEqual(6, len({stage for stage, _ in stages}))
        self.assertEqual(48, sum(len(outcomes) for _, outcomes in cases))
        for description, outcomes in cases:
            for (stage, model), cell in zip(stages, outcomes, strict=True):
                with self.subTest(stage=stage, description=description):
                    self.catalog_case(
                        stage,
                        model,
                        [[catalog_row("B", description)]],
                        "verified" if cell == "A" else "unsupported",
                    )

    def test_raw_catalog_descriptions(self):
        self.approve_fixture_plan()
        canonical = "Legacy balanced model for agentic coding"
        spellings = [
            canonical + ".",
            canonical.upper(),
            "  " + canonical + "  ",
            canonical.replace(" ", "  "),
            "  LeGaCy balanced\tmodel for agentic coding.  ",
        ]

        def row(value):
            return catalog_row("B", value)

        def check(pages, expected):
            self.catalog_case("development", "sonnet", pages, expected)

        self.assertEqual(6, len({canonical, *spellings}))
        for spelling in [canonical, *spellings]:
            with self.subTest(spelling=spelling):
                check([[row(spelling)]], "verified")
        for spelling in spellings:
            for a, b in [(canonical, spelling), (spelling, canonical)]:
                with self.subTest(first=a, second=b):
                    check([[row(a)], [row(b)]], "verified")
        other = "Previous generation model for coding and everyday work"
        for a, b in [(other, canonical), (canonical, other)]:
            with self.subTest(conflict_first=a, conflict_second=b):
                check([[row(a), catalog_row("AA", canonical)], [row(b)]], "unverified")
        for unknown in [
            canonical + " with restricted preview access.",
            "Fast  go-to model for coding and everyday work",
        ]:
            with self.subTest(unknown=unknown):
                check([[row(unknown)]], "unverified")

    def test_alternative_selection_and_confirmation(self):
        self.approve_fixture_plan()
        for stage, model in [
            ("research", "haiku"),
            ("repair_plan", "sonnet"),
            ("development", "sonnet"),
            ("qa", "sonnet"),
            ("plan_review", "opus"),
            ("code_review", "opus"),
        ]:
            fit = (
                "Legacy fast model for simpler tasks."
                if model == "haiku"
                else "Our most capable model for complex coding and professional work."
            )
            rows = [
                [
                    catalog_row(
                        "A",
                        "Efficient model that excels at visual understanding and computer use.",
                    )
                ],
                [catalog_row("B", fit)],
            ]
            with self.subTest(stage=stage):
                self.catalog_case(stage, model, rows, "verified", configured="A", reason="alternative")
                self.catalog_case(
                    stage,
                    model,
                    rows,
                    "unverified",
                    configured="A",
                    confirmation={"model": "A"},
                    reason="confirmation",
                )

    def test_duplicate_selection_facts(self):
        self.approve_fixture_plan()
        regular = catalog_row("B", defaultReasoningEffort="low")
        low = [{"reasoningEffort": "low"}]

        def pages(first, second):
            return [[first, catalog_row("AA")], [second]]

        for field, change, effort, profile in [
            ("supportedReasoningEfforts", low, "high", "high"),
            ("defaultReasoningEffort", "high", "", ""),
            ("isDefault", True, "high", "high"),
        ]:
            altered = dict(regular, **{field: change})
            for first, second in [(regular, altered), (altered, regular)]:
                with self.subTest(field=field, first=first):
                    decision, _ = self.catalog_case(
                        "research",
                        "haiku",
                        pages(first, second),
                        "unverified",
                        effort=effort,
                        profile=profile,
                        configured="A",
                        reason="conflicting",
                    )
                    self.assertEqual("B", decision["conflict"]["model"])
            agreeing = dict(regular, isDefault=True, defaultReasoningEffort="high")
            reordered = dict(
                agreeing,
                supportedReasoningEfforts=list(reversed(agreeing["supportedReasoningEfforts"])),
            )
            for first, second in [(agreeing, reordered), (reordered, agreeing)]:
                self.catalog_case(
                    "research",
                    "haiku",
                    pages(first, second),
                    "verified",
                    configured="A",
                    effort=effort,
                    profile=profile,
                )

    def test_catalog_effort_and_tie_break(self):
        self.approve_fixture_plan()
        row = catalog_row("B")
        alt = catalog_row("AA")
        hidden = catalog_row("0", hidden=True, isDefault=True)
        limited = dict(
            row,
            supportedReasoningEfforts=[{"reasoningEffort": "low"}],
            defaultReasoningEffort="low",
        )
        for rows in [[alt, hidden, row], [row, hidden, alt]]:
            self.catalog_case(
                "research",
                "haiku",
                [rows],
                "verified",
                configured="unlisted",
                chosen="AA",
            )
            self.catalog_case("research", "haiku", [rows], "verified", configured="B", chosen="B")
        self.catalog_case("research", "haiku", [[limited, alt]], "verified", chosen="AA")
        self.catalog_case("research", "haiku", [[limited]], "unsupported")
        for profile, selected in [("xhigh", "xhigh"), ("", "high")]:
            for observed, status in [
                (selected, "verified"),
                ("low", "mismatch"),
                ("", "unverified"),
            ]:
                self.catalog_case(
                    "research",
                    "haiku",
                    [[row]],
                    status,
                    effort="",
                    profile=profile,
                    selected_effort=selected,
                    observed_effort=observed,
                )

    def test_rejected_mapping_inputs(self):
        config = dict(model_provider="openai", model="B", model_reasoning_effort="high")
        for model, claude_auth, eligibility, codex_auth, expected in [
            ("haiku", "authenticated", "eligible", "authenticated", "unverified"),
            ("haiku", "failed", "unusable", "failed", "unverified"),
            ("opus", "failed", "unusable", "authenticated", "unsupported"),
        ]:
            calls, decision, rc = self.submit(
                model,
                config,
                mapped=True,
                effort="high",
                claude_auth=claude_auth,
                claude_eligibility=eligibility,
                codex_auth=codex_auth,
            )
            self.assertNotEqual(0, rc)
            self.assertEqual([], calls)
            self.assertEqual(expected, decision["route_status"])
            self.assertFalse((self.directory / "counts").exists())
        calls, decision, rc = self.submit(
            "haiku",
            config,
            mapped=True,
            effort="high",
            claude_auth="failed",
            claude_eligibility="unusable",
            codex_auth="authenticated",
        )
        self.assertEqual(0, rc, self.stderr)
        self.assertEqual("failed", decision["exact_claude"]["auth"])
        self.assertEqual("authenticated", decision["codex_auth"])
        self.assertEqual(1, sum(c["method"] == "exec" for c in calls))

    def test_incomplete_catalog_never_claims(self):
        config = dict(model_provider="openai", model="B", model_reasoning_effort="high")
        for pages in [
            [dict(data=[catalog_row("B")], nextCursor="0")],
            [dict(data=[catalog_row("B")], nextCursor="1"), dict(data=None)],
            [dict(data=[catalog_row("B", hidden=True), catalog_row("X", "unfamiliar")])],
        ]:
            calls, decision, rc = self.submit("haiku", config, mapped=True, effort="high", pages=pages)
            self.assertNotEqual(0, rc)
            self.assertEqual("unverified", decision["route_status"])
            self.assertFalse((self.directory / "counts").exists())
            self.assertNotIn("selected_model", decision)
            self.assertFalse(any(c["method"] == "exec" for c in calls))

    def test_persisted_decisions(self):
        self.approve_fixture_plan()

        def evidence(configured, profile, effort, confirmed=None):
            rows = [
                [catalog_row("A", "Legacy balanced model for agentic coding.")],
                [catalog_row("B")],
            ]
            decision, final = self.catalog_case(
                "research",
                "haiku",
                rows,
                "unverified" if confirmed else "verified",
                configured=configured,
                effort=effort,
                profile=profile,
                selected_effort=effort or profile or "high",
                confirmation={"model": confirmed} if confirmed else None,
            )
            attempt = self.route_file
            receipt = self.directory / (attempt.name + ".final")
            receipt.write_bytes((self.directory / "observation").read_bytes())
            return attempt, receipt

        def inspect(attempt, receipt, configured, effort_source, disposition):
            data = json.loads(attempt.read_bytes())
            result = json.loads(receipt.read_bytes())
            self.assertEqual(str(attempt), result["evidence_ref"])
            self.assertEqual(
                hashlib.sha256(attempt.read_bytes()).hexdigest(),
                result["attempt_sha256"],
            )
            ref = Path(str(attempt) + ".decision.json")
            for record in (data, result):
                self.assertEqual(str(ref), record["decision_ref"])
                self.assertEqual(
                    hashlib.sha256(ref.read_bytes()).hexdigest(),
                    record["decision_sha256"],
                )
            decision = json.loads(ref.read_bytes())
            self.assertEqual(configured, decision["readback"]["model"])
            self.assertEqual(effort_source, decision["comparison"]["effort_source"])
            self.assertEqual("B", decision["comparison"]["model"])
            self.assertEqual("qualified", decision["catalog"]["B"]["comparison"])
            branch = "configured" if configured == "B" else "alternative"
            self.assertEqual(branch, decision["comparison"]["branch"])
            expected_reason = branch + " model qualified for capability and effort"
            if configured == "A":
                self.assertEqual("insufficient capability", decision["catalog"]["A"]["comparison"])
                self.assertEqual(2, decision["catalog"]["B"]["page"])
            if disposition == "unverified":
                expected_reason = "pinned confirmation differs from catalog selection"
                self.assertEqual("A", decision["confirmation"]["model"])
            self.assertEqual(disposition, result["route_status"])
            self.assertEqual(expected_reason, result["decision_reason"])
            self.assertEqual(expected_reason, decision["reason"])
            for file in (attempt, receipt, ref):
                for marker in (b"/fixture/initial-only", b"/fixture/confirmation-only"):
                    self.assertNotIn(marker, file.read_bytes())

        for profile, effort, source in [
            ("high", "high", "requested"),
            ("xhigh", "", "profile"),
            ("", "", "row default"),
        ]:
            first = evidence("B", profile, effort)
            second = evidence("A", profile, effort)
            for own, other, configured in [(first, second, "B"), (second, first, "A")]:
                inspect(*own, configured, source, "verified")
                receipt = own[1]
                original = receipt.read_bytes()
                for field in ("decision_reason", "decision_ref"):
                    changed = json.loads(original)
                    changed[field] = json.loads(other[1].read_bytes())[field]
                    receipt.write_text(json.dumps(changed))
                    with self.assertRaises(AssertionError):
                        inspect(*own, configured, source, "verified")
                receipt.write_bytes(original)

        for confirmed, status in [(None, "verified"), ("A", "unverified")]:
            attempt, receipt = evidence("A", "high", "high", confirmed)
            ref = Path(str(attempt) + ".decision.json")
            files = (ref, attempt, receipt)
            originals = {p: p.read_bytes() for p in files}
            private_calls = [json.loads(line) for line in (self.directory / "calls").read_text().splitlines()]
            replies = [c["response"] for c in private_calls if c["method"] == "read"]
            self.assertEqual(
                ["/fixture/initial-only", "/fixture/confirmation-only"],
                [r["notify"][0] for r in replies],
            )
            for destination in files:
                for reply in replies:
                    records = {p: json.loads(v) for p, v in originals.items()}
                    records[destination]["extra"] = {"value": copy.deepcopy(reply)}
                    for file in files:
                        if file in (attempt, receipt):
                            records[file]["decision_sha256"] = hashlib.sha256(ref.read_bytes()).hexdigest()
                        if file == receipt:
                            records[file]["attempt_sha256"] = hashlib.sha256(attempt.read_bytes()).hexdigest()
                        file.write_text(json.dumps(records[file]))
                    with self.assertRaisesRegex(AssertionError, "unexpectedly found"):
                        inspect(attempt, receipt, "A", "requested", status)
                for file, value in originals.items():
                    file.write_bytes(value)
            inspect(attempt, receipt, "A", "requested", status)

    def test_preflight_timeout_before_receipt(self):
        timeout = self.directory / "timeout"
        timeout.write_text("#!/bin/sh\nexit 124\n")
        timeout.chmod(0o700)
        calls, decision, rc = self.submit("haiku", {}, mapped=True)
        self.assertNotEqual(0, rc)
        self.assertFalse(calls)
        self.assertFalse((self.directory / "counts").exists())
        self.assertEqual("unverified", decision["route_status"])
        self.assertEqual("haiku", decision["requested_model"])
        self.assertNotIn("selected_model", decision)
        final = json.loads((self.directory / "observation").read_bytes())
        self.assertEqual("unverified", final["route_status"])
        self.assertEqual(str(self.route_file), final["evidence_ref"])

    def test_rejection_retains_only_bounded_host_check_fields(self):
        marker = "/fixture/unfiltered-authentication-message"
        _, decision, rc = self.submit("haiku", {}, mapped=True, claude_auth=marker)
        self.assertNotEqual(0, rc)
        for file in (self.route_file, Path(decision["decision_ref"]), self.directory / "observation"):
            self.assertNotIn(marker, file.read_text())

    def test_changed_decision_cannot_publish_untrusted_evidence(self):
        config = dict(model_provider="openai", model="B", model_reasoning_effort="high")
        for target in ("attempt", "decision"):
            with self.subTest(target=target), patch.dict(os.environ, B077_RECORDING_CORRUPTION=target):
                calls, _, rc = self.submit("haiku", config, mapped=True, effort="high")
                self.assertNotEqual(0, rc)
                self.assertEqual(1, sum(call["method"]=="exec" for call in calls))
                final = json.loads((self.directory/"observation").read_bytes())
                self.assertEqual("unverified", final["route_status"])
                self.assertNotIn("effective_model", final)
                self.assertNotIn("decision_ref",final)
                self.assertEqual("#/reason",final["evidence_ref"])
                self.assertNotIn("status",(self.directory/"counts").read_text().splitlines())

    def test_parent_reads_authentication_from_selected_executable(self):
        config = dict(model_provider="openai", model="B", model_reasoning_effort="high")
        calls, decision, rc = self.submit("haiku", config, mapped=True, effort="high")
        self.assertEqual(0, rc, self.stderr)
        log = self.directory / "local-checks"
        self.assertTrue(log.is_file(), "the selected CLI did not receive a capability check")
        checks = [json.loads(line) for line in log.read_text().splitlines()]
        self.assertEqual(
            [["--version"], ["--profile", "fixture", "login", "status"]], [c["argv"] for c in checks]
        )
        self.assertTrue(all((c["executable"], c["cwd"]) == (str(self.cli), str(self.work)) for c in checks))
        self.assertEqual("authenticated", decision["codex_auth"])
        self.assertEqual(1, sum(c["method"] == "exec" for c in calls))


if __name__ == "__main__":
    unittest.main()
