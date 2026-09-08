package journal

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestListSortedAndLatest(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "daily_logs")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"2026-09-08.md", "2026-09-01.md", "notes.txt"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	logs, err := List(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(logs) != 2 || logs[0].Date != "2026-09-01" || logs[1].Date != "2026-09-08" {
		t.Fatalf("List = %+v", logs)
	}
	latest, err := Latest(root)
	if err != nil || latest == nil || latest.Date != "2026-09-08" {
		t.Fatalf("Latest = %+v, %v", latest, err)
	}
	none, err := Latest(t.TempDir())
	if err != nil || none != nil {
		t.Fatalf("empty Latest = %+v, %v", none, err)
	}
}

func TestAppendCreatesThenAppends(t *testing.T) {
	root := t.TempDir()
	if _, err := Append(root, "2026-09-08", Entry{Time: "10:30", Title: "first", Body: "b1"}); err != nil {
		t.Fatal(err)
	}
	if _, err := Append(root, "2026-09-08", Entry{Time: "11:00", Title: "second", Body: "b2"}); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(root, "daily_logs", "2026-09-08.md"))
	if err != nil {
		t.Fatal(err)
	}
	content := string(data)
	if !strings.HasPrefix(content, "---\nschema_version: 1\n---\n\n## 10:30 first\n") {
		t.Fatalf("content = %q", content)
	}
	if !strings.Contains(content, "## 11:00 second\n\nb2\n") {
		t.Fatalf("second entry missing: %q", content)
	}
	if strings.Index(content, "## 10:30") > strings.Index(content, "## 11:00") {
		t.Fatal("entries out of order")
	}
}

func TestAppendRejectsBadDate(t *testing.T) {
	if _, err := Append(t.TempDir(), "2026-9-8", Entry{Time: "10:00", Title: "t"}); err == nil {
		t.Fatal("want error for non-canonical date")
	}
}

func TestParseEntries(t *testing.T) {
	content := "---\nschema_version: 1\n---\n\n## 10:30 first\n\nbody one\n\n## 11:00 second\n\nbody two\n"
	entries, err := Parse(content)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 2 {
		t.Fatalf("entries = %+v", entries)
	}
	if entries[0].Title != "first" || entries[0].Body != "body one" {
		t.Fatalf("entry0 = %+v", entries[0])
	}
	if entries[1].Time != "11:00" || entries[1].Body != "body two" {
		t.Fatalf("entry1 = %+v", entries[1])
	}
}
