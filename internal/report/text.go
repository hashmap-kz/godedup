package report

import (
	"fmt"
	"io"
	"sort"

	"github.com/hashmap-kz/godedup/internal/hash"
	"github.com/hashmap-kz/godedup/internal/x/fmtx"
)

// Print writes a human-readable report to w.
func Print(w io.Writer, clones []Clone, cwd string) {
	if len(clones) == 0 {
		fmtx.Fprintln(w, "godedup: no structural duplicates found")
		return
	}

	exact := 0
	near := 0
	for _, c := range clones {
		if c.Exact {
			exact++
		} else {
			near++
		}
	}

	fmtx.Fprintf(w, "godedup: found %d clone group(s) (%d exact, %d near)\n\n",
		len(clones), exact, near)

	for i, clone := range clones {
		kind := "EXACT"
		simStr := "100%"
		if !clone.Exact {
			kind = "NEAR"
			simStr = fmt.Sprintf("%.0f%%", clone.Similarity*100)
		}

		fmtx.Fprintf(w, "=== clone group %d [%s %s similarity] ===\n",
			i+1, kind, simStr)

		// sort functions by file+line for stable output
		sorted := make([]hash.FuncInfo, len(clone.Funcs))
		copy(sorted, clone.Funcs)
		sort.Slice(sorted, func(a, b int) bool {
			if sorted[a].File != sorted[b].File {
				return sorted[a].File < sorted[b].File
			}
			return sorted[a].Line < sorted[b].Line
		})

		for _, f := range sorted {
			relPath := relativePath(f.File, cwd)
			fmtx.Fprintf(w, "  %s\n", f.Name)
			fmtx.Fprintf(w, "    %s:%d  (%d stmts, %d lines)\n",
				relPath, f.Line, f.NumStmts, f.NumLines)
		}
		fmtx.Fprintln(w)
	}
}
