# Enhancement Evaluation: Positional Task Number Specification

**Feature**: E10-F20 - Standardize CLI Command Options
**Enhancement**: Support `shark task start e07 f01 001` format
**Created**: 2026-01-03
**Status**: Under Review
**Evaluator**: UX Designer + Product Manager

---

## Executive Summary

**Client Request**: Support positional task number specification:
```bash
shark task start e07 f01 001
```

Instead of requiring:
```bash
shark task start T-E07-F01-001
```

**Recommendation**: ⚠️ **CONDITIONAL APPROVAL** with modifications

This enhancement has **moderate UX benefit** but introduces **significant parsing complexity and ambiguity**. We recommend a **refined approach** that balances usability with clarity.

---

## Analysis Framework

### 1. Parsing Logic Complexity

#### Current State
```bash
# Task commands accept full task key
shark task start T-E07-F01-001
shark task start t-e07-f01-001  # Case insensitive (proposed in F20)
shark task start T-E07-F01-001-implement-auth  # Slugged
```

**Parsing Logic**: Simple
- Single argument
- Validate against task key pattern: `^T-E\d{2}-F\d{2}-\d{3}$`
- Case-insensitive normalization

#### Proposed Enhancement (Raw Request)
```bash
# Option 1: Full key (existing)
shark task start T-E07-F01-001

# Option 2: Positional epic, feature, task number
shark task start e07 f01 001

# Option 3: Just task number (if context clear?)
shark task start 001
```

**Parsing Logic**: Complex ⚠️

##### Challenge 1: Argument Count Ambiguity

```bash
# 1 argument - is this a full key or just a task number?
shark task start T-E07-F01-001    # Full key
shark task start 001              # Task number only?

# 3 arguments - positional format
shark task start e07 f01 001      # Clear

# But what about flags?
shark task start 001 --notes="Done"  # 1 arg + flag
shark task start e07 f01 001 --notes="Done"  # 3 args + flag
```

**Detection logic needed:**
```
IF arg_count == 1:
    IF arg matches task_key_pattern:
        Use as full key
    ELIF arg matches number_pattern AND context_available:
        Construct key from context + number
    ELSE:
        Error: Invalid task key
ELIF arg_count == 3:
    IF all args match component patterns:
        Construct key: T-{arg1}-{arg2}-{arg3}
    ELSE:
        Error: Invalid components
ELSE:
    Error: Invalid argument count
```

##### Challenge 2: False Positive Risk

```bash
# User might accidentally type numbers
shark task start 7 1 1
# Should this be interpreted as E07-F01-001?
# Or should it error?

# What about partial keys?
shark task start E07 F01 001
shark task start E07-F01 001      # Mixed format?
shark task start 07 01 001        # Numbers only?
```

##### Challenge 3: Context Inference (Option 3)

```bash
# "Just task number" requires context
shark task start 001

# Where does context come from?
# Option A: Previous command in shell?
#   - Not feasible (shark is stateless CLI)
# Option B: Current directory?
#   - Could work if you're in docs/plan/E07-feature-name/
#   - But fragile and surprising
# Option C: Configuration file?
#   - Adds complexity
#   - Breaks predictability
```

**Verdict**: Option 3 (context-based) is **NOT RECOMMENDED**
- Too much implicit behavior
- Hard to predict
- Error-prone

---

### 2. UX Impact Analysis

#### Scenario 1: AI Agent Using Task Commands

**Current Proposal (F20)**:
```python
# Agent has task key from database
task_key = "T-E07-F01-001"
run(f"shark task start {task_key.lower()}")
```

**With Positional Enhancement**:
```python
# Option A: Agent still uses full key (simpler)
task_key = "T-E07-F01-001"
run(f"shark task start {task_key.lower()}")

# Option B: Agent decomposes key into parts
epic = extract_epic(task_key)    # "E07"
feature = extract_feature(task_key)  # "F01"
number = extract_number(task_key)    # "001"
run(f"shark task start {epic} {feature} {number}")
```

**Impact**: Neutral to Negative ⚠️
- AI agents already have full task key from queries
- Decomposing key adds complexity for no benefit
- Potential for errors if decomposition logic is wrong

**Cognitive Load**: Increases
- Now agents must choose between two formats
- Decomposition logic is extra code to maintain

#### Scenario 2: Human Developer Starting Task

**Current (with F20 case insensitivity)**:
```bash
# Dev copies task key from previous command output
$ shark task list e07-f01
  T-E07-F01-001  Implement JWT validation
  T-E07-F01-002  Add refresh token logic

# Start task by copying key
$ shark task start t-e07-f01-001
```

**With Positional Enhancement**:
```bash
# Dev could type manually
$ shark task start e07 f01 1

# But wait - is it "1" or "001"?
# Answer: Both should work (normalize to 001)
```

**Impact**: Slight Positive ✅
- Marginally faster to type (if remembering numbers)
- But copying full key is still easier than remembering numbers
- Most humans will still copy-paste the full key

**Typing Comparison**:
```
Full key:     t-e07-f01-001     (15 chars)
Positional:   e07 f01 001       (11 chars, 27% shorter)
              e07 f01 1         (9 chars if leading zeros optional, 40% shorter)
```

**Realistic Usage**:
- **80%** of time: Dev will copy-paste full key (easier)
- **20%** of time: Dev might type positional (if numbers memorized)

#### Scenario 3: Learning Curve for New Users

**Current**:
```bash
# New user sees task key in list output
$ shark task list
  T-E07-F01-001  Implement JWT validation

# User learns: "I use that key for other commands"
$ shark task start T-E07-F01-001
```

**With Positional Enhancement**:
```bash
# New user sees task key in list output
$ shark task list
  T-E07-F01-001  Implement JWT validation

# User might try:
$ shark task start E07 F01 001    # Will this work?
$ shark task start e07 f01 1      # Or this?
$ shark task start 001            # Or this?
```

**Impact**: Mixed ⚠️
- **Positive**: More flexible input (forgiving)
- **Negative**: More options = more cognitive load ("which way should I use?")
- **Negative**: Harder to document ("multiple ways to do same thing")

**Documentation Complexity**:
```
Before (1 format):
  shark task start <TASK_KEY>

After (2+ formats):
  shark task start <TASK_KEY>
  shark task start <EPIC> <FEATURE> <NUMBER>

  Where:
    TASK_KEY is full key like T-E07-F01-001
    OR provide epic, feature, and task number separately
```

---

### 3. Consistency Across Commands

#### Commands That Would Need Positional Support

If we add positional task numbers, consistency requires:

```bash
# start
shark task start e07 f01 001

# get
shark task get e07 f01 001

# complete
shark task complete e07 f01 001 --notes="Done"

# approve
shark task approve e07 f01 001

# reopen
shark task reopen e07 f01 001

# block
shark task block e07 f01 001 --reason="Blocked"

# unblock
shark task unblock e07 f01 001
```

**Implementation Scope**: Medium
- 7 commands need updating
- Shared parsing logic required
- Tests for each command

**Consistency Benefit**: ✅ Good
- If we do this, doing it for all task commands is right
- Inconsistency would be confusing

---

### 4. Implementation Complexity

#### Parsing Logic Implementation

```go
// ParseTaskKey parses task key from arguments
// Supports:
//   1. Full key: T-E07-F01-001
//   2. Positional: e07 f01 001 (or 1)
func ParseTaskKey(args []string) (string, error) {
    switch len(args) {
    case 1:
        // Full key format
        key := strings.ToUpper(args[0])
        if IsTaskKey(key) {
            return key, nil
        }
        return "", fmt.Errorf("invalid task key: %s", args[0])

    case 3:
        // Positional format: epic feature number
        epic := strings.ToUpper(args[0])
        feature := strings.ToUpper(args[1])
        number := args[2]

        // Validate components
        if !IsEpicKey(epic) {
            return "", fmt.Errorf("invalid epic key: %s", epic)
        }
        if !IsFeatureKey(feature) && !IsFeatureSuffix(feature) {
            return "", fmt.Errorf("invalid feature key: %s", feature)
        }

        // Normalize number to 3 digits
        num, err := strconv.Atoi(number)
        if err != nil || num < 1 || num > 999 {
            return "", fmt.Errorf("invalid task number: %s (must be 1-999)", number)
        }
        number = fmt.Sprintf("%03d", num)

        // Construct full key
        // Handle feature format (F01 or E07-F01)
        if IsFeatureSuffix(feature) {
            // Feature is F01, need to add epic
            feature = fmt.Sprintf("%s-%s", epic, feature)
        }

        return fmt.Sprintf("T-%s-%s", feature, number), nil

    default:
        return "", fmt.Errorf("invalid arguments: expected 1 or 3 arguments")
    }
}
```

**Complexity**: Medium ⚠️
- Logic is manageable but non-trivial
- Edge cases to handle (F01 vs E07-F01)
- Number normalization (1 → 001)
- Error messages must be clear

**Testing Requirements**:
```go
func TestParseTaskKey(t *testing.T) {
    tests := []struct {
        args    []string
        want    string
        wantErr bool
    }{
        // Full key formats
        {[]string{"T-E07-F01-001"}, "T-E07-F01-001", false},
        {[]string{"t-e07-f01-001"}, "T-E07-F01-001", false},

        // Positional formats
        {[]string{"e07", "f01", "001"}, "T-E07-F01-001", false},
        {[]string{"E07", "F01", "1"}, "T-E07-F01-001", false},
        {[]string{"e07", "e07-f01", "1"}, "T-E07-F01-001", false},

        // Invalid formats
        {[]string{"001"}, "", true},  // Just number
        {[]string{"e07", "f01"}, "", true},  // Missing number
        {[]string{"e07", "f01", "1", "extra"}, "", true},  // Too many args
        {[]string{"invalid", "f01", "1"}, "", true},  // Invalid epic
        {[]string{"e07", "invalid", "1"}, "", true},  // Invalid feature
        {[]string{"e07", "f01", "abc"}, "", true},  // Non-numeric number
        {[]string{"e07", "f01", "0"}, "", true},  // Zero
        {[]string{"e07", "f01", "1000"}, "", true},  // Too large
    }

    for _, tt := range tests {
        got, err := ParseTaskKey(tt.args)
        if (err != nil) != tt.wantErr {
            t.Errorf("ParseTaskKey(%v) error = %v, wantErr %v", tt.args, err, tt.wantErr)
        }
        if got != tt.want {
            t.Errorf("ParseTaskKey(%v) = %v, want %v", tt.args, got, tt.want)
        }
    }
}
```

**Estimate**:
- Implementation: 2-3 hours
- Testing: 2-3 hours
- Documentation: 1 hour
- **Total**: 5-7 hours (0.5-1 day)

---

## Edge Cases & Gotchas

### Edge Case 1: Feature Format Ambiguity

```bash
# User provides full feature key
shark task start e07 e07-f01 001

# Parsing:
epic = E07
feature = E07-F01
number = 001

# Constructed key: T-E07-F01-001  ✓ Correct
```

```bash
# User provides feature suffix
shark task start e07 f01 001

# Parsing:
epic = E07
feature = F01
number = 001

# Must construct: T-E07-F01-001
# Logic: If feature is F##, prepend epic
```

**Handling**: ✅ Solvable with logic

### Edge Case 2: Leading Zeros in Numbers

```bash
# All should normalize to same key
shark task start e07 f01 1      # → T-E07-F01-001
shark task start e07 f01 01     # → T-E07-F01-001
shark task start e07 f01 001    # → T-E07-F01-001
```

**Handling**: ✅ Convert to int, format with %03d

### Edge Case 3: Flags with Positional Args

```bash
# Valid
shark task start e07 f01 001 --notes="Started"

# Parsing:
args = ["e07", "f01", "001"]
flags = {notes: "Started"}

# Must separate positional from flags
```

**Handling**: ✅ Cobra handles this (flags are separate)

### Edge Case 4: Accidental Numeric Input

```bash
# User types numbers without prefixes
shark task start 7 1 1

# Should this work? Or error?
```

**Options**:
1. **Accept and normalize**: 7 → E07, 1 → F01, 1 → 001
2. **Reject with error**: Must use E07, F01 format

**Recommendation**: Option 2 (reject)
- Too permissive creates surprises
- Explicit format is clearer

### Edge Case 5: Mixed Format

```bash
# Can user mix positional and full key?
shark task start T-E07-F01-001 --notes="Done"  # Full key + flags ✓
shark task start e07 f01 001 --notes="Done"    # Positional + flags ✓

# But NOT:
shark task start e07 T-E07-F01-001             # Gibberish
```

**Handling**: ✅ Argument count check catches this

---

## Cognitive Load Analysis

### For AI Agents

**Current (with F20)**:
```python
# Simple: use full key from database
task_key = get_task_key_from_db()
run(f"shark task start {task_key}")
```

**With Positional Enhancement**:
```python
# Option A: Still use full key (no change)
task_key = get_task_key_from_db()
run(f"shark task start {task_key}")

# Option B: Decompose for no reason
epic, feature, number = parse_task_key(task_key)
run(f"shark task start {epic} {feature} {number}")
```

**Cognitive Load**: No change (agents will use full key)

### For Human Developers

**Current (with F20)**:
```bash
# List tasks, copy key
$ shark task list e07-f01 --json
$ shark task start t-e07-f01-001
```

**With Positional Enhancement**:
```bash
# List tasks, copy key OR type manually
$ shark task list e07-f01 --json
$ shark task start t-e07-f01-001     # Copy-paste
$ shark task start e07 f01 1         # Manual typing (IF memorized)
```

**Cognitive Load**: Slight increase ⚠️
- "Which format should I use?"
- "Can I mix formats?"
- "Do I need leading zeros?"

**Benefit**: Marginal
- Saves ~6 keystrokes IF typing manually
- Most users copy-paste anyway

### For Documentation

**Current**:
```
TASK_KEY: Full task identifier (e.g., T-E07-F01-001)
          Case insensitive
```

**With Positional Enhancement**:
```
TASK_KEY: Full task identifier (e.g., T-E07-F01-001)
          OR provide epic, feature, and number separately

Examples:
  shark task start T-E07-F01-001
  shark task start e07 f01 001
  shark task start e07 f01 1         (leading zeros optional)

Both formats are case insensitive.
```

**Cognitive Load**: Increase ⚠️
- More examples needed
- More questions from users
- Support burden increases

---

## Alternative Proposals

### Alternative 1: Keep Full Key Only (Status Quo + F20)

```bash
# Only support full key with case insensitivity
shark task start T-E07-F01-001
shark task start t-e07-f01-001     # Case insensitive ✓
```

**Pros**:
- Simplest implementation
- Clearest documentation
- No ambiguity
- AI agents already use this

**Cons**:
- No shorthand for manual typing

### Alternative 2: Support Positional for ALL Arguments (Recommended)

```bash
# Full key (existing)
shark task start T-E07-F01-001

# Positional epic, feature, number
shark task start e07 f01 001
shark task start e07 f01 1         # Leading zeros optional
```

**Pros**:
- More flexible for manual typing
- Consistent with create command pattern
- Handles edge cases cleanly

**Cons**:
- More complex parsing
- More documentation
- Minimal real-world benefit

### Alternative 3: Support Short Key Format (NEW PROPOSAL)

Instead of positional arguments, support a **short key format**:

```bash
# Full key
shark task start T-E07-F01-001

# Short key (drop T- prefix, optional leading zeros)
shark task start E07-F01-001
shark task start e07-f01-1

# Parsing logic:
# If matches "T-E##-F##-###" → use as-is
# If matches "E##-F##-###" → prepend "T-"
```

**Pros**:
- Single argument (no ambiguity)
- Shorter to type (saves 2 chars)
- Clear and unambiguous
- Easy to implement

**Cons**:
- Not as short as positional
- Still requires hyphen typing

---

## Recommendation Matrix

| Criteria | Alt 1: Status Quo | Alt 2: Positional | Alt 3: Short Key |
|----------|-------------------|-------------------|------------------|
| **Implementation Complexity** | Low ✅ | Medium ⚠️ | Low ✅ |
| **Parsing Ambiguity** | None ✅ | Some ⚠️ | None ✅ |
| **Documentation Complexity** | Low ✅ | High ⚠️ | Low ✅ |
| **AI Agent Benefit** | High ✅ | None ❌ | Medium ✅ |
| **Human Typing Benefit** | None ❌ | Medium ✅ | Low ⚠️ |
| **Cognitive Load** | Low ✅ | Medium ⚠️ | Low ✅ |
| **Consistency** | High ✅ | Medium ⚠️ | High ✅ |
| **Error Risk** | Low ✅ | Medium ⚠️ | Low ✅ |

---

## Final Recommendation

### Primary Recommendation: Alternative 3 (Short Key Format)

**Implement short key format support**:
```bash
# Full key (existing)
shark task start T-E07-F01-001

# Short key (new)
shark task start E07-F01-001
shark task start e07-f01-1
```

**Why**:
1. **Simple implementation** (~2 hours)
2. **No ambiguity** (single argument)
3. **Clear benefit** (slightly shorter typing)
4. **Low cognitive load** (still one format, just more permissive)
5. **Easy to document** (just mention T- prefix is optional)

**Implementation**:
```go
func ParseTaskKey(key string) (string, error) {
    key = strings.ToUpper(key)

    // Already has T- prefix
    if strings.HasPrefix(key, "T-") {
        return key, nil
    }

    // Check if it matches E##-F##-### pattern
    if regexp.MustCompile(`^E\d{2}-F\d{2}-\d+$`).MatchString(key) {
        // Extract number and normalize
        parts := strings.Split(key, "-")
        number, _ := strconv.Atoi(parts[2])
        return fmt.Sprintf("T-%s-%s-%03d", parts[0], parts[1], number), nil
    }

    return "", fmt.Errorf("invalid task key format: %s", key)
}
```

### Secondary Recommendation: Do NOT Implement Positional Format

**Do NOT support**:
```bash
shark task start e07 f01 001  # ❌ Too complex for marginal benefit
```

**Reasons**:
1. **Complexity doesn't match benefit**
   - Medium implementation effort
   - High documentation burden
   - Minimal real-world usage

2. **AI agents won't use it**
   - They already have full keys
   - Decomposing keys adds code complexity

3. **Humans won't use it much**
   - Copy-paste is easier than typing
   - 80% of usage is copy-paste
   - 20% can use short key format instead

4. **Creates cognitive burden**
   - "Which format should I use?"
   - More options = more confusion
   - Violates "one obvious way" principle

---

## Acceptance Criteria for Short Key Format

If we proceed with Alternative 3:

### Functional Requirements

- [ ] Task commands accept keys without T- prefix
- [ ] Task commands normalize numbers (1 → 001)
- [ ] Case insensitive throughout
- [ ] All task commands support short format:
  - `task get`
  - `task start`
  - `task complete`
  - `task approve`
  - `task reopen`
  - `task block`
  - `task unblock`

### Error Handling

- [ ] Clear error if format is invalid
- [ ] Suggest correct format in error message
- [ ] No false positives (reject gibberish)

### Testing

- [ ] Unit tests for `ParseTaskKey()`
- [ ] Integration tests for each command
- [ ] Test with all variations:
  - Full key: `T-E07-F01-001`
  - Short key: `E07-F01-001`
  - Short key lowercase: `e07-f01-001`
  - Short key no leading zero: `e07-f01-1`
  - Slugged key: `T-E07-F01-001-task-name`

### Documentation

- [ ] Update CLI_REFERENCE.md
- [ ] Update CLAUDE.md
- [ ] Add examples to --help output
- [ ] Update error messages

---

## Implementation Plan (If Approved)

### Phase 1: Core Parsing (2 hours)
- [ ] Implement `ParseTaskKey()` with short key support
- [ ] Add unit tests
- [ ] Test edge cases

### Phase 2: Update Commands (3 hours)
- [ ] Update all task commands to use `ParseTaskKey()`
- [ ] Add integration tests
- [ ] Verify backward compatibility

### Phase 3: Documentation (1 hour)
- [ ] Update CLI_REFERENCE.md
- [ ] Update CLAUDE.md
- [ ] Update --help text

### Phase 4: Testing (1 hour)
- [ ] Full regression testing
- [ ] AI agent workflow testing
- [ ] User acceptance testing

**Total Estimate**: 7 hours (~1 day)

---

## Questions for Product Manager

1. **Do you want short key format (Alt 3)?**
   - Low cost, low risk, modest benefit
   - Recommendation: YES ✅

2. **Do you want full positional format (Alt 2)?**
   - Medium cost, medium risk, minimal benefit
   - Recommendation: NO ❌

3. **Should we update F20 design docs to include short key format?**
   - If yes, I'll update specification and user journey docs
   - If no, we can defer this enhancement

4. **Timeline priority?**
   - Include in F20 implementation (all at once)?
   - Implement as separate follow-up task?

---

## Related Documents

- `/home/jwwelbor/projects/shark-task-manager/docs/workflow/artifacts/F20-cli-ux-specification.md`
- `/home/jwwelbor/projects/shark-task-manager/docs/workflow/artifacts/F20-user-journey-comparison.md`
- `/home/jwwelbor/projects/shark-task-manager/docs/workflow/artifacts/F20-implementation-guide.md`

---

## Approval

**Status**: Awaiting Product Manager Decision

**Decision Required**:
- [ ] Approve short key format (Alternative 3) - RECOMMENDED
- [ ] Approve positional format (Alternative 2) - NOT RECOMMENDED
- [ ] Reject enhancement, keep status quo (Alternative 1)

**Next Steps After Approval**:
1. Update F20 design documents if needed
2. Create implementation tasks
3. Update acceptance criteria
4. Proceed with development
