// Package dependency provides circular dependency detection for task dependencies.
//
// This package implements a graph-based algorithm using Depth-First Search (DFS)
// to detect cycles in task dependency graphs. It ensures that adding new dependencies
// will not create circular dependencies, which would make task completion impossible.
package dependency

import (
	"context"
	"fmt"
)

// Detector implements circular dependency detection using graph traversal.
// It maintains an adjacency list representation of task dependencies and
// uses DFS-based cycle detection.
type Detector struct {
	// graph maps each task to its dependencies (adjacency list)
	// Key: task key (e.g., "T-E01-F01-001")
	// Value: slice of task keys this task depends on
	graph map[string][]string
}

// NewDetector creates a new dependency detector with an empty graph.
func NewDetector() *Detector {
	return &Detector{
		graph: make(map[string][]string),
	}
}

// AddDependency adds a dependency edge to the graph.
// task depends on dependency (task -> dependency).
func (d *Detector) AddDependency(task string, dependency string) {
	if d.graph[task] == nil {
		d.graph[task] = []string{}
	}
	d.graph[task] = append(d.graph[task], dependency)
}

// RemoveDependency removes a dependency edge from the graph.
func (d *Detector) RemoveDependency(task string, dependency string) {
	deps, exists := d.graph[task]
	if !exists {
		return
	}

	// Filter out the dependency
	newDeps := []string{}
	for _, dep := range deps {
		if dep != dependency {
			newDeps = append(newDeps, dep)
		}
	}
	d.graph[task] = newDeps
}

// ClearGraph removes all dependencies from the graph.
func (d *Detector) ClearGraph() {
	d.graph = make(map[string][]string)
}

// DetectCycle performs DFS-based cycle detection starting from the given task.
// Returns:
// - hasCycle: true if a cycle is detected
// - cyclePath: the path of tasks forming the cycle (empty if no cycle)
// - error: non-nil if a cycle is detected, describing the circular dependency
func (d *Detector) DetectCycle(ctx context.Context, startTask string) (bool, []string, error) {
	// Track visited nodes in current path (for cycle detection)
	visiting := make(map[string]bool)
	// Track fully visited nodes (optimization)
	visited := make(map[string]bool)
	// Track the path
	path := []string{}

	hasCycle, cyclePath := d.dfs(startTask, visiting, visited, &path)
	if hasCycle {
		return true, cyclePath, fmt.Errorf("circular dependency detected: %v", cyclePath)
	}
	return false, nil, nil
}

// dfs performs depth-first search to detect cycles.
// Returns (hasCycle, cyclePath).
func (d *Detector) dfs(task string, visiting map[string]bool, visited map[string]bool, path *[]string) (bool, []string) {
	// If we've already fully processed this node, no cycle here
	if visited[task] {
		return false, nil
	}

	// If we're currently visiting this node, we found a cycle
	if visiting[task] {
		// Find where the cycle starts in the path
		cycleStart := -1
		for i, t := range *path {
			if t == task {
				cycleStart = i
				break
			}
		}
		// Return the cycle path (from cycle start to current + the task again to close the cycle)
		cyclePath := append((*path)[cycleStart:], task)
		return true, cyclePath
	}

	// Mark as visiting
	visiting[task] = true
	*path = append(*path, task)

	// Visit all dependencies
	for _, dep := range d.graph[task] {
		if hasCycle, cyclePath := d.dfs(dep, visiting, visited, path); hasCycle {
			return true, cyclePath
		}
	}

	// Done visiting this node
	*path = (*path)[:len(*path)-1]
	visiting[task] = false
	visited[task] = true

	return false, nil
}

// ValidateDependency checks if adding a new dependency would create a cycle.
// It simulates adding the dependency and checks for cycles.
// Returns error if the dependency would create a cycle or if task depends on itself.
func (d *Detector) ValidateDependency(ctx context.Context, task string, dependency string) error {
	// Check for self-reference
	if task == dependency {
		return fmt.Errorf("task cannot depend on itself: %s", task)
	}

	// Temporarily add the dependency
	d.AddDependency(task, dependency)

	// Check for cycles
	hasCycle, cyclePath, _ := d.DetectCycle(ctx, task)

	// Remove the temporary dependency
	d.RemoveDependency(task, dependency)

	if hasCycle {
		return fmt.Errorf("would create circular dependency: %v", cyclePath)
	}

	return nil
}

// ValidateMultipleDependencies checks if adding multiple dependencies would create a cycle.
// It validates each dependency individually.
// Returns error on the first dependency that would create a cycle.
func (d *Detector) ValidateMultipleDependencies(ctx context.Context, task string, dependencies []string) error {
	for _, dep := range dependencies {
		if err := d.ValidateDependency(ctx, task, dep); err != nil {
			return err
		}
	}
	return nil
}

// GetDependencyChain returns the full dependency chain for a task.
// This performs a topological traversal to find all transitive dependencies.
// Returns empty slice if task is not in the graph.
func (d *Detector) GetDependencyChain(ctx context.Context, task string) []string {
	// Check if task exists in graph
	_, exists := d.graph[task]
	if !exists {
		return []string{}
	}

	visited := make(map[string]bool)
	chain := []string{}
	d.buildChain(task, visited, &chain)
	return chain
}

// buildChain recursively builds the dependency chain.
func (d *Detector) buildChain(task string, visited map[string]bool, chain *[]string) {
	if visited[task] {
		return
	}

	visited[task] = true
	*chain = append(*chain, task)

	// Visit all dependencies if they exist
	deps, exists := d.graph[task]
	if exists {
		for _, dep := range deps {
			d.buildChain(dep, visited, chain)
		}
	}
}
