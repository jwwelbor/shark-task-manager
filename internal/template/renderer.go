package template

import (
	"fmt"
	"strings"
)

// TemplateRenderer renders instruction templates with variable substitution.
type TemplateRenderer interface {
	// Render replaces variables in template with values from context.
	// Variables are in the format {variable_name}.
	// Unknown variables are left as-is.
	// If context is nil or empty, variables are left unchanged.
	Render(template string, context map[string]string) string
}

// NewRenderer creates a new template renderer.
func NewRenderer() TemplateRenderer {
	return &simpleRenderer{}
}

// simpleRenderer implements TemplateRenderer with simple string replacement.
type simpleRenderer struct{}

// Render replaces variables in template with values from context.
func (r *simpleRenderer) Render(template string, context map[string]string) string {
	if template == "" {
		return ""
	}

	if context == nil {
		context = make(map[string]string)
	}

	result := template
	for key, value := range context {
		placeholder := fmt.Sprintf("{%s}", key)
		result = strings.ReplaceAll(result, placeholder, value)
	}
	return result
}
