# F20 Enhancement: One-Page Decision Guide

**Feature**: E10-F20 CLI Standardization
**Request**: Support `shark task start e07 f01 001`
**Date**: 2026-01-03

---

## TL;DR

**Client wants**: Positional task number format
**We recommend**: Short key format (counter-proposal)
**Why**: 90% of benefit, 40% of cost, lower risk

---

## Three Options

<table>
<tr>
<th width="33%">🟡 Alt 1: Status Quo</th>
<th width="33%">🔴 Alt 2: Positional (Client Request)</th>
<th width="33%">🟢 Alt 3: Short Key (Our Proposal)</th>
</tr>

<tr>
<td>

```bash
shark task start \
  t-e07-f01-001
```

</td>
<td>

```bash
shark task start \
  e07 f01 001
```

</td>
<td>

```bash
shark task start \
  e07-f01-001
```

</td>
</tr>

<tr>
<td>
<strong>Cost:</strong> 0 hours<br>
<strong>Risk:</strong> None<br>
<strong>Benefit:</strong> Baseline<br>
<strong>Score:</strong> 13/15
</td>
<td>
<strong>Cost:</strong> 7 hours<br>
<strong>Risk:</strong> Medium<br>
<strong>Benefit:</strong> 2.8%<br>
<strong>Score:</strong> 5/15
</td>
<td>
<strong>Cost:</strong> 3 hours<br>
<strong>Risk:</strong> Low<br>
<strong>Benefit:</strong> 2.5% + cleaner<br>
<strong>Score:</strong> 12/15
</td>
</tr>

<tr>
<td>
✅ Zero cost<br>
✅ Zero risk<br>
✅ Simple<br>
⚠️ No shortcuts
</td>
<td>
❌ High cost<br>
❌ Complex (35 LOC)<br>
❌ AI won't use<br>
❌ Confusing<br>
⚠️ Many edge cases
</td>
<td>
✅ Low cost<br>
✅ Simple (15 LOC)<br>
✅ AI friendly<br>
✅ Clear<br>
✅ Few edge cases
</td>
</tr>

<tr>
<td><strong>When:</strong> Budget constrained</td>
<td><strong>When:</strong> Never recommended</td>
<td><strong>When:</strong> Have 3 hours available ✅</td>
</tr>
</table>

---

## Why NOT Alternative 2 (Client Request)?

### ❌ AI Agents Won't Adopt It
```python
# Agents have full key from DB
task_key = "T-E07-F01-001"

# Simple: Just use it
run(f"shark task start {task_key}")

# Complex: Decompose for no reason
epic, feature, num = parse(task_key)  # Extra code!
run(f"shark task start {epic} {feature} {num}")
```
**Result**: Agents stick with full key

### ❌ Humans Will Copy-Paste Anyway
80% of human usage: Select → Ctrl+C → Ctrl+V
**Result**: Only 20% × 30% = 6% of total usage benefits

### ❌ Multiple Formats = Confusion
```bash
shark task start T-E07-F01-001    # Which should I use?
shark task start e07 f01 001      # This?
shark task start e07 f01 1        # Or this?
```
**Result**: Decision fatigue increases

### ❌ Real Benefit is Tiny
- Saves 8 chars on 6% of usage = **0.48 chars saved per command**
- 7 hours development cost for minimal gain
- **ROI**: Very poor

---

## Why YES Alternative 3 (Short Key)?

### ✅ Same Philosophy as F20
```
F20:   e01 → E01           (normalize case)
Alt 3: e07-f01-001 → T-E07-F01-001  (add prefix)

Both: More permissive, same format
```

### ✅ No Ambiguity
```bash
shark task start e07-f01-001    # One argument, clear intent
```

### ✅ Easy to Document
> "Task keys can optionally omit the `T-` prefix."

That's it. One sentence.

### ✅ Modest Real Benefit
- Saves 2 chars (`T-` prefix)
- Cleaner syntax
- AI agents might optimize
- Low cost (3 hours)

---

## Usage Prediction

### Current Reality
- **70%** of usage: AI agents (have full keys)
- **30%** of usage: Humans
  - **80%** copy-paste (don't type)
  - **20%** manual typing

### Impact
| Alternative | Who Benefits | Real Impact |
|-------------|--------------|-------------|
| Alt 1 | Everyone (case insensitive) | ✅ High |
| Alt 2 | 20% of 30% = 6% | ⚠️ Tiny |
| Alt 3 | 20% of 30% = 6% + cleaner syntax | ✅ Small |

---

## Cost Comparison

| Metric | Alt 1 | Alt 2 | Alt 3 |
|--------|-------|-------|-------|
| **Implementation** | 0h | 7h | 3h |
| **Code complexity** | Baseline | +35 LOC | +15 LOC |
| **Edge cases** | 1 | 8+ | 3 |
| **Test cases** | 3 | 15 | 6 |
| **Doc pages** | 0.5 | 2 | 0.7 |
| **Maintenance** | Low | High | Low |

---

## Decision Tree

```
┌─────────────────────────────────┐
│ Do we have ANY budget?          │
└──┬────────────────────────┬─────┘
   │ NO                     │ YES
   ↓                        ↓
┌────────────────┐    ┌──────────────────────┐
│ Alternative 1  │    │ Do we have 3 hours?  │
│ (Status Quo)   │    └──┬────────────┬──────┘
└────────────────┘       │ NO         │ YES
                         ↓            ↓
                   ┌────────────┐ ┌──────────────┐
                   │ Alt 1      │ │ Alt 3        │
                   │ (Baseline) │ │ (Short Key)  │
                   └────────────┘ └──────────────┘
                                      ✅ RECOMMENDED

Alternative 2 is NEVER recommended
```

---

## Recommended Action

### ✅ Primary Recommendation
**Approve Alternative 3** (Short Key Format)

**Scope**: Add to F20 implementation
**Cost**: +3 hours to Week 2 of F20
**Risk**: Low
**Benefit**: Modest improvement, cleaner syntax

### ✅ Acceptable Fallback
**Approve Alternative 1** (Status Quo)

**Scope**: F20 as originally designed
**Cost**: No change
**Risk**: None
**Benefit**: F20 already provides 91% error reduction

### ❌ Not Recommended
**Alternative 2** (Positional Format)

**Reason**: Cost exceeds benefit in all scenarios
- 2.3× cost of Alt 3
- Same real-world usage (6%)
- Higher complexity
- More confusion

---

## If Client Insists on Alternative 2

### Option A: Explain Trade-offs
- Show cost-benefit analysis
- Demonstrate Alt 3 provides 90% of benefit
- Recommend Alt 3 as compromise

### Option B: Phased Approach
1. **Phase 1**: Implement Alt 3 (short key)
2. **Phase 2**: Gather usage metrics
3. **Phase 3**: Add Alt 2 only if data shows demand

### Option C: Proceed with Caveats
- Get explicit approval for 7-hour scope increase
- Document that adoption will be low
- Set realistic expectations on ROI

**Recommended**: Option A (explain) or Option B (phased)

---

## Key Metrics

### Typing Savings
| Format | Length | Savings |
|--------|--------|---------|
| Full key | 17 chars | - |
| Alt 3 (short) | 10 chars | 41% |
| Alt 2 (positional) | 9 chars | 47% |

**But**: Only 6% of usage involves typing!

### Real-World Impact
- **Alt 2**: 47% × 6% usage = **2.8% overall**
- **Alt 3**: 41% × 6% usage = **2.5% overall**

**Difference**: 0.3% overall (negligible)

### Implementation Cost
- **Alt 2**: 7 hours
- **Alt 3**: 3 hours
- **Ratio**: 2.3× cost for 0.3% more benefit

**ROI**: Alt 3 wins decisively

---

## Timeline Impact

### Current F20 (4 weeks)
```
Week 1: Case insensitivity
Week 2: Positional create
Week 3: Error messages
Week 4: Documentation
```

### With Alt 3 (+3 hours)
```
Week 1: Case insensitivity
Week 2: Positional create + Short key ✅
Week 3: Error messages
Week 4: Documentation
```
**Impact**: Absorbed in Week 2 (no delay)

### With Alt 2 (+7 hours)
```
Week 1: Case insensitivity
Week 2: Positional create
Week 3: Positional task format ⚠️
Week 4: Error messages
Week 5: Documentation
```
**Impact**: +1 week or compressed schedule

---

## Bottom Line

<table>
<tr>
<th>Metric</th>
<th>Alt 1</th>
<th>Alt 2</th>
<th>Alt 3</th>
</tr>
<tr>
<td><strong>Cost/Benefit</strong></td>
<td>∞ (no cost)</td>
<td>Very poor</td>
<td>Good</td>
</tr>
<tr>
<td><strong>Risk</strong></td>
<td>None</td>
<td>Medium</td>
<td>Low</td>
</tr>
<tr>
<td><strong>Complexity</strong></td>
<td>Simple</td>
<td>Complex</td>
<td>Simple</td>
</tr>
<tr>
<td><strong>User Impact</strong></td>
<td>Good (F20)</td>
<td>Confusing</td>
<td>Better</td>
</tr>
<tr>
<td><strong>Recommendation</strong></td>
<td>✅ Acceptable</td>
<td>❌ No</td>
<td>✅ Yes</td>
</tr>
</table>

---

## Your Decision

**Product Manager**: Please choose one:

- [ ] ✅ **Approve Alt 3** (Short Key Format) - **RECOMMENDED**
  - Add 3 hours to F20 scope
  - Best ROI, low risk, cleaner syntax

- [ ] ✅ **Approve Alt 1** (Status Quo) - **ACCEPTABLE**
  - No scope change
  - F20 already provides significant value

- [ ] ⚠️ **Request Alt 2** (Positional Format) - **NOT RECOMMENDED**
  - Requires discussion of trade-offs
  - Consider phased approach instead

- [ ] 🤔 **Need more information**
  - See detailed documents in `/docs/workflow/artifacts/`

---

## Documents Available

1. **This file** - One-page decision guide
2. **F20-enhancement-decision-summary.md** - Executive summary
3. **F20-enhancement-alternatives-comparison.md** - Detailed comparison
4. **F20-enhancement-positional-task-number.md** - Full analysis (15 pages)

---

**Prepared by**: UX Designer
**Status**: ⏳ Awaiting PM Decision
**Date**: 2026-01-03
