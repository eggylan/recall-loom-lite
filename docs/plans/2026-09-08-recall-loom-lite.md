# RecallLoom Lite (rll) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build `rll`, a zero-state, single-binary Go CLI + separate skill package that gives AI agents lightweight file-based project memory (context brief, rolling summary, daily logs) with format-only validation and no receipts/state binding.

**Architecture:** A cobra CLI operating on a `.rll/` sidecar directory containing pure-Markdown files with minimal YAML frontmatter (`schema_version: 1`). No state.json, no receipts, no workspace revision, no config.json — daily log cursors are derived by scanning the directory. Write paths are exactly two: `log append` (append-only milestone entries, timestamped by CLI) and `write brief|summary|protocol` (full overwrite, frontmatter auto-prepended). The skill is a single English SKILL.md released alongside binaries.

**Tech Stack:** Go 1.22+, spf13/cobra v1.8.1, gopkg.in/yaml.v3, goreleaser v2, GitHub Actions.

**Design decisions locked in (from grill-me session):**

| # | Decision |
|---|----------|
| Q1 | Zero state: no state.json, no receipts, no revision. Cursors derived from directory scan |
| Q2 | Pure Markdown + simple frontmatter. Daily log entries are `## HH:MM title` sections. No HTML comment markers |
| Q3 | 8 commands: init, status, log append, write, validate, resume, query, archive |
| Q4 | Content via stdin (heredoc-friendly), short values via argv, `--file` backup, `--dry-run` = format preview only |
| Q5 | Skill = single SKILL.md, no references/, no profiles/ |
| Q6 | Sidecar dir `.rll/`, init idempotently appends to `.git/info/exclude` |
| Q7 | resume outputs EVERYTHING in one tier (brief + summary + latest log + protocol if present), Markdown or `--json` |
| Q8 | query = case-insensitive substring over summary + active logs (`--all` includes archive); archive = `--before YYYY-MM-DD` explicit, moves to `.rll/archive/`, `--apply` required to act |
| Q9 | init creates skeleton templates only; agent fills content via write commands per SKILL.md |
| Q10 | Zero config files. schema_version in each file's frontmatter. CLI output English. Local-timezone dates. `update_protocol.md` retained as optional file |
| Q11 | CLI and skill released separately; one GitHub Release carries both binary archives and skill archive |
| Q12 | cobra + goreleaser + yaml.v3, nothing else |
| Q13 | Entry timestamps CLI-generated (local time), `--date`/`--time` override for backfill. No clock sanity checks |
| Q14 | write supports three targets: brief, summary, protocol |
| Q15 | validate = 5 pure format checks, no semantics |
| Q16 | Binary name: `rll` |
| Q17 | License: Apache-2.0 |
| Q18 | Module: `github.com/eggylan/recall-loom-lite` |
| Q19 | SKILL.md in English, README in Chinese |

**Environment note:** Executor runs in WSL bash. All `go` commands are Linux-side. Project root: `/mnt/f/Dev/RecallLoom-Dev/recall-loom-lite` (the directory containing this plan; Task 1 git-inits it).

---

## File Structure

```
recall-loom-lite/
├── go.mod, go.sum
├── main.go                  # entrypoint, version injection point
├── .gitignore               # dist/, .rll/ (dogfooding)
├── .goreleaser.yaml         # Task 17
├── .github/workflows/release.yml  # Task 17
├── LICENSE                  # Apache-2.0
├── NOTICE                   # Task 16
├── README.md                # Chinese, Task 16
├── docs/plans/              # this plan
├── cmd/                     # cobra command layer (thin, no business logic)
│   ├── root.go              # root command, subcommand registry
│   ├── util.go              # readInput (stdin/file helper)
│   ├── init.go              # rll init
│   ├── status.go            # rll status
│   ├── log.go               # rll log append
│   ├── write.go             # rll write brief|summary|protocol
│   ├── validate.go          # rll validate
│   ├── resume.go            # rll resume
│   ├── query.go             # rll query
│   └── archive.go           # rll archive
├── internal/
│   ├── storage/
│   │   ├── root.go          # .rll discovery, project root walk
│   │   ├── frontmatter.go   # parse/render frontmatter, schema_version
│   │   ├── git.go           # GitRoot, EnsureGitExclude
│   │   └── templates.go     # brief/summary skeleton templates
│   ├── journal/
│   │   ├── entry.go         # Entry struct, header parse/render
│   │   └── journal.go       # List/Latest/Append/Parse daily logs
│   └── validate/
│       └── validate.go      # Run(root) []Finding, 5 format checks
└── skill/
    └── SKILL.md             # the whole skill package (Task 16)
```

Each package has a `_test.go` file next to it. Command layer is verified by smoke runs against a built binary (exact commands + expected output per task).

---

### Task 1: Project scaffold

**Files:**
- Create: `go.mod` (via go mod init), `main.go`, `cmd/root.go`, `.gitignore`

- [ ] **Step 1: Initialize git repo and Go module**

```bash
git init
go mod init github.com/eggylan/recall-loom-lite
```

- [ ] **Step 2: Create .gitignore**

`.gitignore`:

```
dist/
.rll/
```

- [ ] **Step 3: Create main.go**

`main.go`:

```go
package main

import "github.com/eggylan/recall-loom-lite/cmd"

// version is injected at build time via -ldflags "-X main.version=...".
var version = "dev"

func main() {
	cmd.Execute(version)
}
```

- [ ] **Step 4: Create cmd/root.go (version-only for now)**

`cmd/root.go`:

```go
package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// NewRootCommand builds the rll command tree.
func NewRootCommand(version string) *cobra.Command {
	root := &cobra.Command{
		Use:     "rll",
		Short:   "RecallLoom Lite - lightweight file-based project memory",
		Version: version,
	}
	return root
}

// Execute runs the root command and exits non-zero on error.
func Execute(version string) {
	if err := NewRootCommand(version).Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
```

- [ ] **Step 5: Fetch dependencies and verify build**

```bash
go get github.com/spf13/cobra@v1.8.1
go get gopkg.in/yaml.v3@v3.0.1
go mod tidy
go run . --version
```

Expected: prints `rll version dev`

- [ ] **Step 6: Commit**

```bash
git add -A
git commit -m "feat: project scaffold with cobra root command"
```

---

### Task 2: Storage root resolution

**Files:**
- Create: `internal/storage/root.go`
- Test: `internal/storage/root_test.go`

- [ ] **Step 1: Write the failing test**

`internal/storage/root_test.go`:

```go
package storage

import (
	"os"
	"path/filepath"
	"testing"
)

func TestProjectRootFindsRllFromSubdir(t *testing.T) {
	base := t.TempDir()
	root := filepath.Join(base, "proj")
	sub := filepath.Join(root, "a", "b")
	if err := os.MkdirAll(filepath.Join(root, DirName), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	got, err := ProjectRoot(sub)
	if err != nil {
		t.Fatal(err)
	}
	if got != root {
		t.Fatalf("ProjectRoot = %q, want %q", got, root)
	}
}

func TestProjectRootNotInitialized(t *testing.T) {
	if _, err := ProjectRoot(t.TempDir()); err != ErrNotInitialized {
		t.Fatalf("err = %v, want ErrNotInitialized", err)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/storage/ -v`
Expected: FAIL — `undefined: ProjectRoot` (compile error)

- [ ] **Step 3: Write minimal implementation**

`internal/storage/root.go`:

```go
package storage

import (
	"errors"
	"os"
	"path/filepath"
)

// DirName is the fixed sidecar directory name.
const DirName = ".rll"

// ErrNotInitialized is returned when no .rll directory is found.
var ErrNotInitialized = errors.New("no .rll directory found (run 'rll init' first)")

// ProjectRoot walks up from start until it finds a directory containing .rll.
func ProjectRoot(start string) (string, error) {
	dir := start
	for {
		if fi, err := os.Stat(filepath.Join(dir, DirName)); err == nil && fi.IsDir() {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", ErrNotInitialized
		}
		dir = parent
	}
}

// Root returns the absolute .rll path for the project containing start.
func Root(start string) (string, error) {
	project, err := ProjectRoot(start)
	if err != nil {
		return "", err
	}
	return filepath.Join(project, DirName), nil
}

// CwdRoot resolves the .rll path for the current working directory.
func CwdRoot() (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	return Root(cwd)
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/storage/ -v`
Expected: PASS (2 tests)

- [ ] **Step 5: Commit**

```bash
git add internal/storage/
git commit -m "feat: .rll storage root discovery"
```

---

### Task 3: Frontmatter parse and render

**Files:**
- Create: `internal/storage/frontmatter.go`
- Test: `internal/storage/frontmatter_test.go`

- [ ] **Step 1: Write the failing test**

`internal/storage/frontmatter_test.go`:

```go
package storage

import (
	"strings"
	"testing"
)

func TestParseDocumentWithFrontmatter(t *testing.T) {
	doc, err := ParseDocument("---\nschema_version: 1\n---\n\n# Title\n\nbody\n")
	if err != nil {
		t.Fatal(err)
	}
	if !doc.HasSchemaVersion() {
		t.Fatal("want schema_version detected")
	}
	if doc.Body != "# Title\n\nbody\n" {
		t.Fatalf("Body = %q", doc.Body)
	}
}

func TestParseDocumentWithoutFrontmatter(t *testing.T) {
	doc, err := ParseDocument("# plain\n")
	if err != nil {
		t.Fatal(err)
	}
	if doc.HasSchemaVersion() {
		t.Fatal("want no schema_version")
	}
	if doc.Body != "# plain\n" {
		t.Fatalf("Body = %q", doc.Body)
	}
}

func TestParseDocumentBrokenFrontmatter(t *testing.T) {
	if _, err := ParseDocument("---\nno end marker"); err == nil {
		t.Fatal("want error for unterminated frontmatter")
	}
}

func TestRenderDocument(t *testing.T) {
	out := RenderDocument("# hi\n\ncontent\n")
	want := "---\nschema_version: 1\n---\n\n# hi\n\ncontent\n"
	if out != want {
		t.Fatalf("RenderDocument = %q, want %q", out, want)
	}
	if !strings.HasPrefix(out, "---\n") || !strings.Contains(out, "\n---\n\n") {
		t.Fatal("frontmatter shape wrong")
	}
}

func TestRenderDocumentEmptyBody(t *testing.T) {
	out := RenderDocument("")
	if out != "---\nschema_version: 1\n---\n" {
		t.Fatalf("RenderDocument(\"\") = %q", out)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/storage/ -run TestParseDocument -v`
Expected: FAIL — `undefined: ParseDocument`

- [ ] **Step 3: Write minimal implementation**

`internal/storage/frontmatter.go`:

```go
package storage

import (
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

// SchemaVersion is the managed-file format version.
const SchemaVersion = 1

// Document is a managed markdown file split into frontmatter and body.
type Document struct {
	Frontmatter map[string]any
	Body        string
}

// ParseDocument splits a managed file into frontmatter and body.
// A file without frontmatter parses successfully with an empty Frontmatter map.
func ParseDocument(content string) (*Document, error) {
	if !strings.HasPrefix(content, "---\n") {
		return &Document{Frontmatter: map[string]any{}, Body: content}, nil
	}
	end := strings.Index(content[4:], "\n---\n")
	if end < 0 {
		return nil, fmt.Errorf("frontmatter start marker found but no end marker")
	}
	raw := content[4 : 4+end]
	body := strings.TrimPrefix(content[4+end+5:], "\n")
	var fm map[string]any
	if err := yaml.Unmarshal([]byte(raw), &fm); err != nil {
		return nil, fmt.Errorf("parse frontmatter: %w", err)
	}
	return &Document{Frontmatter: fm, Body: body}, nil
}

// RenderDocument builds file content with the standard frontmatter prepended.
func RenderDocument(body string) string {
	body = strings.TrimSpace(body)
	if body == "" {
		return fmt.Sprintf("---\nschema_version: %d\n---\n", SchemaVersion)
	}
	return fmt.Sprintf("---\nschema_version: %d\n---\n\n%s\n", SchemaVersion, body)
}

// HasSchemaVersion reports whether the frontmatter carries the expected version.
func (d *Document) HasSchemaVersion() bool {
	v, ok := d.Frontmatter["schema_version"]
	if !ok {
		return false
	}
	switch n := v.(type) {
	case int:
		return n == SchemaVersion
	case float64:
		return n == float64(SchemaVersion)
	case uint64:
		return n == uint64(SchemaVersion)
	}
	return false
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/storage/ -v`
Expected: PASS (all storage tests)

- [ ] **Step 5: Commit**

```bash
git add internal/storage/frontmatter.go internal/storage/frontmatter_test.go
git commit -m "feat: frontmatter parse/render with schema_version"
```

---

### Task 4: Git exclusion

**Files:**
- Create: `internal/storage/git.go`
- Test: `internal/storage/git_test.go`

- [ ] **Step 1: Write the failing test**

`internal/storage/git_test.go`:

```go
package storage

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGitRoot(t *testing.T) {
	base := t.TempDir()
	repo := filepath.Join(base, "repo")
	sub := filepath.Join(repo, "a")
	if err := os.MkdirAll(filepath.Join(repo, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	got, ok := GitRoot(sub)
	if !ok || got != repo {
		t.Fatalf("GitRoot = %q,%v want %q,true", got, ok, repo)
	}
	if _, ok := GitRoot(t.TempDir()); ok {
		t.Fatal("want no git root outside a repo")
	}
}

func TestEnsureGitExcludeAppendsIdempotently(t *testing.T) {
	repo := t.TempDir()
	if err := os.MkdirAll(filepath.Join(repo, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	msg, err := EnsureGitExclude(repo)
	if err != nil {
		t.Fatal(err)
	}
	if msg != "added .rll/ to .git/info/exclude" {
		t.Fatalf("msg = %q", msg)
	}
	data, err := os.ReadFile(filepath.Join(repo, ".git", "info", "exclude"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), ".rll/\n") {
		t.Fatalf("exclude content = %q", data)
	}
	msg, err = EnsureGitExclude(repo)
	if err != nil {
		t.Fatal(err)
	}
	if msg != "already excluded" {
		t.Fatalf("second call msg = %q, want already excluded", msg)
	}
	data2, _ := os.ReadFile(filepath.Join(repo, ".git", "info", "exclude"))
	if strings.Count(string(data2), ".rll/") != 1 {
		t.Fatalf("exclude duplicated: %q", data2)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/storage/ -run Git -v`
Expected: FAIL — `undefined: GitRoot`

- [ ] **Step 3: Write minimal implementation**

`internal/storage/git.go`:

```go
package storage

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// GitRoot returns the repository root if dir is inside a git work tree.
func GitRoot(dir string) (string, bool) {
	for {
		if _, err := os.Stat(filepath.Join(dir, ".git")); err == nil {
			return dir, true
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", false
		}
		dir = parent
	}
}

// EnsureGitExclude idempotently appends ".rll/" to the repository's
// .git/info/exclude file. It never touches tracked files like .gitignore.
func EnsureGitExclude(repoRoot string) (string, error) {
	infoDir := filepath.Join(repoRoot, ".git", "info")
	if err := os.MkdirAll(infoDir, 0o755); err != nil {
		return "", fmt.Errorf("create .git/info: %w", err)
	}
	excludePath := filepath.Join(infoDir, "exclude")
	data, err := os.ReadFile(excludePath)
	if err != nil && !os.IsNotExist(err) {
		return "", fmt.Errorf("read exclude: %w", err)
	}
	content := string(data)
	for _, line := range strings.Split(content, "\n") {
		if strings.TrimSpace(line) == ".rll/" {
			return "already excluded", nil
		}
	}
	f, err := os.OpenFile(excludePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return "", fmt.Errorf("open exclude: %w", err)
	}
	defer f.Close()
	if len(content) > 0 && !strings.HasSuffix(content, "\n") {
		if _, err := f.WriteString("\n"); err != nil {
			return "", err
		}
	}
	if _, err := f.WriteString(".rll/\n"); err != nil {
		return "", err
	}
	return "added .rll/ to .git/info/exclude", nil
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/storage/ -v`
Expected: PASS (all storage tests)

- [ ] **Step 5: Commit**

```bash
git add internal/storage/git.go internal/storage/git_test.go
git commit -m "feat: idempotent .git/info/exclude handling"
```

---

### Task 5: Init templates

**Files:**
- Create: `internal/storage/templates.go`
- Test: `internal/storage/templates_test.go`

- [ ] **Step 1: Write the failing test**

`internal/storage/templates_test.go`:

```go
package storage

import (
	"strings"
	"testing"
)

func TestTemplatesRenderWithFrontmatter(t *testing.T) {
	for name, body := range map[string]string{
		"brief":   BriefTemplate,
		"summary": SummaryTemplate,
	} {
		out := RenderDocument(body)
		if !strings.HasPrefix(out, "---\nschema_version: 1\n---\n") {
			t.Fatalf("%s: missing frontmatter: %q", name, out[:40])
		}
		if !strings.Contains(out, "(TODO)") {
			t.Fatalf("%s: template should contain TODO placeholders", name)
		}
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/storage/ -run TestTemplates -v`
Expected: FAIL — `undefined: BriefTemplate`

- [ ] **Step 3: Write minimal implementation**

`internal/storage/templates.go`:

```go
package storage

// BriefTemplate is the skeleton written to context_brief.md on init.
const BriefTemplate = `# Context Brief

## What this project is

(TODO)

## Current phase

(TODO)

## Source of truth

(TODO)

## Boundaries and constraints

(TODO)
`

// SummaryTemplate is the skeleton written to rolling_summary.md on init.
const SummaryTemplate = `# Rolling Summary

## Current state

(TODO)

## Active risks

(TODO)

## Next step

(TODO)
`
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/storage/ -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/storage/templates.go internal/storage/templates_test.go
git commit -m "feat: init skeleton templates for brief and summary"
```

---

### Task 6: `rll init` command

**Files:**
- Create: `cmd/init.go`
- Modify: `cmd/root.go` (register command)

- [ ] **Step 1: Write the command**

`cmd/init.go`:

```go
package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/eggylan/recall-loom-lite/internal/storage"
)

func newInitCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "init",
		Short: "Create the .rll sidecar skeleton in this project",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			cwd, err := os.Getwd()
			if err != nil {
				return err
			}
			// Prefer the git repository root so .rll lives beside .git.
			target := cwd
			if root, ok := storage.GitRoot(cwd); ok {
				target = root
			}
			rll := filepath.Join(target, storage.DirName)
			if fi, err := os.Stat(rll); err == nil && fi.IsDir() {
				return fmt.Errorf("%s already exists", rll)
			}
			if err := os.MkdirAll(filepath.Join(rll, "daily_logs"), 0o755); err != nil {
				return err
			}
			if err := os.WriteFile(filepath.Join(rll, "context_brief.md"), []byte(storage.RenderDocument(storage.BriefTemplate)), 0o644); err != nil {
				return err
			}
			if err := os.WriteFile(filepath.Join(rll, "rolling_summary.md"), []byte(storage.RenderDocument(storage.SummaryTemplate)), 0o644); err != nil {
				return err
			}
			out := cmd.OutOrStdout()
			fmt.Fprintf(out, "initialized %s\n", rll)
			if root, ok := storage.GitRoot(target); ok {
				msg, err := storage.EnsureGitExclude(root)
				if err != nil {
					fmt.Fprintf(out, "warning: git exclude: %v\n", err)
				} else {
					fmt.Fprintf(out, "git: %s\n", msg)
				}
			}
			fmt.Fprintln(out, "next: fill context_brief.md and rolling_summary.md, e.g.")
			fmt.Fprintln(out, "  rll write brief --stdin   (body from stdin, no frontmatter needed)")
			fmt.Fprintln(out, "  rll write summary --stdin")
			return nil
		},
	}
}
```

Note: the hint text says `--stdin` for readability; the actual interface is plain stdin (no flag), so treat the hint as descriptive. Simpler: change the two hint lines to:

```go
			fmt.Fprintln(out, "next: fill context_brief.md and rolling_summary.md, e.g.")
			fmt.Fprintln(out, "  rll write brief   (body via stdin, no frontmatter needed)")
			fmt.Fprintln(out, "  rll write summary")
```

Use the corrected version.

- [ ] **Step 2: Register in root.go**

In `cmd/root.go`, inside `NewRootCommand`, after the `root := &cobra.Command{...}` literal add:

```go
	root.AddCommand(newInitCmd())
```

- [ ] **Step 3: Smoke test**

```bash
go build -o /tmp/rll . && rm -rf /tmp/rll-smoke && mkdir -p /tmp/rll-smoke && cd /tmp/rll-smoke && git init -q && /tmp/rll init
```

Expected output:

```
initialized /tmp/rll-smoke/.rll
git: added .rll/ to .git/info/exclude
next: fill context_brief.md and rolling_summary.md, e.g.
  rll write brief   (body via stdin, no frontmatter needed)
  rll write summary
```

Then verify: `ls -R /tmp/rll-smoke/.rll && cat /tmp/rll-smoke/.git/info/exclude`
Expected: `.rll/` contains `context_brief.md`, `rolling_summary.md`, `daily_logs/`; exclude file contains `.rll/`.

Run idempotency check: `cd /tmp/rll-smoke && /tmp/rll init; echo "exit=$?"`
Expected: `error: /tmp/rll-smoke/.rll already exists` and exit code 1.

- [ ] **Step 4: Commit**

```bash
git add cmd/
git commit -m "feat: rll init creates sidecar skeleton and git exclude"
```

---

### Task 7: Journal entries

**Files:**
- Create: `internal/journal/entry.go`
- Test: `internal/journal/entry_test.go`

- [ ] **Step 1: Write the failing test**

`internal/journal/entry_test.go`:

```go
package journal

import "testing"

func TestParseEntryHeader(t *testing.T) {
	ts, title, err := ParseEntryHeader("## 10:30 fix login timeout")
	if err != nil {
		t.Fatal(err)
	}
	if ts != "10:30" || title != "fix login timeout" {
		t.Fatalf("got %q %q", ts, title)
	}
}

func TestParseEntryHeaderRejectsBadFormats(t *testing.T) {
	for _, line := range []string{
		"## fix login", // no time
		"## 10:30",     // no title
		"# 10:30 fix",  // wrong heading level
	} {
		if _, _, err := ParseEntryHeader(line); err == nil {
			t.Fatalf("want error for %q", line)
		}
	}
}

func TestParseEntryHeaderShapeOnly(t *testing.T) {
	// Clock sanity is intentionally NOT validated (decision Q13/Q15).
	if _, _, err := ParseEntryHeader("## 25:99 out of range"); err != nil {
		t.Fatalf("shape-valid header rejected: %v", err)
	}
}

func TestRenderEntry(t *testing.T) {
	out := RenderEntry(Entry{Time: "10:30", Title: "fix login", Body: "done\n\ndetails"})
	want := "## 10:30 fix login\n\ndone\n\ndetails\n"
	if out != want {
		t.Fatalf("RenderEntry = %q, want %q", out, want)
	}
	if RenderEntry(Entry{Time: "09:00", Title: "empty"}) != "## 09:00 empty\n" {
		t.Fatal("empty body render wrong")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/journal/ -v`
Expected: FAIL — package does not exist / `undefined: ParseEntryHeader`

- [ ] **Step 3: Write minimal implementation**

`internal/journal/entry.go`:

```go
package journal

import (
	"fmt"
	"regexp"
	"strings"
)

// Entry is one milestone record inside a daily log.
type Entry struct {
	Time  string // HH:MM
	Title string
	Body  string
}

var entryHeaderRe = regexp.MustCompile(`^## (\d{2}:\d{2}) (.+)$`)

// IsEntryHeader reports whether a line opens a new entry.
func IsEntryHeader(line string) bool {
	return entryHeaderRe.MatchString(strings.TrimRight(line, "\r"))
}

// ParseEntryHeader extracts time and title from an entry header line.
// Only digit shape is checked; clock sanity is intentionally not validated.
func ParseEntryHeader(line string) (timeStr, title string, err error) {
	m := entryHeaderRe.FindStringSubmatch(strings.TrimRight(line, "\r"))
	if m == nil {
		return "", "", fmt.Errorf("invalid entry header: %q", line)
	}
	return m[1], m[2], nil
}

// RenderEntry formats an entry as markdown.
func RenderEntry(e Entry) string {
	body := strings.TrimSpace(e.Body)
	if body == "" {
		return fmt.Sprintf("## %s %s\n", e.Time, e.Title)
	}
	return fmt.Sprintf("## %s %s\n\n%s\n", e.Time, e.Title, body)
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/journal/ -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/journal/
git commit -m "feat: daily log entry parse and render"
```

---

### Task 8: Journal files (List / Latest / Append / Parse)

**Files:**
- Create: `internal/journal/journal.go`
- Test: `internal/journal/journal_test.go`

- [ ] **Step 1: Write the failing test**

`internal/journal/journal_test.go`:

```go
package journal

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestListSortedAndLatest(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "daily_logs")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"2026-09-08.md", "2026-09-01.md", "notes.txt"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	logs, err := List(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(logs) != 2 || logs[0].Date != "2026-09-01" || logs[1].Date != "2026-09-08" {
		t.Fatalf("List = %+v", logs)
	}
	latest, err := Latest(root)
	if err != nil || latest == nil || latest.Date != "2026-09-08" {
		t.Fatalf("Latest = %+v, %v", latest, err)
	}
	none, err := Latest(t.TempDir())
	if err != nil || none != nil {
		t.Fatalf("empty Latest = %+v, %v", none, err)
	}
}

func TestAppendCreatesThenAppends(t *testing.T) {
	root := t.TempDir()
	if _, err := Append(root, "2026-09-08", Entry{Time: "10:30", Title: "first", Body: "b1"}); err != nil {
		t.Fatal(err)
	}
	if _, err := Append(root, "2026-09-08", Entry{Time: "11:00", Title: "second", Body: "b2"}); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(root, "daily_logs", "2026-09-08.md"))
	if err != nil {
		t.Fatal(err)
	}
	content := string(data)
	if !strings.HasPrefix(content, "---\nschema_version: 1\n---\n\n## 10:30 first\n") {
		t.Fatalf("content = %q", content)
	}
	if !strings.Contains(content, "## 11:00 second\n\nb2\n") {
		t.Fatalf("second entry missing: %q", content)
	}
	if strings.Index(content, "## 10:30") > strings.Index(content, "## 11:00") {
		t.Fatal("entries out of order")
	}
}

func TestAppendRejectsBadDate(t *testing.T) {
	if _, err := Append(t.TempDir(), "2026-9-8", Entry{Time: "10:00", Title: "t"}); err == nil {
		t.Fatal("want error for non-canonical date")
	}
}

func TestParseEntries(t *testing.T) {
	content := "---\nschema_version: 1\n---\n\n## 10:30 first\n\nbody one\n\n## 11:00 second\n\nbody two\n"
	entries, err := Parse(content)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 2 {
		t.Fatalf("entries = %+v", entries)
	}
	if entries[0].Title != "first" || entries[0].Body != "body one" {
		t.Fatalf("entry0 = %+v", entries[0])
	}
	if entries[1].Time != "11:00" || entries[1].Body != "body two" {
		t.Fatalf("entry1 = %+v", entries[1])
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/journal/ -v`
Expected: FAIL — `undefined: List`

- [ ] **Step 3: Write minimal implementation**

`internal/journal/journal.go`:

```go
package journal

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/eggylan/recall-loom-lite/internal/storage"
)

// LogDirName is the directory holding active daily logs.
const LogDirName = "daily_logs"

// ArchiveDirName is the directory holding archived daily logs.
const ArchiveDirName = "archive"

var dateFileRe = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}\.md$`)

// LogFile is one dated daily log.
type LogFile struct {
	Date string // YYYY-MM-DD
	Path string
}

// ValidDateFileName reports whether name is a canonical YYYY-MM-DD.md file name.
func ValidDateFileName(name string) bool {
	return dateFileRe.MatchString(name)
}

// List returns all active daily logs sorted by date.
// A missing daily_logs directory yields an empty slice, not an error.
func List(rllRoot string) ([]LogFile, error) {
	return listDir(filepath.Join(rllRoot, LogDirName))
}

// ListArchive returns all archived daily logs sorted by date.
func ListArchive(rllRoot string) ([]LogFile, error) {
	return listDir(filepath.Join(rllRoot, ArchiveDirName))
}

func listDir(dir string) ([]LogFile, error) {
	entries, err := os.ReadDir(dir)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var logs []LogFile
	for _, e := range entries {
		if e.IsDir() || !dateFileRe.MatchString(e.Name()) {
			continue
		}
		logs = append(logs, LogFile{
			Date: strings.TrimSuffix(e.Name(), ".md"),
			Path: filepath.Join(dir, e.Name()),
		})
	}
	sort.Slice(logs, func(i, j int) bool { return logs[i].Date < logs[j].Date })
	return logs, nil
}

// Latest returns the newest active daily log, or nil if none exists.
func Latest(rllRoot string) (*LogFile, error) {
	logs, err := List(rllRoot)
	if err != nil {
		return nil, err
	}
	if len(logs) == 0 {
		return nil, nil
	}
	return &logs[len(logs)-1], nil
}

// Append writes an entry to the log for the given date, creating the file
// (with frontmatter) when needed. Returns the written file path.
func Append(rllRoot, date string, e Entry) (string, error) {
	if !dateFileRe.MatchString(date + ".md") {
		return "", fmt.Errorf("invalid date %q, want YYYY-MM-DD", date)
	}
	dir := filepath.Join(rllRoot, LogDirName)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	path := filepath.Join(dir, date+".md")
	data, err := os.ReadFile(path)
	var content string
	switch {
	case os.IsNotExist(err):
		content = storage.RenderDocument("") + "\n" + RenderEntry(e)
	case err != nil:
		return "", err
	default:
		content = strings.TrimRight(string(data), "\n") + "\n\n" + RenderEntry(e)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		return "", err
	}
	return path, nil
}

// Parse extracts entries from a daily log's content.
func Parse(content string) ([]Entry, error) {
	doc, err := storage.ParseDocument(content)
	if err != nil {
		return nil, err
	}
	lines := strings.Split(doc.Body, "\n")
	var entries []Entry
	for _, line := range lines {
		if IsEntryHeader(line) {
			ts, title, err := ParseEntryHeader(line)
			if err != nil {
				return nil, err
			}
			entries = append(entries, Entry{Time: ts, Title: title})
			continue
		}
		if len(entries) > 0 {
			i := len(entries) - 1
			if entries[i].Body == "" {
				entries[i].Body = line
			} else {
				entries[i].Body += "\n" + line
			}
		}
	}
	for i := range entries {
		entries[i].Body = strings.TrimSpace(entries[i].Body)
	}
	return entries, nil
}

// ReadEntries loads and parses a log file from disk.
func ReadEntries(log LogFile) ([]Entry, error) {
	data, err := os.ReadFile(log.Path)
	if err != nil {
		return nil, err
	}
	return Parse(string(data))
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/journal/ -v`
Expected: PASS (all journal tests)

- [ ] **Step 5: Commit**

```bash
git add internal/journal/journal.go internal/journal/journal_test.go
git commit -m "feat: daily log file listing, appending, and parsing"
```

---

### Task 9: `rll log append` command + input util

**Files:**
- Create: `cmd/log.go`, `cmd/util.go`
- Modify: `cmd/root.go` (register)

- [ ] **Step 1: Write the input helper**

`cmd/util.go`:

```go
package cmd

import (
	"io"
	"os"

	"github.com/spf13/cobra"
)

// readInput reads command input from --file, or stdin when no file is given.
func readInput(cmd *cobra.Command, file string) (string, error) {
	var r io.Reader
	if file != "" {
		f, err := os.Open(file)
		if err != nil {
			return "", err
		}
		defer f.Close()
		r = f
	} else {
		r = cmd.InOrStdin()
	}
	data, err := io.ReadAll(r)
	if err != nil {
		return "", err
	}
	return string(data), nil
}
```

- [ ] **Step 2: Write the command**

`cmd/log.go`:

```go
package cmd

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/eggylan/recall-loom-lite/internal/journal"
	"github.com/eggylan/recall-loom-lite/internal/storage"
)

func newLogCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "log",
		Short: "Daily log operations",
	}
	cmd.AddCommand(newLogAppendCmd())
	return cmd
}

func newLogAppendCmd() *cobra.Command {
	var (
		title  string
		date   string
		ts     string
		dryRun bool
		file   string
	)
	cmd := &cobra.Command{
		Use:   "append --title TITLE",
		Short: "Append a milestone entry to the daily log (body from stdin or --file)",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			body, err := readInput(cmd, file)
			if err != nil {
				return err
			}
			now := time.Now()
			if date == "" {
				date = now.Format("2006-01-02")
			}
			if ts == "" {
				ts = now.Format("15:04")
			}
			e := journal.Entry{Time: ts, Title: title, Body: body}
			if dryRun {
				fmt.Fprint(cmd.OutOrStdout(), journal.RenderEntry(e))
				return nil
			}
			root, err := storage.CwdRoot()
			if err != nil {
				return err
			}
			path, err := journal.Append(root, date, e)
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "appended to %s\n", path)
			return nil
		},
	}
	cmd.Flags().StringVarP(&title, "title", "t", "", "entry title (required)")
	cmd.Flags().StringVar(&date, "date", "", "log date YYYY-MM-DD (default: today, local time)")
	cmd.Flags().StringVar(&ts, "time", "", "entry time HH:MM (default: now, local time)")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "print the rendered entry instead of writing")
	cmd.Flags().StringVar(&file, "file", "", "read entry body from this file (default: stdin)")
	_ = cmd.MarkFlagRequired("title")
	return cmd
}
```

- [ ] **Step 3: Register in root.go**

In `cmd/root.go` add next to the init registration:

```go
	root.AddCommand(newLogCmd())
```

- [ ] **Step 4: Smoke test**

```bash
go build -o /tmp/rll . && cd /tmp/rll-smoke && cat <<'EOF' | /tmp/rll log append --title "fix login timeout"
root cause was session expiry; added refresh flow
tests: 12 passed
EOF
```

Expected: `appended to /tmp/rll-smoke/.rll/daily_logs/YYYY-MM-DD.md` (today's date).

Backfill + dry-run check:

```bash
cd /tmp/rll-smoke && printf 'backfilled work' | /tmp/rll log append --title "yesterday work" --date 2026-09-07 --time 18:00 --dry-run
```

Expected output (nothing written):

```
## 18:00 yesterday work

backfilled work
```

Verify file state: `ls /tmp/rll-smoke/.rll/daily_logs/` — contains only today's file (dry-run wrote nothing).

- [ ] **Step 5: Commit**

```bash
git add cmd/
git commit -m "feat: rll log append with stdin input and backfill overrides"
```

---

### Task 10: `rll write` command

**Files:**
- Create: `cmd/write.go`
- Modify: `cmd/root.go` (register)

- [ ] **Step 1: Write the command**

`cmd/write.go`:

```go
package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/eggylan/recall-loom-lite/internal/storage"
)

var writeTargets = map[string]string{
	"brief":    "context_brief.md",
	"summary":  "rolling_summary.md",
	"protocol": "update_protocol.md",
}

func newWriteCmd() *cobra.Command {
	var (
		dryRun bool
		file   string
	)
	cmd := &cobra.Command{
		Use:       "write <brief|summary|protocol>",
		Short:     "Overwrite a managed document (body from stdin or --file, frontmatter added automatically)",
		Args:      cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
		ValidArgs: []string{"brief", "summary", "protocol"},
		RunE: func(cmd *cobra.Command, args []string) error {
			target := args[0]
			body, err := readInput(cmd, file)
			if err != nil {
				return err
			}
			if strings.TrimSpace(body) == "" {
				return fmt.Errorf("refusing to write empty content")
			}
			content := storage.RenderDocument(body)
			if dryRun {
				fmt.Fprint(cmd.OutOrStdout(), content)
				return nil
			}
			root, err := storage.CwdRoot()
			if err != nil {
				return err
			}
			path := filepath.Join(root, writeTargets[target])
			if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "wrote %s\n", path)
			return nil
		},
	}
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "print the rendered document instead of writing")
	cmd.Flags().StringVar(&file, "file", "", "read body from this file (default: stdin)")
	return cmd
}
```

- [ ] **Step 2: Register in root.go**

```go
	root.AddCommand(newWriteCmd())
```

- [ ] **Step 3: Smoke test**

```bash
go build -o /tmp/rll . && cd /tmp/rll-smoke && cat <<'EOF' | /tmp/rll write summary
# Rolling Summary

## Current state

CLI core commands implemented and smoke tested.

## Active risks

None.

## Next step

Ship validate command.
EOF
```

Expected: `wrote /tmp/rll-smoke/.rll/rolling_summary.md`

Verify: `head -3 /tmp/rll-smoke/.rll/rolling_summary.md`

```
---
schema_version: 1
---
```

Empty-guard check: `cd /tmp/rll-smoke && printf '   ' | /tmp/rll write brief; echo "exit=$?"`
Expected: `error: refusing to write empty content`, exit 1.

Invalid target check: `/tmp/rll write diary`
Expected: cobra error `invalid argument "diary"`, exit 1.

- [ ] **Step 4: Commit**

```bash
git add cmd/write.go cmd/root.go
git commit -m "feat: rll write for brief/summary/protocol with auto frontmatter"
```

---

### Task 11: validate package + `rll validate` command

**Files:**
- Create: `internal/validate/validate.go`
- Test: `internal/validate/validate_test.go`
- Create: `cmd/validate.go`
- Modify: `cmd/root.go` (register)

- [ ] **Step 1: Write the failing test**

`internal/validate/validate_test.go`:

```go
package validate

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func has(findings []Finding, severity Severity, msgPart string) bool {
	for _, f := range findings {
		if f.Severity == severity && strings.Contains(f.Message, msgPart) {
			return true
		}
	}
	return false
}

func TestRunHappyPath(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "context_brief.md"), "---\nschema_version: 1\n---\n\n# ok\n")
	writeFile(t, filepath.Join(root, "rolling_summary.md"), "---\nschema_version: 1\n---\n\n# ok\n")
	writeFile(t, filepath.Join(root, "daily_logs", "2026-09-08.md"),
		"---\nschema_version: 1\n---\n\n## 10:30 first\n\nbody\n")
	findings := Run(root)
	if len(findings) != 0 {
		t.Fatalf("findings = %+v", findings)
	}
}

func TestRunDetectsProblems(t *testing.T) {
	root := t.TempDir()
	// missing schema_version
	writeFile(t, filepath.Join(root, "context_brief.md"), "# no frontmatter\n")
	// ok
	writeFile(t, filepath.Join(root, "rolling_summary.md"), "---\nschema_version: 1\n---\n\n# ok\n")
	// bad filename (also covers the "2026-9-8 duplicate date" case)
	writeFile(t, filepath.Join(root, "daily_logs", "2026-9-8.md"), "x")
	// bad entry header inside a valid-named log
	writeFile(t, filepath.Join(root, "daily_logs", "2026-09-08.md"),
		"---\nschema_version: 1\n---\n\n## bad header no time\n")
	// stray file
	writeFile(t, filepath.Join(root, "notes.txt"), "x")
	// bad archive filename
	writeFile(t, filepath.Join(root, "archive", "old.md"), "x")

	findings := Run(root)
	if !has(findings, Error, "context_brief.md") || !has(findings, Error, "schema_version") {
		t.Fatalf("missing frontmatter not flagged: %+v", findings)
	}
	if !has(findings, Error, "2026-9-8.md") {
		t.Fatalf("bad log filename not flagged: %+v", findings)
	}
	if !has(findings, Error, "invalid entry header") {
		t.Fatalf("bad entry header not flagged: %+v", findings)
	}
	if !has(findings, Warning, "notes.txt") {
		t.Fatalf("stray file not warned: %+v", findings)
	}
	if !has(findings, Error, "archive") {
		t.Fatalf("bad archive filename not flagged: %+v", findings)
	}
}

func TestRunMissingCoreDocs(t *testing.T) {
	findings := Run(t.TempDir())
	if !has(findings, Error, "context_brief.md") || !has(findings, Error, "rolling_summary.md") {
		t.Fatalf("missing core docs not flagged: %+v", findings)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/validate/ -v`
Expected: FAIL — `undefined: Run`

- [ ] **Step 3: Write minimal implementation**

`internal/validate/validate.go`:

```go
package validate

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/eggylan/recall-loom-lite/internal/journal"
	"github.com/eggylan/recall-loom-lite/internal/storage"
)

// Severity classifies a finding.
type Severity string

const (
	// Error means the sidecar violates the format contract.
	Error Severity = "error"
	// Warning means something looks unusual but is allowed.
	Warning Severity = "warning"
)

// Finding is one validation result.
type Finding struct {
	Path     string
	Severity Severity
	Message  string
}

// Run performs the five format-only checks against an .rll root.
// It never inspects content semantics.
func Run(rllRoot string) []Finding {
	var findings []Finding
	add := func(sev Severity, rel, format string, a ...any) {
		findings = append(findings, Finding{
			Path:     rel,
			Severity: sev,
			Message:  fmt.Sprintf(format, a...),
		})
	}

	// Check 1: core docs + optional protocol have valid frontmatter with schema_version.
	for _, name := range []string{"context_brief.md", "rolling_summary.md", "update_protocol.md"} {
		p := filepath.Join(rllRoot, name)
		data, err := os.ReadFile(p)
		if os.IsNotExist(err) {
			if name != "update_protocol.md" {
				add(Error, name, "missing required file")
			}
			continue
		}
		if err != nil {
			add(Error, name, "unreadable: %v", err)
			continue
		}
		doc, err := storage.ParseDocument(string(data))
		if err != nil {
			add(Error, name, "%v", err)
			continue
		}
		if !doc.HasSchemaVersion() {
			add(Error, name, "frontmatter missing schema_version: %d", storage.SchemaVersion)
		}
	}

	// Checks 2+3: log filenames canonical (rejects 2026-9-8.md as duplicate-date form).
	checkLogDirNames(rllRoot, journal.LogDirName, &findings)
	checkLogDirNames(rllRoot, journal.ArchiveDirName, &findings)

	// Check 4: entry headers well-formed inside active logs.
	logs, _ := journal.List(rllRoot)
	for _, log := range logs {
		data, err := os.ReadFile(log.Path)
		if err != nil {
			add(Error, relTo(rllRoot, log.Path), "%v", err)
			continue
		}
		doc, err := storage.ParseDocument(string(data))
		if err != nil {
			add(Error, relTo(rllRoot, log.Path), "%v", err)
			continue
		}
		if !doc.HasSchemaVersion() {
			add(Error, relTo(rllRoot, log.Path), "frontmatter missing schema_version: %d", storage.SchemaVersion)
		}
		for _, line := range strings.Split(doc.Body, "\n") {
			trimmed := strings.TrimRight(line, "\r")
			if strings.HasPrefix(trimmed, "## ") && !journal.IsEntryHeader(trimmed) {
				add(Error, relTo(rllRoot, log.Path), "invalid entry header: %q (want \"## HH:MM title\")", trimmed)
			}
		}
	}

	// Check 5: stray files at the .rll root are a warning.
	allowed := map[string]bool{
		"context_brief.md": true, "rolling_summary.md": true,
		"update_protocol.md": true, journal.LogDirName: true, journal.ArchiveDirName: true,
	}
	entries, err := os.ReadDir(rllRoot)
	if err == nil {
		for _, e := range entries {
			if !allowed[e.Name()] {
				add(Warning, e.Name(), "unrecognized file in .rll root")
			}
		}
	}
	return findings
}

func checkLogDirNames(rllRoot, dirName string, findings *[]Finding) {
	dir := filepath.Join(rllRoot, dirName)
	entries, err := os.ReadDir(dir)
	if os.IsNotExist(err) {
		return
	}
	if err != nil {
		*findings = append(*findings, Finding{Path: dirName, Severity: Error, Message: err.Error()})
		return
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		if !journal.ValidDateFileName(e.Name()) {
			*findings = append(*findings, Finding{
				Path:     filepath.Join(dirName, e.Name()),
				Severity: Error,
				Message:  "invalid daily log filename (want YYYY-MM-DD.md; non-canonical dates like 2026-9-8.md count as conflicts)",
			})
		}
	}
}

func relTo(root, path string) string {
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return path
	}
	return filepath.ToSlash(rel)
}

// HasErrors reports whether any finding is an error.
func HasErrors(findings []Finding) bool {
	for _, f := range findings {
		if f.Severity == Error {
			return true
		}
	}
	return false
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/validate/ -v`
Expected: PASS (3 tests)

- [ ] **Step 5: Write the command**

`cmd/validate.go`:

```go
package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/eggylan/recall-loom-lite/internal/storage"
	"github.com/eggylan/recall-loom-lite/internal/validate"
)

func newValidateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "validate",
		Short: "Check .rll file formats (structure only, no semantics)",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			root, err := storage.CwdRoot()
			if err != nil {
				return err
			}
			findings := validate.Run(root)
			out := cmd.OutOrStdout()
			errCount, warnCount := 0, 0
			for _, f := range findings {
				fmt.Fprintf(out, "%s: %s: %s\n", f.Severity, f.Path, f.Message)
				if f.Severity == validate.Error {
					errCount++
				} else {
					warnCount++
				}
			}
			fmt.Fprintf(out, "%d error(s), %d warning(s)\n", errCount, warnCount)
			if validate.HasErrors(findings) {
				os.Exit(1)
			}
			return nil
		},
	}
	return cmd
}
```

- [ ] **Step 6: Register and smoke test**

Register in `cmd/root.go`: `root.AddCommand(newValidateCmd())`

```bash
go build -o /tmp/rll . && cd /tmp/rll-smoke && /tmp/rll validate
```

Expected (sidecar is healthy from Tasks 6-10): `0 error(s), 0 warning(s)`, exit 0.

Break it:

```bash
printf 'junk' > /tmp/rll-smoke/.rll/notes.txt && cd /tmp/rll-smoke && /tmp/rll validate; echo "exit=$?"; rm /tmp/rll-smoke/.rll/notes.txt
```

Expected: a `warning: notes.txt: unrecognized file in .rll root` line, `0 error(s), 1 warning(s)`, exit 0.

- [ ] **Step 7: Commit**

```bash
git add internal/validate/ cmd/validate.go cmd/root.go
git commit -m "feat: format-only validation with five checks"
```

---

### Task 12: `rll status` command

**Files:**
- Create: `cmd/status.go`
- Modify: `cmd/root.go` (register)

- [ ] **Step 1: Write the command**

`cmd/status.go`:

```go
package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/eggylan/recall-loom-lite/internal/journal"
	"github.com/eggylan/recall-loom-lite/internal/storage"
)

func newStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show a one-screen overview of the project memory",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			root, err := storage.CwdRoot()
			if err != nil {
				return err
			}
			out := cmd.OutOrStdout()
			fmt.Fprintf(out, ".rll root: %s\n", root)

			describe := func(name string, required bool) {
				p := filepath.Join(root, name)
				fi, err := os.Stat(p)
				if os.IsNotExist(err) {
					if required {
						fmt.Fprintf(out, "%s: MISSING\n", name)
					} else {
						fmt.Fprintf(out, "%s: absent\n", name)
					}
					return
				}
				if err != nil {
					fmt.Fprintf(out, "%s: unreadable: %v\n", name, err)
					return
				}
				fmt.Fprintf(out, "%s: %d bytes, modified %s\n", name, fi.Size(), fi.ModTime().Format("2006-01-02 15:04"))
			}
			describe("context_brief.md", true)
			describe("rolling_summary.md", true)
			describe("update_protocol.md", false)

			logs, err := journal.List(root)
			if err != nil {
				return err
			}
			archived, err := journal.ListArchive(root)
			if err != nil {
				return err
			}
			fmt.Fprintf(out, "daily logs: %d active, %d archived\n", len(logs), len(archived))
			if latest, _ := journal.Latest(root); latest != nil {
				entries, err := journal.ReadEntries(*latest)
				if err != nil {
					return err
				}
				fmt.Fprintf(out, "latest log: %s (%d entries)\n", latest.Date, len(entries))
			} else {
				fmt.Fprintln(out, "latest log: none")
			}
			return nil
		},
	}
}
```

- [ ] **Step 2: Register and smoke test**

Register: `root.AddCommand(newStatusCmd())`

```bash
go build -o /tmp/rll . && cd /tmp/rll-smoke && /tmp/rll status
```

Expected shape (values vary):

```
.rll root: /tmp/rll-smoke/.rll
context_brief.md: 247 bytes, modified 2026-09-08 14:12
rolling_summary.md: 189 bytes, modified 2026-09-08 14:20
update_protocol.md: absent
daily logs: 1 active, 0 archived
latest log: 2026-09-08 (1 entries)
```

- [ ] **Step 3: Commit**

```bash
git add cmd/status.go cmd/root.go
git commit -m "feat: rll status overview"
```

---

### Task 13: `rll resume` command

**Files:**
- Create: `cmd/resume.go`
- Modify: `cmd/root.go` (register)

- [ ] **Step 1: Write the command**

`cmd/resume.go`:

```go
package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/eggylan/recall-loom-lite/internal/journal"
	"github.com/eggylan/recall-loom-lite/internal/storage"
)

type resumeJSON struct {
	LatestLogDate  string          `json:"latest_log_date,omitempty"`
	EntryCount     int             `json:"entry_count"`
	UpdateProtocol string          `json:"update_protocol,omitempty"`
	Brief          string          `json:"brief"`
	Summary        string          `json:"summary"`
	LatestLog      []journal.Entry `json:"latest_log,omitempty"`
}

func newResumeCmd() *cobra.Command {
	var asJSON bool
	cmd := &cobra.Command{
		Use:   "resume",
		Short: "Print the full project memory context for a cold start (one tier: everything)",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			root, err := storage.CwdRoot()
			if err != nil {
				return err
			}
			read := func(name string) (string, bool) {
				data, err := os.ReadFile(filepath.Join(root, name))
				if err != nil {
					return "", false
				}
				return string(data), true
			}

			latest, err := journal.Latest(root)
			if err != nil {
				return err
			}
			var entries []journal.Entry
			var latestDate string
			if latest != nil {
				entries, err = journal.ReadEntries(*latest)
				if err != nil {
					return err
				}
				latestDate = latest.Date
			}
			protocol, hasProtocol := read("update_protocol.md")
			brief, _ := read("context_brief.md")
			summary, _ := read("rolling_summary.md")

			if asJSON {
				out := resumeJSON{
					LatestLogDate:  latestDate,
					EntryCount:     len(entries),
					UpdateProtocol: protocol,
					Brief:          brief,
					Summary:        summary,
					LatestLog:      entries,
				}
				enc := json.NewEncoder(cmd.OutOrStdout())
				enc.SetIndent("", "  ")
				return enc.Encode(out)
			}

			out := cmd.OutOrStdout()
			fmt.Fprintln(out, "# rll resume")
			fmt.Fprintln(out)
			if latestDate != "" {
				fmt.Fprintf(out, "latest log: %s (%d entries)\n", latestDate, len(entries))
			} else {
				fmt.Fprintln(out, "latest log: none")
			}
			fmt.Fprintf(out, "protocol: %s\n", map[bool]string{true: "present", false: "absent"}[hasProtocol])
			if hasProtocol {
				fmt.Fprintln(out, "\n## Update Protocol\n")
				fmt.Fprint(out, protocol)
			}
			fmt.Fprintln(out, "\n## Context Brief\n")
			fmt.Fprint(out, brief)
			fmt.Fprintln(out, "\n## Rolling Summary\n")
			fmt.Fprint(out, summary)
			fmt.Fprintln(out, "\n## Daily Log: " + latestDate + "\n")
			if latest != nil {
				data, err := os.ReadFile(latest.Path)
				if err != nil {
					return err
				}
				fmt.Fprint(out, string(data))
			} else {
				fmt.Fprintln(out, "(no daily logs yet)")
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&asJSON, "json", false, "output as JSON")
	return cmd
}
```

- [ ] **Step 2: Register and smoke test**

Register: `root.AddCommand(newResumeCmd())`

```bash
go build -o /tmp/rll . && cd /tmp/rll-smoke && /tmp/rll resume | head -30
```

Expected: Markdown starting with `# rll resume`, then `latest log: ...`, sections `## Update Protocol` (absent here → skipped), `## Context Brief`, `## Rolling Summary`, `## Daily Log: YYYY-MM-DD` containing the entry appended in Task 9.

JSON check: `cd /tmp/rll-smoke && /tmp/rll resume --json | head -15`
Expected: valid JSON with `brief`, `summary`, `latest_log` keys.

- [ ] **Step 3: Commit**

```bash
git add cmd/resume.go cmd/root.go
git commit -m "feat: rll resume outputs full memory context in one tier"
```

---

### Task 14: `rll query` command

**Files:**
- Create: `cmd/query.go`
- Modify: `cmd/root.go` (register)

- [ ] **Step 1: Write the command**

`cmd/query.go`:

```go
package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/eggylan/recall-loom-lite/internal/journal"
	"github.com/eggylan/recall-loom-lite/internal/storage"
)

func newQueryCmd() *cobra.Command {
	var all bool
	cmd := &cobra.Command{
		Use:   "query <text...>",
		Short: "Case-insensitive substring search across rolling_summary and daily logs",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			needle := strings.ToLower(strings.Join(args, " "))
			root, err := storage.CwdRoot()
			if err != nil {
				return err
			}
			out := cmd.OutOrStdout()
			matches := 0

			type source struct {
				rel  string
				path string
			}
			var sources []source
			sources = append(sources, source{rel: "rolling_summary.md", path: filepath.Join(root, "rolling_summary.md")})
			logs, err := journal.List(root)
			if err != nil {
				return err
			}
			for _, l := range logs {
				sources = append(sources, source{rel: journal.LogDirName + "/" + l.Date + ".md", path: l.Path})
			}
			if all {
				archived, err := journal.ListArchive(root)
				if err != nil {
					return err
				}
				for _, l := range archived {
					sources = append(sources, source{rel: journal.ArchiveDirName + "/" + l.Date + ".md", path: l.Path})
				}
			}

			for _, s := range sources {
				data, err := os.ReadFile(s.path)
				if err != nil {
					continue
				}
				var printedHeader bool
				for i, line := range strings.Split(string(data), "\n") {
					if strings.Contains(strings.ToLower(line), needle) {
						if !printedHeader {
							fmt.Fprintf(out, "%s\n", s.rel)
							printedHeader = true
						}
						fmt.Fprintf(out, "  %d: %s\n", i+1, strings.TrimRight(line, "\r"))
						matches++
					}
				}
			}
			if matches == 0 {
				fmt.Fprintln(out, "no matches")
				os.Exit(1)
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&all, "all", false, "also search archived daily logs")
	return cmd
}
```

- [ ] **Step 2: Register and smoke test**

Register: `root.AddCommand(newQueryCmd())`

```bash
go build -o /tmp/rll . && cd /tmp/rll-smoke && /tmp/rll query "CLI core"
```

Expected: a `rolling_summary.md` header line and one numbered match line (from the summary written in Task 10).

No-match check: `cd /tmp/rll-smoke && /tmp/rll query "zzzz"; echo "exit=$?"`
Expected: `no matches`, `exit=1`.

- [ ] **Step 3: Commit**

```bash
git add cmd/query.go cmd/root.go
git commit -m "feat: rll query substring search over summary and logs"
```

---

### Task 15: `rll archive` command

**Files:**
- Create: `cmd/archive.go`
- Modify: `cmd/root.go` (register)

- [ ] **Step 1: Write the command**

`cmd/archive.go`:

```go
package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/eggylan/recall-loom-lite/internal/journal"
	"github.com/eggylan/recall-loom-lite/internal/storage"
)

func newArchiveCmd() *cobra.Command {
	var (
		before string
		apply  bool
	)
	cmd := &cobra.Command{
		Use:   "archive --before YYYY-MM-DD [--apply]",
		Short: "Move daily logs older than a date into .rll/archive (preview by default)",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if !journal.ValidDateFileName(before + ".md") {
				return fmt.Errorf("--before is required and must be YYYY-MM-DD")
			}
			root, err := storage.CwdRoot()
			if err != nil {
				return err
			}
			logs, err := journal.List(root)
			if err != nil {
				return err
			}
			var toMove []journal.LogFile
			for _, l := range logs {
				if l.Date < before {
					toMove = append(toMove, l)
				}
			}
			out := cmd.OutOrStdout()
			if len(toMove) == 0 {
				fmt.Fprintln(out, "nothing to archive")
				return nil
			}
			if !apply {
				for _, l := range toMove {
					fmt.Fprintf(out, "would move: %s/%s.md -> %s/%s.md\n",
						journal.LogDirName, l.Date, journal.ArchiveDirName, l.Date)
				}
				fmt.Fprintf(out, "total: %d file(s) (use --apply to move)\n", len(toMove))
				return nil
			}
			dstDir := filepath.Join(root, journal.ArchiveDirName)
			if err := os.MkdirAll(dstDir, 0o755); err != nil {
				return err
			}
			for _, l := range toMove {
				dst := filepath.Join(dstDir, l.Date+".md")
				if err := os.Rename(l.Path, dst); err != nil {
					return err
				}
				fmt.Fprintf(out, "moved: %s -> %s\n", l.Path, dst)
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&before, "before", "", "archive logs with dates strictly before this YYYY-MM-DD (required)")
	cmd.Flags().BoolVar(&apply, "apply", false, "actually move files (default: preview only)")
	return cmd
}
```

- [ ] **Step 2: Register and smoke test**

Register: `root.AddCommand(newArchiveCmd())`

```bash
go build -o /tmp/rll . && cd /tmp/rll-smoke && printf 'old work' | /tmp/rll log append --title "old milestone" --date 2026-01-15 --time 09:00 && /tmp/rll archive --before 2026-06-01
```

Expected preview:

```
would move: daily_logs/2026-01-15.md -> archive/2026-01-15.md
total: 1 file(s) (use --apply to move)
```

Apply and verify isolation:

```bash
cd /tmp/rll-smoke && /tmp/rll archive --before 2026-06-01 --apply && ls .rll/archive/ && /tmp/rll query "old milestone"; echo "exit=$?"
```

Expected: `moved: ...` line; archive dir contains `2026-01-15.md`; query says `no matches` exit=1 (active logs only). Then `/tmp/rll query --all "old milestone"` finds it in `archive/2026-01-15.md`.

- [ ] **Step 3: Commit**

```bash
git add cmd/archive.go cmd/root.go
git commit -m "feat: rll archive with explicit date and preview/apply"
```

---

### Task 16: Skill package, LICENSE, NOTICE, README

**Files:**
- Create: `skill/SKILL.md`, `LICENSE`, `NOTICE`, `README.md`

- [ ] **Step 1: Fetch the Apache-2.0 license text**

```bash
curl -fsSL https://www.apache.org/licenses/LICENSE-2.0.txt -o LICENSE
```

Expected: `LICENSE` is ~11KB starting with `Apache License`.

- [ ] **Step 2: Write NOTICE**

`NOTICE`:

```
RecallLoom Lite (rll)
Copyright 2026 eggylan

This product is an independent Go reimplementation inspired by RecallLoom,
licensed under Apache License 2.0. It reuses no source code from the
original project.
```

- [ ] **Step 3: Write skill/SKILL.md**

`skill/SKILL.md` (English, this IS the whole skill package):

````markdown
---
name: rll
description: Use when continuing a project, restoring project context, maintaining file-based project memory, recording milestone progress, or preparing a session handoff. Drives the rll CLI (RecallLoom Lite) for lightweight zero-state project memory.
---

# RecallLoom Lite (rll)

rll keeps project memory in plain Markdown files under `.rll/` so any session,
model, or tool can pick up where the last one stopped.

**Requirement:** the `rll` binary must be on PATH. If missing, tell the user to
install it from the project's GitHub Releases and stop — never hand-build sidecar files.

## Cold start

Run `rll resume` first. It prints the entire memory in one shot:
update protocol (if any), context brief, rolling summary, and the latest daily
log. Read it, then act. Do not read the files again — resume already gave you everything.

## Command cheatsheet

| Command | What it does |
|---|---|
| `rll init` | Create the `.rll/` skeleton (refuse if it exists) |
| `rll resume` | Print the full memory context for a cold start |
| `rll status` | One-screen overview (file sizes, latest log, entry counts) |
| `rll log append --title "..."` | Append a milestone entry to today's daily log (body via stdin) |
| `rll write brief / summary / protocol` | Overwrite a managed document (body via stdin) |
| `rll query <text...>` | Case-insensitive substring search over summary + active logs (`--all` includes archive) |
| `rll validate` | Check file formats only (5 structural checks) |
| `rll archive --before YYYY-MM-DD` | Preview moving old logs to `.rll/archive/` (`--apply` to move) |

All write commands accept `--dry-run` to preview rendered output and `--file PATH`
instead of stdin. Timestamps and dates default to local time; override with
`--date`/`--time` when backfilling.

## Deciding where a fact goes

Ask: what kind of change is this?

| Content | Target | Command |
|---|---|---|
| Durable rule, boundary, source of truth, project identity | `context_brief.md` | `rll write brief` |
| What is true right now: phase, active risks, next step | `rolling_summary.md` | `rll write summary` |
| Completed milestone, confirmed decision, validation result | daily log | `rll log append --title "..."` |
| Project-local read/write workflow override | `update_protocol.md` | `rll write protocol` |

One fact goes to exactly one place. When an event spans layers, split it:
decision → brief (if durable) or log (as evidence), current picture → summary.

## When NOT to write

Do not touch memory files when:

- you only ran `rll resume` or answered a question
- you explored without reaching a stable conclusion
- the change is wording-only with no new durable fact
- the discussion is still unstable — wait for it to settle

no_write is a successful outcome, not a failure.

## First attach

If `rll resume` fails with "no .rll directory found", ask the user whether to
initialize. If confirmed: run `rll init`, then interview the user (what is this
project, current phase, source of truth, constraints) and fill the skeleton:

```bash
cat <<'EOF' | rll write brief
# Context Brief

## What this project is
...
EOF
```

Body must be pure Markdown WITHOUT frontmatter — `rll write` adds it.

## Zero-state philosophy

There is no state file, no locking, no receipt. Files are the only truth.
Users may hand-edit any file at any time; the only guard is `rll validate`
checking format. If files look wrong, fix them with `rll write` or directly,
then run `rll validate`.
````

- [ ] **Step 4: Write README.md (Chinese)**

`README.md`:

````markdown
# RecallLoom Lite (rll)

让项目自己记住自己——轻量、零状态、单二进制的 AI 协作项目记忆层。

rll 在项目的 `.rll/` 目录里用纯 Markdown 维护三份记忆：

- `context_brief.md` — 项目定位、阶段、边界（稳定框架，很少变）
- `rolling_summary.md` — 当前状态快照（每次覆盖式更新）
- `daily_logs/YYYY-MM-DD.md` — 里程碑日志（只追加）

没有 state.json、没有收据、没有版本绑定、没有安全层。文件即全部事实，
允许随时手工修改，`rll validate` 只做五项纯格式检查。`.rll/` 默认通过
`.git/info/exclude` 排除在 git 之外，不污染仓库。

## 安装

从 GitHub Releases 下载对应平台的二进制（linux / darwin / windows × amd64 / arm64），
放到 PATH 上。Skill 包（`rll_skill` 压缩包）解压到宿主的 skills 目录，例如：

```bash
mkdir -p ~/.config/opencode/skills/rll && tar -xzf rll_skill.tar.gz -C ~/.config/opencode/skills/rll
```

## 快速开始

```bash
cd your-project
rll init                     # 创建 .rll/ 骨架并配置 git 排除
cat <<'EOF' | rll write brief
# Context Brief

## What this project is

（让 agent 替你采访并填写）
EOF
rll log append --title "init memory"   # 正文从 stdin 读
rll status
rll resume                    # 冷启动：一次输出全部记忆
```

## 命令一览

| 命令 | 作用 |
|---|---|
| `rll init` | 初始化 `.rll/` 骨架 |
| `rll resume` | 输出全部记忆上下文（冷启动用） |
| `rll status` | 一屏概览 |
| `rll log append --title T` | 追加里程碑日志（stdin 传正文） |
| `rll write brief/summary/protocol` | 覆盖式更新受管文档 |
| `rll query 关键词` | 子串搜索 summary 与日志 |
| `rll validate` | 五项纯格式检查 |
| `rll archive --before 日期` | 归档旧日志（默认预览，--apply 执行） |

所有写命令支持 `--dry-run` 预览与 `--file` 从文件读入。时间戳默认取本机
时间，可用 `--date`/`--time` 补录。

## 设计取舍

- **零状态**：游标、最新日志等全部由目录扫描即时得出，永无同步错误。
- **无安全层**：删除原版 RecallLoom 的收据绑定与写保护，模型一次写入
  成功率优先；记录被人为改动属于正常使用，由开发者自行负责。
- **CLI 与 Skill 分离**：同一 Release 携带全部平台二进制与 skill 压缩包。

本项目是受 RecallLoom 启发的独立 Go 重写（见 NOTICE），不与其数据格式兼容。

## 许可证

Apache-2.0
````

- [ ] **Step 5: Verify and commit**

```bash
go build -o /tmp/rll . && /tmp/rll --help
```

Expected: help lists init, status, log, write, validate, resume, query, archive.

```bash
git add skill/ LICENSE NOTICE README.md
git commit -m "feat: skill package, license, and Chinese README"
```

---

### Task 17: goreleaser, release workflow, end-to-end verification

**Files:**
- Create: `.goreleaser.yaml`, `.github/workflows/release.yml`

- [ ] **Step 1: Write .goreleaser.yaml**

`.goreleaser.yaml`:

```yaml
version: 2

project_name: rll

before:
  hooks:
    - go mod tidy

builds:
  - id: rll
    main: .
    binary: rll
    env: [CGO_ENABLED=0]
    goos: [linux, darwin, windows]
    goarch: [amd64, arm64]
    ldflags:
      - -s -w -X main.version={{.Version}}

archives:
  - id: binaries
    ids: [rll]
    name_template: "{{ .ProjectName }}_{{ .Os }}_{{ .Arch }}"
    formats: [tar.gz]
    format_overrides:
      - goos: windows
        formats: [zip]
  - id: skill
    meta: true
    name_template: "{{ .ProjectName }}_skill"
    formats: [tar.gz]
    files:
      - src: "skill/*"
        dst: "rll-skill"

checksum:
  name_template: "checksums.txt"

changelog:
  sort: asc
```

- [ ] **Step 2: Verify with goreleaser**

```bash
goreleaser check && goreleaser release --snapshot --clean && ls dist/
```

Expected: `dist/` contains platform archives like `rll_linux_amd64.tar.gz` and `rll_skill.tar.gz`, plus `checksums.txt`. Verify skill archive content: `tar -tzf dist/rll_skill.tar.gz`
Expected: contains `rll-skill/SKILL.md`.

If `goreleaser` is not installed: `go install github.com/goreleaser/goreleaser/v2@latest` (binary lands in `$(go env GOPATH)/bin`).

- [ ] **Step 3: Write the release workflow**

`.github/workflows/release.yml`:

```yaml
name: Release

on:
  push:
    tags: ["v*"]

permissions:
  contents: write

jobs:
  release:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
        with:
          fetch-depth: 0
      - uses: actions/setup-go@v5
        with:
          go-version: "1.22"
      - uses: goreleaser/goreleaser-action@v6
        with:
          distribution: goreleaser
          version: latest
          args: release --clean
        env:
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
```

- [ ] **Step 4: End-to-end verification**

Run the full happy path against a fresh build:

```bash
go build -o /tmp/rll . && rm -rf /tmp/rll-e2e && mkdir -p /tmp/rll-e2e && cd /tmp/rll-e2e && git init -q && /tmp/rll init && \
cat <<'EOF' | /tmp/rll write brief
# Context Brief

## What this project is

Demo project for e2e verification.
EOF
cat <<'EOF' | /tmp/rll write summary
# Rolling Summary

## Current state

Fresh start.

## Next step

Run e2e checks.
EOF
cat <<'EOF' | /tmp/rll log append --title "e2e milestone"
all commands wired
EOF
printf 'first protocol line' | /tmp/rll write protocol && \
/tmp/rll status && /tmp/rll validate && /tmp/rll resume | head -40 && \
/tmp/rll query "Fresh start" && /tmp/rll archive --before 2020-01-01 && \
git status --porcelain | wc -l
```

Expected in order:
1. init output with `git: added .rll/ to .git/info/exclude`
2. `wrote ...` lines for brief, summary, protocol, and the log append
3. status overview mentioning all three docs and 1 active log
4. `0 error(s), 0 warning(s)`
5. resume Markdown containing `## Update Protocol`, `## Context Brief`, `## Rolling Summary`, `## Daily Log:`
6. query match in `rolling_summary.md`
7. `nothing to archive`
8. `0` — nothing in `.rll/` is tracked by git (exclude works)

Also verify unit tests one final time:

```bash
go test ./... && go vet ./...
```

Expected: all PASS, no vet findings.

- [ ] **Step 5: Commit**

```bash
git add .goreleaser.yaml .github/
git commit -m "feat: goreleaser config and release workflow"
```

---

## Self-Review (completed during planning)

**Spec coverage:** all 19 grilled decisions map to tasks — Q1 (zero state: Tasks 2, 8 derive everything by scan), Q2 (frontmatter: Task 3; `## HH:MM` entries: Task 7), Q3 (8 commands: Tasks 6, 9-15), Q4 (stdin + dry-run: Tasks 9, 10), Q5 (single SKILL.md: Task 16), Q6 (`.rll/` + info/exclude: Tasks 2, 4, 6), Q7 (one-tier full resume: Task 13), Q8 (query/archive semantics: Tasks 14, 15), Q9 (skeleton init + skill-guided fill: Tasks 5, 6, 16), Q10 (no config, optional protocol, English CLI, local time: Tasks 3, 6, 13), Q11 (separate skill artifact in same Release: Task 17), Q12 (cobra+goreleaser+yaml.v3: Tasks 1, 17), Q13 (CLI timestamps + overrides: Task 9), Q14 (write three targets: Task 10), Q15 (five checks: Task 11), Q16-Q19 (naming/license/module/languages: Tasks 1, 16).

**Placeholder scan:** no TBD/TODO steps; every code step carries complete code; every run step carries an exact command and expected result.

**Type consistency:** `storage.Document/ParseDocument/RenderDocument/HasSchemaVersion`, `journal.Entry/ParseEntryHeader/RenderEntry/List/Latest/Append/Parse/ReadEntries/ValidDateFileName/LogDirName/ArchiveDirName`, `validate.Finding/Severity/Run/HasErrors` are used with identical signatures across tasks. `readInput` defined once in Task 9 and reused in Task 10.
