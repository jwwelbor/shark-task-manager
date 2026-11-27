#!/usr/bin/env bash
# Resolve and validate the installation that supplied the active SKILL.md.
set -euo pipefail

if [[ -z "${DEEP_REVIEW_SKILL_DIR:-}" ]]; then
  echo "Set DEEP_REVIEW_SKILL_DIR to the absolute directory containing the loaded deep-review SKILL.md." >&2
  exit 2
fi
if [[ "$DEEP_REVIEW_SKILL_DIR" != /* ]]; then
  echo "DEEP_REVIEW_SKILL_DIR must be an absolute path: $DEEP_REVIEW_SKILL_DIR" >&2
  exit 2
fi
if ! skill_dir=$(cd -- "$DEEP_REVIEW_SKILL_DIR" && pwd -P); then
  echo "Loaded deep-review skill directory is unavailable: $DEEP_REVIEW_SKILL_DIR" >&2
  exit 2
fi

required=(
  SKILL.md
  scripts/get_diff.sh
  scripts/prepare_candidate.py
  scripts/adaptive_review.py
  scripts/review_workflow.js
  references/angle-a-bugs.md
  references/angle-b-behavior.md
  references/angle-c-sibling.md
  references/angle-d-cleanup.md
  references/angle-e-tests.md
  references/angle-f-standards.md
  references/consolidator.md
)
for path in "${required[@]}"; do
  if [[ ! -f "$skill_dir/$path" ]]; then
    echo "Loaded deep-review skill is missing required file: $skill_dir/$path" >&2
    exit 2
  fi
done
printf '%s\n' "$skill_dir"
