package runner

import (
	"bytes"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
	"unicode/utf16"
	"unicode/utf8"
)

// reviewGitPaths decodes the path strings from Git/Python before encoding/json
// can collapse an escaped byte and a genuine U+FFFD into the same filename.
type reviewGitPaths []string

func (p *reviewGitPaths) UnmarshalJSON(data []byte) error {
	var members []json.RawMessage
	if err := json.Unmarshal(data, &members); err != nil {
		return err
	}
	if members == nil {
		return fmt.Errorf("expected an array of Git paths")
	}
	paths := make(reviewGitPaths, 0, len(members))
	for _, member := range members {
		path, err := decodeReviewGitPath(member)
		if err != nil {
			return err
		}
		paths = append(paths, path)
	}
	*p = paths
	return nil
}

type reviewGitPath string

func (p *reviewGitPath) UnmarshalJSON(data []byte) error {
	path, err := decodeReviewGitPath(data)
	if err != nil {
		return err
	}
	*p = reviewGitPath(path)
	return nil
}

// token is one syntactically valid JSON value from the packet decoder.
func decodeReviewGitPath(token json.RawMessage) (string, error) {
	if len(token) < 2 || token[0] != '"' || !utf8.Valid(token) {
		return "", fmt.Errorf("expected a UTF-8 JSON string for a Git path")
	}
	data := token[1 : len(token)-1]
	var name strings.Builder
	for len(data) > 0 {
		index := bytes.IndexByte(data, '\\')
		if index < 0 {
			name.Write(data)
			break
		}
		name.Write(data[:index])
		n, err := appendPathEscape(&name, data[index:])
		if err != nil {
			return "", err
		}
		data = data[index+n:]
	}
	path := name.String()
	if !filepath.IsLocal(path) || path == "." || strings.IndexByte(path, 0) >= 0 {
		return "", fmt.Errorf("invalid repository-relative Git path %q", path)
	}
	return path, nil
}

func appendPathEscape(name *strings.Builder, escape []byte) (int, error) {
	if escape[1] != 'u' {
		// JSON syntax validation has already restricted this to a standard escape.
		const encoded = `"\/bfnrt`
		const decoded = "\"\\/\b\f\n\r\t"
		name.WriteByte(decoded[strings.IndexByte(encoded, escape[1])])
		return 2, nil
	}
	value, err := strconv.ParseUint(string(escape[2:6]), 16, 16)
	if err != nil {
		return 0, err
	}
	unit := rune(value)
	switch {
	case !utf16.IsSurrogate(unit):
		name.WriteRune(unit)
	case unit >= 0xdc80 && unit <= 0xdcff:
		name.WriteByte(byte(unit - 0xdc00))
	default:
		return appendPathSurrogatePair(name, unit, escape[6:])
	}
	return 6, nil
}

func appendPathSurrogatePair(name *strings.Builder, high rune, rest []byte) (int, error) {
	if len(rest) >= 6 && bytes.HasPrefix(rest, []byte(`\u`)) {
		low, err := strconv.ParseUint(string(rest[2:6]), 16, 16)
		scalar := utf16.DecodeRune(high, rune(low))
		if err == nil && scalar != utf8.RuneError {
			name.WriteRune(scalar)
			return 12, nil
		}
	}
	return 0, fmt.Errorf("unpaired Unicode surrogate in Git path")
}
