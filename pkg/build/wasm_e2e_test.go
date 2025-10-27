package build

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/withgalaxy/galaxy/pkg/config"
)

func TestSSGBuildWithWasm(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	tmpDir := t.TempDir()
	srcDir := filepath.Join(tmpDir, "src")
	distDir := filepath.Join(tmpDir, "dist")

	if err := os.MkdirAll(filepath.Join(srcDir, "pages"), 0755); err != nil {
		t.Fatalf("Failed to create pages dir: %v", err)
	}

	indexContent := `---
var title = "WASM SSG Test"
---
<html>
<head>
    <title>{title}</title>
</head>
<body>
    <h1>{title}</h1>
    <button id="btn">Click me</button>
    <div id="output"></div>
</body>
</html>

<script>
import "fmt"
import "github.com/withgalaxy/galaxy/pkg/wasmdom"

clicks := 0
btn := wasmdom.GetElementById("btn")
output := wasmdom.GetElementById("output")

btn.AddEventListener("click", func() {
    clicks++
    output.SetTextContent(fmt.Sprintf("Clicked %d times", clicks))
})
</script>
`

	indexPath := filepath.Join(srcDir, "pages", "index.gxc")
	if err := os.WriteFile(indexPath, []byte(indexContent), 0644); err != nil {
		t.Fatalf("Failed to write index.gxc: %v", err)
	}

	cfg := config.DefaultConfig()
	pagesDir := filepath.Join(srcDir, "pages")
	publicDir := filepath.Join(srcDir, "public")
	os.MkdirAll(publicDir, 0755)

	builder := NewSSGBuilder(cfg, srcDir, pagesDir, distDir, publicDir)
	if err := builder.Build(); err != nil {
		t.Fatalf("SSG Build failed: %v", err)
	}

	indexHTML := filepath.Join(distDir, "index.html")
	htmlContent, err := os.ReadFile(indexHTML)
	if err != nil {
		t.Fatalf("Failed to read generated HTML: %v", err)
	}

	htmlStr := string(htmlContent)

	if !strings.Contains(htmlStr, "WASM SSG Test") {
		t.Error("Expected title in HTML")
	}

	if !strings.Contains(htmlStr, `<button id="btn">`) {
		t.Error("Expected button element in HTML")
	}

	if !strings.Contains(htmlStr, `<script src="/wasm_exec.js"></script>`) {
		t.Error("Expected wasm_exec.js script tag")
	}

	if !strings.Contains(htmlStr, "-loader.js") {
		t.Error("Expected WASM loader script tag")
	}

	wasmDir := filepath.Join(distDir, "_assets", "wasm")
	entries, err := os.ReadDir(wasmDir)
	if err != nil {
		t.Fatalf("Failed to read WASM assets dir: %v", err)
	}

	wasmFound := false
	for _, entry := range entries {
		if strings.HasSuffix(entry.Name(), ".wasm") {
			wasmFound = true
			info, _ := entry.Info()
			if info.Size() == 0 {
				t.Error("WASM file should not be empty")
			}
			break
		}
	}

	if !wasmFound {
		t.Error("Expected WASM file in _assets/wasm/")
	}

	assetsDir := filepath.Join(distDir, "_assets")
	assetsEntries, err := os.ReadDir(assetsDir)
	if err != nil {
		t.Fatalf("Failed to read assets dir: %v", err)
	}

	loaderFound := false
	for _, entry := range assetsEntries {
		if strings.HasSuffix(entry.Name(), "-loader.js") {
			loaderFound = true
			loaderPath := filepath.Join(assetsDir, entry.Name())
			loaderContent, err := os.ReadFile(loaderPath)
			if err != nil {
				t.Fatalf("Failed to read loader: %v", err)
			}

			loaderStr := string(loaderContent)
			if !strings.Contains(loaderStr, "WebAssembly.instantiateStreaming") {
				t.Error("Loader should contain WebAssembly.instantiateStreaming")
			}

			if !strings.Contains(loaderStr, "/_assets/wasm/script-") {
				t.Error("Loader should reference WASM file")
			}
			break
		}
	}

	if !loaderFound {
		t.Error("Expected WASM loader file in _assets/")
	}

	wasmExecPath := filepath.Join(distDir, "wasm_exec.js")
	if _, err := os.Stat(wasmExecPath); os.IsNotExist(err) {
		t.Error("Expected wasm_exec.js to be copied to dist root")
	}
}

func TestSSGBuildWithMultipleWasmScripts(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	tmpDir := t.TempDir()
	srcDir := filepath.Join(tmpDir, "src")
	distDir := filepath.Join(tmpDir, "dist")

	if err := os.MkdirAll(filepath.Join(srcDir, "pages"), 0755); err != nil {
		t.Fatalf("Failed to create pages dir: %v", err)
	}

	pageContent := `---
var title = "Multiple WASM Scripts"
---
<html>
<head><title>{title}</title></head>
<body>
    <h1>{title}</h1>
    <div id="output1"></div>
    <div id="output2"></div>
</body>
</html>

<script>
import "github.com/withgalaxy/galaxy/pkg/wasmdom"

output := wasmdom.GetElementById("output1")
output.SetTextContent("Script 1 loaded")
</script>

<script>
import "github.com/withgalaxy/galaxy/pkg/wasmdom"

output := wasmdom.GetElementById("output2")
output.SetTextContent("Script 2 loaded")
</script>
`

	pagePath := filepath.Join(srcDir, "pages", "multi.gxc")
	if err := os.WriteFile(pagePath, []byte(pageContent), 0644); err != nil {
		t.Fatalf("Failed to write multi.gxc: %v", err)
	}

	cfg := config.DefaultConfig()
	pagesDir := filepath.Join(srcDir, "pages")
	publicDir := filepath.Join(srcDir, "public")
	os.MkdirAll(publicDir, 0755)

	builder := NewSSGBuilder(cfg, srcDir, pagesDir, distDir, publicDir)
	if err := builder.Build(); err != nil {
		t.Fatalf("SSG Build failed: %v", err)
	}

	multiHTML := filepath.Join(distDir, "multi", "index.html")
	htmlContent, err := os.ReadFile(multiHTML)
	if err != nil {
		t.Fatalf("Failed to read generated HTML: %v", err)
	}

	htmlStr := string(htmlContent)

	loaderCount := strings.Count(htmlStr, "-loader.js")
	if loaderCount != 2 {
		t.Errorf("Expected 2 WASM loader scripts, got %d", loaderCount)
	}

	wasmExecCount := strings.Count(htmlStr, "wasm_exec.js")
	if wasmExecCount != 1 {
		t.Errorf("Expected wasm_exec.js to appear once, got %d times", wasmExecCount)
	}

	wasmDir := filepath.Join(distDir, "_assets", "wasm")
	entries, err := os.ReadDir(wasmDir)
	if err != nil {
		t.Fatalf("Failed to read WASM dir: %v", err)
	}

	wasmCount := 0
	for _, entry := range entries {
		if strings.HasSuffix(entry.Name(), ".wasm") {
			wasmCount++
		}
	}

	if wasmCount != 2 {
		t.Errorf("Expected 2 WASM files, got %d", wasmCount)
	}
}

func TestSSGBuildWithMixedScripts(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	tmpDir := t.TempDir()
	srcDir := filepath.Join(tmpDir, "src")
	distDir := filepath.Join(tmpDir, "dist")

	if err := os.MkdirAll(filepath.Join(srcDir, "pages"), 0755); err != nil {
		t.Fatalf("Failed to create pages dir: %v", err)
	}

	pageContent := `---
var title = "Mixed Scripts"
---
<html>
<head><title>{title}</title></head>
<body>
    <h1>{title}</h1>
    <div id="js-output"></div>
    <div id="wasm-output"></div>
</body>
</html>

<script type="module">
console.log("JavaScript module");
document.getElementById('js-output').textContent = "JS loaded";
</script>

<script>
import "github.com/withgalaxy/galaxy/pkg/wasmdom"

output := wasmdom.GetElementById("wasm-output")
output.SetTextContent("WASM loaded")
</script>
`

	pagePath := filepath.Join(srcDir, "pages", "mixed.gxc")
	if err := os.WriteFile(pagePath, []byte(pageContent), 0644); err != nil {
		t.Fatalf("Failed to write mixed.gxc: %v", err)
	}

	cfg := config.DefaultConfig()
	pagesDir := filepath.Join(srcDir, "pages")
	publicDir := filepath.Join(srcDir, "public")
	os.MkdirAll(publicDir, 0755)

	builder := NewSSGBuilder(cfg, srcDir, pagesDir, distDir, publicDir)
	if err := builder.Build(); err != nil {
		t.Fatalf("SSG Build failed: %v", err)
	}

	mixedHTML := filepath.Join(distDir, "mixed", "index.html")
	htmlContent, err := os.ReadFile(mixedHTML)
	if err != nil {
		t.Fatalf("Failed to read generated HTML: %v", err)
	}

	htmlStr := string(htmlContent)

	if !strings.Contains(htmlStr, "wasm_exec.js") {
		t.Error("Expected wasm_exec.js for WASM script")
	}

	if !strings.Contains(htmlStr, "-loader.js") {
		t.Error("Expected WASM loader")
	}

	if !strings.Contains(htmlStr, `type="module"`) {
		t.Error("Expected JS module script")
	}

	assetsDir := filepath.Join(distDir, "_assets")
	entries, err := os.ReadDir(assetsDir)
	if err != nil {
		t.Fatalf("Failed to read assets dir: %v", err)
	}

	jsFound := false
	loaderFound := false

	for _, entry := range entries {
		name := entry.Name()
		if strings.HasPrefix(name, "script-") && strings.HasSuffix(name, ".js") && !strings.Contains(name, "loader") {
			jsFound = true
		}
		if strings.HasSuffix(name, "-loader.js") {
			loaderFound = true
		}
	}

	if !jsFound {
		t.Error("Expected JavaScript bundle in assets")
	}

	if !loaderFound {
		t.Error("Expected WASM loader in assets")
	}
}

func TestSSGBuildNoWasm(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	tmpDir := t.TempDir()
	srcDir := filepath.Join(tmpDir, "src")
	distDir := filepath.Join(tmpDir, "dist")

	if err := os.MkdirAll(filepath.Join(srcDir, "pages"), 0755); err != nil {
		t.Fatalf("Failed to create pages dir: %v", err)
	}

	pageContent := `---
var title = "No WASM"
---
<html>
<head><title>{title}</title></head>
<body>
    <h1>{title}</h1>
</body>
</html>

<script type="module">
console.log("Just JavaScript");
</script>
`

	pagePath := filepath.Join(srcDir, "pages", "no-wasm.gxc")
	if err := os.WriteFile(pagePath, []byte(pageContent), 0644); err != nil {
		t.Fatalf("Failed to write page: %v", err)
	}

	cfg := config.DefaultConfig()
	pagesDir := filepath.Join(srcDir, "pages")
	publicDir := filepath.Join(srcDir, "public")
	os.MkdirAll(publicDir, 0755)

	builder := NewSSGBuilder(cfg, srcDir, pagesDir, distDir, publicDir)
	if err := builder.Build(); err != nil {
		t.Fatalf("SSG Build failed: %v", err)
	}

	htmlPath := filepath.Join(distDir, "no-wasm", "index.html")
	htmlContent, err := os.ReadFile(htmlPath)
	if err != nil {
		t.Fatalf("Failed to read HTML: %v", err)
	}

	htmlStr := string(htmlContent)

	if strings.Contains(htmlStr, "wasm_exec.js") {
		t.Error("Should not include wasm_exec.js when no WASM scripts present")
	}

	if strings.Contains(htmlStr, "-loader.js") {
		t.Error("Should not include WASM loader when no WASM scripts present")
	}

	if !strings.Contains(htmlStr, `type="module"`) {
		t.Error("Should still include JavaScript module")
	}

	wasmDir := filepath.Join(distDir, "_assets", "wasm")
	if _, err := os.Stat(wasmDir); err == nil {
		entries, _ := os.ReadDir(wasmDir)
		if len(entries) > 0 {
			t.Error("WASM directory should be empty or not exist when no WASM scripts")
		}
	}
}
