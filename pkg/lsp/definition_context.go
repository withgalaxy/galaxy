package lsp

import (
	"regexp"
	"strings"

	"go.lsp.dev/protocol"
)

type DefinitionContext int

const (
	ContextComponentName DefinitionContext = iota
	ContextPropName
	ContextVariableName
	ContextNone
)

type DefinitionTarget struct {
	Context       DefinitionContext
	ComponentName string
	PropName      string
	VariableName  string
}

type propInfo struct {
	name       string
	value      string
	nameStart  int
	nameEnd    int
	valueStart int
	valueEnd   int
}

func detectDefinitionContext(content string, pos protocol.Position) *DefinitionTarget {
	lines := strings.Split(content, "\n")
	if int(pos.Line) >= len(lines) {
		return &DefinitionTarget{Context: ContextNone}
	}

	line := lines[pos.Line]
	cursor := int(pos.Character)

	if cursor > len(line) {
		cursor = len(line)
	}

	// Find component tag boundaries
	tagStart := findTagStart(line, cursor)
	if tagStart == -1 {
		return &DefinitionTarget{Context: ContextNone}
	}

	tagEnd := findTagEnd(line, tagStart)
	if tagEnd == -1 {
		tagEnd = len(line)
	}

	tagContent := line[tagStart:tagEnd]

	// Extract component name
	componentName := extractComponentName(tagContent)
	if componentName == "" || isStandardHTMLTag(componentName) {
		return &DefinitionTarget{Context: ContextNone}
	}

	// Determine cursor position relative to tag start
	relPos := cursor - tagStart

	// Check if on component name (before first space/> after <)
	compNameEnd := 1 + len(componentName) // 1 for <
	if relPos <= compNameEnd {
		return &DefinitionTarget{
			Context:       ContextComponentName,
			ComponentName: componentName,
		}
	}

	// Parse props and check if cursor is on prop name or value
	props := parsePropsFromTag(tagContent)
	for _, prop := range props {
		// On prop name
		if relPos >= prop.nameStart && relPos < prop.nameEnd {
			return &DefinitionTarget{
				Context:       ContextPropName,
				ComponentName: componentName,
				PropName:      prop.name,
			}
		}

		// Inside prop value
		if relPos >= prop.valueStart && relPos <= prop.valueEnd {
			// Extract variable name from {varName}
			varName := extractVariableFromPropValue(prop.value, relPos-prop.valueStart)
			if varName != "" {
				return &DefinitionTarget{
					Context:      ContextVariableName,
					VariableName: varName,
				}
			}
		}
	}

	return &DefinitionTarget{Context: ContextNone}
}

func findTagStart(line string, cursor int) int {
	// Search backwards for <
	for i := cursor; i >= 0; i-- {
		if line[i] == '<' {
			return i
		}
		if line[i] == '>' {
			return -1 // Not inside a tag
		}
	}
	return -1
}

func findTagEnd(line string, start int) int {
	// Search forward for >
	for i := start; i < len(line); i++ {
		if line[i] == '>' {
			return i + 1
		}
	}
	return -1
}

func extractComponentName(tagContent string) string {
	// Skip < and /
	start := 1
	if start < len(tagContent) && tagContent[start] == '/' {
		start++
	}

	end := start
	for end < len(tagContent) && isTagNameChar(tagContent[end]) {
		end++
	}

	if start >= end {
		return ""
	}

	return tagContent[start:end]
}

func parsePropsFromTag(tagContent string) []propInfo {
	props := make([]propInfo, 0)

	// Match props: propName={value} or propName="value"
	braceRegex := regexp.MustCompile(`(\w+)=\{([^}]*)\}`)
	stringRegex := regexp.MustCompile(`(\w+)="([^"]*)"`)

	// Find brace props
	matches := braceRegex.FindAllStringSubmatchIndex(tagContent, -1)
	for _, match := range matches {
		propName := tagContent[match[2]:match[3]]
		propValue := tagContent[match[4]:match[5]]

		props = append(props, propInfo{
			name:       propName,
			value:      propValue,
			nameStart:  match[2],
			nameEnd:    match[3],
			valueStart: match[4],
			valueEnd:   match[5],
		})
	}

	// Find string props
	matches = stringRegex.FindAllStringSubmatchIndex(tagContent, -1)
	for _, match := range matches {
		propName := tagContent[match[2]:match[3]]

		// Skip if already found as brace prop
		found := false
		for _, p := range props {
			if p.name == propName {
				found = true
				break
			}
		}
		if found {
			continue
		}

		props = append(props, propInfo{
			name:       propName,
			value:      "",
			nameStart:  match[2],
			nameEnd:    match[3],
			valueStart: -1,
			valueEnd:   -1,
		})
	}

	return props
}

func extractVariableFromPropValue(value string, cursorOffset int) string {
	// Simple case: just a variable name
	value = strings.TrimSpace(value)

	// Extract identifier at cursor position
	if cursorOffset < 0 || cursorOffset > len(value) {
		return extractBaseVariable(value)
	}

	// Find word boundaries around cursor
	start := cursorOffset
	end := cursorOffset

	for start > 0 && isIdentChar(value[start-1]) {
		start--
	}
	for end < len(value) && isIdentChar(value[end]) {
		end++
	}

	if start >= end {
		return extractBaseVariable(value)
	}

	word := value[start:end]
	if word == "" {
		return extractBaseVariable(value)
	}

	return word
}
