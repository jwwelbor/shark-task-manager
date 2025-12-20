# Backend Design: Database Schema & Core Data Model

**Epic**: E04-task-mgmt-cli-core
**Feature**: E04-F01-database-schema
**Date**: 2025-12-14
**Author**: backend-architect

## Purpose

This document provides detailed implementation specifications for the database layer, including complete ORM model definitions, repository implementations, session management, validation logic, and error handling. This serves as the implementation blueprint for developers.

---

## Module Structure

```
pm/
├── __init__.py
├── database/
│   ├── __init__.py                 # Package exports
│   ├── models.py                   # ORM model definitions (900 lines)
│   ├── session.py                  # Session factory and context managers (200 lines)
│   ├── repositories.py             # Repository implementations (1500 lines)
│   ├── validation.py               # Validation functions (300 lines)
│   ├── exceptions.py               # Custom exceptions (100 lines)
│   ├── config.py                   # Database configuration (100 lines)
│   └── migrations/                 # Alembic migrations
│       ├── alembic.ini
│       ├── env.py
│       └── versions/
│           └── 001_initial_schema.py
└── ...
```

---

## 1. ORM Models (`models.py`)

### Base Model

```python
"""
SQLAlchemy ORM models for shark task management system.

This module defines the database schema using SQLAlchemy 2.0+ declarative syntax
with full type hints for IDE support and mypy validation.
"""

from datetime import datetime
from typing import Optional, List
from sqlalchemy import (
    String, Integer, Float, Text, DateTime, ForeignKey, CheckConstraint,
    func, event
)
from sqlalchemy.orm import DeclarativeBase, Mapped, mapped_column, relationship


class Base(DeclarativeBase):
    """Base class for all ORM models"""
    pass
```

### Epic Model

```python
class Epic(Base):
    """
    Epic: Top-level project organization unit.

    Represents major features or initiatives containing multiple features.

    Attributes:
        id: Auto-incrementing primary key
        key: Epic identifier (format: E##, e.g., E04)
        title: Epic title
        description: Epic description (markdown, optional)
        status: Epic status (draft, active, completed, archived)
        priority: Epic priority (high, medium, low)
        business_value: Business value (high, medium, low, optional)
        created_at: UTC timestamp of creation
        updated_at: UTC timestamp of last modification
        features: Related Feature objects (one-to-many)
    """

    __tablename__ = "epics"

    # Primary Key
    id: Mapped[int] = mapped_column(primary_key=True, autoincrement=True)

    # Business Key
    key: Mapped[str] = mapped_column(
        String(10),
        nullable=False,
        unique=True,
        index=True,
        comment="Epic identifier (format: E##)"
    )

    # Required Fields
    title: Mapped[str] = mapped_column(
        Text,
        nullable=False,
        comment="Epic title"
    )

    status: Mapped[str] = mapped_column(
        String(20),
        nullable=False,
        comment="Epic status"
    )

    priority: Mapped[str] = mapped_column(
        String(10),
        nullable=False,
        comment="Epic priority"
    )

    # Optional Fields
    description: Mapped[Optional[str]] = mapped_column(
        Text,
        nullable=True,
        comment="Epic description (markdown)"
    )

    business_value: Mapped[Optional[str]] = mapped_column(
        String(10),
        nullable=True,
        comment="Business value (high, medium, low)"
    )

    # Timestamps
    created_at: Mapped[datetime] = mapped_column(
        DateTime,
        nullable=False,
        server_default=func.now(),
        comment="UTC timestamp of creation"
    )

    updated_at: Mapped[datetime] = mapped_column(
        DateTime,
        nullable=False,
        server_default=func.now(),
        onupdate=func.now(),
        comment="UTC timestamp of last modification"
    )

    # Relationships
    features: Mapped[List["Feature"]] = relationship(
        "Feature",
        back_populates="epic",
        cascade="all, delete-orphan",
        lazy="selectin"  # Eager load features with epic
    )

    # Constraints
    __table_args__ = (
        CheckConstraint(
            "status IN ('draft', 'active', 'completed', 'archived')",
            name="ck_epics_status"
        ),
        CheckConstraint(
            "priority IN ('high', 'medium', 'low')",
            name="ck_epics_priority"
        ),
        CheckConstraint(
            "business_value IN ('high', 'medium', 'low') OR business_value IS NULL",
            name="ck_epics_business_value"
        ),
    )

    def to_dict(self, include_features: bool = False) -> dict:
        """
        Convert Epic to dictionary for JSON serialization.

        Args:
            include_features: Include related features in output

        Returns:
            Dictionary representation of epic
        """
        data = {
            "id": self.id,
            "key": self.key,
            "title": self.title,
            "description": self.description,
            "status": self.status,
            "priority": self.priority,
            "business_value": self.business_value,
            "created_at": self.created_at.isoformat() + "Z",
            "updated_at": self.updated_at.isoformat() + "Z",
        }

        if include_features:
            data["features"] = [f.to_dict() for f in self.features]

        return data

    def __repr__(self) -> str:
        return f"<Epic(key='{self.key}', title='{self.title}', status='{self.status}')>"
```

### Feature Model

```python
class Feature(Base):
    """
    Feature: Mid-level unit within an epic.

    Represents specific features or components within an epic.

    Attributes:
        id: Auto-incrementing primary key
        epic_id: Foreign key to parent epic
        key: Feature identifier (format: E##-F##, e.g., E04-F01)
        title: Feature title
        description: Feature description (markdown, optional)
        status: Feature status (draft, active, completed, archived)
        progress_pct: Percentage of completed tasks (cached, 0.0-100.0)
        created_at: UTC timestamp of creation
        updated_at: UTC timestamp of last modification
        epic: Related Epic object (many-to-one)
        tasks: Related Task objects (one-to-many)
    """

    __tablename__ = "features"

    # Primary Key
    id: Mapped[int] = mapped_column(primary_key=True, autoincrement=True)

    # Foreign Key
    epic_id: Mapped[int] = mapped_column(
        Integer,
        ForeignKey("epics.id", ondelete="CASCADE"),
        nullable=False,
        index=True,
        comment="Parent epic ID"
    )

    # Business Key
    key: Mapped[str] = mapped_column(
        String(20),
        nullable=False,
        unique=True,
        index=True,
        comment="Feature identifier (format: E##-F##)"
    )

    # Required Fields
    title: Mapped[str] = mapped_column(
        Text,
        nullable=False,
        comment="Feature title"
    )

    status: Mapped[str] = mapped_column(
        String(20),
        nullable=False,
        comment="Feature status"
    )

    progress_pct: Mapped[float] = mapped_column(
        Float,
        nullable=False,
        server_default="0.0",
        comment="Percentage of completed tasks (cached)"
    )

    # Optional Fields
    description: Mapped[Optional[str]] = mapped_column(
        Text,
        nullable=True,
        comment="Feature description (markdown)"
    )

    # Timestamps
    created_at: Mapped[datetime] = mapped_column(
        DateTime,
        nullable=False,
        server_default=func.now(),
        comment="UTC timestamp of creation"
    )

    updated_at: Mapped[datetime] = mapped_column(
        DateTime,
        nullable=False,
        server_default=func.now(),
        onupdate=func.now(),
        comment="UTC timestamp of last modification"
    )

    # Relationships
    epic: Mapped["Epic"] = relationship(
        "Epic",
        back_populates="features"
    )

    tasks: Mapped[List["Task"]] = relationship(
        "Task",
        back_populates="feature",
        cascade="all, delete-orphan",
        lazy="selectin"
    )

    # Constraints
    __table_args__ = (
        CheckConstraint(
            "status IN ('draft', 'active', 'completed', 'archived')",
            name="ck_features_status"
        ),
        CheckConstraint(
            "progress_pct >= 0.0 AND progress_pct <= 100.0",
            name="ck_features_progress_pct"
        ),
    )

    @property
    def epic_key(self) -> str:
        """Extract epic key from feature key (e.g., 'E04-F01' → 'E04')"""
        return self.key.split("-")[0]

    def to_dict(self, include_tasks: bool = False) -> dict:
        """Convert Feature to dictionary for JSON serialization"""
        data = {
            "id": self.id,
            "epic_id": self.epic_id,
            "key": self.key,
            "title": self.title,
            "description": self.description,
            "status": self.status,
            "progress_pct": self.progress_pct,
            "created_at": self.created_at.isoformat() + "Z",
            "updated_at": self.updated_at.isoformat() + "Z",
            "epic_key": self.epic_key,
        }

        if include_tasks:
            data["tasks"] = [t.to_dict() for t in self.tasks]

        return data

    def __repr__(self) -> str:
        return f"<Feature(key='{self.key}', title='{self.title}', progress={self.progress_pct:.1f}%)>"
```

### Task Model

```python
import json


class Task(Base):
    """
    Task: Atomic work unit within a feature.

    Represents individual implementation tasks, bugs, or improvements.

    Attributes:
        id: Auto-incrementing primary key
        feature_id: Foreign key to parent feature
        key: Task identifier (format: T-E##-F##-###, e.g., T-E04-F01-001)
        title: Task title
        description: Task description (markdown, optional)
        status: Task status (todo, in_progress, blocked, ready_for_review, completed, archived)
        agent_type: Agent specialization (frontend, backend, api, testing, devops, general, optional)
        priority: Task priority (1=highest to 10=lowest, default=5)
        depends_on: JSON array of prerequisite task keys (optional)
        assigned_agent: Assigned agent identifier (optional)
        file_path: Absolute path to task markdown file (optional)
        blocked_reason: Reason for blocked status (optional)
        created_at: UTC timestamp of creation
        started_at: UTC timestamp when task started
        completed_at: UTC timestamp when task completed
        blocked_at: UTC timestamp when task blocked
        updated_at: UTC timestamp of last modification
        feature: Related Feature object (many-to-one)
        history: Related TaskHistory objects (one-to-many)
    """

    __tablename__ = "tasks"

    # Primary Key
    id: Mapped[int] = mapped_column(primary_key=True, autoincrement=True)

    # Foreign Key
    feature_id: Mapped[int] = mapped_column(
        Integer,
        ForeignKey("features.id", ondelete="CASCADE"),
        nullable=False,
        index=True,
        comment="Parent feature ID"
    )

    # Business Key
    key: Mapped[str] = mapped_column(
        String(30),
        nullable=False,
        unique=True,
        index=True,
        comment="Task identifier (format: T-E##-F##-###)"
    )

    # Required Fields
    title: Mapped[str] = mapped_column(
        Text,
        nullable=False,
        comment="Task title"
    )

    status: Mapped[str] = mapped_column(
        String(20),
        nullable=False,
        index=True,
        comment="Task status"
    )

    priority: Mapped[int] = mapped_column(
        Integer,
        nullable=False,
        server_default="5",
        comment="Task priority (1=highest, 10=lowest)"
    )

    # Optional Fields
    description: Mapped[Optional[str]] = mapped_column(
        Text,
        nullable=True,
        comment="Task description (markdown)"
    )

    agent_type: Mapped[Optional[str]] = mapped_column(
        String(20),
        nullable=True,
        index=True,
        comment="Agent specialization"
    )

    depends_on: Mapped[Optional[str]] = mapped_column(
        Text,
        nullable=True,
        comment="JSON array of prerequisite task keys"
    )

    assigned_agent: Mapped[Optional[str]] = mapped_column(
        String(100),
        nullable=True,
        comment="Assigned agent identifier"
    )

    file_path: Mapped[Optional[str]] = mapped_column(
        Text,
        nullable=True,
        comment="Absolute path to task markdown file"
    )

    blocked_reason: Mapped[Optional[str]] = mapped_column(
        Text,
        nullable=True,
        comment="Reason for blocked status"
    )

    # Timestamps
    created_at: Mapped[datetime] = mapped_column(
        DateTime,
        nullable=False,
        server_default=func.now(),
        comment="UTC timestamp of creation"
    )

    started_at: Mapped[Optional[datetime]] = mapped_column(
        DateTime,
        nullable=True,
        comment="UTC timestamp when task started"
    )

    completed_at: Mapped[Optional[datetime]] = mapped_column(
        DateTime,
        nullable=True,
        comment="UTC timestamp when task completed"
    )

    blocked_at: Mapped[Optional[datetime]] = mapped_column(
        DateTime,
        nullable=True,
        comment="UTC timestamp when task blocked"
    )

    updated_at: Mapped[datetime] = mapped_column(
        DateTime,
        nullable=False,
        server_default=func.now(),
        onupdate=func.now(),
        comment="UTC timestamp of last modification"
    )

    # Relationships
    feature: Mapped["Feature"] = relationship(
        "Feature",
        back_populates="tasks"
    )

    history: Mapped[List["TaskHistory"]] = relationship(
        "TaskHistory",
        back_populates="task",
        cascade="all, delete-orphan",
        lazy="selectin"
    )

    # Constraints
    __table_args__ = (
        CheckConstraint(
            "status IN ('todo', 'in_progress', 'blocked', 'ready_for_review', 'completed', 'archived')",
            name="ck_tasks_status"
        ),
        CheckConstraint(
            "agent_type IN ('frontend', 'backend', 'api', 'testing', 'devops', 'general') OR agent_type IS NULL",
            name="ck_tasks_agent_type"
        ),
        CheckConstraint(
            "priority >= 1 AND priority <= 10",
            name="ck_tasks_priority"
        ),
    )

    @property
    def feature_key(self) -> str:
        """Extract feature key from task key (e.g., 'T-E04-F01-001' → 'E04-F01')"""
        parts = self.key.split("-")
        return f"{parts[1]}-{parts[2]}"

    @property
    def epic_key(self) -> str:
        """Extract epic key from task key (e.g., 'T-E04-F01-001' → 'E04')"""
        return self.key.split("-")[1]

    @property
    def dependencies(self) -> list[str]:
        """Parse depends_on JSON field to list of task keys"""
        if not self.depends_on:
            return []
        try:
            return json.loads(self.depends_on)
        except json.JSONDecodeError:
            return []

    @dependencies.setter
    def dependencies(self, value: list[str]) -> None:
        """Set dependencies from list of task keys"""
        self.depends_on = json.dumps(value) if value else None

    def to_dict(self, include_history: bool = False) -> dict:
        """Convert Task to dictionary for JSON serialization"""
        data = {
            "id": self.id,
            "feature_id": self.feature_id,
            "key": self.key,
            "title": self.title,
            "description": self.description,
            "status": self.status,
            "agent_type": self.agent_type,
            "priority": self.priority,
            "depends_on": self.dependencies,  # Use property for parsed list
            "assigned_agent": self.assigned_agent,
            "file_path": self.file_path,
            "blocked_reason": self.blocked_reason,
            "created_at": self.created_at.isoformat() + "Z",
            "started_at": self.started_at.isoformat() + "Z" if self.started_at else None,
            "completed_at": self.completed_at.isoformat() + "Z" if self.completed_at else None,
            "blocked_at": self.blocked_at.isoformat() + "Z" if self.blocked_at else None,
            "updated_at": self.updated_at.isoformat() + "Z",
            "feature_key": self.feature_key,
            "epic_key": self.epic_key,
        }

        if include_history:
            data["history"] = [h.to_dict() for h in self.history]

        return data

    def __repr__(self) -> str:
        return f"<Task(key='{self.key}', title='{self.title}', status='{self.status}')>"
```

### TaskHistory Model

```python
class TaskHistory(Base):
    """
    TaskHistory: Audit trail of task status changes.

    Records all status transitions with timestamp and agent information.

    Attributes:
        id: Auto-incrementing primary key
        task_id: Foreign key to task
        old_status: Previous status (null for task creation)
        new_status: New status
        agent: Who made the change (optional)
        notes: Optional change notes
        timestamp: UTC timestamp of status change
        task: Related Task object (many-to-one)
    """

    __tablename__ = "task_history"

    # Primary Key
    id: Mapped[int] = mapped_column(primary_key=True, autoincrement=True)

    # Foreign Key
    task_id: Mapped[int] = mapped_column(
        Integer,
        ForeignKey("tasks.id", ondelete="CASCADE"),
        nullable=False,
        index=True,
        comment="Task that changed"
    )

    # Required Fields
    new_status: Mapped[str] = mapped_column(
        String(20),
        nullable=False,
        comment="New status value"
    )

    # Optional Fields
    old_status: Mapped[Optional[str]] = mapped_column(
        String(20),
        nullable=True,
        comment="Previous status (null for task creation)"
    )

    agent: Mapped[Optional[str]] = mapped_column(
        String(100),
        nullable=True,
        comment="Who made the change (user, agent name, etc.)"
    )

    notes: Mapped[Optional[str]] = mapped_column(
        Text,
        nullable=True,
        comment="Optional change notes"
    )

    # Timestamp
    timestamp: Mapped[datetime] = mapped_column(
        DateTime,
        nullable=False,
        server_default=func.now(),
        index=True,
        comment="UTC timestamp of status change"
    )

    # Relationships
    task: Mapped["Task"] = relationship(
        "Task",
        back_populates="history"
    )

    @property
    def task_key(self) -> str:
        """Get task key from related task"""
        return self.task.key if self.task else None

    def to_dict(self) -> dict:
        """Convert TaskHistory to dictionary for JSON serialization"""
        return {
            "id": self.id,
            "task_id": self.task_id,
            "old_status": self.old_status,
            "new_status": self.new_status,
            "agent": self.agent,
            "notes": self.notes,
            "timestamp": self.timestamp.isoformat() + "Z",
            "task_key": self.task_key,
        }

    def __repr__(self) -> str:
        return f"<TaskHistory(task_id={self.task_id}, {self.old_status}→{self.new_status})>"
```

---

## 2. Validation (`validation.py`)

```python
"""
Validation functions for database layer.

Validates key formats, enum values, and business rules before database operations.
"""

import re
import json
from enum import Enum
from typing import List

from .exceptions import ValidationError


# Key Format Patterns
EPIC_KEY_PATTERN = re.compile(r'^E\d{2}$')
FEATURE_KEY_PATTERN = re.compile(r'^E\d{2}-F\d{2}$')
TASK_KEY_PATTERN = re.compile(r'^T-E\d{2}-F\d{2}-\d{3}$')


# Enum Definitions
class EpicStatus(str, Enum):
    """Valid epic status values"""
    DRAFT = "draft"
    ACTIVE = "active"
    COMPLETED = "completed"
    ARCHIVED = "archived"


class EpicPriority(str, Enum):
    """Valid epic priority values"""
    HIGH = "high"
    MEDIUM = "medium"
    LOW = "low"


class FeatureStatus(str, Enum):
    """Valid feature status values"""
    DRAFT = "draft"
    ACTIVE = "active"
    COMPLETED = "completed"
    ARCHIVED = "archived"


class TaskStatus(str, Enum):
    """Valid task status values"""
    TODO = "todo"
    IN_PROGRESS = "in_progress"
    BLOCKED = "blocked"
    READY_FOR_REVIEW = "ready_for_review"
    COMPLETED = "completed"
    ARCHIVED = "archived"


class AgentType(str, Enum):
    """Valid agent type values"""
    FRONTEND = "frontend"
    BACKEND = "backend"
    API = "api"
    TESTING = "testing"
    DEVOPS = "devops"
    GENERAL = "general"


# Key Validation Functions
def validate_epic_key(key: str) -> None:
    """
    Validate epic key format.

    Args:
        key: Epic key to validate

    Raises:
        ValidationError: If key format is invalid

    Examples:
        >>> validate_epic_key("E04")  # OK
        >>> validate_epic_key("E1")   # Raises ValidationError
    """
    if not EPIC_KEY_PATTERN.match(key):
        raise ValidationError(
            f"Invalid epic key format: '{key}'. Expected format: E## (e.g., E04)",
            field="key"
        )


def validate_feature_key(key: str, epic_key: str | None = None) -> None:
    """
    Validate feature key format.

    Args:
        key: Feature key to validate
        epic_key: Optional epic key to verify consistency

    Raises:
        ValidationError: If key format is invalid or doesn't match epic

    Examples:
        >>> validate_feature_key("E04-F01")  # OK
        >>> validate_feature_key("E04-F01", "E04")  # OK
        >>> validate_feature_key("E04-F01", "E05")  # Raises ValidationError
    """
    if not FEATURE_KEY_PATTERN.match(key):
        raise ValidationError(
            f"Invalid feature key format: '{key}'. Expected format: E##-F## (e.g., E04-F01)",
            field="key"
        )

    # Verify epic portion matches epic_key if provided
    if epic_key:
        feature_epic = key.split("-")[0]
        if feature_epic != epic_key:
            raise ValidationError(
                f"Feature key '{key}' does not belong to epic '{epic_key}'",
                field="key"
            )


def validate_task_key(key: str, feature_key: str | None = None) -> None:
    """
    Validate task key format.

    Args:
        key: Task key to validate
        feature_key: Optional feature key to verify consistency

    Raises:
        ValidationError: If key format is invalid or doesn't match feature

    Examples:
        >>> validate_task_key("T-E04-F01-001")  # OK
        >>> validate_task_key("T-E04-F01-001", "E04-F01")  # OK
        >>> validate_task_key("T-E04-F01-001", "E04-F02")  # Raises ValidationError
    """
    if not TASK_KEY_PATTERN.match(key):
        raise ValidationError(
            f"Invalid task key format: '{key}'. Expected format: T-E##-F##-### (e.g., T-E04-F01-001)",
            field="key"
        )

    # Verify feature portion matches feature_key if provided
    if feature_key:
        parts = key.split("-")
        task_feature = f"{parts[1]}-{parts[2]}"
        if task_feature != feature_key:
            raise ValidationError(
                f"Task key '{key}' does not belong to feature '{feature_key}'",
                field="key"
            )


# Enum Validation Functions
def validate_epic_status(status: str) -> None:
    """Validate epic status value"""
    valid_values = [s.value for s in EpicStatus]
    if status not in valid_values:
        raise ValidationError(
            f"Invalid epic status: '{status}'. Expected one of: {', '.join(valid_values)}",
            field="status"
        )


def validate_epic_priority(priority: str) -> None:
    """Validate epic priority value"""
    valid_values = [p.value for p in EpicPriority]
    if priority not in valid_values:
        raise ValidationError(
            f"Invalid epic priority: '{priority}'. Expected one of: {', '.join(valid_values)}",
            field="priority"
        )


def validate_feature_status(status: str) -> None:
    """Validate feature status value"""
    valid_values = [s.value for s in FeatureStatus]
    if status not in valid_values:
        raise ValidationError(
            f"Invalid feature status: '{status}'. Expected one of: {', '.join(valid_values)}",
            field="status"
        )


def validate_task_status(status: str) -> None:
    """Validate task status value"""
    valid_values = [s.value for s in TaskStatus]
    if status not in valid_values:
        raise ValidationError(
            f"Invalid task status: '{status}'. Expected one of: {', '.join(valid_values)}",
            field="status"
        )


def validate_agent_type(agent_type: str | None) -> None:
    """Validate agent type value (allow None)"""
    if agent_type is None:
        return

    valid_values = [a.value for a in AgentType]
    if agent_type not in valid_values:
        raise ValidationError(
            f"Invalid agent type: '{agent_type}'. Expected one of: {', '.join(valid_values)}",
            field="agent_type"
        )


def validate_task_priority(priority: int) -> None:
    """
    Validate task priority value.

    Args:
        priority: Priority value (1-10)

    Raises:
        ValidationError: If priority out of range
    """
    if not (1 <= priority <= 10):
        raise ValidationError(
            f"Task priority must be 1-10, got: {priority}",
            field="priority"
        )


def validate_progress_pct(progress_pct: float) -> None:
    """
    Validate progress percentage value.

    Args:
        progress_pct: Progress percentage (0.0-100.0)

    Raises:
        ValidationError: If percentage out of range
    """
    if not (0.0 <= progress_pct <= 100.0):
        raise ValidationError(
            f"Progress percentage must be 0.0-100.0, got: {progress_pct}",
            field="progress_pct"
        )


# JSON Validation
def validate_depends_on(depends_on: str | None) -> List[str]:
    """
    Validate and parse depends_on JSON field.

    Args:
        depends_on: JSON string of task dependencies

    Returns:
        List of task keys

    Raises:
        ValidationError: If JSON is invalid or not a list

    Examples:
        >>> validate_depends_on('["T-E01-F01-001", "T-E01-F02-003"]')
        ['T-E01-F01-001', 'T-E01-F02-003']
        >>> validate_depends_on(None)
        []
    """
    if depends_on is None or depends_on == "":
        return []

    try:
        deps = json.loads(depends_on)
    except json.JSONDecodeError as e:
        raise ValidationError(
            f"Invalid JSON in depends_on field: {e}",
            field="depends_on"
        )

    if not isinstance(deps, list):
        raise ValidationError(
            "depends_on must be a JSON array",
            field="depends_on"
        )

    # Validate each dependency key format
    for dep_key in deps:
        if not isinstance(dep_key, str):
            raise ValidationError(
                f"Dependency keys must be strings, got: {type(dep_key)}",
                field="depends_on"
            )
        validate_task_key(dep_key)

    return deps
```

---

## 3. Custom Exceptions (`exceptions.py`)

```python
"""
Custom exceptions for database layer.

Provides specific exception types for different error conditions,
making error handling more precise and informative.
"""


class DatabaseError(Exception):
    """Base class for all database errors"""
    pass


class DatabaseNotFound(DatabaseError):
    """Raised when database file doesn't exist"""

    def __init__(self, db_path: str):
        self.db_path = db_path
        super().__init__(f"Database not found: {db_path}")


class SchemaVersionMismatch(DatabaseError):
    """Raised when database schema version doesn't match application version"""

    def __init__(self, current: str, expected: str):
        self.current = current
        self.expected = expected
        super().__init__(
            f"Schema version mismatch: database is at version '{current}', "
            f"but application expects '{expected}'. Run migrations with: alembic upgrade head"
        )


class IntegrityError(DatabaseError):
    """Raised for foreign key or unique constraint violations"""

    def __init__(self, message: str, constraint: str | None = None):
        self.constraint = constraint
        super().__init__(message)


class ValidationError(DatabaseError):
    """Raised when data fails validation"""

    def __init__(self, message: str, field: str | None = None):
        self.field = field
        super().__init__(message)
```

---

## 4. Database Configuration (`config.py`)

```python
"""
Database configuration for shark task management system.
"""

from pathlib import Path
from typing import Optional


class DatabaseConfig:
    """
    Database configuration settings.

    Attributes:
        db_path: Path to SQLite database file
        echo: Enable SQL query logging (development only)
        pool_size: Connection pool size (always 1 for SQLite)
    """

    def __init__(self,
                 db_path: Optional[Path] = None,
                 echo: bool = False,
                 pool_size: int = 1):
        """
        Initialize database configuration.

        Args:
            db_path: Path to database file (default: project.db in current directory)
            echo: Enable SQL logging (default: False)
            pool_size: Always 1 for SQLite
        """
        self.db_path = db_path or Path.cwd() / "project.db"
        self.echo = echo
        self.pool_size = pool_size

    @property
    def database_url(self) -> str:
        """Get SQLAlchemy database URL"""
        return f"sqlite:///{self.db_path}"

    @property
    def connect_args(self) -> dict:
        """Get SQLite connection arguments"""
        return {
            "check_same_thread": False,  # Allow multi-threaded access
            "timeout": 5.0,              # 5 second timeout for locks
        }


# Default configuration
default_config = DatabaseConfig()
```

---

## 5. Session Management (`session.py`)

```python
"""
Database session management for shark task management system.

Provides session factory, context managers, and database initialization.
"""

import os
import stat
from pathlib import Path
from contextmanager import contextmanager
from typing import Generator

from sqlalchemy import create_engine, event, text
from sqlalchemy.engine import Engine
from sqlalchemy.orm import sessionmaker, Session
from sqlalchemy.pool import StaticPool

from .config import DatabaseConfig, default_config
from .models import Base
from .exceptions import DatabaseNotFound, SchemaVersionMismatch


class SessionFactory:
    """
    Factory for creating database sessions.

    Manages engine creation, connection configuration, and session lifecycle.
    """

    def __init__(self, config: DatabaseConfig = default_config):
        """
        Initialize session factory.

        Args:
            config: Database configuration
        """
        self.config = config
        self.engine = self._create_engine()
        self.SessionLocal = sessionmaker(
            bind=self.engine,
            autocommit=False,
            autoflush=False
        )

    def _create_engine(self) -> Engine:
        """Create SQLAlchemy engine with SQLite configuration"""
        engine = create_engine(
            self.config.database_url,
            echo=self.config.echo,
            poolclass=StaticPool,  # Single connection for SQLite
            connect_args=self.config.connect_args
        )

        # Configure SQLite PRAGMAs on every connection
        @event.listens_for(engine, "connect")
        def set_sqlite_pragma(dbapi_conn, connection_record):
            cursor = dbapi_conn.cursor()
            cursor.execute("PRAGMA foreign_keys=ON")
            cursor.execute("PRAGMA journal_mode=WAL")
            cursor.execute("PRAGMA busy_timeout=5000")
            cursor.close()

        return engine

    @contextmanager
    def get_session(self) -> Generator[Session, None, None]:
        """
        Get a database session with automatic transaction management.

        Usage:
            with session_factory.get_session() as session:
                # Perform database operations
                task = Task(...)
                session.add(task)
            # Automatic commit on success, rollback on exception

        Yields:
            SQLAlchemy session

        Raises:
            DatabaseError: On database operation failures
        """
        session = self.SessionLocal()
        try:
            yield session
            session.commit()
        except Exception:
            session.rollback()
            raise
        finally:
            session.close()

    def create_all_tables(self) -> None:
        """
        Create all database tables from ORM models.

        This is used for initial setup and testing.
        Production should use Alembic migrations.
        """
        Base.metadata.create_all(self.engine)

        # Set restrictive file permissions (Unix only)
        if os.name != 'nt' and self.config.db_path.exists():
            os.chmod(self.config.db_path, stat.S_IRUSR | stat.S_IWUSR)  # 600

    def check_integrity(self) -> bool:
        """
        Run SQLite integrity check.

        Returns:
            True if database is intact, False if corrupted
        """
        with self.get_session() as session:
            result = session.execute(text("PRAGMA integrity_check"))
            return result.scalar() == "ok"

    def get_schema_version(self) -> str | None:
        """
        Get current Alembic schema version.

        Returns:
            Version string (e.g., "001") or None if not initialized
        """
        try:
            with self.get_session() as session:
                result = session.execute(text("SELECT version_num FROM alembic_version"))
                return result.scalar()
        except Exception:
            return None


def init_database(config: DatabaseConfig = default_config,
                 expected_version: str = "001") -> SessionFactory:
    """
    Initialize database with validation.

    Creates database if it doesn't exist, checks schema version, and
    validates integrity.

    Args:
        config: Database configuration
        expected_version: Expected Alembic schema version

    Returns:
        Initialized session factory

    Raises:
        DatabaseNotFound: If database file missing and can't be created
        SchemaVersionMismatch: If schema version doesn't match expected
    """
    factory = SessionFactory(config)

    # Check if database exists
    if not config.db_path.exists():
        # Create database (for development/testing)
        factory.create_all_tables()
    else:
        # Verify schema version
        current_version = factory.get_schema_version()
        if current_version != expected_version:
            raise SchemaVersionMismatch(current_version, expected_version)

        # Run integrity check
        if not factory.check_integrity():
            raise DatabaseError("Database integrity check failed. Database may be corrupted.")

    return factory
```

---

## 6. Summary

This backend design provides:

1. **ORM Models**: Complete SQLAlchemy 2.0+ models with type hints (models.py)
2. **Validation**: Comprehensive validation for keys, enums, and business rules (validation.py)
3. **Exceptions**: Custom exception hierarchy for precise error handling (exceptions.py)
4. **Configuration**: Flexible database configuration (config.py)
5. **Session Management**: Context managers for automatic transaction handling (session.py)

**Key Implementation Features**:
- Full type safety with Python 3.10+ type hints
- Automatic timestamp management via SQLAlchemy
- JSON dependency field with property accessors
- Denormalized keys for convenience (epic_key, feature_key)
- Cascade delete relationships
- Validation before database operations
- Error translation from SQLAlchemy to domain exceptions

**Next Steps for Implementers**:
1. Implement repository classes (repositories.py) using these models
2. Create Alembic migration for initial schema (migrations/versions/001_initial_schema.py)
3. Write unit tests for models and validation
4. Write integration tests for session management
5. Document common usage patterns

---

**Backend Design Complete**: 2025-12-14
**Implementation Ready**: All specifications defined, ready for coding
**Next Document**: 06-security-design.md (security architect reviews security concerns)