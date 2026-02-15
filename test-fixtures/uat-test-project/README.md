# UAT Test Project

This is a test shark project used for User Acceptance Testing (UAT) and demonstration purposes.

## Setup

This test project has been pre-configured with:
- **Epic E01:** "Test Epic for UAT"
- **Feature E01-F01:** "Test Feature with Related Docs"
- **Task T-E01-F01-001:** "Implement authentication system" (backend agent)
- **3 Related Documents** linked to the task:
  - `docs/design/authentication-spec.md`
  - `docs/design/api-design.md`
  - `docs/testing/test-plan.md`

## Usage

Run shark commands from this directory to see actual CLI behavior with test data:

```bash
cd test-fixtures/uat-test-project

# View task with related documents
shark task get T-E01-F01-001 --json | jq '.related_documents'

# See related_docs as comma-separated paths (matches {related_docs} placeholder format)
shark task get T-E01-F01-001 --json | jq -r '.related_documents | map(.file_path) | join(",")'

# View full task details
shark task get T-E01-F01-001

# List all tasks
shark task list

# View feature
shark feature get E01-F01

# View epic
shark epic get E01
```

## Re-creating Test Data

If you need to recreate the test data from scratch:

```bash
# 1. Delete existing database
rm shark-tasks.db

# 2. Reinitialize
shark init --non-interactive

# 3. Create test entities
shark epic create "Test Epic for UAT" --key=E01
shark feature create E01 "Test Feature with Related Docs"
shark task create E01 F01 "Implement authentication system" --agent=backend

# 4. Add test documents (run from shark project root)
go run test-fixtures/uat-test-project/add_test_data.go
```

## Use Cases

- **UAT Testing:** Demonstrate E07-F29 related documents/tasks functionality
- **Integration Testing:** Verify CLI commands work with real database
- **Documentation:** Generate screenshots and examples for user guides
- **Development:** Quick test environment for trying new shark features

## Database Location

- Database: `./shark-tasks.db` (SQLite)
- Config: `./.sharkconfig.json`
- Docs: `./docs/plan/`

## Notes

- This is a **test fixture** - safe to delete and recreate
- The database file (`shark-tasks.db`) can be committed to source control for reproducible tests
- The `add_test_data.go` script demonstrates how to programmatically add documents to the database
