package parser

import (
	"strings"
	"testing"
)

func TestParseMarkdownBasic(t *testing.T) {
	md := `# Hello World

This is a **bold** test.`

	doc, err := ParseMarkdownWithYAMLFrontmatter(md)
	if err != nil {
		t.Fatalf("Failed to parse markdown: %v", err)
	}

	if !strings.Contains(doc.HTML, "<h1>Hello World</h1>") {
		t.Errorf("Expected h1 tag in HTML, got: %s", doc.HTML)
	}

	if !strings.Contains(doc.HTML, "<strong>bold</strong>") {
		t.Errorf("Expected bold tag in HTML, got: %s", doc.HTML)
	}
}

func TestParseMarkdownWithFrontmatter(t *testing.T) {
	md := `---
layout: "layouts/BlogPost.gxc"
title: "Test Post"
pubDate: "2024-01-15"
tags:
  - test
  - markdown
---

# Content Here

Test content.`

	doc, err := ParseMarkdownWithYAMLFrontmatter(md)
	if err != nil {
		t.Fatalf("Failed to parse markdown: %v", err)
	}

	if doc.Layout != "layouts/BlogPost.gxc" {
		t.Errorf("Expected layout 'layouts/BlogPost.gxc', got: %s", doc.Layout)
	}

	title := doc.GetFrontmatterString("title")
	if title != "Test Post" {
		t.Errorf("Expected title 'Test Post', got: %s", title)
	}

	pubDate := doc.GetFrontmatterString("pubDate")
	if pubDate != "2024-01-15" {
		t.Errorf("Expected pubDate '2024-01-15', got: %s", pubDate)
	}

	if !strings.Contains(doc.HTML, "<h1>Content Here</h1>") {
		t.Errorf("Expected h1 in HTML, got: %s", doc.HTML)
	}
}

func TestParseMarkdownNoFrontmatter(t *testing.T) {
	md := `# Just Content

No frontmatter here.`

	doc, err := ParseMarkdownWithYAMLFrontmatter(md)
	if err != nil {
		t.Fatalf("Failed to parse markdown: %v", err)
	}

	if doc.Layout != "" {
		t.Errorf("Expected empty layout, got: %s", doc.Layout)
	}

	if len(doc.Frontmatter) != 0 {
		t.Errorf("Expected empty frontmatter, got: %v", doc.Frontmatter)
	}
}

func TestParseMarkdownCodeBlock(t *testing.T) {
	md := "```go\nfunc main() {\n    fmt.Println(\"Hello\")\n}\n```"

	doc, err := ParseMarkdownWithYAMLFrontmatter(md)
	if err != nil {
		t.Fatalf("Failed to parse markdown: %v", err)
	}

	if !strings.Contains(doc.HTML, "<pre") {
		t.Errorf("Expected pre tag for code block, got: %s", doc.HTML)
	}

	if !strings.Contains(doc.HTML, "<code>") {
		t.Errorf("Expected code tag for code block, got: %s", doc.HTML)
	}
}
