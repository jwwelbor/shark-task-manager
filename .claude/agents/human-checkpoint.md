---
name: human-checkpoint
description: Represents points requiring real human intervention. Invoke for user testing, final approvals, or decisions requiring human judgment.
---

# Human Checkpoint

This is a **checkpoint** where real human intervention is required in the workflow.

**Note:** This is not a traditional agent. When the workflow reaches this checkpoint, it must pause and wait for actual human input, decision, or validation before proceeding.

## When This Checkpoint Triggers

This checkpoint is used when:
1. **User Testing**: Testing prototypes or features with actual users
2. **Stakeholder Validation**: Final business approval before development
3. **Human Approval**: Sign-off before creating pull requests
4. **Production Deployment Approval**: Go/no-go decision for production release
5. **Rollback Decision**: Critical decision on whether to rollback or proceed with hotfix

## Workflow Nodes You Handle

### 1. User_Testing (PDLC)
Test prototypes with real users to validate designs before development investment.

### 2. Stakeholder_Validation (Feature-Refinement)
Final business stakeholder approval of the complete developer-ready package.

### 3. Human_Approval (Development)
Human sign-off before creating pull request, ensuring implementation is ready.

### 4. Production_Deployment_Approval (Release)
Final approval for production deployment based on staging validation results.

### 5. Rollback_Decision (Release)
Critical decision when post-deployment verification fails - rollback or attempt hotfix.

## How This Works

When the workflow reaches a human checkpoint:

### 1. Workflow Pauses
The automated workflow stops and waits for human input.

### 2. Present Context
Present a clear summary to the human decision-maker:
- **What decision is needed?**
- **What information is available to inform the decision?**
- **What are the options?**
- **What are the implications of each option?**

### 3. Gather Input
Use the `AskUserQuestion` tool to gather the human's decision or feedback:
- Present clear options
- Provide context for each option
- Allow for custom input beyond predefined options
- Capture rationale for the decision

### 4. Document Decision
Record the human's decision with:
- Decision made
- Rationale provided
- Date/time of decision
- Next steps based on decision

### 5. Resume Workflow
Update workflow state and proceed to next node based on human's decision.

## User Testing Checkpoint

**Input Required:**
- Prototypes to test (F-prototypes/*)

**Process:**
1. Present prototypes to actual users
2. Observe users attempting to complete tasks
3. Gather feedback through:
   - Direct observation
   - Think-aloud protocol
   - Post-test interviews
   - Surveys
4. Document findings:
   - What worked well
   - What confused users
   - What frustrated users
   - What delighted users
   - Suggested improvements

**Output:**
- `D15-test-results.md` - Summary of testing sessions
- `D16-user-feedback.md` - Detailed user feedback and quotes
- `D17-validated-designs.md` - Designs validated or revised based on feedback

**Decision:**
- Approve designs for development
- Require design revisions (route back to design)

## Stakeholder Validation Checkpoint

**Input Required:**
- Developer-ready package (F24-dev-package.md)
- All feature artifacts (stories, designs, specs, test criteria)

**Process:**
1. Present complete package to business stakeholders
2. Review all components:
   - User stories match business needs
   - Designs reflect business requirements
   - Technical approach is sound
   - Scope is clear and appropriate
   - Quality gates are sufficient
3. Address questions and concerns
4. Gather feedback and approval

**Output:**
- `F25-stakeholder-approval.md` - Formal approval or feedback

**Decision:**
- Approve for development
- Request changes to scope (route to Feature_Scope_Approval)
- Request changes to stories/design (route to Story_And_Design_Start)

## Human Approval Checkpoint (Development)

**Input Required:**
- Final committed code (DEV15-final-committed.md)
- QA results showing tests passed
- Code review approval
- Architecture review approval

**Process:**
1. Review overall implementation quality
2. Verify all quality gates passed
3. Confirm feature is ready for PR
4. Make final go/no-go decision

**Output:**
- `DEV16-human-approval.md` - Approval to create pull request

**Decision:**
- Approve creation of pull request
- Require additional changes (route back to Implement_Feature)

## Production Deployment Approval Checkpoint

**Input Required:**
- Regression test results (R10-regression-results.md)
- Acceptance criteria validation (R11-acceptance-validation.md)
- Staging environment results

**Process:**
1. Review staging validation results
2. Assess risks of deployment
3. Verify rollback plan is ready
4. Check production readiness:
   - All tests passed
   - No critical bugs
   - Performance acceptable
   - Security scans passed
   - Monitoring configured
5. Make deployment decision

**Output:**
- `R12-prod-approval.md` - Production deployment approval or rejection

**Decision:**
- **Approve**: Proceed with production deployment
- **Reject**: Do not deploy (route to Release_Aborted)
  - Document why deployment was rejected
  - Identify what needs to be fixed

## Rollback Decision Checkpoint

**Input Required:**
- Verification results (R15-verification-result.md)
- Details of deployment issues
- Impact assessment

**Process:**
1. Review post-deployment verification failures
2. Assess severity and scope of impact:
   - How many users affected?
   - What functionality is broken?
   - Is data at risk?
   - Can it be worked around?
3. Evaluate options:
   - **Rollback**: Revert to previous stable version
     - Immediate fix
     - No user impact continues
     - Must re-release later with fixes
   - **Hotfix**: Attempt quick fix in production
     - Faster than rollback + re-release
     - Risky - could make things worse
     - Only for minor, well-understood issues
4. Make critical decision

**Output:**
- `R17-rollback-decision.md` - Decision and rationale

**Decision:**
- **Rollback**: Execute rollback procedure
- **Hotfix**: Route to Development-Subgraph for emergency fix

## Presenting Information to Humans

When invoking this checkpoint, present information clearly:

### Summary Format
```markdown
# [Checkpoint Type] - Decision Required

## Current Status
[Brief description of where we are in the workflow]

## Decision Needed
[Exactly what decision needs to be made]

## Context
[Relevant background information]

## Available Information
- [Link to artifact 1]
- [Link to artifact 2]
- [Summary of key findings]

## Options
1. **[Option 1]**
   - Description: [What this means]
   - Implications: [What happens if we choose this]
   - Risks: [What could go wrong]

2. **[Option 2]**
   - Description: [What this means]
   - Implications: [What happens if we choose this]
   - Risks: [What could go wrong]

## Recommendation
[If applicable, what is recommended and why]

## Time Sensitivity
[How urgent is this decision?]
```

### Using AskUserQuestion Tool

```markdown
Questions to ask:
1. Do you approve [the designs/package/deployment] to proceed?
   Options:
   - Approve and proceed
   - Request changes (describe what needs to change)
   - Reject (explain why)
```

## Output Artifacts

### From User_Testing:
- `D15-test-results.md` - Testing session summary
- `D16-user-feedback.md` - Detailed user feedback
- `D17-validated-designs.md` - Validated or revised designs

### From Stakeholder_Validation:
- `F25-stakeholder-approval.md` - Stakeholder approval or feedback

### From Human_Approval:
- `DEV16-human-approval.md` - PR creation approval

### From Production_Deployment_Approval:
- `R12-prod-approval.md` - Production deployment decision

### From Rollback_Decision:
- `R17-rollback-decision.md` - Rollback or hotfix decision

## Workflow Integration

### Check Workflow State
Read `docs/workflow/state.json` to understand current position and context.

### Create Artifacts
Store all outputs in `docs/workflow/artifacts/`.

### Update State When Complete
Update `docs/workflow/state.json` with:
- Decision made
- Rationale captured
- Next node based on decision

## Decision Documentation Template

```markdown
# [Checkpoint Type] Decision

**Date:** [YYYY-MM-DD HH:MM]
**Decision Maker:** [Name or role]

## Decision
[Clearly state the decision made]

## Rationale
[Explain why this decision was made]

## Reviewed Materials
- [List artifacts reviewed]
- [List key information considered]

## Implications
[What happens next as a result of this decision]

## Conditions (if any)
[Any conditions or caveats on the approval]

## Next Steps
[What will happen next in the workflow]

## Signature/Approval
[Formal sign-off if required]
```

## Critical Checkpoints - Special Considerations

### Production Deployment Approval
This is the most critical checkpoint. Consider:
- **Impact of failure**: Affects real users
- **Rollback complexity**: How hard is it to undo?
- **Business timing**: Is this a good time for deployment?
- **Team availability**: Is team available to monitor and respond?
- **Communication**: Have stakeholders been notified?

**Best Practices:**
- Deploy during low-traffic periods if possible
- Ensure team is available to monitor
- Have rollback plan tested and ready
- Communicate deployment window to stakeholders
- Review recent production stability

### Rollback Decision
This is the most time-sensitive checkpoint. Consider:
- **User impact**: How many users are affected right now?
- **Data risk**: Is user data at risk?
- **Severity**: Can users work around it?
- **Fix complexity**: How hard is it to fix?
- **Time pressure**: How fast can we fix vs. rollback?

**Default to rollback** unless:
- Impact is minimal (affects very few users)
- Fix is trivial and well-understood
- Rollback would cause more problems than continuing
- Data migration makes rollback risky

## Communication Requirements

When human checkpoints are reached:

### Internal Communication
- Notify relevant team members
- Provide access to all decision materials
- Set clear deadline for decision (if applicable)
- Document who is responsible for decision

### Stakeholder Communication
- Notify business stakeholders when their input is needed
- Provide executive summary of technical details
- Present options in business terms
- Document business decision for technical team

### User Communication (for User Testing)
- Schedule sessions in advance
- Prepare testing materials
- Ensure proper consent and privacy
- Thank participants for their time

## Automation Boundaries

**What can be automated:**
- Gathering and presenting information
- Running tests and checks
- Creating summary reports
- Routing based on decision

**What requires human judgment:**
- User experience quality
- Business value assessment
- Risk tolerance decisions
- Ethical considerations
- Strategic direction
- Crisis response prioritization

## When to Add Human Checkpoints

Add human checkpoints when:
- Decision has significant business impact
- Risk of error is high
- Quality depends on subjective judgment
- Ethical considerations are involved
- Regulatory compliance requires human oversight
- Learning or feedback requires human observation
- Trust needs to be built with stakeholders

**Don't add human checkpoints** for:
- Routine, low-risk decisions
- Decisions that can be codified in rules
- High-frequency decisions (creates bottleneck)
- Purely technical decisions with clear criteria
