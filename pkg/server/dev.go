package server

import (
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"encoding/json"
	"github.com/cameron-webmatter/galaxy/internal/assets"
	"github.com/cameron-webmatter/galaxy/pkg/codegen"
	"github.com/cameron-webmatter/galaxy/pkg/compiler"
	"github.com/cameron-webmatter/galaxy/pkg/endpoints"
	"github.com/cameron-webmatter/galaxy/pkg/executor"
	"github.com/cameron-webmatter/galaxy/pkg/lifecycle"
	"github.com/cameron-webmatter/galaxy/pkg/middleware"
	"github.com/cameron-webmatter/galaxy/pkg/parser"
	"github.com/cameron-webmatter/galaxy/pkg/router"
	"github.com/cameron-webmatter/galaxy/pkg/ssr"
	"github.com/cameron-webmatter/galaxy/pkg/template"

	"github.com/cameron-webmatter/galaxy/pkg/hmr"
)

type DevServer struct {
	Router             *router.Router
	RootDir            string
	PagesDir           string
	PublicDir          string
	Port               int
	Bundler            *assets.Bundler
	Compiler           *compiler.ComponentCompiler
	EndpointCompiler   *endpoints.EndpointCompiler
	Verbose            bool
	Lifecycle          *lifecycle.Lifecycle
	MiddlewareCompiler *middleware.MiddlewareCompiler
	LoadedMiddleware   *middleware.LoadedMiddleware
	MiddlewareChain    *middleware.Chain
	HasMiddleware      bool
	UseCodegen         bool
	PageCache          *PageCache
	PluginCompiler     *PluginCompiler
	compileMu          sync.Mutex
	codegenServerCmd   *exec.Cmd
	HMRServer          *hmr.Server
	ChangeTracker      *hmr.ChangeTracker
	ComponentTracker   *hmr.ComponentTracker

	codegenServerPort int
	codegenReady      bool
}

func NewDevServer(rootDir, pagesDir, publicDir string, port int, verbose bool) *DevServer {
	srcDir := filepath.Dir(pagesDir)

	useCodegen := os.Getenv("GALAXY_USE_CODEGEN") != "false"

	galaxyPath := "../../../galaxy"
	if gp := os.Getenv("GALAXY_PATH"); gp != "" {
		galaxyPath = gp
	}

	// Convert to absolute path for plugin go.mod replace directive
	if absPath, err := filepath.Abs(galaxyPath); err == nil {
		galaxyPath = absPath
	}

	srv := &DevServer{
		Router:             router.NewRouter(pagesDir),
		RootDir:            rootDir,
		PagesDir:           pagesDir,
		PublicDir:          publicDir,
		Port:               port,
		Bundler:            assets.NewBundler(".galaxy"),
		Compiler:           compiler.NewComponentCompiler(srcDir),
		EndpointCompiler:   endpoints.NewCompiler(rootDir, ".galaxy/endpoints"),
		MiddlewareCompiler: middleware.NewCompiler(rootDir, ".galaxy/middleware"),
		Verbose:            verbose,

		UseCodegen:     useCodegen,
		PageCache:      NewPageCache(),
		PluginCompiler: NewPluginCompiler(".galaxy", "dev-server", galaxyPath, rootDir),
	}

	srv.ChangeTracker = hmr.NewChangeTracker()
	srv.ComponentTracker = hmr.NewComponentTracker()

	srv.Bundler.DevMode = true

	return srv
}

func (s *DevServer) Start() error {
	if err := s.Router.Discover(); err != nil {
		return err
	}
	s.Router.Sort()

	if s.Lifecycle != nil {
		if err := s.Lifecycle.ExecuteStartup(); err != nil {
			return fmt.Errorf("lifecycle startup: %w", err)
		}
	}

	// If codegen mode is enabled, build and start the codegen server
	if s.UseCodegen {
		if err := s.buildAndStartCodegenServer(); err != nil {
			return fmt.Errorf("start codegen server: %w", err)
		}
	}

	s.HMRServer = hmr.NewServer()
	s.HMRServer.Start()
	http.HandleFunc("/__hmr", s.HMRServer.HandleWebSocket)
	http.HandleFunc("/__hmr/client.js", s.serveHMRClient)
	http.HandleFunc("/__hmr/morph.js", s.serveHMRMorph)
	http.HandleFunc("/__hmr/overlay.js", s.serveHMROverlay)
	http.HandleFunc("/__hmr/render", s.handleHMRRender)

	http.HandleFunc("/", s.logRequest(s.handleRequest))

	addr := fmt.Sprintf(":%d", s.Port)
	fmt.Printf("🚀 Dev server running at http://localhost%s\n", addr)
	fmt.Printf("📁 Pages: %s\n", s.PagesDir)
	fmt.Printf("📦 Public: %s\n\n", s.PublicDir)

	s.printRoutes()

	return http.ListenAndServe(addr, nil)
}

func (s *DevServer) ReloadRoutes() error {
	if err := s.Router.Reload(); err != nil {
		return err
	}

	fmt.Println("\n🔄 Routes reloaded:")
	s.printRoutes()

	return nil
}

func (s *DevServer) ReloadMiddleware() error {
	srcDir := filepath.Dir(s.PagesDir)
	middlewarePath := filepath.Join(srcDir, "middleware.go")

	loaded, err := s.MiddlewareCompiler.Load(middlewarePath)
	if err != nil {
		return err
	}

	s.LoadedMiddleware = loaded
	s.MiddlewareChain = middleware.NewChain()

	if loaded.Sequence != nil && len(loaded.Sequence) > 0 {
		for _, mw := range loaded.Sequence {
			s.MiddlewareChain.Use(mw)
		}
	} else if loaded.OnRequest != nil {
		s.MiddlewareChain.Use(loaded.OnRequest)
	}

	s.HasMiddleware = true
	return nil
}

func (s *DevServer) printRoutes() {
	for _, route := range s.Router.Routes {
		fmt.Printf("  %s\n", route.Pattern)
	}
	fmt.Println()
}

type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

func (s *DevServer) logRequest(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rw := &responseWriter{ResponseWriter: w, statusCode: 200}

		next(rw, r)

		if s.Verbose {
			duration := time.Since(start)
			statusColor := getStatusColor(rw.statusCode)
			methodColor := "\033[36m"
			reset := "\033[0m"

			fmt.Printf("%s%s%s %s - %s%d%s (%dms)\n",
				methodColor, r.Method, reset,
				r.URL.Path,
				statusColor, rw.statusCode, reset,
				duration.Milliseconds())
		}
	}
}

func getStatusColor(status int) string {
	switch {
	case status >= 500:
		return "\033[31m"
	case status >= 400:
		return "\033[33m"
	case status >= 300:
		return "\033[36m"
	case status >= 200:
		return "\033[32m"
	default:
		return "\033[0m"
	}
}

func (s *DevServer) handleRequest(w http.ResponseWriter, r *http.Request) {
	// If codegen server is ready, proxy all requests to it
	if s.UseCodegen && s.codegenReady {
		s.proxyToCodegenServer(w, r)
		return
	}

	if r.URL.Path == "/wasm_exec.js" {
		s.serveWasmExec(w, r)
		return
	}

	if filepath.Ext(r.URL.Path) != "" {
		s.serveStatic(w, r)
		return
	}

	route, params := s.Router.Match(r.URL.Path)
	if route == nil {
		http.NotFound(w, r)
		return
	}

	mwCtx := middleware.NewContext(w, r)
	mwCtx.Params = params

	if s.HasMiddleware && s.MiddlewareChain != nil {
		err := s.MiddlewareChain.Execute(mwCtx, func(ctx *middleware.Context) error {
			if route.IsEndpoint {
				s.handleEndpoint(route, ctx, params)
			} else {
				s.handlePage(route, ctx, params)
			}
			return nil
		})
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}

	s.handlePage(route, mwCtx, params)
}

func (s *DevServer) handleEndpoint(route *router.Route, mwCtx *middleware.Context, params map[string]string) {
	endpoint, err := s.EndpointCompiler.Load(route.FilePath)
	if err != nil {
		http.Error(mwCtx.Response, fmt.Sprintf("Load endpoint: %v", err), http.StatusInternalServerError)
		return
	}

	if err := endpoints.HandleEndpoint(endpoint, mwCtx.Response, mwCtx.Request, params, mwCtx.Locals); err != nil {
		http.Error(mwCtx.Response, err.Error(), http.StatusInternalServerError)
	}
}

func (s *DevServer) handlePage(route *router.Route, mwCtx *middleware.Context, params map[string]string) {
	if s.UseCodegen {
		s.handlePageWithCodegen(route, mwCtx, params)
		return
	}

	content, err := os.ReadFile(route.FilePath)
	if err != nil {
		http.Error(mwCtx.Response, err.Error(), http.StatusInternalServerError)
		return
	}

	comp, err := parser.Parse(string(content))
	if err != nil {
		http.Error(mwCtx.Response, fmt.Sprintf("Parse error: %v", err), http.StatusInternalServerError)
		return
	}

	resolver := s.Compiler.Resolver
	resolver.SetCurrentFile(route.FilePath)

	imports := make([]compiler.Import, len(comp.Imports))
	for i, imp := range comp.Imports {
		imports[i] = compiler.Import{
			Path:        imp.Path,
			Alias:       imp.Alias,
			IsComponent: imp.IsComponent,
		}
	}
	resolver.ParseImports(imports)

	ctx := executor.NewContext()

	// Register common Go functions for frontmatter debugging
	ctx.RegisterPackageFunc("fmt", "Printf", func(args ...interface{}) (interface{}, error) {
		if len(args) > 0 {
			format, ok := args[0].(string)
			if ok {
				fmt.Printf(format, args[1:]...)
			}
		}
		return nil, nil
	})
	ctx.RegisterPackageFunc("fmt", "Println", func(args ...interface{}) (interface{}, error) {
		fmt.Println(args...)
		return nil, nil
	})
	ctx.RegisterPackageFunc("log", "Printf", func(args ...interface{}) (interface{}, error) {
		if len(args) > 0 {
			format, ok := args[0].(string)
			if ok {
				log.Printf(format, args[1:]...)
			}
		}
		return nil, nil
	})
	ctx.RegisterPackageFunc("log", "Println", func(args ...interface{}) (interface{}, error) {
		log.Println(args...)
		return nil, nil
	})

	reqCtx := ssr.NewRequestContext(mwCtx.Request, params)
	ctx.SetRequest(reqCtx)
	ctx.SetLocals(mwCtx.Locals)

	ctx.SetParams(params)

	for k, v := range params {
		ctx.Set(k, v)
	}

	if comp.Frontmatter != "" {
		if err := ctx.Execute(comp.Frontmatter); err != nil {
			http.Error(mwCtx.Response, fmt.Sprintf("Execution error: %v", err), http.StatusInternalServerError)
			return
		}
	}

	if ctx.ShouldRedirect {
		http.Redirect(mwCtx.Response, mwCtx.Request, ctx.RedirectURL, ctx.RedirectStatus)
		return
	}

	s.Compiler.CollectedStyles = nil
	s.Compiler.ResetComponentTracking()
	processedTemplate := s.Compiler.ProcessComponentTags(comp.Template, ctx)

	if s.ComponentTracker != nil && len(s.Compiler.UsedComponents) > 0 {
		s.ComponentTracker.TrackPageComponents(route.FilePath, s.Compiler.UsedComponents)
	}

	engine := template.NewEngine(ctx)
	rendered, err := engine.Render(processedTemplate, nil)
	if err != nil {
		http.Error(mwCtx.Response, fmt.Sprintf("Render error: %v", err), http.StatusInternalServerError)
		return
	}

	allStyles := append(comp.Styles, s.Compiler.CollectedStyles...)
	compWithStyles := &parser.Component{
		Frontmatter: comp.Frontmatter,
		Template:    comp.Template,
		Scripts:     comp.Scripts,
		Styles:      allStyles,
		Imports:     comp.Imports,
	}

	cssPath, err := s.Bundler.BundleStyles(compWithStyles, route.FilePath)
	if err != nil {
		http.Error(mwCtx.Response, fmt.Sprintf("Style bundle error: %v", err), http.StatusInternalServerError)
		return
	}

	jsPath, err := s.Bundler.BundleScripts(comp, route.FilePath)
	if err != nil {
		http.Error(mwCtx.Response, fmt.Sprintf("Script bundle error: %v", err), http.StatusInternalServerError)
		return
	}

	wasmAssets, err := s.Bundler.BundleWasmScripts(comp, route.FilePath)
	if err != nil {
		http.Error(mwCtx.Response, fmt.Sprintf("WASM bundle error: %v", err), http.StatusInternalServerError)
		return
	}

	scopeID := ""
	for _, style := range allStyles {
		if style.Scoped {
			scopeID = s.Bundler.GenerateScopeID(route.FilePath)
			break
		}
	}

	rendered = s.Bundler.InjectAssetsWithWasm(rendered, cssPath, jsPath, scopeID, wasmAssets)

	mwCtx.Response.Header().Set("Content-Type", "text/html; charset=utf-8")
	mwCtx.Response.Write([]byte(rendered))
}

func (s *DevServer) serveStatic(w http.ResponseWriter, r *http.Request) {
	galaxyPath := filepath.Join(".galaxy", r.URL.Path)
	if _, err := os.Stat(galaxyPath); err == nil {
		http.ServeFile(w, r, galaxyPath)
		return
	}

	publicPath := filepath.Join(s.PublicDir, r.URL.Path)
	http.ServeFile(w, r, publicPath)
}

func (s *DevServer) serveWasmExec(w http.ResponseWriter, r *http.Request) {
	goRoot := os.Getenv("GOROOT")
	if goRoot == "" {
		cmd := exec.Command("go", "env", "GOROOT")
		output, _ := cmd.Output()
		goRoot = strings.TrimSpace(string(output))
	}

	wasmExecPath := filepath.Join(goRoot, "misc", "wasm", "wasm_exec.js")
	if _, err := os.Stat(wasmExecPath); os.IsNotExist(err) {
		wasmExecPath = filepath.Join(goRoot, "lib", "wasm", "wasm_exec.js")
	}

	http.ServeFile(w, r, wasmExecPath)
}

func (s *DevServer) buildAndStartCodegenServer() error {
	fmt.Println("🔨 Building codegen server...")

	// Use a different port for the codegen server
	s.codegenServerPort = s.Port + 1000

	// Build the server using CodegenBuilder
	builder := codegen.NewCodegenBuilder(s.Router.Routes, s.PagesDir, "dist", "dev-server", s.PublicDir)
	if err := builder.Build(); err != nil {
		return fmt.Errorf("codegen build failed: %w", err)
	}

	fmt.Println("✅ Codegen server built successfully")
	fmt.Println("🚀 Starting codegen server...")

	// Start the compiled server from dist/server directory
	// so it can find _assets, wasm_exec.js, etc.
	serverBinary := "./server"
	cmd := exec.Command(serverBinary)
	cmd.Dir = filepath.Join(s.RootDir, "dist", "server")

	// Load .env and pass to codegen server
	envVars := os.Environ()

	envPath := filepath.Join(s.RootDir, ".env")
	if data, err := os.ReadFile(envPath); err == nil {
		lines := strings.Split(string(data), "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line != "" && !strings.HasPrefix(line, "#") && strings.Contains(line, "=") {
				// Skip PORT from .env - we'll set it ourselves
				if !strings.HasPrefix(line, "PORT=") {
					envVars = append(envVars, line)
				}
			}
		}
	}

	// Set PORT after loading .env so it doesn't get overridden
	envVars = append(envVars, fmt.Sprintf("PORT=%d", s.codegenServerPort))

	cmd.Env = envVars
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start codegen server: %w", err)
	}

	s.codegenServerCmd = cmd

	// Wait for server to be ready
	serverURL := fmt.Sprintf("http://localhost:%d", s.codegenServerPort)
	for i := 0; i < 50; i++ {
		resp, err := http.Get(serverURL + "/health")
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode < 500 {
				s.codegenReady = true
				fmt.Printf("✅ Codegen server ready on port %d\n", s.codegenServerPort)
				return nil
			}
		}
		time.Sleep(100 * time.Millisecond)
	}

	return fmt.Errorf("codegen server did not become ready")
}

func (s *DevServer) proxyToCodegenServer(w http.ResponseWriter, r *http.Request) {
	target, _ := url.Parse(fmt.Sprintf("http://localhost:%d", s.codegenServerPort))
	proxy := httputil.NewSingleHostReverseProxy(target)
	proxy.ServeHTTP(w, r)
}

func (s *DevServer) handlePageWithCodegen(route *router.Route, mwCtx *middleware.Context, params map[string]string) {
	content, err := os.ReadFile(route.FilePath)
	if err != nil {
		http.Error(mwCtx.Response, err.Error(), http.StatusInternalServerError)
		return
	}

	comp, err := parser.Parse(string(content))
	if err != nil {
		http.Error(mwCtx.Response, fmt.Sprintf("Parse error: %v", err), http.StatusInternalServerError)
		return
	}

	// Process component tags BEFORE compiling
	// This resolves <Layout>, <Nav>, etc.
	resolver := s.Compiler.Resolver
	resolver.SetCurrentFile(route.FilePath)

	imports := make([]compiler.Import, len(comp.Imports))
	for i, imp := range comp.Imports {
		imports[i] = compiler.Import{
			Path:        imp.Path,
			Alias:       imp.Alias,
			IsComponent: imp.IsComponent,
		}
	}
	resolver.ParseImports(imports)

	// Create minimal executor context for component processing only
	dummyCtx := executor.NewContext()
	s.Compiler.CollectedStyles = nil
	s.Compiler.ResetComponentTracking()
	processedTemplate := s.Compiler.ProcessComponentTags(comp.Template, dummyCtx)

	if s.ComponentTracker != nil && len(s.Compiler.UsedComponents) > 0 {
		s.ComponentTracker.TrackPageComponents(route.FilePath, s.Compiler.UsedComponents)
	}

	// Update component with processed template
	comp.Template = processedTemplate

	fmHash := HashContent(comp.Frontmatter)
	tmplHash := HashContent(comp.Template)

	// Check cache without lock (fast path)
	cached, ok := s.PageCache.Get(route.Pattern)
	if ok && cached.FrontmatterHash == fmHash {
		if cached.TemplateHash != tmplHash {
			cached.Template = comp.Template
			cached.TemplateHash = tmplHash
		}
		fmt.Printf("⚡ Cache hit (in-memory): %s\n", route.Pattern)
	} else {
		// Lock during compilation to prevent duplicate compiles
		s.compileMu.Lock()

		// Double-check cache after acquiring lock
		cached, ok = s.PageCache.Get(route.Pattern)
		if ok && cached.FrontmatterHash == fmHash {
			s.compileMu.Unlock()
			fmt.Printf("⚡ Cache hit (after lock): %s\n", route.Pattern)
		} else {
			plugin, err := s.PluginCompiler.CompilePage(route, comp, fmHash)
			s.compileMu.Unlock()

			if err != nil {
				http.Error(mwCtx.Response, fmt.Sprintf("Compile error:\n%v", err), http.StatusInternalServerError)
				return
			}
			s.PageCache.Set(route.Pattern, plugin)
			cached = plugin
		}
	}

	// Bundle styles, scripts, and WASM
	allStyles := append(comp.Styles, s.Compiler.CollectedStyles...)
	compWithStyles := &parser.Component{
		Frontmatter: comp.Frontmatter,
		Template:    comp.Template,
		Scripts:     comp.Scripts,
		Styles:      allStyles,
		Imports:     comp.Imports,
	}

	cssPath, err := s.Bundler.BundleStyles(compWithStyles, route.FilePath)
	if err != nil {
		http.Error(mwCtx.Response, fmt.Sprintf("Style bundle error: %v", err), http.StatusInternalServerError)
		return
	}

	jsPath, err := s.Bundler.BundleScripts(comp, route.FilePath)
	if err != nil {
		http.Error(mwCtx.Response, fmt.Sprintf("Script bundle error: %v", err), http.StatusInternalServerError)
		return
	}

	wasmAssets, err := s.Bundler.BundleWasmScripts(comp, route.FilePath)
	if err != nil {
		http.Error(mwCtx.Response, fmt.Sprintf("WASM bundle error: %v", err), http.StatusInternalServerError)
		return
	}

	scopeID := ""
	for _, style := range allStyles {
		if style.Scoped {
			scopeID = s.Bundler.GenerateScopeID(route.FilePath)
			break
		}
	}

	// Capture handler output using httptest.ResponseRecorder
	originalWriter := mwCtx.Response
	recorder := httptest.NewRecorder()
	mwCtx.Response = recorder

	// Call plugin handler
	cached.Handler(mwCtx.Response, mwCtx.Request, params, mwCtx.Locals)

	// Get captured HTML
	rendered := recorder.Body.String()

	// Inject assets (WASM, CSS, JS)
	rendered = s.Bundler.InjectAssetsWithWasm(rendered, cssPath, jsPath, scopeID, wasmAssets)

	// Write final output to original writer
	originalWriter.Header().Set("Content-Type", "text/html; charset=utf-8")
	originalWriter.Write([]byte(rendered))
}

func (s *DevServer) serveHMRClient(w http.ResponseWriter, r *http.Request) {
	clientJS, err := os.ReadFile(getHMRClientPath())
	if err != nil {
		http.Error(w, "HMR client not found", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/javascript")
	w.Write(clientJS)
}

func getHMRClientPath() string {
	paths := []string{
		"/Users/cameron/dev/galaxy-mono/galaxy/pkg/hmr/client.js",
		"pkg/hmr/client.js",
		"../galaxy/pkg/hmr/client.js",
		"../../galaxy/pkg/hmr/client.js",
		"../../../galaxy/pkg/hmr/client.js",
	}
	for _, p := range paths {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	return ""
}

func (s *DevServer) handleHMRRender(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Query().Get("path")
	if path == "" {
		http.Error(w, "path required", http.StatusBadRequest)
		return
	}

	route, _ := s.Router.Match(r.URL.Path)
	if route == nil {
		route = &router.Route{FilePath: path}
	}

	content, err := os.ReadFile(path)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	comp, err := parser.Parse(string(content))
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"error": err.Error(),
			"stack": fmt.Sprintf("%+v", err),
		})
		return
	}

	resolver := s.Compiler.Resolver
	resolver.SetCurrentFile(path)

	imports := make([]compiler.Import, len(comp.Imports))
	for i, imp := range comp.Imports {
		imports[i] = compiler.Import{
			Path:        imp.Path,
			Alias:       imp.Alias,
			IsComponent: imp.IsComponent,
		}
	}
	resolver.ParseImports(imports)

	ctx := executor.NewContext()
	if comp.Frontmatter != "" {
		if err := ctx.Execute(comp.Frontmatter); err != nil {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]string{
				"error": err.Error(),
				"stack": fmt.Sprintf("%+v", err),
			})
			return
		}
	}

	s.Compiler.CollectedStyles = nil
	processedTemplate := s.Compiler.ProcessComponentTags(comp.Template, ctx)

	engine := template.NewEngine(ctx)
	rendered, err := engine.Render(processedTemplate, nil)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"error": err.Error(),
			"stack": fmt.Sprintf("%+v", err),
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"html": rendered,
		"path": path,
	})
}

func (s *DevServer) serveHMRMorph(w http.ResponseWriter, r *http.Request) {
	morphJS, err := os.ReadFile(getHMRMorphPath())
	if err != nil {
		http.Error(w, "morph.js not found", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/javascript")
	w.Write(morphJS)
}

func getHMRMorphPath() string {
	paths := []string{
		"/Users/cameron/dev/galaxy-mono/galaxy/pkg/hmr/morph.js",
		"pkg/hmr/morph.js",
		"../galaxy/pkg/hmr/morph.js",
		"../../galaxy/pkg/hmr/morph.js",
		"../../../galaxy/pkg/hmr/morph.js",
	}
	for _, p := range paths {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	return ""
}

func (s *DevServer) serveHMROverlay(w http.ResponseWriter, r *http.Request) {
	overlayJS, err := os.ReadFile(getHMROverlayPath())
	if err != nil {
		http.Error(w, "overlay.js not found", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/javascript")
	w.Write(overlayJS)
}

func getHMROverlayPath() string {
	paths := []string{
		"/Users/cameron/dev/galaxy-mono/galaxy/pkg/hmr/overlay.js",
		"pkg/hmr/overlay.js",
		"../galaxy/pkg/hmr/overlay.js",
		"../../galaxy/pkg/hmr/overlay.js",
		"../../../galaxy/pkg/hmr/overlay.js",
	}
	for _, p := range paths {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	return ""
}
