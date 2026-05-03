package hash

import (
	"go/ast"
	"go/parser"
	"go/token"
	"testing"
)

func parseSingleFunc(t *testing.T, src string) (*token.FileSet, *ast.FuncDecl) {
	t.Helper()

	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "sample.go", src, 0)
	if err != nil {
		t.Fatalf("parse source: %v", err)
	}

	for _, decl := range file.Decls {
		if fn, ok := decl.(*ast.FuncDecl); ok {
			return fset, fn
		}
	}
	t.Fatal("source does not contain function declaration")
	return nil, nil
}

func hashFuncFromSource(t *testing.T, src string) FuncInfo {
	t.Helper()

	fset, fn := parseSingleFunc(t, src)
	return New(fset).HashFunc("sample", "sample.go", fn)
}

func TestHashFuncNormalizesIdentifierAndLiteralNames(t *testing.T) {
	a := hashFuncFromSource(t, `package sample
func validateUser(user string) error {
	if user == "" {
		return errors.New("empty user name")
	}
	return nil
}`)

	b := hashFuncFromSource(t, `package sample
func validateOrder(order string) error {
	if order == "abc" {
		return fmt.New("empty order code")
	}
	return nil
}`)

	if a.TopHash != b.TopHash {
		t.Fatalf("expected structurally equivalent functions to have same top hash: %d != %d", a.TopHash, b.TopHash)
	}
}

func TestHashFuncPreservesOperators(t *testing.T) {
	a := hashFuncFromSource(t, `package sample
func add(a, b int) int {
	c := a + b
	d := c + b
	return d
}`)

	b := hashFuncFromSource(t, `package sample
func sub(a, b int) int {
	c := a - b
	d := c + b
	return d
}`)

	if a.TopHash == b.TopHash {
		t.Fatal("expected different operators to produce different top hashes")
	}
}

func TestHashFuncPreservesStatementOrder(t *testing.T) {
	a := hashFuncFromSource(t, `package sample
func first() int {
	a := 1
	if a > 0 {
		return a
	}
	return 0
}`)

	b := hashFuncFromSource(t, `package sample
func second() int {
	if a > 0 {
		return a
	}
	a := 1
	return 0
}`)

	if a.TopHash == b.TopHash {
		t.Fatal("expected statement order to affect top hash")
	}
}

func TestHashFuncInfoMetadata(t *testing.T) {
	info := hashFuncFromSource(t, `package sample
func add(a, b int) int {
	c := a + b
	d := c + b
	return d
}`)

	if info.Name != "sample.add" {
		t.Fatalf("Name = %q, want %q", info.Name, "sample.add")
	}
	if info.File != "sample.go" {
		t.Fatalf("File = %q, want sample.go", info.File)
	}
	if info.Line != 2 {
		t.Fatalf("Line = %d, want 2", info.Line)
	}
	if info.NumStmts != 3 {
		t.Fatalf("NumStmts = %d, want 3", info.NumStmts)
	}
	if len(info.StmtSeq) != info.NumStmts {
		t.Fatalf("len(StmtSeq) = %d, want %d", len(info.StmtSeq), info.NumStmts)
	}
	if info.NumLines != 5 {
		t.Fatalf("NumLines = %d, want 5", info.NumLines)
	}
}

func TestQualifiedName(t *testing.T) {
	tests := []struct {
		name string
		src  string
		want string
	}{
		{
			name: "function",
			src: `package sample
func Run() {}`,
			want: "sample.Run",
		},
		{
			name: "value receiver",
			src: `package sample
func (s Store) Run() {}`,
			want: "sample.(Store).Run",
		},
		{
			name: "pointer receiver",
			src: `package sample
func (s *Store) Run() {}`,
			want: "sample.(*Store).Run",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, fn := parseSingleFunc(t, tt.src)
			if got := qualifiedName("sample", fn); got != tt.want {
				t.Fatalf("qualifiedName() = %q, want %q", got, tt.want)
			}
		})
	}
}
