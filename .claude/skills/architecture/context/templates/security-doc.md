# Security Design Document Template (06-security-design.md)

This template is extracted from the security-architect agent. Target length: 100-150 lines.

---

# Security Design: {Feature Name}

**Epic**: {epic-key}
**Feature**: {feature-key}
**Date**: {YYYY-MM-DD}
**Author**: security-architect

## Security Overview

{Brief description of the feature's security posture and key concerns}

## Threat Model

### Assets to Protect
| Asset | Classification | Impact if Compromised |
|-------|---------------|----------------------|
| {asset} | {PII/Sensitive/Internal/Public} | {impact} |

### Threat Actors
| Actor | Motivation | Capability |
|-------|------------|------------|
| {actor} | {motivation} | {capability level} |

### Attack Vectors
| Vector | Likelihood | Impact | Mitigation |
|--------|------------|--------|------------|
| {vector} | High/Medium/Low | High/Medium/Low | {mitigation approach} |

## Authentication

{Authentication requirements and design for this feature}

### Requirements
- {requirement 1}
- {requirement 2}

### Design
{How authentication will be implemented}

## Authorization

{Authorization model for this feature}

### Permission Model
| Resource | Action | Required Permission |
|----------|--------|-------------------|
| {resource} | {action} | {permission} |

### Access Control Rules
{Specific rules for who can access what}

## Data Protection

### Data Classification
| Data Element | Classification | Protection Required |
|--------------|---------------|-------------------|
| {element} | {classification} | {protection} |

### Encryption
- At Rest: {approach}
- In Transit: {approach}
- Key Management: {approach}

## Input Validation

### Validation Rules
| Input | Type | Validation | Sanitization |
|-------|------|------------|--------------|
| {input} | {type} | {rules} | {sanitization} |

### Injection Prevention
{Specific measures for SQL, XSS, command injection}

## Frontend Security (if applicable)

### Client-Side Protections
- CSRF: {approach}
- CSP: {policy}
- Cookies: {settings}

### Sensitive Data Handling
{How sensitive data is handled in the frontend}

## API Security (if applicable)

### Endpoint Protection
| Endpoint | Auth Required | Rate Limit | Additional |
|----------|--------------|------------|------------|
| {endpoint} | Yes/No | {limit} | {notes} |

### Request/Response Security
{Validation, filtering, error handling}

## Audit & Logging

### Security Events to Log
| Event | Log Level | Data Captured | Retention |
|-------|-----------|---------------|-----------|
| {event} | {level} | {data} | {retention} |

### Monitoring & Alerting
{What to monitor and when to alert}

## Compliance Considerations

{Relevant compliance requirements and how they're addressed}

## Security Testing Requirements

### Required Tests
- [ ] {test 1}
- [ ] {test 2}

### Penetration Testing Scope
{What should be tested}

## Implementation Checklist

- [ ] Authentication implemented per design
- [ ] Authorization rules enforced
- [ ] Input validation in place
- [ ] Data encryption configured
- [ ] Audit logging enabled
- [ ] Security headers configured
- [ ] Rate limiting enabled
- [ ] Security tests passing
