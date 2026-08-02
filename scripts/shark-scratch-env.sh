#!/usr/bin/env bash
# Bootstrap an isolated Shark project for reproduction/testing.
#
# Agents and developers who need to run mutating shark commands (admin init,
# cloud init, task create/update, status transitions, etc.) to reproduce a bug
# or verify behavior MUST do it here, never against this repo's live checkout.
# This repo's .sharkconfig.json points at a real Turso cloud database via
# env-var placeholders; running `shark admin init --force` or `shark cloud
# init` in the repo root overwrites that pointer with a local-SQLite default,
# silently cutting off access to the real project data (see B048 incident).
#
# Usage:
#   scripts/shark-scratch-env.sh [name]
#   cd "$(scripts/shark-scratch-env.sh my-repro)"
#   ./bin/shark task create ...   # <- run from inside the printed directory
#
# The scratch directory is created under $TMPDIR (or /tmp) — always outside
# this repo tree. Shark's project-root auto-detection walks UP from the
# current directory looking for .sharkconfig.json / shark-tasks.db / .git; a
# directory under the repo tree would resolve back to the repo root before
# the scratch config even exists. /tmp has none of those markers, so `shark
# admin init` there is guaranteed to bootstrap a fresh, isolated project.
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
binary="$repo_root/bin/shark"

if [[ ! -x "$binary" ]]; then
  echo "Building shark binary (bin/shark not found)..." >&2
  make -C "$repo_root" shark >&2
fi

name="${1:-scratch}"
scratch_dir="$(mktemp -d "${TMPDIR:-/tmp}/shark-scratch-${name}-XXXXXX")"

(
  cd "$scratch_dir"
  "$binary" admin init --non-interactive >&2
)

cp "$binary" "$scratch_dir/shark"

echo "$scratch_dir"
