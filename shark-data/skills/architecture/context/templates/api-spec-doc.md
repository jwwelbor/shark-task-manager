# API Specification Document Template (04-backend-design.md)

This template is for the backend interface specification. Target length: 150-200 lines.
Adapt based on interface type: API, Library, CLI, or Service.

---

# Backend Design: {Feature Name}

**Epic**: {epic-key}
**Feature**: {feature-key}
**Date**: {YYYY-MM-DD}
**Author**: backend-architect

## Interface Overview

{Brief description of what interfaces this feature provides}

**Interface Type**: {API / Library / CLI / Service / Mixed}

## Codebase Analysis

### Existing Related Interfaces
- `{interface}` at `{file path}` - {brief description}

### Naming Patterns Found
- {Pattern type}: {pattern, e.g., snake_case functions, PascalCase classes}

### Extension vs. New Code Decision
{Analysis of whether to extend existing code or create new}

---

## DTO / Data Structures

### {StructureName}

**Purpose**: {What this structure represents and when used}

**Fields**:
| Field | Type | Required | Validation | Description |
|-------|------|----------|------------|-------------|
| {field} | string | Yes | max 255 chars | {purpose} |
| {field} | integer | No | min 0 | {purpose} |

---

## API Endpoints (if applicable)

### {ResourceName} Endpoints

#### {METHOD} /api/v1/{path}

**Purpose**: {What this endpoint does}

**Authentication**: Required / Optional / None

**Authorization**: {Role/permission requirements}

**Parameters**:
| Parameter | Location | Type | Required | Description |
|-----------|----------|------|----------|-------------|
| {param} | path/query/body | {type} | Yes/No | {purpose} |

**Request Body**: `{DTOName}` (if applicable)

**Response**: `{DTOName}` ({status code})

**Processing Steps**:
1. {Step 1}
2. {Step 2}

**Errors**:
| Code | Condition |
|------|-----------|
| {status} | {when this occurs} |

---

## Library Interface (if applicable)

### {ModuleName}

#### {function_name}

**Purpose**: {What this function does}

**Signature**:
| Parameter | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
| {param} | {type} | Yes/No | {default} | {purpose} |

**Returns**: {type} - {description}

**Raises**:
| Exception | Condition |
|-----------|-----------|
| {ExceptionType} | {when raised} |

**Behavior**:
- {Key behavior 1}
- {Key behavior 2}

---

## CLI Commands (if applicable)

### {command-name}

**Purpose**: {What this command does}

**Usage**: `{program} {command} [options] <arguments>`

**Arguments**:
| Argument | Required | Description |
|----------|----------|-------------|
| {arg} | Yes/No | {purpose} |

**Options**:
| Option | Short | Type | Default | Description |
|--------|-------|------|---------|-------------|
| --{option} | -{x} | {type} | {default} | {purpose} |

**Output**: {What the command outputs}

**Exit Codes**:
| Code | Meaning |
|------|---------|
| 0 | Success |
| 1 | {error condition} |

---

## Event/Message Contracts (if applicable)

### {EventName}

**Trigger**: {What causes this event}

**Payload**:
| Field | Type | Description |
|-------|------|-------------|
| {field} | {type} | {purpose} |

**Consumers**: {What services consume this event}

---

## Error Handling

### Error Codes

| Code | Meaning | Response |
|------|---------|----------|
| {ERROR_CODE} | {description} | {how to handle} |

### Error Response Format

{Standard error structure for this interface}

## Pagination (if applicable)

**Pattern**: {page-based / cursor-based / offset-based}

**Parameters**:
| Parameter | Default | Max | Description |
|-----------|---------|-----|-------------|
| {param} | {default} | {max} | {purpose} |

## Rate Limiting (if applicable)

| Interface | Limit | Window |
|-----------|-------|--------|
| {interface} | {limit} | {per minute/hour} |

## Versioning

- Strategy: {URL-based / Header-based / etc.}
- Current version: {version}
- Deprecation policy: {policy}
