package cli

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/withgalaxy/galaxy/pkg/config"
	galaxyOrbit "github.com/withgalaxy/galaxy/pkg/orbit"
	orbitConfig "github.com/withgalaxy/orbit/config"
	"github.com/withgalaxy/orbit/dev_server"
)

var (
	devPort    int
	devHost    string
	devOpen    bool
	devVerbose bool
	devCodegen bool
)

var devCmd = &cobra.Command{
	Use:   "dev",
	Short: "Start the development server",
	Long:  `Start the development server with hot reload`,
	RunE:  runDev,
}

func init() {
	rootCmd.AddCommand(devCmd)
	devCmd.Flags().IntVar(&devPort, "port", 5173, "port to run server on")
	devCmd.Flags().StringVar(&devHost, "host", "localhost", "host to bind to")
	devCmd.Flags().BoolVar(&devOpen, "open", false, "open browser on start")
	devCmd.Flags().BoolVar(&devVerbose, "verbose", false, "enable request logging")
	devCmd.Flags().BoolVar(&devCodegen, "codegen", true, "use codegen server for production-like performance")
}

func runDev(cmd *cobra.Command, args []string) error {
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}

	if rootDir != "" {
		cwd = rootDir
	}

	_, err = config.LoadFromDir(cwd)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	pagesDir := filepath.Join(cwd, "src/pages")
	publicDir := filepath.Join(cwd, "public")

	if _, err = os.Stat(pagesDir); os.IsNotExist(err) {
		return fmt.Errorf("pages directory not found: %s", pagesDir)
	}

	orbitConfigPath := filepath.Join(cwd, "orbit.toml")
	var cfg *orbitConfig.Config
	if _, err := os.Stat(orbitConfigPath); err == nil {
		cfg, err = orbitConfig.LoadConfig(orbitConfigPath)
		if err != nil {
			log.Printf("Warning: failed to load orbit.toml: %v", err)
			cfg = orbitConfig.DefaultConfig()
		}
	} else {
		cfg = orbitConfig.DefaultConfig()
	}

	cfg.Root = cwd
	cfg.PublicDir = publicDir
	if devPort != 5173 {
		cfg.Server.Port = devPort
	}
	if devHost != "localhost" {
		cfg.Server.Host = devHost
	}
	cfg.HMR.Enabled = true

	srv, err := dev_server.New(cfg)
	if err != nil {
		log.Fatal(err)
	}

	galaxyPlugin := galaxyOrbit.NewGalaxyPlugin(cwd, pagesDir, publicDir)
	galaxyPlugin.UseCodegen = devCodegen
	srv.Use(galaxyPlugin)
	srv.Plugins.AddMiddleware(galaxyPlugin.Middleware())

	return srv.Start()
}
