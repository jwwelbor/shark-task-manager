package runner

import (
	"bytes"
	"fmt"
	"path"
	"strings"
)

// reviewRuntimePrefix names only the parent-designated project's reserved run
// directory. Git's tracked changes under that directory are never filtered.
func reviewRuntimePrefix(project string) (string, error) {
	if project == "" || strings.Contains(project, "\\") || path.IsAbs(project) || path.Clean(project) != project || project == ".." || strings.HasPrefix(project, "../") {
		return "", fmt.Errorf("invalid Shark project location")
	}
	return strings.TrimPrefix(path.Join(project, ".shark", "runs"), "./") + "/", nil
}

func filterReviewStatus(data []byte, prefix string) ([]byte, error) {
	if len(data) != 0 && data[len(data)-1] != 0 {
		return nil, fmt.Errorf("malformed git status")
	}
	parts := bytes.Split(data, []byte{0})
	var out []byte
	for i := 0; i < len(parts); i++ {
		record, next, err := reviewStatusRecord(parts, i, prefix)
		if err != nil {
			return nil, err
		}
		out = append(out, record...)
		i = next
	}
	return out, nil
}

func reviewStatusRecord(parts [][]byte, index int, prefix string) ([]byte, int, error) {
	record := parts[index]
	if len(record) == 0 {
		return nil, index, nil
	}
	if len(record) < 4 || record[2] != ' ' {
		return nil, index, fmt.Errorf("malformed git status record")
	}
	if string(record[:2]) == "??" && strings.HasPrefix(string(record[3:]), prefix) {
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

func selectionOverlapsRuntime(selected, prefix string) bool {
	selected = strings.TrimRight(selected, "/")
	reserved := strings.TrimRight(prefix, "/")
	return selected == reserved || strings.HasPrefix(selected, prefix) || strings.HasPrefix(reserved, selected+"/")
}
