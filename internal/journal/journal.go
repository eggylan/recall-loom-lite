package journal

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/eggylan/recall-loom-lite/internal/storage"
)

// LogDirName is the directory holding active daily logs.
const LogDirName = "daily_logs"

// ArchiveDirName is the directory holding archived daily logs.
const ArchiveDirName = "archive"

var dateFileRe = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}\.md$`)

// LogFile is one dated daily log.
type LogFile struct {
	Date string // YYYY-MM-DD
	Path string
}

// ValidDateFileName reports whether name is a canonical YYYY-MM-DD.md file name.
func ValidDateFileName(name string) bool {
	return dateFileRe.MatchString(name)
}

// List returns all active daily logs sorted by date.
// A missing daily_logs directory yields an empty slice, not an error.
func List(rllRoot string) ([]LogFile, error) {
	return listDir(filepath.Join(rllRoot, LogDirName))
}

// ListArchive returns all archived daily logs sorted by date.
func ListArchive(rllRoot string) ([]LogFile, error) {
	return listDir(filepath.Join(rllRoot, ArchiveDirName))
}

func listDir(dir string) ([]LogFile, error) {
	entries, err := os.ReadDir(dir)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var logs []LogFile
	for _, e := range entries {
		if e.IsDir() || !dateFileRe.MatchString(e.Name()) {
			continue
		}
		logs = append(logs, LogFile{
			Date: strings.TrimSuffix(e.Name(), ".md"),
			Path: filepath.Join(dir, e.Name()),
		})
	}
	sort.Slice(logs, func(i, j int) bool { return logs[i].Date < logs[j].Date })
	return logs, nil
}

// Latest returns the newest active daily log, or nil if none exists.
func Latest(rllRoot string) (*LogFile, error) {
	logs, err := List(rllRoot)
	if err != nil {
		return nil, err
	}
	if len(logs) == 0 {
		return nil, nil
	}
	return &logs[len(logs)-1], nil
}

// Append writes an entry to the log for the given date, creating the file
// (with frontmatter) when needed. Returns the written file path.
func Append(rllRoot, date string, e Entry) (string, error) {
	if !dateFileRe.MatchString(date + ".md") {
		return "", fmt.Errorf("invalid date %q, want YYYY-MM-DD", date)
	}
	dir := filepath.Join(rllRoot, LogDirName)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	path := filepath.Join(dir, date+".md")
	data, err := os.ReadFile(path)
	var content string
	switch {
	case os.IsNotExist(err):
		content = storage.RenderDocument("") + "\n" + RenderEntry(e)
	case err != nil:
		return "", err
	default:
		content = strings.TrimRight(strings.ReplaceAll(string(data), "\r\n", "\n"), "\n") + "\n\n" + RenderEntry(e)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		return "", err
	}
	return path, nil
}

// Parse extracts entries from a daily log's content.
func Parse(content string) ([]Entry, error) {
	doc, err := storage.ParseDocument(content)
	if err != nil {
		return nil, err
	}
	lines := strings.Split(doc.Body, "\n")
	var entries []Entry
	for _, line := range lines {
		if IsEntryHeader(line) {
			ts, title, err := ParseEntryHeader(line)
			if err != nil {
				return nil, err
			}
			entries = append(entries, Entry{Time: ts, Title: title})
			continue
		}
		if len(entries) > 0 {
			i := len(entries) - 1
			if entries[i].Body == "" {
				entries[i].Body = line
			} else {
				entries[i].Body += "\n" + line
			}
		}
	}
	for i := range entries {
		entries[i].Body = strings.TrimSpace(entries[i].Body)
	}
	return entries, nil
}

// ReadEntries loads and parses a log file from disk.
func ReadEntries(log LogFile) ([]Entry, error) {
	data, err := os.ReadFile(log.Path)
	if err != nil {
		return nil, err
	}
	return Parse(string(data))
}
