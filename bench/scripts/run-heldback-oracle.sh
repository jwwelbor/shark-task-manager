#!/usr/bin/env bash
# TC-068 / REQ-F-004: run the held-back predicate through I-05 and I-04.
set -euo pipefail
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BENCH_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
REPO_ROOT="$(cd "$BENCH_DIR/.." && pwd)"
exec python3 - "$SCRIPT_DIR/verify-stage-evidence.sh" "$BENCH_DIR/scenarios/scenarios.yaml" "$REPO_ROOT" "$@" <<'PYEOF'
import argparse
import hashlib
import json
import os
import shutil
import subprocess
import sys
import tempfile

import yaml

guard, scenarios_path, repo_root = sys.argv[1:4]
parser = argparse.ArgumentParser()
parser.add_argument("--scenario", required=True)
parser.add_argument("--i07", required=True)
parser.add_argument("--stage-bundle", required=True)
parser.add_argument("--checkout", required=True)
parser.add_argument("--output", required=True)
args = parser.parse_args(sys.argv[4:])


def digest_file(path):
    with open(path, "rb") as stream:
        return hashlib.sha256(stream.read()).hexdigest()


def load_rows(path):
    with open(path, encoding="utf-8") as stream:
        return [json.loads(line) for line in stream if line.strip()]


def snapshot_root(root):
    """Bounded, symlink-aware snapshot of an agent-visible root."""
    root = os.path.realpath(root)
    snapshot = {}
    if not os.path.isdir(root):
        return snapshot
    for current, directories, files in os.walk(root, followlinks=False):
        directories.sort(); files.sort()
        for name in directories + files:
            path = os.path.join(current, name)
            relative = os.path.relpath(path, root).replace(os.sep, "/")
            if os.path.islink(path):
                value = "symlink:" + os.readlink(path)
            elif os.path.isfile(path):
                value = "file:" + digest_file(path)
            else:
                value = "other"
            snapshot[relative] = value
    return snapshot


def invalid(code, path, detail):
    return {"schema_version": "1.0", "predicate_kind": None, "adapter_id": None, "adapter_version": None, "adapter_calls": 0, "access_event": None, "test_digest": None, "reference_digest": None, "observed_result": "not_run", "cleanup": False, "summary": str(detail)[:240], "invalidity_reasons": [{"code": code, "path": path, "detail": str(detail)[:240]}]}


backup_root = None
backup_checkout = None


def restore_checkout():
    global backup_root, backup_checkout
    if not backup_checkout or not os.path.isdir(backup_checkout):
        return
    restored = args.checkout + ".f09-restore"
    if os.path.lexists(restored):
        shutil.rmtree(restored)
    shutil.copytree(backup_checkout, restored, symlinks=True)
    if os.path.lexists(args.checkout):
        shutil.rmtree(args.checkout)
    os.replace(restored, args.checkout)
    shutil.rmtree(backup_root, ignore_errors=True)
    backup_root = backup_checkout = None


def finish(result, status):
    restore_checkout()
    os.makedirs(os.path.dirname(os.path.abspath(args.output)), exist_ok=True)
    with open(args.output, "w", encoding="utf-8") as stream:
        json.dump(result, stream, sort_keys=True, separators=(",", ":"))
        stream.write("\n")
    if status:
        print("oracle_access " + result["invalidity_reasons"][0]["code"], file=sys.stderr)
    raise SystemExit(status)


try:
    with open(args.scenario, encoding="utf-8") as stream:
        package = yaml.safe_load(stream)
    if not isinstance(package, dict):
        finish(invalid("source_malformed", "/scenario", "scenario package is not an object"), 2)
    lifecycle = load_rows(args.i07)
    if len(lifecycle) != 1:
        finish(invalid("missing_join" if not lifecycle else "duplicate_join", "/i07", "expected exactly one lifecycle record"), 1)
    terminal = (lifecycle[0].get("outcome") or {}).get("terminal")
    if terminal not in {"complete", "resource_limit", "lease_loss", "unresolved_gate", "pause", "archive", "error", "cancellation", "worker_failure", "timeout"}:
        finish(invalid("pre_terminal", "/outcome/terminal", "lifecycle is not terminal; adapter was not invoked"), 1)

    with open(scenarios_path, encoding="utf-8") as stream:
        adapters = (yaml.safe_load(stream) or {}).get("adapters", {})
    adapter_name = (package.get("adapter") or {}).get("name")
    entry = adapters.get(adapter_name)
    if not isinstance(entry, dict) or not entry.get("path"):
        finish(invalid("source_malformed", "/adapter/name", "adapter is not registered"), 2)
    adapter = os.path.join(repo_root, entry["path"], "adapter.sh")
    if not os.access(adapter, os.X_OK):
        finish(invalid("source_missing", "/adapter", "registered adapter is not executable"), 2)

    package_root = os.path.dirname(os.path.realpath(args.scenario))
    evaluator_root = os.path.realpath(os.path.join(package_root, "evaluator"))
    roots_before = {
        "agent_fixture_checkout": snapshot_root(args.checkout),
        "evaluator_only": snapshot_root(evaluator_root),
    }
    evaluator = package.get("evaluator_only") or {}
    sources = []
    for relative in evaluator.get("oracle_tests") or []:
        if not isinstance(relative, str) or os.path.isabs(relative):
            finish(invalid("isolation_violation", "/evaluator_only/oracle_tests", "oracle path must be relative"), 1)
        source = os.path.realpath(os.path.join(package_root, relative))
        if not source.startswith(evaluator_root + os.sep) or not os.path.isfile(source):
            finish(invalid("isolation_violation", relative, "oracle path is outside evaluator root or missing"), 1)
        sources.append(source)
    if not sources:
        finish(invalid("source_missing", "/evaluator_only/oracle_tests", "no held-back tests declared"), 1)

    predicate = package.get("final_predicate") or {}
    kind = predicate.get("kind")
    test_ids = list(predicate.get("acceptance_test_ids") or []) + list(predicate.get("integration_test_ids") or []) + list(predicate.get("child_oracles") or [])
    if kind not in {"f2p_p2p", "acceptance_tests", "p2p_plus_rule_drop", "child_oracles_union"}:
        finish(invalid("source_malformed", "/final_predicate/kind", "unknown predicate kind"), 2)
    if kind == "f2p_p2p":
        test_ids = list(predicate.get("f2p_test_ids") or [])
    if kind != "p2p_plus_rule_drop" and not test_ids:
        finish(invalid("source_malformed", "/final_predicate", "predicate has no test IDs"), 2)

    # The adapter is the sole owner of destination resolution. Preserve a
    # complete copy before granting access so an adapter-resolved collision
    # can be restored byte-for-byte even though the broker reports the
    # authoritative destination only after the adapter has run.
    backup_root = tempfile.mkdtemp(prefix="f09-oracle-checkout-")
    backup_checkout = os.path.join(backup_root, "checkout")
    shutil.copytree(args.checkout, backup_checkout, symlinks=True)
    grant = subprocess.run([guard, args.stage_bundle, "--grant-access", "inject-tests", "--accessor", "f09-heldback-oracle", "--adapter", adapter, "--checkout", args.checkout, "--files", *sources], capture_output=True, text=True)
    if grant.returncode:
        code = "pre_terminal" if "pre_terminal" in grant.stderr else "isolation_violation"
        finish(invalid(code, "/execution_oracle/access_event", grant.stderr.strip() or "I-05 broker refused access"), 1)

    access_path = os.path.join(args.stage_bundle, "access.jsonl")
    events = load_rows(access_path) if os.path.isfile(access_path) else []
    injected = []
    collisions = []
    checkout_root = os.path.realpath(args.checkout)
    for event in events:
        if event.get("accessor") != "f09-heldback-oracle" or not event.get("destination"):
            continue
        destination = event["destination"]
        destination_abs = os.path.realpath(os.path.join(args.checkout, destination))
        if not destination_abs.startswith(checkout_root + os.sep):
            finish(invalid("isolation_violation", "/execution_oracle/access_event/destination", "adapter returned a destination outside the checkout"), 1)
        injected.append(destination_abs)
        if os.path.lexists(os.path.join(backup_checkout, destination)):
            collisions.append((destination, destination_abs))
    if len(injected) != len(sources):
        finish(invalid("isolation_violation", "/execution_oracle/access_event", "broker did not return an authoritative destination for every injected source"), 1)
    if collisions:
        for destination, destination_abs in collisions:
            backup_path = os.path.join(backup_checkout, destination)
            if os.path.islink(backup_path):
                os.remove(destination_abs)
                os.symlink(os.readlink(backup_path), destination_abs)
            elif os.path.isfile(backup_path):
                shutil.copy2(backup_path, destination_abs)
        finish(invalid("cleanup_failure", "/execution_oracle/cleanup", "adapter-resolved destination collided with pre-existing files: " + ",".join(item[0] for item in collisions[:8])), 1)

    command = [adapter, "test", "--checkout", args.checkout]
    adapter_calls = 0
    if kind == "p2p_plus_rule_drop":
        command += ["--include", "tests"]
        execution = subprocess.run(command, capture_output=True, text=True)
        adapter_calls += 1
        try:
            test_result = json.loads(execution.stdout)
        except json.JSONDecodeError:
            test_result = {}
        entries = test_result.get("entries") if isinstance(test_result, dict) else None
        passed = isinstance(entries, list) and bool(entries) and all(isinstance(item, dict) and item.get("outcome") == "pass" for item in entries)
    elif kind == "f2p_p2p":
        # Evaluate the named F2P repro and the shared P2P selection as two
        # adapter calls, then let the established predicate evaluator own the
        # arithmetic and its declared f2p_test_ids/p2p_selection semantics.
        predicate_tmp = tempfile.mkdtemp(prefix="f09-predicate-")
        p2p_command = [adapter, "test", "--checkout", args.checkout]
        for include in (predicate.get("p2p_selection") or {}).get("include") or []:
            p2p_command += ["--include", include]
        for excluded in (predicate.get("p2p_selection") or {}).get("exclude_test_ids") or []:
            p2p_command += ["--exclude-id", excluded]
        named_command = [adapter, "test", "--checkout", args.checkout, "--only-id", *test_ids]
        p2p_run = subprocess.run(p2p_command, capture_output=True, text=True)
        adapter_calls += 1
        named_run = subprocess.run(named_command, capture_output=True, text=True)
        adapter_calls += 1
        try:
            p2p_doc = json.loads(p2p_run.stdout)
            named_doc = json.loads(named_run.stdout)
            merged = {"entries": (p2p_doc.get("entries") or []) + (named_doc.get("entries") or [])}
            test_path = os.path.join(predicate_tmp, "tests.json")
            lint_path = os.path.join(predicate_tmp, "lint.json")
            with open(test_path, "w", encoding="utf-8") as stream:
                json.dump(merged, stream)
            with open(lint_path, "w", encoding="utf-8") as stream:
                json.dump({"issues": []}, stream)
            evaluated = subprocess.run([os.path.join(os.path.dirname(guard), "eval-predicate.sh"), args.scenario, test_path, lint_path], capture_output=True, text=True)
            predicate_result = json.loads(evaluated.stdout)
            passed = evaluated.returncode == 0 and predicate_result.get("result") is True
        except (OSError, TypeError, json.JSONDecodeError):
            passed = False
        finally:
            shutil.rmtree(predicate_tmp, ignore_errors=True)
    else:
        command += ["--only-id", *test_ids]
        execution = subprocess.run(command, capture_output=True, text=True)
        adapter_calls += 1
        try:
            test_result = json.loads(execution.stdout)
        except json.JSONDecodeError:
            test_result = {}
        entries = test_result.get("entries") if isinstance(test_result, dict) else None
        passed = isinstance(entries, list) and bool(entries) and all(isinstance(item, dict) and item.get("outcome") == "pass" for item in entries)
    cleanup = True
    for path in injected:
        try:
            if os.path.lexists(path):
                os.remove(path)
        except OSError:
            cleanup = False
    tests_dir = os.path.join(args.checkout, "tests")
    tests_dir_preexisted = "tests" in roots_before["agent_fixture_checkout"] or os.path.isdir(tests_dir) and any(
        path.startswith("tests/") for path in roots_before["agent_fixture_checkout"]
    )
    if not tests_dir_preexisted:
        try:
            os.rmdir(tests_dir)
        except OSError:
            cleanup = False
    roots_after = {
        "agent_fixture_checkout": snapshot_root(args.checkout),
        "evaluator_only": snapshot_root(evaluator_root),
    }
    residue = []
    for root_name in roots_before:
        before, after = roots_before[root_name], roots_after[root_name]
        for relative, value in after.items():
            if relative not in before or before[relative] != value:
                residue.append(root_name + ":" + relative)
    reasons = []
    if not passed:
        reasons.append({"code": "oracle_failure", "path": "/execution_oracle/observed_result", "detail": "held-back predicate did not pass"})
    if not cleanup:
        reasons.append({"code": "cleanup_failure", "path": "/execution_oracle/cleanup", "detail": "injected evaluator file could not be removed"})
    if residue:
        cleanup = False
        reasons.append({"code": "evaluator_only_residue", "path": "/execution_oracle/cleanup", "detail": "post-cleanup root scan found: " + ",".join(sorted(residue)[:8])})
    reference = evaluator.get("reference_solution")
    reference_path = os.path.realpath(os.path.join(package_root, reference)) if isinstance(reference, str) else ""
    result = {"schema_version": "1.0", "predicate_kind": kind, "adapter_id": adapter_name, "adapter_version": (package.get("adapter") or {}).get("version"), "adapter_calls": adapter_calls, "access_event": events[-1] if events else None, "test_digest": hashlib.sha256(b"".join(open(source, "rb").read() for source in sources)).hexdigest(), "reference_digest": digest_file(reference_path) if reference_path and os.path.isfile(reference_path) else None, "observed_result": "pass" if passed and cleanup else "fail", "cleanup": cleanup, "summary": "held-back predicate completed; output is bounded", "invalidity_reasons": reasons}
    finish(result, 0 if not reasons else 1)
except (OSError, ValueError, TypeError, KeyError, yaml.YAMLError, json.JSONDecodeError) as exc:
    finish(invalid("source_malformed", "/input", str(exc)), 2)
PYEOF
