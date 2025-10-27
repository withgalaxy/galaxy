package assets

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/withgalaxy/galaxy/pkg/parser"
)

func TestNewBundler(t *testing.T) {
	bundler := NewBundler("/out")

	if bundler.OutDir != "/out" {
		t.Errorf("Expected OutDir /out, got %s", bundler.OutDir)
	}
}

func TestBundleStylesEmpty(t *testing.T) {
	bundler := NewBundler(t.TempDir())
	comp := &parser.Component{
		Styles: []parser.Style{},
	}

	path, err := bundler.BundleStyles(comp, "/pages/index.gxc")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if path != "" {
		t.Errorf("Expected empty path for no styles, got %s", path)
	}
}

func TestBundleStyles(t *testing.T) {
	tmpDir := t.TempDir()
	bundler := NewBundler(tmpDir)

	comp := &parser.Component{
		Styles: []parser.Style{
			{Content: ".header { color: blue; }", Scoped: false},
			{Content: ".footer { color: red; }", Scoped: false},
		},
	}

	path, err := bundler.BundleStyles(comp, "/pages/index.gxc")
	if err != nil {
		t.Fatalf("BundleStyles failed: %v", err)
	}

	if !strings.HasPrefix(path, "/_assets/styles-") {
		t.Errorf("Expected path to start with /_assets/styles-, got %s", path)
	}

	if !strings.HasSuffix(path, ".css") {
		t.Errorf("Expected path to end with .css, got %s", path)
	}

	filePath := filepath.Join(tmpDir, strings.TrimPrefix(path, "/"))
	content, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("Failed to read bundled file: %v", err)
	}

	if !strings.Contains(string(content), ".header") {
		t.Error("Expected bundled CSS to contain .header")
	}

	if !strings.Contains(string(content), ".footer") {
		t.Error("Expected bundled CSS to contain .footer")
	}
}

func TestBundleStylesScoped(t *testing.T) {
	tmpDir := t.TempDir()
	bundler := NewBundler(tmpDir)

	comp := &parser.Component{
		Styles: []parser.Style{
			{Content: ".container { padding: 10px; }", Scoped: true},
		},
	}

	path, err := bundler.BundleStyles(comp, "/pages/test.gxc")
	if err != nil {
		t.Fatalf("BundleStyles failed: %v", err)
	}

	filePath := filepath.Join(tmpDir, strings.TrimPrefix(path, "/"))
	content, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("Failed to read bundled file: %v", err)
	}

	scopeID := bundler.GenerateScopeID("/pages/test.gxc")
	expectedAttr := "[data-gxc-" + scopeID + "]"

	if !strings.Contains(string(content), expectedAttr) {
		t.Errorf("Expected scoped CSS to contain %s, got %s", expectedAttr, string(content))
	}
}

func TestBundleScriptsEmpty(t *testing.T) {
	bundler := NewBundler(t.TempDir())
	comp := &parser.Component{
		Scripts: []parser.Script{},
	}

	path, err := bundler.BundleScripts(comp, "/pages/index.gxc")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if path != "" {
		t.Errorf("Expected empty path for no scripts, got %s", path)
	}
}

func TestBundleScripts(t *testing.T) {
	tmpDir := t.TempDir()
	bundler := NewBundler(tmpDir)

	comp := &parser.Component{
		Scripts: []parser.Script{
			{Content: "console.log('hello');", IsModule: false},
			{Content: "console.log('world');", IsModule: true},
		},
	}

	path, err := bundler.BundleScripts(comp, "/pages/index.gxc")
	if err != nil {
		t.Fatalf("BundleScripts failed: %v", err)
	}

	if !strings.HasPrefix(path, "/_assets/script-") {
		t.Errorf("Expected path to start with /_assets/script-, got %s", path)
	}

	if !strings.HasSuffix(path, ".js") {
		t.Errorf("Expected path to end with .js, got %s", path)
	}

	filePath := filepath.Join(tmpDir, strings.TrimPrefix(path, "/"))
	content, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("Failed to read bundled file: %v", err)
	}

	if !strings.Contains(string(content), "hello") {
		t.Error("Expected bundled JS to contain first script")
	}

	if !strings.Contains(string(content), "world") {
		t.Error("Expected bundled JS to contain second script")
	}
}

func TestScopeCSS(t *testing.T) {
	bundler := NewBundler("/out")

	css := `.header { color: red; }
.footer { color: blue; }`

	scoped := bundler.scopeCSS(css, "/pages/test.gxc")
	scopeID := bundler.GenerateScopeID("/pages/test.gxc")
	expectedAttr := "[data-gxc-" + scopeID + "]"

	if !strings.Contains(scoped, expectedAttr+" .header") {
		t.Errorf("Expected scoped selector with %s .header", expectedAttr)
	}

	if !strings.Contains(scoped, expectedAttr+" .footer") {
		t.Errorf("Expected scoped selector with %s .footer", expectedAttr)
	}
}

func TestScopeCSSWithComments(t *testing.T) {
	bundler := NewBundler("/out")

	css := `/* Comment */
.test { color: red; }`

	scoped := bundler.scopeCSS(css, "/page.gxc")

	if !strings.Contains(scoped, "/* Comment */") {
		t.Error("Expected comment to be preserved")
	}
}

func TestScopeCSSEmptyLines(t *testing.T) {
	bundler := NewBundler("/out")

	css := `.a { color: red; }

.b { color: blue; }`

	scoped := bundler.scopeCSS(css, "/page.gxc")
	scopeID := bundler.GenerateScopeID("/page.gxc")
	expectedAttr := "[data-gxc-" + scopeID + "]"

	if !strings.Contains(scoped, expectedAttr+" .a") {
		t.Error("Expected scoped .a selector")
	}

	if !strings.Contains(scoped, expectedAttr+" .b") {
		t.Error("Expected scoped .b selector")
	}
}

func TestGenerateScopeID(t *testing.T) {
	bundler := NewBundler("/out")

	id1 := bundler.GenerateScopeID("/pages/index.gxc")
	id2 := bundler.GenerateScopeID("/pages/index.gxc")

	if id1 != id2 {
		t.Error("Expected same path to generate same scope ID")
	}

	if len(id1) != 6 {
		t.Errorf("Expected scope ID length 6, got %d", len(id1))
	}

	id3 := bundler.GenerateScopeID("/pages/about.gxc")
	if id1 == id3 {
		t.Error("Expected different paths to generate different scope IDs")
	}
}

func TestInjectAssets(t *testing.T) {
	bundler := NewBundler("/out")

	html := `<html>
<head>
<title>Test</title>
</head>
<body>
<h1>Hello</h1>
</body>
</html>`

	result := bundler.InjectAssets(html, "/_assets/styles.css", "/_assets/script.js", "abc123")

	if !strings.Contains(result, `<link rel="stylesheet" href="/_assets/styles.css">`) {
		t.Error("Expected CSS link tag in head")
	}

	if !strings.Contains(result, `<script type="module" src="/_assets/script.js"></script>`) {
		t.Error("Expected script tag before closing body")
	}

	if !strings.Contains(result, `<body data-gxc-abc123>`) {
		t.Error("Expected scope attribute on body")
	}

	headIdx := strings.Index(result, "</head>")
	cssIdx := strings.Index(result, "styles.css")
	if cssIdx >= headIdx {
		t.Error("Expected CSS link before </head>")
	}

	bodyCloseIdx := strings.Index(result, "</body>")
	jsIdx := strings.Index(result, "script.js")
	if jsIdx >= bodyCloseIdx {
		t.Error("Expected script before </body>")
	}
}

func TestInjectAssetsNoCSS(t *testing.T) {
	bundler := NewBundler("/out")

	html := `<html><head></head><body></body></html>`
	result := bundler.InjectAssets(html, "", "/_assets/script.js", "")

	if strings.Contains(result, "<link") {
		t.Error("Expected no CSS link when cssPath is empty")
	}

	if !strings.Contains(result, "script.js") {
		t.Error("Expected script tag")
	}
}

func TestInjectAssetsNoJS(t *testing.T) {
	bundler := NewBundler("/out")

	html := `<html><head></head><body></body></html>`
	result := bundler.InjectAssets(html, "/_assets/styles.css", "", "")

	if !strings.Contains(result, "styles.css") {
		t.Error("Expected CSS link")
	}

	if strings.Contains(result, "<script") {
		t.Error("Expected no script tag when jsPath is empty")
	}
}

func TestInjectAssetsNoScope(t *testing.T) {
	bundler := NewBundler("/out")

	html := `<html><head></head><body></body></html>`
	result := bundler.InjectAssets(html, "", "", "")

	if strings.Contains(result, "data-gxc-") {
		t.Error("Expected no scope attribute when scopeID is empty")
	}
}

func TestBundleWasmScripts(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping WASM integration test in short mode")
	}

	tmpDir := t.TempDir()
	bundler := NewBundler(tmpDir)

	comp := &parser.Component{
		Scripts: []parser.Script{
			{
				Content: `import "fmt"
fmt.Println("Hello WASM")`,
				Language: "go",
			},
		},
	}

	assets, err := bundler.BundleWasmScripts(comp, "/pages/test.gxc")
	if err != nil {
		t.Fatalf("BundleWasmScripts failed: %v", err)
	}

	if len(assets) != 1 {
		t.Fatalf("Expected 1 WASM asset, got %d", len(assets))
	}

	if !strings.HasPrefix(assets[0].WasmPath, "/_assets/wasm/script-") {
		t.Errorf("Expected WasmPath to start with /_assets/wasm/script-, got %s", assets[0].WasmPath)
	}

	if !strings.HasSuffix(assets[0].WasmPath, ".wasm") {
		t.Errorf("Expected WasmPath to end with .wasm, got %s", assets[0].WasmPath)
	}

	if !strings.HasPrefix(assets[0].LoaderPath, "/_assets/script-") {
		t.Errorf("Expected LoaderPath to start with /_assets/script-, got %s", assets[0].LoaderPath)
	}

	if !strings.HasSuffix(assets[0].LoaderPath, "-loader.js") {
		t.Errorf("Expected LoaderPath to end with -loader.js, got %s", assets[0].LoaderPath)
	}

	wasmFilePath := filepath.Join(tmpDir, strings.TrimPrefix(assets[0].WasmPath, "/"))
	if _, err := os.Stat(wasmFilePath); os.IsNotExist(err) {
		t.Errorf("Expected WASM file to exist at %s", wasmFilePath)
	}

	loaderFilePath := filepath.Join(tmpDir, strings.TrimPrefix(assets[0].LoaderPath, "/"))
	loaderContent, err := os.ReadFile(loaderFilePath)
	if err != nil {
		t.Fatalf("Failed to read loader file: %v", err)
	}

	loaderStr := string(loaderContent)
	if !strings.Contains(loaderStr, "WebAssembly.instantiateStreaming") {
		t.Error("Expected loader to contain WebAssembly.instantiateStreaming")
	}

	if !strings.Contains(loaderStr, assets[0].WasmPath) {
		t.Errorf("Expected loader to reference WASM path %s", assets[0].WasmPath)
	}
}

func TestBundleWasmScriptsEmpty(t *testing.T) {
	tmpDir := t.TempDir()
	bundler := NewBundler(tmpDir)

	comp := &parser.Component{
		Scripts: []parser.Script{},
	}

	assets, err := bundler.BundleWasmScripts(comp, "/pages/test.gxc")
	if err != nil {
		t.Fatalf("BundleWasmScripts failed: %v", err)
	}

	if len(assets) != 0 {
		t.Errorf("Expected 0 WASM assets for empty scripts, got %d", len(assets))
	}
}

func TestBundleWasmScriptsOnlyJS(t *testing.T) {
	tmpDir := t.TempDir()
	bundler := NewBundler(tmpDir)

	comp := &parser.Component{
		Scripts: []parser.Script{
			{
				Content:  "console.log('test');",
				Language: "javascript",
				IsModule: true,
			},
		},
	}

	assets, err := bundler.BundleWasmScripts(comp, "/pages/test.gxc")
	if err != nil {
		t.Fatalf("BundleWasmScripts failed: %v", err)
	}

	if len(assets) != 0 {
		t.Errorf("Expected 0 WASM assets for JS-only scripts, got %d", len(assets))
	}
}

func TestBundleWasmScriptsMultiple(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping WASM integration test in short mode")
	}

	tmpDir := t.TempDir()
	bundler := NewBundler(tmpDir)

	comp := &parser.Component{
		Scripts: []parser.Script{
			{
				Content: `import "fmt"
fmt.Println("WASM 1")`,
				Language: "go",
			},
			{
				Content:  "console.log('JS');",
				Language: "javascript",
			},
			{
				Content: `import "fmt"
fmt.Println("WASM 2")`,
				Language: "go",
			},
		},
	}

	assets, err := bundler.BundleWasmScripts(comp, "/pages/test.gxc")
	if err != nil {
		t.Fatalf("BundleWasmScripts failed: %v", err)
	}

	if len(assets) != 2 {
		t.Fatalf("Expected 2 WASM assets, got %d", len(assets))
	}

	if assets[0].WasmPath == assets[1].WasmPath {
		t.Error("Expected different WASM paths for different scripts")
	}
}

func TestInjectAssetsWithWasm(t *testing.T) {
	bundler := NewBundler("/out")

	html := `<html>
<head>
<title>Test</title>
</head>
<body>
<h1>Hello</h1>
</body>
</html>`

	wasmAssets := []WasmAsset{
		{
			WasmPath:   "/_assets/wasm/script-abc123.wasm",
			LoaderPath: "/_assets/script-abc123-loader.js",
		},
	}

	result := bundler.InjectAssetsWithWasm(html, "", "", "", wasmAssets)

	if !strings.Contains(result, `<script src="/wasm_exec.js"></script>`) {
		t.Error("Expected wasm_exec.js script tag")
	}

	if !strings.Contains(result, `<script src="/_assets/script-abc123-loader.js"></script>`) {
		t.Error("Expected loader script tag")
	}

	wasmExecIdx := strings.Index(result, "wasm_exec.js")
	loaderIdx := strings.Index(result, "loader.js")
	bodyCloseIdx := strings.Index(result, "</body>")

	if wasmExecIdx >= bodyCloseIdx {
		t.Error("Expected wasm_exec script before </body>")
	}

	if loaderIdx >= bodyCloseIdx {
		t.Error("Expected loader script before </body>")
	}

	if wasmExecIdx >= loaderIdx {
		t.Error("Expected wasm_exec.js to load before loader script")
	}
}

func TestInjectAssetsWithWasmMultiple(t *testing.T) {
	bundler := NewBundler("/out")

	html := `<html><head></head><body></body></html>`

	wasmAssets := []WasmAsset{
		{
			WasmPath:   "/_assets/wasm/script-aaa.wasm",
			LoaderPath: "/_assets/script-aaa-loader.js",
		},
		{
			WasmPath:   "/_assets/wasm/script-bbb.wasm",
			LoaderPath: "/_assets/script-bbb-loader.js",
		},
	}

	result := bundler.InjectAssetsWithWasm(html, "", "", "", wasmAssets)

	wasmExecCount := strings.Count(result, "wasm_exec.js")
	if wasmExecCount != 1 {
		t.Errorf("Expected wasm_exec.js to appear once, got %d times", wasmExecCount)
	}

	if !strings.Contains(result, "script-aaa-loader.js") {
		t.Error("Expected first loader script")
	}

	if !strings.Contains(result, "script-bbb-loader.js") {
		t.Error("Expected second loader script")
	}
}

func TestInjectAssetsWithWasmAndCSS(t *testing.T) {
	bundler := NewBundler("/out")

	html := `<html><head></head><body></body></html>`

	wasmAssets := []WasmAsset{
		{
			WasmPath:   "/_assets/wasm/script-test.wasm",
			LoaderPath: "/_assets/script-test-loader.js",
		},
	}

	result := bundler.InjectAssetsWithWasm(html, "/_assets/styles.css", "", "", wasmAssets)

	if !strings.Contains(result, `<link rel="stylesheet" href="/_assets/styles.css">`) {
		t.Error("Expected CSS link tag")
	}

	if !strings.Contains(result, "wasm_exec.js") {
		t.Error("Expected wasm_exec.js script tag")
	}

	cssIdx := strings.Index(result, "styles.css")
	headCloseIdx := strings.Index(result, "</head>")

	if cssIdx >= headCloseIdx {
		t.Error("Expected CSS link in head")
	}
}

func TestInjectAssetsWithWasmAndJS(t *testing.T) {
	bundler := NewBundler("/out")

	html := `<html><head></head><body></body></html>`

	wasmAssets := []WasmAsset{
		{
			WasmPath:   "/_assets/wasm/script-test.wasm",
			LoaderPath: "/_assets/script-test-loader.js",
		},
	}

	result := bundler.InjectAssetsWithWasm(html, "", "/_assets/script.js", "", wasmAssets)

	if !strings.Contains(result, "wasm_exec.js") {
		t.Error("Expected wasm_exec.js script tag")
	}

	if !strings.Contains(result, "script-test-loader.js") {
		t.Error("Expected WASM loader script tag")
	}

	if !strings.Contains(result, `<script type="module" src="/_assets/script.js"></script>`) {
		t.Error("Expected JS module script tag")
	}
}

func TestInjectAssetsComplete(t *testing.T) {
	bundler := NewBundler("/out")

	html := `<html><head></head><body></body></html>`

	wasmAssets := []WasmAsset{
		{
			WasmPath:   "/_assets/wasm/script-xyz.wasm",
			LoaderPath: "/_assets/script-xyz-loader.js",
		},
	}

	result := bundler.InjectAssetsWithWasm(html, "/_assets/styles.css", "/_assets/script.js", "abc123", wasmAssets)

	if !strings.Contains(result, `<link rel="stylesheet" href="/_assets/styles.css">`) {
		t.Error("Expected CSS link")
	}

	if !strings.Contains(result, "wasm_exec.js") {
		t.Error("Expected wasm_exec.js")
	}

	if !strings.Contains(result, "script-xyz-loader.js") {
		t.Error("Expected WASM loader")
	}

	if !strings.Contains(result, "script.js") {
		t.Error("Expected JS script")
	}

	if !strings.Contains(result, `<body data-gxc-abc123>`) {
		t.Error("Expected scope attribute on body")
	}
}

func TestBundleStylesHash(t *testing.T) {
	tmpDir := t.TempDir()
	bundler := NewBundler(tmpDir)

	comp1 := &parser.Component{
		Styles: []parser.Style{{Content: ".a { color: red; }", Scoped: false}},
	}

	comp2 := &parser.Component{
		Styles: []parser.Style{{Content: ".b { color: blue; }", Scoped: false}},
	}

	path1, _ := bundler.BundleStyles(comp1, "/page1.gxc")
	path2, _ := bundler.BundleStyles(comp2, "/page2.gxc")

	if path1 == path2 {
		t.Error("Expected different content to produce different hashes")
	}

	path3, _ := bundler.BundleStyles(comp1, "/page1.gxc")
	if path1 != path3 {
		t.Error("Expected same content to produce same hash")
	}
}

func TestBundleScriptsHash(t *testing.T) {
	tmpDir := t.TempDir()
	bundler := NewBundler(tmpDir)

	comp1 := &parser.Component{
		Scripts: []parser.Script{{Content: "console.log(1);", IsModule: false}},
	}

	comp2 := &parser.Component{
		Scripts: []parser.Script{{Content: "console.log(2);", IsModule: false}},
	}

	path1, _ := bundler.BundleScripts(comp1, "/page1.gxc")
	path2, _ := bundler.BundleScripts(comp2, "/page2.gxc")

	if path1 == path2 {
		t.Error("Expected different content to produce different hashes")
	}
}
