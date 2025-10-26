package lsp

import (
	"fmt"
	"os"
	"strings"

	"github.com/cameron-webmatter/galaxy/pkg/parser"
	"go.lsp.dev/protocol"
)

func (s *Server) getCompletions(content string, pos protocol.Position) []protocol.CompletionItem {
	items := make([]protocol.CompletionItem, 0)

	// Check if inside component tag for prop completions
	propItems, insideComponent := s.getComponentPropCompletions(content, pos)
	fmt.Fprintf(os.Stderr, "=== getCompletions: insideComponent=%v propCount=%d\n", insideComponent, len(propItems))
	if insideComponent {
		fmt.Fprintf(os.Stderr, "=== RETURNING COMPONENT PROPS\n")
		return propItems
	}
	fmt.Fprintf(os.Stderr, "=== NOT INSIDE COMPONENT, CONTINUING\n")

	comp, err := parser.Parse(content)
	if err != nil {
		return items
	}

	if comp.Frontmatter == "" {
		// Still show import suggestions
		return append(s.getDirectiveCompletions(), s.getImportCompletions()...)
	}

	// Check if we're in import context
	lines := strings.Split(content, "\n")
	if int(pos.Line) < len(lines) {
		currentLine := lines[pos.Line]
		if strings.Contains(currentLine, "import") {
			return s.getImportCompletions()
		}
	}

	// Infer types from frontmatter
	inferencer := NewTypeInferencer()
	if err := inferencer.InferTypes(comp.Frontmatter); err != nil {
		// Fallback to basic completions
		return s.getDirectiveCompletions()
	}

	// Get line at cursor position
	lines = strings.Split(content, "\n")
	if int(pos.Line) >= len(lines) {
		return s.getDirectiveCompletions()
	}

	currentLine := lines[pos.Line]
	if int(pos.Character) > len(currentLine) {
		return s.getDirectiveCompletions()
	}
	beforeCursor := currentLine[:pos.Character]

	// Check if typing after '.' for member access
	if idx := strings.LastIndex(beforeCursor, "."); idx != -1 {
		// Extract variable name before dot
		varStart := idx - 1
		for varStart >= 0 && (isIdentChar(beforeCursor[varStart])) {
			varStart--
		}
		varName := beforeCursor[varStart+1 : idx]

		// Get completions for this variable's members
		return s.getMemberCompletions(inferencer, varName)
	}

	// Default: show all variables + directives
	allTypes := inferencer.GetAllTypes()
	for name, typeInfo := range allTypes {
		if name == "Galaxy" {
			continue // Don't show raw Galaxy, show its members
		}

		items = append(items, protocol.CompletionItem{
			Label:  name,
			Kind:   protocol.CompletionItemKindVariable,
			Detail: typeInfo.TypeName,
		})
	}

	items = append(items, s.getDirectiveCompletions()...)
	// Note: Galaxy.* completions (Redirect, Locals, Params) are NOT added here
	// because they're only available in frontmatter Go code, not in templates

	return items
}

func (s *Server) getMemberCompletions(inferencer *TypeInferencer, varName string) []protocol.CompletionItem {
	items := make([]protocol.CompletionItem, 0)

	typeInfo, ok := inferencer.GetType(varName)
	if !ok {
		return items
	}

	// Struct/Map fields
	if typeInfo.Fields != nil {
		for fieldName, fieldType := range typeInfo.Fields {
			items = append(items, protocol.CompletionItem{
				Label:  fieldName,
				Kind:   protocol.CompletionItemKindField,
				Detail: fieldType.TypeName,
			})
		}
	}

	return items
}

func (s *Server) getDirectiveCompletions() []protocol.CompletionItem {
	return []protocol.CompletionItem{
		{
			Label:  "galaxy:if",
			Kind:   protocol.CompletionItemKindKeyword,
			Detail: "Conditional rendering",
		},
		{
			Label:  "galaxy:elsif",
			Kind:   protocol.CompletionItemKindKeyword,
			Detail: "Else-if conditional branch",
		},
		{
			Label:  "galaxy:else",
			Kind:   protocol.CompletionItemKindKeyword,
			Detail: "Else conditional branch",
		},
		{
			Label:  "galaxy:for",
			Kind:   protocol.CompletionItemKindKeyword,
			Detail: "Loop rendering",
		},
	}
}

func (s *Server) getGalaxyCompletions() []protocol.CompletionItem {
	return []protocol.CompletionItem{
		{
			Label:  "Galaxy.Redirect",
			Kind:   protocol.CompletionItemKindMethod,
			Detail: "func(url string, status int)",
		},
		{
			Label:  "Galaxy.Locals",
			Kind:   protocol.CompletionItemKindField,
			Detail: "map[string]any - middleware data",
		},
		{
			Label:  "Galaxy.Params",
			Kind:   protocol.CompletionItemKindField,
			Detail: "map[string]interface{} - route params",
		},
	}
}

func (s *Server) getImportCompletions() []protocol.CompletionItem {
	items := make([]protocol.CompletionItem, 0)

	if s.project == nil {
		// Return common imports
		return []protocol.CompletionItem{
			{Label: "fmt", Kind: protocol.CompletionItemKindModule, Detail: "Standard library"},
			{Label: "time", Kind: protocol.CompletionItemKindModule, Detail: "Standard library"},
			{Label: "github.com/google/uuid", Kind: protocol.CompletionItemKindModule},
			{Label: "gorm.io/gorm", Kind: protocol.CompletionItemKindModule},
		}
	}

	// Add project imports
	for _, path := range s.project.GetImportPaths() {
		items = append(items, protocol.CompletionItem{
			Label:  path,
			Kind:   protocol.CompletionItemKindModule,
			Detail: "Project import",
		})
	}

	return items
}

func isIdentChar(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '_'
}

func (s *Server) getComponentPropCompletions(content string, pos protocol.Position) ([]protocol.CompletionItem, bool) {
	lines := strings.Split(content, "\n")
	if int(pos.Line) >= len(lines) {
		return nil, false
	}

	line := lines[pos.Line]
	cursor := int(pos.Character)

	if cursor > len(line) {
		cursor = len(line)
	}

	fmt.Fprintf(os.Stderr, "=== PROP COMPLETION CHECK: line=%q cursor=%d\n", line, cursor)

	// Find if we're inside a component tag
	tagStart := -1
	for i := cursor - 1; i >= 0; i-- {
		if line[i] == '<' {
			tagStart = i
			break
		}
		if line[i] == '>' {
			return nil, false
		}
	}

	if tagStart == -1 {
		return nil, false
	}

	// Check if we're before the closing >
	foundClosing := false
	for i := tagStart; i < len(line); i++ {
		if line[i] == '>' {
			if i >= cursor {
				// Cursor is before closing >, we're inside tag
				foundClosing = true
			}
			break
		}
	}

	if !foundClosing {
		return nil, false
	}

	// Extract component name
	tagContent := line[tagStart:]
	parts := strings.Fields(tagContent)
	if len(parts) == 0 {
		return nil, false
	}

	compName := strings.TrimLeft(parts[0], "</")
	fmt.Fprintf(os.Stderr, "=== Component name: %q isHTML=%v\n", compName, isStandardHTMLTag(compName))
	if isStandardHTMLTag(compName) {
		return nil, false
	}

	// Find component file
	componentPath := findComponentFile(s.rootPath, compName)
	fmt.Fprintf(os.Stderr, "=== Component path: %q\n", componentPath)
	if componentPath == "" {
		return nil, false
	}

	// Load component info
	componentInfo, err := s.loadComponentInfo(componentPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "=== Load error: %v\n", err)
		return nil, false
	}
	fmt.Fprintf(os.Stderr, "=== Component has %d props\n", len(componentInfo.Props))

	// Extract already-used props
	usedProps := extractUsedPropsFromTag(line[tagStart:])
	fmt.Fprintf(os.Stderr, "=== Used props: %v\n", usedProps)

	// Return prop completions (excluding already used)
	items := make([]protocol.CompletionItem, 0)
	for _, prop := range componentInfo.Props {
		if usedProps[prop.Name] {
			continue
		}

		detail := prop.Type
		if prop.DefaultValue != "" {
			detail += " = " + prop.DefaultValue
		}

		items = append(items, protocol.CompletionItem{
			Label:  prop.Name,
			Kind:   protocol.CompletionItemKindProperty,
			Detail: detail,
		})
	}

	return items, true
}

func extractUsedPropsFromTag(tagContent string) map[string]bool {
	used := make(map[string]bool)

	// Simple regex to find attribute names
	parts := strings.Fields(tagContent)
	for i := 1; i < len(parts); i++ {
		attr := parts[i]
		if idx := strings.Index(attr, "="); idx != -1 {
			attr = attr[:idx]
		}
		attr = strings.TrimSpace(attr)
		if attr != "" && attr != "/" && attr != ">" {
			used[attr] = true
		}
	}

	return used
}
