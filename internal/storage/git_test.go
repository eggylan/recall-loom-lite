package storage

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestIsGitRoot(t *testing.T) {
	t.Run("no .git", func(t *testing.T) {
		if IsGitRoot(t.TempDir()) {
			t.Fatal("want false without .git")
		}
	})

	t.Run("empty .git placeholder dir", func(t *testing.T) {
		dir := t.TempDir()
		if err := os.MkdirAll(filepath.Join(dir, ".git"), 0o755); err != nil {
			t.Fatal(err)
		}
		if IsGitRoot(dir) {
			t.Fatal("want false for an empty .git dir")
		}
	})

	t.Run(".git as a file", func(t *testing.T) {
		dir := t.TempDir()
		if err := os.WriteFile(filepath.Join(dir, ".git"), []byte("gitdir: /elsewhere\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		if IsGitRoot(dir) {
			t.Fatal("want false when .git is a file")
		}
	})

	t.Run("minimal valid repo", func(t *testing.T) {
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
		if !IsGitRoot(dir) {
			t.Fatal("want true for a minimal valid repo")
		}
	})
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
