# Flow Diagram Documentation Generation

**Project**: Shark Task Manager
**Date**: 2025-12-19
**Purpose**: Comprehensive flow diagrams and documentation for Epic, Feature, and Task creation workflows

## Quick Start

These documents provide detailed flow diagrams and architectural documentation for the three main creation workflows in Shark Task Manager.

### Essential Reading Order

1. **Start Here**: `SUMMARY.md` - Overview of all deliverables and key insights
2. **Simple First**: `analysis/EPIC_CREATION_FLOW.md` - Foundation pattern (no dependencies)
3. **Add Complexity**: `analysis/FEATURE_CREATION_FLOW.md` - Parent dependency pattern
4. **Most Complex**: `analysis/TASK_CREATION_FLOW.md` - Full orchestration pattern

---

## Document Structure

### Summary Document
- **File**: `SUMMARY.md`
- **Purpose**: Overview and navigation guide
- **Contents**: Deliverables, key insights, usage guide

### Flow Diagrams (3 files)

#### 1. EPIC_CREATION_FLOW.md
**Lines**: 610 | **Diagrams**: 2 (Sequence + Flowchart)

Epic creation is the simplest workflow showing the fundamental pattern:
- No parent dependencies
- Global key generation (E##)
- Flat directory structure
- Basic validation and file operations

**Key Sections**:
- Overview with workflow visualization
- Sequence diagram: CLI → Validation → KeyGen → FileOps → Repository → Database
- Flowchart: 30+ decision points with error paths
- Layer-by-layer breakdown (6 layers)
- Data transformations with examples
- Error handling paths and exit codes
- Concurrency considerations and race conditions

**Audience**: Anyone new to Shark Task Manager architecture

---

#### 2. FEATURE_CREATION_FLOW.md
**Lines**: 725 | **Diagrams**: 2 (Sequence + Flowchart)

Feature creation extends the epic pattern with parent dependency:
- Requires parent epic verification
- Per-epic key generation (E##-F##)
- Nested hierarchical directory structure
- Foreign key constraints
- Required --epic flag

**Key Sections**:
- Overview with hierarchical workflow
- Sequence diagram: Epic validation + hierarchical directory handling
- Flowchart: 35+ decision points including epic lookup
- Layer-by-layer breakdown (6 layers with epic specifics)
- Key differences from epic creation
- Hierarchical directory structure details
- File system operations with parent directory lookup
- Database schema with foreign keys and CASCADE DELETE

**Audience**: Understanding parent-child relationships in the system

---

#### 3. TASK_CREATION_FLOW.md
**Lines**: 937 | **Diagrams**: 2 (Sequence + Flowchart)

Task creation is the most sophisticated workflow with full orchestration:
- Dedicated Creator component orchestrating operations
- Multi-step validation (format, existence, range, dependency)
- Circular dependency detection
- File collision detection with force override
- Transaction management for consistency
- Template selection with priority
- Custom filename validation
- Task history audit trail

**Key Sections**:
- Overview with complex workflow
- Sequence diagram: 50+ interaction points with Creator orchestration
- Flowchart: 45+ decision points with transaction management
- Component architecture explanation
- Layer-by-layer breakdown with complex details
- Multi-step validation scenario documentation
- Transaction atomicity explanation
- Complex validation scenarios (normalization, dependencies, collisions)
- Error handling with cleanup operations

**Audience**: Understanding complex orchestrated workflows

---

## Architecture Patterns Documented

### 1. Clean Architecture
Clear separation into layers:
- **CLI Layer**: User interface and argument parsing
- **Validation Layer**: Model validation and business rule enforcement
- **Business Logic Layer**: Key generation, orchestration
- **Repository Layer**: Data access abstraction
- **Database Layer**: Persistence with constraints

### 2. Data Transformation Pipeline
```
User Input
  ↓ (Parse & Validate)
Model Objects
  ↓ (Transform to SQL)
Database Operations
  ↓ (Transform to Response)
User Output
```

### 3. Error Handling Strategy
- **Exit Code 1**: Validation/file system errors (recoverable)
- **Exit Code 2**: Database errors (more serious)
- **Cleanup**: Remove partial results on failure
- **Rollback**: Atomic transactions prevent inconsistency

### 4. Database Safety
- Parameterized queries prevent SQL injection
- Constraints enforce business rules
- Triggers maintain timestamps
- Transactions ensure consistency
- Foreign keys maintain referential integrity
- CASCADE DELETE ensures data cleanup

---

## Key Technical Insights

### Epic Creation
- **Key Generation**: Global sequential E## across entire system
- **Potential Issue**: Race condition on concurrent creation (UNIQUE constraint catches it)
- **Cleanup**: If database fails, removes created files

### Feature Creation
- **Key Generation**: Per-epic sequential F## scoped to specific epic
- **Parent Dependency**: Verifies epic exists in database
- **Directory Hierarchy**: `docs/plan/E##-*/E##-F##-*/`
- **Data Consistency**: Foreign key ensures epic is not deleted while feature exists

### Task Creation
- **Creator Component**: Orchestrates multiple responsibilities (key generation, validation, rendering)
- **Multi-Step Validation**: Format → Existence → Range → Dependency → Custom path validation
- **Dependency Handling**: Parses JSON, validates each dependency, detects circular references
- **File Management**: Collision detection, force override, custom path validation (prevents path traversal)
- **Transaction Atomicity**: Database and file operations are atomic (all or nothing)
- **Template Selection**: Priority-based (custom → agent-specific → general → default)
- **Audit Trail**: Task history record tracks creation with agent and timestamp

---

## Visual Diagrams

### Mermaid Sequence Diagrams
Show interaction between components with:
- Actor (User)
- Sequential interactions
- Error branches
- Return values
- Data flow

**Usage**: Understand the flow of interactions and order of operations

### Mermaid Flowcharts
Show decision logic with:
- Sequential operations (rectangles)
- Decision points (diamonds)
- Error paths and exit codes
- Cleanup operations
- Success/failure outcomes

**Usage**: Understand the logic flow and all possible paths including errors

---

## Detailed Content Overview

### EPIC_CREATION_FLOW.md
```
1. Overview
   - Workflow diagram
   - Component interaction
   
2. Sequence Diagram
   - User → CLI → Validation → KeyGen → FileOps → Template → Repo → DB
   - 35+ interaction points
   
3. Flowchart
   - 30+ decision nodes
   - Error paths with cleanup
   - Exit codes 1 and 2
   
4. CLI Layer
   - runEpicCreate function
   - Argument parsing
   - Database connection
   
5. Validation Layer
   - Epic.Validate() method
   - Constraint enforcement
   
6. Key Generation
   - getNextEpicKey() algorithm
   - Query existing epics
   - Find max number
   
7. File System Layer
   - Directory creation
   - Template rendering
   - File writing
   
8. Repository Layer
   - EpicRepository.Create()
   - Parameterized insert
   
9. Database Layer
   - Table schema
   - Constraints (UNIQUE, CHECK)
   - Indexes
   - PRAGMAs for performance
   
10. Data Transformations
    - Input → Model → SQL → Response
    - Concrete examples
    
11. Error Handling Paths
    - 6+ scenarios with root causes
    - Exit codes and cleanup
    
12. Concurrency Considerations
    - Race conditions identified
    - Mitigations documented
    
13. Summary Table
    - Responsibilities by layer
```

### FEATURE_CREATION_FLOW.md
```
[Similar structure to EPIC with additions:]

- Epic existence verification
- Per-epic key generation
- Hierarchical directory structure
- filepath.Glob() for parent lookup
- Foreign key constraints
- CASCADE DELETE behavior
- Key differences from epic creation
```

### TASK_CREATION_FLOW.md
```
[Extended structure with:]

- Creator component orchestration
- KeyGenerator implementation
- Multi-step Validator
- Template selection with priority
- File collision detection
- Force override logic
- Custom filename validation
- Transaction management
- History record creation
- Complex validation scenarios
- Dependency resolution
- Circular reference detection
```

---

## Code References

All diagrams reference actual implementation code:

### Files Analyzed
- `internal/cli/commands/epic.go`
- `internal/cli/commands/feature.go`
- `internal/cli/commands/task.go`
- `internal/repository/epic_repository.go`
- `internal/repository/feature_repository.go`
- `internal/repository/task_repository.go`
- `internal/models/validation.go`
- `internal/taskcreation/creator.go`
- `internal/db/db.go`

### Key Functions Documented
- `runEpicCreate()` - Epic creation entry point
- `runFeatureCreate()` - Feature creation entry point
- `runTaskCreate()` - Task creation entry point
- `getNextEpicKey()` - Epic key generation
- `getNextFeatureKey()` - Feature key generation
- `KeyGenerator.GenerateTaskKey()` - Task key generation
- `Creator.CreateTask()` - Task creation orchestration
- `*Repository.Create()` - Database insertion for each entity

---

## How to Use These Documents

### For New Developers
1. Read EPIC_CREATION_FLOW.md to understand the basic pattern
2. Read FEATURE_CREATION_FLOW.md to understand parent-child relationships
3. Read TASK_CREATION_FLOW.md to understand complex orchestration

### For Architecture Review
1. Compare actual implementation against flowcharts
2. Verify all error paths are handled
3. Check that cleanup operations are performed
4. Ensure transaction atomicity is maintained

### For Debugging
1. Locate error in flowchart
2. Find corresponding section in layer-by-layer breakdown
3. Check error handling path documentation
4. Review error scenarios with similar root cause

### For Feature Development
1. Review sequence diagram to understand interaction points
2. Check flowchart for decision logic
3. Identify which layer(s) need modification
4. Review error handling for new paths

### For Testing
1. Use flowchart to identify all decision paths
2. Create test cases for each error scenario
3. Test data transformations at each layer
4. Verify cleanup operations on failure

---

## File Manifest

```
/dev-artifacts/2025-12-19-documentation-generation/
│
├── README.md                    (this file)
│   Overview and navigation guide
│
├── SUMMARY.md                   (677 lines)
│   Complete summary of all deliverables
│
└── analysis/
    ├── EPIC_CREATION_FLOW.md    (610 lines)
    │   Epic creation workflow documentation
    │
    ├── FEATURE_CREATION_FLOW.md (725 lines)
    │   Feature creation workflow documentation
    │
    ├── TASK_CREATION_FLOW.md    (937 lines)
    │   Task creation workflow documentation
    │
    ├── DATABASE_SCHEMA_ER_DIAGRAM.md      (from previous work)
    ├── DATABASE_INDEXES.md                (from previous work)
    ├── DATABASE_TRIGGERS.md               (from previous work)
    └── SYNC_LOGIC_DOCUMENTATION.md        (from previous work)
```

**Total Documentation**: 5,806+ lines

---

## Key Statistics

| Document | Lines | Diagrams | Decision Points | Error Paths |
|----------|-------|----------|-----------------|-------------|
| Epic | 610 | 2 | 30+ | 6+ |
| Feature | 725 | 2 | 35+ | 6+ |
| Task | 937 | 2 | 45+ | 8+ |
| **Total** | **2,272** | **6** | **110+** | **20+** |

---

## Next Steps

### For Understanding
1. Open `SUMMARY.md` for overview
2. Choose flow to study (Epic → Feature → Task)
3. Read sequence diagram first
4. Then read flowchart
5. Review layer-by-layer breakdown

### For Implementation
1. Identify which creation flow is relevant
2. Review the complete flow documentation
3. Check layer-by-layer breakdown for your component
4. Verify error handling matches documentation
5. Test error paths identified in flowchart

### For Maintenance
1. Update corresponding flow diagram when code changes
2. Keep error paths in sync with actual implementation
3. Document new validation steps
4. Update cleanup operations if changed
5. Review concurrency considerations for new features

---

## Questions & Clarification

For questions about these diagrams:
1. Check the specific flow document for your scenario
2. Review layer-by-layer breakdown for the relevant component
3. Look at error handling paths for similar scenarios
4. Refer to actual source code for implementation details

