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

// CatalogResourcePath is the supported project resource path for the recipe catalog.
const CatalogResourcePath = "research/recipes.yaml"

// CatalogSource identifies the layer that supplied the research recipe catalog.
type CatalogSource string

const (
	CatalogSourceOverride CatalogSource = "override"
	CatalogSourceDisk     CatalogSource = "disk"
	CatalogSourceEmbedded CatalogSource = "embedded"
)

// Catalog is the declarative recipe catalog shipped in shark-data.
type Catalog struct {
	Version string            `yaml:"version"`
	Recipes map[string]Recipe `yaml:"recipes"`
}

// Recipe defines the allowed selections and structural document contract.
type Recipe struct {
	EntityTypes            []string          `yaml:"entity_types"`
	RequiredPlanSections   []string          `yaml:"required_plan_sections"` // v1 compatibility
	RequiredReportSections []string          `yaml:"required_report_sections"`
	Categories             []string          `yaml:"categories"`
	Modules                map[string]Module `yaml:"modules"`
	Rigor                  map[string]Rigor  `yaml:"rigor"`
}

// Module is an atomic research activity. Categories select applicable modules;
// they do not impose prose requirements of their own.
type Module struct {
	EntityTypes     []string `yaml:"entity_types"`
	Categories      []string `yaml:"categories"`
	MinimumEvidence string   `yaml:"minimum_evidence"`
	ExpectedOutput  string   `yaml:"expected_output"`
}

// Rigor remains available for installed v1 catalogs. V2 derives the required
// module coverage from the selected tier rather than a prose description.
type Rigor struct {
	Description string `yaml:"description"`
}

// Paths identifies the one current research artifact belonging to an entity.
type Paths struct {
	Report string
}

// ReadCatalog returns the raw catalog content and its source. The embedded
// fallback keeps default workflows usable before shark-data is installed.
func ReadCatalog(projectRoot string) ([]byte, CatalogSource, error) {
	data, source, found, err := readCatalogFromDisk(projectRoot)
	if err != nil {
		return nil, "", err
	}
	if found {
		return data, source, nil
	}
	data, err = sharkdata.ReadEmbedded(CatalogResourcePath)
	if err != nil {
		return nil, "", fmt.Errorf("read embedded research recipe catalog: %w", err)
	}
	return data, CatalogSourceEmbedded, nil
}

// LoadCatalog loads a project materialization when present and otherwise uses
// the embedded catalog.
func LoadCatalog(projectRoot string) (*Catalog, error) {
	data, _, err := ReadCatalog(projectRoot)
	if err != nil {
		return nil, err
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

func readCatalogFromDisk(projectRoot string) ([]byte, CatalogSource, bool, error) {
	configBytes, err := os.ReadFile(filepath.Join(projectRoot, ".sharkconfig.json"))
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, "", false, fmt.Errorf("read .sharkconfig.json: %w", err)
	}
	dataRoot, err := config.ResolveSharkDataRoot(projectRoot, configBytes)
	if err != nil {
		return nil, "", false, fmt.Errorf("resolve shark data root: %w", err)
	}
	for _, candidate := range []struct {
		path   string
		source CatalogSource
	}{
		{path: filepath.Join(dataRoot, "overrides", CatalogResourcePath), source: CatalogSourceOverride},
		{path: filepath.Join(dataRoot, CatalogResourcePath), source: CatalogSourceDisk},
	} {
		data, err := os.ReadFile(candidate.path)
		if err == nil {
			return data, candidate.source, true, nil
		}
		if !errors.Is(err, os.ErrNotExist) {
			return nil, "", false, fmt.Errorf("read research recipe catalog: %w", err)
		}
	}
	return nil, "", false, nil
}

// ArtifactPaths returns the co-located report path. Epic and feature reports
// retain their directory-local names; file-backed entities use key sidecars.
func ArtifactPaths(projectRoot string, entity models.Entity) (Paths, error) {
	filePath := strings.TrimSpace(entity.GetFilePath())
	if filePath == "" {
		return Paths{}, fmt.Errorf("%s %s has no file path for research artifacts", entity.GetEntityType(), entity.GetKey())
	}
	dir := filepath.Dir(filepath.Join(projectRoot, filePath))
	if entity.GetEntityType() == models.EntityTypeEpic || entity.GetEntityType() == models.EntityTypeFeature {
		return Paths{Report: filepath.Join(dir, "research-report.md")}, nil
	}
	base := entity.GetKey()
	return Paths{Report: filepath.Join(dir, base+".research-report.md")}, nil
}
