package testcapture

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewDrainingPipeCapturesOutputBeyondPipeBuffer(t *testing.T) {
	reader, writer, err := NewDrainingPipe()
	require.NoError(t, err)

	originalStdout := os.Stdout
	os.Stdout = writer
	t.Cleanup(func() {
		os.Stdout = originalStdout
		_ = writer.Close()
		_ = reader.Close()
	})

	// Two mebibytes exceeds the pipe buffers used by the supported platforms.
	// The concurrent drain must let this synchronous write finish before the
	// test attempts to read from the returned reader.
	want := strings.Repeat("capture", (2<<20)/len("capture"))
	_, writeErr := io.WriteString(os.Stdout, want)
	require.NoError(t, writeErr)
	require.NoError(t, writer.Close())
	os.Stdout = originalStdout

	got, err := io.ReadAll(reader)
	require.NoError(t, err)
	require.Equal(t, want, string(got))
}

func TestStdioPipeCapturesUseDrainingHelper(t *testing.T) {
	root := repositoryRoot(t)
	streamingCaptures := map[string]string{
		"internal/runner/liveness_test.go:captureStderrLines":                             "the liveness helper parses stderr while the child is still running",
		"tests/contracts/e40_i03_liveness_contract_test.go:TestTC001_I03LivenessContract": "the contract checks liveness output before the child exits",
	}
	seenStreamingCaptures := make(map[string]bool, len(streamingCaptures))

	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			switch entry.Name() {
			case ".git", ".worktrees", ".claude", "vendor", "node_modules":
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, "_test.go") {
			return nil
		}

		file, parseErr := parser.ParseFile(token.NewFileSet(), path, nil, 0)
		if parseErr != nil {
			return parseErr
		}
		relative, relErr := filepath.Rel(root, path)
		if relErr != nil {
			return relErr
		}
		relative = filepath.ToSlash(relative)
		for _, declaration := range file.Decls {
			function, ok := declaration.(*ast.FuncDecl)
			if !ok || function.Body == nil {
				continue
			}
			hasPipe, hasStdioRedirect, hasReaderGoroutine, hasConcurrentRead := inspectCaptureFunction(function)
			if !hasPipe || !hasStdioRedirect {
				continue
			}

			key := relative + ":" + function.Name.Name
			if _, allowed := streamingCaptures[key]; !allowed {
				t.Errorf("%s redirects process output through os.Pipe; use NewDrainingPipe for post-run capture", key)
				continue
			}
			seenStreamingCaptures[key] = true
			if !hasReaderGoroutine || !hasConcurrentRead {
				t.Errorf("%s is an approved streaming capture and must keep a concurrent reader", key)
			}
		}
		return nil
	})
	require.NoError(t, err)
	for key := range streamingCaptures {
		require.True(t, seenStreamingCaptures[key], "streaming capture allowlist entry must exist")
	}
}

func repositoryRoot(t *testing.T) string {
	t.Helper()
	current, err := os.Getwd()
	require.NoError(t, err)
	for {
		if _, err := os.Stat(filepath.Join(current, "go.mod")); err == nil {
			return current
		}
		parent := filepath.Dir(current)
		require.NotEqual(t, current, parent, "could not find repository root from working directory")
		current = parent
	}
}

func inspectCaptureFunction(function *ast.FuncDecl) (hasPipe, hasStdioRedirect, hasReaderGoroutine, hasConcurrentRead bool) {
	ast.Inspect(function.Body, func(node ast.Node) bool {
		switch node := node.(type) {
		case *ast.CallExpr:
			if selector, ok := node.Fun.(*ast.SelectorExpr); ok {
				if receiver, ok := selector.X.(*ast.Ident); ok && receiver.Name == "os" && selector.Sel.Name == "Pipe" {
					hasPipe = true
				}
			}
		case *ast.AssignStmt:
			for _, expression := range node.Lhs {
				selector, ok := expression.(*ast.SelectorExpr)
				if !ok {
					continue
				}
				receiver, ok := selector.X.(*ast.Ident)
				if ok && receiver.Name == "os" && (selector.Sel.Name == "Stdout" || selector.Sel.Name == "Stderr") {
					hasStdioRedirect = true
				}
			}
		case *ast.GoStmt:
			hasReaderGoroutine = true
			ast.Inspect(node.Call, func(concurrentNode ast.Node) bool {
				call, ok := concurrentNode.(*ast.CallExpr)
				if !ok {
					return true
				}
				selector, ok := call.Fun.(*ast.SelectorExpr)
				if ok && selector.Sel.Name == "Read" {
					hasConcurrentRead = true
				}
				return true
			})
		}
		return true
	})
	return hasPipe, hasStdioRedirect, hasReaderGoroutine, hasConcurrentRead
}
