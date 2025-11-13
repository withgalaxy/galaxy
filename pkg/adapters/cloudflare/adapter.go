package cloudflare

import (
	"fmt"
	"path/filepath"

	"github.com/withgalaxy/galaxy/pkg/adapters"
)

type CloudflareAdapter struct{}

func New() *CloudflareAdapter {
	return &CloudflareAdapter{}
}

func (a *CloudflareAdapter) Name() string {
	return "cloudflare"
}

func (a *CloudflareAdapter) Build(cfg *adapters.BuildConfig) error {
	if cfg.Config.Output.Type != "static" {
		return fmt.Errorf(`cloudflare adapter only supports static site generation (SSG)

Current configuration: output.type = "%s"

Cloudflare Pages does not support Go-based serverless functions. To deploy to Cloudflare:
  1. Set output.type = "static" in galaxy.config.toml
  2. Run: galaxy build
  3. Deploy: wrangler pages deploy dist

For server-side rendering (SSR), use the "standalone" adapter and deploy to a platform that supports Go (e.g., Docker, VPS, Railway, Fly.io)`, cfg.Config.Output.Type)
	}

	redirectsPath := filepath.Join(cfg.OutDir, "_redirects")
	if err := generateRedirects(redirectsPath); err != nil {
		return fmt.Errorf("failed to generate _redirects file: %w", err)
	}

	headersPath := filepath.Join(cfg.OutDir, "_headers")
	if err := generateHeaders(headersPath, cfg); err != nil {
		return fmt.Errorf("failed to generate _headers file: %w", err)
	}

	fmt.Printf("\n✅ Cloudflare adapter complete\n")
	fmt.Printf("📁 Output: %s\n", cfg.OutDir)
	fmt.Printf("📄 Generated: _redirects, _headers\n")
	fmt.Printf("🚀 Deploy: wrangler pages deploy dist\n")

	return nil
}
