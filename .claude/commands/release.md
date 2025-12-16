---
description: Start release cycle workflow for production deployment
---

# Start Release Workflow

Initiate the Release workflow to plan releases, merge features, build artifacts, test in staging, and deploy to production.

## Usage

```bash
/release
```

Or specify a release version:

```bash
/release v2.1.0
```

## What This Does

This command:
1. Initializes the Release workflow state
2. Launches the ProductManager agent for release planning
3. Coordinates feature merging, building, and testing
4. Manages staging validation and production deployment
5. Includes rollback capabilities for failed deployments

## Workflow Integration

**Workflow Graph**: Release (08-release.csv)
**Entry Node**: Release_Planning
**First Agent**: ProductManager
**Skills Used**: specification, orchestration, devops, quality

## Prerequisites

**Required artifacts** (must exist before running):
- DEV-completed-features/* - Completed features with PRs merged
  - DEV17-pr-created.md
  - DEV-pull-request-url.md
  - Feature branches merged to main

**Infrastructure requirements**:
- INFRA12-staging-env.md - Staging environment configured
- INFRA06-deployment-strategy.md - Deployment strategy defined
- CI/CD pipeline configured
- Monitoring and alerting set up

## Implementation

The command:
1. Updates `/home/jwwelbor/projects/ai-dev-team/docs/workflow/state.json` with Release workflow state
2. Sets current_node to "Release_Planning"
3. Invokes the ProductManager agent to select features and define scope
4. Hooks automatically advance through build, test, and deployment stages

## Workflow Stages

### Stage 1: Release Planning
- **Release_Planning** (ProductManager): Select features, define scope, coordinate with stakeholders, draft release notes

### Stage 2: Merge and Test
- **Merge_Features** (TechLead): Merge feature branches to release branch, run full test suite post-merge

### Stage 3: Build
- **Build_Release** (DevOps): Build release artifacts, create version tag, package for deployment

### Stage 4: Staging Deployment
- **Deploy_To_Staging** (DevOps): Deploy to staging environment, run automated smoke tests

### Stage 5: Staging Validation
- **Staging_Validation** (QA): Execute full regression test suite, validate acceptance criteria against staging

### Stage 6: Production Approval
- **Production_Deployment_Approval** (Human): Final business approval for production deployment

### Stage 7: Production Deployment
- **Deploy_To_Production** (DevOps): Execute deployment strategy (blue-green/canary/rolling), activate monitoring

### Stage 8: Post-Deploy Verification
- **Post_Deploy_Verification** (DevOps): Verify deployment, health check, validate monitoring

### Stage 9: Rollback (if needed)
- **Rollback_Decision** (Human): Assess failure severity, decide rollback or hotfix
- **Rollback_Deployment** (DevOps): Revert to previous stable release
- **Release_Aborted**: Terminal node for cancelled/rolled back releases

### Stage 10: Success
- **Release_Complete**: Terminal node, successful production deployment

## Output Artifacts

### Planning
- R01-release-scope.md - Features included in this release
- R02-release-features.md - Detailed feature list
- R03-release-notes-draft.md - Draft release notes for stakeholders

### Merge and Build
- R04-merge-result.md - Feature branch merge results
- R05-test-suite-results.md - Post-merge test results
- R06-release-artifacts/* - Built release artifacts (binaries, packages, containers)
- R07-version-tag.md - Git version tag information

### Staging
- R08-staging-deploy.md - Staging deployment details
- R09-smoke-tests.md - Smoke test results

### Validation
- R10-regression-results.md - Regression test suite results
- R11-acceptance-validation.md - Acceptance criteria validation against staging

### Production
- R12-prod-approval.md - Production deployment approval
- R13-prod-deploy.md - Production deployment details
- R14-monitoring-active.md - Monitoring and alerting confirmation

### Verification
- R15-verification-result.md - Post-deploy verification results
- R16-health-check.md - Production health check results

### Rollback (if applicable)
- R17-rollback-decision.md - Rollback decision rationale
- R18-rollback-result.md - Rollback execution results
- R-aborted-report.md - Release abort report

### Success
- R-production-release.md - Successful release report
- R-release-notes.md - Final release notes
- R-deployment-report.md - Complete deployment report

## Deployment Strategies

The workflow supports multiple deployment strategies:

### Blue-Green Deployment
- Deploy to "green" environment
- Switch traffic from "blue" to "green"
- Keep "blue" as instant rollback target

### Canary Deployment
- Deploy to small subset of servers
- Monitor metrics
- Gradually roll out to remaining servers
- Rollback if issues detected

### Rolling Deployment
- Deploy to servers in batches
- Monitor each batch
- Continue or halt based on health

Strategy is defined in INFRA06-deployment-strategy.md.

## Monitoring Progress

Check workflow status:
```bash
cat /home/jwwelbor/projects/ai-dev-team/docs/workflow/state.json
```

View release artifacts:
```bash
ls /home/jwwelbor/projects/ai-dev-team/docs/workflow/artifacts/R06-release-artifacts/
```

Check deployment status:
```bash
cat /home/jwwelbor/projects/ai-dev-team/docs/workflow/artifacts/R15-verification-result.md
```

## Skills Used

**Primary Skills**:
- `specification` - Release planning and notes
- `orchestration` - Workflow coordination across teams
- `devops` - Build, deployment, infrastructure
- `quality` - Testing, validation, verification

## Failure Handling

### Merge Failures
If test suite fails after merge:
- Routes back to Development-Subgraph
- Failed tests are fixed
- Re-merge attempted

### Staging Validation Failures
If regression tests fail in staging:
- Routes back to Development-Subgraph
- Issues are resolved
- Re-deploy to staging

### Production Deployment Failures
If post-deploy verification fails:
- Human decision point (Rollback_Decision)
- Options:
  1. Rollback to previous version (safe, fast)
  2. Hotfix in Development-Subgraph (if minor issue)
- Rollback executes automatically if chosen

## Example Session

```
User: /release v2.1.0

System: [Initializes Release workflow]
System: [Launches ProductManager agent]

ProductManager: [Release_Planning]
Reviewing completed features for release v2.1.0...
[Reads DEV-completed-features/*]

Features selected:
1. Multi-factor Authentication (DEV-mfa)
2. Password Reset Flow (DEV-password-reset)
3. Email Verification (DEV-email-verify)

Creating release scope and draft release notes...
[Creates R01-release-scope.md, R02-release-features.md, R03-release-notes-draft.md]

System: [Launches TechLead]

TechLead: [Merge_Features]
Merging feature branches to release/v2.1.0...
[Executes: git merge feature/mfa feature/password-reset feature/email-verify]
Merge successful.

Running full test suite post-merge...
[Executes: npm test]
✓ 247 tests passing
✓ 0 tests failing
[Creates R04-merge-result.md, R05-test-suite-results.md]

System: [Launches DevOps]

DevOps: [Build_Release]
Building release artifacts...
[Executes: npm run build]
Creating version tag v2.1.0...
[Executes: git tag v2.1.0]
Packaging for deployment...
[Creates R06-release-artifacts/*, R07-version-tag.md]

DevOps: [Deploy_To_Staging]
Deploying v2.1.0 to staging environment...
[Executes deployment scripts]
Running automated smoke tests...
✓ Health check: PASS
✓ Critical paths: PASS
✓ API endpoints: PASS
[Creates R08-staging-deploy.md, R09-smoke-tests.md]

System: [Launches QA]

QA: [Staging_Validation]
Executing full regression test suite against staging...
✓ Regression suite: 312/312 tests passing
Validating acceptance criteria for all features:
✓ MFA: All acceptance criteria met
✓ Password Reset: All acceptance criteria met
✓ Email Verification: All acceptance criteria met
[Creates R10-regression-results.md, R11-acceptance-validation.md]

System: [Human approval required]