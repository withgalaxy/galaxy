package parser

import (
	"bytes"
	"fmt"

	"github.com/yuin/goldmark"
	highlighting "github.com/yuin/goldmark-highlighting"
	meta "github.com/yuin/goldmark-meta"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"gopkg.in/yaml.v3"
)

type MarkdownDocument struct {
	Frontmatter map[string]interface{}
	Content     string
	HTML        string
	Layout      string
}

func ParseMarkdown(content string) (*MarkdownDocument, error) {
	md := goldmark.New(
		goldmark.WithExtensions(
			meta.Meta,
			extension.GFM,
			highlighting.NewHighlighting(
				highlighting.WithStyle("monokai"),
			),
		),
	)

	var buf bytes.Buffer
	context := parser.NewContext()

	if err := md.Convert([]byte(content), &buf, parser.WithContext(context)); err != nil {
		return nil, fmt.Errorf("convert markdown: %w", err)
	}

	frontmatter := meta.Get(context)
	if frontmatter == nil {
		frontmatter = make(map[string]interface{})
	}

	doc := &MarkdownDocument{
		Frontmatter: frontmatter,
		Content:     content,
		HTML:        buf.String(),
	}

	if layout, ok := frontmatter["layout"].(string); ok {
		doc.Layout = layout
	}

	return doc, nil
}

func ParseMarkdownWithYAMLFrontmatter(content string) (*MarkdownDocument, error) {
	var frontmatter map[string]interface{}
	var body string

	if len(content) > 4 && content[:4] == "---\n" {
		endIndex := -1
		for i := 4; i < len(content); i++ {
			if i+4 <= len(content) && content[i:i+4] == "\n---" {
				endIndex = i
				break
			}
		}

		if endIndex != -1 {
			yamlContent := content[4:endIndex]
			if err := yaml.Unmarshal([]byte(yamlContent), &frontmatter); err != nil {
				return nil, fmt.Errorf("parse frontmatter: %w", err)
			}
			if endIndex+5 < len(content) {
				body = content[endIndex+5:]
			}
		} else {
			body = content
		}
	} else {
		body = content
	}

	if frontmatter == nil {
		frontmatter = make(map[string]interface{})
	}

	md := goldmark.New(
		goldmark.WithExtensions(
			extension.GFM,
			highlighting.NewHighlighting(
				highlighting.WithStyle("monokai"),
			),
		),
	)

	var buf bytes.Buffer
	if err := md.Convert([]byte(body), &buf); err != nil {
		return nil, fmt.Errorf("convert markdown: %w", err)
	}

	doc := &MarkdownDocument{
		Frontmatter: frontmatter,
		Content:     body,
		HTML:        buf.String(),
	}

	if layout, ok := frontmatter["layout"].(string); ok {
		doc.Layout = layout
	}

	return doc, nil
}

func (d *MarkdownDocument) GetFrontmatterString(key string) string {
	if val, ok := d.Frontmatter[key].(string); ok {
		return val
	}
	return ""
}

func (d *MarkdownDocument) GetFrontmatterInt(key string) int {
	if val, ok := d.Frontmatter[key].(int); ok {
		return val
	}
	return 0
}

func (d *MarkdownDocument) GetFrontmatterBool(key string) bool {
	if val, ok := d.Frontmatter[key].(bool); ok {
		return val
	}
	return false
}
