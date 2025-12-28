package dependency

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDetector_DetectCycle(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name           string
		dependencies   map[string][]string // task -> dependencies
		startTask      string
		expectedCycle  bool
		expectedPath   []string // expected cycle path if cycle exists
		expectedErrMsg string
	}{
		{
			name: "no cycle - linear chain",
			dependencies: map[string][]string{
				"T-E01-F01-001": {},
				"T-E01-F01-002": {"T-E01-F01-001"},
				"T-E01-F01-003": {"T-E01-F01-002"},
			},
			startTask:     "T-E01-F01-003",
			expectedCycle: false,
		},
		{
			name: "simple cycle - A depends on B, B depends on A",
			dependencies: map[string][]string{
				"T-E01-F01-001": {"T-E01-F01-002"},
				"T-E01-F01-002": {"T-E01-F01-001"},
			},
			startTask:      "T-E01-F01-001",
			expectedCycle:  true,
			expectedPath:   []string{"T-E01-F01-001", "T-E01-F01-002", "T-E01-F01-001"},
			expectedErrMsg: "circular dependency detected",
		},
		{
			name: "three-task cycle - A->B->C->A",
			dependencies: map[string][]string{
				"T-E01-F01-001": {"T-E01-F01-002"},
				"T-E01-F01-002": {"T-E01-F01-003"},
				"T-E01-F01-003": {"T-E01-F01-001"},
			},
			startTask:      "T-E01-F01-001",
			expectedCycle:  true,
			expectedPath:   []string{"T-E01-F01-001", "T-E01-F01-002", "T-E01-F01-003", "T-E01-F01-001"},
			expectedErrMsg: "circular dependency detected",
		},
		{
			name: "complex cycle with branch - A->B, A->C, B->D, C->D, D->A",
			dependencies: map[string][]string{
				"T-E01-F01-001": {"T-E01-F01-002", "T-E01-F01-003"},
				"T-E01-F01-002": {"T-E01-F01-004"},
				"T-E01-F01-003": {"T-E01-F01-004"},
				"T-E01-F01-004": {"T-E01-F01-001"},
			},
			startTask:      "T-E01-F01-001",
			expectedCycle:  true,
			expectedErrMsg: "circular dependency detected",
		},
		{
			name: "self-reference",
			dependencies: map[string][]string{
				"T-E01-F01-001": {"T-E01-F01-001"},
			},
			startTask:      "T-E01-F01-001",
			expectedCycle:  true,
			expectedPath:   []string{"T-E01-F01-001", "T-E01-F01-001"},
			expectedErrMsg: "circular dependency detected",
		},
		{
			name: "no cycle - diamond dependency",
			dependencies: map[string][]string{
				"T-E01-F01-001": {},
				"T-E01-F01-002": {"T-E01-F01-001"},
				"T-E01-F01-003": {"T-E01-F01-001"},
				"T-E01-F01-004": {"T-E01-F01-002", "T-E01-F01-003"},
			},
			startTask:     "T-E01-F01-004",
			expectedCycle: false,
		},
		{
			name: "cycle in middle of chain - A->B->C->D->B",
			dependencies: map[string][]string{
				"T-E01-F01-001": {"T-E01-F01-002"},
				"T-E01-F01-002": {"T-E01-F01-003"},
				"T-E01-F01-003": {"T-E01-F01-004"},
				"T-E01-F01-004": {"T-E01-F01-002"},
			},
			startTask:      "T-E01-F01-001",
			expectedCycle:  true,
			expectedErrMsg: "circular dependency detected",
		},
		{
			name: "no dependencies",
			dependencies: map[string][]string{
				"T-E01-F01-001": {},
			},
			startTask:     "T-E01-F01-001",
			expectedCycle: false,
		},
		{
			name: "task not in graph",
			dependencies: map[string][]string{
				"T-E01-F01-001": {},
			},
			startTask:     "T-E01-F01-002",
			expectedCycle: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			detector := NewDetector()

			// Build the graph
			for task, deps := range tt.dependencies {
				for _, dep := range deps {
					detector.AddDependency(task, dep)
				}
			}

			// Detect cycle
			hasCycle, cyclePath, err := detector.DetectCycle(ctx, tt.startTask)

			if tt.expectedCycle {
				assert.True(t, hasCycle, "expected cycle to be detected")
				require.Error(t, err, "expected error when cycle detected")
				assert.Contains(t, err.Error(), tt.expectedErrMsg)
				assert.NotEmpty(t, cyclePath, "expected cycle path to be populated")

				if len(tt.expectedPath) > 0 {
					assert.Equal(t, tt.expectedPath, cyclePath, "cycle path mismatch")
				}

				// Verify cycle path is valid
				if len(cyclePath) > 0 {
					// First and last should be the same (completing the cycle)
					assert.Equal(t, cyclePath[0], cyclePath[len(cyclePath)-1], "cycle should start and end with same task")
				}
			} else {
				assert.False(t, hasCycle, "expected no cycle")
				assert.NoError(t, err, "expected no error when no cycle")
				assert.Empty(t, cyclePath, "expected empty cycle path")
			}
		})
	}
}

func TestDetector_ValidateDependency(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name           string
		existingDeps   map[string][]string
		newTask        string
		newDependency  string
		expectValid    bool
		expectedErrMsg string
	}{
		{
			name: "valid - no cycle created",
			existingDeps: map[string][]string{
				"T-E01-F01-001": {},
				"T-E01-F01-002": {"T-E01-F01-001"},
			},
			newTask:       "T-E01-F01-003",
			newDependency: "T-E01-F01-002",
			expectValid:   true,
		},
		{
			name: "invalid - would create simple cycle",
			existingDeps: map[string][]string{
				"T-E01-F01-001": {"T-E01-F01-002"},
				"T-E01-F01-002": {},
			},
			newTask:        "T-E01-F01-002",
			newDependency:  "T-E01-F01-001",
			expectValid:    false,
			expectedErrMsg: "would create circular dependency",
		},
		{
			name: "invalid - would create complex cycle",
			existingDeps: map[string][]string{
				"T-E01-F01-001": {"T-E01-F01-002"},
				"T-E01-F01-002": {"T-E01-F01-003"},
				"T-E01-F01-003": {},
			},
			newTask:        "T-E01-F01-003",
			newDependency:  "T-E01-F01-001",
			expectValid:    false,
			expectedErrMsg: "would create circular dependency",
		},
		{
			name:           "invalid - self-reference",
			existingDeps:   map[string][]string{},
			newTask:        "T-E01-F01-001",
			newDependency:  "T-E01-F01-001",
			expectValid:    false,
			expectedErrMsg: "task cannot depend on itself",
		},
		{
			name: "valid - adding to independent branch",
			existingDeps: map[string][]string{
				"T-E01-F01-001": {"T-E01-F01-002"},
				"T-E01-F01-002": {},
				"T-E01-F01-003": {"T-E01-F01-004"},
				"T-E01-F01-004": {},
			},
			newTask:       "T-E01-F01-005",
			newDependency: "T-E01-F01-003",
			expectValid:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			detector := NewDetector()

			// Build existing graph
			for task, deps := range tt.existingDeps {
				for _, dep := range deps {
					detector.AddDependency(task, dep)
				}
			}

			// Validate new dependency
			err := detector.ValidateDependency(ctx, tt.newTask, tt.newDependency)

			if tt.expectValid {
				assert.NoError(t, err, "expected validation to pass")
			} else {
				require.Error(t, err, "expected validation to fail")
				assert.Contains(t, err.Error(), tt.expectedErrMsg)
			}
		})
	}
}

func TestDetector_ValidateMultipleDependencies(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name            string
		existingDeps    map[string][]string
		newTask         string
		newDependencies []string
		expectValid     bool
		expectedErrMsg  string
	}{
		{
			name: "valid - multiple dependencies, no cycle",
			existingDeps: map[string][]string{
				"T-E01-F01-001": {},
				"T-E01-F01-002": {},
				"T-E01-F01-003": {},
			},
			newTask:         "T-E01-F01-004",
			newDependencies: []string{"T-E01-F01-001", "T-E01-F01-002", "T-E01-F01-003"},
			expectValid:     true,
		},
		{
			name: "invalid - one dependency creates cycle",
			existingDeps: map[string][]string{
				"T-E01-F01-001": {"T-E01-F01-002"},
				"T-E01-F01-002": {"T-E01-F01-003"},
				"T-E01-F01-003": {},
			},
			newTask:         "T-E01-F01-003",
			newDependencies: []string{"T-E01-F01-001", "T-E01-F01-004"},
			expectValid:     false,
			expectedErrMsg:  "would create circular dependency",
		},
		{
			name:            "invalid - self-reference in list",
			existingDeps:    map[string][]string{},
			newTask:         "T-E01-F01-001",
			newDependencies: []string{"T-E01-F01-002", "T-E01-F01-001", "T-E01-F01-003"},
			expectValid:     false,
			expectedErrMsg:  "task cannot depend on itself",
		},
		{
			name:            "valid - empty dependencies",
			existingDeps:    map[string][]string{},
			newTask:         "T-E01-F01-001",
			newDependencies: []string{},
			expectValid:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			detector := NewDetector()

			// Build existing graph
			for task, deps := range tt.existingDeps {
				for _, dep := range deps {
					detector.AddDependency(task, dep)
				}
			}

			// Validate new dependencies
			err := detector.ValidateMultipleDependencies(ctx, tt.newTask, tt.newDependencies)

			if tt.expectValid {
				assert.NoError(t, err, "expected validation to pass")
			} else {
				require.Error(t, err, "expected validation to fail")
				assert.Contains(t, err.Error(), tt.expectedErrMsg)
			}
		})
	}
}

func TestDetector_GetDependencyChain(t *testing.T) {
	ctx := context.Background()

	detector := NewDetector()

	// Build a dependency chain: 004 -> 003 -> 002 -> 001
	detector.AddDependency("T-E01-F01-004", "T-E01-F01-003")
	detector.AddDependency("T-E01-F01-003", "T-E01-F01-002")
	detector.AddDependency("T-E01-F01-002", "T-E01-F01-001")

	chain := detector.GetDependencyChain(ctx, "T-E01-F01-004")

	expected := []string{"T-E01-F01-004", "T-E01-F01-003", "T-E01-F01-002", "T-E01-F01-001"}
	assert.Equal(t, expected, chain, "dependency chain mismatch")
}

func TestDetector_ClearGraph(t *testing.T) {
	detector := NewDetector()

	// Add some dependencies
	detector.AddDependency("T-E01-F01-001", "T-E01-F01-002")
	detector.AddDependency("T-E01-F01-002", "T-E01-F01-003")

	// Clear the graph
	detector.ClearGraph()

	// Graph should be empty
	ctx := context.Background()
	chain := detector.GetDependencyChain(ctx, "T-E01-F01-001")
	assert.Empty(t, chain, "expected empty chain after clear")
}

func TestDetector_RemoveDependency(t *testing.T) {
	ctx := context.Background()

	detector := NewDetector()

	// Build graph
	detector.AddDependency("T-E01-F01-001", "T-E01-F01-002")
	detector.AddDependency("T-E01-F01-002", "T-E01-F01-003")

	// Remove a dependency
	detector.RemoveDependency("T-E01-F01-001", "T-E01-F01-002")

	// Should no longer have path from 001 to 003
	chain := detector.GetDependencyChain(ctx, "T-E01-F01-001")
	assert.NotContains(t, chain, "T-E01-F01-002", "dependency should be removed")
}
