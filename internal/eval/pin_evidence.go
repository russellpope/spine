package eval

import (
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/russellpope/spine/internal/meta"
)

// PinEvidencePin is the small host-independent input needed to evaluate a
// pin's selected local eval references.
type PinEvidencePin struct {
	Key          string
	Model        string
	EvidenceRefs []string
}

// PinEvidenceKind is a sanitized selected-evidence result. It deliberately
// carries no source error, reference, model ID, or file body.
type PinEvidenceKind string

const (
	PinEvidenceNoReference   PinEvidenceKind = "no-reference"
	PinEvidenceBadReference  PinEvidenceKind = "bad-reference"
	PinEvidenceMissing       PinEvidenceKind = "missing"
	PinEvidenceMalformed     PinEvidenceKind = "malformed"
	PinEvidenceStale         PinEvidenceKind = "stale"
	PinEvidenceModelMismatch PinEvidenceKind = "model-mismatch"
	PinEvidenceNoBattery     PinEvidenceKind = "no-battery"
	PinEvidenceFailedBattery PinEvidenceKind = "failed-battery"
)

// PinEvidenceFinding is a safe outcome for doctor. Path is always logical and
// repository-relative, except routing-host.json for reference grammar errors.
type PinEvidenceFinding struct {
	PinKey string
	Kind   PinEvidenceKind
	Path   string
}

var pinEvidenceReference = regexp.MustCompile(`^eval:(\d{4}-\d{2}-\d{2}-[a-z0-9]+(?:-[a-z0-9]+)*)/runs/([A-Za-z0-9][A-Za-z0-9_-]*)\.md$`)
var pinEvidenceDate = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)

var pinEvidenceBatteryKeys = []string{
	"invocation", "wiring", "flag-honoured", "column-presence", "column-order",
	"ordering", "units-labels", "security-default", "lifecycle", "error-path-behaviour",
}

// CheckPinEvidence reads only the selected repo-local runs. It neither walks
// docs/evals nor invokes generic eval listing, so unrelated evals have no
// bearing on a pin's evidence result.
func CheckPinEvidence(repoDir string, pins []PinEvidencePin, today time.Time) []PinEvidenceFinding {
	sortedPins := append([]PinEvidencePin(nil), pins...)
	sort.Slice(sortedPins, func(i, j int) bool { return sortedPins[i].Key < sortedPins[j].Key })
	var findings []PinEvidenceFinding
	for _, pin := range sortedPins {
		refs := make([]string, 0, len(pin.EvidenceRefs))
		for _, ref := range pin.EvidenceRefs {
			if strings.HasPrefix(ref, "eval:") {
				refs = append(refs, ref)
			}
		}
		if len(refs) == 0 {
			findings = append(findings, PinEvidenceFinding{PinKey: pin.Key, Kind: PinEvidenceNoReference, Path: "routing-host.json"})
			continue
		}
		sort.Strings(refs)
		for _, ref := range refs {
			kind, path := checkPinEvidenceReference(repoDir, pin.Model, ref, today)
			if kind != "" {
				findings = append(findings, PinEvidenceFinding{PinKey: pin.Key, Kind: kind, Path: path})
			}
		}
	}
	return findings
}

func checkPinEvidenceReference(repoDir, model, ref string, today time.Time) (PinEvidenceKind, string) {
	match := pinEvidenceReference.FindStringSubmatch(ref)
	if match == nil || !validCalendarDate(match[1][:10]) {
		return PinEvidenceBadReference, "routing-host.json"
	}
	evalDir, runName := match[1], match[2]
	runPath := filepath.ToSlash(filepath.Join("docs", "evals", evalDir, "runs", runName+".md"))
	evalPath := filepath.ToSlash(filepath.Join("docs", "evals", evalDir, "eval.md"))
	for _, rel := range []string{"docs", filepath.Join("docs", "evals"), filepath.Join("docs", "evals", evalDir), filepath.Join("docs", "evals", evalDir, "runs")} {
		if kind := checkedDirectory(filepath.Join(repoDir, rel)); kind != "" {
			if kind == PinEvidenceMissing {
				return kind, runPath
			}
			return kind, runPath
		}
	}
	if kind := checkedRegularFile(filepath.Join(repoDir, "docs", "evals", evalDir, "eval.md")); kind != "" {
		return kind, evalPath
	}
	if kind := checkedRegularFile(filepath.Join(repoDir, "docs", "evals", evalDir, "runs", runName+".md")); kind != "" {
		return kind, runPath
	}
	parent, err := os.ReadFile(filepath.Join(repoDir, "docs", "evals", evalDir, "eval.md"))
	if err != nil || !hasRequiredFrontMatter(string(parent), evalKeys) {
		return PinEvidenceMalformed, evalPath
	}
	raw, err := os.ReadFile(filepath.Join(repoDir, "docs", "evals", evalDir, "runs", runName+".md"))
	if err != nil {
		return PinEvidenceMalformed, runPath
	}
	kv, has := meta.Parse(string(raw))
	if !has || !hasRequiredKeys(kv, runKeys) {
		return PinEvidenceMalformed, runPath
	}
	created, ok := parsePinEvidenceDate(kv["created"])
	if !ok || created.After(utcDate(today)) {
		return PinEvidenceMalformed, runPath
	}
	if utcDate(today).Sub(created).Hours()/24 > 90 {
		return PinEvidenceStale, runPath
	}
	runModel, ok := parsePinEvidenceScalar(kv["model"])
	if !ok {
		return PinEvidenceMalformed, runPath
	}
	if runModel != model {
		return PinEvidenceModelMismatch, runPath
	}
	return checkPinEvidenceBattery(kv), runPath
}

func checkedDirectory(path string) PinEvidenceKind {
	info, err := os.Lstat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return PinEvidenceMissing
		}
		return PinEvidenceMalformed
	}
	if info.Mode()&fs.ModeSymlink != 0 || !info.IsDir() {
		return PinEvidenceMalformed
	}
	return ""
}

func checkedRegularFile(path string) PinEvidenceKind {
	info, err := os.Lstat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return PinEvidenceMissing
		}
		return PinEvidenceMalformed
	}
	if info.Mode()&fs.ModeSymlink != 0 || !info.Mode().IsRegular() {
		return PinEvidenceMalformed
	}
	return ""
}

func hasRequiredFrontMatter(content string, keys []string) bool {
	kv, has := meta.Parse(content)
	return has && hasRequiredKeys(kv, keys)
}

func hasRequiredKeys(kv map[string]string, keys []string) bool {
	for _, key := range keys {
		if _, ok := kv[key]; !ok {
			return false
		}
	}
	return true
}

func validCalendarDate(s string) bool {
	_, ok := parsePinEvidenceDate(s)
	return ok
}

func parsePinEvidenceDate(s string) (time.Time, bool) {
	if len(s) >= 1 && s[0] == '"' {
		var ok bool
		s, ok = parsePinEvidenceScalar(s)
		if !ok {
			return time.Time{}, false
		}
	}
	if !pinEvidenceDate.MatchString(s) {
		return time.Time{}, false
	}
	date, err := time.Parse("2006-01-02", s)
	return date, err == nil
}

func utcDate(value time.Time) time.Time {
	value = value.UTC()
	return time.Date(value.Year(), value.Month(), value.Day(), 0, 0, 0, 0, time.UTC)
}

func parsePinEvidenceScalar(s string) (string, bool) {
	if strings.HasPrefix(s, "\"") {
		if !strings.HasSuffix(s, "\"") {
			return "", false
		}
		unquoted, err := strconv.Unquote(s)
		if err != nil {
			return "", false
		}
		s = unquoted
	}
	if s == "" || strings.IndexFunc(s, unicode.IsControl) >= 0 {
		return "", false
	}
	return s, true
}

func checkPinEvidenceBattery(kv map[string]string) PinEvidenceKind {
	_, version := kv["battery_version"]
	_, verdict := kv["battery_verdict"]
	_, results := kv["battery_results"]
	if !version && !verdict && !results {
		return PinEvidenceNoBattery
	}
	if !version || !verdict || !results || kv["battery_version"] != "1" {
		return PinEvidenceMalformed
	}
	failed, ok := validPinEvidenceMatrix(kv["battery_results"])
	if !ok {
		return PinEvidenceMalformed
	}
	switch kv["battery_verdict"] {
	case "pass":
		if failed {
			return PinEvidenceMalformed
		}
		return ""
	case "fail":
		if !failed {
			return PinEvidenceMalformed
		}
		return PinEvidenceFailedBattery
	default:
		return PinEvidenceMalformed
	}
}

func validPinEvidenceMatrix(value string) (failed, ok bool) {
	if strings.ContainsAny(value, " \t\r\n") {
		return false, false
	}
	entries := strings.Split(value, ",")
	if len(entries) != len(pinEvidenceBatteryKeys) {
		return false, false
	}
	allowed := map[string]bool{"KILLED": true, "SURVIVED": true, "NO-SITE": true, "BUILD-ERR": true, "REPORT-ONLY": true}
	for i, entry := range entries {
		key, result, found := strings.Cut(entry, "=")
		if !found || key != pinEvidenceBatteryKeys[i] || !allowed[result] {
			return false, false
		}
		if result == "REPORT-ONLY" && key != "security-default" && key != "lifecycle" {
			return false, false
		}
		if key != "security-default" && key != "lifecycle" && result != "KILLED" {
			failed = true
		}
	}
	return failed, true
}
