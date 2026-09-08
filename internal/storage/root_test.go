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

func TestCheckInitTarget(t *testing.T) {
	base := t.TempDir()
	dir := filepath.Join(base, "d")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	file := filepath.Join(base, "f")
	if err := os.WriteFile(file, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	absent := filepath.Join(base, "absent")

	if err := CheckInitTarget(dir); err == nil || err.Error() != dir+" already exists" {
		t.Fatalf("dir case: err = %v", err)
	}
	if err := CheckInitTarget(file); err == nil {
		t.Fatal("file case: want error")
	}
	if err := CheckInitTarget(absent); err != nil {
		t.Fatalf("absent case: err = %v", err)
	}
}
