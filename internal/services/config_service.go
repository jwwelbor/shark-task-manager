package services

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/jwwelbor/shark-task-manager/internal/config"
	"github.com/jwwelbor/shark-task-manager/internal/patterns"
)

// ConfigService encapsulates business logic for configuration management,
// including pattern validation, preset management, format generation, and
// workflow status-action resolution.
type ConfigService struct{}

// NewConfigService creates a new ConfigService.
func NewConfigService() *ConfigService {
	return &ConfigService{}
}

// PatternTestResult holds the result of testing a pattern against a string.
type PatternTestResult struct {
	Pattern    string            `json:"pattern"`
	TestString string            `json:"test_string"`
	Matched    bool              `json:"matched"`
	Groups     map[string]string `json:"groups,omitempty"`
}

// PatternValidationReport holds counts and details from validating all patterns.
type PatternValidationReport struct {
	EpicValid       int      `json:"epic_valid"`
	EpicErrors      []string `json:"epic_errors"`
	EpicWarnings    []string `json:"epic_warnings"`
	FeatureValid    int      `json:"feature_valid"`
	FeatureErrors   []string `json:"feature_errors"`
	FeatureWarnings []string `json:"feature_warnings"`
	TaskValid       int      `json:"task_valid"`
	TaskErrors      []string `json:"task_errors"`
	TaskWarnings    []string `json:"task_warnings"`
}

// HasErrors returns true if any pattern errors were found.
func (r *PatternValidationReport) HasErrors() bool {
	return len(r.EpicErrors) > 0 || len(r.FeatureErrors) > 0 || len(r.TaskErrors) > 0
}

// FormatOutput holds the result of a get-format query.
type FormatOutput struct {
	Format       string   `json:"format"`
	Example      string   `json:"example"`
	Placeholders []string `json:"placeholders"`
}

// PresetAddResult holds statistics from adding a pattern preset.
type PresetAddResult struct {
	Preset  string   `json:"preset"`
	Added   int      `json:"added"`
	Skipped int      `json:"skipped"`
	Details []string `json:"details"`
}

// LoadPatternsFromConfig loads pattern configuration from a file.
// Returns default patterns if the file has no patterns section.
func (s *ConfigService) LoadPatternsFromConfig(configPath string) (*patterns.PatternConfig, error) {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg struct {
		Patterns *patterns.PatternConfig `json:"patterns"`
	}

	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	if cfg.Patterns == nil {
		return patterns.GetDefaultPatterns(), nil
	}

	return cfg.Patterns, nil
}

// LoadDatabaseConfigFromFile loads the database configuration section from a config file.
func (s *ConfigService) LoadDatabaseConfigFromFile(configPath string) (map[string]interface{}, error) {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg struct {
		Database map[string]interface{} `json:"database"`
	}

	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	return cfg.Database, nil
}

// TestPattern tests a regex pattern against a test string and returns captured groups.
func (s *ConfigService) TestPattern(pattern, testString string) (*PatternTestResult, error) {
	re, err := regexp.Compile(pattern)
	if err != nil {
		return nil, fmt.Errorf("invalid regex syntax: %w", err)
	}

	result := &PatternTestResult{
		Pattern:    pattern,
		TestString: testString,
	}

	if !re.MatchString(testString) {
		return result, nil
	}

	result.Matched = true

	match := re.FindStringSubmatch(testString)
	names := re.SubexpNames()
	groups := make(map[string]string)
	for i, name := range names {
		if i > 0 && name != "" && i < len(match) {
			groups[name] = match[i]
		}
	}
	result.Groups = groups

	return result, nil
}

// FindMatchingPatterns finds patterns from a config that match a given test string.
func (s *ConfigService) FindMatchingPatterns(cfg *patterns.PatternConfig, testString, entityType string) []string {
	var patternsToCheck []string
	switch entityType {
	case "epic":
		patternsToCheck = cfg.Epic.Folder
	case "feature":
		patternsToCheck = cfg.Feature.Folder
	case "task":
		patternsToCheck = append(cfg.Task.Folder, cfg.Task.File...)
	}

	var matching []string
	for _, p := range patternsToCheck {
		result, err := s.TestPattern(p, testString)
		if err == nil && result.Matched {
			matching = append(matching, p)
		}
	}

	return matching
}

// ValidateAllPatterns validates all patterns in a PatternConfig and returns a report.
func (s *ConfigService) ValidateAllPatterns(cfg *patterns.PatternConfig) *PatternValidationReport {
	report := &PatternValidationReport{}

	report.EpicValid, report.EpicErrors = s.countPatternResults(cfg.Epic, "epic")
	report.FeatureValid, report.FeatureErrors = s.countPatternResults(cfg.Feature, "feature")
	report.TaskValid, report.TaskErrors = s.countPatternResults(cfg.Task, "task")

	report.EpicWarnings = s.getPatternWarnings(cfg.Epic, "epic")
	report.FeatureWarnings = s.getPatternWarnings(cfg.Feature, "feature")
	report.TaskWarnings = s.getPatternWarnings(cfg.Task, "task")

	return report
}

// countPatternResults counts valid patterns and errors for an entity.
func (s *ConfigService) countPatternResults(entity patterns.EntityPatterns, entityType string) (int, []string) {
	valid := 0
	var errs []string

	for i, p := range entity.Folder {
		if err := patterns.ValidatePattern(p, entityType); err != nil {
			errs = append(errs, fmt.Sprintf("folder pattern #%d: %v", i+1, err))
		} else {
			valid++
		}
	}

	for i, p := range entity.File {
		if err := patterns.ValidatePatternSyntaxOnly(p); err != nil {
			errs = append(errs, fmt.Sprintf("file pattern #%d: %v", i+1, err))
		} else {
			valid++
		}
	}

	return valid, errs
}

// getPatternWarnings gets all warnings for an entity's folder patterns.
func (s *ConfigService) getPatternWarnings(entity patterns.EntityPatterns, entityType string) []string {
	var warnings []string

	for i, p := range entity.Folder {
		for _, w := range patterns.GetPatternWarnings(p, entityType) {
			warnings = append(warnings, fmt.Sprintf("folder pattern #%d: %s", i+1, w))
		}
	}

	return warnings
}

// GetFormat returns format, example, and placeholders for an entity type.
func (s *ConfigService) GetFormat(cfg *patterns.PatternConfig, entityType string) (*FormatOutput, error) {
	var format string
	switch entityType {
	case "epic":
		format = cfg.Epic.Generation.Format
	case "feature":
		format = cfg.Feature.Generation.Format
	case "task":
		format = cfg.Task.Generation.Format
	default:
		return nil, fmt.Errorf("invalid type: %s (must be epic, feature, or task)", entityType)
	}

	example := s.generateFormatExample(entityType, format)
	placeholders := s.getPlaceholdersForType(entityType)

	return &FormatOutput{
		Format:       format,
		Example:      example,
		Placeholders: placeholders,
	}, nil
}

// generateFormatExample generates an example using a format template.
func (s *ConfigService) generateFormatExample(entityType, format string) string {
	values := map[string]interface{}{
		"number":  4,
		"epic":    4,
		"feature": 7,
		"slug":    "example-" + entityType,
	}

	result, err := patterns.ApplyGenerationFormat(format, values)
	if err != nil {
		return ""
	}
	return result
}

// getPlaceholdersForType returns available placeholders for an entity type.
func (s *ConfigService) getPlaceholdersForType(entityType string) []string {
	switch entityType {
	case "epic":
		return []string{"number", "slug"}
	case "feature":
		return []string{"epic", "number", "slug"}
	case "task":
		return []string{"epic", "feature", "number", "slug"}
	default:
		return []string{}
	}
}

// AddPreset merges a named pattern preset into the config file at configPath.
// Creates a backup of the existing file before writing. Returns merge statistics.
func (s *ConfigService) AddPreset(configPath, presetName string) (*PresetAddResult, error) {
	preset, err := patterns.GetPreset(presetName)
	if err != nil {
		return nil, fmt.Errorf("unknown preset %q: %w", presetName, err)
	}

	// Load existing patterns from config
	var currentConfig *patterns.PatternConfig
	if _, statErr := os.Stat(configPath); statErr == nil {
		data, readErr := os.ReadFile(configPath)
		if readErr != nil {
			return nil, fmt.Errorf("failed to read config file: %w", readErr)
		}

		var cfgData struct {
			Patterns json.RawMessage `json:"patterns"`
		}
		if parseErr := json.Unmarshal(data, &cfgData); parseErr != nil {
			return nil, fmt.Errorf("failed to parse config file: %w", parseErr)
		}

		if len(cfgData.Patterns) > 0 {
			currentConfig = &patterns.PatternConfig{}
			if parseErr := json.Unmarshal(cfgData.Patterns, currentConfig); parseErr != nil {
				return nil, fmt.Errorf("failed to parse patterns: %w", parseErr)
			}
		}
	}

	if currentConfig == nil {
		currentConfig = patterns.GetDefaultPatterns()
	}

	// Merge patterns
	mergedConfig, stats := patterns.MergePatternsWithStats(currentConfig, preset)

	// Validate merged patterns
	if validateErr := patterns.ValidatePatternConfig(mergedConfig); validateErr != nil {
		return nil, fmt.Errorf("pattern validation failed after merge: %w", validateErr)
	}

	// Read full config to preserve all other fields
	var fullConfig map[string]interface{}
	if data, readErr := os.ReadFile(configPath); readErr == nil {
		_ = json.Unmarshal(data, &fullConfig)
	} else {
		fullConfig = make(map[string]interface{})
	}

	// Update patterns section only
	fullConfig["patterns"] = mergedConfig

	// Marshal updated config
	data, marshalErr := json.MarshalIndent(fullConfig, "", "  ")
	if marshalErr != nil {
		return nil, fmt.Errorf("failed to marshal config: %w", marshalErr)
	}

	// Create backup before overwriting
	if _, statErr := os.Stat(configPath); statErr == nil {
		backupPath := configPath + ".bak"
		if renameErr := os.Rename(configPath, backupPath); renameErr == nil {
			defer func() {
				os.Remove(backupPath)
			}()
		}
		// Non-fatal: continue even if backup fails
	}

	// Write updated config
	if writeErr := os.WriteFile(configPath, data, 0644); writeErr != nil {
		return nil, fmt.Errorf("failed to write config file: %w", writeErr)
	}

	return &PresetAddResult{
		Preset:  presetName,
		Added:   stats.Added,
		Skipped: stats.Skipped,
		Details: stats.Details,
	}, nil
}

// ResolveStatusAction resolves the orchestrator action for a given status.
// If taskVars is non-nil, template variables are substituted.
// Returns (action, rawTemplate, nil) or (nil, "", err).
type StatusActionResult struct {
	Status      string   `json:"status"`
	Action      string   `json:"action"`
	AgentType   string   `json:"agent_type,omitempty"`
	Skills      []string `json:"skills,omitempty"`
	Instruction string   `json:"instruction"`
}

// GetStatusAction looks up the orchestrator action for a status in the workflow config.
// taskVars is used to populate template placeholders; pass nil for raw template.
func (s *ConfigService) GetStatusAction(configPath, status string, taskVars map[string]string) (*StatusActionResult, error) {
	workflowConfig, err := config.LoadWorkflowConfig(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load workflow config: %w", err)
	}

	if workflowConfig == nil || workflowConfig.StatusMetadata == nil {
		return nil, fmt.Errorf("no workflow configuration found")
	}

	metadata, exists := workflowConfig.StatusMetadata[status]
	if !exists {
		return nil, fmt.Errorf("status %q not found in workflow config", status)
	}

	action := metadata.OrchestratorAction
	if action == nil {
		return &StatusActionResult{
			Status: status,
		}, nil
	}

	var instruction string
	if taskVars != nil {
		instruction = action.PopulateTemplate(taskVars)
	} else {
		instruction = action.InstructionTemplate
	}

	return &StatusActionResult{
		Status:      status,
		Action:      action.Action,
		AgentType:   action.AgentType,
		Skills:      action.Skills,
		Instruction: instruction,
	}, nil
}

// ShowConfig returns all configuration sections for display.
type ShowConfigResult struct {
	JSON            bool                    `json:"json,omitempty"`
	NoColor         bool                    `json:"no_color,omitempty"`
	Verbose         bool                    `json:"verbose,omitempty"`
	Database        map[string]interface{}  `json:"database,omitempty"`
	Patterns        *patterns.PatternConfig `json:"patterns,omitempty"`
	WorkflowSources map[string]string       `json:"workflow_sources,omitempty"`
}

// BuildShowConfig assembles the full configuration summary from a config file.
func (s *ConfigService) BuildShowConfig(configPath string, jsonFlag, noColor, verbose bool) (*ShowConfigResult, error) {
	result := &ShowConfigResult{
		JSON:    jsonFlag,
		NoColor: noColor,
		Verbose: verbose,
	}

	if configPath == "" {
		return result, nil
	}

	dbConfig, err := s.LoadDatabaseConfigFromFile(configPath)
	if err == nil && dbConfig != nil {
		result.Database = dbConfig
	}

	patternsConfig, err := s.LoadPatternsFromConfig(configPath)
	if err == nil {
		result.Patterns = patternsConfig
	}

	multi := config.LoadMultiLevelWorkflowOrDefault(configPath)
	if multi.Sources != nil {
		result.WorkflowSources = multi.Sources
	}

	return result, nil
}

// PatternsOnlyConfig returns the patterns configuration from a config file.
// Falls back to defaults if no patterns section is found.
func (s *ConfigService) PatternsOnlyConfig(configPath string) (*patterns.PatternConfig, error) {
	if configPath == "" {
		return patterns.GetDefaultPatterns(), nil
	}

	cfg, err := s.LoadPatternsFromConfig(configPath)
	if err != nil {
		return patterns.GetDefaultPatterns(), nil
	}

	return cfg, nil
}

// FileExists reports whether the file at the given path exists.
func (s *ConfigService) FileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// PlaceholderString returns the available placeholders for an entity type as a comma-joined string.
func (s *ConfigService) PlaceholderString(entityType string) string {
	return strings.Join(s.getPlaceholdersForType(entityType), ", ")
}
