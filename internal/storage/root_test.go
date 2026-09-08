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
