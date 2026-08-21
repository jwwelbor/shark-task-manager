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

# symlink_dest <path> -- true (exit 0) iff <path> itself exists as a
# symlink (any target: in-root, out-of-root, dangling, file, or
# directory). assert_within_out_root above only ever validates where a
# path's FINAL resolved target lands relative to --out (via `realpath -m`,
# which resolves an existing symlink's final component) -- it does not, and
# is not meant to, answer "is the directory entry AT this exact path itself
# a symlink" (a distinct question: a symlink can resolve to a location
# fully inside --out and still be a "confused deputy" redirect to another
# retained artifact, which containment-by-resolved-path alone cannot
# distinguish from a legitimate real directory/file at that name).
#
# code-review-2026-08-21T0459-E40-F10.md (round 4) finding 7 / non-blocker:
# this exact `[[ -L "$path" ]]` test used to be duplicated independently in
# run-lifecycle-batch.sh's retain_pair() and run-review-comparison.sh's
# retain_gate(), and each of this round's fixes to the "already retained,
# skip" fast paths and the comparison.json publish path would otherwise
# have added a THIRD and FOURTH independent copy. One shared predicate here
# means the two drivers structurally cannot drift apart on what "already a
# symlink" means, the same rationale that established assert_within_out_root
# itself (NEW-1 above). Callers keep ownership of their own contextual
# diagnostic message and of whether a hit is a hard refusal or a soft
# "reclassify, don't dispatch" outcome -- this helper only answers the
# yes/no filesystem question.
symlink_dest() {
	[[ -L "$1" ]]
}
