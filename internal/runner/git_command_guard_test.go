package runner

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// Git can start filters that inherit its pipes; runner callers must use the
// shared cancellation-aware command, not a plain os/exec Git command.
func TestRunnerGitCallsUseCancellationAwareCommand(t *testing.T) {
	entries, err := os.ReadDir(".")
	require.NoError(t, err)
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		file, parseErr := parser.ParseFile(token.NewFileSet(), name, nil, 0)
		require.NoError(t, parseErr)
		ast.Inspect(file, func(node ast.Node) bool {
			if method := plainGitExec(node); method != "" {
				t.Errorf("%s uses plain exec.%s for Git; use gitutil.Command", name, method)
			}
			return true
		})
	}
}

func plainGitExec(node ast.Node) string {
	call, method := plainExecCall(node)
	if call == nil {
		return ""
	}
	index := 0
	if method == "CommandContext" {
		index = 1
	}
	if len(call.Args) <= index {
		return ""
	}
	arg, ok := call.Args[index].(*ast.BasicLit)
	if !ok || arg.Kind != token.STRING {
		return ""
	}
	value, err := strconv.Unquote(arg.Value)
	if err != nil || value != "git" {
		return ""
	}
	return method
}

func plainExecCall(node ast.Node) (*ast.CallExpr, string) {
	call, ok := node.(*ast.CallExpr)
	if !ok {
		return nil, ""
	}
	selector, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return nil, ""
	}
	qualifier, ok := selector.X.(*ast.Ident)
	if !ok || qualifier.Name != "exec" {
		return nil, ""
	}
	method := selector.Sel.Name
	if method != "Command" && method != "CommandContext" {
		return nil, ""
	}
	return call, method
}
