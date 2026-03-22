package services

import (
	"context"
	"fmt"

	"github.com/jwwelbor/shark-task-manager/internal/models"
)

// epicRepoAsEntityRepo wraps a mockEpicRepo as an EntityRepository for tests
// that exercise TransitionStatus / GetNextStatus.
func epicRepoAsEntityRepo(repo *mockEpicRepo) EntityRepository {
	return &mockEntityRepo{
		getByKeyFn: func(ctx context.Context, key string) (models.Entity, error) {
			e, err := repo.GetByKey(ctx, key)
			if err != nil {
				return nil, err
			}
			if e == nil {
				return nil, nil
			}
			return e, nil
		},
		getByIDFn: func(ctx context.Context, id int64) (models.Entity, error) {
			e, err := repo.GetByID(ctx, id)
			if err != nil {
				return nil, err
			}
			if e == nil {
				return nil, nil
			}
			return e, nil
		},
		updateStatusFn: func(ctx context.Context, id int64, status string) error {
			return repo.UpdateStatus(ctx, id, models.EpicStatus(status))
		},
		updateFn: func(ctx context.Context, entity models.Entity) error {
			epic, ok := entity.(*models.Epic)
			if !ok {
				return fmt.Errorf("epicRepoAsEntityRepo: expected *models.Epic, got %T", entity)
			}
			return repo.Update(ctx, epic)
		},
	}
}

// featureRepoAsEntityRepo wraps a mockFeatureRepo as an EntityRepository for tests
// that exercise TransitionStatus / GetNextStatus.
func featureRepoAsEntityRepo(repo *mockFeatureRepo) EntityRepository {
	return &mockEntityRepo{
		getByKeyFn: func(ctx context.Context, key string) (models.Entity, error) {
			f, err := repo.GetByKey(ctx, key)
			if err != nil {
				return nil, err
			}
			if f == nil {
				return nil, nil
			}
			return f, nil
		},
		getByIDFn: func(ctx context.Context, id int64) (models.Entity, error) {
			f, err := repo.GetByID(ctx, id)
			if err != nil {
				return nil, err
			}
			if f == nil {
				return nil, nil
			}
			return f, nil
		},
		updateStatusFn: func(ctx context.Context, id int64, status string) error {
			f, err := repo.GetByID(ctx, id)
			if err != nil {
				return err
			}
			f.Status = models.FeatureStatus(status)
			return repo.Update(ctx, f)
		},
		updateFn: func(ctx context.Context, entity models.Entity) error {
			feature, ok := entity.(*models.Feature)
			if !ok {
				return fmt.Errorf("featureRepoAsEntityRepo: expected *models.Feature, got %T", entity)
			}
			return repo.Update(ctx, feature)
		},
	}
}
