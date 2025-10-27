package orbit

import (
	"fmt"
	"net/http"
	"os"

	"github.com/withgalaxy/galaxy/pkg/endpoints"
	"github.com/withgalaxy/galaxy/pkg/executor"
	"github.com/withgalaxy/galaxy/pkg/parser"
	"github.com/withgalaxy/galaxy/pkg/router"
	"github.com/withgalaxy/galaxy/pkg/server"
)

func (p *GalaxyPlugin) handleRoute(w http.ResponseWriter, r *http.Request, route *router.Route, params map[string]string) {
	if route.IsEndpoint {
		p.handleEndpoint(w, r, route, params)
		return
	}

	switch route.Type {
	case router.RouteStatic, router.RouteDynamic, router.RouteCatchAll:
		p.handlePage(w, r, route, params)
	case router.RouteMarkdown:
		p.handleMarkdown(w, r, route, params)
	default:
		http.Error(w, "Unknown route type", http.StatusInternalServerError)
	}
}

func (p *GalaxyPlugin) handlePage(w http.ResponseWriter, r *http.Request, route *router.Route, params map[string]string) {
	cacheKey := route.FilePath
	if len(params) > 0 {
		cacheKey = fmt.Sprintf("%s?%v", route.FilePath, params)
	}

	if cached, ok := p.Cache.Get(cacheKey); ok {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(cached.Template))
		return
	}

	p.Compiler.ClearCache()
	p.Compiler.ResetComponentTracking()

	ctx := executor.NewContext()
	for k, v := range params {
		ctx.Set(k, v)
	}

	html, err := p.Compiler.CompileWithContext(route.FilePath, nil, nil, ctx)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	p.ComponentTracker.TrackPageComponents(route.FilePath, p.Compiler.UsedComponents)

	content, err := os.ReadFile(route.FilePath)
	if err == nil {
		comp, parseErr := parser.Parse(string(content))
		if parseErr == nil {
			cssPath, _ := p.Bundler.BundleStyles(comp, route.FilePath)
			jsPath, _ := p.Bundler.BundleScripts(comp, route.FilePath)
			wasmAssets, _ := p.Bundler.BundleWasmScripts(comp, route.FilePath)

			scopeID := ""
			if cssPath != "" {
				scopeID = p.Bundler.GenerateScopeID(route.FilePath)
			}

			html = p.Bundler.InjectAssetsWithWasm(html, cssPath, jsPath, scopeID, wasmAssets)
		}
	}

	p.Cache.Set(cacheKey, &server.PagePlugin{
		Template: html,
	})

	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(html))
}

func (p *GalaxyPlugin) handleEndpoint(w http.ResponseWriter, r *http.Request, route *router.Route, params map[string]string) {
	loaded, err := p.EndpointCompiler.Load(route.FilePath)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	method := endpoints.HTTPMethod(r.Method)
	handler, ok := loaded.Handlers[method]
	if !ok {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	ctx := endpoints.NewContext(w, r, params, nil)

	if err := handler(ctx); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
