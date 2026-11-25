"""Correlate one fresh codex exec with its sole completed rollout turn."""
import argparse
import json
from pathlib import Path
import sys
from uuid import UUID


def records(path):
    with path.open(encoding="utf-8") as stream:
        for line in stream:
            yield json.loads(line)


def argv_agrees(capture, selection, result_file):
    expected = [selection["executable"], "exec", "-C", selection["workdir"],
                "--config", "model_provider=" + selection["selected_provider"]]
    if selection["profile"]:
        expected.extend(["--profile", selection["profile"]])
    if selection["sandbox"]:
        expected.extend(["--sandbox", selection["sandbox"]])
    if selection.get("mapping_ref") or selection["requested_model"]:
        expected.extend(["--model", selection["selected_model"]])
    if selection.get("effort"):
        expected.extend(["--config", "model_reasoning_effort=" + selection["effort"]])
    expected.extend(["--json", "--output-last-message", str(result_file), "-", ""])
    return capture.read_bytes().split(b"\0") == [arg.encode() for arg in expected]


def observe(events, home, selection, receipt, *, rejected):
    threads = []
    started = completed = 0
    completion_turn_ids = []
    for event in records(events):
        kind = event.get("type")
        if kind == "thread.started":
            threads.append(event.get("thread_id"))
        elif kind == "turn.started":
            started += 1
        elif kind in ("turn.completed", "turn.failed", "error"):
            completed += kind == "turn.completed"
            rejected |= kind != "turn.completed"
            if kind == "turn.completed":
                completion_turn_ids.append(event.get("turn_id"))
            if "usage" in event:
                receipt["usage"] = event["usage"]
            if "cost" in event:
                receipt["cost"] = event["cost"]
    if len(threads) != 1 or not isinstance(threads[0], str):
        return "unverified"
    worker = threads[0]
    if str(UUID(worker)) != worker:
        return "unverified"
    receipt["worker_id"] = worker
    if started != 1:
        return "unverified"
    files = list((home / "sessions").rglob("rollout-*-" + worker + ".jsonl"))
    if len(files) != 1:
        return "unverified"
    providers, turns, models, efforts, completions = set(), set(), set(), set(), []
    for item in records(files[0]):
        data = item.get("payload", {})
        if item.get("type") == "session_meta":
            if data.get("id") != worker:
                return "unverified"
            providers.add(data.get("model_provider"))
        elif item.get("type") == "turn_context":
            turns.add(data.get("turn_id"))
            models.add(data.get("model"))
            efforts.add(data.get("effort"))
        elif item.get("type") == "event_msg" and data.get("type") == "task_complete":
            completions.append(data.get("turn_id"))
    if rejected:
        # Identity of a failed attempt is diagnostic only. There is no admitted result.
        if len(providers) == 1 and all(providers):
            receipt["attempt_provider"] = providers.pop()
        if len(turns) == 1 and all(turns) and len(models) == 1 and all(models):
            receipt["attempt_model"] = models.pop()
        return "unverified"
    if completed != 1:
        return "unverified"
    if (len(completion_turn_ids) != 1
            or len(completions) != 1 or not completions[0]
            or (completion_turn_ids[0] is not None
                and completion_turn_ids[0] != completions[0])
            or turns != {completions[0]}):
        return "unverified"
    if len(providers) != 1 or len(models) != 1 or not all(providers) or not all(models):
        return "unverified"
    if "effort" in selection and (len(efforts) != 1 or not all(efforts)):
        return "unverified"
    provider, model = providers.pop(), models.pop()
    receipt.update(turn_id=completions[0], effective_model=model, effective_provider=provider)
    if "effort" in selection:
        receipt["effective_effort"] = efforts.pop()
        if receipt["effective_effort"] != selection["effort"]:
            return "mismatch"
    if (provider, model) != (selection["selected_provider"], selection["selected_model"]):
        return "mismatch"
    return "verified"


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    for name in ("events", "home", "selection", "argv", "result_file"):
        parser.add_argument(name, type=Path)
    parser.add_argument("--worker-exit", type=int, required=True)
    args = parser.parse_args()
    receipt = dict(phase="post-dispatch", route_status="unverified", evidence_ref=str(args.events))
    receipt["exit_status"] = args.worker_exit
    try:
        selection = json.loads(args.selection.read_text())
        for key in ("host_worker_class", "requested_provider", "requested_model", "selected_provider", "selected_model"):
            receipt[key] = selection[key]
        try:
            matched = argv_agrees(args.argv, selection, args.result_file)
        except OSError:
            matched = False
        if not matched:
            receipt["reason"] = "captured invocation differs from selected route"
        receipt["route_status"] = observe(args.events, args.home, selection, receipt,
                                          rejected=args.worker_exit != 0 or not matched)
    except (OSError, KeyError, TypeError, ValueError, AttributeError):
        # A missing or unsupported record cannot establish a producing identity.
        receipt["reason"] = "unreadable or invalid invocation metadata"
    print(json.dumps(receipt))
    return 0 if receipt["route_status"] == "verified" else 1


if __name__ == "__main__":
    sys.exit(main())
