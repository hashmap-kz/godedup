package report

import (
	"bytes"
	"encoding/json"
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

func TestPrintHumanReadable(t *testing.T) {
	clones := []Clone{{
		Exact:      true,
		Similarity: 1.0,
		Funcs: []hash.FuncInfo{
			funcInfo("pkg.B", "/repo/b.go", 20, 3, 7, 100, 1, 2, 3),
			funcInfo("pkg.A", "/repo/a.go", 10, 3, 7, 100, 1, 2, 3),
		},
	}}

	var buf bytes.Buffer
	Print(&buf, clones, "/repo")
	got := buf.String()
	for _, want := range []string{
		"godedup: found 1 clone group(s) (1 exact, 0 near)",
		"=== clone group 1 [EXACT 100% similarity] ===",
		"pkg.A",
		"a.go:10  (3 stmts, 7 lines)",
		"pkg.B",
		"b.go:20  (3 stmts, 7 lines)",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("Print() missing %q in:\n%s", want, got)
		}
	}
	if strings.Contains(got, "suggestion:") {
		t.Fatalf("Print() contains superfluous suggestion:\n%s", got)
	}
	if strings.Index(got, "pkg.A") > strings.Index(got, "pkg.B") {
		t.Fatalf("functions are not sorted by file/line:\n%s", got)
	}
}

func TestPrintTable(t *testing.T) {
	clones := []Clone{{
		Exact:      false,
		Similarity: 0.91,
		Funcs: []hash.FuncInfo{
			funcInfo("api.handleOrderCreate", "/repo/pkg/api/order.go", 51, 19, 47, 200, 1, 2, 9),
			funcInfo("api.handleUserCreate", "/repo/pkg/api/user.go", 44, 18, 45, 100, 1, 2, 3),
		},
	}}

	var buf bytes.Buffer
	PrintTable(&buf, clones, "/repo")
	got := buf.String()
	for _, want := range []string{
		"GROUP",
		"TYPE",
		"SIM",
		"FUNCTION",
		"LOCATION",
		"1      NEAR  91%",
		"api.handleOrderCreate",
		"pkg/api/order.go:51",
		"api.handleUserCreate",
		"pkg/api/user.go:44",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("PrintTable() missing %q in:\n%s", want, got)
		}
	}
}

func TestPrintHTML(t *testing.T) {
	clones := []Clone{{
		Exact:      true,
		Similarity: 1.0,
		Funcs: []hash.FuncInfo{
			funcInfo("pkg.B", "/repo/b.go", 20, 3, 7, 100, 1, 2, 3),
			funcInfo("pkg.A", "/repo/a.go", 10, 3, 7, 100, 1, 2, 3),
		},
	}}
	clones[0].Funcs[0].Source = "func B() int {\n\tx := 1\n\ty := 2\n\treturn x + y\n}"
	clones[0].Funcs[1].Source = "func A() int {\n\tx := 1\n\ty := 2\n\treturn x + y\n}"

	var buf bytes.Buffer
	PrintHTML(&buf, clones, "/repo")
	got := buf.String()
	for _, want := range []string{
		"<!doctype html>",
		"godedup report",
		"1 groups",
		"1 exact",
		"class=\"group exact funcs-2\"",
		"pkg.A",
		"a.go:10",
		"func A() int",
		"pkg.B",
		"b.go:20",
		"file:///repo/a.go:10",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("PrintHTML() missing %q in:\n%s", want, got)
		}
	}
	for _, unwanted := range []string{"Suggestion:", "review this clone group"} {
		if strings.Contains(got, unwanted) {
			t.Fatalf("PrintHTML() contains unwanted %q in:\n%s", unwanted, got)
		}
	}
}

func TestPrintJSON(t *testing.T) {
	clones := []Clone{{
		Exact:      true,
		Similarity: 1.0,
		Funcs: []hash.FuncInfo{
			funcInfo("pkg.A", "a.go", 10, 3, 7, 100, 1, 2, 3),
		},
	}}

	var buf bytes.Buffer
	PrintJSON(&buf, clones)

	var decoded []struct {
		Exact      bool    `json:"exact"`
		Similarity float64 `json:"similarity"`
		Functions  []struct {
			Name  string `json:"name"`
			File  string `json:"file"`
			Line  int    `json:"line"`
			Stmts int    `json:"stmts"`
		} `json:"functions"`
	}
	if err := json.Unmarshal(buf.Bytes(), &decoded); err != nil {
		t.Fatalf("PrintJSON produced invalid JSON: %v\n%s", err, buf.String())
	}
	if len(decoded) != 1 {
		t.Fatalf("len(decoded) = %d, want 1", len(decoded))
	}
	if !decoded[0].Exact || decoded[0].Similarity != 1.0 {
		t.Fatalf("decoded clone = %+v, want exact similarity 1.0", decoded[0])
	}
	if got := decoded[0].Functions[0].Name; got != "pkg.A" {
		t.Fatalf("function name = %q, want pkg.A", got)
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
