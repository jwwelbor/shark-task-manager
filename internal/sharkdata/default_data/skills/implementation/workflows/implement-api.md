---
inputs:
  - task_spec_path: absolute path to the task spec markdown
  - feature_prd_path: absolute path to the feature PRD markdown
  - api_spec_path: absolute path to the API specification markdown (canonical DTO source)
  - architecture_path: absolute path to the architecture document (optional)
  - security_spec_path: absolute path to security/performance requirements (optional)
  - coding_standards_path: absolute path to coding standards doc (optional)
  - existing_files: list of paths to existing source files relevant to the change
  - test_plan_path: absolute path to a pre-authored QA test plan, if one exists (optional)
  - dto_destination: absolute path/dir where DTO/schema files should be written (e.g. `app/schemas/`)
  - api_destination: absolute path/dir where endpoint handlers should be written (e.g. `app/api/v1/`)
  - service_destination: absolute path/dir where service-layer code should be written
  - contract_test_destination: absolute path/dir for contract tests
  - unit_test_destination: absolute path/dir for unit tests
  - integration_test_destination: absolute path/dir for integration tests
  - frontend_contract_contact: how to communicate with the frontend implementer for contract sync (optional)
outputs:
  - implementation_files: list of paths to created/modified source files (DTOs, endpoints, service code)
  - test_files: list of paths to created/modified test files (contract, unit, integration)
  - validation_gate_results: structured record of lint, type-check, unit, integration, coverage outcomes
  - contract_sync_status: SYNCHRONIZED | PENDING | MISMATCH (with details if not SYNCHRONIZED)
  - summary: structured markdown describing endpoints implemented, DTOs, business rules, error handling
---

# API Implementation Workflow

## Purpose

This workflow guides systematic implementation of backend API endpoints following contract-first discipline and validation gates.

Use this workflow when implementing:
- REST API endpoints
- GraphQL resolvers
- API route handlers
- Request/response handling

## Prerequisites

Before starting API implementation:

1. **Design Documentation Available**
   - Architecture document
   - API specification (the canonical DTO source)
   - Security requirements

2. **DTO Definitions Clear**
   - Request DTOs specified
   - Response DTOs specified
   - Validation rules documented

3. **Dependencies Ready**
   - Database migrations complete (if needed)
   - Authentication middleware available (if needed)
   - Prerequisite tasks completed

## Phase 1: Contract-First Implementation (CRITICAL)

**Time: 30-45 minutes | Prevents: Days of integration debugging**

### Step 1.1: Extract DTO Requirements (5 minutes)

Open the API specification (`api_spec_path`).

Find the "DTO Definitions" section with:
- EXACT field names (snake_case, camelCase, etc.)
- EXACT types (string, number, UUID, datetime format)
- EXACT validation rules (required, optional, constraints)
- Contract Synchronization Table (your source of truth)

**Verify:** "Codebase Analysis" section confirms no duplicate implementations.

### Step 1.2: Implement DTOs as Interfaces Only (10 minutes)

**Python/FastAPI Example:**
```python
# File: app/schemas/{feature}_schema.py
from pydantic import BaseModel, EmailStr, Field
from typing import Optional
from datetime import datetime
from uuid import UUID

class CreateUserRequest(BaseModel):
    """
    From: <api_spec_path>#CreateUserRequest

    Contract: This DTO must match frontend TypeScript interface EXACTLY
    """
    email: EmailStr              # Spec: string, required, email validation
    first_name: str = Field(min_length=1, max_length=255)  # Spec: required, 1-255 chars
    last_name: str = Field(min_length=1, max_length=255)   # Spec: required, 1-255 chars
    # NO business logic here - pure data structure

class UserResponse(BaseModel):
    """
    From: <api_spec_path>#UserResponse

    Contract: This DTO must match frontend TypeScript interface EXACTLY
    """
    id: UUID
    email: str
    first_name: str              # MUST match request field name (not firstName!)
    last_name: str               # MUST match request field name (not lastName!)
    created_at: datetime

    class Config:
        json_encoders = {
            datetime: lambda v: v.isoformat()  # ISO 8601 format
        }
```

**Node/Express Example:**
```typescript
// File: src/schemas/{feature}.schema.ts
import { z } from 'zod';

/**
 * From: <api_spec_path>#CreateUserRequest
 *
 * Contract: This schema must match frontend TypeScript interface EXACTLY
 */
export const CreateUserRequestSchema = z.object({
  email: z.string().email(),                    // Spec: email validation
  first_name: z.string().min(1).max(255),       // Spec: required, 1-255 chars
  last_name: z.string().min(1).max(255),        // Spec: required, 1-255 chars
});

export type CreateUserRequest = z.infer<typeof CreateUserRequestSchema>;

/**
 * From: <api_spec_path>#UserResponse
 */
export const UserResponseSchema = z.object({
  id: z.string().uuid(),
  email: z.string().email(),
  first_name: z.string(),                       // MUST match request (not firstName!)
  last_name: z.string(),                        // MUST match request (not lastName!)
  created_at: z.string().datetime(),            // ISO 8601 string
});

export type UserResponse = z.infer<typeof UserResponseSchema>;
```

**Anti-Patterns (DO NOT DO):**
```python
# WRONG - Different field names than spec
class CreateUserRequest(BaseModel):
    email: str
    firstName: str    # Spec says first_name!
    lastName: str     # Spec says last_name!

# WRONG - Adding business logic to DTO
class CreateUserRequest(BaseModel):
    email: str

    @validator('email')
    def validate_corporate_email(cls, v):
        if not v.endswith('@company.com'):
            raise ValueError('Must be corporate email')
        return v
    # Business rules belong in service layer!
```

### Step 1.3: Implement Function Signatures Only (20 minutes)

Stub out endpoints with types, no business logic yet:

```python
# File: app/api/v1/{feature}.py
from fastapi import APIRouter, Depends, HTTPException, status
from app.schemas.user_schema import CreateUserRequest, UserResponse
from app.db.session import get_db
from sqlalchemy.orm import Session

router = APIRouter(prefix="/users", tags=["users"])

@router.post("/", response_model=UserResponse, status_code=status.HTTP_201_CREATED)
async def create_user(
    request: CreateUserRequest,
    db: Session = Depends(get_db)
) -> UserResponse:
    """
    Create a new user.

    Spec: <api_spec_path>#post-users

    Contract:
    - Receives: CreateUserRequest
    - Returns: UserResponse (201 Created)
    - Errors: 400 (validation), 409 (conflict), 500 (server error)
    """
    # Phase 2: Business logic implemented after contract validation
    raise NotImplementedError("Business logic implemented in Phase 2")

@router.get("/{user_id}", response_model=UserResponse)
async def get_user(
    user_id: str,
    db: Session = Depends(get_db)
) -> UserResponse:
    """Get user by ID."""
    raise NotImplementedError("Business logic implemented in Phase 2")
```

**Why stub?** This validates:
- DTO types compile/pass type checking
- Endpoint signatures match specification
- No syntax errors in DTOs
- Ready for contract tests

### Step 1.4: Run Contract Validation Tests (10 minutes)

Create tests that verify DTO structure ONLY (not behavior):

```python
# File: tests/contract/test_user_contracts.py
import pytest
from app.schemas.user_schema import CreateUserRequest, UserResponse

def test_create_user_request_dto_structure():
    """Verify CreateUserRequest DTO matches API spec exactly."""
    expected_fields = {'email', 'first_name', 'last_name'}
    actual_fields = set(CreateUserRequest.__fields__.keys())

    assert actual_fields == expected_fields, \
        f"DTO fields diverged from spec. Expected: {expected_fields}, Got: {actual_fields}"

def test_user_response_dto_structure():
    """Verify UserResponse DTO matches API spec exactly."""
    expected_fields = {'id', 'email', 'first_name', 'last_name', 'created_at'}
    actual_fields = set(UserResponse.__fields__.keys())

    assert actual_fields == expected_fields, \
        f"DTO fields diverged from spec. Expected: {expected_fields}, Got: {actual_fields}"

def test_dto_field_types():
    """Verify DTO field types match specification."""
    from pydantic import EmailStr
    from uuid import UUID
    from datetime import datetime

    assert CreateUserRequest.__fields__['email'].type_ == EmailStr
    assert CreateUserRequest.__fields__['first_name'].type_ == str
    assert CreateUserRequest.__fields__['last_name'].type_ == str

    assert UserResponse.__fields__['id'].type_ == UUID
    assert UserResponse.__fields__['created_at'].type_ == datetime
```

Run tests with the project's test runner. Expected: all contract tests pass.

**If tests fail:** DTOs don't match spec. Fix DTOs before proceeding.

### Step 1.5: Synchronize with Frontend Team

**Communication Required:**

Share with frontend developer:
```
DTOs implemented for {feature}:
- CreateUserRequest: email, first_name, last_name
- UserResponse: id, email, first_name, last_name, created_at

Location: <dto_destination>/{feature}_schema.py
Contract tests passing: yes

Please verify your TypeScript interfaces match exactly:
- Field names (first_name not firstName)
- Types (created_at is ISO 8601 string)
- Validation rules (email format, string lengths)
```

Await confirmation:
```
Frontend confirms:
- TypeScript interfaces match backend DTOs
- Field names identical (first_name, last_name)
- Types compatible (string, UUID string, ISO datetime)
- Contract synchronized

Proceeding to business logic implementation.
```

**STOP if not synchronized.** Align before Phase 2. Record `contract_sync_status` accordingly.

## Phase 2: Business Logic Implementation

With validated contracts, implement the actual logic.

### Step 2.1: Implement Service Layer

Separate business logic from HTTP handling:

```python
# File: app/services/user_service.py
from sqlalchemy.orm import Session
from app.models.user import User
from app.schemas.user_schema import CreateUserRequest, UserResponse
from app.core.security import hash_password
from uuid import uuid4
from datetime import datetime

class UserService:
    """Business logic for user management."""

    def __init__(self, db: Session):
        self.db = db

    async def create_user(self, request: CreateUserRequest) -> User:
        """
        Create a new user.

        Business rules:
        - Email must be unique
        - Password must be hashed
        - User starts with 'active' status
        """
        existing = self.db.query(User).filter(User.email == request.email).first()
        if existing:
            raise ValueError(f"User with email {request.email} already exists")

        user = User(
            id=uuid4(),
            email=request.email,
            first_name=request.first_name,
            last_name=request.last_name,
            created_at=datetime.utcnow(),
            status='active'
        )

        self.db.add(user)
        self.db.commit()
        self.db.refresh(user)

        return user

    async def get_user_by_id(self, user_id: str) -> User:
        """Get user by ID."""
        user = self.db.query(User).filter(User.id == user_id).first()
        if not user:
            raise ValueError(f"User {user_id} not found")
        return user
```

### Step 2.2: Wire Service Layer to Endpoints

Replace `NotImplementedError` stubs:

```python
# File: app/api/v1/users.py
from app.services.user_service import UserService

@router.post("/", response_model=UserResponse, status_code=status.HTTP_201_CREATED)
async def create_user(
    request: CreateUserRequest,
    db: Session = Depends(get_db)
) -> UserResponse:
    """Create a new user."""
    service = UserService(db)

    try:
        user = await service.create_user(request)
        return UserResponse(
            id=user.id,
            email=user.email,
            first_name=user.first_name,
            last_name=user.last_name,
            created_at=user.created_at
        )
    except ValueError as e:
        if "already exists" in str(e):
            raise HTTPException(status_code=409, detail=str(e))
        raise HTTPException(status_code=400, detail=str(e))
    except Exception as e:
        logger.error(f"Unexpected error creating user: {e}")
        raise HTTPException(status_code=500, detail="Internal server error")
```

### Step 2.3: Implement Error Handling

Follow project error-handling patterns:

1. **Expected errors** - Return appropriate HTTP status codes
2. **Validation errors** - 400 Bad Request with field details
3. **Conflict errors** - 409 Conflict with clear message
4. **Not found errors** - 404 Not Found
5. **Server errors** - 500 Internal Server Error (log details, hide from user)

### Step 2.4: Add Authentication/Authorization (if needed)

```python
from app.api.dependencies import get_current_user

@router.post("/", response_model=UserResponse, status_code=201)
async def create_user(
    request: CreateUserRequest,
    current_user: User = Depends(get_current_user),  # Require authentication
    db: Session = Depends(get_db)
) -> UserResponse:
    if not current_user.has_permission('create_user'):
        raise HTTPException(status_code=403, detail="Insufficient permissions")

    # ... implementation
```

## Phase 3: Testing

### Step 3.1: Unit Test Service Layer

```python
# File: tests/unit/services/test_user_service.py
import pytest
from app.services.user_service import UserService
from app.schemas.user_schema import CreateUserRequest

def test_create_user_success(db_session):
    """Test successful user creation."""
    service = UserService(db_session)
    request = CreateUserRequest(
        email="test@example.com",
        first_name="John",
        last_name="Doe"
    )

    user = await service.create_user(request)

    assert user.email == "test@example.com"
    assert user.first_name == "John"
    assert user.last_name == "Doe"
    assert user.id is not None

def test_create_user_duplicate_email(db_session):
    """Test error on duplicate email."""
    service = UserService(db_session)
    request = CreateUserRequest(
        email="test@example.com",
        first_name="John",
        last_name="Doe"
    )

    await service.create_user(request)

    with pytest.raises(ValueError, match="already exists"):
        await service.create_user(request)
```

### Step 3.2: Integration Test API Endpoints

```python
# File: tests/integration/test_user_api.py
from fastapi.testclient import TestClient

def test_create_user_endpoint(client: TestClient):
    """Test POST /users endpoint."""
    response = client.post(
        "/api/v1/users",
        json={
            "email": "test@example.com",
            "first_name": "John",
            "last_name": "Doe"
        }
    )

    assert response.status_code == 201
    data = response.json()
    assert data["email"] == "test@example.com"
    assert data["first_name"] == "John"
    assert "id" in data
    assert "created_at" in data

def test_create_user_invalid_email(client: TestClient):
    """Test validation error on invalid email."""
    response = client.post(
        "/api/v1/users",
        json={
            "email": "invalid-email",
            "first_name": "John",
            "last_name": "Doe"
        }
    )

    assert response.status_code == 422  # Validation error
```

## Phase 4: Validation Gates

Run all gates before considering work complete. Choose the appropriate runner by inspecting the project; the principles below apply regardless of language/runtime.

### Gate 1: Linting

Run the project's linter and formatter against the new/modified files. Fail-fast on errors; do not auto-fix and proceed silently.

### Gate 2: Type Checking

Run the project's type checker (e.g., `mypy`, `tsc --noEmit`). Type errors are blockers — they often indicate DTO drift.

### Gate 3: Unit Tests

Run unit tests scoped to the changed service / endpoint files.

### Gate 4: Integration Tests

Run integration tests that exercise the endpoints end-to-end through the test client / harness.

### Gate 5: Coverage

Generate a coverage report and ensure the configured threshold (typical: >= 80% for new code) is met.

**All gates must pass.** Fix failures immediately. Record outcomes in `validation_gate_results`.

## Phase 5: Documentation

Produce a structured implementation summary covering:

- **Endpoints implemented** — method, path, purpose
- **DTOs** — request and response shapes (field name + type)
- **Service layer** — public methods and business rules
- **Error handling** — exception types and HTTP status mappings
- **Validation gate results** — pass/fail for each gate
- **Contract synchronization** — confirmation that frontend interfaces match

## Completion Checklist

- [ ] DTOs implemented matching specification exactly
- [ ] Contract tests passing
- [ ] Frontend synchronization confirmed
- [ ] Service layer implemented with business logic
- [ ] Endpoints wired to service layer
- [ ] Error handling comprehensive
- [ ] Unit tests passing (80%+ coverage)
- [ ] Integration tests passing
- [ ] All validation gates passed
- [ ] Implementation summary returned
- [ ] Outstanding TODOs documented

## Common Issues

| Issue | Solution |
|-------|----------|
| Frontend/backend DTO mismatch | Review Step 1.5 synchronization |
| Contract tests failing | DTOs don't match spec - fix before Phase 2 |
| Integration tests flaky | Check database state, use fixtures |
| Type errors | Ensure DTOs have explicit types |
| Validation not working | Check Pydantic/Zod schema definitions |

---

**Remember:** Contract-first prevents integration pain. Invest 45 minutes in Phase 1 to save days in debugging.
