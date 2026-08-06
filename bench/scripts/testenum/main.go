// Command testenum performs a STATIC enumeration of the top-level Go test
// functions declared in a package directory's *_test.go files, using
// go/parser + go/ast. It never builds or executes any code belonging to
// the target package -- it only parses source text -- so nothing that
// package's own init(), TestMain, or test bodies do at runtime (including
// calling os.Exit at any point) can influence, suppress, or forge this
// program's output.
//
// This is the independent enumeration bench/scripts/admit.sh and
// bench/scripts/build-ledgers.sh cross-reference `go test -json`'s
// per-test terminal events against, per the property those callers must
// satisfy: no signal used to conclude "no failures occurred" or "the
// record is complete" may be forgeable by code inside the package under
// test. Per-test JSON events themselves are reliable (each is emitted as
// its test completes, before TestMain regains control to call os.Exit),
// but nothing upstream of this tool cross-referenced them against an
// independent count of how many tests were supposed to produce one.
// go test -list executes TestMain (verified empirically -- it is NOT a
// safe independent signal); go list resolves package metadata only and
// runs nothing. This tool goes one step further than go list: it reads
// the actual test source, which is exactly as trustworthy as reading any
// other file in the checkout, and identically inert.
//
// A function is counted as a Go test using the same rule `go test`
// itself uses (see https://pkg.go.dev/testing): it is a package-level
// (non-method) function whose name starts with "Test", where the rune
// immediately following "Test" (if any) is not lowercase, and whose
// signature is exactly func(*testing.T).
//
// usage: testenum <package-dir>
// Prints one bare test function name per line to stdout, sorted and
// de-duplicated. Exits non-zero, with a message on stderr, if the
// directory cannot be read or any *_test.go file in it fails to parse as
// Go source -- a directory whose test files cannot be read is exactly the
// case callers must not silently treat as "zero tests expected".
package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"
)

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintln(os.Stderr, "usage: testenum <package-dir>")
		os.Exit(2)
	}
	dir := os.Args[1]

	names, err := enumerate(dir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "testenum: %v\n", err)
		os.Exit(1)
	}

	sort.Strings(names)
	for _, n := range names {
		fmt.Println(n)
	}
}

func enumerate(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("reading directory %s: %w", dir, err)
	}

	seen := map[string]struct{}{}
	fset := token.NewFileSet()
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !strings.HasSuffix(name, "_test.go") {
			continue
		}
		path := filepath.Join(dir, name)
		file, err := parser.ParseFile(fset, path, nil, 0)
		if err != nil {
			return nil, fmt.Errorf("parsing %s: %w", path, err)
		}
		for _, decl := range file.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok {
				continue
			}
			if !isTestFunc(fn) {
				continue
			}
			seen[fn.Name.Name] = struct{}{}
		}
	}

	names := make([]string, 0, len(seen))
	for n := range seen {
		names = append(names, n)
	}
	return names, nil
}

// isTestFunc reports whether fn matches the exact shape `go test` itself
// recognizes as a test function: a top-level (non-method) function named
// "Test" or "Test"+<name where the first rune is not lowercase>, taking
// exactly one parameter of type *testing.T.
func isTestFunc(fn *ast.FuncDecl) bool {
	if fn.Recv != nil {
		return false // methods are never tests
	}
	name := fn.Name.Name
	if !strings.HasPrefix(name, "Test") {
		return false
	}
	rest := name[len("Test"):]
	if rest != "" {
		r, _ := utf8.DecodeRuneInString(rest)
		if unicode.IsLower(r) {
			return false // e.g. "Testable" is not a test
		}
	}

	params := fn.Type.Params
	if params == nil || len(params.List) != 1 || len(params.List[0].Names) != 1 {
		return false
	}
	star, ok := params.List[0].Type.(*ast.StarExpr)
	if !ok {
		return false
	}
	sel, ok := star.X.(*ast.SelectorExpr)
	if !ok {
		return false
	}
	pkgIdent, ok := sel.X.(*ast.Ident)
	if !ok {
		return false
	}
	return pkgIdent.Name == "testing" && sel.Sel.Name == "T"
}
