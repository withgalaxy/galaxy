package orbit

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/withgalaxy/galaxy/pkg/assets"
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

	if p.Lifecycle != nil {
		if err := p.Lifecycle.ExecuteStartup(); err != nil {
			return fmt.Errorf("lifecycle startup: %w", err)
		}
	}

	return nil
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
			route, params := p.Router.Match(r.URL.Path)
			if route == nil {
				next.ServeHTTP(w, r)
				return
			}

			if p.MiddlewareChain != nil {
				mwCtx := middleware.NewContext(w, r)
				mwCtx.Params = params

				err := p.MiddlewareChain.Execute(mwCtx, func(ctx *middleware.Context) error {
					p.handleRoute(ctx.Response, ctx.Request, route, params)
					return nil
				})
				if err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
				}
				return
			}

			p.handleRoute(w, r, route, params)
		})
	}
}
