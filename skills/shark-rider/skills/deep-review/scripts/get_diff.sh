#!/usr/bin/env bash
# Run from within the project git repo root.
# Outputs JSON: { "diff_path": "...", "changed_files": [...], "changed_file_count": N, "diff_shortstat": "...", "project_root": "..." }
set -euo pipefail

project_root=$(git rev-parse --show-toplevel)

# A committed candidate makes the tree identity and the manifest unambiguous.
# Fail closed instead of silently omitting staged, unstaged or untracked files.
if [ -n "$(git status --porcelain --untracked-files=all)" ]; then
  echo '{"error": "Working tree is not clean; prepare an isolated snapshot of explicit candidate paths before review"}' >&2
  exit 1
fi

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

# REVIEW_BASE is the accepted baseline when one is specified by the plan.
# Otherwise use the actual merge base, not the moving branch tip.
base_commit=$(git rev-parse --verify "${REVIEW_BASE:-$(git merge-base "$base_branch" HEAD)}^{commit}")
candidate_commit=$(git rev-parse HEAD)
candidate_tree=$(git rev-parse 'HEAD^{tree}')
diff_path=$(mktemp /tmp/code-review-diff-XXXXXX.txt)
git diff --binary "$base_commit" "$candidate_commit" > "$diff_path"
diff_sha256=$(sha256sum "$diff_path" | cut -d' ' -f1)

if [ ! -s "$diff_path" ]; then
  echo '{"error": "Diff is empty — nothing to review. Is this branch ahead of '"$base_branch"'?"}' >&2
  exit 1
fi

diff_shortstat=$(git diff --shortstat "$base_commit" "$candidate_commit" || true)

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
    # Pull the first E##-F## feature key, if any. Bash-native regex (unanchored,
    # so GitFlow-prefixed branches like feature-E07-F01-x still match) avoids the
    # grep pipeline whose exit-1-on-no-match would abort under `set -euo pipefail`.
    if [[ "$branch_slug" =~ E[0-9]+-F[0-9]+ ]]; then
      feature_key="${BASH_REMATCH[0]}"
    else
      feature_key=""
    fi
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
DIFF_SHORTSTAT="$diff_shortstat" \
BRANCH="$branch" \
BASE_COMMIT="$base_commit" \
CANDIDATE_COMMIT="$candidate_commit" \
CANDIDATE_TREE="$candidate_tree" \
DIFF_SHA256="$diff_sha256" \
REVIEW_OUTPUT_PATH="$review_output_path" \
REVIEW_SOURCE_METADATA="${REVIEW_SOURCE_METADATA:-}" \
python3 - <<'PYEOF'
import os, json, subprocess
diff_path = os.environ["DIFF_PATH"]
project_root = os.environ["PROJECT_ROOT"]
coding_standards_path = os.environ.get("CODING_STANDARDS_PATH", "")
changed_files = [os.fsdecode(p) for p in subprocess.check_output(["git", "diff", "--name-only", "-z", os.environ["BASE_COMMIT"], os.environ["CANDIDATE_COMMIT"]]).split(b"\0") if p]
scope = {
  "diff_path": diff_path,
  "changed_files": changed_files,
  "changed_file_count": len(changed_files),
  "diff_shortstat": os.environ.get("DIFF_SHORTSTAT") or None,
  "project_root": project_root,
  "coding_standards_path": coding_standards_path or None,
  "branch": os.environ.get("BRANCH") or None,
  "base_commit": os.environ.get("BASE_COMMIT") or None,
  "candidate_commit": os.environ.get("CANDIDATE_COMMIT") or None,
  "candidate_tree": os.environ.get("CANDIDATE_TREE") or None,
  "diff_sha256": os.environ.get("DIFF_SHA256") or None,
  "review_output_path": os.environ.get("REVIEW_OUTPUT_PATH") or None,
}
source_path = os.environ.get("REVIEW_SOURCE_METADATA")
if source_path:
  with open(source_path, encoding="utf-8") as source_file:
    source = json.load(source_file)
  if source["candidate_commit"] != scope["candidate_commit"] or source["candidate_tree"] != scope["candidate_tree"] or os.path.realpath(source["project_root"]) != os.path.realpath(project_root):
    raise ValueError("source snapshot metadata does not match candidate")
  scope["source_snapshot"] = source
print(json.dumps(scope))
PYEOF
