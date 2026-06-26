package templates

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/jwwelbor/shark-task-manager/internal/pathutil"
)

const sharkDataTemplatesLeaf = "templates"

var getwd = os.Getwd

// LoadEntityTemplate loads an entity markdown skeleton from the Shark 2.0
// shark-data tree. Disk overrides win, then disk defaults, then the embedded
// canonical defaults so create commands work without legacy shark-templates/.
func LoadEntityTemplate(fileName string) ([]byte, error) {
	if err := validateEntityTemplateFileName(fileName); err != nil {
		return nil, err
	}

	relPath := filepath.ToSlash(filepath.Join(sharkDataTemplatesLeaf, fileName))
	resolver := NewIncludeResolverWithEmbed(findSharkDataRoot())
	_, content, err := resolver.resolveContent(relPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load entity template %s: %w", fileName, err)
	}
	return content, nil
}

func validateEntityTemplateFileName(fileName string) error {
	if fileName == "" || fileName == "." || fileName == ".." ||
		strings.ContainsAny(fileName, `/\`) || filepath.Base(fileName) != fileName {
		return fmt.Errorf("entity template path must be a file name: %q", fileName)
	}
	return nil
}

func findSharkDataRoot() string {
	root := configuredSharkDataPath
	if root == "" {
		root = defaultSharkDataDirName
	}
	root = pathutil.ExpandHome(root)
	if filepath.IsAbs(root) {
		return root
	}

	wd, err := getwd()
	if err != nil {
		return ""
	}

	currentDir := wd
	for {
		candidate := filepath.Join(currentDir, root)
		if info, statErr := os.Stat(candidate); statErr == nil && info.IsDir() {
			return candidate
		}

		parentDir := filepath.Dir(currentDir)
		if parentDir == currentDir {
			break
		}
		currentDir = parentDir
	}

	return ""
}
