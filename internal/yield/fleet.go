package yield

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// runFleet scans only immediate non-hidden primary repositories. A `.git`
// file identifies a linked worktree and is deliberately not a fleet member.
func runFleet(parent string) (Report, error) {
	info, err := os.Stat(parent)
	if err != nil || !info.IsDir() {
		return Report{}, fmt.Errorf("%w: --fleet", ErrInvalidRoot)
	}
	entries, err := os.ReadDir(parent)
	if err != nil {
		return Report{}, fmt.Errorf("%w: --fleet", ErrInvalidRoot)
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Name() < entries[j].Name() })
	combined := localResult{}
	statuses := make([]RepositoryStatus, 0)
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), ".") || !entry.IsDir() || entry.Type()&fs.ModeSymlink != 0 {
			continue
		}
		child := filepath.Join(parent, entry.Name())
		gitInfo, gitErr := os.Lstat(filepath.Join(child, ".git"))
		if gitErr != nil || !gitInfo.IsDir() || gitInfo.Mode()&fs.ModeSymlink != 0 {
			continue
		}
		local, status := readRepository(child, entry.Name())
		statuses = append(statuses, status)
		combined.merge(local)
	}
	report := finalize(combined, "fleet")
	report.Repositories = statuses
	for _, status := range statuses {
		if status.Status == "error" {
			report.childError = true
			report.Diagnostics = append(report.Diagnostics, Diagnostic{Repository: status.Name, Message: "progress ledger unreadable"})
		}
	}
	return report, nil
}
