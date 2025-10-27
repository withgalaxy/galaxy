package main

import (
	"log"
	"os"
	"path/filepath"

	galaxyOrbit "github.com/cameron-webmatter/galaxy/pkg/orbit"
	"github.com/cameron-webmatter/orbit/config"
	"github.com/cameron-webmatter/orbit/dev_server"
)

func main() {
	cwd, _ := os.Getwd()
	pagesDir := filepath.Join(cwd, "src/pages")
	publicDir := filepath.Join(cwd, "public")

	cfg := config.DefaultConfig()
	cfg.Root = cwd
	cfg.PublicDir = publicDir
	cfg.Server.Port = 5173
	cfg.HMR.Enabled = true

	srv, err := dev_server.New(cfg)
	if err != nil {
		log.Fatal(err)
	}

	galaxyPlugin := galaxyOrbit.NewGalaxyPlugin(cwd, pagesDir, publicDir)

	srv.Use(galaxyPlugin)
	srv.Plugins.AddMiddleware(galaxyPlugin.Middleware())

	if err := srv.Start(); err != nil {
		log.Fatal(err)
	}
}
