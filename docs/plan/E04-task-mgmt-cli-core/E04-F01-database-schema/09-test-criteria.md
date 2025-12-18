# Test Criteria: Database Schema & Core Data Model

**Epic**: E04-task-mgmt-cli-core
**Feature**: E04-F01-database-schema
**Date**: 2025-12-14
**Author**: tdd-agent

## Purpose

This document defines comprehensive test criteria for validating the database layer implementation. It covers unit tests, integration tests, performance tests, security tests, and acceptance tests aligned with PRD requirements.

---

## Test Coverage Requirements

**From Epic NFR**: >80% unit test coverage

**Target Coverage**:
- `models.py`: 90% (ORM models)
- `validation.py`: 95% (critical validation logic)
- `session.py`: 85% (session management)
- `repositories.py`: 90% (data access layer)
- `exceptions.py`: 100% (simple exception classes)
- `config.py`: 80% (configuration)

**Overall Target**: >85% code coverage

---

## Unit Tests

### 1. ORM Models (`tests/unit/database/test_models.py`)

#### Test: Epic Model

```python
def test_epic_model_creation():
    """Test Epic model can be instantiated with required fields"""
    epic = Epic(
        key="E04",
        title="Test Epic",
        status="active",
        priority="high"
    )
    assert epic.key == "E04"
    assert epic.title == "Test Epic"
    assert epic.status == "active"
    assert epic.priority == "high"

def test_epic_to_dict():
    """Test Epic serialization to dictionary"""
    epic = Epic(
        id=1,
        key="E04",
        title="Test Epic",
        status="active",
        priority="high",
        created_at=datetime(2025, 12, 14, 10, 0, 0, tzinfo=timezone.utc),
        updated_at=datetime(2025, 12, 14, 10, 0, 0, tzinfo=timezone.utc)
    )
    data = epic.to_dict()

    assert data["id"] == 1
    assert data["key"] == "E04"
    assert data["created_at"] == "2025-12-14T10:00:00Z"

def test_epic_relationships():
    """Test Epic → Features relationship loads correctly"""
    epic = Epic(key="E04", title="Test", status="active", priority="high")
    feature = Feature(epic=epic, key="E04-F01", title="Test", status="active")

    assert feature in epic.features
    assert feature.epic == epic
```

#### Test: Feature Model

```python
def test_feature_model_creation():
    """Test Feature model with all fields"""
    feature = Feature(
        epic_id=1,
        key="E04-F01",
        title="Test Feature",
        status="active",
        progress_pct=42.5
    )
    assert feature.epic_id == 1
    assert feature.key == "E04-F01"
    assert feature.progress_pct == 42.5

def test_feature_epic_key_property():
    """Test feature.epic_key computed property"""
    feature = Feature(epic_id=1, key="E04-F01", title="Test", status="active")
    assert feature.epic_key == "E04"
```

#### Test: Task Model

```python
def test_task_model_creation():
    """Test Task model with all fields"""
    task = Task(
        feature_id=1,
        key="T-E04-F01-001",
        title="Test Task",
        status="todo",
        priority=5
    )
    assert task.key == "T-E04-F01-001"
    assert task.status == "todo"
    assert task.priority == 5

def test_task_dependencies_property():
    """Test task.dependencies property parses JSON"""
    task = Task(
        feature_id=1,
        key="T-E04-F01-001",
        title="Test",
        status="todo",
        depends_on='["T-E01-F01-001", "T-E01-F02-003"]'
    )
    assert task.dependencies == ["T-E01-F01-001", "T-E01-F02-003"]

def test_task_dependencies_setter():
    """Test task.dependencies setter converts to JSON"""
    task = Task(feature_id=1, key="T-E04-F01-001", title="Test", status="todo")
    task.dependencies = ["T-E01-F01-001"]

    assert task.depends_on == '["T-E01-F01-001"]'

def test_task_feature_key_property():
    """Test task.feature_key computed property"""
    task = Task(feature_id=1, key="T-E04-F01-001", title="Test", status="todo")
    assert task.feature_key == "E04-F01"

def test_task_epic_key_property():
    """Test task.epic_key computed property"""
    task = Task(feature_id=1, key="T-E04-F01-001", title="Test", status="todo")
    assert task.epic_key == "E04"
```

#### Test: TaskHistory Model

```python
def test_task_history_model_creation():
    """Test TaskHistory model"""
    history = TaskHistory(
        task_id=1,
        old_status="todo",
        new_status="in_progress",
        agent="claude"
    )
    assert history.task_id == 1
    assert history.old_status == "todo"
    assert history.new_status == "in_progress"
    assert history.agent == "claude"
```

**Total Model Tests**: ~20 tests

---

### 2. Validation (`tests/unit/database/test_validation.py`)

#### Test: Key Validation

```python
def test_validate_epic_key_valid():
    """Test valid epic keys pass validation"""
    validate_epic_key("E01")  # Should not raise
    validate_epic_key("E99")  # Should not raise

def test_validate_epic_key_invalid():
    """Test invalid epic keys raise ValidationError"""
    with pytest.raises(ValidationError, match="Invalid epic key format"):
        validate_epic_key("E1")  # Too short

    with pytest.raises(ValidationError):
        validate_epic_key("E001")  # Too long

    with pytest.raises(ValidationError):
        validate_epic_key("Epic01")  # Wrong format

def test_validate_feature_key_valid():
    """Test valid feature keys pass validation"""
    validate_feature_key("E04-F01")  # Should not raise

def test_validate_feature_key_with_epic_match():
    """Test feature key matches epic key"""
    validate_feature_key("E04-F01", epic_key="E04")  # Should not raise

def test_validate_feature_key_epic_mismatch():
    """Test feature key epic mismatch raises error"""
    with pytest.raises(ValidationError, match="does not belong to epic"):
        validate_feature_key("E04-F01", epic_key="E05")

def test_validate_task_key_valid():
    """Test valid task keys pass validation"""
    validate_task_key("T-E04-F01-001")  # Should not raise

def test_validate_task_key_feature_match():
    """Test task key matches feature key"""
    validate_task_key("T-E04-F01-001", feature_key="E04-F01")  # Should not raise

def test_validate_task_key_feature_mismatch():
    """Test task key feature mismatch raises error"""
    with pytest.raises(ValidationError, match="does not belong to feature"):
        validate_task_key("T-E04-F01-001", feature_key="E04-F02")
```

#### Test: Enum Validation

```python
def test_validate_task_status_valid():
    """Test valid task statuses"""
    for status in ["todo", "in_progress", "blocked", "ready_for_review", "completed", "archived"]:
        validate_task_status(status)  # Should not raise

def test_validate_task_status_invalid():
    """Test invalid task status raises error"""
    with pytest.raises(ValidationError, match="Invalid task status"):
        validate_task_status("invalid_status")

def test_validate_agent_type_null():
    """Test agent_type can be None"""
    validate_agent_type(None)  # Should not raise

def test_validate_task_priority_valid():
    """Test valid priorities"""
    validate_task_priority(1)  # Highest
    validate_task_priority(5)  # Default
    validate_task_priority(10)  # Lowest

def test_validate_task_priority_invalid():
    """Test invalid priorities"""
    with pytest.raises(ValidationError, match="must be 1-10"):
        validate_task_priority(0)

    with pytest.raises(ValidationError):
        validate_task_priority(11)
```

#### Test: JSON Validation

```python
def test_validate_depends_on_valid():
    """Test valid dependency JSON"""
    deps = validate_depends_on('["T-E01-F01-001", "T-E01-F02-003"]')
    assert deps == ["T-E01-F01-001", "T-E01-F02-003"]

def test_validate_depends_on_empty():
    """Test empty dependencies"""
    assert validate_depends_on(None) == []
    assert validate_depends_on("") == []
    assert validate_depends_on("[]") == []

def test_validate_depends_on_invalid_json():
    """Test invalid JSON raises error"""
    with pytest.raises(ValidationError, match="Invalid JSON"):
        validate_depends_on('["T-E01-F01-001"')  # Missing closing bracket

def test_validate_depends_on_not_array():
    """Test non-array JSON raises error"""
    with pytest.raises(ValidationError, match="must be a JSON array"):
        validate_depends_on('{"key": "value"}')  # Object, not array

def test_validate_depends_on_invalid_task_keys():
    """Test invalid task key in dependencies"""
    with pytest.raises(ValidationError, match="Invalid task key format"):
        validate_depends_on('["INVALID-KEY"]')
```

**Total Validation Tests**: ~30 tests

---

### 3. Session Management (`tests/unit/database/test_session.py`)

#### Test: Session Factory

```python
def test_session_factory_creation():
    """Test SessionFactory creates engine and session maker"""
    config = DatabaseConfig(db_path=Path(":memory:"))
    factory = SessionFactory(config)

    assert factory.engine is not None
    assert factory.SessionLocal is not None

def test_get_session_context_manager():
    """Test get_session context manager commits on success"""
    factory = SessionFactory(DatabaseConfig(db_path=Path(":memory:")))
    factory.create_all_tables()

    with factory.get_session() as session:
        epic = Epic(key="E01", title="Test", status="active", priority="high")
        session.add(epic)
    # Should auto-commit

    with factory.get_session() as session:
        result = session.query(Epic).filter_by(key="E01").first()
        assert result is not None

def test_get_session_rollback_on_exception():
    """Test get_session rolls back on exception"""
    factory = SessionFactory(DatabaseConfig(db_path=Path(":memory:")))
    factory.create_all_tables()

    with pytest.raises(Exception, match="Test error"):
        with factory.get_session() as session:
            epic = Epic(key="E01", title="Test", status="active", priority="high")
            session.add(epic)
            raise Exception("Test error")

    # Verify rollback: epic should not exist
    with factory.get_session() as session:
        result = session.query(Epic).filter_by(key="E01").first()
        assert result is None

def test_create_all_tables():
    """Test create_all_tables creates schema"""
    factory = SessionFactory(DatabaseConfig(db_path=Path(":memory:")))
    factory.create_all_tables()

    with factory.get_session() as session:
        # Verify tables exist by querying
        session.query(Epic).count()  # Should not raise
        session.query(Feature).count()
        session.query(Task).count()
        session.query(TaskHistory).count()

def test_check_integrity():
    """Test check_integrity returns True for valid database"""
    factory = SessionFactory(DatabaseConfig(db_path=Path(":memory:")))
    factory.create_all_tables()

    assert factory.check_integrity() is True

def test_get_schema_version_not_initialized():
    """Test get_schema_version returns None before Alembic"""
    factory = SessionFactory(DatabaseConfig(db_path=Path(":memory:")))
    factory.create_all_tables()

    assert factory.get_schema_version() is None
```

**Total Session Tests**: ~10 tests

---

### 4. Repositories (`tests/unit/database/test_repositories.py`)

#### Test: EpicRepository

```python
def test_epic_repository_create(in_memory_session):
    """Test creating epic via repository"""
    repo = EpicRepository(in_memory_session)

    epic = repo.create(
        key="E04",
        title="Test Epic",
        status="active",
        priority="high"
    )

    assert epic.id is not None
    assert epic.key == "E04"
    assert epic.created_at is not None

def test_epic_repository_create_validates_key(in_memory_session):
    """Test create validates epic key format"""
    repo = EpicRepository(in_memory_session)

    with pytest.raises(ValidationError, match="Invalid epic key format"):
        repo.create(key="INVALID", title="Test", status="active", priority="high")

def test_epic_repository_get_by_key(in_memory_session):
    """Test get epic by key"""
    repo = EpicRepository(in_memory_session)
    repo.create(key="E04", title="Test", status="active", priority="high")

    epic = repo.get_by_key("E04")
    assert epic is not None
    assert epic.key == "E04"

def test_epic_repository_get_by_key_not_found(in_memory_session):
    """Test get_by_key returns None if not found"""
    repo = EpicRepository(in_memory_session)

    epic = repo.get_by_key("E99")
    assert epic is None

def test_epic_repository_list_all(in_memory_session):
    """Test listing all epics"""
    repo = EpicRepository(in_memory_session)
    repo.create(key="E01", title="Epic 1", status="active", priority="high")
    repo.create(key="E02", title="Epic 2", status="draft", priority="medium")

    epics = repo.list_all()
    assert len(epics) == 2

def test_epic_repository_list_all_filtered(in_memory_session):
    """Test listing epics filtered by status"""
    repo = EpicRepository(in_memory_session)
    repo.create(key="E01", title="Epic 1", status="active", priority="high")
    repo.create(key="E02", title="Epic 2", status="draft", priority="medium")

    active_epics = repo.list_all(status="active")
    assert len(active_epics) == 1
    assert active_epics[0].key == "E01"

def test_epic_repository_update(in_memory_session):
    """Test updating epic"""
    repo = EpicRepository(in_memory_session)
    epic = repo.create(key="E04", title="Test", status="draft", priority="high")

    updated = repo.update(epic.id, status="active", title="Updated Title")
    assert updated.status == "active"
    assert updated.title == "Updated Title"

def test_epic_repository_delete(in_memory_session):
    """Test deleting epic"""
    repo = EpicRepository(in_memory_session)
    epic = repo.create(key="E04", title="Test", status="active", priority="high")

    repo.delete(epic.id)

    assert repo.get_by_key("E04") is None

def test_epic_repository_calculate_progress(in_memory_session):
    """Test epic progress calculation"""
    epic_repo = EpicRepository(in_memory_session)
    feature_repo = FeatureRepository(in_memory_session)

    epic = epic_repo.create(key="E04", title="Test", status="active", priority="high")
    feature_repo.create(epic_id=epic.id, key="E04-F01", title="F1", status="active")
    feature_repo.update_progress(1)  # Set progress to some value

    progress = epic_repo.calculate_progress(epic.id)
    assert 0.0 <= progress <= 100.0
```

#### Test: TaskRepository

```python
def test_task_repository_create(in_memory_session):
    """Test creating task"""
    task_repo = TaskRepository(in_memory_session)

    task = task_repo.create(
        feature_id=1,
        key="T-E04-F01-001",
        title="Test Task",
        status="todo"
    )

    assert task.id is not None
    assert task.key == "T-E04-F01-001"
    assert task.status == "todo"
    assert task.priority == 5  # Default

def test_task_repository_filter_combined(in_memory_session):
    """Test combined filtering"""
    task_repo = TaskRepository(in_memory_session)

    # Create mix of tasks
    task_repo.create(feature_id=1, key="T-E04-F01-001", title="T1", status="todo", agent_type="backend", priority=3)
    task_repo.create(feature_id=1, key="T-E04-F01-002", title="T2", status="todo", agent_type="frontend", priority=5)
    task_repo.create(feature_id=1, key="T-E04-F01-003", title="T3", status="in_progress", agent_type="backend", priority=2)

    # Filter: status=todo, agent_type=backend, priority<=3
    filtered = task_repo.filter_combined(
        status="todo",
        agent_type="backend",
        max_priority=3
    )

    assert len(filtered) == 1
    assert filtered[0].key == "T-E04-F01-001"

def test_task_repository_update_status_atomic(in_memory_session):
    """Test atomic status update creates history"""
    task_repo = TaskRepository(in_memory_session)
    history_repo = TaskHistoryRepository(in_memory_session)

    task = task_repo.create(feature_id=1, key="T-E04-F01-001", title="Test", status="todo")

    updated = task_repo.update_status(task.id, "in_progress", agent="claude", notes="Starting work")

    assert updated.status == "in_progress"
    assert updated.started_at is not None

    # Verify history entry created
    history = history_repo.list_by_task(task.id)
    assert len(history) == 1
    assert history[0].old_status == "todo"
    assert history[0].new_status == "in_progress"
    assert history[0].agent == "claude"
```

**Total Repository Tests**: ~50 tests

---

## Integration Tests

### 5. End-to-End Operations (`tests/integration/test_database_operations.py`)

#### Test: Complete Epic Creation Flow

```python
def test_create_epic_with_features_and_tasks(temp_db):
    """Test creating epic → features → tasks flow"""
    epic_repo = EpicRepository(temp_db)
    feature_repo = FeatureRepository(temp_db)
    task_repo = TaskRepository(temp_db)

    # Create epic
    epic = epic_repo.create(key="E04", title="Test Epic", status="active", priority="high")

    # Create features
    f1 = feature_repo.create(epic_id=epic.id, key="E04-F01", title="Feature 1", status="active")
    f2 = feature_repo.create(epic_id=epic.id, key="E04-F02", title="Feature 2", status="active")

    # Create tasks
    t1 = task_repo.create(feature_id=f1.id, key="T-E04-F01-001", title="Task 1", status="todo")
    t2 = task_repo.create(feature_id=f1.id, key="T-E04-F01-002", title="Task 2", status="completed")
    t3 = task_repo.create(feature_id=f2.id, key="T-E04-F02-001", title="Task 3", status="todo")

    # Verify relationships
    epic_loaded = epic_repo.get_by_id(epic.id)
    assert len(epic_loaded.features) == 2

    feature_loaded = feature_repo.get_by_id(f1.id)
    assert len(feature_loaded.tasks) == 2
```

#### Test: Cascade Delete

```python
def test_cascade_delete_epic_deletes_all_children(temp_db):
    """Test deleting epic cascades to features, tasks, history"""
    epic_repo = EpicRepository(temp_db)
    feature_repo = FeatureRepository(temp_db)
    task_repo = TaskRepository(temp_db)
    history_repo = TaskHistoryRepository(temp_db)

    # Create hierarchy
    epic = epic_repo.create(key="E04", title="Test", status="active", priority="high")
    feature = feature_repo.create(epic_id=epic.id, key="E04-F01", title="Test", status="active")
    task = task_repo.create(feature_id=feature.id, key="T-E04-F01-001", title="Test", status="todo")
    history = history_repo.create(task_id=task.id, new_status="todo")

    # Delete epic
    epic_repo.delete(epic.id)

    # Verify all children deleted
    assert feature_repo.get_by_id(feature.id) is None
    assert task_repo.get_by_id(task.id) is None
    assert len(history_repo.list_by_task(task.id)) == 0
```

#### Test: Foreign Key Constraints

```python
def test_foreign_key_prevents_orphan_feature(temp_db):
    """Test cannot create feature with invalid epic_id"""
    feature_repo = FeatureRepository(temp_db)

    with pytest.raises(IntegrityError, match="references non-existent epic"):
        feature_repo.create(epic_id=9999, key="E04-F01", title="Test", status="active")

def test_foreign_key_prevents_orphan_task(temp_db):
    """Test cannot create task with invalid feature_id"""
    task_repo = TaskRepository(temp_db)

    with pytest.raises(IntegrityError, match="references non-existent feature"):
        task_repo.create(feature_id=9999, key="T-E04-F01-001", title="Test", status="todo")
```

#### Test: Transaction Rollback

```python
def test_transaction_rollback_on_error(temp_db):
    """Test multi-step operation rolls back on error"""
    task_repo = TaskRepository(temp_db)
    feature_repo = FeatureRepository(temp_db)

    # Create feature and task
    feature = feature_repo.create(epic_id=1, key="E04-F01", title="Test", status="active")
    task = task_repo.create(feature_id=feature.id, key="T-E04-F01-001", title="Test", status="todo")

    # Attempt multi-step update in transaction
    with pytest.raises(Exception):
        with temp_db.begin():
            task_repo.update_status(task.id, "in_progress")
            feature_repo.update_progress(feature.id)
            raise Exception("Simulated error")  # Force rollback

    # Verify rollback: task status unchanged
    task_reloaded = task_repo.get_by_id(task.id)
    assert task_reloaded.status == "todo"  # Not "in_progress"
```

#### Test: Unique Constraint Violations

```python
def test_unique_constraint_epic_key(temp_db):
    """Test cannot create duplicate epic keys"""
    epic_repo = EpicRepository(temp_db)

    epic_repo.create(key="E04", title="Test 1", status="active", priority="high")

    with pytest.raises(IntegrityError, match="key 'E04' already exists"):
        epic_repo.create(key="E04", title="Test 2", status="active", priority="high")
```

**Total Integration Tests**: ~15 tests

---

## Performance Tests

### 6. Performance Benchmarks (`tests/performance/test_database_performance.py`)

#### Test: PRD Performance Requirements

```python
def test_database_initialization_performance(tmp_path):
    """PRD: Database initialization <500ms"""
    db_path = tmp_path / "perf_test.db"
    config = DatabaseConfig(db_path=db_path)

    start = time.time()
    factory = SessionFactory(config)
    factory.create_all_tables()
    duration_ms = (time.time() - start) * 1000

    assert duration_ms < 500, f"Initialization took {duration_ms:.2f}ms (target: <500ms)"

def test_single_task_insert_performance(populated_db):
    """PRD: Single task INSERT <50ms"""
    task_repo = TaskRepository(populated_db)

    start = time.time()
    task_repo.create(
        feature_id=1,
        key="T-E04-F01-999",
        title="Performance Test",
        status="todo"
    )
    duration_ms = (time.time() - start) * 1000

    assert duration_ms < 50, f"Single INSERT took {duration_ms:.2f}ms (target: <50ms)"

def test_get_by_key_performance(db_with_10k_tasks):
    """PRD: get_by_key <10ms"""
    task_repo = TaskRepository(db_with_10k_tasks)

    start = time.time()
    task = task_repo.get_by_key("T-E04-F01-5000")  # Middle of dataset
    duration_ms = (time.time() - start) * 1000

    assert task is not None
    assert duration_ms < 10, f"get_by_key took {duration_ms:.2f}ms (target: <10ms)"

def test_filter_query_performance_10k_tasks(db_with_10k_tasks):
    """PRD: Filter query <100ms with 10,000 tasks"""
    task_repo = TaskRepository(db_with_10k_tasks)

    start = time.time()
    tasks = task_repo.filter_by_status("todo")
    duration_ms = (time.time() - start) * 1000

    assert len(tasks) > 0
    assert duration_ms < 100, f"Filter query took {duration_ms:.2f}ms (target: <100ms)"

def test_progress_calculation_performance(db_with_50_features):
    """PRD: Progress calculation <200ms for 50 features"""
    epic_repo = EpicRepository(db_with_50_features)

    start = time.time()
    progress = epic_repo.calculate_progress(epic_id=1)
    duration_ms = (time.time() - start) * 1000

    assert 0.0 <= progress <= 100.0
    assert duration_ms < 200, f"Progress calc took {duration_ms:.2f}ms (target: <200ms)"

def test_bulk_insert_performance(populated_db):
    """PRD: Bulk INSERT (100 tasks) <2,000ms"""
    task_repo = TaskRepository(populated_db)

    tasks_data = [
        {
            "feature_id": 1,
            "key": f"T-E04-F01-{i:03d}",
            "title": f"Bulk Task {i}",
            "status": "todo"
        }
        for i in range(100)
    ]

    start = time.time()
    with populated_db.begin():
        for data in tasks_data:
            task_repo.create(**data)
    duration_ms = (time.time() - start) * 1000

    assert duration_ms < 2000, f"Bulk INSERT took {duration_ms:.2f}ms (target: <2000ms)"

def test_cascade_delete_performance(db_with_1000_tasks):
    """PRD: CASCADE DELETE <500ms for epic with 100 features, 1000 tasks"""
    epic_repo = EpicRepository(db_with_1000_tasks)

    start = time.time()
    epic_repo.delete(epic_id=1)
    duration_ms = (time.time() - start) * 1000

    assert duration_ms < 500, f"CASCADE DELETE took {duration_ms:.2f}ms (target: <500ms)"
```

**Total Performance Tests**: ~8 tests

---

## Security Tests

### 7. Security Validation (`tests/security/test_database_security.py`)

#### Test: SQL Injection Prevention

```python
def test_sql_injection_in_filter_query(temp_db):
    """Test SQL injection attempt in filter query"""
    task_repo = TaskRepository(temp_db)

    # Create task
    task_repo.create(feature_id=1, key="T-E04-F01-001", title="Test", status="todo")

    # Attempt SQL injection
    malicious_input = "todo'; DROP TABLE tasks; --"

    # Should return empty list (no match), not execute DROP TABLE
    tasks = task_repo.filter_by_status(malicious_input)
    assert len(tasks) == 0

    # Verify table still exists
    assert task_repo.get_by_key("T-E04-F01-001") is not None

def test_sql_injection_in_task_title(temp_db):
    """Test SQL injection in task title"""
    task_repo = TaskRepository(temp_db)

    malicious_title = "Test'; DROP TABLE tasks; --"

    task = task_repo.create(
        feature_id=1,
        key="T-E04-F01-001",
        title=malicious_title,
        status="todo"
    )

    # Title stored as literal string (not executed)
    assert task.title == malicious_title

    # Table still exists
    assert task_repo.list_all() is not None
```

#### Test: File Permission Security

```python
@pytest.mark.skipif(os.name == 'nt', reason="Unix-only test")
def test_database_file_permissions(tmp_path):
    """Test database file has 600 permissions on Unix"""
    db_path = tmp_path / "secure.db"
    config = DatabaseConfig(db_path=db_path)
    factory = SessionFactory(config)
    factory.create_all_tables()

    # Check permissions (should be 600 = owner read/write only)
    file_mode = oct(db_path.stat().st_mode)[-3:]
    assert file_mode == "600", f"Database has permissions {file_mode}, expected 600"
```

#### Test: Path Traversal Prevention

```python
def test_path_traversal_in_file_path(temp_db):
    """Test path traversal attempt in file_path"""
    task_repo = TaskRepository(temp_db)

    malicious_path = "../../../../../../etc/passwd"

    with pytest.raises(ValidationError, match="contains invalid traversal"):
        task_repo.create(
            feature_id=1,
            key="T-E04-F01-001",
            title="Test",
            status="todo",
            file_path=malicious_path
        )
```

**Total Security Tests**: ~5 tests

---

## Acceptance Tests (PRD Alignment)

### 8. PRD Acceptance Criteria (`tests/acceptance/test_prd_acceptance.py`)

#### AC: Database Schema Creation

```python
def test_ac_database_schema_creation(tmp_path):
    """
    PRD AC: Database Schema Creation

    Given: Shark CLI is run for the first time in a new project
    When: Database initialization code executes
    Then: project.db file is created with all four tables
    And: All foreign key constraints are enabled
    And: All CHECK constraints are present
    And: All UNIQUE indexes are created
    """
    db_path = tmp_path / "project.db"
    config = DatabaseConfig(db_path=db_path)
    factory = SessionFactory(config)
    factory.create_all_tables()

    # Verify file exists
    assert db_path.exists()

    # Verify tables exist
    with factory.get_session() as session:
        session.query(Epic).count()
        session.query(Feature).count()
        session.query(Task).count()
        session.query(TaskHistory).count()

    # Verify foreign keys enabled
    assert factory.verify_foreign_keys_enabled()
```

#### AC: Referential Integrity Enforcement

```python
def test_ac_referential_integrity_cascade(temp_db):
    """
    PRD AC: Referential Integrity Enforcement

    Given: A feature exists with id=5
    When: I attempt to delete the parent epic
    Then: The feature is automatically deleted (CASCADE)
    And: All tasks belonging to that feature are also deleted
    """
    epic_repo = EpicRepository(temp_db)
    feature_repo = FeatureRepository(temp_db)
    task_repo = TaskRepository(temp_db)

    epic = epic_repo.create(key="E04", title="Test", status="active", priority="high")
    feature = feature_repo.create(epic_id=epic.id, key="E04-F01", title="Test", status="active")
    task = task_repo.create(feature_id=feature.id, key="T-E04-F01-001", title="Test", status="todo")

    # Delete epic
    epic_repo.delete(epic.id)

    # Verify cascade
    assert feature_repo.get_by_id(feature.id) is None
    assert task_repo.get_by_id(task.id) is None

def test_ac_referential_integrity_orphan_prevention(temp_db):
    """
    PRD AC: Referential Integrity Enforcement

    Given: I attempt to insert a task with feature_id=999 (non-existent)
    When: The INSERT executes
    Then: An IntegrityError is raised
    And: The transaction is rolled back
    And: No task record is created
    """
    task_repo = TaskRepository(temp_db)

    with pytest.raises(IntegrityError):
        task_repo.create(feature_id=999, key="T-E04-F01-001", title="Test", status="todo")

    # Verify no task created
    assert len(task_repo.list_all()) == 0
```

#### AC: Task Key Validation

```python
def test_ac_task_key_validation_invalid(temp_db):
    """
    PRD AC: Task Key Validation

    Given: I attempt to create a task with key "INVALID-KEY"
    When: The create_task() method is called
    Then: A ValidationError is raised with message "Invalid task key format"
    And: No database record is created
    """
    task_repo = TaskRepository(temp_db)

    with pytest.raises(ValidationError, match="Invalid task key format"):
        task_repo.create(feature_id=1, key="INVALID-KEY", title="Test", status="todo")

    assert len(task_repo.list_all()) == 0

def test_ac_task_key_validation_valid(temp_db):
    """
    PRD AC: Task Key Validation

    Given: I create a task with valid key "T-E01-F02-003"
    When: The create_task() method is called
    Then: The task is inserted successfully
    And: The key is stored exactly as provided
    """
    task_repo = TaskRepository(temp_db)
    feature_repo = FeatureRepository(temp_db)

    epic = EpicRepository(temp_db).create(key="E01", title="Test", status="active", priority="high")
    feature = feature_repo.create(epic_id=epic.id, key="E01-F02", title="Test", status="active")

    task = task_repo.create(feature_id=feature.id, key="T-E01-F02-003", title="Test", status="todo")

    assert task.key == "T-E01-F02-003"
```

#### AC: Progress Calculation

```python
def test_ac_progress_calculation(temp_db):
    """
    PRD AC: Progress Calculation

    Given: A feature has 10 tasks: 7 completed, 2 in_progress, 1 todo
    When: I call calculate_feature_progress(feature_id)
    Then: The result is 70.0 (7/10 × 100)
    """
    feature_repo = FeatureRepository(temp_db)
    task_repo = TaskRepository(temp_db)

    epic = EpicRepository(temp_db).create(key="E04", title="Test", status="active", priority="high")
    feature = feature_repo.create(epic_id=epic.id, key="E04-F01", title="Test", status="active")

    # Create 10 tasks
    for i in range(7):
        task_repo.create(feature_id=feature.id, key=f"T-E04-F01-{i:03d}", title="Test", status="completed")
    for i in range(7, 9):
        task_repo.create(feature_id=feature.id, key=f"T-E04-F01-{i:03d}", title="Test", status="in_progress")
    task_repo.create(feature_id=feature.id, key="T-E04-F01-009", title="Test", status="todo")

    progress = feature_repo.calculate_progress(feature.id)
    assert progress == 70.0

def test_ac_progress_calculation_zero_tasks(temp_db):
    """
    PRD AC: Progress Calculation

    Given: A feature has 0 tasks
    When: I call calculate_feature_progress(feature_id)
    Then: The result is 0.0 (not an error)
    """
    feature_repo = FeatureRepository(temp_db)

    epic = EpicRepository(temp_db).create(key="E04", title="Test", status="active", priority="high")
    feature = feature_repo.create(epic_id=epic.id, key="E04-F01", title="Test", status="active")

    progress = feature_repo.calculate_progress(feature.id)
    assert progress == 0.0
```

**Total Acceptance Tests**: ~10 tests (covering all PRD ACs)

---

## Test Summary

### Total Test Count

| Test Category | Test Count | Coverage Target |
|---------------|------------|-----------------|
| Unit Tests - Models | 20 | 90% |
| Unit Tests - Validation | 30 | 95% |
| Unit Tests - Session | 10 | 85% |
| Unit Tests - Repositories | 50 | 90% |
| Integration Tests | 15 | N/A |
| Performance Tests | 8 | N/A |
| Security Tests | 5 | N/A |
| Acceptance Tests | 10 | N/A |
| **Total** | **148 tests** | **>85% overall** |

### Test Fixtures

Required fixtures (`tests/conftest.py`):

```python
@pytest.fixture
def in_memory_session():
    """In-memory database for fast unit tests"""
    engine = create_engine("sqlite:///:memory:")
    Base.metadata.create_all(engine)
    session = Session(engine)
    yield session
    session.close()

@pytest.fixture
def temp_db(tmp_path):
    """Temporary file database for integration tests"""
    db_path = tmp_path / "test.db"
    config = DatabaseConfig(db_path=db_path)
    factory = SessionFactory(config)
    factory.create_all_tables()
    with factory.get_session() as session:
        yield session

@pytest.fixture
def populated_db(temp_db):
    """Database with sample epic, feature, tasks"""
    # Create sample data
    # ...
    yield temp_db

@pytest.fixture
def db_with_10k_tasks(temp_db):
    """Database with 10,000 tasks for performance testing"""
    # Create 10K tasks
    # ...
    yield temp_db
```

---

## Test Execution

### Running Tests

```bash
# All tests
pytest

# Unit tests only
pytest tests/unit/

# Integration tests
pytest tests/integration/

# Performance tests
pytest tests/performance/

# Acceptance tests
pytest tests/acceptance/

# With coverage
pytest --cov=pm/database --cov-report=html

# Parallel execution
pytest -n auto
```

### Coverage Report

```bash
# Generate HTML coverage report
pytest --cov=pm/database --cov-report=html

# Open report
open htmlcov/index.html
```

---

## Definition of Done (Testing)

Feature is complete when:

- [ ] All 148+ tests written
- [ ] All tests pass
- [ ] >85% code coverage achieved
- [ ] All PRD acceptance criteria tests pass
- [ ] All performance benchmarks meet targets
- [ ] All security tests pass
- [ ] No test warnings or failures
- [ ] Coverage report generated and reviewed

---

**Test Criteria Complete**: 2025-12-14
**Total Tests Defined**: 148+
**Next Step**: Begin implementation with TDD approach