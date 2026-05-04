package report

import (
	"bytes"
	"strings"
	"testing"

	"github.com/hashmap-kz/godedup/internal/hash"
)

func funcInfo(name, file string, line, stmts, lines int, top uint64, seq ...uint64) hash.FuncInfo {
	return hash.FuncInfo{
		Name:     name,
		File:     file,
		Line:     line,
		TopHash:  top,
		StmtSeq:  seq,
		NumStmts: stmts,
		NumLines: lines,
	}
}

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.MinSimilarity != 0.85 {
		t.Fatalf("MinSimilarity = %v, want 0.85", cfg.MinSimilarity)
	}
	if cfg.MinStmts != 3 {
		t.Fatalf("MinStmts = %d, want 3", cfg.MinStmts)
	}
	if cfg.ExactOnly {
		t.Fatal("ExactOnly = true, want false")
	}
}

func TestDetectExactClones(t *testing.T) {
	funcs := []hash.FuncInfo{
		funcInfo("pkg.A", "a.go", 10, 3, 5, 100, 1, 2, 3),
		funcInfo("pkg.B", "b.go", 20, 3, 5, 100, 1, 2, 3),
		funcInfo("pkg.C", "c.go", 30, 3, 5, 200, 7, 8, 9),
	}

	clones := Detect(funcs, Config{MinSimilarity: 0.85, MinStmts: 3})
	if len(clones) != 1 {
		t.Fatalf("len(clones) = %d, want 1", len(clones))
	}
	if !clones[0].Exact {
		t.Fatal("clone is not exact")
	}
	if clones[0].Similarity != 1.0 {
		t.Fatalf("Similarity = %v, want 1.0", clones[0].Similarity)
	}
	if len(clones[0].Funcs) != 2 {
		t.Fatalf("len(Funcs) = %d, want 2", len(clones[0].Funcs))
	}
}

func TestDetectExactOnlySkipsNearClones(t *testing.T) {
	funcs := []hash.FuncInfo{
		funcInfo("pkg.A", "a.go", 10, 4, 5, 100, 1, 2, 3, 4),
		funcInfo("pkg.B", "b.go", 20, 4, 5, 200, 1, 2, 9, 4),
	}

	clones := Detect(funcs, Config{MinSimilarity: 0.75, MinStmts: 3, ExactOnly: true})
	if len(clones) != 0 {
		t.Fatalf("len(clones) = %d, want 0", len(clones))
	}
}

func TestDetectNearClones(t *testing.T) {
	funcs := []hash.FuncInfo{
		funcInfo("pkg.A", "a.go", 10, 4, 5, 100, 1, 2, 3, 4),
		funcInfo("pkg.B", "b.go", 20, 4, 5, 200, 1, 2, 9, 4),
		funcInfo("pkg.C", "c.go", 30, 4, 5, 300, 8, 8, 8, 8),
	}

	clones := Detect(funcs, Config{MinSimilarity: 0.75, MinStmts: 3})
	if len(clones) != 1 {
		t.Fatalf("len(clones) = %d, want 1", len(clones))
	}
	if clones[0].Exact {
		t.Fatal("clone is exact, want near")
	}
	if clones[0].Similarity != 0.75 {
		t.Fatalf("Similarity = %v, want 0.75", clones[0].Similarity)
	}
	if len(clones[0].Funcs) != 2 {
		t.Fatalf("len(Funcs) = %d, want 2", len(clones[0].Funcs))
	}
}

func TestDetectRespectsMinStmts(t *testing.T) {
	funcs := []hash.FuncInfo{
		funcInfo("pkg.A", "a.go", 10, 2, 5, 100, 1, 2),
		funcInfo("pkg.B", "b.go", 20, 2, 5, 100, 1, 2),
	}

	clones := Detect(funcs, Config{MinSimilarity: 0.85, MinStmts: 3})
	if len(clones) != 0 {
		t.Fatalf("len(clones) = %d, want 0", len(clones))
	}
}

func TestDetectNearCloneStatementCountPrefilter(t *testing.T) {
	funcs := []hash.FuncInfo{
		funcInfo("pkg.A", "a.go", 10, 3, 5, 100, 1, 2, 3),
		funcInfo("pkg.B", "b.go", 20, 10, 12, 200, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10),
	}

	clones := Detect(funcs, Config{MinSimilarity: 0.30, MinStmts: 3})
	if len(clones) != 0 {
		t.Fatalf("len(clones) = %d, want 0", len(clones))
	}
}

func TestPrintNoClones(t *testing.T) {
	var buf bytes.Buffer
	Print(&buf, nil, "")
	if got := strings.TrimSpace(buf.String()); got != "godedup: no structural duplicates found" {
		t.Fatalf("Print() = %q", got)
	}
}

func TestRelativePath(t *testing.T) {
	tests := []struct {
		name string
		path string
		cwd  string
		want string
	}{
		{name: "inside cwd", path: "/repo/internal/a.go", cwd: "/repo", want: "internal/a.go"},
		{name: "outside cwd", path: "/other/a.go", cwd: "/repo", want: "/other/a.go"},
		{name: "empty cwd", path: "/repo/a.go", cwd: "", want: "/repo/a.go"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := relativePath(tt.path, tt.cwd); got != tt.want {
				t.Fatalf("relativePath(%q, %q) = %q, want %q", tt.path, tt.cwd, got, tt.want)
			}
		})
	}
}
