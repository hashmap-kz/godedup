package hash

// Similarity computes the structural similarity between two functions
// as a value in [0.0, 1.0] where 1.0 is identical.
//
// Uses edit distance on the statement hash sequences - two functions
// are similar if you can transform one's statement list into the other's
// with few insertions/deletions.
func Similarity(a, b *FuncInfo) float64 {
	if len(a.StmtSeq) == 0 && len(b.StmtSeq) == 0 {
		return 1.0
	}
	if len(a.StmtSeq) == 0 || len(b.StmtSeq) == 0 {
		return 0.0
	}

	dist := editDistance(a.StmtSeq, b.StmtSeq)
	maxLen := len(a.StmtSeq)
	if len(b.StmtSeq) > maxLen {
		maxLen = len(b.StmtSeq)
	}

	return 1.0 - float64(dist)/float64(maxLen)
}

// editDistance computes the Levenshtein distance between two uint64 slices.
// Each element represents one statement's structural hash.
func editDistance(a, b []uint64) int {
	la, lb := len(a), len(b)

	// dp[i][j] = edit distance between a[:i] and b[:j]
	dp := make([][]int, la+1)
	for i := range dp {
		dp[i] = make([]int, lb+1)
	}

	for i := 0; i <= la; i++ {
		dp[i][0] = i
	}
	for j := 0; j <= lb; j++ {
		dp[0][j] = j
	}

	for i := 1; i <= la; i++ {
		for j := 1; j <= lb; j++ {
			if a[i-1] == b[j-1] {
				dp[i][j] = dp[i-1][j-1] // exact match
			} else {
				// substitution, deletion, insertion
				dp[i][j] = 1 + min(dp[i-1][j-1], dp[i-1][j], dp[i][j-1])
			}
		}
	}

	return dp[la][lb]
}
