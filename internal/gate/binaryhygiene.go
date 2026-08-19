package gate

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// magic is one executable/archive signature: want bytes at offset off.
type magic struct {
	off  int
	want []byte
	kind string
}

// magics is the documented signature list binary-hygiene detects by
// content. Detection is on content, not extension, because the failure this
// class exists to prevent (a multi-megabyte executable committed to
// history) arrives with any name at all.
var magics = []magic{
	{0, []byte{0x7f, 'E', 'L', 'F'}, "ELF executable"},
	{0, []byte{0xfe, 0xed, 0xfa, 0xce}, "Mach-O executable"},
	{0, []byte{0xfe, 0xed, 0xfa, 0xcf}, "Mach-O executable"},
	{0, []byte{0xce, 0xfa, 0xed, 0xfe}, "Mach-O executable"},
	{0, []byte{0xcf, 0xfa, 0xed, 0xfe}, "Mach-O executable"},
	{0, []byte{0xca, 0xfe, 0xba, 0xbe}, "Mach-O fat binary or Java class"},
	{0, []byte{0xca, 0xfe, 0xba, 0xbf}, "Mach-O fat binary"},
	{0, []byte{0xbe, 0xba, 0xfe, 0xca}, "Mach-O fat binary"},
	{0, []byte{0xbf, 0xba, 0xfe, 0xca}, "Mach-O fat binary"},
	{0, []byte("MZ"), "PE/DOS executable"},
	{0, []byte("!<arch>\n"), "static library archive"},
	{0, []byte("PK\x03\x04"), "zip archive"},
	{0, []byte("PK\x05\x06"), "zip archive"},
	{0, []byte("PK\x07\x08"), "zip archive"},
	{0, []byte{0x1f, 0x8b}, "gzip archive"},
	{0, []byte("BZh"), "bzip2 archive"},
	{0, []byte{0xfd, '7', 'z', 'X', 'Z', 0x00}, "xz archive"},
	{0, []byte{'7', 'z', 0xbc, 0xaf, 0x27, 0x1c}, "7z archive"},
	{0, []byte{0x28, 0xb5, 0x2f, 0xfd}, "zstd archive"},
	{0, []byte{'R', 'a', 'r', '!', 0x1a, 0x07}, "rar archive"},
	{257, []byte("ustar"), "tar archive"},
}

// headerBytes is how much of each tracked file is read for signature
// matching; enough for the tar signature at offset 257.
const headerBytes = 512

// checkBinaryHygiene reports tracked files that are executables or archives
// by content, plus stray second module trees — a go.mod tracked anywhere
// other than the repo root, which makes `go build ./...` silently skip a
// subtree. It reads the tracked set from git, so --dir must be a git repo.
func checkBinaryHygiene(dir string, cfg Config) ([]Finding, error) {
	tracked, err := gitLsFiles(dir)
	if err != nil {
		return nil, err
	}
	var findings []Finding
	for _, rel := range tracked {
		if rel == "go.mod" {
			continue
		}
		if filepath.Base(rel) == "go.mod" {
			findings = append(findings, Finding{
				Severity: SeverityError,
				Message:  "stray second module tree: tracked go.mod outside the repo root",
				File:     rel,
				Line:     0,
				Code:     Code("binary-hygiene"),
			})
			continue
		}
		kind, err := detectBinary(filepath.Join(dir, filepath.FromSlash(rel)))
		if err != nil {
			return nil, err
		}
		if kind != "" {
			findings = append(findings, Finding{
				Severity: SeverityError,
				Message:  "committed binary: tracked file is a " + kind + " by content",
				File:     rel,
				Line:     0,
				Code:     Code("binary-hygiene"),
			})
		}
	}
	return findings, nil
}

// detectBinary returns the signature kind matched by the file's header, or
// "" when nothing matches. A file that has vanished since git listed it is
// not a finding.
func detectBinary(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", err
	}
	defer f.Close()
	head := make([]byte, headerBytes)
	n, err := f.Read(head)
	if err != nil && n == 0 {
		return "", nil // empty or unreadable-at-offset-0; nothing to match
	}
	head = head[:n]
	for _, m := range magics {
		end := m.off + len(m.want)
		if len(head) >= end && bytes.Equal(head[m.off:end], m.want) {
			return m.kind, nil
		}
	}
	return "", nil
}

// gitLsFiles returns the tracked paths in dir, slash-separated and relative
// to dir. A dir that is not a git repo is misconfiguration, not a finding.
func gitLsFiles(dir string) ([]string, error) {
	cmd := exec.Command("git", "ls-files", "-z")
	cmd.Dir = dir
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = err.Error()
		}
		return nil, fmt.Errorf("--dir %s: reading tracked files with git: %s", dir, msg)
	}
	var files []string
	for _, p := range strings.Split(string(out), "\x00") {
		if p != "" {
			files = append(files, p)
		}
	}
	return files, nil
}
