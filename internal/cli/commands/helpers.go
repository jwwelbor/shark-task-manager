package commands

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/jwwelbor/shark-task-manager/internal/cli"
	"github.com/jwwelbor/shark-task-manager/internal/config"
	"github.com/jwwelbor/shark-task-manager/internal/keys"
	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/repository"
)

// NormalizeKey converts a key to canonical uppercase format.
// This enables case-insensitive key handling throughout the CLI.
// Delegates to keys.Normalize for implementation.
//
// Examples:
//
//	e01 -> E01
//	t-e04-f02-001 -> T-E04-F02-001
//	E01-FEATURE-NAME -> E01-FEATURE-NAME
func NormalizeKey(key string) string {
	return keys.Normalize(key)
}

// IsEpicKey validates if a string is a valid epic key format (E##)
// Case insensitive: e01, E01, and E-01 are all normalized to E01 before validation
// Returns true for valid epic keys like E01, e04, E99
// Returns false for invalid formats like E1, E001, etc.
// Delegates to keys.IsEpicKey for implementation.
func IsEpicKey(s string) bool {
	return keys.IsEpicKey(s)
}

// IsFeatureKey validates if a string is a valid feature key format (E##-F##)
// Case insensitive: e04-f01, E04-F01 are normalized before validation
// Returns true for valid feature keys like E04-F01, e01-f99
// Returns false for invalid formats like E04F01, E4-F01, etc.
// Delegates to keys.IsFeatureKey for implementation.
func IsFeatureKey(s string) bool {
	return keys.IsFeatureKey(s)
}

// IsFeatureKeySuffix validates if a string is a valid feature key suffix (F##)
// Case insensitive: f01, F01 are normalized before validation
// Returns true for valid suffixes like F01, f99
// Returns false for invalid formats like F1, etc.
// Delegates to keys.IsFeatureKeySuffix for implementation.
func IsFeatureKeySuffix(s string) bool {
	return keys.IsFeatureKeySuffix(s)
}

// IsBugKey validates if a string is a valid bug key format (B###).
// Case insensitive: b001 is normalized to B001 before validation.
// Delegates to keys.IsBugKey for implementation.
func IsBugKey(s string) bool {
	return keys.IsBugKey(s)
}

// IsChangeKey validates if a string is a valid change key format (C###).
// Case insensitive: c001 is normalized to C001 before validation.
// Delegates to keys.IsChangeKey for implementation.
func IsChangeKey(s string) bool {
	return keys.IsChangeKey(s)
}

// IsChangeCardKey validates if a string is a valid change-card key format (CC-###).
// Case insensitive: cc-001 is normalized to CC-001 before validation.
// Delegates to keys.IsChangeCardKey for implementation.
func IsChangeCardKey(s string) bool {
	return keys.IsChangeCardKey(s)
}

// IsTechDebtKey validates if a string is a valid tech-debt key format (TD-###).
// Case insensitive: td-001 is normalized to TD-001 before validation.
// Delegates to keys.IsTechDebtKey for implementation.
func IsTechDebtKey(s string) bool {
	return keys.IsTechDebtKey(s)
}

// IsIdeaKey validates if a string is a valid idea key format (I-YYYY-MM-DD-##).
// Case insensitive: i-2026-01-01-01 is normalized to I-2026-01-01-01 before validation.
// Delegates to keys.IsIdeaKey for implementation.
func IsIdeaKey(s string) bool {
	return keys.IsIdeaKey(s)
}

// IsSprintKey validates if a string is a valid sprint key format (S###).
// Sprint keys are strict 3-digit zero-padded: S001, S024, S999.
// Delegates to keys.IsSprintKey for implementation.
func IsSprintKey(s string) bool {
	return keys.IsSprintKey(s)
}

// ParseFeatureKey parses a combined feature key format (E##-F##) into epic and feature parts
// Case insensitive: normalizes input to uppercase before parsing
// Returns (epic, feature, nil) for valid input like "E04-F01" or "e04-f01"
// Returns ("", "", error) for invalid input with clear error message
// Delegates to keys.ParseFeatureKey for implementation.
func ParseFeatureKey(s string) (epic, feature string, err error) {
	epic, feature, err = keys.ParseFeatureKey(s)
	if err != nil {
		return "", "", InvalidFeatureKeyError(s)
	}
	return epic, feature, nil
}

// ParseFeatureListArgs parses positional arguments for feature list command
// Case insensitive: normalizes epic key to uppercase
// Returns (epicKey, nil) if valid, or (nil, error) if invalid
// Handles 0 or 1 positional arguments
func ParseFeatureListArgs(args []string) (*string, error) {
	if len(args) == 0 {
		// No filter
		return nil, nil
	}

	if len(args) > 1 {
		return nil, TooManyArgumentsError(1, len(args))
	}

	// Normalize the epic key
	epicKey := NormalizeKey(args[0])

	// Validate format
	if !IsEpicKey(epicKey) {
		return nil, InvalidEpicKeyError(args[0])
	}

	return &epicKey, nil
}

// ParseTaskListArgs parses positional arguments for task list command
// Case insensitive: normalizes all keys to uppercase
// Supports 0-2 arguments with multiple syntaxes:
// - No args: list all tasks
// - 1 arg (E##): filter by epic
// - 1 arg (E##-F##): filter by epic and feature
// - 2 args (E## and F## or E##-F##): filter by epic and feature
// Returns (epicKey, featureKey, nil) on success, or (nil, nil, error) on failure
func ParseTaskListArgs(args []string) (*string, *string, error) {
	if len(args) == 0 {
		// No filter
		return nil, nil, nil
	}

	if len(args) > 2 {
		return nil, nil, TooManyArgumentsError(2, len(args))
	}

	// Single argument case
	if len(args) == 1 {
		normalized := NormalizeKey(args[0])

		// Check if it's a combined feature key (E##-F##)
		if IsFeatureKey(normalized) {
			epic, feature, err := ParseFeatureKey(normalized)
			if err != nil {
				return nil, nil, err
			}
			return &epic, &feature, nil
		}

		// Check if it's just an epic key (E##)
		if IsEpicKey(normalized) {
			return &normalized, nil, nil
		}

		// Check if it looks like it was trying to be an epic key (starts with E)
		if len(normalized) > 0 && normalized[0] == 'E' {
			return nil, nil, InvalidEpicKeyError(args[0])
		}

		// Check if it looks like it was trying to be a feature key (contains dash or starts with F)
		if strings.Contains(normalized, "-") || (len(normalized) > 0 && normalized[0] == 'F') {
			return nil, nil, InvalidFeatureKeyError(args[0])
		}

		// Generic invalid format
		return nil, nil, InvalidPositionalArgsError("task list",
			fmt.Sprintf("invalid key format %q - expected E## or E##-F##", args[0]),
			[]string{
				"shark task list E07",
				"shark task list E07-F01",
				"shark task list e07",
			})
	}

	// Two argument case
	epicNormalized := NormalizeKey(args[0])
	featureNormalized := NormalizeKey(args[1])

	// First argument must be an epic key
	if !IsEpicKey(epicNormalized) {
		return nil, nil, InvalidEpicKeyError(args[0])
	}

	// Second argument can be a feature suffix (F##) or full feature key (E##-F##)
	featureSuffix, err := resolveFeatureSuffix(featureNormalized, args[1])
	if err != nil {
		return nil, nil, err
	}
	return &epicNormalized, &featureSuffix, nil
}

// ParseListArgs parses positional arguments for the list command dispatcher
// Case insensitive: normalizes all keys to uppercase
// Returns (command, epicKey, featureKey, error)
// - command: "epic", "feature", or "task"
// - epicKey: pointer to epic key if applicable
// - featureKey: pointer to feature key if applicable
// Supports:
// - 0 args: list epics
// - 1 arg (E##): list features in epic
// - 1 arg (E##-F##): list tasks in feature
// - 2 args (E## F## or E## E##-F##): list tasks in epic+feature
func ParseListArgs(args []string) (command string, epicKey, featureKey *string, err error) {
	if len(args) == 0 {
		// No args: list epics
		return "epic", nil, nil, nil
	}

	if len(args) > 2 {
		return "", nil, nil, TooManyArgumentsError(2, len(args))
	}

	// Single argument case
	if len(args) == 1 {
		normalized := NormalizeKey(args[0])

		// Check if it's "idea" or "ideas" keyword
		if normalized == "IDEA" || normalized == "IDEAS" {
			return "idea", nil, nil, nil
		}

		// Check if it's "bug" or "bugs" keyword
		if normalized == "BUG" || normalized == "BUGS" {
			return "bug", nil, nil, nil
		}

		// Check if it's "change", "changes", or "change-card(s)" keyword
		if normalized == "CHANGE" || normalized == "CHANGES" || normalized == "CHANGE-CARD" || normalized == "CHANGE-CARDS" {
			return "change", nil, nil, nil
		}

		// Check if it's "tech-debt", "tech_debt", "techdebt", or "td" keyword.
		// The dispatcher in list.go already has a "tech_debt" case; this enables
		// `shark list tech-debt` / `shark list td` to reach it.
		if normalized == "TECH-DEBT" || normalized == "TECH_DEBT" || normalized == "TECHDEBT" || normalized == "TD" {
			return "tech_debt", nil, nil, nil
		}

		// Check if it's "sprint" or "sprints" keyword
		if normalized == "SPRINT" || normalized == "SPRINTS" {
			return "sprint", nil, nil, nil
		}

		// Check if it's a sprint key (S###) — list sprint backlog
		if IsSprintKey(normalized) {
			return "sprint", &normalized, nil, nil
		}

		// Check if it's a combined feature key (E##-F##)
		if IsFeatureKey(normalized) {
			epic, feature, err := ParseFeatureKey(normalized)
			if err != nil {
				return "", nil, nil, err
			}
			return "task", &epic, &feature, nil
		}

		// Check if it's just an epic key (E##)
		if IsEpicKey(normalized) {
			return "feature", &normalized, nil, nil
		}

		// Check if it looks like it was trying to be an epic key (starts with E)
		if len(normalized) > 0 && normalized[0] == 'E' {
			return "", nil, nil, InvalidEpicKeyError(args[0])
		}

		// Check if it looks like it was trying to be a feature key (contains dash or starts with F)
		if strings.Contains(normalized, "-") || (len(normalized) > 0 && normalized[0] == 'F') {
			return "", nil, nil, InvalidFeatureKeyError(args[0])
		}

		// Generic invalid format
		return "", nil, nil, InvalidPositionalArgsError("list",
			fmt.Sprintf("invalid key format %q - expected E##, E##-F##, bugs, changes, tech-debt, or ideas", args[0]),
			[]string{
				"shark list E07",
				"shark list E07-F01",
				"shark list bugs",
				"shark list changes",
				"shark list tech-debt",
			})
	}

	// Two argument case
	epicNormalized := NormalizeKey(args[0])
	featureNormalized := NormalizeKey(args[1])

	// First argument must be an epic key
	if !IsEpicKey(epicNormalized) {
		return "", nil, nil, InvalidEpicKeyError(args[0])
	}

	// Second argument can be a feature suffix (F##) or full feature key (E##-F##)
	featureSuffix, err := resolveFeatureSuffix(featureNormalized, args[1])
	if err != nil {
		return "", nil, nil, err
	}
	return "task", &epicNormalized, &featureSuffix, nil
}

// ParseGetArgs parses positional arguments for the get command dispatcher
// Now delegates to scope.Interpreter for DRY (Don't Repeat Yourself) principle
// Case insensitive: normalizes all keys to uppercase
// Returns (command, key, error)
// - command: "epic", "feature", or "task"
// - key: The full key to pass to the get command (E10, E10-F01, T-E10-F01-001)
// Supports:
// - 1 arg (E##): get epic
// - 1 arg (E##-F##): get feature
// - 1 arg (T-E##-F##-###): get task
// - 2 args (E## F## or E## E##-F##): get feature
// - 3 args (E## F## ### or E## F## #): get task
func ParseGetArgs(args []string) (command string, key string, err error) {
	// Import scope package to avoid circular dependency
	// Use scope.Interpreter for parsing
	interpreter := newScopeInterpreter()
	parsedScope, err := interpreter.ParseScope(args)
	if err != nil {
		// Wrap error with context if needed for backward compatibility
		if len(args) == 0 {
			return "", "", MissingArgumentsError(1, 0, []string{
				"shark get E07",
				"shark get E07-F01",
				"shark get T-E07-F01-001",
			})
		}
		return "", "", err
	}

	return string(parsedScope.Type), parsedScope.Key, nil
}

// newScopeInterpreter creates a scope interpreter instance
// This is a thin wrapper to avoid direct dependency on scope package in this file
func newScopeInterpreter() scopeInterpreter {
	return &scopeInterpreterImpl{}
}

// scopeInterpreter interface defines the contract for scope parsing
type scopeInterpreter interface {
	ParseScope(args []string) (*parsedScope, error)
}

// parsedScope represents a parsed CLI scope
type parsedScope struct {
	Type scopeType
	Key  string
}

type scopeType string

const (
	scopeEpic       scopeType = "epic"
	scopeFeature    scopeType = "feature"
	scopeTask       scopeType = "task"
	scopeBug        scopeType = "bug"
	scopeChange     scopeType = "change"
	scopeChangeCard scopeType = "change_card"
	scopeTechDebt   scopeType = "tech_debt"
	scopeIdea       scopeType = "idea"
	scopeSprint     scopeType = "sprint"
)

// scopeInterpreterImpl implements scopeInterpreter using existing helper functions
type scopeInterpreterImpl struct{}

func (s *scopeInterpreterImpl) ParseScope(args []string) (*parsedScope, error) {
	if len(args) == 0 {
		return nil, fmt.Errorf("no arguments provided")
	}

	if len(args) > 3 {
		return nil, TooManyArgumentsError(3, len(args))
	}

	// Single argument case
	if len(args) == 1 {
		normalized := NormalizeKey(args[0])

		// Check if it's a bug key (B###)
		if IsBugKey(normalized) {
			return &parsedScope{Type: scopeBug, Key: normalized}, nil
		}

		// Check if it's a change key (C###)
		if IsChangeKey(normalized) {
			return &parsedScope{Type: scopeChange, Key: normalized}, nil
		}

		// Check if it's a change-card key (CC-###)
		if IsChangeCardKey(normalized) {
			return &parsedScope{Type: scopeChangeCard, Key: normalized}, nil
		}

		// Check if it's an idea key (I-YYYY-MM-DD-##)
		if IsIdeaKey(normalized) {
			return &parsedScope{Type: scopeIdea, Key: normalized}, nil
		}

		// Check if it's a tech-debt key (TD-###)
		// Must be checked BEFORE task key (T-E##-F##-###) to prevent TD- being
		// matched as a T- prefix collision
		if IsTechDebtKey(normalized) {
			return &parsedScope{Type: scopeTechDebt, Key: normalized}, nil
		}

		// Check if it's a task key (T-E##-F##-###)
		if isTaskKey(normalized) {
			return &parsedScope{Type: scopeTask, Key: normalized}, nil
		}

		// Check if it's a short task key (E##-F##-###)
		normalizedTaskKey, err := NormalizeTaskKey(normalized)
		if err == nil {
			return &parsedScope{Type: scopeTask, Key: normalizedTaskKey}, nil
		}

		// Check if it's a combined feature key (E##-F##)
		if IsFeatureKey(normalized) {
			return &parsedScope{Type: scopeFeature, Key: normalized}, nil
		}

		// Check if it's just an epic key (E##)
		if IsEpicKey(normalized) {
			return &parsedScope{Type: scopeEpic, Key: normalized}, nil
		}

		// Check if it's a sprint key (S###)
		if IsSprintKey(normalized) {
			return &parsedScope{Type: scopeSprint, Key: normalized}, nil
		}

		// Check if it looks like it was trying to be a tech-debt key (starts with TD)
		if strings.HasPrefix(normalized, "TD") {
			return nil, InvalidPositionalArgsError("get",
				fmt.Sprintf("invalid tech-debt key format %q - expected TD-### (e.g., TD-001)", args[0]),
				[]string{
					"shark get TD-001",
					"shark get TD-042",
					"shark td get TD-001",
				})
		}

		// Check if it looks like it was trying to be a task key (starts with T)
		if len(normalized) > 0 && normalized[0] == 'T' {
			return nil, InvalidTaskKeyError(args[0])
		}

		// Check if it looks like it was trying to be an epic key (starts with E but no dash)
		if len(normalized) > 0 && normalized[0] == 'E' && !strings.Contains(normalized, "-") {
			return nil, InvalidEpicKeyError(args[0])
		}

		// Check if it looks like it was trying to be a feature key (contains dash or starts with F)
		if strings.Contains(normalized, "-") || (len(normalized) > 0 && normalized[0] == 'F') {
			return nil, InvalidFeatureKeyError(args[0])
		}

		// Check if it looks like it was trying to be a bug key (starts with B but no digits)
		if len(normalized) > 0 && normalized[0] == 'B' {
			return nil, InvalidPositionalArgsError("get",
				fmt.Sprintf("invalid bug key format %q - expected B### (e.g., B001)", args[0]),
				[]string{
					"shark get B001",
					"shark get B042",
					"shark bug get B001",
				})
		}

		// Check if it looks like it was trying to be a change-card key (starts with C but no digits)
		if len(normalized) > 0 && normalized[0] == 'C' {
			return nil, InvalidPositionalArgsError("get",
				fmt.Sprintf("invalid change card key format %q - expected C### (e.g., C001)", args[0]),
				[]string{
					"shark get C001",
					"shark get C042",
					"shark change get C001",
				})
		}

		// Generic invalid format
		return nil, InvalidPositionalArgsError("get",
			fmt.Sprintf("invalid key format %q - expected E## (epic), E##-F## (feature), E##-F##-### (task), B### (bug), C### (change card), TD-### (tech debt), or S### (sprint)", args[0]),
			[]string{
				"shark get E07",
				"shark get E07-F01",
				"shark get T-E07-F01-001",
				"shark get B001",
				"shark get C001",
				"shark get TD-001",
				"shark get S001",
			})
	}

	// Two argument case - must be epic + feature
	if len(args) == 2 {
		epicNormalized := NormalizeKey(args[0])
		featureNormalized := NormalizeKey(args[1])

		// First argument must be an epic key
		if !IsEpicKey(epicNormalized) {
			return nil, InvalidEpicKeyError(args[0])
		}

		// Second argument can be a feature suffix (F##) or full feature key (E##-F##)
		featureSuffix, err := resolveFeatureSuffix(featureNormalized, args[1])
		if err != nil {
			return nil, err
		}

		// Construct full feature key
		fullFeatureKey := epicNormalized + "-" + featureSuffix
		return &parsedScope{Type: scopeFeature, Key: fullFeatureKey}, nil
	}

	// Three argument case - must be epic + feature + task number
	epicNormalized := NormalizeKey(args[0])
	featureNormalized := NormalizeKey(args[1])
	arg3 := args[2]

	// First argument must be an epic key
	if !IsEpicKey(epicNormalized) {
		return nil, InvalidEpicKeyError(args[0])
	}

	// Second argument can be a feature suffix (F##) or full feature key (E##-F##)
	featureSuffix, err := resolveFeatureSuffix(featureNormalized, args[1])
	if err != nil {
		return nil, err
	}

	// Third argument must be a task number (1-999)
	taskNum, err := parseTaskNumber(arg3)
	if err != nil {
		return nil, err
	}

	// Construct full task key
	fullTaskKey := fmt.Sprintf("T-%s-%s-%03d", epicNormalized, featureSuffix, taskNum)
	return &parsedScope{Type: scopeTask, Key: fullTaskKey}, nil
}

// resolveFeatureSuffix resolves a feature argument to a feature suffix (F##).
// Accepts either a feature suffix (F##) or a full feature key (E##-F##).
// Returns the suffix and nil on success, or empty string and an error on failure.
func resolveFeatureSuffix(featureArg, rawArg string) (string, error) {
	if IsFeatureKeySuffix(featureArg) {
		return featureArg, nil
	}
	if IsFeatureKey(featureArg) {
		_, suffix, err := ParseFeatureKey(featureArg)
		if err != nil {
			return "", err
		}
		return suffix, nil
	}
	return "", InvalidFeatureKeyError(rawArg)
}

// isTaskKey validates if a string is a valid task key format (T-E##-F##-###)
// Delegates to keys.IsTaskKey for implementation.
func isTaskKey(s string) bool {
	return keys.IsTaskKey(s)
}

// parseTaskNumber parses a task number string and validates it's in range 1-999
// Delegates to keys.ParseTaskNumber for implementation.
func parseTaskNumber(s string) (int, error) {
	return keys.ParseTaskNumber(s)
}

// isShortTaskKey validates if a string matches the short task key pattern (E##-F##-###)
// This is a helper function for NormalizeTaskKey to detect short format task keys.
// Short format omits the T- prefix for brevity: "E01-F02-001" instead of "T-E01-F02-001"
// Delegates to keys.IsShortTaskKey for implementation.
func isShortTaskKey(s string) bool {
	return keys.IsShortTaskKey(s)
}

// NormalizeTaskKey converts a task key to canonical format with T- prefix.
// Accepts both full format (T-E##-F##-###) and short format (E##-F##-###).
// This enables users to type shorter commands while maintaining backward compatibility.
// Delegates to keys.NormalizeTaskKey for implementation.
//
// Examples:
//
//	T-E01-F02-001 → T-E01-F02-001 (no change)
//	e01-f02-001 → T-E01-F02-001 (add prefix, uppercase)
//	E01-F02-001 → T-E01-F02-001 (add prefix)
//	e01-f02-001-task-name → T-E01-F02-001-TASK-NAME (slugged, add prefix)
//
// This function is part of T-E07-F20-006: Add short task key pattern and normalization
func NormalizeTaskKey(input string) (string, error) {
	result, err := keys.NormalizeTaskKey(input)
	if err != nil {
		return "", InvalidTaskKeyError(input)
	}
	return result, nil
}

// ParseFeatureCreateArgs parses positional arguments for feature create command
// Supports: shark feature create [EPIC] "TITLE"
// Returns (epicKey, title, nil) on success, or (nil, nil, error) on failure
// Case insensitive: normalizes epic key to uppercase
func ParseFeatureCreateArgs(args []string) (*string, *string, error) {
	// Expected: 2 arguments (EPIC TITLE)
	if len(args) < 2 {
		return nil, nil, MissingArgumentsError(2, len(args), []string{
			"shark feature create E07 \"Feature Title\"",
			"shark feature create E04 \"User Management\"",
		})
	}

	if len(args) > 2 {
		return nil, nil, TooManyArgumentsError(2, len(args))
	}

	// Parse and normalize epic key
	epicKey := NormalizeKey(args[0])

	// Validate epic key format
	if !IsEpicKey(epicKey) {
		return nil, nil, InvalidEpicKeyError(args[0])
	}

	// Title is the second argument (taken as-is, no normalization)
	title := args[1]

	return &epicKey, &title, nil
}

// ParseTaskCreateArgs parses positional arguments for task create command
// Supports:
//   - shark task create [EPIC] [FEATURE] "TITLE" (3 arguments)
//   - shark task create [EPIC-FEATURE] "TITLE" (2 arguments)
//
// Returns (epicKey, featureKey, title, nil) on success, or (nil, nil, nil, error) on failure
// Case insensitive: normalizes epic and feature keys to uppercase
func ParseTaskCreateArgs(args []string) (*string, *string, *string, error) {
	// Expected: 2 or 3 arguments
	if len(args) < 2 {
		return nil, nil, nil, MissingArgumentsError(2, len(args), []string{
			"shark task create E07 F01 \"Task Title\"",
			"shark task create E07-F01 \"Task Title\"",
		})
	}

	if len(args) > 3 {
		return nil, nil, nil, TooManyArgumentsError(3, len(args))
	}

	// Case 1: 3 arguments (EPIC FEATURE TITLE)
	if len(args) == 3 {
		epicKey := NormalizeKey(args[0])
		featureArg := NormalizeKey(args[1])
		title := args[2]

		// Validate epic key format
		if !IsEpicKey(epicKey) {
			return nil, nil, nil, InvalidEpicKeyError(args[0])
		}

		// Feature can be either F## (suffix) or E##-F## (full key)
		featureKey, err := resolveFeatureSuffix(featureArg, args[1])
		if err != nil {
			return nil, nil, nil, err
		}

		return &epicKey, &featureKey, &title, nil
	}

	// Case 2: 2 arguments (EPIC-FEATURE TITLE)
	combinedKey := NormalizeKey(args[0])
	title := args[1]

	// Must be a valid feature key (E##-F##)
	if !IsFeatureKey(combinedKey) {
		return nil, nil, nil, InvalidFeatureKeyError(args[0])
	}

	// Parse the combined key
	epicKey, featureKey, err := ParseFeatureKey(combinedKey)
	if err != nil {
		return nil, nil, nil, err
	}

	return &epicKey, &featureKey, &title, nil
}

// DetectEntityType detects the entity type from a key string.
// Returns "epic", "feature", "task", "bug", "change", "change_card", "tech_debt", or "unknown" based on the key format.
// Case insensitive: e07, E07, E07-enhancements all return "epic"
//
// Examples:
//
//	B001 -> "bug"
//	C001 -> "change"
//	CC-001 -> "change_card"
//	TD-001 -> "tech_debt"
//	E07 -> "epic"
//	E07-user-management -> "epic"
//	E07-F01 -> "feature"
//	F01 -> "feature"
//	F01-auth -> "feature"
//	E07-F01-auth -> "feature"
//	T-E07-F01-001 -> "task"
//	E07-F01-001 -> "task"
//	E07-F01-001-slug -> "task"
//	invalid -> "unknown"
func DetectEntityType(key string) string {
	// Handle empty string
	if key == "" {
		return "unknown"
	}

	// Normalize to uppercase for case-insensitive matching
	normalized := NormalizeKey(key)

	// Check bug key (B###) before task patterns to avoid false matches
	if IsBugKey(normalized) {
		return "bug"
	}

	// Check change key (C###) before task patterns
	if IsChangeKey(normalized) {
		return "change"
	}

	// Check change-card key (CC-###) before task patterns
	if IsChangeCardKey(normalized) {
		return "change_card"
	}

	// Check idea key (I-YYYY-MM-DD-##)
	if IsIdeaKey(normalized) {
		return "idea"
	}

	// Check tech-debt key (TD-###) BEFORE task patterns
	// TD- starts with T, so it must be checked before T- task prefix matching
	if IsTechDebtKey(normalized) {
		return "tech_debt"
	}

	// Check sprint key (S###)
	if IsSprintKey(normalized) {
		return "sprint"
	}

	// Check task patterns first (most specific)
	// Full format: T-E##-F##-###
	if isTaskKey(normalized) {
		return "task"
	}

	// Short format task keys (E##-F##-### or with slug)
	// Try normalizing as task key - this handles both short format and slugged keys
	if _, err := NormalizeTaskKey(normalized); err == nil {
		return "task"
	}

	// Check feature key patterns
	// Full feature key: E##-F##
	if IsFeatureKey(normalized) {
		return "feature"
	}

	// Feature suffix: F##
	if IsFeatureKeySuffix(normalized) {
		return "feature"
	}

	// Check epic patterns
	// Base epic key: E##
	if IsEpicKey(normalized) {
		return "epic"
	}

	// Check for slugged keys: E##-slug (epic) or E##-F##-slug (feature)
	// Split and validate parts - reject keys with empty segments (e.g., "E07-")
	parts := strings.Split(normalized, "-")
	for _, p := range parts {
		if p == "" {
			return "unknown"
		}
	}

	// Feature with slug: E##-F##-slug (parts[0]=E##, parts[1]=F##, rest=slug)
	// Must check before epic-with-slug since E##-F##-slug also starts with E##
	if len(parts) >= 3 && IsEpicKey(parts[0]) && IsFeatureKeySuffix(parts[1]) {
		return "feature"
	}

	// Feature suffix with slug: F##-slug (parts[0]=F##, rest=slug)
	// This handles F01-authentication, F20-cli-improvements, etc.
	if len(parts) >= 2 && IsFeatureKeySuffix(parts[0]) {
		return "feature"
	}

	// Epic with slug: E##-slug (parts[0]=E##, rest=slug)
	// This handles E07-enhancements, E07-user-management, etc.
	if len(parts) >= 2 && IsEpicKey(parts[0]) {
		return "epic"
	}

	return "unknown"
}

// applySizeLabelToMap injects "size_label" into a JSON map when the entity carries a
// non-nil Size field. This satisfies E07-F42 REQ-F-007 ("--field size_label" must be
// a separately extractable field) for all six entity types.
//
// sizable is any value whose concrete type implements GetSize() *int (models.Entity,
// *models.Idea, or any future sizable type). The function is a no-op when s is nil,
// when GetSize() returns nil, or when the size does not map to a canonical label.
func applySizeLabelToMap(result map[string]interface{}, sizable interface{}) {
	type sizeGetter interface {
		GetSize() *int
	}
	sg, ok := sizable.(sizeGetter)
	if !ok || sg == nil {
		return
	}
	size := sg.GetSize()
	if size == nil {
		return
	}
	if label, err := models.SizeLabel(*size); err == nil {
		result["size_label"] = label
	}
}

// buildEnrichedJSON converts an entity struct to a map[string]interface{} and adds
// valid_transitions, orchestrator_action, and (when present) size_label fields.
// This is a shared helper used by bug get and change get commands.
func buildEnrichedJSON(entity interface{}, orchestratorAction *config.PopulatedAction, validTransitions []string) (map[string]interface{}, error) {
	entityJSON, err := json.Marshal(entity)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal entity: %w", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(entityJSON, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal entity JSON to map: %w", err)
	}

	// E07-F42 REQ-F-007: inject size_label so --field size_label is extractable.
	applySizeLabelToMap(result, entity)

	result["valid_transitions"] = validTransitions
	result["orchestrator_action"] = orchestratorAction

	return result, nil
}

// getAgentIdentifier returns flagValue if non-empty, otherwise falls back to the USER env var.
func getAgentIdentifier(flagValue string) string {
	if flagValue != "" {
		return flagValue
	}
	if user := os.Getenv("USER"); user != "" {
		return user
	}
	return ""
}

// derefString safely dereferences a *string, returning "" if nil.
func derefString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// handleServiceError translates service errors into user-friendly CLI messages and exit codes.
//
// Exit codes:
//   - 1: Not found (entity doesn't exist)
//   - 2: Database/system error
//   - 3: Invalid state (business rule violation)
//
// Parameters:
//   - err: the error returned by a service method (nil returns immediately)
//   - entityType: human-readable entity type for messages (e.g., "feature", "epic", "task")
//   - key: entity key for messages (e.g., "E07-F01", "E07")
func handleServiceError(err error, entityType, key string) {
	if err == nil {
		return
	}

	if errors.Is(err, repository.ErrNotFound) {
		displayType := strings.ToUpper(entityType[:1]) + entityType[1:]
		cli.Error(fmt.Sprintf("%s not found: %s", displayType, key))
		cli.Info(fmt.Sprintf("Use 'shark %s list' to see available %ss", entityType, entityType))
		os.Exit(1)
	}

	// Check if error message contains "not found" patterns (no specific NotFoundError type in repo)
	errMsg := err.Error()
	if strings.Contains(errMsg, "not found") || strings.Contains(errMsg, "does not exist") {
		displayType := strings.ToUpper(entityType[:1]) + entityType[1:]
		cli.Error(fmt.Sprintf("%s not found: %s", displayType, key))
		cli.Info(fmt.Sprintf("Use 'shark %s list' to see available %ss", entityType, entityType))
		os.Exit(1)
	}

	// Check for conflict/validation errors (exit code 3 = invalid state)
	if strings.Contains(errMsg, "already exists") ||
		strings.Contains(errMsg, "validation failed") ||
		strings.Contains(errMsg, "cannot be empty") ||
		strings.Contains(errMsg, "invalid transition") ||
		strings.Contains(errMsg, "cannot start") ||
		strings.Contains(errMsg, "cannot complete") {
		cli.Error(fmt.Sprintf("Error: %v", err))
		os.Exit(3)
	}

	// Default: database/system error
	cli.Error(fmt.Sprintf("Error: %v", err))
	if cli.GlobalConfig.Verbose {
		slog.Debug("Error details", "error", err)
	}
	os.Exit(2)
}

// formatSize renders a size pointer for human-readable display.
// Returns "—" when s is nil; otherwise returns "<label> (<num>)" (e.g., "L (5)").
// Per spec.md §3.6 and REQ-F-006 (E07-F42).
//
// The defensive branch ("should never trigger") handles the theoretical case
// where a non-canonical value somehow reached the model — it falls back to
// the raw number rather than panicking.
func formatSize(s *int) string {
	if s == nil {
		return "—"
	}
	label, err := models.SizeLabel(*s)
	if err != nil {
		// Defensive: canonical validation should have caught this upstream.
		return fmt.Sprintf("%d", *s)
	}
	return fmt.Sprintf("%s (%d)", label, *s)
}
