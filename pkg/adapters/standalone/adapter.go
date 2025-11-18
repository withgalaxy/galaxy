package standalone

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/withgalaxy/galaxy/pkg/adapters"
	"github.com/withgalaxy/galaxy/pkg/version"
)

type StandaloneAdapter struct{}

func New() *StandaloneAdapter {
	return &StandaloneAdapter{}
}

func (a *StandaloneAdapter) Name() string {
	return "standalone"
}

func (a *StandaloneAdapter) Build(cfg *adapters.BuildConfig) error {
	if err := a.generateMain(cfg); err != nil {
		return fmt.Errorf("generate main: %w", err)
	}

	if err := a.generateGoMod(cfg); err != nil {
		return fmt.Errorf("generate go.mod: %w", err)
	}

	if err := a.compile(cfg); err != nil {
		return fmt.Errorf("compile: %w", err)
	}

	return nil
}

func (a *StandaloneAdapter) generateMain(cfg *adapters.BuildConfig) error {
	mainPath := filepath.Join(cfg.ServerDir, "main.go")

	if err := a.copyProjectFiles(cfg); err != nil {
		return fmt.Errorf("copy project files: %w", err)
	}

	endpoints := a.buildEndpointData(cfg)
	imports := a.buildEndpointImports(cfg)
	hasMiddleware := a.checkMiddleware(cfg)
	hasSequence := a.checkSequence(cfg)
	hasLifecycle := a.checkLifecycle(cfg)

	tmpl := template.Must(template.New("main").Parse(mainTemplate))

	f, err := os.Create(mainPath)
	if err != nil {
		return err
	}
	defer f.Close()

	routes := []map[string]interface{}{}
	for _, r := range cfg.Routes {
		relPath, _ := filepath.Rel(cfg.PagesDir, r.FilePath)
		routes = append(routes, map[string]interface{}{
			"Pattern":    r.Pattern,
			"RelPath":    "/" + relPath,
			"IsEndpoint": r.IsEndpoint,
		})
	}

	hasSecurity := cfg.Config.Security.CheckOrigin && cfg.Config.IsSSR()

	securityAllowOrigins := []string{}
	if hasSecurity {
		securityAllowOrigins = cfg.Config.Security.AllowOrigins
	}

	hasBodyLimit := cfg.Config.Security.BodyLimit.Enabled
	bodyLimitMaxBytes := cfg.Config.Security.BodyLimit.MaxBytes
	if hasBodyLimit && bodyLimitMaxBytes == 0 {
		bodyLimitMaxBytes = 10 * 1024 * 1024
	}

	hasForwardedHost := len(cfg.Config.Security.AllowedDomains) > 0
	hasHeaders := cfg.Config.Security.Headers.Enabled

	data := map[string]interface{}{
		"Port":                 cfg.Config.Server.Port,
		"Host":                 cfg.Config.Server.Host,
		"SiteURL":              cfg.Config.Site,
		"PublicDir":            filepath.Join(cfg.OutDir, "public"),
		"StaticDir":            cfg.OutDir,
		"PagesDir":             cfg.PagesDir,
		"Routes":               routes,
		"Endpoints":            endpoints,
		"EndpointImports":      imports,
		"HasMiddleware":        hasMiddleware,
		"HasSequence":          hasSequence,
		"HasLifecycle":         hasLifecycle,
		"HasSecurity":          hasSecurity,
		"SecurityCheckOrigin":  cfg.Config.Security.CheckOrigin,
		"SecurityAllowOrigins": securityAllowOrigins,
		"HasBodyLimit":         hasBodyLimit,
		"BodyLimitMaxBytes":    bodyLimitMaxBytes,
		"HasForwardedHost":     hasForwardedHost,
		"AllowedDomains":       cfg.Config.Security.AllowedDomains,
		"HasHeaders":           hasHeaders,
		"HeadersConfig":        cfg.Config.Security.Headers,
	}

	return tmpl.Execute(f, data)
}

func (a *StandaloneAdapter) buildEndpointData(cfg *adapters.BuildConfig) []map[string]interface{} {
	endpoints := []map[string]interface{}{}

	for _, route := range cfg.Routes {
		if !route.IsEndpoint {
			continue
		}

		pkgName := a.getPackageName(route.FilePath)
		methods := a.detectMethods(route.FilePath, pkgName)

		if len(methods) > 0 {
			endpoints = append(endpoints, map[string]interface{}{
				"Pattern": route.Pattern,
				"Methods": methods,
				"Package": pkgName,
			})
		}
	}

	return endpoints
}

func (a *StandaloneAdapter) detectMethods(filePath, pkgName string) []map[string]string {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil
	}

	src := string(content)
	methods := []map[string]string{}

	httpMethods := []string{"GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS", "ALL"}
	for _, method := range httpMethods {
		pattern := fmt.Sprintf("func %s(", method)
		if contains(src, pattern) {
			methods = append(methods, map[string]string{
				"Method":  method,
				"Package": pkgName,
			})
		}
	}

	return methods
}

func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func (a *StandaloneAdapter) buildEndpointImports(cfg *adapters.BuildConfig) []map[string]string {
	imports := []map[string]string{}
	seen := make(map[string]bool)

	for _, route := range cfg.Routes {
		if !route.IsEndpoint {
			continue
		}

		pkgName := a.getPackageName(route.FilePath)
		if seen[pkgName] {
			continue
		}
		seen[pkgName] = true

		relPath, _ := filepath.Rel(cfg.PagesDir, filepath.Dir(route.FilePath))
		importPath := filepath.Join("galaxy-server/pages", relPath)

		imports = append(imports, map[string]string{
			"Alias": pkgName,
			"Path":  importPath,
		})
	}

	return imports
}

func (a *StandaloneAdapter) getPackageName(filePath string) string {
	dir := filepath.Dir(filePath)
	base := filepath.Base(dir)
	if base == "api" || base == "." {
		return "api"
	}
	return base
}

func (a *StandaloneAdapter) checkMiddleware(cfg *adapters.BuildConfig) bool {
	projectDir := filepath.Dir(cfg.PagesDir)
	middlewarePath := filepath.Join(projectDir, "src", "middleware.go")
	_, err := os.Stat(middlewarePath)
	return err == nil
}

func (a *StandaloneAdapter) checkSequence(cfg *adapters.BuildConfig) bool {
	projectDir := filepath.Dir(cfg.PagesDir)
	middlewarePath := filepath.Join(projectDir, "src", "middleware.go")
	content, err := os.ReadFile(middlewarePath)
	if err != nil {
		return false
	}
	return contains(string(content), "func Sequence()")
}

func (a *StandaloneAdapter) checkLifecycle(cfg *adapters.BuildConfig) bool {
	projectDir := filepath.Dir(cfg.PagesDir)
	lifecyclePath := filepath.Join(projectDir, "src", "lifecycle.go")
	_, err := os.Stat(lifecyclePath)
	return err == nil
}

func (a *StandaloneAdapter) copyProjectFiles(cfg *adapters.BuildConfig) error {
	pagesOutDir := filepath.Join(cfg.ServerDir, "pages")
	if err := os.MkdirAll(pagesOutDir, 0755); err != nil {
		return err
	}

	projectDir := filepath.Dir(cfg.PagesDir)

	srcDir := filepath.Join(projectDir, "src")
	if _, err := os.Stat(srcDir); err == nil {
		srcOut := filepath.Join(cfg.ServerDir, "src")
		if err := os.MkdirAll(srcOut, 0755); err != nil {
			return err
		}

		filepath.Walk(srcDir, func(path string, info os.FileInfo, err error) error {
			if err != nil || info.IsDir() {
				return err
			}
			relPath, _ := filepath.Rel(srcDir, path)
			destPath := filepath.Join(srcOut, relPath)
			os.MkdirAll(filepath.Dir(destPath), 0755)
			data, _ := os.ReadFile(path)
			return os.WriteFile(destPath, data, 0644)
		})
	}

	filepath.Walk(projectDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}

		if info.IsDir() {
			dirName := filepath.Base(path)
			if path == projectDir {
				return nil
			}
			if dirName == "src" || dirName == "pages" || dirName == "node_modules" || dirName == ".git" || dirName == "dist" || strings.HasPrefix(dirName, ".") {
				return filepath.SkipDir
			}
			return nil
		}

		if filepath.Ext(path) == ".gxc" {
			relPath, _ := filepath.Rel(projectDir, path)
			destPath := filepath.Join(cfg.ServerDir, relPath)
			os.MkdirAll(filepath.Dir(destPath), 0755)
			data, _ := os.ReadFile(path)
			os.WriteFile(destPath, data, 0644)
		}

		return nil
	})

	return filepath.Walk(cfg.PagesDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}

		relPath, err := filepath.Rel(cfg.PagesDir, path)
		if err != nil {
			return err
		}

		destPath := filepath.Join(pagesOutDir, relPath)

		if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
			return err
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		return os.WriteFile(destPath, data, 0644)
	})
}

func (a *StandaloneAdapter) generateGoMod(cfg *adapters.BuildConfig) error {
	modPath := filepath.Join(cfg.ServerDir, "go.mod")

	galaxyPath, err := a.getGalaxyModulePath()
	hasLocalGalaxy := err == nil && galaxyPath != ""

	// Verify path exists
	if hasLocalGalaxy {
		if _, err := os.Stat(filepath.Join(galaxyPath, "go.mod")); os.IsNotExist(err) {
			hasLocalGalaxy = false
		}
	}

	content := `module galaxy-server

go 1.21

`
	if hasLocalGalaxy {
		content += fmt.Sprintf(`replace github.com/withgalaxy/galaxy => %s

require github.com/withgalaxy/galaxy v0.0.0
`, galaxyPath)
	} else {
		v := version.Version
		if v[0] != 'v' {
			v = "v" + v
		}
		content += fmt.Sprintf("require github.com/withgalaxy/galaxy %s\n", v)
	}

	return os.WriteFile(modPath, []byte(content), 0644)
}

func (a *StandaloneAdapter) getGalaxyModulePath() (string, error) {
	cmd := exec.Command("go", "list", "-m", "-f", "{{.Dir}}", "github.com/withgalaxy/galaxy")
	out, err := cmd.Output()
	if err != nil {
		wd, _ := os.Getwd()
		for wd != "/" {
			modPath := filepath.Join(wd, "go.mod")
			if _, err := os.Stat(modPath); err == nil {
				return wd, nil
			}
			wd = filepath.Dir(wd)
		}
		return "", fmt.Errorf("cannot find galaxy module")
	}
	return string(out), nil
}

func (a *StandaloneAdapter) compile(cfg *adapters.BuildConfig) error {
	binaryPath := filepath.Join(cfg.ServerDir, "server")

	tidyCmd := exec.Command("go", "mod", "tidy")
	tidyCmd.Dir = cfg.ServerDir
	tidyCmd.Stdout = os.Stdout
	tidyCmd.Stderr = os.Stderr
	if err := tidyCmd.Run(); err != nil {
		return fmt.Errorf("go mod tidy: %w", err)
	}

	cmd := exec.Command("go", "build", "-o", binaryPath, ".")
	cmd.Dir = cfg.ServerDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

const mainTemplate = `package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	{{if .HasLifecycle}}
	"os/signal"
	{{end}}
	"path/filepath"
	"strings"
	{{if .HasLifecycle}}
	"syscall"
	{{end}}

	"github.com/withgalaxy/galaxy/pkg/compiler"
	{{if or .HasBodyLimit .HasForwardedHost .HasHeaders}}
	"github.com/withgalaxy/galaxy/pkg/config"
	{{end}}
	"github.com/withgalaxy/galaxy/pkg/endpoints"
	"github.com/withgalaxy/galaxy/pkg/executor"
	{{if .HasLifecycle}}
	"github.com/withgalaxy/galaxy/pkg/lifecycle"
	{{end}}
	"github.com/withgalaxy/galaxy/pkg/middleware"
	"github.com/withgalaxy/galaxy/pkg/parser"
	"github.com/withgalaxy/galaxy/pkg/router"
	{{if or .HasSecurity .HasBodyLimit .HasForwardedHost .HasHeaders}}
	"github.com/withgalaxy/galaxy/pkg/security"
	{{end}}
	"github.com/withgalaxy/galaxy/pkg/ssr"
	"github.com/withgalaxy/galaxy/pkg/template"
	"github.com/withgalaxy/galaxy/pkg/wasm"

	{{range .EndpointImports}}
	{{.Alias}} "{{.Path}}"
	{{end}}
	{{if .HasMiddleware}}
	usermw "galaxy-server/src"
	{{end}}
	{{if .HasLifecycle}}
	userlc "galaxy-server/src"
	{{end}}
)

var (
	rt                     *router.Router
	comp                   *compiler.ComponentCompiler
	baseDir                string
	pagesDir               = "pages"
	wasmManifest           *wasm.WasmManifest
	{{if .HasBodyLimit}}
	bodyLimitMiddleware    *security.BodyLimitMiddleware
	{{end}}
	{{if .HasForwardedHost}}
	forwardedHostValidator *security.ForwardedHostValidator
	{{end}}
	{{if .HasSecurity}}
	csrfMiddleware         *security.CSRFMiddleware
	{{end}}
	{{if .HasHeaders}}
	headersMiddleware      *security.HeadersMiddleware
	{{end}}
	endpointHandlers = map[string]map[string]endpoints.HandlerFunc{
		{{range .Endpoints}}
		"{{.Pattern}}": {
			{{range .Methods}}
			"{{.Method}}": {{.Package}}.{{.Method}},
			{{end}}
		},
		{{end}}
	}
)

func main() {
	exePath, err := os.Executable()
	if err != nil {
		log.Fatal(err)
	}
	baseDir = filepath.Dir(exePath)
	comp = compiler.NewComponentCompiler(baseDir)

	rt = router.NewRouter(filepath.Join(baseDir, pagesDir))
	if err := rt.Discover(); err != nil {
		log.Fatalf("Route discovery failed: %v", err)
	}
	rt.Sort()

	manifestPath := filepath.Join(baseDir, "_assets", "wasm-manifest.json")
	wasmManifest, _ = wasm.LoadManifest(manifestPath)

	{{if .HasBodyLimit}}
	bodyLimitMiddleware = security.NewBodyLimitMiddleware({{.BodyLimitMaxBytes}})
	{{end}}

	{{if .HasForwardedHost}}
	allowedDomains := []config.RemotePattern{
		{{range .AllowedDomains}}
		{
			Protocol: "{{.Protocol}}",
			Hostname: "{{.Hostname}}",
			{{if .Port}}Port: func() *int { p := {{.Port}}; return &p }(),{{end}}
		},
		{{end}}
	}
	forwardedHostValidator = security.NewForwardedHostValidator(allowedDomains)
	{{end}}

	{{if .HasSecurity}}
	csrfMiddleware = security.NewCSRFMiddleware(&security.CSRFConfig{
		CheckOrigin:  {{.SecurityCheckOrigin}},
		AllowOrigins: []string{ {{range .SecurityAllowOrigins}}"{{.}}",{{end}} },
		SiteURL:      "{{.SiteURL}}",
	})
	{{end}}

	{{if .HasHeaders}}
	headersMiddleware = security.NewHeadersMiddleware(config.HeadersConfig{
		Enabled: {{.HeadersConfig.Enabled}},
		XFrameOptions: "{{.HeadersConfig.XFrameOptions}}",
		XContentTypeOptions: "{{.HeadersConfig.XContentTypeOptions}}",
		XXSSProtection: "{{.HeadersConfig.XXSSProtection}}",
		ReferrerPolicy: "{{.HeadersConfig.ReferrerPolicy}}",
		StrictTransportSecurity: "{{.HeadersConfig.StrictTransportSecurity}}",
		ContentSecurityPolicy: "{{.HeadersConfig.ContentSecurityPolicy}}",
		PermissionsPolicy: "{{.HeadersConfig.PermissionsPolicy}}",
	})
	{{end}}

	{{if .HasLifecycle}}
	lc := lifecycle.NewLifecycle()
	lc.Register(userlc.Lifecycle())
	
	if err := lc.ExecuteStartup(); err != nil {
		log.Fatalf("Startup failed: %v", err)
	}
	
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigChan
		log.Println("Shutting down gracefully...")
		if err := lc.ExecuteShutdown(); err != nil {
			log.Printf("Shutdown error: %v", err)
		}
		os.Exit(0)
	}()
	{{end}}

	http.HandleFunc("/", handleRequest)

	addr := "{{.Host}}:{{.Port}}"
	log.Printf("🚀 Server running at http://%s\n", addr)
	
	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatal(err)
	}
}

func handleRequest(w http.ResponseWriter, r *http.Request) {
	if strings.HasPrefix(r.URL.Path, "/_assets/") {
		assetsPath := filepath.Join(baseDir, r.URL.Path)
		http.ServeFile(w, r, assetsPath)
		return
	}

	if r.URL.Path == "/wasm_exec.js" {
		wasmExecPath := filepath.Join(baseDir, "wasm_exec.js")
		http.ServeFile(w, r, wasmExecPath)
		return
	}

	if filepath.Ext(r.URL.Path) != "" {
		http.ServeFile(w, r, filepath.Join("{{.PublicDir}}", r.URL.Path))
		return
	}

	staticPath := filepath.Join("{{.StaticDir}}", r.URL.Path)
	if r.URL.Path == "/" {
		staticPath = filepath.Join("{{.StaticDir}}", "index.html")
	} else {
		staticPath = filepath.Join("{{.StaticDir}}", r.URL.Path, "index.html")
	}
	
	if _, err := os.Stat(staticPath); err == nil {
		http.ServeFile(w, r, staticPath)
		return
	}

	route, params := rt.Match(r.URL.Path)
	if route == nil {
		http.NotFound(w, r)
		return
	}

	mwCtx := middleware.NewContext(w, r)
	mwCtx.Params = params

	{{if .HasBodyLimit}}
	if err := bodyLimitMiddleware.Middleware(mwCtx, func() error { return nil }); err != nil {
		return
	}
	{{end}}

	{{if .HasForwardedHost}}
	currentURL := mwCtx.Request.URL
	if currentURL.Scheme == "" {
		currentURL.Scheme = "http"
	}
	if currentURL.Host == "" {
		currentURL.Host = mwCtx.Request.Host
	}
	validatedURL := forwardedHostValidator.ValidateForwardedHost(mwCtx.Request, currentURL)
	mwCtx.Request.URL = validatedURL
	{{end}}

	{{if .HasSecurity}}
	if err := csrfMiddleware.Middleware(mwCtx, func() error { return nil }); err != nil {
		return
	}
	{{end}}

	{{if .HasHeaders}}
	if err := headersMiddleware.Middleware(mwCtx, func() error { return nil }); err != nil {
		return
	}
	{{end}}

	{{if .HasMiddleware}}
	{{if .HasSequence}}
	chain := middleware.NewChain()
	for _, mw := range usermw.Sequence() {
		chain.Use(mw)
	}
	if err := chain.Execute(mwCtx, func(ctx *middleware.Context) error {
		if route.IsEndpoint {
			handleEndpoint(route.Pattern, mwCtx)
		} else {
			handlePage(route.FilePath, mwCtx)
		}
		return nil
	}); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	{{else}}
	if err := usermw.OnRequest(mwCtx, func() error {
		if route.IsEndpoint {
			handleEndpoint(route.Pattern, mwCtx)
		} else {
			handlePage(route.FilePath, mwCtx)
		}
		return nil
	}); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	{{end}}
	{{else}}
	if route.IsEndpoint {
		handleEndpoint(route.Pattern, mwCtx)
		return
	}
	handlePage(route.FilePath, mwCtx)
	{{end}}
}

func handleEndpoint(pattern string, mwCtx *middleware.Context) {
	ep, ok := endpointHandlers[pattern]
	if !ok {
		http.Error(mwCtx.Response, "Endpoint not found", http.StatusNotFound)
		return
	}

	method := mwCtx.Request.Method
	handler, ok := ep[method]
	if !ok {
		handler, ok = ep["ALL"]
		if !ok {
			http.Error(mwCtx.Response, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
	}

	ctx := endpoints.NewContext(mwCtx.Response, mwCtx.Request, mwCtx.Params, mwCtx.Locals)
	if err := handler(ctx); err != nil {
		http.Error(mwCtx.Response, err.Error(), http.StatusInternalServerError)
	}
}

func handlePage(filePath string, mwCtx *middleware.Context) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		http.Error(mwCtx.Response, err.Error(), http.StatusInternalServerError)
		return
	}

	parsed, err := parser.Parse(string(content))
	if err != nil {
		http.Error(mwCtx.Response, fmt.Sprintf("Parse error: %v", err), http.StatusInternalServerError)
		return
	}

	resolver := comp.Resolver
	resolver.SetCurrentFile(filePath)

	imports := make([]compiler.Import, len(parsed.Imports))
	for i, imp := range parsed.Imports {
		imports[i] = compiler.Import{
			Path:        imp.Path,
			Alias:       imp.Alias,
			IsComponent: imp.IsComponent,
		}
	}
	resolver.ParseImports(imports)

	ctx := executor.NewContext()

	reqCtx := ssr.NewRequestContext(mwCtx.Request, mwCtx.Params)
	ctx.SetRequest(reqCtx)
	ctx.SetLocals(mwCtx.Locals)

	ctx.SetParams(mwCtx.Params)

	for k, v := range mwCtx.Params {
		ctx.Set(k, v)
	}

	if parsed.Frontmatter != "" {
		if err := ctx.Execute(parsed.Frontmatter); err != nil {
			http.Error(mwCtx.Response, fmt.Sprintf("Execution error: %v", err), http.StatusInternalServerError)
			return
		}
	}

	if ctx.ShouldRedirect {
		http.Redirect(mwCtx.Response, mwCtx.Request, ctx.RedirectURL, ctx.RedirectStatus)
		return
	}

	comp.CollectedStyles = nil
	processedTemplate := comp.ProcessComponentTags(parsed.Template, ctx)

	engine := template.NewEngine(ctx)
	rendered, err := engine.Render(processedTemplate, nil)
	if err != nil {
		http.Error(mwCtx.Response, fmt.Sprintf("Render error: %v", err), http.StatusInternalServerError)
		return
	}

	allStyles := append(parsed.Styles, comp.CollectedStyles...)
	if len(allStyles) > 0 {
		var styleContent string
		for _, style := range allStyles {
			styleContent += style.Content + "\n"
		}
		styleTag := "<style>" + styleContent + "</style>"
		rendered = strings.Replace(rendered, "</head>", styleTag+"\n</head>", 1)
	}

	if wasmManifest != nil {
		pageAssets, ok := wasmManifest.Assets[filePath]
		if ok && len(pageAssets.WasmModules) > 0 {
			wasmExecTag := "<script src=\"/wasm_exec.js\"></script>"
			rendered = strings.Replace(rendered, "</body>", wasmExecTag+"\n</body>", 1)

			for _, mod := range pageAssets.WasmModules {
				loaderTag := fmt.Sprintf("<script src=\"%s\"></script>", mod.LoaderPath)
				rendered = strings.Replace(rendered, "</body>", loaderTag+"\n</body>", 1)
			}
		}

		if len(pageAssets.JSScripts) > 0 {
			for _, jsPath := range pageAssets.JSScripts {
				jsTag := fmt.Sprintf("<script type=\"module\" src=\"%s\"></script>", jsPath)
				rendered = strings.Replace(rendered, "</body>", jsTag+"\n</body>", 1)
			}
		}
	}

	mwCtx.Response.Header().Set("Content-Type", "text/html; charset=utf-8")
	mwCtx.Response.Write([]byte(rendered))
}
`
