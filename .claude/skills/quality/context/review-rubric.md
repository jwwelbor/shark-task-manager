# Code Review Rubric

This document defines the scoring rubric and standards for code reviews.

## Quality Scoring Rubric (0-5 Scale)

Apply this rubric to each modified file:

### (a) Readability (0-5)

**5 - Excellent**
- Self-documenting code with clear naming
- Logical flow that's easy to follow
- Comments only where truly necessary
- Consistent style throughout

**4 - Good**
- Generally clear with minor issues
- Mostly self-documenting
- Few unnecessary comments

**3 - Acceptable**
- Understandable with some effort
- Some confusing sections
- Adequate but not optimal naming

**2 - Needs Improvement**
- Hard to follow in places
- Poor naming conventions
- Missing necessary comments

**1 - Poor**
- Very difficult to understand
- Cryptic naming
- Confusing structure

**0 - Unacceptable**
- Impossible to understand without author explanation

### (b) Maintainability (0-5)

**5 - Excellent**
- Single Responsibility Principle followed
- Low coupling, high cohesion
- Easy to modify or extend
- Clear separation of concerns

**4 - Good**
- Generally well-structured
- Some minor coupling issues

**3 - Acceptable**
- Modifiable with moderate effort
- Some tight coupling present

**2 - Needs Improvement**
- Difficult to modify
- Tight coupling between components

**1 - Poor**
- Very hard to change
- High coupling, low cohesion

**0 - Unacceptable**
- Cannot be safely modified

### (c) Performance (0-5)

**5 - Excellent**
- Optimal algorithms and data structures
- No unnecessary operations
- Efficient resource usage

**4 - Good**
- Generally efficient
- Minor optimization opportunities

**3 - Acceptable**
- Adequate performance
- Some inefficiencies acceptable for clarity

**2 - Needs Improvement**
- Notable inefficiencies
- Suboptimal algorithms

**1 - Poor**
- Significant performance issues
- Wasteful resource usage

**0 - Unacceptable**
- Critical performance problems

### (d) Testability (0-5)

**5 - Excellent**
- Pure functions where possible
- Clear interfaces
- Easy to mock/stub dependencies
- Comprehensive test coverage

**4 - Good**
- Generally testable
- Minor testing challenges

**3 - Acceptable**
- Testable with moderate effort
- Some dependencies hard to mock

**2 - Needs Improvement**
- Difficult to test
- Tight coupling to dependencies

**1 - Poor**
- Very hard to test
- Major testing challenges

**0 - Unacceptable**
- Effectively untestable

### (e) Standards Compliance (0-5)

**5 - Excellent**
- Fully compliant with all project standards
- Idiomatic language usage
- Follows architecture patterns

**4 - Good**
- Mostly compliant
- Minor deviations

**3 - Acceptable**
- Generally compliant
- Some notable deviations

**2 - Needs Improvement**
- Multiple standards violations
- Non-idiomatic code

**1 - Poor**
- Major standards violations
- Inconsistent with codebase

**0 - Unacceptable**
- Completely non-compliant

## Review Principles

### Non-Negotiable Standards

**DRY (Don't Repeat Yourself)**
- No duplicated logic
- Search codebase for existing implementations
- Reuse utilities and modules
- Propose refactors with specific file paths

**SOLID Principles**
- **S**ingle Responsibility: Each unit does one thing
- **O**pen/Closed: Open for extension, closed for modification
- **L**iskov Substitution: Subtypes must be substitutable
- **I**nterface Segregation: Many specific interfaces better than one general
- **D**ependency Inversion: Depend on abstractions, not concretions

**YAGNI (You Aren't Gonna Need It)**
- Only implement PRD requirements
- No speculative features
- Flag out-of-scope work

**Principle of Least Surprise**
- Code should be obvious
- Consistent with project idioms
- No unexpected behavior

**Language Best Practices**
- Follow idiomatic patterns
- Use standard libraries
- Community-standard approaches

## Size & Style Guidelines

These are guides, not hard rules:

- **Files**: Ideally < 500 lines
- **Functions**: Ideally < 50 lines
- **Classes**: Focused responsibilities
- **Comments**: Only where non-obvious
- **Docstrings**: Proportional to code (avoid longer than code itself)
- **Trivial Functions**: Avoid 1-2 line functions unless justified:
  - Interface conformity
  - Test seam
  - Clarity improvement

## Issue Severity Levels

### Blocker
- Must be fixed before merge
- Examples:
  - Security vulnerabilities
  - Data loss risks
  - Critical performance issues
  - PRD requirement not implemented
  - Major SOLID violations

### Major
- Should be fixed before merge
- Can be addressed in follow-up if time-critical
- Examples:
  - DRY violations
  - Non-idiomatic code
  - Missing tests
  - Standards deviations
  - Maintainability issues

### Minor
- Nice to have fixes
- Can be addressed in follow-up
- Examples:
  - Style inconsistencies
  - Minor optimization opportunities
  - Documentation improvements
  - Naming suggestions

## Language-Specific Standards

### Python
- PEP 8 compliance
- Type hints for public APIs
- Pythonic patterns (list comprehensions, context managers)
- Use standard library (itertools, functools, etc.)

### TypeScript
- Strict mode enabled
- Explicit types (no `any`)
- Functional patterns where appropriate
- Use built-in utility types

### JavaScript
- Modern ES6+ patterns
- Avoid var, use const/let
- Arrow functions where appropriate
- Promise-based async patterns

### Go
- gofmt compliance
- Idiomatic error handling
- Goroutine safety
- Interface-based design

## Standards References

Reviews should cite:
- `/docs/architecture/system-design-*.md` - Architecture patterns
- `/docs/architecture/coding-standards.md` - Coding standards
- PRD requirements - Feature specifications
- Language style guides - Official language standards

## Review Output Requirements

Every review must include:

1. **Executive Summary**
   - What PR accomplishes
   - PRD fitness
   - Risk level
   - Ship/no-ship recommendation

2. **Findings Table**
   - All issues with severity, location, rule violated

3. **Reuse Opportunities**
   - Existing code that can replace new code

4. **Standards Crosswalk**
   - Map issues to violated standards

5. **Quality Scores**
   - Score each file on rubric dimensions

6. **Action Plan**
   - Ordered fixes (blockers first)
   - Specific, actionable steps
