package update

import (
	"fmt"
	"strings"
)

// Diff returns a minimal LCS line diff of a -> b, or "" when equal. Files
// here are small (tens of lines), so the O(n*m) table is fine.
func Diff(path, a, b string) string {
	if a == b {
		return ""
	}
	al, bl := splitLines(a), splitLines(b)
	m, n := len(al), len(bl)
	lcs := make([][]int, m+1)
	for i := range lcs {
		lcs[i] = make([]int, n+1)
	}
	for i := m - 1; i >= 0; i-- {
		for j := n - 1; j >= 0; j-- {
			switch {
			case al[i] == bl[j]:
				lcs[i][j] = lcs[i+1][j+1] + 1
			case lcs[i+1][j] >= lcs[i][j+1]:
				lcs[i][j] = lcs[i+1][j]
			default:
				lcs[i][j] = lcs[i][j+1]
			}
		}
	}
	var sb strings.Builder
	fmt.Fprintf(&sb, "--- %s (on disk)\n+++ %s (regenerated)\n", path, path)
	i, j := 0, 0
	for i < m && j < n {
		switch {
		case al[i] == bl[j]:
			sb.WriteString("  " + al[i] + "\n")
			i++
			j++
		case lcs[i+1][j] >= lcs[i][j+1]:
			sb.WriteString("- " + al[i] + "\n")
			i++
		default:
			sb.WriteString("+ " + bl[j] + "\n")
			j++
		}
	}
	for ; i < m; i++ {
		sb.WriteString("- " + al[i] + "\n")
	}
	for ; j < n; j++ {
		sb.WriteString("+ " + bl[j] + "\n")
	}
	return sb.String()
}
