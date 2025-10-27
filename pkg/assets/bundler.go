package assets

import (
	"fmt"
	"strings"

	"github.com/cameron-webmatter/galaxy/pkg/parser"
	"github.com/cameron-webmatter/galaxy/pkg/plugins"
	"github.com/cameron-webmatter/orbit/bundler"
	"github.com/cameron-webmatter/orbit/wasm"
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

		module, err := b.wasmCompiler.Compile(preparedScript, pagePath, &wasm.CompileOptions{
			UseTinyGo: false,
		})
		if err != nil {
			return nil, fmt.Errorf("compile wasm: %w", err)
		}

		loaderContent := wasm.GenerateLoader("/_assets/wasm/script-"+module.Hash+".wasm", moduleID)
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
	return "package main\n\nfunc main() {\n\t// " + moduleID + "\n}\n"
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
