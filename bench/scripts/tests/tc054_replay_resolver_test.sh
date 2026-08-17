#!/usr/bin/env bash
# TC-054 (test-plan.md AC test matrix rows AC-004, AC-005; Caller-Path
# Contracts row tc054; T-E40-F07-005 task spec Acceptance Criteria
# AC-T1/AC-T2/AC-T3).
#
# Proves REQ-F-006/REQ-F-007's three named outcomes and their
# reproducibility over real bundle fixtures, driving the real
# bench/scripts/replay-answer.sh entrypoint directly -- it IS the resolver,
# so there is no caller above it to substitute (test-plan.md tc054 row):
#   (i)   a matching --stage/--kind/--topic call supplies the lowest
#         unconsumed ordinal's response and appends exactly one consumption
#         record (AC-004 case i).
#   (ii)  the same stage called again after its entries are exhausted fails
#         unresolved_gate, naming stage/kind/topic (AC-004 case ii).
#   (iii) a --topic that disagrees with the entry at the current ordinal
#         fails replay_desync, naming both the expected and supplied topic
#         (AC-004 case iii).
#   (iv)  a --topic that is a genuine one-character near-miss of an
#         unconsumed entry's real topic_key never supplies a response --
#         proving no nearest/fuzzy-match path exists (AC-004 case iv).
#   (v)   a {path, digest} response resolves relative to the bundle
#         directory and is supplied byte-for-byte (Notes for Agent:
#         resolve_within-style containment, matching F06's conventions).
#   (vi)  two consecutive resolver-driven passes over the SAME committed
#         bundle (T-E40-F07-003's reference-bundle.json, whose D02 entries
#         already carry the same-stage/distinct-ordinal topic ambiguity
#         this reproducibility case must exercise) and the same recorded
#         call sequence produce byte-identical supplied response sequences
#         and byte-identical consumption ledgers (same entry ids, same
#         order, same digests) (AC-005).
#
# Caller-Path Contract (test-plan.md tc054 row): real bundle-file reads,
# real ordinal/topic matching logic performed by replay-answer.sh itself,
# real consumption-ledger appends. This test never hand-computes "the
# lowest-unconsumed-ordinal response should be X" independently of the
# script and merely diffs against it -- every expected response/digest
# value below is read from the real fixture file at test run time, and
# every verdict is the script's own exit code and stderr text.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SCRIPTS_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
REPO_ROOT="$(cd "$SCRIPTS_DIR/../.." && pwd)"

RESOLVER="$SCRIPTS_DIR/replay-answer.sh"
FIXTURE_BUNDLE="$SCRIPTS_DIR/testdata/replay/resolver/bundle.json"
FIXTURE_RESPONSES_DIR="$SCRIPTS_DIR/testdata/replay/resolver/responses"
REFERENCE_BUNDLE="$REPO_ROOT/bench/scenarios/packages/py-feature-recurring-tasks/evaluator/replay/reference-bundle.json"

fail() {
	echo "TC-054 FAIL: $1" >&2
	exit 1
}

[[ -x "$RESOLVER" ]] || fail "replay-answer.sh missing or not executable: $RESOLVER"
[[ -f "$FIXTURE_BUNDLE" ]] || fail "resolver fixture bundle missing: $FIXTURE_BUNDLE"
[[ -f "$REFERENCE_BUNDLE" ]] || fail "T-E40-F07-003 reference bundle missing: $REFERENCE_BUNDLE"
command -v python3 >/dev/null 2>&1 || fail "python3 not found on PATH"

WORKDIR="$(mktemp -d)"
cleanup() { rm -rf "$WORKDIR"; }
trap cleanup EXIT

# field_from_bundle <bundle> <stage> <topic> <field> -- reads the REAL,
# committed fixture at test time so expectations track the file rather than
# a value hand-copied here a second time.
field_from_bundle() {
	python3 - "$1" "$2" "$3" "$4" <<'PYEOF'
import json
import sys

bundle_path, stage, topic, field = sys.argv[1:5]
with open(bundle_path) as f:
    doc = json.load(f)
for entry in doc["entries"]:
    if entry["stage"] == stage and entry["topic_key"] == topic:
        value = entry[field]
        sys.stdout.write(value if isinstance(value, str) else json.dumps(value))
        sys.exit(0)
sys.stderr.write(f"no entry found for stage={stage} topic={topic}\n")
sys.exit(1)
PYEOF
}

# fresh_bundle_copy <dest_dir> -- copies the resolver fixture bundle AND its
# responses/ directory (for the {path, digest} case) into a fresh directory
# with no consumption ledger side-file, so each case below starts from an
# empty ledger independently of any other case in this test.
fresh_bundle_copy() {
	local dest="$1"
	mkdir -p "$dest"
	cp "$FIXTURE_BUNDLE" "$dest/bundle.json"
	cp -r "$FIXTURE_RESPONSES_DIR" "$dest/responses"
}

# ledger_entry_ids <ledger_jsonl> -- newline-separated entry_id sequence, in
# append order, for assertion.
ledger_entry_ids() {
	if [[ -f "$1" ]]; then
		python3 -c "
import json, sys
for line in open(sys.argv[1]):
    line = line.strip()
    if line:
        print(json.loads(line)['entry_id'])
" "$1"
	fi
}

# ---------------------------------------------------------------------------
# Case (i) + (ii): a matching call supplies the lowest unconsumed ordinal
# and records exactly one consumption; after the stage is exhausted the
# same call fails unresolved_gate, naming stage/kind/topic.
# ---------------------------------------------------------------------------
echo "TC-054: case (i)+(ii) - matching supply, then unresolved_gate after exhaustion"

CASE_I_DIR="$WORKDIR/case-i"
fresh_bundle_copy "$CASE_I_DIR"
BUNDLE_I="$CASE_I_DIR/bundle.json"
LEDGER_I="$BUNDLE_I.consumption.jsonl"

EXPECTED_D01_1=$(field_from_bundle "$BUNDLE_I" D01 problem_statement response)
out1="$WORKDIR/i-call1.out"
err1="$WORKDIR/i-call1.err"
set +e
"$RESOLVER" --bundle "$BUNDLE_I" --stage D01 --kind human_question --topic problem_statement >"$out1" 2>"$err1"
code1=$?
set -e
[[ "$code1" -eq 0 ]] || fail "case (i): first D01 call exited $code1, want 0: $(cat "$err1")"
[[ "$(cat "$out1")" == "$EXPECTED_D01_1" ]] || fail "case (i): supplied response does not match the fixture's D01-01 response"
[[ -f "$LEDGER_I" ]] || fail "case (i): no consumption ledger written after a successful supply"
[[ "$(wc -l <"$LEDGER_I")" -eq 1 ]] || fail "case (i): ledger has $(wc -l <"$LEDGER_I") records after one successful supply, want exactly 1"
[[ "$(ledger_entry_ids "$LEDGER_I")" == "D01-01" ]] || fail "case (i): ledger's first record is not D01-01: $(ledger_entry_ids "$LEDGER_I")"

echo "TC-054(case i: matching call supplies the lowest unconsumed ordinal, exactly one consumption record) PASS"

# Consume the stage's second (and last) entry to exhaust D01.
"$RESOLVER" --bundle "$BUNDLE_I" --stage D01 --kind human_question --topic target_users >/dev/null 2>"$WORKDIR/i-call2.err" \
	|| fail "case (i): second D01 call (target_users) failed to consume the stage's remaining entry: $(cat "$WORKDIR/i-call2.err")"
[[ "$(wc -l <"$LEDGER_I")" -eq 2 ]] || fail "case (i): ledger has $(wc -l <"$LEDGER_I") records after exhausting D01, want exactly 2"

err3="$WORKDIR/i-call3.err"
out3="$WORKDIR/i-call3.out"
set +e
"$RESOLVER" --bundle "$BUNDLE_I" --stage D01 --kind human_question --topic problem_statement >"$out3" 2>"$err3"
code3=$?
set -e
[[ "$code3" -eq 1 ]] || fail "case (ii): call against an exhausted stage exited $code3, want 1: $(cat "$err3")"
[[ ! -s "$out3" ]] || fail "case (ii): call against an exhausted stage printed a response on stdout, want empty: $(cat "$out3")"
grep -q "unresolved_gate" "$err3" || fail "case (ii): failure message does not name unresolved_gate: $(cat "$err3")"
grep -q "stage=D01" "$err3" || fail "case (ii): failure message does not name the stage: $(cat "$err3")"
grep -q "kind=human_question" "$err3" || fail "case (ii): failure message does not name the kind: $(cat "$err3")"
grep -q "topic=problem_statement" "$err3" || fail "case (ii): failure message does not name the topic: $(cat "$err3")"
[[ "$(wc -l <"$LEDGER_I")" -eq 2 ]] || fail "case (ii): unresolved_gate call appended a consumption record, ledger now has $(wc -l <"$LEDGER_I") records, want still 2"

echo "TC-054(case ii: exhausted stage -> unresolved_gate naming stage/kind/topic, no answer invented, no phantom consumption) PASS"

# ---------------------------------------------------------------------------
# Case (iii): a --topic disagreeing with the entry at the current ordinal
# fails replay_desync, naming both expected and supplied topic.
# ---------------------------------------------------------------------------
echo "TC-054: case (iii) - disagreeing --topic fails replay_desync naming both topics"

CASE_III_DIR="$WORKDIR/case-iii"
fresh_bundle_copy "$CASE_III_DIR"
BUNDLE_III="$CASE_III_DIR/bundle.json"
LEDGER_III="$BUNDLE_III.consumption.jsonl"
EXPECTED_D03_TOPIC=$(field_from_bundle "$BUNDLE_III" D03 competitor_landscape topic_key)

err="$WORKDIR/iii.err"
out="$WORKDIR/iii.out"
set +e
"$RESOLVER" --bundle "$BUNDLE_III" --stage D03 --kind research_query --topic wrong_topic >"$out" 2>"$err"
code=$?
set -e
[[ "$code" -eq 1 ]] || fail "case (iii): disagreeing-topic call exited $code, want 1: $(cat "$err")"
[[ ! -s "$out" ]] || fail "case (iii): disagreeing-topic call printed a response on stdout, want empty"
grep -q "replay_desync" "$err" || fail "case (iii): failure message does not name replay_desync: $(cat "$err")"
grep -q "expected kind='research_query' topic='$EXPECTED_D03_TOPIC'" "$err" || fail "case (iii): failure message does not name the expected topic: $(cat "$err")"
grep -q "supplied kind='research_query' topic='wrong_topic'" "$err" || fail "case (iii): failure message does not name the supplied topic: $(cat "$err")"
[[ ! -f "$LEDGER_III" ]] || fail "case (iii): replay_desync appended a consumption record, ledger should not exist"

echo "TC-054(case iii: replay_desync names both expected and supplied topic, no consumption recorded) PASS"

# ---------------------------------------------------------------------------
# Case (iv): a --topic that is a genuine one-character near-miss of an
# unconsumed entry's real topic never supplies a response -- no nearest or
# fuzzy match exists.
# ---------------------------------------------------------------------------
echo "TC-054: case (iv) - one-character near-miss --topic never supplies a response (no fuzzy match)"

CASE_IV_DIR="$WORKDIR/case-iv"
fresh_bundle_copy "$CASE_IV_DIR"
BUNDLE_IV="$CASE_IV_DIR/bundle.json"
LEDGER_IV="$BUNDLE_IV.consumption.jsonl"
REAL_D03_TOPIC=$(field_from_bundle "$BUNDLE_IV" D03 competitor_landscape topic_key)
NEAR_MISS_TOPIC="${REAL_D03_TOPIC%?}s" # differs by exactly the last character
[[ "$NEAR_MISS_TOPIC" != "$REAL_D03_TOPIC" ]] || fail "case (iv): test setup bug -- near-miss topic equals the real topic"
[[ "${#NEAR_MISS_TOPIC}" -eq "${#REAL_D03_TOPIC}" ]] || fail "case (iv): test setup bug -- near-miss topic is not the same length as the real topic"

err="$WORKDIR/iv.err"
out="$WORKDIR/iv.out"
set +e
"$RESOLVER" --bundle "$BUNDLE_IV" --stage D03 --kind research_query --topic "$NEAR_MISS_TOPIC" >"$out" 2>"$err"
code=$?
set -e
[[ "$code" -ne 0 ]] || fail "case (iv): near-miss topic call exited 0 (supplied a response), want non-zero: fuzzy-match fabrication path"
[[ ! -s "$out" ]] || fail "case (iv): near-miss topic call printed a response on stdout, want empty: $(cat "$out")"
(grep -q "replay_desync" "$err" || grep -q "unresolved_gate" "$err") || fail "case (iv): failure message names neither replay_desync nor unresolved_gate: $(cat "$err")"
[[ ! -f "$LEDGER_IV" ]] || fail "case (iv): near-miss topic call appended a consumption record, ledger should not exist"

echo "TC-054(case iv: one-character near-miss topic -> no supply, proving no nearest/fuzzy-match path) PASS"

# ---------------------------------------------------------------------------
# Case (v): a {path, digest} response resolves relative to the bundle
# directory (resolve_within-style containment) and is supplied byte-for-
# byte, matching F06's bench-script conventions.
# ---------------------------------------------------------------------------
echo "TC-054: case (v) - {path, digest} response resolves relative to the bundle directory"

CASE_V_DIR="$WORKDIR/case-v"
fresh_bundle_copy "$CASE_V_DIR"
BUNDLE_V="$CASE_V_DIR/bundle.json"
EXPECTED_D05_FILE="$CASE_V_DIR/responses/d05-note.txt"
[[ -f "$EXPECTED_D05_FILE" ]] || fail "case (v): test setup bug -- expected response file missing: $EXPECTED_D05_FILE"

out="$WORKDIR/v.out"
err="$WORKDIR/v.err"
set +e
"$RESOLVER" --bundle "$BUNDLE_V" --stage D05 --kind human_question --topic interview_mode_choice >"$out" 2>"$err"
code=$?
set -e
[[ "$code" -eq 0 ]] || fail "case (v): {path, digest} response call exited $code, want 0: $(cat "$err")"
cmp -s "$out" "$EXPECTED_D05_FILE" || fail "case (v): supplied bytes do not match the referenced response file byte-for-byte"

echo "TC-054(case v: {path, digest} response resolved relative to bundle directory, supplied byte-for-byte) PASS"

# ---------------------------------------------------------------------------
# Case (vi) / AC-005: two consecutive resolver-driven passes over the SAME
# committed bundle (T-E40-F07-003's reference-bundle.json) and the same
# recorded call sequence produce byte-identical supplied response
# sequences and byte-identical consumption ledgers. The call sequence below
# exercises D02's real same-stage/distinct-ordinal topic ambiguity
# (ordinal 1 and ordinal 3 both carry topic_key "success_metrics"; ordinal
# 2 carries "constraints") -- a resolver that matched by
# first-topic-in-file-order rather than lowest-unconsumed-ordinal would be
# exercised by this exact call sequence, not merely assumed correct on a
# bundle with no such ambiguity.
# ---------------------------------------------------------------------------
echo "TC-054: case (vi)/AC-005 - two-pass reproducibility over the T-E40-F07-003 reference bundle's D02 ambiguity"

D02_ORD1_TOPIC=$(field_from_bundle "$REFERENCE_BUNDLE" D02 success_metrics topic_key)
[[ "$D02_ORD1_TOPIC" == "success_metrics" ]] || fail "case (vi): test setup bug -- could not resolve D02 success_metrics entry from reference bundle"

run_call_sequence() {
	# Args: <pass_dir> -- runs the fixed 3-call sequence against a fresh
	# copy of the reference bundle, writing stdout-per-call and the
	# resulting ledger under <pass_dir>.
	local pass_dir="$1"
	mkdir -p "$pass_dir"
	cp "$REFERENCE_BUNDLE" "$pass_dir/bundle.json"
	"$RESOLVER" --bundle "$pass_dir/bundle.json" --stage D02 --kind human_question --topic success_metrics >"$pass_dir/call1.out" 2>"$pass_dir/call1.err" \
		|| fail "case (vi) $pass_dir: call1 (success_metrics) failed: $(cat "$pass_dir/call1.err")"
	"$RESOLVER" --bundle "$pass_dir/bundle.json" --stage D02 --kind human_question --topic constraints >"$pass_dir/call2.out" 2>"$pass_dir/call2.err" \
		|| fail "case (vi) $pass_dir: call2 (constraints) failed: $(cat "$pass_dir/call2.err")"
	"$RESOLVER" --bundle "$pass_dir/bundle.json" --stage D02 --kind human_question --topic success_metrics >"$pass_dir/call3.out" 2>"$pass_dir/call3.err" \
		|| fail "case (vi) $pass_dir: call3 (success_metrics again) failed: $(cat "$pass_dir/call3.err")"
}

PASS1_DIR="$WORKDIR/ac005-pass1"
PASS2_DIR="$WORKDIR/ac005-pass2"
run_call_sequence "$PASS1_DIR"
run_call_sequence "$PASS2_DIR"

for n in 1 2 3; do
	cmp -s "$PASS1_DIR/call$n.out" "$PASS2_DIR/call$n.out" \
		|| fail "case (vi): call$n's supplied response differs between pass 1 and pass 2"
done
echo "TC-054(case vi part 1: three supplied responses are byte-identical across both passes) PASS"

LEDGER1="$PASS1_DIR/bundle.json.consumption.jsonl"
LEDGER2="$PASS2_DIR/bundle.json.consumption.jsonl"
[[ -f "$LEDGER1" && -f "$LEDGER2" ]] || fail "case (vi): one or both passes produced no consumption ledger"

# ledger_projection <ledger_jsonl> -- {entry_id, entry_digest, stage,
# ordinal, request_kind, topic_key, response_digest} per record, in append
# order, EXCLUDING supplied_at (wall-clock, expected to differ between two
# independently timed passes -- AC-005's own text scopes "byte-identical"
# to "same entry ids, same order, same digests", not to the ledger's raw
# file bytes).
ledger_projection() {
	python3 -c "
import json, sys
for line in open(sys.argv[1]):
    line = line.strip()
    if not line:
        continue
    r = json.loads(line)
    r.pop('supplied_at', None)
    print(json.dumps(r, sort_keys=True))
" "$1"
}

PROJ1="$WORKDIR/ledger1.projection"
PROJ2="$WORKDIR/ledger2.projection"
ledger_projection "$LEDGER1" >"$PROJ1"
ledger_projection "$LEDGER2" >"$PROJ2"
[[ "$(wc -l <"$PROJ1")" -eq 3 ]] || fail "case (vi): pass 1 ledger has $(wc -l <"$PROJ1") records, want 3"
cmp -s "$PROJ1" "$PROJ2" || fail "case (vi): consumption ledgers diverge between pass 1 and pass 2 (entry ids/order/digests): $(diff "$PROJ1" "$PROJ2" || true)"

# The ambiguity itself: prove ordinal-primacy was actually exercised, not
# merely assumed absent -- the FIRST success_metrics call must consume
# ordinal 1 (D02-01), and the THIRD (repeat) call must consume ordinal 3
# (D02-03), never a first-topic-in-file-order match.
mapfile -t IDS1 < <(ledger_entry_ids "$LEDGER1")
[[ "${IDS1[0]}" == "D02-01" ]] || fail "case (vi): first success_metrics call consumed ${IDS1[0]}, want D02-01 (lowest unconsumed ordinal, not file order)"
[[ "${IDS1[1]}" == "D02-02" ]] || fail "case (vi): constraints call consumed ${IDS1[1]}, want D02-02"
[[ "${IDS1[2]}" == "D02-03" ]] || fail "case (vi): repeat success_metrics call consumed ${IDS1[2]}, want D02-03 (the second, higher-ordinal same-topic entry)"

echo "TC-054(case vi part 2: consumption ledgers byte-identical on entry ids/order/digests, D02's real ordinal ambiguity exercised, not assumed absent) PASS"

echo "TC-054 PASS"
