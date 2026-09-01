package yield

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"sort"
	"strings"
)

// runFleet scans only immediate non-hidden primary repositories. A `.git`
// file identifies a linked worktree and is deliberately not a fleet member.
func runFleet(parent string) (Report, error) {
	return runFleetWithOps(parent, fleetOps{})
}

type fleetOps struct {
	parentLstat            func(string) (fs.FileInfo, error)
	openRoot               func(string) (*os.Root, error)
	childGitLstat          func(*os.Root, string, string) (fs.FileInfo, error)
	ledgerOps              ledgerOps
	afterParentObserved    func()
	afterChildGitObserved  func(string)
	beforeParentRevalidate func()
}

func runFleetWithOps(parent string, ops fleetOps) (report Report, err error) {
	if ops.parentLstat == nil {
		ops.parentLstat = os.Lstat
	}
	if ops.openRoot == nil {
		ops.openRoot = os.OpenRoot
	}
	if ops.childGitLstat == nil {
		ops.childGitLstat = func(root *os.Root, _ string, name string) (fs.FileInfo, error) { return root.Lstat(name) }
	}

	info, err := ops.parentLstat(parent)
	if err != nil || info.Mode()&fs.ModeSymlink != 0 || !info.IsDir() {
		return Report{}, fmt.Errorf("%w: --fleet", ErrInvalidRoot)
	}
	if ops.afterParentObserved != nil {
		ops.afterParentObserved()
	}
	root, err := ops.openRoot(parent)
	if err != nil {
		return Report{}, fmt.Errorf("%w: --fleet", ErrInvalidRoot)
	}
	defer func() {
		if closeErr := root.Close(); err == nil && closeErr != nil {
			report, err = Report{}, fmt.Errorf("%w: --fleet", ErrInvalidRoot)
		}
	}()
	if !fleetRootMatches(parent, info, root, ops.parentLstat) {
		return Report{}, fmt.Errorf("%w: --fleet", ErrInvalidRoot)
	}
	entries, readErr := readFleetEntries(root)
	if readErr != nil {
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
		local, status, inspectionFailure := inspectFleetChild(root, entry.Name(), ops)
		if status.Name == "" {
			continue
		}
		statuses = append(statuses, status)
		if inspectionFailure {
			inspectionFailures[status.Name] = true
		}
		if status.Status == "ok" {
			combined.merge(local)
		}
	}
	if ops.beforeParentRevalidate != nil {
		ops.beforeParentRevalidate()
	}
	if !fleetRootMatches(parent, info, root, ops.parentLstat) {
		return Report{}, fmt.Errorf("%w: --fleet", ErrInvalidRoot)
	}
	report = finalize(combined, "fleet")
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
	report.sortDiagnostics()
	return report, nil
}

func fleetRootMatches(parent string, observed fs.FileInfo, root *os.Root, lstat func(string) (fs.FileInfo, error)) bool {
	opened, openedErr := root.Stat(".")
	current, currentErr := lstat(parent)
	return openedErr == nil && currentErr == nil && current.Mode()&fs.ModeSymlink == 0 && opened.IsDir() && os.SameFile(observed, opened) && os.SameFile(observed, current)
}

func inspectFleetChild(parent *os.Root, name string, ops fleetOps) (localResult, RepositoryStatus, bool) {
	observed, err := parent.Lstat(name)
	if err != nil || observed.Mode()&fs.ModeSymlink != 0 || !observed.IsDir() {
		return localResult{}, RepositoryStatus{Name: name, Status: "error"}, false
	}
	child, err := parent.OpenRoot(name)
	if err != nil {
		return localResult{}, RepositoryStatus{Name: name, Status: "error"}, false
	}
	if !fleetChildMatches(parent, child, name, observed) {
		_ = child.Close()
		return localResult{}, RepositoryStatus{Name: name, Status: "error"}, false
	}
	git, gitErr := ops.childGitLstat(child, name, ".git")
	if gitErr != nil {
		childMatches := fleetChildMatches(parent, child, name, observed)
		closeErr := child.Close()
		if errors.Is(gitErr, fs.ErrNotExist) && closeErr == nil && childMatches {
			return localResult{}, RepositoryStatus{}, false
		}
		return localResult{}, RepositoryStatus{Name: name, Status: "error"}, !errors.Is(gitErr, fs.ErrNotExist)
	}
	if !git.IsDir() || git.Mode()&fs.ModeSymlink != 0 {
		childMatches := fleetChildMatches(parent, child, name, observed)
		closeErr := child.Close()
		if closeErr == nil && childMatches {
			return localResult{}, RepositoryStatus{}, false
		}
		return localResult{}, RepositoryStatus{Name: name, Status: "error"}, false
	}
	if ops.afterChildGitObserved != nil {
		ops.afterChildGitObserved(name)
	}
	var local localResult
	var status RepositoryStatus
	if ops.ledgerOps.beforeOpen == nil && ops.ledgerOps.afterRead == nil {
		local, status = readRepositoryRoot(child, name)
	} else {
		local, status = readRepositoryRootWithLedgerOps(child, name, ops.ledgerOps)
	}
	childMatches := fleetChildMatches(parent, child, name, observed)
	currentGit, currentGitErr := child.Lstat(".git")
	gitMatches := currentGitErr == nil && currentGit.Mode()&fs.ModeSymlink == 0 && currentGit.IsDir() && os.SameFile(git, currentGit)
	closeErr := child.Close()
	if !childMatches || !gitMatches || closeErr != nil {
		return localResult{}, RepositoryStatus{Name: name, Status: "error"}, false
	}
	return local, status, false
}

func fleetChildMatches(parent *os.Root, child *os.Root, name string, observed fs.FileInfo) bool {
	opened, openedErr := child.Stat(".")
	current, currentErr := parent.Lstat(name)
	return openedErr == nil && currentErr == nil && current.Mode()&fs.ModeSymlink == 0 && opened.IsDir() && os.SameFile(observed, opened) && os.SameFile(observed, current)
}

func readFleetEntries(root *os.Root) ([]os.DirEntry, error) {
	dir, err := root.Open(".")
	if err != nil {
		return nil, err
	}
	entries, readErr := dir.ReadDir(-1)
	closeErr := dir.Close()
	if readErr != nil {
		return nil, readErr
	}
	if closeErr != nil {
		return nil, closeErr
	}
	return entries, nil
}
