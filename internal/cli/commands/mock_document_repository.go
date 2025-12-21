package commands

import (
	"context"

	"github.com/jwwelbor/shark-task-manager/internal/models"
)

// DocumentRepositoryInterface defines methods for testing
type DocumentRepositoryInterface interface {
	CreateOrGet(ctx context.Context, title, filePath string) (*models.Document, error)
	GetByID(ctx context.Context, id int64) (*models.Document, error)
	Delete(ctx context.Context, id int64) error
	LinkToEpic(ctx context.Context, epicID, documentID int64) error
	LinkToFeature(ctx context.Context, featureID, documentID int64) error
	LinkToTask(ctx context.Context, taskID, documentID int64) error
	UnlinkFromEpic(ctx context.Context, epicID, documentID int64) error
	UnlinkFromFeature(ctx context.Context, featureID, documentID int64) error
	UnlinkFromTask(ctx context.Context, taskID, documentID int64) error
	ListForEpic(ctx context.Context, epicID int64) ([]*models.Document, error)
	ListForFeature(ctx context.Context, featureID int64) ([]*models.Document, error)
	ListForTask(ctx context.Context, taskID int64) ([]*models.Document, error)
}

// MockDocumentRepository for testing
type MockDocumentRepository struct {
	documents           map[int64]*models.Document
	EpicDocuments       map[int64][]*models.Document
	FeatureDocuments    map[int64][]*models.Document
	TaskDocuments       map[int64][]*models.Document
	nextID              int64
	CreateOrGetCalls    int
	GetByIDCalls        int
	DeleteCalls         int
	LinkToEpicCalls     int
	LinkToFeatureCalls  int
	LinkToTaskCalls     int
	UnlinkFromEpicCalls int
	UnlinkFromFeatureCalls int
	UnlinkFromTaskCalls int
	ListForEpicCalls    int
	ListForFeatureCalls int
	ListForTaskCalls    int
	LastCreateOrGetTitle string
	LastCreateOrGetPath  string
}

// NewMockDocumentRepository creates a new mock
func NewMockDocumentRepository() *MockDocumentRepository {
	return &MockDocumentRepository{
		documents:        make(map[int64]*models.Document),
		EpicDocuments:    make(map[int64][]*models.Document),
		FeatureDocuments: make(map[int64][]*models.Document),
		TaskDocuments:    make(map[int64][]*models.Document),
		nextID:           1,
	}
}

// AddDocument adds a document to the mock
func (m *MockDocumentRepository) AddDocument(doc *models.Document) {
	if doc.ID == 0 {
		doc.ID = m.nextID
		m.nextID++
	}
	m.documents[doc.ID] = doc
}

// CreateOrGet simulates document creation or retrieval
func (m *MockDocumentRepository) CreateOrGet(ctx context.Context, title, filePath string) (*models.Document, error) {
	m.CreateOrGetCalls++
	m.LastCreateOrGetTitle = title
	m.LastCreateOrGetPath = filePath

	// Check if exists
	for _, doc := range m.documents {
		if doc.Title == title && doc.FilePath == filePath {
			return doc, nil
		}
	}

	// Create new
	doc := &models.Document{
		ID:       m.nextID,
		Title:    title,
		FilePath: filePath,
	}
	m.nextID++
	m.documents[doc.ID] = doc
	return doc, nil
}

// GetByID retrieves a document by ID
func (m *MockDocumentRepository) GetByID(ctx context.Context, id int64) (*models.Document, error) {
	m.GetByIDCalls++
	doc, exists := m.documents[id]
	if !exists {
		return nil, &models.NotFoundError{Entity: "document"}
	}
	return doc, nil
}

// Delete removes a document
func (m *MockDocumentRepository) Delete(ctx context.Context, id int64) error {
	m.DeleteCalls++
	delete(m.documents, id)
	return nil
}

// LinkToEpic links document to epic
func (m *MockDocumentRepository) LinkToEpic(ctx context.Context, epicID, documentID int64) error {
	m.LinkToEpicCalls++
	doc, exists := m.documents[documentID]
	if !exists {
		return &models.NotFoundError{Entity: "document"}
	}
	m.EpicDocuments[epicID] = append(m.EpicDocuments[epicID], doc)
	return nil
}

// LinkToFeature links document to feature
func (m *MockDocumentRepository) LinkToFeature(ctx context.Context, featureID, documentID int64) error {
	m.LinkToFeatureCalls++
	doc, exists := m.documents[documentID]
	if !exists {
		return &models.NotFoundError{Entity: "document"}
	}
	m.FeatureDocuments[featureID] = append(m.FeatureDocuments[featureID], doc)
	return nil
}

// LinkToTask links document to task
func (m *MockDocumentRepository) LinkToTask(ctx context.Context, taskID, documentID int64) error {
	m.LinkToTaskCalls++
	doc, exists := m.documents[documentID]
	if !exists {
		return &models.NotFoundError{Entity: "document"}
	}
	m.TaskDocuments[taskID] = append(m.TaskDocuments[taskID], doc)
	return nil
}

// UnlinkFromEpic removes document link from epic
func (m *MockDocumentRepository) UnlinkFromEpic(ctx context.Context, epicID, documentID int64) error {
	m.UnlinkFromEpicCalls++
	if docs, exists := m.EpicDocuments[epicID]; exists {
		for i, doc := range docs {
			if doc.ID == documentID {
				m.EpicDocuments[epicID] = append(docs[:i], docs[i+1:]...)
				break
			}
		}
	}
	return nil
}

// UnlinkFromFeature removes document link from feature
func (m *MockDocumentRepository) UnlinkFromFeature(ctx context.Context, featureID, documentID int64) error {
	m.UnlinkFromFeatureCalls++
	if docs, exists := m.FeatureDocuments[featureID]; exists {
		for i, doc := range docs {
			if doc.ID == documentID {
				m.FeatureDocuments[featureID] = append(docs[:i], docs[i+1:]...)
				break
			}
		}
	}
	return nil
}

// UnlinkFromTask removes document link from task
func (m *MockDocumentRepository) UnlinkFromTask(ctx context.Context, taskID, documentID int64) error {
	m.UnlinkFromTaskCalls++
	if docs, exists := m.TaskDocuments[taskID]; exists {
		for i, doc := range docs {
			if doc.ID == documentID {
				m.TaskDocuments[taskID] = append(docs[:i], docs[i+1:]...)
				break
			}
		}
	}
	return nil
}

// ListForEpic returns documents for epic
func (m *MockDocumentRepository) ListForEpic(ctx context.Context, epicID int64) ([]*models.Document, error) {
	m.ListForEpicCalls++
	return m.EpicDocuments[epicID], nil
}

// ListForFeature returns documents for feature
func (m *MockDocumentRepository) ListForFeature(ctx context.Context, featureID int64) ([]*models.Document, error) {
	m.ListForFeatureCalls++
	return m.FeatureDocuments[featureID], nil
}

// ListForTask returns documents for task
func (m *MockDocumentRepository) ListForTask(ctx context.Context, taskID int64) ([]*models.Document, error) {
	m.ListForTaskCalls++
	return m.TaskDocuments[taskID], nil
}

// EpicRepositoryInterface for testing
type EpicRepositoryInterface interface {
	GetByKey(ctx context.Context, key string) (*models.Epic, error)
	GetByID(ctx context.Context, id int64) (*models.Epic, error)
}

// MockEpicRepository for testing
type MockEpicRepository struct {
	epics map[string]*models.Epic
}

// NewMockEpicRepository creates a new mock
func NewMockEpicRepository() *MockEpicRepository {
	return &MockEpicRepository{
		epics: make(map[string]*models.Epic),
	}
}

// AddEpic adds an epic to the mock
func (m *MockEpicRepository) AddEpic(epic *models.Epic) {
	m.epics[epic.Key] = epic
}

// GetByKey retrieves epic by key
func (m *MockEpicRepository) GetByKey(ctx context.Context, key string) (*models.Epic, error) {
	epic, exists := m.epics[key]
	if !exists {
		return nil, &models.NotFoundError{Entity: "epic"}
	}
	return epic, nil
}

// GetByID retrieves epic by ID
func (m *MockEpicRepository) GetByID(ctx context.Context, id int64) (*models.Epic, error) {
	for _, epic := range m.epics {
		if epic.ID == id {
			return epic, nil
		}
	}
	return nil, &models.NotFoundError{Entity: "epic"}
}

// FeatureRepositoryInterface for testing
type FeatureRepositoryInterface interface {
	GetByKey(ctx context.Context, key string) (*models.Feature, error)
	GetByID(ctx context.Context, id int64) (*models.Feature, error)
}

// MockFeatureRepository for testing
type MockFeatureRepository struct {
	features map[string]*models.Feature
}

// NewMockFeatureRepository creates a new mock
func NewMockFeatureRepository() *MockFeatureRepository {
	return &MockFeatureRepository{
		features: make(map[string]*models.Feature),
	}
}

// AddFeature adds a feature to the mock
func (m *MockFeatureRepository) AddFeature(feature *models.Feature) {
	m.features[feature.Key] = feature
}

// GetByKey retrieves feature by key
func (m *MockFeatureRepository) GetByKey(ctx context.Context, key string) (*models.Feature, error) {
	feature, exists := m.features[key]
	if !exists {
		return nil, &models.NotFoundError{Entity: "feature"}
	}
	return feature, nil
}

// GetByID retrieves feature by ID
func (m *MockFeatureRepository) GetByID(ctx context.Context, id int64) (*models.Feature, error) {
	for _, feature := range m.features {
		if feature.ID == id {
			return feature, nil
		}
	}
	return nil, &models.NotFoundError{Entity: "feature"}
}
