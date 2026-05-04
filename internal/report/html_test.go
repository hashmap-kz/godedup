package report

import (
	"bytes"
	"strings"
	"testing"

	"github.com/hashmap-kz/godedup/internal/hash"
)

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
		`class="group exact funcs-2"`,
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
	for _, unwanted := range []string{"Suggestion:", "review this clone group", "#ZgotmplZ"} {
		if strings.Contains(got, unwanted) {
			t.Fatalf("PrintHTML() contains unwanted %q in:\n%s", unwanted, got)
		}
	}
}

func TestPrintHTMLEmpty(t *testing.T) {
	var buf bytes.Buffer
	PrintHTML(&buf, nil, "/repo")
	got := buf.String()
	if !strings.Contains(got, "No structural duplicates found") {
		t.Fatalf("PrintHTML() empty case missing expected message in:\n%s", got)
	}
	if strings.Contains(got, "<article") {
		t.Fatalf("PrintHTML() empty case should not contain <article> in:\n%s", got)
	}
}

func TestPrintHTMLNearClone(t *testing.T) {
	clones := []Clone{{
		Exact:      false,
		Similarity: 0.88,
		Funcs: []hash.FuncInfo{
			funcInfo("pkg.A", "/repo/a.go", 10, 4, 8, 100, 1, 2, 3, 4),
			funcInfo("pkg.B", "/repo/b.go", 20, 4, 8, 200, 1, 2, 9, 4),
		},
	}}

	var buf bytes.Buffer
	PrintHTML(&buf, clones, "/repo")
	got := buf.String()

	for _, want := range []string{
		`class="group near funcs-2"`,
		`class="badge near"`,
		"88%",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("PrintHTML() near clone missing %q in:\n%s", want, got)
		}
	}
}

func TestPrintHTMLSortsByFileLine(t *testing.T) {
	clones := []Clone{{
		Exact:      true,
		Similarity: 1.0,
		Funcs: []hash.FuncInfo{
			funcInfo("pkg.B", "/repo/b.go", 20, 3, 5, 100, 1, 2, 3),
			funcInfo("pkg.A", "/repo/a.go", 10, 3, 5, 100, 1, 2, 3),
		},
	}}

	var buf bytes.Buffer
	PrintHTML(&buf, clones, "/repo")
	got := buf.String()

	posA := strings.Index(got, "pkg.A")
	posB := strings.Index(got, "pkg.B")
	if posA > posB {
		t.Fatalf("PrintHTML() functions not sorted by file/line: pkg.A at %d, pkg.B at %d", posA, posB)
	}
}
