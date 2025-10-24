package codegen

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/cameron-webmatter/galaxy/pkg/router"
)

func NewMainGenerator(handlers []*GeneratedHandler, routes []*router.Route, moduleName, manifestPath string) *MainGenerator {
	return &MainGenerator{
		Handlers:     handlers,
		Routes:       routes,
		ModuleName:   moduleName,
		ManifestPath: manifestPath,
	}
}

func (g *MainGenerator) Generate() string {
	imports := g.collectImports()
	endpointImports := g.collectEndpointImports()
	routeRegistrations := g.generateRouteRegistrations()
	endpointRoutes := g.generateEndpointRoutes()
	handlerFunctions := g.generateHandlerFunctions()
	endpointHandlers := g.generateEndpointHandlers()
	helpers := g.generateHelpers()

	regexpImport := ""
	for _, route := range g.Routes {
		if hasParams(route.Pattern) {
			regexpImport = `"regexp"`
			break
		}
	}

	middlewareImport := ""
	if g.HasMiddleware {
		middlewareImport = `
	"github.com/cameron-webmatter/galaxy/pkg/middleware"`
	}

	return fmt.Sprintf(`package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	%s
	"strings"
	"github.com/cameron-webmatter/galaxy/pkg/executor"
	"github.com/cameron-webmatter/galaxy/pkg/template"%s
	"%s/runtime"
	%s
	%s
)

func main() {
	log.Println("Starting server...")
	%s
	
	http.Handle("/_assets/", http.StripPrefix("/_assets/", http.FileServer(http.Dir("_assets"))))
	http.Handle("/wasm_exec.js", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "wasm_exec.js")
	}))
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})
	
	%s
	%s
	
	port := os.Getenv("PORT")
	if port == "" {
		port = "4322"
	}
	addr := ":" + port
	log.Printf("🚀 Server running at http://localhost%%s\n", addr)
	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatal(err)
	}
}

%s

%s

%s
`, regexpImport, middlewareImport, g.ModuleName, imports, endpointImports, g.generateMiddlewareSetup(), endpointRoutes, routeRegistrations, helpers, handlerFunctions, endpointHandlers)
}

func (g *MainGenerator) generateHelpers() string {
	helpers := `func tryServeStatic(w http.ResponseWriter, r *http.Request) bool {
	// Check if this looks like a static file (has extension)
	if filepath.Ext(r.URL.Path) != "" {
		publicPath := filepath.Join("public", r.URL.Path)
		if _, err := os.Stat(publicPath); err == nil {
			http.ServeFile(w, r, publicPath)
			return true
		}
	}
	return false
}

func extractParams(path, pattern string) map[string]string {
	params := make(map[string]string)
	
	pathParts := strings.Split(strings.Trim(path, "/"), "/")
	patternParts := strings.Split(strings.Trim(pattern, "/"), "/")
	
	if len(pathParts) != len(patternParts) {
		return params
	}
	
	for i, part := range patternParts {
		if strings.HasPrefix(part, "[") && strings.HasSuffix(part, "]") {
			key := strings.Trim(part, "[]")
			params[key] = pathParts[i]
		} else if strings.HasPrefix(part, "{") && strings.HasSuffix(part, "}") {
			key := strings.Trim(part, "{}")
			params[key] = pathParts[i]
		}
	}
	
	return params
}`

	if g.HasMiddleware {
		helpers += `

type MiddlewareChain struct {
	middlewares []MiddlewareFunc
}

type MiddlewareFunc func(w http.ResponseWriter, r *http.Request, locals map[string]interface{}, next func())

func NewMiddlewareChain() *MiddlewareChain {
	return &MiddlewareChain{
		middlewares: make([]MiddlewareFunc, 0),
	}
}

func (c *MiddlewareChain) Use(mw MiddlewareFunc) {
	c.middlewares = append(c.middlewares, mw)
}

func (c *MiddlewareChain) Execute(w http.ResponseWriter, r *http.Request, handler func(http.ResponseWriter, *http.Request, map[string]interface{})) {
	locals := make(map[string]interface{})
	
	var runMiddleware func(int)
	runMiddleware = func(index int) {
		if index >= len(c.middlewares) {
			handler(w, r, locals)
			return
		}
		
		c.middlewares[index](w, r, locals, func() {
			runMiddleware(index + 1)
		})
	}
	
	runMiddleware(0)
}
`
	}

	return helpers
}

func (g *MainGenerator) collectImports() string {
	importMap := make(map[string]bool)

	for _, handler := range g.Handlers {
		for _, imp := range handler.Imports {
			importMap[imp] = true
		}
	}

	var imports []string
	for imp := range importMap {
		imports = append(imports, "\t"+imp)
	}

	if len(imports) == 0 {
		return ""
	}

	return strings.Join(imports, "\n")
}

func (g *MainGenerator) generateRouteRegistrations() string {
	var staticRoutes []string
	var dynamicRoutes []string
	var indexHandler string

	for i, handler := range g.Handlers {
		route := g.Routes[i]
		pattern := route.Pattern

		if hasParams(pattern) {
			matcher := generateMatcher(pattern)
			extractor := generateParamExtractor(pattern)

			if g.HasMiddleware {
				dynamicRoutes = append(dynamicRoutes,
					fmt.Sprintf("\t\tif %s {\n\t\t\tparams := %s\n\t\t\tchain.Execute(w, r, func(w http.ResponseWriter, r *http.Request, locals map[string]interface{}) {\n\t\t\t\t%s(w, r, params, locals)\n\t\t\t})\n\t\t\treturn\n\t\t}",
						matcher, extractor, handler.FunctionName))
			} else {
				dynamicRoutes = append(dynamicRoutes,
					fmt.Sprintf("\t\tif %s {\n\t\t\tparams := %s\n\t\t\t%s(w, r, params, make(map[string]interface{}))\n\t\t\treturn\n\t\t}",
						matcher, extractor, handler.FunctionName))
			}
		} else if pattern == "/" {
			if g.HasMiddleware {
				indexHandler = fmt.Sprintf("\t\tif r.URL.Path == \"/\" {\n\t\t\tparams := make(map[string]string)\n\t\t\tchain.Execute(w, r, func(w http.ResponseWriter, r *http.Request, locals map[string]interface{}) {\n\t\t\t\t%s(w, r, params, locals)\n\t\t\t})\n\t\t\treturn\n\t\t}",
					handler.FunctionName)
			} else {
				indexHandler = fmt.Sprintf("\t\tif r.URL.Path == \"/\" {\n\t\t\tparams := make(map[string]string)\n\t\t\t%s(w, r, params, make(map[string]interface{}))\n\t\t\treturn\n\t\t}",
					handler.FunctionName)
			}
		} else {
			if g.HasMiddleware {
				staticRoutes = append(staticRoutes,
					fmt.Sprintf("\thttp.HandleFunc(%q, func(w http.ResponseWriter, r *http.Request) {\n\t\tparams := make(map[string]string)\n\t\tchain.Execute(w, r, func(w http.ResponseWriter, r *http.Request, locals map[string]interface{}) {\n\t\t\t%s(w, r, params, locals)\n\t\t})\n\t})",
						pattern, handler.FunctionName))
			} else {
				staticRoutes = append(staticRoutes,
					fmt.Sprintf("\thttp.HandleFunc(%q, func(w http.ResponseWriter, r *http.Request) {\n\t\tparams := make(map[string]string)\n\t\t%s(w, r, params, make(map[string]interface{}))\n\t})",
						pattern, handler.FunctionName))
			}
		}
	}

	var all []string
	all = append(all, staticRoutes...)

	if len(dynamicRoutes) > 0 || indexHandler != "" {
		var checks []string
		if indexHandler != "" {
			checks = append(checks, indexHandler)
		}
		checks = append(checks, dynamicRoutes...)

		all = append(all, fmt.Sprintf("\thttp.HandleFunc(\"/\", func(w http.ResponseWriter, r *http.Request) {\n\t\t// Try serving static file first\n\t\tif tryServeStatic(w, r) {\n\t\t\treturn\n\t\t}\n%s\n\t\thttp.NotFound(w, r)\n\t})",
			strings.Join(checks, "\n")))
	}

	return strings.Join(all, "\n")
}

func hasParams(pattern string) bool {
	return strings.Contains(pattern, "[") || strings.Contains(pattern, "{")
}

func generateMatcher(pattern string) string {
	re := pattern
	re = regexp.MustCompile(`\[([^\]]+)\]`).ReplaceAllString(re, "([^/]+)")
	re = regexp.MustCompile(`\{([^}]+)\}`).ReplaceAllString(re, "([^/]+)")
	re = strings.ReplaceAll(re, "/", `\/`)
	return fmt.Sprintf("regexp.MustCompile(`^%s$`).MatchString(r.URL.Path)", re)
}

func generateParamExtractor(pattern string) string {
	return "extractParams(r.URL.Path, \"" + pattern + "\")"
}

func (g *MainGenerator) generateMiddlewareSetup() string {
	if !g.HasMiddleware {
		return ""
	}

	return `chain := NewMiddlewareChain()
	for _, mw := range Sequence() {
		chain.Use(func(w http.ResponseWriter, r *http.Request, locals map[string]interface{}, next func()) {
			ctx := &middleware.Context{
				Request:  r,
				Response: w,
				Locals:   locals,
			}
			mw(ctx, func() error {
				next()
				return nil
			})
		})
	}
	`
}

func (g *MainGenerator) generateHandlerFunctions() string {
	var functions []string

	for _, handler := range g.Handlers {
		functions = append(functions, handler.Code)
	}

	return strings.Join(functions, "\n\n")
}
