package scope

import (
	"fmt"

	"github.com/jwwelbor/shark-task-manager/internal/keys"
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
		normalized := keys.Normalize(args[0])

		// Check if it's a task key (T-E##-F##-### or E##-F##-###)
		if keys.IsTaskKey(normalized) {
			return "task", normalized, nil
		}

		// Check if it's a short task key (E##-F##-###)
		normalizedTaskKey, err := keys.NormalizeTaskKey(normalized)
		if err == nil {
			return "task", normalizedTaskKey, nil
		}

		// Check if it's a combined feature key (E##-F##)
		if keys.IsFeatureKey(normalized) {
			return "feature", normalized, nil
		}

		// Check if it's just an epic key (E##)
		if keys.IsEpicKey(normalized) {
			return "epic", normalized, nil
		}

		// Invalid format
		return "", "", fmt.Errorf("invalid key format: %q", args[0])
	}

	// Two argument case - must be epic + feature
	if len(args) == 2 {
		epicNormalized := keys.Normalize(args[0])
		featureNormalized := keys.Normalize(args[1])

		// First argument must be an epic key
		if !keys.IsEpicKey(epicNormalized) {
			return "", "", fmt.Errorf("invalid epic key: %q", args[0])
		}

		// Second argument can be a feature suffix (F##) or full feature key (E##-F##)
		var featureSuffix string
		if keys.IsFeatureKeySuffix(featureNormalized) {
			featureSuffix = featureNormalized
		} else if keys.IsFeatureKey(featureNormalized) {
			// Full feature key - extract the feature suffix
			_, suffix, err := keys.ParseFeatureKey(featureNormalized)
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
	epicNormalized := keys.Normalize(args[0])
	featureNormalized := keys.Normalize(args[1])
	arg3 := args[2]

	// First argument must be an epic key
	if !keys.IsEpicKey(epicNormalized) {
		return "", "", fmt.Errorf("invalid epic key: %q", args[0])
	}

	// Second argument can be a feature suffix (F##) or full feature key (E##-F##)
	var featureSuffix string
	if keys.IsFeatureKeySuffix(featureNormalized) {
		featureSuffix = featureNormalized
	} else if keys.IsFeatureKey(featureNormalized) {
		// Full feature key - extract the feature suffix
		_, suffix, err := keys.ParseFeatureKey(featureNormalized)
		if err != nil {
			return "", "", err
		}
		featureSuffix = suffix
	} else {
		return "", "", fmt.Errorf("invalid feature key: %q", args[1])
	}

	// Third argument must be a task number (1-999)
	taskNum, err := keys.ParseTaskNumber(arg3)
	if err != nil {
		return "", "", err
	}

	// Construct full task key
	fullTaskKey := fmt.Sprintf("T-%s-%s-%03d", epicNormalized, featureSuffix, taskNum)
	return "task", fullTaskKey, nil
}
