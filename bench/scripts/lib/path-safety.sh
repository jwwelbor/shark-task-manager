#!/usr/bin/env bash
# lib/path-safety.sh -- shared containment guard, sourced by both
# run-batch.sh and replay-manifest.sh.
#
# ---------------------------------------------------------------------------
# SCOPE FREEZE (de-facto ADR for bench/'s symlink policy, settled after six
# code-review rounds of narrowing instance patches -- code-review-2026-08-
# 21T1141-E40-F10.md round 6, adopted as final policy): the symlink-safety
# surface for bench/ is CLOSED at exactly two invariants:
#
#   invariant 1 -- chain-walk on retention-root WRITES/reuse-decisions:
#     assert_no_symlink_in_chain(root, leaf), used by every write and every
#     "is this already retained?" read under out_root_canon.
#   invariant 2 -- dereference-on-copy for operator-declared SOURCES:
#     copy_tree_dereferenced(src, dst), used by every driver copy of an
#     operator-supplied scratch_root (or any other source-side template
#     directory) into an ephemeral, worker-visible location. This closes the
#     source-side class BY CONSTRUCTION: every symlink encountered anywhere
#     in the copied tree, top-level source argument included, is
#     dereferenced into a real, independent copy -- there is no narrower
#     "top-level only" vs. "nested" shape left to find, which is why the
#     former `assert_source_not_symlink` hard-refusal (fail-loud on a
#     top-level symlinked source) was removed as redundant once this
#     primitive was in place at every scratch_root/source-copy call site.
#   override -- SHARK_BENCH_ALLOW_SYMLINKS=1: when set, invariant 1's hard
#     refusal (and any other still-present hard-refuse symlink guard in this
#     file) prints a warning and proceeds instead of refusing. Unset (the
#     default) stays hard-refuse -- this is "refuse by default, explicit
#     opt-out," never "warn and always proceed." Invariant 2 has no override:
#     dereferencing is already safe-by-default, so there is nothing to opt
#     out of.
#
# A finding proposing a NEW symlink guard beyond these two invariants is out
# of scope for this feature unless it demonstrates a concrete bypass of
# invariant 1 or invariant 2 above -- not merely a new named shape of the
# same already-closed class.
# ---------------------------------------------------------------------------
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
	if [[ -L "$1" && "${SHARK_BENCH_ALLOW_SYMLINKS:-}" == "1" ]]; then
		echo "$(basename "$0"): WARNING (SHARK_BENCH_ALLOW_SYMLINKS=1): '$1' is a symlink in the trusted chain -- this operation may read/write a location outside the retention layout you think you're using. Proceeding because you overrode the guard." >&2
		return 1
	fi
	[[ -L "$1" ]]
}

# assert_no_symlink_in_chain <root> <leaf> -- STRUCTURAL FIX,
# code-review-2026-08-21T1335-E40-F10.md (round 5) finding 1. Refuses
# (nonzero, loud diagnostic on stderr; never a silent skip) if ANY path
# component from <root>'s immediate child down to <leaf>, INCLUSIVE of
# <leaf> itself, is a pre-existing symlink.
#
# Why symlink_dest() above was not enough: every symlink guard in this
# codebase before this fix -- including symlink_dest() itself -- tested
# only the LEAF path's own -L status. round 5 found the identical defect
# class one directory level higher than round 4 closed it: a symlink at an
# ANCESTOR component (e.g. `scenarios/<scenario_id>` itself, one level
# above the rep directory `scenarios/<scenario_id>/<rep>`) redirects to
# another location fully INSIDE the same retention root. That redirect
# passes assert_within_out_root's containment check (the resolved path
# still lands inside out_root_canon) and is completely invisible to a
# leaf-only -L test, because the leaf-level directory ENTRY itself is
# real -- only an ancestor redirects underneath it. Patching one more named
# ancestor path per review round (round 4's fix for the leaf, round 5's own
# finding for one ancestor level) never converges; walking the WHOLE chain
# once closes the class structurally.
#
# Caller contract:
#   <root> MUST already be a canonicalized, trusted anchor -- the SAME
#   out_root_canon="$(realpath -m -- "$out_root")" every caller in this
#   codebase already establishes for assert_within_out_root. This function
#   does not re-validate root's own identity (matching that established
#   convention: the retention root itself is the trust anchor, never
#   re-authenticated on every call).
#
#   <leaf> MUST be a literal path string built as "$root/..." via plain
#   bash string concatenation (e.g.
#   "$out_root_canon/scenarios/$scenario_id/$rep") -- NEVER independently
#   realpath()'d or otherwise resolved before being passed in. Resolving it
#   first would follow -- and therefore hide -- the very symlink this
#   function exists to catch.
#
# lstat() semantics (verified empirically for this fix): checking `-L` on
# each successively-longer literal prefix, walked in order from root's
# immediate child down to leaf, is sufficient. lstat() resolves every path
# component EXCEPT the final one, so `-L "$root/a/b"` (with `a` left
# unresolved by the test itself) already answers "is the entry named `b`,
# reached by walking through whatever `a` denotes, itself a symlink" -- and
# this function has already independently rejected `a` being a symlink in
# the PRIOR loop iteration before ever reaching that check. No component's
# own redirect can silently satisfy a later component's test.
#
# This is the SINGLE shared primitive every reuse-decision ("is this
# (scenario, rep)/gate pair already retained here?") and every write
# (mkdir -p a fresh destination, quarantine-move an incomplete prior
# attempt) in run-lifecycle-batch.sh and run-review-comparison.sh now calls,
# instead of a fresh leaf-only -L check invented per review round.
assert_no_symlink_in_chain() {
	local root="$1" leaf="$2"
	case "$leaf" in
	"$root") return 0 ;;
	"$root"/*) ;;
	*)
		echo "$(basename "$0"): assert_no_symlink_in_chain: '$leaf' is not a literal descendant of trusted root '$root' -- refusing to walk it" >&2
		return 1
		;;
	esac
	local rel="${leaf#"$root"/}"
	local cur="$root" part
	local IFS=/
	local -a parts
	# shellcheck disable=SC2206 -- IFS=/ splitting on a literal relative
	# path (never operator-supplied shell metacharacters) is the intended
	# behavior here, not accidental word-splitting.
	parts=($rel)
	for part in "${parts[@]}"; do
		[[ -n "$part" ]] || continue
		cur="$cur/$part"
		if [[ -L "$cur" ]]; then
			# SHARK_BENCH_ALLOW_SYMLINKS=1 override (scope-freeze paragraph
			# above): explicit opt-out, warn-and-proceed instead of refuse.
			# Default (unset) stays hard-refuse.
			if [[ "${SHARK_BENCH_ALLOW_SYMLINKS:-}" == "1" ]]; then
				echo "$(basename "$0"): WARNING (SHARK_BENCH_ALLOW_SYMLINKS=1): '$cur' is a symlink in the trusted chain -- this operation may read/write a location outside the retention layout you think you're using. Proceeding because you overrode the guard." >&2
				continue
			fi
			echo "$(basename "$0"): refusing filesystem operation -- '$cur' is a pre-existing symlink, found while walking the trusted chain from '$root' down to '$leaf'; a symlink ANYWHERE in this chain, not just at the final path, can silently redirect this operation to a different, unrelated location" >&2
			return 1
		fi
	done
	return 0
}

# copy_tree_dereferenced <src> <dst> -- STRUCTURAL FIX, invariant 2 of the
# scope-freeze paragraph above. Originally landed as code-review-2026-08-
# 21T1141-E40-F10.md (round 6) finding 1's fix for a NESTED symlink inside
# an otherwise-real scratch_root; subsequently generalized to be the SOLE
# source-side guard (round-5 finding 2's `assert_source_not_symlink`
# hard-refusal removed as redundant once this was in place at every call
# site) once it was confirmed to also safely dereference a TOP-LEVEL
# symlinked source. Copies <src> to <dst>, dereferencing EVERY symlink
# encountered anywhere in the copied tree -- top-level source argument
# included -- so there is no narrower "top-level only" vs. "nested" shape
# of this defect class left to find.
#
# Why a hard-refusal guard was not needed here: `cp -a` preserves every
# symlink it walks during a recursive copy (top-level source argument or
# nested, `--no-dereference` either way), so a worker write through the
# believed-isolated "ephemeral" copy's corresponding path could silently
# reach whatever a preserved symlink targeted, violating both drivers' own
# "this driver never mutates the template" guarantee.
# shutil.copytree(..., symlinks=False) closes this BY CONSTRUCTION instead
# of by refusal: it lists the resolved directory's entries and dereferences
# every symlink it encounters (empirically verified for both the top-level
# and nested shapes), producing a genuinely independent tree in every case
# -- reusing the exact primitive already proven safe by this codebase in
# the preview-mode scratch_root copy (both drivers' own preview print
# loops), rather than inventing a third copy strategy (e.g. `cp -aL`).
#
# No SHARK_BENCH_ALLOW_SYMLINKS override here (unlike invariant 1's
# hard-refuse guards): dereferencing is already safe-by-default, so there
# is nothing to opt out of.
#
# Caller contract: <dst> MUST NOT already exist (shutil.copytree's own
# contract) -- every current caller passes a fresh path under a
# just-created `mktemp -d` directory, so this is never violated.
copy_tree_dereferenced() {
	local src="$1" dst="$2"
	python3 -c '
import shutil
import sys

src, dst = sys.argv[1], sys.argv[2]
shutil.copytree(src, dst, symlinks=False)
' "$src" "$dst"
}
