package vercel

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/withgalaxy/galaxy/pkg/adapters"
)

type VercelAdapter struct{}

func New() *VercelAdapter {
	return &VercelAdapter{}
}

func (a *VercelAdapter) Name() string {
	return "vercel"
}

func (a *VercelAdapter) Build(cfg *adapters.BuildConfig) error {
	if cfg.Config.Output.Type != "static" {
		return fmt.Errorf(`vercel adapter only supports static site generation (SSG)

Current configuration: output.type = "%s"

Vercel does not support Go-based serverless functions. To deploy to Vercel:
  1. Set output.type = "static" in galaxy.config.toml
  2. Run: galaxy build
  3. Deploy: vercel deploy

For server-side rendering (SSR), use the "standalone" adapter and deploy to a platform that supports Go (e.g., Docker, VPS, Railway, Fly.io)`, cfg.Config.Output.Type)
	}

	vercelDir := filepath.Join(cfg.OutDir, ".vercel")
	outputDir := filepath.Join(vercelDir, "output")
	staticDir := filepath.Join(outputDir, "static")

	if err := os.MkdirAll(staticDir, 0755); err != nil {
		return fmt.Errorf("failed to create .vercel/output directory: %w", err)
	}

	if err := a.copyStaticFiles(cfg.OutDir, staticDir); err != nil {
		return fmt.Errorf("failed to copy static files to .vercel/output/static: %w\n\nThis usually means the build output is missing. Ensure the SSG build completed successfully.", err)
	}

	configPath := filepath.Join(outputDir, "config.json")
	if err := a.generateConfig(cfg, configPath); err != nil {
		return fmt.Errorf("failed to generate .vercel/output/config.json: %w", err)
	}

	fmt.Printf("\n✅ Vercel adapter complete\n")
	fmt.Printf("📁 Output: %s\n", outputDir)
	fmt.Printf("🚀 Deploy: vercel deploy\n")

	return nil
}

func (a *VercelAdapter) copyStaticFiles(srcDir, destDir string) error {
	entries, err := os.ReadDir(srcDir)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		if entry.Name() == ".vercel" {
			continue
		}

		srcPath := filepath.Join(srcDir, entry.Name())
		destPath := filepath.Join(destDir, entry.Name())

		if entry.IsDir() {
			if err := os.MkdirAll(destPath, 0755); err != nil {
				return err
			}
			if err := a.copyDir(srcPath, destPath); err != nil {
				return err
			}
		} else {
			data, err := os.ReadFile(srcPath)
			if err != nil {
				return err
			}
			if err := os.WriteFile(destPath, data, 0644); err != nil {
				return err
			}
		}
	}

	return nil
}

func (a *VercelAdapter) copyDir(src, dest string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}

		destPath := filepath.Join(dest, relPath)

		if info.IsDir() {
			return os.MkdirAll(destPath, 0755)
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		return os.WriteFile(destPath, data, 0644)
	})
}

func (a *VercelAdapter) generateConfig(cfg *adapters.BuildConfig, configPath string) error {
	config := &VercelConfig{
		Version: 3,
		Routes: []Route{
			{
				Src:     "^/_assets/(.*)$",
				Headers: map[string]string{"cache-control": "public, max-age=31536000, immutable"},
			},
			{Handle: "filesystem"},
		},
	}

	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(configPath, data, 0644)
}
