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

python3 - <<PYEOF
import json
diff_path = "$diff_path"
project_root = "$project_root"
coding_standards_path = "$coding_standards_path"
changed_files = """$changed_files_raw""".strip().split('\n')
changed_files = [f for f in changed_files if f]
print(json.dumps({
  "diff_path": diff_path,
  "changed_files": changed_files,
  "project_root": project_root,
  "coding_standards_path": coding_standards_path or None,
}))
PYEOF
