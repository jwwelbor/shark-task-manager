# Working with Partial Documentation

## Overview

The `/task` command now supports generating implementation tasks from partial design documentation. You don't need to complete all design documents before creating tasks.

## Quick Start

### Minimum Requirement

Only `prd.md` is required to generate tasks:

```bash
/task E04-task-mgmt-cli-core E04-F08-distribution-release
```

The workflow will:
1. Check what documentation exists
2. Show you what's available and missing
3. Explain what kinds of tasks can be generated
4. Ask for your confirmation to proceed

## What Happens with Partial Docs

### Documentation Detection

When you run `/task`, it first analyzes your feature directory:

```
Documentation Analysis for E04-F08-distribution-release:

✅ Available Documents:
- prd.md
- 02-architecture.md
- 06-security-design.md
- 07-performance-design.md
- 08-implementation-phases.md

❌ Missing Documents:
- 03-database-design.md
- 04-api-specification.md
- 05-frontend-design.md

Task Detail Level: MEDIUM
```

### User Confirmation

You'll be asked to confirm whether to proceed:

```
Recommendation:
- If this is infrastructure/DevOps work → PROCEED (no frontend/API needed)
- If this is full-stack feature → CONSIDER completing design docs first
- If you want to proceed anyway → Tasks will be high-level planning tasks

Continue with available documentation? (yes/no)
```

### Task Adaptation

Tasks are automatically adapted to your documentation level:

| Documentation Level | Task Types | Example Tasks |
|-------------------|------------|--------------|
| **PRD only** | Planning, research, design | - Research implementation approaches<br>- Create technical design docs<br>- Define architecture |
| **PRD + Architecture** | Architecture, integration | - Implement core architecture<br>- Set up integration points<br>- Create component specs |
| **PRD + Architecture + Database** | Database, data layer | - Implement database schema<br>- Create repositories<br>- Design API layer |
| **PRD + Architecture + Backend** | Backend, API | - Implement API endpoints<br>- Create service layer<br>- Design frontend contracts |
| **Full documentation** | Complete implementation | - Contract validation<br>- Full-stack implementation<br>- E2E testing |

## When to Use Partial Documentation

### Good Use Cases

1. **Infrastructure/DevOps Features**
   - Don't need frontend or API specs
   - Architecture + security/performance docs are sufficient
   - Example: E04-F08-distribution-release

2. **Iterative Development**
   - Start with PRD + architecture
   - Generate initial implementation tasks
   - Add detailed specs as you go

3. **Research-Heavy Features**
   - PRD defines the goal
   - Implementation requires research and prototyping
   - Detailed design emerges from implementation

4. **Backend-Only Features**
   - No frontend needed
   - API + database + backend design sufficient
   - Frontend can be added later

### When to Complete Full Documentation First

1. **Complex Full-Stack Features**
   - Multiple interacting components
   - Critical contract synchronization needed
   - High risk of rework without detailed specs

2. **Team Coordination Required**
   - Multiple agents/developers working in parallel
   - Need explicit contracts to prevent conflicts
   - Frontend/backend split teams

3. **Production-Critical Features**
   - High quality bar
   - Security/performance requirements strict
   - Need comprehensive validation gates

## Task Quality by Documentation Level

### High Detail (Full Documentation)

Tasks include:
- Specific contract definitions (exact DTOs, interfaces)
- Detailed validation gates
- Explicit codebase analysis references
- Contract synchronization requirements
- Comprehensive integration specifications

**Example**: "Implement CreateTaskDTO with exact fields as specified in API spec section 4.2"

### Medium Detail (Partial Documentation)

Tasks include:
- High-level requirements from PRD
- References to available design docs
- Notes on what's missing
- Guidance for implementation agents to make design decisions
- Instruction to document decisions made

**Example**: "Implement task creation API based on PRD requirements. Design and document the DTO structure."

### Low Detail (PRD Only)

Tasks include:
- Strategic goals from PRD
- Research and planning work
- Design document creation
- High-level architecture decisions

**Example**: "Research and design the task creation workflow. Create API specification document."

## File Name Variations Supported

The workflow recognizes these naming variations:

| Document Type | Variations |
|--------------|------------|
| Database | `03-database-design.md` or `03-data-design.md` |
| API/Backend | `04-api-specification.md` or `04-backend-design.md` |
| Security | `06-security-performance.md` or `06-security-design.md` |
| Performance | `07-performance-design.md` (separate from security) |

## Generated Task Characteristics

### With Full Documentation

```markdown
### Contract Specifications

Reference the exact contract definitions:
- **DTOs to Implement**: See [API Spec - DTO Definitions](../04-api-specification.md#dto-definitions)
- **Field Names**: Must match EXACTLY as specified in API spec
```

### With Partial Documentation

```markdown
### Contract Specifications

**API specification is missing**, note in task:
- **Contract Definition Required**: Implementation agent must define DTOs/interfaces
- **Documentation Requirement**: Agent must document contract definitions
- **Recommendation**: Consider creating API specification before implementation
```

## Common Workflows

### Workflow 1: Infrastructure Feature (No Frontend)

```bash
# 1. Create PRD
/prd E04-task-mgmt-cli-core E04-F08-distribution-release

# 2. Create architecture + security docs (no DB/API/Frontend needed)
# Use appropriate design workflows

# 3. Generate tasks from partial docs
/task E04-task-mgmt-cli-core E04-F08-distribution-release

# Result: DevOps-focused tasks (infrastructure, deployment, monitoring)
```

### Workflow 2: Iterative Full-Stack Feature

```bash
# 1. Create PRD
/prd E04-task-mgmt-cli-core E04-F10-new-feature

# 2. Generate initial planning tasks
/task E04-task-mgmt-cli-core E04-F10-new-feature
# Says "yes" to proceed with PRD only

# 3. Complete first task: Create architecture document
# 4. Generate updated tasks with architecture
/task E04-task-mgmt-cli-core E04-F10-new-feature

# 5. Continue iteratively adding design docs and regenerating tasks
```

### Workflow 3: Complete Design First (Traditional)

```bash
# 1. Create all design documents
/prd E04-task-mgmt-cli-core E04-F11-critical-feature
# Create architecture, database, API, frontend, security, performance docs

# 2. Generate comprehensive implementation tasks
/task E04-task-mgmt-cli-core E04-F11-critical-feature

# Result: Full 6-task sequence with contract validation
```

## Best Practices

### 1. Be Honest About What You Have

The workflow will detect missing docs. Don't try to fake it - let it adapt to what you actually have.

### 2. Match Documentation to Feature Type

- Infrastructure? You don't need frontend specs.
- Backend API? Frontend can come later.
- Full-stack? Consider completing all design docs first.

### 3. Document as You Go

If you proceed with partial docs:
- Implementation agents will make design decisions
- Ensure those decisions are documented
- Update design docs after implementation if needed

### 4. Use Confirmation as a Decision Point

When asked to confirm, consider:
- Is this the right documentation level for this feature?
- Do I need more detail before implementation?
- Can implementation agents handle the missing pieces?

### 5. Regenerate Tasks as Docs Evolve

It's OK to:
- Generate tasks from partial docs
- Complete more design documents
- Regenerate tasks with more detail

## FAQs

**Q: Can I generate tasks with only a PRD?**
A: Yes! You'll get high-level planning and design tasks.

**Q: Will partial-doc tasks be lower quality?**
A: No - they'll be appropriate for the documentation level. They'll be more strategic and less tactical.

**Q: Can I add design docs later and regenerate tasks?**
A: Yes! Generate tasks whenever you want, with whatever docs you have.

**Q: What if I have some docs but they're incomplete?**
A: The workflow checks if docs exist, not if they're complete. Incomplete docs are better than no docs.

**Q: Will contract validation still work?**
A: Only if both API and Frontend docs exist. Otherwise it's skipped gracefully.

**Q: Can I skip docs for DevOps features?**
A: Absolutely! Many DevOps features don't need API or frontend specs.

**Q: What happens if I say "no" to the confirmation?**
A: Task generation stops. You can complete more design docs and try again.

**Q: Does this work with the shark CLI?**
A: Yes! Tasks are created in the database regardless of documentation level.

## Summary

The `/task` workflow is now **documentation-level aware** and **adaptive**:

- **Minimum**: PRD required
- **Recommended**: Add design docs for more detailed tasks
- **Flexible**: Works with any combination of available docs
- **Transparent**: Shows you what's missing and what that means
- **User-controlled**: You decide whether to proceed

Choose the documentation level that matches your feature type and development approach.
