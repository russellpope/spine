package update

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// maipipeTimeout bounds the validate run. Validation is a parse of one small
// file; anything slower than this is a maipipe that is not going to answer.
var maipipeTimeout = 30 * time.Second

const (
	noMaipipePreflight       = "maipipe pre-flight skipped: no maipipe binary on PATH"
	maipipeValidatePreflight = "maipipe validate"
)

// maipipeLookup is a seam for the one I104 preflight resolution decision.
// Run pins its returned executable path for the candidate validation.
var maipipeLookup = exec.LookPath

const duplicateStageHint = "hint: the region spine renders declares each stage once, so a duplicate almost always means a copy of that stage now sits outside the " +
	gateRegionBegin + "… / " + gateRegionEnd + " markers — move or delete it by hand; spine will not rewrite what is outside its region"

// checkMaipipeContent asks maipipe, the sole grammar authority under I104,
// whether candidate content can load. The candidate is written to a temporary
// file, so a validation refusal never touches the real maipipe.toml.
func checkMaipipeContent(bin, path, content string) error {
	dir, err := os.MkdirTemp("", "spine-maipipe")
	if err != nil {
		return fmt.Errorf("maipipe pre-flight for %s: %w", path, err)
	}
	defer func() { _ = os.RemoveAll(dir) }()
	candidate := filepath.Join(dir, MaipipeFile)
	if err := os.WriteFile(candidate, []byte(content), 0o644); err != nil {
		return fmt.Errorf("maipipe pre-flight for %s: %w", path, err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), maipipeTimeout)
	defer cancel()
	out, err := exec.CommandContext(ctx, bin, "validate", candidate).CombinedOutput()
	if err == nil {
		return nil
	}
	if ctx.Err() != nil {
		return fmt.Errorf("refusing to write %s: maipipe validate did not finish within %s (%v)", path, maipipeTimeout, err)
	}
	var exitErr *exec.ExitError
	if !errors.As(err, &exitErr) {
		return fmt.Errorf("refusing to write %s: could not run maipipe validate (%v): %s", path, err, strings.TrimSpace(string(out)))
	}
	msg := strings.TrimSpace(string(out))
	if msg == "" {
		msg = err.Error()
	}
	if !exitErr.ProcessState.Exited() || exitErr.ExitCode() == 126 || exitErr.ExitCode() == 127 {
		return fmt.Errorf("refusing to write %s: could not run maipipe validate (%v): %s", path, err, msg)
	}
	if strings.Contains(msg, "duplicate stage name") {
		msg += "\n" + duplicateStageHint
	}
	return fmt.Errorf("refusing to write %s: maipipe validate rejected the result:\n%s", path, msg)
}
