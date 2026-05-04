package report

import (
	"sort"
	"strings"

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
