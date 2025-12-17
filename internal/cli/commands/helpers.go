package commands

import (
	"regexp"
	"strings"
	"unicode"
)

// generateSlug creates a URL-friendly slug from a title
func generateSlug(title string) string {
	// Convert to lowercase
	slug := strings.ToLower(title)

	// Replace spaces and underscores with hyphens
	slug = strings.ReplaceAll(slug, " ", "-")
	slug = strings.ReplaceAll(slug, "_", "-")

	// Remove non-alphanumeric characters (except hyphens)
	slug = removeNonAlphanumeric(slug)

	// Remove multiple consecutive hyphens
	re := regexp.MustCompile("-+")
	slug = re.ReplaceAllString(slug, "-")

	// Trim hyphens from start and end
	slug = strings.Trim(slug, "-")

	// Limit length to 50 characters
	if len(slug) > 50 {
		slug = slug[:50]
		slug = strings.Trim(slug, "-")
	}

	return slug
}

// removeNonAlphanumeric removes all non-alphanumeric characters except hyphens
func removeNonAlphanumeric(s string) string {
	var result strings.Builder
	for _, r := range s {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '-' {
			result.WriteRune(r)
		}
	}
	return result.String()
}

// isValidEpicKey validates epic key format (E##)
func isValidEpicKey(key string) bool {
	matched, _ := regexp.MatchString(`^E\d{2}$`, key)
	return matched
}

// isValidFeatureKey validates feature key format (F##)
func isValidFeatureKey(key string) bool {
	matched, _ := regexp.MatchString(`^F\d{2}$`, key)
	return matched
}
