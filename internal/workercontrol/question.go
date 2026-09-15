package workercontrol

import (
	"strconv"
	"strings"
)

type questionCategory string

const (
	questionCategoryProduct      questionCategory = "product"
	questionCategoryRequirements questionCategory = "requirements"
	questionCategoryArchitecture questionCategory = "architecture"
	questionCategoryQuality      questionCategory = "quality"
	questionCategoryProcess      questionCategory = "process"
)

// validateQuestionFields validates the fields unique to a question envelope.
func (e *Envelope) validateQuestionFields() error {
	if !e.hasRequiredQuestionFields() {
		return newValidationError("", ErrorClassShape, "entity_key, category, question, and why_blocking are required when kind is question")
	}
	if err := boundedText("entity_key", e.EntityKey, 1, IdentityMaxBytes); err != nil {
		return err
	}
	if err := boundedText("category", e.Category, 1, IdentityMaxBytes); err != nil {
		return err
	}
	if !validQuestionCategory(e.Category) {
		return newValidationError("category", ErrorClassShape, "must be product, requirements, architecture, quality, or process")
	}
	if err := validateOptionalQuestionText(e); err != nil {
		return err
	}
	for i, option := range e.Options {
		if err := boundedText("options["+strconv.Itoa(i)+"]", option, 1, SummaryMaxBytes); err != nil {
			return err
		}
	}
	return nil
}

func (e *Envelope) hasRequiredQuestionFields() bool {
	return strings.TrimSpace(e.EntityKey) != "" && strings.TrimSpace(e.Category) != "" &&
		strings.TrimSpace(e.Question) != "" && strings.TrimSpace(e.WhyBlocking) != ""
}

func validateOptionalQuestionText(e *Envelope) error {
	for field, value := range map[string]string{"question": e.Question, "why_blocking": e.WhyBlocking, "recommendation": e.Recommendation} {
		if value != "" {
			if err := boundedText(field, value, 1, SummaryMaxBytes); err != nil {
				return err
			}
		}
	}
	return nil
}

func (e *Envelope) validateQuestionVariant() error {
	if e.Kind == KindQuestion {
		return e.validateQuestionFields()
	}
	if e.EntityKey != "" || e.Category != "" || e.Question != "" || e.WhyBlocking != "" || len(e.Options) > 0 || e.Recommendation != "" {
		return newValidationError("", ErrorClassShape, "question fields must be absent unless kind is question")
	}
	return nil
}

func (e *Envelope) validateFinalFields() error {
	if e.Kind == KindFinal {
		if strings.TrimSpace(e.RecommendedOutcome) == "" {
			return newValidationError("recommended_outcome", ErrorClassShape, "is required when kind is final")
		}
		return boundedText("recommended_outcome", e.RecommendedOutcome, 1, IdentityMaxBytes)
	}
	if e.RecommendedOutcome != "" {
		return newValidationError("recommended_outcome", ErrorClassShape, "must be absent unless kind is final")
	}
	if len(e.GateResult) > 0 {
		return newValidationError("gate_result", ErrorClassShape, "must be absent unless kind is final")
	}
	return nil
}

func (e *Envelope) validateEvidence() error {
	if len(e.Evidence) > MaxEvidenceItems {
		return newValidationError("evidence", ErrorClassBounds, "must not exceed the maximum evidence collection size")
	}
	for i, ev := range e.Evidence {
		if err := ev.validate(i); err != nil {
			return err
		}
	}
	return nil
}

func validQuestionCategory(category string) bool {
	switch questionCategory(category) {
	case questionCategoryProduct, questionCategoryRequirements, questionCategoryArchitecture, questionCategoryQuality, questionCategoryProcess:
		return true
	default:
		return false
	}
}
