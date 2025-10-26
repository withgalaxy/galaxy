package lsp

import (
	"fmt"
	"strings"

	"github.com/cameron-webmatter/galaxy/pkg/parser"
	"go.lsp.dev/protocol"
)

func (s *Server) getHover(content string, pos protocol.Position) *protocol.Hover {
	// Check for component attribute hover first
	if hover := s.getComponentAttributeHover(content, pos); hover != nil {
		return hover
	}

	comp, err := parser.Parse(content)
	if err != nil {
		return nil
	}

	if comp.Frontmatter == "" {
		return nil
	}

	// Infer types
	inferencer := NewTypeInferencer()
	if err := inferencer.InferTypes(comp.Frontmatter); err != nil {
		return nil
	}

	// Get word at cursor
	lines := strings.Split(content, "\n")
	if int(pos.Line) >= len(lines) {
		return nil
	}

	currentLine := lines[pos.Line]
	word := s.getWordAtPosition(currentLine, int(pos.Character))

	if word == "" {
		return nil
	}

	// Check if hovering over member access (e.g., Galaxy.Locals)
	beforeCursor := currentLine[:pos.Character]
	if idx := strings.LastIndex(beforeCursor, "."); idx != -1 {
		// Get base variable
		varStart := idx - 1
		for varStart >= 0 && isIdentChar(beforeCursor[varStart]) {
			varStart--
		}
		baseVar := beforeCursor[varStart+1 : idx]

		// Get member info
		if typeInfo, ok := inferencer.GetType(baseVar); ok {
			if typeInfo.Fields != nil {
				if fieldInfo, ok := typeInfo.Fields[word]; ok {
					return &protocol.Hover{
						Contents: protocol.MarkupContent{
							Kind:  protocol.Markdown,
							Value: fmt.Sprintf("**%s.%s**: `%s`", baseVar, word, fieldInfo.TypeName),
						},
					}
				}
			}
		}
	}

	// Variable hover
	if typeInfo, ok := inferencer.GetType(word); ok {
		hoverText := fmt.Sprintf("**%s**: `%s`", word, typeInfo.TypeName)

		if typeInfo.IsStruct {
			hoverText += "\n\n*struct*"
		} else if typeInfo.IsMap {
			hoverText += "\n\n*map*"
		} else if typeInfo.IsSlice {
			hoverText += "\n\n*slice*"
		}

		return &protocol.Hover{
			Contents: protocol.MarkupContent{
				Kind:  protocol.Markdown,
				Value: hoverText,
			},
		}
	}

	return nil
}

func (s *Server) getWordAtPosition(line string, col int) string {
	if col > len(line) {
		col = len(line)
	}

	// Find word boundaries
	start := col
	for start > 0 && isIdentChar(line[start-1]) {
		start--
	}

	end := col
	for end < len(line) && isIdentChar(line[end]) {
		end++
	}

	if start >= end {
		return ""
	}

	return line[start:end]
}

func (s *Server) getComponentAttributeHover(content string, pos protocol.Position) *protocol.Hover {
	componentName, attributeName, found := getAttributeAtPosition(content, pos)
	if !found {
		return nil
	}

	componentPath := findComponentFile(s.rootPath, componentName)
	if componentPath == "" {
		return nil
	}

	componentInfo, err := s.loadComponentInfo(componentPath)
	if err != nil {
		return nil
	}

	for _, prop := range componentInfo.Props {
		if prop.Name == attributeName {
			hoverText := fmt.Sprintf("**%s**: `%s`", prop.Name, prop.Type)

			if prop.DefaultValue != "" {
				hoverText += fmt.Sprintf("\n\nDefault: `%s`", prop.DefaultValue)
			}

			if prop.Documentation != "" {
				hoverText += fmt.Sprintf("\n\n%s", prop.Documentation)
			}

			return &protocol.Hover{
				Contents: protocol.MarkupContent{
					Kind:  protocol.Markdown,
					Value: hoverText,
				},
			}
		}
	}

	return nil
}
