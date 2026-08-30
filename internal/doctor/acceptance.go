package doctor

import "github.com/russellpope/spine/internal/acceptance"

func acceptanceCheck(dir string) []Finding {
	summary := acceptance.ScanAllTickets(dir)
	findings := make([]Finding, 0, len(summary.Problems)+len(summary.ScanErrors))
	for _, problem := range summary.Problems {
		findings = append(findings, Finding{"D15", "warn", problem.Path, problem.Message()})
	}
	for _, scanErr := range summary.ScanErrors {
		findings = append(findings, Finding{"D15", "warn", scanErr.Path, scanErr.Message()})
	}
	return findings
}
