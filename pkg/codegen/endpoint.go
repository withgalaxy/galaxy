package codegen

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/withgalaxy/galaxy/pkg/router"
)

func (b *CodegenBuilder) processEndpoint(route *router.Route, serverDir string) (*EndpointHandler, error) {
	content, err := os.ReadFile(route.FilePath)
	if err != nil {
		return nil, err
	}

	methods := detectHTTPMethods(string(content))
	if len(methods) == 0 {
		return nil, fmt.Errorf("no HTTP methods found in %s", route.FilePath)
	}

	// Get relative path from pages dir to endpoint file (including filename)
	relPath, _ := filepath.Rel(b.PagesDir, route.FilePath)

	// Remove .go extension
	relPath = strings.TrimSuffix(relPath, ".go")

	// Sanitize: replace special chars with underscores
	sanitized := strings.ReplaceAll(relPath, "[", "_")
	sanitized = strings.ReplaceAll(sanitized, "]", "_")
	sanitized = strings.ReplaceAll(sanitized, "/", "_")
	sanitized = strings.ReplaceAll(sanitized, "-", "_")
	sanitized = strings.ReplaceAll(sanitized, ".", "_")

	endpointDir := filepath.Join(serverDir, "endpoints", sanitized)
	if err := os.MkdirAll(endpointDir, 0755); err != nil {
		return nil, err
	}

	srcContent := string(content)
	srcContent = stripBuildTags(srcContent)
	srcContent = regexp.MustCompile(`(?m)^package\s+\w+`).ReplaceAllString(srcContent, "package "+sanitized)

	destPath := filepath.Join(endpointDir, filepath.Base(route.FilePath))
	if err := os.WriteFile(destPath, []byte(srcContent), 0644); err != nil {
		return nil, err
	}

	importPath := filepath.Join(b.ModuleName, "endpoints", sanitized)

	return &EndpointHandler{
		Route:       route,
		Methods:     methods,
		PackageName: sanitized,
		ImportPath:  importPath,
	}, nil
}

func detectHTTPMethods(content string) []string {
	methods := []string{"GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS"}
	var found []string

	for _, method := range methods {
		pattern := fmt.Sprintf(`func %s(`, method)
		if strings.Contains(content, pattern) {
			found = append(found, method)
		}
	}

	return found
}

func stripBuildTags(content string) string {
	lines := strings.Split(content, "\n")
	var filtered []string
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if !strings.HasPrefix(trimmed, "//go:build") && !strings.HasPrefix(trimmed, "// +build") {
			filtered = append(filtered, line)
		}
	}
	return strings.Join(filtered, "\n")
}

func (g *MainGenerator) generateEndpointHandlers() string {
	var handlers strings.Builder

	for _, ep := range g.Endpoints {
		for _, method := range ep.Methods {
			funcName := fmt.Sprintf("handle%s_%s", ep.PackageName, method)
			handlers.WriteString(fmt.Sprintf(`
func %s(w http.ResponseWriter, r *http.Request, params map[string]string, locals map[string]interface{}) {
	ctx := &endpoints.Context{
		Request:  r,
		Response: w,
		Params:   params,
		Locals:   locals,
	}
	
	if err := %s.%s(ctx); err != nil {
		http.Error(w, err.Error(), 500)
	}
}
`, funcName, ep.PackageName, method))
		}
	}

	return handlers.String()
}

func (g *MainGenerator) generateEndpointRoutes() string {
	var routes strings.Builder

	// Group endpoints by pattern to avoid duplicates
	patternMap := make(map[string][]*EndpointHandler)
	for _, ep := range g.Endpoints {
		pattern := ep.Route.Pattern
		patternMap[pattern] = append(patternMap[pattern], ep)
	}

	for pattern, endpointGroup := range patternMap {
		if hasParams(pattern) {
			matcher := generateMatcher(pattern)
			extractor := generateParamExtractor(pattern)

			routes.WriteString(fmt.Sprintf("\thttp.HandleFunc(\"%s\", func(w http.ResponseWriter, r *http.Request) {\n", pattern))
			routes.WriteString(fmt.Sprintf("\t\tif %s {\n", matcher))
			routes.WriteString(fmt.Sprintf("\t\t\tparams := %s\n", extractor))

			for _, ep := range endpointGroup {
				for _, method := range ep.Methods {
					funcName := fmt.Sprintf("handle%s_%s", ep.PackageName, method)
					routes.WriteString(fmt.Sprintf("\t\t\tif r.Method == \"%s\" {\n", method))
					if g.HasMiddleware {
						routes.WriteString(fmt.Sprintf("\t\t\t\tchain.Execute(w, r, func(w http.ResponseWriter, r *http.Request, locals map[string]interface{}) {\n"))
						routes.WriteString(fmt.Sprintf("\t\t\t\t\t%s(w, r, params, locals)\n", funcName))
						routes.WriteString("\t\t\t\t})\n")
					} else {
						routes.WriteString(fmt.Sprintf("\t\t\t\t%s(w, r, params, make(map[string]interface{}))\n", funcName))
					}
					routes.WriteString("\t\t\t\treturn\n")
					routes.WriteString("\t\t\t}\n")
				}
			}

			routes.WriteString("\t\t\thttp.Error(w, \"Method not allowed\", 405)\n")
			routes.WriteString("\t\t\treturn\n")
			routes.WriteString("\t\t}\n")
			routes.WriteString("\t})\n\n")
		} else {
			routes.WriteString(fmt.Sprintf("\thttp.HandleFunc(\"%s\", func(w http.ResponseWriter, r *http.Request) {\n", pattern))

			for _, ep := range endpointGroup {
				for _, method := range ep.Methods {
					funcName := fmt.Sprintf("handle%s_%s", ep.PackageName, method)
					routes.WriteString(fmt.Sprintf("\t\tif r.Method == \"%s\" {\n", method))
					if g.HasMiddleware {
						routes.WriteString(fmt.Sprintf("\t\t\tchain.Execute(w, r, func(w http.ResponseWriter, r *http.Request, locals map[string]interface{}) {\n"))
						routes.WriteString(fmt.Sprintf("\t\t\t\t%s(w, r, nil, locals)\n", funcName))
						routes.WriteString("\t\t\t})\n")
					} else {
						routes.WriteString(fmt.Sprintf("\t\t\t%s(w, r, nil, make(map[string]interface{}))\n", funcName))
					}
					routes.WriteString("\t\t\treturn\n")
					routes.WriteString("\t\t}\n")
				}
			}

			routes.WriteString("\t\thttp.Error(w, \"Method not allowed\", 405)\n")
			routes.WriteString("\t})\n\n")
		}
	}

	return routes.String()
}

func (g *MainGenerator) collectEndpointImports() string {
	if len(g.Endpoints) == 0 {
		return ""
	}

	var imports []string
	imports = append(imports, `"github.com/withgalaxy/galaxy/pkg/endpoints"`)

	for _, ep := range g.Endpoints {
		imports = append(imports, fmt.Sprintf(`%s "%s"`, ep.PackageName, ep.ImportPath))
	}

	return strings.Join(imports, "\n\t")
}
