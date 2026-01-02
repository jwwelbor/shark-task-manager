# Acceptance Criteria: Cascading Status Calculation

**Feature**: E07-F14
**Document Type**: Acceptance Criteria (Given/When/Then format)

---

## AC1: Feature Status Calculation - Empty Feature

### AC1.1: Feature with no tasks defaults to draft
```gherkin
Given a feature "E07-F14" exists in the database
And the feature has zero tasks
When the feature status is calculated
Then the feature status SHOULD be "draft"
And the status_source SHOULD be "calculated"
```

### AC1.2: Newly created feature has draft status
```gherkin
Given an epic "E07" exists
When I create a new feature with command `shark feature create --epic=E07 "New Feature"`
Then the feature status SHOULD be "draft"
And no tasks exist for the feature
```

---

## AC2: Feature Status Calculation - Task-Based

### AC2.1: All tasks in todo results in draft status
```gherkin
Given a feature "E07-F14" with 3 tasks
And all 3 tasks have status "todo"
When the feature status is calculated
Then the feature status SHOULD be "draft"
```

### AC2.2: Any task in_progress results in active status
```gherkin
Given a feature "E07-F14" with 3 tasks
And 2 tasks have status "todo"
And 1 task has status "in_progress"
When the feature status is calculated
Then the feature status SHOULD be "active"
```

### AC2.3: Any task ready_for_review results in active status
```gherkin
Given a feature "E07-F14" with 3 tasks
And 2 tasks have status "completed"
And 1 task has status "ready_for_review"
When the feature status is calculated
Then the feature status SHOULD be "active"
```

### AC2.4: Any task blocked results in active status
```gherkin
Given a feature "E07-F14" with 3 tasks
And 2 tasks have status "todo"
And 1 task has status "blocked"
When the feature status is calculated
Then the feature status SHOULD be "active"
```

### AC2.5: All tasks completed results in completed status
```gherkin
Given a feature "E07-F14" with 3 tasks
And all 3 tasks have status "completed"
When the feature status is calculated
Then the feature status SHOULD be "completed"
```

### AC2.6: All tasks archived results in completed status
```gherkin
Given a feature "E07-F14" with 3 tasks
And all 3 tasks have status "archived"
When the feature status is calculated
Then the feature status SHOULD be "completed"
```

### AC2.7: Mix of completed and archived results in completed status
```gherkin
Given a feature "E07-F14" with 3 tasks
And 2 tasks have status "completed"
And 1 task has status "archived"
When the feature status is calculated
Then the feature status SHOULD be "completed"
```

---

## AC3: Epic Status Calculation - Empty Epic

### AC3.1: Epic with no features defaults to draft
```gherkin
Given an epic "E07" exists in the database
And the epic has zero features
When the epic status is calculated
Then the epic status SHOULD be "draft"
And the status_source SHOULD be "calculated"
```

### AC3.2: Newly created epic has draft status
```gherkin
When I create a new epic with command `shark epic create "New Epic"`
Then the epic status SHOULD be "draft"
And no features exist for the epic
```

---

## AC4: Epic Status Calculation - Feature-Based

### AC4.1: All features in draft results in draft status
```gherkin
Given an epic "E07" with 3 features
And all 3 features have status "draft"
When the epic status is calculated
Then the epic status SHOULD be "draft"
```

### AC4.2: Any feature active results in active status
```gherkin
Given an epic "E07" with 3 features
And 2 features have status "draft"
And 1 feature has status "active"
When the epic status is calculated
Then the epic status SHOULD be "active"
```

### AC4.3: All features completed results in completed status
```gherkin
Given an epic "E07" with 3 features
And all 3 features have status "completed"
When the epic status is calculated
Then the epic status SHOULD be "completed"
```

### AC4.4: All features archived results in completed status
```gherkin
Given an epic "E07" with 3 features
And all 3 features have status "archived"
When the epic status is calculated
Then the epic status SHOULD be "completed"
```

### AC4.5: Mix of completed and archived results in completed status
```gherkin
Given an epic "E07" with 3 features
And 2 features have status "completed"
And 1 feature has status "archived"
When the epic status is calculated
Then the epic status SHOULD be "completed"
```

---

## AC5: Cascading Updates - Task to Feature

### AC5.1: Starting a task triggers feature recalculation
```gherkin
Given a feature "E07-F14" with status "draft"
And the feature has 2 tasks with status "todo"
When I run `shark task start T-E07-F14-001`
Then task T-E07-F14-001 status SHOULD be "in_progress"
And feature "E07-F14" status SHOULD be "active"
```

### AC5.2: Completing last task triggers feature completion
```gherkin
Given a feature "E07-F14" with status "active"
And the feature has 2 tasks
And task T-E07-F14-001 has status "completed"
And task T-E07-F14-002 has status "in_progress"
When I run `shark task complete T-E07-F14-002`
And I run `shark task approve T-E07-F14-002`
Then feature "E07-F14" status SHOULD be "completed"
```

### AC5.3: Reopening a task triggers feature recalculation
```gherkin
Given a feature "E07-F14" with status "completed"
And all tasks have status "completed"
When I run `shark task reopen T-E07-F14-001`
Then task T-E07-F14-001 status SHOULD be "in_progress"
And feature "E07-F14" status SHOULD be "active"
```

### AC5.4: Blocking a task keeps feature active
```gherkin
Given a feature "E07-F14" with status "active"
And the feature has 2 tasks with status "in_progress"
When I run `shark task block T-E07-F14-001 --reason="Waiting on API"`
Then task T-E07-F14-001 status SHOULD be "blocked"
And feature "E07-F14" status SHOULD be "active"
```

### AC5.5: Creating a task triggers feature recalculation
```gherkin
Given a feature "E07-F14" with status "completed"
And all existing tasks have status "completed"
When I run `shark task create "New Task" --epic=E07 --feature=F14`
Then a new task with status "todo" is created
And feature "E07-F14" status SHOULD be "active"
```

### AC5.6: Deleting last non-completed task triggers completion
```gherkin
Given a feature "E07-F14" with status "active"
And the feature has 2 tasks
And task T-E07-F14-001 has status "completed"
And task T-E07-F14-002 has status "todo"
When task T-E07-F14-002 is deleted
Then feature "E07-F14" status SHOULD be "completed"
```

---

## AC6: Cascading Updates - Feature to Epic

### AC6.1: Activating a feature activates the epic
```gherkin
Given an epic "E07" with status "draft"
And the epic has 2 features with status "draft"
When feature "E07-F14" status changes to "active"
Then epic "E07" status SHOULD be "active"
```

### AC6.2: Completing all features completes the epic
```gherkin
Given an epic "E07" with status "active"
And the epic has 2 features
And feature "E07-F01" has status "completed"
And feature "E07-F02" has status "active"
When feature "E07-F02" status changes to "completed"
Then epic "E07" status SHOULD be "completed"
```

### AC6.3: Creating a feature activates completed epic
```gherkin
Given an epic "E07" with status "completed"
And all existing features have status "completed"
When I run `shark feature create --epic=E07 "New Feature"`
Then a new feature with status "draft" is created
And epic "E07" status SHOULD be "active"
```

### AC6.4: Deleting features triggers epic recalculation
```gherkin
Given an epic "E07" with status "active"
And the epic has 2 features
And feature "E07-F01" has status "completed"
And feature "E07-F02" has status "active"
When feature "E07-F02" is deleted
Then epic "E07" status SHOULD be "completed"
```

---

## AC7: Manual Status Override

### AC7.1: Setting manual override on feature
```gherkin
Given a feature "E07-F14" with calculated status "active"
When I run `shark feature update E07-F14 --status=blocked`
Then feature "E07-F14" status SHOULD be "blocked"
And status_override SHOULD be true
And status_source SHOULD be "manual"
```

### AC7.2: Clearing manual override on feature
```gherkin
Given a feature "E07-F14" with manual override status "blocked"
And the calculated status would be "active"
When I run `shark feature update E07-F14 --status=auto`
Then feature "E07-F14" status SHOULD be "active"
And status_override SHOULD be false
And status_source SHOULD be "calculated"
```

### AC7.3: Setting manual override on epic
```gherkin
Given an epic "E07" with calculated status "active"
When I run `shark epic update E07 --status=archived`
Then epic "E07" status SHOULD be "archived"
And status_override SHOULD be true
And status_source SHOULD be "manual"
```

### AC7.4: Child changes do not affect overridden parent
```gherkin
Given a feature "E07-F14" with manual override status "blocked"
And the feature has tasks with status "todo"
When I run `shark task start T-E07-F14-001`
Then task status changes to "in_progress"
And feature "E07-F14" status SHOULD remain "blocked"
And status_override SHOULD remain true
```

---

## AC8: CLI Output - Status Source Visibility

### AC8.1: Calculated status shows in feature get output
```gherkin
Given a feature "E07-F14" with calculated status "active"
And status_override is false
When I run `shark feature get E07-F14`
Then the output SHOULD include "Status: active (calculated)"
```

### AC8.2: Override status shows in feature get output
```gherkin
Given a feature "E07-F14" with overridden status "blocked"
And status_override is true
When I run `shark feature get E07-F14`
Then the output SHOULD include "Status: blocked (manual override)"
```

### AC8.3: JSON output includes status_source field
```gherkin
Given a feature "E07-F14" with calculated status "active"
When I run `shark feature get E07-F14 --json`
Then the JSON output SHOULD include field "status_source"
And the "status_source" value SHOULD be "calculated"
```

### AC8.4: JSON output shows manual for overrides
```gherkin
Given a feature "E07-F14" with overridden status "blocked"
When I run `shark feature get E07-F14 --json`
Then the JSON output SHOULD include field "status_source"
And the "status_source" value SHOULD be "manual"
```

---

## AC9: Edge Cases

### AC9.1: Deep cascade - task change affects feature and epic
```gherkin
Given an epic "E07" with status "draft"
And feature "E07-F14" with status "draft"
And all tasks in feature "E07-F14" have status "todo"
When I run `shark task start T-E07-F14-001`
Then task status changes to "in_progress"
And feature "E07-F14" status SHOULD be "active"
And epic "E07" status SHOULD be "active"
```

### AC9.2: All tasks blocked still means active feature
```gherkin
Given a feature "E07-F14" with 3 tasks
And all 3 tasks have status "blocked"
When the feature status is calculated
Then the feature status SHOULD be "active"
```

### AC9.3: Feature with single archived task
```gherkin
Given a feature "E07-F14" with 1 task
And the task has status "archived"
When the feature status is calculated
Then the feature status SHOULD be "completed"
```

### AC9.4: Epic inherits active status from nested task change
```gherkin
Given an epic "E07" with status "completed"
And all features have status "completed"
And all tasks in all features have status "completed"
When I add a new task to any feature
Then the feature status changes to "active"
And the epic status changes to "active"
```

---

## AC10: Backward Compatibility

### AC10.1: Existing entities without override column work
```gherkin
Given an existing database from before this feature
And features table does not have status_override column
When the migration runs
Then status_override column is added with default value false
And all existing features have status_override = false
```

### AC10.2: Legacy status values are preserved
```gherkin
Given an existing feature with status "active"
And no status_override column exists
When the migration runs
Then the feature status remains "active"
And status_source is treated as "calculated"
```
