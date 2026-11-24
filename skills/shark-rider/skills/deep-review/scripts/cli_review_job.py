#!/usr/bin/env python3
"""Keep the selected Runner 3 producer reference across publication processes."""
import argparse
import json
from pathlib import Path
import re
import subprocess
import sys
import tempfile

from safe_review_io import MAX_ARTIFACT_BYTES, read_review_file


WRITER = Path(__file__).with_name("adaptive_review.py")
SIDECARS = {"report_path": "", "retained_attempts": ".attempts.json",
            "retained_body": ".body.txt", "retained_decisions": ".decisions.json"}


def check_reference(job, result):
    for key, suffix in SIDECARS.items():
        if result.get(key) != str(job / ("producer.md" + suffix)):
            raise ValueError(f"unexpected {key} for the selected execution")
    if not re.fullmatch(r"[0-9a-f]{64}", result.get("attempts_sha256") or ""):
        raise ValueError("missing producer attempts checksum")
    if result.get("adversarial_model") not in ("codex", "claude") or result.get("verdict") not in (
            "PASS", "PASS-WITH-TRIAGE", "FAIL"):
        raise ValueError("producer did not return a completed CLI review")


def produce(job, arguments):
    # Exclusive directory reservation binds both the destination and its ready record.
    job.mkdir(mode=0o700)
    response = job / "result.json"
    with response.open("xb") as output:
        run = subprocess.run([sys.executable, WRITER, "run-cli", *arguments, "--require-verified-identity",
                              "--review-output-path", job / "producer.md"], stdout=output)
    raw = read_review_file(response, MAX_ARTIFACT_BYTES)
    if run.returncode == 0:
        result = json.loads(raw)
        check_reference(job, result)
        reference = {key: result[key] for key in (*SIDECARS, "attempts_sha256", "adversarial_model", "verdict")}
        pending = job / "job.pending"
        with pending.open("x", encoding="utf-8") as output:
            json.dump(reference, output)
        pending.rename(job / "job.json")
    sys.stdout.buffer.write(raw)
    return run.returncode


def publish(job, arguments):
    try:
        reference = json.loads(read_review_file(job / "job.json", 64 * 1024))
        if not isinstance(reference, dict):
            raise ValueError("invalid producer reference")
        check_reference(job, reference)
    except (OSError, ValueError, TypeError) as exc:
        reason = f"producer reference unavailable: {exc}"
        with tempfile.TemporaryDirectory(prefix="shark-review-diagnostic-") as temp:
            body = Path(temp) / "diagnostic.md"
            body.write_text("**INCOMPLETE** — " + reason, encoding="utf-8")
            return subprocess.call([sys.executable, WRITER, "write-report", *arguments,
                                    "--runner-mode", "incomplete", "--specialists-completed", "0",
                                    "--consolidator-completed", "false", "--verdict", "INCOMPLETE",
                                    "--fallback-reason", reason, "--body-file", body])
    return subprocess.call([sys.executable, WRITER, "write-report", *arguments,
                            "--runner-mode", "adversarial-cli", "--specialists-completed", "0",
                            "--consolidator-completed", "true", "--adversarial-model", reference["adversarial_model"],
                            "--verdict", reference["verdict"], "--body-file", reference["retained_body"],
                            "--review-decisions", reference["retained_decisions"],
                            "--attempts-file", reference["retained_attempts"],
                            "--expected-attempts-sha256", reference["attempts_sha256"]])


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("command", choices=("produce", "publish"))
    parser.add_argument("job", type=Path)
    parser.add_argument("arguments", nargs=argparse.REMAINDER)
    args = parser.parse_args()
    try:
        command = produce if args.command == "produce" else publish
        return command(args.job.resolve(), args.arguments)
    except (OSError, ValueError) as exc:
        print(f"CLI review job failed: {exc}", file=sys.stderr)
        return 2


if __name__ == "__main__":
    sys.exit(main())
