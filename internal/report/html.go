package report

import (
	"fmt"
	"html/template"
	"io"
	"sort"
	"strings"

	"github.com/hashmap-kz/godedup/internal/hash"
	"github.com/hashmap-kz/godedup/internal/x/fmtx"
)

// htmlLine is one source line within a function card.
type htmlLine struct {
	No      int
	FileURL template.URL
	Text    string
}

// htmlFuncView is the template data for a single function card.
type htmlFuncView struct {
	Name     string
	Location string
	FileURL  template.URL
	NumStmts int
	NumLines int
	Lines    []htmlLine
}

// htmlGroupView is the template data for one clone group.
type htmlGroupView struct {
	No        int
	KindClass string // "exact" or "near"
	Kind      string // "EXACT" or "NEAR"
	Sim       string
	IsTwoFunc bool
	Funcs     []htmlFuncView
}

// htmlReportView is the top-level template data.
type htmlReportView struct {
	Groups  []htmlGroupView
	Total   int
	Exact   int
	Near    int
	FnCount int
}

const htmlCSS = `*, *::before, *::after { box-sizing: border-box; margin: 0; padding: 0; }
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
.empty-msg { padding: 32px; text-align: center; color: var(--muted); }`

const htmlTmpl = `<!doctype html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>godedup report</title>
<style>{{.CSS}}</style>
</head>
<body>
<div class="hdr">
  <div>
    <div class="hdr-title">godedup report</div>
    <div class="hdr-sub">Structural duplicate detection for Go</div>
  </div>
  <div class="tags">
    <span class="tag">{{.Data.Total}} groups</span>
    <span class="tag">{{.Data.FnCount}} functions</span>
    <span class="tag tag-e">{{.Data.Exact}} exact</span>
    <span class="tag tag-n">{{.Data.Near}} near</span>
  </div>
</div>
{{- if not .Data.Groups}}
<div class="empty-msg">No structural duplicates found.</div>
{{- else}}
{{- range .Data.Groups}}
<article class="group {{.KindClass}}{{if .IsTwoFunc}} funcs-2{{end}}">
  <div class="group-hdr">
    <span class="group-num">#{{.No}}</span>
    <span class="badge {{.KindClass}}">{{.Kind}}</span>
    <span class="group-sim">{{.Sim}}</span>
    <span class="group-meta">{{len .Funcs}} functions</span>
  </div>
  <div class="fn-row-wrap"><div class="fn-row">
    {{- range .Funcs}}
    <section class="fn-card">
      <div class="fn-card-hdr">
        <div class="fn-name">{{.Name}}</div>
        <div class="fn-loc"><a href="{{.FileURL}}">{{.Location}}</a></div>
        <div class="fn-stat">{{.NumStmts}} stmts &middot; {{.NumLines}} lines</div>
      </div>
      <div class="code"><pre>{{- range .Lines}}<div class="code-line"><a class="line-no" href="{{.FileURL}}">{{.No}}</a><span class="code-text">{{.Text}}</span></div>{{end}}</pre></div>
    </section>
    {{- end}}
  </div></div>
</article>
{{- end}}
{{- end}}
</body>
</html>`

var htmlReport = template.Must(
	template.New("report").
		Funcs(template.FuncMap{
			"not": func(groups []htmlGroupView) bool { return len(groups) == 0 },
		}).
		Parse(htmlTmpl),
)

// PrintHTML writes a self-contained HTML report.
func PrintHTML(w io.Writer, clones []Clone, cwd string) {
	exact, near, fnCount := cloneStats(clones)

	data := htmlReportView{
		Total:   len(clones),
		Exact:   exact,
		Near:    near,
		FnCount: fnCount,
		Groups:  make([]htmlGroupView, 0, len(clones)),
	}
	for i, clone := range clones {
		data.Groups = append(data.Groups, buildHTMLGroup(i+1, clone, cwd))
	}

	err := htmlReport.Execute(w, struct {
		CSS  template.CSS
		Data htmlReportView
	}{
		CSS:  htmlCSS,
		Data: data,
	})
	if err != nil {
		fmtx.Fprintf(w, "<!-- template error: %v -->\n", err)
	}
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

func buildHTMLGroup(no int, clone Clone, cwd string) htmlGroupView {
	kind := "EXACT"
	sim := "100%"
	kindClass := "exact"
	if !clone.Exact {
		kind = "NEAR"
		sim = fmt.Sprintf("%.0f%%", clone.Similarity*100)
		kindClass = "near"
	}
	sorted := sortedFuncs(clone.Funcs)
	funcs := make([]htmlFuncView, 0, len(sorted))
	for _, f := range sorted {
		funcs = append(funcs, buildHTMLFunc(f, cwd))
	}
	return htmlGroupView{
		No:        no,
		KindClass: kindClass,
		Kind:      kind,
		Sim:       sim,
		IsTwoFunc: len(sorted) == 2,
		Funcs:     funcs,
	}
}

func buildHTMLFunc(f hash.FuncInfo, cwd string) htmlFuncView {
	loc := fmt.Sprintf("%s:%d", relativePath(f.File, cwd), f.Line)
	src := f.Source
	if src == "" {
		src = "(source unavailable)"
	}
	rawLines := strings.Split(src, "\n")
	lines := make([]htmlLine, 0, len(rawLines))
	for i, text := range rawLines {
		lineNo := f.Line + i
		lines = append(lines, htmlLine{
			No:      lineNo,
			FileURL: template.URL(fileURL(f.File, lineNo)),
			Text:    text,
		})
	}
	return htmlFuncView{
		Name:     f.Name,
		Location: loc,
		FileURL:  template.URL(fileURL(f.File, f.Line)),
		NumStmts: f.NumStmts,
		NumLines: f.NumLines,
		Lines:    lines,
	}
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
