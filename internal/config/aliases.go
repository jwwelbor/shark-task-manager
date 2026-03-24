package config

// aliases.go provides backward-compatible type aliases and function re-exports
// for symbols moved to sub-packages. All existing callers continue to use the
// config.XxxType and config.XxxFunc names unchanged.
//
// This file covers symbols from:
// - internal/config/validation/ (OrchestratorValidationError, ValidationError)
// - internal/config/template/ (placeholder functions, enrichment types)
//
// Pattern follows internal/repository/aliases.go (established in E07-F36).

import (
	cfgtemplate "github.com/jwwelbor/shark-task-manager/internal/config/template"
	"github.com/jwwelbor/shark-task-manager/internal/config/validation"
)

// --- validation/ types ---

// OrchestratorValidationError is an alias for validation.OrchestratorValidationError.
type OrchestratorValidationError = validation.OrchestratorValidationError

// ValidationError is an alias for validation.ValidationError (which is itself
// an alias for OrchestratorValidationError).
type ValidationError = validation.ValidationError

// --- template/ types ---

// TemplateEnrichmentData is an alias for template.TemplateEnrichmentData.
type TemplateEnrichmentData = cfgtemplate.TemplateEnrichmentData

// TemplateEnrichmentRepository is an alias for template.TemplateEnrichmentRepository.
type TemplateEnrichmentRepository = cfgtemplate.TemplateEnrichmentRepository

// DocumentRepository is an alias for template.DocumentRepository.
type DocumentRepository = cfgtemplate.DocumentRepository

// FeatureRelationshipRepository is an alias for template.FeatureRelationshipRepository.
type FeatureRelationshipRepository = cfgtemplate.FeatureRelationshipRepository

// EpicRelationshipRepository is an alias for template.EpicRelationshipRepository.
type EpicRelationshipRepository = cfgtemplate.EpicRelationshipRepository

// TaskRelationshipRepository is an alias for template.TaskRelationshipRepository.
type TaskRelationshipRepository = cfgtemplate.TaskRelationshipRepository

// --- template/ functions ---

// EntityPlaceholders creates a map of template placeholders from any Entity.
var EntityPlaceholders = cfgtemplate.EntityPlaceholders

// TaskPlaceholders creates a map of template placeholders from a Task.
var TaskPlaceholders = cfgtemplate.TaskPlaceholders

// FeaturePlaceholders creates a map of template placeholders from a Feature.
var FeaturePlaceholders = cfgtemplate.FeaturePlaceholders

// EpicPlaceholders creates a map of template placeholders from an Epic.
var EpicPlaceholders = cfgtemplate.EpicPlaceholders

// BugPlaceholders creates a map of template placeholders from a Bug.
var BugPlaceholders = cfgtemplate.BugPlaceholders

// ChangeCardPlaceholders creates a map of template placeholders from a ChangeCard.
var ChangeCardPlaceholders = cfgtemplate.ChangeCardPlaceholders

// TaskPlaceholdersWithRelated extends TaskPlaceholders with relationship data.
var TaskPlaceholdersWithRelated = cfgtemplate.TaskPlaceholdersWithRelated

// FeaturePlaceholdersWithRelated extends FeaturePlaceholders with relationship data.
var FeaturePlaceholdersWithRelated = cfgtemplate.FeaturePlaceholdersWithRelated

// EpicPlaceholdersWithRelated extends EpicPlaceholders with relationship data.
var EpicPlaceholdersWithRelated = cfgtemplate.EpicPlaceholdersWithRelated

// ApplyEnrichmentData merges enrichment data into the placeholder map.
var ApplyEnrichmentData = cfgtemplate.ApplyEnrichmentData

// ParseEpicKeyFromEntityKey extracts the epic key (E##) from a task or feature key.
var ParseEpicKeyFromEntityKey = cfgtemplate.ParseEpicKeyFromEntityKey

// ParseFeatureKeyFromTaskKey extracts the feature key (E##-F##) from a task key.
var ParseFeatureKeyFromTaskKey = cfgtemplate.ParseFeatureKeyFromTaskKey
