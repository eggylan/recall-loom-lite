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
			return "", fmt.Errorf("write newline: %w", err)
		}
	}
	if _, err := f.WriteString(".rll/\n"); err != nil {
		return "", fmt.Errorf("append .rll/: %w", err)
	}
	return "added .rll/ to .git/info/exclude", nil
}
