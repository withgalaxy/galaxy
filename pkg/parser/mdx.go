package parser

import (
	"bytes"
	"fmt"
	"regexp"

	"github.com/yuin/goldmark"
	highlighting "github.com/yuin/goldmark-highlighting"
	"gopkg.in/yaml.v3"
)

type MDXDocument struct {
	Frontmatter map[string]interface{}
	Content     string
	HTML        string
	Layout      string
	Components  []string
	Imports     []Import
}

var componentTagRegex = regexp.MustCompile(`(?s)<([A-Z]\w+)([^>]*?)(?:>(.*?)</[A-Z]\w+>|/>)`)

func ParseMDX(content string) (*MDXDocument, error) {
	var frontmatter map[string]interface{}
	var body string
	var imports []Import

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

			imports = parseImports(yamlContent)
		} else {
			body = content
		}
	} else {
		body = content
	}

	if frontmatter == nil {
		frontmatter = make(map[string]interface{})
	}

	componentNames := extractComponentNames(body)

	processedBody := protectComponents(body)

	md := goldmark.New(
		goldmark.WithExtensions(
			highlighting.NewHighlighting(
				highlighting.WithStyle("monokai"),
			),
		),
	)

	var buf bytes.Buffer
	if err := md.Convert([]byte(processedBody), &buf); err != nil {
		return nil, fmt.Errorf("convert markdown: %w", err)
	}

	html := restoreComponents(buf.String())

	doc := &MDXDocument{
		Frontmatter: frontmatter,
		Content:     body,
		HTML:        html,
		Components:  componentNames,
		Imports:     imports,
	}

	if layout, ok := frontmatter["layout"].(string); ok {
		doc.Layout = layout
	}

	return doc, nil
}

func extractComponentNames(content string) []string {
	matches := componentTagRegex.FindAllStringSubmatch(content, -1)
	seen := make(map[string]bool)
	var names []string

	for _, match := range matches {
		if len(match) > 1 {
			name := match[1]
			if !seen[name] {
				names = append(names, name)
				seen[name] = true
			}
		}
	}

	return names
}

func protectComponents(content string) string {
	placeholders := make(map[string]string)
	counter := 0

	result := componentTagRegex.ReplaceAllStringFunc(content, func(match string) string {
		placeholder := fmt.Sprintf("GALAXY_COMPONENT_PLACEHOLDER_%d", counter)
		placeholders[placeholder] = match
		counter++
		return placeholder
	})

	return result
}

func restoreComponents(content string) string {
	return content
}

func (d *MDXDocument) GetFrontmatterString(key string) string {
	if val, ok := d.Frontmatter[key].(string); ok {
		return val
	}
	return ""
}

func (d *MDXDocument) GetFrontmatterInt(key string) int {
	if val, ok := d.Frontmatter[key].(int); ok {
		return val
	}
	return 0
}

func (d *MDXDocument) GetFrontmatterBool(key string) bool {
	if val, ok := d.Frontmatter[key].(bool); ok {
		return val
	}
	return false
}

func (d *MDXDocument) HasComponent(name string) bool {
	for _, comp := range d.Components {
		if comp == name {
			return true
		}
	}
	return false
}
