package server

import (
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"plugin"
	"strings"

	"github.com/withgalaxy/galaxy/pkg/codegen"
	"github.com/withgalaxy/galaxy/pkg/parser"
	"github.com/withgalaxy/galaxy/pkg/router"
	"github.com/withgalaxy/galaxy/pkg/version"
)

type PluginCompiler struct {
	CacheDir    string
	ModuleName  string
	GalaxyPath  string
	ProjectRoot string
}

func NewPluginCompiler(cacheDir, moduleName, galaxyPath, projectRoot string) *PluginCompiler {
	return &PluginCompiler{
		CacheDir:    cacheDir,
		ModuleName:  moduleName,
		GalaxyPath:  galaxyPath,
		ProjectRoot: projectRoot,
	}
}

func (pc *PluginCompiler) CompilePage(route *router.Route, comp *parser.Component, fmHash string) (*PagePlugin, error) {
	tmplHash := HashContent(comp.Template)

	// Use hash in plugin name to ensure unique builds for different code
	// This prevents "plugin already loaded" errors
	pluginName := fmt.Sprintf("%s-%s", sanitizeRouteName(route.Pattern), fmHash[:16])
	pluginDir, _ := filepath.Abs(filepath.Join(pc.CacheDir, "plugins", pluginName))
	soPath := filepath.Join(pluginDir, "handler.so")

	// Check if plugin already exists on disk (from previous dev session)
	if _, err := os.Stat(soPath); err == nil {
		// Plugin exists, try to load it
		fmt.Printf("📦 Loading cached plugin from disk: %s\n", pluginName)
		p, err := plugin.Open(soPath)
		if err == nil {
			sym, err := p.Lookup("Handler")
			if err == nil {
				if handlerFunc, ok := sym.(*func(http.ResponseWriter, *http.Request, map[string]string, map[string]interface{})); ok {
					return &PagePlugin{
						Handler:         *handlerFunc,
						Template:        comp.Template,
						FrontmatterHash: fmHash,
						TemplateHash:    tmplHash,
						PluginPath:      soPath,
					}, nil
				}
			}
		}
		// If loading failed, fall through to recompile
		fmt.Printf("⚠️ Failed to load cached plugin, recompiling: %v\n", err)
	}

	if err := os.MkdirAll(pluginDir, 0755); err != nil {
		return nil, fmt.Errorf("create plugin dir: %w", err)
	}

	fmt.Printf("🔨 Compiling plugin: %s\n", pluginName)

	gen := codegen.NewHandlerGenerator(comp, route, pc.ModuleName, filepath.Dir(route.FilePath))
	handler, err := gen.Generate()
	if err != nil {
		return nil, fmt.Errorf("generate handler: %w", err)
	}

	// Filter out imports that are already hardcoded
	hardcodedImports := map[string]bool{
		`"fmt"`:      true,
		`"net/http"`: true,
		`"reflect"`:  true,
		`"regexp"`:   true,
		`"strings"`:  true,
	}
	var filteredImports []string
	for _, imp := range handler.Imports {
		if !hardcodedImports[imp] {
			filteredImports = append(filteredImports, imp)
		}
	}

	handlerCode := fmt.Sprintf(`package main

import (
	"fmt"
	"net/http"
	"github.com/withgalaxy/galaxy/pkg/executor"
	"github.com/withgalaxy/galaxy/pkg/template"
	%s
)

var Handler func(w http.ResponseWriter, r *http.Request, params map[string]string, locals map[string]interface{})

func init() {
	Handler = %s
}

%s
`, joinImports(filteredImports), handler.FunctionName, handler.Code)

	handlerPath := filepath.Join(pluginDir, "handler.go")
	if err := os.WriteFile(handlerPath, []byte(handlerCode), 0644); err != nil {
		return nil, fmt.Errorf("write handler: %w", err)
	}

	// Use unique module name per plugin to prevent "plugin already loaded" errors
	uniqueModuleName := fmt.Sprintf("%s/%s", pc.ModuleName, pluginName)

	// Check if project has a go.mod to inherit dependencies
	projectGoMod := filepath.Join(pc.ProjectRoot, "go.mod")
	projectModule := ""
	projectPath := ""

	if content, err := os.ReadFile(projectGoMod); err == nil {
		// Extract module name from first line: "module github.com/foo/bar"
		lines := strings.Split(string(content), "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "module ") {
				projectModule = strings.TrimSpace(strings.TrimPrefix(line, "module"))
				projectPath, _ = filepath.Abs(pc.ProjectRoot)
				break
			}
		}
	}

	// Check if GalaxyPath is valid
	hasLocalGalaxy := pc.GalaxyPath != ""
	if hasLocalGalaxy {
		if _, err := os.Stat(filepath.Join(pc.GalaxyPath, "go.mod")); os.IsNotExist(err) {
			hasLocalGalaxy = false
		}
	}

	goMod := fmt.Sprintf(`module %s

go 1.23

`, uniqueModuleName)

	if hasLocalGalaxy {
		goMod += fmt.Sprintf(`replace github.com/withgalaxy/galaxy => %s

`, pc.GalaxyPath)
	}

	// Add project module replace if it exists
	if projectModule != "" && projectPath != "" {
		goMod += fmt.Sprintf("replace %s => %s\n\n", projectModule, projectPath)
	}

	if hasLocalGalaxy {
		goMod += "require github.com/withgalaxy/galaxy v0.0.0\n"
	} else {
		v := version.Version
		if v[0] != 'v' {
			v = "v" + v
		}
		goMod += fmt.Sprintf("require github.com/withgalaxy/galaxy %s\n", v)
	}

	// Require project module if it exists
	if projectModule != "" {
		goMod += fmt.Sprintf("require %s v0.0.0\n", projectModule)
	}

	if err := os.WriteFile(filepath.Join(pluginDir, "go.mod"), []byte(goMod), 0644); err != nil {
		return nil, fmt.Errorf("write go.mod: %w", err)
	}

	tidyCmd := exec.Command("go", "mod", "tidy")
	tidyCmd.Dir = pluginDir
	if output, err := tidyCmd.CombinedOutput(); err != nil {
		return nil, fmt.Errorf("go mod tidy: %w\n%s", err, output)
	}

	buildCmd := exec.Command("go", "build", "-buildmode=plugin", "-o", soPath, ".")
	buildCmd.Dir = pluginDir
	if output, err := buildCmd.CombinedOutput(); err != nil {
		return nil, fmt.Errorf("build plugin: %w\n%s", err, output)
	}

	p, err := plugin.Open(soPath)
	if err != nil {
		return nil, fmt.Errorf("load plugin: %w", err)
	}

	sym, err := p.Lookup("Handler")
	if err != nil {
		return nil, fmt.Errorf("lookup Handler: %w", err)
	}

	handlerFunc, ok := sym.(*func(http.ResponseWriter, *http.Request, map[string]string, map[string]interface{}))
	if !ok {
		return nil, fmt.Errorf("invalid handler type: %T", sym)
	}

	return &PagePlugin{
		Handler:         *handlerFunc,
		Template:        comp.Template,
		FrontmatterHash: fmHash,
		TemplateHash:    tmplHash,
		PluginPath:      soPath,
	}, nil
}

func sanitizeRouteName(pattern string) string {
	name := pattern
	replacements := map[string]string{
		"/": "_", "{": "", "}": "", "[": "", "]": "",
		".": "", "-": "_", ":": "",
	}
	for old, new := range replacements {
		name = strings.ReplaceAll(name, old, new)
	}
	if name == "" || name == "_" {
		name = "index"
	}
	return strings.Trim(name, "_")
}

func joinImports(imports []string) string {
	if len(imports) == 0 {
		return ""
	}
	var lines []string
	for _, imp := range imports {
		lines = append(lines, "\t"+imp)
	}
	return strings.Join(lines, "\n")
}
