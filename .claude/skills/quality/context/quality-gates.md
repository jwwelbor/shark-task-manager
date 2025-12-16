# Quality Gates

This document defines general quality standards and gates that apply across all validation and review activities.

## General Quality Principles

### Completeness
- All required sections/files present
- No placeholder content (TODO, TBD)
- Comprehensive coverage of requirements
- All edge cases considered

### Specificity
- Clear, measurable criteria
- Concrete examples where helpful
- Avoid vague language
- Quantify where possible

### Consistency
- Terminology used uniformly
- Cross-references accurate
- Style consistent throughout
- Naming conventions followed

### Actionability
- Clear pass/fail thresholds
- Specific feedback for failures
- Recommended fixes included
- Next steps明确

## Document Quality Gates

### Design Documents
- [ ] All required files exist
- [ ] All required sections complete
- [ ] No code implementation (descriptions only)
- [ ] Mermaid diagrams where required
- [ ] Cross-references valid
- [ ] Length within guidelines (±20%)
- [ ] No placeholders (TODO, TBD)

### PRPs
- [ ] Valid YAML frontmatter
- [ ] All required sections present
- [ ] High-level directives (no code)
- [ ] Design doc references included
- [ ] Success criteria specific and measurable
- [ ] Valid dependency chain
- [ ] Appropriate agent assignment
- [ ] Length reasonable (50-100 lines)

### Code
- [ ] PRD requirements implemented
- [ ] No DRY violations
- [ ] SOLID principles followed
- [ ] Standards compliant
- [ ] Adequate test coverage
- [ ] Performance acceptable
- [ ] Security considered
- [ ] Idiomatic language usage

## Validation Report Standards

All validation reports must include:

### Summary Section
- Total items validated
- Items passing/failing
- Overall status (PASS/WARNING/FAIL)
- Quick summary of key issues

### Detailed Results
- Item-by-item breakdown
- Specific issues with locations
- Evidence (quotes, line numbers)
- Severity levels (blocker/major/minor)

### Recommendations
- Ordered by priority (blockers first)
- Specific, actionable fixes
- Next steps clearly stated

### Status Determination
- Clear ready/not ready decision
- Prerequisites for next phase
- Estimated time to resolve issues

## Pass/Fail Thresholds

### PASS ✅
- All critical criteria met
- No blocking issues
- Minor issues acceptable
- Ready to proceed

### PASS WITH WARNINGS ⚠️
- Critical criteria met
- Some non-blocking issues present
- Recommended to fix but not required
- Can proceed with caution

### FAIL ❌
- Critical criteria not met
- Blocking issues present
- Cannot proceed until fixed
- Specific fixes required

## Issue Severity Definitions

### Blocker (Must Fix)
- Prevents system from working
- Violates critical requirements
- Security or data integrity risks
- Major standards violations

### Major (Should Fix)
- Significantly impacts quality
- Makes maintenance difficult
- Non-compliant with standards
- Missing important requirements

### Minor (Nice to Fix)
- Minor quality improvements
- Style inconsistencies
- Optional enhancements
- Documentation improvements

## Anti-Patterns

Validation should flag these common anti-patterns:

### Documentation Anti-Patterns
- Code implementation in design docs
- Placeholder content (TODO, TBD)
- Missing diagrams where required
- Broken cross-references
- Duplicate content across files

### PRP Anti-Patterns
- Step-by-step code tutorials
- Code samples (SQL, Python, etc.)
- Duplicating design doc content
- Circular dependencies
- Vague success criteria
- Missing validation gates

### Code Anti-Patterns
- Duplicated logic (DRY violation)
- God classes (SRP violation)
- Tight coupling
- Speculative features (YAGNI)
- Non-idiomatic patterns
- Missing error handling

## Validation Consistency

All validations should:
- Use consistent terminology
- Apply same criteria uniformly
- Generate similarly formatted reports
- Provide comparable level of detail

## Feedback Quality

All feedback should be:
- **Objective**: Based on measurable criteria
- **Specific**: Reference exact locations
- **Actionable**: Provide clear fixes
- **Constructive**: Focus on improvement
- **Evidence-Based**: Include quotes/examples

## Next Steps Clarity

Every validation report should clearly state:
- Overall ready/not ready status
- What must be fixed (blockers)
- What should be fixed (major issues)
- What could be fixed (minor issues)
- Next phase or action
- How to re-validate

## Quality Improvement

Validation feedback helps improve:
- Documentation quality
- Code quality
- Process adherence
- Standards compliance
- Team knowledge

Use validation results to:
- Identify patterns in issues
- Update standards/templates
- Improve training/documentation
- Refine validation criteria
