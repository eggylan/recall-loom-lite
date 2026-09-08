package storage

import (
	"strings"
	"testing"
)

func TestTemplatesRenderWithFrontmatter(t *testing.T) {
	for name, body := range map[string]string{
		"brief":   BriefTemplate,
		"summary": SummaryTemplate,
	} {
		out := RenderDocument(body)
		if !strings.HasPrefix(out, "---\nschema_version: 1\n---\n") {
			t.Fatalf("%s: missing frontmatter: %q", name, out[:40])
		}
		if !strings.Contains(out, "(TODO)") {
			t.Fatalf("%s: template should contain TODO placeholders", name)
		}
	}
}
