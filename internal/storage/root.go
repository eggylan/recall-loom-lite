package storage

import (
	"errors"
	"fmt"
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

// CheckInitTarget validates that rll is usable as a fresh sidecar root:
// it must be absent. An existing directory or file is an error.
func CheckInitTarget(rll string) error {
	fi, err := os.Stat(rll)
	switch {
	case err == nil && fi.IsDir():
		return fmt.Errorf("%s already exists", rll)
	case err == nil:
		return fmt.Errorf("%s exists and is not a directory", rll)
	case !os.IsNotExist(err):
		return fmt.Errorf("stat %s: %w", rll, err)
	}
	return nil
}
