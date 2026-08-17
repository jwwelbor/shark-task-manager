#!/usr/bin/env bash
# lifecycle-prelude.sh --scenario <package.yaml> [--replay <result.json>]
#   --run-id <id> --output <prelude.jsonl>
#   [--fixture-root <dir> --scratch-root <dir> --evaluator-root <dir>]
#
# F08-004's independent pre-dispatch contract.  It consumes an admitted I-04
# package and, for feature scenarios, an already completed I-06 result.  It
# does not run a provider or implement replay/question persistence.  Questions
# are routed through the public Shark commands so F08-002 can join their
# durable result to I-07.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BENCH_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
command -v python3 >/dev/null 2>&1 || {
	echo "lifecycle-prelude: python3 not found on PATH" >&2
	exit 2
}

LIFECYCLE_PRELUDE_BENCH_DIR="$BENCH_DIR" python3 - "$@" <<'PY'
import json
import os
import subprocess
import sys
import tempfile
from pathlib import Path

try:
    import yaml
except ImportError as exc:
    print(f"lifecycle-prelude: PyYAML is required: {exc}", file=sys.stderr)
    raise SystemExit(2)


STAGES = ("D01", "D02", "D03", "D04", "D05")
FEATURE_FAMILY = "feature"


class GateError(RuntimeError):
    """A normal, durable gate result rather than a script-authoring error."""


def usage():
    print(
        "usage: lifecycle-prelude.sh --scenario <package.yaml> "
        "[--replay <result.json>] --run-id <id> --output <prelude.jsonl> "
        "[--fixture-root <dir> --scratch-root <dir> --evaluator-root <dir>]",
        file=sys.stderr,
    )
    raise SystemExit(2)


def parse_args(argv):
    values = {"replay": "", "fixture_root": "", "scratch_root": "", "evaluator_root": ""}
    required = {"--scenario": "scenario", "--run-id": "run_id", "--output": "output"}
    options = {**required, "--replay": "replay", "--fixture-root": "fixture_root", "--scratch-root": "scratch_root", "--evaluator-root": "evaluator_root"}
    index = 0
    while index < len(argv):
        option = argv[index]
        if option not in options or index + 1 >= len(argv):
            usage()
        values[options[option]] = argv[index + 1]
        index += 2
    if any(not values[key] for key in required.values()):
        usage()
    roots = tuple(values[key] for key in ("fixture_root", "scratch_root", "evaluator_root"))
    if any(roots) and not all(roots):
        print("lifecycle-prelude: all three isolation roots are required together", file=sys.stderr)
        raise SystemExit(2)
    return values


def read_yaml(path, label):
    try:
        value = yaml.safe_load(Path(path).read_text(encoding="utf-8")) or {}
    except (OSError, yaml.YAMLError) as exc:
        print(f"lifecycle-prelude: cannot read {label} {path}: {exc}", file=sys.stderr)
        raise SystemExit(2)
    if not isinstance(value, dict):
        print(f"lifecycle-prelude: {label} must be a YAML mapping: {path}", file=sys.stderr)
        raise SystemExit(2)
    return value


def read_json(path, label):
    try:
        value = json.loads(Path(path).read_text(encoding="utf-8"))
    except (OSError, json.JSONDecodeError) as exc:
        print(f"lifecycle-prelude: cannot read {label} {path}: {exc}", file=sys.stderr)
        raise SystemExit(2)
    if not isinstance(value, dict):
        print(f"lifecycle-prelude: {label} must be a JSON object: {path}", file=sys.stderr)
        raise SystemExit(2)
    return value


def write_record(path, record):
    destination = Path(path).resolve()
    destination.parent.mkdir(parents=True, exist_ok=True)
    fd, temporary = tempfile.mkstemp(prefix=f".{destination.name}.", dir=destination.parent)
    try:
        with os.fdopen(fd, "w", encoding="utf-8") as stream:
            stream.write(json.dumps(record, sort_keys=True, separators=(",", ":")))
            stream.write("\n")
            stream.flush()
            os.fsync(stream.fileno())
        os.replace(temporary, destination)
    except OSError:
        try:
            os.unlink(temporary)
        except OSError:
            pass
        raise


def stage_records(package):
    prelude = (package.get("stage_matrix") or {}).get("prelude")
    if not isinstance(prelude, dict):
        raise GateError("I-04 package has no stage_matrix.prelude")
    records = []
    for stage in STAGES:
        config = prelude.get(stage)
        if not isinstance(config, dict) or not isinstance(config.get("applicable"), bool):
            raise GateError(f"I-04 prelude stage {stage} must declare boolean applicable")
        if not config["applicable"] and not str(config.get("reason", "")).strip():
            raise GateError(f"I-04 non-applicable stage {stage} must declare a reason")
        records.append({"stage": stage, "applicable": config["applicable"], "reason": config.get("reason", "")})
    return records


def validate_replay(package, replay_path, expected_stages):
    if not replay_path:
        raise GateError("feature scenario is missing authorized I-06 replay result")
    result = read_json(replay_path, "I-06 replay result")
    if result.get("terminal_outcome") != "complete":
        raise GateError(f"I-06 replay terminal_outcome is not complete: {result.get('terminal_outcome')!r}")
    scenario = result.get("scenario") or {}
    if scenario.get("scenario_id") not in (None, package.get("scenario_id")):
        raise GateError("I-06 replay scenario_id does not match I-04 package")
    stages = result.get("stages")
    if not isinstance(stages, list):
        raise GateError("I-06 replay result has no stages array")
    seen = []
    for item in stages:
        if not isinstance(item, dict) or item.get("stage") not in STAGES:
            raise GateError("I-06 replay contains a malformed or unknown stage")
        if item["stage"] in seen:
            raise GateError(f"I-06 replay contains duplicate stage {item['stage']}")
        seen.append(item["stage"])
    if seen != list(STAGES):
        raise GateError(f"I-06 replay stages are incomplete or out of order: {seen}")
    return result


def run_shark(args, cwd):
    shark = os.environ.get("SHARK_BIN", "shark")
    try:
        process = subprocess.run([shark, *args], cwd=cwd, text=True, capture_output=True, check=False)
    except OSError as exc:
        raise GateError(f"unable to execute public Shark command: {exc}") from exc
    if process.returncode != 0:
        detail = process.stderr.strip() or process.stdout.strip()
        raise GateError(f"shark {' '.join(args)} failed: {detail}")
    try:
        return json.loads(process.stdout.strip() or "{}")
    except json.JSONDecodeError as exc:
        raise GateError(f"shark {' '.join(args)} returned non-JSON output: {exc}") from exc


def route_questions(result, run_id, scratch_root):
    questions = result.get("questions", [])
    if questions is None:
        questions = []
    if not isinstance(questions, list):
        raise GateError("I-06 questions must be an array")
    routed = []
    for question in questions:
        if not isinstance(question, dict):
            raise GateError("I-06 question entry must be an object")
        fields = ("question_key", "current_responder", "owner", "summary", "evidence_pointer", "resolution_kind", "resolution_pointer")
        if any(not str(question.get(field, "")).strip() for field in fields):
            raise GateError(f"Question entry is missing an authorized field: {question.get('question_key', '?')}")
        key = question["question_key"]
        next_response = run_shark(["next", key, "--json"], scratch_root)
        block = next_response.get("question_block") or {}
        if block.get("question_key") not in (None, key) or block.get("current_responder") not in (None, question["current_responder"]):
            raise GateError(f"Question {key} does not match the authorized replay handoff")
        claim = run_shark(["claim", key, "--by", f"bench-{run_id}", "--json"], scratch_root)
        session = claim.get("session_id")
        if not session:
            raise GateError(f"Question {key} claim returned no session_id")
        run_shark(["question", "respond", key, "--session", session, "--responder", question["current_responder"], "--summary", question["summary"], "--evidence-pointer", question["evidence_pointer"]], scratch_root)
        run_shark(["question", "resolve", key, "--owner", question["owner"], "--resolution-kind", question["resolution_kind"], "--resolution-pointer", question["resolution_pointer"]], scratch_root)
        routed.append({**question, "session_id": session, "terminal_result": question["resolution_kind"]})
    return routed


def isolation_gate(package_path, fixture_root, scratch_root, evaluator_root):
    guard = Path(os.environ["LIFECYCLE_PRELUDE_BENCH_DIR"]) / "scripts" / "verify-evidence-roots.sh"
    try:
        process = subprocess.run([str(guard), package_path, fixture_root, scratch_root, evaluator_root], text=True, capture_output=True, check=False)
    except OSError as exc:
        raise GateError(f"unable to execute I-05 isolation guard: {exc}") from exc
    if process.returncode != 0:
        detail = process.stderr.strip() or process.stdout.strip()
        raise GateError(f"I-05 isolation guard rejected the roots: {detail}")


def main(argv):
    args = parse_args(argv)
    package_path = str(Path(args["scenario"]).resolve())
    package = read_yaml(package_path, "I-04 package")
    if (package.get("admission") or {}).get("status") != "admitted":
        raise GateError("I-04 scenario package is not admitted")
    family = package.get("entity_family")
    stages = stage_records(package)
    record = {"run_id": args["run_id"], "scenario_id": package.get("scenario_id", Path(package_path).stem), "entity_family": family, "prelude": stages, "questions": [], "publication_eligible": False, "terminal_outcome": "complete"}
    try:
        if args["fixture_root"]:
            isolation_gate(package_path, args["fixture_root"], args["scratch_root"], args["evaluator_root"])
        if family == FEATURE_FAMILY:
            replay = validate_replay(package, args["replay"], stages)
            record["replay"] = {"terminal_outcome": replay["terminal_outcome"], "stage_count": len(replay["stages"])}
            record["questions"] = route_questions(replay, args["run_id"], args["scratch_root"] or str(Path(package_path).parent))
        else:
            for stage in record["prelude"]:
                stage["outcome"] = "not_applicable"
                stage["reason"] = stage["reason"] or f"{family} scenarios bypass product-design prelude"
            record["terminal_outcome"] = "not_applicable"
            record["publication_eligible"] = True
        record["prelude"] = [{**stage, "outcome": stage.get("outcome", "complete")} for stage in record["prelude"]]
        record["publication_eligible"] = record["terminal_outcome"] in {"complete", "not_applicable"}
    except GateError as exc:
        record["terminal_outcome"] = "error" if str(exc).startswith("I-05 isolation") else "unresolved_gate"
        record["reason"] = str(exc)
        record["partial_evidence"] = True
        record["publication_eligible"] = False
        write_record(args["output"], record)
        print(f"lifecycle-prelude: {record['terminal_outcome']}: {exc}", file=sys.stderr)
        return 1
    write_record(args["output"], record)
    return 0


try:
    raise SystemExit(main(sys.argv[1:]))
except GateError as exc:
    print(f"lifecycle-prelude: {exc}", file=sys.stderr)
    raise SystemExit(2)
PY
