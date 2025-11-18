package wasm

import (
	"crypto/sha256"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/withgalaxy/galaxy/pkg/cli"
	"github.com/withgalaxy/galaxy/pkg/moduleutil"
)

type Compiler struct {
	TempDir   string
	CacheDir  string
	UseTinyGo bool
}

type CompiledModule struct {
	WasmPath   string
	LoaderPath string
	Hash       string
}

func NewCompiler(tempDir, cacheDir string) *Compiler {
	return &Compiler{
		TempDir:  tempDir,
		CacheDir: cacheDir,
	}
}

func (c *Compiler) Compile(script, pagePath string) (*CompiledModule, error) {
	hash := fmt.Sprintf("%x", sha256.Sum256([]byte(script)))[:8]

	cachedWasm := filepath.Join(c.CacheDir, fmt.Sprintf("script-%s.wasm", hash))
	if _, err := os.Stat(cachedWasm); err == nil {
		return &CompiledModule{
			WasmPath: cachedWasm,
			Hash:     hash,
		}, nil
	}

	buildDir := filepath.Join(c.TempDir, hash)
	if err := os.MkdirAll(buildDir, 0755); err != nil {
		return nil, fmt.Errorf("create build dir: %w", err)
	}
	defer os.RemoveAll(buildDir)

	moduleID := filepath.Base(pagePath)
	scriptWithHMR := injectHMRHelpers(script, moduleID)
	preparedScript, err := prepareScript(scriptWithHMR, hash, c.UseTinyGo && isTinyGoAvailable())
	if err != nil {
		return nil, fmt.Errorf("prepare script: %w", err)
	}

	mainGo := filepath.Join(buildDir, "main.go")
	if err := os.WriteFile(mainGo, []byte(preparedScript), 0644); err != nil {
		return nil, fmt.Errorf("write main.go: %w", err)
	}

	goMod := filepath.Join(buildDir, "go.mod")
	moduleRoot := moduleutil.FindGalaxyModuleRoot()
	moduleContent := "module wasmscript\n\ngo 1.21\n\n" + moduleutil.GetGalaxyModuleRequirement(moduleRoot, cli.Version)

	if err := os.WriteFile(goMod, []byte(moduleContent), 0644); err != nil {
		return nil, fmt.Errorf("write go.mod: %w", err)
	}

	tidyCmd := exec.Command("go", "mod", "tidy")
	tidyCmd.Dir = buildDir
	if output, err := tidyCmd.CombinedOutput(); err != nil {
		return nil, fmt.Errorf("go mod tidy failed: %s\n%s", err, output)
	}

	outWasm := filepath.Join(buildDir, "script.wasm")
	absOutWasm, err := filepath.Abs(outWasm)
	if err != nil {
		absOutWasm = outWasm
	}

	var cmd *exec.Cmd
	if c.UseTinyGo && isTinyGoAvailable() {
		cmd = exec.Command("tinygo", "build", "-o", absOutWasm, "-target", "wasm", ".")
		cmd.Dir = buildDir
	} else {
		cmd = exec.Command("go", "build", "-o", absOutWasm, ".")
		cmd.Dir = buildDir
		cmd.Env = append(os.Environ(),
			"GOOS=js",
			"GOARCH=wasm",
		)
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("compile failed: %s\n%s", err, output)
	}

	if _, err := os.Stat(outWasm); os.IsNotExist(err) {
		entries, _ := os.ReadDir(buildDir)
		var files []string
		for _, e := range entries {
			files = append(files, e.Name())
		}
		return nil, fmt.Errorf("wasm file not generated at %s, build output: %s, files in buildDir: %v", outWasm, output, files)
	}

	if err := os.MkdirAll(c.CacheDir, 0755); err != nil {
		return nil, fmt.Errorf("create cache dir: %w", err)
	}

	finalWasm := filepath.Join(c.CacheDir, fmt.Sprintf("script-%s.wasm", hash))
	if err := os.Rename(outWasm, finalWasm); err != nil {
		data, err := os.ReadFile(outWasm)
		if err != nil {
			return nil, fmt.Errorf("read wasm: %w", err)
		}
		if err := os.WriteFile(finalWasm, data, 0644); err != nil {
			return nil, fmt.Errorf("write cached wasm: %w", err)
		}
	}

	return &CompiledModule{
		WasmPath: finalWasm,
		Hash:     hash,
	}, nil
}

func prepareScript(script, hash string, useTinyGo bool) (string, error) {
	imports := extractImports(script)
	body := removeImports(script)
	body = removePackageDecl(body)
	hasMain := containsMainFunc(body)

	// Auto-add syscall/js if HMR helpers or js.* usage detected
	needsSyscallJS := strings.Contains(script, "js.") || strings.Contains(script, "hmrAccept") || strings.Contains(script, "hmrOnDispose")
	if needsSyscallJS {
		hasSyscallJS := false
		for _, imp := range imports {
			if strings.Contains(imp, "syscall/js") {
				hasSyscallJS = true
				break
			}
		}
		if !hasSyscallJS {
			imports = append([]string{`"syscall/js"`}, imports...)
		}
	}

	var final strings.Builder
	final.WriteString("package main\n\n")

	if len(imports) > 0 {
		final.WriteString("import (\n")
		for _, imp := range imports {
			final.WriteString(fmt.Sprintf("\t%s\n", imp))
		}
		final.WriteString(")\n\n")
	}

	if !hasMain {
		vars, funcs, execCode := separateFunctionsFromCode(body)

		for _, v := range vars {
			final.WriteString(v)
			final.WriteString("\n")
		}
		if len(vars) > 0 {
			final.WriteString("\n")
		}

		for _, fn := range funcs {
			final.WriteString(fn)
			final.WriteString("\n\n")
		}

		// Create a rerunnable function for HMR
		final.WriteString("func __galaxyRun() {\n")
		if execCode != "" {
			final.WriteString(indentCode(execCode))
			final.WriteString("\n")
		}
		final.WriteString("}\n\n")

		final.WriteString("func main() {\n")
		final.WriteString("\t__galaxyRun()\n")
		final.WriteString("\t\n")
		final.WriteString("\t// Auto-register HMR accept handler if not manually registered\n")
		final.WriteString("\tif !__hmrManuallyRegistered {\n")
		final.WriteString("\t\thmrAccept(__galaxyRun)\n")
		final.WriteString("\t}\n")
		final.WriteString("\t\n")
		final.WriteString("\tselect {}\n")
		final.WriteString("}\n")
	} else {
		final.WriteString(body)
	}

	return final.String(), nil
}

func extractImports(script string) []string {
	var imports []string
	importRegex := regexp.MustCompile(`(?m)^import\s+(.+)$`)
	matches := importRegex.FindAllStringSubmatch(script, -1)

	for _, match := range matches {
		imports = append(imports, match[1])
	}

	return imports
}

func removeImports(script string) string {
	importRegex := regexp.MustCompile(`(?m)^import\s+.+$\n?`)
	return importRegex.ReplaceAllString(script, "")
}

func removePackageDecl(script string) string {
	pkgRegex := regexp.MustCompile(`(?m)^package\s+\w+\s*\n?`)
	return pkgRegex.ReplaceAllString(script, "")
}

func containsMainFunc(body string) bool {
	fset := token.NewFileSet()
	wrapped := "package temp\n" + body

	f, err := parser.ParseFile(fset, "", wrapped, 0)
	if err != nil {
		return false
	}

	for _, decl := range f.Decls {
		if fn, ok := decl.(*ast.FuncDecl); ok {
			if fn.Name.Name == "main" {
				return true
			}
		}
	}

	return false
}

func separateFunctionsFromCode(body string) (variables []string, functions []string, executableCode string) {
	lines := strings.Split(body, "\n")
	var funcLines []string
	var execLines []string
	inFunc := false
	braceDepth := 0

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		if !inFunc && strings.HasPrefix(trimmed, "func ") {
			inFunc = true
			funcLines = append(funcLines, line)
			braceDepth = strings.Count(line, "{") - strings.Count(line, "}")
			if braceDepth == 0 {
				inFunc = false
			}
			continue
		}

		if inFunc {
			funcLines = append(funcLines, line)
			braceDepth += strings.Count(line, "{") - strings.Count(line, "}")
			if braceDepth == 0 {
				inFunc = false
				functions = append(functions, strings.Join(funcLines, "\n"))
				funcLines = nil
			}
			continue
		}

		if trimmed != "" {
			if strings.HasPrefix(trimmed, "var ") || strings.HasPrefix(trimmed, "const ") {
				variables = append(variables, line)
			} else {
				execLines = append(execLines, line)
			}
		}
	}

	executableCode = strings.Join(execLines, "\n")
	return variables, functions, executableCode
}

func indentCode(code string) string {
	lines := strings.Split(strings.TrimSpace(code), "\n")
	var result strings.Builder

	for i, line := range lines {
		if i > 0 {
			result.WriteString("\n")
		}
		if strings.TrimSpace(line) != "" {
			result.WriteString("\t")
		}
		result.WriteString(line)
	}

	return result.String()
}

func isTinyGoAvailable() bool {
	_, err := exec.LookPath("tinygo")
	return err == nil
}

func injectHMRHelpers(script, moduleID string) string {
	var sb strings.Builder

	sb.WriteString("// HMR helpers (auto-injected)\n")
	sb.WriteString(fmt.Sprintf("var __hmrModuleID = %q\n", moduleID))
	sb.WriteString("var __hmrManuallyRegistered = false\n\n")

	sb.WriteString("func hmrAccept(callback func()) {\n")
	sb.WriteString("\t__hmrManuallyRegistered = true\n")
	sb.WriteString("\tensureHMRGlobals()\n")
	sb.WriteString("\thandlers := js.Global().Get(\"__galaxyWasmAcceptHandlers\")\n")
	sb.WriteString("\tcb := js.FuncOf(func(this js.Value, args []js.Value) interface{} {\n")
	sb.WriteString("\t\tcallback()\n")
	sb.WriteString("\t\treturn nil\n")
	sb.WriteString("\t})\n")
	sb.WriteString("\thandlers.Set(__hmrModuleID, cb)\n")
	sb.WriteString("}\n\n")

	sb.WriteString("func hmrOnDispose(handler func()) {\n")
	sb.WriteString("\tensureHMRGlobals()\n")
	sb.WriteString("\tmodules := js.Global().Get(\"__galaxyWasmModules\")\n")
	sb.WriteString("\tmodule := modules.Get(__hmrModuleID)\n")
	sb.WriteString("\tif !module.IsUndefined() {\n")
	sb.WriteString("\t\tcb := js.FuncOf(func(this js.Value, args []js.Value) interface{} {\n")
	sb.WriteString("\t\t\thandler()\n")
	sb.WriteString("\t\t\treturn nil\n")
	sb.WriteString("\t\t})\n")
	sb.WriteString("\t\tmodule.Set(\"disposeHandler\", cb)\n")
	sb.WriteString("\t}\n")
	sb.WriteString("}\n\n")

	sb.WriteString("func hmrSaveState(key string, value interface{}) {\n")
	sb.WriteString("\tensureHMRGlobals()\n")
	sb.WriteString("\tstate := js.Global().Get(\"__galaxyWasmState\")\n")
	sb.WriteString("\tmoduleState := state.Get(__hmrModuleID)\n")
	sb.WriteString("\tif moduleState.IsUndefined() {\n")
	sb.WriteString("\t\tmoduleState = js.Global().Get(\"Object\").New()\n")
	sb.WriteString("\t\tstate.Set(__hmrModuleID, moduleState)\n")
	sb.WriteString("\t}\n")
	sb.WriteString("\tmoduleState.Set(key, js.ValueOf(value))\n")
	sb.WriteString("}\n\n")

	sb.WriteString("func hmrLoadState(key string) js.Value {\n")
	sb.WriteString("\tensureHMRGlobals()\n")
	sb.WriteString("\tstate := js.Global().Get(\"__galaxyWasmState\")\n")
	sb.WriteString("\tmoduleState := state.Get(__hmrModuleID)\n")
	sb.WriteString("\tif moduleState.IsUndefined() {\n")
	sb.WriteString("\t\treturn js.Undefined()\n")
	sb.WriteString("\t}\n")
	sb.WriteString("\treturn moduleState.Get(key)\n")
	sb.WriteString("}\n\n")

	sb.WriteString("func ensureHMRGlobals() {\n")
	sb.WriteString("\tif js.Global().Get(\"__galaxyWasmModules\").IsUndefined() {\n")
	sb.WriteString("\t\tjs.Global().Set(\"__galaxyWasmModules\", js.Global().Get(\"Object\").New())\n")
	sb.WriteString("\t}\n")
	sb.WriteString("\tif js.Global().Get(\"__galaxyWasmAcceptHandlers\").IsUndefined() {\n")
	sb.WriteString("\t\tjs.Global().Set(\"__galaxyWasmAcceptHandlers\", js.Global().Get(\"Object\").New())\n")
	sb.WriteString("\t}\n")
	sb.WriteString("\tif js.Global().Get(\"__galaxyWasmState\").IsUndefined() {\n")
	sb.WriteString("\t\tjs.Global().Set(\"__galaxyWasmState\", js.Global().Get(\"Object\").New())\n")
	sb.WriteString("\t}\n")
	sb.WriteString("}\n\n")

	sb.WriteString(script)
	return sb.String()
}
