package services

import (
	"context"
	"fmt"
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/workflow"
)

func TestEpicService_GetEpic_Tracing(t *testing.T) {
	tracer, exporter := newTestTracer("test")
	repo := &mockEpicRepo{
		getByKeyFn: func(ctx context.Context, key string) (*models.Epic, error) {
			return &models.Epic{
				BaseEntity: models.BaseEntity{ID: 1, Key: key, Title: "Test Epic"},
				Status:     models.EpicStatusDraft,
			}, nil
		},
	}
	entitySvc := NewEntityService(workflow.NewService(""))
	svc := NewEpicService(repo, entitySvc, epicRepoAsEntityRepo(repo), nil, nil)
	svc.SetTracer(tracer)

	epic, err := svc.GetEpic(context.Background(), "E07")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if epic.Key != "E07" {
		t.Errorf("expected key E07, got %s", epic.Key)
	}

	spans := exporter.GetSpans()
	span := findSpanByName(spans, "EpicService.GetEpic")
	if span == nil {
		t.Fatal("expected span EpicService.GetEpic not found")
	}
	if !spanHasAttribute(span, "epic.key", "E07") {
		t.Error("expected span to have attribute epic.key=E07")
	}
}

func TestEpicService_GetEpic_Tracing_Error(t *testing.T) {
	tracer, exporter := newTestTracer("test")
	repo := &mockEpicRepo{
		getByKeyFn: func(ctx context.Context, key string) (*models.Epic, error) {
			return nil, fmt.Errorf("not found")
		},
	}
	entitySvc := NewEntityService(workflow.NewService(""))
	svc := NewEpicService(repo, entitySvc, epicRepoAsEntityRepo(repo), nil, nil)
	svc.SetTracer(tracer)

	_, err := svc.GetEpic(context.Background(), "E99")
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	spans := exporter.GetSpans()
	span := findSpanByName(spans, "EpicService.GetEpic")
	if span == nil {
		t.Fatal("expected span EpicService.GetEpic not found")
	}
	// codes.Error = 1 in Go OTel SDK
	if span.Status.Code != 1 {
		t.Errorf("expected span status Error (1), got %d", span.Status.Code)
	}
}

func TestEpicService_ListEpics_Tracing(t *testing.T) {
	tracer, exporter := newTestTracer("test")
	repo := &mockEpicRepo{
		listFn: func(ctx context.Context, status *models.EpicStatus) ([]*models.Epic, error) {
			return []*models.Epic{
				{BaseEntity: models.BaseEntity{Key: "E07"}, Status: models.EpicStatusDraft},
			}, nil
		},
	}
	entitySvc := NewEntityService(workflow.NewService(""))
	svc := NewEpicService(repo, entitySvc, epicRepoAsEntityRepo(repo), nil, nil)
	svc.SetTracer(tracer)

	epics, err := svc.ListEpics(context.Background(), EpicFilters{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(epics) != 1 {
		t.Fatalf("expected 1 epic, got %d", len(epics))
	}

	spans := exporter.GetSpans()
	span := findSpanByName(spans, "EpicService.ListEpics")
	if span == nil {
		t.Fatal("expected span EpicService.ListEpics not found")
	}
}

func TestEpicService_NilTracer(t *testing.T) {
	repo := &mockEpicRepo{
		getByKeyFn: func(ctx context.Context, key string) (*models.Epic, error) {
			return &models.Epic{
				BaseEntity: models.BaseEntity{ID: 1, Key: key, Title: "Test"},
				Status:     models.EpicStatusDraft,
			}, nil
		},
	}
	entitySvc := NewEntityService(workflow.NewService(""))
	svc := NewEpicService(repo, entitySvc, epicRepoAsEntityRepo(repo), nil, nil)
	// Do NOT call SetTracer

	epic, err := svc.GetEpic(context.Background(), "E07")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if epic == nil {
		t.Fatal("expected epic, got nil")
	}
}
