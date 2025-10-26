package router

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
)

type RouteType int

const (
	RouteStatic RouteType = iota
	RouteDynamic
	RouteCatchAll
	RouteEndpoint
	RouteMarkdown
)

type Route struct {
	Pattern    string
	FilePath   string
	Type       RouteType
	ParamNames []string
	Priority   int
	Regex      *regexp.Regexp
	IsEndpoint bool
}

type Router struct {
	Routes   []*Route
	PagesDir string
	mu       sync.RWMutex
}

func NewRouter(pagesDir string) *Router {
	return &Router{
		Routes:   make([]*Route, 0),
		PagesDir: pagesDir,
	}
}

func (r *Router) Discover() error {
	return r.discover()
}

func (r *Router) createRoute(relPath, fullPath string) *Route {
	route := &Route{
		FilePath: fullPath,
	}

	pattern := relPath
	pattern = strings.TrimSuffix(pattern, ".gxc")
	pattern = strings.TrimSuffix(pattern, ".go")
	pattern = strings.TrimSuffix(pattern, ".md")
	pattern = strings.TrimSuffix(pattern, ".mdx")

	// Strip HTTP method suffixes for endpoints
	for _, method := range []string{"/GET", "/POST", "/PUT", "/DELETE", "/PATCH"} {
		if strings.HasSuffix(pattern, method) {
			pattern = strings.TrimSuffix(pattern, method)
			break
		}
	}

	if strings.HasSuffix(pattern, "/route") {
		pattern = strings.TrimSuffix(pattern, "/route")
	}

	if strings.HasSuffix(pattern, "/index") {
		pattern = strings.TrimSuffix(pattern, "/index")
	}

	if pattern == "index" || pattern == "route" {
		pattern = ""
	}

	if pattern == "" {
		pattern = "/"
	} else {
		pattern = "/" + filepath.ToSlash(pattern)
	}

	route.Pattern = pattern
	route.Type = RouteStatic
	route.Priority = 100

	catchAllRegex := regexp.MustCompile(`\[\.\.\.(\w+)\]`)
	if catchAllRegex.MatchString(pattern) {
		route.Type = RouteCatchAll
		route.Priority = 10
		matches := catchAllRegex.FindAllStringSubmatch(pattern, -1)
		for _, match := range matches {
			route.ParamNames = append(route.ParamNames, match[1])
		}
		regexPattern := catchAllRegex.ReplaceAllString(pattern, `(.*)`)
		regexPattern = "^" + regexPattern + "$"
		route.Regex = regexp.MustCompile(regexPattern)
		return route
	}

	dynamicRegex := regexp.MustCompile(`\[(\w+)\]`)
	if dynamicRegex.MatchString(pattern) {
		route.Type = RouteDynamic
		route.Priority = 50
		matches := dynamicRegex.FindAllStringSubmatch(pattern, -1)
		for _, match := range matches {
			route.ParamNames = append(route.ParamNames, match[1])
		}
		regexPattern := dynamicRegex.ReplaceAllString(pattern, `([^/]+)`)
		regexPattern = "^" + regexPattern + "$"
		route.Regex = regexp.MustCompile(regexPattern)
	}

	return route
}

func (r *Router) Sort() {
	r.sort()
}

func (r *Router) Match(path string) (*Route, map[string]string) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, route := range r.Routes {
		if params := r.matchRoute(route, path); params != nil {
			return route, params
		}
	}
	return nil, nil
}

func (r *Router) matchRoute(route *Route, path string) map[string]string {
	if route.Type == RouteStatic || route.Type == RouteEndpoint || route.Type == RouteMarkdown {
		if route.Pattern == path {
			return make(map[string]string)
		}
		return nil
	}

	if route.Regex == nil {
		return nil
	}

	matches := route.Regex.FindStringSubmatch(path)
	if matches == nil {
		return nil
	}

	params := make(map[string]string)
	for i, name := range route.ParamNames {
		if i+1 < len(matches) {
			params[name] = matches[i+1]
		}
	}

	return params
}

func (r *Router) Reload() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.Routes = make([]*Route, 0)

	if err := r.discover(); err != nil {
		return err
	}

	r.sort()
	return nil
}

func (r *Router) discover() error {
	return filepath.Walk(r.PagesDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		isGxc := strings.HasSuffix(path, ".gxc")
		isGoEndpoint := strings.HasSuffix(path, ".go")
		isMarkdown := strings.HasSuffix(path, ".md") || strings.HasSuffix(path, ".mdx")

		if !isGxc && !isGoEndpoint && !isMarkdown {
			return nil
		}

		relPath, err := filepath.Rel(r.PagesDir, path)
		if err != nil {
			return err
		}

		route := r.createRoute(relPath, path)
		if isGoEndpoint {
			route.IsEndpoint = true
			route.Type = RouteEndpoint
		} else if isMarkdown {
			route.Type = RouteMarkdown
		}
		r.Routes = append(r.Routes, route)

		return nil
	})
}

func (r *Router) sort() {
	sort.Slice(r.Routes, func(i, j int) bool {
		if r.Routes[i].Priority != r.Routes[j].Priority {
			return r.Routes[i].Priority > r.Routes[j].Priority
		}
		return r.Routes[i].Pattern < r.Routes[j].Pattern
	})
}

func (r *Router) String() string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var sb strings.Builder
	sb.WriteString("=== Router ===\n")
	sb.WriteString(fmt.Sprintf("Pages Dir: %s\n", r.PagesDir))
	sb.WriteString(fmt.Sprintf("Routes: %d\n\n", len(r.Routes)))

	for _, route := range r.Routes {
		typeStr := "static"
		if route.Type == RouteDynamic {
			typeStr = "dynamic"
		} else if route.Type == RouteCatchAll {
			typeStr = "catch-all"
		}

		sb.WriteString(fmt.Sprintf("  %s [%s] (priority: %d)\n", route.Pattern, typeStr, route.Priority))
		if len(route.ParamNames) > 0 {
			sb.WriteString(fmt.Sprintf("    params: %v\n", route.ParamNames))
		}
	}

	return sb.String()
}
