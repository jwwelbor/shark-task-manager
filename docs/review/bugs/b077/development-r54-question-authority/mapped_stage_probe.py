"""Observe the selected CLI boundary for each approved tuple in a fresh Shark project."""
import hashlib
import json
import subprocess
import sys
from pathlib import Path
from unittest.mock import patch

source_dir = Path(__file__).resolve().parent.parent / "development-6c6345cf"
sys.path.insert(0, str(source_dir))
from assignment_probe import fixture

stages = (
    ("research", "researcher", "haiku"),
    ("repair_plan", "architect", "sonnet"),
    ("plan_review", "tech-lead", "opus"),
    ("development", "developer", "sonnet"),
    ("code_review", "tech-lead", "opus"),
    ("qa", "qa", "sonnet"),
)


def approve_fixture_plan(case):
    def git(*args):
        return subprocess.check_output(["git", "-c", "user.name=Fixture", "-c",
                                        "user.email=test@example.invalid", *args], cwd=case.work).decode().strip()
    plan = Path("docs/plan/bugs/B001.repair-plan.md")
    report = Path("docs/review/plan-review.md")
    baseline = git("rev-parse", "HEAD")
    (case.work / plan).write_text("Fixture plan for the isolated route observation.\n")
    git("add", "-f", str(plan))
    git("commit", "-qm", "fixture plan")
    digest = "sha256:" + hashlib.sha256((case.work / plan).read_bytes()).hexdigest()
    (case.work / report).parent.mkdir(parents=True, exist_ok=True)
    (case.work / report).write_text(f"plan_path: {plan}\nplan_sha256: {digest}\nFixture approval.\n")
    case.shark("related-docs", "add", "Plan review", str(report), "--bug=B001")
    proof = [dict(kind="bug_repair_plan", pointer=str(plan), summary=digest),
             dict(kind="bug_review_baseline", pointer=baseline),
             dict(kind="plan_review_report", pointer=str(report),
                  summary="sha256:" + hashlib.sha256((case.work / report).read_bytes()).hexdigest())]
    result = case.work / ".shark/runs/fixture-input/result.json"
    result.parent.mkdir(parents=True, exist_ok=True)
    result.write_text(json.dumps(dict(kind="final", recommended_outcome="pass", evidence=proof,
                                      gate_result=dict(schema_version=1, summary="Fixture approval"))))
    case.shark("status", "set", "B001", "plan_review", "--force", "--reason", "isolated plan review")
    case.shark("claim", "B001", "--by", "fixture", "--session", "fixture-plan")
    case.shark("run", "B001", "--apply-result", str(result), "--run-id", "fixture-plan", "--session", "fixture-plan")
    case.shark("release", "B001", "--session", "fixture-plan", "--outcome", "pass")


def run(stage, persona, request_model):
    case = fixture.RiderCLI()
    try:
        case.setUp()
        if stage in ("development", "code_review", "qa"):
            approve_fixture_plan(case)
        if stage not in ("research", "development"):
            case.shark("status", "set", "B001", stage, "--force", "--reason", "isolated mapped selector")
        selected = "from-reader-" + stage
        config = dict(model_provider="openai", model=selected, model_reasoning_effort="xhigh")
        procedure = fixture.commands().split("OUTCOME=$(sed", 1)[0]
        procedure += '\nshark release "$ENTITY_KEY" --session "$SID" >/dev/null'
        with patch.object(fixture, "commands", return_value=procedure):
            calls, preflight, rc = case.submit(request_model, config, effort="", mapped=True)
        assert rc == 0, case.stderr
        assert [call["method"] for call in calls] == ["read", "exec"]
        reader, receiving = calls
        assert receiving["cwd"] == reader["cwd"] == str(case.work)
        current = json.loads((case.directory / "current").read_bytes())
        assert [current[k] for k in ("entity_type", "status", "agent_type", "provider", "model")] == [
            "bug", stage, persona, "anthropic", request_model]
        assert current["prompt"].encode() == (case.directory / "prompt").read_bytes() == receiving["prompt"].encode()
        argv = receiving["argv"]
        assert argv.count("--model") == 1
        expected = reader["response"]["model"]
        assert preflight["selected_model"] == receiving["claim"][2] == argv[argv.index("--model")+1] == expected
        observation = json.loads((case.directory / "observation").read_bytes())
        assert observation["route_status"] == "verified"
        assert observation["worker_id"] == receiving["thread"]
        assert observation["effective_provider"] == "openai" and observation["effective_model"] == expected
        assert observation["usage"] == {"input_tokens": 7} and "cost" not in observation
        assert (case.directory / "counts").read_text().splitlines().count("claim") == 1
        return dict(stage=stage, persona=persona, requested=request_model, selected=expected,
                    observed=observation["effective_model"], worker=observation["worker_id"],
                    calls=[call["method"] for call in calls], prompt_bytes=current["prompt_bytes"],
                    input_tokens=7, cost=None)
    finally:
        case.doCleanups()


if __name__ == "__main__":
    for row in stages:
        print(json.dumps(run(*row)), flush=True)
