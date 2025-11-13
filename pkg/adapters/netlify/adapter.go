package netlify

import (
	"fmt"
	"path/filepath"

	"github.com/withgalaxy/galaxy/pkg/adapters"
)

type NetlifyAdapter struct{}

func New() *NetlifyAdapter {
	return &NetlifyAdapter{}
}

func (a *NetlifyAdapter) Name() string {
	return "netlify"
}

func (a *NetlifyAdapter) Build(cfg *adapters.BuildConfig) error {
	if cfg.Config.Output.Type != "static" {
		return fmt.Errorf(`netlify adapter only supports static site generation (SSG)

Current configuration: output.type = "%s"

Netlify does not support Go-based serverless functions. To deploy to Netlify:
  1. Set output.type = "static" in galaxy.config.toml
  2. Run: galaxy build
  3. Deploy: netlify deploy --prod

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

	fmt.Printf("\n✅ Netlify adapter complete\n")
	fmt.Printf("📁 Output: %s\n", cfg.OutDir)
	fmt.Printf("📄 Generated: _redirects, _headers\n")
	fmt.Printf("🚀 Deploy: netlify deploy --prod\n")

	return nil
}
