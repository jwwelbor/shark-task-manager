# Test Plan — E32-F05: Repoint harness, deprecate slash commands, simplify shark/SKILL.md

**Scope**: Harness-side file edits only. No Go binary changes. No shark-data changes.

---

## 1. Pre-Implementation State Capture

Before starting, record the current state:

```bash
# Capture skill list (will diff against this after Step 1)
ls ~/.claude/skills/ > /tmp/f05-skills-before.txt

# Capture agent list (will diff against this after Step 2)
ls ~/.claude/agents/ > /tmp/f05-agents-before.txt

# Capture shark/SKILL.md line count (should be ~246 before rewrite)
wc -l ~/.claude/skills/shark/SKILL.md

# Capture first 5 lines of each slash command (none should have deprecation header yet)
for f in run feature epic task prd dispatch develop release; do
  echo "=== $f ==="; head -10 ~/.claude/commands/$f.md
done

# Verify orchestration/workflows/run.md exists (source for Step 3)
ls ~/.claude/skills/orchestration/workflows/run.md
```

---

## 2. Post-Implementation Verification Checklist

### Step 1 — Skills deleted

| Check | Command | Pass condition |
|-------|---------|----------------|
| specification-writing absent | `ls ~/.claude/skills/specification-writing 2>&1` | "No such file or directory" |
| quality absent | `ls ~/.claude/skills/quality 2>&1` | "No such file or directory" |
| architecture absent | `ls ~/.claude/skills/architecture 2>&1` | "No such file or directory" |
| research absent | `ls ~/.claude/skills/research 2>&1` | "No such file or directory" |
| implementation absent | `ls ~/.claude/skills/implementation 2>&1` | "No such file or directory" |
| test-driven-development absent | `ls ~/.claude/skills/test-driven-development 2>&1` | "No such file or directory" |
| assessment absent | `ls ~/.claude/skills/assessment 2>&1` | "No such file or directory" |
| uat absent | `ls ~/.claude/skills/uat 2>&1` | "No such file or directory" |
| debugging absent | `ls ~/.claude/skills/debugging 2>&1` | "No such file or directory" |
| orchestration absent | `ls ~/.claude/skills/orchestration 2>&1` | "No such file or directory" |
| shark skill still present | `ls ~/.claude/skills/shark/SKILL.md` | File exists |

One-shot verification:
```bash
for s in specification-writing quality architecture research implementation \
          test-driven-development assessment uat debugging orchestration; do
  [ -d ~/.claude/skills/$s ] && echo "FAIL: $s still exists" || echo "PASS: $s absent"
done
ls ~/.claude/skills/shark/SKILL.md && echo "PASS: shark present" || echo "FAIL: shark missing"
```

### Step 2 — Agents deleted, keepers present

```bash
# 9 deleted agents — each must be absent
for a in architect.md business-analyst.md developer.md qa.md tech-lead.md \
          product-manager.md tech-director.md researcher.md uat-agent.md; do
  [ -f ~/.claude/agents/$a ] && echo "FAIL: $a still exists" || echo "PASS: $a absent"
done

# Exactly 6 keepers must remain
ls ~/.claude/agents/
# Expected: client.md  code-simplifier.md  cx-designer.md  devops.md  human-checkpoint.md  ux-designer.md
# Verify count:
[ "$(ls ~/.claude/agents/ | wc -l)" -eq 6 ] && echo "PASS: 6 agents" || echo "FAIL: wrong count"
```

### Step 3 — shark/SKILL.md rewritten

```bash
# File must exist
ls ~/.claude/skills/shark/SKILL.md && echo "PASS" || echo "FAIL: missing"

# Must NOT be the old CLI reference (old file starts with a Quick Reference section)
grep -q "Quick Reference" ~/.claude/skills/shark/SKILL.md && echo "FAIL: old CLI content present" || echo "PASS: old content gone"

# Must contain the dispatch loop
grep -q "spawn_agent\|shark next\|shark advance" ~/.claude/skills/shark/SKILL.md && echo "PASS: dispatch loop present" || echo "FAIL: dispatch loop missing"

# Must contain rejection escalation guard (key behavior from orchestration/workflows/run.md)
grep -q "2-strike\|rejection.*escalat\|escalat.*reject" ~/.claude/skills/shark/SKILL.md && echo "PASS: escalation guard present" || echo "FAIL: escalation guard missing"

# Key format table must be retained
grep -q "E##-F##\|Auto-Detect" ~/.claude/skills/shark/SKILL.md && echo "PASS: key table present" || echo "FAIL: key table missing"

# Common Mistakes list must be retained
grep -q "Common Mistakes\|Mistakes to Avoid" ~/.claude/skills/shark/SKILL.md && echo "PASS: mistakes list present" || echo "FAIL: mistakes list missing"

# Context file references must be present
grep -q "context/task-execution-pattern\|context/entity-crud\|context/workflow-and-status" ~/.claude/skills/shark/SKILL.md && echo "PASS: context refs present" || echo "FAIL: context refs missing"
```

### Step 4 — Deprecation headers in slash commands

For each command, the deprecation block must appear before the first heading (`# `):

```bash
for cmd in run feature epic task prd dispatch develop release; do
  FILE=~/.claude/commands/$cmd.md
  # Deprecation block present
  grep -q "DEPRECATED (F05)" $FILE && echo "PASS: $cmd has deprecation" || echo "FAIL: $cmd missing deprecation"
  # File still has a functional body (first heading still present)
  grep -q "^# " $FILE && echo "PASS: $cmd has body" || echo "FAIL: $cmd body missing"
  # Deprecation appears BEFORE first heading
  DEP_LINE=$(grep -n "DEPRECATED" $FILE | head -1 | cut -d: -f1)
  HEAD_LINE=$(grep -n "^# " $FILE | head -1 | cut -d: -f1)
  [ "$DEP_LINE" -lt "$HEAD_LINE" ] && echo "PASS: $cmd order correct" || echo "FAIL: $cmd order wrong (dep=$DEP_LINE head=$HEAD_LINE)"
done

# /run must use the specific orchestration-skill deprecation wording
grep -q "orchestration.*skill\|dispatch loop has moved" ~/.claude/commands/run.md && echo "PASS: /run specific wording" || echo "FAIL: /run wording wrong"
```

### AC-2 — shark admin validate passes with quality moved aside

```bash
# Move quality aside
mv ~/.claude/skills/quality /tmp/quality-aside-f05

# Verify quality exists in embedded sharkdata
ls /home/jwwel/projects/shark-task-manager/internal/sharkdata/default_data/skills/quality/SKILL.md && echo "PASS: quality in sharkdata" || echo "FAIL: missing from sharkdata"

# Run validate (must exit 0)
cd /home/jwwel/projects/shark-task-manager
./bin/shark admin validate
echo "Exit code: $?"  # Must be 0

# Restore
mv /tmp/quality-aside-f05 ~/.claude/skills/quality
```

---

## 3. Rollback Procedure

If something goes wrong mid-implementation:

**Steps 1–2 (skill/agent deletion) — restore from git:**
```bash
# Skills and agents are not in the repo; restore from backup made in pre-implementation capture.
# If no backup was made, reinstall the harness from source.
# The shark repo itself is unaffected (no changes to internal/).
```

**Step 3 (SKILL.md rewrite) — restore original:**
```bash
# If the original shark/SKILL.md was the 246-line CLI reference, restore it:
git -C ~/.claude show HEAD:skills/shark/SKILL.md > ~/.claude/skills/shark/SKILL.md
# If ~/.claude is not a git repo, restore from any copy or re-run the F05 pre-flight capture.
```

**Step 4 (deprecation headers) — strip headers:**
```bash
# Remove only the > **DEPRECATED (F05)** block from each file.
# Each file had the block inserted immediately after the closing '---' of front matter.
# The original body is intact below it — just delete the 3-line block.
```

**Step 5 (AC-2 test) — always safe (mv is reversible):**
```bash
mv /tmp/quality-aside-f05 ~/.claude/skills/quality  # Undo the move
```

---

## 4. Pass/Fail Criteria

**PASS**: All of the following are true after implementation:

- All 10 skill directories are absent from `~/.claude/skills/`
- `~/.claude/skills/shark/` is present and SKILL.md contains the dispatch loop (not the old CLI reference)
- All 9 in-scope agent files are absent from `~/.claude/agents/`
- Exactly 6 agent files remain: `client.md`, `code-simplifier.md`, `cx-designer.md`, `devops.md`, `human-checkpoint.md`, `ux-designer.md`
- All 8 slash command files have `> **DEPRECATED (F05)**` appearing before the first `# ` heading
- `shark admin validate` exits 0 with `~/.claude/skills/quality/` moved to `/tmp`

**FAIL** (any single check fails): Implementation is incomplete. Do not advance to F06.
