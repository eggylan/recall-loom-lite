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
		if f.Severity == severity && (strings.Contains(f.Path, msgPart) || strings.Contains(f.Message, msgPart)) {
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

func TestRunNoFalsePositives(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "context_brief.md"), "---\nschema_version: 1\n---\n\n# ok\n")
	writeFile(t, filepath.Join(root, "rolling_summary.md"), "---\nschema_version: 1\n---\n\n# ok\n")
	// shape-valid but clock-invalid header must NOT be flagged (format-only design)
	writeFile(t, filepath.Join(root, "daily_logs", "2026-09-08.md"),
		"---\nschema_version: 1\n---\n\n## 25:99 night shift\n\nbody\n")
	// update_protocol.md absent and archive/ absent — neither may produce findings
	findings := Run(root)
	if len(findings) != 0 {
		t.Fatalf("false positives: %+v", findings)
	}
}
