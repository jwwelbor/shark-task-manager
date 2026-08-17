#!/usr/bin/env bash
# TC-064 / T-E40-F08-006: contract mode is provider-free and repeatable.
set -euo pipefail
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SCRIPTS_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
RUNNER="$SCRIPTS_DIR/run-lifecycle.sh"
fail() { echo "TC-064 FAIL: $1" >&2; exit 1; }
WORKDIR="$(mktemp -d)"
trap 'rm -rf "$WORKDIR"' EXIT
mkdir -p "$WORKDIR/bin" "$WORKDIR/scratch"
cat >"$WORKDIR/bin/shark" <<'SHARK'
#!/usr/bin/env bash
set -euo pipefail
[[ "${1:-}" == next && "${2:-}" == ROOT-064 ]] || { echo "unexpected shark argv: $*" >&2; exit 1; }
printf '%s\n' '{"action":"unresolved_gate","error":"fixture contract stop"}'
SHARK
chmod +x "$WORKDIR/bin/shark"
SCENARIO="$SCRIPTS_DIR/../scenarios/packages/py-bug-due-date-boundary/package.yaml"
for n in one two; do
  PATH="$WORKDIR/bin:$PATH" "$RUNNER" --mode contract --scenario "$SCENARIO" --run-id tc064 --root ROOT-064 --scratch-root "$WORKDIR/scratch" --output "$WORKDIR/$n.jsonl" >/dev/null
done
cmp "$WORKDIR/one.jsonl" "$WORKDIR/two.jsonl" || fail "repeated contract verdicts differ"
grep -q '"publication_eligible":false' "$WORKDIR/one.jsonl" || fail "contract stop was publishable"
echo "TC-064: pass (contract mode has no provider adapter and repeated verdicts are byte-identical)"
