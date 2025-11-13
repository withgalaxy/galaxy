package lsp

import (
	"fmt"
	"go/ast"
	goparser "go/parser"
	"go/token"
	"regexp"
	"strings"

	"github.com/withgalaxy/galaxy/pkg/parser"
	"go.lsp.dev/protocol"
)

var (
	exprRegex         = regexp.MustCompile(`\{([^}]+)\}`)
	ifDirectiveRegex  = regexp.MustCompile(`galaxy:if=\{([^}]+)\}`)
	forDirectiveRegex = regexp.MustCompile(`galaxy:for=\{([^}]+)\}`)
	classListRegex    = regexp.MustCompile(`classList=\{\{([^}]+)\}\}`)
)

type scopeInfo struct {
	variables map[string]string // varName -> type
	inLoop    bool
	loopVar   string
	loopIndex string
}

func (s *Server) analyzeTemplate(content string) []protocol.Diagnostic {
	diagnostics := make([]protocol.Diagnostic, 0)

	comp, err := parser.Parse(content)
	if err != nil {
		return diagnostics
	}

	// Build scope from frontmatter
	scope := s.buildScopeFromFrontmatter(comp.Frontmatter)

	// Add script variables to scope
	for _, script := range comp.Scripts {
		if script.Language == "go" {
			s.addScriptVariablesToScope(script.Content, scope)
		}
	}

	// Validate template expressions
	diagnostics = append(diagnostics, s.validateExpressions(comp.Template, scope, content)...)

	// Validate directives
	diagnostics = append(diagnostics, s.validateDirectives(comp.Template, scope, content)...)

	// Validate component usage
	diagnostics = append(diagnostics, s.validateComponentUsage(content, scope)...)

	return diagnostics
}

func (s *Server) buildScopeFromFrontmatter(frontmatter string) *scopeInfo {
	scope := &scopeInfo{
		variables: make(map[string]string),
	}

	// Always add Galaxy API
	scope.variables["Galaxy"] = "GalaxyAPI"

	if frontmatter == "" {
		return scope
	}

	fset := token.NewFileSet()
	wrapped := "package main\nfunc init() {\n" + frontmatter + "\n}"

	node, err := goparser.ParseFile(fset, "", wrapped, goparser.AllErrors)
	if err != nil {
		return scope
	}

	for _, decl := range node.Decls {
		if funcDecl, ok := decl.(*ast.FuncDecl); ok {
			if funcDecl.Body != nil {
				s.extractVariablesFromStmts(funcDecl.Body.List, scope)
			}
		}
	}

	return scope
}

func (s *Server) addScriptVariablesToScope(scriptContent string, scope *scopeInfo) {
	fset := token.NewFileSet()
	wrapped := "package main\nfunc init() {\n" + scriptContent + "\n}"

	node, err := goparser.ParseFile(fset, "", wrapped, goparser.AllErrors)
	if err != nil {
		return
	}

	for _, decl := range node.Decls {
		if funcDecl, ok := decl.(*ast.FuncDecl); ok {
			if funcDecl.Body != nil {
				s.extractVariablesFromStmts(funcDecl.Body.List, scope)
			}
		}
	}
}

func (s *Server) extractVariablesFromStmts(stmts []ast.Stmt, scope *scopeInfo) {
	for _, stmt := range stmts {
		switch s := stmt.(type) {
		case *ast.DeclStmt:
			if genDecl, ok := s.Decl.(*ast.GenDecl); ok {
				if genDecl.Tok == token.VAR {
					for _, spec := range genDecl.Specs {
						if valueSpec, ok := spec.(*ast.ValueSpec); ok {
							for _, name := range valueSpec.Names {
								typeName := "interface{}"
								if valueSpec.Type != nil {
									typeName = exprToTypeName(valueSpec.Type)
								}
								scope.variables[name.Name] = typeName
							}
						}
					}
				}
			}
		case *ast.AssignStmt:
			for _, lhs := range s.Lhs {
				if ident, ok := lhs.(*ast.Ident); ok {
					scope.variables[ident.Name] = "interface{}"
				}
			}
		}
	}
}

func (s *Server) validateExpressions(template string, scope *scopeInfo, fullContent string) []protocol.Diagnostic {
	diagnostics := make([]protocol.Diagnostic, 0)

	// Extract loop variables from galaxy:for directives for scoping
	loopVars := s.extractLoopVariables(template)

	// Find all expressions {var}
	matches := exprRegex.FindAllStringSubmatchIndex(template, -1)
	for _, match := range matches {
		expr := template[match[2]:match[3]]
		expr = strings.TrimSpace(expr)

		// Skip @html expressions
		if strings.HasPrefix(expr, "@html") {
			continue
		}

		// Skip directive expressions (handled separately)
		if strings.Contains(template[max(0, match[0]-20):match[0]], "galaxy:if=") ||
			strings.Contains(template[max(0, match[0]-20):match[0]], "galaxy:for=") {
			continue
		}

		// Skip classList expressions (handled separately)
		if strings.Contains(template[max(0, match[0]-15):match[0]], "classList=") {
			continue
		}

		// Extract base variable (before dots, brackets, etc)
		baseVar := extractBaseVariable(expr)
		if baseVar == "" {
			continue
		}

		// Skip literals (numbers, strings, bools)
		if isLiteral(baseVar) {
			continue
		}

		// Check if it's a loop variable
		if loopVars[baseVar] {
			continue
		}

		// Check if variable exists in scope
		if _, exists := scope.variables[baseVar]; !exists {
			// Calculate position in full content
			templateStart := findTemplateStart(fullContent)
			exprPos := templateStart + match[0]
			line, col := lineColFromOffset(fullContent, exprPos)

			diagnostics = append(diagnostics, protocol.Diagnostic{
				Range: protocol.Range{
					Start: protocol.Position{Line: uint32(line - 1), Character: uint32(col)},
					End:   protocol.Position{Line: uint32(line - 1), Character: uint32(col + len(baseVar))},
				},
				Severity: protocol.DiagnosticSeverityError,
				Source:   "gxc-template",
				Message:  fmt.Sprintf("Undefined variable: %s", baseVar),
			})
		}
	}

	return diagnostics
}

func (s *Server) extractLoopVariables(template string) map[string]bool {
	loopVars := make(map[string]bool)

	matches := forDirectiveRegex.FindAllStringSubmatch(template, -1)
	for _, match := range matches {
		if len(match) < 2 {
			continue
		}
		forExpr := strings.TrimSpace(match[1])

		// Parse: item, index in items
		parts := strings.Split(forExpr, " in ")
		if len(parts) != 2 {
			continue
		}

		// Extract loop variables
		varsPart := strings.TrimSpace(parts[0])
		vars := strings.Split(varsPart, ",")
		for _, v := range vars {
			loopVars[strings.TrimSpace(v)] = true
		}
	}

	return loopVars
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func isLiteral(value string) bool {
	value = strings.TrimSpace(value)

	// String literals
	if strings.HasPrefix(value, `"`) || strings.HasPrefix(value, "`") || strings.HasPrefix(value, "'") {
		return true
	}

	// Number literals
	if regexp.MustCompile(`^\d+(\.\d+)?$`).MatchString(value) {
		return true
	}

	// Boolean literals
	if value == "true" || value == "false" || value == "nil" {
		return true
	}

	return false
}

func (s *Server) validateDirectives(template string, scope *scopeInfo, fullContent string) []protocol.Diagnostic {
	diagnostics := make([]protocol.Diagnostic, 0)

	// Validate galaxy:if
	diagnostics = append(diagnostics, s.validateIfDirective(template, scope, fullContent)...)

	// Validate galaxy:for
	diagnostics = append(diagnostics, s.validateForDirective(template, scope, fullContent)...)

	// Validate classList
	diagnostics = append(diagnostics, s.validateClassList(template, scope, fullContent)...)

	return diagnostics
}

func (s *Server) validateIfDirective(template string, scope *scopeInfo, fullContent string) []protocol.Diagnostic {
	diagnostics := make([]protocol.Diagnostic, 0)

	matches := ifDirectiveRegex.FindAllStringSubmatchIndex(template, -1)
	for _, match := range matches {
		condition := template[match[2]:match[3]]
		condition = strings.TrimSpace(condition)

		// Extract variables from condition
		vars := extractVariablesFromExpression(condition)
		for _, v := range vars {
			if _, exists := scope.variables[v]; !exists {
				templateStart := findTemplateStart(fullContent)
				exprPos := templateStart + match[0]
				line, col := lineColFromOffset(fullContent, exprPos)

				diagnostics = append(diagnostics, protocol.Diagnostic{
					Range: protocol.Range{
						Start: protocol.Position{Line: uint32(line - 1), Character: uint32(col)},
						End:   protocol.Position{Line: uint32(line - 1), Character: uint32(col + len(v))},
					},
					Severity: protocol.DiagnosticSeverityError,
					Source:   "gxc-directive",
					Message:  fmt.Sprintf("Undefined variable in galaxy:if: %s", v),
				})
			}
		}
	}

	return diagnostics
}

func (s *Server) validateForDirective(template string, scope *scopeInfo, fullContent string) []protocol.Diagnostic {
	diagnostics := make([]protocol.Diagnostic, 0)

	matches := forDirectiveRegex.FindAllStringSubmatchIndex(template, -1)
	for _, match := range matches {
		forExpr := template[match[2]:match[3]]
		forExpr = strings.TrimSpace(forExpr)

		// Parse: item, index in items
		parts := strings.Split(forExpr, " in ")
		if len(parts) != 2 {
			templateStart := findTemplateStart(fullContent)
			exprPos := templateStart + match[0]
			line, col := lineColFromOffset(fullContent, exprPos)

			diagnostics = append(diagnostics, protocol.Diagnostic{
				Range: protocol.Range{
					Start: protocol.Position{Line: uint32(line - 1), Character: uint32(col)},
					End:   protocol.Position{Line: uint32(line - 1), Character: uint32(col + len(forExpr))},
				},
				Severity: protocol.DiagnosticSeverityError,
				Source:   "gxc-directive",
				Message:  "Invalid galaxy:for syntax. Expected: item, index in items",
			})
			continue
		}

		// Extract loop variables (item, index)
		loopVars := strings.Split(strings.TrimSpace(parts[0]), ",")

		// Check if iterable variable exists
		iterableVar := strings.TrimSpace(parts[1])
		baseVar := extractBaseVariable(iterableVar)

		// Don't check loop variables themselves as undefined
		isLoopVar := false
		for _, lv := range loopVars {
			if strings.TrimSpace(lv) == baseVar {
				isLoopVar = true
				break
			}
		}

		if !isLoopVar && baseVar != "" {
			if _, exists := scope.variables[baseVar]; !exists {
				templateStart := findTemplateStart(fullContent)
				exprPos := templateStart + match[0]
				line, col := lineColFromOffset(fullContent, exprPos)

				diagnostics = append(diagnostics, protocol.Diagnostic{
					Range: protocol.Range{
						Start: protocol.Position{Line: uint32(line - 1), Character: uint32(col)},
						End:   protocol.Position{Line: uint32(line - 1), Character: uint32(col + len(baseVar))},
					},
					Severity: protocol.DiagnosticSeverityError,
					Source:   "gxc-directive",
					Message:  fmt.Sprintf("Undefined variable in galaxy:for: %s", baseVar),
				})
			}
		}
	}

	return diagnostics
}

func (s *Server) validateClassList(template string, scope *scopeInfo, fullContent string) []protocol.Diagnostic {
	diagnostics := make([]protocol.Diagnostic, 0)

	matches := classListRegex.FindAllStringSubmatchIndex(template, -1)
	for _, match := range matches {
		classListExpr := template[match[2]:match[3]]

		// Parse classList object: "key": value, "key2": value2
		// Split by comma, then extract value part
		pairs := strings.Split(classListExpr, ",")
		for _, pair := range pairs {
			// Split by colon to get key: value
			parts := strings.Split(pair, ":")
			if len(parts) == 2 {
				value := strings.TrimSpace(parts[1])
				// Remove trailing } if present
				value = strings.TrimSuffix(value, "}")
				value = strings.TrimSpace(value)

				// Check if value is a variable (not a literal)
				if !strings.HasPrefix(value, `"`) && !strings.HasPrefix(value, `'`) &&
					value != "true" && value != "false" && value != "nil" {
					if _, exists := scope.variables[value]; !exists {
						templateStart := findTemplateStart(fullContent)
						exprPos := templateStart + match[0]
						line, col := lineColFromOffset(fullContent, exprPos)

						diagnostics = append(diagnostics, protocol.Diagnostic{
							Range: protocol.Range{
								Start: protocol.Position{Line: uint32(line - 1), Character: uint32(col)},
								End:   protocol.Position{Line: uint32(line - 1), Character: uint32(col + len(value))},
							},
							Severity: protocol.DiagnosticSeverityError,
							Source:   "gxc-directive",
							Message:  fmt.Sprintf("Undefined variable in classList: %s", value),
						})
					}
				}
			}
		}
	}

	return diagnostics
}

func (s *Server) validateComponentUsage(fullContent string, scope *scopeInfo) []protocol.Diagnostic {
	diagnostics := make([]protocol.Diagnostic, 0)

	if s.rootPath == "" {
		return diagnostics
	}

	// Parse components directly from full content to get accurate positions
	compUsages := s.parseComponentUsages(fullContent)

	for _, usage := range compUsages {
		// Find component file
		componentPath := findComponentFile(s.rootPath, usage.name)

		if componentPath == "" {
			line, col := lineColFromOffset(fullContent, usage.namePos)

			diagnostics = append(diagnostics, protocol.Diagnostic{
				Range: protocol.Range{
					Start: protocol.Position{Line: uint32(line - 1), Character: uint32(col)},
					End:   protocol.Position{Line: uint32(line - 1), Character: uint32(col + len(usage.name))},
				},
				Severity: protocol.DiagnosticSeverityWarning,
				Source:   "gxc-component",
				Message:  fmt.Sprintf("Component not found: %s", usage.name),
			})
			continue
		}

		// Load component props
		compInfo, err := s.loadComponentInfo(componentPath)
		if err != nil {
			continue
		}

		// Validate prop types
		for propName, propUsage := range usage.props {
			// Find expected prop
			var expectedProp *PropInfo
			for i := range compInfo.Props {
				if compInfo.Props[i].Name == propName {
					expectedProp = &compInfo.Props[i]
					break
				}
			}

			if expectedProp == nil {
				// Unknown prop warning
				line, col := lineColFromOffset(fullContent, propUsage.namePos)

				diagnostics = append(diagnostics, protocol.Diagnostic{
					Range: protocol.Range{
						Start: protocol.Position{Line: uint32(line - 1), Character: uint32(col)},
						End:   protocol.Position{Line: uint32(line - 1), Character: uint32(col + len(propName))},
					},
					Severity: protocol.DiagnosticSeverityWarning,
					Source:   "gxc-component",
					Message:  fmt.Sprintf("Unknown prop '%s' for component %s", propName, usage.name),
				})
				continue
			}

			// Check type compatibility
			actualType := propUsage.valueType
			if actualType == "" {
				// Try to infer from scope if it's a variable reference
				actualType = s.inferTypeFromScope(propUsage.value, scope)
			}

			if actualType != "" && expectedProp.Type != "" {
				if !isTypeCompatible(actualType, expectedProp.Type) {
					line, col := lineColFromOffset(fullContent, propUsage.valuePos)

					msg := formatTypeMismatchMessage(propName, expectedProp.Type, actualType, propUsage.value)
					diagnostics = append(diagnostics, protocol.Diagnostic{
						Range: protocol.Range{
							Start: protocol.Position{Line: uint32(line - 1), Character: uint32(col)},
							End:   protocol.Position{Line: uint32(line - 1), Character: uint32(col + len(propUsage.value))},
						},
						Severity: protocol.DiagnosticSeverityError,
						Source:   "gxc-component",
						Message:  msg,
					})
				}
			}
		}
	}

	return diagnostics
}

type componentUsage struct {
	name    string
	namePos int
	props   map[string]propUsage
}

type propUsage struct {
	value     string
	valueType string
	namePos   int
	valuePos  int
}

func (s *Server) parseComponentUsages(template string) []componentUsage {
	usages := make([]componentUsage, 0)

	// Find all component tags
	componentTagRegex := regexp.MustCompile(`<([A-Z][a-zA-Z0-9]*)\b([^>]*)>`)
	matches := componentTagRegex.FindAllStringSubmatchIndex(template, -1)

	for _, match := range matches {
		componentName := template[match[2]:match[3]]
		attrsString := ""
		if match[5] > match[4] {
			attrsString = template[match[4]:match[5]]
		}

		usage := componentUsage{
			name:    componentName,
			namePos: match[2],
			props:   make(map[string]propUsage),
		}

		// Parse attributes
		attrRegex := regexp.MustCompile(`(\w+)=\{([^}]+)\}`)
		attrMatches := attrRegex.FindAllStringSubmatchIndex(attrsString, -1)

		for _, attrMatch := range attrMatches {
			propName := attrsString[attrMatch[2]:attrMatch[3]]
			propValue := attrsString[attrMatch[4]:attrMatch[5]]

			usage.props[propName] = propUsage{
				value:     propValue,
				valueType: inferTypeFromValue(propValue),
				namePos:   match[4] + attrMatch[2],
				valuePos:  match[4] + attrMatch[4],
			}
		}

		usages = append(usages, usage)
	}

	return usages
}

func inferTypeFromValue(value string) string {
	value = strings.TrimSpace(value)

	// String literals
	if strings.HasPrefix(value, `"`) || strings.HasPrefix(value, "`") || strings.HasPrefix(value, "'") {
		return "string"
	}

	// Integer literals
	if regexp.MustCompile(`^-?\d+$`).MatchString(value) {
		return "int"
	}

	// Float literals
	if regexp.MustCompile(`^-?\d+\.\d+$`).MatchString(value) {
		return "float64"
	}

	// Boolean literals
	if value == "true" || value == "false" {
		return "bool"
	}

	// Nil
	if value == "nil" {
		return "nil"
	}

	// Otherwise unknown (could be variable reference)
	return ""
}

func (s *Server) inferTypeFromScope(value string, scope *scopeInfo) string {
	value = strings.TrimSpace(value)

	// Extract base variable name
	baseVar := extractBaseVariable(value)
	if baseVar == "" {
		return ""
	}

	// Look up in scope
	if varType, exists := scope.variables[baseVar]; exists {
		return varType
	}

	return ""
}

func isTypeCompatible(actualType, expectedType string) bool {
	if actualType == "" || expectedType == "" {
		return true // Can't determine, skip check
	}

	// Normalize types
	actualType = normalizeType(actualType)
	expectedType = normalizeType(expectedType)

	if actualType == expectedType {
		return true
	}

	// interface{} and any accept anything
	if expectedType == "interface{}" || expectedType == "any" {
		return true
	}

	// nil compatible with pointers, interfaces, maps, slices, channels
	if actualType == "nil" {
		return isNilCompatible(expectedType)
	}

	// Numeric compatibility with warnings
	if isNumericType(actualType) && isNumericType(expectedType) {
		return true
	}

	// String types
	if (actualType == "string" || actualType == "rune") &&
		(expectedType == "string" || expectedType == "rune") {
		return true
	}

	return false
}

func normalizeType(t string) string {
	// Remove pointer indicators for comparison
	t = strings.TrimPrefix(t, "*")

	// Normalize interface{} to any
	if t == "interface{}" {
		return "any"
	}

	return t
}

func isNumericType(t string) bool {
	numericTypes := map[string]bool{
		"int": true, "int8": true, "int16": true, "int32": true, "int64": true,
		"uint": true, "uint8": true, "uint16": true, "uint32": true, "uint64": true,
		"float32": true, "float64": true,
		"byte": true, "rune": true,
		"complex64": true, "complex128": true,
	}
	return numericTypes[t]
}

func isNilCompatible(t string) bool {
	// nil can be assigned to pointers, interfaces, maps, slices, channels, funcs
	if strings.HasPrefix(t, "*") {
		return true
	}
	if strings.HasPrefix(t, "map[") || strings.HasPrefix(t, "[]") ||
		strings.HasPrefix(t, "chan ") || strings.HasPrefix(t, "func(") {
		return true
	}
	if t == "interface{}" || t == "any" {
		return true
	}
	return false
}

func formatTypeMismatchMessage(propName, expectedType, actualType, _ string) string {
	msg := fmt.Sprintf("Type mismatch for prop '%s': expected %s, got %s", propName, expectedType, actualType)

	// Add helpful suggestions
	if isNumericType(actualType) && expectedType == "string" {
		msg += ". Hint: convert with fmt.Sprintf"
	} else if actualType == "string" && isNumericType(expectedType) {
		msg += ". Hint: parse with strconv"
	} else if actualType == "int" && strings.HasPrefix(expectedType, "int") && expectedType != "int" {
		msg += fmt.Sprintf(". Hint: convert with %s(...)", expectedType)
	}

	return msg
}

// Helper functions

func extractBaseVariable(expr string) string {
	// Handle field access: foo.bar -> foo
	if idx := strings.Index(expr, "."); idx > 0 {
		return strings.TrimSpace(expr[:idx])
	}

	// Handle array/map access: foo[0] -> foo
	if idx := strings.Index(expr, "["); idx > 0 {
		return strings.TrimSpace(expr[:idx])
	}

	// Handle function calls: foo() -> foo
	if idx := strings.Index(expr, "("); idx > 0 {
		return strings.TrimSpace(expr[:idx])
	}

	return strings.TrimSpace(expr)
}

func extractVariablesFromExpression(expr string) []string {
	vars := make([]string, 0)
	varRegex := regexp.MustCompile(`\b([a-zA-Z_][a-zA-Z0-9_]*)\b`)
	matches := varRegex.FindAllString(expr, -1)

	keywords := map[string]bool{
		"true": true, "false": true, "nil": true,
		"if": true, "else": true, "for": true,
		"in": true, "range": true,
	}

	for _, match := range matches {
		if !keywords[match] {
			vars = append(vars, match)
		}
	}

	return vars
}

func findTemplateStart(content string) int {
	// Find where template starts (after frontmatter, scripts, styles)
	frontmatterEnd := 0
	if idx := strings.Index(content, "---\n"); idx >= 0 {
		if endIdx := strings.Index(content[idx+4:], "---\n"); endIdx >= 0 {
			frontmatterEnd = idx + 4 + endIdx + 4
		}
	}

	// Skip scripts and styles
	templateContent := content[frontmatterEnd:]
	scriptStart := strings.Index(templateContent, "<script")
	styleStart := strings.Index(templateContent, "<style")

	lastTagEnd := frontmatterEnd
	if scriptStart >= 0 {
		if scriptEnd := strings.Index(templateContent[scriptStart:], "</script>"); scriptEnd >= 0 {
			lastTagEnd = frontmatterEnd + scriptStart + scriptEnd + 9
		}
	}
	if styleStart >= 0 {
		if styleEnd := strings.Index(templateContent[styleStart:], "</style>"); styleEnd >= 0 {
			possibleEnd := frontmatterEnd + styleStart + styleEnd + 8
			if possibleEnd > lastTagEnd {
				lastTagEnd = possibleEnd
			}
		}
	}

	return lastTagEnd
}

func lineColFromOffset(content string, offset int) (int, int) {
	line := 1
	col := 0
	for i := 0; i < offset && i < len(content); i++ {
		if content[i] == '\n' {
			line++
			col = 0
		} else {
			col++
		}
	}
	return line, col
}
