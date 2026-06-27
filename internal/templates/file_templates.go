package templates

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/jwwelbor/shark-task-manager/internal/pathutil"
	"github.com/jwwelbor/shark-task-manager/internal/sharkdata"
)

const sharkDataFileTemplatesLeaf = "file_templates"

// sharkDataFileTemplatesSubdir returns the file-template directory derived
// from the configured shark_data_path bundle root.
func sharkDataFileTemplatesSubdir() string {
	root := configuredSharkDataPath
	if root == "" {
		root = defaultSharkDataDirName
	}
	root = pathutil.ExpandHome(root)
	return filepath.Join(root, sharkDataFileTemplatesLeaf)
}

// GetFileTemplatesDirName returns the configured file-template directory.
func GetFileTemplatesDirName() string {
	return sharkDataFileTemplatesSubdir()
}

// ReadFileTemplate reads a markdown file template from shark-data with the
// standard override order: <shark_data_path>/overrides/file_templates first,
// then <shark_data_path>/file_templates, then embedded defaults.
func ReadFileTemplate(filename string) ([]byte, error) {
	cleanName, err := cleanFileTemplateName(filename)
	if err != nil {
		return nil, err
	}

	dataRoot := findFileTemplatesDataRoot()
	for _, diskPath := range []string{
		filepath.Join(dataRoot, "overrides", sharkDataFileTemplatesLeaf, cleanName),
		filepath.Join(dataRoot, sharkDataFileTemplatesLeaf, cleanName),
	} {
		data, err := os.ReadFile(diskPath)
		if err == nil {
			return data, nil
		}
		if err != nil && !errors.Is(err, fs.ErrNotExist) {
			return nil, fmt.Errorf("read file template %s: %w", diskPath, err)
		}
	}

	embeddedPath := filepath.ToSlash(filepath.Join(sharkDataFileTemplatesLeaf, cleanName))
	data, err := sharkdata.ReadEmbedded(embeddedPath)
	if err == nil {
		return data, nil
	}
	if errors.Is(err, fs.ErrNotExist) {
		return nil, fmt.Errorf("file template not found: %s", cleanName)
	}
	return nil, fmt.Errorf("read embedded file template %s: %w", embeddedPath, err)
}

func cleanFileTemplateName(filename string) (string, error) {
	if filepath.IsAbs(filename) {
		return "", fmt.Errorf("file template path must be relative: %s", filename)
	}
	clean := filepath.Clean(filename)
	if clean == "." || clean == "" || strings.HasPrefix(clean, ".."+string(filepath.Separator)) || clean == ".." {
		return "", fmt.Errorf("file template path must be relative and stay within file_templates: %s", filename)
	}
	return clean, nil
}

func findFileTemplatesDataRoot() string {
	root := configuredSharkDataPath
	if root == "" {
		root = defaultSharkDataDirName
	}
	root = pathutil.ExpandHome(root)
	if filepath.IsAbs(root) {
		return root
	}

	wd, err := os.Getwd()
	if err != nil {
		return root
	}

	currentDir := wd
	for {
		candidate := filepath.Join(currentDir, root)
		if hasFileTemplateRoot(candidate) {
			return candidate
		}
		parentDir := filepath.Dir(currentDir)
		if parentDir == currentDir {
			break
		}
		currentDir = parentDir
	}

	return root
}

func hasFileTemplateRoot(root string) bool {
	for _, dir := range []string{
		filepath.Join(root, "overrides", sharkDataFileTemplatesLeaf),
		filepath.Join(root, sharkDataFileTemplatesLeaf),
	} {
		if hasFileTemplateFiles(dir) {
			return true
		}
	}
	return false
}

func hasFileTemplateFiles(dir string) bool {
	matches, _ := filepath.Glob(filepath.Join(dir, "*.md"))
	return len(matches) > 0
}
