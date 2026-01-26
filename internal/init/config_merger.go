package init

// ConfigMerger handles intelligent configuration merging with smart logic
type ConfigMerger struct {
	// No state needed - stateless service
}

// NewConfigMerger creates a new config merger instance
func NewConfigMerger() *ConfigMerger {
	return &ConfigMerger{}
}

// ConfigMergeOptions controls merge behavior
type ConfigMergeOptions struct {
	PreserveFields  []string // Fields to never overwrite (unless force=true)
	OverwriteFields []string // Fields to replace entirely
	Force           bool     // If true, overwrite even protected fields
}

// Merge performs intelligent merge of overlay into base
// Returns merged config and change report
func (m *ConfigMerger) Merge(
	base, overlay map[string]interface{},
	opts ConfigMergeOptions,
) (map[string]interface{}, *ChangeReport, error) {
	// 1. Deep copy base to avoid mutations
	merged := deepCopy(base)

	// 2. Track changes
	report := &ChangeReport{
		Added:       []string{},
		Preserved:   []string{},
		Overwritten: []string{},
		Stats:       &ChangeStats{},
	}

	// 3. Process each field in overlay
	for key, value := range overlay {
		// Skip if in preserve list (unless force mode)
		if contains(opts.PreserveFields, key) && !opts.Force {
			report.Preserved = append(report.Preserved, key)
			continue
		}

		// Check if field exists in base
		if _, exists := merged[key]; exists {
			// Field exists - check if we should overwrite
			if contains(opts.OverwriteFields, key) || opts.Force {
				merged[key] = value
				report.Overwritten = append(report.Overwritten, key)
			} else {
				// Merge nested structures
				merged[key] = m.mergeValue(merged[key], value)
				report.Added = append(report.Added, key)
			}
		} else {
			// New field - add it
			merged[key] = value
			report.Added = append(report.Added, key)
		}
	}

	// 4. Calculate statistics
	report.Stats = m.calculateStats(base, merged, overlay)

	return merged, report, nil
}

// mergeValue handles merging of nested values
func (m *ConfigMerger) mergeValue(base, overlay interface{}) interface{} {
	// Handle maps recursively
	baseMap, baseIsMap := base.(map[string]interface{})
	overlayMap, overlayIsMap := overlay.(map[string]interface{})

	if baseIsMap && overlayIsMap {
		return m.DeepMerge(baseMap, overlayMap)
	}

	// For non-maps, overlay wins
	return overlay
}

// DeepMerge recursively merges two maps
func (m *ConfigMerger) DeepMerge(base, overlay map[string]interface{}) map[string]interface{} {
	result := deepCopy(base)

	for key, value := range overlay {
		if baseValue, exists := result[key]; exists {
			result[key] = m.mergeValue(baseValue, value)
		} else {
			result[key] = value
		}
	}

	return result
}

// DetectChanges compares two configs and reports differences
func (m *ConfigMerger) DetectChanges(old, new map[string]interface{}) *ChangeReport {
	report := &ChangeReport{
		Added:       []string{},
		Preserved:   []string{},
		Overwritten: []string{},
		Stats:       &ChangeStats{},
	}

	// Find added and modified fields
	for key, newValue := range new {
		if oldValue, exists := old[key]; exists {
			// Check if value changed
			if !deepEqual(oldValue, newValue) {
				report.Overwritten = append(report.Overwritten, key)
			} else {
				report.Preserved = append(report.Preserved, key)
			}
		} else {
			report.Added = append(report.Added, key)
		}
	}

	report.Stats = m.calculateStats(old, new, new)
	return report
}

// calculateStats computes detailed change statistics
func (m *ConfigMerger) calculateStats(base, merged, overlay map[string]interface{}) *ChangeStats {
	stats := &ChangeStats{}

	// Count statuses added
	if statusMeta, ok := overlay["status_metadata"].(map[string]interface{}); ok {
		stats.StatusesAdded = len(statusMeta)
	}

	// Count flows added
	if statusFlow, ok := overlay["status_flow"].(map[string]interface{}); ok {
		stats.FlowsAdded = len(statusFlow)
	}

	// Count special status groups
	if specialStatuses, ok := overlay["special_statuses"].(map[string]interface{}); ok {
		stats.GroupsAdded = len(specialStatuses)
	}

	// Count preserved fields
	for key := range base {
		if _, exists := overlay[key]; !exists {
			stats.FieldsPreserved++
		}
	}

	return stats
}

// deepCopy creates a deep copy of a map
func deepCopy(src map[string]interface{}) map[string]interface{} {
	dst := make(map[string]interface{})

	for key, value := range src {
		switch v := value.(type) {
		case map[string]interface{}:
			dst[key] = deepCopy(v)
		case []interface{}:
			dst[key] = deepCopySlice(v)
		default:
			dst[key] = v
		}
	}

	return dst
}

// deepCopySlice creates a deep copy of a slice
func deepCopySlice(src []interface{}) []interface{} {
	dst := make([]interface{}, len(src))

	for i, value := range src {
		switch v := value.(type) {
		case map[string]interface{}:
			dst[i] = deepCopy(v)
		case []interface{}:
			dst[i] = deepCopySlice(v)
		default:
			dst[i] = v
		}
	}

	return dst
}

// contains checks if a string is in a slice
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// deepEqual checks if two values are deeply equal
func deepEqual(a, b interface{}) bool {
	aMap, aIsMap := a.(map[string]interface{})
	bMap, bIsMap := b.(map[string]interface{})

	if aIsMap && bIsMap {
		if len(aMap) != len(bMap) {
			return false
		}
		for key, aValue := range aMap {
			bValue, exists := bMap[key]
			if !exists || !deepEqual(aValue, bValue) {
				return false
			}
		}
		return true
	}

	// For non-maps, use simple equality
	return a == b
}
