package report

import (
	"bytes"
	"strings"
	"testing"

	"github.com/hashmap-kz/godedup/internal/hash"
)

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
