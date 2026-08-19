package gate

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strconv"
)

// ResultsEnvVar is the environment variable maipipe sets to the path a
// stage writes its results contract to. When it is unset the check prints a
// human table on stdout and writes no file.
const ResultsEnvVar = "MAIPIPE_RESULTS"

// results is the maipipe results contract, one JSON object per stage. Field
// order here is the field order on disk: encoding/json emits struct fields
// in declaration order, which together with sortFindings makes the file
// deterministic.
type results struct {
	MaipipeResults int           `json:"maipipe_results"`
	Status         string        `json:"status"`
	Summary        string        `json:"summary"`
	Findings       []jsonFinding `json:"findings"`
}

type jsonFinding struct {
	Severity string `json:"severity"`
	Message  string `json:"message"`
	File     string `json:"file"`
	Line     int    `json:"line"`
	Code     string `json:"code"`
}

// emit reports one check class's outcome: the results-contract JSON to the
// path in MAIPIPE_RESULTS when that variable is set, otherwise a human table
// on stdout and no file. Every check class shares this one emitter. A class
// that owns its own judgement (Report.Advisory, Report.Summary) overrides
// the pack defaults here; Report.Detail is human-only.
func emit(check string, rep Report, stdout io.Writer) error {
	findings := rep.Findings
	status, summary := "pass", fmt.Sprintf("%s: no findings", Code(check))
	if len(findings) > 0 && !rep.Advisory {
		status = "fail"
		summary = fmt.Sprintf("%s: %d finding(s)", Code(check), len(findings))
	}
	if rep.Summary != "" {
		summary = rep.Summary
	}
	if path, ok := os.LookupEnv(ResultsEnvVar); ok {
		out := results{MaipipeResults: 0, Status: status, Summary: summary, Findings: []jsonFinding{}}
		for _, f := range findings {
			out.Findings = append(out.Findings, jsonFinding{f.Severity, f.Message, f.File, f.Line, f.Code})
		}
		b, err := json.MarshalIndent(out, "", "  ")
		if err != nil {
			return err
		}
		if err := os.WriteFile(path, append(b, '\n'), 0o644); err != nil {
			return fmt.Errorf("writing results to %s (%s): %w", path, ResultsEnvVar, err)
		}
		return nil
	}
	writeTable(stdout, findings)
	fmt.Fprintln(stdout, summary)
	for _, line := range rep.Detail {
		fmt.Fprintln(stdout, line)
	}
	return nil
}

// writeTable prints the human fallback: one row per finding, columns sized
// to content (the same shape as spine's other report tables).
func writeTable(w io.Writer, findings []Finding) {
	if len(findings) == 0 {
		return
	}
	wSev, wFile, wLine, wCode := len("severity"), len("file"), len("line"), len("code")
	for _, f := range findings {
		wSev = max(wSev, len(f.Severity))
		wFile = max(wFile, len(f.File))
		wLine = max(wLine, len(strconv.Itoa(f.Line)))
		wCode = max(wCode, len(f.Code))
	}
	fmt.Fprintf(w, "%-*s  %-*s  %-*s  %-*s  %s\n", wSev, "severity", wFile, "file", wLine, "line", wCode, "code", "message")
	for _, f := range findings {
		fmt.Fprintf(w, "%-*s  %-*s  %-*d  %-*s  %s\n", wSev, f.Severity, wFile, f.File, wLine, f.Line, wCode, f.Code, f.Message)
	}
}
