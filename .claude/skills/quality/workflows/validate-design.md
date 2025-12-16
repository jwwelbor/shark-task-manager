# Workflow: Validate Feature Design Documentation

## Purpose

Verify that all required design documents for a feature are present and complete before proceeding to PRP generation. This ensures `prp-generator` has all the context it needs.

## Usage

This workflow is invoked when validating feature design documentation:
- By `/validate-feature-design {epic} {feature}` command
- By feature-architect agent to validate its own output
- Before invoking prp-generator

## What This Workflow Checks

### 1. Required Files Exist

Verify all design documents are present in `/docs/plan/{epic}/{feature}/`:

- `README.md` - Navigation hub and overview
- `prd.md` - Feature PRD
- `00-research-report.md` - Project research findings
- `01-api-contracts.md` - Shared DTOs and endpoint contracts
- `02-architecture.md` - System design and integration
- `03-database-design.md` - Schema and data model
- `04-api-specification.md` - API endpoints and contracts
- `05-frontend-design.md` - UI components and state
- `06-security-performance.md` - Security and optimization
- `07-implementation-phases.md` - Timeline and phases
- `08-test-criteria.md` - Test specifications
- `prps/README.md` - PRP placeholder

### 2. File Completeness Check

For each design document, verify required sections are present. See `../context/design-validation-criteria.md` for detailed section requirements.

### 3. Cross-Reference Validation

Check that documents properly reference each other:
- README.md links to all design documents
- Design docs use relative paths
- No broken internal links

### 4. Anti-Pattern Detection

Flag issues that violate feature-architect guidelines:
- **No code implementation** - Check for SQL, Python, TypeScript, etc.
- **No placeholders** - Check for "TODO", "TBD", "[to be completed]"
- **Mermaid diagrams present** - Architecture and database sections have diagrams
- **No PRPs created** - prps/ folder should only have placeholder README
- **Descriptions, not code** - All specs should be prose, not implementation

## Execution Steps

### Step 1: Validate File Existence

Use Read tool to check each file exists. Run all reads in parallel for speed.

### Step 2: Content Analysis

For each file that exists, analyze content for:
- Required sections present (from design-validation-criteria.md)
- Appropriate length (with ±20% tolerance)
- No code implementation
- No placeholders/TODOs
- Mermaid diagrams where required

### Step 3: Generate Validation Report

Create a validation report at `/docs/plan/{epic}/{feature}/validation-report.md` with:

- Summary (files found, issues, warnings)
- Detailed results for each file
- Anti-pattern detection results
- Cross-reference check results
- Recommendations
- Ready/Not Ready determination

### Step 4: Return Summary

Output a concise summary to the user with:
- Status (PASS/PASS WITH WARNINGS/FAIL)
- Files validated
- Issues found (errors and warnings)
- Link to full report
- Next steps

## Success Criteria

Validation passes when:
1. All required files exist
2. All required sections present in each file
3. No code implementation found (only descriptions)
4. No placeholder text (TODO, TBD, etc.)
5. Mermaid diagrams present where required
6. No PRPs created yet
7. All cross-references and links are valid
8. File lengths within acceptable ranges

## Common Issues

- **Missing Mermaid Diagrams**: Add diagrams to architecture and database docs
- **SQL Code Found**: Replace with prose descriptions
- **TODO/TBD Placeholders**: Complete sections or remove placeholders
- **PRPs Already Created**: Delete PRPs - only placeholder README should exist
- **File Too Short/Long**: Adjust detail level (±20% of target)

## Output Format

The validation report should clearly indicate:
- ✅ Items that pass
- ⚠️ Items with warnings
- ❌ Items that fail
- Specific line numbers for issues
- Actionable recommendations for fixing

See `../context/design-validation-criteria.md` for complete validation criteria.
