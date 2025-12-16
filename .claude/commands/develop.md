---
description: Start development workflow from developer-ready package
---

# Start Development Workflow

Initiate the Development workflow to implement features using TDD, create tests, write code, and prepare pull requests.

## Usage

```bash
/develop
```

Or specify a feature branch name:

```bash
/develop feature/user-authentication
```

## What This Does

This command:
1. Initializes the Development workflow state
2. Launches the TechLead agent to review the developer-ready package
3. Creates a feature branch and sets up tracking
4. Follows TDD workflow:
   - Write tests first (unit and integration)
   - Implement code to pass tests
   - Code review and quality gates
   - QA testing and architecture review
   - Create pull request

## Workflow Integration

**Workflow Graph**: Development (06-development.csv)
**Entry Node**: Dev_Package_Review
**First Agent**: TechLead
**Skills Used**: development, tdd, quality, devops, code-review

## Prerequisites

**Required artifacts** (must exist before running):
- F-developer-ready-package/* - Complete package from Feature-Refinement workflow
  - F24-dev-package.md
  - S-refined-stories.md
  - T-api-contracts.md
  - T-data-models.md
  - T-flow-diagrams.md
  - F20-test-criteria.md
  - F21-edge-cases.md
  - P-prototypes/*

**Environment requirements**:
- Git repository initialized
- Main/master branch exists
- Development environment set up

## Implementation

The command:
1. Updates `docs/workflow/state.json` with Development workflow state
2. Sets current_node to "Dev_Package_Review"
3. Invokes the TechLead agent to verify package completeness
4. Hooks automatically advance through TDD cycle and quality gates

## Workflow Stages

### Stage 1: Package Review and Setup
- **Dev_Package_Review** (TechLead): Verify developer-ready package, clarify ambiguities
- **Create_Feature_Branch** (Developer): Create feature branch from main, set up story tracking

### Stage 2: Test Writing (Parallel)
- **Write_Unit_Tests** (Developer): Write unit tests covering all acceptance criteria
- **Write_Integration_Tests** (Developer): Write integration tests simulating real workflows

### Stage 3: Test Review and Commit
- **Test_Review** (TechLead): Ensure tests are meaningful, test real functionality (not mocks)
- **Commit_Tests** (Developer): Atomic commit containing only test code

### Stage 4: Implementation
- **Implement_Feature** (Developer): Write production code to make tests pass
- **Code_Review** (TechLead): Review for standards compliance, verify implementation matches plan

### Stage 5: Implementation Commit
- **Commit_Implementation** (Developer): Atomic commit containing implementation code

### Stage 6: Quality Validation (Parallel)
- **QA_Testing** (QA): Run full test suite, exploratory testing, validate acceptance criteria
- **Design_Compliance_Review** (Architect): Verify implementation adheres to architecture

### Stage 7: Verification and Approval
- **Verification_Gate** (TechLead): Consolidate reviews, route failures appropriately
- **Commit_Final** (Developer): Commit any final fixes from verification
- **Human_Approval** (Human): Final sign-off before creating PR

### Stage 8: Pull Request
- **Push_And_Create_PR** (Developer): Push feature branch, create PR with description
- **Development_Complete**: Terminal node, outputs PR URL and test results

## Output Artifacts

### Package Review
- DEV00-package-verified.md
- DEV01-clarifications.md

### Branch and Tracking
- DEV02-branch-created.md
- DEV03-tracking-setup.md

### Tests
- DEV04-unit-tests/*
- DEV05-integration-tests/*
- DEV06-test-review.md
- DEV07-tests-committed.md

### Implementation
- DEV08-implementation/*
- DEV09-code-review.md
- DEV10-impl-committed.md

### Quality Validation
- DEV11-qa-results.md
- DEV12-exploratory-findings.md
- DEV13-arch-review.md
- DEV14-verification-result.md

### Final Approval and PR
- DEV15-final-committed.md
- DEV16-human-approval.md
- DEV17-pr-created.md
- DEV-pull-request-url.md

### Completion Package
- DEV-completed-features/* (Feature branch, PR, test results, QA report)

## TDD Workflow Pattern

The workflow enforces Test-Driven Development:

1. **Tests First**: Write unit and integration tests before any implementation
2. **Tests Fail**: Verify tests fail (proving they test something)
3. **Atomic Test Commit**: Commit tests separately for easy rollback
4. **Implement**: Write minimal code to make tests pass
5. **Tests Pass**: All tests green before proceeding
6. **Code Review**: Verify implementation matches specifications
7. **Atomic Implementation Commit**: Commit implementation separately
8. **Quality Gates**: QA and architecture validation

## Monitoring Progress

Check workflow status:
```bash
cat /home/jwwelbor/projects/ai-dev-team/docs/workflow/state.json
```

View test results:
```bash
cat /home/jwwelbor/projects/ai-dev-team/docs/workflow/artifacts/DEV11-qa-results.md
```

Check feature branch:
```bash
git branch --list feature/*
```

## Skills Used

**Primary Skills**:
- `development` - Code implementation, git operations
- `tdd` - Test-first development workflow
- `quality` - Code review, testing, validation
- `devops` - Branch management, CI/CD integration
- `code-review` - Standards compliance verification
- `architecture` - Design compliance review

## Example Session

```
User: /develop feature/multi-factor-auth

System: [Initializes Development workflow]
System: [Launches TechLead agent]

TechLead: Reviewing developer-ready package for completeness...
[Reads F-developer-ready-package/*]
Package verified. All specifications and stories are clear.
[Creates DEV00-package-verified.md]

System: [Launches Developer agent]

Developer: Creating feature branch 'feature/multi-factor-auth' from main...
[Executes: git checkout -b feature/multi-factor-auth]
[Creates DEV02-branch-created.md, DEV03-tracking-setup.md]

System: [Launches parallel test writing]

Developer: [Write_Unit_Tests]
Writing unit tests for MFA service...
[Creates DEV04-unit-tests/test_mfa_service.py]
- test_generate_totp_secret()
- test_verify_totp_code_valid()
- test_verify_totp_code_invalid()
- test_generate_backup_codes()

Developer: [Write_Integration_Tests]
Writing integration tests for MFA flow...
[Creates DEV05-integration-tests/test_mfa_flow.py]
- test_enable_mfa_complete_flow()
- test_login_with_mfa()
- test_backup_code_usage()

System: [Both test tasks complete, launches TechLead]

TechLead: [Test_Review]
Reviewing tests for quality...
✓ Tests verify real behavior (not mock behavior)
✓ Tests cover all acceptance criteria
✓ Integration tests use real components
[Creates DEV06-test-review.md]

Developer: [Commit_Tests]
[Executes: git add tests/ && git commit -m "Add MFA tests"]
[Creates DEV07-tests-committed.md]

Developer: [Implement_Feature]
Running tests... ALL TESTS FAILING (expected)
Implementing MFA service to make tests pass...
[Writes code in src/services/mfa_service.py]
Running tests... ALL TESTS PASSING
[Creates DEV08-implementation/*]

TechLead: [Code_Review]
Reviewing implementation against API contracts...
✓ Follows coding standards
✓ Matches API contract specifications
✓ Proper error handling
✓ Security best practices applied
[Creates DEV09-code-review.md]

Developer: [Commit_Implementation]
[Executes: git add src/ && git commit -m "Implement MFA service"]
[Creates DEV10-impl-committed.md]

System: [Launches parallel quality validation]

QA: [QA_Testing]
Running test suite...
✓ All 47 tests passing
Performing exploratory testing...
✓ Acceptance criteria validated
[Creates DEV11-qa-results.md, DEV12-exploratory-findings.md]

Architect: [Design_Compliance_Review]
Verifying architecture compliance...
✓ Follows security architecture patterns
✓ Data model matches specifications
✓ API contracts implemented correctly
[Creates DEV13-arch-review.md]

TechLead: [Verification_Gate]
All quality gates passed. Ready for final approval.
[Creates DEV14-verification-result.md]

Developer: [Commit_Final]
No fixes needed. Proceeding to approval.
[Creates DEV15-final-committed.md]