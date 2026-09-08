package journal

import "testing"

func TestParseEntryHeader(t *testing.T) {
	ts, title, err := ParseEntryHeader("## 10:30 fix login timeout")
	if err != nil {
		t.Fatal(err)
	}
	if ts != "10:30" || title != "fix login timeout" {
		t.Fatalf("got %q %q", ts, title)
	}
}

func TestParseEntryHeaderRejectsBadFormats(t *testing.T) {
	for _, line := range []string{
		"## fix login", // no time
		"## 10:30",     // no title
		"# 10:30 fix",  // wrong heading level
	} {
		if _, _, err := ParseEntryHeader(line); err == nil {
			t.Fatalf("want error for %q", line)
		}
	}
}

func TestParseEntryHeaderShapeOnly(t *testing.T) {
	// Clock sanity is intentionally NOT validated (decision Q13/Q15).
	if _, _, err := ParseEntryHeader("## 25:99 out of range"); err != nil {
		t.Fatalf("shape-valid header rejected: %v", err)
	}
}

func TestRenderEntry(t *testing.T) {
	out := RenderEntry(Entry{Time: "10:30", Title: "fix login", Body: "done\n\ndetails"})
	want := "## 10:30 fix login\n\ndone\n\ndetails\n"
	if out != want {
		t.Fatalf("RenderEntry = %q, want %q", out, want)
	}
	if RenderEntry(Entry{Time: "09:00", Title: "empty"}) != "## 09:00 empty\n" {
		t.Fatal("empty body render wrong")
	}
}
