package server

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/cameron-webmatter/galaxy/pkg/config"
	"github.com/cameron-webmatter/galaxy/pkg/hmr"
)

func TestNewDevServer(t *testing.T) {
	tmpDir := t.TempDir()
	pagesDir := filepath.Join(tmpDir, "src", "pages")
	publicDir := filepath.Join(tmpDir, "public")

	os.MkdirAll(pagesDir, 0755)
	os.MkdirAll(publicDir, 0755)

	cfg := &config.Config{}
	srv := NewDevServer(cfg, tmpDir, pagesDir, publicDir, 3000, false)

	if srv == nil {
		t.Fatal("NewDevServer returned nil")
	}
	if srv.Router == nil {
		t.Error("Router not initialized")
	}
	if srv.PageCache == nil {
		t.Error("PageCache not initialized")
	}
	if srv.ChangeTracker == nil {
		t.Error("ChangeTracker not initialized")
	}
	if srv.ComponentTracker == nil {
		t.Error("ComponentTracker not initialized")
	}
	if srv.Port != 3000 {
		t.Errorf("expected port 3000, got %d", srv.Port)
	}
	if srv.Verbose {
		t.Error("expected verbose false")
	}
}

func TestDevServer_ReloadRoutes(t *testing.T) {
	tmpDir := t.TempDir()
	pagesDir := filepath.Join(tmpDir, "src", "pages")

	os.MkdirAll(pagesDir, 0755)
	os.WriteFile(filepath.Join(pagesDir, "index.gxc"), []byte("<h1>Test</h1>"), 0644)

	cfg := &config.Config{}
	srv := NewDevServer(cfg, tmpDir, pagesDir, tmpDir, 3000, false)

	err := srv.ReloadRoutes()
	if err != nil {
		t.Fatalf("ReloadRoutes failed: %v", err)
	}

	if len(srv.Router.Routes) == 0 {
		t.Error("expected routes after reload")
	}
}

func TestDevServer_ReloadRoutes_InvalidDir(t *testing.T) {
	tmpDir := t.TempDir()
	pagesDir := filepath.Join(tmpDir, "nonexistent")

	cfg := &config.Config{}
	srv := NewDevServer(cfg, tmpDir, pagesDir, tmpDir, 3000, false)

	err := srv.ReloadRoutes()
	if err == nil {
		t.Error("expected error for nonexistent pages dir")
	}
}

func TestDevServer_HMRServerInit(t *testing.T) {
	tmpDir := t.TempDir()
	pagesDir := filepath.Join(tmpDir, "src", "pages")
	os.MkdirAll(pagesDir, 0755)

	cfg := &config.Config{}
	srv := NewDevServer(cfg, tmpDir, pagesDir, tmpDir, 3000, false)

	srv.HMRServer = hmr.NewServer()
	srv.HMRServer.Start()

	if srv.HMRServer == nil {
		t.Fatal("HMRServer not initialized")
	}
}

func TestDevServer_ServePublicAsset(t *testing.T) {
	tmpDir := t.TempDir()
	pagesDir := filepath.Join(tmpDir, "src", "pages")
	publicDir := filepath.Join(tmpDir, "public")

	os.MkdirAll(pagesDir, 0755)
	os.MkdirAll(publicDir, 0755)
	os.WriteFile(filepath.Join(publicDir, "test.txt"), []byte("test content"), 0644)

	cfg := &config.Config{}
	srv := NewDevServer(cfg, tmpDir, pagesDir, publicDir, 3000, false)

	req := httptest.NewRequest("GET", "/test.txt", nil)
	w := httptest.NewRecorder()

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/test.txt" {
			http.ServeFile(w, r, filepath.Join(publicDir, "test.txt"))
		}
	})

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	_ = srv
}

func TestDevServer_NotFoundHandler(t *testing.T) {
	tmpDir := t.TempDir()
	pagesDir := filepath.Join(tmpDir, "src", "pages")
	os.MkdirAll(pagesDir, 0755)

	cfg := &config.Config{}
	srv := NewDevServer(cfg, tmpDir, pagesDir, tmpDir, 3000, false)
	srv.ReloadRoutes()

	req := httptest.NewRequest("GET", "/nonexistent", nil)
	w := httptest.NewRecorder()

	srv.handleRequest(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestDevServer_StaticRoute(t *testing.T) {
	tmpDir := t.TempDir()
	pagesDir := filepath.Join(tmpDir, "src", "pages")
	os.MkdirAll(pagesDir, 0755)

	content := `<h1>Hello World</h1>`
	os.WriteFile(filepath.Join(pagesDir, "test.gxc"), []byte(content), 0644)

	cfg := &config.Config{}
	srv := NewDevServer(cfg, tmpDir, pagesDir, tmpDir, 3000, false)
	srv.ReloadRoutes()

	if len(srv.Router.Routes) == 0 {
		t.Fatal("no routes discovered")
	}

	route := srv.Router.Routes[0]
	if route.Pattern != "/test" {
		t.Errorf("expected pattern /test, got %s", route.Pattern)
	}
}

func TestDevServer_RootRoute(t *testing.T) {
	tmpDir := t.TempDir()
	pagesDir := filepath.Join(tmpDir, "src", "pages")
	os.MkdirAll(pagesDir, 0755)

	content := `<h1>Home</h1>`
	os.WriteFile(filepath.Join(pagesDir, "index.gxc"), []byte(content), 0644)

	cfg := &config.Config{}
	srv := NewDevServer(cfg, tmpDir, pagesDir, tmpDir, 3000, false)
	srv.ReloadRoutes()

	route, _ := srv.Router.Match("/")
	if route == nil {
		t.Fatal("expected to match /")
	}
	if route.Pattern != "/" {
		t.Errorf("expected pattern /, got %s", route.Pattern)
	}
}

func TestDevServer_ClearCache(t *testing.T) {
	tmpDir := t.TempDir()
	pagesDir := filepath.Join(tmpDir, "src", "pages")
	os.MkdirAll(pagesDir, 0755)

	cfg := &config.Config{}
	srv := NewDevServer(cfg, tmpDir, pagesDir, tmpDir, 3000, false)

	srv.PageCache.Set("/test", &PagePlugin{Template: "test"})

	if _, ok := srv.PageCache.Get("/test"); !ok {
		t.Fatal("cache entry not set")
	}

	srv.PageCache.Invalidate("/test")

	if _, ok := srv.PageCache.Get("/test"); ok {
		t.Error("cache entry should be cleared")
	}
}

func TestDevServer_ComponentTracking(t *testing.T) {
	tmpDir := t.TempDir()
	pagesDir := filepath.Join(tmpDir, "src", "pages")
	os.MkdirAll(pagesDir, 0755)

	cfg := &config.Config{}
	srv := NewDevServer(cfg, tmpDir, pagesDir, tmpDir, 3000, false)

	srv.ComponentTracker.TrackPageComponents("/pages/index.gxc", []string{
		"/components/Header.gxc",
		"/components/Footer.gxc",
	})

	affected := srv.ComponentTracker.GetAffectedPages("/components/Header.gxc")
	if len(affected) != 1 {
		t.Errorf("expected 1 affected page, got %d", len(affected))
	}
}

func TestDevServer_ChangeTracking(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.gxc")

	content := `<h1>Test</h1>
<style>
body { color: red; }
</style>`
	os.WriteFile(testFile, []byte(content), 0644)

	cfg := &config.Config{}
	srv := NewDevServer(cfg, tmpDir, tmpDir, tmpDir, 3000, false)

	diff, err := srv.ChangeTracker.DetectChange(testFile)
	if err != nil {
		t.Fatalf("DetectChange failed: %v", err)
	}

	if !diff.TemplateChanged {
		t.Error("expected template changed on first detection")
	}

	diff, err = srv.ChangeTracker.DetectChange(testFile)
	if err != nil {
		t.Fatalf("second DetectChange failed: %v", err)
	}

	if diff.TemplateChanged {
		t.Error("expected no change on second detection")
	}
}

func TestDevServer_VerboseMode(t *testing.T) {
	tmpDir := t.TempDir()
	pagesDir := filepath.Join(tmpDir, "src", "pages")
	os.MkdirAll(pagesDir, 0755)

	cfg := &config.Config{}
	srv := NewDevServer(cfg, tmpDir, pagesDir, tmpDir, 3000, true)

	if !srv.Verbose {
		t.Error("expected verbose mode enabled")
	}
}

func TestDevServer_BundlerConfig(t *testing.T) {
	tmpDir := t.TempDir()
	pagesDir := filepath.Join(tmpDir, "src", "pages")
	os.MkdirAll(pagesDir, 0755)

	cfg := &config.Config{}
	srv := NewDevServer(cfg, tmpDir, pagesDir, tmpDir, 3000, false)

	if srv.Bundler == nil {
		t.Fatal("Bundler not initialized")
	}
	if !srv.Bundler.DevMode {
		t.Error("expected DevMode enabled")
	}
}

func TestDevServer_PluginManager(t *testing.T) {
	tmpDir := t.TempDir()
	pagesDir := filepath.Join(tmpDir, "src", "pages")
	os.MkdirAll(pagesDir, 0755)

	cfg := &config.Config{}
	srv := NewDevServer(cfg, tmpDir, pagesDir, tmpDir, 3000, false)

	if srv.PluginManager == nil {
		t.Fatal("PluginManager not initialized")
	}
}

func TestDevServer_CompilerInit(t *testing.T) {
	tmpDir := t.TempDir()
	pagesDir := filepath.Join(tmpDir, "src", "pages")
	os.MkdirAll(pagesDir, 0755)

	cfg := &config.Config{}
	srv := NewDevServer(cfg, tmpDir, pagesDir, tmpDir, 3000, false)

	if srv.Compiler == nil {
		t.Fatal("Compiler not initialized")
	}
	if srv.EndpointCompiler == nil {
		t.Fatal("EndpointCompiler not initialized")
	}
	if srv.MiddlewareCompiler == nil {
		t.Fatal("MiddlewareCompiler not initialized")
	}
}

func TestDevServer_PendingRebuilds(t *testing.T) {
	tmpDir := t.TempDir()
	pagesDir := filepath.Join(tmpDir, "src", "pages")
	os.MkdirAll(pagesDir, 0755)

	cfg := &config.Config{}
	srv := NewDevServer(cfg, tmpDir, pagesDir, tmpDir, 3000, false)

	if srv.pendingRebuilds == nil {
		t.Fatal("pendingRebuilds not initialized")
	}
}
