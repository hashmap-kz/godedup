package report

import (
	"fmt"
	"html"
	"io"
	"sort"
	"strings"

	"github.com/hashmap-kz/godedup/internal/hash"
	"github.com/hashmap-kz/godedup/internal/x/fmtx"
)

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
.fn-row-wrap {
  overflow-x: auto;
}
.fn-row {
  display: flex;
  align-items: stretch;
  width: max-content;
  min-width: 100%;
}
.group.funcs-2 .fn-row {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  width: 100%;
}
.fn-card {
  min-width: 360px;
  border-right: 1px solid var(--border);
  display: flex;
  flex-direction: column;
}
.group.funcs-2 .fn-card { min-width: 0; }
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
.code { flex: 1; }
pre {
  font-family: var(--mono);
  font-size: 12px;
  line-height: 1.5;
  padding: 8px 0;
  white-space: pre;
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
