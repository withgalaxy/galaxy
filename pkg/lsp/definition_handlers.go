package lsp

import (
	"regexp"
	"strings"

	"github.com/withgalaxy/galaxy/pkg/parser"
	"go.lsp.dev/protocol"
)

func (s *Server) goToComponent(componentName string) ([]protocol.Location, error) {
	componentPath := findComponentFile(s.rootPath, componentName)
	if componentPath == "" {
		return nil, nil
	}

	return []protocol.Location{{
		URI: protocol.DocumentURI("file://" + componentPath),
		Range: protocol.Range{
			Start: protocol.Position{Line: 0, Character: 0},
			End:   protocol.Position{Line: 0, Character: 0},
		},
	}}, nil
}

func (s *Server) goToPropDefinition(componentName, propName string) ([]protocol.Location, error) {
	componentPath := findComponentFile(s.rootPath, componentName)
	if componentPath == "" {
		return nil, nil
	}

	lineNum, err := FindPropDefinitionLine(componentPath, propName)
	if err != nil {
		return nil, nil
	}

	return []protocol.Location{{
		URI: protocol.DocumentURI("file://" + componentPath),
		Range: protocol.Range{
			Start: protocol.Position{Line: uint32(lineNum), Character: 0},
			End:   protocol.Position{Line: uint32(lineNum), Character: 100},
		},
	}}, nil
}

func (s *Server) goToVariableDefinition(uri protocol.DocumentURI, content, varName string) ([]protocol.Location, error) {
	comp, err := parser.Parse(content)
	if err != nil {
		return nil, nil
	}

	// Calculate frontmatter position
	lines := strings.Split(content, "\n")
	frontmatterStart := -1
	dashCount := 0

	for i, line := range lines {
		if strings.TrimSpace(line) == "---" {
			dashCount++
			if dashCount == 1 {
				frontmatterStart = i + 1
			} else if dashCount == 2 {
				break
			}
		}
	}

	// Search in frontmatter
	if frontmatterStart >= 0 && comp.Frontmatter != "" {
		if lineNum := findVarInCode(comp.Frontmatter, varName); lineNum >= 0 {
			return []protocol.Location{{
				URI: uri,
				Range: protocol.Range{
					Start: protocol.Position{Line: uint32(frontmatterStart + lineNum), Character: 0},
					End:   protocol.Position{Line: uint32(frontmatterStart + lineNum), Character: 100},
				},
			}}, nil
		}
	}

	// Search in scripts (future enhancement)
	// for _, script := range comp.Scripts {
	// 	if lineNum := findVarInCode(script.Content, varName); lineNum >= 0 {
	// 		return []protocol.Location{{...}}, nil
	// 	}
	// }

	return nil, nil
}

func findVarInCode(code, varName string) int {
	lines := strings.Split(code, "\n")

	// Regex patterns for variable declarations
	patterns := []*regexp.Regexp{
		regexp.MustCompile(`^\s*var\s+` + varName + `\s`),
		regexp.MustCompile(`^\s*var\s+` + varName + `=`),
		regexp.MustCompile(`^\s*` + varName + `\s*:=`),
	}

	for i, line := range lines {
		for _, pattern := range patterns {
			if pattern.MatchString(line) {
				return i
			}
		}
	}

	return -1
}
