package report

import (
	"fmt"
	"html"
	"io"
	"sort"
	"strings"

	"github.com/hashmap-kz/godedup/internal/x/fmtx"

	"github.com/hashmap-kz/godedup/internal/hash"
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

// PrintHTML writes a self-contained HTML report.
func PrintHTML(w io.Writer, clones []Clone, cwd string) {
	exact, near, fnCount := cloneStats(clones)

	fmtx.Fprint(w, "<!doctype html>\n<html lang=\"en\">\n<head>\n<meta charset=\"utf-8\">\n<meta name=\"viewport\" content=\"width=device-width, initial-scale=1\">\n<title>godedup report</title>\n<style>\n")
	fmtx.Fprint(w, `*, *::before, *::after { box-sizing: border-box; margin: 0; padding: 0; }
:root {
  --bg:     #f6f8fa;
  --card:   #ffffff;
  --border: #d0d7de;
  --text:   #24292f;
  --muted:  #57606a;
  --code-bg:#f6f8fa;
  --blue:   #0969da;
  --purple: #8250df;
  --mono: ui-monospace,SFMono-Regular,Menlo,Monaco,Consolas,monospace;
}
body {
  background: var(--bg);
  color: var(--text);
  font-family: system-ui,-apple-system,BlinkMacSystemFont,"Segoe UI",sans-serif;
  font-size: 14px;
  line-height: 1.5;
  padding: 20px;
}
a { color: var(--blue); text-decoration: none; }
a:hover { text-decoration: underline; }
.hdr {
  background: var(--card);
  border: 1px solid var(--border);
  border-radius: 8px;
  padding: 14px 20px;
  margin-bottom: 14px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  flex-wrap: wrap;
}
.hdr-title { font-size: 17px; font-weight: 700; }
.hdr-sub   { color: var(--muted); font-size: 12px; margin-top: 2px; }
.tags      { display: flex; gap: 6px; flex-wrap: wrap; align-items: center; }
.tag {
  border: 1px solid var(--border);
  border-radius: 999px;
  padding: 2px 10px;
  font-size: 12px;
  white-space: nowrap;
}
.tag-e { color: var(--blue);   border-color: #b6d4fe; background: #dbeafe; }
.tag-n { color: var(--purple); border-color: #d8b4fe; background: #ede9fe; }
.group {
  background: var(--card);
  border: 1px solid var(--border);
  border-radius: 8px;
  margin-bottom: 10px;
  overflow: hidden;
}
.group-hdr {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 14px;
  border-bottom: 1px solid var(--border);
  background: var(--code-bg);
  border-left: 3px solid transparent;
}
.group.exact .group-hdr { border-left-color: var(--blue); }
.group.near  .group-hdr { border-left-color: var(--purple); }
.badge {
  border-radius: 4px;
  padding: 1px 6px;
  font-size: 11px;
  font-weight: 700;
  font-family: var(--mono);
}
.badge.exact { color: var(--blue);   border: 1px solid #b6d4fe; background: #dbeafe; }
.badge.near  { color: var(--purple); border: 1px solid #d8b4fe; background: #ede9fe; }
.group-num  { font-family: var(--mono); font-size: 11px; color: var(--muted); }
.group-sim  { font-family: var(--mono); font-size: 12px; }
.group-meta { margin-left: auto; font-size: 12px; color: var(--muted); }
.fn-row-wrap { overflow-x: auto; }
.fn-row {
  display: flex;
  min-width: 100%;
  width: max-content;
  align-items: stretch;
}
.group.funcs-2 .fn-row-wrap { overflow-x: visible; }
.group.funcs-2 .fn-row {
  width: 100%;
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
}
.fn-card {
  min-width: 320px;
  border-right: 1px solid var(--border);
  display: flex;
  flex-direction: column;
  flex: 1 1 0;
}
.fn-card:last-child { border-right: none; }
.fn-card-hdr {
  padding: 7px 12px;
  border-bottom: 1px solid var(--border);
  background: var(--code-bg);
  flex-shrink: 0;
}
.fn-name { font-weight: 600; font-size: 13px; word-break: break-all; }
.fn-loc  { font-size: 11px; color: var(--muted); margin-top: 2px; }
.fn-stat { font-size: 11px; color: var(--muted); margin-top: 1px; }
.code    { overflow-x: auto; flex: 1; }
pre {
  font-family: var(--mono);
  font-size: 12px;
  line-height: 1.5;
  padding: 8px 0;
  white-space: pre;
  min-width: max-content;
}
.code-line { display: flex; }
.code-line:hover { background: rgba(0,0,0,.04); }
.line-no {
  color: #6e7781;
  text-align: right;
  padding: 0 12px;
  user-select: none;
  min-width: 48px;
  flex-shrink: 0;
}
.code-text { padding-right: 16px; }
.empty-msg { padding: 32px; text-align: center; color: var(--muted); }
`)
	fmtx.Fprint(w, "</style>\n</head>\n<body>\n")

	fmtx.Fprintf(w, `<div class="hdr">
  <div><div class="hdr-title">godedup report</div><div class="hdr-sub">Structural duplicate detection for Go</div></div>
  <div class="tags">
    <span class="tag">%d groups</span>
    <span class="tag">%d functions</span>
    <span class="tag tag-e">%d exact</span>
    <span class="tag tag-n">%d near</span>
  </div>
</div>
`, len(clones), fnCount, exact, near)

	if len(clones) == 0 {
		fmtx.Fprint(w, "<div class=\"empty-msg\">No structural duplicates found.</div>\n")
		fmtx.Fprint(w, "</body>\n</html>\n")
		return
	}

	for i, clone := range clones {
		writeHTMLCloneGroup(w, i+1, clone, cwd)
	}

	fmtx.Fprint(w, "</body>\n</html>\n")
}

func cloneStats(clones []Clone) (exact int, near int, funcs int) {
	for _, c := range clones {
		funcs += len(c.Funcs)
		if c.Exact {
			exact++
		} else {
			near++
		}
	}
	return exact, near, funcs
}

func writeHTMLCloneGroup(w io.Writer, groupNo int, clone Clone, cwd string) {
	kind := "EXACT"
	sim := "100%"
	kindClass := "exact"
	if !clone.Exact {
		kind = "NEAR"
		sim = fmt.Sprintf("%.0f%%", clone.Similarity*100)
		kindClass = "near"
	}

	sorted := sortedFuncs(clone.Funcs)
	groupClass := "group " + kindClass
	if len(sorted) == 2 {
		groupClass += " funcs-2"
	}

	fmtx.Fprintf(w, "<article class=\"%s\">\n", groupClass)
	fmtx.Fprintf(w, "  <div class=\"group-hdr\">\n")
	fmtx.Fprintf(w, "    <span class=\"group-num\">#%d</span>\n", groupNo)
	fmtx.Fprintf(w, "    <span class=\"badge %s\">%s</span>\n", kindClass, kind)
	fmtx.Fprintf(w, "    <span class=\"group-sim\">%s</span>\n", sim)
	fmtx.Fprintf(w, "    <span class=\"group-meta\">%d functions</span>\n", len(sorted))
	fmtx.Fprint(w, "  </div>\n")
	fmtx.Fprint(w, "  <div class=\"fn-row-wrap\"><div class=\"fn-row\">\n")

	for _, f := range sorted {
		writeHTMLFunctionCard(w, f, cwd)
	}

	fmtx.Fprint(w, "  </div></div>\n</article>\n")
}

func writeHTMLFunctionCard(w io.Writer, f hash.FuncInfo, cwd string) {
	loc := fmt.Sprintf("%s:%d", relativePath(f.File, cwd), f.Line)
	fmtx.Fprint(w, "    <section class=\"fn-card\">\n")
	fmtx.Fprintf(w, "      <div class=\"fn-card-hdr\">\n")
	fmtx.Fprintf(w, "        <div class=\"fn-name\">%s</div>\n", html.EscapeString(f.Name))
	fmtx.Fprintf(w, "        <div class=\"fn-loc\"><a href=\"%s\">%s</a></div>\n",
		html.EscapeString(fileURL(f.File, f.Line)), html.EscapeString(loc))
	fmtx.Fprintf(w, "        <div class=\"fn-stat\">%d stmts &middot; %d lines</div>\n", f.NumStmts, f.NumLines)
	fmtx.Fprint(w, "      </div>\n")
	fmtx.Fprint(w, "      <div class=\"code\"><pre>")

	lines := strings.Split(f.Source, "\n")
	if f.Source == "" {
		lines = []string{"(source unavailable)"}
	}
	for i, line := range lines {
		lineNo := f.Line + i
		fmtx.Fprintf(w, "<div class=\"code-line\"><a class=\"line-no\" href=\"%s\">%d</a><span class=\"code-text\">%s</span></div>",
			html.EscapeString(fileURL(f.File, lineNo)), lineNo, html.EscapeString(line))
	}

	fmtx.Fprint(w, "</pre></div>\n    </section>\n")
}

func fileURL(path string, line int) string {
	return fmt.Sprintf("file://%s:%d", path, line)
}

func sortedFuncs(funcs []hash.FuncInfo) []hash.FuncInfo {
	sorted := make([]hash.FuncInfo, len(funcs))
	copy(sorted, funcs)
	sort.Slice(sorted, func(a, b int) bool {
		if sorted[a].File != sorted[b].File {
			return sorted[a].File < sorted[b].File
		}
		return sorted[a].Line < sorted[b].Line
	})
	return sorted
}

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
