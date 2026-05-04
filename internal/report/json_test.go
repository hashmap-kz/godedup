package report

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/hashmap-kz/godedup/internal/hash"
)

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
