package codegen

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/cameron-webmatter/galaxy/internal/assets"
	"github.com/cameron-webmatter/galaxy/pkg/compiler"
	"github.com/cameron-webmatter/galaxy/pkg/executor"
	"github.com/cameron-webmatter/galaxy/pkg/parser"
	"github.com/cameron-webmatter/galaxy/pkg/router"
	"github.com/cameron-webmatter/galaxy/pkg/wasm"
)

type CodegenBuilder struct {
	Routes         []*router.Route
	PagesDir       string
	OutDir         string
	ModuleName     string
	MiddlewarePath string
	PublicDir      string
	Bundler        *assets.Bundler
	ManifestPath   string
}

func NewCodegenBuilder(routes []*router.Route, pagesDir, outDir, moduleName, publicDir string) *CodegenBuilder {
	srcDir := filepath.Dir(pagesDir)
	middlewarePath := filepath.Join(srcDir, "middleware.go")

	return &CodegenBuilder{
		Routes:         routes,
		PagesDir:       pagesDir,
		OutDir:         outDir,
		ModuleName:     moduleName,
		MiddlewarePath: middlewarePath,
		PublicDir:      publicDir,
		Bundler:        assets.NewBundler(".galaxy"),
		ManifestPath:   filepath.Join(outDir, "server", "_assets", "wasm-manifest.json"),
	}
}

func (b *CodegenBuilder) Build() error {
	serverDir := filepath.Join(b.OutDir, "server")

	if err := os.MkdirAll(serverDir, 0755); err != nil {
		return err
	}

	var handlers []*GeneratedHandler
	var endpoints []*EndpointHandler
	var nonEndpointRoutes []*router.Route

	for _, route := range b.Routes {
		if route.IsEndpoint {
			ep, err := b.processEndpoint(route, serverDir)
			if err != nil {
				return fmt.Errorf("process endpoint %s: %w", route.Pattern, err)
			}
			endpoints = append(endpoints, ep)
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

		// Process component tags (<Layout>, <Nav>, etc.) before codegen
		processedComp, err := b.processComponentTags(comp, route)
		if err != nil {
			return fmt.Errorf("process components for %s: %w", route.Pattern, err)
		}

		gen := NewHandlerGenerator(processedComp, route, b.ModuleName, b.PagesDir)
		handler, err := gen.Generate()
		if err != nil {
			return fmt.Errorf("generate handler for %s: %w", route.Pattern, err)
		}

		handlers = append(handlers, handler)
		nonEndpointRoutes = append(nonEndpointRoutes, route)
	}

	manifestPath := filepath.Join(serverDir, "_assets", "wasm-manifest.json")
	hasMiddleware := false
	if _, err := os.Stat(b.MiddlewarePath); err == nil {
		hasMiddleware = true
	}

	mainGen := NewMainGenerator(handlers, nonEndpointRoutes, b.ModuleName, manifestPath)
	mainGen.HasMiddleware = hasMiddleware
	mainGen.Endpoints = endpoints
	mainGo := mainGen.Generate()

	if err := os.WriteFile(filepath.Join(serverDir, "main.go"), []byte(mainGo), 0644); err != nil {
		return err
	}

	runtimeDir := filepath.Join(serverDir, "runtime")
	if err := os.MkdirAll(runtimeDir, 0755); err != nil {
		return err
	}

	runtime := mainGen.GenerateRuntime()
	if err := os.WriteFile(filepath.Join(runtimeDir, "runtime.go"), []byte(runtime), 0644); err != nil {
		return err
	}

	if err := b.generateGoMod(serverDir); err != nil {
		return err
	}

	if err := b.copyMiddleware(serverDir); err != nil {
		return fmt.Errorf("copy middleware: %w", err)
	}

	if err := b.copyPublicAssets(serverDir); err != nil {
		return fmt.Errorf("copy public assets: %w", err)
	}

	if err := b.compileWasmScripts(nonEndpointRoutes); err != nil {
		return fmt.Errorf("compile wasm: %w", err)
	}

	if err := b.copyWasmAssets(serverDir); err != nil {
		return fmt.Errorf("copy wasm assets: %w", err)
	}

	if err := b.copyWasmExec(serverDir); err != nil {
		return fmt.Errorf("copy wasm exec: %w", err)
	}

	if err := b.compile(serverDir); err != nil {
		return err
	}

	return nil
}

func (b *CodegenBuilder) copyMiddleware(serverDir string) error {
	if _, err := os.Stat(b.MiddlewarePath); os.IsNotExist(err) {
		return nil
	}

	data, err := os.ReadFile(b.MiddlewarePath)
	if err != nil {
		return err
	}

	content := string(data)
	content = regexp.MustCompile(`(?m)^package\s+\w+`).ReplaceAllString(content, "package main")

	destPath := filepath.Join(serverDir, "middleware.go")
	return os.WriteFile(destPath, []byte(content), 0644)
}

func (b *CodegenBuilder) generateGoMod(serverDir string) error {
	// First, try to find Galaxy path from the project's go.mod
	galaxyPath := ""
	projectModule := ""
	cwd, _ := os.Getwd()
	projectGoMod := filepath.Join(cwd, "go.mod")

	if data, err := os.ReadFile(projectGoMod); err == nil {
		// Look for module name and replace directives
		lines := strings.Split(string(data), "\n")
		for _, line := range lines {
			// Get project module name
			if strings.HasPrefix(line, "module ") {
				projectModule = strings.TrimSpace(strings.TrimPrefix(line, "module"))
			}
			// Get Galaxy replace path
			if strings.Contains(line, "replace github.com/cameron-webmatter/galaxy") {
				parts := strings.Split(line, "=>")
				if len(parts) == 2 {
					galaxyPath = strings.TrimSpace(parts[1])
				}
			}
		}
	}

	// Fallback: try to find galaxy root
	if galaxyPath == "" {
		var err error
		galaxyPath, err = findGalaxyRoot()
		if err != nil {
			// Last resort: assume sibling directory
			galaxyPath = filepath.Join(filepath.Dir(cwd), "galaxy")
		}
	}

	// ALWAYS convert to absolute path
	if !filepath.IsAbs(galaxyPath) {
		galaxyPath, _ = filepath.Abs(galaxyPath)
	}

	// Verify galaxy path exists
	if _, err := os.Stat(filepath.Join(galaxyPath, "go.mod")); os.IsNotExist(err) {
		// Try finding galaxy relative to current binary
		execPath, _ := os.Executable()
		binDir := filepath.Dir(execPath)
		// Try ../galaxy from binary location
		testPath := filepath.Join(binDir, "..", "galaxy")
		if absTest, err := filepath.Abs(testPath); err == nil {
			if _, err := os.Stat(filepath.Join(absTest, "go.mod")); err == nil {
				galaxyPath = absTest
			}
		}
	}

	// Convert cwd to absolute for project replace
	absCwd, err := filepath.Abs(cwd)
	if err != nil {
		absCwd = cwd
	}

	// Build go.mod with replace directives
	goMod := fmt.Sprintf(`module %s

go 1.23

replace github.com/cameron-webmatter/galaxy => %s
`, b.ModuleName, galaxyPath)

	// Add replace for project's own module (so local imports work)
	// This is necessary when the project imports its own packages
	if projectModule != "" && projectModule != b.ModuleName {
		goMod += fmt.Sprintf(`replace %s => %s

`, projectModule, absCwd)
	}

	goMod += `require github.com/cameron-webmatter/galaxy v0.0.0
`

	return os.WriteFile(filepath.Join(serverDir, "go.mod"), []byte(goMod), 0644)
}

func findGalaxyRoot() (string, error) {
	wd, err := os.Getwd()
	if err != nil {
		return "", err
	}

	for dir := wd; dir != "/"; dir = filepath.Dir(dir) {
		modPath := filepath.Join(dir, "go.mod")
		data, err := os.ReadFile(modPath)
		if err != nil {
			continue
		}

		if strings.Contains(string(data), "module github.com/cameron-webmatter/galaxy") {
			return dir, nil
		}
	}

	return "", fmt.Errorf("galaxy root not found")
}

func (b *CodegenBuilder) copyPublicAssets(serverDir string) error {
	// Copy public directory to server/public so files are accessible
	if b.PublicDir == "" {
		return nil
	}

	if _, err := os.Stat(b.PublicDir); os.IsNotExist(err) {
		return nil
	}

	publicOutDir := filepath.Join(serverDir, "public")
	if err := os.MkdirAll(publicOutDir, 0755); err != nil {
		return err
	}

	return filepath.Walk(b.PublicDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		relPath, err := filepath.Rel(b.PublicDir, path)
		if err != nil {
			return err
		}

		destPath := filepath.Join(publicOutDir, relPath)
		if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
			return err
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		return os.WriteFile(destPath, data, info.Mode())
	})
}

func (b *CodegenBuilder) compileWasmScripts(routes []*router.Route) error {
	manifest := wasm.NewManifest()

	for _, route := range routes {

		content, err := os.ReadFile(route.FilePath)
		if err != nil {
			continue
		}

		comp, err := parser.Parse(string(content))
		if err != nil {
			continue
		}

		wasmAssets, err := b.Bundler.BundleWasmScripts(comp, route.FilePath)
		if err != nil {
			return err
		}

		jsPath, err := b.Bundler.BundleScripts(comp, route.FilePath)
		if err != nil {
			return err
		}

		if len(wasmAssets) == 0 && jsPath == "" {
			continue
		}

		pageAssets := wasm.WasmPageAssets{}
		for _, asset := range wasmAssets {
			hash := extractHash(asset.LoaderPath)
			pageAssets.WasmModules = append(pageAssets.WasmModules, wasm.WasmModule{
				Hash:       hash,
				WasmPath:   asset.WasmPath,
				LoaderPath: asset.LoaderPath,
			})
		}
		if jsPath != "" {
			pageAssets.JSScripts = append(pageAssets.JSScripts, jsPath)
		}

		relPath, err := filepath.Rel(b.PagesDir, route.FilePath)
		if err != nil {
			relPath = route.FilePath
		}
		manifestKey := "pages/" + relPath
		manifest.Assets[manifestKey] = pageAssets
	}

	if err := os.MkdirAll(filepath.Dir(b.ManifestPath), 0755); err != nil {
		return err
	}

	return manifest.Save(b.ManifestPath)
}

func (b *CodegenBuilder) copyWasmAssets(serverDir string) error {
	// Bundler writes to .galaxy/_assets
	assetsDir := filepath.Join(".galaxy", "_assets")
	if _, err := os.Stat(assetsDir); os.IsNotExist(err) {
		return nil
	}

	serverAssetsDir := filepath.Join(serverDir, "_assets")
	if err := os.MkdirAll(serverAssetsDir, 0755); err != nil {
		return err
	}

	return filepath.Walk(assetsDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}

		relPath, _ := filepath.Rel(assetsDir, path)
		destPath := filepath.Join(serverAssetsDir, relPath)

		if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
			return err
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		return os.WriteFile(destPath, data, info.Mode())
	})
}

func extractHash(loaderPath string) string {
	parts := strings.Split(filepath.Base(loaderPath), "-")
	if len(parts) >= 2 {
		return strings.TrimSuffix(parts[1], "-loader.js")
	}
	return ""
}

func (b *CodegenBuilder) processComponentTags(comp *parser.Component, route *router.Route) (*parser.Component, error) {
	// Create a compiler instance for component resolution
	srcDir := filepath.Dir(b.PagesDir)
	compilerInstance := compiler.NewComponentCompiler(srcDir)

	// Set up resolver for this file
	resolver := compilerInstance.Resolver
	resolver.SetCurrentFile(route.FilePath)

	// Parse imports
	imports := make([]compiler.Import, len(comp.Imports))
	for i, imp := range comp.Imports {
		imports[i] = compiler.Import{
			Path:        imp.Path,
			Alias:       imp.Alias,
			IsComponent: imp.IsComponent,
		}
	}
	resolver.ParseImports(imports)

	// Create minimal executor context for component processing
	ctx := executor.NewContext()

	// Process component tags
	compilerInstance.CollectedStyles = nil
	processedTemplate := compilerInstance.ProcessComponentTags(comp.Template, ctx)

	// Return new component with processed template and collected styles
	return &parser.Component{
		Frontmatter: comp.Frontmatter,
		Template:    processedTemplate,
		Scripts:     comp.Scripts,
		Styles:      append(comp.Styles, compilerInstance.CollectedStyles...),
		Imports:     comp.Imports,
	}, nil
}

func (b *CodegenBuilder) copyWasmExec(serverDir string) error {
	goRoot := os.Getenv("GOROOT")
	if goRoot == "" {
		cmd := exec.Command("go", "env", "GOROOT")
		output, err := cmd.Output()
		if err != nil {
			return err
		}
		goRoot = strings.TrimSpace(string(output))
	}

	wasmExecSrc := filepath.Join(goRoot, "misc", "wasm", "wasm_exec.js")
	if _, err := os.Stat(wasmExecSrc); os.IsNotExist(err) {
		wasmExecSrc = filepath.Join(goRoot, "lib", "wasm", "wasm_exec.js")
		if _, err := os.Stat(wasmExecSrc); os.IsNotExist(err) {
			return fmt.Errorf("wasm_exec.js not found in GOROOT")
		}
	}

	data, err := os.ReadFile(wasmExecSrc)
	if err != nil {
		return err
	}

	wasmExecDest := filepath.Join(serverDir, "wasm_exec.js")
	return os.WriteFile(wasmExecDest, data, 0644)
}

func (b *CodegenBuilder) compile(serverDir string) error {
	tidyCmd := exec.Command("go", "mod", "tidy")
	tidyCmd.Dir = serverDir
	if output, err := tidyCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("go mod tidy: %w\n%s", err, output)
	}

	cmd := exec.Command("go", "build", "-o", "server", ".")
	cmd.Dir = serverDir
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("compile server: %w\n%s", err, output)
	}
	return nil
}
