package lsp

import (
	"fmt"
	"go/ast"
	goparser "go/parser"
	"go/token"
	"strings"

	"github.com/withgalaxy/galaxy/pkg/parser"
	"go.lsp.dev/protocol"
)

type PropInfo struct {
	Name          string
	Type          string
	DefaultValue  string
	Required      bool
	Documentation string
	Position      protocol.Range
}

type ComponentInfo struct {
	FilePath string
	Props    []PropInfo
}

func ParseComponentProps(filePath string, content string) (*ComponentInfo, error) {
	comp, err := parser.Parse(content)
	if err != nil {
		return nil, fmt.Errorf("parse error: %w", err)
	}

	if comp.Frontmatter == "" {
		return &ComponentInfo{
			FilePath: filePath,
			Props:    []PropInfo{},
		}, nil
	}

	props := ExtractPropsFromFrontmatter(comp.Frontmatter)

	return &ComponentInfo{
		FilePath: filePath,
		Props:    props,
	}, nil
}

func ExtractPropsFromFrontmatter(frontmatter string) []PropInfo {
	fset := token.NewFileSet()
	wrapped := "package main\nfunc init() {\n" + frontmatter + "\n}"

	node, err := goparser.ParseFile(fset, "", wrapped, goparser.AllErrors)
	if err != nil {
		return []PropInfo{}
	}

	props := []PropInfo{}

	for _, decl := range node.Decls {
		if funcDecl, ok := decl.(*ast.FuncDecl); ok {
			if funcDecl.Body != nil {
				for _, stmt := range funcDecl.Body.List {
					props = append(props, extractPropsFromStmt(stmt)...)
				}
			}
		}
	}

	return props
}

func extractPropsFromStmt(stmt ast.Stmt) []PropInfo {
	props := []PropInfo{}

	switch s := stmt.(type) {
	case *ast.DeclStmt:
		if genDecl, ok := s.Decl.(*ast.GenDecl); ok {
			if genDecl.Tok == token.VAR {
				for _, spec := range genDecl.Specs {
					if valueSpec, ok := spec.(*ast.ValueSpec); ok {
						props = append(props, extractPropsFromValueSpec(valueSpec)...)
					}
				}
			}
		}
	case *ast.AssignStmt:
		for i, lhs := range s.Lhs {
			if ident, ok := lhs.(*ast.Ident); ok {
				prop := PropInfo{
					Name: ident.Name,
					Type: "interface{}",
				}

				if i < len(s.Rhs) {
					prop.Type = inferTypeFromExpr(s.Rhs[i])
					if basicLit, ok := s.Rhs[i].(*ast.BasicLit); ok {
						prop.DefaultValue = basicLit.Value
					}
				}

				props = append(props, prop)
			}
		}
	}

	return props
}

func extractPropsFromValueSpec(spec *ast.ValueSpec) []PropInfo {
	props := []PropInfo{}

	for i, name := range spec.Names {
		prop := PropInfo{
			Name:     name.Name,
			Type:     "interface{}",
			Required: true,
		}

		if spec.Type != nil {
			prop.Type = exprToTypeName(spec.Type)
		}

		if i < len(spec.Values) {
			if prop.Type == "interface{}" {
				prop.Type = inferTypeFromExpr(spec.Values[i])
			}

			if basicLit, ok := spec.Values[i].(*ast.BasicLit); ok {
				prop.DefaultValue = basicLit.Value
			}

			prop.Required = false
		}

		props = append(props, prop)
	}

	return props
}

func inferTypeFromExpr(expr ast.Expr) string {
	switch e := expr.(type) {
	case *ast.BasicLit:
		switch e.Kind {
		case token.INT:
			return "int"
		case token.FLOAT:
			return "float64"
		case token.STRING:
			return "string"
		case token.CHAR:
			return "rune"
		}
	case *ast.CompositeLit:
		if e.Type != nil {
			return exprToTypeName(e.Type)
		}
	}
	return "interface{}"
}

func exprToTypeName(expr ast.Expr) string {
	switch e := expr.(type) {
	case *ast.Ident:
		return e.Name
	case *ast.SelectorExpr:
		return exprToTypeName(e.X) + "." + e.Sel.Name
	case *ast.StarExpr:
		return "*" + exprToTypeName(e.X)
	case *ast.ArrayType:
		return "[]" + exprToTypeName(e.Elt)
	case *ast.MapType:
		return "map[" + exprToTypeName(e.Key) + "]" + exprToTypeName(e.Value)
	}
	return "interface{}"
}

func getAttributeAtPosition(content string, pos protocol.Position) (componentName, attributeName string, found bool) {
	lines := strings.Split(content, "\n")
	if int(pos.Line) >= len(lines) {
		return "", "", false
	}

	line := lines[pos.Line]
	if int(pos.Character) >= len(line) {
		return "", "", false
	}

	cursor := int(pos.Character)

	tagStart := -1
	for i := cursor; i >= 0; i-- {
		if line[i] == '<' {
			tagStart = i
			break
		}
		if line[i] == '>' {
			return "", "", false
		}
	}

	if tagStart == -1 {
		return "", "", false
	}

	tagEnd := -1
	for i := tagStart; i < len(line); i++ {
		if line[i] == '>' {
			tagEnd = i
			break
		}
	}

	if tagEnd == -1 {
		return "", "", false
	}

	tagContent := line[tagStart:tagEnd]

	parts := strings.Fields(tagContent)
	if len(parts) == 0 {
		return "", "", false
	}

	compName := strings.TrimLeft(parts[0], "</")
	if isStandardHTMLTag(compName) {
		return "", "", false
	}

	attrStart := cursor
	for attrStart > tagStart && line[attrStart] != ' ' && line[attrStart] != '=' && line[attrStart] != '<' {
		attrStart--
	}
	if attrStart > tagStart && (line[attrStart] == ' ' || line[attrStart] == '<') {
		attrStart++
	}

	attrEnd := cursor
	for attrEnd < tagEnd && line[attrEnd] != ' ' && line[attrEnd] != '=' && line[attrEnd] != '>' {
		attrEnd++
	}

	if attrStart >= attrEnd {
		return "", "", false
	}

	attrName := line[attrStart:attrEnd]
	attrName = strings.TrimSpace(attrName)

	if attrName == "" || attrName == compName {
		return "", "", false
	}

	return compName, attrName, true
}
