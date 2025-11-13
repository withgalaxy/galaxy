package vercel

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/withgalaxy/galaxy/pkg/adapters"
	"github.com/withgalaxy/galaxy/pkg/config"
)

func TestVercelAdapter_Name(t *testing.T) {
	adapter := New()
	if adapter.Name() != "vercel" {
		t.Errorf("expected name 'vercel', got '%s'", adapter.Name())
	}
}

func TestVercelAdapter_RejectsNonStatic(t *testing.T) {
	tmpDir := t.TempDir()

	adapter := New()
	cfg := &adapters.BuildConfig{
		Config: &config.Config{
			Output: config.OutputConfig{
				Type: config.OutputServer,
			},
		},
		OutDir: tmpDir,
	}

	err := adapter.Build(cfg)
	if err == nil {
		t.Fatal("expected error for non-static output type, got nil")
	}
}

func TestVercelAdapter_Build(t *testing.T) {
	tmpDir := t.TempDir()
	distDir := filepath.Join(tmpDir, "dist")

	if err := os.MkdirAll(distDir, 0755); err != nil {
		t.Fatal(err)
	}

	indexHTML := filepath.Join(distDir, "index.html")
	if err := os.WriteFile(indexHTML, []byte("<h1>Test</h1>"), 0644); err != nil {
		t.Fatal(err)
	}

	assetsDir := filepath.Join(distDir, "_assets")
	if err := os.MkdirAll(assetsDir, 0755); err != nil {
		t.Fatal(err)
	}
	styleCSS := filepath.Join(assetsDir, "style.css")
	if err := os.WriteFile(styleCSS, []byte("body{}"), 0644); err != nil {
		t.Fatal(err)
	}

	adapter := New()
	cfg := &adapters.BuildConfig{
		Config: &config.Config{
			Output: config.OutputConfig{
				Type: config.OutputStatic,
			},
		},
		OutDir:    distDir,
		PagesDir:  filepath.Join(tmpDir, "pages"),
		PublicDir: filepath.Join(tmpDir, "public"),
		Routes:    []adapters.RouteInfo{},
	}

	if err := adapter.Build(cfg); err != nil {
		t.Fatalf("build failed: %v", err)
	}

	vercelOutputDir := filepath.Join(distDir, ".vercel", "output")
	staticDir := filepath.Join(vercelOutputDir, "static")

	if _, err := os.Stat(staticDir); os.IsNotExist(err) {
		t.Fatal("static directory was not created")
	}

	staticIndex := filepath.Join(staticDir, "index.html")
	if _, err := os.Stat(staticIndex); os.IsNotExist(err) {
		t.Fatal("index.html was not copied to static directory")
	}

	staticStyle := filepath.Join(staticDir, "_assets", "style.css")
	if _, err := os.Stat(staticStyle); os.IsNotExist(err) {
		t.Fatal("_assets/style.css was not copied to static directory")
	}

	configPath := filepath.Join(vercelOutputDir, "config.json")
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Fatal("config.json was not created")
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}

	var vcfg VercelConfig
	if err := json.Unmarshal(data, &vcfg); err != nil {
		t.Fatalf("invalid config.json: %v", err)
	}

	if vcfg.Version != 3 {
		t.Errorf("expected version 3, got %d", vcfg.Version)
	}

	if len(vcfg.Routes) < 2 {
		t.Errorf("expected at least 2 routes, got %d", len(vcfg.Routes))
	}

	foundAssetRoute := false
	foundFilesystem := false

	for _, route := range vcfg.Routes {
		if route.Src == "^/_assets/(.*)$" {
			foundAssetRoute = true
			if route.Headers["cache-control"] != "public, max-age=31536000, immutable" {
				t.Error("asset route missing correct cache-control header")
			}
		}
		if route.Handle == "filesystem" {
			foundFilesystem = true
		}
	}

	if !foundAssetRoute {
		t.Error("asset caching route not found")
	}
	if !foundFilesystem {
		t.Error("filesystem route not found")
	}
}

func TestVercelAdapter_BuildOutputAPIv3Compliance(t *testing.T) {
	tmpDir := t.TempDir()
	distDir := filepath.Join(tmpDir, "dist")

	if err := os.MkdirAll(distDir, 0755); err != nil {
		t.Fatal(err)
	}

	indexHTML := filepath.Join(distDir, "index.html")
	if err := os.WriteFile(indexHTML, []byte("<!DOCTYPE html><html><body>Test</body></html>"), 0644); err != nil {
		t.Fatal(err)
	}

	aboutDir := filepath.Join(distDir, "about")
	if err := os.MkdirAll(aboutDir, 0755); err != nil {
		t.Fatal(err)
	}
	aboutHTML := filepath.Join(aboutDir, "index.html")
	if err := os.WriteFile(aboutHTML, []byte("<!DOCTYPE html><html><body>About</body></html>"), 0644); err != nil {
		t.Fatal(err)
	}

	adapter := New()
	cfg := &adapters.BuildConfig{
		Config: &config.Config{
			Output: config.OutputConfig{
				Type: config.OutputStatic,
			},
		},
		OutDir:    distDir,
		PagesDir:  filepath.Join(tmpDir, "pages"),
		PublicDir: filepath.Join(tmpDir, "public"),
		Routes:    []adapters.RouteInfo{},
	}

	if err := adapter.Build(cfg); err != nil {
		t.Fatalf("build failed: %v", err)
	}

	vercelOutputDir := filepath.Join(distDir, ".vercel", "output")

	t.Run("output directory structure", func(t *testing.T) {
		if _, err := os.Stat(vercelOutputDir); os.IsNotExist(err) {
			t.Fatal(".vercel/output directory not created")
		}

		staticDir := filepath.Join(vercelOutputDir, "static")
		if _, err := os.Stat(staticDir); os.IsNotExist(err) {
			t.Fatal(".vercel/output/static directory not created")
		}
	})

	t.Run("config.json exists and valid", func(t *testing.T) {
		configPath := filepath.Join(vercelOutputDir, "config.json")
		if _, err := os.Stat(configPath); os.IsNotExist(err) {
			t.Fatal("config.json not found")
		}

		data, err := os.ReadFile(configPath)
		if err != nil {
			t.Fatal(err)
		}

		var cfg VercelConfig
		if err := json.Unmarshal(data, &cfg); err != nil {
			t.Fatalf("config.json invalid JSON: %v", err)
		}

		if cfg.Version != 3 {
			t.Errorf("Build Output API version must be 3, got %d", cfg.Version)
		}
	})

	t.Run("static files copied correctly", func(t *testing.T) {
		staticDir := filepath.Join(vercelOutputDir, "static")

		indexPath := filepath.Join(staticDir, "index.html")
		if _, err := os.Stat(indexPath); os.IsNotExist(err) {
			t.Error("index.html not copied to static/")
		}

		aboutPath := filepath.Join(staticDir, "about", "index.html")
		if _, err := os.Stat(aboutPath); os.IsNotExist(err) {
			t.Error("about/index.html not copied to static/about/")
		}
	})

	t.Run("routes array present", func(t *testing.T) {
		configPath := filepath.Join(vercelOutputDir, "config.json")
		data, _ := os.ReadFile(configPath)

		var cfg VercelConfig
		json.Unmarshal(data, &cfg)

		if cfg.Routes == nil {
			t.Error("routes array must be present (can be empty)")
		}

		if len(cfg.Routes) == 0 {
			t.Error("expected at least filesystem route")
		}
	})

	t.Run("no .vercel directory in static output", func(t *testing.T) {
		staticVercelDir := filepath.Join(vercelOutputDir, "static", ".vercel")
		if _, err := os.Stat(staticVercelDir); !os.IsNotExist(err) {
			t.Error(".vercel directory should not be copied into static/ output")
		}
	})
}

func TestVercelAdapter_RouteConfiguration(t *testing.T) {
	tmpDir := t.TempDir()
	distDir := filepath.Join(tmpDir, "dist")

	if err := os.MkdirAll(distDir, 0755); err != nil {
		t.Fatal(err)
	}

	assetsDir := filepath.Join(distDir, "_assets")
	if err := os.MkdirAll(assetsDir, 0755); err != nil {
		t.Fatal(err)
	}
	styleCSS := filepath.Join(assetsDir, "styles-abc123.css")
	if err := os.WriteFile(styleCSS, []byte("body{}"), 0644); err != nil {
		t.Fatal(err)
	}

	adapter := New()
	cfg := &adapters.BuildConfig{
		Config: &config.Config{
			Output: config.OutputConfig{
				Type: config.OutputStatic,
			},
		},
		OutDir:    distDir,
		PagesDir:  filepath.Join(tmpDir, "pages"),
		PublicDir: filepath.Join(tmpDir, "public"),
		Routes:    []adapters.RouteInfo{},
	}

	if err := adapter.Build(cfg); err != nil {
		t.Fatalf("build failed: %v", err)
	}

	configPath := filepath.Join(distDir, ".vercel", "output", "config.json")
	data, _ := os.ReadFile(configPath)

	var vcfg VercelConfig
	json.Unmarshal(data, &vcfg)

	t.Run("asset route has correct regex", func(t *testing.T) {
		found := false
		for _, route := range vcfg.Routes {
			if route.Src == "^/_assets/(.*)$" {
				found = true
				break
			}
		}
		if !found {
			t.Error("asset route regex must match ^/_assets/(.*)$")
		}
	})

	t.Run("asset route has immutable cache headers", func(t *testing.T) {
		for _, route := range vcfg.Routes {
			if route.Src == "^/_assets/(.*)$" {
				expected := "public, max-age=31536000, immutable"
				actual := route.Headers["cache-control"]
				if actual != expected {
					t.Errorf("cache-control header incorrect: got %q, want %q", actual, expected)
				}
				return
			}
		}
		t.Error("asset route not found")
	})

	t.Run("filesystem handler is last route", func(t *testing.T) {
		if len(vcfg.Routes) == 0 {
			t.Fatal("no routes configured")
		}

		lastRoute := vcfg.Routes[len(vcfg.Routes)-1]
		if lastRoute.Handle != "filesystem" {
			t.Errorf("last route must be filesystem handler, got handle=%q", lastRoute.Handle)
		}
	})

	t.Run("no invalid route fields", func(t *testing.T) {
		for i, route := range vcfg.Routes {
			if route.Handle == "filesystem" {
				if route.Src != "" {
					t.Errorf("route %d: filesystem handler should not have 'src' field", i)
				}
				if route.Dest != "" {
					t.Errorf("route %d: filesystem handler should not have 'dest' field", i)
				}
			}
		}
	})
}
