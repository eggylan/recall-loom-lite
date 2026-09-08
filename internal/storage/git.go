package storage

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// IsGitRoot reports whether dir itself is the root of a valid git
// repository, i.e. dir/.git exists with the minimal repository layout
// created by git init (HEAD file, objects/ and refs/ directories).
// A mere .git placeholder directory (e.g. empty) is not a repository.
func IsGitRoot(dir string) bool {
	gitDir := filepath.Join(dir, ".git")
	fi, err := os.Stat(gitDir)
	if err != nil || !fi.IsDir() {
		// Missing, or .git is a file (linked worktree): unsupported.
		return false
	}
	head, err := os.Stat(filepath.Join(gitDir, "HEAD"))
	if err != nil || !head.Mode().IsRegular() {
		return false
	}
	objects, err := os.Stat(filepath.Join(gitDir, "objects"))
	if err != nil || !objects.IsDir() {
		return false
	}
	refs, err := os.Stat(filepath.Join(gitDir, "refs"))
	if err != nil || !refs.IsDir() {
		return false
	}
	return true
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
