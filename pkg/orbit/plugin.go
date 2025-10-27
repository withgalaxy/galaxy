package orbit

import (
	"fmt"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/withgalaxy/galaxy/pkg/assets"
	"github.com/withgalaxy/galaxy/pkg/compiler"
	"github.com/withgalaxy/galaxy/pkg/endpoints"
	"github.com/withgalaxy/galaxy/pkg/hmr"
	"github.com/withgalaxy/galaxy/pkg/router"
	"github.com/withgalaxy/galaxy/pkg/server"
	orbit "github.com/withgalaxy/orbit/plugin"
)

type GalaxyPlugin struct {
	orbit.BasePlugin

	Compiler         *compiler.ComponentCompiler
	Router           *router.Router
	EndpointCompiler *endpoints.EndpointCompiler
	Bundler          *assets.Bundler
	Cache            *server.PageCache
	ChangeTracker    *hmr.ChangeTracker
	ComponentTracker *hmr.ComponentTracker

	RootDir   string
	PagesDir  string
	PublicDir string
}

func NewGalaxyPlugin(rootDir, pagesDir, publicDir string) *GalaxyPlugin {
	srcDir := filepath.Dir(pagesDir)
	bundler := assets.NewBundler(".galaxy")
	bundler.DevMode = true

	return &GalaxyPlugin{
		Compiler:         compiler.NewComponentCompiler(srcDir),
		Router:           router.NewRouter(pagesDir),
		EndpointCompiler: endpoints.NewCompiler(rootDir, ".galaxy/endpoints"),
		Bundler:          bundler,
		Cache:            server.NewPageCache(),
		ChangeTracker:    hmr.NewChangeTracker(),
		ComponentTracker: hmr.NewComponentTracker(),
		RootDir:          rootDir,
		PagesDir:         pagesDir,
		PublicDir:        publicDir,
	}
}

func (p *GalaxyPlugin) Name() string {
	return "galaxy"
}

func (p *GalaxyPlugin) ConfigResolved(config any) error {
	if err := p.Router.Discover(); err != nil {
		return fmt.Errorf("discover routes: %w", err)
	}
	p.Router.Sort()
	return nil
}

func (p *GalaxyPlugin) ConfigureServer(server any) error {
	return nil
}

func (p *GalaxyPlugin) HandleHotUpdate(file string) ([]string, error) {
	if !strings.HasSuffix(file, ".gxc") {
		return nil, nil
	}

	_, err := p.ChangeTracker.DetectChange(file)
	if err != nil {
		return nil, err
	}

	p.Cache.Invalidate(file)

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

			p.handleRoute(w, r, route, params)
		})
	}
}
