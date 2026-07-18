// Package research loads and structurally validates the research artifacts
// required before workflow-managed work can advance beyond research.
package research

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/jwwelbor/shark-task-manager/internal/config"
	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/sharkdata"
	"gopkg.in/yaml.v3"
)

const catalogPath = "research/recipes.yaml"

// Catalog is the declarative recipe catalog shipped in shark-data.
type Catalog struct {
	Version string            `yaml:"version"`
	Recipes map[string]Recipe `yaml:"recipes"`
}

// Recipe defines the allowed selections and structural document contract.
type Recipe struct {
	EntityTypes            []string         `yaml:"entity_types"`
	RequiredPlanSections   []string         `yaml:"required_plan_sections"`
	RequiredReportSections []string         `yaml:"required_report_sections"`
	Categories             []string         `yaml:"categories"`
	Rigor                  map[string]Rigor `yaml:"rigor"`
}

// Rigor describes the depth expected for a tier.
type Rigor struct {
	Description string `yaml:"description"`
}

// Paths identifies the two artifacts belonging to an entity.
type Paths struct {
	Plan   string
	Report string
}

// LoadCatalog loads a project materialization when present and otherwise uses
// the embedded catalog. The embedded fallback keeps default workflows usable
// before shark-data is installed.
func LoadCatalog(projectRoot string) (*Catalog, error) {
	data, found, err := readCatalogFromDisk(projectRoot)
	if err != nil {
		return nil, err
	}
	if !found {
		data, err = sharkdata.ReadEmbedded(catalogPath)
		if err != nil {
			return nil, fmt.Errorf("read embedded research recipe catalog: %w", err)
		}
	}

	var catalog Catalog
	if err := yaml.Unmarshal(data, &catalog); err != nil {
		return nil, fmt.Errorf("parse research recipe catalog: %w", err)
	}
	if catalog.Version == "" || len(catalog.Recipes) == 0 {
		return nil, fmt.Errorf("research recipe catalog is empty or missing version")
	}
	return &catalog, nil
}

func readCatalogFromDisk(projectRoot string) ([]byte, bool, error) {
	configBytes, err := os.ReadFile(filepath.Join(projectRoot, ".sharkconfig.json"))
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, false, fmt.Errorf("read .sharkconfig.json: %w", err)
	}
	dataRoot, err := config.ResolveSharkDataRoot(projectRoot, configBytes)
	if err != nil {
		return nil, false, fmt.Errorf("resolve shark data root: %w", err)
	}
	for _, path := range []string{
		filepath.Join(dataRoot, "overrides", catalogPath),
		filepath.Join(dataRoot, catalogPath),
	} {
		data, err := os.ReadFile(path)
		if err == nil {
			return data, true, nil
		}
		if !errors.Is(err, os.ErrNotExist) {
			return nil, false, fmt.Errorf("read research recipe catalog: %w", err)
		}
	}
	return nil, false, nil
}

// ArtifactPaths returns co-located research paths. Epic and feature reports
// retain their directory-local names; file-backed entities use key sidecars.
func ArtifactPaths(projectRoot string, entity models.Entity) (Paths, error) {
	filePath := strings.TrimSpace(entity.GetFilePath())
	if filePath == "" {
		return Paths{}, fmt.Errorf("%s %s has no file path for research artifacts", entity.GetEntityType(), entity.GetKey())
	}
	dir := filepath.Dir(filepath.Join(projectRoot, filePath))
	if entity.GetEntityType() == models.EntityTypeEpic || entity.GetEntityType() == models.EntityTypeFeature {
		return Paths{Plan: filepath.Join(dir, "research-plan.md"), Report: filepath.Join(dir, "research-report.md")}, nil
	}
	base := entity.GetKey()
	return Paths{
		Plan:   filepath.Join(dir, base+".research-plan.md"),
		Report: filepath.Join(dir, base+".research-report.md"),
	}, nil
}
