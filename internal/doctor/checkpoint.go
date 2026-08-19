package doctor

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/russellpope/spine/internal/checkpoint"
)

// checkpointCheck is D11, the checkpoint advisory (I081). Advisory means
// warn and only warn: a hand-edited facts region, a hole in the ordinal
// sequence, or an uncommitted-by-convention working home that git would
// actually commit are all drift worth surfacing mid-effort, none of them a
// reason to fail. `spine audit stages` is deliberately untouched.
//
// It covers three things:
//
//   - the facts region of each checkpoint: absent, unparseable, or
//     byte-drifted from its canonical rendering (sole-writer violated);
//   - ordinal gaps in the checkpoint working home (001, 003 => 002 is
//     missing) — reservation markers are directories and never count;
//   - `.superpowers/` not being gitignored, since the working home is
//     uncommitted by convention and spine does not manage .gitignore.
//
// The first two run only when the working home exists; the third only when
// `.superpowers/` exists and dir is a git repo. Otherwise: silence.
func checkpointCheck(dir string) []Finding {
	var findings []Finding
	home := checkpoint.Home(dir)
	rel := relTo(dir, home)
	if _, err := os.Stat(home); err == nil {
		entries, err := checkpoint.List(dir)
		if err != nil {
			findings = append(findings, Finding{"D11", "warn", rel,
				"checkpoint working home unreadable: " + err.Error()})
		}
		for _, e := range entries {
			findings = append(findings, factsFindings(dir, e)...)
		}
		findings = append(findings, gapFindings(dir, rel, entries)...)
	}
	findings = append(findings, gitignoreFindings(dir)...)
	return findings
}

// factsFindings reports the facts region of one checkpoint.
func factsFindings(dir string, e checkpoint.Entry) []Finding {
	path := relTo(dir, e.Path)
	raw, err := os.ReadFile(e.Path)
	if err != nil {
		return []Finding{{"D11", "warn", path, "checkpoint unreadable: " + err.Error()}}
	}
	body := checkpoint.Split(string(raw)).Facts
	if body == "" {
		return []Finding{{"D11", "warn", path,
			"facts region missing or its markers are damaged — the checkpoint carries no machine facts"}}
	}
	if _, err := checkpoint.ParseFacts(body); err != nil {
		return []Finding{{"D11", "warn", path, "facts region malformed: " + err.Error()}}
	}
	if !checkpoint.Canonical(body) {
		return []Finding{{"D11", "warn", path,
			"facts region is not canonical — only spine checkpoint new writes it; reconcile the hand edit"}}
	}
	return nil
}

// gapFindings reports every ordinal missing between the checkpoints present.
func gapFindings(dir, rel string, entries []checkpoint.Entry) []Finding {
	var findings []Finding
	for i := 1; i < len(entries); i++ {
		for n := entries[i-1].Ordinal + 1; n < entries[i].Ordinal; n++ {
			findings = append(findings, Finding{"D11", "warn", rel,
				fmt.Sprintf("ordinal gap: no checkpoint %03d between %03d and %03d",
					n, entries[i-1].Ordinal, entries[i].Ordinal)})
		}
	}
	return findings
}

// gitignoreFindings advises when `.superpowers/` would be committed. Silent
// when dir is not a git repo (check-ignore exits 128) or when `.superpowers/`
// does not exist at all — nothing to leak yet.
func gitignoreFindings(dir string) []Finding {
	if _, err := os.Stat(filepath.Join(dir, ".superpowers")); err != nil {
		return nil
	}
	cmd := exec.Command("git", "check-ignore", "-q", ".superpowers")
	cmd.Dir = dir
	err := cmd.Run()
	if err == nil {
		return nil // ignored
	}
	if ee, ok := err.(*exec.ExitError); !ok || ee.ExitCode() != 1 {
		return nil // not a git repo, or git unavailable — not our business
	}
	return []Finding{{"D11", "warn", ".superpowers",
		"not gitignored — the checkpoint working home is uncommitted by convention; add `.superpowers/` to .gitignore"}}
}

func relTo(dir, path string) string {
	rel, err := filepath.Rel(dir, path)
	if err != nil {
		return path
	}
	return strings.TrimPrefix(rel, "./")
}
