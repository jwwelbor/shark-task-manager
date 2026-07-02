# Test Design Techniques Reference

Use this file when Step 5.5 of `workflows/test-planning.md` needs the full ISTQB and adversarial-technique catalog.

## Core techniques

| Technique | When to apply | What it forces you to enumerate |
|---|---|---|
| **Equivalence Partitioning** | AC accepts a range or set of inputs and treats them the same way | One test per equivalence class: valid set plus each invalid set |
| **Boundary Value Analysis (BVA)** | AC has numeric ranges, sizes, lengths, dates, or any ordered domain | Min, min-1, min, min+1, max-1, max, max+1; empty, null, zero, negative-one for collections |
| **Decision Tables** | AC has multiple conditions combining to determine behavior | Every reachable combination of conditions and resulting actions |
| **State Transitions** | AC describes an entity moving between states | Every legal transition plus every illegal transition |

## Adversarial and contract-focused techniques

| Technique | When to apply | What it forces |
|---|---|---|
| **Attack-class enumeration** | AC asserts a defensive property such as "immutable", "secure", or "isolated" | Enumerate mutation, injection, bypass, and exhaustion classes; each class yields at least one test |
| **Contract surface enumeration** | AC asserts an interaction with another component | Enumerate every public method, input class, and output assertion |

## Technique application rule

For each AC, declare which technique or techniques were applied and list the resulting test cases. ACs without a technique annotation are treated as untestable and must return to refinement.

## Example annotation

```markdown
### AC-T2: SourceBlock and SourcePage instances are immutable

**Techniques applied:**
- Attack-class enumeration: mutation via direct assignment, collection methods, subclass coercion, nested mutation
- BVA: zero-element collections, single-element, large collections, nested-depth boundary

**Test cases generated:** TC-T2-01 through TC-T2-12
```

If a technique would produce an unbounded set, pick representative members of each class and document the rationale.
