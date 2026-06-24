# Security Patterns

This document contains common security patterns and best practices for authentication, authorization, encryption, and data protection.

## Authentication Patterns

### JWT (JSON Web Token)

**Pattern**: Stateless token-based authentication

**Flow**:
1. User sends credentials to `/auth/login`
2. Server validates and returns JWT
3. Client includes JWT in subsequent requests: `Authorization: Bearer <token>`
4. Server validates JWT signature and claims

**JWT Structure**:
- Header: Algorithm and token type
- Payload: Claims (user_id, email, roles, exp, iat)
- Signature: HMAC or RSA signature

**Pros**: Stateless, scalable, works across services
**Cons**: Can't revoke before expiry, token size

**Best Practices**:
- Short expiration (15-60 minutes)
- Use refresh tokens for long-lived sessions
- Sign with strong secret (HS256) or private key (RS256)
- Validate expiration, issuer, audience
- Don't store sensitive data in payload (it's base64, not encrypted)

### Session-Based Authentication

**Pattern**: Server stores session state

**Flow**:
1. User logs in, server creates session
2. Server sends session ID in HTTP-only cookie
3. Client sends cookie with each request
4. Server looks up session in database/cache

**Pros**: Can revoke immediately, server controls session
**Cons**: Stateful, harder to scale, requires session store

**Best Practices**:
- Use HTTP-only, Secure, SameSite cookies
- Store sessions in Redis for performance
- Set appropriate session timeout
- Regenerate session ID after login
- Clear session on logout

### OAuth 2.0

**Authorization Code Flow** (for web apps):
1. Redirect user to OAuth provider
2. User authorizes, provider redirects back with code
3. Exchange code for access token
4. Use access token to access resources

**Client Credentials Flow** (for service-to-service):
1. Service sends client_id and client_secret
2. Receives access token
3. Uses token for API calls

**Best Practices**:
- Use PKCE for mobile/SPA apps
- Validate redirect URIs
- Store client secrets securely
- Use short-lived access tokens with refresh tokens

### API Key Authentication

**Pattern**: Client sends API key in header

```
X-API-Key: sk_live_abc123xyz
```

**Pros**: Simple for service-to-service, third-party integrations
**Cons**: If leaked, full access until revoked

**Best Practices**:
- Generate cryptographically random keys
- Hash keys in database (bcrypt, like passwords)
- Support key rotation
- Allow multiple keys per user
- Log all API key usage
- Rate limit by key

### Multi-Factor Authentication (MFA)

**TOTP (Time-Based One-Time Password)**:
- User scans QR code to register device
- Generates 6-digit code every 30 seconds
- Server validates code against server time

**SMS/Email Codes**:
- Send code to verified phone/email
- User enters code to complete authentication

**WebAuthn/FIDO2**:
- Hardware security keys (YubiKey, etc.)
- Biometric authentication

**Best Practices**:
- Offer backup codes for account recovery
- Rate limit MFA attempts
- Don't lock out after too many failures (prevents DoS)

## Authorization Patterns

### Role-Based Access Control (RBAC)

**Pattern**: Permissions grouped by role

```
Roles:
  - admin: users:*, posts:*, settings:*
  - editor: posts:read, posts:write, posts:delete
  - viewer: posts:read

Users:
  - Alice: [admin]
  - Bob: [editor]
  - Charlie: [viewer]
```

**Implementation**:
1. Assign roles to users
2. Define permissions for each role
3. Check user's roles have required permission

**Pros**: Simple, easy to understand
**Cons**: Can be rigid, role explosion

### Attribute-Based Access Control (ABAC)

**Pattern**: Permissions based on attributes (user, resource, environment)

```
Rule: Allow if
  - user.department == resource.department
  - resource.status == "published"
  - current_time within business_hours
```

**Pros**: Fine-grained, flexible, contextual
**Cons**: Complex to define and test

### Resource Ownership

**Pattern**: Users can access their own resources

```
SELECT * FROM posts WHERE user_id = current_user_id
```

**Implementation**:
- Check ownership before operations
- Combine with RBAC (admins can access all)

### Row Level Security (RLS) - PostgreSQL

**Pattern**: Database-level access control

```
CREATE POLICY user_posts_policy ON posts
  FOR ALL
  TO app_user
  USING (user_id = current_user_id() OR is_admin());
```

**Pros**: Enforced at database level, can't bypass
**Cons**: PostgreSQL-specific, complex policies

**Best Practices**:
- Set user context at connection time
- Test policies thoroughly
- Monitor performance impact

## Data Protection Patterns

### Encryption at Rest

**Database Encryption**:
- Transparent Data Encryption (TDE): Entire database
- Field-level encryption: Specific sensitive fields

**Field-Level Encryption**:
```
Users table:
  id: INTEGER
  email: VARCHAR
  ssn_encrypted: BYTEA  -- Encrypted with AES-256
  ssn_key_id: VARCHAR   -- Which key encrypted it
```

**Best Practices**:
- Use AES-256 for encryption
- Rotate encryption keys periodically
- Store keys in Key Management Service (KMS, not in code)
- Encrypt PII, financial data, health records

### Encryption in Transit

**TLS/HTTPS**:
- All API communication over HTTPS
- TLS 1.2 or higher
- Strong cipher suites
- Valid certificates from trusted CA

**Database Connections**:
- Encrypt connections to database (SSL/TLS)
- Verify server certificates

### Password Hashing

**Pattern**: Never store passwords in plaintext

**Use bcrypt, scrypt, or Argon2**:
```
// Hashing (on registration/password change)
hash = bcrypt.hash(password, salt_rounds=12)

// Verification (on login)
is_valid = bcrypt.verify(password, stored_hash)
```

**Don't use**: MD5, SHA1, SHA256 alone (too fast, no salt)

**Best Practices**:
- Use bcrypt with cost factor 12+
- Never log passwords
- Require password strength (length, complexity)
- Implement password history (prevent reuse)

### Secrets Management

**Pattern**: Store secrets in secure vault, not in code

**Options**:
- HashiCorp Vault
- AWS Secrets Manager
- Azure Key Vault
- Environment variables (for simple cases)

**Best Practices**:
- Never commit secrets to version control
- Rotate secrets regularly
- Use different secrets per environment
- Audit secret access

## Input Validation Patterns

### Whitelist Validation

**Pattern**: Define what is allowed, reject everything else

```
// Good
allowed_statuses = ["active", "inactive", "pending"]
if status not in allowed_statuses:
    raise ValidationError("Invalid status")

// Bad - blacklist approach
if status == "admin":
    raise ValidationError("Invalid status")
```

### Server-Side Validation

**Pattern**: Always validate on server, even if client validates

**Validate**:
- Data types
- Ranges (min/max, length)
- Formats (email, URL, date)
- Business rules
- Referential integrity

### Parameterized Queries (SQL Injection Prevention)

**Good - Parameterized**:
```
query = "SELECT * FROM users WHERE email = ?"
execute(query, [email])
```

**Bad - String concatenation**:
```
query = f"SELECT * FROM users WHERE email = '{email}'"  // SQL injection risk!
```

### Output Encoding (XSS Prevention)

**Pattern**: Encode output based on context

**HTML Context**:
```
Encode: & < > " ' /
Result: &amp; &lt; &gt; &quot; &#x27; &#x2F;
```

**JavaScript Context**:
```
Use JSON.stringify() for data insertion
```

**URL Context**:
```
Use URL encoding for parameters
```

**Best Practices**:
- Use templating engines with auto-escaping
- Content Security Policy (CSP) headers
- Sanitize HTML inputs (if allowing HTML)

### Content Security Policy (CSP)

**Pattern**: HTTP header restricting resource loading

```
Content-Security-Policy:
  default-src 'self';
  script-src 'self' https://trusted-cdn.com;
  style-src 'self' 'unsafe-inline';
  img-src 'self' data: https:;
  font-src 'self';
  connect-src 'self' https://api.example.com;
  frame-ancestors 'none';
```

**Prevents**:
- XSS attacks
- Clickjacking
- Code injection

## CSRF Protection Pattern

**Pattern**: Verify requests originated from your site

**Synchronizer Token**:
1. Server generates random CSRF token
2. Sends token in page (hidden form field or meta tag)
3. Client includes token in POST/PUT/DELETE requests
4. Server validates token matches session

**SameSite Cookies**:
```
Set-Cookie: session=abc123; SameSite=Strict; Secure; HttpOnly
```

**Modes**:
- Strict: Never sent on cross-site requests
- Lax: Sent on top-level navigation (GET only)
- None: Always sent (requires Secure flag)

## Rate Limiting Patterns

### Token Bucket

**Pattern**: Refill tokens over time, consume per request

**Algorithm**:
- Bucket has capacity (e.g., 100 tokens)
- Refills at rate (e.g., 10 tokens/second)
- Each request consumes 1 token
- Reject if bucket empty

**Pros**: Handles bursts, smooth rate
**Cons**: More complex to implement

### Fixed Window

**Pattern**: Max requests per time window

**Example**: 100 requests per minute
- Reset counter every minute
- Increment on each request
- Reject if over limit

**Pros**: Simple
**Cons**: Burst at window boundaries

### Sliding Window Log

**Pattern**: Track timestamp of each request

**Algorithm**:
- Store timestamp of each request
- Remove timestamps older than window
- Count remaining timestamps
- Reject if over limit

**Pros**: Accurate, no boundary bursts
**Cons**: Storage overhead

**Best Practices**:
- Different limits for authenticated vs. unauthenticated
- Different limits per endpoint (search more restrictive)
- Return Retry-After header
- Log rate limit violations

## Security Headers

**Essential Headers**:
```
Strict-Transport-Security: max-age=31536000; includeSubDomains
X-Frame-Options: DENY
X-Content-Type-Options: nosniff
X-XSS-Protection: 1; mode=block
Referrer-Policy: strict-origin-when-cross-origin
Permissions-Policy: geolocation=(), microphone=(), camera=()
```

## Audit Logging Pattern

**What to Log**:
- Authentication events (login, logout, failures)
- Authorization failures
- Data access (who, what, when)
- Configuration changes
- Security events

**What NOT to Log**:
- Passwords or credentials
- Full credit card numbers
- Session tokens
- Encryption keys
- Sensitive PII (unless required for audit)

**Log Format**:
```
{
  "timestamp": "2024-12-09T10:30:00Z",
  "event_type": "login_failure",
  "user_id": null,
  "email": "user@example.com",
  "ip_address": "203.0.113.42",
  "user_agent": "Mozilla/5.0...",
  "result": "invalid_password",
  "request_id": "req_abc123"
}
```

**Best Practices**:
- Structured logging (JSON)
- Include correlation/request IDs
- Immutable logs (append-only)
- Centralized log aggregation
- Retention policy (GDPR compliance)
- Monitor for anomalies

## Principle of Least Privilege

**Pattern**: Grant minimum permissions needed

**Examples**:
- Database users: Read-only for reporting services
- API keys: Scope to specific resources
- IAM roles: Only required AWS services
- User permissions: Start with nothing, grant as needed

## Defense in Depth

**Pattern**: Multiple layers of security

**Layers**:
1. Network: Firewalls, VPCs, security groups
2. Application: Authentication, authorization, input validation
3. Data: Encryption at rest and in transit
4. Monitoring: Logging, alerting, intrusion detection

**Philosophy**: If one layer fails, others still protect
