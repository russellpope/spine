package yield

import (
	"errors"
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
	return runFleetWithLstat(parent, os.Lstat)
}

func runFleetWithLstat(parent string, lstat func(string) (fs.FileInfo, error)) (Report, error) {
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
	inspectionFailures := map[string]bool{}
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), ".") || !entry.IsDir() || entry.Type()&fs.ModeSymlink != 0 {
			continue
		}
		child := filepath.Join(parent, entry.Name())
		gitInfo, gitErr := lstat(filepath.Join(child, ".git"))
		if gitErr != nil {
			if errors.Is(gitErr, fs.ErrNotExist) {
				continue
			}
			statuses = append(statuses, RepositoryStatus{Name: entry.Name(), Status: "error"})
			inspectionFailures[entry.Name()] = true
			continue
		}
		if !gitInfo.IsDir() || gitInfo.Mode()&fs.ModeSymlink != 0 {
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
			message := "progress ledger unreadable"
			if inspectionFailures[status.Name] {
				message = "repository inspection failed"
			}
			report.Diagnostics = append(report.Diagnostics, Diagnostic{Repository: status.Name, Message: message})
		}
	}
	return report, nil
}
