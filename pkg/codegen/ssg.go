package codegen

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/withgalaxy/galaxy/pkg/parser"
	"github.com/withgalaxy/galaxy/pkg/router"
)

type SSGCodegenBuilder struct {
	Routes     []*router.Route
	PagesDir   string
	OutDir     string
	ModuleName string
}

func NewSSGCodegenBuilder(routes []*router.Route, pagesDir, outDir, moduleName string) *SSGCodegenBuilder {
	return &SSGCodegenBuilder{
		Routes:     routes,
		PagesDir:   pagesDir,
		OutDir:     outDir,
		ModuleName: moduleName,
	}
}

func (b *SSGCodegenBuilder) Build() error {
	buildDir := filepath.Join(b.OutDir, "_build")
	if err := os.MkdirAll(buildDir, 0755); err != nil {
		return err
	}

	var handlers []*GeneratedHandler
	var nonEndpointRoutes []*router.Route

	for _, route := range b.Routes {
		if route.IsEndpoint {
			continue
		}

		if route.Type == router.RouteMarkdown {
			continue
		}

		content, err := os.ReadFile(route.FilePath)
		if err != nil {
			return fmt.Errorf("read %s: %w", route.FilePath, err)
		}

		comp, err := parser.Parse(string(content))
		if err != nil {
			return fmt.Errorf("parse %s: %w", route.FilePath, err)
		}

		gen := NewHandlerGenerator(comp, route, b.ModuleName, b.PagesDir)
		handler, err := gen.Generate()
		if err != nil {
			return fmt.Errorf("generate handler: %w", err)
		}

		handlers = append(handlers, handler)
		nonEndpointRoutes = append(nonEndpointRoutes, route)
	}

	manifestPath := filepath.Join(buildDir, "_assets", "wasm-manifest.json")
	mainGo := b.generateMain(handlers, nonEndpointRoutes, manifestPath)

	if err := os.WriteFile(filepath.Join(buildDir, "main.go"), []byte(mainGo), 0644); err != nil {
		return err
	}

	runtimeDir := filepath.Join(buildDir, "runtime")
	if err := os.MkdirAll(runtimeDir, 0755); err != nil {
		return err
	}

	mainGen := NewMainGenerator(handlers, nonEndpointRoutes, b.ModuleName, manifestPath)
	runtime := mainGen.GenerateRuntime()
	if err := os.WriteFile(filepath.Join(runtimeDir, "runtime.go"), []byte(runtime), 0644); err != nil {
		return err
	}

	if err := b.generateGoMod(buildDir); err != nil {
		return err
	}

	if err := b.compile(buildDir); err != nil {
		return err
	}

	if err := b.execute(buildDir); err != nil {
		return err
	}

	return nil
}

func (b *SSGCodegenBuilder) generateMain(handlers []*GeneratedHandler, routes []*router.Route, manifestPath string) string {
	var handlerFuncs []string
	var renderCalls []string

	for i, handler := range handlers {
		route := routes[i]
		handlerFuncs = append(handlerFuncs, handler.Code)

		outPath := b.getOutputPath(route.Pattern)
		renderCalls = append(renderCalls,
			fmt.Sprintf("\trenderPage(%q, %q, %s)",
				route.Pattern, outPath, handler.FunctionName))
	}

	imports := b.collectImports(handlers)

	return fmt.Sprintf(`package main

import (
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	
	"github.com/withgalaxy/galaxy/pkg/executor"
	"github.com/withgalaxy/galaxy/pkg/template"
	%s
	"%s/runtime"
)

func main() {
	fmt.Println("Pre-rendering pages...")
	
%s
	
	fmt.Println("✓ Done")
}

func renderPage(pattern, outPath string, handler func(http.ResponseWriter, *http.Request, map[string]string, map[string]interface{})) {
	w := &responseWriter{body: make([]byte, 0)}
	r := &http.Request{
		URL: &url.URL{Path: pattern},
	}
	params := make(map[string]string)
	locals := make(map[string]interface{})
	
	handler(w, r, params, locals)
	
	if err := os.MkdirAll(filepath.Dir(outPath), 0755); err != nil {
		panic(err)
	}
	
	if err := os.WriteFile(outPath, w.body, 0644); err != nil {
		panic(err)
	}
	
	fmt.Printf("  ✓ %%s -> %%s\n", pattern, outPath)
}

type responseWriter struct {
	body []byte
}

func (w *responseWriter) Header() http.Header { return http.Header{} }
func (w *responseWriter) Write(b []byte) (int, error) {
	w.body = append(w.body, b...)
	return len(b), nil
}
func (w *responseWriter) WriteHeader(statusCode int) {}

%s
`, imports, b.ModuleName, strings.Join(renderCalls, "\n"), strings.Join(handlerFuncs, "\n\n"))
}

func (b *SSGCodegenBuilder) collectImports(handlers []*GeneratedHandler) string {
	importMap := make(map[string]bool)
	for _, handler := range handlers {
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

func (b *SSGCodegenBuilder) getOutputPath(pattern string) string {
	if pattern == "/" {
		return filepath.Join(b.OutDir, "index.html")
	}

	pattern = strings.TrimPrefix(pattern, "/")
	pattern = strings.ReplaceAll(pattern, "{", "")
	pattern = strings.ReplaceAll(pattern, "}", "")
	pattern = strings.ReplaceAll(pattern, "[", "")
	pattern = strings.ReplaceAll(pattern, "]", "")

	return filepath.Join(b.OutDir, pattern, "index.html")
}

func (b *SSGCodegenBuilder) generateGoMod(buildDir string) error {
	galaxyPath, err := findGalaxyRoot()
	if err != nil {
		galaxyPath = "../../.."
	}

	goMod := fmt.Sprintf(`module %s

go 1.23

replace github.com/withgalaxy/galaxy => %s

require github.com/withgalaxy/galaxy v0.0.0
`, b.ModuleName, galaxyPath)

	return os.WriteFile(filepath.Join(buildDir, "go.mod"), []byte(goMod), 0644)
}

func (b *SSGCodegenBuilder) compile(buildDir string) error {
	tidyCmd := exec.Command("go", "mod", "tidy")
	tidyCmd.Dir = buildDir
	if output, err := tidyCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("go mod tidy: %w\n%s", err, output)
	}

	cmd := exec.Command("go", "build", "-o", "generator", ".")
	cmd.Dir = buildDir
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("compile: %w\n%s", err, output)
	}
	return nil
}

func (b *SSGCodegenBuilder) execute(buildDir string) error {
	cmd := exec.Command("./generator")
	cmd.Dir = buildDir
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("execute: %w\n%s", err, output)
	}
	fmt.Print(string(output))
	return nil
}
