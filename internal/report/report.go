package report

import (
	"fmt"
	"io"
	"sort"
	"strings"

	"godedup/internal/wrapx"

	"godedup/internal/hash"
)

// Clone represents a group of structurally identical or similar functions.
type Clone struct {
	Funcs      []hash.FuncInfo
	Similarity float64 // 1.0 = exact, < 1.0 = near-clone
	Exact      bool
}

// Config controls detection thresholds.
type Config struct {
	MinSimilarity float64 // minimum similarity to report (default 0.85)
	MinStmts      int     // minimum statements to consider (default 3)
	ExactOnly     bool    // only report exact structural clones
}

func DefaultConfig() Config {
	return Config{
		MinSimilarity: 0.85,
		MinStmts:      3,
	}
}

// Detect finds clone groups from the analyzed functions.
func Detect(funcs []hash.FuncInfo, cfg Config) []Clone {
	if len(funcs) == 0 {
		return nil
	}

	// Pass 1: exact clones via hash grouping.
	exactGroups := make(map[uint64][]hash.FuncInfo)
	for _, f := range funcs {
		if f.NumStmts < cfg.MinStmts {
			continue
		}
		exactGroups[f.TopHash] = append(exactGroups[f.TopHash], f)
	}

	var clones []Clone
	inExactClone := make(map[string]bool) // track functions already in an exact group

	for _, group := range exactGroups {
		if len(group) < 2 {
			continue
		}
		clones = append(clones, Clone{
			Funcs:      group,
			Similarity: 1.0,
			Exact:      true,
		})
		for _, f := range group {
			inExactClone[f.Name] = true
		}
	}

	if cfg.ExactOnly {
		sortClones(clones)
		return clones
	}

	// Pass 2: near-clones via pairwise similarity.
	// Only compare functions not already in an exact clone group.
	// O(n^2) but functions are typically < 10k in any codebase.
	var candidates []hash.FuncInfo
	for _, f := range funcs {
		if !inExactClone[f.Name] && f.NumStmts >= cfg.MinStmts {
			candidates = append(candidates, f)
		}
	}

	// build near-clone groups using union-find
	parent := make(map[string]string)
	similarity := make(map[[2]string]float64)

	var getRoot func(string) string
	getRoot = func(s string) string {
		if parent[s] == "" || parent[s] == s {
			return s
		}
		parent[s] = getRoot(parent[s])
		return parent[s]
	}

	union := func(a, b string, sim float64) {
		ra, rb := getRoot(a), getRoot(b)
		if ra == rb {
			return
		}
		parent[rb] = ra
		key := [2]string{ra, rb}
		if sim > similarity[key] {
			similarity[key] = sim
		}
	}

	for i := 0; i < len(candidates); i++ {
		for j := i + 1; j < len(candidates); j++ {
			a, b := candidates[i], candidates[j]

			// quick pre-filter: statement count must be within 30%
			ratio := float64(a.NumStmts) / float64(b.NumStmts)
			if ratio < 0.7 || ratio > 1.43 {
				continue
			}

			sim := hash.Similarity(&a, &b)
			if sim >= cfg.MinSimilarity {
				union(a.Name, b.Name, sim)
				key := [2]string{a.Name, b.Name}
				similarity[key] = sim
			}
		}
	}

	// collect groups
	groups := make(map[string][]hash.FuncInfo)
	for _, f := range candidates {
		root := getRoot(f.Name)
		groups[root] = append(groups[root], f)
	}

	for _, group := range groups {
		if len(group) < 2 {
			continue
		}
		// compute minimum similarity across the group
		// try both key orderings since the map was populated with candidates[i],candidates[j]
		// but group order comes from map iteration and may differ
		minSim := 1.0
		for i := 0; i < len(group); i++ {
			for j := i + 1; j < len(group); j++ {
				na, nb := group[i].Name, group[j].Name
				if s, ok := similarity[[2]string{na, nb}]; ok && s < minSim {
					minSim = s
				} else if s, ok := similarity[[2]string{nb, na}]; ok && s < minSim {
					minSim = s
				}
			}
		}
		clones = append(clones, Clone{
			Funcs:      group,
			Similarity: minSim,
			Exact:      false,
		})
	}

	sortClones(clones)
	return clones
}

func sortClones(clones []Clone) {
	sort.Slice(clones, func(i, j int) bool {
		// exact clones first, then by group size, then by similarity
		if clones[i].Exact != clones[j].Exact {
			return clones[i].Exact
		}
		if len(clones[i].Funcs) != len(clones[j].Funcs) {
			return len(clones[i].Funcs) > len(clones[j].Funcs)
		}
		return clones[i].Similarity > clones[j].Similarity
	})
}

// Print writes a human-readable report to w.
func Print(w io.Writer, clones []Clone, cwd string) {
	if len(clones) == 0 {
		wrapx.Fprintln(w, "godedup: no structural duplicates found")
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

	wrapx.Fprintf(w, "godedup: found %d clone group(s) (%d exact, %d near)\n\n",
		len(clones), exact, near)

	for i, clone := range clones {
		kind := "EXACT"
		simStr := "100%"
		if !clone.Exact {
			kind = "NEAR"
			simStr = fmt.Sprintf("%.0f%%", clone.Similarity*100)
		}

		wrapx.Fprintf(w, "=== clone group %d [%s %s similarity] ===\n",
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
			wrapx.Fprintf(w, "  %s\n", f.Name)
			wrapx.Fprintf(w, "    %s:%d  (%d stmts, %d lines)\n",
				relPath, f.Line, f.NumStmts, f.NumLines)
		}
		wrapx.Fprintln(w)
	}

	wrapx.Fprintf(w, "suggestion: extract shared logic into a common function\n")
	wrapx.Fprintf(w, "            or use generics if types differ\n")
}

// PrintJSON writes machine-readable JSON output.
func PrintJSON(w io.Writer, clones []Clone) {
	wrapx.Fprintln(w, "[")
	for i, clone := range clones {
		wrapx.Fprintf(w, `  {"exact":%v,"similarity":%.2f,"functions":[`,
			clone.Exact, clone.Similarity)
		for j, f := range clone.Funcs {
			if j > 0 {
				wrapx.Fprint(w, ",")
			}
			wrapx.Fprintf(w, `{"name":%q,"file":%q,"line":%d,"stmts":%d}`,
				f.Name, f.File, f.Line, f.NumStmts)
		}
		wrapx.Fprint(w, "]}")
		if i < len(clones)-1 {
			wrapx.Fprintln(w, ",")
		} else {
			wrapx.Fprintln(w)
		}
	}
	wrapx.Fprintln(w, "]")
}

func relativePath(path, cwd string) string {
	if cwd == "" {
		return path
	}
	rel := strings.TrimPrefix(path, cwd+"/")
	if rel == path {
		return path
	}
	return rel
}

// PrintTable writes aligned tabular output suitable for terminal viewing.
// Columns: GROUP  TYPE   SIM   FUNCTION  LOCATION  STMTS  LINES
func PrintTable(w io.Writer, clones []Clone, cwd string) {
	if len(clones) == 0 {
		wrapx.Fprintln(w, "godedup: no structural duplicates found")
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
		typ := "EXACT"
		sim := "100%"
		if !clone.Exact {
			typ = "NEAR"
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
	wrapx.Fprintln(w, fmtRow(headers))

	// separator using only dashes
	sep := ""
	total := widths[0] + widths[1] + widths[2] + widths[3] + widths[4] + widths[5] + widths[6] + 12
	for i := 0; i < total; i++ {
		sep += "-"
	}
	wrapx.Fprintln(w, sep)

	// rows: emit the separator between groups
	prevGroup := ""
	for _, r := range rows {
		if prevGroup != "" && r.group != prevGroup {
			wrapx.Fprintln(w, sep)
		}
		wrapx.Fprintln(w, fmtRow(r))
		prevGroup = r.group
	}
}
