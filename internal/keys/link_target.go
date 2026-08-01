package keys

import (
	"fmt"
	"strings"

	"github.com/jwwelbor/shark-task-manager/internal/models"
)

// ClassifyQuestionLinkTarget resolves rawKey to the models.EntityType and
// normalized key of an entity a Question can link to (epic, feature, task,
// bug, change, tech-debt). Question keys are rejected — a Question cannot
// link to another Question through this surface — and any other pattern is
// invalid. Single source of truth for this classification; both the CLI and
// HTTP API Question surfaces call it rather than each re-deriving the switch.
func ClassifyQuestionLinkTarget(rawKey string) (models.EntityType, string, error) {
	upper := strings.ToUpper(rawKey)
	ks := NewKeyService()
	switch {
	case IsEpicKey(upper):
		return models.EntityTypeEpic, ks.Normalize(upper), nil
	case IsFeatureKey(upper):
		return models.EntityTypeFeature, ks.Normalize(upper), nil
	case IsShortTaskKey(upper), IsTaskKey(upper):
		return models.EntityTypeTask, ks.NormalizeTaskKey(upper), nil
	case IsBugKey(upper):
		return models.EntityTypeBug, ks.Normalize(upper), nil
	case IsChangeKey(upper):
		return models.EntityTypeChange, ks.Normalize(upper), nil
	case IsTechDebtKey(upper):
		return models.EntityTypeTechDebt, upper, nil
	case ks.DetectEntityType(upper) == EntityTypeQuestion:
		return "", "", fmt.Errorf("entity_key must not identify a Question")
	default:
		return "", "", fmt.Errorf("entity_key is invalid")
	}
}
