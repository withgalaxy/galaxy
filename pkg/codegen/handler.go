package codegen

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/cameron-webmatter/galaxy/pkg/executor"
	"github.com/cameron-webmatter/galaxy/pkg/parser"
	"github.com/cameron-webmatter/galaxy/pkg/router"
)

func NewHandlerGenerator(comp *parser.Component, route *router.Route, moduleName, baseDir string) *HandlerGenerator {
	return &HandlerGenerator{
		Component:  comp,
		Route:      route,
		ModuleName: moduleName,
		BaseDir:    baseDir,
	}
}

func (g *HandlerGenerator) Generate() (*GeneratedHandler, error) {
	imports := g.extractImports()
	code := g.extractCode()
	funcName := g.functionName()

	handler := &GeneratedHandler{
		PackageName:  "handlers",
		Imports:      imports,
		FunctionName: funcName,
	}

	handler.Code = g.generateHandlerFunc(funcName, code, imports)

	return handler, nil
}

func (g *HandlerGenerator) extractImports() []string {
	imports, _ := executor.ExtractImports(g.Component.Frontmatter)

	lines := strings.Split(imports, "\n")
	var result []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		if strings.HasPrefix(line, "import (") || line == ")" {
			continue
		}

		line = strings.TrimPrefix(line, "import ")
		line = strings.TrimSpace(line)

		if line != "" {
			result = append(result, line)
		}
	}
	return result
}

func (g *HandlerGenerator) extractCode() string {
	_, code := executor.ExtractImports(g.Component.Frontmatter)
	code = g.transformCode(code)
	return strings.TrimSpace(code)
}

func (g *HandlerGenerator) transformCode(code string) string {
	params := extractRouteParams(g.Route.Pattern)

	for _, param := range params {
		pattern := fmt.Sprintf(`Galaxy\.Params\["%s"\]`, param)
		code = regexp.MustCompile(pattern).ReplaceAllString(code, param)
	}

	code = regexp.MustCompile(`Galaxy\.[Rr]edirect\(([^,]+),\s*(\d+)\)`).ReplaceAllString(code,
		"http.Redirect(w, r, $1, $2); return")

	code = regexp.MustCompile(`Galaxy\.Locals\.(\w+)`).ReplaceAllString(code, "locals[\"$1\"]")

	code = regexp.MustCompile(`Locals\.(\w+)`).ReplaceAllString(code, "locals[\"$1\"]")

	return code
}

func (g *HandlerGenerator) functionName() string {
	name := strings.ReplaceAll(g.Route.Pattern, "/", "_")
	name = strings.ReplaceAll(name, "{", "")
	name = strings.ReplaceAll(name, "}", "")
	name = strings.ReplaceAll(name, "[", "")
	name = strings.ReplaceAll(name, "]", "")
	name = strings.ReplaceAll(name, ".", "")
	name = strings.ReplaceAll(name, "-", "_")

	if name == "" || name == "_" {
		name = "index"
	}
	name = strings.Trim(name, "_")

	return "Handle" + toPascalCase(name)
}

func (g *HandlerGenerator) generateHandlerFunc(funcName, frontmatterCode string, imports []string) string {
	template := escapeTemplate(g.Component.Template)
	paramExtraction := g.generateParamExtraction()

	return fmt.Sprintf(`func %s(w http.ResponseWriter, r *http.Request, params map[string]string, locals map[string]interface{}) {
	%s
	_ = locals
	
	%s
	%s
	
	// Create executor context for template engine
	ctx := executor.NewContext()
	for k, v := range params {
		ctx.Set(k, v)
	}
	for k, v := range locals {
		ctx.Set(k, v)
	}
	
	%s
	
	// Use Galaxy template engine for full directive support (galaxy:for, galaxy:if, etc.)
	engine := template.NewEngine(ctx)
	html, err := engine.Render(template%s, nil)
	if err != nil {
		http.Error(w, fmt.Sprintf("Template render error: %%v", err), http.StatusInternalServerError)
		return
	}
	
	// Inject CSS if present
	html = runtime.InjectCSS(html, %q)
	
	// Inject WASM assets if present
	html = runtime.InjectWasmAssets(html, r.URL.Path)
	
	w.Write([]byte(html))
}

const template%s = %s
`, funcName, paramExtraction, frontmatterCode, g.generateUseStatements(), g.generateVarAssignments(), funcName, g.CSSPath, funcName, template)
}

func (g *HandlerGenerator) getRoutePath() string {
	rel, _ := filepath.Rel(g.BaseDir, g.Route.FilePath)
	return "pages/" + rel
}

func (g *HandlerGenerator) generateUseStatements() string {
	varNames := g.extractVariableNames()
	if len(varNames) == 0 {
		return ""
	}

	var statements []string
	for _, name := range varNames {
		statements = append(statements, fmt.Sprintf("\t_ = %s", name))
	}

	return strings.Join(statements, "\n")
}

func (g *HandlerGenerator) generateVarAssignments() string {
	varNames := g.extractVariableNames()
	if len(varNames) == 0 {
		return ""
	}

	var assignments []string
	for _, name := range varNames {
		assignments = append(assignments, fmt.Sprintf("\tctx.Set(%q, %s)", name, name))
	}

	return strings.Join(assignments, "\n")
}

func (g *HandlerGenerator) extractVariableNames() []string {
	code := g.extractCode()

	seen := make(map[string]bool)
	var vars []string

	// Process line by line - only extract from lines at column 0 (no indentation)
	// This ensures we only get top-level declarations
	lines := strings.Split(code, "\n")

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Skip empty lines, comments, and type declarations
		if trimmed == "" || strings.HasPrefix(trimmed, "//") || strings.HasPrefix(trimmed, "type ") {
			continue
		}

		// Only process lines that start at column 0 (no leading whitespace)
		// This ensures we're at top level, not inside a block
		if len(line) > 0 && (line[0] == ' ' || line[0] == '\t') {
			continue
		}

		// Match var declarations: var name = ...
		if strings.HasPrefix(trimmed, "var ") {
			varRegex := regexp.MustCompile(`^var\s+(\w+)`)
			if match := varRegex.FindStringSubmatch(trimmed); len(match) > 1 {
				varName := match[1]
				if !seen[varName] && varName != "_" {
					seen[varName] = true
					vars = append(vars, varName)
				}
			}
			continue
		}

		// Match short declarations with := (but not if it's part of a control structure)
		if strings.Contains(trimmed, ":=") && !strings.HasPrefix(trimmed, "if ") && !strings.HasPrefix(trimmed, "switch ") && !strings.HasPrefix(trimmed, "for ") {
			// Extract the left side of :=
			parts := strings.Split(trimmed, ":=")
			if len(parts) >= 2 {
				leftSide := strings.TrimSpace(parts[0])

				// Handle comma-separated variables: a, b := ...
				varNames := strings.Split(leftSide, ",")
				for _, v := range varNames {
					varName := strings.TrimSpace(v)
					if !seen[varName] && varName != "_" {
						seen[varName] = true
						vars = append(vars, varName)
					}
				}
			}
		}
	}

	return vars
}

func toPascalCase(s string) string {
	words := strings.Split(s, "_")
	for i, word := range words {
		if len(word) > 0 {
			words[i] = strings.ToUpper(word[:1]) + word[1:]
		}
	}
	return strings.Join(words, "")
}

func (g *HandlerGenerator) generateParamExtraction() string {
	params := extractRouteParams(g.Route.Pattern)
	if len(params) == 0 {
		return ""
	}

	var lines []string
	for _, param := range params {
		lines = append(lines, fmt.Sprintf("\t%s := params[%q]", param, param))
	}
	return strings.Join(lines, "\n")
}

func extractRouteParams(pattern string) []string {
	var params []string

	curlyRegex := regexp.MustCompile(`\{(\w+)\}`)
	matches := curlyRegex.FindAllStringSubmatch(pattern, -1)
	for _, match := range matches {
		if len(match) > 1 {
			params = append(params, match[1])
		}
	}

	bracketRegex := regexp.MustCompile(`\[(\w+)\]`)
	matches = bracketRegex.FindAllStringSubmatch(pattern, -1)
	for _, match := range matches {
		if len(match) > 1 {
			params = append(params, match[1])
		}
	}

	catchAllRegex := regexp.MustCompile(`\[\.\.\.(\w+)\]`)
	matches = catchAllRegex.FindAllStringSubmatch(pattern, -1)
	for _, match := range matches {
		if len(match) > 1 {
			for i, p := range params {
				if p == "..."+match[1] {
					params[i] = match[1]
					break
				}
			}
		}
	}

	return params
}

func escapeTemplate(template string) string {
	escaped := strings.ReplaceAll(template, "\\", "\\\\")
	escaped = strings.ReplaceAll(escaped, "`", "` + \"`\" + `")
	return "`" + escaped + "`"
}
