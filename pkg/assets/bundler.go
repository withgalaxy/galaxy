package assets

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/withgalaxy/galaxy/pkg/parser"
	"github.com/withgalaxy/galaxy/pkg/plugins"
	"github.com/withgalaxy/orbit/bundler"
	"github.com/withgalaxy/orbit/wasm"
)

type Bundler struct {
	orbitBundler  *bundler.Bundler
	wasmCompiler  *wasm.Compiler
	DevMode       bool
	PluginManager *plugins.Manager
}

type WasmAsset struct {
	WasmPath   string
	LoaderPath string
}

func NewBundler(outDir string) *Bundler {
	return &Bundler{
		orbitBundler: bundler.New(outDir),
		wasmCompiler: wasm.NewCompiler(".galaxy/wasm-build", outDir+"/_assets/wasm"),
	}
}

func (b *Bundler) BundleStyles(comp *parser.Component, pagePath string) (string, error) {
	if len(comp.Styles) == 0 {
		return "", nil
	}

	var combined strings.Builder
	for _, style := range comp.Styles {
		content := style.Content

		if b.PluginManager != nil {
			transformed, err := b.PluginManager.TransformCSS(content, pagePath)
			if err != nil {
				return "", err
			}
			content = transformed
		}

		combined.WriteString(content)
		combined.WriteString("\n")
	}

	var transforms []bundler.TransformFunc
	scopeID := ""
	scoped := false

	for _, style := range comp.Styles {
		if style.Scoped {
			scoped = true
			scopeID = b.GenerateScopeID(pagePath)
			break
		}
	}

	asset, err := b.orbitBundler.BundleCSS(combined.String(), pagePath, &bundler.CSSOptions{
		Scoped:     scoped,
		ScopeID:    scopeID,
		Transforms: transforms,
	})
	if err != nil {
		return "", err
	}

	return asset.Path, nil
}

func (b *Bundler) BundleScripts(comp *parser.Component, pagePath string) (string, error) {
	if len(comp.Scripts) == 0 {
		return "", nil
	}

	var combined strings.Builder
	for i, script := range comp.Scripts {
		if script.Language == "go" {
			continue
		}
		if i > 0 {
			combined.WriteString("\n")
		}

		content := script.Content
		if b.PluginManager != nil {
			transformed, err := b.PluginManager.TransformJS(content, pagePath)
			if err != nil {
				return "", err
			}
			content = transformed
		}

		combined.WriteString(content)
	}

	if combined.Len() == 0 {
		return "", nil
	}

	asset, err := b.orbitBundler.BundleJS(combined.String(), pagePath, nil)
	if err != nil {
		return "", err
	}

	return asset.Path, nil
}

func (b *Bundler) BundleWasmScripts(comp *parser.Component, pagePath string) ([]WasmAsset, error) {
	var assets []WasmAsset

	for _, script := range comp.Scripts {
		if script.Language != "go" {
			continue
		}

		moduleID := pagePath
		preparedScript := b.prepareWasmScript(script.Content, moduleID)

		// Determine galaxy module path for local development
		galaxyPath := os.Getenv("GALAXY_PATH")
		if galaxyPath == "" {
			// Try to find galaxy in parent directories
			cwd, _ := os.Getwd()
			testPaths := []string{
				filepath.Join(cwd, "..", "galaxy"),
				filepath.Join(cwd, "../..", "galaxy"),
				filepath.Join(cwd, "../../..", "galaxy"),
			}
			for _, p := range testPaths {
				absPath, _ := filepath.Abs(p)
				if _, err := os.Stat(absPath); err == nil {
					galaxyPath = absPath
					break
				}
			}
		}

		module, err := b.wasmCompiler.Compile(preparedScript, pagePath, &wasm.CompileOptions{
			UseTinyGo:     false,
			ModuleName:    "github.com/withgalaxy/galaxy",
			ModuleVersion: "v0.0.0",
			ModulePath:    galaxyPath,
		})
		if err != nil {
			return nil, fmt.Errorf("compile wasm: %w", err)
		}

		loaderContent := GenerateWasmLoader("/_assets/wasm/script-"+module.Hash+".wasm", moduleID)
		loaderAsset, err := b.orbitBundler.BundleJS(loaderContent, pagePath+"-loader", nil)
		if err != nil {
			return nil, err
		}

		assets = append(assets, WasmAsset{
			WasmPath:   "/_assets/wasm/script-" + module.Hash + ".wasm",
			LoaderPath: loaderAsset.Path,
		})
	}

	return assets, nil
}

func (b *Bundler) prepareWasmScript(script, moduleID string) string {
	// Split imports, functions, variables, and main code
	lines := strings.Split(script, "\n")
	var imports []string
	var variables []string
	var functions []string
	var mainCode []string

	inFunction := false
	braceCount := 0
	currentFunc := []string{}

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Track imports
		if strings.HasPrefix(trimmed, "import ") {
			imports = append(imports, line)
			continue
		}

		// Detect package-level variable declarations
		if strings.HasPrefix(trimmed, "var ") || strings.HasPrefix(trimmed, "const ") {
			variables = append(variables, line)
			continue
		}

		// Detect function definitions
		if strings.HasPrefix(trimmed, "func ") && !inFunction {
			inFunction = true
			currentFunc = []string{line}
			braceCount = strings.Count(line, "{") - strings.Count(line, "}")
			if braceCount == 0 {
				// Single-line function (unlikely but handle it)
				functions = append(functions, line)
				inFunction = false
				currentFunc = []string{}
			}
			continue
		}

		if inFunction {
			currentFunc = append(currentFunc, line)
			braceCount += strings.Count(line, "{") - strings.Count(line, "}")
			if braceCount == 0 {
				functions = append(functions, strings.Join(currentFunc, "\n"))
				inFunction = false
				currentFunc = []string{}
			}
			continue
		}

		// Everything else goes in main
		if trimmed != "" {
			mainCode = append(mainCode, line)
		}
	}

	// Build complete Go program
	var builder strings.Builder
	builder.WriteString("package main\n\n")

	if len(imports) > 0 {
		builder.WriteString("import (\n")
		for _, imp := range imports {
			imp = strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(imp), "import "))
			builder.WriteString("\t")
			builder.WriteString(imp)
			builder.WriteString("\n")
		}
		builder.WriteString(")\n\n")
	}

	// Add package-level variables
	for _, v := range variables {
		builder.WriteString(v)
		builder.WriteString("\n")
	}
	if len(variables) > 0 {
		builder.WriteString("\n")
	}

	// Add functions at package level
	for _, fn := range functions {
		builder.WriteString(fn)
		builder.WriteString("\n\n")
	}

	// Add main with blocking select
	builder.WriteString(fmt.Sprintf("func main() {\n\t// Module: %s\n", moduleID))
	for _, line := range mainCode {
		builder.WriteString("\t")
		builder.WriteString(line)
		builder.WriteString("\n")
	}
	builder.WriteString("\t// Block forever to keep event listeners active\n")
	builder.WriteString("\tselect {}\n")
	builder.WriteString("}\n")

	return builder.String()
}

func (b *Bundler) GenerateScopeID(pagePath string) string {
	return b.orbitBundler.GenerateScopeID(pagePath)
}

func (b *Bundler) InjectAssets(html, cssPath, jsPath, scopeID string) string {
	return b.InjectAssetsWithWasm(html, cssPath, jsPath, scopeID, nil)
}

func (b *Bundler) InjectAssetsWithWasm(html, cssPath, jsPath, scopeID string, wasmAssets []WasmAsset) string {
	if scopeID != "" {
		bodyScopeAttr := fmt.Sprintf(`data-gxc-%s`, scopeID)
		html = strings.Replace(html, "<body>", fmt.Sprintf(`<body %s>`, bodyScopeAttr), 1)
	}

	if b.DevMode {
		hmrScript := `<script src="/__hmr/client.js"></script>`
		html = strings.Replace(html, "</head>", hmrScript+"\n</head>", 1)
	}

	if cssPath != "" {
		cssTag := fmt.Sprintf(`<link rel="stylesheet" href="%s">`, cssPath)
		html = strings.Replace(html, "</head>", cssTag+"\n</head>", 1)
	}

	if len(wasmAssets) > 0 {
		wasmExecTag := `<script src="/wasm_exec.js"></script>`
		html = strings.Replace(html, "</body>", wasmExecTag+"\n</body>", 1)

		for _, asset := range wasmAssets {
			loaderTag := fmt.Sprintf(`<script src="%s"></script>`, asset.LoaderPath)
			html = strings.Replace(html, "</body>", loaderTag+"\n</body>", 1)
		}
	}

	if jsPath != "" {
		jsTag := fmt.Sprintf(`<script type="module" src="%s"></script>`, jsPath)
		html = strings.Replace(html, "</body>", jsTag+"\n</body>", 1)
	}

	return html
}
