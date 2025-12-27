package taskfile

import (
	"regexp"
	"strings"

	"github.com/jwwelbor/shark-task-manager/internal/models"
)

var (
	// Match checkbox lines in markdown:
	// - [ ] unchecked criterion (pending)
	// - [x] checked criterion (complete)
	// - [X] checked criterion (complete, uppercase variant)
	checkboxPattern = regexp.MustCompile(`^\s*-\s+\[([ xX])\]\s+(.+)$`)
)

// CriterionItem represents a single acceptance criterion parsed from markdown
type CriterionItem struct {
	Criterion string
	Status    models.CriteriaStatus
}

// ParseCriteria extracts acceptance criteria from task markdown content.
// It finds checkbox items (- [ ] or - [x]) and converts them to criterion items.
// Unchecked boxes become "pending", checked boxes become "complete".
func ParseCriteria(content string) []CriterionItem {
	criteria := make([]CriterionItem, 0)
	lines := strings.Split(content, "\n")

	for _, line := range lines {
		if match := checkboxPattern.FindStringSubmatch(line); match != nil {
			checkbox := match[1]
			text := strings.TrimSpace(match[2])

			if text == "" {
				continue // Skip empty criteria
			}

			var status models.CriteriaStatus
			if checkbox == " " {
				status = models.CriteriaStatusPending
			} else {
				status = models.CriteriaStatusComplete
			}

			criteria = append(criteria, CriterionItem{
				Criterion: text,
				Status:    status,
			})
		}
	}

	return criteria
}

// ParseCriteriaFromFile reads a task file and extracts acceptance criteria
func ParseCriteriaFromFile(filePath string) ([]CriterionItem, error) {
	taskFile, err := ParseTaskFile(filePath)
	if err != nil {
		return nil, err
	}

	return ParseCriteria(taskFile.Content), nil
}
