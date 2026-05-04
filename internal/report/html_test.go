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
