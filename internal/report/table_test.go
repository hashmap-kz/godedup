package report

import (
	"bytes"
	"strings"
	"testing"

	"github.com/hashmap-kz/godedup/internal/hash"
)

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
