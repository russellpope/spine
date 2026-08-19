package update

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/russellpope/spine/internal/gate"
)

// The gate pack's delivery region lives in maipipe.toml at the repo root
// (ADR 0016): maipipe reads exactly one file with no include mechanism, so
// the pack must be present inside a file the repo also owns. spine owns the
// region between these two marker lines and nothing else in the file.
const (
	MaipipeFile     = "maipipe.toml"
	gateRegionBegin = "# spine:begin gate-pack "
	gateRegionEnd   = "# spine:end"
	// maipipeSchemaLine is the required top-level key of a maipipe.toml;
	// spine writes it only when creating the file from scratch (I091).
	maipipeSchemaLine = "schema = 0\n"
	gatePipelineName  = "gate-go"
	// The battery is the pack's advisory lane, so it gets its own pipeline
	// on maipipe's audit profile rather than a stage in the enforcement
	// lane (ADR 0015 item 5): a survivor must never block a push.
	mutationPipelineName = "mutation-go"
	mutateCheck          = "mutate"
)

// gatePackConfigKeys are the gate_pack_config sub-keys WORKFLOW.md renders,
// in render order. Each is extracted as the dotted key
// "gate_pack_config.<key>" and reaches its stage as gate.EnvVar(key).
var gatePackConfigKeys = []string{
	"test_enum_spec", "fixture_manifest", "build_outputs",
	"n_plus_one_clients", "tskip_allow",
}

// gateCheckConfig maps a check class to the one gate_pack_config key it
// consumes. Classes absent from the map take no configuration.
var gateCheckConfig = map[string]string{
	"tskip":             "tskip_allow",
	"gitignore-control": "build_outputs",
	"fixture-manifest":  "fixture_manifest",
	"test-enum-vs-spec": "test_enum_spec",
	"n-plus-one":        "n_plus_one_clients",
}

// gateRegionComment is the fixed header comment spine writes inside the
// region. It is the only comment the canonical render emits, so any other
// comment line inside the region is a local edit.
var gateRegionComment = []string{
	"# spine manages this region. Change it through the gate_pack keys in",
	"# WORKFLOW.md and re-run `spine update`, never by hand.",
	"# Compose the pack into your own lane with a stage: pipeline = \"" + gatePipelineName + "\"",
	"# and the advisory battery with: pipeline = \"" + mutationPipelineName + "\"",
}

// gatePackSettings is the WORKFLOW.md-side input to the region render.
type gatePackSettings struct {
	pack     string            // the repo's gate_pack value, "" when unset
	disabled map[string]bool   // check classes dropped by gate_pack_disabled
	config   map[string]string // gate_pack_config values, empty ones omitted
}

// gateSettings reads the gate pack keys out of WORKFLOW.md content.
func gateSettings(content string) gatePackSettings {
	keys := ExtractKeys(content)
	s := gatePackSettings{
		pack:     keys["gate_pack"],
		disabled: map[string]bool{},
		config:   map[string]string{},
	}
	for _, name := range parseList(keys["gate_pack_disabled"]) {
		s.disabled[name] = true
	}
	for _, k := range gatePackConfigKeys {
		if v := keys["gate_pack_config."+k]; v != "" {
			s.config[k] = v
		}
	}
	return s
}

// parseList splits a WORKFLOW.md "[a, b]" list value into its entries.
func parseList(v string) []string {
	v = strings.TrimSpace(v)
	v = strings.TrimPrefix(v, "[")
	v = strings.TrimSuffix(v, "]")
	var out []string
	for _, part := range strings.Split(v, ",") {
		if p := strings.TrimSpace(part); p != "" {
			out = append(out, p)
		}
	}
	return out
}

// renderGateRegion is the canonical region for (pack, disabled, config): a
// pure function of its inputs, byte-deterministic, ending in a newline.
// gate-go carries one stage per enabled check class in gate.CheckNames()
// order, except mutate; mutation-go carries mutate alone. Disabling mutate
// omits the mutation-go pipeline entirely, the same way disabling any other
// class omits its stage.
func renderGateRegion(s gatePackSettings) string {
	var b strings.Builder
	b.WriteString(gateRegionBegin + s.pack + "\n")
	for _, c := range gateRegionComment {
		b.WriteString(c + "\n")
	}
	b.WriteString("\n[pipelines." + gatePipelineName + "]\nprofile = \"full\"\n")
	for _, check := range gate.CheckNames() {
		if s.disabled[check] || check == mutateCheck {
			continue
		}
		b.WriteString("\n[[pipelines." + gatePipelineName + ".stage]]\n")
		fmt.Fprintf(&b, "name = %q\n", check)
		fmt.Fprintf(&b, "run = %q\n", "spine gate "+gate.PackName+" "+check)
		if key := gateCheckConfig[check]; key != "" {
			if v, ok := s.config[key]; ok {
				fmt.Fprintf(&b, "env = { %s = %s }\n", gate.EnvVar(key), strconv.Quote(v))
			}
		}
	}
	if !s.disabled[mutateCheck] {
		b.WriteString("\n[pipelines." + mutationPipelineName + "]\nprofile = \"audit\"\n")
		b.WriteString("\n[[pipelines." + mutationPipelineName + ".stage]]\n")
		fmt.Fprintf(&b, "name = %q\n", mutateCheck)
		fmt.Fprintf(&b, "run = %q\n", "spine gate "+gate.PackName+" "+mutateCheck)
	}
	b.WriteString(gateRegionEnd + "\n")
	return b.String()
}

// planMaipipe plans the gate-pack region in maipipe.toml. ok is false when
// there is nothing to report: no gate_pack is set, which is also the fleet
// negative control — the file is neither created nor touched, and an
// existing region is left alone rather than deleted.
func planMaipipe(dir, workflow string) (FileReport, bool, error) {
	s := gateSettings(workflow)
	if s.pack == "" {
		return FileReport{}, false, nil
	}
	report := FileReport{Path: MaipipeFile}
	if s.pack != gate.PackID() {
		// An unknown pack is never rendered and never guessed at: the repo
		// pinned a version this binary does not ship (ADR 0015).
		report.State = SkippedUnrecognized
		report.Unrecognized = []string{fmt.Sprintf(
			"gate_pack: %s is not a pack this spine binary ships (known: %s)", s.pack, gate.PackID())}
		return report, true, nil
	}
	region := renderGateRegion(s)
	path := filepath.Join(dir, MaipipeFile)
	raw, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		// A file maipipe will load needs its top-level `schema` key before
		// any table header (I091): created = schema line + region; the
		// repo adds its lanes below.
		created := maipipeSchemaLine + "\n" + region
		report.State = Pending
		report.Created = true
		report.Diff = Diff(report.Path, "", created)
		report.newContent = created
		return report, true, nil
	}
	if err != nil {
		return report, true, err
	}
	old := string(raw)
	begin, end, mErr := gateRegionBounds(old)
	if mErr != nil {
		// Marker damage is never force-droppable: no newContent, so --force
		// cannot overwrite a file whose region boundaries are unknown.
		report.Unrecognized = []string{mErr.Error()}
		report.State = SkippedUnrecognized
		return report, true, nil
	}
	var newContent string
	if begin < 0 {
		// No region yet: append it, preserving the file byte-for-byte above.
		prefix := old
		if prefix != "" && !strings.HasSuffix(prefix, "\n") {
			prefix += "\n"
		}
		if prefix != "" {
			prefix += "\n"
		}
		newContent = prefix + region
	} else {
		lines := splitLines(old)
		report.Unrecognized = unrecognizedRegionLines(lines[begin+1 : end])
		newContent = strings.Join(append(append(append([]string{}, lines[:begin]...),
			splitLines(strings.TrimSuffix(region, "\n"))...), lines[end+1:]...), "\n")
	}
	if d := Diff(report.Path, old, newContent); d != "" {
		report.State = Pending
		report.Diff = d
		report.newContent = newContent
	}
	return report, true, nil
}

// gateRegionBounds locates the region's marker lines by index. It returns
// (-1, -1, nil) when the file carries no region at all, and an error when
// the markers are damaged — a begin without an end, duplicates, or a pair in
// the wrong order — which is hand-repair work, not update's.
func gateRegionBounds(content string) (int, int, error) {
	begin, end := -1, -1
	var begins, ends int
	for i, line := range splitLines(content) {
		switch {
		case strings.HasPrefix(line, gateRegionBegin):
			begins++
			if begin < 0 {
				begin = i
			}
		case strings.TrimRight(line, " ") == gateRegionEnd:
			ends++
			if end < 0 {
				end = i
			}
		}
	}
	switch {
	case begins == 0 && ends == 0:
		return -1, -1, nil
	case begins != 1 || ends != 1:
		return 0, 0, fmt.Errorf("%s gate-pack markers unbalanced (%d begin / %d end); fix by hand",
			MaipipeFile, begins, ends)
	case end < begin:
		return 0, 0, fmt.Errorf("%s gate-pack markers out of order; fix by hand", MaipipeFile)
	}
	return begin, end, nil
}

// unrecognizedRegionLines returns the lines inside an existing region that
// no configuration of the pack could have rendered. Recognition is by shape
// rather than by exact text so that a changed gate_pack_config value or a
// newly disabled class refreshes silently (it is spine's own render of the
// repo's own choices, ADR 0002), while a hand-edited stage — a rewritten
// run line, an invented env var, a stray comment — is reported.
func unrecognizedRegionLines(lines []string) []string {
	checks := map[string]bool{}
	for _, c := range gate.CheckNames() {
		checks[c] = true
	}
	envVars := map[string]bool{}
	for _, k := range gatePackConfigKeys {
		envVars[gate.EnvVar(k)] = true
	}
	comments := map[string]bool{}
	for _, c := range gateRegionComment {
		comments[c] = true
	}
	var extra []string
	for _, raw := range lines {
		l := strings.TrimRight(raw, " ")
		switch {
		case l == "", comments[l]:
			continue
		case l == "[pipelines."+gatePipelineName+"]", l == `profile = "full"`,
			l == "[[pipelines."+gatePipelineName+".stage]]",
			l == "[pipelines."+mutationPipelineName+"]", l == `profile = "audit"`,
			l == "[[pipelines."+mutationPipelineName+".stage]]":
			continue
		}
		if v, ok := quotedValue(l, "name = "); ok && checks[v] {
			continue
		}
		if v, ok := quotedValue(l, "run = "); ok {
			if check, found := strings.CutPrefix(v, "spine gate "+gate.PackName+" "); found && checks[check] {
				continue
			}
		}
		if rest, ok := strings.CutPrefix(l, "env = { "); ok {
			name, val, split := strings.Cut(strings.TrimSuffix(rest, " }"), " = ")
			if split && envVars[name] {
				if _, err := strconv.Unquote(val); err == nil {
					continue
				}
			}
		}
		extra = append(extra, l)
	}
	return extra
}

// quotedValue unquotes the value of a `<prefix>"…"` TOML line.
func quotedValue(line, prefix string) (string, bool) {
	rest, ok := strings.CutPrefix(line, prefix)
	if !ok {
		return "", false
	}
	v, err := strconv.Unquote(rest)
	if err != nil {
		return "", false
	}
	return v, true
}
