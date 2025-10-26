package lsp

import (
	"os"
	"path/filepath"
	"strings"

	"go.lsp.dev/protocol"
)

func getComponentAtPosition(content string, pos protocol.Position) string {
	lines := strings.Split(content, "\n")
	if int(pos.Line) >= len(lines) {
		return ""
	}

	line := lines[pos.Line]
	if int(pos.Character) >= len(line) {
		return ""
	}

	start := int(pos.Character)
	end := int(pos.Character)

	for start > 0 && line[start] != '<' {
		start--
	}
	if start >= len(line) || line[start] != '<' {
		return ""
	}

	start++
	if start < len(line) && line[start] == '/' {
		start++
	}

	end = start
	for end < len(line) && isTagNameChar(line[end]) {
		end++
	}

	if start >= end {
		return ""
	}

	tagName := line[start:end]

	if isStandardHTMLTag(tagName) {
		return ""
	}

	return tagName
}

func findComponentFile(rootPath, componentName string) string {
	if rootPath == "" || componentName == "" {
		return ""
	}

	searchPaths := []string{
		filepath.Join(rootPath, "src", "components", componentName+".gxc"),
		filepath.Join(rootPath, "src", "layouts", componentName+".gxc"),
		filepath.Join(rootPath, "components", componentName+".gxc"),
		filepath.Join(rootPath, "layouts", componentName+".gxc"),
	}

	for _, path := range searchPaths {
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}

	return ""
}

func isTagNameChar(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '-' || c == '_'
}

func isStandardHTMLTag(tag string) bool {
	// Components start with uppercase
	if len(tag) > 0 && tag[0] >= 'A' && tag[0] <= 'Z' {
		return false
	}

	tag = strings.ToLower(tag)
	standardTags := map[string]bool{
		"a": true, "div": true, "span": true, "p": true, "h1": true, "h2": true, "h3": true,
		"h4": true, "h5": true, "h6": true, "ul": true, "ol": true, "li": true, "table": true,
		"tr": true, "td": true, "th": true, "thead": true, "tbody": true, "form": true,
		"input": true, "button": true, "label": true, "select": true, "option": true,
		"textarea": true, "img": true, "video": true, "audio": true, "nav": true, "header": true,
		"footer": true, "section": true, "article": true, "aside": true, "main": true,
		"script": true, "style": true, "link": true, "meta": true, "title": true, "body": true,
		"html": true, "head": true, "br": true, "hr": true, "strong": true, "em": true,
		"code": true, "pre": true, "blockquote": true, "iframe": true, "canvas": true,
	}
	return standardTags[tag]
}
