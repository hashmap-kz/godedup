package load

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"

	"github.com/hashmap-kz/godedup/internal/cmd"

	"github.com/hashmap-kz/godedup/internal/hash"
)

// Result holds all analyzed functions from the given paths.
type Result struct {
	Funcs []hash.FuncInfo
}

// TODO: later this may be done parallel in three steps:
// 1. Collect files
// 2. Parse them concurrently into []funcs
// 3. Join and sort results

// Load parses all Go files under the given paths and returns
// a FuncInfo for every function declaration found.
// Paths may be files or directories (walked recursively).
func Load(paths []string, inp *cmd.LoadInput) (*Result, error) {
	fset := token.NewFileSet()
	hasher := hash.New(fset)
	var funcs []hash.FuncInfo

	for _, path := range paths {
		info, err := os.Stat(path)
		if err != nil {
			return nil, err
		}

		if info.IsDir() {
			if err := walkDir(path, fset, hasher, inp, &funcs); err != nil {
				return nil, err
			}
		} else {
			if err := parseFile(path, fset, hasher, inp, &funcs); err != nil {
				return nil, err
			}
		}
	}

	return &Result{Funcs: funcs}, nil
}

func walkDir(root string, fset *token.FileSet, hasher *hash.Hasher, inp *cmd.LoadInput, out *[]hash.FuncInfo) error {
	return filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			// skip hidden dirs and vendor
			base := d.Name()
			if base != "." && (strings.HasPrefix(base, ".") || base == "vendor") {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") {
			return nil
		}
		if inp.ExcludeRegex != nil && inp.ExcludeRegex.MatchString(path) {
			return nil
		}
		return parseFile(path, fset, hasher, inp, out)
	})
}

func parseFile(path string, fset *token.FileSet, hasher *hash.Hasher, inp *cmd.LoadInput, out *[]hash.FuncInfo) error {
	if inp.ExcludeRegex != nil && inp.ExcludeRegex.MatchString(path) {
		return nil
	}

	f, err := parser.ParseFile(fset, path, nil, 0)
	if err != nil {
		// skip unparseable files
		return nil
	}

	pkg := f.Name.Name

	for _, decl := range f.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Body == nil {
			continue
		}

		// skip very short functions - not interesting for dedup
		if len(fn.Body.List) < 3 {
			continue
		}

		info := hasher.HashFunc(pkg, path, fn)
		if info.Name == "" {
			continue
		}
		*out = append(*out, info)
	}

	return nil
}
