package report

import (
	"io"

	"github.com/hashmap-kz/godedup/internal/x/fmtx"
)

func PrintJSON(w io.Writer, clones []Clone) {
	fmtx.Fprintln(w, "[")
	for i, clone := range clones {
		fmtx.Fprintf(w, `  {"exact":%v,"similarity":%.2f,"functions":[`,
			clone.Exact, clone.Similarity)
		for j, f := range clone.Funcs {
			if j > 0 {
				fmtx.Fprint(w, ",")
			}
			fmtx.Fprintf(w, `{"name":%q,"file":%q,"line":%d,"stmts":%d}`,
				f.Name, f.File, f.Line, f.NumStmts)
		}
		fmtx.Fprint(w, "]}")
		if i < len(clones)-1 {
			fmtx.Fprintln(w, ",")
		} else {
			fmtx.Fprintln(w)
		}
	}
	fmtx.Fprintln(w, "]")
}
