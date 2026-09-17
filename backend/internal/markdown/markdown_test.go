package markdown

import (
	"reflect"
	"testing"
)

func TestParseFrontMatter(t *testing.T) {
	content := `---
title: Git Commands
tags:
  - Git
  - CLI
---

# Ignored Heading

Body text.
`
	p := Parse(content)
	if p.Title != "Git Commands" {
		t.Errorf("Title = %q, want Git Commands", p.Title)
	}
	if !reflect.DeepEqual(p.Tags, []string{"Git", "CLI"}) {
		t.Errorf("Tags = %v", p.Tags)
	}
	if p.Body != "# Ignored Heading\n\nBody text.\n" {
		t.Errorf("Body = %q", p.Body)
	}
}

func TestParseFrontMatterCommaTags(t *testing.T) {
	content := "---\ntags: git, cli, git\n---\n# Title\n"
	p := Parse(content)
	if !reflect.DeepEqual(p.Tags, []string{"git", "cli"}) {
		t.Errorf("Tags = %v, want [git cli]", p.Tags)
	}
}

func TestParseHeadingFallback(t *testing.T) {
	p := Parse("\n\n# My Heading\n\ncontent")
	if p.Title != "My Heading" {
		t.Errorf("Title = %q, want My Heading", p.Title)
	}
}

func TestParseNoHeading(t *testing.T) {
	p := Parse("just some text\nmore text")
	if p.Title != "" {
		t.Errorf("Title = %q, want empty", p.Title)
	}
}

func TestParseMalformedFrontMatter(t *testing.T) {
	content := "---\n: : invalid yaml :\n---\n# Real Title\n"
	p := Parse(content)
	if p.Title != "Real Title" {
		t.Errorf("Title = %q, want Real Title", p.Title)
	}
}

func TestParseCRLF(t *testing.T) {
	content := "---\r\ntitle: Windows File\r\n---\r\n# Body\r\n"
	p := Parse(content)
	if p.Title != "Windows File" {
		t.Errorf("Title = %q, want Windows File", p.Title)
	}
}

func TestMergeTags(t *testing.T) {
	got := MergeTags([]string{"Git", "CLI"}, []string{"cli", "Docs"})
	want := []string{"Git", "CLI", "Docs"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("MergeTags = %v, want %v", got, want)
	}
}

func TestSplitTags(t *testing.T) {
	got := SplitTags("a, b , A, c")
	want := []string{"a", "b", "c"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("SplitTags = %v, want %v", got, want)
	}
}
