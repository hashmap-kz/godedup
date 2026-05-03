package main

import (
	"flag"
	"fmt"
	"os"

	"godedup/internal/wrapx"

	"godedup/internal/load"
	"godedup/internal/report"
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
  godedup --min-similarity 0.90 ./pkg/...
  godedup --exact ./...
  godedup --json ./... | jq .

Flags:
`

func main() {
	minSim := flag.Float64("min-similarity", 0.85, "minimum similarity threshold (0.0-1.0)")
	minStmts := flag.Int("min-stmts", 3, "minimum statements in a function to analyze")
	exactOnly := flag.Bool("exact", false, "report only exact structural clones")
	noTests := flag.Bool("no-tests", false, "exclude test files")
	jsonOut := flag.Bool("json", false, "output JSON instead of human-readable text")
	tableOut := flag.Bool("table", false, "output aligned table")
	showVer := flag.Bool("version", false, "print version and exit")

	flag.Usage = func() {
		wrapx.Fprint(os.Stderr, usage)
		flag.PrintDefaults()
		wrapx.Fprintln(os.Stderr)
	}
	flag.Parse()

	if *showVer {
		fmt.Printf("godedup %s\n", Version)
		os.Exit(0)
	}

	paths := flag.Args()
	if len(paths) == 0 {
		paths = []string{"."}
	}

	// expand ./... pattern
	paths = expandPaths(paths)

	result, err := load.Load(paths, *noTests)
	if err != nil {
		wrapx.Fprintf(os.Stderr, "godedup: load error: %v\n", err)
		os.Exit(1)
	}

	if len(result.Funcs) == 0 {
		wrapx.Fprintln(os.Stderr, "godedup: no functions found")
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
		fmt.Printf("cannot get cwd: %v", err)
		os.Exit(2)
	}

	switch {
	case *jsonOut:
		report.PrintJSON(os.Stdout, clones)
	case *tableOut:
		report.PrintTable(os.Stdout, clones, cwd)
	default:
		report.Print(os.Stdout, clones, cwd)
	}

	// exit 1 if clones found (useful for CI)
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
