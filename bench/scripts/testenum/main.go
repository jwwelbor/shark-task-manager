// Command testenum performs a STATIC enumeration of the top-level Go test
// functions declared in a package directory's *.go files, using go/parser
// and go/types. It never builds, links, executes, or runs go:generate for
// any code belonging to the target package -- parsing and type-checking
// are both purely static analyses (the same class of operation gofmt,
// go vet, and gopls perform), so nothing that package's own init(),
// TestMain, or test bodies do at runtime (including calling os.Exit at
// any point) can influence, suppress, or forge this program's output.
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
// and type-checks the actual test source, which is exactly as
// trustworthy as reading any other file in the checkout, and identically
// inert.
//
// # Why go/types, not syntax pattern-matching (code review round 6, 6a)
//
// An earlier version of this tool matched a test function's parameter
// type by AST shape alone: it required a literal `*ast.SelectorExpr`
// spelled exactly `*testing.T`. That under-recognizes real, running,
// failing Go tests: `go test` itself genuinely compiles and runs a test
// function typed `*T` via a dot import (`import . "testing"`) or `*foo.T`
// via a renamed import (`import foo "testing"`) -- verified empirically,
// both forms produce real `--- FAIL` output and a non-zero process exit
// in isolation. A syntax-only enumerator that doesn't recognize these
// spellings under-counts the "expected" set, which is the dangerous
// direction: a real, currently-failing test becomes invisible to the
// completeness check that is supposed to catch exactly this.
//
// go/types resolves each candidate parameter's type to its declaring
// package's canonical IMPORT PATH and type name, independent of
// whatever local identifier spells it in the source -- so a normal
// import, a renamed import, and a dot import of the standard library
// "testing" package all resolve to the same identity (path "testing",
// name "T"), while a same- or different-package type that is merely
// ALSO named "T" (e.g. a hand-rolled `type T struct{}` sharing a
// parameter position) resolves to a different declaring package and is
// correctly rejected. Type-checking is still fully static: it never
// executes the code being checked, the same way `go vet` and editor
// tooling type-check code they never run.
//
// Import resolution here is intentionally scoped to what this tool
// actually needs: go/importer's default (GOROOT-based) importer resolves
// the standard library "testing" package correctly regardless of import
// spelling, which is the only identity this tool needs to establish.
// Resolving arbitrary same-module or third-party imports would need
// full go/packages-style module resolution (shelling out to `go list`),
// which this tool deliberately does not adopt, to keep testenum a
// self-contained module with zero external dependencies and no reliance
// on network-resolvable module metadata (REQ-NF-005, offline capability).
// A parameter type whose import cannot be resolved this way type-checks
// to an invalid/unknown type and is correctly NOT counted as a test --
// failing closed, not open: this tool's job is only to prove a LOWER
// BOUND on what must have a terminal event, and undercounting in that
// direction is a narrower, non-exploitable gap (see admit.sh's
// check_p2p_green, 6b: its fail-check no longer depends on this
// enumeration to catch a real observed failure at all -- it scans every
// terminal event go test actually produced, so an enumeration gap here
// can at most weaken the "missing evidence" completeness check, never
// hide a real, already-observed failure).
//
// usage: testenum <package-dir>
// Prints one bare test function name per line to stdout, sorted and
// de-duplicated. Exits non-zero, with a message on stderr, if the
// directory cannot be read or any *.go file in it fails to parse as Go
// source -- a directory whose source cannot be read is exactly the case
// callers must not silently treat as "zero tests expected".
package main

import (
	"fmt"
	"go/ast"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
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

// enumerate parses every *.go file in dir (production and test sources
// together, exactly as `go build`/`go test` see the package -- a test
// function's parameter type can only be resolved correctly if the
// import block that names "testing" is present, and cross-file symbol
// references within the same package resolve cleanly only when every
// file in it is parsed and type-checked together) and returns the
// top-level Go test function names whose single parameter resolves,
// under go/types, to the standard library testing.T.
func enumerate(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("reading directory %s: %w", dir, err)
	}

	fset := token.NewFileSet()
	filesByPackage := map[string][]*ast.File{}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") {
			continue
		}
		path := filepath.Join(dir, entry.Name())
		file, err := parser.ParseFile(fset, path, nil, 0)
		if err != nil {
			return nil, fmt.Errorf("parsing %s: %w", path, err)
		}
		pkgName := file.Name.Name
		filesByPackage[pkgName] = append(filesByPackage[pkgName], file)
	}

	seen := map[string]struct{}{}
	for pkgName, files := range filesByPackage {
		info := &types.Info{
			Types: make(map[ast.Expr]types.TypeAndValue),
		}
		conf := types.Config{
			Importer: importer.Default(),
			// Type errors are expected and tolerated here: this
			// directory's files are type-checked in isolation from the
			// rest of their module (no module-aware import resolution,
			// see this file's header), so references to anything beyond
			// the standard library may fail to resolve. go/types still
			// populates Info.Types with everything it COULD resolve
			// despite such errors -- exactly what this tool needs for
			// parameter-type identity, and nothing more is required to
			// be error-free.
			Error: func(error) {},
		}
		// Check's own return values are deliberately ignored for the same
		// reason: its error reflects the same tolerated errors already
		// discarded above, not a reason to abandon the partial type
		// info that was still recorded.
		_, _ = conf.Check(pkgName, fset, files, info)

		for _, file := range files {
			for _, decl := range file.Decls {
				fn, ok := decl.(*ast.FuncDecl)
				if !ok {
					continue
				}
				if !isTestName(fn) {
					continue
				}
				if !hasTestingTParam(fn, info) {
					continue
				}
				seen[fn.Name.Name] = struct{}{}
			}
		}
	}

	names := make([]string, 0, len(seen))
	for n := range seen {
		names = append(names, n)
	}
	return names, nil
}

// isTestName reports whether fn's name and shape match the naming rule
// `go test` itself uses for a candidate test function: a top-level
// (non-method) function, exactly one parameter, named "Test" or
// "Test"+<name where the first rune is not lowercase> (e.g. "Testable"
// is not a test, "TestFoo" is).
func isTestName(fn *ast.FuncDecl) bool {
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
			return false
		}
	}
	params := fn.Type.Params
	return params != nil && len(params.List) == 1 && len(params.List[0].Names) == 1
}

// hasTestingTParam reports whether fn's single parameter's TYPE --
// resolved via go/types, not its surface AST spelling -- is a pointer to
// the standard library's testing.T. This is what makes a normal import,
// a renamed import, and a dot import of "testing" all recognized
// identically, and what makes an unrelated type merely also named "T"
// correctly rejected regardless of how it's imported or spelled.
func hasTestingTParam(fn *ast.FuncDecl, info *types.Info) bool {
	paramType := fn.Type.Params.List[0].Type
	tv, ok := info.Types[paramType]
	if !ok || tv.Type == nil {
		return false
	}
	ptr, ok := tv.Type.(*types.Pointer)
	if !ok {
		return false
	}
	named, ok := ptr.Elem().(*types.Named)
	if !ok {
		return false
	}
	obj := named.Obj()
	return obj != nil && obj.Pkg() != nil && obj.Pkg().Path() == "testing" && obj.Name() == "T"
}
