package storage

import (
	"strings"
	"testing"
)

func TestParseDocumentWithFrontmatter(t *testing.T) {
	doc, err := ParseDocument("---\nschema_version: 1\n---\n\n# Title\n\nbody\n")
	if err != nil {
		t.Fatal(err)
	}
	if !doc.HasSchemaVersion() {
		t.Fatal("want schema_version detected")
	}
	if doc.Body != "# Title\n\nbody\n" {
		t.Fatalf("Body = %q", doc.Body)
	}
}

func TestParseDocumentWithoutFrontmatter(t *testing.T) {
	doc, err := ParseDocument("# plain\n")
	if err != nil {
		t.Fatal(err)
	}
	if doc.HasSchemaVersion() {
		t.Fatal("want no schema_version")
	}
	if doc.Body != "# plain\n" {
		t.Fatalf("Body = %q", doc.Body)
	}
}

func TestParseDocumentBrokenFrontmatter(t *testing.T) {
	if _, err := ParseDocument("---\nno end marker"); err == nil {
		t.Fatal("want error for unterminated frontmatter")
	}
}

func TestRenderDocument(t *testing.T) {
	out := RenderDocument("# hi\n\ncontent\n")
	want := "---\nschema_version: 1\n---\n\n# hi\n\ncontent\n"
	if out != want {
		t.Fatalf("RenderDocument = %q, want %q", out, want)
	}
	if !strings.HasPrefix(out, "---\n") || !strings.Contains(out, "\n---\n\n") {
		t.Fatal("frontmatter shape wrong")
	}
}

func TestRenderDocumentEmptyBody(t *testing.T) {
	out := RenderDocument("")
	if out != "---\nschema_version: 1\n---\n" {
		t.Fatalf("RenderDocument(\"\") = %q", out)
	}
}
