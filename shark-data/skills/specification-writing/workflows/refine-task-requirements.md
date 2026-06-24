---
inputs:
  - task_id: opaque task identifier (string)
  - task_title: task title (string)
  - task_description: task description as captured in shark (string)
  - task_acceptance_criteria: existing AC list (may be empty / partial — that's why refinement is needed)
  - blocker_notes: list of {type, message, created_by, created_at} — especially relevant: blocker notes from developers/QA explaining what's unclear
  - feature_id: opaque feature identifier (string)
  - feature_prd_path: absolute path to the feature PRD markdown
  - epic_doc_paths: object with epic-level doc paths (optional, for upward context)
  - related_doc_paths: list of {label, path} for documents already linked to the task / feature / epic
  - related_code_paths: list of {label, path} for source files relevant to the task (optional)
  - refinement_role: "ba" | "tech" — which refinement perspective is being applied
  - design_doc_paths: object — `{architecture, database, api_spec, security_performance}` — for the architect role; values may be null
outputs:
  - updated_documents: list of {path, sections_modified, kind: "prd" | "api_spec" | "architecture" | "database" | "adr" | "other"}
  - decisions_log: list of {decision, rationale, doc_referenced} — particularly important for architect's ADR-style decisions
  - business_rules_added: list of {rule, source_doc} — for BA refinement
  - api_contracts_defined: list of {endpoint, method, source_doc} — for tech refinement
  - data_models_defined: list of {model_name, source_doc} — for tech refinement
  - open_questions: list of unresolved items needing user input (with recommended defaults and rationale)
  - contradictions_found: list of {description, source_a, source_b, requires_user_resolution: bool}
  - new_documents_created: list of {path, kind} — e.g., a fresh ADR file, design doc spawned during refinement
---

# Workflow: Refine Task Requirements (craft)

## Purpose

This workflow is for **Business Analysts** and **Architects** working on tasks that need requirements clarification or technical design. It applies when:

- BA needs to clarify requirements
- Architect needs to design solution
- Developer sent task back due to spec gaps

## When You Are Assigned a Task

### Step 1: Understand Why You Were Assigned

Read `blocker_notes` — especially entries with `type: "blocker"`. Example:

```json
{
  "type": "blocker",
  "message": "Requirements unclear: How should password reset emails be sent?",
  "created_by": "developer",
  "created_at": "2025-01-12T10:30:00Z"
}
```

This tells you WHY you were assigned and WHAT needs clarification. Combine with `task_description` and `task_acceptance_criteria` to scope the refinement.

### Step 2: Read Related Context

For every entry in `related_doc_paths` and (if relevant) `related_code_paths`, read the file. Pay special attention to:

- The feature PRD (`feature_prd_path`) — what's already defined, what's missing, what's contradictory.
- Any prior task notes or ADRs — the design rationale already on record.
- Source code referenced by the task — to understand existing patterns rather than inventing new ones.

**Purpose:** understand the feature context, what's already defined, and what's missing.

### Step 3: Identify What Needs Refinement

Based on `refinement_role` and `blocker_notes`:

#### For Business Analysts (`refinement_role = "ba"`)

**Common scenarios:**

- Requirements are vague or incomplete
- User stories missing or unclear
- Acceptance criteria not defined
- Business rules not documented
- Edge cases not considered

**What to do:**

1. Review the feature PRD.
2. Identify gaps in requirements.
3. Ask the user for clarification if needed.
4. Update user stories in the PRD.
5. Define clear acceptance criteria.
6. Document business rules and edge cases.

#### For Architects (`refinement_role = "tech"`)

**Common scenarios:**

- Technical approach not defined
- API contracts missing
- Data models not specified
- Integration patterns unclear
- Security/performance considerations not addressed

**What to do:**

1. Review business requirements from BA.
2. Design technical solution.
3. Update the API specification doc with API contracts (single source of truth — never duplicate contracts elsewhere).
4. Document architecture decisions in an ADR or the architecture doc.
5. Define data models in the database design doc.
6. Address non-functional requirements in the security/performance doc.

### Step 4: Do Your Refinement Work

#### Business Analyst Refinement

**Update the feature PRD** at `feature_prd_path`. Update sections:

- User Stories
- Functional Requirements
- Acceptance Criteria
- Business Rules

**Add specific, testable criteria.** Compare:

**Bad (vague):**

```markdown
- User can reset password
```

**Good (specific):**

```markdown
### User Story: Password Reset
As a user who forgot my password
I want to reset it via email
So that I can regain access to my account

**Acceptance Criteria:**
- [ ] User can request password reset from login page
- [ ] System sends reset link to registered email within 2 minutes
- [ ] Reset link expires after 1 hour
- [ ] Reset link can only be used once
- [ ] User must create new password meeting requirements:
  - Minimum 8 characters
  - At least 1 uppercase, 1 lowercase, 1 number
- [ ] After successful reset, old password is invalid
- [ ] If email not found, show generic "check your email" message (security)
```

**Document business rules.** Example:

```markdown
## Business Rules

### Password Reset
1. Rate limiting: Max 3 reset requests per email per hour
2. Reset emails sent only to verified email addresses
3. Active reset links invalidated if user successfully resets password
4. Failed reset attempts (wrong link) logged for security monitoring
```

Capture each rule in `business_rules_added`.

#### Architect Refinement

**Update the architecture and API design docs in place** (paths in `design_doc_paths`). Create an ADR alongside if a decision warrants its own record.

**Define API contracts in the API specification doc.** The API spec is the single source of truth for contracts; never duplicate request/response schemas elsewhere. Example:

```markdown
## API Contract: Password Reset

### POST /api/auth/password-reset/request

**Request:**
```json
{
  "email": "user@example.com"
}
```

**Response (Success - 200):**
```json
{
  "message": "If that email exists, a reset link has been sent"
}
```

**Note:** Always return success even if email not found (security best practice).

### POST /api/auth/password-reset/confirm

**Request:**
```json
{
  "token": "abc123...",
  "newPassword": "SecureP@ss123"
}
```

**Response (Success - 200):**
```json
{
  "message": "Password reset successful",
  "userId": "user-uuid"
}
```

**Response (Invalid Token - 400):**
```json
{
  "error": "INVALID_RESET_TOKEN",
  "message": "Reset link is invalid or expired"
}
```
```

Capture endpoints in `api_contracts_defined`.

**Document data models** in the database design doc. Example:

```markdown
## Data Model: Password Reset Token

```typescript
interface PasswordResetToken {
  id: string;
  userId: string;
  token: string;  // hashed
  expiresAt: Date;
  used: boolean;
  createdAt: Date;
}
```
```

Capture in `data_models_defined`.

**Document technical decisions** as ADRs. Each ADR captures: decision, rationale, consequences. Example:

```markdown
## ADR: Password Reset Token Storage

**Decision:** Store reset tokens in database (not JWT)

**Rationale:**
- Allows one-time use enforcement
- Enables token invalidation
- Better security (can revoke if needed)

**Consequences:**
- Requires database table for tokens
- Need cleanup job for expired tokens
```

Capture each decision in `decisions_log`. If you spawn a fresh ADR file, add it to `new_documents_created`.

### Step 5: Compose the Refinement Summary

Return a concise summary of what was refined and the key decisions made. For BA refinement, lead with requirements clarifications; for architect refinement, lead with architecture decisions.

Example BA summary:

```
DONE: <task_id>

Summary:
- Updated PRD with detailed password reset requirements
- Defined acceptance criteria with specific constraints
- Documented business rules for rate limiting and security

Key decisions:
- 1-hour token expiry
- 3 requests/hour rate limit
- Generic response for security (don't reveal if email exists)
```

Example architect summary:

```
DONE: <task_id>

Summary:
- Updated API specification with password-reset contracts
- Documented data model in database design (reset tokens table)
- Created ADR for database-backed token storage decision

Key decisions:
- Database storage (not JWT) for one-time use enforcement
- Token hashing for security
- Separate request/confirm endpoints
```

## Error Handling

### Specification Contradiction

If you find contradictory specs (e.g., PRD says 24-hour expiry, security policy says 1-hour max), STOP and surface the contradiction. Add to `contradictions_found` with both source documents and `requires_user_resolution: true`.

### Missing Information (BA needs user input)

If you need user clarification, present a structured question with recommended defaults:

```markdown
I need clarification on password reset requirements:

1. What should the token expiry time be?
   - PRD doesn't specify
   - Industry standard is 1 hour
   - Recommendation: 1 hour

2. Should we rate-limit reset requests?
   - Not mentioned in requirements
   - Prevents abuse
   - Recommendation: 3 requests per hour per email

3. What happens if user clicks expired link?
   - Should we show specific error?
   - Or generic "invalid link" for security?
   - Recommendation: Generic message

Please confirm or provide guidance, then I'll update the PRD.
```

After the user responds, update the PRD and continue.

### Requirements Gap (Architect finds missing info)

If the BA didn't specify something critical (e.g., "Should admins be able to force-expire all reset tokens for a user?"), document the gap in `open_questions` and return a summary so the orchestrator can route back to BA. Do not invent business rules in the architecture pass — that's not the architect's authority.

## Quality Checklist

### Business Analyst Checklist

- [ ] User stories are specific and testable
- [ ] Acceptance criteria are measurable
- [ ] Business rules are documented
- [ ] Edge cases are considered
- [ ] Security implications noted
- [ ] No ambiguous language ("should be fast", "user-friendly")
- [ ] All blocker notes from developers addressed

### Architect Checklist

- [ ] API contracts fully defined (request/response schemas)
- [ ] Data models documented
- [ ] Architecture decisions recorded with rationale
- [ ] Security considerations addressed
- [ ] Performance requirements noted
- [ ] Integration points identified
- [ ] All blocker notes from developers addressed
- [ ] No undefined "TBD" or "TODO" in critical sections

## Common Mistakes

- **Not reading task notes** — miss blocker notes from developers; don't know WHY you were assigned; waste time on the wrong problems.
- **Vague specifications** — "User can reset password" (not testable); "System sends email quickly" (not measurable). Leads to more rework later.
- **Asking "should I continue?"** — you were assigned the task, that means yes. Just do the work. Only block if there's a contradiction or missing business decision.

## Remember

**This workflow is for task-level refinement**, not creating new features from scratch.

- For a **new feature**, use `write-feature-prd.md`.
- For a **new epic**, use `write-epic.md`.

This workflow is specifically for when a **task** needs requirements or technical design work.
