# Task Generation Update - Partial Documentation Support

## Summary

Updated the `/task` slash command and related specification-writing skill to gracefully handle partial design documentation. The workflow now adapts to whatever documentation is available, rather than requiring all design documents upfront.

## Problem Solved

Previously, the `/task` command would fail or block if not all design documents were present. This created friction because:
- Not all features need full design documentation (e.g., DevOps/infrastructure features)
- Iterative development sometimes starts with partial specs
- Different feature types require different levels of design detail

## Changes Made

### 1. Updated `/task` Command Documentation

**File**: `.claude/commands/task.md`

**Changes**:
- Changed "Prerequisites" from requiring all design docs to requiring only `prd.md`
- Listed all design documents as "recommended" rather than "required"
- Added note that workflow detects available docs and adapts task generation
- Documented file name variations (e.g., `03-data-design.md` vs `03-database-design.md`)

### 2. Enhanced Task Generation Workflow

**File**: `.claude/skills/specification-writing/workflows/write-task.md`

**Major additions**:

#### A. New Step 0: Detect Available Documentation
- Checks for PRD (required - stops if missing)
- Detects which design documents are present
- Presents comprehensive summary to user showing:
  - Available documents
  - Missing documents
  - Task detail level (HIGH/MEDIUM/LOW)
  - Capabilities with current documentation
  - Recommendations
- Waits for user confirmation before proceeding
- Adjusts task generation strategy based on available docs

**Example output**:
```
Documentation Analysis for E04-F08-distribution-release:

✅ Available Documents:
- prd.md
- 02-architecture.md
- 06-security-design.md
- 07-performance-design.md
- 08-implementation-phases.md
- 09-test-criteria.md

❌ Missing Documents:
- 03-database-design.md / 03-data-design.md
- 04-api-specification.md / 04-backend-design.md
- 05-frontend-design.md

Task Detail Level: MEDIUM
- Can generate high-level architectural tasks
- Can generate security and performance tasks
- Cannot generate detailed database implementation tasks
- Cannot generate API contract tasks
- Cannot generate frontend component tasks

Recommendation:
- If this is infrastructure/DevOps work → PROCEED (no frontend/API needed)
- If this is full-stack feature → CONSIDER completing design docs first
- If you want to proceed anyway → Tasks will be high-level planning tasks

Continue with available documentation? (yes/no)
```

#### B. Updated Step 1: Analyze Available Documents
- Changed from analyzing "all" documents to analyzing "available" documents
- Made PRD the only required document
- All other documents are conditional ("if present")

#### C. Updated Step 2: Validate Contract Consistency (Now Conditional)
- Only performs contract validation if BOTH API and Frontend docs exist
- Skips validation gracefully if either is missing
- Notes in tasks that contract definition will happen during implementation

#### D. Enhanced Step 3: Determine Task Scope
- Added flexible task structures based on documentation level
- Provides multiple patterns for different documentation scenarios:

**Full Documentation**: Standard 6-task sequence
- Contract Validation → Database → API → Frontend → Integration → Deployment

**PRD + Architecture only**:
- Architecture Implementation
- Define detailed design specs
- Integration planning

**PRD + Architecture + Database (no API/Frontend)**:
- Database Schema Implementation
- Data access layer design
- API design task

**PRD + Architecture + Backend (no Frontend)**:
- Backend API Implementation
- API documentation
- Frontend design task

**PRD + Security/Performance (infrastructure/DevOps)**:
- Infrastructure setup
- Security implementation
- Performance optimization
- Deployment pipeline

#### E. New Section: Handling Incomplete or Missing Documentation
- Provides guidance for missing documents vs incomplete documents
- Explains task quality expectations with partial docs
- Emphasizes that high-level tasks are acceptable and expected

#### F. Enhanced Final Output Summary
- Now includes documentation status
- Shows task detail level
- Provides recommendations if documentation is partial
- Suggests completing missing docs or confirms current level is appropriate

### 3. Updated Task Template

**File**: `.claude/skills/specification-writing/context/task-template.md`

**Changes**:
- Made all design doc references conditional ("if exists")
- Added fallback guidance for missing documentation
- Documented what implementation agents should do when specs are missing

**Specific updates**:

**Codebase Analysis Results**:
- If API spec exists → reference codebase analysis
- If API spec missing → note that agent must perform analysis

**Key Requirements**:
- Reference available design docs
- If docs missing → state requirements from PRD with note that agent makes design decisions

**Contract Specifications**:
- If API spec exists → reference exact DTOs
- If API spec missing → note that contract definition is required during implementation

**Data Flow**:
- If API spec exists → reference detailed flow
- If API spec missing → provide high-level flow from PRD/architecture

**Integration Points**:
- If architecture exists → reference integration points
- If architecture missing → describe from PRD, note agent should document approach

**Context & Resources**:
- Always link to PRD
- Conditionally link to other design docs
- Note when implementation is guided primarily by PRD

## Benefits

1. **Flexibility**: Can generate tasks at any stage of design completion
2. **Transparency**: Users see exactly what's missing and what that means
3. **User Control**: Explicit confirmation before proceeding with partial docs
4. **Adaptability**: Task detail level matches available documentation
5. **Workflow Support**: Supports iterative and waterfall approaches
6. **Feature-Type Awareness**: Recognizes that DevOps features don't need frontend specs

## Task Detail Levels

### HIGH (Full Documentation)
- All design docs present
- Implementation-ready tasks with specific contracts
- Contract validation enforced
- Detailed integration specifications
- Comprehensive validation gates

### MEDIUM (Partial Documentation)
- Some design docs present
- Mix of implementation and design tasks
- High-level integration guidance
- Agents have design authority in undocumented areas

### LOW (PRD Only)
- Only PRD available
- Planning and research tasks
- Design creation tasks
- High-level strategic tasks
- Significant agent autonomy

## Testing Recommendations

Test with these scenarios:

1. **Full docs feature**: E04-F01-database-schema (has all docs)
2. **Partial docs feature**: E04-F08-distribution-release (missing DB/API/Frontend)
3. **PRD only**: Create a new feature with only PRD

Expected behaviors:

1. **Full docs**: Standard 6-task sequence, contract validation first
2. **Partial docs**: Adapted task structure, user confirmation, no contract validation
3. **PRD only**: High-level planning tasks, design creation tasks

## Files Modified

1. `.claude/commands/task.md` - Command documentation
2. `.claude/skills/specification-writing/workflows/write-task.md` - Core workflow
3. `.claude/skills/specification-writing/context/task-template.md` - Task template

## Backward Compatibility

✅ **Fully backward compatible**

Features with complete documentation will generate the same task structure as before. The changes only affect:
- Features with partial documentation (now supported instead of blocked)
- User experience (now includes documentation analysis step)

## Next Steps

1. Test the workflow with E04-F08-distribution-release (partial docs)
2. Test with a new PRD-only feature
3. Verify that full-doc features still work correctly
4. Consider updating validation workflows to accept partial-doc tasks

## Success Criteria Met

✅ `/task` command works with partial documentation
✅ User gets clear feedback about what's missing
✅ User can choose to proceed or stop
✅ Generated tasks are appropriate for available documentation level
✅ Workflow is flexible and intelligent about requirements

## Notes

- The workflow maintains high standards for task quality even with partial docs
- Tasks generated from partial docs are higher-level but still actionable
- Implementation agents get clear guidance on their responsibilities
- Documentation gaps are explicitly noted in tasks
- Agents are instructed to document design decisions they make
