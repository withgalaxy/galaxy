package cloudflare

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/withgalaxy/galaxy/pkg/adapters"
	"github.com/withgalaxy/galaxy/pkg/config"
)

func TestCloudflareAdapter_Name(t *testing.T) {
	adapter := New()
	if adapter.Name() != "cloudflare" {
		t.Errorf("expected name 'cloudflare', got '%s'", adapter.Name())
	}
}

func TestCloudflareAdapter_RejectsNonStatic(t *testing.T) {
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

func TestCloudflareAdapter_Build(t *testing.T) {
	tmpDir := t.TempDir()
	distDir := filepath.Join(tmpDir, "dist")

	if err := os.MkdirAll(distDir, 0755); err != nil {
		t.Fatal(err)
	}

	indexHTML := filepath.Join(distDir, "index.html")
	if err := os.WriteFile(indexHTML, []byte("<h1>Test</h1>"), 0644); err != nil {
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

	redirectsPath := filepath.Join(distDir, "_redirects")
	if _, err := os.Stat(redirectsPath); os.IsNotExist(err) {
		t.Fatal("_redirects file was not created")
	}

	headersPath := filepath.Join(distDir, "_headers")
	if _, err := os.Stat(headersPath); os.IsNotExist(err) {
		t.Fatal("_headers file was not created")
	}

	redirectsContent, _ := os.ReadFile(redirectsPath)
	if !strings.Contains(string(redirectsContent), "/* /index.html 200") {
		t.Error("_redirects missing SPA fallback rule")
	}

	headersContent, _ := os.ReadFile(headersPath)
	if !strings.Contains(string(headersContent), "/_assets/*") {
		t.Error("_headers missing asset path")
	}
	if !strings.Contains(string(headersContent), "max-age=31536000, immutable") {
		t.Error("_headers missing immutable cache-control")
	}
}

func TestCloudflareAdapter_RedirectsFormat(t *testing.T) {
	tmpDir := t.TempDir()

	adapter := New()
	cfg := &adapters.BuildConfig{
		Config: &config.Config{
			Output: config.OutputConfig{
				Type: config.OutputStatic,
			},
		},
		OutDir:    tmpDir,
		PagesDir:  "",
		PublicDir: "",
		Routes:    []adapters.RouteInfo{},
	}

	if err := adapter.Build(cfg); err != nil {
		t.Fatalf("build failed: %v", err)
	}

	redirectsPath := filepath.Join(tmpDir, "_redirects")
	content, _ := os.ReadFile(redirectsPath)
	lines := strings.Split(string(content), "\n")

	foundSPA := false
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "/* /index.html 200" {
			foundSPA = true
		}
	}

	if !foundSPA {
		t.Error("_redirects must contain SPA fallback: /* /index.html 200")
	}
}

func TestCloudflareAdapter_HeadersFormat(t *testing.T) {
	tmpDir := t.TempDir()

	adapter := New()
	cfg := &adapters.BuildConfig{
		Config: &config.Config{
			Output: config.OutputConfig{
				Type: config.OutputStatic,
			},
		},
		OutDir:    tmpDir,
		PagesDir:  "",
		PublicDir: "",
		Routes:    []adapters.RouteInfo{},
	}

	if err := adapter.Build(cfg); err != nil {
		t.Fatalf("build failed: %v", err)
	}

	headersPath := filepath.Join(tmpDir, "_headers")
	content, _ := os.ReadFile(headersPath)

	if !strings.Contains(string(content), "/_assets/*") {
		t.Error("_headers must have /_assets/* path matcher")
	}

	if !strings.Contains(string(content), "  Cache-Control: public, max-age=31536000, immutable") {
		t.Error("_headers must have indented Cache-Control header")
	}
}
