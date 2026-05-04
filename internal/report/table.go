package report

import (
	"fmt"
	"io"
	"sort"

	"github.com/hashmap-kz/godedup/internal/hash"
	"github.com/hashmap-kz/godedup/internal/x/fmtx"
)

// PrintTable writes aligned tabular output suitable for terminal viewing.
// Columns: GROUP  TYPE   SIM   FUNCTION  LOCATION  STMTS  LINES
func PrintTable(w io.Writer, clones []Clone, cwd string) {
	if len(clones) == 0 {
		fmtx.Fprintln(w, "godedup: no structural duplicates found")
		return
	}

	// collect all rows first so we can compute column widths
	type row struct {
		group    string
		typ      string
		sim      string
		function string
		location string
		stmts    string
		lines    string
	}

	var rows []row
	for i, clone := range clones {
		typ := kindExact
		sim := sim100Percent
		if !clone.Exact {
			typ = kindNear
			sim = fmt.Sprintf("%.0f%%", clone.Similarity*100)
		}

		sorted := make([]hash.FuncInfo, len(clone.Funcs))
		copy(sorted, clone.Funcs)
		sort.Slice(sorted, func(a, b int) bool {
			if sorted[a].File != sorted[b].File {
				return sorted[a].File < sorted[b].File
			}
			return sorted[a].Line < sorted[b].Line
		})

		for _, f := range sorted {
			loc := fmt.Sprintf("%s:%d", relativePath(f.File, cwd), f.Line)
			rows = append(rows, row{
				group:    fmt.Sprintf("%d", i+1),
				typ:      typ,
				sim:      sim,
				function: f.Name,
				location: loc,
				stmts:    fmt.Sprintf("%d", f.NumStmts),
				lines:    fmt.Sprintf("%d", f.NumLines),
			})
		}
	}

	// compute column widths
	headers := row{"GROUP", "TYPE", "SIM", "FUNCTION", "LOCATION", "STMTS", "LINES"}
	widths := [7]int{
		len(headers.group),
		len(headers.typ),
		len(headers.sim),
		len(headers.function),
		len(headers.location),
		len(headers.stmts),
		len(headers.lines),
	}
	for _, r := range rows {
		vals := [7]string{r.group, r.typ, r.sim, r.function, r.location, r.stmts, r.lines}
		for i, v := range vals {
			if len(v) > widths[i] {
				widths[i] = len(v)
			}
		}
	}

	fmtRow := func(r row) string {
		return fmt.Sprintf("%-*s  %-*s  %-*s  %-*s  %-*s  %-*s  %s",
			widths[0], r.group,
			widths[1], r.typ,
			widths[2], r.sim,
			widths[3], r.function,
			widths[4], r.location,
			widths[5], r.stmts,
			r.lines,
		)
	}

	// header
	fmtx.Fprintln(w, fmtRow(headers))

	// separator using only dashes
	sep := ""
	total := widths[0] + widths[1] + widths[2] + widths[3] + widths[4] + widths[5] + widths[6] + 12
	for i := 0; i < total; i++ {
		sep += "-"
	}
	fmtx.Fprintln(w, sep)

	// rows: emit the separator between groups
	prevGroup := ""
	for _, r := range rows {
		if prevGroup != "" && r.group != prevGroup {
			fmtx.Fprintln(w, sep)
		}
		fmtx.Fprintln(w, fmtRow(r))
		prevGroup = r.group
	}
}
