package orbit

import (
	"fmt"
	"net/http"

	"github.com/cameron-webmatter/galaxy/pkg/executor"
	"github.com/cameron-webmatter/galaxy/pkg/router"
	"github.com/cameron-webmatter/galaxy/pkg/server"
)

func (p *GalaxyPlugin) handleRoute(w http.ResponseWriter, r *http.Request, route *router.Route, params map[string]string) {
	if route.IsEndpoint {
		http.Error(w, "Endpoints not implemented in Orbit plugin yet", http.StatusNotImplemented)
		return
	}

	switch route.Type {
	case router.RouteStatic, router.RouteDynamic, router.RouteCatchAll:
		p.handlePage(w, r, route, params)
	case router.RouteMarkdown:
		http.Error(w, "Markdown not implemented in Orbit plugin yet", http.StatusNotImplemented)
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

	p.Cache.Set(cacheKey, &server.PagePlugin{
		Template: html,
	})

	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(html))
}
