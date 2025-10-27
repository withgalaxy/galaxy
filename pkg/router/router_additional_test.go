package router

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNewRouter(t *testing.T) {
	router := NewRouter("/test/pages")
	if router == nil {
		t.Fatal("NewRouter returned nil")
	}
	if router.PagesDir != "/test/pages" {
		t.Errorf("expected PagesDir /test/pages, got %s", router.PagesDir)
	}
	if router.Routes == nil {
		t.Error("Routes not initialized")
	}
}

func TestRouter_EmptyDirectory(t *testing.T) {
	tmpDir := t.TempDir()

	router := NewRouter(tmpDir)
	err := router.Discover()
	if err != nil {
		t.Fatalf("Discover failed: %v", err)
	}

	if len(router.Routes) != 0 {
		t.Errorf("expected 0 routes for empty dir, got %d", len(router.Routes))
	}
}

func TestRouter_NestedRoutes(t *testing.T) {
	tmpDir := t.TempDir()

	os.MkdirAll(filepath.Join(tmpDir, "blog", "posts"), 0755)
	os.WriteFile(filepath.Join(tmpDir, "blog", "index.gxc"), []byte(""), 0644)
	os.WriteFile(filepath.Join(tmpDir, "blog", "posts", "first.gxc"), []byte(""), 0644)

	router := NewRouter(tmpDir)
	err := router.Discover()
	if err != nil {
		t.Fatalf("Discover failed: %v", err)
	}

	router.Sort()

	route, _ := router.Match("/blog")
	if route == nil {
		t.Error("expected /blog to match")
	}

	route, _ = router.Match("/blog/posts/first")
	if route == nil {
		t.Error("expected /blog/posts/first to match")
	}
}

func TestRouter_MarkdownRoutes(t *testing.T) {
	tmpDir := t.TempDir()

	os.WriteFile(filepath.Join(tmpDir, "readme.md"), []byte(""), 0644)
	os.WriteFile(filepath.Join(tmpDir, "guide.mdx"), []byte(""), 0644)

	router := NewRouter(tmpDir)
	err := router.Discover()
	if err != nil {
		t.Fatalf("Discover failed: %v", err)
	}

	router.Sort()

	if len(router.Routes) != 2 {
		t.Errorf("expected 2 routes, got %d", len(router.Routes))
	}

	route, _ := router.Match("/readme")
	if route == nil {
		t.Error("expected /readme to match")
	}
	if route.Type != RouteMarkdown {
		t.Error("expected RouteMarkdown type")
	}
}

func TestRouter_EndpointRoutes(t *testing.T) {
	tmpDir := t.TempDir()

	os.MkdirAll(filepath.Join(tmpDir, "api"), 0755)
	os.WriteFile(filepath.Join(tmpDir, "api", "GET.go"), []byte(""), 0644)
	os.WriteFile(filepath.Join(tmpDir, "api", "POST.go"), []byte(""), 0644)

	router := NewRouter(tmpDir)
	err := router.Discover()
	if err != nil {
		t.Fatalf("Discover failed: %v", err)
	}

	router.Sort()

	if len(router.Routes) != 2 {
		t.Errorf("expected 2 routes, got %d", len(router.Routes))
	}

	for _, route := range router.Routes {
		if !route.IsEndpoint {
			t.Error("expected route to be endpoint")
		}
		if route.Pattern != "/api" {
			t.Errorf("expected pattern /api, got %s", route.Pattern)
		}
	}
}

func TestRouter_MultipleDynamicParams(t *testing.T) {
	tmpDir := t.TempDir()

	os.MkdirAll(filepath.Join(tmpDir, "[category]"), 0755)
	os.WriteFile(filepath.Join(tmpDir, "[category]", "[slug].gxc"), []byte(""), 0644)

	router := NewRouter(tmpDir)
	err := router.Discover()
	if err != nil {
		t.Fatalf("Discover failed: %v", err)
	}

	router.Sort()

	route, params := router.Match("/tech/my-post")
	if route == nil {
		t.Fatal("expected to match /tech/my-post")
	}

	if params["category"] != "tech" {
		t.Errorf("expected category=tech, got %s", params["category"])
	}
	if params["slug"] != "my-post" {
		t.Errorf("expected slug=my-post, got %s", params["slug"])
	}
}

func TestRouter_IndexRoute(t *testing.T) {
	tmpDir := t.TempDir()

	os.WriteFile(filepath.Join(tmpDir, "index.gxc"), []byte(""), 0644)

	router := NewRouter(tmpDir)
	err := router.Discover()
	if err != nil {
		t.Fatalf("Discover failed: %v", err)
	}

	router.Sort()

	route, _ := router.Match("/")
	if route == nil {
		t.Fatal("expected to match /")
	}
	if route.Pattern != "/" {
		t.Errorf("expected pattern /, got %s", route.Pattern)
	}
}

func TestRouter_NoMatch(t *testing.T) {
	tmpDir := t.TempDir()

	os.WriteFile(filepath.Join(tmpDir, "about.gxc"), []byte(""), 0644)

	router := NewRouter(tmpDir)
	err := router.Discover()
	if err != nil {
		t.Fatalf("Discover failed: %v", err)
	}

	router.Sort()

	route, params := router.Match("/nonexistent")
	if route != nil {
		t.Error("expected no match for /nonexistent")
	}
	if params != nil {
		t.Error("expected nil params for no match")
	}
}

func TestRouter_Reload(t *testing.T) {
	tmpDir := t.TempDir()

	os.WriteFile(filepath.Join(tmpDir, "initial.gxc"), []byte(""), 0644)

	router := NewRouter(tmpDir)
	err := router.Discover()
	if err != nil {
		t.Fatalf("Discover failed: %v", err)
	}

	initialCount := len(router.Routes)

	os.WriteFile(filepath.Join(tmpDir, "new.gxc"), []byte(""), 0644)

	err = router.Reload()
	if err != nil {
		t.Fatalf("Reload failed: %v", err)
	}

	if len(router.Routes) <= initialCount {
		t.Error("expected more routes after reload")
	}
}

func TestRouter_CatchAllWithPrefix(t *testing.T) {
	tmpDir := t.TempDir()

	os.MkdirAll(filepath.Join(tmpDir, "docs"), 0755)
	os.WriteFile(filepath.Join(tmpDir, "docs", "[...path].gxc"), []byte(""), 0644)

	router := NewRouter(tmpDir)
	err := router.Discover()
	if err != nil {
		t.Fatalf("Discover failed: %v", err)
	}

	router.Sort()

	route, params := router.Match("/docs/guide/intro")
	if route == nil {
		t.Fatal("expected to match /docs/guide/intro")
	}

	if params["path"] != "guide/intro" {
		t.Errorf("expected path=guide/intro, got %s", params["path"])
	}
}

func TestRouter_SortByPriority(t *testing.T) {
	tmpDir := t.TempDir()

	os.WriteFile(filepath.Join(tmpDir, "static.gxc"), []byte(""), 0644)
	os.WriteFile(filepath.Join(tmpDir, "[dynamic].gxc"), []byte(""), 0644)
	os.WriteFile(filepath.Join(tmpDir, "[...catchall].gxc"), []byte(""), 0644)

	router := NewRouter(tmpDir)
	err := router.Discover()
	if err != nil {
		t.Fatalf("Discover failed: %v", err)
	}

	router.Sort()

	if len(router.Routes) != 3 {
		t.Fatalf("expected 3 routes, got %d", len(router.Routes))
	}

	if router.Routes[0].Type != RouteStatic {
		t.Error("expected static route first")
	}
	if router.Routes[1].Type != RouteDynamic {
		t.Error("expected dynamic route second")
	}
	if router.Routes[2].Type != RouteCatchAll {
		t.Error("expected catch-all route last")
	}
}

func TestRouter_RouteWithHTTPMethod(t *testing.T) {
	tmpDir := t.TempDir()

	os.MkdirAll(filepath.Join(tmpDir, "api", "users"), 0755)
	os.WriteFile(filepath.Join(tmpDir, "api", "users", "GET.go"), []byte(""), 0644)
	os.WriteFile(filepath.Join(tmpDir, "api", "users", "POST.go"), []byte(""), 0644)
	os.WriteFile(filepath.Join(tmpDir, "api", "users", "DELETE.go"), []byte(""), 0644)

	router := NewRouter(tmpDir)
	err := router.Discover()
	if err != nil {
		t.Fatalf("Discover failed: %v", err)
	}

	if len(router.Routes) != 3 {
		t.Errorf("expected 3 routes, got %d", len(router.Routes))
	}

	for _, route := range router.Routes {
		if route.Pattern != "/api/users" {
			t.Errorf("expected pattern /api/users, got %s", route.Pattern)
		}
	}
}

func TestRouter_IgnoreNonRouteFiles(t *testing.T) {
	tmpDir := t.TempDir()

	os.WriteFile(filepath.Join(tmpDir, "index.gxc"), []byte(""), 0644)
	os.WriteFile(filepath.Join(tmpDir, "readme.txt"), []byte(""), 0644)
	os.WriteFile(filepath.Join(tmpDir, "config.json"), []byte(""), 0644)

	router := NewRouter(tmpDir)
	err := router.Discover()
	if err != nil {
		t.Fatalf("Discover failed: %v", err)
	}

	if len(router.Routes) != 1 {
		t.Errorf("expected 1 route (ignoring non-route files), got %d", len(router.Routes))
	}
}

func TestRouter_DynamicRouteNoMatch(t *testing.T) {
	tmpDir := t.TempDir()

	os.MkdirAll(filepath.Join(tmpDir, "posts"), 0755)
	os.WriteFile(filepath.Join(tmpDir, "posts", "[id].gxc"), []byte(""), 0644)

	router := NewRouter(tmpDir)
	err := router.Discover()
	if err != nil {
		t.Fatalf("Discover failed: %v", err)
	}

	router.Sort()

	route, _ := router.Match("/posts/123/extra")
	if route != nil {
		t.Error("expected no match for /posts/123/extra")
	}
}

func TestRouter_ConcurrentAccess(t *testing.T) {
	tmpDir := t.TempDir()

	os.WriteFile(filepath.Join(tmpDir, "test.gxc"), []byte(""), 0644)

	router := NewRouter(tmpDir)
	err := router.Discover()
	if err != nil {
		t.Fatalf("Discover failed: %v", err)
	}

	router.Sort()

	done := make(chan bool, 10)

	for i := 0; i < 10; i++ {
		go func() {
			for j := 0; j < 100; j++ {
				_, _ = router.Match("/test")
			}
			done <- true
		}()
	}

	for i := 0; i < 10; i++ {
		<-done
	}
}
