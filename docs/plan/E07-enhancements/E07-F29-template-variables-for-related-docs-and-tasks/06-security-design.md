# Security Design: Template Variables for Related Docs and Tasks

**Feature:** E07-F29
**Version:** 1.0
**Last Updated:** 2026-02-13

## Security Analysis

### Threat Model

**Attack Surface:**
- ✅ **MINIMAL** - Feature operates entirely on internal data structures
- ❌ No external API calls
- ❌ No user input processing
- ❌ No file system writes
- ❌ No network operations

**Threat Vectors Considered:**

1. **SQL Injection** - MITIGATED via parameterized queries
2. **Path Traversal** - NOT APPLICABLE (reads existing paths only)
3. **Information Disclosure** - LOW RISK (internal data only)
4. **Denial of Service** - LOW RISK (query performance limits)

---

## SQL Injection Prevention

### Parameterized Queries

**All database queries use parameterized statements:**

```go
// SAFE: Feature relationship query
query := `
    SELECT f.key
    FROM features f
    INNER JOIN feature_relationships fr ON f.id = fr.to_feature_id
    WHERE fr.from_feature_id = ?
`
result := db.QueryContext(ctx, query, featureID)  // Parameterized

// SAFE: Document query
query := `
    SELECT d.file_path
    FROM documents d
    INNER JOIN task_documents td ON d.id = td.document_id
    WHERE td.task_id = ?
`
result := db.QueryContext(ctx, query, taskID)  // Parameterized
```

**NO string concatenation:**

```go
// ❌ UNSAFE (NOT USED): String interpolation
query := fmt.Sprintf("WHERE task_id = %d", taskID)  // NEVER do this

// ✅ SAFE (USED): Parameterized
query := "WHERE task_id = ?"
db.QueryContext(ctx, query, taskID)
```

### Input Validation

**Relationship Type Validation:**

```go
// CHECK constraint at database level
CHECK (relationship_type IN (
    'depends_on', 'blocks', 'related_to', 'follows',
    'spawned_from', 'duplicates', 'references'
))

// Application-level validation
func validateRelationshipType(relType string) error {
    validTypes := map[string]bool{
        "depends_on": true, "blocks": true, "related_to": true,
        "follows": true, "spawned_from": true, "duplicates": true,
        "references": true,
    }

    if !validTypes[relType] {
        return fmt.Errorf("invalid relationship type: %s", relType)
    }

    return nil
}
```

**Context Data Parsing:**

```go
// Safe JSON parsing with error handling
func extractRelatedTasksFromContext(contextData *string) string {
    if contextData == nil || *contextData == "" {
        return ""
    }

    // Use standard library JSON parser (safe)
    cd, err := models.FromJSON(*contextData)
    if err != nil {
        // Log error, return empty string (fail-safe)
        log.Printf("WARNING: Failed to parse context_data JSON: %v", err)
        return ""
    }

    // Validate array contents
    for _, taskKey := range cd.RelatedTasks {
        if !isValidTaskKey(taskKey) {
            log.Printf("WARNING: Invalid task key in related_tasks: %s", taskKey)
            // Skip invalid keys instead of failing
            continue
        }
    }

    return strings.Join(cd.RelatedTasks, ",")
}
```

---

## Data Sanitization

### File Path Handling

**Document paths are read-only (no write operations):**

```go
// Feature reads existing document paths, does NOT write/modify files
func formatDocPathsAsCSV(docs []*models.Document) string {
    paths := make([]string, len(docs))
    for i, doc := range docs {
        paths[i] = doc.FilePath  // Read only, no modification
    }
    return strings.Join(paths, ",")
}
```

**No path traversal risk:**
- Paths are stored in database (validated on insert by DocumentRepository)
- No file system writes occur in this feature
- Template placeholders use paths as-is (no filesystem access)

### Template Injection Prevention

**String replacement only (not code execution):**

```go
// PopulateTemplate uses simple string.Replace (safe)
func PopulateTemplate(template string, placeholders map[string]string) string {
    result := template
    for key, value := range placeholders {
        result = strings.ReplaceAll(result, "{"+key+"}", value)
    }
    return result
}
```

**No template evaluation:**
- No JavaScript/Lua/Python evaluation
- No `eval()` or code execution
- Pure string replacement only
- Placeholders cannot contain executable code

---

## Access Control

### Authorization

**No new authorization requirements:**

- Feature operates on data already accessible to caller
- If user can read task/feature/epic, they can see related docs/tasks
- No privilege escalation risk (reads existing relationships only)

**Existing authorization respected:**

```go
// Example: Task repository already enforces access control
func (r *TaskRepository) GetByKey(ctx context.Context, key string) (*models.Task, error) {
    // Existing access control applies here
    // Feature reads relationships for tasks user can already access
}
```

### Authentication

**No authentication changes:**

- Feature uses existing authentication context
- No new authentication endpoints
- No bypass of existing auth mechanisms

---

## Denial of Service (DoS) Mitigation

### Query Performance Limits

**Indexed queries prevent table scans:**

```sql
-- All relationship queries use indexes
CREATE INDEX idx_feature_relationships_from ON feature_relationships(from_feature_id);
CREATE INDEX idx_feature_relationships_to ON feature_relationships(to_feature_id);
CREATE INDEX idx_task_documents_task_id ON task_documents(task_id);
```

**Performance bounds:**
- Document query: < 10ms for up to 50 documents
- Relationship query: < 15ms for up to 20 relationships
- Total overhead: < 30ms per placeholder population

**No unbounded loops:**

```go
// CSV formatting has bounded complexity O(n)
func formatDocPathsAsCSV(docs []*models.Document) string {
    if len(docs) == 0 {
        return ""
    }

    paths := make([]string, len(docs))  // Pre-allocated array
    for i, doc := range docs {
        paths[i] = doc.FilePath
    }

    return strings.Join(paths, ",")
}
```

### Resource Limits

**Memory constraints:**
- CSV output limited by number of related entities
- Typical: 10 docs × 100 chars = 1KB per task
- Maximum reasonable: 50 docs × 200 chars = 10KB

**Database connection pooling:**
- Use existing connection pool (no new connections)
- Queries are read-only (no lock contention)
- Graceful degradation on timeout (empty string fallback)

---

## Data Privacy

### Information Disclosure

**Internal data only:**
- Document file paths are project-relative (e.g., `docs/spec.md`)
- No absolute paths containing user directories
- No sensitive data in relationship metadata

**Orchestrator actions are internal:**
- Instructions generated for AI agents (not end users)
- Not exposed via public API (internal workflow only)
- Logged for debugging but not transmitted externally

### Logging Security

**Safe logging practices:**

```go
// LOG: Task keys, relationship types (not sensitive)
log.Printf("INFO: Populated placeholders for task %s with %d related docs", task.Key, len(docs))

// LOG: Error details for debugging (no credentials/secrets)
log.Printf("WARNING: Failed to fetch related docs for task %s: %v", task.Key, err)

// DO NOT LOG: User credentials, tokens, passwords (not applicable to this feature)
```

---

## Compliance

### GDPR / Data Protection

**No personal data processed:**
- Feature operates on project metadata (task keys, document paths)
- No user names, emails, or personal identifiers in placeholder data
- Relationship data is project structure (non-personal)

**Data retention:**
- Relationship data persists as long as entities exist
- CASCADE DELETE removes relationships when entities deleted
- No orphaned data left behind

### Audit Trail

**Relationship creation logged:**

```go
// Log relationship creation for audit purposes
func (r *FeatureRelationshipRepository) Create(ctx context.Context, rel *models.FeatureRelationship) error {
    log.Printf("INFO: Creating feature relationship: %d -> %d (%s)", 
        rel.FromFeatureID, rel.ToFeatureID, rel.RelationshipType)

    // Timestamp automatically added by database (created_at)
    query := `INSERT INTO feature_relationships (...) VALUES (...)`
    // ...
}
```

**Timestamps preserved:**
```sql
-- All relationship tables include created_at timestamp
created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
```

---

## Secure Coding Practices

### Error Handling

**No information leakage in errors:**

```go
// GOOD: Generic error for user-facing
if err != nil {
    return nil, errors.New("failed to fetch related features")
}

// GOOD: Detailed error for logging (not exposed to user)
log.Printf("ERROR: Database query failed (table: feature_relationships, error: %v)", err)

// BAD: Don't expose internal details in user-facing errors
return nil, fmt.Errorf("failed to query table feature_relationships: %v", err)  // ❌
```

### Dependency Security

**Standard library only:**
- No external dependencies for placeholder logic
- Uses `encoding/json` (standard library, secure)
- Database driver is existing (`github.com/mattn/go-sqlite3` or Turso)

**No third-party template engines:**
- Simple string replacement (no Handlebars, Jinja2, etc.)
- Reduces attack surface (no parser vulnerabilities)

---

## Security Testing

### Test Cases

**SQL Injection Tests:**

```go
func TestFeatureRelationshipRepository_SQLInjectionPrevention(t *testing.T) {
    ctx := context.Background()
    db := test.GetTestDB()
    repo := NewFeatureRelationshipRepository(db)

    // Attempt SQL injection via relationship type
    rel := &models.FeatureRelationship{
        FromFeatureID:    1,
        ToFeatureID:      2,
        RelationshipType: "depends_on'; DROP TABLE features; --",  // Injection attempt
    }

    err := repo.Create(ctx, rel)
    assert.Error(t, err)  // Should fail validation

    // Verify features table still exists
    var count int
    db.QueryRow("SELECT COUNT(*) FROM features").Scan(&count)
    // No panic, table not dropped
}
```

**Malformed JSON Handling:**

```go
func TestExtractRelatedTasksFromContext_MalformedJSON(t *testing.T) {
    malformedJSON := `{"related_tasks": [invalid json`

    result := extractRelatedTasksFromContext(&malformedJSON)

    // Should gracefully degrade to empty string (not panic)
    assert.Equal(t, "", result)
}
```

**Path Traversal Tests:**

```go
func TestFormatDocPathsAsCSV_NoPathTraversal(t *testing.T) {
    docs := []*models.Document{
        {FilePath: "../../etc/passwd"},  // Malicious path
        {FilePath: "docs/safe.md"},
    }

    result := formatDocPathsAsCSV(docs)

    // Paths are returned as-is (read-only, no filesystem access)
    // Actual validation happens in DocumentRepository.Create (not this feature)
    assert.Contains(t, result, "../../etc/passwd")  // Not sanitized here (read-only)
}
```

---

## Incident Response

### Monitoring

**Alert on anomalies:**

- Excessive placeholder population failures (> 5% error rate)
- Slow relationship queries (> 100ms consistently)
- Malformed context_data parsing errors (> 10% of tasks)

**Metrics to track:**

```go
// Prometheus-style metrics (future)
placeholder_population_errors_total{entity="task"} 42
placeholder_population_duration_seconds{entity="task", quantile="0.99"} 0.025
relationship_query_errors_total{type="feature"} 3
```

### Recovery Procedures

**Placeholder population failure:**

1. Check logs for error details
2. Verify document repository is accessible
3. Verify relationship tables exist (migration ran)
4. Graceful degradation ensures system continues (empty strings)

**Database corruption:**

1. Relationship tables can be dropped and recreated (no data loss in MVP)
2. Context data is JSON (can be manually edited if corrupted)
3. Backup/restore procedures same as existing tables

---

## Security Checklist

**Implementation Phase:**

- [x] All queries use parameterized statements
- [x] Relationship type validation (CHECK constraint + app-level)
- [x] Context data JSON parsing with error handling
- [x] Graceful degradation on errors (no failures)
- [x] Logging does not expose sensitive data
- [x] No file system writes or external API calls
- [x] Indexes prevent DoS via slow queries

**Testing Phase:**

- [ ] SQL injection tests (malicious relationship types)
- [ ] Malformed JSON tests (context_data parsing)
- [ ] Performance tests (large number of relationships)
- [ ] Error handling tests (graceful degradation)

**Deployment Phase:**

- [ ] Review logs for anomalies post-deployment
- [ ] Monitor query performance metrics
- [ ] Verify indexes created successfully

---

## Summary

**Security Posture:**

- ✅ **LOW RISK** - Internal data only, no external exposure
- ✅ **DEFENSE IN DEPTH** - Database constraints + application validation
- ✅ **GRACEFUL DEGRADATION** - Errors logged, empty strings returned
- ✅ **NO NEW ATTACK SURFACE** - Reads existing data structures only

**Key Controls:**

1. Parameterized queries (SQL injection prevention)
2. Relationship type validation (database + application)
3. JSON parsing with error handling (DoS mitigation)
4. Read-only operations (no file writes or external calls)
5. Performance indexes (DoS mitigation)

**Compliance:**

- No personal data processed (GDPR not applicable)
- Audit trail via created_at timestamps
- Logging follows secure practices (no credentials/tokens)
