package models

// ViewerTaskWithRelationships wraps a Task with pre-resolved relationship JSON
// from the viewer_task_relationships database view.
//
// RelationshipsJSON is a JSON array of objects with fields:
// direction, relationship_type, entity_type, entity_key.
// It is populated by ListWithViewerRelationships and
// ListByFeatureWithViewerRelationships in the task repository.
type ViewerTaskWithRelationships struct {
	*Task
	RelationshipsJSON string
}
