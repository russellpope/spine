package gate

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"go/importer"
	"go/token"
	"go/types"
)

const exportDataMismatchPanic = "export data version 4 is greater than maximum supported version 2"

type panickingImporter struct{ value string }

func (p panickingImporter) Import(string) (*types.Package, error) {
	panic(p.value)
}

func loaderFixture(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	for name, contents := range map[string]string{
		"go.mod":     "module example.com/fixture\n\ngo 1.26\n",
		"fixture.go": "package fixture\n\nimport \"fmt\"\n\nvar _ = fmt.Sprintf\n",
	} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(contents), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

func withPanickingLoaderImporter(t *testing.T, value string) {
	t.Helper()
	old := newLoaderImporter
	newLoaderImporter = func(*token.FileSet, string, importer.Lookup) types.Importer {
		return panickingImporter{value: value}
	}
	t.Cleanup(func() { newLoaderImporter = old })
}

func withGoVersionProbe(t *testing.T, probe func() (string, error)) {
	t.Helper()
	old := runGoVersionCommand
	runGoVersionCommand = func() ([]byte, error) {
		version, err := probe()
		return []byte(version), err
	}
	t.Cleanup(func() { runGoVersionCommand = old })
}

func loadModuleWithoutPanic(t *testing.T, dir string) (got *loaded, err error) {
	t.Helper()
	defer func() {
		if recovered := recover(); recovered != nil {
			t.Fatalf("loadModule panicked: %v", recovered)
		}
	}()
	return loadModule(dir)
}

func TestLoadModuleReturnsActionableErrorForExportDataPanic(t *testing.T) {
	withPanickingLoaderImporter(t, exportDataMismatchPanic)

	_, err := loadModuleWithoutPanic(t, loaderFixture(t))
	if err == nil {
		t.Fatal("loadModule returned nil error after export-data panic")
	}
	message := err.Error()
	for _, want := range []string{exportDataMismatchPanic, runtime.Version(), "make install"} {
		if !strings.Contains(message, want) {
			t.Errorf("mismatch error = %q; want %q", message, want)
		}
	}
	if strings.Contains(message, "does not type-check") {
		t.Errorf("mismatch error reused type-check wording: %q", message)
	}
}

func TestLoadModuleReturnsInternalErrorForUnrelatedPanic(t *testing.T) {
	const panicValue = "unexpected importer failure"
	withPanickingLoaderImporter(t, panicValue)

	_, err := loadModuleWithoutPanic(t, loaderFixture(t))
	if err == nil {
		t.Fatal("loadModule returned nil error after unrelated panic")
	}
	message := err.Error()
	for _, want := range []string{"example.com/fixture", panicValue, "goroutine"} {
		if !strings.Contains(message, want) {
			t.Errorf("internal error = %q; want %q", message, want)
		}
	}
	if strings.Contains(message, "make install") {
		t.Errorf("internal error suggested rebuild: %q", message)
	}
}

func TestLoadModuleMismatchErrorNamesPathToolchain(t *testing.T) {
	withPanickingLoaderImporter(t, exportDataMismatchPanic)
	withGoVersionProbe(t, func() (string, error) { return "go1.path", nil })

	_, err := loadModuleWithoutPanic(t, loaderFixture(t))
	if err == nil {
		t.Fatal("loadModule returned nil error after export-data panic")
	}
	if !strings.Contains(err.Error(), "go1.path") {
		t.Errorf("mismatch error does not name PATH toolchain: %q", err)
	}
}

func TestLoadModuleMismatchErrorOmitsFailedPathToolchainProbe(t *testing.T) {
	const probeFailure = "PATH probe failed"
	withPanickingLoaderImporter(t, exportDataMismatchPanic)
	withGoVersionProbe(t, func() (string, error) { return "", errors.New(probeFailure) })

	_, err := loadModuleWithoutPanic(t, loaderFixture(t))
	if err == nil {
		t.Fatal("loadModule returned nil error after export-data panic")
	}
	message := err.Error()
	for _, want := range []string{exportDataMismatchPanic, runtime.Version(), "make install"} {
		if !strings.Contains(message, want) {
			t.Errorf("mismatch error = %q; want %q", message, want)
		}
	}
	for _, unwanted := range []string{"toolchain on PATH", probeFailure} {
		if strings.Contains(message, unwanted) {
			t.Errorf("mismatch error contains %q: %q", unwanted, message)
		}
	}
}
