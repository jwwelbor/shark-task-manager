# Acceptance Criteria Writing Rules

Use this file when `workflows/write-task.md` needs the full anti-pattern catalog and rewrite guidance for task acceptance criteria.

## Core rule

Acceptance criteria must be enumerable and verifiable in finite work. If the truth condition is effectively "for all possible adversarial inputs", narrow the scope or enumerate the model explicitly.

## Concrete versus open-ended

| Concrete AC (good) | Open-ended AC (bad) |
|---|---|
| "Manager.Load() parses `tag_required_for` from rawData into TagRequiredForTypes; returns nil only when key absent" | "Configuration parsing is robust" |
| "Tags row renders when `service.SetTagService` is called; verified by TC-005" | "Tag display works correctly" |
| "SourceBlock fields cannot be mutated through: attribute rebinding, collection mutation, dict assignment, nested mutation, mutable subclass coercion — TC-T2-01..12" | "SourceBlock instances are immutable" |
| "Token validation rejects tokens older than 1 hour: TC-009 asserts age=59m59s passes, age=60m1s rejects" | "Reset links expire after 1 hour" |

## Anti-patterns

- Robustness assertions without an attack model
- Quality vibes such as "high quality" or "clean code"
- Wishful adverbs such as "correctly" or "gracefully"
- Quantifierless universals such as "no SQL injection" without an enumerated surface

## Fix strategy

1. Enumerate the model.
2. Narrow the scope.
3. Defer to a referenced standard with specific control IDs or clauses.

## Required self-check

Before finalizing an AC, ask:

- Can a reviewer decide in finite work whether this is met?
- Can I write at least one concrete test case with exact input and expected output?
- Could another agent enumerate the cases to test without looping forever?
- Does the AC reference a file, line, function, contract, explicit input set, or explicit output rather than abstract quality words?

If not, rewrite it before returning.
