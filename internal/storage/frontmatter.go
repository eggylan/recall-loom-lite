package storage

import (
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

// SchemaVersion is the managed-file format version.
const SchemaVersion = 1

// Document is a managed markdown file split into frontmatter and body.
type Document struct {
	Frontmatter map[string]any
	Body        string
}

// ParseDocument splits a managed file into frontmatter and body.
// A file without frontmatter parses successfully with an empty Frontmatter map.
func ParseDocument(content string) (*Document, error) {
	content = strings.ReplaceAll(content, "\r\n", "\n")
	if !strings.HasPrefix(content, "---\n") {
		return &Document{Frontmatter: map[string]any{}, Body: content}, nil
	}
	end := strings.Index(content[4:], "\n---\n")
	if end < 0 {
		return nil, fmt.Errorf("frontmatter start marker found but no end marker")
	}
	raw := content[4 : 4+end]
	body := strings.TrimPrefix(content[4+end+5:], "\n")
	var fm map[string]any
	if err := yaml.Unmarshal([]byte(raw), &fm); err != nil {
		return nil, fmt.Errorf("parse frontmatter: %w", err)
	}
	return &Document{Frontmatter: fm, Body: body}, nil
}

// RenderDocument builds file content with the standard frontmatter prepended.
func RenderDocument(body string) string {
	body = strings.TrimSpace(body)
	if body == "" {
		return fmt.Sprintf("---\nschema_version: %d\n---\n", SchemaVersion)
	}
	return fmt.Sprintf("---\nschema_version: %d\n---\n\n%s\n", SchemaVersion, body)
}

// HasSchemaVersion reports whether the frontmatter carries the expected version.
func (d *Document) HasSchemaVersion() bool {
	v, ok := d.Frontmatter["schema_version"]
	if !ok {
		return false
	}
	switch n := v.(type) {
	case int:
		return n == SchemaVersion
	case float64:
		return n == float64(SchemaVersion)
	case uint64:
		return n == uint64(SchemaVersion)
	}
	return false
}
