package services

import (
	"context"
	"fmt"
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/workflow"
)

func TestFeatureService_GetFeature_Tracing(t *testing.T) {
	tracer, exporter := newTestTracer("test")
	repo := &mockFeatureRepo{
		getByKeyFn: func(ctx context.Context, key string) (*models.Feature, error) {
			return &models.Feature{
				BaseEntity: models.BaseEntity{ID: 1, Key: key, Title: "Test Feature"},
				Status:     models.FeatureStatusDraft,
			}, nil
		},
	}
	entitySvc := NewEntityService(workflow.NewService(""))
	svc := NewFeatureService(repo, entitySvc, featureRepoAsEntityRepo(repo), nil, nil)
	svc.SetTracer(tracer)

	feature, err := svc.GetFeature(context.Background(), "E07-F01")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if feature.Key != "E07-F01" {
		t.Errorf("expected key E07-F01, got %s", feature.Key)
	}

	spans := exporter.GetSpans()
	span := findSpanByName(spans, "FeatureService.GetFeature")
	if span == nil {
		t.Fatal("expected span FeatureService.GetFeature not found")
	}
	if !spanHasAttribute(span, "feature.key", "E07-F01") {
		t.Error("expected span to have attribute feature.key=E07-F01")
	}
}

func TestFeatureService_GetFeature_Tracing_Error(t *testing.T) {
	tracer, exporter := newTestTracer("test")
	repo := &mockFeatureRepo{
		getByKeyFn: func(ctx context.Context, key string) (*models.Feature, error) {
			return nil, fmt.Errorf("not found")
		},
	}
	entitySvc := NewEntityService(workflow.NewService(""))
	svc := NewFeatureService(repo, entitySvc, featureRepoAsEntityRepo(repo), nil, nil)
	svc.SetTracer(tracer)

	_, err := svc.GetFeature(context.Background(), "E07-F99")
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	spans := exporter.GetSpans()
	span := findSpanByName(spans, "FeatureService.GetFeature")
	if span == nil {
		t.Fatal("expected span FeatureService.GetFeature not found")
	}
	// codes.Error = 1 in Go OTel SDK
	if span.Status.Code != 1 {
		t.Errorf("expected span status Error (1), got %d", span.Status.Code)
	}
}

func TestFeatureService_ListFeatures_Tracing(t *testing.T) {
	tracer, exporter := newTestTracer("test")
	repo := &mockFeatureRepo{
		listFn: func(ctx context.Context) ([]*models.Feature, error) {
			return []*models.Feature{
				{BaseEntity: models.BaseEntity{Key: "E07-F01"}, Status: models.FeatureStatusDraft},
			}, nil
		},
	}
	entitySvc := NewEntityService(workflow.NewService(""))
	svc := NewFeatureService(repo, entitySvc, featureRepoAsEntityRepo(repo), nil, nil)
	svc.SetTracer(tracer)

	features, err := svc.ListFeatures(context.Background(), FeatureFilters{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(features) != 1 {
		t.Fatalf("expected 1 feature, got %d", len(features))
	}

	spans := exporter.GetSpans()
	span := findSpanByName(spans, "FeatureService.ListFeatures")
	if span == nil {
		t.Fatal("expected span FeatureService.ListFeatures not found")
	}
}

func TestFeatureService_CompleteFeature_Tracing(t *testing.T) {
	tracer, exporter := newTestTracer("test")
	repo := &mockFeatureRepo{
		getByKeyFn: func(ctx context.Context, key string) (*models.Feature, error) {
			return &models.Feature{
				BaseEntity: models.BaseEntity{ID: 1, Key: key, Title: "Test Feature"},
				Status:     models.FeatureStatusDraft,
			}, nil
		},
		updateFn: func(ctx context.Context, feature *models.Feature) error {
			return nil
		},
	}
	entitySvc := NewEntityService(workflow.NewService(""))
	svc := NewFeatureService(repo, entitySvc, featureRepoAsEntityRepo(repo), nil, nil)
	svc.SetTracer(tracer)

	// CompleteFeature with no tasks should succeed
	_, err := svc.CompleteFeature(context.Background(), "E07-F01", false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	spans := exporter.GetSpans()
	span := findSpanByName(spans, "FeatureService.CompleteFeature")
	if span == nil {
		t.Fatal("expected span FeatureService.CompleteFeature not found")
	}
	if !spanHasAttribute(span, "feature.key", "E07-F01") {
		t.Error("expected span to have attribute feature.key=E07-F01")
	}
}

func TestFeatureService_NilTracer(t *testing.T) {
	repo := &mockFeatureRepo{
		getByKeyFn: func(ctx context.Context, key string) (*models.Feature, error) {
			return &models.Feature{
				BaseEntity: models.BaseEntity{ID: 1, Key: key, Title: "Test"},
				Status:     models.FeatureStatusDraft,
			}, nil
		},
	}
	entitySvc := NewEntityService(workflow.NewService(""))
	svc := NewFeatureService(repo, entitySvc, featureRepoAsEntityRepo(repo), nil, nil)
	// Do NOT call SetTracer

	feature, err := svc.GetFeature(context.Background(), "E07-F01")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if feature == nil {
		t.Fatal("expected feature, got nil")
	}
}
