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
