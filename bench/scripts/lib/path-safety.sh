#!/usr/bin/env bash
# lib/path-safety.sh -- shared containment guard, sourced by both
# run-batch.sh and replay-manifest.sh.
#
# NEW-1 (UAT round 3, uat-20260809-013000-E40-F03.md): run-batch.sh's own
# R2-F-13 fix (assert_within_out_root) canonicalized BOTH sides of its
# containment comparison with `realpath -m`, but replay-manifest.sh
# re-implemented the same check independently and compared a fully
# canonicalized path against a merely LOGICAL one (`cd "$out_root" && pwd`,
# which preserves symlinks) -- a legitimate, fully-inside --out that
# resolves through a symlink was wrongly rejected. Lifting the ONE
# implementation here means the two call sites structurally cannot drift
# apart again.
#
# Caller contract: set `out_root_canon="$(realpath -m -- "$out_root")"`
# before calling assert_within_out_root -- `realpath -m` never requires
# the path to exist, so this works even before --out itself has been
# created (a --dry-run run-batch.sh invocation never creates it).
assert_within_out_root() {
	local path="$1" canon
	canon="$(realpath -m -- "$path")"
	case "$canon" in
	"$out_root_canon" | "$out_root_canon"/*) return 0 ;;
	esac
	echo "$(basename "$0"): refusing filesystem operation outside --out: '$path' resolves to '$canon' (out root: '$out_root_canon')" >&2
	return 1
}
