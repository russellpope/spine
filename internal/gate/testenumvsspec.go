package gate

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// testEnumSpecKey is the gate_pack_config key holding the spec file the
// enums are compared against; it reaches the stage as
// SPINE_GATE_TEST_ENUM_SPEC.
const testEnumSpecKey = "test_enum_spec"

// Enum markers in the spec file. A marked block names one type and encloses
// that type's documented values:
//
//	<!-- spine:enum Severity -->
//	`low`, `med`, `high`
//	<!-- /spine:enum -->
//
// Every backticked token inside the block is a value; prose between them is
// ignored, so the block can read as documentation.
const (
	enumOpenPrefix = "<!-- spine:enum "
	enumOpenSuffix = "-->"
	enumClose      = "<!-- /spine:enum -->"
)

// checkTestEnumVsSpec compares the typed string enums declared in code
// under --dir against the values the configured spec file enumerates,
// reporting each side's extras: a value in code the spec does not document,
// and a value the spec documents that no const declares.
//
// The code side is syntactic: a const spec with an explicit named type and
// a string literal value (`const ( Low Severity = "low" )`) contributes
// that value under that type name. A const without an explicit type
// contributes nothing — the enum has to say what it is.
//
// The spec side is the marked blocks described above. Only types named by a
// marker are compared, so a repo documents the enums it cares about. A spec
// file with no marker at all is misconfiguration, not a clean pass.
func checkTestEnumVsSpec(dir string, cfg Config) ([]Finding, error) {
	rel, _ := cfg.Get(testEnumSpecKey)
	rel = strings.TrimSpace(rel)
	if rel == "" {
		return nil, fmt.Errorf("%s is unset or empty: set it to the spec file path relative to --dir", EnvVar(testEnumSpecKey))
	}
	b, err := os.ReadFile(resolveUnder(dir, rel))
	if err != nil {
		return nil, fmt.Errorf("%s: reading %s: %w", EnvVar(testEnumSpecKey), rel, err)
	}
	specEnums, err := parseSpecEnums(string(b), rel)
	if err != nil {
		return nil, err
	}
	if len(specEnums) == 0 {
		return nil, fmt.Errorf("%s: %s has no enum marker: enclose each documented enum in %s<TypeName> %s … %s", EnvVar(testEnumSpecKey), rel, enumOpenPrefix, enumOpenSuffix, enumClose)
	}
	codeEnums, err := parseCodeEnums(dir)
	if err != nil {
		return nil, err
	}
	var findings []Finding
	for typeName, spec := range specEnums {
		code := codeEnums[typeName]
		for value, pos := range code {
			if _, ok := spec[value]; ok {
				continue
			}
			findings = append(findings, Finding{
				Severity: SeverityError,
				Message:  fmt.Sprintf("enum value %q of %s is declared in code but not enumerated in %s", value, typeName, rel),
				File:     pos.file,
				Line:     pos.line,
				Code:     Code("test-enum-vs-spec"),
			})
		}
		for value, line := range spec {
			if _, ok := code[value]; ok {
				continue
			}
			findings = append(findings, Finding{
				Severity: SeverityError,
				Message:  fmt.Sprintf("enum value %q of %s is enumerated in the spec but no const declares it", value, typeName),
				File:     rel,
				Line:     line,
				Code:     Code("test-enum-vs-spec"),
			})
		}
	}
	return findings, nil
}

// enumPos is where a code-side enum value is declared.
type enumPos struct {
	file string
	line int
}

// parseSpecEnums reads the marked enum blocks: type name -> value -> the
// spec line the value appears on. An unterminated or nested block is
// misconfiguration — the intended value set would otherwise be a guess.
func parseSpecEnums(content, rel string) (map[string]map[string]int, error) {
	enums := map[string]map[string]int{}
	openType, openLine := "", 0
	for i, line := range strings.Split(content, "\n") {
		lineNo := i + 1
		trimmed := strings.TrimSpace(line)
		switch {
		case strings.HasPrefix(trimmed, enumOpenPrefix):
			name := strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(trimmed, enumOpenPrefix), enumOpenSuffix))
			if openType != "" {
				return nil, fmt.Errorf("%s: %s:%d: enum block for %s opens inside the block for %s opened at line %d", EnvVar(testEnumSpecKey), rel, lineNo, name, openType, openLine)
			}
			if name == "" {
				return nil, fmt.Errorf("%s: %s:%d: enum marker names no type: expected %s<TypeName> %s", EnvVar(testEnumSpecKey), rel, lineNo, enumOpenPrefix, enumOpenSuffix)
			}
			openType, openLine = name, lineNo
			if enums[name] == nil {
				enums[name] = map[string]int{}
			}
		case trimmed == enumClose:
			if openType == "" {
				return nil, fmt.Errorf("%s: %s:%d: %s with no open enum block", EnvVar(testEnumSpecKey), rel, lineNo, enumClose)
			}
			openType = ""
		case openType != "":
			for _, value := range backtickedTokens(line) {
				if _, seen := enums[openType][value]; !seen {
					enums[openType][value] = lineNo
				}
			}
		}
	}
	if openType != "" {
		return nil, fmt.Errorf("%s: %s:%d: enum block for %s is never closed with %s", EnvVar(testEnumSpecKey), rel, openLine, openType, enumClose)
	}
	return enums, nil
}

// backtickedTokens returns the `backticked` tokens on one spec line.
func backtickedTokens(line string) []string {
	var out []string
	rest := line
	for {
		_, after, ok := strings.Cut(rest, "`")
		if !ok {
			return out
		}
		token, remainder, ok := strings.Cut(after, "`")
		if !ok {
			return out
		}
		if t := strings.TrimSpace(token); t != "" {
			out = append(out, t)
		}
		rest = remainder
	}
}

// parseCodeEnums collects the typed string enum values declared under dir:
// type name -> value -> declaration position. _test.go files are not read —
// the enum a spec documents is the one the code ships.
func parseCodeEnums(dir string) (map[string]map[string]enumPos, error) {
	enums := map[string]map[string]enumPos{}
	fset := token.NewFileSet()
	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if skipDir(d.Name()) && path != dir {
				return filepath.SkipDir
			}
			return nil
		}
		name := d.Name()
		if !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			return nil
		}
		rel := relSlash(dir, path)
		file, perr := parser.ParseFile(fset, path, nil, 0)
		if perr != nil {
			return fmt.Errorf("parsing %s: %w", rel, perr)
		}
		for _, decl := range file.Decls {
			gen, ok := decl.(*ast.GenDecl)
			if !ok || gen.Tok != token.CONST {
				continue
			}
			for _, spec := range gen.Specs {
				vs, ok := spec.(*ast.ValueSpec)
				if !ok {
					continue
				}
				typeIdent, ok := vs.Type.(*ast.Ident)
				if !ok {
					continue
				}
				for _, v := range vs.Values {
					lit, ok := v.(*ast.BasicLit)
					if !ok || lit.Kind != token.STRING {
						continue
					}
					value, uerr := strconv.Unquote(lit.Value)
					if uerr != nil {
						continue
					}
					if enums[typeIdent.Name] == nil {
						enums[typeIdent.Name] = map[string]enumPos{}
					}
					if _, seen := enums[typeIdent.Name][value]; !seen {
						enums[typeIdent.Name][value] = enumPos{file: rel, line: fset.Position(lit.Pos()).Line}
					}
				}
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return enums, nil
}
