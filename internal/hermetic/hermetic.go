// Package hermetic keeps a test binary off the developer's machine. Code
// under test reads owner-local state through os.UserConfigDir (the host
// routing file, hostconfig.DefaultPath) and HOME; a suite that sees the real
// ones passes or fails by whatever the developer has configured. Every test
// package's TestMain calls Main, and TestEveryTestPackageIsHermetic holds the
// tree to that.
package hermetic

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Main runs m under Isolate and returns its exit code, or 2 when the
// environment cannot be isolated. Use it as os.Exit(hermetic.Main(m)).
func Main(m *testing.M) int {
	_, restore, err := Isolate()
	if err != nil {
		fmt.Fprintf(os.Stderr, "hermetic: %v\n", err)
		return 2
	}
	defer restore()
	return m.Run()
}

// Isolate points HOME at a fresh temporary directory and unsets
// XDG_CONFIG_HOME, so os.UserConfigDir resolves inside it on every platform
// spine runs on; a test that sets HOME itself still moves the config dir.
// GOCACHE and GOMODCACHE stay on the developer's caches so child go builds
// are not cold. It fails unless os.UserConfigDir lands under the new root.
// restore puts the environment back and removes the directory.
func Isolate() (root string, restore func(), err error) {
	keep, err := goCaches()
	if err != nil {
		return "", nil, err
	}
	root, err = os.MkdirTemp("", "spine-hermetic-home-")
	if err != nil {
		return "", nil, err
	}
	saved := map[string]*string{}
	set := func(k string, v *string) {
		if _, ok := saved[k]; !ok {
			if old, had := os.LookupEnv(k); had {
				saved[k] = &old
			} else {
				saved[k] = nil
			}
		}
		if v == nil {
			os.Unsetenv(k)
		} else {
			os.Setenv(k, *v)
		}
	}
	restore = func() {
		for k, v := range saved {
			if v == nil {
				os.Unsetenv(k)
			} else {
				os.Setenv(k, *v)
			}
		}
		os.RemoveAll(root)
	}
	for k, v := range keep {
		set(k, &v)
	}
	set("HOME", &root)
	set("XDG_CONFIG_HOME", nil)
	appData := filepath.Join(root, "AppData")
	set("AppData", &appData)

	dir, err := os.UserConfigDir()
	if err != nil {
		restore()
		return "", nil, fmt.Errorf("locate user configuration directory after isolating: %w", err)
	}
	if !within(root, dir) {
		restore()
		return "", nil, fmt.Errorf("user configuration directory %q is outside the isolated home %q", dir, root)
	}
	return root, restore, nil
}

// goCaches resolves the go command's default cache locations from the real
// HOME, before Isolate moves it, for the variables the environment leaves
// unset.
func goCaches() (map[string]string, error) {
	keep := map[string]string{}
	if os.Getenv("GOCACHE") == "" {
		dir, err := os.UserCacheDir()
		if err != nil {
			return nil, fmt.Errorf("locate user cache directory: %w", err)
		}
		keep["GOCACHE"] = filepath.Join(dir, "go-build")
	}
	if os.Getenv("GOMODCACHE") == "" {
		gopath := filepath.SplitList(os.Getenv("GOPATH"))
		if len(gopath) == 0 || gopath[0] == "" {
			home, err := os.UserHomeDir()
			if err != nil {
				return nil, fmt.Errorf("locate home directory: %w", err)
			}
			gopath = []string{filepath.Join(home, "go")}
		}
		keep["GOMODCACHE"] = filepath.Join(gopath[0], "pkg", "mod")
	}
	return keep, nil
}

func within(root, path string) bool {
	rel, err := filepath.Rel(root, path)
	return err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}
