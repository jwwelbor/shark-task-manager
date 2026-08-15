#!/usr/bin/env bash
# canary-usagemapping.sh [--transcript <path>]
#
# The X-09 canary (REQ-F-009, REQ-F-019, AC-007, AC-021): re-verifies
# bench/evidence/usage-mapping.yaml against a REAL captured provider
# envelope -- never a hand-authored fixture shaped to make the mapping
# agree -- following bench/scripts/canary-runsurface.sh's real-invocation
# discipline verbatim: assert the real shape, never re-derive it from
# memory, and name the exact drifted field.
#
# Default (no --transcript): checks every committed envelope fixture under
# bench/scripts/testdata/run/clean-completed/run/transcripts/ -- the
# fixtures bench/README.md's "Confirmed claude CLI JSON envelope field
# names" section already names as the ones asserting real-capture envelope
# shape (T-E40-F02-006). --transcript <path> lets an operator (or a test)
# point the canary at any other transcript instead: a live real transcript
# for a manual Tier 2 spot-check (AC-T3), or a committed mutated-copy
# fixture under bench/scripts/testdata/usagemapping/ for the drift cases
# T-E40-F06-009 itself adds (AC-T2, AC-T4). This canary never modifies or
# reuses bench/scripts/testdata/run/missing-envelope-field/ -- those
# fixtures are collect-run.sh/TC-015's own, and ADR-F06-11 forbids fixture
# coupling across features.
#
# I-05's usage slots (REQ-F-009) read the retained provider envelope from a
# stage transcript's ---STDOUT--- block. REQ-F-019 names two distinct drift
# classes this canary must tell apart in its own output (ADR-F06-11):
#
#   (a) envelope-field drift -- the block decodes as JSON, but one of
#       usage-mapping.yaml's mapped envelope paths is absent from it. Fails
#       naming the exact slot and envelope path: "usage_slot_unavailable
#       slot=<slot> envelope_path=<path>".
#   (b) envelope-availability drift -- the block itself is not decodable
#       JSON at all (e.g. a lifecycle change that starts persisting
#       assistant prose instead of the raw envelope). Fails as ONE whole-
#       source failure naming the transcript path -- never nine
#       independent per-slot failures: "envelope_source_unavailable
#       transcript=<path>".
#
# Only the anthropic_claude_cli provider is checked: usage-mapping.yaml
# declares openai_codex_cli unmapped (REQ-F-009), so it has no slots to
# verify here -- TC-042 (T-E40-F06-002) owns the unmapped-provider
# fail-closed assertion for I-05 snapshots, not this canary.
#
# Output contract (matching canary-runsurface.sh's established convention):
# every diagnostic this script prints goes to STDERR, ending in exactly one
# line reading "PASS" or "FAIL: <field>" as the LAST line.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BENCH_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"

MAPPING_FILE="$BENCH_DIR/evidence/usage-mapping.yaml"
DEFAULT_FIXTURE_DIR="$SCRIPT_DIR/testdata/run/clean-completed/run/transcripts"

usage() {
	echo "usage: canary-usagemapping.sh [--transcript <path>]" >&2
	exit 2
}

transcript_override=""
while [[ $# -gt 0 ]]; do
	case "$1" in
	--transcript)
		[[ $# -ge 2 ]] || usage
		transcript_override="$2"
		shift 2
		;;
	*)
		usage
		;;
	esac
done

command -v python3 >/dev/null 2>&1 || {
	echo "canary-usagemapping: python3 not found on PATH" >&2
	exit 2
}

[[ -f "$MAPPING_FILE" ]] || {
	echo "canary-usagemapping: usage-mapping.yaml not found: $MAPPING_FILE" >&2
	exit 2
}

transcripts=()
if [[ -n "$transcript_override" ]]; then
	[[ -f "$transcript_override" ]] || {
		echo "canary-usagemapping: --transcript file not found: $transcript_override" >&2
		exit 2
	}
	transcripts=("$transcript_override")
else
	shopt -s nullglob
	transcripts=("$DEFAULT_FIXTURE_DIR"/*.log)
	shopt -u nullglob
	[[ "${#transcripts[@]}" -gt 0 ]] || {
		echo "canary-usagemapping: no committed envelope fixtures found under $DEFAULT_FIXTURE_DIR" >&2
		exit 2
	}
fi

python3 - "$MAPPING_FILE" "${transcripts[@]}" <<'PYEOF'
import json
import sys

import yaml


def resolve_slot(envelope, envelope_path):
    """Resolves a usage-mapping.yaml envelope_path against a decoded
    envelope dict. Returns (value, found). The one non-dot-path special
    case (REQ-F-009's model_ids slot) reads the sorted key set of the
    modelUsage object rather than a single scalar field."""
    if envelope_path == "sorted(modelUsage keys)":
        model_usage = envelope.get("modelUsage")
        if not isinstance(model_usage, dict):
            return None, False
        return sorted(model_usage.keys()), True

    cur = envelope
    for part in envelope_path.split("."):
        if not isinstance(cur, dict) or part not in cur:
            return None, False
        cur = cur[part]
    return cur, True


def extract_stdout_block(transcript_text):
    """Extracts the exact bytes between the transcript's ---STDOUT--- and
    ---STDERR--- markers (internal/runner/transcript.go's writer format),
    decoded not inferred -- REQ-F-019 requires the canary decode the block
    rather than assume availability from the transcript file's mere
    presence."""
    start_marker = "---STDOUT---\n"
    start = transcript_text.find(start_marker)
    if start == -1:
        return None
    rest = transcript_text[start + len(start_marker):]
    end_marker = "\n---STDERR---"
    end = rest.find(end_marker)
    if end == -1:
        return None
    return rest[:end]


def fail(field):
    sys.stderr.write("FAIL: %s\n" % field)
    sys.exit(1)


mapping_path = sys.argv[1]
transcript_paths = sys.argv[2:]

with open(mapping_path) as f:
    mapping = yaml.safe_load(f)

provider = mapping.get("providers", {}).get("anthropic_claude_cli", {})
if provider.get("status") != "mapped":
    sys.stderr.write(
        "canary-usagemapping: anthropic_claude_cli is not declared 'mapped' in %s\n" % mapping_path
    )
    sys.exit(2)

slots = provider.get("slots", {})
if not slots:
    sys.stderr.write(
        "canary-usagemapping: anthropic_claude_cli has no slots in %s\n" % mapping_path
    )
    sys.exit(2)

for transcript_path in transcript_paths:
    with open(transcript_path) as f:
        content = f.read()

    stdout_block = extract_stdout_block(content)
    if stdout_block is None:
        sys.stderr.write(
            "canary-usagemapping: transcript has no ---STDOUT---/---STDERR--- block: %s\n" % transcript_path
        )
        sys.exit(2)

    # Drift class (b), REQ-F-019/ADR-F06-11: decode the block BEFORE
    # resolving any individual slot, so a non-envelope source reports ONE
    # whole-source failure -- never nine independent per-slot failures.
    try:
        envelope = json.loads(stdout_block)
        if not isinstance(envelope, dict):
            raise ValueError("decoded JSON is not an object")
    except (ValueError, TypeError) as exc:
        sys.stderr.write(
            "canary-usagemapping: envelope_source_unavailable: %s: %s\n" % (transcript_path, exc)
        )
        fail("envelope_source_unavailable transcript=%s" % transcript_path)

    # Drift class (a), REQ-F-009: every mapped slot must resolve against
    # the decoded envelope. Sorted iteration keeps the reported failure
    # deterministic across runs.
    for slot in sorted(slots.keys()):
        envelope_path = slots[slot].get("envelope_path", "")
        _, found = resolve_slot(envelope, envelope_path)
        if not found:
            sys.stderr.write(
                "canary-usagemapping: usage_slot_unavailable: slot=%s envelope_path=%s transcript=%s\n"
                % (slot, envelope_path, transcript_path)
            )
            fail("usage_slot_unavailable slot=%s envelope_path=%s" % (slot, envelope_path))

    sys.stderr.write(
        "canary-usagemapping: all %d anthropic_claude_cli slots resolved against %s\n"
        % (len(slots), transcript_path)
    )

sys.stderr.write("PASS\n")
PYEOF
