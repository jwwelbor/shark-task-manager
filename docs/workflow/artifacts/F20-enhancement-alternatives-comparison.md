# Enhancement Alternatives: Quick Comparison

**Feature**: E10-F20 - Standardize CLI Command Options
**Enhancement**: Task key format flexibility
**Created**: 2026-01-03

---

## Visual Comparison

### Alternative 1: Status Quo (F20 Only)

```bash
# Create task
shark task create e07 f01 "Task title"
# Result: Task T-E07-F01-001 created

# Start task (ONLY THIS FORMAT)
shark task start T-E07-F01-001
shark task start t-e07-f01-001     # Case insensitive ✓

# Complete task
shark task complete t-e07-f01-001
```

**Character count**: 17 chars for task key

---

### Alternative 2: Positional Format (Client Request)

```bash
# Create task
shark task create e07 f01 "Task title"
# Result: Task T-E07-F01-001 created

# Start task (MULTIPLE FORMATS)
shark task start T-E07-F01-001              # Full key (17 chars)
shark task start t-e07-f01-001              # Lowercase (17 chars)
shark task start e07 f01 001                # Positional (11 chars)
shark task start e07 f01 1                  # Short number (9 chars)

# Complete task
shark task complete e07 f01 1               # Works with all commands
```

**Character count**: 9-17 chars (depending on format)

**Savings**: Up to 47% fewer characters

---

### Alternative 3: Short Key Format (Recommended)

```bash
# Create task
shark task create e07 f01 "Task title"
# Result: Task T-E07-F01-001 created

# Start task (SIMPLIFIED FORMATS)
shark task start T-E07-F01-001              # Full key (17 chars)
shark task start E07-F01-001                # Drop T- prefix (12 chars)
shark task start e07-f01-1                  # Lowercase + short (10 chars)

# Complete task
shark task complete e07-f01-1               # Works with all commands
```

**Character count**: 10-17 chars (depending on format)

**Savings**: Up to 41% fewer characters

---

## Complexity Comparison

### Parsing Logic

#### Alternative 1: Status Quo
```go
func ParseTaskKey(key string) (string, error) {
    key = strings.ToUpper(key)
    if IsTaskKey(key) {
        return key, nil
    }
    return "", errors.New("invalid task key")
}
```
**Lines**: 6
**Complexity**: O(1)
**Edge cases**: 1 (invalid format)

#### Alternative 2: Positional Format
```go
func ParseTaskKey(args []string) (string, error) {
    switch len(args) {
    case 1:
        // Full key format
        key := strings.ToUpper(args[0])
        if IsTaskKey(key) {
            return key, nil
        }
        return "", errors.New("invalid task key")

    case 3:
        // Positional format: epic feature number
        epic := strings.ToUpper(args[0])
        feature := strings.ToUpper(args[1])
        number := args[2]

        // Validate each component
        if !IsEpicKey(epic) {
            return "", fmt.Errorf("invalid epic: %s", epic)
        }
        if !IsFeatureKey(feature) && !IsFeatureSuffix(feature) {
            return "", fmt.Errorf("invalid feature: %s", feature)
        }

        // Normalize number
        num, err := strconv.Atoi(number)
        if err != nil || num < 1 || num > 999 {
            return "", fmt.Errorf("invalid number: %s", number)
        }
        number = fmt.Sprintf("%03d", num)

        // Construct key
        if IsFeatureSuffix(feature) {
            feature = fmt.Sprintf("%s-%s", epic, feature)
        }
        return fmt.Sprintf("T-%s-%s", feature, number), nil

    default:
        return "", errors.New("expected 1 or 3 arguments")
    }
}
```
**Lines**: 35
**Complexity**: O(n) where n is argument count
**Edge cases**: 8+ (see enhancement doc)

#### Alternative 3: Short Key Format
```go
func ParseTaskKey(key string) (string, error) {
    key = strings.ToUpper(key)

    // Already has T- prefix
    if strings.HasPrefix(key, "T-") {
        return NormalizeTaskKey(key), nil
    }

    // Check E##-F##-### pattern (without T-)
    if regexp.MustCompile(`^E\d{2}-F\d{2}-\d+$`).MatchString(key) {
        parts := strings.Split(key, "-")
        number, _ := strconv.Atoi(parts[2])
        return fmt.Sprintf("T-%s-%s-%03d", parts[0], parts[1], number), nil
    }

    return "", errors.New("invalid task key format")
}
```
**Lines**: 15
**Complexity**: O(1)
**Edge cases**: 3 (invalid format, bad number, malformed)

---

## Usage Patterns

### AI Agent

#### Alternative 1
```python
task_key = "T-E07-F01-001"  # From database
run(f"shark task start {task_key.lower()}")
```
**Code complexity**: Simple ✅

#### Alternative 2
```python
# Option A: Use full key (no change)
task_key = "T-E07-F01-001"
run(f"shark task start {task_key.lower()}")

# Option B: Decompose (unnecessary complexity)
epic, feature, num = parse_key(task_key)
run(f"shark task start {epic} {feature} {num}")
```
**Code complexity**: Same or worse ⚠️

#### Alternative 3
```python
task_key = "T-E07-F01-001"  # From database
# Can drop T- prefix for slight savings
short_key = task_key[2:]  # "E07-F01-001"
run(f"shark task start {short_key.lower()}")
```
**Code complexity**: Simple ✅

---

### Human Developer

#### Alternative 1
```bash
# Must copy full key
$ shark task list e07-f01
  T-E07-F01-001  Implement JWT

$ shark task start t-e07-f01-001
```
**Typing**: Copy-paste or 17 chars

#### Alternative 2
```bash
# Can copy OR type manually
$ shark task list e07-f01
  T-E07-F01-001  Implement JWT

# Option A: Copy-paste (most common)
$ shark task start t-e07-f01-001

# Option B: Type manually (if numbers memorized)
$ shark task start e07 f01 1
```
**Typing**: Copy-paste or 9-17 chars

**Decision fatigue**: Which format to use? ⚠️

#### Alternative 3
```bash
# Can copy OR simplify
$ shark task list e07-f01
  T-E07-F01-001  Implement JWT

# Option A: Copy-paste
$ shark task start t-e07-f01-001

# Option B: Drop T- prefix
$ shark task start e07-f01-1
```
**Typing**: Copy-paste or 10-17 chars

**Decision fatigue**: Lower (still same format, just prefix optional) ✅

---

## Documentation Impact

### Alternative 1
```
shark task start <TASK_KEY>

  TASK_KEY: Full task identifier (e.g., T-E07-F01-001)
            Case insensitive

Examples:
  shark task start T-E07-F01-001
  shark task start t-e07-f01-001
```
**Doc length**: Short ✅
**Clarity**: High ✅

### Alternative 2
```
shark task start <TASK_KEY>
shark task start <EPIC> <FEATURE> <NUMBER>

  TASK_KEY: Full task identifier (e.g., T-E07-F01-001)
  EPIC: Epic key (e.g., E07)
  FEATURE: Feature key (e.g., F01 or E07-F01)
  NUMBER: Task number (1-999, leading zeros optional)

  Case insensitive for all formats

Examples:
  shark task start T-E07-F01-001
  shark task start t-e07-f01-001
  shark task start e07 f01 001
  shark task start E07 F01 1
  shark task start e07 e07-f01 1

Note: Both feature formats (F01 and E07-F01) work
```
**Doc length**: Long ⚠️
**Clarity**: Medium (many examples needed) ⚠️

### Alternative 3
```
shark task start <TASK_KEY>

  TASK_KEY: Task identifier (e.g., T-E07-F01-001)
            T- prefix is optional
            Case insensitive
            Leading zeros optional in number

Examples:
  shark task start T-E07-F01-001
  shark task start E07-F01-001      (T- prefix omitted)
  shark task start e07-f01-1        (lowercase, no zeros)
```
**Doc length**: Medium ✅
**Clarity**: High ✅

---

## Error Handling Examples

### Alternative 1

```bash
$ shark task start e07-f01-001
Error: Invalid task key "e07-f01-001"
  Expected format: T-E##-F##-### (e.g., T-E07-F01-001)
  Tip: Task keys must start with "T-"
```

### Alternative 2

```bash
# Error: Wrong argument count
$ shark task start e07 f01
Error: Invalid arguments for task key
  Expected: 1 argument (full key) OR 3 arguments (epic feature number)
  Got: 2 arguments

  Examples:
    shark task start T-E07-F01-001
    shark task start e07 f01 001

# Error: Invalid component
$ shark task start e7 f01 001
Error: Invalid epic key "e7"
  Expected format: E## (e.g., E01, E07, E99)
  Tip: Use two-digit numbers
```

### Alternative 3

```bash
$ shark task start e07-f01
Error: Invalid task key "e07-f01"
  Expected format: T-E##-F##-### or E##-F##-###
  Missing: task number

  Examples:
    shark task start T-E07-F01-001
    shark task start E07-F01-001    (T- prefix optional)
```

**Error clarity**: All alternatives can have good errors ✅

---

## Real-World Usage Prediction

Based on user behavior analysis:

### AI Agents (70% of usage)

| Alternative | Will Use | Reasoning |
|-------------|----------|-----------|
| Alt 1 | Full key from DB | Default behavior ✅ |
| Alt 2 | Full key from DB | Decomposing adds complexity ❌ |
| Alt 3 | Full key OR short | Might optimize by dropping T- ✅ |

**Verdict**: Alternatives 1 and 3 are equivalent for AI agents

### Human Developers (30% of usage)

| Scenario | Alt 1 | Alt 2 | Alt 3 |
|----------|-------|-------|-------|
| Copy-paste (80% of time) | Full key | Full key | Full key |
| Manual typing (20% of time) | Full key (17 chars) | Positional (9 chars) | Short key (10 chars) |

**Actual typing savings**:
- Alt 1: 0% savings
- Alt 2: 47% savings on 20% of 30% of usage = **2.8% overall**
- Alt 3: 41% savings on 20% of 30% of usage = **2.5% overall**

**Verdict**: Real-world benefit is MINIMAL for all alternatives

---

## Risk Assessment

### Alternative 1: Status Quo
- **Implementation risk**: None (already done in F20)
- **User confusion risk**: Low
- **Maintenance burden**: Low
- **Breaking changes**: None
- **Overall risk**: ✅ LOW

### Alternative 2: Positional Format
- **Implementation risk**: Medium (complex parsing)
- **User confusion risk**: Medium (multiple formats)
- **Maintenance burden**: High (more edge cases)
- **Breaking changes**: None (additive)
- **Overall risk**: ⚠️ MEDIUM

### Alternative 3: Short Key Format
- **Implementation risk**: Low (simple parsing)
- **User confusion risk**: Low (same format, just more permissive)
- **Maintenance burden**: Low (minimal edge cases)
- **Breaking changes**: None (additive)
- **Overall risk**: ✅ LOW

---

## Cost-Benefit Analysis

### Alternative 1: Status Quo
- **Cost**: 0 hours (already in F20)
- **Benefit**: Case insensitivity (91% error reduction)
- **ROI**: ∞ (benefit with no additional cost)

### Alternative 2: Positional Format
- **Cost**: 7 hours (implementation + testing + docs)
- **Benefit**: 2.8% overall typing reduction (minimal)
- **ROI**: Low (high cost, minimal benefit)

### Alternative 3: Short Key Format
- **Cost**: 3 hours (implementation + testing + docs)
- **Benefit**: 2.5% overall typing reduction + cleaner syntax
- **ROI**: Medium (low cost, small benefit, cleaner design)

---

## Final Recommendation

### ✅ Approve: Alternative 3 (Short Key Format)

**Why**:
1. **Low cost** (3 hours)
2. **Low risk** (simple implementation)
3. **Modest benefit** (cleaner syntax)
4. **Consistent with F20 philosophy** (more permissive, not more complex)
5. **Easy to explain** ("T- prefix is optional")

### ❌ Reject: Alternative 2 (Positional Format)

**Why**:
1. **Medium cost** (7 hours)
2. **Medium risk** (parsing complexity)
3. **Minimal benefit** (2.8% real-world savings)
4. **High cognitive load** (multiple formats to remember)
5. **Violates F20 philosophy** (simplicity over flexibility)

---

## Decision Matrix

|  | Implementation | Complexity | UX Benefit | AI Benefit | Risk | **SCORE** |
|--|----------------|------------|------------|------------|------|-----------|
| **Alt 1** | ✅✅✅ (0h) | ✅✅✅ (simple) | ⚠️ (none) | ✅✅✅ (same) | ✅✅✅ (low) | **13/15** |
| **Alt 2** | ⚠️ (7h) | ⚠️ (complex) | ⚠️ (minimal) | ❌ (worse) | ⚠️ (medium) | **5/15** |
| **Alt 3** | ✅✅ (3h) | ✅✅ (simple) | ✅ (modest) | ✅✅ (same/better) | ✅✅✅ (low) | **12/15** |

**Winner**: Alternative 1 (Status Quo) if we value zero cost
**Winner**: Alternative 3 (Short Key) if we want modest improvement for low cost

---

## Recommendation to Product Manager

**Proceed with Alternative 3 (Short Key Format)** as an **optional enhancement** to F20.

**Rationale**:
- Low cost, low risk
- Aligns with F20's permissive philosophy
- Provides modest but real benefit
- Doesn't add cognitive complexity
- Easy to implement and document

**If time/budget is constrained**:
- Alternative 1 (Status Quo) is perfectly acceptable
- F20 already provides 91% error reduction through case insensitivity
- Short key format is "nice to have" not "must have"

**Do NOT proceed with Alternative 2 (Positional Format)**:
- Cost doesn't justify minimal benefit
- Adds complexity that violates F20 design principles
- Users won't adopt it (copy-paste is easier)
- AI agents won't use it (have full keys already)

---

## Next Steps

**If Alternative 3 is approved**:
1. Update F20-cli-ux-specification.md to include short key format
2. Add short key examples to F20-user-journey-comparison.md
3. Update F20-implementation-guide.md with parsing logic
4. Create implementation task in shark
5. Estimate: Add 3 hours to F20 implementation timeline

**If Alternative 1 is chosen (status quo)**:
1. Close enhancement request
2. Document decision (positional format rejected)
3. Proceed with F20 as originally designed
4. No additional work needed

**Timeline impact**:
- Alt 1: No change to F20 timeline
- Alt 3: +3 hours (~10% increase to F20 scope)
