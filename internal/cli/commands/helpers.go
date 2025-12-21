package commands

import (
	"regexp"
)

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
