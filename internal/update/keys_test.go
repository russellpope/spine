package update

import (
	"testing"
)

const gen0Hbmview = `# Workflow — hbmview

profile: rust
reviewers: [rust-reviewer, security-review]
functional_harness: cli    # cli | rest | framebuffer | none
gates: [grill, verify]             # mandatory; everything else advisory
model_default: claude-opus-4-8     # swappable; re-evaluate on major model/platform releases
security_routing: quality-framing-opus-4-8
stages: [grill, prd, issues, implement, functional-test, review, verify, ship, deploy, docs, handoff]

See ` + "`docs/harness-interface.md`" + ` for the functional-test harness contract.
Mandatory gates: a PRD up front (grill-with-docs -> to-prd) and verification before completion.
`

func TestExtractKeys(t *testing.T) {
	keys := ExtractKeys(gen0Hbmview)
	want := map[string]string{
		"profile":            "rust",
		"reviewers":          "[rust-reviewer, security-review]",
		"functional_harness": "cli",
		"gates":              "[grill, verify]",
		"model_default":      "claude-opus-4-8",
		"security_routing":   "quality-framing-opus-4-8",
	}
	for k, v := range want {
		if keys[k] != v {
			t.Errorf("keys[%q] = %q, want %q", k, keys[k], v)
		}
	}
	if _, ok := keys["template_version"]; ok {
		t.Error("gen0 file must have no template_version")
	}
}

func TestExtractKeysRoutingSubBlock(t *testing.T) {
	content := "model_routing:\n  primary: claude-fable-5   # x\n  fallback: claude-opus-4-8\n  routine: claude-sonnet-5\neffort: high\n"
	keys := ExtractKeys(content)
	if keys["model_routing.primary"] != "claude-fable-5" ||
		keys["model_routing.routine"] != "claude-sonnet-5" ||
		keys["effort"] != "high" {
		t.Errorf("keys = %#v", keys)
	}
}

func TestProjectFromWorkflow(t *testing.T) {
	if got := ProjectFromWorkflow(gen0Hbmview, "fb"); got != "hbmview" {
		t.Errorf("got %q", got)
	}
	if got := ProjectFromWorkflow("no title", "fb"); got != "fb" {
		t.Errorf("fallback got %q", got)
	}
}

// The heart of un-stranding: values equal to their own generation's defaults
// are NOT choices; hbmview's gen0 model_default must not survive.
func TestChoicesDropsGenerationDefaults(t *testing.T) {
	choices, err := Choices(ExtractKeys(gen0Hbmview), "gen0", "hbmview")
	if err != nil {
		t.Fatal(err)
	}
	if choices["profile"] != "rust" {
		t.Errorf("profile must always be preserved, got %#v", choices)
	}
	if _, ok := choices["model_default"]; ok {
		t.Errorf("gen0-default model_default wrongly kept as a choice: %#v", choices)
	}
	if _, ok := choices["reviewers"]; ok {
		t.Errorf("profile-derived reviewers wrongly kept: %#v", choices)
	}
}

func TestChoicesKeepsRealChoices(t *testing.T) {
	custom := ExtractKeys(gen0Hbmview)
	custom["functional_harness"] = "rest" // user overrode the rust default (cli)
	choices, err := Choices(custom, "gen0", "hbmview")
	if err != nil {
		t.Fatal(err)
	}
	if choices["functional_harness"] != "rest" {
		t.Errorf("real choice dropped: %#v", choices)
	}
}

func TestSetKey(t *testing.T) {
	content := "profile: rust\nmodel_routing:\n  primary: claude-fable-5          # comment\neffort: high    # c2\n"
	got := setKey(content, "model_routing.primary", "custom-model")
	if want := "  primary: custom-model    # comment"; !containsLine(got, want) {
		t.Errorf("sub-key: got\n%s", got)
	}
	got = setKey(content, "effort", "xhigh")
	if want := "effort: xhigh    # c2"; !containsLine(got, want) {
		t.Errorf("top key: got\n%s", got)
	}
}

// template_version is a stamp, never a preservable choice — even when the
// extracted value diverges from anything a template would render.
func TestChoicesSkipsTemplateVersion(t *testing.T) {
	extracted := ExtractKeys(gen0Hbmview)
	extracted["template_version"] = "0"
	choices, err := Choices(extracted, "gen0", "hbmview")
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := choices["template_version"]; ok {
		t.Errorf("template_version wrongly kept as a choice: %#v", choices)
	}
}

// A '#' is only a comment start at the start of the value or when preceded
// by whitespace; a value with an interior '#' (no adjacent whitespace) must
// survive extraction whole.
func TestValueWithInteriorHashSurvivesExtraction(t *testing.T) {
	content := "security_routing: quality#framing\n"
	keys := ExtractKeys(content)
	if keys["security_routing"] != "quality#framing" {
		t.Fatalf("got %q, want %q", keys["security_routing"], "quality#framing")
	}
	// round-trips through setKey unchanged when the value is unchanged
	if got := setKey(content, "security_routing", "quality#framing"); got != content {
		t.Errorf("round-trip mutated line: got %q want %q", got, content)
	}
}

// A real trailing comment (space-separated from the value) must still strip,
// even when the value itself contains an interior '#'.
func TestValueWithInteriorHashRealCommentStrips(t *testing.T) {
	content := "security_routing: quality#framing    # swappable; re-evaluate\n"
	keys := ExtractKeys(content)
	if keys["security_routing"] != "quality#framing" {
		t.Fatalf("got %q, want %q", keys["security_routing"], "quality#framing")
	}
}

func containsLine(content, line string) bool {
	for _, l := range splitLines(content) {
		if l == line {
			return true
		}
	}
	return false
}
