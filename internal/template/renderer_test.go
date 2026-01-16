package template

import (
	"testing"
)

func TestRenderer_BasicSubstitution(t *testing.T) {
	renderer := NewRenderer()

	tests := []struct {
		name     string
		template string
		context  map[string]string
		want     string
	}{
		{
			name:     "single task_id substitution",
			template: "Implement task {task_id}",
			context:  map[string]string{"task_id": "T-E07-F21-002"},
			want:     "Implement task T-E07-F21-002",
		},
		{
			name:     "multiple task_id occurrences",
			template: "Task {task_id} ({task_id})",
			context:  map[string]string{"task_id": "T-001"},
			want:     "Task T-001 (T-001)",
		},
		{
			name:     "unknown variable left unchanged",
			template: "Task {task_id} with {unknown}",
			context:  map[string]string{"task_id": "T-001"},
			want:     "Task T-001 with {unknown}",
		},
		{
			name:     "no variables",
			template: "This is plain text",
			context:  map[string]string{"task_id": "T-001"},
			want:     "This is plain text",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := renderer.Render(tt.template, tt.context)
			if got != tt.want {
				t.Errorf("Render() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestRenderer_EdgeCases(t *testing.T) {
	renderer := NewRenderer()

	tests := []struct {
		name     string
		template string
		context  map[string]string
		want     string
	}{
		{
			name:     "empty template",
			template: "",
			context:  map[string]string{"task_id": "T-001"},
			want:     "",
		},
		{
			name:     "nil context",
			template: "Task {task_id}",
			context:  nil,
			want:     "Task {task_id}",
		},
		{
			name:     "empty context",
			template: "Task {task_id}",
			context:  map[string]string{},
			want:     "Task {task_id}",
		},
		{
			name:     "variable not in context",
			template: "Task {task_id} blocked by {blocker_id}",
			context:  map[string]string{"task_id": "T-001"},
			want:     "Task T-001 blocked by {blocker_id}",
		},
		{
			name:     "malformed variable - missing closing brace",
			template: "Task {task_id with {nested}",
			context:  map[string]string{"task_id": "T-001", "nested": "N-001"},
			want:     "Task {task_id with N-001",
		},
		{
			name:     "malformed variable - missing opening brace",
			template: "Task task_id} with value",
			context:  map[string]string{"task_id": "T-001"},
			want:     "Task task_id} with value",
		},
		{
			name:     "case sensitive",
			template: "Task {Task_Id} vs {task_id}",
			context:  map[string]string{"task_id": "T-001"},
			want:     "Task {Task_Id} vs T-001",
		},
		{
			name:     "empty braces",
			template: "Task {} is empty",
			context:  map[string]string{"": "T-001"},
			want:     "Task T-001 is empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := renderer.Render(tt.template, tt.context)
			if got != tt.want {
				t.Errorf("Render() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestRenderer_ContextVariations(t *testing.T) {
	renderer := NewRenderer()

	tests := []struct {
		name     string
		template string
		context  map[string]string
		want     string
	}{
		{
			name:     "single variable in context",
			template: "Task {task_id}",
			context:  map[string]string{"task_id": "T-E07-F21-002"},
			want:     "Task T-E07-F21-002",
		},
		{
			name:     "multiple variables in context, only task_id used",
			template: "Task {task_id}",
			context: map[string]string{
				"task_id":    "T-001",
				"epic_id":    "E07",
				"feature_id": "F21",
				"priority":   "5",
			},
			want: "Task T-001",
		},
		{
			name:     "complex template with instruction",
			template: "Launch developer agent to implement task {task_id}. Write tests first.",
			context:  map[string]string{"task_id": "T-E07-F21-002"},
			want:     "Launch developer agent to implement task T-E07-F21-002. Write tests first.",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := renderer.Render(tt.template, tt.context)
			if got != tt.want {
				t.Errorf("Render() = %q, want %q", got, tt.want)
			}
		})
	}
}

// BenchmarkRenderer tests performance - target <1ms per render
func BenchmarkRenderer_SimpleTemplate(b *testing.B) {
	renderer := NewRenderer()
	context := map[string]string{"task_id": "T-E07-F21-002"}
	template := "Launch developer agent to implement task {task_id}. Write tests first."

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = renderer.Render(template, context)
	}
}

func BenchmarkRenderer_ComplexTemplate(b *testing.B) {
	renderer := NewRenderer()
	context := map[string]string{
		"task_id":    "T-E07-F21-002",
		"epic_id":    "E07",
		"feature_id": "F21",
		"priority":   "5",
	}
	// Longer template with multiple variables
	template := "Task {task_id} in feature {feature_id} is blocked. Reason: {blocker}. Priority: {priority}. Epic: {epic_id}."

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = renderer.Render(template, context)
	}
}

func BenchmarkRenderer_1000Renders(b *testing.B) {
	renderer := NewRenderer()
	context := map[string]string{"task_id": "T-E07-F21-002"}
	template := "Implement {task_id} following TDD practices and project standards"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for j := 0; j < 1000; j++ {
			_ = renderer.Render(template, context)
		}
	}
}
