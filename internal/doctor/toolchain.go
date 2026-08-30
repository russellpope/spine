package doctor

import (
	"fmt"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
)

// toolchainCheck is D14: an advisory-only early signal that the Go toolchain
// which built spine differs materially from the toolchain selected on PATH.
// Unlike I107's gate-side detection, this is deliberately a proxy: it cannot
// prove an export-data importer will fail, so it never reports an error. Patch
// releases are ignored because they need not change that export-data format.
func toolchainCheck() []Finding {
	pathOutput, err := exec.Command("go", "env", "GOVERSION").Output()
	if err != nil {
		return nil
	}
	pathVersion := strings.TrimSpace(string(pathOutput))
	binaryVersion := runtime.Version()
	binaryMajor, binaryMinor, binaryOK := goMajorMinor(binaryVersion)
	pathMajor, pathMinor, pathOK := goMajorMinor(pathVersion)
	if !binaryOK || !pathOK || (binaryMajor == pathMajor && binaryMinor == pathMinor) {
		return nil
	}
	return []Finding{{"D14", "warn", "go", fmt.Sprintf(
		"Go major/minor toolchain skew — binary toolchain: %s; toolchain on PATH: %s; rebuild spine with make install",
		binaryVersion, pathVersion)}}
}

// goMajorMinor parses the Go release prefix shared by runtime.Version and
// `go env GOVERSION`. Development and prerelease strings stay quiet: D14's
// version comparison is only an advisory proxy when both values are definite.
func goMajorMinor(version string) (int, int, bool) {
	parts := strings.Split(strings.TrimPrefix(version, "go"), ".")
	if len(parts) < 2 {
		return 0, 0, false
	}
	major, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0, 0, false
	}
	minor, err := strconv.Atoi(parts[1])
	if err != nil {
		return 0, 0, false
	}
	return major, minor, true
}
