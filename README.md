# godedup

Find structurally duplicate functions in Go code.

[![License](https://img.shields.io/github/license/hashmap-kz/godedup)](https://github.com/hashmap-kz/godedup/blob/master/LICENSE)
[![Go Report Card](https://goreportcard.com/badge/github.com/hashmap-kz/godedup)](https://goreportcard.com/report/github.com/hashmap-kz/godedup)
[![Go Reference](https://pkg.go.dev/badge/github.com/hashmap-kz/godedup.svg)](https://pkg.go.dev/github.com/hashmap-kz/godedup)
[![Workflow Status](https://img.shields.io/github/actions/workflow/status/hashmap-kz/godedup/ci.yml?branch=master)](https://github.com/hashmap-kz/godedup/actions/workflows/ci.yml?query=branch:master)
[![GitHub Issues](https://img.shields.io/github/issues/hashmap-kz/godedup)](https://github.com/hashmap-kz/godedup/issues)
[![Go Version](https://img.shields.io/github/go-mod/go-version/hashmap-kz/godedup)](https://github.com/hashmap-kz/godedup/blob/master/go.mod#L3)
[![Latest Release](https://img.shields.io/github/v/release/hashmap-kz/godedup)](https://github.com/hashmap-kz/godedup/releases/latest)
[![Start contributing](https://img.shields.io/github/issues/hashmap-kz/godedup/good%20first%20issue?color=7057ff&label=Contribute)](https://github.com/hashmap-kz/godedup/issues?q=is%3Aissue+is%3Aopen+sort%3Aupdated-desc+label%3A%22good+first+issue%22)

`godedup` detects copy-pasted Go functions even when variable names, literals, and package calls were changed.

---

## Example

```bash
godedup ./...
```

**HTML output**

![HTML](https://raw.githubusercontent.com/hashmap-kz/assets/main/godedup/godedup-html-v1.png)

**Table output with clickable `file.go:line` locations in supported terminals and editors**

```
$ godedup --output=table --exclude '_test\.go$'
 
GROUP  TYPE   SIM   FUNCTION                     LOCATION                     STMTS  LINES
------------------------------------------------------------------------------------------
1      EXACT  100%  store.(*UserRepo).findByID   internal/store/user.go:84    5      14
1      EXACT  100%  store.(*OrderRepo).findByID  internal/store/order.go:91   5      14
------------------------------------------------------------------------------------------
2      EXACT  100%  api.writeJSON                internal/api/user.go:31      4      9
2      EXACT  100%  api.writeJSON                internal/api/order.go:28     4      9
------------------------------------------------------------------------------------------
3      NEAR   88%   worker.(*EmailJob).validate  internal/worker/email.go:55  7      19
3      NEAR   88%   worker.(*SMSJob).validate    internal/worker/sms.go:48    8      21
```

---

## How It Works

godedup hashes every function's AST structure, normalizing away:

- **Identifier names** - `userID` and `orderID` are treated the same
- **Literal values** - `"users"` and `"orders"` are treated the same
- **Package qualifiers** - `fmt.Println` and `log.Println` are treated the same

Two functions that do the same thing with different variable names, different string literals, and different package
calls hash identically.

**Exact clones** - top-level hash collision: identical control flow and structure.

**Near clones** - edit distance on per-statement hash sequences: same structure with a few added/removed statements.

---

## Install

#### Package

```bash
go install github.com/hashmap-kz/godedup@latest
```

#### Brew

```bash
brew tap hashmap-kz/homebrew-tap
brew install godedup
```

---

## Usage

```
godedup [flags] [path ...]

Examples:
  godedup ./...
  godedup --exact ./...
  godedup --exclude '_test\.go$' --exclude '\.pb\.go$' ./...
  godedup --exclude '(_test|[.]pb|[.]deepcopy)[.]go$' ./...
  godedup --output table ./...
  godedup --output html ./... > godedup.html
  godedup --output json ./... | jq .

Flags:
  --min-similarity  float    minimum similarity threshold (default: 0.85)
  --min-stmts       int      minimum statements to analyze (default: 3)
  --exact                    report only exact structural clones
  --exclude                  exclude files matching regexp (may be repeated)
  --output          string   output format: text, table, html, json (default: text)
  --version                  print version
```

**JSON output** for custom tooling:

```bash
godedup --output=json ./... | jq '.[] | select(.exact) | .functions[].name'
```

---

## Normalization Rules

| What                     | How normalized                        |
|--------------------------|---------------------------------------|
| Variable names           | all -> same token                     |
| String literals          | all -> `""`                           |
| Numeric literals         | all -> `0`                            |
| Package qualifiers       | `fmt.X` and `log.X` -> same structure |
| `nil` / `true` / `false` | preserved (semantic meaning)          |
| Operator types           | preserved (`+` != `-`)                |
| Statement order          | preserved (order matters)             |
| Control flow structure   | preserved (the whole point)           |

---

## Limitations

`godedup` is intentionally simple. It does not try to prove semantic equivalence.

It finds structural duplication, not all possible duplication.

For example, these may not match:

- same behavior implemented with different control flow
- helper functions extracted in only one copy
- loops rewritten as recursion
- code generated by tools, unless you include it in the scanned paths

---

## License

MIT License. See [LICENSE](./LICENSE) for details.
