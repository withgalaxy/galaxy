package cli

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/withgalaxy/galaxy/pkg/config"
	galaxyOrbit "github.com/withgalaxy/galaxy/pkg/orbit"
	orbitConfig "github.com/withgalaxy/orbit/config"
	"github.com/withgalaxy/orbit/dev_server"
	"github.com/spf13/cobra"
)

var (
	devPort    int
	devHost    string
	devOpen    bool
	devVerbose bool
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

	cfg := orbitConfig.DefaultConfig()
	cfg.Root = cwd
	cfg.PublicDir = publicDir
	cfg.Server.Port = devPort
	cfg.Server.Host = devHost
	cfg.HMR.Enabled = true

	srv, err := dev_server.New(cfg)
	if err != nil {
		log.Fatal(err)
	}

	galaxyPlugin := galaxyOrbit.NewGalaxyPlugin(cwd, pagesDir, publicDir)
	srv.Use(galaxyPlugin)
	srv.Plugins.AddMiddleware(galaxyPlugin.Middleware())

	return srv.Start()
}
