#!/usr/bin/env bash
# tests/lib/path-shim-denial.sh -- shared PATH-shim provider/network-denial
# harness (T-E40-F10-004 scope), sourced by TC-079, TC-080, and (later)
# TC-090's self-tests. spec.md REQ-NF-002 requires the zero-provider-call /
# zero-network-call property of preview and other offline commands be
# "provable under a provider-denial and network-denial harness rather than
# asserted by inspection" -- this file is that harness, built once so the
# three test cases that need it cannot drift into three private copies
# (REQ-F-018's "schema-owned vocabulary, not a private copy" discipline
# extended to test infrastructure).
#
# Binary list: read LIVE from bench/reports/lifecycle-baseline-schema.yaml's
# `provider_and_network_binaries:` array (REQ-F-018) -- never a hardcoded
# literal list in this file. The schema's own header comment documents that
# today's list (claude, codex) is deliberately narrower than the full
# provider/network surface named in that comment (curl, wget, ssh, scp,
# turso, nc, rsync); if a future change adds a network tool to the call
# chain, the schema is amended first (per its own header) and this harness
# picks the new entry up automatically -- no code change here.
#
# Caller contract:
#   source ".../tests/lib/path-shim-denial.sh"
#   path_shim_denial_setup "$WORKDIR"
#   export PATH="$PATH_SHIM_DENIAL_BIN_DIR:$PATH"
#   ... invoke the command under test ...
#   path_shim_denial_assert_empty "label"

_path_shim_denial_lib_dir() {
	cd "$(dirname "${BASH_SOURCE[0]}")" && pwd
}

PATH_SHIM_DENIAL_LIB_DIR="$(_path_shim_denial_lib_dir)"
PATH_SHIM_DENIAL_BENCH_DIR="$(cd "$PATH_SHIM_DENIAL_LIB_DIR/../../.." && pwd)"
PATH_SHIM_DENIAL_SCHEMA="$PATH_SHIM_DENIAL_BENCH_DIR/reports/lifecycle-baseline-schema.yaml"

# path_shim_denial_setup <workdir>
# Creates <workdir>/denybin/<binary> for every entry in the schema's
# provider_and_network_binaries list. Each shim, when invoked, appends one
# line (binary name + argv) to <workdir>/denybin.log and exits 1 -- it never
# succeeds, so a caller that accidentally depends on a shimmed binary's
# actual output fails loudly rather than silently proceeding.
#
# Sets PATH_SHIM_DENIAL_BIN_DIR (the directory to prepend onto PATH) and
# PATH_SHIM_DENIAL_LOG (the invocation-attempt log, truncated fresh here).
path_shim_denial_setup() {
	local workdir="$1"
	[[ -n "$workdir" ]] || {
		echo "path-shim-denial: path_shim_denial_setup requires a workdir argument" >&2
		return 1
	}
	[[ -f "$PATH_SHIM_DENIAL_SCHEMA" ]] || {
		echo "path-shim-denial: schema not found: $PATH_SHIM_DENIAL_SCHEMA" >&2
		return 1
	}
	command -v python3 >/dev/null 2>&1 || {
		echo "path-shim-denial: python3 not found on PATH" >&2
		return 1
	}

	PATH_SHIM_DENIAL_BIN_DIR="$workdir/denybin"
	PATH_SHIM_DENIAL_LOG="$workdir/denybin.log"
	mkdir -p "$PATH_SHIM_DENIAL_BIN_DIR"
	: >"$PATH_SHIM_DENIAL_LOG"

	local binaries_tmp
	binaries_tmp="$(mktemp)"
	python3 - "$PATH_SHIM_DENIAL_SCHEMA" >"$binaries_tmp" <<'PYEOF'
import sys

import yaml

with open(sys.argv[1]) as f:
    data = yaml.safe_load(f)

for name in data.get("provider_and_network_binaries") or []:
    print(name)
PYEOF
	local rc=$?
	if [[ "$rc" -ne 0 ]]; then
		rm -f "$binaries_tmp"
		echo "path-shim-denial: failed to read provider_and_network_binaries from $PATH_SHIM_DENIAL_SCHEMA" >&2
		return 1
	fi

	local bin_count=0
	local name
	while IFS= read -r name; do
		[[ -n "$name" ]] || continue
		bin_count=$((bin_count + 1))
		cat >"$PATH_SHIM_DENIAL_BIN_DIR/$name" <<SHIM
#!/usr/bin/env bash
printf '%s %s\n' "$name" "\$*" >>"$PATH_SHIM_DENIAL_LOG"
exit 1
SHIM
		chmod +x "$PATH_SHIM_DENIAL_BIN_DIR/$name"
	done <"$binaries_tmp"
	rm -f "$binaries_tmp"

	if [[ "$bin_count" -eq 0 ]]; then
		echo "path-shim-denial: schema's provider_and_network_binaries list is empty; nothing to deny" >&2
		return 1
	fi
	return 0
}

# path_shim_denial_assert_empty <label>
# Fails (non-zero return, message to stderr) when the invocation-attempt log
# is non-empty. Callers are expected to `|| fail "..."` this in their own
# idiom, matching the rest of bench/scripts/tests/*.sh.
path_shim_denial_assert_empty() {
	local label="$1"
	if [[ -s "$PATH_SHIM_DENIAL_LOG" ]]; then
		echo "path-shim-denial: $label: unexpected provider/network invocation(s) recorded:" >&2
		cat "$PATH_SHIM_DENIAL_LOG" >&2
		return 1
	fi
	return 0
}
