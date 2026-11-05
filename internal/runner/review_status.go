package runner

import (
	"bytes"
	"fmt"
	"path"
	"path/filepath"
	"strings"

	"github.com/jwwelbor/shark-task-manager/internal/projectdocs"
)

// reviewRuntimePrefix names only the parent-designated project's reserved run
// directory. Git's tracked changes under that directory are never filtered.
func reviewRuntimePrefix(project string) (string, error) {
	if project == "" || strings.Contains(project, "\\") || path.IsAbs(project) || path.Clean(project) != project || project == ".." || strings.HasPrefix(project, "../") {
		return "", fmt.Errorf("invalid Shark project location")
	}
	return strings.TrimPrefix(path.Join(project, ".shark", "runs"), "./") + "/", nil
}

// rejectHiddenIndexFlags prevents Git's index hints from concealing tracked
// source edits from status, diff and candidate freshness checks.
func rejectHiddenIndexFlags(data []byte) error {
	return projectdocs.ValidateIndexFlags(data)
}

func filterReviewStatus(data []byte, prefix string, worktrees ...string) ([]byte, error) {
	if len(data) != 0 && data[len(data)-1] != 0 {
		return nil, fmt.Errorf("malformed git status")
	}
	parts := bytes.Split(data, []byte{0})
	var out []byte
	for i := 0; i < len(parts); i++ {
		record, next, err := reviewStatusRecord(parts, i, prefix, worktrees)
		if err != nil {
			return nil, err
		}
		out = append(out, record...)
		i = next
	}
	return out, nil
}

func reviewStatusRecord(parts [][]byte, index int, prefix string, worktrees []string) ([]byte, int, error) {
	record := parts[index]
	if len(record) == 0 {
		return nil, index, nil
	}
	if len(record) < 4 || record[2] != ' ' {
		return nil, index, fmt.Errorf("malformed git status record")
	}
	if string(record[:2]) == "??" && reviewRuntimeUntracked(string(record[3:]), prefix, worktrees) {
		return nil, index, nil
	}
	out := append(append([]byte{}, record...), 0)
	if !bytes.ContainsAny(record[:2], "RC") {
		return out, index, nil
	}
	index++
	if index >= len(parts) || len(parts[index]) == 0 {
		return nil, index, fmt.Errorf("malformed git rename record")
	}
	out = append(out, parts[index]...)
	return append(out, 0), index, nil
}

func reviewRuntimeUntracked(name, prefix string, worktrees []string) bool {
	if strings.HasPrefix(name, prefix) {
		return true
	}
	for _, worktree := range worktrees {
		if strings.TrimSuffix(name, "/") == strings.TrimSuffix(worktree, "/") || strings.HasPrefix(name, worktree) {
			return true
		}
	}
	return false
}

// reviewWorktreePrefixes filters only registered worktrees in this project's
// reserved runtime directory; unrelated untracked content remains visible.
func reviewWorktreePrefixes(root, project string, run bugGitRunner) ([]string, error) {
	root, err := filepath.EvalSymlinks(root)
	if err != nil {
		return nil, err
	}
	out, err := run("worktree", "list", "--porcelain", "-z")
	if err != nil {
		return nil, err
	}
	base := filepath.Join(root, filepath.FromSlash(project), DefaultWorktreeBaseDir)
	var result []string
	for _, field := range bytes.Split(out, []byte{0}) {
		if !bytes.HasPrefix(field, []byte("worktree ")) {
			continue
		}
		worktree := string(bytes.TrimPrefix(field, []byte("worktree ")))
		if filepath.Clean(filepath.Dir(worktree)) != filepath.Clean(base) {
			continue
		}
		rel, err := filepath.Rel(root, worktree)
		if err != nil || !filepath.IsLocal(rel) {
			return nil, fmt.Errorf("invalid registered worktree path")
		}
		result = append(result, filepath.ToSlash(rel)+"/")
	}
	return result, nil
}

func selectionOverlapsRuntime(selected, prefix string) bool {
	selected = strings.TrimRight(selected, "/")
	reserved := strings.TrimRight(prefix, "/")
	return selected == reserved || strings.HasPrefix(selected, prefix) || strings.HasPrefix(reserved, selected+"/")
}
