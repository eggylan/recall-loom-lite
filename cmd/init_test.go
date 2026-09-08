package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInitCreatesRllInCwdNotAncestor(t *testing.T) {
	parent := t.TempDir()
	// Parent has an empty .git placeholder directory (issue repro).
	if err := os.MkdirAll(filepath.Join(parent, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	proj := filepath.Join(parent, "proj")
	if err := os.MkdirAll(proj, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Chdir(proj)

	var outBuf, errBuf bytes.Buffer
	cmd := newInitCmd()
	cmd.SetOut(&outBuf)
	cmd.SetErr(&errBuf)
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}

	rll := filepath.Join(proj, ".rll")
	if fi, err := os.Stat(rll); err != nil || !fi.IsDir() {
		t.Fatalf("expected .rll at %s, err=%v", rll, err)
	}
	for _, name := range []string{"daily_logs", "context_brief.md", "rolling_summary.md"} {
		if _, err := os.Stat(filepath.Join(rll, name)); err != nil {
			t.Fatalf("expected %s in .rll: %v", name, err)
		}
	}
	if _, err := os.Stat(filepath.Join(parent, ".rll")); !os.IsNotExist(err) {
		t.Fatalf("parent .rll should not exist, err=%v", err)
	}

	out := outBuf.String()
	if !strings.Contains(out, "initialized "+rll) {
		t.Fatalf("output missing initialized path:\n%s", out)
	}
	if !strings.Contains(out, "git: skipped") {
		t.Fatalf("output missing git: skipped:\n%s", out)
	}
	if _, err := os.Stat(filepath.Join(parent, ".git", "info", "exclude")); !os.IsNotExist(err) {
		t.Fatalf("parent .git/info/exclude should not be created, err=%v", err)
	}
}

func TestInitGitExcludeAtCwdRepoRoot(t *testing.T) {
	dir := t.TempDir()
	gitDir := filepath.Join(dir, ".git")
	if err := os.MkdirAll(filepath.Join(gitDir, "objects"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(gitDir, "refs"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(gitDir, "HEAD"), []byte("ref: refs/heads/master\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Chdir(dir)

	var outBuf, errBuf bytes.Buffer
	cmd := newInitCmd()
	cmd.SetOut(&outBuf)
	cmd.SetErr(&errBuf)
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}

	rll := filepath.Join(dir, ".rll")
	if _, err := os.Stat(rll); err != nil {
		t.Fatalf("expected .rll at %s: %v", rll, err)
	}
	exclude := filepath.Join(gitDir, "info", "exclude")
	data, err := os.ReadFile(exclude)
	if err != nil {
		t.Fatalf("expected .git/info/exclude to exist: %v", err)
	}
	if !strings.Contains(string(data), ".rll/") {
		t.Fatalf("exclude content = %q, want .rll/", data)
	}
	if !strings.Contains(outBuf.String(), "git: added") {
		t.Fatalf("output missing git: added:\n%s", outBuf.String())
	}
}

func TestInitFailsWhenRllExists(t *testing.T) {
	dir := t.TempDir()
	rll := filepath.Join(dir, ".rll")
	if err := os.MkdirAll(rll, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Chdir(dir)

	cmd := newInitCmd()
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	if err := cmd.Execute(); err == nil {
		t.Fatal("want error when .rll already exists")
	}
}
