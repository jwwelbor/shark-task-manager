package services

import (
	"errors"
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/models"
)

// stubSizeCfg is a minimal SizeEnforcementConfig for tests.
type stubSizeCfg struct{ types []string }

func (s stubSizeCfg) SizeRequiredFor() []string { return s.types }

func TestEnforceSizeRequired_NoConfig_NoError(t *testing.T) {
	if err := enforceSizeRequired(nil, models.EntityTypeTask, nil); err != nil {
		t.Fatalf("nil cfg should not error, got %v", err)
	}
}

func TestEnforceSizeRequired_EmptyCfg_NoError(t *testing.T) {
	cfg := stubSizeCfg{types: nil}
	if err := enforceSizeRequired(cfg, models.EntityTypeTask, nil); err != nil {
		t.Fatalf("empty cfg should not error, got %v", err)
	}
}

func TestEnforceSizeRequired_TypeNotListed_NoError(t *testing.T) {
	cfg := stubSizeCfg{types: []string{"feature"}}
	if err := enforceSizeRequired(cfg, models.EntityTypeTask, nil); err != nil {
		t.Fatalf("type not in list should not error, got %v", err)
	}
}

func TestEnforceSizeRequired_TypeListed_SizeNil_Errors(t *testing.T) {
	cfg := stubSizeCfg{types: []string{"task"}}
	err := enforceSizeRequired(cfg, models.EntityTypeTask, nil)
	if err == nil {
		t.Fatal("expected SizeRequiredError, got nil")
	}
	var sizeErr *SizeRequiredError
	if !errors.As(err, &sizeErr) {
		t.Fatalf("expected *SizeRequiredError, got %T", err)
	}
	if sizeErr.EntityType != "task" {
		t.Errorf("EntityType = %q, want %q", sizeErr.EntityType, "task")
	}
}

func TestEnforceSizeRequired_TypeListed_SizeProvided_NoError(t *testing.T) {
	cfg := stubSizeCfg{types: []string{"task"}}
	size := 3
	if err := enforceSizeRequired(cfg, models.EntityTypeTask, &size); err != nil {
		t.Fatalf("size provided should not error, got %v", err)
	}
}

func TestEnforceSizeRequired_MisCasedTypeSilentlyDisables(t *testing.T) {
	// Mirrors the documented mis-case behavior of tag enforcement.
	cfg := stubSizeCfg{types: []string{"Task"}} // capital T — won't match
	if err := enforceSizeRequired(cfg, models.EntityTypeTask, nil); err != nil {
		t.Fatalf("mis-cased entry should disable enforcement, got %v", err)
	}
}

func TestEmptySizeEnforcementConfig_ReturnsNil(t *testing.T) {
	var cfg SizeEnforcementConfig = EmptySizeEnforcementConfig{}
	if got := cfg.SizeRequiredFor(); got != nil {
		t.Errorf("EmptySizeEnforcementConfig.SizeRequiredFor() = %v, want nil", got)
	}
}
