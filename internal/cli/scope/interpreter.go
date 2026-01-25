package scope

import (
	"fmt"
	"strings"
)

// Interpreter parses CLI arguments to determine scope (epic, feature, or task)
// This is a reusable component that can be used across multiple commands
type Interpreter struct{}

// NewInterpreter creates a new scope interpreter
func NewInterpreter() *Interpreter {
	return &Interpreter{}
}

// ParseScope parses CLI arguments and returns the scope
// It supports multiple formats:
//
//	Epic:
//	  - ParseScope(["E01"]) -> (ScopeEpic, "E01", nil)
//
//	Feature:
//	  - ParseScope(["E01", "F01"]) -> (ScopeFeature, "E01-F01", nil)
//	  - ParseScope(["E01-F01"]) -> (ScopeFeature, "E01-F01", nil)
//	  - ParseScope(["E01", "E01-F01"]) -> (ScopeFeature, "E01-F01", nil)
//
//	Task:
//	  - ParseScope(["T-E01-F01-001"]) -> (ScopeTask, "T-E01-F01-001", nil)
//	  - ParseScope(["E01-F01-001"]) -> (ScopeTask, "T-E01-F01-001", nil) (short format)
//	  - ParseScope(["E01", "F01", "001"]) -> (ScopeTask, "T-E01-F01-001", nil)
//
// All keys are normalized to uppercase before returning
func (i *Interpreter) ParseScope(args []string) (*Scope, error) {
	if i == nil {
		return nil, fmt.Errorf("interpreter is nil")
	}

	if len(args) == 0 {
		return nil, fmt.Errorf("no arguments provided")
	}

	if len(args) > 3 {
		return nil, fmt.Errorf("too many arguments: expected 1-3, got %d", len(args))
	}

	// Import the helper functions from commands package
	// We'll delegate to the existing logic in ParseGetArgs
	command, key, err := parseGetArgsLogic(args)
	if err != nil {
		return nil, err
	}

	// Convert command string to ScopeType
	var scopeType ScopeType
	switch command {
	case "epic":
		scopeType = ScopeEpic
	case "feature":
		scopeType = ScopeFeature
	case "task":
		scopeType = ScopeTask
	default:
		return nil, fmt.Errorf("unknown scope type: %s", command)
	}

	return &Scope{
		Type: scopeType,
		Key:  key,
	}, nil
}

// parseGetArgsLogic is extracted from the commands.ParseGetArgs function
// This contains the core parsing logic without the dependency on the commands package
func parseGetArgsLogic(args []string) (command string, key string, err error) {
	// Single argument case
	if len(args) == 1 {
		normalized := normalizeKey(args[0])

		// Check if it's a task key (T-E##-F##-### or E##-F##-###)
		if isTaskKey(normalized) {
			return "task", normalized, nil
		}

		// Check if it's a short task key (E##-F##-###)
		normalizedTaskKey, err := normalizeTaskKey(normalized)
		if err == nil {
			return "task", normalizedTaskKey, nil
		}

		// Check if it's a combined feature key (E##-F##)
		if isFeatureKey(normalized) {
			return "feature", normalized, nil
		}

		// Check if it's just an epic key (E##)
		if isEpicKey(normalized) {
			return "epic", normalized, nil
		}

		// Invalid format
		return "", "", fmt.Errorf("invalid key format: %q", args[0])
	}

	// Two argument case - must be epic + feature
	if len(args) == 2 {
		epicNormalized := normalizeKey(args[0])
		featureNormalized := normalizeKey(args[1])

		// First argument must be an epic key
		if !isEpicKey(epicNormalized) {
			return "", "", fmt.Errorf("invalid epic key: %q", args[0])
		}

		// Second argument can be a feature suffix (F##) or full feature key (E##-F##)
		var featureSuffix string
		if isFeatureKeySuffix(featureNormalized) {
			featureSuffix = featureNormalized
		} else if isFeatureKey(featureNormalized) {
			// Full feature key - extract the feature suffix
			_, suffix, err := parseFeatureKey(featureNormalized)
			if err != nil {
				return "", "", err
			}
			featureSuffix = suffix
		} else {
			return "", "", fmt.Errorf("invalid feature key: %q", args[1])
		}

		// Construct full feature key
		fullFeatureKey := epicNormalized + "-" + featureSuffix
		return "feature", fullFeatureKey, nil
	}

	// Three argument case - must be epic + feature + task number
	epicNormalized := normalizeKey(args[0])
	featureNormalized := normalizeKey(args[1])
	arg3 := args[2]

	// First argument must be an epic key
	if !isEpicKey(epicNormalized) {
		return "", "", fmt.Errorf("invalid epic key: %q", args[0])
	}

	// Second argument can be a feature suffix (F##) or full feature key (E##-F##)
	var featureSuffix string
	if isFeatureKeySuffix(featureNormalized) {
		featureSuffix = featureNormalized
	} else if isFeatureKey(featureNormalized) {
		// Full feature key - extract the feature suffix
		_, suffix, err := parseFeatureKey(featureNormalized)
		if err != nil {
			return "", "", err
		}
		featureSuffix = suffix
	} else {
		return "", "", fmt.Errorf("invalid feature key: %q", args[1])
	}

	// Third argument must be a task number (1-999)
	taskNum, err := parseTaskNumber(arg3)
	if err != nil {
		return "", "", err
	}

	// Construct full task key
	fullTaskKey := fmt.Sprintf("T-%s-%s-%03d", epicNormalized, featureSuffix, taskNum)
	return "task", fullTaskKey, nil
}

// Helper functions (duplicated from commands package for now, will refactor)

func normalizeKey(key string) string {
	return strings.ToUpper(key)
}

func isEpicKey(s string) bool {
	// Epic key format: E##
	if len(s) != 3 {
		return false
	}
	if s[0] != 'E' {
		return false
	}
	// Check both digits
	return s[1] >= '0' && s[1] <= '9' && s[2] >= '0' && s[2] <= '9'
}

func isFeatureKey(s string) bool {
	// Feature key format: E##-F##
	if len(s) != 7 {
		return false
	}
	if s[3] != '-' {
		return false
	}
	return isEpicKey(s[:3]) && isFeatureKeySuffix(s[4:])
}

func isFeatureKeySuffix(s string) bool {
	// Feature suffix format: F##
	if len(s) != 3 {
		return false
	}
	if s[0] != 'F' {
		return false
	}
	return s[1] >= '0' && s[1] <= '9' && s[2] >= '0' && s[2] <= '9'
}

func parseFeatureKey(s string) (epic, feature string, err error) {
	if !isFeatureKey(s) {
		return "", "", fmt.Errorf("invalid feature key format: %q", s)
	}
	return s[:3], s[4:7], nil
}

func isTaskKey(s string) bool {
	// Task key format: T-E##-F##-###
	if len(s) < 13 {
		return false
	}
	if !strings.HasPrefix(s, "T-") {
		return false
	}
	// Check epic part (E##)
	if !isEpicKey(s[2:5]) {
		return false
	}
	if s[5] != '-' {
		return false
	}
	// Check feature part (F##)
	if !isFeatureKeySuffix(s[6:9]) {
		return false
	}
	if s[9] != '-' {
		return false
	}
	// Check task number part (###)
	if len(s) >= 13 {
		taskNumStr := s[10:13]
		for _, ch := range taskNumStr {
			if ch < '0' || ch > '9' {
				return false
			}
		}
		return true
	}
	return false
}

func normalizeTaskKey(input string) (string, error) {
	if input == "" {
		return "", fmt.Errorf("empty task key")
	}

	normalized := strings.ToUpper(input)

	// Already has T- prefix
	if strings.HasPrefix(normalized, "T-") {
		if isTaskKey(normalized) {
			return normalized, nil
		}
		return "", fmt.Errorf("invalid task key format: %q", input)
	}

	// Check if it matches short format (E##-F##-###)
	if isShortTaskKey(normalized) {
		return "T-" + normalized, nil
	}

	// Check for slugged short format (E##-F##-###-slug)
	parts := strings.SplitN(normalized, "-", 4)
	if len(parts) >= 4 {
		keyPart := strings.Join(parts[:3], "-")
		if isShortTaskKey(keyPart) {
			return "T-" + normalized, nil
		}
	}

	return "", fmt.Errorf("invalid task key format: %q", input)
}

func isShortTaskKey(s string) bool {
	// Short task key format: E##-F##-###
	if len(s) < 11 {
		return false
	}
	// Extract first 11 characters for the key part
	keyPart := s
	if len(s) > 11 {
		keyPart = s[:11]
	}

	if !isEpicKey(keyPart[:3]) {
		return false
	}
	if keyPart[3] != '-' {
		return false
	}
	if !isFeatureKeySuffix(keyPart[4:7]) {
		return false
	}
	if keyPart[7] != '-' {
		return false
	}
	// Check task number
	taskNum := keyPart[8:11]
	for _, ch := range taskNum {
		if ch < '0' || ch > '9' {
			return false
		}
	}
	return true
}

func parseTaskNumber(s string) (int, error) {
	num := 0
	for _, ch := range s {
		if ch < '0' || ch > '9' {
			return 0, fmt.Errorf("invalid task number: %q (must be numeric 1-999)", s)
		}
		num = num*10 + int(ch-'0')
	}

	if num < 1 || num > 999 {
		return 0, fmt.Errorf("invalid task number: %q (must be between 1 and 999)", s)
	}

	return num, nil
}
