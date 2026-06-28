#!/usr/bin/env bash
# Run from within the project git repo root.
# Outputs JSON: { "diff_path": "...", "changed_files": [...], "project_root": "..." }
set -euo pipefail

project_root=$(git rev-parse --show-toplevel)

# Find base branch
base_branch=""
for candidate in main master origin/main origin/master; do
  if git rev-parse --verify "$candidate" &>/dev/null 2>&1; then
    base_branch="$candidate"
    break
  fi
done

if [ -z "$base_branch" ]; then
  echo '{"error": "Could not find base branch (tried main, master, origin/main, origin/master)"}' >&2
  exit 1
fi

diff_path=$(mktemp /tmp/code-review-diff-XXXXXX.txt)
git diff "${base_branch}...HEAD" > "$diff_path"

if [ ! -s "$diff_path" ]; then
  echo '{"error": "Diff is empty — nothing to review. Is this branch ahead of '"$base_branch"'?"}' >&2
  exit 1
fi

changed_files_raw=$(git diff --name-only "${base_branch}...HEAD" | head -60)

# Find coding standards doc — check common locations, first match wins
coding_standards_path=""
for candidate in \
  "docs/architecture/coding-standards.md" \
  "docs/coding-standards.md" \
  "CODING_STANDARDS.md" \
  "coding-standards.md" \
  ".claude/coding-standards.md"
do
  if [ -f "${project_root}/${candidate}" ]; then
    coding_standards_path="${project_root}/${candidate}"
    break
  fi
done

# Fallback: broader search (capped to avoid slow scans on huge repos)
if [ -z "$coding_standards_path" ]; then
  coding_standards_path=$(find "$project_root" -maxdepth 4 \
    \( -name "coding-standards.md" -o -name "CODING_STANDARDS.md" -o -name "style-guide.md" \) \
    -not -path "*/node_modules/*" -not -path "*/.git/*" \
    2>/dev/null | head -1)
fi

# ─── Compute persist location for the overall (multi-angle) review report ─────
# Mirrors the existing convention: docs/review/<epic>/<feature>/code_review/.
# Match the branch against existing review dirs (feature dir at depth 2, epic
# dir at depth 1); fall back to a branch-named dir if nothing matches.
branch=$(git rev-parse --abbrev-ref HEAD 2>/dev/null || echo "HEAD")
branch_slug=$(printf '%s' "$branch" | tr '/' '-')
review_root="${project_root}/docs/review"
review_date=$(date +%Y-%m-%d)

review_match=""
if [ -d "$review_root" ]; then
  review_match=$(find "$review_root" -maxdepth 2 -type d -name "$branch_slug" 2>/dev/null | head -1)
  if [ -z "$review_match" ]; then
    # Remove the ^-anchor so GitFlow-prefixed branches (feature/E07-F01-x → feature-E07-F01-x) also match
    feature_key=$(printf '%s' "$branch_slug" | grep -oE 'E[0-9]+-F[0-9]+' | head -1)
    if [ -n "$feature_key" ]; then
      review_match=$(find "$review_root" -maxdepth 2 -type d -name "${feature_key}-*" 2>/dev/null | head -1)
    fi
  fi
fi

if [ -n "$review_match" ]; then
  review_dir="${review_match}/code_review"
  review_label="$(basename "$review_match")"
else
  review_dir="${review_root}/${branch_slug}/code_review"
  review_label="$branch_slug"
fi
review_output_path="${review_dir}/${review_date}-${review_label}-overall-review.md"

DIFF_PATH="$diff_path" \
PROJECT_ROOT="$project_root" \
CODING_STANDARDS_PATH="$coding_standards_path" \
CHANGED_FILES="$changed_files_raw" \
BRANCH="$branch" \
REVIEW_OUTPUT_PATH="$review_output_path" \
python3 - <<'PYEOF'
import os, json
diff_path = os.environ["DIFF_PATH"]
project_root = os.environ["PROJECT_ROOT"]
coding_standards_path = os.environ.get("CODING_STANDARDS_PATH", "")
changed_files = os.environ.get("CHANGED_FILES", "").strip().split('\n')
changed_files = [f for f in changed_files if f]
print(json.dumps({
  "diff_path": diff_path,
  "changed_files": changed_files,
  "project_root": project_root,
  "coding_standards_path": coding_standards_path or None,
  "branch": os.environ.get("BRANCH") or None,
  "review_output_path": os.environ.get("REVIEW_OUTPUT_PATH") or None,
}))
PYEOF
