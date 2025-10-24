package assets

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/cameron-webmatter/galaxy/internal/wasm"
	"github.com/cameron-webmatter/galaxy/pkg/parser"
	"github.com/cameron-webmatter/galaxy/pkg/plugins"
)

type Bundler struct {
	OutDir        string
	PluginManager *plugins.Manager
	WasmCompiler  *wasm.Compiler
}

type WasmAsset struct {
	WasmPath   string
	LoaderPath string
}

func NewBundler(outDir string) *Bundler {
	compiler := wasm.NewCompiler(filepath.Join(".galaxy", "wasm-build"), filepath.Join(outDir, "_assets", "wasm"))
	compiler.UseTinyGo = false
	return &Bundler{
		OutDir:       outDir,
		WasmCompiler: compiler,
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

		if style.Scoped {
			scopedCSS := b.scopeCSS(content, pagePath)
			combined.WriteString(scopedCSS)
		} else {
			combined.WriteString(content)
		}
		combined.WriteString("\n")
	}

	hash := fmt.Sprintf("%x", sha256.Sum256([]byte(combined.String())))[:8]
	filename := fmt.Sprintf("styles-%s.css", hash)
	outPath := filepath.Join(b.OutDir, "_assets", filename)

	if err := os.MkdirAll(filepath.Dir(outPath), 0755); err != nil {
		return "", err
	}

	if err := os.WriteFile(outPath, []byte(combined.String()), 0644); err != nil {
		return "", err
	}

	return "/_assets/" + filename, nil
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

	hash := fmt.Sprintf("%x", sha256.Sum256([]byte(combined.String())))[:8]
	filename := fmt.Sprintf("script-%s.js", hash)
	outPath := filepath.Join(b.OutDir, "_assets", filename)

	if err := os.MkdirAll(filepath.Dir(outPath), 0755); err != nil {
		return "", err
	}

	if err := os.WriteFile(outPath, []byte(combined.String()), 0644); err != nil {
		return "", err
	}

	return "/_assets/" + filename, nil
}

func (b *Bundler) BundleWasmScripts(comp *parser.Component, pagePath string) ([]WasmAsset, error) {
	var assets []WasmAsset

	for _, script := range comp.Scripts {
		if script.Language != "go" {
			continue
		}

		module, err := b.WasmCompiler.Compile(script.Content, pagePath)
		if err != nil {
			return nil, fmt.Errorf("compile wasm: %w", err)
		}

		wasmFilename := fmt.Sprintf("script-%s.wasm", module.Hash)
		loaderFilename := fmt.Sprintf("script-%s-loader.js", module.Hash)

		wasmDest := filepath.Join(b.OutDir, "_assets", "wasm", wasmFilename)
		if err := os.MkdirAll(filepath.Dir(wasmDest), 0755); err != nil {
			return nil, err
		}

		if module.WasmPath != wasmDest {
			data, err := os.ReadFile(module.WasmPath)
			if err != nil {
				return nil, err
			}
			if err := os.WriteFile(wasmDest, data, 0644); err != nil {
				return nil, err
			}
		}

		loaderContent := wasm.GenerateLoader("/_assets/wasm/" + wasmFilename)
		loaderPath := filepath.Join(b.OutDir, "_assets", loaderFilename)
		if err := os.WriteFile(loaderPath, []byte(loaderContent), 0644); err != nil {
			return nil, err
		}

		assets = append(assets, WasmAsset{
			WasmPath:   "/_assets/wasm/" + wasmFilename,
			LoaderPath: "/_assets/" + loaderFilename,
		})
	}

	return assets, nil
}

func (b *Bundler) scopeCSS(css, pagePath string) string {
	scope := b.GenerateScopeID(pagePath)

	lines := strings.Split(css, "\n")
	var scoped strings.Builder

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "/*") {
			scoped.WriteString(line)
			scoped.WriteString("\n")
			continue
		}

		if strings.Contains(trimmed, "{") {
			parts := strings.SplitN(trimmed, "{", 2)
			selector := strings.TrimSpace(parts[0])
			rest := parts[1]

			scoped.WriteString(fmt.Sprintf("[data-gxc-%s] %s { %s\n", scope, selector, rest))
		} else {
			scoped.WriteString(line)
			scoped.WriteString("\n")
		}
	}

	return scoped.String()
}

func (b *Bundler) GenerateScopeID(pagePath string) string {
	hash := sha256.Sum256([]byte(pagePath))
	return fmt.Sprintf("%x", hash)[:6]
}

func (b *Bundler) InjectAssets(html, cssPath, jsPath, scopeID string) string {
	return b.InjectAssetsWithWasm(html, cssPath, jsPath, scopeID, nil)
}

func (b *Bundler) InjectAssetsWithWasm(html, cssPath, jsPath, scopeID string, wasmAssets []WasmAsset) string {
	if scopeID != "" {
		bodyScopeAttr := fmt.Sprintf(`data-gxc-%s`, scopeID)
		html = strings.Replace(html, "<body>", fmt.Sprintf(`<body %s>`, bodyScopeAttr), 1)
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
