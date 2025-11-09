package codegen

import "fmt"

func (g *MainGenerator) GenerateRuntime() string {
	return fmt.Sprintf(`package runtime

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	
	"github.com/withgalaxy/galaxy/pkg/compiler"
	"github.com/withgalaxy/galaxy/pkg/executor"
	"github.com/withgalaxy/galaxy/pkg/template"
	"github.com/withgalaxy/galaxy/pkg/wasm"
)

var comp *compiler.ComponentCompiler
var wasmManifest *wasm.WasmManifest
var baseDir string

func init() {
	// Detect executable path
	exePath, err := os.Executable()
	if err == nil {
		baseDir = filepath.Dir(exePath)
	} else {
		baseDir = "."
	}
	
	comp = compiler.NewComponentCompiler(baseDir)
	loadWasmManifest()
}

func loadWasmManifest() {
	manifestPath := filepath.Join(baseDir, "_assets", "wasm-manifest.json")
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		return
	}
	wasmManifest = &wasm.WasmManifest{}
	json.Unmarshal(data, wasmManifest)
}

type RenderContext struct {
	*executor.Context
	RoutePath string
}

func NewRenderContext() *RenderContext {
	return &RenderContext{
		Context: executor.NewContext(),
	}
}

func RenderTemplate(ctx *RenderContext, templateHTML string) string {
	processed := comp.ProcessComponentTags(templateHTML, ctx.Context)
	
	engine := template.NewEngine(ctx.Context)
	rendered, _ := engine.Render(processed, nil)
	
	rendered = InjectWasmAssets(rendered, ctx.RoutePath)
	
	return rendered
}

func InjectCSS(html, cssPath string) string {
	if cssPath != "" {
		cssTag := "<link rel=\"stylesheet\" href=\"" + cssPath + "\">"
		html = strings.Replace(html, "</head>", "\t" + cssTag + "\n</head>", 1)
	}
	return html
}

func InjectWasmAssets(html, urlPath string) string {
	if wasmManifest == nil {
		// Manifest not loaded, try loading now
		loadWasmManifest()
		if wasmManifest == nil {
			return html
		}
	}
	
	// Try to find matching route in manifest
	var assets *wasm.WasmPageAssets
	for key, val := range wasmManifest.Assets {
		if matchesRoute(key, urlPath) {
			v := val
			assets = &v
			break
		}
	}
	
	if assets == nil {
		// No match found - this is expected for pages without WASM
		return html
	}
	
	var scripts []string
	
	// Add wasm_exec.js once
	if len(assets.WasmModules) > 0 {
		scripts = append(scripts, "<script src=\"/wasm_exec.js\"></script>")
	}
	
	for _, mod := range assets.WasmModules {
		scripts = append(scripts, "<script src=\"" + mod.LoaderPath + "\"></script>")
	}
	
	for _, js := range assets.JSScripts {
		scripts = append(scripts, "<script src=\"" + js + "\"></script>")
	}
	
	// Inject HMR client in dev mode at end of head
	if os.Getenv("DEV_MODE") == "true" {
		hmrScript := "\t<script src=\"/__hmr/client.js\"></script>"
		html = strings.Replace(html, "</head>", hmrScript + "\n</head>", 1)
	}
	
	// Inject WASM scripts at end of body
	if len(scripts) > 0 {
		injection := strings.Join(scripts, "\n\t")
		html = strings.Replace(html, "</body>", "\t" + injection + "\n</body>", 1)
	}
	
	return html
}

func matchesRoute(manifestKey, urlPath string) bool {
	// manifestKey is like "pages/login.gxc"
	// urlPath is like "/login"
	// Strip "pages/" prefix and ".gxc" suffix
	route := strings.TrimPrefix(manifestKey, "pages/")
	route = strings.TrimSuffix(route, ".gxc")
	route = "/" + route
	
	// Handle index
	if route == "/index" {
		route = "/"
	}
	
	return route == urlPath
}
`)
}
