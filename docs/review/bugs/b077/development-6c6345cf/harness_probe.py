"""C06/C16. Observe a claim-backed plan-review prompt at the selected CLI."""
import hashlib
import json
import os
from pathlib import Path
import sqlite3
from unittest.mock import patch

from assignment_probe import fixture

MARKERS = ('CODEX_INSTRUCTION', 'CLAUDE_INSTRUCTION', 'GENERIC_INSTRUCTION')
template = '{{if isCodex .harness}}' + MARKERS[0] + '{{else if isClaude .harness}}' + MARKERS[1] + '{{else}}' + MARKERS[2] + '{{end}}'
special = "\nUnicode 世界 · 'quoted' · \"quoted\" · $SHELL · `cmd` · two trailing spaces  \n"


def run(mode):
    case = fixture.RiderCLI()
    try:
        case.setUp()
        case.shark("status", "set", "B001", "plan_review", "--force", "--reason", "isolated plan review")
        prompt = case.work / 'shark-data/prompts/bug/plan_review.md'
        prompt.write_text((MARKERS[2] if mode == 'generic-template' else template) + special)
        commands = fixture.commands().split('OUTCOME=$(sed', 1)[0]
        commands += '\nshark release "$ENTITY_KEY" --session "$SID" >/dev/null'
        if mode == 'wrong-family':
            commands = 'SELECTED_HARNESS=codex-cli\n' + commands
        config = dict(model_provider='openai', model='discovered-reviewer', model_reasoning_effort='xhigh')
        with patch.object(fixture, "commands", return_value=commands):
            calls, receipt, rc = case.submit("opus", config, mapped=True, effort="")
        assert rc == 0, case.stderr
        current = json.loads((case.directory / "current").read_text())
        initial = json.loads((case.directory / "request").read_text())
        captured = calls[1]['prompt'].encode()
        keyed = current['prompt'].encode()
        disk = (case.directory / "prompt").read_bytes()
        assert captured == disk == keyed
        digest = hashlib.sha256(captured).hexdigest()
        assert digest == current['prompt_sha256'] and len(captured) == current['prompt_bytes']
        assert current['agent_type'] == "tech-lead" and current['model'] == "opus"
        assert initial["provider"] == "anthropic" and MARKERS[2] in initial["prompt"]
        marker, = [marker for marker in MARKERS if marker.encode() in captured]
        expected = mode == 'matching'
        assert (marker == MARKERS[0]) == expected
        assert receipt['selected_model'] == calls[1]['claim'][2] == current['harness_model'] == config['model']
        return dict(mode=mode, original=[current['status'], current['agent_type'], current['provider'], current['model']],
                    checked_model=receipt['selected_model'], stored_claim=calls[1]['claim'],
                    initial_marker=MARKERS[2], received_marker=marker, expected_marker=MARKERS[0],
                    independently_matched=(marker == MARKERS[0]), keyed_and_received_sha256=digest,
                    keyed_and_received_bytes=len(captured), executable_calls=[c['method'] for c in calls])
    finally:
        case.doCleanups()


if __name__ == "__main__":
    for field in ("SHARK_HARNESS", "SHARK_HARNESS_MODEL", "SHARK_HARNESS_VERSION"):
        os.environ.pop(field, None)
    for mode in ("matching", "wrong-family", "generic-template"):
        print(json.dumps(run(mode)), flush=True)
