package wasm

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/withgalaxy/galaxy/pkg/moduleutil"
)

func TestNewCompiler(t *testing.T) {
	tmpDir := t.TempDir()
	cacheDir := t.TempDir()

	compiler := NewCompiler(tmpDir, cacheDir)

	if compiler.TempDir != tmpDir {
		t.Errorf("Expected TempDir %s, got %s", tmpDir, compiler.TempDir)
	}

	if compiler.CacheDir != cacheDir {
		t.Errorf("Expected CacheDir %s, got %s", cacheDir, compiler.CacheDir)
	}

	if compiler.UseTinyGo {
		t.Error("Expected UseTinyGo to be false by default")
	}
}

func TestCompileSimpleScript(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	tmpDir := t.TempDir()
	cacheDir := t.TempDir()
	compiler := NewCompiler(tmpDir, cacheDir)

	script := `import "fmt"

fmt.Println("Hello from WASM")`

	module, err := compiler.Compile(script, "/pages/test.gxc")
	if err != nil {
		t.Fatalf("Compile failed: %v", err)
	}

	if module.WasmPath == "" {
		t.Error("Expected WasmPath to be set")
	}

	if module.Hash == "" {
		t.Error("Expected Hash to be set")
	}

	if len(module.Hash) != 8 {
		t.Errorf("Expected Hash length 8, got %d", len(module.Hash))
	}

	if _, err := os.Stat(module.WasmPath); os.IsNotExist(err) {
		t.Errorf("Expected WASM file to exist at %s", module.WasmPath)
	}

	info, _ := os.Stat(module.WasmPath)
	if info.Size() == 0 {
		t.Error("Expected WASM file to have non-zero size")
	}
}

func TestCompileWithImports(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	tmpDir := t.TempDir()
	cacheDir := t.TempDir()
	compiler := NewCompiler(tmpDir, cacheDir)

	script := `import "fmt"
import "github.com/withgalaxy/galaxy/pkg/wasmdom"

wasmdom.ConsoleLog("Testing WASM DOM")
fmt.Println("Multiple imports")`

	module, err := compiler.Compile(script, "/pages/test.gxc")
	if err != nil {
		t.Fatalf("Compile with imports failed: %v", err)
	}

	if module.WasmPath == "" {
		t.Error("Expected WasmPath to be set")
	}
}

func TestCompileWithMainFunc(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	tmpDir := t.TempDir()
	cacheDir := t.TempDir()
	compiler := NewCompiler(tmpDir, cacheDir)

	script := `import "fmt"

func main() {
	fmt.Println("Custom main")
	select {}
}`

	module, err := compiler.Compile(script, "/pages/test.gxc")
	if err != nil {
		t.Fatalf("Compile with main func failed: %v", err)
	}

	if module.WasmPath == "" {
		t.Error("Expected WasmPath to be set")
	}
}

func TestCompileCaching(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	tmpDir := t.TempDir()
	cacheDir := t.TempDir()
	compiler := NewCompiler(tmpDir, cacheDir)

	script := `import "fmt"
fmt.Println("Cached test")`

	module1, err := compiler.Compile(script, "/pages/test.gxc")
	if err != nil {
		t.Fatalf("First compile failed: %v", err)
	}

	module2, err := compiler.Compile(script, "/pages/test.gxc")
	if err != nil {
		t.Fatalf("Second compile failed: %v", err)
	}

	if module1.Hash != module2.Hash {
		t.Error("Expected same hash for identical scripts")
	}

	if module1.WasmPath != module2.WasmPath {
		t.Error("Expected same WasmPath for cached compilation")
	}

	differentScript := `import "fmt"
fmt.Println("Different script")`

	module3, err := compiler.Compile(differentScript, "/pages/test.gxc")
	if err != nil {
		t.Fatalf("Third compile failed: %v", err)
	}

	if module1.Hash == module3.Hash {
		t.Error("Expected different hash for different scripts")
	}
}

func TestExtractImports(t *testing.T) {
	script := `import "fmt"
import "strings"

fmt.Println("test")`

	imports := extractImports(script)

	if len(imports) != 2 {
		t.Errorf("Expected 2 import statements, got %d", len(imports))
	}

	if !contains(imports, `"fmt"`) {
		t.Error("Expected fmt import")
	}

	if !contains(imports, `"strings"`) {
		t.Error("Expected strings import")
	}
}

func TestRemoveImports(t *testing.T) {
	script := `import "fmt"
import "strings"

fmt.Println("hello")
strings.ToUpper("world")`

	result := removeImports(script)

	if strings.Contains(result, "import") {
		t.Error("Expected imports to be removed")
	}

	if !strings.Contains(result, "fmt.Println") {
		t.Error("Expected code to remain")
	}
}

func TestRemovePackageDecl(t *testing.T) {
	script := `package main

func main() {}`

	result := removePackageDecl(script)

	if strings.Contains(result, "package main") {
		t.Error("Expected package declaration to be removed")
	}

	if !strings.Contains(result, "func main()") {
		t.Error("Expected function to remain")
	}
}

func TestContainsMainFunc(t *testing.T) {
	tests := []struct {
		name     string
		body     string
		expected bool
	}{
		{
			name: "has main func",
			body: `func main() {
	fmt.Println("hello")
}`,
			expected: true,
		},
		{
			name: "no main func",
			body: `func helper() {
	fmt.Println("helper")
}`,
			expected: false,
		},
		{
			name:     "just code",
			body:     `fmt.Println("hello")`,
			expected: false,
		},
		{
			name: "main func with select",
			body: `func main() {
	setup()
	select {}
}`,
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := containsMainFunc(tt.body)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v for body: %s", tt.expected, result, tt.body)
			}
		})
	}
}

func TestIndentCode(t *testing.T) {
	code := `fmt.Println("line1")
fmt.Println("line2")
fmt.Println("line3")`

	result := indentCode(code)

	lines := strings.Split(result, "\n")
	for _, line := range lines {
		if !strings.HasPrefix(line, "\t") {
			t.Errorf("Expected line to be indented: %s", line)
		}
	}
}

func TestIndentCodeEmpty(t *testing.T) {
	result := indentCode("")

	if result != "" {
		t.Errorf("Expected empty string, got %q", result)
	}
}

func TestPrepareScriptSimple(t *testing.T) {
	script := `import "fmt"

fmt.Println("hello")`

	result, err := prepareScript(script, "abc12345", false)
	if err != nil {
		t.Fatalf("prepareScript failed: %v", err)
	}

	if !strings.Contains(result, "package main") {
		t.Error("Expected package main declaration")
	}

	if !strings.Contains(result, `import (`) {
		t.Error("Expected import block")
	}

	if !strings.Contains(result, `"fmt"`) {
		t.Error("Expected fmt import")
	}

	if !strings.Contains(result, "func main()") {
		t.Error("Expected main function wrapper")
	}

	if !strings.Contains(result, "select {}") {
		t.Error("Expected select statement to keep process alive")
	}

	if !strings.Contains(result, "fmt.Println") {
		t.Error("Expected original code in main")
	}
}

func TestPrepareScriptWithMain(t *testing.T) {
	script := `import "fmt"

func main() {
	fmt.Println("custom main")
	select {}
}`

	result, err := prepareScript(script, "xyz98765", false)
	if err != nil {
		t.Fatalf("prepareScript failed: %v", err)
	}

	if !strings.Contains(result, "package main") {
		t.Error("Expected package main declaration")
	}

	mainCount := strings.Count(result, "func main()")
	if mainCount != 1 {
		t.Errorf("Expected exactly 1 main function, got %d", mainCount)
	}
}

func TestPrepareScriptTinyGo(t *testing.T) {
	script := `import "fmt"
fmt.Println("tinygo test")`

	result, err := prepareScript(script, "def45678", true)
	if err != nil {
		t.Fatalf("prepareScript failed: %v", err)
	}

	if !strings.Contains(result, "package main") {
		t.Error("Expected 'package main' for TinyGo")
	}

	if strings.Contains(result, "wasmscript_") {
		t.Error("TinyGo should not use custom package name")
	}
}

func TestPrepareScriptWithHelperFunctions(t *testing.T) {
	script := `import "fmt"

func helper(x int) int {
	return x * 2
}

func anotherHelper() {
	fmt.Println("helper")
}

result := helper(5)
fmt.Println(result)`

	result, err := prepareScript(script, "abc12345", false)
	if err != nil {
		t.Fatalf("prepareScript failed: %v", err)
	}

	if !strings.Contains(result, "package main") {
		t.Error("Expected package main")
	}

	if !strings.Contains(result, "func helper(x int) int") {
		t.Error("Expected helper function at package level")
	}

	if !strings.Contains(result, "func anotherHelper()") {
		t.Error("Expected anotherHelper function at package level")
	}

	if !strings.Contains(result, "func main()") {
		t.Error("Expected main function")
	}

	helperIdx := strings.Index(result, "func helper")
	mainIdx := strings.Index(result, "func main()")

	if helperIdx == -1 || mainIdx == -1 || helperIdx >= mainIdx {
		t.Error("Expected helper functions to appear before main function")
	}

	if !strings.Contains(result, "result := helper(5)") {
		t.Error("Expected executable code in main")
	}

	lines := strings.Split(result, "\n")
	inMain := false
	for _, line := range lines {
		if strings.Contains(line, "func main()") {
			inMain = true
		}
		if inMain && strings.Contains(line, "func helper") {
			t.Error("Helper function should not be nested inside main")
		}
	}
}

func TestPrepareScriptWithVariables(t *testing.T) {
	script := `import "syscall/js"

var counter js.Value
var currentState = "active"

func increment() {
	counter.Set("value", counter.Get("value").Int()+1)
}

counter = js.Global().Get("document").Call("getElementById", "counter")
increment()`

	result, err := prepareScript(script, "test1234", false)
	if err != nil {
		t.Fatalf("prepareScript failed: %v", err)
	}

	if !strings.Contains(result, "var counter js.Value") {
		t.Error("Expected counter variable at package level")
	}

	if !strings.Contains(result, `var currentState = "active"`) {
		t.Error("Expected currentState variable at package level")
	}

	varIdx := strings.Index(result, "var counter")
	funcIdx := strings.Index(result, "func increment")
	mainIdx := strings.Index(result, "func main()")

	if varIdx == -1 || funcIdx == -1 || mainIdx == -1 {
		t.Error("Expected all declarations to be present")
	}

	if varIdx >= funcIdx {
		t.Error("Expected variables before functions")
	}

	if funcIdx >= mainIdx {
		t.Error("Expected functions before main")
	}

	lines := strings.Split(result, "\n")
	inMain := false
	for _, line := range lines {
		if strings.Contains(line, "func main()") {
			inMain = true
		}
		if inMain && (strings.HasPrefix(strings.TrimSpace(line), "var counter") || strings.HasPrefix(strings.TrimSpace(line), "var currentState")) {
			t.Error("Variables should not be inside main function")
		}
	}
}

func TestFindModuleRoot(t *testing.T) {
	root := moduleutil.FindGalaxyModuleRoot()

	if root == "" {
		t.Error("Expected to find module root")
	}

	goModPath := filepath.Join(root, "go.mod")
	if _, err := os.Stat(goModPath); os.IsNotExist(err) {
		t.Errorf("Expected go.mod to exist at %s", goModPath)
	}
}

func TestCompileInvalidSyntax(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	tmpDir := t.TempDir()
	cacheDir := t.TempDir()
	compiler := NewCompiler(tmpDir, cacheDir)

	script := `import "fmt"

fmt.Println("unclosed string`

	_, err := compiler.Compile(script, "/pages/test.gxc")
	if err == nil {
		t.Error("Expected compilation to fail with invalid syntax")
	}

	if !strings.Contains(err.Error(), "compile failed") {
		t.Errorf("Expected 'compile failed' error, got: %v", err)
	}
}

func contains(slice []string, str string) bool {
	for _, s := range slice {
		if s == str {
			return true
		}
	}
	return false
}
