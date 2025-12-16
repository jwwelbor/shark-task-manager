# Security Architecture Design Workflow

This workflow guides you through creating comprehensive security design documentation for a feature. It produces the security design document (06-security-design.md) covering authentication, authorization, data protection, and compliance across all layers.

## Prerequisites

Before starting this workflow, ensure you have:
1. Feature PRD at `/docs/plan/{epic-key}/{feature-key}/prd.md`
2. Research report at `/docs/plan/{epic-key}/{feature-key}/00-research-report.md`
3. Backend design (04-backend-design.md) - API endpoints and DTOs
4. Frontend design (05-frontend-design.md) - UI components if applicable
5. Data design (03-data-design.md) - Data models and sensitive data

## Step 1: Analyze Security Requirements

### Read All Design Documents

Security is cross-cutting - analyze all layers:
- **Backend**: What APIs need protection? What data is exposed?
- **Frontend**: What client-side risks exist? What user data is handled?
- **Data**: What sensitive data is stored? What needs encryption?

### Identify Security Concerns

From PRD and design docs:
- Authentication requirements (who can access?)
- Authorization requirements (what can they do?)
- Data classification (PII, sensitive, internal, public)
- Compliance requirements (GDPR, HIPAA, SOC2, etc.)
- External integration security
- Input validation needs
- Audit/logging requirements

### Review Project Security Patterns

From research report:
- Existing authentication mechanism (JWT, OAuth, session)
- Authorization pattern (RBAC, ABAC, etc.)
- Encryption standards in use
- Security libraries/frameworks
- Existing security policies

## Step 2: Create Threat Model

### Identify Assets to Protect

Create a table:
| Asset | Classification | Impact if Compromised |
|-------|---------------|----------------------|
| User credentials | PII | Account takeover, data breach |
| Payment data | Sensitive/PCI | Financial loss, compliance violation |
| API keys | Internal | Unauthorized access |
| User content | Public/Private | Privacy violation |

**Classifications**:
- **PII**: Personally Identifiable Information
- **Sensitive**: Business-critical or regulated data
- **Internal**: Not for public access
- **Public**: Intentionally public data

### Identify Threat Actors

| Actor | Motivation | Capability Level |
|-------|------------|-----------------|
| Malicious user | Data theft, disruption | Low-Medium |
| Competitor | Business intelligence | Medium |
| Insider | Various | High |
| Automated bot | Spam, abuse | Low |

### Identify Attack Vectors

| Vector | Likelihood | Impact | Mitigation Strategy |
|--------|------------|--------|---------------------|
| SQL Injection | Medium | High | Parameterized queries, input validation |
| XSS | Medium | Medium | Output encoding, CSP |
| CSRF | Low | Medium | CSRF tokens, SameSite cookies |
| Broken Authentication | Medium | High | MFA, secure session management |
| Data Exposure | Low | High | Encryption, access controls |

Apply patterns from `context/patterns/security-patterns.md`

## Step 3: Design Authentication

### Requirements
- Who needs to authenticate? (users, services, APIs)
- What authentication methods? (password, OAuth, SSO, API keys, MFA)
- Session management approach
- Token lifecycle (generation, refresh, revocation)

### Design

Document:
- **Authentication Flow**: Describe the login process
- **Credentials**: What credentials are used (password, API key, certificate)
- **Session/Token Management**: How sessions are created, stored, validated
- **Multi-Factor Authentication**: If required, how it's implemented
- **Password Policy**: Requirements (length, complexity, rotation)
- **Account Lockout**: Brute-force protection

Apply patterns from `context/patterns/security-patterns.md`

## Step 4: Design Authorization

### Permission Model

Create a table:
| Resource | Action | Required Permission | Notes |
|----------|--------|-------------------|-------|
| /api/users | GET | users:read | List all users |
| /api/users/{id} | GET | users:read OR owner | View user details |
| /api/users/{id} | UPDATE | users:write OR owner | Update user |
| /api/admin/* | * | admin:* | Admin panel access |

**Common permission patterns**:
- Role-Based Access Control (RBAC): Permissions grouped by role
- Attribute-Based Access Control (ABAC): Permissions based on attributes
- Resource ownership: User can access their own resources
- Hierarchical permissions: Admin inherits all lower permissions

### Access Control Rules

Document specific rules:
- **Public Access**: What's accessible without authentication
- **Authenticated Access**: What requires login
- **Role-Based Access**: What each role can access
- **Resource Ownership**: Owner-specific permissions
- **Conditional Access**: Context-based rules (IP, time, etc.)

Apply patterns from `context/patterns/security-patterns.md`

## Step 5: Design Data Protection

### Data Classification

| Data Element | Classification | Protection Required |
|--------------|---------------|-------------------|
| email | PII | Encryption at rest, access logging |
| password | Sensitive | Hashing (bcrypt), never logged |
| credit_card | PCI | Tokenization, PCI-DSS compliance |
| session_token | Internal | Encrypted in transit, HttpOnly cookie |
| user_name | Public | None special |

### Encryption Strategy

**At Rest**:
- What data is encrypted in database
- Encryption algorithm (AES-256, etc.)
- Key management approach
- Field-level vs. disk-level encryption

**In Transit**:
- TLS/HTTPS requirements
- Certificate management
- API communication encryption
- Internal service communication

**Key Management**:
- Where keys are stored (KMS, HSM, environment)
- Key rotation policy
- Access controls on keys

Apply patterns from `context/patterns/security-patterns.md`

## Step 6: Design Input Validation

### Validation Rules

For each input (API parameters, form fields):
| Input | Type | Validation | Sanitization |
|-------|------|------------|--------------|
| email | string | Email format, max 255 chars | Lowercase, trim |
| age | integer | Range 0-150 | Parse as int |
| description | string | Max 1000 chars, no HTML | Strip HTML tags |
| file_upload | file | Max 10MB, allowed types | Virus scan |

### Injection Prevention

Document prevention strategies:
- **SQL Injection**: Parameterized queries, ORM usage, input validation
- **XSS**: Output encoding, Content Security Policy, sanitize HTML
- **Command Injection**: Avoid shell commands, whitelist inputs
- **Path Traversal**: Validate file paths, no user input in paths
- **LDAP Injection**: Escape LDAP special characters
- **XML Injection**: Disable external entities, validate XML

## Step 7: Design Frontend Security

### Client-Side Protections

**CSRF Protection**:
- CSRF token mechanism
- Token generation and validation
- SameSite cookie attribute

**Content Security Policy (CSP)**:
- Allowed script sources
- Allowed style sources
- Allowed image/media sources
- Frame options
- Form action restrictions

**Cookie Security**:
- HttpOnly flag (prevent JS access)
- Secure flag (HTTPS only)
- SameSite attribute (CSRF protection)
- Cookie expiration

### Sensitive Data Handling

Document:
- What sensitive data is sent to client
- How it's protected in browser memory
- When it's cleared (logout, timeout)
- Local storage vs. session storage vs. cookies
- Data exposure risks and mitigations

## Step 8: Design API Security

### Endpoint Protection

| Endpoint | Auth Required | Rate Limit | Additional Security |
|----------|--------------|------------|---------------------|
| POST /api/login | No | 5/min per IP | Captcha after 3 failures |
| GET /api/users | Yes | 100/min per user | Pagination required |
| POST /api/admin/delete | Yes, Admin | 10/min per user | MFA confirmation |

### Request/Response Security

**Request Validation**:
- Schema validation (required fields, types)
- Size limits (body, file uploads)
- Content-Type verification
- Origin/Referer checks

**Response Filtering**:
- Remove sensitive fields based on permissions
- Consistent error messages (don't leak info)
- Security headers (HSTS, X-Frame-Options, etc.)

### Error Handling

- Generic error messages to users
- Detailed logging for developers (not exposed)
- No stack traces in production responses
- Consistent error format

## Step 9: Design Audit & Logging

### Security Events to Log

| Event | Log Level | Data Captured | Retention |
|-------|-----------|---------------|-----------|
| Login success | INFO | user_id, timestamp, IP | 90 days |
| Login failure | WARN | attempt_email, IP, timestamp | 90 days |
| Permission denied | WARN | user_id, resource, action | 90 days |
| Data access | INFO | user_id, resource, timestamp | 1 year |
| Configuration change | INFO | user_id, change, timestamp | 1 year |
| Security incident | ERROR | Full context | Indefinite |

**What to log**:
- Who (user/service ID)
- What (action performed)
- When (timestamp)
- Where (IP, location if available)
- Result (success/failure)

**What NOT to log**:
- Passwords or credentials
- Full credit card numbers
- Session tokens
- Encryption keys

### Monitoring & Alerting

Document:
- Anomaly detection (unusual access patterns)
- Brute force detection (login failures)
- Rate limit violations
- Unauthorized access attempts
- Alert thresholds and recipients
- Incident response procedures

## Step 10: Address Compliance

### Regulatory Requirements

Identify applicable regulations:
- **GDPR** (EU data protection)
- **HIPAA** (US healthcare)
- **SOC2** (Security controls)
- **PCI-DSS** (Payment cards)
- **CCPA** (California privacy)

### Compliance Measures

For each applicable regulation, document:
- Data protection requirements
- User rights (access, deletion, portability)
- Consent management
- Data retention policies
- Breach notification procedures
- Documentation requirements

## Step 11: Define Security Testing

### Required Tests

Create checklist:
- [ ] Authentication bypass attempts
- [ ] Authorization escalation tests
- [ ] Input validation (SQL injection, XSS)
- [ ] CSRF protection verification
- [ ] Session management tests
- [ ] Encryption verification
- [ ] Rate limiting tests
- [ ] Security header validation
- [ ] Sensitive data exposure checks
- [ ] Error message information leakage

### Penetration Testing Scope

Document what should be tested:
- Authentication mechanisms
- Authorization rules
- Input validation
- API security
- Client-side security
- Data protection

## Step 12: Create Implementation Checklist

- [ ] Authentication implemented per design
- [ ] Authorization rules enforced on all endpoints
- [ ] Input validation in place
- [ ] Output encoding configured
- [ ] Data encryption at rest configured
- [ ] TLS/HTTPS enforced
- [ ] Security headers configured
- [ ] CSRF protection enabled
- [ ] Rate limiting enabled
- [ ] Audit logging implemented
- [ ] Error messages sanitized
- [ ] Security tests passing
- [ ] Penetration testing completed
- [ ] Compliance requirements verified

## Step 13: Quality Checklist

Before finalizing, verify:

### Completeness
- [ ] Threat model documented
- [ ] Authentication design complete
- [ ] Authorization rules defined
- [ ] Data protection strategy documented
- [ ] Input validation comprehensive
- [ ] Frontend security addressed
- [ ] API security covered
- [ ] Audit logging defined
- [ ] Compliance considered
- [ ] Testing requirements specified

### Cross-Layer Coverage
- [ ] Backend security addressed
- [ ] Frontend security addressed
- [ ] Data layer security addressed
- [ ] Infrastructure security noted
- [ ] All integration points secured

### Practicality
- [ ] All recommendations are implementable
- [ ] Aligns with existing project patterns
- [ ] Security doesn't break usability
- [ ] Performance impact considered

## Step 14: Create Document

### File Location
Create `/docs/plan/{epic-key}/{feature-key}/06-security-design.md`

### Use Template
Follow `context/templates/security-doc.md` structure

### Target Length
100-150 lines (skip sections that don't apply to this feature)

### Review
- Verify all applicable sections complete
- Check cross-references to other docs
- Ensure threat model is comprehensive
- Validate no security gaps exist

## Common Security Patterns

### Defense in Depth
- Multiple layers of security controls
- If one fails, others still protect
- Never rely on a single security measure

### Least Privilege
- Users/services get minimum permissions needed
- Default deny, explicitly allow
- Time-limited elevated permissions

### Fail Secure
- Errors should deny access, not grant it
- Fallback to most restrictive state
- No "default allow" on failures

### Security by Design
- Security is not bolted on after
- Consider security from the start
- Build it into the architecture

## Output Requirements

Upon completion, you will have:
1. **06-security-design.md** - Comprehensive security design
2. Threat model with assets, actors, vectors
3. Authentication and authorization design
4. Data protection strategy
5. Input validation rules
6. Security measures across all layers
7. Audit and logging plan
8. Compliance considerations
9. Security testing requirements
10. Implementation checklist
