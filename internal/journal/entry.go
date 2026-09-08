package journal

import (
	"fmt"
	"regexp"
	"strings"
)

// Entry is one milestone record inside a daily log.
type Entry struct {
	Time  string // HH:MM
	Title string
	Body  string
}

var entryHeaderRe = regexp.MustCompile(`^## (\d{2}:\d{2}) (.+)$`)

// IsEntryHeader reports whether a line opens a new entry.
func IsEntryHeader(line string) bool {
	return entryHeaderRe.MatchString(strings.TrimRight(line, "\r"))
}

// ParseEntryHeader extracts time and title from an entry header line.
// Only digit shape is checked; clock sanity is intentionally not validated.
func ParseEntryHeader(line string) (timeStr, title string, err error) {
	m := entryHeaderRe.FindStringSubmatch(strings.TrimRight(line, "\r"))
	if m == nil {
		return "", "", fmt.Errorf("invalid entry header: %q", line)
	}
	return m[1], m[2], nil
}

// RenderEntry formats an entry as markdown.
func RenderEntry(e Entry) string {
	body := strings.TrimSpace(e.Body)
	if body == "" {
		return fmt.Sprintf("## %s %s\n", e.Time, e.Title)
	}
	return fmt.Sprintf("## %s %s\n\n%s\n", e.Time, e.Title, body)
}
