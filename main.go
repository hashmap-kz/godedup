package main

import (
	"flag"
	"fmt"
	"os"
	"regexp"

	"github.com/hashmap-kz/godedup/internal/cmd"
	"github.com/hashmap-kz/godedup/internal/load"
	"github.com/hashmap-kz/godedup/internal/report"
	"github.com/hashmap-kz/godedup/internal/x/fmtx"
)

var Version = "dev"

var usage = `godedup - find structurally duplicate functions in Go code

Detects copy-pasted functions that have been superficially modified:
different variable names, different literals, different package calls -
but identical control flow and structure.

Usage:
  godedup [flags] [path ...]

Examples:
  godedup ./...
  godedup --exact ./...
  godedup --exclude '_test\.go$' ./...
  godedup --exclude '_test\.go$|\.pb\.go$|\.deepcopy\.go$' ./...
  godedup --exclude '(_test|[.]pb|[.]deepcopy)[.]go$' ./...
  godedup --output table ./...
  godedup --output json ./... | jq .

Flags:
`

func main() {
	minSim := flag.Float64("min-similarity", 0.85, "minimum similarity threshold (0.0-1.0)")
	minStmts := flag.Int("min-stmts", 3, "minimum statements in a function to analyze")
	exactOnly := flag.Bool("exact", false, "report only exact structural clones")
	excludePat := flag.String("exclude", "", "exclude files matching this regular expression (matched against full path)")
	output := flag.String("output", "text", "output format: text, table, json")
	showVer := flag.Bool("version", false, "print version and exit")

	flag.Usage = func() {
		fmtx.Fprint(os.Stderr, usage)
		flag.PrintDefaults()
		fmtx.Fprintln(os.Stderr)
	}
	flag.Parse()

	if *showVer {
		fmt.Printf("godedup %s\n", Version)
		os.Exit(0)
	}

	switch *output {
	case "text", "table", "json":
		// valid
	default:
		fmtx.Fprintf(os.Stderr, "godedup: unknown output format %q (want: text, table, json)\n", *output)
		os.Exit(1)
	}

	var exclude *regexp.Regexp
	if *excludePat != "" {
		var err error
		exclude, err = regexp.Compile(*excludePat)
		if err != nil {
			fmtx.Fprintf(os.Stderr, "godedup: invalid --exclude pattern: %v\n", err)
			os.Exit(1)
		}
	}

	paths := flag.Args()
	if len(paths) == 0 {
		paths = []string{"."}
	}

	paths = expandPaths(paths)

	result, err := load.Load(paths, &cmd.LoadInput{
		ExcludeRegex: exclude,
	})
	if err != nil {
		fmtx.Fprintf(os.Stderr, "godedup: load error: %v\n", err)
		os.Exit(1)
	}

	if len(result.Funcs) == 0 {
		fmtx.Fprintln(os.Stderr, "godedup: no functions found")
		os.Exit(0)
	}

	cfg := report.Config{
		MinSimilarity: *minSim,
		MinStmts:      *minStmts,
		ExactOnly:     *exactOnly,
	}

	clones := report.Detect(result.Funcs, cfg)

	cwd, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(os.Stderr, "godedup: cannot get cwd: %v\n", err)
		os.Exit(2)
	}

	switch *output {
	case "json":
		report.PrintJSON(os.Stdout, clones)
	case "table":
		report.PrintTable(os.Stdout, clones, cwd)
	default:
		report.Print(os.Stdout, clones, cwd)
	}

	if len(clones) > 0 {
		os.Exit(1)
	}
}

// expandPaths handles the ./... pattern by walking from the given root.
func expandPaths(patterns []string) []string {
	var result []string
	for _, p := range patterns {
		if p == "./..." || p == "..." {
			result = append(result, ".")
		} else {
			// strip trailing /... - load.Load walks recursively anyway
			p = trimSuffix(p, "/...")
			result = append(result, p)
		}
	}
	return result
}

func trimSuffix(s, suffix string) string {
	if len(s) >= len(suffix) && s[len(s)-len(suffix):] == suffix {
		return s[:len(s)-len(suffix)]
	}
	return s
}
