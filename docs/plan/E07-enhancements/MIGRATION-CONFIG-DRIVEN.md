# Migration Guide: Config-Driven Architecture

**Date:** 2026-01-16
**Features:** E07-F14 (v2), E07-F22, E07-F23
**Breaking Changes:** None (additive only)

## Overview

The config-driven architecture adds enhanced metadata fields to `.sharkconfig.json` to enable:
- Weighted progress calculation (recognizes work before completion)
- Automatic rejection reason detection (backward transitions)
- Responsibility-based work breakdown
- Feature/epic status blocking logic

## Migration Steps

### 1. Backup Current Config

```bash
cp .sharkconfig.json .sharkconfig.json.backup
```

### 2. Update Config with Enhanced Metadata

Use the following prompt with an AI agent to automatically add the required fields:

---

## LLM Migration Prompt

```
Update .sharkconfig.json to add enhanced metadata fields for config-driven architecture.

Add these fields to ALL statuses in status_metadata:
- progress_weight (float 0.0-1.0): How much status contributes to progress
- responsibility (string): Who's responsible - "agent", "human", "qa_team", or "none"
- blocks_feature (boolean): Should this status make parent feature/epic blocked

Also add top-level field:
- require_rejection_reason: true

Use these recommended values:

Status Progress Weights:
- completed: 1.0
- in_approval: 0.95
- ready_for_approval: 0.9 (IMPORTANT: agent work done, waiting on human!)
- in_qa: 0.85
- ready_for_qa: 0.8
- in_code_review: 0.8
- ready_for_code_review: 0.75
- in_development: 0.5
- ready_for_development: 0.25
- in_refinement: 0.15
- ready_for_refinement: 0.1
- draft: 0.0
- blocked: 0.0
- on_hold: 0.0
- cancelled: 0.0

Responsibility Values:
- ready_for_approval: "human" (awaiting human approval)
- in_approval: "human" (under human review)
- in_qa: "qa_team"
- ready_for_qa: "qa_team"
- in_code_review: "agent" (code reviewer agent)
- ready_for_code_review: "agent"
- in_development: "agent"
- in_refinement: "agent"
- ready_for_refinement: "agent"
- All others: "none"

Blocks Feature:
- blocked: true (ONLY this status blocks parent feature)
- All others: false

Preserve all existing fields (color, description, phase, agent_types) - only ADD new fields.
```

---

### 3. Verify Migration

Check that all statuses have the new fields:

```bash
# Check that require_rejection_reason was added
jq '.require_rejection_reason' .sharkconfig.json

# Check that all statuses have progress_weight
jq '.status_metadata | to_entries[] | select(.value.progress_weight == null) | .key' .sharkconfig.json

# Should return empty (no statuses missing progress_weight)
```

### 4. Test Configuration

```bash
# Validate config loads correctly
./bin/shark config get require_rejection_reason

# Test that progress calculation works
./bin/shark feature get <any-feature-key>
```

## What This Enables

### E07-F14: Cascading Status Calculation (v2)
- Feature/epic status ALWAYS calculated from children
- No manual override - status reflects reality
- Weighted progress recognizes partial completion

### E07-F22: Rejection Reason for Backward Transitions
- Automatic detection: if progress_weight decreases, rejection reason required
- Example: ready_for_code_review (0.75) → in_development (0.5) = backward transition
- Top-level config: `require_rejection_reason: true`

### E07-F23: Enhanced Status Tracking
- Progress breakdown by responsibility
- Action items surfaced automatically
- Work summary showing what needs attention

## Rollback

If needed, restore from backup:

```bash
cp .sharkconfig.json.backup .sharkconfig.json
```

## Customization

You can adjust progress weights for your workflow:

```json
{
  "status_metadata": {
    "ready_for_approval": {
      "progress_weight": 0.95  // Adjust if you want different recognition
    }
  }
}
```

**Key principle:** `ready_for_approval` at 0.9 (90%) recognizes that agent work is complete and it's waiting on human approval!

## Next Steps

After migration:
1. Features (E07-F14, E07-F22, E07-F23) will automatically use new config metadata
2. Progress calculations will reflect weighted progress
3. Backward transitions will require rejection reasons
4. Feature status will block when tasks are blocked

## Support

See documentation:
- `CONFIG-DRIVEN-ARCHITECTURE.md` - Master guide
- `E07-F14-cascading-status-calculation/architecture-design-v2-config-driven.md`
- `E07-F22-rejection-reason-for-status-transitions/prd.md`
- `E07-F23-enhanced-status-tracking/architecture.md`
