// Package markdown parses imported Markdown files: YAML front-matter and the
// first level-1 heading, used to derive a document title and tags.
package markdown

import (
	"strings"

	"gopkg.in/yaml.v3"
)

// Parsed holds the fields extracted from a Markdown file.
type Parsed struct {
	// Title is the front-matter `title`, else the first H1, else empty.
	Title string
	// Tags merges front-matter `tags` (or a comma-separated `tags` string).
	Tags []string
	// Body is the Markdown content with the front-matter block removed.
	Body string
}

// frontMatter is the subset of YAML keys recognized. Tags accepts either a
// YAML sequence (`tags: [a, b]`) or a comma-separated scalar (`tags: a, b`).
type frontMatter struct {
	Title string  `yaml:"title"`
	Tags  tagList `yaml:"tags"`
}

// tagList unmarshals both scalar and sequence tag forms.
type tagList []string

// UnmarshalYAML implements yaml.Unmarshaler.
func (t *tagList) UnmarshalYAML(value *yaml.Node) error {
	switch value.Kind {
	case yaml.SequenceNode:
		var items []string
		if err := value.Decode(&items); err != nil {
			return err
		}
		*t = items
		return nil
	case yaml.ScalarNode:
		if value.Value == "" {
			return nil
		}
		*t = splitComma(value.Value)
		return nil
	default:
		return nil
	}
}

func splitComma(value string) []string {
	parts := strings.Split(value, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}

// Parse extracts front-matter and the first H1 from content. It never fails:
// malformed front-matter is treated as absent so imports stay resilient.
func Parse(content string) Parsed {
	body := content
	var fm frontMatter

	if meta, rest, ok := splitFrontMatter(content); ok {
		body = rest
		_ = yaml.Unmarshal([]byte(meta), &fm)
	}

	p := Parsed{
		Title: strings.TrimSpace(fm.Title),
		Tags:  normalizeTags([]string(fm.Tags)),
		Body:  body,
	}
	if p.Title == "" {
		p.Title = firstHeading(body)
	}
	return p
}

// splitFrontMatter returns the YAML block and remaining body when content
// starts with a `---` delimited front-matter section.
func splitFrontMatter(content string) (string, string, bool) {
	content = strings.TrimPrefix(content, "\ufeff")
	normalized := strings.ReplaceAll(content, "\r\n", "\n")
	if !strings.HasPrefix(normalized, "---\n") {
		return "", content, false
	}

	rest := normalized[len("---\n"):]
	end := strings.Index(rest, "\n---")
	if end < 0 {
		return "", content, false
	}
	meta := rest[:end]
	body := rest[end+len("\n---"):]
	body = strings.TrimLeft(body, "\n")
	return meta, body, true
}

// firstHeading returns the text of the first ATX level-1 heading.
func firstHeading(body string) string {
	for _, line := range strings.Split(body, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		if !strings.HasPrefix(trimmed, "# ") {
			// Only skip leading blank lines; stop at the first non-heading
			// content so a later H1 is not misread as the title.
			return ""
		}
		return strings.TrimSpace(strings.TrimPrefix(trimmed, "# "))
	}
	return ""
}

func normalizeTags(tags []string) []string {
	out := make([]string, 0, len(tags))
	seen := make(map[string]struct{}, len(tags))
	for _, tag := range tags {
		// A front-matter `tags: a, b` line parses as one string; split it too.
		for _, part := range strings.Split(tag, ",") {
			part = strings.TrimSpace(part)
			if part == "" {
				continue
			}
			key := strings.ToLower(part)
			if _, dup := seen[key]; dup {
				continue
			}
			seen[key] = struct{}{}
			out = append(out, part)
		}
	}
	return out
}

// MergeTags combines two tag lists, preserving order and dropping duplicates
// case-insensitively.
func MergeTags(a, b []string) []string {
	seen := make(map[string]struct{}, len(a)+len(b))
	out := make([]string, 0, len(a)+len(b))
	for _, list := range [][]string{a, b} {
		for _, tag := range list {
			tag = strings.TrimSpace(tag)
			if tag == "" {
				continue
			}
			key := strings.ToLower(tag)
			if _, dup := seen[key]; dup {
				continue
			}
			seen[key] = struct{}{}
			out = append(out, tag)
		}
	}
	return out
}

// SplitTags parses a comma-separated tag field.
func SplitTags(raw string) []string {
	return normalizeTags(splitComma(raw))
}
