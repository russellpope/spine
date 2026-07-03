# spine v3 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Clear the v1/v2 deferred-work ledger: TOCTOU-free create paths, correct fleet ages, no swallowed errors, list headers, preservation notice, and the gen-3 template batch (YAML-safe ADR front matter).

**Architecture:** One new fsutil primitive (`WriteFileExclusive`, temp+link) adopted by all four create paths; three error-propagation fixes; one calendar-day computation fix behind an injectable clock; three text-render tweaks in cmd/spine; one template edit riding a generation bump to 3 with a stamp-only fixture lock. Spec: `docs/specs/2026-07-03-spine-v3-design.md` (approved).

**Tech Stack:** Go, stdlib only (ADR 0001). No new dependencies.

## Global Constraints

- Branch: `build/v3` off main. Commits stay local — **NEVER push** (origin has no main; first push is Russell's call).
- Go stdlib only (ADR 0001). `gofmt` clean; `go vet ./...` clean.
- **Never edit `templates/current/*` content except in Task 6, and there the edit and the `templates/VERSION` bump to 3 must land in the SAME commit** (generation invariant; `TestGen1To2IsStampOnly` guards the past, Task 6 adds the gen-2→3 lock).
- Task 6's fixture-generation step MUST run (and be committed) BEFORE any template edit — it snapshots gen-2 behavior using gen-2 code.
- Error-path tests use symlink loops (ELOOP), never chmod (chmod probes proven vacuous in this repo's review history).
- User-facing error strings that already exist must survive verbatim; each task lists them.
- CLI flags come BEFORE positional args in tests and examples (`spine handoff new -dir X "topic"`), because Go's flag parsing stops at the first non-flag argument.
- Line numbers reference main @ `c24e79a`.

---

### Task 1: fsutil.WriteFileExclusive

**Files:**
- Modify: `internal/fsutil/fsutil.go`
- Test: `internal/fsutil/fsutil_test.go` (create if absent; extend if present)

**Interfaces:**
- Consumes: nothing new.
- Produces: `func WriteFileExclusive(path string, data []byte) error` — atomically creates `path` with `data` (mode 0644) ONLY if it does not exist; a pre-existing path (file, dir, or symlink — even dangling) yields an error satisfying `errors.Is(err, fs.ErrExist)`. Tasks 2 and 6 call it.

- [ ] **Step 1: Write the failing tests**

Append to `internal/fsutil/fsutil_test.go` (create the file with `package fsutil` + imports if it does not exist):

```go
func TestWriteFileExclusiveCreates(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "new.md")
	if err := WriteFileExclusive(path, []byte("hello")); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(path)
	if err != nil || string(raw) != "hello" {
		t.Fatalf("content=%q err=%v", raw, err)
	}
	fi, err := os.Stat(path)
	if err != nil || fi.Mode().Perm() != 0o644 {
		t.Fatalf("mode=%v err=%v", fi.Mode(), err)
	}
	// No temp residue.
	des, _ := os.ReadDir(dir)
	if len(des) != 1 {
		t.Fatalf("residue in dir: %v", des)
	}
}

func TestWriteFileExclusiveRefusesExisting(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "taken.md")
	if err := os.WriteFile(path, []byte("original"), 0o644); err != nil {
		t.Fatal(err)
	}
	err := WriteFileExclusive(path, []byte("usurper"))
	if !errors.Is(err, fs.ErrExist) {
		t.Fatalf("want fs.ErrExist, got %v", err)
	}
	raw, _ := os.ReadFile(path)
	if string(raw) != "original" {
		t.Fatalf("existing content clobbered: %q", raw)
	}
	des, _ := os.ReadDir(dir)
	if len(des) != 1 {
		t.Fatalf("residue in dir: %v", des)
	}
}

func TestWriteFileExclusiveRefusesDanglingSymlink(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "link.md")
	if err := os.Symlink(filepath.Join(dir, "nowhere"), path); err != nil {
		t.Fatal(err)
	}
	// link(2) fails EEXIST on an existing path even when it is a dangling
	// symlink — the never-overwrite contract must hold for links too.
	if err := WriteFileExclusive(path, []byte("x")); !errors.Is(err, fs.ErrExist) {
		t.Fatalf("want fs.ErrExist, got %v", err)
	}
}

func TestWriteFileExclusiveConcurrentSingleWinner(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "raced.md")
	const n = 8
	start := make(chan struct{})
	errs := make(chan error, n)
	var wg sync.WaitGroup
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			<-start
			errs <- WriteFileExclusive(path, []byte(fmt.Sprintf("writer-%d", i)))
		}(i)
	}
	close(start)
	wg.Wait()
	close(errs)
	wins, exists := 0, 0
	for err := range errs {
		switch {
		case err == nil:
			wins++
		case errors.Is(err, fs.ErrExist):
			exists++
		default:
			t.Fatalf("unexpected error: %v", err)
		}
	}
	if wins != 1 || exists != n-1 {
		t.Fatalf("wins=%d exists=%d (want 1 / %d)", wins, exists, n-1)
	}
	des, _ := os.ReadDir(dir)
	if len(des) != 1 {
		t.Fatalf("temp residue after race: %v", des)
	}
}
```

Test-file imports: `errors`, `fmt`, `io/fs`, `os`, `path/filepath`, `sync`, `testing`.

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/fsutil/ -run TestWriteFileExclusive -v`
Expected: FAIL — `undefined: WriteFileExclusive` (compile error).

- [ ] **Step 3: Implement**

Append to `internal/fsutil/fsutil.go` (same shape as `WriteFileAtomic`, fsutil.go:15-40):

```go
// WriteFileExclusive writes data to path only if path does not already
// exist. The content lands via temp-file + link(2), so the create is atomic
// and a crash never leaves a partial file at path. Any pre-existing path —
// regular file, directory, or symlink (even dangling) — fails with an error
// satisfying errors.Is(err, fs.ErrExist); callers own the user-facing
// message. Mode is normalized to 0644, matching WriteFileAtomic.
func WriteFileExclusive(path string, data []byte) error {
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, ".spine-*")
	if err != nil {
		return err
	}
	name := tmp.Name()
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		os.Remove(name)
		return err
	}
	if err := tmp.Close(); err != nil {
		os.Remove(name)
		return err
	}
	if err := os.Chmod(name, 0o644); err != nil {
		os.Remove(name)
		return err
	}
	if err := os.Link(name, path); err != nil {
		os.Remove(name)
		return err
	}
	os.Remove(name)
	return nil
}
```

No new imports needed (`os`, `path/filepath` already imported).

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/fsutil/ -v`
Expected: PASS (all, including any pre-existing WriteFileAtomic tests).

- [ ] **Step 5: Commit**

```bash
git add internal/fsutil/
git commit -m "feat(fsutil): WriteFileExclusive — atomic create-only primitive (TOCTOU close, v3 C1)"
```

---

### Task 2: exclusive create at the four call sites

**Files:**
- Modify: `internal/handoff/handoff.go:57-77` (New)
- Modify: `internal/eval/eval.go:56-90` (New), `eval.go:137-156` (AddRun)
- Modify: `internal/adr/adr.go:138-141` (New)
- Test: `internal/handoff/handoff_test.go`, `internal/eval/eval_test.go` (extend existing files)

**Interfaces:**
- Consumes: `fsutil.WriteFileExclusive` (Task 1).
- Produces: no signature changes; behavior only. Preserved error strings (verbatim): handoff `"%s already exists — pick a more specific topic"`; eval.New `"%s already exists"` (evalDir); eval.AddRun `"%s already exists"` (run path).

**Why no failing test for the adr conversion:** a genuine `adr new` collision is only reachable through a race — any file pre-placed at the computed path is itself counted by `List`, which bumps `next` past it. The race semantics are proven at the fsutil layer (Task 1's concurrent test); here the adr change is conversion-only, guarded by the existing suite passing unchanged plus the new error branch shown below.

- [ ] **Step 1: Write the failing message-pin tests (handoff + eval)**

Append to `internal/handoff/handoff_test.go`:

```go
func TestNewSecondSameTopicSameDayFails(t *testing.T) {
	dir := t.TempDir()
	if _, err := New(dir, "same topic"); err != nil {
		t.Fatal(err)
	}
	_, err := New(dir, "same topic")
	if err == nil || !strings.Contains(err.Error(), "already exists — pick a more specific topic") {
		t.Fatalf("want already-exists error, got %v", err)
	}
}
```

Append to `internal/eval/eval_test.go`:

```go
func TestAddRunSecondSameNameFails(t *testing.T) {
	dir := t.TempDir()
	evalDir, err := New(dir, "collision eval")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := AddRun(dir, filepath.Base(evalDir), "run1"); err != nil {
		t.Fatal(err)
	}
	_, err = AddRun(dir, filepath.Base(evalDir), "run1")
	if err == nil || !strings.Contains(err.Error(), "already exists") {
		t.Fatalf("want already-exists error, got %v", err)
	}
}
```

(If equivalent tests already exist under different names, keep the existing ones and skip the duplicates — the pins matter, not the names.)

- [ ] **Step 2: Run to check current state**

Run: `go test ./internal/handoff/ ./internal/eval/ -run "SameTopicSameDay|AddRunSecondSameName" -v`
Expected: PASS already (the Stat guards catch the sequential case). These tests pin the messages through the conversion — they must STILL pass after Step 3.

- [ ] **Step 3: Convert the four sites**

`internal/handoff/handoff.go` — delete the Stat guard (lines 58-65) and convert the write (line 74). New body from line 57:

```go
	path := filepath.Join(hdir, today+"-"+slug+".md")
	raw, err := templates.FS.ReadFile("current/handoff.tmpl.md")
	if err != nil {
		return "", err
	}
	content := strings.NewReplacer(
		"{{HANDOFF_TITLE}}", topic,
		"{{HANDOFF_DATE}}", today,
	).Replace(string(raw))
	if err := fsutil.WriteFileExclusive(path, []byte(content)); err != nil {
		if errors.Is(err, fs.ErrExist) {
			return "", fmt.Errorf("%s already exists — pick a more specific topic", path)
		}
		return "", err
	}
	return path, nil
```

Add imports `errors`, `io/fs` to handoff.go.

`internal/eval/eval.go` New — KEEP the evalDir Stat guard (lines 57-64: it is a directory-level fast path with the right message). Convert the README block (lines 68-77) to always-attempt-exclusive (EEXIST = someone already wrote it = success):

```go
	readme := filepath.Join(root, "README.md")
	rawReadme, rerr := templates.FS.ReadFile("current/evals-README.md")
	if rerr != nil {
		return "", rerr
	}
	if werr := fsutil.WriteFileExclusive(readme, rawReadme); werr != nil && !errors.Is(werr, fs.ErrExist) {
		return "", werr
	}
```

Convert the eval.md write (lines 86-88):

```go
	if err := fsutil.WriteFileExclusive(filepath.Join(evalDir, "eval.md"), []byte(content)); err != nil {
		if errors.Is(err, fs.ErrExist) {
			return "", fmt.Errorf("%s already exists", evalDir)
		}
		return "", err
	}
```

`internal/eval/eval.go` AddRun — delete the Stat guard (lines 138-144) and convert the write (lines 153-155):

```go
	if err := fsutil.WriteFileExclusive(path, []byte(content)); err != nil {
		if errors.Is(err, fs.ErrExist) {
			return "", fmt.Errorf("%s already exists", path)
		}
		return "", err
	}
```

Add imports `errors`, `io/fs` to eval.go.

`internal/adr/adr.go` — convert the new-ADR write (lines 139-141); the supersede flip at line 146 is an intentional overwrite and STAYS `WriteFileAtomic`:

```go
	if err := fsutil.WriteFileExclusive(path, []byte(content)); err != nil {
		if errors.Is(err, fs.ErrExist) {
			return "", fmt.Errorf("%s already exists — a concurrent adr new likely took id %s; re-run", path, id)
		}
		return "", err
	}
```

Add imports `errors`, `io/fs` to adr.go.

- [ ] **Step 4: Full-package regression**

Run: `go test ./internal/... ./cmd/...`
Expected: PASS everywhere — the conversion is behavior-preserving for every sequentially-constructible case.

- [ ] **Step 5: Commit**

```bash
git add internal/handoff/ internal/eval/ internal/adr/
git commit -m "fix: exclusive create in handoff/eval/adr New paths — Stat-then-Write TOCTOU closed (v3 C1)"
```

---

### Task 3: stop swallowing errors (three sites)

**Files:**
- Modify: `internal/eval/eval.go:209-216` (checkDoc) and its caller at `eval.go:177`
- Modify: `internal/handoff/handoff.go:101-105` (List)
- Modify: `internal/update/update.go:101-108` (evals-dir Stat)
- Test: `internal/eval/eval_test.go`, `internal/handoff/handoff_test.go`, `internal/update/update_test.go`

**Interfaces:**
- Consumes: nothing new.
- Produces: `checkDoc(path string, keys []string) ([]Problem, error)` — signature change, private to package eval. `handoff.List` / `eval.List` / `update.Run` signatures unchanged; they now return errors they previously ate.

- [ ] **Step 1: Write the failing ELOOP tests**

Append to `internal/eval/eval_test.go`:

```go
func TestListSurfacesEvalDocReadError(t *testing.T) {
	dir := t.TempDir()
	evalDir := filepath.Join(dir, "docs", "evals", "2026-07-03-loop")
	if err := os.MkdirAll(filepath.Join(evalDir, "runs"), 0o755); err != nil {
		t.Fatal(err)
	}
	// Self-referential symlink: ReadFile fails ELOOP — a read error, not absence.
	if err := os.Symlink("eval.md", filepath.Join(evalDir, "eval.md")); err != nil {
		t.Fatal(err)
	}
	_, _, err := List(dir)
	if err == nil {
		t.Fatal("want read error surfaced, got nil (was mislabeled 'missing eval.md')")
	}
}
```

Append to `internal/handoff/handoff_test.go`:

```go
func TestListSurfacesHandoffReadError(t *testing.T) {
	dir := t.TempDir()
	hdir := filepath.Join(dir, "docs", "handoffs")
	if err := os.MkdirAll(hdir, 0o755); err != nil {
		t.Fatal(err)
	}
	name := "2026-07-03-loop.md"
	if err := os.Symlink(name, filepath.Join(hdir, name)); err != nil {
		t.Fatal(err)
	}
	_, err := List(dir)
	if err == nil {
		t.Fatal("want read error surfaced, got nil (Title silently degraded before v3)")
	}
}
```

Append to `internal/update/update_test.go` (seed the repo the same way TestGen1To2IsStampOnly does, gen1to2_test.go:14-23):

```go
func TestRunSurfacesEvalsDirStatError(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{"WORKFLOW.md", "CLAUDE.md"} {
		raw, err := os.ReadFile(filepath.Join("testdata", "ccq", name))
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, name), raw, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.MkdirAll(filepath.Join(dir, "docs"), 0o755); err != nil {
		t.Fatal(err)
	}
	// docs/evals as a symlink loop: Stat fails with ELOOP, not ENOENT.
	if err := os.Symlink("evals", filepath.Join(dir, "docs", "evals")); err != nil {
		t.Fatal(err)
	}
	if _, err := Run(Options{Dir: dir}); err == nil {
		t.Fatal("want Stat error surfaced, got nil (silently skipped evals-README before v3)")
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/eval/ ./internal/handoff/ ./internal/update/ -run "SurfacesEvalDocReadError|SurfacesHandoffReadError|SurfacesEvalsDirStatError" -v`
Expected: all three FAIL with the `got nil` fatals.

- [ ] **Step 3: Implement the three fixes**

`internal/eval/eval.go` — replace checkDoc (lines 209-216):

```go
func checkDoc(path string, keys []string) ([]Problem, error) {
	raw, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return []Problem{{Path: path, Message: "missing eval.md"}}, nil
	}
	if err != nil {
		return nil, err
	}
	kv, has := meta.Parse(string(raw))
	return checkKeys(path, kv, has, keys), nil
}
```

and its caller (line 177):

```go
		probs, err := checkDoc(filepath.Join(e.Path, "eval.md"), evalKeys)
		if err != nil {
			return nil, nil, err
		}
		problems = append(problems, probs...)
```

`internal/handoff/handoff.go` — replace the read block (lines 101-105):

```go
		raw, err := os.ReadFile(e.Path)
		if err != nil {
			return nil, err
		}
		if kv, has := meta.Parse(string(raw)); has && kv["title"] != "" {
			e.Title = kv["title"]
		}
```

(`Fleet`, handoff.go:145-148, already skips children whose `Latest` errors — the fleet-resilience contract from v2 T8 holds without changes; its test pins it.)

`internal/update/update.go` — replace the evals-dir block (lines 101-108) with a three-way guard:

```go
	// docs/evals/README.md is opt-in machine-owned: managed only where the
	// convention is in use (the directory exists); never created by init/adopt.
	fi, err := os.Stat(filepath.Join(opts.Dir, "docs", "evals"))
	switch {
	case err == nil && fi.IsDir():
		r, err := planSimple(opts.Dir, gen, "evals-README.md", "docs/evals/README.md", false, vals)
		if err != nil {
			return nil, err
		}
		reports = append(reports, r)
	case err != nil && !os.IsNotExist(err):
		return nil, err
	}
```

- [ ] **Step 4: Run tests to verify they pass, then the full suite**

Run: `go test ./internal/eval/ ./internal/handoff/ ./internal/update/ -v` then `go test ./...`
Expected: PASS. (Doctor's D7 path calls eval.List, which now propagates read errors into doctor's existing error/exit-2 handling — no doctor code change needed.)

- [ ] **Step 5: Commit**

```bash
git add internal/eval/ internal/handoff/ internal/update/
git commit -m "fix: surface swallowed errors — checkDoc read vs missing, handoff.List title reads, evals-dir Stat (v3 C3)"
```

---

### Task 4: ageDays calendar-day fix

**Files:**
- Modify: `cmd/spine/main.go:358-364`
- Test: `cmd/spine/main_test.go`

**Interfaces:**
- Consumes: nothing new.
- Produces: `var now = time.Now` (package main, test seam) and calendar-day `ageDays`. Task 5's tests may also use `now`.

- [ ] **Step 1: Write the failing test**

Append to `cmd/spine/main_test.go`:

```go
func TestAgeDaysIsCalendarLocal(t *testing.T) {
	defer func() { now = time.Now }()
	la, err := time.LoadLocation("America/Los_Angeles")
	if err != nil {
		t.Fatal(err)
	}
	// 17:00 PDT on 2026-07-03 == 2026-07-04 00:00 UTC: the old
	// hours/24-since-UTC-midnight math reported a today-dated handoff as 1d.
	now = func() time.Time { return time.Date(2026, 7, 3, 17, 0, 0, 0, la) }
	cases := []struct {
		filenameDate string
		want         int
	}{
		{"2026-07-03", 0}, // today — the observed off-by-one
		{"2026-07-02", 1}, // yesterday
		{"2026-06-26", 7},
		{"2026-07-04", 0}, // future-dated clamps to 0
	}
	for _, c := range cases {
		d, err := time.Parse("2006-01-02", c.filenameDate)
		if err != nil {
			t.Fatal(err)
		}
		if got := ageDays(d); got != c.want {
			t.Errorf("ageDays(%s) = %d, want %d", c.filenameDate, got, c.want)
		}
	}
}
```

Test-file needs import `time` (add if absent).

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./cmd/spine/ -run TestAgeDaysIsCalendarLocal -v`
Expected: FAIL — compile error `undefined: now`, and after a naive stub the `{"2026-07-03", 0}` case would fail with 1.

- [ ] **Step 3: Implement**

Replace `ageDays` (main.go:358-364):

```go
// now is a seam for tests; production code always leaves it as time.Now.
var now = time.Now

// ageDays is a calendar-day difference. The handoff filename date is a plain
// local calendar date (handoff.New stamps time.Now().Format("2006-01-02"),
// handoff.go:52) that arrives parsed as UTC midnight; comparing instants
// against time.Now() made today's handoffs show "1d" west of UTC. Compare
// calendar dates instead.
func ageDays(d time.Time) int {
	n := now()
	today := time.Date(n.Year(), n.Month(), n.Day(), 0, 0, 0, 0, time.UTC)
	that := time.Date(d.Year(), d.Month(), d.Day(), 0, 0, 0, 0, time.UTC)
	age := int(today.Sub(that).Hours() / 24)
	if age < 0 {
		return 0
	}
	return age
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./cmd/spine/ -v`
Expected: PASS (all cmd tests).

- [ ] **Step 5: Commit**

```bash
git add cmd/spine/
git commit -m "fix(handoff): fleet age_days is a local calendar-day diff, not hours/24 since UTC midnight (v3 C2)"
```

---

### Task 5: text-output cosmetics (headers, path column, preservation notice)

**Files:**
- Modify: `cmd/spine/main.go:286-288` (handoff list text), `main.go:489-495` (eval list text), `main.go:124-126` (update UpToDate render)
- Test: `cmd/spine/main_test.go`

**Interfaces:**
- Consumes: `runCmd` helper (main_test.go:14-19), `update.FileReport.Preserved` (update.go:47-50).
- Produces: exact text formats pinned below; `--json` output untouched everywhere.

- [ ] **Step 1: Write the failing tests**

Append to `cmd/spine/main_test.go`:

```go
func TestHandoffListTextHasHeaderAndPath(t *testing.T) {
	dir := t.TempDir()
	if code, _, errs := runCmd(t, "handoff", "new", "-dir", dir, "v3 cosmetics"); code != 0 {
		t.Fatal(errs)
	}
	code, out, errs := runCmd(t, "handoff", "list", "-dir", dir)
	if code != 0 {
		t.Fatal(errs)
	}
	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
	if len(lines) != 2 {
		t.Fatalf("want header + 1 row, got %d lines: %q", len(lines), out)
	}
	if !strings.HasPrefix(lines[0], "date") || !strings.Contains(lines[0], "topic") || !strings.Contains(lines[0], "path") {
		t.Errorf("header missing/wrong: %q", lines[0])
	}
	if !strings.Contains(lines[1], "v3-cosmetics") || !strings.Contains(lines[1], filepath.Join(dir, "docs", "handoffs")) {
		t.Errorf("row missing topic or path: %q", lines[1])
	}
}

func TestEvalListTextHasHeader(t *testing.T) {
	dir := t.TempDir()
	if code, _, errs := runCmd(t, "eval", "new", "-dir", dir, "header eval"); code != 0 {
		t.Fatal(errs)
	}
	code, out, errs := runCmd(t, "eval", "list", "-dir", dir)
	if code != 0 {
		t.Fatal(errs)
	}
	first := strings.SplitN(out, "\n", 2)[0]
	if !strings.HasPrefix(first, "eval") || !strings.Contains(first, "run") ||
		!strings.Contains(first, "stage") || !strings.Contains(first, "score") {
		t.Errorf("header missing/wrong: %q", first)
	}
}

func TestUpdateTextNamesPreservedFiles(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{"WORKFLOW.md", "CLAUDE.md"} {
		raw, err := os.ReadFile(filepath.Join("..", "..", "internal", "update", "testdata", "ccq", name))
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, name), raw, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.MkdirAll(filepath.Join(dir, "docs", "adr"), 0o755); err != nil {
		t.Fatal(err)
	}
	// Hand-authored ADR index: ADR-0009 territory — update must SAY so.
	if err := os.WriteFile(filepath.Join(dir, "docs", "adr", "README.md"), []byte("# my hand-rolled index\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, out, _ := runCmd(t, "update", "-dir", dir)
	if !strings.Contains(out, "preserved (hand-authored): docs/adr/README.md") {
		t.Errorf("no preservation notice in:\n%s", out)
	}
}
```

(Slug note: `handoff new "v3 cosmetics"` produces filename slug `v3-cosmetics`; the topic column prints the raw topic — assert on the slug in the path OR relax to `strings.Contains(lines[1], "v3 cosmetics") || strings.Contains(lines[1], "v3-cosmetics")` if the first run shows the raw-topic form. Pin whichever the implementation prints; do not weaken both assertions.)

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./cmd/spine/ -run "HandoffListText|EvalListText|UpdateTextNamesPreserved" -v`
Expected: FAIL — no header lines, no preservation notice.

- [ ] **Step 3: Implement the three renders**

`main.go` handoff list text (replace lines 286-288):

```go
		fmt.Fprintf(stdout, "%-10s  %-28s  %s\n", "date", "topic", "path")
		for _, e := range entries {
			fmt.Fprintf(stdout, "%-10s  %-28s  %s\n", e.Date.Format("2006-01-02"), e.Topic, e.Path)
		}
```

`main.go` eval list text (insert header before the loop at line 489):

```go
		fmt.Fprintf(stdout, "%-30s  %-20s  %-10s  %s\n", "eval", "run", "stage", "score")
```

`main.go` update UpToDate case (replace lines 125-126):

```go
		case update.UpToDate:
			if r.Preserved {
				fmt.Fprintf(stdout, "preserved (hand-authored): %s\n", r.Path)
			} else {
				fmt.Fprintf(stdout, "up-to-date: %s\n", r.Path)
			}
```

- [ ] **Step 4: Run tests to verify they pass, plus full cmd suite**

Run: `go test ./cmd/spine/ -v`
Expected: PASS. If any pre-existing test pinned the old headerless output, update THAT test's expectation in this task (it is part of this deliverable).

- [ ] **Step 5: Commit**

```bash
git add cmd/spine/
git commit -m "feat(cli): list headers + handoff path column + 'preserved (hand-authored)' update notice (v3 C4)"
```

---

### Task 6: gen-3 template batch

**Files:**
- Create: `internal/update/testdata/ccq-gen2/WORKFLOW.md`, `internal/update/testdata/ccq-gen2/CLAUDE.md` (generated, step 1)
- Create: `internal/update/gen2to3_test.go`
- Modify: `templates/current/adr.tmpl.md`, `templates/VERSION`, `internal/adr/adr.go` (New replacer + parseFrontMatter), `internal/tmpl/tmpl_test.go:10-14`
- Test: `internal/adr/adr_test.go`

**Interfaces:**
- Consumes: `fsutil.WriteFileExclusive` (already wired, Task 2).
- Produces: generation 3. New template placeholder `{{ADR_TITLE_YAML}}` consumed ONLY by `adr.New`. `parseFrontMatter` now unquotes quoted titles for display.

**ORDERING IS LOAD-BEARING:** Step 1 snapshots gen-2 output using gen-2 code and is committed BEFORE any template edit. Steps 3-5 (template edit + VERSION bump + code) land as ONE commit.

- [ ] **Step 1: Generate and commit the gen-2 fixture (BEFORE any template edit)**

```bash
cd ~/Projects/github.com/spine
tmp=$(mktemp -d)
cp internal/update/testdata/ccq/WORKFLOW.md internal/update/testdata/ccq/CLAUDE.md "$tmp"/
go run ./cmd/spine update -dir "$tmp" -write
mkdir -p internal/update/testdata/ccq-gen2
cp "$tmp"/WORKFLOW.md "$tmp"/CLAUDE.md internal/update/testdata/ccq-gen2/
grep template_version internal/update/testdata/ccq-gen2/WORKFLOW.md   # must say 2
git add internal/update/testdata/ccq-gen2/
git commit -m "test(update): snapshot ccq gen-2 fixture ahead of the gen-3 bump"
```

Expected: grep prints a `template_version` line containing `2`. If `go run` exits nonzero or the grep shows anything else, STOP — do not proceed to template edits.

- [ ] **Step 2: Write the failing tests (stamp-only lock + quoting + display)**

Create `internal/update/gen2to3_test.go` (mirrors gen1to2_test.go exactly, different fixture + message):

```go
package update

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The ccq-gen2 fixture is the gen-2 output of the ccq gen-1 fixture,
// generated by gen-2 code before the gen-3 template edit. Gen 3's only
// template change (adr.tmpl.md) is embedded-only — absent from update's
// emitted manifest — so updating 2→3 must be exactly the stamp + marker
// diff. Anything else means the gen-3 batch leaked into emitted content.
func TestGen2To3IsStampOnly(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{"WORKFLOW.md", "CLAUDE.md"} {
		raw, err := os.ReadFile(filepath.Join("testdata", "ccq-gen2", name))
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, name), raw, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	reports, err := Run(Options{Dir: dir})
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range reports {
		switch r.Path {
		case "WORKFLOW.md", "CLAUDE.md":
			if r.State != Pending {
				t.Errorf("%s: want Pending, got %v", r.Path, r.State)
				continue
			}
			for _, line := range strings.Split(r.Diff, "\n") {
				if !strings.HasPrefix(line, "+") && !strings.HasPrefix(line, "-") {
					continue
				}
				if strings.HasPrefix(line, "+++") || strings.HasPrefix(line, "---") {
					continue
				}
				if strings.Contains(line, "template_version") || strings.Contains(line, "spine:begin") {
					continue
				}
				t.Errorf("%s: unexpected changed line %q — gen 2→3 must be stamp-only", r.Path, line)
			}
		}
	}
}
```

Append to `internal/adr/adr_test.go`:

```go
func TestNewQuotesFrontMatterScalars(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "docs", "adr"), 0o755); err != nil {
		t.Fatal(err)
	}
	title := `spine v3: the "sweep" release`
	path, err := New(dir, title, 0)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	s := string(raw)
	if !strings.Contains(s, "\nid: \"0001\"\n") {
		t.Errorf("id not quoted:\n%s", s)
	}
	if !strings.Contains(s, "\ntitle: "+strconv.Quote(title)+"\n") {
		t.Errorf("title not quoted/escaped:\n%s", s)
	}
	if !strings.Contains(s, "# 0001: "+title+"\n") {
		t.Errorf("body H1 must keep the raw title:\n%s", s)
	}
	entries, err := List(dir)
	if err != nil || len(entries) != 1 {
		t.Fatalf("entries=%v err=%v", entries, err)
	}
	if entries[0].Title != title {
		t.Errorf("display Title = %q, want unquoted %q", entries[0].Title, title)
	}
}

func TestNewQuotesSupersedes(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "docs", "adr"), 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := New(dir, "first", 0); err != nil {
		t.Fatal(err)
	}
	path, err := New(dir, "second", 1)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), "\nsupersedes: \"0001\"\n") {
		t.Errorf("supersedes not quoted (octal quirk lives):\n%s", raw)
	}
}
```

(Add `strconv` to adr_test.go imports.)

- [ ] **Step 3: Run tests to verify they fail**

Run: `go test ./internal/update/ -run TestGen2To3 -v; go test ./internal/adr/ -run "QuotesFrontMatter|QuotesSupersedes" -v`
Expected: ALL THREE FAIL — TestGen2To3IsStampOnly with `want Pending, got UpToDate` (the fixture is current against gen-2 templates; only the bump makes it Pending), the two adr tests with unquoted scalars. That is the correct failing-first shape; Step 5 flips all three.

- [ ] **Step 4: The atomic template edit (one commit with Step 5)**

`templates/current/adr.tmpl.md` — replace the front-matter block only; body untouched:

```
---
id: "{{ADR_ID}}"
title: {{ADR_TITLE_YAML}}
status: Accepted
date: {{ADR_DATE}}{{ADR_SUPERSEDES}}
---
```

`templates/VERSION` — content becomes `3` followed by a trailing newline (`printf '3\n' > templates/VERSION`; `tmpl.Version` trims whitespace, tmpl.go:63).

`internal/adr/adr.go` — in New, quote the supersedes line (line 129) and extend the replacer (lines 132-137). **`{{ADR_TITLE_YAML}}` must precede `{{ADR_TITLE}}` in the replacer args** (strings.NewReplacer checks patterns in argument order at each position; the reverse order would corrupt the YAML token):

```go
	sup := ""
	if supersedes > 0 {
		sup = fmt.Sprintf("\nsupersedes: %q", fmt.Sprintf("%04d", supersedes))
	}
	id := fmt.Sprintf("%04d", next)
	content := strings.NewReplacer(
		"{{ADR_ID}}", id,
		"{{ADR_TITLE_YAML}}", strconv.Quote(title),
		"{{ADR_TITLE}}", title,
		"{{ADR_DATE}}", time.Now().Format("2006-01-02"),
		"{{ADR_SUPERSEDES}}", sup,
	).Replace(string(raw))
```

(Add `strconv` to adr.go imports.)

`internal/adr/adr.go` — parseFrontMatter (lines 70-76) learns to unquote for display:

```go
func parseFrontMatter(content string) (title, status string, hasFrontMatter bool) {
	kv, has := meta.Parse(content)
	if !has {
		return "", "", false
	}
	title = kv["title"]
	// Gen-3 templates YAML-quote the title (strconv.Quote in New). Unquote
	// for display; unquoted pre-gen-3 titles pass through verbatim.
	if len(title) >= 2 && title[0] == '"' && title[len(title)-1] == '"' {
		if u, err := strconv.Unquote(title); err == nil {
			title = u
		}
	}
	return title, kv["status"], true
}
```

`internal/tmpl/tmpl_test.go` — replace TestVersionIsOne (lines 10-14; name has been lying since v2):

```go
func TestVersionMatchesCurrentGeneration(t *testing.T) {
	if got := tmpl.Version(); got != 3 {
		t.Fatalf("Version() = %d, want 3", got)
	}
}
```

- [ ] **Step 5: Run the full suite, then commit template+code together**

Run: `go test ./...`
Expected: PASS everywhere — including TestGen1To2IsStampOnly (its diff filter keys on `template_version`/`spine:begin` lines, value-agnostic, so the 1→3 stamp passes), TestGen2To3IsStampOnly (the lock doing its job against the live edit), and both new adr tests.

```bash
git add templates/ internal/adr/ internal/tmpl/ internal/update/gen2to3_test.go
git commit -m "feat!: generation 3 — adr.tmpl.md YAML-quoted id/title/supersedes, stamp-only fixture lock (v3 C5)"
```

---

### Task 7: final regression + hygiene sweep

**Files:**
- No source changes expected; fixes only if the sweep finds drift.

- [ ] **Step 1: Full verification battery**

```bash
cd ~/Projects/github.com/spine
gofmt -l .          # expected: no output
go vet ./...        # expected: no output
go test ./...       # expected: ok in all packages
go build ./...      # expected: silent
go run ./cmd/spine version   # expected: 3
go run ./cmd/spine 2>&1 | head -3   # usage text unchanged (v3 adds no commands)
```

- [ ] **Step 2: Commit only if something needed fixing**

If gofmt/vet surfaced anything, fix and commit as `chore: gofmt/vet sweep`. Otherwise no commit.

---

### After the tasks (controller-owned, NOT subagent work)

Final whole-branch review (fresh reviewer, requirements-attack first per CLAUDE.md), then C6 live acceptance INLINE with Russell: install gen-3 binary (`go build -o ~/bin/spine ./cmd/spine`), spine repo self-update + doctor clean, two fleet dry-runs (praxis, ccq — expect exactly one pending stamp-only WORKFLOW.md diff each), scratch-repo `adr new` with a colon-and-quotes title (strict-YAML front matter + correct display), then finishing-a-development-branch (merge build/v3 → main FF, local only, NEVER push).

## Self-review notes

- Spec coverage: C1→Tasks 1-2, C2→Task 4, C3→Task 3, C4→Task 5, C5→Task 6, C6→post-task controller block. Non-goals untouched by any task. ✓
- The adr no-failing-test rationale is stated in Task 2 rather than hidden. ✓
- Type consistency: `WriteFileExclusive(path string, data []byte) error` used identically in Tasks 1/2/6; `checkDoc ([]Problem, error)` change is package-private with its one caller updated in the same task. ✓
- Task 5's `TestUpdateTextNamesPreserved` depends only on Task 5's render change (Preserved exists since v2); Task 6's tests depend on Tasks 1-2 only through unchanged behavior — task order is safe for sequential execution. ✓
