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
