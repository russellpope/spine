package eval

import (
	"io"
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

// pinEvidenceBeforeOpen is a deterministic test seam for replacement attacks.
// Production leaves it nil. It is called after an entry is Lstat'd and before
// the rooted descriptor is opened, which is the formerly vulnerable boundary.
var pinEvidenceBeforeOpen func(string)

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
	reader, kind := openPinEvidenceRoot(repoDir)
	if kind != "" {
		return kind, runPath
	}
	for _, component := range []string{"docs", "evals", evalDir} {
		next, kind := reader.openDirectory(component)
		if kind != "" {
			_ = reader.Close()
			return kind, runPath
		}
		reader = next
	}
	defer func() { _ = reader.Close() }()
	parent, kind := reader.readRegularFile("eval.md")
	if kind != "" {
		return kind, evalPath
	}
	if !hasRequiredFrontMatter(string(parent), evalKeys) {
		return PinEvidenceMalformed, evalPath
	}
	next, kind := reader.openDirectory("runs")
	if kind != "" {
		return kind, runPath
	}
	reader = next
	raw, kind := reader.readRegularFile(runName + ".md")
	if kind != "" {
		return kind, runPath
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

// pinEvidenceRoot holds a descriptor for a selected directory. Every nested
// component is opened relative to that descriptor, so a pathname replacement
// cannot redirect the read outside repoDir. Each opened descriptor is also
// matched to the object Lstat observed before it was opened.
type pinEvidenceRoot struct {
	root    *os.Root
	path    string
	parents []*os.Root
}

func openPinEvidenceRoot(repoDir string) (*pinEvidenceRoot, PinEvidenceKind) {
	root, err := os.OpenRoot(repoDir)
	if err != nil {
		return nil, PinEvidenceMalformed
	}
	return &pinEvidenceRoot{root: root}, ""
}

func (r *pinEvidenceRoot) Close() error {
	err := r.root.Close()
	for i := len(r.parents) - 1; i >= 0; i-- {
		if closeErr := r.parents[i].Close(); err == nil {
			err = closeErr
		}
	}
	return err
}

func (r *pinEvidenceRoot) openDirectory(name string) (*pinEvidenceRoot, PinEvidenceKind) {
	info, err := r.root.Lstat(name)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, PinEvidenceMissing
		}
		return nil, PinEvidenceMalformed
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
		return nil, PinEvidenceMalformed
	}
	path := filepath.ToSlash(filepath.Join(r.path, name))
	if pinEvidenceBeforeOpen != nil {
		pinEvidenceBeforeOpen(path)
	}
	child, err := r.root.OpenRoot(name)
	if err != nil {
		return nil, PinEvidenceMalformed
	}
	opened, err := child.Stat(".")
	if err != nil || !os.SameFile(info, opened) {
		_ = child.Close()
		return nil, PinEvidenceMalformed
	}
	parents := append(append([]*os.Root(nil), r.parents...), r.root)
	return &pinEvidenceRoot{root: child, path: path, parents: parents}, ""
}

func (r *pinEvidenceRoot) readRegularFile(name string) ([]byte, PinEvidenceKind) {
	info, err := r.root.Lstat(name)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, PinEvidenceMissing
		}
		return nil, PinEvidenceMalformed
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
		return nil, PinEvidenceMalformed
	}
	path := filepath.ToSlash(filepath.Join(r.path, name))
	if pinEvidenceBeforeOpen != nil {
		pinEvidenceBeforeOpen(path)
	}
	file, err := r.root.Open(name)
	if err != nil {
		return nil, PinEvidenceMalformed
	}
	defer file.Close()
	opened, err := file.Stat()
	if err != nil || !opened.Mode().IsRegular() || !os.SameFile(info, opened) {
		return nil, PinEvidenceMalformed
	}
	content, err := io.ReadAll(file)
	if err != nil {
		return nil, PinEvidenceMalformed
	}
	return content, ""
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
	return date, err == nil && date.Year() != 0
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
