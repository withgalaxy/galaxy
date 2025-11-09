package orbit

import (
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/withgalaxy/galaxy/pkg/assets"
	"github.com/withgalaxy/galaxy/pkg/codegen"
	"github.com/withgalaxy/galaxy/pkg/compiler"
	"github.com/withgalaxy/galaxy/pkg/endpoints"
	"github.com/withgalaxy/galaxy/pkg/hmr"
	"github.com/withgalaxy/galaxy/pkg/lifecycle"
	"github.com/withgalaxy/galaxy/pkg/middleware"
	"github.com/withgalaxy/galaxy/pkg/router"
	"github.com/withgalaxy/galaxy/pkg/server"
	orbit "github.com/withgalaxy/orbit/plugin"
)

type GalaxyPlugin struct {
	orbit.BasePlugin

	Compiler           *compiler.ComponentCompiler
	Router             *router.Router
	EndpointCompiler   *endpoints.EndpointCompiler
	Bundler            *assets.Bundler
	Cache              *server.PageCache
	ChangeTracker      *hmr.ChangeTracker
	ComponentTracker   *hmr.ComponentTracker
	MiddlewareCompiler *middleware.MiddlewareCompiler
	MiddlewareChain    *middleware.Chain
	LoadedMiddleware   *middleware.LoadedMiddleware
	Lifecycle          *lifecycle.Lifecycle
	UseCodegen         bool
	CodegenPort        int
	codegenCmd         *exec.Cmd
	codegenReady       bool

	RootDir   string
	PagesDir  string
	PublicDir string
}

func NewGalaxyPlugin(rootDir, pagesDir, publicDir string) *GalaxyPlugin {
	srcDir := filepath.Dir(pagesDir)
	bundler := assets.NewBundler(".galaxy")
	bundler.DevMode = true

	p := &GalaxyPlugin{
		Compiler:           compiler.NewComponentCompiler(srcDir),
		Router:             router.NewRouter(pagesDir),
		EndpointCompiler:   endpoints.NewCompiler(rootDir, ".galaxy/endpoints"),
		Bundler:            bundler,
		Cache:              server.NewPageCache(),
		ChangeTracker:      hmr.NewChangeTracker(),
		ComponentTracker:   hmr.NewComponentTracker(),
		MiddlewareCompiler: middleware.NewCompiler(rootDir, ".galaxy/middleware"),
		RootDir:            rootDir,
		PagesDir:           pagesDir,
		PublicDir:          publicDir,
	}

	middlewarePath := filepath.Join(srcDir, "middleware.go")
	if _, err := os.Stat(middlewarePath); err == nil {
		p.loadMiddleware()
	}

	if lifecycle.DetectLifecycle(srcDir) {
		loaded, err := lifecycle.LoadFromDir(srcDir)
		if err == nil && loaded != nil {
			p.Lifecycle = lifecycle.NewLifecycle()
			p.Lifecycle.Register(loaded)
		}
	}

	return p
}

func (p *GalaxyPlugin) loadMiddleware() error {
	srcDir := filepath.Dir(p.PagesDir)
	middlewarePath := filepath.Join(srcDir, "middleware.go")

	loaded, err := p.MiddlewareCompiler.Load(middlewarePath)
	if err != nil {
		return err
	}

	p.LoadedMiddleware = loaded
	p.MiddlewareChain = middleware.NewChain()

	if loaded.Sequence != nil && len(loaded.Sequence) > 0 {
		for _, fn := range loaded.Sequence {
			p.MiddlewareChain.Use(fn)
		}
	}

	return nil
}

func (p *GalaxyPlugin) Name() string {
	return "galaxy"
}

func (p *GalaxyPlugin) ConfigResolved(config any) error {
	if err := p.Router.Discover(); err != nil {
		return fmt.Errorf("discover routes: %w", err)
	}
	p.Router.Sort()

	// Print discovered routes
	for _, route := range p.Router.Routes {
		fmt.Printf("  %s\n", route.Pattern)
	}
	fmt.Println()

	if p.UseCodegen {
		if err := p.buildCodegenServer(); err != nil {
			return fmt.Errorf("codegen: %w", err)
		}
	}

	if p.Lifecycle != nil {
		if err := p.Lifecycle.ExecuteStartup(); err != nil {
			return fmt.Errorf("lifecycle startup: %w", err)
		}
	}

	return nil
}

func (p *GalaxyPlugin) buildCodegenServer() error {
	builder := codegen.NewCodegenBuilder(p.Router.Routes, p.PagesDir, ".galaxy", "dev-server", p.PublicDir)
	builder.Bundler = p.Bundler
	if err := builder.Build(); err != nil {
		return fmt.Errorf("build failed: %w", err)
	}

	// Start codegen server
	if err := p.startCodegenServer(); err != nil {
		return fmt.Errorf("start codegen server: %w", err)
	}

	return nil
}

func (p *GalaxyPlugin) startCodegenServer() error {
	p.CodegenPort = 6173 // Fixed port for now

	cmd := exec.Command("./galaxy-codegen-server")
	cmd.Dir = filepath.Join(p.RootDir, ".galaxy", "server")
	cmd.Env = append(os.Environ(),
		fmt.Sprintf("PORT=%d", p.CodegenPort),
		"DEV_MODE=true",
	)
	// Suppress codegen server output
	cmd.Stdout = nil
	cmd.Stderr = nil

	if err := cmd.Start(); err != nil {
		return err
	}

	p.codegenCmd = cmd

	// Wait for ready
	for i := 0; i < 50; i++ {
		resp, err := http.Get(fmt.Sprintf("http://localhost:%d/health", p.CodegenPort))
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode < 500 {
				p.codegenReady = true
				return nil
			}
		}
		time.Sleep(100 * time.Millisecond)
	}

	return fmt.Errorf("codegen server did not become ready")
}

func (p *GalaxyPlugin) ConfigureServer(server any) error {
	return nil
}

func (p *GalaxyPlugin) HandleHotUpdate(file string) ([]string, error) {
	if strings.HasSuffix(file, "middleware.go") {
		if err := p.loadMiddleware(); err != nil {
			return nil, fmt.Errorf("reload middleware: %w", err)
		}
		return []string{file}, nil
	}

	if !strings.HasSuffix(file, ".gxc") {
		return nil, nil
	}

	_, err := p.ChangeTracker.DetectChange(file)
	if err != nil {
		return nil, err
	}

	p.Cache.Invalidate(file)

	affectedPages := p.ComponentTracker.GetAffectedPages(file)
	if len(affectedPages) > 0 {
		for _, page := range affectedPages {
			p.Cache.Invalidate(page)
		}
		return affectedPages, nil
	}

	return []string{file}, nil
}

func (p *GalaxyPlugin) Middleware() orbit.Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			// Wrap response writer to capture status code
			rw := &responseWriter{ResponseWriter: w, statusCode: 200}

			route, params := p.Router.Match(r.URL.Path)
			if route == nil {
				next.ServeHTTP(rw, r)
				p.logRequest(r, rw.statusCode, time.Since(start))
				return
			}

			// Proxy to codegen server if ready
			if p.UseCodegen && p.codegenReady {
				p.proxyToCodegen(rw, r)
				p.logRequest(r, rw.statusCode, time.Since(start))
				return
			}

			if p.MiddlewareChain != nil {
				mwCtx := middleware.NewContext(rw, r)
				mwCtx.Params = params

				err := p.MiddlewareChain.Execute(mwCtx, func(ctx *middleware.Context) error {
					p.handleRoute(ctx.Response, ctx.Request, route, params)
					return nil
				})
				if err != nil {
					http.Error(rw, err.Error(), http.StatusInternalServerError)
				}
				p.logRequest(r, rw.statusCode, time.Since(start))
				return
			}

			p.handleRoute(rw, r, route, params)
			p.logRequest(r, rw.statusCode, time.Since(start))
		})
	}
}

type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

func (p *GalaxyPlugin) logRequest(r *http.Request, statusCode int, duration time.Duration) {
	methodColor := "\033[36m"
	statusColor := getStatusColor(statusCode)
	reset := "\033[0m"

	fmt.Printf("%s%s%s %s - %s%d%s (%dms)\n",
		methodColor, r.Method, reset,
		r.URL.Path,
		statusColor, statusCode, reset,
		duration.Milliseconds())
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

func (p *GalaxyPlugin) proxyToCodegen(w http.ResponseWriter, r *http.Request) {
	target := fmt.Sprintf("http://localhost:%d%s", p.CodegenPort, r.URL.Path)

	proxyReq, _ := http.NewRequest(r.Method, target, r.Body)
	proxyReq.Header = r.Header

	client := &http.Client{}
	resp, err := client.Do(proxyReq)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	for k, v := range resp.Header {
		w.Header()[k] = v
	}
	w.WriteHeader(resp.StatusCode)

	buf := make([]byte, 32*1024)
	for {
		n, err := resp.Body.Read(buf)
		if n > 0 {
			w.Write(buf[:n])
		}
		if err != nil {
			break
		}
	}
}
